package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdInitFresh(t *testing.T) {
	// Create a bare git repo (not initialized with beadwork)
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")
	os.WriteFile(dir+"/README", []byte("test"), 0644)
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	var buf bytes.Buffer
	err := cmdInit(nil, []string{"--prefix", "fresh"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdInit: %v", err)
	}
	if !strings.Contains(buf.String(), "initialized") {
		t.Errorf("output = %q, want 'initialized'", buf.String())
	}
	if !strings.Contains(buf.String(), "fresh") {
		t.Errorf("output = %q, want prefix 'fresh'", buf.String())
	}
}

func TestCmdInitAlreadyInitialized(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdInit(nil, []string{}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for already initialized")
	}
	if !strings.Contains(err.Error(), "already initialized") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestCmdInitForce(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdInit(nil, []string{"--force", "--prefix", "newpfx"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdInit --force: %v", err)
	}
	if !strings.Contains(buf.String(), "reinitialized") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestCmdInitDefaultPrefix(t *testing.T) {
	// Init without --prefix should derive prefix from repo name
	dir := t.TempDir()
	runInDir := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}
	runInDir("git", "init")
	runInDir("git", "config", "user.email", "test@test.com")
	runInDir("git", "config", "user.name", "Test")
	os.WriteFile(dir+"/README", []byte("test"), 0644)
	runInDir("git", "add", ".")
	runInDir("git", "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	var buf bytes.Buffer
	err := cmdInit(nil, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdInit: %v", err)
	}
	if !strings.Contains(buf.String(), "initialized") {
		t.Errorf("output = %q, want 'initialized'", buf.String())
	}
}

func TestCmdInitForceInvalidPrefix(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdInit(nil, []string{"--force", "--prefix", "has space"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for invalid prefix")
	}
}

func TestCmdInitUnknownFlag(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdInit(nil, []string{"--verbose"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestCmdInitForceDefaultPrefix(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Force reinit without specifying prefix (uses empty string -> derives from repo)
	var buf bytes.Buffer
	err := cmdInit(nil, []string{"--force"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdInit --force: %v", err)
	}
	if !strings.Contains(buf.String(), "reinitialized") {
		t.Errorf("output = %q, want 'reinitialized'", buf.String())
	}
}

// setupFreshRepoWithRemotes creates a temp dir, inits a git repo with
// one commit, adds the given named bare remotes, and chdirs into the
// repo. Returns the repo dir and a cleanup function to restore cwd.
// Used by the multi-remote init tests below.
func setupFreshRepoWithRemotes(t *testing.T, remotes ...string) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")
	os.WriteFile(dir+"/README", []byte("test"), 0644)
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")
	for _, name := range remotes {
		bare := dir + "/" + name + ".git"
		run("git", "init", "--bare", bare)
		run("git", "remote", "add", name, bare)
	}
	orig, _ := os.Getwd()
	os.Chdir(dir)
	return dir, func() { os.Chdir(orig) }
}

// TestCmdInitMultiRemotePrompts exercises the new init prompt: multiple
// remotes exist, none have the beadwork branch yet, no origin, no git
// config — init must ask the user which remote to seed and persist the
// choice. Unlike sync, init never consults isInteractiveStdin.
func TestCmdInitMultiRemotePrompts(t *testing.T) {
	dir, cleanup := setupFreshRepoWithRemotes(t, "alpha", "beta")
	defer cleanup()

	origStdin := initStdin
	defer func() { initStdin = origStdin }()
	initStdin = strings.NewReader("2\n") // pick beta

	var buf bytes.Buffer
	if err := cmdInit(nil, []string{"--prefix", "multi"}, PlainWriter(&buf)); err != nil {
		t.Fatalf("cmdInit: %v: %s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "initialized") {
		t.Errorf("output missing 'initialized': %q", buf.String())
	}

	chosen := getGitConfig(t, dir, "beadwork.remote")
	if chosen != "beta" {
		t.Errorf("git config beadwork.remote = %q, want 'beta'", chosen)
	}
}

// TestCmdInitMultiRemoteIgnoresTTYStub confirms the init prompt fires
// even when isInteractiveStdin (sync's gate) is stubbed to return false.
// Init is human-initiated; the TTY check is sync's concern, not init's.
func TestCmdInitMultiRemoteIgnoresTTYStub(t *testing.T) {
	_, cleanup := setupFreshRepoWithRemotes(t, "alpha", "beta")
	defer cleanup()

	origStdin := initStdin
	origInteractive := isInteractiveStdin
	defer func() {
		initStdin = origStdin
		isInteractiveStdin = origInteractive
	}()
	initStdin = strings.NewReader("1\n")
	isInteractiveStdin = func() bool { return false }

	var buf bytes.Buffer
	if err := cmdInit(nil, []string{"--prefix", "alpha"}, PlainWriter(&buf)); err != nil {
		t.Fatalf("cmdInit: %v: %s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "initialized") {
		t.Errorf("output missing 'initialized': %q", buf.String())
	}
}

// TestCmdInitMultiRemoteOriginShortcut verifies origin is auto-selected
// without prompting, even when initStdin has primed input.
func TestCmdInitMultiRemoteOriginShortcut(t *testing.T) {
	dir, cleanup := setupFreshRepoWithRemotes(t, "origin", "upstream")
	defer cleanup()

	origStdin := initStdin
	defer func() { initStdin = origStdin }()
	primed := strings.NewReader("2\n")
	initStdin = primed

	var buf bytes.Buffer
	if err := cmdInit(nil, []string{"--prefix", "o"}, PlainWriter(&buf)); err != nil {
		t.Fatalf("cmdInit: %v: %s", err, buf.String())
	}
	if primed.Len() == 0 {
		t.Error("init prompted despite origin being present")
	}
	// Should not have persisted a choice either, since the origin rule
	// triggers inside resolveSingleRemote without calling the resolver.
	if v := getGitConfig(t, dir, "beadwork.remote"); v != "" {
		t.Errorf("git config beadwork.remote = %q, want unset (origin picked silently)", v)
	}
}

// TestCmdInitMultiRemoteGitConfigShortcut verifies a pre-existing
// beadwork.remote git config short-circuits the prompt during init.
func TestCmdInitMultiRemoteGitConfigShortcut(t *testing.T) {
	dir, cleanup := setupFreshRepoWithRemotes(t, "alpha", "beta")
	defer cleanup()

	if out, err := exec.Command("git", "-C", dir, "config", "beadwork.remote", "beta").CombinedOutput(); err != nil {
		t.Fatalf("git config: %s: %v", out, err)
	}

	origStdin := initStdin
	defer func() { initStdin = origStdin }()
	primed := strings.NewReader("1\n")
	initStdin = primed

	var buf bytes.Buffer
	if err := cmdInit(nil, []string{"--prefix", "g"}, PlainWriter(&buf)); err != nil {
		t.Fatalf("cmdInit: %v: %s", err, buf.String())
	}
	if primed.Len() == 0 {
		t.Error("init prompted despite git config beadwork.remote being set")
	}
}

// TestCmdInitBootstrapsFromRemoteWithBeadwork confirms that if some
// remote already has the beadwork branch, init silently fetches from it
// (no prompt, regardless of initStdin).
func TestCmdInitBootstrapsFromRemoteWithBeadwork(t *testing.T) {
	// Seed a bare repo by doing a full init-and-push in a source repo.
	srcDir, cleanupSrc := setupFreshRepoWithRemotes(t, "alpha")
	var buf bytes.Buffer
	if err := cmdInit(nil, []string{"--prefix", "seed"}, PlainWriter(&buf)); err != nil {
		cleanupSrc()
		t.Fatalf("seed cmdInit: %v: %s", err, buf.String())
	}
	// Push beadwork to alpha so that bare has the branch.
	alphaBare := srcDir + "/alpha.git"
	if out, err := exec.Command("git", "-C", srcDir, "push", "alpha", "refs/heads/beadwork:refs/heads/beadwork").CombinedOutput(); err != nil {
		cleanupSrc()
		t.Fatalf("seed push: %s: %v", out, err)
	}
	cleanupSrc()

	// Fresh repo, add the same bare (with beadwork) as "beta" alongside
	// a second empty bare "alpha2". Init should auto-pick beta (the one
	// with beadwork) with no prompt, no persistence.
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "clone@test.com")
	run("git", "config", "user.name", "Clone")
	os.WriteFile(dir+"/README", []byte("x"), 0644)
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")
	run("git", "init", "--bare", dir+"/alpha2.git")
	run("git", "remote", "add", "alpha2", dir+"/alpha2.git")
	run("git", "remote", "add", "beta", alphaBare)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	origStdin := initStdin
	defer func() { initStdin = origStdin }()
	primed := strings.NewReader("1\n")
	initStdin = primed

	var buf2 bytes.Buffer
	if err := cmdInit(nil, []string{}, PlainWriter(&buf2)); err != nil {
		t.Fatalf("cmdInit: %v: %s", err, buf2.String())
	}
	if primed.Len() == 0 {
		t.Error("init prompted despite beta already having beadwork")
	}
	if !strings.Contains(buf2.String(), "initialized") {
		t.Errorf("output = %q", buf2.String())
	}
	// Prefix came from the seed (the beadwork branch carries .bwconfig
	// with prefix=seed), not the caller.
	if !strings.Contains(buf2.String(), "seed") {
		t.Errorf("expected prefix 'seed' (bootstrapped from remote); output=%q", buf2.String())
	}
}

func TestCmdInitInvalidPrefix(t *testing.T) {
	dir := t.TempDir()
	runInDir := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}
	runInDir("git", "init")
	runInDir("git", "config", "user.email", "test@test.com")
	runInDir("git", "config", "user.name", "Test")
	os.WriteFile(dir+"/README", []byte("test"), 0644)
	runInDir("git", "add", ".")
	runInDir("git", "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	var buf bytes.Buffer
	err := cmdInit(nil, []string{"--prefix", "invalid prefix!"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for invalid prefix")
	}
}
