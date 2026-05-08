package test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// bwCapture runs a bw command and returns stdout + stderr separately.
func (e *bwEnv) bwCapture(args ...string) (string, string) {
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
	return stdout.String(), stderr.String()
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

// bwAtClock runs bw with BW_CLOCK (and git commit-date envs) overridden.
// Returns stdout; fatals on non-zero exit.
func (e *bwEnv) bwAtClock(clock string, args ...string) string {
	out, _ := e.bwAtClockCapture(clock, args...)
	return out
}

// bwAtClockCapture is bwAtClock that also returns stderr.
func (e *bwEnv) bwAtClockCapture(clock string, args ...string) (string, string) {
	e.t.Helper()
	cmd := exec.Command(bwBin, args...)
	cmd.Dir = e.dir
	overridden := make([]string, 0, len(e.env)+3)
	for _, kv := range e.env {
		switch {
		case strings.HasPrefix(kv, "BW_CLOCK="),
			strings.HasPrefix(kv, "GIT_AUTHOR_DATE="),
			strings.HasPrefix(kv, "GIT_COMMITTER_DATE="):
			continue
		}
		overridden = append(overridden, kv)
	}
	overridden = append(overridden,
		"BW_CLOCK="+clock,
		"GIT_AUTHOR_DATE="+clock,
		"GIT_COMMITTER_DATE="+clock,
	)
	cmd.Env = overridden
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		e.t.Fatalf("bw %s (clock=%s):\nstdout: %s\nstderr: %s\nerr: %v",
			strings.Join(args, " "), clock, stdout.String(), stderr.String(), err)
	}
	return stdout.String(), stderr.String()
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

// recapCursor reads the recap cursor ref from the git dir.
func (e *bwEnv) recapCursor() string {
	e.t.Helper()
	path := filepath.Join(e.dir, ".git", "refs", "beadwork", "recap-cursor")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// recapCursorExists returns true if the recap cursor ref file exists.
func (e *bwEnv) recapCursorExists() bool {
	e.t.Helper()
	path := filepath.Join(e.dir, ".git", "refs", "beadwork", "recap-cursor")
	_, err := os.Stat(path)
	return err == nil
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

// TestRecapEmpty verifies recap output when there's no activity.
func TestRecapEmpty(t *testing.T) {
	env := newBwEnv(t)
	out := env.bw("recap", "today")
	if !strings.Contains(out, "nothing to report") {
		t.Errorf("expected 'nothing to report' message:\n%s", out)
	}
}

// TestRecapWithEvents verifies recap groups events by issue.
func TestRecapWithEvents(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "Task Alpha", "--id", "re-1")
	env.bw("start", "re-1")
	env.bw("close", "re-1")

	out := env.bw("recap", "today")
	if !strings.Contains(out, "re-1") {
		t.Errorf("recap missing re-1:\n%s", out)
	}
	if !strings.Contains(out, "Task Alpha") {
		t.Errorf("recap missing title 'Task Alpha':\n%s", out)
	}
}

// TestRecapJSON verifies JSON output format and scope field.
func TestRecapJSON(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "json target", "--id", "rj-1")

	out := env.bw("recap", "today", "--json")
	if !strings.Contains(out, `"scope": "single"`) {
		t.Errorf("recap --json missing scope=single:\n%s", out)
	}
	if !strings.Contains(out, `"rj-1"`) {
		t.Errorf("recap --json missing issue id:\n%s", out)
	}
}

// TestRecapDryRun verifies --dry-run doesn't advance the cursor.
func TestRecapDryRun(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "dry", "--id", "dr-1")

	env.bw("recap", "today", "--dry-run")
	// Cursor lives in refs/beadwork/recap-cursor — should not exist after --dry-run.
	cmd := exec.Command("git", "show-ref", "--verify", "refs/beadwork/recap-cursor")
	cmd.Dir = env.dir
	if err := cmd.Run(); err == nil {
		t.Error("--dry-run should not have set the recap cursor ref")
	}
}

