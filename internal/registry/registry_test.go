package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	r, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if r.SchemaVersion != SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", r.SchemaVersion, SchemaVersion)
	}
	if len(r.Repos) != 0 {
		t.Errorf("Repos = %v, want empty", r.Repos)
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	r, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	r.Touch("/home/user/project-a", now)
	r.Touch("/home/user/project-b", now.Add(time.Hour))

	if err := r.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	r2, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}
	if len(r2.Repos) != 2 {
		t.Fatalf("Repos count = %d, want 2", len(r2.Repos))
	}

	ea := r2.Repos["/home/user/project-a"]
	if ea.LastSeenAt != "2026-01-15T10:00:00Z" {
		t.Errorf("project-a LastSeenAt = %q", ea.LastSeenAt)
	}
}

func TestSchemaVersionNewerRefused(t *testing.T) {
	dir := t.TempDir()
	data := `{"schema_version": 999, "repos": {}}`
	os.WriteFile(filepath.Join(dir, registryFile), []byte(data), 0644)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for newer schema version")
	}
}

func TestUnknownFieldPreservation(t *testing.T) {
	dir := t.TempDir()
	original := `{"schema_version":1,"repos":{},"future_field":"hello"}`
	os.WriteFile(filepath.Join(dir, registryFile), []byte(original), 0644)

	r, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	r.Touch("/tmp/repo", now)
	if err := r.Save(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, registryFile))
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	json.Unmarshal(data, &raw)
	if _, ok := raw["future_field"]; !ok {
		t.Error("future_field not preserved after save")
	}
}

func TestTouchAndSave(t *testing.T) {
	dir := t.TempDir()
	r, _ := Load(dir)

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	if err := r.TouchAndSave("/tmp/repo", "", now); err != nil {
		t.Fatalf("TouchAndSave: %v", err)
	}

	r2, _ := Load(dir)
	if len(r2.Repos) != 1 {
		t.Fatalf("Repos count = %d, want 1", len(r2.Repos))
	}
}

func TestAdvanceCursorAndSave(t *testing.T) {
	dir := t.TempDir()
	r, _ := Load(dir)

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	r.Touch("/tmp/repo", now)
	r.Save()

	if err := r.AdvanceCursorAndSave("/tmp/repo", "abc123"); err != nil {
		t.Fatalf("AdvanceCursorAndSave: %v", err)
	}

	r2, _ := Load(dir)
	e := r2.Repos["/tmp/repo"]
	if e.Cursor != "abc123" {
		t.Errorf("Cursor = %q, want abc123", e.Cursor)
	}
	if e.LastSeenAt != "2026-01-15T10:00:00Z" {
		t.Errorf("LastSeenAt lost after cursor advance: %q", e.LastSeenAt)
	}
}

func TestPrune(t *testing.T) {
	dir := t.TempDir()
	r, _ := Load(dir)

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	r.Touch("/keep", now)
	r.Touch("/remove-me", now.Add(-30*24*time.Hour))

	cutoff := now.Add(-7 * 24 * time.Hour)
	removed := r.Prune(func(path string, e Entry) bool {
		t, err := time.Parse(time.RFC3339, e.LastSeenAt)
		if err != nil {
			return false
		}
		return t.Before(cutoff)
	})

	if len(removed) != 1 || removed[0] != "/remove-me" {
		t.Errorf("Prune removed = %v, want [/remove-me]", removed)
	}
	if _, ok := r.Repos["/keep"]; !ok {
		t.Error("/keep was incorrectly pruned")
	}
}

func TestConcurrentAdvanceCursorAndSave(t *testing.T) {
	dir := t.TempDir()
	r, _ := Load(dir)

	// Pre-populate with N repos.
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	for i := range 20 {
		r.Touch("/repo/"+string(rune('a'+i)), now)
	}
	r.Save()

	// Concurrently advance cursors for different repos.
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			r.AdvanceCursorAndSave("/repo/"+string(rune('a'+n)), "cur-"+string(rune('a'+n)))
		}(i)
	}
	wg.Wait()

	r2, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after concurrent cursor advances: %v", err)
	}
	// Count how many entries got their cursor set.
	withCursor := 0
	for _, e := range r2.Repos {
		if e.Cursor != "" {
			withCursor++
		}
	}
	// Due to last-writer-wins at the file level we expect at least one cursor set;
	// since each goroutine re-reads/writes the whole file atomically, this is best-effort.
	// What we must guarantee is no corruption: the file is valid JSON with all repos.
	if len(r2.Repos) != 20 {
		t.Errorf("lost entries during concurrent cursor advance: %d/20", len(r2.Repos))
	}
	if withCursor == 0 {
		t.Errorf("no cursors set after concurrent advances")
	}
}

func TestConcurrentSave(t *testing.T) {
	dir := t.TempDir()
	r, _ := Load(dir)

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC).Add(time.Duration(n) * time.Second)
			r.TouchAndSave("/repo/"+string(rune('a'+n)), "", now)
		}(i)
	}
	wg.Wait()

	r2, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after concurrent saves: %v", err)
	}
	if len(r2.Repos) != 20 {
		t.Errorf("Repos count = %d, want 20", len(r2.Repos))
	}
}

func TestEntries(t *testing.T) {
	dir := t.TempDir()
	r, _ := Load(dir)
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	r.Touch("/a", now)
	r.Touch("/b", now)

	entries := r.Entries()
	if len(entries) != 2 {
		t.Errorf("Entries count = %d, want 2", len(entries))
	}
	// Mutating the snapshot should not affect the original.
	delete(entries, "/a")
	if len(r.Repos) != 2 {
		t.Error("mutating Entries() snapshot affected original")
	}
}
