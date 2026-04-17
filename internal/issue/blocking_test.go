package issue_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
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

func TestLinkDirectCycle(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID) // A blocks B
	env.CommitIntent("link a->b")

	err := env.Store.Link(b.ID, a.ID) // B blocks A → cycle
	if err == nil {
		t.Fatal("expected error for direct cycle")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error = %q, want mention of circular", err)
	}
}

func TestLinkDeepCycle(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	c, _ := env.Store.Create("C", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID) // A blocks B
	env.Store.Link(b.ID, c.ID) // B blocks C
	env.CommitIntent("chain a->b->c")

	err := env.Store.Link(c.ID, a.ID) // C blocks A → cycle
	if err == nil {
		t.Fatal("expected error for deep cycle")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error = %q, want mention of circular", err)
	}
}

func TestLinkDiamondNoCycle(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	c, _ := env.Store.Create("C", issue.CreateOpts{})
	d, _ := env.Store.Create("D", issue.CreateOpts{})

	// A→B, A→C, B→D, C→D (diamond, not a cycle)
	if err := env.Store.Link(a.ID, b.ID); err != nil {
		t.Fatalf("Link a->b: %v", err)
	}
	if err := env.Store.Link(a.ID, c.ID); err != nil {
		t.Fatalf("Link a->c: %v", err)
	}
	if err := env.Store.Link(b.ID, d.ID); err != nil {
		t.Fatalf("Link b->d: %v", err)
	}
	if err := env.Store.Link(c.ID, d.ID); err != nil {
		t.Fatalf("Link c->d: %v", err)
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

func TestLinkChildBlocksParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Parent", issue.CreateOpts{})
	env.CommitIntent("create parent")
	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: parent.ID})
	env.CommitIntent("create child")

	err := env.Store.Link(child.ID, parent.ID)
	if err == nil {
		t.Fatal("expected error for child blocking parent")
	}
	if !strings.Contains(err.Error(), "ancestor") {
		t.Errorf("error = %q, want mention of ancestor", err)
	}
}

func TestLinkGrandchildBlocksGrandparent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	gp, _ := env.Store.Create("Grandparent", issue.CreateOpts{})
	env.CommitIntent("create gp")
	parent, _ := env.Store.Create("Parent", issue.CreateOpts{Parent: gp.ID})
	env.CommitIntent("create parent")
	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: parent.ID})
	env.CommitIntent("create child")

	err := env.Store.Link(child.ID, gp.ID)
	if err == nil {
		t.Fatal("expected error for grandchild blocking grandparent")
	}
	if !strings.Contains(err.Error(), "ancestor") {
		t.Errorf("error = %q, want mention of ancestor", err)
	}
}