// TestRecapCursorIsIncremental verifies that after a first recap, a second
// recap with no window flag only shows NEW events, not everything again.
func TestRecapCursorIsIncremental(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "first", "--id", "ic-1")

	// First recap advances the cursor.
	out1 := env.bw("recap")
	if !strings.Contains(out1, "ic-1") {
		t.Fatalf("first recap missing ic-1:\n%s", out1)
	}

	// Second recap with no new activity should report nothing.
	out2 := env.bw("recap")
	if strings.Contains(out2, "ic-1") {
		t.Errorf("second recap should not re-report ic-1:\n%s", out2)
	}
	if !strings.Contains(out2, "nothing to report") {
		t.Errorf("second recap should show 'nothing to report':\n%s", out2)
	}

	// Create new activity — it must show up on the next recap.
	env.bw("create", "second", "--id", "ic-2")
	out3 := env.bw("recap")
	if !strings.Contains(out3, "ic-2") {
		t.Errorf("third recap missing new ic-2:\n%s", out3)
	}
	if strings.Contains(out3, "ic-1") {
		t.Errorf("third recap should not re-show ic-1:\n%s", out3)
	}
}

// TestRecapStampsLastRecapAtWithNoCommits verifies that running recap with
// nothing new bumps the cursor ref's mtime so the "since last recap (Xh ago)"
// header reflects the most recent run, not the last cursor advance. Without
// this, a quiet stretch (no commits between runs) leaves the header growing
// unboundedly even though the user just ran recap.
func TestRecapStampsLastRecapAtWithNoCommits(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "x", "--id", "lr-1")
	env.bw("recap")

	if !env.recapCursorExists() {
		t.Fatalf("first recap did not set cursor ref")
	}
	cursor1 := env.recapCursor()

	// Simulate a quiet stretch: backdate the cursor file's mtime as if the
	// last recap ran two hours ago. Using a real-time delta (not BW_CLOCK)
	// because mtime is wall-clock filesystem state.
	cursorPath := filepath.Join(env.dir, ".git", "refs", "beadwork", "recap-cursor")
	backdate := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(cursorPath, backdate, backdate); err != nil {
		t.Fatalf("chtimes backdate: %v", err)
	}
	info, err := os.Stat(cursorPath)
	if err != nil {
		t.Fatalf("stat after backdate: %v", err)
	}
	if time.Since(info.ModTime()) < time.Hour {
		t.Fatalf("backdate did not stick: mtime %v is only %v ago", info.ModTime(), time.Since(info.ModTime()))
	}
	backdated := info.ModTime()

	// Run again with nothing new. Cursor value must not change, mtime must
	// advance beyond the backdated time.
	env.bw("recap")
	if !env.recapCursorExists() {
		t.Errorf("second recap lost cursor ref")
	}
	if env.recapCursor() != cursor1 {
		t.Errorf("cursor changed with no new commits")
	}
	info2, err := os.Stat(cursorPath)
	if err != nil {
		t.Fatalf("stat after second recap: %v", err)
	}
	if !info2.ModTime().After(backdated) {
		t.Errorf("cursor mtime did not advance on no-event recap: was %v, still %v", backdated, info2.ModTime())
	}
	if time.Since(info2.ModTime()) > time.Minute {
		t.Errorf("cursor mtime not recent after recap: %v ago", time.Since(info2.ModTime()))
	}
}

// TestRecapDryRunDoesNotStamp verifies --dry-run leaves the cursor ref alone.
func TestRecapDryRunDoesNotStamp(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "x", "--id", "dr-2")

	env.bw("recap", "--dry-run")
	if env.recapCursorExists() {
		t.Errorf("--dry-run should not create cursor ref")
	}
}

// TestRecapAdvancesCursor verifies that a non-dry-run recap advances the cursor.
func TestRecapAdvancesCursor(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "normal", "--id", "cr-1")

	env.bw("recap", "today")
	if env.recapCursor() == "" {
		t.Errorf("recap should advance cursor")
	}
}

// TestRecapSince verifies the --since flag.
func TestRecapSince(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "src", "--id", "sn-1")

	out := env.bw("recap", "--since", "2026-01-01")
	if !strings.Contains(out, "sn-1") {
		t.Errorf("recap --since missing event:\n%s", out)
	}
}

// TestRecapSinceInvalid verifies rejection of a bad --since value.
func TestRecapSinceInvalid(t *testing.T) {
	env := newBwEnv(t)
	out := env.bwFail("recap", "--since", "not-a-date")
	if !strings.Contains(out, "invalid") {
		t.Errorf("expected error for invalid --since:\n%s", out)
	}
}

// TestRecapASCII verifies that --ascii uses plain tree characters
// (only affects --verbose tree output).
func TestRecapASCII(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "ascii test", "--id", "as-1")

	out := env.bw("recap", "today", "--ascii", "--verbose")
	// ASCII tree should use | and ` and -, not ├ └ │
	if strings.ContainsRune(out, '├') || strings.ContainsRune(out, '│') {
		t.Errorf("--ascii output contains unicode box chars:\n%s", out)
	}
}

