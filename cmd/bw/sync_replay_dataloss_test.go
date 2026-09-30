package main

// End-to-end regression tests for data loss during conflict replay.
//
// Each test drives two clones of one bare remote through the real
// commands (so the intent strings are the ones users actually produce),
// forces bw sync down the reset-and-replay path, and checks that no
// local work was silently dropped.
//
// Tests that expose unfixed bugs are marked with "BUG:" comments and use
// t.Skip() so the suite stays green until the corresponding fix lands.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/config"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

type cmdFunc func(*issue.Store, []string, Writer, *config.Config) (*config.Config, error)

func runCmd(t *testing.T, env *testutil.Env, fn cmdFunc, args ...string) string {
	t.Helper()
	env.SwitchTo()
	var buf bytes.Buffer
	if _, err := fn(env.Store, args, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("command %v: %v", args, err)
	}
	env.Store.ClearCache()
	return buf.String()
}

// replayPair seeds an epic, pushes it, and returns the local env, a
// second clone standing in for a teammate, and the epic's ID.
func replayPair(t *testing.T) (*testutil.Env, *testutil.Env, string) {
	t.Helper()
	local := testutil.NewEnv(t)
	t.Cleanup(local.Cleanup)
	bare := local.NewBareRemote()

	epicID := strings.TrimSpace(runCmd(t, local, cmdCreate, "Shared epic", "--type", "epic", "--silent"))
	runCmd(t, local, cmdSync)

	remote := local.CloneEnv(bare)
	t.Cleanup(remote.Cleanup)
	return local, remote, epicID
}

func mustGet(t *testing.T, env *testutil.Env, id string) *issue.Issue {
	t.Helper()
	env.Store.ClearCache()
	iss, err := env.Store.Get(id)
	if err != nil {
		t.Fatalf("Get(%s): %v", id, err)
	}
	return iss
}

