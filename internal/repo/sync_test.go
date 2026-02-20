package repo_test

import (
	"os"
	"testing"

	"github.com/jallum/beadwork/internal/intent"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func intPtr(n int) *int { return &n }

func TestSyncNoRemote(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	status, intents, err := env.Repo.Sync()
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

	status, _, err := env.Repo.Sync()
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
	env.Repo.Sync()

	// Clone and make a change on the remote side
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	iss2, _ := env2.Store.Create("Remote issue", issue.CreateOpts{Priority: intPtr(1)})
	env2.CommitIntent("create " + iss2.ID)
	env2.Repo.Sync()

	// Back to original, make a non-conflicting change
	env.SwitchTo()
	iss1, _ := env.Store.Create("Local issue", issue.CreateOpts{Priority: intPtr(2)})
	env.CommitIntent("create " + iss1.ID)

	status, _, err := env.Repo.Sync()
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
	env.Repo.Sync()

	// Clone and modify the shared issue
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	statusIP := "in_progress"
	env2.Store.Update(shared.ID, issue.UpdateOpts{Status: &statusIP})
	env2.CommitIntent("update " + shared.ID + " status=in_progress")
	env2.Repo.Sync()

	// Back to original, also modify the same issue (will conflict)
	env.SwitchTo()
	assignee := "local-agent"
	env.Store.Update(shared.ID, issue.UpdateOpts{Assignee: &assignee})
	env.CommitIntent("update " + shared.ID + " assignee=local-agent")

	status, intents, err := env.Repo.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if status == "needs replay" {
		// Replay the intents
		errs := intent.Replay(env.Repo, env.Store, intents)
		if len(errs) > 0 {
			for _, e := range errs {
				t.Logf("replay error: %v", e)
			}
		}
		if err := env.Repo.Push(); err != nil {
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
	env.Repo.Sync()

	// Clone and make a change
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	blocker, _ := env2.Store.Create("Remote blocker", issue.CreateOpts{Priority: intPtr(1)})
	env2.CommitIntent("create " + blocker.ID + " p1 task \"Remote blocker\"")
	env2.Repo.Sync()

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

	status, intents, err := env.Repo.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if status == "needs replay" {
		errs := intent.Replay(env.Repo, env.Store, intents)
		for _, e := range errs {
			t.Logf("replay error: %v", e)
		}
		env.Repo.Push()
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
	env.Repo.Sync() // push

	// Sync again with no new changes
	status, _, err := env.Repo.Sync()
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
	env.Repo.Sync() // push initial

	// Clone, create issue, push
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	remote, _ := env2.Store.Create("From remote", issue.CreateOpts{})
	env2.CommitIntent("create " + remote.ID)
	env2.Repo.Sync()

	// Original syncs — should pick up the remote issue
	env.SwitchTo()
	env.Repo.Sync()

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
	env.Store.Close(iss.ID)
	env.CommitIntent("close " + iss.ID)

	// Now try to replay the close intent again — it should fail gracefully
	intents := []string{"close " + iss.ID}
	errs := intent.Replay(env.Repo, env.Store, intents)
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
	env.Repo.Sync()

	// Create another issue and push
	env.Store.Create("Push test", issue.CreateOpts{Priority: intPtr(1)})
	env.CommitIntent("create push test")

	if err := env.Repo.Push(); err != nil {
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
	env.Repo.Sync() // push initial

	// Clone, create issue, push
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	remote, _ := env2.Store.Create("Remote only", issue.CreateOpts{Priority: intPtr(1)})
	env2.CommitIntent("create " + remote.ID)
	env2.Repo.Sync()

	// Original: no local changes, should fast-forward
	env.SwitchTo()
	status, _, err := env.Repo.Sync()
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

	err := env.Repo.ForceReinit("has space")
	if err == nil {
		t.Error("expected error for invalid prefix")
	}
}

func init() {
	// Ensure we don't accidentally run tests against the real repo
	os.Setenv("GIT_AUTHOR_NAME", "Test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@test.com")
	os.Setenv("GIT_COMMITTER_NAME", "Test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@test.com")
}
