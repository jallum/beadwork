package main_test

import (
	"encoding/json"
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

// --- Export command ---

func TestExportFormat(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Export me", issue.CreateOpts{
		Priority:    2,
		Type:        "bug",
		Description: "Something broke",
		Assignee:    "agent-1",
	})
	env.CommitIntent("create " + iss.ID)

	out := bw(t, env.Dir, "export")
	lines := nonEmptyLines(out)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d:\n%s", len(lines), out)
	}

	var rec map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Verify beads field names
	if rec["id"] != iss.ID {
		t.Errorf("id = %v, want %s", rec["id"], iss.ID)
	}
	if rec["title"] != "Export me" {
		t.Errorf("title = %v", rec["title"])
	}
	if rec["issue_type"] != "bug" {
		t.Errorf("issue_type = %v, want bug", rec["issue_type"])
	}
	if rec["owner"] != "agent-1" {
		t.Errorf("owner = %v, want agent-1", rec["owner"])
	}
	if rec["status"] != "open" {
		t.Errorf("status = %v, want open", rec["status"])
	}
	if rec["description"] != "Something broke" {
		t.Errorf("description = %v", rec["description"])
	}
	if int(rec["priority"].(float64)) != 2 {
		t.Errorf("priority = %v, want 2", rec["priority"])
	}
	if _, ok := rec["created_at"]; !ok {
		t.Error("missing created_at field")
	}
}

func TestExportFilterByStatus(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open issue", issue.CreateOpts{})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Closed issue", issue.CreateOpts{})
	env.Store.Close(b.ID)
	env.CommitIntent("create and close " + b.ID)

	out := bw(t, env.Dir, "export", "--status", "open")
	lines := nonEmptyLines(out)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d:\n%s", len(lines), out)
	}
	assertContains(t, lines[0], a.ID)
	assertNotContains(t, lines[0], b.ID)
}

func TestExportDependencies(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	out := bw(t, env.Dir, "export")
	// The blocked issue should have a dependency entry
	for _, line := range nonEmptyLines(out) {
		var rec map[string]interface{}
		json.Unmarshal([]byte(line), &rec)
		if rec["id"] == b.ID {
			deps, ok := rec["dependencies"].([]interface{})
			if !ok || len(deps) == 0 {
				t.Errorf("blocked issue missing dependencies")
				break
			}
			dep := deps[0].(map[string]interface{})
			if dep["depends_on_id"] != a.ID {
				t.Errorf("depends_on_id = %v, want %s", dep["depends_on_id"], a.ID)
			}
			if dep["type"] != "blocks" {
				t.Errorf("dep type = %v, want blocks", dep["type"])
			}
		}
	}
}

// --- Import command ---

func TestImportFromFile(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	jsonl := `{"id":"ext-001","title":"Imported task","status":"open","priority":2,"issue_type":"task","description":"From external","owner":"someone","created_at":"2026-01-01T00:00:00Z"}
{"id":"ext-002","title":"Another import","status":"closed","priority":1,"issue_type":"bug","created_at":"2026-01-02T00:00:00Z"}`

	f := writeTemp(t, env.Dir, jsonl)

	out := bw(t, env.Dir, "import", f)
	assertContains(t, out, "imported 2")

	// Verify issues exist
	showOut := bw(t, env.Dir, "show", "ext-001")
	assertContains(t, showOut, "Imported task")
	assertContains(t, showOut, "OPEN")

	showOut2 := bw(t, env.Dir, "show", "ext-002")
	assertContains(t, showOut2, "Another import")
	assertContains(t, showOut2, "CLOSED")
}

func TestImportPreservesIDs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	jsonl := `{"id":"pma-abc","title":"Foreign prefix","status":"open","priority":3,"issue_type":"task","created_at":"2026-01-01T00:00:00Z"}`
	f := writeTemp(t, env.Dir, jsonl)

	bw(t, env.Dir, "import", f)

	// ID should be preserved as pma-abc, not rewritten to test-abc
	out := bw(t, env.Dir, "show", "pma-abc")
	assertContains(t, out, "pma-abc")
	assertContains(t, out, "Foreign prefix")
}

