package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdSyncNoRemote(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdSync(env.Repo, env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdSync: %v", err)
	}
	if !strings.Contains(buf.String(), "no remote") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestCmdSyncPush(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.NewBareRemote()

	env.Store.Create("Sync test", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	err := cmdSync(env.Repo, env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdSync: %v", err)
	}
	if !strings.Contains(buf.String(), "pushed") {
		t.Errorf("output = %q, want 'pushed'", buf.String())
	}
}

func TestCmdSyncUpToDate(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.NewBareRemote()

	env.Store.Create("Sync test", issue.CreateOpts{})
	env.Repo.Commit("create issue")
	env.Repo.Sync() // initial push

	var buf bytes.Buffer
	err := cmdSync(env.Repo, env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdSync: %v", err)
	}
	if !strings.Contains(buf.String(), "up to date") {
		t.Errorf("output = %q, want 'up to date'", buf.String())
	}
}

func TestCmdSyncReplay(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()

	// Create a shared issue and push
	shared, _ := env.Store.Create("Shared", issue.CreateOpts{})
	env.Repo.Commit("create " + shared.ID + " p3 task \"Shared\"")
	env.Repo.Sync()

	// Clone and modify the same issue
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	statusIP := "in_progress"
	env2.Store.Update(shared.ID, issue.UpdateOpts{Status: &statusIP})
	env2.Repo.Commit("update " + shared.ID + " status=in_progress")
	env2.Repo.Sync()

	// Back to original, modify the same issue (potential conflict)
	env.SwitchTo()
	assignee := "agent"
	env.Store.Update(shared.ID, issue.UpdateOpts{Assignee: &assignee})
	env.Repo.Commit("update " + shared.ID + " assignee=agent")

	var buf bytes.Buffer
	err := cmdSync(env.Repo, env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdSync: %v", err)
	}
	out := buf.String()
	// Should either replay or clean rebase
	if !strings.Contains(out, "replayed") && !strings.Contains(out, "rebased") {
		t.Errorf("output = %q, want replay or rebase", out)
	}
}
