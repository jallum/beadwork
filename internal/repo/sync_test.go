package repo_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/intent"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func intPtr(n int) *int { return &n }

func TestSyncNoRemote(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	status, intents, err := env.Repo.Sync(nil)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if status != "no remote configured" {
		t.Errorf("status = %q, want 'no remote configured'", status)
	}
	if intents != nil {
		t.Errorf("intents should be nil, got %v", intents)
	}
}

func TestSyncPushToEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()
	_ = bare

	env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create test")

	status, _, err := env.Repo.Sync(nil)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if status != "pushed" {
		t.Errorf("status = %q, want 'pushed'", status)
	}
}

func TestSyncCleanRebase(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()

	// Push initial state
	env.Store.Create("Initial", issue.CreateOpts{})
	env.CommitIntent("create initial")
	env.Repo.Sync(nil)

	// Clone and make a change on the remote side
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	iss2, _ := env2.Store.Create("Remote issue", issue.CreateOpts{Priority: intPtr(1)})
	env2.CommitIntent("create " + iss2.ID)
	env2.Repo.Sync(nil)

	// Back to original, make a non-conflicting change
	env.SwitchTo()
	iss1, _ := env.Store.Create("Local issue", issue.CreateOpts{Priority: intPtr(2)})
	env.CommitIntent("create " + iss1.ID)

	status, _, err := env.Repo.Sync(nil)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if status != "rebased and pushed" {
		t.Errorf("status = %q, want 'rebased and pushed'", status)
	}

	// Both issues should exist
	all, _ := env.Store.List(issue.Filter{})
	if len(all) < 3 {
		t.Errorf("expected at least 3 issues, got %d", len(all))
	}
}

func TestSyncDirtyRebaseIntentReplay(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()

	// Create a shared issue and push
	shared, _ := env.Store.Create("Shared issue", issue.CreateOpts{})
	env.CommitIntent("create " + shared.ID + " p3 task \"Shared issue\"")
	env.Repo.Sync(nil)

	// Clone and modify the shared issue
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	statusIP := "in_progress"
	env2.Store.Update(shared.ID, issue.UpdateOpts{Status: &statusIP})
	env2.CommitIntent("update " + shared.ID + " status=in_progress")
	env2.Repo.Sync(nil)

	// Back to original, also modify the same issue (will conflict)
	env.SwitchTo()
	assignee := "local-agent"
	env.Store.Update(shared.ID, issue.UpdateOpts{Assignee: &assignee})
	env.CommitIntent("update " + shared.ID + " assignee=local-agent")

	status, intents, err := env.Repo.Sync(nil)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// After sync the underlying tree changed; discard stale cache.
	env.Store.ClearCache()

	if status == "needs replay" {
		// Replay the intents
		errs := intent.Replay(env.Store, intents)
		if len(errs) > 0 {
			for _, e := range errs {
				t.Logf("replay error: %v", e)
			}
		}
		if err := env.Repo.Push(nil); err != nil {
			t.Fatalf("Push after replay: %v", err)
		}

		// Verify the result: remote's status=in_progress + local's assignee=local-agent
		got, err := env.Store.Get(shared.ID)
		if err != nil {
			t.Fatalf("Get after replay: %v", err)
		}
		if got.Status != "in_progress" {
			t.Errorf("status = %q, want 'in_progress' (from remote)", got.Status)
		}
		if got.Assignee != "local-agent" {
			t.Errorf("assignee = %q, want 'local-agent' (from local replay)", got.Assignee)
		}
	} else if status == "rebased and pushed" {
		// Git managed to cleanly rebase (possible if changes don't overlap in file)
		got, _ := env.Store.Get(shared.ID)
		if got.Assignee != "local-agent" {
			t.Errorf("assignee = %q after clean rebase", got.Assignee)
		}
	} else {
		t.Errorf("unexpected sync status: %q", status)
	}
}