func TestLinkSiblingDepsAllowed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Parent", issue.CreateOpts{})
	env.CommitIntent("create parent")
	a, _ := env.Store.Create("A", issue.CreateOpts{Parent: parent.ID})
	env.CommitIntent("create a")
	b, _ := env.Store.Create("B", issue.CreateOpts{Parent: parent.ID})
	env.CommitIntent("create b")

	if err := env.Store.Link(a.ID, b.ID); err != nil {
		t.Fatalf("sibling link should be allowed: %v", err)
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

// TestReadySortedByPriority verifies that Ready() returns issues sorted by
// priority (ascending) then creation date.
func TestReadySortedByPriority(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create in reverse priority order so directory order != priority order.
	p3 := 3
	env.Store.Create("Low priority", issue.CreateOpts{Priority: &p3})
	env.CommitIntent("create low")
	p1 := 1
	env.Store.Create("High priority", issue.CreateOpts{Priority: &p1})
	env.CommitIntent("create high")
	p2 := 2
	env.Store.Create("Medium priority", issue.CreateOpts{Priority: &p2})
	env.CommitIntent("create medium")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if len(ready) != 3 {
		t.Fatalf("got %d ready, want 3", len(ready))
	}
	if ready[0].Priority != 1 || ready[1].Priority != 2 || ready[2].Priority != 3 {
		t.Errorf("priorities = [%d, %d, %d], want [1, 2, 3]",
			ready[0].Priority, ready[1].Priority, ready[2].Priority)
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

func TestClosedBlockerSet(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker A", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked B", issue.CreateOpts{})
	c, _ := env.Store.Create("Blocker C", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Link(c.ID, b.ID)
	env.Store.Close(a.ID, "done")
	env.CommitIntent("setup")

	// Re-read B to get updated BlockedBy
	b, _ = env.Store.Get(b.ID)

	set := env.Store.ClosedBlockerSet([]*issue.Issue{b})
	if !set[a.ID] {
		t.Errorf("closed blocker A should be in set")
	}
	if set[c.ID] {
		t.Errorf("open blocker C should NOT be in set")
	}
}

func TestClosedBlockerSetEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("No deps", issue.CreateOpts{})
	env.CommitIntent("setup")

	set := env.Store.ClosedBlockerSet([]*issue.Issue{a})
	if len(set) != 0 {
		t.Errorf("got %d entries, want 0", len(set))
	}
}

func TestClosedBlockerSetDeduplicates(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Shared blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked B", issue.CreateOpts{})
	c, _ := env.Store.Create("Blocked C", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Link(a.ID, c.ID)
	env.Store.Close(a.ID, "done")
	env.CommitIntent("setup")

	b, _ = env.Store.Get(b.ID)
	c, _ = env.Store.Get(c.ID)

	set := env.Store.ClosedBlockerSet([]*issue.Issue{b, c})
	if !set[a.ID] {
		t.Errorf("closed blocker A should be in set")
	}
	if len(set) != 1 {
		t.Errorf("got %d entries, want 1 (deduplicated)", len(set))
	}
}

func TestReadyIncludesExpiredDeferral(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Deferred with expired defer_until
	a, _ := env.Store.Create("Expired deferral", issue.CreateOpts{DeferUntil: "2027-04-01"})
	env.CommitIntent("create " + a.ID)

	// Deferred with future defer_until
	b, _ := env.Store.Create("Future deferral", issue.CreateOpts{DeferUntil: "2027-12-01"})
	env.CommitIntent("create " + b.ID)

	// Regular open issue
	c, _ := env.Store.Create("Open task", issue.CreateOpts{})
	env.CommitIntent("create " + c.ID)

	// Set clock to April 15
	t.Setenv("BW_CLOCK", "2027-04-15T12:00:00Z")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}

	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}

	if !ids[a.ID] {
		t.Error("expired deferral should appear in ready")
	}
	if ids[b.ID] {
		t.Error("future deferral should NOT appear in ready")
	}
	if !ids[c.ID] {
		t.Error("open task should appear in ready")
	}

	// Verify on-disk status is still deferred
	got, _ := env.Store.Get(a.ID)
	if got.Status != "deferred" {
		t.Errorf("on-disk status = %q, want deferred (no write-on-read)", got.Status)
	}
}

func TestDeferralExpiredStartOfDay(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Deferred until today", issue.CreateOpts{DeferUntil: "2027-04-15"})
	env.CommitIntent("create " + a.ID)

	// Set clock to morning of April 15 — should be expired (start-of-day)
	t.Setenv("BW_CLOCK", "2027-04-15T08:00:00Z")

	ready, _ := env.Store.Ready()
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[a.ID] {
		t.Error("date-only deferral should be expired at start of day (April 15)")
	}

	// Day before — should NOT be expired
	t.Setenv("BW_CLOCK", "2027-04-14T23:00:00Z")
	ready2, _ := env.Store.Ready()
	ids2 := make(map[string]bool)
	for _, r := range ready2 {
		ids2[r.ID] = true
	}
	if ids2[a.ID] {
		t.Error("date-only deferral should NOT be expired the day before")
	}
}

func TestExpiredDeferralNoStatusChange(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Will expire", issue.CreateOpts{DeferUntil: "2027-04-01"})
	env.CommitIntent("create " + a.ID)

	t.Setenv("BW_CLOCK", "2027-04-15T12:00:00Z")

	// Query ready (which includes the expired deferral)
	env.Store.Ready()
	// Query list
	env.Store.List(issue.Filter{Statuses: []string{"open", "in_progress"}, IncludeExpiredDeferred: true})

	// Status should still be deferred on disk
	got, _ := env.Store.Get(a.ID)
	if got.Status != "deferred" {
		t.Errorf("status = %q, want deferred (reads must not change status)", got.Status)
	}
}

// --- Subtree-aware Ready/Blocked tests ---

func TestReadySubtreeInternalBlockerSuppressed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	childA, _ := env.Store.Create("Child A", issue.CreateOpts{Parent: epic.ID})
	childB, _ := env.Store.Create("Child B", issue.CreateOpts{Parent: epic.ID})
	env.Store.Link(childA.ID, childB.ID) // A blocks B (internal)
	env.CommitIntent("setup")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[epic.ID] {
		t.Error("epic should be ready (only internal blockers)")
	}

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	for _, bi := range blocked {
		if bi.ID == childB.ID {
			t.Error("child B should be suppressed from blocked (internal blocker)")
		}
	}
}

