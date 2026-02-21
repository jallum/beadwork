package issue_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestLink(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.CommitIntent("create " + b.ID)

	if err := env.Store.Link(a.ID, b.ID); err != nil {
		t.Fatalf("Link: %v", err)
	}
	env.CommitIntent("link " + a.ID + " blocks " + b.ID)

	// Check marker file
	if !env.MarkerExists(filepath.Join("blocks", a.ID, b.ID)) {
		t.Error("blocks marker missing")
	}

	// Check JSON updated on both sides
	aGot, _ := env.Store.Get(a.ID)
	if len(aGot.Blocks) != 1 || aGot.Blocks[0] != b.ID {
		t.Errorf("blocker.Blocks = %v, want [%s]", aGot.Blocks, b.ID)
	}
	bGot, _ := env.Store.Get(b.ID)
	if len(bGot.BlockedBy) != 1 || bGot.BlockedBy[0] != a.ID {
		t.Errorf("blocked.BlockedBy = %v, want [%s]", bGot.BlockedBy, a.ID)
	}
}

func TestUnlink(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("link " + a.ID + " blocks " + b.ID)

	if err := env.Store.Unlink(a.ID, b.ID); err != nil {
		t.Fatalf("Unlink: %v", err)
	}
	env.CommitIntent("unlink " + a.ID + " blocks " + b.ID)

	// Marker should be gone
	if env.MarkerExists(filepath.Join("blocks", a.ID, b.ID)) {
		t.Error("blocks marker should be gone")
	}

	// JSON updated
	aGot, _ := env.Store.Get(a.ID)
	if len(aGot.Blocks) != 0 {
		t.Errorf("blocker.Blocks = %v, want empty", aGot.Blocks)
	}
	bGot, _ := env.Store.Get(b.ID)
	if len(bGot.BlockedBy) != 0 {
		t.Errorf("blocked.BlockedBy = %v, want empty", bGot.BlockedBy)
	}
}

func TestLinkSelfBlocking(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Self", issue.CreateOpts{})
	err := env.Store.Link(a.ID, a.ID)
	if err == nil {
		t.Error("expected error for self-blocking")
	}
}

func TestReady(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Blocked task", issue.CreateOpts{})
	env.CommitIntent("create " + b.ID)
	c, _ := env.Store.Create("Free task", issue.CreateOpts{})
	env.CommitIntent("create " + c.ID)

	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("link " + a.ID + " blocks " + b.ID)

	// B is blocked, A and C are ready
	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[a.ID] {
		t.Error("blocker should be ready")
	}
	if ids[b.ID] {
		t.Error("blocked task should NOT be ready")
	}
	if !ids[c.ID] {
		t.Error("free task should be ready")
	}

	// Close the blocker -> B becomes ready
	env.Store.Close(a.ID, "")
	env.CommitIntent("close " + a.ID)

	ready, _ = env.Store.Ready()
	ids = make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[b.ID] {
		t.Error("blocked task should be ready after blocker closed")
	}
}

func TestReadyJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Unblocked task", issue.CreateOpts{Priority: intPtr(2), Type: "task"})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Blocked task", issue.CreateOpts{Priority: intPtr(3), Type: "task"})
	env.CommitIntent("create " + b.ID)

	// a blocks b
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("link " + a.ID + " blocks " + b.ID)

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}

	data, err := json.MarshalIndent(ready, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}

	var parsed []issue.Issue
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Only the unblocked task should be ready
	if len(parsed) != 1 {
		t.Fatalf("got %d ready issues, want 1", len(parsed))
	}
	assertJSONFields(t, parsed[0], "open", 2, "task", "Unblocked task", "")

	// Verify blocks/blocked_by are present in JSON
	if len(parsed[0].Blocks) != 1 || parsed[0].Blocks[0] != b.ID {
		t.Errorf("blocks = %v, want [%s]", parsed[0].Blocks, b.ID)
	}
}

func TestLinkIdempotent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)

	// Link again — should be a noop, not create duplicate entries
	if err := env.Store.Link(a.ID, b.ID); err != nil {
		t.Fatalf("second Link: %v", err)
	}

	aGot, _ := env.Store.Get(a.ID)
	if len(aGot.Blocks) != 1 {
		t.Errorf("blocks = %v, want exactly 1 entry", aGot.Blocks)
	}
	bGot, _ := env.Store.Get(b.ID)
	if len(bGot.BlockedBy) != 1 {
		t.Errorf("blockedBy = %v, want exactly 1 entry", bGot.BlockedBy)
	}
}

func TestUnlinkIdempotent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Unlink(a.ID, b.ID)

	// Unlink again — should be a noop
	if err := env.Store.Unlink(a.ID, b.ID); err != nil {
		t.Fatalf("second Unlink: %v", err)
	}

	aGot, _ := env.Store.Get(a.ID)
	if len(aGot.Blocks) != 0 {
		t.Errorf("blocks = %v, want empty", aGot.Blocks)
	}
}

