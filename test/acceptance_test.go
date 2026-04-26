package test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/config"
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
	t       *testing.T
	dir     string
	cfgPath string
	env     []string
}

const fixedClock = "2026-01-15T10:00:00Z"

func newBwEnv(t *testing.T) *bwEnv {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".bw")

	env := &bwEnv{
		t:       t,
		dir:     dir,
		cfgPath: cfgPath,
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
			"BW_CONFIG="+cfgPath,
			"GIT_AUTHOR_DATE="+fixedClock,
			"GIT_COMMITTER_DATE="+fixedClock,
			"NO_COLOR=1",
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

	// Enable auto-registration so tests see the expected registry behavior.
	env.setConfig("registry.auto", true)

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
// config file, simulating multiple repos registered in a single config.
func newMultiRepoEnv(t *testing.T, n int) []*bwEnv {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), ".bw")
	envs := make([]*bwEnv, n)
	for i := range n {
		dir := t.TempDir()
		env := &bwEnv{
			t:       t,
			dir:     dir,
			cfgPath: cfgPath,
			env: append(os.Environ(),
				"BW_CLOCK="+fixedClock,
				"BW_CONFIG="+cfgPath,
				"GIT_AUTHOR_DATE="+fixedClock,
				"GIT_COMMITTER_DATE="+fixedClock,
				"NO_COLOR=1",
			),
		}
		env.git("init")
		env.git("config", "user.email", "test@beadwork.dev")
		env.git("config", "user.name", "Test User")
		os.WriteFile(filepath.Join(dir, "README"), []byte(fmt.Sprintf("repo %d", i)), 0644)
		env.git("add", ".")
		env.git("commit", "-m", "initial")
		env.bw("init", "--prefix", fmt.Sprintf("r%d", i))
		if i == 0 {
			env.setConfig("registry.auto", true)
		}
		envs[i] = env
	}
	return envs
}

// setConfig sets a key in the global config file.
func (e *bwEnv) setConfig(key string, value any) {
	e.t.Helper()
	cfg, err := config.Load(e.cfgPath)
	if err != nil {
		e.t.Fatalf("setConfig load: %v", err)
	}
	cfg = cfg.Set(key, value)
	if err := cfg.Save(); err != nil {
		e.t.Fatalf("setConfig save: %v", err)
	}
}

// seedRegistry adds paths to the config's registry.repos key.
func (e *bwEnv) seedRegistry(paths ...string) {
	e.t.Helper()
	cfg, err := config.Load(e.cfgPath)
	if err != nil {
		e.t.Fatalf("seedRegistry load: %v", err)
	}
	existing := cfg.StringSlice("registry.repos")
	cfg = cfg.Set("registry.repos", append(existing, paths...))
	if err := cfg.Save(); err != nil {
		e.t.Fatalf("seedRegistry save: %v", err)
	}
}

// registryPaths returns the registered repo paths from the config.
func (e *bwEnv) registryPaths() []string {
	e.t.Helper()
	cfg, err := config.Load(e.cfgPath)
	if err != nil {
		e.t.Fatalf("registryPaths load: %v", err)
	}
	return cfg.StringSlice("registry.repos")
}

// TestScaffoldingHelpers verifies that the test scaffolding helpers work correctly.
func TestScaffoldingHelpers(t *testing.T) {
	env := newBwEnv(t)
	if env.cfgPath == "" {
		t.Fatal("cfgPath not set")
	}

	// seedRegistry + registryPaths round-trip
	env.seedRegistry("/a", "/b")
	got := env.registryPaths()
	found := strings.Join(got, " ")
	if !strings.Contains(found, "/a") || !strings.Contains(found, "/b") {
		t.Errorf("registryPaths() = %v, want /a and /b", got)
	}

	// registryPaths populated after first NeedsStore command
	env2 := newBwEnv(t)
	env2.bw("list") // triggers auto-registration
	paths := env2.registryPaths()
	if len(paths) == 0 {
		t.Errorf("registryPaths() empty after bw list, expected auto-reg data")
	}

	// newMultiRepoEnv creates n envs sharing a config file
	envs := newMultiRepoEnv(t, 3)
	if len(envs) != 3 {
		t.Fatalf("newMultiRepoEnv(3) returned %d envs", len(envs))
	}
	sharedFile := envs[0].cfgPath
	for i, e := range envs {
		if e.cfgPath != sharedFile {
			t.Errorf("env[%d].cfgPath = %q, want %q (shared)", i, e.cfgPath, sharedFile)
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

	got := env.registryPaths()
	if len(got) == 0 {
		t.Fatal("registry empty after bw list")
	}
	canonDir, _ := filepath.EvalSymlinks(env.dir)
	found := false
	for _, p := range got {
		if p == env.dir || p == canonDir {
			found = true
		}
	}
	if !found {
		t.Errorf("registry does not contain repo path %q: %v", env.dir, got)
	}
}

// TestAutoRegFiresForReadOnlyCommands verifies registration happens even
// for commands that don't modify state.
func TestAutoRegFiresForReadOnlyCommands(t *testing.T) {
	env := newBwEnv(t)
	env.bw("ready")

	got := env.registryPaths()
	if len(got) == 0 {
		t.Fatal("registry empty after bw ready")
	}
}

// TestAutoRegistrationSilentFailure verifies that if the config dir is
// unwritable, bw still runs the command successfully.
func TestAutoRegistrationSilentFailure(t *testing.T) {
	dir := t.TempDir()
	cfgPath := "/nonexistent/path/that/should/fail/.bw"
	env := &bwEnv{
		t:       t,
		dir:     dir,
		cfgPath: cfgPath,
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
			"BW_CONFIG="+cfgPath,
			"GIT_AUTHOR_DATE="+fixedClock,
			"GIT_COMMITTER_DATE="+fixedClock,
			"NO_COLOR=1",
		),
	}
	env.git("init")
	env.git("config", "user.email", "test@beadwork.dev")
	env.git("config", "user.name", "Test User")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	env.git("add", ".")
	env.git("commit", "-m", "initial")
	env.bw("init", "--prefix", "test")

	// Should succeed despite unwritable config path.
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

	got := env.registryPaths()
	if len(got) == 0 {
		t.Fatal("registry empty after bw list from worktree")
	}
	for _, p := range got {
		if p == wtDir {
			t.Errorf("registry should not contain worktree path %q: %v", wtDir, got)
		}
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
	paths := env.registryPaths()
	for _, p := range paths {
		if p == "/nonexistent/repo" {
			t.Errorf("pruned entry still in registry: %v", paths)
		}
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