func TestReadySubtreeExternalBlocker(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	ext, _ := env.Store.Create("External", issue.CreateOpts{})
	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: epic.ID})
	env.Store.Link(ext.ID, child.ID) // external blocks child
	env.CommitIntent("setup")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	readyIDs := make(map[string]bool)
	for _, r := range ready {
		readyIDs[r.ID] = true
	}
	if readyIDs[epic.ID] {
		t.Error("epic should NOT be ready (child has external blocker)")
	}

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	var epicBlocked *issue.BlockedIssue
	for i := range blocked {
		if blocked[i].ID == epic.ID {
			epicBlocked = &blocked[i]
			break
		}
	}
	if epicBlocked == nil {
		t.Fatal("epic should appear in blocked list")
	}
	found := false
	for _, ob := range epicBlocked.OpenBlockers {
		if ob == ext.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("epic open blockers = %v, should contain %s", epicBlocked.OpenBlockers, ext.ID)
	}
}

func TestReadySubtreeMixedBlockers(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	childA, _ := env.Store.Create("Child A", issue.CreateOpts{Parent: epic.ID})
	childB, _ := env.Store.Create("Child B", issue.CreateOpts{Parent: epic.ID})
	childC, _ := env.Store.Create("Child C", issue.CreateOpts{Parent: epic.ID})
	ext, _ := env.Store.Create("External", issue.CreateOpts{})
	env.Store.Link(childA.ID, childB.ID) // internal
	env.Store.Link(ext.ID, childC.ID)    // external
	env.CommitIntent("setup")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	readyIDs := make(map[string]bool)
	for _, r := range ready {
		readyIDs[r.ID] = true
	}
	if readyIDs[epic.ID] {
		t.Error("epic should NOT be ready (has external blocker)")
	}

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	var epicBlocked *issue.BlockedIssue
	for i := range blocked {
		if blocked[i].ID == epic.ID {
			epicBlocked = &blocked[i]
			break
		}
	}
	if epicBlocked == nil {
		t.Fatal("epic should appear in blocked list")
	}
	hasExt := false
	for _, ob := range epicBlocked.OpenBlockers {
		if ob == ext.ID {
			hasExt = true
		}
		if ob == childA.ID {
			t.Errorf("internal blocker %s should not appear in epic's open blockers", childA.ID)
		}
	}
	if !hasExt {
		t.Errorf("epic open blockers should contain external %s", ext.ID)
	}
}

func TestReadySubtreeGrandchild(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: epic.ID})
	grandchild, _ := env.Store.Create("Grandchild", issue.CreateOpts{Parent: child.ID})
	ext, _ := env.Store.Create("External", issue.CreateOpts{})
	env.Store.Link(ext.ID, grandchild.ID)
	env.CommitIntent("setup")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	readyIDs := make(map[string]bool)
	for _, r := range ready {
		readyIDs[r.ID] = true
	}
	if readyIDs[epic.ID] {
		t.Error("epic should NOT be ready (grandchild has external blocker)")
	}

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	var epicBlocked *issue.BlockedIssue
	for i := range blocked {
		if blocked[i].ID == epic.ID {
			epicBlocked = &blocked[i]
			break
		}
	}
	if epicBlocked == nil {
		t.Fatal("epic should appear in blocked list")
	}
	found := false
	for _, ob := range epicBlocked.OpenBlockers {
		if ob == ext.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("external blocker should bubble up to epic, got %v", epicBlocked.OpenBlockers)
	}
	// Grandchild should be suppressed from blocked
	for _, bi := range blocked {
		if bi.ID == grandchild.ID {
			t.Error("grandchild should be suppressed from blocked list")
		}
	}
}