func TestLinkNonExistentBlocker(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})

	err := env.Store.Link("test-zzzz", b.ID)
	if err == nil {
		t.Error("expected error for non-existent blocker")
	}
}

func TestLinkNonExistentBlocked(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})

	err := env.Store.Link(a.ID, "test-zzzz")
	if err == nil {
		t.Error("expected error for non-existent blocked")
	}
}

func TestUnlinkNonExistentBlocker(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	err := env.Store.Unlink("test-zzzz", b.ID)
	if err == nil {
		t.Error("expected error for non-existent blocker")
	}
}

func TestUnlinkNonExistentBlocked(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	err := env.Store.Unlink(a.ID, "test-zzzz")
	if err == nil {
		t.Error("expected error for non-existent blocked")
	}
}

func TestBlockedSingle(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("link")

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
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

func TestBlockedMultipleBlockers(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker A", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocker B", issue.CreateOpts{})
	c, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, c.ID)
	env.Store.Link(b.ID, c.ID)
	env.Store.Close(b.ID, "") // close one blocker
	env.CommitIntent("setup")

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	if len(blocked) != 1 {
		t.Fatalf("got %d blocked, want 1", len(blocked))
	}
	// Only the open blocker should appear
	if len(blocked[0].OpenBlockers) != 1 || blocked[0].OpenBlockers[0] != a.ID {
		t.Errorf("open blockers = %v, want [%s]", blocked[0].OpenBlockers, a.ID)
	}
}

func TestBlockedResolves(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("link")

	// Close the blocker
	env.Store.Close(a.ID, "")
	env.CommitIntent("close")

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	if len(blocked) != 0 {
		t.Errorf("got %d blocked, want 0 after resolving blocker", len(blocked))
	}
}

func TestBlockedNoBlockers(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("No deps", issue.CreateOpts{})
	env.CommitIntent("create")

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	if len(blocked) != 0 {
		t.Errorf("got %d blocked, want 0", len(blocked))
	}
}

func TestBlockedClosedIssueExcluded(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked and closed", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(b.ID, "") // close the blocked issue itself
	env.CommitIntent("setup")

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	// Closed issues shouldn't appear even if they have open blockers
	for _, bi := range blocked {
		if bi.ID == b.ID {
			t.Error("closed issue should not appear in blocked list")
		}
	}
}

func TestReadyExcludesInProgress(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open task", issue.CreateOpts{})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("In-progress task", issue.CreateOpts{})
	env.CommitIntent("create " + b.ID)

	// Move B to in_progress
	status := "in_progress"
	env.Store.Update(b.ID, issue.UpdateOpts{Status: &status})
	env.CommitIntent("update " + b.ID)

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}

	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[a.ID] {
		t.Error("open task should be ready")
	}
	if ids[b.ID] {
		t.Error("in_progress task should NOT be ready")
	}
}

func TestReadyExcludesDeferred(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open task", issue.CreateOpts{})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Deferred task", issue.CreateOpts{})
	env.CommitIntent("create " + b.ID)

	// Move B to deferred
	status := "deferred"
	deferDate := "2027-06-01"
	env.Store.Update(b.ID, issue.UpdateOpts{Status: &status, DeferUntil: &deferDate})
	env.CommitIntent("defer " + b.ID)

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}

	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[a.ID] {
		t.Error("open task should be ready")
	}
	if ids[b.ID] {
		t.Error("deferred task should NOT be ready")
	}
}

func TestUpdatedAtOnLink(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.CommitIntent("create issues")

	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("link")

	gotA, _ := env.Store.Get(a.ID)
	gotB, _ := env.Store.Get(b.ID)
	if gotA.UpdatedAt == "" {
		t.Error("blocker updated_at should be set after link")
	}
	if gotB.UpdatedAt == "" {
		t.Error("blocked updated_at should be set after link")
	}
}

// --- Statuses filter (bw-17q) ---

func TestNewlyUnblockedSingle(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(a.ID, "")
	env.CommitIntent("close")

	unblocked, err := env.Store.NewlyUnblocked(a.ID)
	if err != nil {
		t.Fatalf("NewlyUnblocked: %v", err)
	}
	if len(unblocked) != 1 || unblocked[0].ID != b.ID {
		t.Errorf("got %d unblocked, want [%s]", len(unblocked), b.ID)
	}
}