// TestRecapTodayLocalTimezone verifies that "today" honors the caller's
// local timezone, not UTC. Simulates a user at 1am US/Eastern (which is
// 5am UTC): work done the previous local evening (e.g. 10pm ET = 2am UTC
// "today" UTC) should fall into "today" local.
func TestRecapTodayLocalTimezone(t *testing.T) {
	// Local wall clock: 2026-01-15 01:00:00 -0500 (EST).
	// That's 2026-01-15 06:00:00 UTC — safely inside UTC "today" as well,
	// but the start of local "today" is 2026-01-15 00:00:00 -0500
	// (= 2026-01-15 05:00:00 UTC), while start of UTC "today" would be
	// 2026-01-15 00:00:00 UTC — a 5-hour difference. Seed an event at
	// 2026-01-15 00:30:00 -0500 (= 05:30 UTC): inside local today,
	// inside UTC today too. Then seed 2026-01-14 23:30:00 -0500
	// (= 2026-01-15 04:30:00 UTC): inside local *yesterday*, but inside
	// UTC *today*. A TZ-correct "today" must EXCLUDE the second event.
	envEarlyLocal := "2026-01-15T00:30:00-05:00" // inside local today
	envLateYesterdayLocal := "2026-01-14T23:30:00-05:00"

	dir := t.TempDir()
	cfgPathTZ := filepath.Join(t.TempDir(), ".bw")
	baseEnv := append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
		"NO_COLOR=1",
		"BW_CONFIG="+cfgPathTZ,
	)

	run := func(clock string, args ...string) string {
		cmd := exec.Command(bwBin, args...)
		cmd.Dir = dir
		cmd.Env = append(baseEnv,
			"BW_CLOCK="+clock,
			"GIT_AUTHOR_DATE="+clock,
			"GIT_COMMITTER_DATE="+clock,
		)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("bw %s:\nstderr: %s\nerr: %v",
				strings.Join(args, " "), stderr.String(), err)
		}
		return stdout.String()
	}
	gitRun := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = baseEnv
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %s: %v", strings.Join(args, " "), out, err)
		}
	}

	// Setup
	gitRun("init")
	gitRun("config", "user.email", "test@test.com")
	gitRun("config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("t"), 0644)
	gitRun("add", ".")
	gitRun("commit", "-m", "initial")
	run("2026-01-15T00:00:00-05:00", "init", "--prefix", "tz")

	// Seed one issue "today local" and one "yesterday local, today UTC".
	run(envLateYesterdayLocal, "create", "yday-local", "--id", "ytd-1")
	run(envEarlyLocal, "create", "today-local", "--id", "tdy-1")

	// Now ask for today at 1am local on 2026-01-15.
	out := run("2026-01-15T01:00:00-05:00", "recap", "today", "--dry-run")

	if !strings.Contains(out, "tdy-1") {
		t.Errorf("today-local event missing from 'today' recap:\n%s", out)
	}
	if strings.Contains(out, "ytd-1") {
		t.Errorf("yesterday-local event leaked into 'today' recap (UTC bug):\n%s", out)
	}
}

// TestRecapDurationToken verifies support for duration tokens like 1h, 15m.
func TestRecapDurationToken(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "recent", "--id", "dt-1")

	for _, token := range []string{"1h", "15m", "2d", "1w", "3h30m"} {
		out := env.bw("recap", token)
		if !strings.Contains(out, "dt-1") {
			t.Errorf("recap %s missing event:\n%s", token, out)
		}
	}
}

// TestRecapNoANSIWhenPiped verifies that piped output (non-TTY) has no
// ANSI escape sequences. LLM consumers (Claude Code, etc.) rely on this.
// Same treatment as `bw prime`.
func TestRecapNoANSIWhenPiped(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "noansi", "--id", "na-1")
	env.bw("start", "na-1")
	env.bw("close", "na-1")

	// env.bw() executes bw with stdout captured to a buffer → non-TTY.
	out := env.bw("recap", "today")
	if strings.ContainsRune(out, '\x1b') {
		t.Errorf("recap piped output contains ANSI escape (\\x1b):\n%q", out)
	}
	if strings.Contains(out, "use --verbose") {
		t.Errorf("recap piped output leaks TTY-only hint:\n%s", out)
	}

	// Verbose must also be ANSI-free when piped.
	vOut := env.bw("recap", "today", "--verbose")
	if strings.ContainsRune(vOut, '\x1b') {
		t.Errorf("recap --verbose piped output contains ANSI escape:\n%q", vOut)
	}
}

