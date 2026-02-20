package main_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
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

func TestListDefaultsToOpen(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	open, _ := env.Store.Create("Open one", issue.CreateOpts{})
	closed, _ := env.Store.Create("Closed one", issue.CreateOpts{})
	env.Store.Close(closed.ID)
	env.CommitIntent("setup")

	out := bw(t, env.Dir, "list")
	assertContains(t, out, open.ID)
	assertNotContains(t, out, closed.ID)

	// --all shows both
	outAll := bw(t, env.Dir, "list", "--all")
	assertContains(t, outAll, open.ID)
	assertContains(t, outAll, closed.ID)

	// --status closed shows only closed
	outClosed := bw(t, env.Dir, "list", "--status", "closed")
	assertNotContains(t, outClosed, open.ID)
	assertContains(t, outClosed, closed.ID)
}

func TestListDefaultLimit(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	for i := 0; i < 12; i++ {
		env.Store.Create(fmt.Sprintf("Issue %d", i), issue.CreateOpts{})
	}
	env.CommitIntent("create 12 issues")

	out := bw(t, env.Dir, "list")
	lines := nonEmptyLines(out)
	// 10 issues + 1 "... and N more" line
	issueLines := 0
	for _, l := range lines {
		if strings.Contains(l, "test-") {
			issueLines++
		}
	}
	if issueLines != 10 {
		t.Errorf("expected 10 issue lines, got %d:\n%s", issueLines, out)
	}
	assertContains(t, out, "... and 2 more")

	// --all removes the limit
	outAll := bw(t, env.Dir, "list", "--all")
	allLines := 0
	for _, l := range nonEmptyLines(outAll) {
		if strings.Contains(l, "test-") {
			allLines++
		}
	}
	if allLines != 12 {
		t.Errorf("expected 12 issue lines with --all, got %d", allLines)
	}

	// --limit 5 overrides
	outLimit := bw(t, env.Dir, "list", "--limit", "5")
	limitLines := 0
	for _, l := range nonEmptyLines(outLimit) {
		if strings.Contains(l, "test-") {
			limitLines++
		}
	}
	if limitLines != 5 {
		t.Errorf("expected 5 issue lines with --limit 5, got %d", limitLines)
	}
	assertContains(t, outLimit, "... and 7 more")

	// --limit 0 means no limit
	outZero := bw(t, env.Dir, "list", "--limit", "0")
	zeroLines := 0
	for _, l := range nonEmptyLines(outZero) {
		if strings.Contains(l, "test-") {
			zeroLines++
		}
	}
	if zeroLines != 12 {
		t.Errorf("expected 12 issue lines with --limit 0, got %d", zeroLines)
	}
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

// --- Close/Reopen ---

func TestCloseWithReason(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Close with reason", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	out := bw(t, env.Dir, "close", iss.ID, "--reason", "completed")
	assertContains(t, out, "closed "+iss.ID)

	// Verify it's actually closed
	show := bw(t, env.Dir, "show", iss.ID, "--json")
	assertContains(t, show, `"status": "closed"`)
}

func TestCloseNonExistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "close", "test-zzzz")
	assertContains(t, out, "no issue found")
}

func TestCloseNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "close")
	assertContains(t, out, "usage:")
}

func TestCloseAlreadyClosedCLI(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Already closed", issue.CreateOpts{})
	env.Store.Close(iss.ID)
	env.CommitIntent("close " + iss.ID)

	out := bwFail(t, env.Dir, "close", iss.ID)
	assertContains(t, out, "already closed")
}

func TestReopenOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Closed task", issue.CreateOpts{})
	env.Store.Close(iss.ID)
	env.CommitIntent("close " + iss.ID)

	out := bw(t, env.Dir, "reopen", iss.ID)
	assertContains(t, out, "reopened "+iss.ID)

	// Verify reopened
	show := bw(t, env.Dir, "show", iss.ID, "--json")
	assertContains(t, show, `"status": "open"`)
}

func TestReopenNotClosedCLI(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Open task", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	out := bwFail(t, env.Dir, "reopen", iss.ID)
	assertContains(t, out, "not closed")
}

func TestReopenNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "reopen")
	assertContains(t, out, "usage:")
}