func TestSyncMultipleIntentsReplay(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()

	// Push initial state
	env.Repo.Sync(nil)

	// Clone and make a change
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	blocker, _ := env2.Store.Create("Remote blocker", issue.CreateOpts{Priority: intPtr(1)})
	env2.CommitIntent("create " + blocker.ID + " p1 task \"Remote blocker\"")
	env2.Repo.Sync(nil)

	// Local side: create multiple issues, link them, label one
	env.SwitchTo()
	a, _ := env.Store.Create("Local A", issue.CreateOpts{Priority: intPtr(1)})
	env.CommitIntent("create " + a.ID + " p1 task \"Local A\"")
	b, _ := env.Store.Create("Local B", issue.CreateOpts{Priority: intPtr(2)})
	env.CommitIntent("create " + b.ID + " p2 task \"Local B\"")
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("link " + a.ID + " blocks " + b.ID)
	env.Store.Label(a.ID, []string{"urgent"}, nil)
	env.CommitIntent("label " + a.ID + " +urgent")

	status, intents, err := env.Repo.Sync(nil)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if status == "needs replay" {
		errs := intent.Replay(env.Store, intents)
		for _, e := range errs {
			t.Logf("replay error: %v", e)
		}
		env.Repo.Push(nil)
	}

	// Verify everything landed: remote blocker + local A + local B + link + label
	all, _ := env.Store.List(issue.Filter{})
	if len(all) < 3 {
		t.Errorf("expected at least 3 issues after sync, got %d", len(all))
	}

	// The link should exist (A blocks B)
	// We need to re-resolve IDs since replay may have assigned new IDs
	// Just check that some issue has the "urgent" label
	labeled, _ := env.Store.List(issue.Filter{Label: "urgent"})
	if len(labeled) < 1 {
		t.Error("expected at least 1 issue with 'urgent' label after replay")
	}
}

func TestSyncUpToDate(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()
	_ = bare

	env.Store.Create("Test", issue.CreateOpts{})
	env.CommitIntent("create test")
	env.Repo.Sync(nil) // push

	// Sync again with no new changes
	status, _, err := env.Repo.Sync(nil)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if status != "up to date" {
		t.Errorf("status = %q, want 'up to date'", status)
	}
}

func TestSyncPicksUpRemoteChanges(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()
	env.Repo.Sync(nil) // push initial

	// Clone, create issue, push
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	remote, _ := env2.Store.Create("From remote", issue.CreateOpts{})
	env2.CommitIntent("create " + remote.ID)
	env2.Repo.Sync(nil)

	// Original syncs — should pick up the remote issue
	env.SwitchTo()
	env.Repo.Sync(nil)

	got, err := env.Store.Get(remote.ID)
	if err != nil {
		t.Fatalf("remote issue not found after sync: %v", err)
	}
	if got.Title != "From remote" {
		t.Errorf("title = %q, want 'From remote'", got.Title)
	}
}

// TestIntentReplayIdempotent verifies that replaying the same intents
// twice produces the same result.
func TestIntentReplayIdempotent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create issue, close it, record intents
	iss, _ := env.Store.Create("Idempotent test", issue.CreateOpts{Priority: intPtr(1)})
	env.CommitIntent("create " + iss.ID + " p1 task \"Idempotent test\"")
	env.Store.Close(iss.ID, "")
	env.CommitIntent("close " + iss.ID)

	// Now try to replay the close intent again — it should fail gracefully
	intents := []string{"close " + iss.ID}
	errs := intent.Replay(env.Store, intents)
	if len(errs) != 1 {
		t.Errorf("expected 1 error (already closed), got %d", len(errs))
	}

	// Issue should still be closed
	got, _ := env.Store.Get(iss.ID)
	if got.Status != "closed" {
		t.Errorf("status = %q, want closed", got.Status)
	}
}

func TestPush(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()
	_ = bare

	// Push initial (to create remote branch)
	env.Store.Create("Initial", issue.CreateOpts{})
	env.CommitIntent("create initial")
	env.Repo.Sync(nil)

	// Create another issue and push
	env.Store.Create("Push test", issue.CreateOpts{Priority: intPtr(1)})
	env.CommitIntent("create push test")

	if err := env.Repo.Push(nil); err != nil {
		t.Fatalf("Push: %v", err)
	}

	// Clone to verify push worked
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	all, _ := env2.Store.List(issue.Filter{})
	if len(all) < 2 {
		t.Errorf("expected at least 2 issues after push, got %d", len(all))
	}
}