// findByTitle returns the only issue with the given title.
func findByTitle(t *testing.T, env *testutil.Env, title string) *issue.Issue {
	t.Helper()
	env.Store.ClearCache()
	all, err := env.Store.List(issue.Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var found []*issue.Issue
	for _, iss := range all {
		if iss.Title == title {
			found = append(found, iss)
		}
	}
	if len(found) != 1 {
		t.Fatalf("want exactly one issue titled %q, found %d", title, len(found))
	}
	return found[0]
}

// ---------------------------------------------------------------------------
// BUG: child IDs are allocated as parent.N+1 from the local view, so two
// clones adding a child to the same epic both allocate parent.1. On sync
// the replayed create fails with `ID "parent.1" already exists`, the local
// child is dropped, and the comment and dependency made on it land on the
// remote's child instead.
// ---------------------------------------------------------------------------
func TestSyncReplayKeepsCollidingChild(t *testing.T) {
	t.Skip("BUG: colliding child ID drops the local child on replay")

	local, remote, epic := replayPair(t)

	remoteChild := strings.TrimSpace(runCmd(t, remote, cmdCreate, "Remote task", "--parent", epic, "--silent"))
	runCmd(t, remote, cmdSync)

	localChild := strings.TrimSpace(runCmd(t, local, cmdCreate, "Local task", "--parent", epic, "--silent"))
	runCmd(t, local, cmdComment, localChild, "local breadcrumb")
	blocker := strings.TrimSpace(runCmd(t, local, cmdCreate, "Local blocker", "--silent"))
	runCmd(t, local, cmdDep, "add", blocker, "blocks", localChild)

	if out := runCmd(t, local, cmdSync); !strings.Contains(out, "replayed") {
		t.Fatalf("expected conflict replay, got %q", out)
	}

	if got := mustGet(t, local, remoteChild); got.Title != "Remote task" {
		t.Errorf("%s title = %q, want the remote task untouched", remoteChild, got.Title)
	}
	if rc := mustGet(t, local, remoteChild); len(rc.Comments) != 0 || len(rc.BlockedBy) != 0 {
		t.Errorf("local comment or dep landed on the remote child: comments=%+v blocked_by=%v", rc.Comments, rc.BlockedBy)
	}

	got := findByTitle(t, local, "Local task")
	if got.Parent != epic {
		t.Errorf("local child parent = %q, want %s", got.Parent, epic)
	}
	if len(got.Comments) != 1 || got.Comments[0].Text != "local breadcrumb" {
		t.Errorf("local child comments = %+v, want the local breadcrumb", got.Comments)
	}
	if len(got.BlockedBy) != 1 || got.BlockedBy[0] != blocker {
		t.Errorf("local child blocked_by = %v, want [%s]", got.BlockedBy, blocker)
	}

	runCmd(t, remote, cmdSync)
	findByTitle(t, remote, "Local task")
}

// ---------------------------------------------------------------------------
// BUG: bw create records parent= in its intent but replayCreate ignores
// it, and labels and the deferral date are not recorded at all, so a
// replayed create loses all three.
// ---------------------------------------------------------------------------
func TestSyncReplayKeepsCreateParentLabelsAndDefer(t *testing.T) {
	t.Skip("BUG: replayed create drops parent, labels and defer")

	local, remote, epic := replayPair(t)

	// Both sides edit the epic so the sync conflicts and replays.
	runCmd(t, remote, cmdComment, epic, "remote note")
	runCmd(t, remote, cmdSync)
	runCmd(t, local, cmdComment, epic, "local note")

	id := strings.TrimSpace(runCmd(t, local, cmdCreate, "Parked child",
		"--parent", epic, "--labels", "area:sync,systemic", "--defer", "2099-01-01", "--silent"))

	if out := runCmd(t, local, cmdSync); !strings.Contains(out, "replayed") {
		t.Fatalf("expected conflict replay, got %q", out)
	}

	got := mustGet(t, local, id)
	if got.Parent != epic {
		t.Errorf("parent = %q, want %s", got.Parent, epic)
	}
	if strings.Join(got.Labels, ",") != "area:sync,systemic" {
		t.Errorf("labels = %v, want [area:sync systemic]", got.Labels)
	}
	if got.Status != "deferred" || got.DeferUntil != "2099-01-01" {
		t.Errorf("status/defer = %q/%q, want deferred/2099-01-01", got.Status, got.DeferUntil)
	}
}

// ---------------------------------------------------------------------------
// BUG: the comment intent does not record --author, so a replayed
// comment loses its author.
// ---------------------------------------------------------------------------
func TestSyncReplayKeepsCommentAuthor(t *testing.T) {
	t.Skip("BUG: replayed comment loses its author")

	local, remote, epic := replayPair(t)

	runCmd(t, remote, cmdComment, epic, "remote note", "--author", "alice")
	runCmd(t, remote, cmdSync)
	runCmd(t, local, cmdComment, epic, "local note", "--author", "bob")

	if out := runCmd(t, local, cmdSync); !strings.Contains(out, "replayed") {
		t.Fatalf("expected conflict replay, got %q", out)
	}

	authors := map[string]string{}
	for _, c := range mustGet(t, local, epic).Comments {
		authors[c.Text] = c.Author
	}
	if authors["remote note"] != "alice" || authors["local note"] != "bob" {
		t.Errorf("comment authors = %v, want alice and bob preserved", authors)
	}
}

// ---------------------------------------------------------------------------
// BUG: when both sides change the same field, replayUpdate applies the
// local value over the remote one with no warning, so the remote edit
// disappears. Which side should win is a design choice; this test only
// requires that neither value vanishes silently.
// ---------------------------------------------------------------------------
func TestSyncReplaySameFieldConflictLosesNothingSilently(t *testing.T) {
	t.Skip("BUG: concurrent edits to one field silently discard one side")

	local, remote, epic := replayPair(t)

	runCmd(t, remote, cmdUpdate, epic, "--description", "remote description")
	runCmd(t, remote, cmdSync)

	// One conflicting field and one the remote never touched.
	runCmd(t, local, cmdUpdate, epic, "--description", "local description", "--priority", "1")

	out := runCmd(t, local, cmdSync)
	if !strings.Contains(out, "replayed") {
		t.Fatalf("expected conflict replay, got %q", out)
	}
	if !strings.Contains(out, "description") {
		t.Errorf("sync output should warn about the description conflict, got %q", out)
	}

	got := mustGet(t, local, epic)
	if got.Priority != 1 {
		t.Errorf("priority = %d, want the non-conflicting local change applied", got.Priority)
	}
	kept := got.Description
	for _, c := range got.Comments {
		kept += "\n" + c.Text
	}
	for _, want := range []string{"remote description", "local description"} {
		if !strings.Contains(kept, want) {
			t.Errorf("%q was lost: not in the description or any comment (description %q, comments %+v)", want, got.Description, got.Comments)
		}
	}
}

// Guards against a fix for the test above reporting a conflict when only
// the local side changed a field, however many times.
func TestSyncReplayRepeatedLocalUpdateIsNotAConflict(t *testing.T) {
	local, remote, epic := replayPair(t)

	// The remote touches the same issue, but not its description.
	runCmd(t, remote, cmdComment, epic, "remote note")
	runCmd(t, remote, cmdSync)

	runCmd(t, local, cmdUpdate, epic, "--description", "first draft")
	runCmd(t, local, cmdUpdate, epic, "--description", "second draft")

	out := runCmd(t, local, cmdSync)
	if !strings.Contains(out, "replayed") {
		t.Fatalf("expected conflict replay, got %q", out)
	}
	if strings.Contains(out, "warning") {
		t.Errorf("no warning expected when only the local side changed the field, got %q", out)
	}
	if got := mustGet(t, local, epic); got.Description != "second draft" {
		t.Errorf("description = %q, want second draft", got.Description)
	}
}

// Not a current bug (replayed --defer never moves the status today, see
// below), but a guard for that fix: once replay sets the deferred status,
// a local defer must not silently reopen an issue the remote closed.
func TestSyncReplayDeferDoesNotSilentlyUndoRemoteClose(t *testing.T) {
	local, remote, epic := replayPair(t)

	runCmd(t, remote, cmdClose, epic)
	runCmd(t, remote, cmdSync)
	runCmd(t, local, cmdUpdate, epic, "--defer", "2099-01-01")

	out := runCmd(t, local, cmdSync)
	if !strings.Contains(out, "replayed") {
		t.Fatalf("expected conflict replay, got %q", out)
	}
	if got := mustGet(t, local, epic); got.Status != "closed" && !strings.Contains(out, "warning") {
		t.Errorf("status = %q with no warning: the remote close was silently undone (output %q)", got.Status, out)
	}
}

// ---------------------------------------------------------------------------
// BUG: `bw update --defer` records only defer=, and replayUpdate does not
// move the status, so a replayed deferral leaves the issue open.
// ---------------------------------------------------------------------------
func TestSyncReplayUpdateDeferSetsDeferredStatus(t *testing.T) {
	t.Skip("BUG: replayed update --defer leaves the issue open")

	local, remote, epic := replayPair(t)

	runCmd(t, remote, cmdComment, epic, "remote note")
	runCmd(t, remote, cmdSync)
	runCmd(t, local, cmdUpdate, epic, "--defer", "2099-01-01")

	if out := runCmd(t, local, cmdSync); !strings.Contains(out, "replayed") {
		t.Fatalf("expected conflict replay, got %q", out)
	}
	if got := mustGet(t, local, epic); got.Status != "deferred" || got.DeferUntil != "2099-01-01" {
		t.Errorf("status/defer = %q/%q, want deferred/2099-01-01", got.Status, got.DeferUntil)
	}
}