// --- Label ---

func TestLabelAddOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Label test", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	out := bw(t, env.Dir, "label", iss.ID, "+bug", "+urgent")
	assertContains(t, out, "labeled "+iss.ID)
	assertContains(t, out, "bug")
	assertContains(t, out, "urgent")
}

func TestLabelRemoveOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Label remove", issue.CreateOpts{})
	env.Store.Label(iss.ID, []string{"bug", "urgent"}, nil)
	env.CommitIntent("setup " + iss.ID)

	out := bw(t, env.Dir, "label", iss.ID, "-urgent")
	assertContains(t, out, "labeled "+iss.ID)
	assertContains(t, out, "bug")
}

func TestLabelBareNameAdd(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Bare label", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// Bare label name (without +) should add the label
	out := bw(t, env.Dir, "label", iss.ID, "feature")
	assertContains(t, out, "feature")
}

func TestLabelNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "label")
	assertContains(t, out, "usage:")
}

func TestLabelOnlyID(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "label", "test-1234")
	assertContains(t, out, "usage:")
}

// --- Link/Unlink ---

func TestLinkOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.CommitIntent("setup")

	out := bw(t, env.Dir, "link", a.ID, "blocks", b.ID)
	assertContains(t, out, "linked")
	assertContains(t, out, a.ID)
	assertContains(t, out, b.ID)

	// Verify link via show
	show := bw(t, env.Dir, "show", a.ID)
	assertContains(t, show, "Blocks: "+b.ID)
}

func TestUnlinkOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup link")

	out := bw(t, env.Dir, "unlink", a.ID, "blocks", b.ID)
	assertContains(t, out, "unlinked")

	// Verify unlinked
	show := bw(t, env.Dir, "show", a.ID)
	assertNotContains(t, show, "Blocks:")
}

func TestLinkMissingBlocks(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "link", "a", "b")
	assertContains(t, out, "usage:")
}

func TestUnlinkMissingBlocks(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "unlink", "a", "b")
	assertContains(t, out, "usage:")
}

func TestLinkSelfBlockingCLI(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Self", issue.CreateOpts{})
	env.CommitIntent("create " + a.ID)

	out := bwFail(t, env.Dir, "link", a.ID, "blocks", a.ID)
	assertContains(t, out, "cannot block itself")
}

// --- Update ---

func TestUpdateTitleOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Original", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	out := bw(t, env.Dir, "update", iss.ID, "--title", "New title")
	assertContains(t, out, "updated "+iss.ID)

	show := bw(t, env.Dir, "show", iss.ID)
	assertContains(t, show, "New title")
}

func TestUpdatePriorityOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Priority test", issue.CreateOpts{Priority: 3})
	env.CommitIntent("create " + iss.ID)

	bw(t, env.Dir, "update", iss.ID, "-p", "1")
	show := bw(t, env.Dir, "show", iss.ID)
	assertContains(t, show, "P1")
}

func TestUpdateStatusOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Status test", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	bw(t, env.Dir, "update", iss.ID, "-s", "in_progress")
	show := bw(t, env.Dir, "show", iss.ID)
	assertContains(t, show, "IN_PROGRESS")
}

func TestUpdateAssigneeOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Assignee test", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	bw(t, env.Dir, "update", iss.ID, "-a", "agent-1")
	show := bw(t, env.Dir, "show", iss.ID)
	assertContains(t, show, "Assignee: agent-1")
}

func TestUpdateDescriptionOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Desc test", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	bw(t, env.Dir, "update", iss.ID, "-d", "New description")
	show := bw(t, env.Dir, "show", iss.ID)
	assertContains(t, show, "New description")
}

func TestUpdateTypeOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Type test", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	bw(t, env.Dir, "update", iss.ID, "-t", "bug")
	show := bw(t, env.Dir, "show", iss.ID)
	assertContains(t, show, "Type: bug")
}

func TestUpdateMultipleFlags(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Multi update", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	bw(t, env.Dir, "update", iss.ID, "-p", "1", "-a", "alice", "-t", "bug")
	show := bw(t, env.Dir, "show", iss.ID)
	assertContains(t, show, "P1")
	assertContains(t, show, "Assignee: alice")
	assertContains(t, show, "Type: bug")
}

func TestUpdateJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("JSON update", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	out := bw(t, env.Dir, "update", iss.ID, "--title", "Updated", "--json")
	assertContains(t, out, `"title": "Updated"`)
}

func TestUpdateNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "update")
	assertContains(t, out, "usage:")
}

func TestUpdateNonExistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "update", "test-zzzz", "--title", "X")
	assertContains(t, out, "no issue found")
}

// --- Graph ---

func TestGraphRootedOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Root", issue.CreateOpts{})
	b, _ := env.Store.Create("Child", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup graph")

	out := bw(t, env.Dir, "graph", a.ID)
	assertContains(t, out, a.ID)
	assertContains(t, out, b.ID)
}

func TestGraphAllOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	out := bw(t, env.Dir, "graph", "--all")
	assertContains(t, out, a.ID)
	assertContains(t, out, b.ID)
}

func TestGraphJSONOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Graph JSON", issue.CreateOpts{})
	b, _ := env.Store.Create("Dep", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	out := bw(t, env.Dir, "graph", a.ID, "--json")
	assertContains(t, out, `"id"`)
	assertContains(t, out, a.ID)
}

func TestGraphNoIssues(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bw(t, env.Dir, "graph", "--all")
	assertContains(t, out, "no issues")
}

func TestGraphNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "graph")
	assertContains(t, out, "issue ID required")
}

func TestGraphAllExcludesClosedUnlinked(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open issue", issue.CreateOpts{})
	b, _ := env.Store.Create("Closed unlinked", issue.CreateOpts{})
	env.Store.Close(b.ID)
	env.CommitIntent("setup")

	// --all without a root filters closed nodes that have no relationships
	out := bw(t, env.Dir, "graph", "--all")
	assertContains(t, out, a.ID)
	assertNotContains(t, out, b.ID)
}

// --- Init ---

func TestInitOutput(t *testing.T) {
	dir := t.TempDir()
	setupGitRepo(t, dir)

	out := bw(t, dir, "init", "--prefix", "myapp")
	assertContains(t, out, "initialized beadwork")
	assertContains(t, out, "prefix: myapp")
}

func TestInitAlreadyInitialized(t *testing.T) {
	dir := t.TempDir()
	setupGitRepo(t, dir)

	bw(t, dir, "init")
	out := bwFail(t, dir, "init")
	assertContains(t, out, "already initialized")
}

func TestInitForceOutput(t *testing.T) {
	dir := t.TempDir()
	setupGitRepo(t, dir)

	bw(t, dir, "init", "--prefix", "old")
	out := bw(t, dir, "init", "--force", "--prefix", "new")
	assertContains(t, out, "reinitialized beadwork")
	assertContains(t, out, "prefix: new")
}

func TestInitDerivedPrefix(t *testing.T) {
	dir := t.TempDir()
	setupGitRepo(t, dir)

	out := bw(t, dir, "init")
	assertContains(t, out, "initialized beadwork")
	assertContains(t, out, "prefix:")
}

// --- Sync (no-remote case) ---

func TestSyncNoRemoteOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bw(t, env.Dir, "sync")
	assertContains(t, out, "no remote configured")
}

// --- Prime ---

func TestPrimeOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Open task", issue.CreateOpts{})
	env.CommitIntent("create task")

	out := bw(t, env.Dir, "prime")
	assertContains(t, out, "Current State")
	assertContains(t, out, "1 open")
	assertContains(t, out, "Ready for work:")
}

func TestPrimeWithInProgress(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("In progress task", issue.CreateOpts{})
	status := "in_progress"
	env.Store.Update(iss.ID, issue.UpdateOpts{Status: &status})
	env.CommitIntent("setup")

	out := bw(t, env.Dir, "prime")
	assertContains(t, out, "1 in progress")
	assertContains(t, out, "In progress:")
}

// --- Onboard ---

func TestOnboardOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bw(t, env.Dir, "onboard")
	assertContains(t, out, "bw")
}

// --- Version ---

func TestVersionOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bw(t, env.Dir, "--version")
	assertContains(t, out, "bw ")
}