func TestSyncFetchOnly(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()
	env.Repo.Sync(nil) // push initial

	// Clone, create issue, push
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	remote, _ := env2.Store.Create("Remote only", issue.CreateOpts{Priority: intPtr(1)})
	env2.CommitIntent("create " + remote.ID)
	env2.Repo.Sync(nil)

	// Original: no local changes, should fast-forward
	env.SwitchTo()
	status, _, err := env.Repo.Sync(nil)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if status != "up to date" {
		t.Errorf("status = %q, want 'up to date'", status)
	}

	// Remote issue should now exist locally
	got, err := env.Store.Get(remote.ID)
	if err != nil {
		t.Fatalf("remote issue not found: %v", err)
	}
	if got.Title != "Remote only" {
		t.Errorf("title = %q", got.Title)
	}
}

func TestForceReinitInvalidPrefix(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	err := env.Repo.ForceReinit("has space", nil)
	if err == nil {
		t.Error("expected error for invalid prefix")
	}
}

func TestSyncLocalAheadNoDiverge(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()
	_ = bare

	// Push initial
	env.Store.Create("Initial", issue.CreateOpts{})
	env.CommitIntent("create initial")
	env.Repo.Sync(nil)

	// Create another issue locally (remote hasn't changed)
	env.Store.Create("Local only", issue.CreateOpts{Priority: intPtr(2)})
	env.CommitIntent("create local only")

	status, _, err := env.Repo.Sync(nil)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if status != "pushed" {
		t.Errorf("status = %q, want 'pushed'", status)
	}
}

func TestSyncMultipleLocalCommits(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()
	_ = bare

	// Push initial
	env.Store.Create("Initial", issue.CreateOpts{})
	env.CommitIntent("create initial")
	env.Repo.Sync(nil)

	// Create several local issues
	for i := 0; i < 3; i++ {
		iss, _ := env.Store.Create("Batch issue", issue.CreateOpts{Priority: intPtr(2)})
		env.CommitIntent("create " + iss.ID)
	}

	status, _, err := env.Repo.Sync(nil)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if status != "pushed" {
		t.Errorf("status = %q, want 'pushed'", status)
	}

	all, _ := env.Store.List(issue.Filter{})
	if len(all) < 4 {
		t.Errorf("expected at least 4 issues, got %d", len(all))
	}
}

func TestPushNoRemote(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("No remote", issue.CreateOpts{})
	env.CommitIntent("create no remote")

	err := env.Repo.Push(nil)
	if err == nil {
		t.Error("expected error when pushing with no remote")
	}
}

// TestSyncWithNonOriginRemoteViaGitConfig exercises the case where the
// only git remote is named "upstream" (no origin) and beadwork has been
// told to use it via git config beadwork.remote.
func TestSyncWithNonOriginRemoteViaGitConfig(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.Dir + "/remote.git"
	gitRun(t, env.Dir, "init", "--bare", bare)
	gitRun(t, env.Dir, "remote", "add", "upstream", bare)

	gitRun(t, env.Dir, "config", "beadwork.remote", "upstream")

	env.Store.Create("Upstream issue", issue.CreateOpts{})
	env.CommitIntent("create upstream issue")

	status, _, err := env.Repo.Sync(nil)
	if err != nil {
		t.Fatalf("Sync against upstream: %v", err)
	}
	if status != "pushed" {
		t.Errorf("status = %q, want 'pushed'", status)
	}

	out, err := exec.Command("git", "-C", bare, "rev-parse", "refs/heads/beadwork").CombinedOutput()
	if err != nil {
		t.Fatalf("remote beadwork ref missing: %s: %v", out, err)
	}
}