func TestNewlyUnblockedMultipleBlockers(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker A", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	c, _ := env.Store.Create("Blocker C", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Link(c.ID, b.ID)
	env.Store.Close(a.ID, "")
	env.CommitIntent("close")

	unblocked, err := env.Store.NewlyUnblocked(a.ID)
	if err != nil {
		t.Fatalf("NewlyUnblocked: %v", err)
	}
	if len(unblocked) != 0 {
		t.Errorf("got %d unblocked, want 0 (C still open)", len(unblocked))
	}
}

func TestNewlyUnblockedNoBlocks(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("No deps", issue.CreateOpts{})
	env.Store.Close(a.ID, "")
	env.CommitIntent("close")

	unblocked, err := env.Store.NewlyUnblocked(a.ID)
	if err != nil {
		t.Fatalf("NewlyUnblocked: %v", err)
	}
	if len(unblocked) != 0 {
		t.Errorf("got %d unblocked, want 0", len(unblocked))
	}
}

func TestNewlyUnblockedSkipsClosed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Already closed", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(b.ID, "")
	env.Store.Close(a.ID, "")
	env.CommitIntent("close")

	unblocked, err := env.Store.NewlyUnblocked(a.ID)
	if err != nil {
		t.Fatalf("NewlyUnblocked: %v", err)
	}
	if len(unblocked) != 0 {
		t.Errorf("got %d unblocked, want 0 (B is closed)", len(unblocked))
	}
}

