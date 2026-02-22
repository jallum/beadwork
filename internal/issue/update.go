package issue

import (
	"fmt"
	"strings"
)

func (s *Store) Update(id string, opts UpdateOpts) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	issue, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}

	if opts.Title != nil {
		issue.Title = *opts.Title
	}
	if opts.Description != nil {
		issue.Description = *opts.Description
	}
	if opts.Priority != nil {
		issue.Priority = *opts.Priority
	}
	if opts.Assignee != nil {
		issue.Assignee = *opts.Assignee
	}
	if opts.Type != nil {
		issue.Type = *opts.Type
	}
	if opts.DeferUntil != nil {
		issue.DeferUntil = *opts.DeferUntil
	}
	if opts.Parent != nil {
		if *opts.Parent != "" {
			if *opts.Parent == id {
				return nil, fmt.Errorf("cannot set issue as its own parent")
			}
			parentID, err := s.resolveID(*opts.Parent)
			if err != nil {
				return nil, fmt.Errorf("parent: %w", err)
			}
			// Cycle detection: walk ancestor chain
			current := parentID
			for current != "" {
				if current == id {
					return nil, fmt.Errorf("circular parent reference: %s is an ancestor of %s", id, parentID)
				}
				ancestor, err := s.readIssue(current)
				if err != nil {
					break
				}
				current = ancestor.Parent
			}
			issue.Parent = parentID
		} else {
			issue.Parent = ""
		}
	}
	if opts.Status != nil {
		oldStatus := issue.Status
		newStatus := *opts.Status
		if oldStatus != newStatus {
			if err := s.moveStatus(id, oldStatus, newStatus); err != nil {
				return nil, err
			}
			issue.Status = newStatus
		}
	}

	issue.UpdatedAt = s.nowRFC3339()
	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	return issue, nil
}

func (s *Store) Close(id, reason string) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	issue, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}
	if issue.Status == "closed" {
		return nil, fmt.Errorf("%s is already closed", id)
	}

	if err := s.moveStatus(id, issue.Status, "closed"); err != nil {
		return nil, err
	}
	now := s.nowRFC3339()
	issue.Status = "closed"
	issue.ClosedAt = now
	issue.CloseReason = reason
	issue.UpdatedAt = now
	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	return issue, nil
}

func (s *Store) Reopen(id string) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	issue, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}
	if issue.Status != "closed" {
		return nil, fmt.Errorf("%s is not closed", id)
	}

	if err := s.moveStatus(id, "closed", "open"); err != nil {
		return nil, err
	}
	issue.Status = "open"
	issue.ClosedAt = ""
	issue.CloseReason = ""
	issue.UpdatedAt = s.nowRFC3339()
	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	return issue, nil
}

// BlockedError is returned by Start when the issue has open blockers.
type BlockedError struct {
	ID       string
	Blockers []string // IDs of open blockers
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf("%s is blocked by %s", e.ID, strings.Join(e.Blockers, ", "))
}

// Start transitions an open issue to in_progress and sets its assignee.
// Returns a BlockedError if the issue has unresolved blockers.
func (s *Store) Start(id, assignee string) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	iss, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}
	if iss.Status != "open" {
		return nil, fmt.Errorf("%s is %s, not open", id, iss.Status)
	}

	// Check for open blockers
	if len(iss.BlockedBy) > 0 {
		var open []string
		for _, blockerID := range iss.BlockedBy {
			blocker, err := s.readIssue(blockerID)
			if err != nil {
				open = append(open, blockerID)
				continue
			}
			if blocker.Status != "closed" {
				open = append(open, blockerID)
			}
		}
		if len(open) > 0 {
			return nil, &BlockedError{ID: id, Blockers: open}
		}
	}

	if err := s.moveStatus(id, "open", "in_progress"); err != nil {
		return nil, err
	}
	iss.Status = "in_progress"
	iss.Assignee = assignee
	iss.UpdatedAt = s.nowRFC3339()
	if err := s.writeIssue(iss); err != nil {
		return nil, err
	}
	return iss, nil
}
