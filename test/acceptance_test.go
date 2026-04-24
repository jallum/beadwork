package test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var bwBin string // path to built bw binary

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "bw-acceptance-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	bwBin = filepath.Join(tmp, "bw")
	cmd := exec.Command("go", "build", "-o", bwBin, "../cmd/bw")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build bw: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// bwEnv is a self-contained environment for running bw commands against
// a deterministic git repo.
type bwEnv struct {
	t           *testing.T
	dir         string
	registryDir string
	env         []string
}

const fixedClock = "2026-01-15T10:00:00Z"

func newBwEnv(t *testing.T) *bwEnv {
	t.Helper()
	dir := t.TempDir()
	registryFile := filepath.Join(t.TempDir(), ".bw")

	env := &bwEnv{
		t:           t,
		dir:         dir,
		registryDir: registryFile,
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
			"BW_CONFIG="+filepath.Join(dir, ".bw"),
			"GIT_AUTHOR_DATE="+fixedClock,
			"GIT_COMMITTER_DATE="+fixedClock,
			"NO_COLOR=1",
			"BEADWORK_HOME="+registryFile,
		),
	}

	// Initialize a git repo with deterministic config.
	env.git("init")
	env.git("config", "user.email", "test@beadwork.dev")
	env.git("config", "user.name", "Test User")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	env.git("add", ".")
	env.git("commit", "-m", "initial")

	// Initialize beadwork.
	env.bw("init", "--prefix", "test")

	return env
}

func (e *bwEnv) git(args ...string) string {
	e.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = e.dir
	cmd.Env = e.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		e.t.Fatalf("git %s: %s: %v", strings.Join(args, " "), out, err)
	}
	return string(out)
}

func (e *bwEnv) bw(args ...string) string {
	e.t.Helper()
	cmd := exec.Command(bwBin, args...)
	cmd.Dir = e.dir
	cmd.Env = e.env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		e.t.Fatalf("bw %s:\nstdout: %s\nstderr: %s\nerr: %v",
			strings.Join(args, " "), stdout.String(), stderr.String(), err)
	}
	return stdout.String()
}

// bwFail runs a bw command that is expected to fail.
// Returns combined stdout+stderr. Fatals if the command succeeds.
func (e *bwEnv) bwFail(args ...string) string {
	e.t.Helper()
	cmd := exec.Command(bwBin, args...)
	cmd.Dir = e.dir
	cmd.Env = e.env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		e.t.Fatalf("bw %s: expected failure but succeeded\nstdout: %s",
			strings.Join(args, " "), stdout.String())
	}
	return stdout.String() + stderr.String()
}

// bwAt runs bw from a custom directory instead of the default e.dir.
func (e *bwEnv) bwAt(dir string, args ...string) string {
	e.t.Helper()
	cmd := exec.Command(bwBin, args...)
	cmd.Dir = dir
	cmd.Env = e.env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		e.t.Fatalf("bw %s (at %s):\nstdout: %s\nstderr: %s\nerr: %v",
			strings.Join(args, " "), dir, stdout.String(), stderr.String(), err)
	}
	return stdout.String()
}

// goldenCompare runs `bw export` and compares output against a golden file.
// If UPDATE_GOLDEN=1, writes the output as the new golden file instead.
func (e *bwEnv) goldenCompare(name string) {
	e.t.Helper()
	got := e.bw("export")
	compareGolden(e.t, name+".golden.jsonl", got)
}

// compareGolden compares got against a golden file in testdata/.
// If UPDATE_GOLDEN=1, writes got as the new golden file instead.
func compareGolden(t *testing.T, name, got string) {
	t.Helper()
	goldenPath := filepath.Join("testdata", name)

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
		t.Logf("updated golden file: %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file (run with UPDATE_GOLDEN=1 to create): %v", err)
	}

	if got != string(want) {
		t.Errorf("output does not match golden file %s\n--- want ---\n%s\n--- got ---\n%s",
			goldenPath, string(want), got)
	}
}