func TestNewlyUnblockedMultiple(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked B", issue.CreateOpts{})
	c, _ := env.Store.Create("Blocked C", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Link(a.ID, c.ID)
	env.Store.Close(a.ID, "")
	env.CommitIntent("close")

	unblocked, err := env.Store.NewlyUnblocked(a.ID)
	if err != nil {
		t.Fatalf("NewlyUnblocked: %v", err)
	}
	if len(unblocked) != 2 {
		t.Errorf("got %d unblocked, want 2", len(unblocked))
	}
}

// --- LoadEdges tests ---

func TestLoadEdges(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	c, _ := env.Store.Create("C", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID) // a blocks b
	env.Store.Link(b.ID, c.ID) // b blocks c
	env.CommitIntent("setup")

	fwd, rev := env.Store.LoadEdges()

	// Forward: a→[b], b→[c]
	if len(fwd[a.ID]) != 1 || fwd[a.ID][0] != b.ID {
		t.Errorf("forward[a] = %v, want [%s]", fwd[a.ID], b.ID)
	}
	if len(fwd[b.ID]) != 1 || fwd[b.ID][0] != c.ID {
		t.Errorf("forward[b] = %v, want [%s]", fwd[b.ID], c.ID)
	}

	// Reverse: b→[a], c→[b]
	if len(rev[b.ID]) != 1 || rev[b.ID][0] != a.ID {
		t.Errorf("reverse[b] = %v, want [%s]", rev[b.ID], a.ID)
	}
	if len(rev[c.ID]) != 1 || rev[c.ID][0] != b.ID {
		t.Errorf("reverse[c] = %v, want [%s]", rev[c.ID], b.ID)
	}

	// No edges from/to c in forward, no edges from/to a in reverse
	if len(fwd[c.ID]) != 0 {
		t.Errorf("forward[c] should be empty, got %v", fwd[c.ID])
	}
	if len(rev[a.ID]) != 0 {
		t.Errorf("reverse[a] should be empty, got %v", rev[a.ID])
	}
}

// --- Tips tests ---

func TestTipsSimpleChain(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// A blocked by B blocked by C
	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	c, _ := env.Store.Create("C", issue.CreateOpts{})
	env.Store.Link(c.ID, b.ID) // c blocks b
	env.Store.Link(b.ID, a.ID) // b blocks a
	env.CommitIntent("setup")

	// Re-read a to get updated BlockedBy
	a, _ = env.Store.Get(a.ID)

	// Walk reverse edges (blocked_by direction) from A's blockers
	_, rev := env.Store.LoadEdges()
	tips, err := env.Store.Tips(a.BlockedBy, rev)
	if err != nil {
		t.Fatalf("Tips: %v", err)
	}

	// Should find C (the leaf), not B
	if len(tips) != 1 {
		t.Fatalf("got %d tips, want 1", len(tips))
	}
	if tips[0].ID != c.ID {
		t.Errorf("tip = %s, want %s", tips[0].ID, c.ID)
	}
}

func TestTipsDiamond(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// A blocked by B and C, both blocked by D
	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	c, _ := env.Store.Create("C", issue.CreateOpts{})
	d, _ := env.Store.Create("D", issue.CreateOpts{})
	env.Store.Link(b.ID, a.ID) // b blocks a
	env.Store.Link(c.ID, a.ID) // c blocks a
	env.Store.Link(d.ID, b.ID) // d blocks b
	env.Store.Link(d.ID, c.ID) // d blocks c
	env.CommitIntent("setup")

	// Reload a to get updated BlockedBy
	a, _ = env.Store.Get(a.ID)

	_, rev := env.Store.LoadEdges()
	tips, err := env.Store.Tips(a.BlockedBy, rev)
	if err != nil {
		t.Fatalf("Tips: %v", err)
	}

	// Should find D (deduplicated)
	if len(tips) != 1 {
		t.Fatalf("got %d tips, want 1", len(tips))
	}
	if tips[0].ID != d.ID {
		t.Errorf("tip = %s, want %s", tips[0].ID, d.ID)
	}
}

func TestTipsClosedIntermediary(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// A blocked by B blocked by C, B is closed
	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	c, _ := env.Store.Create("C", issue.CreateOpts{})
	env.Store.Link(c.ID, b.ID)
	env.Store.Link(b.ID, a.ID)
	env.Store.Close(b.ID, "done")
	env.CommitIntent("setup")

	// Re-read a to get updated BlockedBy
	a, _ = env.Store.Get(a.ID)

	_, rev := env.Store.LoadEdges()
	tips, err := env.Store.Tips(a.BlockedBy, rev)
	if err != nil {
		t.Fatalf("Tips: %v", err)
	}

	// Should walk through closed B and find C
	if len(tips) != 1 {
		t.Fatalf("got %d tips, want 1", len(tips))
	}
	if tips[0].ID != c.ID {
		t.Errorf("tip = %s, want %s", tips[0].ID, c.ID)
	}
}

func TestTipsNoBlockers(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("A", issue.CreateOpts{})
	env.CommitIntent("setup")

	_, rev := env.Store.LoadEdges()
	tips, err := env.Store.Tips(nil, rev)
	if err != nil {
		t.Fatalf("Tips: %v", err)
	}

	if len(tips) != 0 {
		t.Errorf("got %d tips, want 0", len(tips))
	}
}

func TestTipsMultipleRoots(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Two independent chains: A←B and X←Y
	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	x, _ := env.Store.Create("X", issue.CreateOpts{})
	y, _ := env.Store.Create("Y", issue.CreateOpts{})
	env.Store.Link(b.ID, a.ID)
	env.Store.Link(y.ID, x.ID)
	env.CommitIntent("setup")

	// Re-read to get updated BlockedBy
	a, _ = env.Store.Get(a.ID)
	x, _ = env.Store.Get(x.ID)

	_, rev := env.Store.LoadEdges()
	tips, err := env.Store.Tips([]string{a.BlockedBy[0], x.BlockedBy[0]}, rev)
	if err != nil {
		t.Fatalf("Tips: %v", err)
	}

	// Should find B and Y
	if len(tips) != 2 {
		t.Fatalf("got %d tips, want 2", len(tips))
	}
	ids := map[string]bool{tips[0].ID: true, tips[1].ID: true}
	if !ids[b.ID] || !ids[y.ID] {
		t.Errorf("tips = %v, want {%s, %s}", ids, b.ID, y.ID)
	}
}

// TestReadyTipsChain verifies that in A←B←C (all open), only C (the tip) is ready.
func TestReadyTipsChain(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Target", issue.CreateOpts{})
	b, _ := env.Store.Create("Middle", issue.CreateOpts{})
	c, _ := env.Store.Create("Leaf", issue.CreateOpts{})
	env.Store.Link(c.ID, b.ID) // C blocks B
	env.Store.Link(b.ID, a.ID) // B blocks A
	env.CommitIntent("setup chain")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[c.ID] {
		t.Error("leaf C should be ready (no blockers)")
	}
	if ids[b.ID] {
		t.Error("middle B should NOT be ready (blocked by C)")
	}
	if ids[a.ID] {
		t.Error("target A should NOT be ready (blocked by B)")
	}
}

// TestReadyBlockerIsActionable verifies that a blocker with no blockers appears in ready.
func TestReadyBlockerIsActionable(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[a.ID] {
		t.Error("blocker A should be ready (it has no blockers itself)")
	}
	if ids[b.ID] {
		t.Error("blocked B should NOT be ready")
	}
}

// TestReadyChainPartialClose verifies that closing a leaf promotes the next in line.
func TestReadyChainPartialClose(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Target", issue.CreateOpts{})
	b, _ := env.Store.Create("Middle", issue.CreateOpts{})
	c, _ := env.Store.Create("Leaf", issue.CreateOpts{})
	env.Store.Link(c.ID, b.ID)
	env.Store.Link(b.ID, a.ID)
	env.CommitIntent("setup chain")

	// Close C → B should become ready
	env.Store.Close(c.ID, "")
	env.CommitIntent("close leaf")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[b.ID] {
		t.Error("middle B should be ready after leaf C closed")
	}
	if ids[a.ID] {
		t.Error("target A should NOT be ready (still blocked by B)")
	}
}

