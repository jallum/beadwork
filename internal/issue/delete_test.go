package issue_test

import (
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestDeletePreviewNoDeps(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Solo issue", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	env.CommitIntent("create " + iss.ID)

	plan, err := env.Store.DeletePreview(iss.ID)
	if err != nil {
		t.Fatalf("DeletePreview: %v", err)
	}
	if plan.Issue.ID != iss.ID {
		t.Errorf("plan.Issue.ID = %q, want %q", plan.Issue.ID, iss.ID)
	}
	if len(plan.Blocks) != 0 {
		t.Errorf("plan.Blocks = %v, want empty", plan.Blocks)
	}
	if len(plan.BlockedBy) != 0 {
		t.Errorf("plan.BlockedBy = %v, want empty", plan.BlockedBy)
	}
	if len(plan.Children) != 0 {
		t.Errorf("plan.Children = %v, want empty", plan.Children)
	}
}

func TestDeletePreviewBlocks(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	plan, err := env.Store.DeletePreview(a.ID)
	if err != nil {
		t.Fatalf("DeletePreview: %v", err)
	}
	if len(plan.Blocks) != 1 || plan.Blocks[0] != b.ID {
		t.Errorf("plan.Blocks = %v, want [%s]", plan.Blocks, b.ID)
	}
}

func TestDeletePreviewBlockedBy(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	plan, err := env.Store.DeletePreview(b.ID)
	if err != nil {
		t.Fatalf("DeletePreview: %v", err)
	}
	if len(plan.BlockedBy) != 1 || plan.BlockedBy[0] != a.ID {
		t.Errorf("plan.BlockedBy = %v, want [%s]", plan.BlockedBy, a.ID)
	}
}

func TestDeletePreviewChildren(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Parent", issue.CreateOpts{})
	env.CommitIntent("create parent")
	env.Store.Import(&issue.Issue{
		ID:        "test-child",
		Title:     "Child",
		Status:    "open",
		Priority:  2,
		Type:      "task",
		Created:   "2024-01-01",
		Parent:    parent.ID,
		Labels:    []string{},
		Blocks:    []string{},
		BlockedBy: []string{},
	})
	env.CommitIntent("setup")

	plan, err := env.Store.DeletePreview(parent.ID)
	if err != nil {
		t.Fatalf("DeletePreview: %v", err)
	}
	if len(plan.Children) != 1 || plan.Children[0] != "test-child" {
		t.Errorf("plan.Children = %v, want [test-child]", plan.Children)
	}
}

func TestDeletePreviewNonexistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.DeletePreview("test-zzzz")
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

// --- Delete tests ---

func TestDeleteRemovesIssueFile(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To delete", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	_, err := env.Store.Delete(iss.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	env.CommitIntent("delete " + iss.ID)

	_, err = env.Store.Get(iss.ID)
	if err == nil {
		t.Error("expected error getting deleted issue")
	}
}

func TestDeleteRemovesStatusMarker(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To delete", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	env.Store.Delete(iss.ID)
	env.CommitIntent("delete " + iss.ID)

	issues, _ := env.Store.List(issue.Filter{})
	for _, i := range issues {
		if i.ID == iss.ID {
			t.Error("deleted issue still appears in list")
		}
	}
}

func TestDeleteCleansUpBlocksMarkers(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	env.Store.Delete(a.ID)
	env.CommitIntent("delete " + a.ID)

	// The blocks marker file should be gone
	if env.MarkerExists("blocks/" + a.ID + "/" + b.ID) {
		t.Error("blocks marker still exists after delete")
	}
}

func TestDeleteUpdatesRelatedBlockedBy(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	env.Store.Delete(a.ID)
	env.CommitIntent("delete " + a.ID)

	gotB, _ := env.Store.Get(b.ID)
	if len(gotB.BlockedBy) != 0 {
		t.Errorf("blocked.BlockedBy = %v, want empty", gotB.BlockedBy)
	}
}

func TestDeleteUpdatesRelatedBlocks(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	env.Store.Delete(b.ID)
	env.CommitIntent("delete " + b.ID)

	gotA, _ := env.Store.Get(a.ID)
	if len(gotA.Blocks) != 0 {
		t.Errorf("blocker.Blocks = %v, want empty", gotA.Blocks)
	}
}

func TestDeleteOrphansChildren(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Parent", issue.CreateOpts{})
	env.CommitIntent("create parent")
	env.Store.Import(&issue.Issue{
		ID:        "test-child",
		Title:     "Child",
		Status:    "open",
		Priority:  2,
		Type:      "task",
		Created:   "2024-01-01",
		Parent:    parent.ID,
		Labels:    []string{},
		Blocks:    []string{},
		BlockedBy: []string{},
	})
	env.CommitIntent("setup")

	env.Store.Delete(parent.ID)
	env.CommitIntent("delete " + parent.ID)

	gotChild, _ := env.Store.Get("test-child")
	if gotChild.Parent != "" {
		t.Errorf("child.Parent = %q, want empty", gotChild.Parent)
	}
}

func TestDeleteNoRelationshipsClean(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Solo", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	deleted, err := env.Store.Delete(iss.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted.ID != iss.ID {
		t.Errorf("deleted.ID = %q, want %q", deleted.ID, iss.ID)
	}
	if deleted.Title != "Solo" {
		t.Errorf("deleted.Title = %q, want Solo", deleted.Title)
	}
}

func TestDeleteNonexistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.Delete("test-zzzz")
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