func TestBlockedSubtreeParentDirectAndChildExternal(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	extA, _ := env.Store.Create("Ext A", issue.CreateOpts{})
	extB, _ := env.Store.Create("Ext B", issue.CreateOpts{})
	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: epic.ID})
	env.Store.Link(extA.ID, epic.ID)  // epic directly blocked
	env.Store.Link(extB.ID, child.ID) // child externally blocked
	env.CommitIntent("setup")

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	var epicBlocked *issue.BlockedIssue
	for i := range blocked {
		if blocked[i].ID == epic.ID {
			epicBlocked = &blocked[i]
			break
		}
	}
	if epicBlocked == nil {
		t.Fatal("epic should appear in blocked list")
	}
	hasA, hasB := false, false
	for _, ob := range epicBlocked.OpenBlockers {
		if ob == extA.ID {
			hasA = true
		}
		if ob == extB.ID {
			hasB = true
		}
	}
	if !hasA {
		t.Errorf("epic should have direct blocker %s", extA.ID)
	}
	if !hasB {
		t.Errorf("epic should have child's external blocker %s", extB.ID)
	}
}

func TestReadySubtreeBlockedRootDescendantsExcluded(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Epic h3p with two children; external ticket blocks one child,
	// making the whole subtree blocked. The unblocked child (h3p.2)
	// should NOT appear in ready because its root is blocked.
	epic, _ := env.Store.Create("Epic h3p", issue.CreateOpts{Type: "epic"})
	child1, _ := env.Store.Create("Child h3p.1", issue.CreateOpts{Parent: epic.ID})
	child2, _ := env.Store.Create("Child h3p.2", issue.CreateOpts{Parent: epic.ID})
	ext, _ := env.Store.Create("External blocker", issue.CreateOpts{})
	env.Store.Link(ext.ID, child1.ID) // external blocks child1 → epic is blocked
	env.CommitIntent("setup")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	readyIDs := make(map[string]bool)
	for _, r := range ready {
		readyIDs[r.ID] = true
	}

	if readyIDs[epic.ID] {
		t.Error("epic should NOT be in ready (child has external blocker)")
	}
	if readyIDs[child1.ID] {
		t.Error("child1 should NOT be in ready (directly blocked by external)")
	}
	if readyIDs[child2.ID] {
		t.Error("child2 should NOT be in ready (descendant of blocked root)")
	}

	_ = child2 // suppress unused warning
}

func TestReadySubtreeReadyRootDescendantsExcluded(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Epic with two children linked internally. No external blockers,
	// so the root is ready — but children are part of the root ticket
	// and should not appear individually in the ready list.
	epic, _ := env.Store.Create("Ready epic", issue.CreateOpts{Type: "epic"})
	childA, _ := env.Store.Create("Child A", issue.CreateOpts{Parent: epic.ID})
	childB, _ := env.Store.Create("Child B", issue.CreateOpts{Parent: epic.ID})
	env.Store.Link(childA.ID, childB.ID) // internal only
	env.CommitIntent("setup")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	readyIDs := make(map[string]bool)
	for _, r := range ready {
		readyIDs[r.ID] = true
	}

	if !readyIDs[epic.ID] {
		t.Error("epic should be in ready (only internal blockers)")
	}
	if readyIDs[childA.ID] {
		t.Error("childA should NOT be in ready (descendant, part of root ticket)")
	}
	if readyIDs[childB.ID] {
		t.Error("childB should NOT be in ready (descendant, part of root ticket)")
	}
}

