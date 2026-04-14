// Package registry tracks which repositories on this host use beadwork.
// The registry is a single JSON file stored under the beadwork home
// directory (~/.beadwork by default, or $BEADWORK_HOME). It records the
// last time each repo was seen and an opaque cursor for incremental
// recap processing.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const SchemaVersion = 1
const registryFile = "registry.json"

// Entry represents a single tracked repository.
type Entry struct {
	LastSeenAt  string   `json:"last_seen_at"`
	LastRecapAt string   `json:"last_recap_at,omitempty"`
	Cursor      string   `json:"cursor,omitempty"`
	Prefix      string   `json:"prefix,omitempty"`
	// Aliases are former prefixes this repo has used. Cross-repo lookups
	// match against Prefix and Aliases so closed issues with the old
	// prefix remain reachable after a rename.
	Aliases []string `json:"aliases,omitempty"`
}

// Registry holds the in-memory state of the registry file.
type Registry struct {
	SchemaVersion int              `json:"schema_version"`
	Repos         map[string]Entry `json:"repos"`

	// extra preserves unknown top-level fields across load/save cycles.
	extra map[string]json.RawMessage

	dir  string // directory containing the registry file
	mu   sync.Mutex
}

// Load reads the registry from dir. If the file does not exist, returns
// an empty registry. Returns an error if the file exists but the schema
// version is newer than this binary supports.
func Load(dir string) (*Registry, error) {
	r := &Registry{
		SchemaVersion: SchemaVersion,
		Repos:         make(map[string]Entry),
		dir:           dir,
	}

	path := filepath.Join(dir, registryFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, fmt.Errorf("read registry: %w", err)
	}

	// Decode into a raw map first to preserve unknown fields.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}

	// Extract known fields.
	if v, ok := raw["schema_version"]; ok {
		if err := json.Unmarshal(v, &r.SchemaVersion); err != nil {
			return nil, fmt.Errorf("parse schema_version: %w", err)
		}
		delete(raw, "schema_version")
	}

	if r.SchemaVersion > SchemaVersion {
		return nil, fmt.Errorf("registry schema version %d is newer than supported (%d); upgrade bw", r.SchemaVersion, SchemaVersion)
	}

	if v, ok := raw["repos"]; ok {
		if err := json.Unmarshal(v, &r.Repos); err != nil {
			return nil, fmt.Errorf("parse repos: %w", err)
		}
		delete(raw, "repos")
	}

	r.extra = raw
	return r, nil
}

// Save atomically writes the registry to disk using a temp-file + rename.
func (r *Registry) Save() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveLocked()
}

func (r *Registry) saveLocked() error {
	if err := os.MkdirAll(r.dir, 0755); err != nil {
		return fmt.Errorf("create registry dir: %w", err)
	}

	// Build the output map preserving unknown fields.
	out := make(map[string]interface{}, len(r.extra)+2)
	for k, v := range r.extra {
		out[k] = v
	}
	out["schema_version"] = r.SchemaVersion
	out["repos"] = r.Repos

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}
	data = append(data, '\n')

	path := filepath.Join(r.dir, registryFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write temp registry: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename registry: %w", err)
	}
	return nil
}

// Touch registers or updates a repo entry with the given timestamp.
func (r *Registry) Touch(repoPath string, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	e := r.Repos[repoPath]
	e.LastSeenAt = now.UTC().Format(time.RFC3339)
	r.Repos[repoPath] = e
}

// TouchAndSave is a convenience that calls Touch then Save.
// If prefix is non-empty, it replaces the stored Prefix.
// aliases replaces the stored Aliases list when non-nil (pass nil to
// leave Aliases unchanged; pass []string{} to clear them explicitly).
func (r *Registry) TouchAndSave(repoPath, prefix string, aliases []string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e := r.Repos[repoPath]
	e.LastSeenAt = now.UTC().Format(time.RFC3339)
	if prefix != "" {
		e.Prefix = prefix
	}
	if aliases != nil {
		cp := make([]string, len(aliases))
		copy(cp, aliases)
		sort.Strings(cp)
		e.Aliases = cp
	}
	r.Repos[repoPath] = e
	return r.saveLocked()
}

// LookupPrefix returns all repo paths whose primary Prefix or Aliases
// list contains the given prefix. Used for cross-repo ID resolution.
// Returning a slice lets callers detect ambiguous prefix collisions
// (multiple repos sharing the same prefix) instead of silently picking
// one in random map order.
func (r *Registry) LookupPrefix(prefix string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var paths []string
	for path, e := range r.Repos {
		if e.Prefix == prefix {
			paths = append(paths, path)
			continue
		}
		for _, a := range e.Aliases {
			if a == prefix {
				paths = append(paths, path)
				break
			}
		}
	}
	sort.Strings(paths)
	return paths
}


// AdvanceCursorAndSave updates the cursor for a repo and saves atomically.
func (r *Registry) AdvanceCursorAndSave(repoPath, cursor string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e := r.Repos[repoPath]
	e.Cursor = cursor
	r.Repos[repoPath] = e
	return r.saveLocked()
}

// StampRecapAndSave records that a recap ran for the repo at `now` and
// optionally advances the cursor. Pass cursor="" to leave the cursor
// untouched (e.g. when there were no new commits).
func (r *Registry) StampRecapAndSave(repoPath, cursor string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e := r.Repos[repoPath]
	e.LastRecapAt = now.UTC().Format(time.RFC3339)
	if cursor != "" {
		e.Cursor = cursor
	}
	r.Repos[repoPath] = e
	return r.saveLocked()
}

// Prune removes entries for which the predicate returns true.
// Returns the list of removed repo paths.
func (r *Registry) Prune(predicate func(path string, e Entry) bool) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var removed []string
	for path, e := range r.Repos {
		if predicate(path, e) {
			delete(r.Repos, path)
			removed = append(removed, path)
		}
	}
	return removed
}

// Entries returns a snapshot of all registry entries.
func (r *Registry) Entries() map[string]Entry {
	r.mu.Lock()
	defer r.mu.Unlock()

	cp := make(map[string]Entry, len(r.Repos))
	for k, v := range r.Repos {
		cp[k] = v
	}
	return cp
}

// Dir returns the directory where the registry file lives.
func (r *Registry) Dir() string {
	return r.dir
}