// TestRecapCondensedDefault verifies default output is condensed.
func TestRecapCondensedDefault(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "Task One", "--id", "co-1")
	env.bw("start", "co-1")
	env.bw("close", "co-1")
	env.bw("comment", "co-1", "done")

	out := env.bw("recap", "today")
	// Default should be one-line-per-issue (not full tree).
	// So it should NOT contain unicode box chars or per-leaf timestamps.
	if strings.ContainsRune(out, '├') {
		t.Errorf("default output should not be a tree:\n%s", out)
	}
	// Should contain the issue, title, and a state hint ("closed").
	if !strings.Contains(out, "co-1") || !strings.Contains(out, "Task One") {
		t.Errorf("condensed output missing id/title:\n%s", out)
	}
	if !strings.Contains(out, "closed") {
		t.Errorf("condensed output should show 'closed' state:\n%s", out)
	}
	// Count lines — should be much shorter than verbose.
	lines := strings.Count(out, "\n")
	if lines > 5 {
		t.Errorf("condensed output too long (%d lines):\n%s", lines, out)
	}
}

// TestRecapVerbose verifies --verbose gives the full tree.
func TestRecapVerbose(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "Task One", "--id", "vb-1")
	env.bw("start", "vb-1")
	env.bw("close", "vb-1")

	out := env.bw("recap", "today", "--verbose")
	// Verbose should be a tree — one leaf per event.
	if !strings.ContainsRune(out, '├') && !strings.ContainsRune(out, '└') {
		t.Errorf("--verbose should render a tree:\n%s", out)
	}
	// Should contain each event type.
	for _, ev := range []string{"create", "start", "close"} {
		if !strings.Contains(out, ev) {
			t.Errorf("--verbose missing event %q:\n%s", ev, out)
		}
	}
}

// TestRecapVerboseShortFlag verifies -v is a shorthand for --verbose.
func TestRecapVerboseShortFlag(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "v test", "--id", "sv-1")

	out := env.bw("recap", "today", "-v")
	if !strings.ContainsRune(out, '├') && !strings.ContainsRune(out, '└') {
		t.Errorf("-v should render a tree:\n%s", out)
	}
}

// TestRecapFirstRecap24h verifies first-recap uses 24h backfill when no cursor.
func TestRecapFirstRecap24h(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "recent", "--id", "fr-1")

	out := env.bw("recap")
	if !strings.Contains(out, "first recap") {
		t.Errorf("first recap should show backfill label:\n%s", out)
	}
}

// TestRecapFromSubdir verifies recap walks up to find the repo.
func TestRecapFromSubdir(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "from sub", "--id", "sb-1")

	sub := filepath.Join(env.dir, "a", "b")
	os.MkdirAll(sub, 0755)

	out := env.bwAt(sub, "recap", "today")
	if !strings.Contains(out, "sb-1") {
		t.Errorf("recap from subdir missing event:\n%s", out)
	}
}

// TestRecapNotInRepo verifies error when not in a git repo.
func TestRecapNotInRepo(t *testing.T) {
	dir := t.TempDir()
	env := &bwEnv{
		t:   t,
		dir: dir,
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
			"BW_CONFIG="+filepath.Join(t.TempDir(), ".bw"),
			"NO_COLOR=1",
		),
	}
	out := env.bwFail("recap")
	if !strings.Contains(out, "not a git repository") {
		t.Errorf("expected 'not a git repository' error:\n%s", out)
	}
}

// TestRecapHelp verifies the recap help output.
func TestRecapHelp(t *testing.T) {
	env := newBwEnv(t)
	out := env.bw("recap", "--help")
	for _, flag := range []string{"--since", "--dry-run", "--all", "--json", "--ascii"} {
		if !strings.Contains(out, flag) {
			t.Errorf("recap help missing %s:\n%s", flag, out)
		}
	}
}

// TestRecapInBwHelp verifies recap appears in top-level help.
func TestRecapInBwHelp(t *testing.T) {
	env := newBwEnv(t)
	out := env.bw("--help")
	if !strings.Contains(out, "recap") {
		t.Errorf("bw --help missing recap:\n%s", out)
	}
}

