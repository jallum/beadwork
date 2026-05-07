package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

// addBareRemote creates a bare repo at <env.Dir>/<name>.git and adds it
// as a remote named <name> to the working repo.
func addBareRemote(t *testing.T, env *testutil.Env, name string) string {
	t.Helper()
	bare := env.Dir + "/" + name + ".git"
	if out, err := exec.Command("git", "-C", env.Dir, "init", "--bare", bare).CombinedOutput(); err != nil {
		t.Fatalf("init bare %s: %s: %v", name, out, err)
	}
	if out, err := exec.Command("git", "-C", env.Dir, "remote", "add", name, bare).CombinedOutput(); err != nil {
		t.Fatalf("remote add %s: %s: %v", name, out, err)
	}
	return bare
}

func getGitConfig(t *testing.T, dir, key string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "config", "--get", key).CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func TestCmdSyncNoRemote(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdSync(env.Store, []string{}, PlainWriter(&buf), nil)
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
	_, err := cmdSync(env.Store, []string{}, PlainWriter(&buf), nil)
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
	env.Repo.Sync(nil) // initial push

	var buf bytes.Buffer
	_, err := cmdSync(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdSync: %v", err)
	}
	if !strings.Contains(buf.String(), "up to date") {
		t.Errorf("output = %q, want 'up to date'", buf.String())
	}
}

// TestIsInteractiveStdinPipe proves the TTY primitive returns false when
// stdin is a pipe. Guards against the underlying check being accidentally
// removed or inverted in future refactors.
func TestIsInteractiveStdinPipe(t *testing.T) {
	rPipe, wPipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer rPipe.Close()
	defer wPipe.Close()

	origStdin := os.Stdin
	os.Stdin = rPipe
	defer func() { os.Stdin = origStdin }()

	if isInteractiveStdin() {
		t.Error("isInteractiveStdin() = true for pipe-backed stdin; want false")
	}
}

// TestCmdSyncNonInteractiveFailsWithoutPrompt verifies that when
// isInteractiveStdin reports false, the resolver never reads from
// syncStdin and never writes to git config, even if syncStdin has been
// primed with what would otherwise be a valid selection.
func TestCmdSyncNonInteractiveFailsWithoutPrompt(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	addBareRemote(t, env, "alpha")
	addBareRemote(t, env, "beta")

	origInteractive := isInteractiveStdin
	origStdin := syncStdin
	defer func() {
		isInteractiveStdin = origInteractive
		syncStdin = origStdin
	}()
	isInteractiveStdin = func() bool { return false }
	primed := strings.NewReader("1\n")
	syncStdin = primed

	env.Store.Create("Needs resolution", issue.CreateOpts{})
	env.Repo.Commit("create needs resolution")

	var buf bytes.Buffer
	_, err := cmdSync(env.Store, []string{}, PlainWriter(&buf), nil)
	if err == nil {
		t.Fatalf("expected error in non-interactive multi-remote run; output=%q", buf.String())
	}
	if !strings.Contains(err.Error(), "no default remote") {
		t.Errorf("error = %v, want a 'no default remote' message", err)
	}
	// syncStdin must remain untouched.
	if primed.Len() == 0 {
		t.Error("resolver consumed syncStdin input despite non-interactive stdin")
	}
	// git config must remain unset.
	if v := getGitConfig(t, env.Dir, "beadwork.remote"); v != "" {
		t.Errorf("git config beadwork.remote = %q, want unset", v)
	}
}

// TestCmdSyncInteractivePromptPersists verifies the happy interactive
// path: TTY stub true, valid numeric selection in syncStdin → prompt
// fires, selection is persisted to git config, a subsequent sync does
// not re-prompt (confirmed by no further input being consumed).
func TestCmdSyncInteractivePromptPersists(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	addBareRemote(t, env, "alpha")
	addBareRemote(t, env, "beta")

	origInteractive := isInteractiveStdin
	origStdin := syncStdin
	defer func() {
		isInteractiveStdin = origInteractive
		syncStdin = origStdin
	}()
	isInteractiveStdin = func() bool { return true }
	// Input for the first sync only — the second sync should use the
	// persisted git config and never read from syncStdin again.
	syncStdin = strings.NewReader("1\n")

	env.Store.Create("Interactive", issue.CreateOpts{})
	env.Repo.Commit("create interactive")

	var buf bytes.Buffer
	if _, err := cmdSync(env.Store, []string{}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdSync: %v: %s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "pushed") {
		t.Errorf("output = %q, want it to include 'pushed'", buf.String())
	}
	// Resolver must have written the selection.
	chosen := getGitConfig(t, env.Dir, "beadwork.remote")
	if chosen != "alpha" {
		t.Errorf("git config beadwork.remote = %q, want 'alpha'", chosen)
	}

	// Second sync with empty syncStdin and an issue to push: must not
	// need to prompt, because the config is now set.
	syncStdin = strings.NewReader("")
	env.Store.Create("Second", issue.CreateOpts{})
	env.Repo.Commit("create second")

	var buf2 bytes.Buffer
	if _, err := cmdSync(env.Store, []string{}, PlainWriter(&buf2), nil); err != nil {
		t.Fatalf("second cmdSync: %v: %s", err, buf2.String())
	}
	if strings.Contains(buf2.String(), "multiple remotes") {
		t.Errorf("second sync re-prompted: %q", buf2.String())
	}
}