// beadworkTip reads the beadwork branch tip from a bare repo, returning
// an empty string if the branch doesn't exist. Used by multi-remote tests
// to assert which bare repos have the branch after sync.
func beadworkTip(t *testing.T, bare string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", bare, "rev-parse", "refs/heads/beadwork").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// TestSyncMultiRemoteUsesFirst verifies that when multiple remotes have
// the beadwork branch, sync uses only the first one alphabetically and
// leaves the others untouched.
func TestSyncMultiRemoteUsesFirst(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare1 := env.Dir + "/bare1.git"
	bare2 := env.Dir + "/bare2.git"
	gitRun(t, env.Dir, "init", "--bare", bare1)
	gitRun(t, env.Dir, "init", "--bare", bare2)
	gitRun(t, env.Dir, "remote", "add", "alpha", bare1)
	gitRun(t, env.Dir, "remote", "add", "beta", bare2)

	// Initial push via git config; only alpha has beadwork now.
	gitRun(t, env.Dir, "config", "beadwork.remote", "alpha")
	env.Store.Create("Seed", issue.CreateOpts{})
	env.CommitIntent("create seed")
	if _, _, err := env.Repo.Sync(nil); err != nil {
		t.Fatalf("initial sync: %v", err)
	}
	// Manually seed beta so both have beadwork.
	gitRun(t, env.Dir, "push", "beta", "refs/heads/beadwork:refs/heads/beadwork")
	seedTip := beadworkTip(t, bare1)
	if seedTip == "" || seedTip != beadworkTip(t, bare2) {
		t.Fatalf("pre-test seed failed: alpha=%q beta=%q", seedTip, beadworkTip(t, bare2))
	}

	// New commit locally, then sync — should push only to alpha (first alphabetically).
	env.Store.Create("Multi", issue.CreateOpts{})
	env.CommitIntent("create multi")

	status, _, err := env.Repo.Sync(nil)
	if err != nil {
		t.Fatalf("multi-remote sync: %v", err)
	}
	if status != "pushed" {
		t.Errorf("status = %q, want 'pushed'", status)
	}

	tip1, tip2 := beadworkTip(t, bare1), beadworkTip(t, bare2)
	if tip1 == seedTip {
		t.Errorf("alpha tip did not advance: still %q", tip1)
	}
	if tip2 != seedTip {
		t.Errorf("beta was updated but should have been left at seed tip: %q → %q", seedTip, tip2)
	}
}

// TestSyncMultiRemoteOnlyOneHasBeadwork covers the mixed case: two
// remotes exist, one has beadwork and one does not. Sync must push only
// to the one that has it; the bare remote without beadwork stays empty.
func TestSyncMultiRemoteOnlyOneHasBeadwork(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare1 := env.Dir + "/bare1.git"
	bare2 := env.Dir + "/bare2.git"
	gitRun(t, env.Dir, "init", "--bare", bare1)
	gitRun(t, env.Dir, "init", "--bare", bare2)
	gitRun(t, env.Dir, "remote", "add", "alpha", bare1)
	gitRun(t, env.Dir, "remote", "add", "beta", bare2)

	gitRun(t, env.Dir, "config", "beadwork.remote", "alpha")
	env.Store.Create("Only alpha", issue.CreateOpts{})
	env.CommitIntent("create only alpha")
	if _, _, err := env.Repo.Sync(nil); err != nil {
		t.Fatalf("initial sync: %v", err)
	}
	if beadworkTip(t, bare1) == "" {
		t.Fatal("alpha did not receive beadwork on initial sync")
	}
	if beadworkTip(t, bare2) != "" {
		t.Fatal("beta unexpectedly has beadwork after initial sync")
	}

	// Add another commit. alpha has beadwork, beta still doesn't — sync
	// should push only to alpha.
	env.Store.Create("Second", issue.CreateOpts{})
	env.CommitIntent("create second")

	if _, _, err := env.Repo.Sync(nil); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if beadworkTip(t, bare2) != "" {
		t.Errorf("beta received beadwork but it had none before sync")
	}
}

// TestSyncConflictReplayIgnoresSecondaryRemote confirms that conflict →
// replay → sync works correctly against the primary remote (alpha) while
// a secondary remote with beadwork (beta) is left untouched.
func TestSyncConflictReplayIgnoresSecondaryRemote(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare1 := env.Dir + "/bare1.git"
	bare2 := env.Dir + "/bare2.git"
	gitRun(t, env.Dir, "init", "--bare", bare1)
	gitRun(t, env.Dir, "init", "--bare", bare2)
	gitRun(t, env.Dir, "remote", "add", "alpha", bare1)
	gitRun(t, env.Dir, "remote", "add", "beta", bare2)
	gitRun(t, env.Dir, "config", "beadwork.remote", "alpha")

	// Seed alpha, then manually push to beta so both have beadwork.
	shared, _ := env.Store.Create("Shared", issue.CreateOpts{})
	env.CommitIntent("create " + shared.ID + " p3 task \"Shared\"")
	if _, _, err := env.Repo.Sync(nil); err != nil {
		t.Fatalf("seed sync: %v", err)
	}
	gitRun(t, env.Dir, "push", "beta", "refs/heads/beadwork:refs/heads/beadwork")
	betaSeedTip := beadworkTip(t, bare2)

	// Clone alpha, push a diverging commit back to alpha.
	env2 := env.CloneEnv(bare1)
	defer env2.Cleanup()
	env2.SwitchTo()
	statusIP := "in_progress"
	env2.Store.Update(shared.ID, issue.UpdateOpts{Status: &statusIP})
	env2.CommitIntent("update " + shared.ID + " status=in_progress")
	if _, _, err := env2.Repo.Sync(nil); err != nil {
		t.Fatalf("clone sync: %v", err)
	}

	// Back to original, make a conflicting local edit.
	env.SwitchTo()
	assignee := "local-agent"
	env.Store.Update(shared.ID, issue.UpdateOpts{Assignee: &assignee})
	env.CommitIntent("update " + shared.ID + " assignee=local-agent")

	// Sync with alpha (the primary): expect conflict or clean rebase.
	status, intents, err := env.Repo.Sync(nil)
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}
	env.Store.ClearCache()

	if status == "needs replay" {
		errs := intent.Replay(env.Store, intents)
		for _, e := range errs {
			t.Logf("replay error: %v", e)
		}
		if _, _, err := env.Repo.Sync(nil); err != nil {
			t.Fatalf("post-replay sync: %v", err)
		}
	}

	// Alpha should have the resolved state; beta should still be at its seed tip.
	if a := beadworkTip(t, bare1); a == "" {
		t.Error("alpha has no beadwork tip after replay")
	}
	if b := beadworkTip(t, bare2); b != betaSeedTip {
		t.Errorf("beta was modified but should have been left at seed tip: was %q, now %q", betaSeedTip, b)
	}
}

// TestSyncStaleBeadworkRemoteConfig ensures that a beadwork.remote value
// pointing at a non-existent remote produces a clear error, not a
// misleading git-level failure.
func TestSyncStaleBeadworkRemoteConfig(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare1 := env.Dir + "/bare1.git"
	bare2 := env.Dir + "/bare2.git"
	gitRun(t, env.Dir, "init", "--bare", bare1)
	gitRun(t, env.Dir, "init", "--bare", bare2)
	gitRun(t, env.Dir, "remote", "add", "alpha", bare1)
	gitRun(t, env.Dir, "remote", "add", "beta", bare2)

	// Config names a remote that doesn't exist.
	gitRun(t, env.Dir, "config", "beadwork.remote", "ghost")

	env.Store.Create("Stale", issue.CreateOpts{})
	env.CommitIntent("create stale")

	_, _, err := env.Repo.Sync(nil)
	if err == nil {
		t.Fatal("expected error for stale beadwork.remote")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("error should name the missing remote: %v", err)
	}
}

func init() {
	// Ensure we don't accidentally run tests against the real repo
	os.Setenv("GIT_AUTHOR_NAME", "Test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@test.com")
	os.Setenv("GIT_COMMITTER_NAME", "Test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@test.com")
}
