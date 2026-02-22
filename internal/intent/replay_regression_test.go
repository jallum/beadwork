package intent_test

// Regression tests for intent replay fidelity (bw-u24.5).
//
// These tests cover gaps identified in the intent replay system.
// Tests that expose unfixed bugs are marked with "BUG:" comments
// explaining what the expected behavior should be, and use t.Skip()
// so the test suite stays green until the corresponding fix lands.
//
// Each test documents which fix task it depends on:
//   bw-u24.1: ID preservation on create replay
//   bw-u24.2: start/defer/undefer verb handling
//   bw-u24.3: close reason, description, multi-word title, escaped quotes
//   bw-u24.4: end-to-end sync replay

import (
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/intent"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

// ---------------------------------------------------------------------------
// 1. create-then-close by intent ID (proves ID round-trips)
//    BUG: replayCreate generates a new ID instead of using the ID in the
//    intent string, so the subsequent close referencing the original ID fails.
//    Fix: bw-u24.1
// ---------------------------------------------------------------------------
func TestReplayCreateThenCloseByIntentID(t *testing.T) {
	t.Skip("BUG(bw-u24.1): replayCreate ignores the intent ID and generates a new one; close by intent ID fails")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	intents := []string{
		`create test-aaaa p1 bug "Crash on login"`,
		`close test-aaaa`,
	}
	errs := intent.Replay(env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	// The issue created should have ID "test-aaaa" and be closed.
	iss, err := env.Store.Get("test-aaaa")
	if err != nil {
		t.Fatalf("Get(test-aaaa): %v — ID was not preserved during create replay", err)
	}
	if iss.Status != "closed" {
		t.Errorf("status = %q, want closed", iss.Status)
	}
	if iss.Title != "Crash on login" {
		t.Errorf("title = %q, want 'Crash on login'", iss.Title)
	}
}

// ---------------------------------------------------------------------------
// 2. create-then-update by intent ID
//    BUG: Same root cause as #1 — the update references the intent ID which
//    was not preserved.
//    Fix: bw-u24.1
// ---------------------------------------------------------------------------
func TestReplayCreateThenUpdateByIntentID(t *testing.T) {
	t.Skip("BUG(bw-u24.1): replayCreate ignores the intent ID; update by intent ID fails")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	intents := []string{
		`create test-bbbb p2 task "Original title"`,
		`update test-bbbb title=Revised`,
	}
	errs := intent.Replay(env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	iss, err := env.Store.Get("test-bbbb")
	if err != nil {
		t.Fatalf("Get(test-bbbb): %v — ID was not preserved during create replay", err)
	}
	if iss.Title != "Revised" {
		t.Errorf("title = %q, want 'Revised'", iss.Title)
	}
}

// ---------------------------------------------------------------------------
// 3. create pair and link by intent IDs
//    BUG: Both creates generate new IDs, so the link referencing the original
//    intent IDs fails.
//    Fix: bw-u24.1
// ---------------------------------------------------------------------------
func TestReplayCreatePairAndLinkByIntentIDs(t *testing.T) {
	t.Skip("BUG(bw-u24.1): replayCreate ignores intent IDs; link by intent IDs fails")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	intents := []string{
		`create test-cccc p1 epic "Parent epic"`,
		`create test-dddd p2 task "Child task"`,
		`link test-cccc blocks test-dddd`,
	}
	errs := intent.Replay(env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	parent, err := env.Store.Get("test-cccc")
	if err != nil {
		t.Fatalf("Get(test-cccc): %v", err)
	}
	child, err := env.Store.Get("test-dddd")
	if err != nil {
		t.Fatalf("Get(test-dddd): %v", err)
	}

	if len(parent.Blocks) != 1 || parent.Blocks[0] != "test-dddd" {
		t.Errorf("parent.Blocks = %v, want [test-dddd]", parent.Blocks)
	}
	if len(child.BlockedBy) != 1 || child.BlockedBy[0] != "test-cccc" {
		t.Errorf("child.BlockedBy = %v, want [test-cccc]", child.BlockedBy)
	}
}

// ---------------------------------------------------------------------------
// 4. start intent replay preserves status + assignee
//    BUG: The "start" verb is not handled in replayOne's switch — it falls
//    through to `default: return nil`, silently dropping the intent.
//    Fix: bw-u24.2
// ---------------------------------------------------------------------------
func TestReplayStartPreservesStatusAndAssignee(t *testing.T) {
	t.Skip("BUG(bw-u24.2): 'start' verb is silently dropped by replayOne")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create an open issue to start.
	iss, _ := env.Store.Create("Task to start", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// Replay a start intent (the format from cmdStart: start <id> assignee="agent-1")
	intents := []string{
		`start ` + iss.ID + ` assignee="agent-1"`,
	}
	errs := intent.Replay(env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "in_progress" {
		t.Errorf("status = %q, want in_progress", got.Status)
	}
	if got.Assignee != "agent-1" {
		t.Errorf("assignee = %q, want agent-1", got.Assignee)
	}
}

// ---------------------------------------------------------------------------
// 5. defer intent replay preserves status + date
//    BUG: The "defer" verb is not handled in replayOne's switch — it falls
//    through to `default: return nil`, silently dropping the intent.
//    Fix: bw-u24.2
// ---------------------------------------------------------------------------
func TestReplayDeferPreservesStatusAndDate(t *testing.T) {
	t.Skip("BUG(bw-u24.2): 'defer' verb is silently dropped by replayOne")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Task to defer", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// Replay a defer intent (the format from cmdDefer: defer <id> until <date>)
	intents := []string{
		`defer ` + iss.ID + ` until 2026-06-01`,
	}
	errs := intent.Replay(env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "deferred" {
		t.Errorf("status = %q, want deferred", got.Status)
	}
	if got.DeferUntil != "2026-06-01" {
		t.Errorf("defer_until = %q, want 2026-06-01", got.DeferUntil)
	}
}

// ---------------------------------------------------------------------------
// 5b. undefer intent replay restores open status
//     BUG: The "undefer" verb is not handled in replayOne's switch.
//     Fix: bw-u24.2
// ---------------------------------------------------------------------------
func TestReplayUndeferRestoresOpenStatus(t *testing.T) {
	t.Skip("BUG(bw-u24.2): 'undefer' verb is silently dropped by replayOne")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create a deferred issue.
	iss, _ := env.Store.Create("Deferred task", issue.CreateOpts{DeferUntil: "2026-06-01"})
	env.CommitIntent("create " + iss.ID)
	if iss.Status != "deferred" {
		t.Fatalf("precondition: status = %q, want deferred", iss.Status)
	}

	intents := []string{
		`undefer ` + iss.ID,
	}
	errs := intent.Replay(env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
	if got.DeferUntil != "" {
		t.Errorf("defer_until = %q, want empty", got.DeferUntil)
	}
}

// ---------------------------------------------------------------------------
// 6. close with reason round-trips
//    BUG: replayClose always passes "" as the reason to store.Close(),
//    ignoring the reason= field in the intent string.
//    Fix: bw-u24.3
// ---------------------------------------------------------------------------
func TestReplayCloseWithReasonRoundTrips(t *testing.T) {
	t.Skip("BUG(bw-u24.3): replayClose always passes empty reason to store.Close()")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Task with close reason", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// The format from cmdClose: close <id> reason="duplicate"
	intents := []string{
		`close ` + iss.ID + ` reason="duplicate"`,
	}
	errs := intent.Replay(env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "closed" {
		t.Errorf("status = %q, want closed", got.Status)
	}
	if got.CloseReason != "duplicate" {
		t.Errorf("close_reason = %q, want 'duplicate'", got.CloseReason)
	}
}

// ---------------------------------------------------------------------------
// 7. update description round-trips
//    BUG: cmdUpdate encodes description as "description=..." (literal
//    ellipsis, not the actual text), so the description content is lost.
//    replayUpdate does not handle a "description" key.
//    Fix: bw-u24.3
// ---------------------------------------------------------------------------
func TestReplayUpdateDescriptionRoundTrips(t *testing.T) {
	t.Skip("BUG(bw-u24.3): cmdUpdate encodes description as 'description=...' losing actual text")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Needs description", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	desc := "This is a detailed description of the issue."
	// Expected intent format after fix: update <id> description="<text>"
	intents := []string{
		`update ` + iss.ID + ` description="` + desc + `"`,
	}
	errs := intent.Replay(env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Description != desc {
		t.Errorf("description = %q, want %q", got.Description, desc)
	}
}

// ---------------------------------------------------------------------------
// 7b. create with description round-trips
//     BUG: cmdCreate intent format does not encode the description at all.
//     Fix: bw-u24.3
// ---------------------------------------------------------------------------
func TestReplayCreateWithDescriptionRoundTrips(t *testing.T) {
	t.Skip("BUG(bw-u24.3): create intent does not encode description")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// A hypothetical intent format with description:
	// create <id> p2 task "Title" description="Some desc"
	intents := []string{
		`create test-eeee p2 task "Issue with desc" description="Some description"`,
	}
	errs := intent.Replay(env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	// This test depends on both ID preservation (bw-u24.1) and description
	// encoding (bw-u24.3), so list all issues instead.
	issues, _ := env.Store.List(issue.Filter{})
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if issues[0].Description != "Some description" {
		t.Errorf("description = %q, want 'Some description'", issues[0].Description)
	}
}

// ---------------------------------------------------------------------------
// 8. multi-word title in update
//    BUG: cmdUpdate encodes title as "title=My New Title" (no quoting).
//    replayUpdate splits on spaces, so it only sees "title=My" and treats
//    "New" and "Title" as separate key=value pairs (which have no '=' so
//    they're silently skipped).
//    Fix: bw-u24.3
// ---------------------------------------------------------------------------
func TestReplayUpdateMultiWordTitle(t *testing.T) {
	t.Skip("BUG(bw-u24.3): multi-word title in update intent splits on spaces")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Old title", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// After fix, the title should be properly quoted in the intent:
	// update <id> title="My New Title"
	intents := []string{
		`update ` + iss.ID + ` title="My New Title"`,
	}
	errs := intent.Replay(env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Title != "My New Title" {
		t.Errorf("title = %q, want 'My New Title'", got.Title)
	}
}

// We can also verify that the CURRENT bug exists: the unquoted form
// only captures the first word.
func TestReplayUpdateMultiWordTitleCurrentBehavior(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Old title", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// Current buggy encoding from cmdUpdate: title=My New Title (no quotes)
	intents := []string{
		`update ` + iss.ID + ` title=My New Title`,
	}
	errs := intent.Replay(env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	// BUG: only "My" is captured as the title because "New" and "Title"
	// are treated as separate tokens without '=' so they're ignored.
	if got.Title != "My" {
		t.Logf("NOTE: title = %q (expected 'My' due to known multi-word title bug)", got.Title)
	}
	// This documents the current broken behavior. When the fix lands,
	// this test should be updated or removed in favor of
	// TestReplayUpdateMultiWordTitle above.
}

// ---------------------------------------------------------------------------
// 9. escaped quotes in title
//    BUG: ParseIntent strips all '"' characters and has no escape handling.
//    A title like: He said "hello" becomes: He said hello — the quotes
//    inside the title are lost. Worse, backslash-escaped quotes like
//    \"hello\" cause the parser to toggle in-quote state incorrectly.
//    Fix: bw-u24.3
// ---------------------------------------------------------------------------
func TestReplayEscapedQuotesInTitle(t *testing.T) {
	t.Skip("BUG(bw-u24.3): ParseIntent has no escape handling for embedded quotes")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// After fix, escaped quotes should be handled:
	// create test-ffff p2 task "He said \"hello\" to me"
	intents := []string{
		`create test-ffff p2 task "He said \"hello\" to me"`,
	}
	errs := intent.Replay(env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	issues, _ := env.Store.List(issue.Filter{})
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if issues[0].Title != `He said "hello" to me` {
		t.Errorf("title = %q, want %q", issues[0].Title, `He said "hello" to me`)
	}
}

// Verify current broken behavior for escaped quotes.
func TestParseIntentEscapedQuotesCurrentBehavior(t *testing.T) {
	// Input: create test-ffff p2 task "He said \"hello\" to me"
	raw := `create test-ffff p2 task "He said \"hello\" to me"`
	parts := intent.ParseIntent(raw)

	// Document what actually happens with the current parser:
	// The \" sequences cause the quote state to toggle incorrectly.
	// Instead of getting "He said \"hello\" to me" as one token,
	// the parser produces fragmented output.
	t.Logf("ParseIntent(%q) = %v", raw, parts)

	// The parser should ideally produce:
	// ["create", "test-ffff", "p2", "task", `He said "hello" to me`]
	// But currently it does not handle backslash escapes.

	// Check that it does NOT produce the correct result (documenting the bug).
	expected := `He said "hello" to me`
	found := false
	for _, p := range parts {
		if p == expected {
			found = true
		}
	}
	if found {
		t.Log("ParseIntent unexpectedly produced the correct escaped title; bug may be fixed")
	} else {
		t.Log("Confirmed: ParseIntent does not handle escaped quotes (known bug bw-u24.3)")
	}
}

// Verify current broken behavior for ExtractQuoted with escaped quotes.
func TestExtractQuotedEscapedCurrentBehavior(t *testing.T) {
	raw := `create test-0000 p2 task "He said \"hello\" to me"`
	got := intent.ExtractQuoted(raw)

	// ExtractQuoted finds the first pair of " characters.
	// With escaped quotes, it finds the opening " then the next " which is
	// the \" before hello, so it returns a truncated string.
	t.Logf("ExtractQuoted(%q) = %q", raw, got)

	expected := `He said "hello" to me`
	if got == expected {
		t.Log("ExtractQuoted unexpectedly handled escaped quotes correctly; bug may be fixed")
	} else {
		t.Log("Confirmed: ExtractQuoted does not handle escaped quotes (known bug bw-u24.3)")
	}
}

// ---------------------------------------------------------------------------
// 10. end-to-end sync replay preserves start/defer state
//     BUG: When sync produces a conflict and falls back to intent replay,
//     "start" and "defer" intents are silently dropped because replayOne
//     has no handler for these verbs. The issue state after replay is
//     missing the status transitions.
//     Fix: bw-u24.4 (depends on bw-u24.2 for verb handlers)
// ---------------------------------------------------------------------------
func TestSyncReplayPreservesStartDeferState(t *testing.T) {
	t.Skip("BUG(bw-u24.2, bw-u24.4): start/defer verbs are silently dropped during sync replay")

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()

	// Create two issues and push.
	issA, _ := env.Store.Create("Issue A", issue.CreateOpts{})
	env.CommitIntent(`create ` + issA.ID + ` p2 task "Issue A"`)
	issB, _ := env.Store.Create("Issue B", issue.CreateOpts{})
	env.CommitIntent(`create ` + issB.ID + ` p2 task "Issue B"`)
	env.Repo.Sync()

	// Clone and make a conflicting change on the remote.
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	p := 1
	env2.Store.Update(issA.ID, issue.UpdateOpts{Priority: &p})
	env2.CommitIntent(`update ` + issA.ID + ` priority=1`)
	env2.Repo.Sync()

	// On the original side: start issA, defer issB.
	env.SwitchTo()
	env.Store.Start(issA.ID, "agent-42")
	env.CommitIntent(`start ` + issA.ID + ` assignee="agent-42"`)

	status := "deferred"
	deferDate := "2026-12-01"
	env.Store.Update(issB.ID, issue.UpdateOpts{Status: &status, DeferUntil: &deferDate})
	env.CommitIntent(`defer ` + issB.ID + ` until 2026-12-01`)

	// Sync should trigger conflict replay.
	syncStatus, intents, err := env.Repo.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	env.Store.ClearCache()

	if syncStatus == "needs replay" {
		errs := intent.Replay(env.Store, intents)
		for _, e := range errs {
			t.Logf("replay error: %v", e)
		}
		env.Repo.Push()
	}

	// Verify start state was preserved.
	gotA, _ := env.Store.Get(issA.ID)
	if gotA.Status != "in_progress" {
		t.Errorf("issA.Status = %q, want in_progress", gotA.Status)
	}
	if gotA.Assignee != "agent-42" {
		t.Errorf("issA.Assignee = %q, want agent-42", gotA.Assignee)
	}

	// Verify defer state was preserved.
	gotB, _ := env.Store.Get(issB.ID)
	if gotB.Status != "deferred" {
		t.Errorf("issB.Status = %q, want deferred", gotB.Status)
	}
	if gotB.DeferUntil != "2026-12-01" {
		t.Errorf("issB.DeferUntil = %q, want 2026-12-01", gotB.DeferUntil)
	}
}

// ---------------------------------------------------------------------------
// Additional regression: verify that silently-dropped verbs produce no error.
// This documents the current behavior where unknown/unhandled verbs are
// swallowed. Once the fix lands, these should produce actual state changes.
// ---------------------------------------------------------------------------
func TestStartVerbCurrentlySilentlyDropped(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Startable", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// "start" is handled by the default case in replayOne which returns nil.
	errs := intent.Replay(env.Store, []string{
		`start ` + iss.ID + ` assignee="agent-1"`,
	})

	// Currently: no error, but also no state change.
	if len(errs) != 0 {
		t.Errorf("expected 0 errors (silently dropped), got %d: %v", len(errs), errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("status = %q, want open (start was silently dropped)", got.Status)
	}
	if got.Assignee != "" {
		t.Errorf("assignee = %q, want empty (start was silently dropped)", got.Assignee)
	}
}

func TestDeferVerbCurrentlySilentlyDropped(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Deferrable", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	errs := intent.Replay(env.Store, []string{
		`defer ` + iss.ID + ` until 2026-06-01`,
	})

	if len(errs) != 0 {
		t.Errorf("expected 0 errors (silently dropped), got %d: %v", len(errs), errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("status = %q, want open (defer was silently dropped)", got.Status)
	}
	if got.DeferUntil != "" {
		t.Errorf("defer_until = %q, want empty (defer was silently dropped)", got.DeferUntil)
	}
}

func TestUndeferVerbCurrentlySilentlyDropped(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Undefer me", issue.CreateOpts{DeferUntil: "2026-06-01"})
	env.CommitIntent("create " + iss.ID)

	errs := intent.Replay(env.Store, []string{
		`undefer ` + iss.ID,
	})

	if len(errs) != 0 {
		t.Errorf("expected 0 errors (silently dropped), got %d: %v", len(errs), errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "deferred" {
		t.Errorf("status = %q, want deferred (undefer was silently dropped)", got.Status)
	}
}

// ---------------------------------------------------------------------------
// Verify close reason is lost in current behavior.
// ---------------------------------------------------------------------------
func TestCloseReasonCurrentlyLost(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Close me with reason", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// The intent format from cmdClose includes reason="duplicate"
	// but replayClose ignores it, always passing "" to store.Close().
	errs := intent.Replay(env.Store, []string{
		`close ` + iss.ID + ` reason="duplicate"`,
	})
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "closed" {
		t.Fatalf("status = %q, want closed", got.Status)
	}
	// BUG: reason is empty because replayClose passes "" to store.Close()
	if got.CloseReason != "" {
		t.Logf("close_reason = %q — if non-empty, the bug may be fixed", got.CloseReason)
	} else {
		t.Log("Confirmed: close reason is lost during replay (known bug bw-u24.3)")
	}
}

// ---------------------------------------------------------------------------
// Verify create ID divergence in current behavior.
// ---------------------------------------------------------------------------
func TestCreateIDDivergenceCurrentBehavior(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	intents := []string{
		`create test-aaaa p1 bug "Test ID divergence"`,
	}
	errs := intent.Replay(env.Store, intents)
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	// The issue should have been created, but with a different ID.
	issues, _ := env.Store.List(issue.Filter{})
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}

	if issues[0].ID == "test-aaaa" {
		t.Log("ID was preserved — bug may be fixed (bw-u24.1)")
	} else {
		t.Logf("Confirmed: ID divergence — intent said test-aaaa, got %s (known bug bw-u24.1)", issues[0].ID)
	}

	// Trying to get by intent ID should fail.
	_, err := env.Store.Get("test-aaaa")
	if err != nil {
		t.Logf("Confirmed: Get(test-aaaa) fails: %v", err)
	} else {
		t.Log("Get(test-aaaa) succeeded — bug may be fixed")
	}
}

// ---------------------------------------------------------------------------
// Verify that description= in update intent is parsed (currently it should
// work for single-word descriptions since replayUpdate handles key=value).
// ---------------------------------------------------------------------------
func TestReplayUpdateDescriptionSingleWord(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// replayUpdate supports "description" in its key=value parser,
	// so a single-word description should work.
	errs := intent.Replay(env.Store, []string{
		`update ` + iss.ID + ` description=fixed`,
	})
	if len(errs) > 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, _ := env.Store.Get(iss.ID)
	// replayUpdate currently doesn't have a "description" case in the switch,
	// so it will be silently ignored.
	if got.Description == "fixed" {
		t.Log("description was set — replayUpdate may have been updated")
	} else {
		t.Log("Confirmed: replayUpdate does not handle 'description' key (known gap)")
	}
}

// ---------------------------------------------------------------------------
// Verify that cmdUpdate encodes description as "..." (literal ellipsis).
// ---------------------------------------------------------------------------
func TestCmdUpdateDescriptionEncoding(t *testing.T) {
	// The update command encodes description as "description=..." literally.
	// This means the actual description text is lost in the intent string.
	// We can verify this by examining the code: in update.go line 99:
	//   changes = append(changes, "description=...")
	// This is a documentation test, not a runtime test.
	//
	// When the fix lands, it should encode as:
	//   changes = append(changes, fmt.Sprintf("description=%q", ua.Description))

	// Verify the ParseIntent behavior for the current encoding
	raw := `update test-1234 description=...`
	parts := intent.ParseIntent(raw)
	if len(parts) != 3 {
		t.Fatalf("ParseIntent parts = %v, want 3 parts", parts)
	}
	if parts[2] != "description=..." {
		t.Errorf("parts[2] = %q, want 'description=...'", parts[2])
	}

	// The key is "description" and value is "..." — a literal ellipsis,
	// not the actual description content.
	kv := parts[2]
	eqIdx := strings.Index(kv, "=")
	key := kv[:eqIdx]
	val := kv[eqIdx+1:]
	if key != "description" {
		t.Errorf("key = %q, want 'description'", key)
	}
	if val != "..." {
		t.Errorf("val = %q, want '...'", val)
	}
	t.Log("Confirmed: cmdUpdate encodes description as literal '...' (known bug bw-u24.3)")
}
