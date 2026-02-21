package issue

import (
	"sort"
	"strings"
)

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
			ids = append(ids, e.Name())
		}
	}

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
		if filter.Grep != "" {
			needle := strings.ToLower(filter.Grep)
			if !strings.Contains(strings.ToLower(issue.Title), needle) &&
				!strings.Contains(strings.ToLower(issue.Description), needle) {
				continue
			}
		}
		issues = append(issues, issue)
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Priority != issues[j].Priority {
			return issues[i].Priority < issues[j].Priority
		}
		return issues[i].Created < issues[j].Created
	})

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
