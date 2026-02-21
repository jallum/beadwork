package issue

import (
	"fmt"
	"io"

	"github.com/jallum/beadwork/internal/treefs"
)

type Comment struct {
	Text      string `json:"text"`
	Author    string `json:"author,omitempty"`
	Timestamp string `json:"timestamp"`
}

type Issue struct {
	Assignee    string    `json:"assignee"`
	BlockedBy   []string  `json:"blocked_by"`
	Blocks      []string  `json:"blocks"`
	ClosedAt    string    `json:"closed_at,omitempty"`
	CloseReason string    `json:"close_reason,omitempty"`
	Created     string    `json:"created"`
	DeferUntil  string    `json:"defer_until,omitempty"`
	Description string    `json:"description"`
	ID          string    `json:"id"`
	Labels      []string  `json:"labels"`
	Parent      string    `json:"parent,omitempty"`
	Comments    []Comment `json:"comments,omitempty"`
	Priority    int       `json:"priority"`
	Status      string    `json:"status"`
	Title       string    `json:"title"`
	Type        string    `json:"type"`
	UpdatedAt   string    `json:"updated_at,omitempty"`
}

// Committer persists pending TreeFS mutations to the underlying storage.
type Committer interface {
	Commit(message string) error
}

type Store struct {
	FS              *treefs.TreeFS
	Prefix          string
	Committer       Committer // nil for read-only use
	DefaultPriority *int
	IDRetries       int       // retries per length before bumping; 0 means 10
	RandReader      io.Reader // random source; nil means crypto/rand.Reader
	cache           map[string]*Issue
	idSet           map[string]bool // lazily populated on first resolveID/ExistingIDs call
}

// Commit persists pending mutations with the given intent message.
func (s *Store) Commit(intent string) error {
	if s.Committer == nil {
		return fmt.Errorf("store is read-only: no committer configured")
	}
	return s.Committer.Commit(intent)
}

// ClearCache discards all cached issues and the lazy ID set.
// Call after operations that change the underlying TreeFS externally
// (e.g. sync/rebase).
func (s *Store) ClearCache() {
	s.cache = nil
	s.idSet = nil
}

type CreateOpts struct {
	Parent      string
	Description string
	Priority    *int
	Type        string
	Assignee    string
	DeferUntil  string
}

type UpdateOpts struct {
	Parent      *string
	Title       *string
	Description *string
	Priority    *int
	Assignee    *string
	Type        *string
	Status      *string
	DeferUntil  *string
}

type Filter struct {
	Status   string
	Statuses []string
	Assignee string
	Priority *int
	Type     string
	Label    string
	Grep     string
}
