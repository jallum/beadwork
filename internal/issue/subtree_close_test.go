package issue_test

import (
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

// child creates an issue parented to parentID.
func child(t *testing.T, env *testutil.Env, title, parentID string) *issue.Issue {
	t.Helper()
	iss, err := env.Store.Create(title, issue.CreateOpts{Parent: parentID})
	if err != nil {
		t.Fatalf("create %q: %v", title, err)
	}
	return iss
}

func TestCloseSubtreeNested(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	c := child(t, env, "Child", root.ID)
	gc := child(t, env, "Grandchild", c.ID)
	env.CommitIntent("setup")

	result, err := env.Store.CloseSubtree(root.ID, "shipping it")
	if err != nil {
		t.Fatalf("CloseSubtree: %v", err)
	}

	if len(result.Closed) != 3 {
		t.Fatalf("closed %d issues, want 3", len(result.Closed))
	}
	// Leaf-up order: grandchild, child, root.
	wantOrder := []string{gc.ID, c.ID, root.ID}
	for i, id := range wantOrder {
		if result.Closed[i].ID != id {
			t.Errorf("closed[%d] = %s, want %s", i, result.Closed[i].ID, id)
		}
	}

	for _, id := range wantOrder {
		got, _ := env.Store.Get(id)
		if got.Status != "closed" {
			t.Errorf("%s status = %q, want closed", id, got.Status)
		}
	}

	rootIss, _ := env.Store.Get(root.ID)
	if rootIss.CloseReason != "shipping it" {
		t.Errorf("root reason = %q, want %q", rootIss.CloseReason, "shipping it")
	}
	wantNote := "closed with parent " + root.ID
	for _, id := range []string{c.ID, gc.ID} {
		got, _ := env.Store.Get(id)
		if got.CloseReason != wantNote {
			t.Errorf("%s reason = %q, want %q", id, got.CloseReason, wantNote)
		}
	}
}

func TestCloseSubtreeSkipsAlreadyClosed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	c1 := child(t, env, "Child 1", root.ID)
	c2 := child(t, env, "Child 2", root.ID)
	env.CommitIntent("setup")

	if _, err := env.Store.Close(c1.ID, ""); err != nil {
		t.Fatalf("pre-close: %v", err)
	}
	env.CommitIntent("pre-close")

	result, err := env.Store.CloseSubtree(root.ID, "")
	if err != nil {
		t.Fatalf("CloseSubtree: %v", err)
	}

	if len(result.Skipped) != 1 || result.Skipped[0].ID != c1.ID {
		t.Errorf("skipped = %v, want [%s]", idsOf(result.Skipped), c1.ID)
	}
	if len(result.Closed) != 2 {
		t.Fatalf("closed %d, want 2 (%s, %s)", len(result.Closed), c2.ID, root.ID)
	}
	for _, id := range []string{c2.ID, root.ID} {
		got, _ := env.Store.Get(id)
		if got.Status != "closed" {
			t.Errorf("%s status = %q, want closed", id, got.Status)
		}
	}
}

func TestCloseSubtreeAlreadyClosedRoot(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	c := child(t, env, "Open child under closed root", root.ID)
	env.CommitIntent("setup")

	if _, err := env.Store.Close(root.ID, ""); err != nil {
		t.Fatalf("pre-close root: %v", err)
	}
	env.CommitIntent("pre-close root")

	// Re-running against an already-closed root should mop up the open child.
	result, err := env.Store.CloseSubtree(root.ID, "")
	if err != nil {
		t.Fatalf("CloseSubtree: %v", err)
	}
	if len(result.Closed) != 1 || result.Closed[0].ID != c.ID {
		t.Errorf("closed = %v, want [%s]", idsOf(result.Closed), c.ID)
	}
	if len(result.Skipped) != 1 || result.Skipped[0].ID != root.ID {
		t.Errorf("skipped = %v, want [%s]", idsOf(result.Skipped), root.ID)
	}
}

func TestCloseSubtreeExternalUnblockSurfaced(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	c := child(t, env, "Child blocker", root.ID)
	outsider, _ := env.Store.Create("Outside, blocked by child", issue.CreateOpts{})
	// child blocks outsider; outsider lives outside the subtree.
	if err := env.Store.Link(c.ID, outsider.ID); err != nil {
		t.Fatalf("link: %v", err)
	}
	env.CommitIntent("setup")

	result, err := env.Store.CloseSubtree(root.ID, "")
	if err != nil {
		t.Fatalf("CloseSubtree: %v", err)
	}

	if len(result.Unblocked) != 1 || result.Unblocked[0].ID != outsider.ID {
		t.Errorf("unblocked = %v, want [%s]", idsOf(result.Unblocked), outsider.ID)
	}
}

func TestCloseSubtreeInternalUnblockSuppressed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	a := child(t, env, "A blocks B", root.ID)
	b := child(t, env, "B blocked by A", root.ID)
	if err := env.Store.Link(a.ID, b.ID); err != nil {
		t.Fatalf("link: %v", err)
	}
	env.CommitIntent("setup")

	result, err := env.Store.CloseSubtree(root.ID, "")
	if err != nil {
		t.Fatalf("CloseSubtree: %v", err)
	}

	// B is inside the subtree and gets closed too, so it must not appear as
	// "unblocked" — that would be noise.
	if len(result.Unblocked) != 0 {
		t.Errorf("unblocked = %v, want none", idsOf(result.Unblocked))
	}
}

func idsOf(issues []*issue.Issue) []string {
	ids := make([]string, len(issues))
	for i, iss := range issues {
		ids[i] = iss.ID
	}
	return ids
}
