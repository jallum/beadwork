package issue

import (
	"fmt"
	"io"
	"os"
	"time"

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
	DryRun          bool      // when true, Commit logs the intent but skips persistence
	DefaultPriority *int
	IDRetries       int       // retries per length before bumping; 0 means 10
	RandReader      io.Reader // random source; nil means crypto/rand.Reader
	cache           map[string]*Issue
	idSet           map[string]bool // lazily populated on first resolveID/ExistingIDs call
}

// Commit persists pending mutations with the given intent message.
// When DryRun is true, the intent is logged to stderr and no commit is made.
func (s *Store) Commit(intent string) error {
	if s.Committer == nil {
		return fmt.Errorf("store is read-only: no committer configured")
	}
	if s.DryRun {
		fmt.Fprintf(os.Stderr, "[dry-run] would commit: %s\n", intent)
		return nil
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

// Now returns the current time in UTC. If the BW_CLOCK environment variable
// is set to an RFC3339 value, that fixed time is used instead of the real
// clock. This enables deterministic timestamps for testing and migration.
func (s *Store) Now() time.Time {
	if v := os.Getenv("BW_CLOCK"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

// nowRFC3339 returns s.Now() formatted as an RFC3339 string.
func (s *Store) nowRFC3339() string {
	return s.Now().Format(time.RFC3339)
}

type CreateOpts struct {
	ID          string // explicit ID; skips random generation but still validates
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