// TestRecapAllThreeHealthy verifies cross-repo recap over 3 healthy repos.
func TestRecapAllThreeHealthy(t *testing.T) {
	envs := newMultiRepoEnv(t, 3)

	// Create activity in each repo.
	envs[0].bw("create", "Alpha", "--id", "a-1")
	envs[1].bw("create", "Beta", "--id", "b-1")
	envs[2].bw("create", "Gamma", "--id", "g-1")

	out := envs[0].bw("recap", "today", "--all")
	for _, id := range []string{"a-1", "b-1", "g-1"} {
		if !strings.Contains(out, id) {
			t.Errorf("cross-repo recap missing %s:\n%s", id, out)
		}
	}
	if !strings.Contains(out, "3 repo") {
		t.Errorf("expected '3 repo(s)' summary:\n%s", out)
	}
}

// TestRecapAllWarnsOnMissing verifies that missing repos warn on stderr
// and get skipped rather than failing.
func TestRecapAllWarnsOnMissing(t *testing.T) {
	envs := newMultiRepoEnv(t, 2)
	envs[0].bw("create", "Real", "--id", "re-1")

	// Inject a nonexistent path into the registry.
	envs[0].seedRegistry("/nonexistent/path")

	stdout, stderr := envs[0].bwCapture("recap", "today", "--all")
	if !strings.Contains(stderr, "skipping") || !strings.Contains(stderr, "/nonexistent/path") {
		t.Errorf("expected 'skipping' warning for missing repo on stderr:\n%s", stderr)
	}
	if !strings.Contains(stdout, "re-1") {
		t.Errorf("healthy repo activity missing from stdout:\n%s", stdout)
	}
}

// TestRecapAllWarnsOnCFlag verifies that -C is warned about with --all.
func TestRecapAllWarnsOnCFlag(t *testing.T) {
	envs := newMultiRepoEnv(t, 2)
	envs[0].bw("create", "A", "--id", "c-1")

	_, stderr := envs[0].bwCapture("-C", envs[0].dir, "recap", "today", "--all")
	if !strings.Contains(stderr, "-C is ignored with --all") {
		t.Errorf("expected '-C ignored' warning:\n%s", stderr)
	}
}

// TestRecapAllJSONScope verifies the --json shape has scope=cross.
func TestRecapAllJSONScope(t *testing.T) {
	envs := newMultiRepoEnv(t, 2)
	envs[0].bw("create", "J", "--id", "j-1")

	out := envs[0].bw("recap", "today", "--all", "--json")
	if !strings.Contains(out, `"scope": "cross"`) {
		t.Errorf("cross-repo --json missing scope=cross:\n%s", out)
	}
	if !strings.Contains(out, `"repos"`) {
		t.Errorf("cross-repo --json missing repos array:\n%s", out)
	}
}

// TestRecapAllAdvancesPerRepoCursors verifies each repo gets its own cursor advance.
func TestRecapAllAdvancesPerRepoCursors(t *testing.T) {
	envs := newMultiRepoEnv(t, 2)
	envs[0].bw("create", "A", "--id", "ca-1")
	envs[1].bw("create", "B", "--id", "cb-1")

	envs[0].bw("recap", "--all")

	// Both repos should now have a cursor ref.
	cursors := 0
	for _, e := range envs {
		if e.recapCursor() != "" {
			cursors++
		}
	}
	if cursors != 2 {
		t.Errorf("expected 2 cursors after recap --all, got %d", cursors)
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

// TestCloseStampsUnblockedEvents verifies that closing an issue stamps
// "unblocked <id>" lines into the commit message for each newly unblocked issue.
func TestCloseStampsUnblockedEvents(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "blocker", "--id", "bl-1")
	env.bw("create", "blocked", "--id", "bl-2")
	env.bw("dep", "add", "bl-1", "blocks", "bl-2")

	out := env.bw("close", "bl-1")
	if !strings.Contains(out, "unblocked") {
		t.Fatalf("close output missing unblocked info:\n%s", out)
	}

	// Check the commit message on the beadwork branch.
	log := env.git("log", "-1", "--format=%B", "beadwork")
	if !strings.Contains(log, "unblocked bl-2") {
		t.Errorf("close commit missing 'unblocked bl-2':\n%s", log)
	}
}

// TestCloseChainStampsOnlyRemainingDep verifies that when an issue has
// multiple blockers, closing one does NOT stamp an unblocked event for
// the still-blocked dependent.
func TestCloseChainStampsOnlyRemainingDep(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "blocker A", "--id", "ca-1")
	env.bw("create", "blocker B", "--id", "ca-2")
	env.bw("create", "dependent", "--id", "ca-3")
	env.bw("dep", "add", "ca-1", "blocks", "ca-3")
	env.bw("dep", "add", "ca-2", "blocks", "ca-3")

	// Close only ca-1; ca-3 still blocked by ca-2.
	env.bw("close", "ca-1")
	log := env.git("log", "-1", "--format=%B", "beadwork")
	if strings.Contains(log, "unblocked ca-3") {
		t.Errorf("close commit should NOT contain 'unblocked ca-3' (still blocked by ca-2):\n%s", log)
	}

	// Now close ca-2; ca-3 should be unblocked.
	env.bw("close", "ca-2")
	log = env.git("log", "-1", "--format=%B", "beadwork")
	if !strings.Contains(log, "unblocked ca-3") {
		t.Errorf("close commit should contain 'unblocked ca-3':\n%s", log)
	}
}

