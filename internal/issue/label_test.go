package issue_test

import (
	"path/filepath"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestLabel(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Labeled issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// Add labels
	iss, err := env.Store.Label(iss.ID, []string{"bug", "frontend"}, nil)
	if err != nil {
		t.Fatalf("Label add: %v", err)
	}
	env.CommitIntent("label " + iss.ID + " +bug +frontend")

	// Check markers
	if !env.MarkerExists(filepath.Join("labels", "bug", iss.ID)) {
		t.Error("bug label marker missing")
	}
	if !env.MarkerExists(filepath.Join("labels", "frontend", iss.ID)) {
		t.Error("frontend label marker missing")
	}

	// Check JSON
	got, _ := env.Store.Get(iss.ID)
	if len(got.Labels) != 2 {
		t.Fatalf("labels = %v, want 2", got.Labels)
	}

	// Filter by label
	filtered, _ := env.Store.List(issue.Filter{Label: "bug"})
	if len(filtered) != 1 {
		t.Errorf("filtered by bug: got %d, want 1", len(filtered))
	}

	// Remove label
	iss, err = env.Store.Label(iss.ID, nil, []string{"frontend"})
	if err != nil {
		t.Fatalf("Label remove: %v", err)
	}
	env.CommitIntent("label " + iss.ID + " -frontend")

	if env.MarkerExists(filepath.Join("labels", "frontend", iss.ID)) {
		t.Error("frontend label marker should be gone")
	}
	got, _ = env.Store.Get(iss.ID)
	if len(got.Labels) != 1 || got.Labels[0] != "bug" {
		t.Errorf("labels = %v, want [bug]", got.Labels)
	}
}

func TestLabelDirectoryCleanup(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test", issue.CreateOpts{})
	env.Store.Label(iss.ID, []string{"temp"}, nil)
	env.Store.Label(iss.ID, nil, []string{"temp"})

	// The temp/ directory should be cleaned up (no entry in TreeFS)
	_, err := env.Repo.TreeFS().Stat("labels/temp")
	if err == nil {
		t.Error("empty label directory should be removed")
	}
}

func TestLabelIdempotent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Label test", issue.CreateOpts{})
	env.Store.Label(iss.ID, []string{"bug"}, nil)

	// Add same label again — should not duplicate
	iss2, err := env.Store.Label(iss.ID, []string{"bug"}, nil)
	if err != nil {
		t.Fatalf("second Label: %v", err)
	}
	if len(iss2.Labels) != 1 {
		t.Errorf("labels = %v, want exactly 1 entry", iss2.Labels)
	}
}

func TestRemoveNonexistentLabel(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("No labels", issue.CreateOpts{})

	// Remove a label that doesn't exist — should be fine
	iss2, err := env.Store.Label(iss.ID, nil, []string{"nonexistent"})
	if err != nil {
		t.Fatalf("Label remove nonexistent: %v", err)
	}
	if len(iss2.Labels) != 0 {
		t.Errorf("labels = %v, want empty", iss2.Labels)
	}
}

func TestLabelNonExistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.Label("test-zzzz", []string{"bug"}, nil)
	if err == nil {
		t.Error("expected error for non-existent issue")
	}
}

func TestUpdatedAtOnLabel(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Label me", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	labeled, _ := env.Store.Label(iss.ID, []string{"bug"}, nil)
	if labeled.UpdatedAt == "" {
		t.Error("updated_at should be set after label")
	}
}

