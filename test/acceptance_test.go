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
	registryDir := t.TempDir()

	env := &bwEnv{
		t:           t,
		dir:         dir,
		registryDir: registryDir,
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
			"GIT_AUTHOR_DATE="+fixedClock,
			"GIT_COMMITTER_DATE="+fixedClock,
			"NO_COLOR=1",
			"BEADWORK_HOME="+registryDir,
			// Suppress any global commit-signing config (1Password,
			// gpg, etc.) so tests don't depend on the user's keyring.
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
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
// registry directory, simulating multiple repos registered in a single registry.
func newMultiRepoEnv(t *testing.T, n int) []*bwEnv {
	t.Helper()
	registryDir := t.TempDir()
	envs := make([]*bwEnv, n)
	for i := range n {
		dir := t.TempDir()
		env := &bwEnv{
			t:           t,
			dir:         dir,
			registryDir: registryDir,
			env: append(os.Environ(),
				"BW_CLOCK="+fixedClock,
				"GIT_AUTHOR_DATE="+fixedClock,
				"GIT_COMMITTER_DATE="+fixedClock,
				"NO_COLOR=1",
				"BEADWORK_HOME="+registryDir,
				"GIT_CONFIG_GLOBAL=/dev/null",
				"GIT_CONFIG_SYSTEM=/dev/null",
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

// seedRegistry writes raw JSON content to a file named "registry.json" in
// the env's registry directory. Use this to set up registry state before
// running commands that read the registry.
func (e *bwEnv) seedRegistry(content string) {
	e.t.Helper()
	path := filepath.Join(e.registryDir, "registry.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		e.t.Fatalf("seedRegistry: %v", err)
	}
}

// registryContents reads and returns the raw content of the registry file.
// Returns an empty string if the file does not exist.
func (e *bwEnv) registryContents() string {
	e.t.Helper()
	path := filepath.Join(e.registryDir, "registry.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		e.t.Fatalf("registryContents: %v", err)
	}
	return string(data)
}

// bwNow returns the time that bw commands will use as "now", respecting BW_CLOCK.
// Panics if BW_CLOCK is set but not parseable (test setup error).
func bwNow() time.Time {
	t, err := time.Parse(time.RFC3339, fixedClock)
	if err != nil {
		panic("fixedClock is not valid RFC3339: " + err.Error())
	}
	return t.UTC()
}

// TestScaffoldingHelpers verifies that the test scaffolding helpers work correctly.
func TestScaffoldingHelpers(t *testing.T) {
	// bwNow should match fixedClock
	now := bwNow()
	if now.Format(time.RFC3339) != fixedClock {
		t.Errorf("bwNow() = %v, want %v", now.Format(time.RFC3339), fixedClock)
	}

	// newBwEnv should set up a registry dir
	env := newBwEnv(t)
	if env.registryDir == "" {
		t.Fatal("registryDir not set")
	}

	// seedRegistry + registryContents round-trip
	env.seedRegistry(`{"repos":{}}`)
	got := env.registryContents()
	if got != `{"repos":{}}` {
		t.Errorf("registryContents() = %q, want %q", got, `{"repos":{}}`)
	}

	// registryContents on a fresh env returns content from auto-registration
	// (bw init triggers touchRegistry).
	env2 := newBwEnv(t)
	c := env2.registryContents()
	if !strings.Contains(c, "last_seen_at") {
		t.Errorf("registryContents() on fresh env missing auto-reg data: %q", c)
	}

	// newMultiRepoEnv creates n envs sharing a registry dir
	envs := newMultiRepoEnv(t, 3)
	if len(envs) != 3 {
		t.Fatalf("newMultiRepoEnv(3) returned %d envs", len(envs))
	}
	sharedDir := envs[0].registryDir
	for i, e := range envs {
		if e.registryDir != sharedDir {
			t.Errorf("env[%d].registryDir = %q, want %q (shared)", i, e.registryDir, sharedDir)
		}
		// Each env should be an independent bw repo
		out := e.bw("list")
		_ = out // no issues yet, just verify it runs
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
	if !strings.Contains(got, "2026-01-15T10:00:00Z") {
		t.Errorf("registry last_seen_at does not reflect BW_CLOCK:\n%s", got)
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
		registryDir: "/nonexistent/path/that/should/fail",
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
			"GIT_AUTHOR_DATE="+fixedClock,
			"GIT_COMMITTER_DATE="+fixedClock,
			"NO_COLOR=1",
			"BEADWORK_HOME=/nonexistent/path/that/should/fail",
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
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
	contents := env.registryContents()
	if strings.Contains(contents, `"cursor"`) {
		t.Errorf("--dry-run should not have set a cursor:\n%s", contents)
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
// nothing new still updates last_recap_at, so repeated recaps update the
// "since last recap" label even against an unchanged HEAD.
func TestRecapStampsLastRecapAtWithNoCommits(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "x", "--id", "lr-1")
	env.bw("recap") // initial — stamps last_recap_at

	before := env.registryContents()
	if !strings.Contains(before, `"last_recap_at"`) {
		t.Fatalf("first recap did not set last_recap_at:\n%s", before)
	}

	// Run again with nothing new. last_recap_at should still be rewritten
	// (same value under BW_CLOCK, but the field must exist and be stamped).
	env.bw("recap")
	after := env.registryContents()
	if !strings.Contains(after, `"last_recap_at"`) {
		t.Errorf("second recap lost last_recap_at:\n%s", after)
	}
}

// TestRecapDryRunDoesNotStamp verifies --dry-run leaves last_recap_at alone.
func TestRecapDryRunDoesNotStamp(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "x", "--id", "dr-2")

	env.bw("recap", "--dry-run")
	contents := env.registryContents()
	if strings.Contains(contents, `"last_recap_at"`) {
		t.Errorf("--dry-run should not stamp last_recap_at:\n%s", contents)
	}
}

// TestRecapAdvancesCursor verifies that a non-dry-run recap advances the cursor.
func TestRecapAdvancesCursor(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "normal", "--id", "cr-1")

	env.bw("recap", "today")
	contents := env.registryContents()
	if !strings.Contains(contents, `"cursor"`) {
		t.Errorf("recap should advance cursor:\n%s", contents)
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
	registryDir := t.TempDir()
	baseEnv := append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
		"NO_COLOR=1",
		"BEADWORK_HOME="+registryDir,
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
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
		t:           t,
		dir:         dir,
		registryDir: t.TempDir(),
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
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

// TestCrossRepoShow verifies `bw show <other-prefix>-<id>` routes to the
// repo registered for that prefix.
func TestCrossRepoShow(t *testing.T) {
	envs := newMultiRepoEnv(t, 2)
	envs[0].bw("list")
	envs[1].bw("list")
	envs[1].bw("create", "Foreign task", "--id", "r1-x1")

	// From envs[0] (prefix r0), run `bw show r1-x1` — should resolve to envs[1].
	out := envs[0].bw("show", "r1-x1")
	if !strings.Contains(out, "Foreign task") {
		t.Errorf("cross-repo show failed to resolve r1-x1:\n%s", out)
	}
}

// TestCrossRepoCloseAndComment verifies mutations route cross-repo.
func TestCrossRepoCloseAndComment(t *testing.T) {
	envs := newMultiRepoEnv(t, 2)
	envs[0].bw("list")
	envs[1].bw("list")
	envs[1].bw("create", "To close", "--id", "r1-x9")

	// From envs[0], comment + close on the other repo's issue.
	envs[0].bw("comment", "r1-x9", "from other repo")
	envs[0].bw("close", "r1-x9")

	// Verify state landed in envs[1].
	out := envs[1].bw("show", "r1-x9", "--json")
	if !strings.Contains(out, `"status": "closed"`) && !strings.Contains(out, `"status":"closed"`) {
		t.Errorf("cross-repo close did not land:\n%s", out)
	}
	if !strings.Contains(out, "from other repo") {
		t.Errorf("cross-repo comment did not land:\n%s", out)
	}
}

// TestCrossRepoExplicitCWins verifies -C is honored and cross-repo resolver
// does not override it.
func TestCrossRepoExplicitCWins(t *testing.T) {
	envs := newMultiRepoEnv(t, 2)
	envs[0].bw("list")
	envs[1].bw("list")
	envs[1].bw("create", "target", "--id", "r1-x5")

	// Pass -C to envs[0] AND an id that belongs to envs[1]. -C should win,
	// which means the command fails because r1-x5 doesn't exist in envs[0].
	out := envs[0].bwFail("-C", envs[0].dir, "show", "r1-x5")
	if !strings.Contains(out, "no issue found") && !strings.Contains(out, "not found") {
		t.Errorf("expected not-found error with -C:\n%s", out)
	}
}

// TestCrossRepoMixedPrefixesRejected verifies mixing prefixes in a single
// command (e.g. dep add linking across repos) fails loudly.
func TestCrossRepoMixedPrefixesRejected(t *testing.T) {
	envs := newMultiRepoEnv(t, 2)
	envs[0].bw("list")
	envs[1].bw("list")

	envs[0].bw("create", "local", "--id", "r0-l1")
	envs[1].bw("create", "remote", "--id", "r1-r1")

	// Try to link a local and remote issue: should error.
	out := envs[0].bwFail("dep", "add", "r0-l1", "blocks", "r1-r1")
	if !strings.Contains(out, "cross-repo") && !strings.Contains(out, "prefixes") {
		t.Errorf("expected cross-repo rejection:\n%s", out)
	}
}

// TestCrossRepoFromNonBeadworkDir verifies that cross-repo commands work
// from a directory that isn't a beadwork repo (or even a git repo). The
// prefix alone is enough to route.
func TestCrossRepoFromNonBeadworkDir(t *testing.T) {
	envs := newMultiRepoEnv(t, 1)
	envs[0].bw("list")
	envs[0].bw("create", "remote task", "--id", "r0-nb1")

	// Use a plain temp dir (no git, no beadwork).
	nonRepo := t.TempDir()
	caller := &bwEnv{
		t:           t,
		dir:         nonRepo,
		registryDir: envs[0].registryDir,
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
			"GIT_AUTHOR_DATE="+fixedClock,
			"GIT_COMMITTER_DATE="+fixedClock,
			"NO_COLOR=1",
			"BEADWORK_HOME="+envs[0].registryDir,
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
		),
	}

	// From a non-beadwork dir, show should resolve via the registry.
	out := caller.bw("show", "r0-nb1")
	if !strings.Contains(out, "remote task") {
		t.Errorf("cross-repo show from non-beadwork dir failed:\n%s", out)
	}

	// And mutations should work too.
	caller.bw("close", "r0-nb1")
	out2 := envs[0].bw("show", "r0-nb1", "--json")
	if !strings.Contains(out2, `"closed"`) {
		t.Errorf("cross-repo close from non-beadwork dir did not land:\n%s", out2)
	}
}

// TestCrossRepoRecapAllFromNonBeadworkDir verifies --all works from
// anywhere, not just inside a beadwork repo.
func TestCrossRepoRecapAllFromNonBeadworkDir(t *testing.T) {
	envs := newMultiRepoEnv(t, 2)
	envs[0].bw("create", "a", "--id", "r0-a1")
	envs[1].bw("create", "b", "--id", "r1-b1")

	nonRepo := t.TempDir()
	caller := &bwEnv{
		t:           t,
		dir:         nonRepo,
		registryDir: envs[0].registryDir,
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
			"GIT_AUTHOR_DATE="+fixedClock,
			"GIT_COMMITTER_DATE="+fixedClock,
			"NO_COLOR=1",
			"BEADWORK_HOME="+envs[0].registryDir,
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
		),
	}

	out := caller.bw("recap", "today", "--all")
	for _, id := range []string{"r0-a1", "r1-b1"} {
		if !strings.Contains(out, id) {
			t.Errorf("recap --all from non-beadwork dir missing %s:\n%s", id, out)
		}
	}
}

// TestCFlagAcceptsPrefix verifies that `bw -C <prefix>` resolves through
// the registry to the repo's real path.
func TestCFlagAcceptsPrefix(t *testing.T) {
	envs := newMultiRepoEnv(t, 2)
	envs[0].bw("list")
	envs[1].bw("list")
	envs[1].bw("create", "via prefix", "--id", "r1-p1")

	// From envs[0], use -C r1 (the prefix of envs[1]) to show r1-p1.
	out := envs[0].bw("-C", "r1", "show", "r1-p1")
	if !strings.Contains(out, "via prefix") {
		t.Errorf("-C <prefix> did not expand correctly:\n%s", out)
	}
}

// TestCFlagAcceptsAlias verifies -C works with an alias after a rename.
func TestCFlagAcceptsAlias(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "legacy", "--id", "test-lg1")
	env.bw("close", "test-lg1")
	env.bw("config", "set", "prefix", "renamed")

	// The old prefix "test" is now an alias. -C test should still find it.
	out := env.bw("-C", "test", "show", "test-lg1")
	if !strings.Contains(out, "legacy") {
		t.Errorf("-C <alias> did not resolve:\n%s", out)
	}
}

// TestCFlagPathFallsThroughWhenNotPrefix verifies that an explicit path
// (absolute or relative) still works even if it isn't a registered prefix.
func TestCFlagPathFallsThroughWhenNotPrefix(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "here", "--id", "test-p1")

	// Absolute path — has '/' so can't be a prefix, must fall through.
	out := env.bw("-C", env.dir, "show", "test-p1")
	if !strings.Contains(out, "here") {
		t.Errorf("-C <abs path> did not work:\n%s", out)
	}
}

// TestCFlagCollisionErrors verifies -C errors when a prefix resolves to
// multiple repos.
func TestCFlagCollisionErrors(t *testing.T) {
	envs := newMultiRepoEnv(t, 1)
	envs[0].bw("list")

	// Inject a duplicate prefix in the registry.
	otherPath := filepath.Join(filepath.Dir(envs[0].dir), "twin")
	os.MkdirAll(otherPath, 0755)
	contents := envs[0].registryContents()
	inject := `"` + otherPath + `": {"last_seen_at":"2026-01-15T10:00:00Z","prefix":"r0"},`
	contents = strings.Replace(contents, `"repos": {`, `"repos": {`+"\n    "+inject, 1)
	envs[0].seedRegistry(contents)

	out := envs[0].bwFail("-C", "r0", "list")
	if !strings.Contains(out, "registered for 2 repositories") {
		t.Errorf("expected -C collision error:\n%s", out)
	}
}

// TestCrossRepoPrefixCollision verifies that when two repos share the
// same prefix, the resolver fails with a clear error listing both paths
// and pointing the user at -C, instead of silently picking one.
func TestCrossRepoPrefixCollision(t *testing.T) {
	envs := newMultiRepoEnv(t, 1)
	envs[0].bw("list")

	// Manually inject a second registry entry with the same prefix as envs[0].
	// envs[0]'s prefix is "r0". Add a second path under the same prefix.
	otherPath := filepath.Join(filepath.Dir(envs[0].dir), "twin")
	os.MkdirAll(otherPath, 0755)
	contents := envs[0].registryContents()
	// Insert a sibling entry by string-rewriting the JSON.
	inject := `"` + otherPath + `": {"last_seen_at":"2026-01-15T10:00:00Z","prefix":"r0"},`
	contents = strings.Replace(contents, `"repos": {`, `"repos": {`+"\n    "+inject, 1)
	envs[0].seedRegistry(contents)

	envs[0].bw("create", "duped", "--id", "r0-d1")

	// Use a non-beadwork dir to force the resolver to look up the prefix
	// (when called from within envs[0], the local prefix matches and
	// resolver short-circuits — no cross-repo lookup happens).
	nonRepo := t.TempDir()
	caller := &bwEnv{
		t:           t,
		dir:         nonRepo,
		registryDir: envs[0].registryDir,
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
			"NO_COLOR=1",
			"BEADWORK_HOME="+envs[0].registryDir,
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
		),
	}

	out := caller.bwFail("show", "r0-d1")
	if !strings.Contains(out, "registered for 2 repositories") {
		t.Errorf("expected collision error mentioning both repos:\n%s", out)
	}
	if !strings.Contains(out, "use -C <path>") {
		t.Errorf("collision error should suggest -C:\n%s", out)
	}
}

// TestConfigSetPrefixRenamesOpen verifies `bw config set prefix` rewrites
// open/in_progress issue IDs while keeping closed ones as historical
// record. References from closed issues to renamed open issues update.
func TestConfigSetPrefixRenamesOpen(t *testing.T) {
	env := newBwEnv(t)

	// 4 issues: 2 open (one with a child), 2 closed (one referencing the open one).
	env.bw("create", "open root", "--id", "test-or1")
	env.bw("create", "open child", "--id", "test-or1.1", "--parent", "test-or1")
	env.bw("create", "another open", "--id", "test-or2")
	env.bw("create", "closed one", "--id", "test-cl1")
	env.bw("create", "closed referrer", "--id", "test-cl2")

	// test-or2 blocks test-cl2; test-cl2 will be closed.
	env.bw("dep", "add", "test-or2", "blocks", "test-cl2")

	env.bw("close", "test-cl1")
	env.bw("close", "test-cl2")

	// Now rename prefix.
	out := env.bw("config", "set", "prefix", "newt")
	if !strings.Contains(out, "renamed") {
		t.Errorf("expected rename summary:\n%s", out)
	}

	// Open issues moved to new prefix.
	out2 := env.bw("show", "newt-or1")
	if !strings.Contains(out2, "open root") {
		t.Errorf("renamed open root not found:\n%s", out2)
	}
	out3 := env.bw("show", "newt-or1.1")
	if !strings.Contains(out3, "open child") {
		t.Errorf("renamed open child not found:\n%s", out3)
	}

	// Old IDs for renamed issues are gone.
	gone := env.bwFail("show", "test-or1")
	if !strings.Contains(gone, "no issue found") {
		t.Errorf("old open ID still resolves:\n%s", gone)
	}

	// Closed issues keep their old prefix.
	out4 := env.bw("show", "test-cl1")
	if !strings.Contains(out4, "closed one") {
		t.Errorf("closed issue prefix changed unexpectedly:\n%s", out4)
	}

	// Closed-referrer's blocked_by should now point at the renamed ID.
	out5 := env.bw("show", "test-cl2", "--json")
	if strings.Contains(out5, "test-or2") || !strings.Contains(out5, "newt-or2") {
		t.Errorf("blocked_by ref not updated in closed issue:\n%s", out5)
	}

	// .bwconfig prefix updated.
	out6 := env.bw("config", "get", "prefix")
	if strings.TrimSpace(out6) != "newt" {
		t.Errorf("config get prefix = %q, want newt", strings.TrimSpace(out6))
	}
}

// TestConfigSetPrefixNoOpenIssues verifies the rename works on a repo
// with no open issues — just changes .bwconfig.
func TestConfigSetPrefixNoOpenIssues(t *testing.T) {
	env := newBwEnv(t)
	env.bw("create", "x", "--id", "test-x1")
	env.bw("close", "test-x1")

	out := env.bw("config", "set", "prefix", "renamed")
	if !strings.Contains(out, "no open issues to rename") {
		t.Errorf("expected 'no open issues to rename' message:\n%s", out)
	}
	if !strings.Contains(out, "renamed") {
		t.Errorf("expected new prefix in output:\n%s", out)
	}

	// Closed issue still has old prefix.
	out2 := env.bw("show", "test-x1")
	if !strings.Contains(out2, "test-x1") {
		t.Errorf("closed issue ID changed unexpectedly:\n%s", out2)
	}
}

// TestConfigSetPrefixAliasesOldForCrossRepo verifies that after a rename,
// the OLD prefix is registered as an alias so closed issues with the old
// prefix remain reachable cross-repo (and reopening them still routes
// correctly from other machines).
func TestConfigSetPrefixAliasesOldForCrossRepo(t *testing.T) {
	env := newBwEnv(t)

	// One open + one closed before rename. Open will be migrated; closed
	// keeps the old prefix.
	env.bw("create", "open", "--id", "test-op1")
	env.bw("create", "closed", "--id", "test-cl1")
	env.bw("close", "test-cl1")

	env.bw("config", "set", "prefix", "newt")

	// Registry should now show prefix=newt and aliases=[test].
	contents := env.registryContents()
	if !strings.Contains(contents, `"prefix": "newt"`) {
		t.Errorf("registry prefix not updated:\n%s", contents)
	}
	if !strings.Contains(contents, `"test"`) || !strings.Contains(contents, `"aliases"`) {
		t.Errorf("old prefix not registered as alias:\n%s", contents)
	}

	// From a non-beadwork dir, BOTH the new-prefix open issue and the
	// old-prefix closed issue must resolve.
	nonRepo := t.TempDir()
	caller := &bwEnv{
		t:           t,
		dir:         nonRepo,
		registryDir: env.registryDir,
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
			"NO_COLOR=1",
			"BEADWORK_HOME="+env.registryDir,
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
		),
	}
	if out := caller.bw("show", "newt-op1"); !strings.Contains(out, "open") {
		t.Errorf("renamed open issue not reachable cross-repo:\n%s", out)
	}
	if out := caller.bw("show", "test-cl1"); !strings.Contains(out, "closed") {
		t.Errorf("old-prefix closed issue not reachable cross-repo:\n%s", out)
	}

	// Reopening a closed old-prefix issue should still route correctly.
	caller.bw("reopen", "test-cl1")
	if out := caller.bw("show", "test-cl1", "--json"); !strings.Contains(out, `"open"`) {
		t.Errorf("reopen of old-prefix closed issue did not land:\n%s", out)
	}
}

// TestConfigSetPrefixSkipsAliasWhenCollides verifies that renaming one of
// two repos sharing a prefix does NOT add the old prefix as an alias on
// the renamed repo — otherwise the collision would persist. After the
// rename, the old prefix should unambiguously resolve to the *other* repo.
func TestConfigSetPrefixSkipsAliasWhenCollides(t *testing.T) {
	envs := newMultiRepoEnv(t, 2)
	envs[0].bw("list")
	envs[1].bw("list")

	// Force the prefix collision: change envs[0] to share prefix "r1".
	// Seed both repos with an issue so rename has something to log.
	envs[1].bw("create", "target", "--id", "r1-t1")

	// Force the prefix collision by flipping envs[0]'s recorded prefix
	// from r0 to r1 directly. Mutating the existing entry avoids
	// OS-specific path canonicalization (macOS resolves /var/folders/…
	// to /private/var/folders/…, so injecting a new entry keyed on
	// envs[0].dir produces a distinct key on macOS but a duplicate key
	// on Linux — and Go's JSON parser gives duplicate-key wins to the
	// last one, silently dropping the injection on Linux).
	contents := envs[1].registryContents()
	contents = strings.Replace(contents, `"prefix": "r0"`, `"prefix": "r1"`, 1)
	envs[1].seedRegistry(contents)

	// Rename envs[1] from r1 -> newr1.
	out := envs[1].bw("config", "set", "prefix", "newr1")
	if !strings.Contains(out, "NOT aliased") {
		t.Errorf("expected collision note in rename output:\n%s", out)
	}

	// After rename, looking up the old prefix "r1" from a neutral dir
	// should unambiguously point to envs[0] (the non-renamed repo) —
	// envs[1] should NOT claim "r1" via alias anymore.
	nonRepo := t.TempDir()
	caller := &bwEnv{
		t:           t,
		dir:         nonRepo,
		registryDir: envs[1].registryDir,
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
			"NO_COLOR=1",
			"BEADWORK_HOME="+envs[1].registryDir,
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
		),
	}

	// The injected entry points at envs[0].dir (which isn't actually a
	// beadwork repo for prefix r1), so commands will still fail — but
	// the failure should be about envs[0].dir, not a collision error.
	out2 := caller.bwFail("show", "r1-t1")
	if strings.Contains(out2, "registered for 2 repositories") {
		t.Errorf("collision should be resolved after rename:\n%s", out2)
	}
}

// TestConfigSetPrefixSameIsRejected verifies setting the same prefix errors.
func TestConfigSetPrefixSameIsRejected(t *testing.T) {
	env := newBwEnv(t)
	out := env.bwFail("config", "set", "prefix", "test")
	if !strings.Contains(out, "already") {
		t.Errorf("expected 'already' in error:\n%s", out)
	}
}

// TestConfigSetPrefixInvalidIsRejected verifies bad prefixes are rejected.
func TestConfigSetPrefixInvalidIsRejected(t *testing.T) {
	env := newBwEnv(t)
	out := env.bwFail("config", "set", "prefix", "has spaces")
	if !strings.Contains(out, "invalid") && !strings.Contains(out, "must") {
		t.Errorf("expected validation error:\n%s", out)
	}
}

// TestCrossRepoUnknownPrefixFallsThrough verifies an unrecognized prefix
// is not rewritten (command runs against current repo and fails normally).
func TestCrossRepoUnknownPrefixFallsThrough(t *testing.T) {
	env := newBwEnv(t)
	env.bw("list")

	// "bogus-1" has a prefix that's not registered anywhere. The command
	// should just run locally and fail with the local repo's normal error.
	out := env.bwFail("show", "bogus-1")
	if strings.Contains(out, "cross-repo") {
		t.Errorf("unexpected cross-repo path taken:\n%s", out)
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

	// Seed an extra registry entry for a nonexistent repo.
	path := filepath.Join(envs[0].registryDir, "registry.json")
	existing, _ := os.ReadFile(path)
	// Add a missing repo to the existing registry JSON.
	modified := strings.Replace(string(existing), `"repos": {`,
		`"repos": {
    "/nonexistent/path": {"last_seen_at": "2026-01-15T10:00:00Z"},`, 1)
	os.WriteFile(path, []byte(modified), 0644)

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

	contents := envs[0].registryContents()
	// Both repos should now have a "cursor" field.
	cursorCount := strings.Count(contents, `"cursor"`)
	if cursorCount < 2 {
		t.Errorf("expected 2 cursors after recap --all, got %d:\n%s", cursorCount, contents)
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
	env.seedRegistry(`{"schema_version":1,"repos":{"/nonexistent/repo":{"last_seen_at":"2026-01-15T10:00:00Z"}}}`)

	out := env.bw("registry", "list")
	if !strings.Contains(out, "MISSING") {
		t.Errorf("registry list should show MISSING for nonexistent path:\n%s", out)
	}
}

// TestRegistryPruneYes verifies that prune --yes removes missing entries.
func TestRegistryPruneYes(t *testing.T) {
	env := newBwEnv(t)
	env.seedRegistry(`{"schema_version":1,"repos":{"/nonexistent/repo":{"last_seen_at":"2026-01-15T10:00:00Z"}}}`)

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
	env.seedRegistry(`{"schema_version":1,"repos":{"/nonexistent/repo":{"last_seen_at":"2026-01-15T10:00:00Z"}}}`)

	out := env.bwFail("registry", "prune")
	if !strings.Contains(out, "non-interactive") {
		t.Errorf("expected non-interactive error:\n%s", out)
	}
}

// TestRegistryPruneHalfWarning verifies the half-removal warning.
func TestRegistryPruneHalfWarning(t *testing.T) {
	env := newBwEnv(t)
	// 3 out of 4 missing (real repo auto-registered = 1 existing).
	env.seedRegistry(`{"schema_version":1,"repos":{"/missing1":{"last_seen_at":"2026-01-15T10:00:00Z"},"/missing2":{"last_seen_at":"2026-01-15T10:00:00Z"},"/missing3":{"last_seen_at":"2026-01-15T10:00:00Z"},"` + env.dir + `":{"last_seen_at":"2026-01-15T10:00:00Z"}}}`)

	out := env.bw("registry", "prune", "--yes")
	if !strings.Contains(out, "more than half") {
		t.Errorf("expected half-removal warning:\n%s", out)
	}
}

// TestRegistryPruneShortFlag verifies -y works as shorthand for --yes.
func TestRegistryPruneShortFlag(t *testing.T) {
	env := newBwEnv(t)
	env.seedRegistry(`{"schema_version":1,"repos":{"/nonexistent/repo":{"last_seen_at":"2026-01-15T10:00:00Z"}}}`)

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
// that starts AFTER the current cursor leaves both cursor and last_recap_at
// untouched, and prints a gap notice on stderr naming the unrendered count.
// See ADR: recap-explicit-window-conditional-advance.
func TestRecapExplicitWindowGapSkipsAdvance(t *testing.T) {
	env := newBwEnv(t)

	const (
		day1 = "2026-04-10T09:00:00Z" // cursor gets set here
		day2 = "2026-04-12T09:00:00Z" // gap commit (older than window, newer than cursor)
		day3 = "2026-04-14T12:00:00Z" // explicit "today" is day3 midnight → day3 noon
	)

	// Day 1: create an issue, run bare recap → cursor advances to this commit.
	env.bwAtClock(day1, "create", "gap-origin", "--id", "gp-1")
	env.bwAtClock(day1, "recap")
	cursor1 := registryField(t, env.registryContents(), "cursor")
	lastRecap1 := registryField(t, env.registryContents(), "last_recap_at")
	if cursor1 == "" {
		t.Fatalf("day1 bare recap did not stamp cursor:\n%s", env.registryContents())
	}
	if lastRecap1 == "" {
		t.Fatalf("day1 bare recap did not stamp last_recap_at:\n%s", env.registryContents())
	}

	// Day 2: another commit lands. This is the "gap" — it's newer than the
	// day-1 cursor and older than the day-3 "today" window.
	env.bwAtClock(day2, "create", "gap-middle", "--id", "gp-2")

	// Day 3: explicit `recap today`. Window = day3 00:00 → day3 12:00. The
	// day-2 gp-2 commit is in the gap. gp-2 is NOT rendered, cursor must
	// NOT advance, stderr must carry the gap notice.
	stdout, stderr := env.bwAtClockCapture(day3, "recap", "today")

	// Output: gp-2 should not appear (it's older than the window).
	if strings.Contains(stdout, "gp-2") {
		t.Errorf("explicit window should not render gap commit gp-2:\n%s", stdout)
	}

	// Stderr notice must mention the gap count (1).
	if !strings.Contains(stderr, "1 commit older than this window") {
		t.Errorf("expected gap notice on stderr, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "bw recap") {
		t.Errorf("gap notice should reference 'bw recap':\n%s", stderr)
	}

	// Registry: cursor + last_recap_at must still match the day-1 snapshot.
	// (last_seen_at moves on every command via the auto-register hook; that's
	// orthogonal to the recap-stamp behavior under test.)
	cursor3 := registryField(t, env.registryContents(), "cursor")
	lastRecap3 := registryField(t, env.registryContents(), "last_recap_at")
	if cursor3 != cursor1 {
		t.Errorf("gapped explicit run advanced cursor\n  was: %s\n  now: %s", cursor1, cursor3)
	}
	if lastRecap3 != lastRecap1 {
		t.Errorf("gapped explicit run stamped last_recap_at\n  was: %s\n  now: %s", lastRecap1, lastRecap3)
	}
}

// registryField extracts a top-level string field from a single-repo
// registry JSON blob. Test helper only — assumes exactly one repo entry.
func registryField(t *testing.T, raw, field string) string {
	t.Helper()
	needle := `"` + field + `":`
	i := strings.Index(raw, needle)
	if i < 0 {
		return ""
	}
	rest := raw[i+len(needle):]
	// Skip whitespace then expect a quoted string.
	for len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t' || rest[0] == '\n') {
		rest = rest[1:]
	}
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	rest = rest[1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// TestRecapExplicitWindowNoGapAdvances verifies that an explicit window that
// covers the full unseen range (window.Start <= cursor_time) advances the
// cursor to HEAD exactly like a bare recap.
func TestRecapExplicitWindowNoGapAdvances(t *testing.T) {
	env := newBwEnv(t)

	const (
		day1 = "2026-04-14T03:00:00Z" // cursor here, inside day-1
		day2 = "2026-04-14T05:00:00Z" // new commit, later same day
		now  = "2026-04-14T12:00:00Z" // `recap today` fires here → window covers both
	)

	env.bwAtClock(day1, "create", "first", "--id", "ng-1")
	env.bwAtClock(day1, "recap")
	before := env.registryContents()

	env.bwAtClock(day2, "create", "second", "--id", "ng-2")

	// `recap today` at noon. Window start is midnight (before day1 cursor),
	// so no gap — cursor should advance and stderr should be clean.
	stdout, stderr := env.bwAtClockCapture(now, "recap", "today")
	if !strings.Contains(stdout, "ng-2") {
		t.Errorf("no-gap explicit recap missing ng-2:\n%s", stdout)
	}
	if strings.Contains(stderr, "older than this window") {
		t.Errorf("no-gap explicit recap should not emit gap notice:\n%s", stderr)
	}

	after := env.registryContents()
	if after == before {
		t.Errorf("no-gap explicit recap should advance cursor; registry unchanged:\n%s", after)
	}
}