func TestImportDependencies(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	jsonl := `{"id":"dep-aaa","title":"Blocker","status":"open","priority":1,"issue_type":"task","created_at":"2026-01-01T00:00:00Z"}
{"id":"dep-bbb","title":"Blocked","status":"open","priority":2,"issue_type":"task","created_at":"2026-01-01T00:00:00Z","dependencies":[{"issue_id":"dep-bbb","depends_on_id":"dep-aaa","type":"blocks","created_at":"2026-01-01T00:00:00Z","created_by":"test","metadata":"{}"}]}
{"id":"dep-ccc","title":"Child","status":"open","priority":3,"issue_type":"task","created_at":"2026-01-01T00:00:00Z","dependencies":[{"issue_id":"dep-ccc","depends_on_id":"dep-aaa","type":"parent-child","created_at":"2026-01-01T00:00:00Z","created_by":"test","metadata":"{}"}]}`
	f := writeTemp(t, env.Dir, jsonl)

	bw(t, env.Dir, "import", f)

	// Check blocks relationship
	outB := bw(t, env.Dir, "show", "dep-bbb")
	assertContains(t, outB, "Blocked by: dep-aaa")

	outA := bw(t, env.Dir, "show", "dep-aaa")
	assertContains(t, outA, "Blocks: dep-bbb")

	// Check parent relationship
	outC := bw(t, env.Dir, "show", "dep-ccc")
	assertContains(t, outC, "Parent: dep-aaa")
}

func TestImportCollisionSkips(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Pre-create an issue with the same ID
	iss, _ := env.Store.Create("Existing", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	jsonl := `{"id":"` + iss.ID + `","title":"Duplicate","status":"open","priority":1,"issue_type":"task","created_at":"2026-01-01T00:00:00Z"}
{"id":"new-001","title":"Fresh import","status":"open","priority":2,"issue_type":"task","created_at":"2026-01-01T00:00:00Z"}`
	f := writeTemp(t, env.Dir, jsonl)

	out := bw(t, env.Dir, "import", f)
	assertContains(t, out, "skipping")
	assertContains(t, out, "imported 1")

	// The existing issue should be unchanged
	showOut := bw(t, env.Dir, "show", iss.ID)
	assertContains(t, showOut, "Existing")
	assertNotContains(t, showOut, "Duplicate")

	// The new issue should exist
	showNew := bw(t, env.Dir, "show", "new-001")
	assertContains(t, showNew, "Fresh import")
}

func TestImportDryRun(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	jsonl := `{"id":"dry-001","title":"Should not exist","status":"open","priority":3,"issue_type":"task","created_at":"2026-01-01T00:00:00Z"}`
	f := writeTemp(t, env.Dir, jsonl)

	out := bw(t, env.Dir, "import", f, "--dry-run")
	assertContains(t, out, "dry run")

	// Issue should NOT have been created
	bwFail(t, env.Dir, "show", "dry-001")
}

func TestImportRoundtrip(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create issues with various properties
	a, _ := env.Store.Create("Roundtrip A", issue.CreateOpts{Priority: 1, Type: "bug", Assignee: "alice", Description: "desc A"})
	b, _ := env.Store.Create("Roundtrip B", issue.CreateOpts{Priority: 2, Type: "task"})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(b.ID)
	env.CommitIntent("setup roundtrip")

	// Export
	exported := bw(t, env.Dir, "export")
	exportLines := nonEmptyLines(exported)
	if len(exportLines) != 2 {
		t.Fatalf("expected 2 export lines, got %d", len(exportLines))
	}

	// Write to file and import into a fresh env
	env2 := testutil.NewEnv(t)
	defer env2.Cleanup()

	f := writeTemp(t, env2.Dir, exported)
	bw(t, env2.Dir, "import", f)

	// Verify issues exist with correct properties
	showA := bw(t, env2.Dir, "show", a.ID)
	assertContains(t, showA, "Roundtrip A")
	assertContains(t, showA, "P1")
	assertContains(t, showA, "Assignee: alice")
	assertContains(t, showA, "desc A")

	showB := bw(t, env2.Dir, "show", b.ID)
	assertContains(t, showB, "Roundtrip B")
	assertContains(t, showB, "CLOSED")
}

// --- helpers ---

func writeTemp(t *testing.T, dir, content string) string {
	t.Helper()
	f := filepath.Join(dir, "test.jsonl")
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return f
}

func nonEmptyLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

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
