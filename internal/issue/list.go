package issue

import (
	"sort"
	"strings"
	"time"
)

// IsOverdue reports whether an issue's due date has passed relative to now.
// Date-only values use end-of-day semantics: "2027-04-15" is not overdue on
// April 15, only on April 16+. This avoids the Taskwarrior pain point where
// an issue appears overdue all day on its due date.
func IsOverdue(due string, now time.Time) bool {
	if due == "" {
		return false
	}
	if strings.Contains(due, "T") {
		t, err := time.Parse(time.RFC3339, due)
		if err != nil {
			return false
		}
		return now.After(t)
	}
	// Date-only: compare against today in local timezone.
	today := now.In(time.Local).Format("2006-01-02")
	return today > due
}

// IsDeferralExpired reports whether a deferred issue's defer_until has passed.
// Date-only values use start-of-day semantics: "2027-04-15" is expired on April 15.
// This is the opposite of due date end-of-day semantics because deferral means
// "don't show me until then" — the item should reappear on that day.
func IsDeferralExpired(deferUntil string, now time.Time) bool {
	if deferUntil == "" {
		return false
	}
	if strings.Contains(deferUntil, "T") {
		t, err := time.Parse(time.RFC3339, deferUntil)
		if err != nil {
			return false
		}
		return now.After(t)
	}
	// Date-only: start-of-day. Expired when today >= defer_until.
	today := now.In(time.Local).Format("2006-01-02")
	return today >= deferUntil
}

// sortIssues sorts by priority ascending, overdue-first within priority band,
// then created ascending. The now parameter is used for overdue detection.
func sortIssues(issues []*Issue, now time.Time) {
	today := now.In(time.Local).Format("2006-01-02")
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Priority != issues[j].Priority {
			return issues[i].Priority < issues[j].Priority
		}
		oi := isOverdueQuick(issues[i].Due, issues[i].Status, today, now)
		oj := isOverdueQuick(issues[j].Due, issues[j].Status, today, now)
		if oi != oj {
			return oi
		}
		return issues[i].Created < issues[j].Created
	})
}

// isOverdueQuick checks overdue status for sorting. Closed issues are never overdue.
func isOverdueQuick(due, status, today string, now time.Time) bool {
	if due == "" || status == "closed" {
		return false
	}
	if strings.Contains(due, "T") {
		t, err := time.Parse(time.RFC3339, due)
		if err != nil {
			return false
		}
		return now.After(t)
	}
	return today > due
}

// StatusCount returns the number of issues with the given status by counting
// entries in the status index directory. No deserialization.
func (s *Store) StatusCount(status string) int {
	entries, err := s.FS.ReadDir("status/" + status)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.Name() != ".gitkeep" {
			n++
		}
	}
	return n
}

// IDsWithStatus returns issue IDs from a status index directory.
func (s *Store) IDsWithStatus(status string) []string {
	entries, err := s.FS.ReadDir("status/" + status)
	if err != nil {
		return nil
	}
	var ids []string
	for _, e := range entries {
		if e.Name() != ".gitkeep" {
			ids = append(ids, e.Name())
		}
	}
	return ids
}

// IsClosed checks whether a single issue ID appears in the closed index.
func (s *Store) IsClosed(id string) bool {
	_, err := s.FS.Stat("status/closed/" + id)
	return err == nil
}

func (s *Store) List(filter Filter) ([]*Issue, error) {
	statuses := StatusNames()
	if len(filter.Statuses) > 0 {
		statuses = filter.Statuses
	} else if filter.Status != "" {
		statuses = []string{filter.Status}
	}

	// Collect IDs from requested statuses.
	seen := make(map[string]bool)
	var ids []string
	for _, status := range statuses {
		dir := "status/" + status
		entries, err := s.FS.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.Name() == ".gitkeep" {
				continue
			}
			if !seen[e.Name()] {
				ids = append(ids, e.Name())
				seen[e.Name()] = true
			}
		}
	}

	// When IncludeExpiredDeferred is set, also pull in deferred items
	// (they'll be filtered to expired-only below).
	var deferredIDs []string
	if filter.IncludeExpiredDeferred {
		entries, err := s.FS.ReadDir("status/deferred")
		if err == nil {
			for _, e := range entries {
				if e.Name() != ".gitkeep" && !seen[e.Name()] {
					deferredIDs = append(deferredIDs, e.Name())
					seen[e.Name()] = true
				}
			}
		}
	}

	now := s.Now()

	var issues []*Issue
	for _, id := range ids {
		issue, err := s.readIssue(id)
		if err != nil {
			continue
		}

		if filter.Assignee != "" && issue.Assignee != filter.Assignee {
			continue
		}
		if filter.Priority != nil && issue.Priority != *filter.Priority {
			continue
		}
		if filter.Type != "" && issue.Type != filter.Type {
			continue
		}
		if filter.Label != "" && !containsStr(issue.Labels, filter.Label) {
			continue
		}
		if filter.Parent != "" && issue.Parent != filter.Parent {
			continue
		}
		if filter.Grep != "" {
			needle := strings.ToLower(filter.Grep)
			if !strings.Contains(strings.ToLower(issue.Title), needle) &&
				!strings.Contains(strings.ToLower(issue.Description), needle) {
				continue
			}
		}
		issues = append(issues, issue)
	}

	// Add expired deferred items (filtered to only those whose deferral has passed).
	for _, id := range deferredIDs {
		iss, err := s.readIssue(id)
		if err != nil {
			continue
		}
		if !IsDeferralExpired(iss.DeferUntil, now) {
			continue
		}
		if filter.Assignee != "" && iss.Assignee != filter.Assignee {
			continue
		}
		if filter.Priority != nil && iss.Priority != *filter.Priority {
			continue
		}
		if filter.Type != "" && iss.Type != filter.Type {
			continue
		}
		if filter.Label != "" && !containsStr(iss.Labels, filter.Label) {
			continue
		}
		if filter.Parent != "" && iss.Parent != filter.Parent {
			continue
		}
		if filter.Grep != "" {
			needle := strings.ToLower(filter.Grep)
			if !strings.Contains(strings.ToLower(iss.Title), needle) &&
				!strings.Contains(strings.ToLower(iss.Description), needle) {
				continue
			}
		}
		issues = append(issues, iss)
	}

	if filter.Overdue {
		var overdue []*Issue
		for _, iss := range issues {
			if iss.Status != "closed" && IsOverdue(iss.Due, now) {
				overdue = append(overdue, iss)
			}
		}
		issues = overdue
	}

	sortIssues(issues, now)

	return issues, nil
}

// Children returns all issues whose Parent field matches parentID,
// sorted by priority then creation date.
func (s *Store) Children(parentID string) ([]*Issue, error) {
	all, err := s.List(Filter{})
	if err != nil {
		return nil, err
	}
	var children []*Issue
	for _, iss := range all {
		if iss.Parent == parentID {
			children = append(children, iss)
		}
	}
	return children, nil
}