// --- List JSON ---

func TestListJSONOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("JSON list", issue.CreateOpts{Priority: 1, Type: "bug"})
	env.CommitIntent("create " + iss.ID)

	out := bw(t, env.Dir, "list", "--json")
	assertContains(t, out, `"title": "JSON list"`)
	assertContains(t, out, `"priority": 1`)
}

func TestListEmptyOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bw(t, env.Dir, "list")
	assertContains(t, out, "no issues found")
}

func TestListFilterByType(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Bug one", issue.CreateOpts{Type: "bug"})
	env.Store.Create("Task one", issue.CreateOpts{Type: "task"})
	env.CommitIntent("setup")

	out := bw(t, env.Dir, "list", "--type", "bug")
	assertContains(t, out, "Bug one")
	assertNotContains(t, out, "Task one")
}

func TestListFilterByAssignee(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Alice task", issue.CreateOpts{Assignee: "alice"})
	env.Store.Create("Bob task", issue.CreateOpts{Assignee: "bob"})
	env.CommitIntent("setup")

	out := bw(t, env.Dir, "list", "--assignee", "alice")
	assertContains(t, out, "Alice task")
	assertNotContains(t, out, "Bob task")
}

func TestListFilterByPriority(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("P1 task", issue.CreateOpts{Priority: 1})
	env.Store.Create("P3 task", issue.CreateOpts{Priority: 3})
	env.CommitIntent("setup")

	out := bw(t, env.Dir, "list", "--priority", "1")
	assertContains(t, out, "P1 task")
	assertNotContains(t, out, "P3 task")
}

func TestListFilterByLabel(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Bug task", issue.CreateOpts{})
	env.Store.Label(a.ID, []string{"bug"}, nil)
	env.Store.Create("No label", issue.CreateOpts{})
	env.CommitIntent("setup")

	out := bw(t, env.Dir, "list", "--label", "bug")
	assertContains(t, out, "Bug task")
	assertNotContains(t, out, "No label")
}

// --- Ready JSON ---

func TestReadyJSONOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Ready task", issue.CreateOpts{Priority: 1})
	env.CommitIntent("setup")

	out := bw(t, env.Dir, "ready", "--json")
	assertContains(t, out, `"title": "Ready task"`)
}

// --- Create ---

func TestCreateWithAllFlags(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bw(t, env.Dir, "create", "Full issue", "-p", "1", "-t", "bug", "-a", "alice", "-d", "some description")
	assertContains(t, out, "created test-")
	assertContains(t, out, "Full issue")
}

func TestCreateNoTitle(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "create")
	assertContains(t, out, "title is required")
}

func TestCreateWithDescription(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bw(t, env.Dir, "create", "Desc test", "-d", "This is a description")
	assertContains(t, out, "created test-")

	// Extract the ID from "created test-XXXX: ..."
	parts := strings.Fields(out)
	id := ""
	for _, p := range parts {
		if strings.HasPrefix(p, "test-") {
			id = strings.TrimSuffix(p, ":")
			break
		}
	}
	if id == "" {
		t.Fatal("could not extract issue ID from output")
	}

	show := bw(t, env.Dir, "show", id)
	assertContains(t, show, "This is a description")
}

// --- Show ---

func TestShowNonExistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "show", "test-zzzz")
	assertContains(t, out, "no issue found")
}

func TestShowNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "show")
	assertContains(t, out, "usage:")
}

// --- Config ---

func TestConfigNoSubcommand(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "config")
	assertContains(t, out, "usage:")
}

func TestConfigBadSubcommand(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "config", "bad")
	assertContains(t, out, "usage:")
}

func TestConfigSetMissingArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "config", "set", "key")
	assertContains(t, out, "usage:")
}

func TestConfigGetMissingArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	out := bwFail(t, env.Dir, "config", "get")
	assertContains(t, out, "usage:")
}

// --- helpers ---

func setupGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s: %v", c, out, err)
		}
	}
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	cmd.Run()
}

func init() {
	os.Setenv("GIT_AUTHOR_NAME", "Test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@test.com")
	os.Setenv("GIT_COMMITTER_NAME", "Test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@test.com")
}
