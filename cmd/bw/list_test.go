package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdListBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Issue A", issue.CreateOpts{})
	env.Store.Create("Issue B", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	_, err := cmdList(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	if !strings.Contains(buf.String(), "Issue A") || !strings.Contains(buf.String(), "Issue B") {
		t.Errorf("output missing issues: %q", buf.String())
	}
}

func TestCmdListJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("JSON list", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	_, err := cmdList(env.Store, []string{"--json"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}

	var issues []issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &issues); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(issues) == 0 {
		t.Error("expected issues in JSON output")
	}
}

func TestCmdListEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdList(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	if !strings.Contains(buf.String(), "no issues found") {
		t.Errorf("output = %q, want 'no issues found'", buf.String())
	}
}

func TestCmdListFilterByStatus(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To close", issue.CreateOpts{})
	env.Store.Close(iss.ID, "")
	env.Repo.Commit("create and close")

	var buf bytes.Buffer
	_, err := cmdList(env.Store, []string{"--status", "closed"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	if !strings.Contains(buf.String(), "To close") {
		t.Errorf("output missing closed issue: %q", buf.String())
	}
}

func TestCmdListDefaultShowsOpenAndInProgress(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Open task", issue.CreateOpts{})
	b, _ := env.Store.Create("WIP task", issue.CreateOpts{})
	statusIP := "in_progress"
	env.Store.Update(b.ID, issue.UpdateOpts{Status: &statusIP})
	c, _ := env.Store.Create("Closed task", issue.CreateOpts{})
	env.Store.Close(c.ID, "")
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	_, err := cmdList(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Open task") {
		t.Errorf("default list should show open task: %q", out)
	}
	if !strings.Contains(out, "WIP task") {
		t.Errorf("default list should show in_progress task: %q", out)
	}
	if strings.Contains(out, "Closed task") {
		t.Error("default list should NOT show closed task")
	}
}

func TestCmdListGrep(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Login page broken", issue.CreateOpts{Description: "The form is blank"})
	env.Store.Create("Update readme", issue.CreateOpts{Description: "Add auth instructions"})
	env.Store.Create("Fix sidebar", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	_, err := cmdList(env.Store, []string{"--grep", "login"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList --grep: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Login page broken") {
		t.Errorf("--grep login should match: %q", out)
	}
	if strings.Contains(out, "Update readme") {
		t.Error("--grep login should NOT match 'Update readme'")
	}
}

func TestCmdListGrepShortFlag(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Login page broken", issue.CreateOpts{})
	env.Store.Create("Fix sidebar", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	_, err := cmdList(env.Store, []string{"-g", "login"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList -g: %v", err)
	}
	if !strings.Contains(buf.String(), "Login page broken") {
		t.Errorf("-g login should match: %q", buf.String())
	}
}

func TestCmdListShowsDeps(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker task", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked task", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("create and link")

	var buf bytes.Buffer
	_, err := cmdList(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	out := buf.String()

	// Blocker should show [blocks: <b.ID>]
	if !strings.Contains(out, "[blocks: "+b.ID+"]") {
		t.Errorf("output should show blocks for blocker: %q", out)
	}

	// Blocked should show [blocked by: <a.ID>]
	if !strings.Contains(out, "[blocked by: "+a.ID+"]") {
		t.Errorf("output should show blocked by for blocked task: %q", out)
	}
}

func TestCmdListNoDepsNoBrackets(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Standalone task", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	_, err := cmdList(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	out := buf.String()

	if strings.Contains(out, "[blocks:") || strings.Contains(out, "[blocked by:") {
		t.Errorf("standalone issue should not show dep brackets: %q", out)
	}
}

func TestCmdListShowsMultipleDeps(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker A", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocker B", issue.CreateOpts{})
	c, _ := env.Store.Create("Blocked by both", issue.CreateOpts{})
	env.Store.Link(a.ID, c.ID)
	env.Store.Link(b.ID, c.ID)
	env.Repo.Commit("create and link")

	var buf bytes.Buffer
	_, err := cmdList(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	out := buf.String()

	// C should show both blockers (order is sorted alphabetically)
	if !strings.Contains(out, "[blocked by:") {
		t.Errorf("output should show blocked by for C: %q", out)
	}
	if !strings.Contains(out, a.ID) {
		t.Errorf("output should mention blocker A (%s): %q", a.ID, out)
	}
	if !strings.Contains(out, b.ID) {
		t.Errorf("output should mention blocker B (%s): %q", b.ID, out)
	}
}

func TestCmdListOverdue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Set BW_CLOCK to 2027-06-15 so we can control what is overdue
	t.Setenv("BW_CLOCK", "2027-06-15T12:00:00Z")

	// Create an overdue issue (due in the past relative to BW_CLOCK)
	env.Store.Create("Overdue task", issue.CreateOpts{Due: "2027-06-01"})
	// Create a non-overdue issue (due in the future)
	env.Store.Create("Future task", issue.CreateOpts{Due: "2027-12-01"})
	// Create an issue with no due date
	env.Store.Create("No due date", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	// --overdue should show only the overdue issue
	var buf bytes.Buffer
	_, err := cmdList(env.Store, []string{"--overdue"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList --overdue: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Overdue task") {
		t.Errorf("--overdue should show overdue task: %q", out)
	}
	if strings.Contains(out, "Future task") {
		t.Error("--overdue should NOT show future task")
	}
	if strings.Contains(out, "No due date") {
		t.Error("--overdue should NOT show issue with no due date")
	}
}

func TestLsAliasExists(t *testing.T) {
	cmd, ok := commandMap["ls"]
	if !ok {
		t.Fatal("ls alias not registered in commandMap")
	}
	if cmd.Name != "list" {
		t.Errorf("ls alias points to %q, want list", cmd.Name)
	}
}

func TestCmdListFilterByParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic task", issue.CreateOpts{})
	env.Store.Create("Child A", issue.CreateOpts{Parent: epic.ID})
	env.Store.Create("Child B", issue.CreateOpts{Parent: epic.ID})
	env.Store.Create("Unrelated", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	_, err := cmdList(env.Store, []string{"--parent", epic.ID}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList --parent: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Child A") || !strings.Contains(out, "Child B") {
		t.Errorf("--parent should show children: %q", out)
	}
	if strings.Contains(out, "Unrelated") {
		t.Errorf("--parent should NOT show unrelated issues: %q", out)
	}
	if strings.Contains(out, "Epic task") {
		t.Errorf("--parent should NOT show the epic itself: %q", out)
	}
}