// newMultiRepoEnv creates n independent bwEnv instances that share the same
// registry directory, simulating multiple repos registered in a single registry.
func newMultiRepoEnv(t *testing.T, n int) []*bwEnv {
	t.Helper()
	registryFile := filepath.Join(t.TempDir(), ".bw")
	envs := make([]*bwEnv, n)
	for i := range n {
		dir := t.TempDir()
		env := &bwEnv{
			t:           t,
			dir:         dir,
			registryDir: registryFile,
			env: append(os.Environ(),
				"BW_CLOCK="+fixedClock,
				"GIT_AUTHOR_DATE="+fixedClock,
				"GIT_COMMITTER_DATE="+fixedClock,
				"NO_COLOR=1",
				"BEADWORK_HOME="+registryFile,
			),
		}
		env.git("init")
		env.git("config", "user.email", "test@beadwork.dev")
		env.git("config", "user.name", "Test User")
		os.WriteFile(filepath.Join(dir, "README"), []byte(fmt.Sprintf("repo %d", i)), 0644)
		env.git("add", ".")
		env.git("commit", "-m", "initial")
		env.bw("init", "--prefix", fmt.Sprintf("r%d", i))
		envs[i] = env
	}
	return envs
}

// seedRegistry writes paths (one per line) to the registry file.
func (e *bwEnv) seedRegistry(paths ...string) {
	e.t.Helper()
	content := strings.Join(paths, "\n") + "\n"
	if err := os.WriteFile(e.registryDir, []byte(content), 0644); err != nil {
		e.t.Fatalf("seedRegistry: %v", err)
	}
}

// registryContents reads and returns the raw content of the registry file.
// Returns an empty string if the file does not exist.
func (e *bwEnv) registryContents() string {
	e.t.Helper()
	data, err := os.ReadFile(e.registryDir)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		e.t.Fatalf("registryContents: %v", err)
	}
	return string(data)
}

// TestScaffoldingHelpers verifies that the test scaffolding helpers work correctly.
func TestScaffoldingHelpers(t *testing.T) {
	// newBwEnv should set up a registry file path
	env := newBwEnv(t)
	if env.registryDir == "" {
		t.Fatal("registryDir not set")
	}

	// seedRegistry + registryContents round-trip
	env.seedRegistry("/a", "/b")
	got := env.registryContents()
	if !strings.Contains(got, "/a") || !strings.Contains(got, "/b") {
		t.Errorf("registryContents() = %q, want /a and /b", got)
	}

	// registryContents on a fresh env returns content from auto-registration
	// (bw init triggers touchRegistry).
	env2 := newBwEnv(t)
	c := env2.registryContents()
	if c == "" {
		t.Errorf("registryContents() on fresh env is empty, expected auto-reg data")
	}

	// newMultiRepoEnv creates n envs sharing a registry file
	envs := newMultiRepoEnv(t, 3)
	if len(envs) != 3 {
		t.Fatalf("newMultiRepoEnv(3) returned %d envs", len(envs))
	}
	sharedFile := envs[0].registryDir
	for i, e := range envs {
		if e.registryDir != sharedFile {
			t.Errorf("env[%d].registryDir = %q, want %q (shared)", i, e.registryDir, sharedFile)
		}
		out := e.bw("list")
		_ = out
	}
}

// TestGoldenBasicScenario exercises the full CLI pipeline: create, parent,
// dependencies, labels, status transitions, comments, defer, then verifies
// export output against a golden file. Finally, imports into a fresh repo
// and verifies round-trip.
func TestGoldenBasicScenario(t *testing.T) {
	env := newBwEnv(t)

	// Create an epic and two child tasks.
	env.bw("create", "Project Epic", "--id", "ep-1", "-t", "epic")
	env.bw("create", "Task Alpha", "--id", "ta-1", "--parent", "ep-1")
	env.bw("create", "Task Beta", "--id", "ta-2", "--parent", "ep-1")

	// Wire dependency: ta-1 blocks ta-2.
	env.bw("dep", "add", "ta-1", "blocks", "ta-2")

	// Labels on ta-1.
	env.bw("label", "ta-1", "+bug", "+urgent")

	// Start and close ta-1.
	env.bw("start", "ta-1")
	env.bw("close", "ta-1")

	// Comment on ta-2.
	env.bw("comment", "ta-2", "needs review")

	// Defer ta-2.
	env.bw("defer", "ta-2", "2026-06-01")

	// Compare against golden file.
	env.goldenCompare("basic_scenario")

	// Round-trip: export → import into fresh repo → export again.
	exported := env.bw("export")

	env2 := newBwEnv(t)
	tmpFile := filepath.Join(env2.dir, "export.jsonl")
	if err := os.WriteFile(tmpFile, []byte(exported), 0644); err != nil {
		t.Fatalf("write export file: %v", err)
	}
	env2.bw("import", tmpFile)

	reimported := env2.bw("export")
	if exported != reimported {
		t.Errorf("round-trip mismatch\n--- original ---\n%s\n--- reimported ---\n%s",
			exported, reimported)
	}
}