// An in_progress epic is claimed work — its ready children should surface in
// Ready(), since they're the next actionable step inside the epic.
func TestReadySurfacesChildrenOfInProgressRoot(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: epic.ID})
	env.CommitIntent("setup")

	if _, err := env.Store.Start(epic.ID, ""); err != nil {
		t.Fatalf("Start: %v", err)
	}
	env.CommitIntent("start " + epic.ID)

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[child.ID] {
		t.Errorf("child should surface in ready when root epic is in_progress, got %v", ids)
	}
	if ids[epic.ID] {
		t.Errorf("in_progress epic should not appear in ready, got %v", ids)
	}
}

// When the epic is in_progress AND the epic-child is also in_progress, Ready()
// should drill past both and surface the open grandchild.
func TestReadyDrillsPastInProgressIntermediateNodes(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: epic.ID})
	grandchild, _ := env.Store.Create("Grandchild", issue.CreateOpts{Parent: child.ID})
	env.CommitIntent("setup")

	if _, err := env.Store.Start(epic.ID, ""); err != nil {
		t.Fatalf("Start epic: %v", err)
	}
	if _, err := env.Store.Start(child.ID, ""); err != nil {
		t.Fatalf("Start child: %v", err)
	}
	env.CommitIntent("start chain")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[grandchild.ID] {
		t.Errorf("grandchild should surface in ready, got %v", ids)
	}
}

// When an intermediate node is open (not started), it represents the unit of
// work and surfaces — its descendants should stay hidden.
func TestReadyStopsAtOpenIntermediateNode(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: epic.ID})
	grandchild, _ := env.Store.Create("Grandchild", issue.CreateOpts{Parent: child.ID})
	env.CommitIntent("setup")

	if _, err := env.Store.Start(epic.ID, ""); err != nil {
		t.Fatalf("Start: %v", err)
	}
	env.CommitIntent("start " + epic.ID)

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[child.ID] {
		t.Errorf("open child should surface, got %v", ids)
	}
	if ids[grandchild.ID] {
		t.Errorf("grandchild should be hidden under open child, got %v", ids)
	}
}

// External blockers should bubble to the display root (which can be a deeper
// node than the top-level epic when the epic is in_progress).
func TestReadyExternalBlockerBubblesToDisplayRoot(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: epic.ID})
	grandchild, _ := env.Store.Create("Grandchild", issue.CreateOpts{Parent: child.ID})
	ext, _ := env.Store.Create("External", issue.CreateOpts{})
	env.Store.Link(ext.ID, grandchild.ID)
	env.CommitIntent("setup")

	if _, err := env.Store.Start(epic.ID, ""); err != nil {
		t.Fatalf("Start: %v", err)
	}
	env.CommitIntent("start " + epic.ID)

	// Epic is in_progress → child is the display root. Child has a descendant
	// (grandchild) with an external blocker, so child should appear as blocked,
	// not ready.
	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	for _, r := range ready {
		if r.ID == child.ID {
			t.Errorf("child should NOT be ready — grandchild has external blocker")
		}
	}

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	var childBlocked *issue.BlockedIssue
	for i := range blocked {
		if blocked[i].ID == child.ID {
			childBlocked = &blocked[i]
			break
		}
	}
	if childBlocked == nil {
		t.Fatal("child should appear in blocked list (external blocker bubbles from grandchild)")
	}
	found := false
	for _, ob := range childBlocked.OpenBlockers {
		if ob == ext.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("child blockers = %v, should contain %s", childBlocked.OpenBlockers, ext.ID)
	}
}

func TestHiddenBlockerSetIncludesDescendants(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Epic with children that block the parent via explicit links.
	// HiddenBlockerSet should include children so they're stripped
	// from the dep annotation in the ready display.
	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	childA, _ := env.Store.Create("Child A", issue.CreateOpts{Parent: epic.ID})
	childB, _ := env.Store.Create("Child B", issue.CreateOpts{Parent: epic.ID})
	env.Store.Link(childA.ID, epic.ID)
	env.Store.Link(childB.ID, epic.ID)
	env.CommitIntent("setup")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}

	hidden := env.Store.HiddenBlockerSet(ready)

	if !hidden[childA.ID] {
		t.Errorf("childA (%s) should be in hidden set (internal blocker)", childA.ID)
	}
	if !hidden[childB.ID] {
		t.Errorf("childB (%s) should be in hidden set (internal blocker)", childB.ID)
	}
	if hidden[epic.ID] {
		t.Error("epic should NOT be in hidden set (it's the root)")
	}
}