// TestCmdSyncGitConfigShortcutSkipsPrompt verifies that when
// git config beadwork.remote is already set, the resolver short-circuits
// before reaching the prompt — even with isInteractiveStdin stubbed to
// true and syncStdin primed with input that would otherwise be consumed.
func TestCmdSyncGitConfigShortcutSkipsPrompt(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	addBareRemote(t, env, "alpha")
	addBareRemote(t, env, "beta")
	if out, err := exec.Command("git", "-C", env.Dir, "config", "beadwork.remote", "beta").CombinedOutput(); err != nil {
		t.Fatalf("git config: %s: %v", out, err)
	}

	origInteractive := isInteractiveStdin
	origStdin := syncStdin
	defer func() {
		isInteractiveStdin = origInteractive
		syncStdin = origStdin
	}()
	isInteractiveStdin = func() bool { return true }
	primed := strings.NewReader("1\n")
	syncStdin = primed

	env.Store.Create("Config shortcut", issue.CreateOpts{})
	env.Repo.Commit("create config shortcut")

	var buf bytes.Buffer
	if _, err := cmdSync(env.Store, []string{}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdSync: %v: %s", err, buf.String())
	}
	if primed.Len() == 0 {
		t.Error("resolver prompted despite git config beadwork.remote being set")
	}
	// beadwork branch should be on beta, not alpha.
	if beadworkTipInDir(t, env.Dir+"/beta.git") == "" {
		t.Error("expected beadwork on beta (the configured remote)")
	}
	if beadworkTipInDir(t, env.Dir+"/alpha.git") != "" {
		t.Error("beadwork unexpectedly pushed to alpha")
	}
}

// TestCmdSyncOriginShortcutSkipsPrompt verifies the origin-by-name
// fallback short-circuits the prompt when no git config is set but a
// remote literally named "origin" exists.
func TestCmdSyncOriginShortcutSkipsPrompt(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.NewBareRemote()                // adds "origin"
	addBareRemote(t, env, "upstream") // second remote, without beadwork

	origInteractive := isInteractiveStdin
	origStdin := syncStdin
	defer func() {
		isInteractiveStdin = origInteractive
		syncStdin = origStdin
	}()
	isInteractiveStdin = func() bool { return true }
	primed := strings.NewReader("2\n")
	syncStdin = primed

	env.Store.Create("Origin shortcut", issue.CreateOpts{})
	env.Repo.Commit("create origin shortcut")

	var buf bytes.Buffer
	if _, err := cmdSync(env.Store, []string{}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdSync: %v: %s", err, buf.String())
	}
	if primed.Len() == 0 {
		t.Error("resolver prompted despite an 'origin' remote being available")
	}
	if beadworkTipInDir(t, env.Dir+"/upstream.git") != "" {
		t.Error("beadwork unexpectedly pushed to upstream; expected origin only")
	}
}

// beadworkTipInDir returns the beadwork tip hash in a bare repo, or "".
func beadworkTipInDir(t *testing.T, bare string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", bare, "rev-parse", "refs/heads/beadwork").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func TestCmdSyncReplay(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()

	// Create a shared issue and push
	shared, _ := env.Store.Create("Shared", issue.CreateOpts{})
	env.Repo.Commit("create " + shared.ID + " p3 task \"Shared\"")
	env.Repo.Sync(nil)

	// Clone and modify the same issue
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	statusIP := "in_progress"
	env2.Store.Update(shared.ID, issue.UpdateOpts{Status: &statusIP})
	env2.Repo.Commit("update " + shared.ID + " status=in_progress")
	env2.Repo.Sync(nil)

	// Back to original, modify the same issue (potential conflict)
	env.SwitchTo()
	assignee := "agent"
	env.Store.Update(shared.ID, issue.UpdateOpts{Assignee: &assignee})
	env.Repo.Commit("update " + shared.ID + " assignee=agent")

	var buf bytes.Buffer
	_, err := cmdSync(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdSync: %v", err)
	}
	out := buf.String()
	// Should either replay or clean rebase
	if !strings.Contains(out, "replayed") && !strings.Contains(out, "rebased") {
		t.Errorf("output = %q, want replay or rebase", out)
	}
}