// TestAutoRegistrationOnAnyCommand verifies that running any bw command
// in an initialized repo creates a registry entry.
func TestAutoRegistrationOnAnyCommand(t *testing.T) {
	env := newBwEnv(t)

	// Even a read-only command should register.
	env.bw("list")

	got := env.registryContents()
	if got == "" {
		t.Fatal("registry file not created after bw list")
	}
	if !strings.Contains(got, env.dir) {
		t.Errorf("registry does not contain repo path %q:\n%s", env.dir, got)
	}
}

// TestAutoRegFiresForReadOnlyCommands verifies registration happens even
// for commands that don't modify state.
func TestAutoRegFiresForReadOnlyCommands(t *testing.T) {
	env := newBwEnv(t)
	env.bw("ready")

	got := env.registryContents()
	if got == "" {
		t.Fatal("registry file not created after bw ready")
	}
}

// TestAutoRegistrationSilentFailure verifies that if the registry dir is
// unwritable, bw still runs the command successfully.
func TestAutoRegistrationSilentFailure(t *testing.T) {
	dir := t.TempDir()
	env := &bwEnv{
		t:           t,
		dir:         dir,
		registryDir: "/nonexistent/path/that/should/fail/.bw",
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
			"GIT_AUTHOR_DATE="+fixedClock,
			"GIT_COMMITTER_DATE="+fixedClock,
			"NO_COLOR=1",
			"BEADWORK_HOME=/nonexistent/path/that/should/fail/.bw",
		),
	}
	env.git("init")
	env.git("config", "user.email", "test@beadwork.dev")
	env.git("config", "user.name", "Test User")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	env.git("add", ".")
	env.git("commit", "-m", "initial")
	env.bw("init", "--prefix", "test")

	// Should succeed despite unwritable registry dir.
	out := env.bw("list")
	_ = out
}

// TestWorktreeRegistersSameAsMain verifies that running bw from a worktree
// registers the main repo path, not the worktree path.
func TestWorktreeRegistersSameAsMain(t *testing.T) {
	env := newBwEnv(t)

	// Create a worktree.
	wtDir := filepath.Join(filepath.Dir(env.dir), "worktree")
	env.git("worktree", "add", wtDir, "-b", "wt-branch")
	t.Cleanup(func() {
		env.git("worktree", "remove", "--force", wtDir)
	})

	// Run bw from the worktree.
	env.bwAt(wtDir, "list")

	got := env.registryContents()
	if got == "" {
		t.Fatal("registry not created after bw list from worktree")
	}
	// Should contain the main repo dir, not the worktree dir.
	if strings.Contains(got, wtDir) {
		t.Errorf("registry should not contain worktree path %q:\n%s", wtDir, got)
	}
}

// TestRegistryList verifies the registry list command shows registered repos.
func TestRegistryList(t *testing.T) {
	env := newBwEnv(t)
	env.bw("list") // trigger auto-registration

	out := env.bw("registry", "list")
	if !strings.Contains(out, env.dir) {
		t.Errorf("registry list missing repo dir:\n%s", out)
	}
}

// TestRegistryListJSON verifies JSON output format.
func TestRegistryListJSON(t *testing.T) {
	env := newBwEnv(t)
	env.bw("list") // trigger registration

	out := env.bw("registry", "list", "--json")
	if !strings.Contains(out, `"path"`) {
		t.Errorf("registry list --json missing 'path' key:\n%s", out)
	}
}

// TestRegistryListMissing verifies that deleted repos are flagged as MISSING.
func TestRegistryListMissing(t *testing.T) {
	env := newBwEnv(t)
	// Seed a registry entry for a nonexistent path.
	env.seedRegistry("/nonexistent/repo")

	out := env.bw("registry", "list")
	if !strings.Contains(out, "MISSING") {
		t.Errorf("registry list should show MISSING for nonexistent path:\n%s", out)
	}
}

