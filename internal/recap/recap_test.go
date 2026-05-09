package recap

import (
	"testing"
	"time"

	"github.com/jallum/beadwork/internal/treefs"
)

type fakeLookup struct {
	titles map[string]string
}

func (f *fakeLookup) Title(id string) string {
	return f.titles[id]
}

func TestBuildEmpty(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	w := Window{Start: now.Add(-24 * time.Hour), End: now, Label: "test"}
	r := Build(nil, w, nil)
	if len(r.Sections) != 0 {
		t.Errorf("expected empty recap, got %d sections", len(r.Sections))
	}
}

func TestBuildGroupsByIssue(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	w := Window{Start: now.Add(-24 * time.Hour), End: now, Label: "test"}

	commits := []treefs.CommitInfo{
		{Message: `create bw-1 p2 task "First"`, Time: now.Add(-2 * time.Hour)},
		{Message: `create bw-2 p2 task "Second"`, Time: now.Add(-1 * time.Hour)},
		{Message: "start bw-1", Time: now.Add(-30 * time.Minute)},
	}

	lookup := &fakeLookup{titles: map[string]string{
		"bw-1": "First",
		"bw-2": "Second",
	}}

	r := Build(commits, w, lookup)
	if len(r.Sections) != 2 {
		t.Fatalf("sections = %d, want 2", len(r.Sections))
	}

	// bw-1 should have 2 events (create + start)
	s1 := findSection(r, "bw-1")
	if s1 == nil {
		t.Fatal("section bw-1 not found")
	}
	if len(s1.Leaves) != 2 {
		t.Errorf("bw-1 events = %d, want 2", len(s1.Leaves))
	}
	if s1.Title != "First" {
		t.Errorf("bw-1 title = %q, want 'First'", s1.Title)
	}
}

func TestBuildFiltersWindow(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	w := Window{Start: now.Add(-1 * time.Hour), End: now, Label: "test"}

	commits := []treefs.CommitInfo{
		{Message: "create bw-old", Time: now.Add(-2 * time.Hour)}, // outside window
		{Message: "create bw-new", Time: now.Add(-30 * time.Minute)},
	}

	r := Build(commits, w, nil)
	if len(r.Sections) != 1 {
		t.Fatalf("sections = %d, want 1", len(r.Sections))
	}
	if r.Sections[0].ID != "bw-new" {
		t.Errorf("section ID = %q, want bw-new", r.Sections[0].ID)
	}
}

func TestBuildDedup(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	w := Window{Start: now.Add(-24 * time.Hour), End: now}
	ts := now.Add(-1 * time.Hour)

	// Same event appearing in two commits (e.g., replay after sync)
	commits := []treefs.CommitInfo{
		{Message: "create bw-1", Time: ts},
		{Message: "create bw-1", Time: ts},
	}

	r := Build(commits, w, nil)
	if len(r.Sections) != 1 || len(r.Sections[0].Leaves) != 1 {
		t.Errorf("expected 1 section with 1 event (deduped), got %v", r.Sections)
	}
}

func TestBuildUnblockedSecondary(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	w := Window{Start: now.Add(-24 * time.Hour), End: now}

	commits := []treefs.CommitInfo{
		{Message: "close bw-1\nunblocked bw-2", Time: now.Add(-1 * time.Hour)},
	}

	r := Build(commits, w, nil)
	if len(r.Sections) != 2 {
		t.Fatalf("sections = %d, want 2 (bw-1 close + bw-2 unblocked)", len(r.Sections))
	}
}

func findSection(r Recap, id string) *Section {
	for i := range r.Sections {
		if r.Sections[i].ID == id {
			return &r.Sections[i]
		}
	}
	return nil
}