func TestReadyScoped(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create an epic with three children: two ready, one blocked by the other.
	epic, _ := env.Store.Create("Epic", issue.CreateOpts{})
	childA, _ := env.Store.Create("Child A", issue.CreateOpts{Parent: epic.ID})
	childB, _ := env.Store.Create("Child B", issue.CreateOpts{Parent: epic.ID})
	childC, _ := env.Store.Create("Child C", issue.CreateOpts{Parent: epic.ID})
	// childC is blocked by childA (both within the subtree)
	env.Store.Link(childA.ID, childC.ID)
	env.CommitIntent("setup")

	ready, err := env.Store.ReadyScoped(epic.ID)
	if err != nil {
		t.Fatalf("ReadyScoped: %v", err)
	}

	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}

	// Parent should NOT appear
	if ids[epic.ID] {
		t.Error("parent epic should NOT appear in scoped ready list")
	}
	// childA and childB are ready (no blockers)
	if !ids[childA.ID] {
		t.Errorf("childA (%s) should be ready", childA.ID)
	}
	if !ids[childB.ID] {
		t.Errorf("childB (%s) should be ready", childB.ID)
	}
	// childC is blocked by childA
	if ids[childC.ID] {
		t.Errorf("childC (%s) should NOT be ready (blocked by childA)", childC.ID)
	}
}

func TestReadyScopedExcludesClosedChildren(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{})
	childA, _ := env.Store.Create("Child A", issue.CreateOpts{Parent: epic.ID})
	childB, _ := env.Store.Create("Child B", issue.CreateOpts{Parent: epic.ID})
	env.Store.Close(childA.ID, "done")
	env.CommitIntent("setup")

	ready, err := env.Store.ReadyScoped(epic.ID)
	if err != nil {
		t.Fatalf("ReadyScoped: %v", err)
	}

	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}

	if ids[childA.ID] {
		t.Errorf("closed childA should NOT appear")
	}
	if !ids[childB.ID] {
		t.Errorf("open childB should appear")
	}
}

func TestReadyScopedBlockedByExternalUnblocks(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{})
	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: epic.ID})
	external, _ := env.Store.Create("External blocker", issue.CreateOpts{})
	env.Store.Link(external.ID, child.ID)
	env.CommitIntent("setup")

	// child is blocked by external, so not ready
	ready, err := env.Store.ReadyScoped(epic.ID)
	if err != nil {
		t.Fatalf("ReadyScoped: %v", err)
	}
	if len(ready) != 0 {
		t.Errorf("expected 0 ready, got %d", len(ready))
	}

	// Close the external blocker -> child becomes ready
	env.Store.Close(external.ID, "done")
	env.CommitIntent("close external")

	ready, err = env.Store.ReadyScoped(epic.ID)
	if err != nil {
		t.Fatalf("ReadyScoped after close: %v", err)
	}
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[child.ID] {
		t.Errorf("child should be ready after external blocker closed")
	}
}

func TestReadyScopedRespectsDeferred(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{})
	env.Store.Create("Deferred child", issue.CreateOpts{Parent: epic.ID, DeferUntil: "2099-01-01"})
	childB, _ := env.Store.Create("Open child", issue.CreateOpts{Parent: epic.ID})
	env.CommitIntent("setup")

	ready, err := env.Store.ReadyScoped(epic.ID)
	if err != nil {
		t.Fatalf("ReadyScoped: %v", err)
	}

	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}

	if len(ready) != 1 {
		t.Errorf("expected 1 ready, got %d", len(ready))
	}
	if !ids[childB.ID] {
		t.Errorf("open child should be ready")
	}
}

func TestReadyScopedNonExistentParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.ReadyScoped("test-zzzz")
	if err == nil {
		t.Error("expected error for non-existent parent")
	}
}

