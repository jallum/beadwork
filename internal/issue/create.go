package issue

import (
	"fmt"
	"time"

	"github.com/jallum/beadwork/internal/treefs"
)

func NewStore(fs *treefs.TreeFS, prefix string) *Store {
	return &Store{FS: fs, Prefix: prefix}
}

func (s *Store) Create(title string, opts CreateOpts) (*Issue, error) {
	// Resolve parent first (needed for dotted child ID generation).
	var parentID string
	if opts.Parent != "" {
		var err error
		parentID, err = s.resolveID(opts.Parent)
		if err != nil {
			return nil, fmt.Errorf("parent: %w", err)
		}
	}

	var id string
	var err error
	if opts.ID != "" {
		id, err = s.validateExplicitID(opts.ID)
	} else if parentID != "" {
		id, err = s.generateChildID(parentID)
	} else {
		id, err = s.generateID()
	}
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	status := "open"
	if opts.DeferUntil != "" {
		status = "deferred"
	}
	issue := &Issue{
		Assignee:    opts.Assignee,
		BlockedBy:   []string{},
		Blocks:      []string{},
		Created:     now,
		DeferUntil:  opts.DeferUntil,
		Description: opts.Description,
		ID:          id,
		Labels:      []string{},
		Parent:      parentID,
		Status:      status,
		Title:       title,
		Type:        opts.Type,
		UpdatedAt:   now,
	}
	if issue.Type == "" {
		issue.Type = "task"
	}
	if opts.Priority != nil {
		issue.Priority = *opts.Priority
	} else if s.DefaultPriority != nil {
		issue.Priority = *s.DefaultPriority
	} else {
		issue.Priority = 2
	}

	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	if err := s.setStatus(id, status); err != nil {
		return nil, err
	}
	s.trackID(id)
	return issue, nil
}

// Import writes an issue with a caller-provided ID and fields.
// Used for importing from external sources where the ID is already known.
func (s *Store) Import(iss *Issue) error {
	if err := s.writeIssue(iss); err != nil {
		return err
	}
	if err := s.setStatus(iss.ID, iss.Status); err != nil {
		return err
	}
	s.trackID(iss.ID)
	return nil
}

func (s *Store) Get(id string) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	return s.readIssue(id)
}
