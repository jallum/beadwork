package issue_test

import (
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestDeleteSubtreeNested(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	c := child(t, env, "Child", root.ID)
	gc := child(t, env, "Grandchild", c.ID)
	env.CommitIntent("setup")

	result, err := env.Store.DeleteSubtree(root.ID)
	if err != nil {
		t.Fatalf("DeleteSubtree: %v", err)
	}

	if len(result.Deleted) != 3 {
		t.Fatalf("deleted %d issues, want 3", len(result.Deleted))
	}
	// Leaf-up order: grandchild, child, root.
	wantOrder := []string{gc.ID, c.ID, root.ID}
	for i, id := range wantOrder {
		if result.Deleted[i].ID != id {
			t.Errorf("deleted[%d] = %s, want %s", i, result.Deleted[i].ID, id)
		}
	}

	// All three must be gone from the live tree.
	for _, id := range wantOrder {
		if _, err := env.Store.Get(id); err == nil {
			t.Errorf("%s should no longer resolve after delete", id)
		}
	}
}

func TestDeleteSubtreeDeletesClosedAndOpenMembers(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	open := child(t, env, "Open child", root.ID)
	closed := child(t, env, "Closed child", root.ID)
	env.Store.Close(closed.ID, "")
	env.CommitIntent("setup")

	result, err := env.Store.DeleteSubtree(root.ID)
	if err != nil {
		t.Fatalf("DeleteSubtree: %v", err)
	}
	// Unlike archive/close, delete removes members regardless of status.
	if len(result.Deleted) != 3 {
		t.Fatalf("deleted %d, want 3 (root, %s, %s)", len(result.Deleted), open.ID, closed.ID)
	}
	for _, id := range []string{root.ID, open.ID, closed.ID} {
		if _, err := env.Store.Get(id); err == nil {
			t.Errorf("%s should be gone", id)
		}
	}
}

func TestDeleteSubtreeThroughClosedNodeReachesOpenDescendant(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	mid := child(t, env, "Closed mid", root.ID)
	leaf := child(t, env, "Open leaf", mid.ID)
	env.Store.Close(mid.ID, "")
	env.CommitIntent("setup")

	result, err := env.Store.DeleteSubtree(root.ID)
	if err != nil {
		t.Fatalf("DeleteSubtree: %v", err)
	}
	// The walk must pass through the closed mid to reach the open leaf.
	if len(result.Deleted) != 3 {
		t.Fatalf("deleted %d, want 3", len(result.Deleted))
	}
	if _, err := env.Store.Get(leaf.ID); err == nil {
		t.Error("open leaf under a closed mid should be deleted")
	}
}

func TestDeleteSubtreeExternalUnblockSurfaced(t *testing.T) {
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

	result, err := env.Store.DeleteSubtree(root.ID)
	if err != nil {
		t.Fatalf("DeleteSubtree: %v", err)
	}

	if len(result.Unblocked) != 1 || result.Unblocked[0].ID != outsider.ID {
		t.Errorf("unblocked = %v, want [%s]", idsOf(result.Unblocked), outsider.ID)
	}
	// The edge must actually be gone from the surviving outsider.
	o, _ := env.Store.Get(outsider.ID)
	if len(o.BlockedBy) != 0 {
		t.Errorf("outsider.BlockedBy = %v, want empty after subtree delete", o.BlockedBy)
	}
}

func TestDeleteSubtreeInternalUnblockSuppressed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	a := child(t, env, "A blocks B", root.ID)
	b := child(t, env, "B blocked by A", root.ID)
	if err := env.Store.Link(a.ID, b.ID); err != nil {
		t.Fatalf("link: %v", err)
	}
	env.CommitIntent("setup")

	result, err := env.Store.DeleteSubtree(root.ID)
	if err != nil {
		t.Fatalf("DeleteSubtree: %v", err)
	}

	// B is inside the subtree and gets deleted too, so it must not appear as
	// "unblocked" — that would be noise (and a dangling reference).
	if len(result.Unblocked) != 0 {
		t.Errorf("unblocked = %v, want none", idsOf(result.Unblocked))
	}
}

func TestDeleteSubtreeExternalDependentStillBlockedNotSurfaced(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	c := child(t, env, "Child blocker", root.ID)
	otherBlocker, _ := env.Store.Create("Other open blocker", issue.CreateOpts{})
	outsider, _ := env.Store.Create("Outside, two blockers", issue.CreateOpts{})
	env.Store.Link(c.ID, outsider.ID)
	env.Store.Link(otherBlocker.ID, outsider.ID)
	env.CommitIntent("setup")

	result, err := env.Store.DeleteSubtree(root.ID)
	if err != nil {
		t.Fatalf("DeleteSubtree: %v", err)
	}
	// outsider still has an open blocker outside the subtree, so deleting the
	// subtree must not report it as unblocked.
	if len(result.Unblocked) != 0 {
		t.Errorf("unblocked = %v, want none (other blocker still open)", idsOf(result.Unblocked))
	}
}

func TestDeleteSubtreePreviewMutatesNothing(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	c := child(t, env, "Child", root.ID)
	gc := child(t, env, "Grandchild", c.ID)
	outsider, _ := env.Store.Create("Outside, blocked by child", issue.CreateOpts{})
	env.Store.Link(c.ID, outsider.ID)
	env.CommitIntent("setup")

	plan, err := env.Store.DeleteSubtreePreview(root.ID)
	if err != nil {
		t.Fatalf("DeleteSubtreePreview: %v", err)
	}
	if len(plan.Deleted) != 3 {
		t.Errorf("preview Deleted = %v, want 3 members", idsOf(plan.Deleted))
	}
	if len(plan.Unblocked) != 1 || plan.Unblocked[0].ID != outsider.ID {
		t.Errorf("preview Unblocked = %v, want [%s]", idsOf(plan.Unblocked), outsider.ID)
	}
	// Nothing should have been mutated.
	for _, id := range []string{root.ID, c.ID, gc.ID, outsider.ID} {
		if _, err := env.Store.Get(id); err != nil {
			t.Errorf("preview must not delete %s: %v", id, err)
		}
	}
	o, _ := env.Store.Get(outsider.ID)
	if len(o.BlockedBy) != 1 {
		t.Errorf("preview must not sever edges; outsider.BlockedBy = %v", o.BlockedBy)
	}
}
