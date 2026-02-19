package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/j5n/beadwork/internal/issue"
	"github.com/j5n/beadwork/internal/testutil"
)

var bwBin string

func TestMain(m *testing.M) {
	// Build the binary once for all tests
	tmp, err := os.MkdirTemp("", "bw-test-bin")
	if err != nil {
		panic(err)
	}
	bwBin = filepath.Join(tmp, "bw")
	cmd := exec.Command("go", "build", "-o", bwBin, ".")
	cmd.Dir = filepath.Join(mustCwd(), ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

func mustCwd() string {
	d, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return d
}

// bw runs the bw binary in the test env dir and returns stdout.
func bw(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bwBin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bw %s: %s: %v", strings.Join(args, " "), out, err)
	}
	return string(out)
}

// bwFail runs the bw binary expecting a non-zero exit.
func bwFail(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bwBin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected bw %s to fail, got: %s", strings.Join(args, " "), out)
	}
	return string(out)
}

// --- Show command ---

func TestShowFormat(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Fix login timeout", issue.CreateOpts{
		Priority:    1,
		Type:        "bug",
		Description: "Users get kicked out after 30s",
		Assignee:    "agent-1",
	})
	env.CommitIntent("create " + iss.ID)

	out := bw(t, env.Dir, "show", iss.ID)

	// Header line: icon, ID, title, priority, status
	assertContains(t, out, iss.ID+" · Fix login timeout")
	assertContains(t, out, "P1 · OPEN")

	// Metadata line
	assertContains(t, out, "Assignee: agent-1 · Type: bug")

	// Created date (just the date portion)
	assertContains(t, out, "Created: ")

	// Description block
	assertContains(t, out, "DESCRIPTION")
	assertContains(t, out, "  Users get kicked out after 30s")
}

func TestShowNoAssignee(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Unassigned task", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	out := bw(t, env.Dir, "show", iss.ID)
	assertContains(t, out, "Assignee: — · Type: task")
}

func TestShowNoDescription(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("No description", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	out := bw(t, env.Dir, "show", iss.ID)
	assertNotContains(t, out, "DESCRIPTION")
}

func TestShowWithLabels(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Labeled", issue.CreateOpts{})
	env.Store.Label(iss.ID, []string{"bug", "urgent"}, nil)
	env.CommitIntent("create and label " + iss.ID)

	out := bw(t, env.Dir, "show", iss.ID)
	assertContains(t, out, "Labels: bug, urgent")
}

func TestShowWithDependencies(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup deps")

	// Check blocker shows "Blocks:"
	outA := bw(t, env.Dir, "show", a.ID)
	assertContains(t, outA, "Blocks: "+b.ID)

	// Check blocked shows "Blocked by:"
	outB := bw(t, env.Dir, "show", b.ID)
	assertContains(t, outB, "Blocked by: "+a.ID)
}

func TestShowClosedStatus(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Done task", issue.CreateOpts{})
	env.Store.Close(iss.ID)
	env.CommitIntent("create and close " + iss.ID)

	out := bw(t, env.Dir, "show", iss.ID)
	assertContains(t, out, "CLOSED")
	assertContains(t, out, "✓")
}

func TestShowJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("JSON test", issue.CreateOpts{Priority: 2, Type: "task"})
	env.CommitIntent("create " + iss.ID)

	out := bw(t, env.Dir, "show", iss.ID, "--json")
	assertContains(t, out, `"title": "JSON test"`)
	assertContains(t, out, `"priority": 2`)
}

// --- Create command ---

func TestCreateOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bw(t, env.Dir, "create", "New feature", "-p", "2", "-t", "task")
	assertContains(t, out, "created test-")
	assertContains(t, out, "New feature")
}

// --- List command ---

func TestListOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Listed issue", issue.CreateOpts{Priority: 2, Type: "bug"})
	env.CommitIntent("create " + iss.ID)

	out := bw(t, env.Dir, "list")
	assertContains(t, out, iss.ID)
	assertContains(t, out, "Listed issue")
}

// --- Ready command ---

func TestReadyOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{Priority: 1})
	b, _ := env.Store.Create("Blocked task", issue.CreateOpts{Priority: 2})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	out := bw(t, env.Dir, "ready")

	// Blocker should appear (it's ready)
	assertContains(t, out, a.ID)
	assertContains(t, out, "Blocker")

	// Blocked task should not appear
	assertNotContains(t, out, b.ID)

	// Footer
	assertContains(t, out, "Ready:")
}

func TestReadyEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// No issues at all
	out := bw(t, env.Dir, "ready")
	assertContains(t, out, "no ready issues")
}

// --- Config command ---

func TestConfigListOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bw(t, env.Dir, "config", "list")
	assertContains(t, out, "prefix=test")
}

func TestConfigGetOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bw(t, env.Dir, "config", "get", "prefix")
	if strings.TrimSpace(out) != "test" {
		t.Errorf("config get prefix = %q, want %q", strings.TrimSpace(out), "test")
	}
}

func TestConfigSetOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bw(t, env.Dir, "config", "set", "default.priority", "2")
	assertContains(t, out, "default.priority=2")

	// Verify it persisted
	out = bw(t, env.Dir, "config", "get", "default.priority")
	if strings.TrimSpace(out) != "2" {
		t.Errorf("config get default.priority = %q, want %q", strings.TrimSpace(out), "2")
	}
}

func TestConfigGetMissing(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Should exit non-zero for missing key
	bwFail(t, env.Dir, "config", "get", "nonexistent")
}

// --- Close command ---

func TestCloseOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To close", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	out := bw(t, env.Dir, "close", iss.ID)
	assertContains(t, out, "closed "+iss.ID)
	assertContains(t, out, "To close")
}

// --- Usage ---

func TestUsageOnNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir)
	assertContains(t, out, "Usage: bw <command>")
}

func TestUnknownCommand(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "bogus")
	assertContains(t, out, "unknown command: bogus")
}

// --- helpers ---

func assertContains(t *testing.T, output, substr string) {
	t.Helper()
	if !strings.Contains(output, substr) {
		t.Errorf("output does not contain %q:\n%s", substr, output)
	}
}

func assertNotContains(t *testing.T, output, substr string) {
	t.Helper()
	if strings.Contains(output, substr) {
		t.Errorf("output should not contain %q:\n%s", substr, output)
	}
}

func init() {
	os.Setenv("GIT_AUTHOR_NAME", "Test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@test.com")
	os.Setenv("GIT_COMMITTER_NAME", "Test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@test.com")
}