// TestCloseReasonContainingUnblockedWord verifies that a close reason
// containing the word "unblocked" does not create a spurious unblocked event.
func TestCloseReasonContainingUnblockedWord(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "solo issue", "--id", "cu-1")

	env.bw("close", "cu-1", "--reason", "unblocked by external team")
	log := env.git("log", "-1", "--format=%B", "beadwork")
	// The reason appears in the first line (close intent), but there should be
	// no second line matching "unblocked <id>".
	lines := strings.Split(strings.TrimSpace(log), "\n")
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "unblocked ") && !strings.Contains(line, "=") {
			// This looks like a stamped event, but it's just the reason text
			// on the first line. Additional lines should not match.
			t.Errorf("spurious unblocked line in commit: %q", line)
		}
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

// TestRecapExplicitWindowGapSkipsAdvance verifies that an explicit window
// that starts AFTER the current cursor leaves the cursor untouched, and
// prints a gap notice on stderr naming the unrendered count.
func TestRecapExplicitWindowGapSkipsAdvance(t *testing.T) {
	env := newBwEnv(t)

	const (
		day1 = "2026-04-10T09:00:00Z"
		day2 = "2026-04-12T09:00:00Z"
		day3 = "2026-04-14T12:00:00Z"
	)

	env.bwAtClock(day1, "create", "gap-origin", "--id", "gp-1")
	env.bwAtClock(day1, "recap")
	cursor1 := env.recapCursor()
	if cursor1 == "" {
		t.Fatalf("day1 bare recap did not stamp cursor")
	}

	env.bwAtClock(day2, "create", "gap-middle", "--id", "gp-2")

	stdout, stderr := env.bwAtClockCapture(day3, "recap", "today")

	if strings.Contains(stdout, "gp-2") {
		t.Errorf("explicit window should not render gap commit gp-2:\n%s", stdout)
	}
	if !strings.Contains(stderr, "1 commit older than this window") {
		t.Errorf("expected gap notice on stderr, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "bw recap") {
		t.Errorf("gap notice should reference 'bw recap':\n%s", stderr)
	}

	cursor3 := env.recapCursor()
	if cursor3 != cursor1 {
		t.Errorf("gapped explicit run advanced cursor\n  was: %s\n  now: %s", cursor1, cursor3)
	}
}

// TestRecapExplicitWindowNoGapAdvances verifies that an explicit window that
// covers the full unseen range (window.Start <= cursor_time) advances the
// cursor to HEAD exactly like a bare recap.
func TestRecapExplicitWindowNoGapAdvances(t *testing.T) {
	env := newBwEnv(t)

	const (
		day1 = "2026-04-14T03:00:00Z"
		day2 = "2026-04-14T05:00:00Z"
		now  = "2026-04-14T12:00:00Z"
	)

	env.bwAtClock(day1, "create", "first", "--id", "ng-1")
	env.bwAtClock(day1, "recap")
	cursorBefore := env.recapCursor()

	env.bwAtClock(day2, "create", "second", "--id", "ng-2")

	stdout, stderr := env.bwAtClockCapture(now, "recap", "today")
	if !strings.Contains(stdout, "ng-2") {
		t.Errorf("no-gap explicit recap missing ng-2:\n%s", stdout)
	}
	if strings.Contains(stderr, "older than this window") {
		t.Errorf("no-gap explicit recap should not emit gap notice:\n%s", stderr)
	}

	cursorAfter := env.recapCursor()
	if cursorAfter == cursorBefore {
		t.Errorf("no-gap explicit recap should advance cursor")
	}
}
