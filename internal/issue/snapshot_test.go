package issue_test

import (
	"sort"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestNewSnapshot(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Issue A", issue.CreateOpts{})
	b, _ := env.Store.Create("Issue B", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(a.ID, "")
	env.CommitIntent("setup")

	snap, err := issue.NewSnapshot(env.Store)
	if err != nil {
		t.Fatalf("NewSnapshot: %v", err)
	}

	// All issues present
	if len(snap.Issues) != 2 {
		t.Errorf("Issues count = %d, want 2", len(snap.Issues))
	}
	if snap.Issues[a.ID] == nil {
		t.Error("issue A missing from snapshot")
	}
	if snap.Issues[b.ID] == nil {
		t.Error("issue B missing from snapshot")
	}

	// IDs set
	if !snap.IDs[a.ID] || !snap.IDs[b.ID] {
		t.Error("IDs set incomplete")
	}

	// Closed set
	if !snap.Closed[a.ID] {
		t.Error("A should be in Closed set")
	}
	if snap.Closed[b.ID] {
		t.Error("B should not be in Closed set")
	}

	// Forward edges: a → [b]
	if len(snap.Forward[a.ID]) != 1 || snap.Forward[a.ID][0] != b.ID {
		t.Errorf("Forward[a] = %v, want [%s]", snap.Forward[a.ID], b.ID)
	}

	// Reverse edges: b → [a]
	if len(snap.Reverse[b.ID]) != 1 || snap.Reverse[b.ID][0] != a.ID {
		t.Errorf("Reverse[b] = %v, want [%s]", snap.Reverse[b.ID], a.ID)
	}
}

func TestSnapshotTips(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Chain: A blocked by B blocked by C
	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	c, _ := env.Store.Create("C", issue.CreateOpts{})
	env.Store.Link(c.ID, b.ID) // c blocks b
	env.Store.Link(b.ID, a.ID) // b blocks a
	env.CommitIntent("setup")

	snap, _ := issue.NewSnapshot(env.Store)

	// Re-read a to get updated BlockedBy
	aSnap := snap.Issues[a.ID]
	tips := snap.Tips(aSnap.BlockedBy, snap.Reverse)

	if len(tips) != 1 {
		t.Fatalf("got %d tips, want 1", len(tips))
	}
	if tips[0].ID != c.ID {
		t.Errorf("tip = %s, want %s", tips[0].ID, c.ID)
	}
}

func TestSnapshotTipsDiamond(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	c, _ := env.Store.Create("C", issue.CreateOpts{})
	d, _ := env.Store.Create("D", issue.CreateOpts{})
	env.Store.Link(b.ID, a.ID)
	env.Store.Link(c.ID, a.ID)
	env.Store.Link(d.ID, b.ID)
	env.Store.Link(d.ID, c.ID)
	env.CommitIntent("setup")

	snap, _ := issue.NewSnapshot(env.Store)
	aSnap := snap.Issues[a.ID]
	tips := snap.Tips(aSnap.BlockedBy, snap.Reverse)

	if len(tips) != 1 {
		t.Fatalf("got %d tips, want 1", len(tips))
	}
	if tips[0].ID != d.ID {
		t.Errorf("tip = %s, want %s", tips[0].ID, d.ID)
	}
}

func TestSnapshotTipsNoRoots(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("A", issue.CreateOpts{})
	env.CommitIntent("setup")

	snap, _ := issue.NewSnapshot(env.Store)
	tips := snap.Tips(nil, snap.Reverse)

	if len(tips) != 0 {
		t.Errorf("got %d tips, want 0", len(tips))
	}
}

func TestSnapshotReady(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	c, _ := env.Store.Create("Free", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	snap, _ := issue.NewSnapshot(env.Store)
	ready := snap.Ready()
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}

	if !ids[a.ID] {
		t.Error("blocker should be ready")
	}
	if ids[b.ID] {
		t.Error("blocked should NOT be ready")
	}
	if !ids[c.ID] {
		t.Error("free should be ready")
	}
}

func TestSnapshotReadyAfterClose(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(a.ID, "")
	env.CommitIntent("setup")

	snap, _ := issue.NewSnapshot(env.Store)
	ready := snap.Ready()
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}

	if !ids[b.ID] {
		t.Error("blocked should be ready after blocker closed")
	}
}

func TestSnapshotReadyExcludesInProgress(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open", issue.CreateOpts{})
	b, _ := env.Store.Create("WIP", issue.CreateOpts{})
	status := "in_progress"
	env.Store.Update(b.ID, issue.UpdateOpts{Status: &status})
	env.CommitIntent("setup")

	snap, _ := issue.NewSnapshot(env.Store)
	ready := snap.Ready()
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}

	if !ids[a.ID] {
		t.Error("open should be ready")
	}
	if ids[b.ID] {
		t.Error("in_progress should NOT be ready")
	}
}

func TestSnapshotBlocked(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	snap, _ := issue.NewSnapshot(env.Store)
	blocked := snap.Blocked()

	if len(blocked) != 1 {
		t.Fatalf("got %d blocked, want 1", len(blocked))
	}
	if blocked[0].ID != b.ID {
		t.Errorf("blocked ID = %q, want %q", blocked[0].ID, b.ID)
	}
	if len(blocked[0].OpenBlockers) != 1 || blocked[0].OpenBlockers[0] != a.ID {
		t.Errorf("open blockers = %v, want [%s]", blocked[0].OpenBlockers, a.ID)
	}
}

func TestSnapshotBlockedResolved(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(a.ID, "")
	env.CommitIntent("setup")

	snap, _ := issue.NewSnapshot(env.Store)
	blocked := snap.Blocked()

	if len(blocked) != 0 {
		t.Errorf("got %d blocked, want 0 after resolving", len(blocked))
	}
}

func TestSnapshotChildren(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	child1, _ := env.Store.Create("Child 1", issue.CreateOpts{Parent: parent.ID})
	child2, _ := env.Store.Create("Child 2", issue.CreateOpts{Parent: parent.ID})
	env.Store.Create("Orphan", issue.CreateOpts{})
	env.CommitIntent("setup")

	snap, _ := issue.NewSnapshot(env.Store)
	children := snap.Children(parent.ID)

	ids := make([]string, len(children))
	for i, c := range children {
		ids[i] = c.ID
	}
	sort.Strings(ids)

	expected := []string{child1.ID, child2.ID}
	sort.Strings(expected)

	if len(ids) != 2 {
		t.Fatalf("got %d children, want 2", len(ids))
	}
	if ids[0] != expected[0] || ids[1] != expected[1] {
		t.Errorf("children = %v, want %v", ids, expected)
	}
}

func TestSnapshotEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	snap, err := issue.NewSnapshot(env.Store)
	if err != nil {
		t.Fatalf("NewSnapshot: %v", err)
	}
	if len(snap.Issues) != 0 {
		t.Errorf("Issues count = %d, want 0", len(snap.Issues))
	}
	if len(snap.Forward) != 0 {
		t.Errorf("Forward count = %d, want 0", len(snap.Forward))
	}
	if len(snap.Reverse) != 0 {
		t.Errorf("Reverse count = %d, want 0", len(snap.Reverse))
	}
}
