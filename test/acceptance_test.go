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
	t   *testing.T
	dir string
	env []string
}

const fixedClock = "2026-01-15T10:00:00Z"

func newBwEnv(t *testing.T) *bwEnv {
	t.Helper()
	dir := t.TempDir()

	env := &bwEnv{
		t:   t,
		dir: dir,
		env: append(os.Environ(),
			"BW_CLOCK="+fixedClock,
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

// goldenCompare runs `bw export` and compares output against a golden file.
// If UPDATE_GOLDEN=1, writes the output as the new golden file instead.
func (e *bwEnv) goldenCompare(name string) {
	e.t.Helper()
	got := e.bw("export")

	goldenPath := filepath.Join("testdata", name+".golden.jsonl")

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
			e.t.Fatalf("write golden file: %v", err)
		}
		e.t.Logf("updated golden file: %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		e.t.Fatalf("read golden file (run with UPDATE_GOLDEN=1 to create): %v", err)
	}

	if got != string(want) {
		e.t.Errorf("export output does not match golden file %s\n--- want ---\n%s\n--- got ---\n%s",
			goldenPath, string(want), got)
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