// TestRegistryPruneYes verifies that prune --yes removes missing entries.
func TestRegistryPruneYes(t *testing.T) {
	env := newBwEnv(t)
	env.seedRegistry("/nonexistent/repo")

	out := env.bw("registry", "prune", "--yes")
	if !strings.Contains(out, "pruned 1") {
		t.Errorf("expected 'pruned 1' in output:\n%s", out)
	}

	// Verify it's actually gone.
	contents := env.registryContents()
	if strings.Contains(contents, "/nonexistent/repo") {
		t.Errorf("pruned entry still in registry:\n%s", contents)
	}
}

// TestRegistryPruneNonTTY verifies prune refuses without --yes in non-TTY.
func TestRegistryPruneNonTTY(t *testing.T) {
	env := newBwEnv(t)
	env.seedRegistry("/nonexistent/repo")

	out := env.bwFail("registry", "prune")
	if !strings.Contains(out, "non-interactive") {
		t.Errorf("expected non-interactive error:\n%s", out)
	}
}

// TestRegistryPruneHalfWarning verifies the half-removal warning.
func TestRegistryPruneHalfWarning(t *testing.T) {
	env := newBwEnv(t)
	// 3 out of 4 missing (real repo dir = 1 existing).
	env.seedRegistry("/missing1", "/missing2", "/missing3", env.dir)

	out := env.bw("registry", "prune", "--yes")
	if !strings.Contains(out, "more than half") {
		t.Errorf("expected half-removal warning:\n%s", out)
	}
}

// TestRegistryPruneShortFlag verifies -y works as shorthand for --yes.
func TestRegistryPruneShortFlag(t *testing.T) {
	env := newBwEnv(t)
	env.seedRegistry("/nonexistent/repo")

	out := env.bw("registry", "prune", "-y")
	if !strings.Contains(out, "pruned 1") {
		t.Errorf("expected 'pruned 1' with -y:\n%s", out)
	}
}

// TestRegistryHelp verifies the help output.
func TestRegistryHelp(t *testing.T) {
	env := newBwEnv(t)
	out := env.bw("registry", "--help")
	if !strings.Contains(out, "list") || !strings.Contains(out, "prune") {
		t.Errorf("registry help missing subcommands:\n%s", out)
	}
}

// TestRegistryInBwHelp verifies registry appears in top-level help.
func TestRegistryInBwHelp(t *testing.T) {
	env := newBwEnv(t)
	out := env.bw("--help")
	if !strings.Contains(out, "registry") {
		t.Errorf("bw --help missing registry:\n%s", out)
	}
}

// TestWorktreeRefWrites verifies that bw operations run from inside a git
// worktree write refs to the shared git dir, so tickets are visible from
// the main checkout.
func TestWorktreeRefWrites(t *testing.T) {
	env := newBwEnv(t)

	// Create a worktree as a sibling directory.
	wtDir := filepath.Join(filepath.Dir(env.dir), "worktree")
	env.git("worktree", "add", wtDir, "-b", "wt-branch")
	t.Cleanup(func() {
		env.git("worktree", "remove", "--force", wtDir)
	})

	// From the worktree, create a ticket.
	env.bwAt(wtDir, "create", "worktree test ticket", "--id", "wt-1", "-t", "task")

	// From the worktree, list tickets — verifies read path.
	wtList := env.bwAt(wtDir, "list")
	if !strings.Contains(wtList, "wt-1") {
		t.Fatalf("ticket wt-1 not visible from worktree list:\n%s", wtList)
	}
	if !strings.Contains(wtList, "worktree test ticket") {
		t.Fatalf("ticket title not visible from worktree list:\n%s", wtList)
	}

	// From the main checkout, verify the ticket is visible.
	mainList := env.bw("list")
	if !strings.Contains(mainList, "wt-1") {
		t.Fatalf("ticket wt-1 created in worktree not visible from main checkout:\n%s", mainList)
	}

	// Verify the beadwork branch ref exists in the shared git dir.
	// `git branch` from main should show the beadwork branch.
	branches := env.git("branch", "--list", "beadwork")
	if !strings.Contains(branches, "beadwork") {
		t.Fatalf("beadwork branch ref not found in shared git dir:\n%s", branches)
	}

	// Also verify that git log on beadwork from main sees the commit
	// made by the worktree bw create.
	log := env.git("log", "--oneline", "beadwork")
	if !strings.Contains(log, "wt-1") {
		t.Fatalf("worktree commit not visible in beadwork log from main:\n%s", log)
	}
}
