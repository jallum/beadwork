package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdReadyBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Ready issue", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	err := cmdReady(env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	if !strings.Contains(buf.String(), "Ready issue") {
		t.Errorf("output missing issue: %q", buf.String())
	}
}

func TestCmdReadyJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Ready JSON", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	err := cmdReady(env.Store, []string{"--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}

	var issues []issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &issues); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(issues) == 0 {
		t.Error("expected issues in JSON output")
	}
}

func TestCmdReadyEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdReady(env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	if !strings.Contains(buf.String(), "no ready issues") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestCmdReadyExcludesInProgress(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open task", issue.CreateOpts{})
	b, _ := env.Store.Create("WIP task", issue.CreateOpts{})
	statusIP := "in_progress"
	env.Store.Update(b.ID, issue.UpdateOpts{Status: &statusIP})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	err := cmdReady(env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, a.ID) {
		t.Errorf("output should contain open task %s: %q", a.ID, out)
	}
	if strings.Contains(out, b.ID) {
		t.Errorf("output should NOT contain in_progress task %s: %q", b.ID, out)
	}
}

func TestCmdReadyExcludesDeferred(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open task", issue.CreateOpts{})
	env.Store.Create("Deferred task", issue.CreateOpts{DeferUntil: "2027-06-01"})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	err := cmdReady(env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, a.ID) {
		t.Errorf("output should contain open task %s: %q", a.ID, out)
	}
	if strings.Contains(out, "Deferred task") {
		t.Error("output should NOT contain deferred task")
	}
}

func TestCmdReadyShowsBlocksDeps(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Ready blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked task", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("create and link")

	var buf bytes.Buffer
	err := cmdReady(env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()

	// The ready issue (a) blocks b, so it should show [blocks: b.ID]
	if !strings.Contains(out, "[blocks: "+b.ID+"]") {
		t.Errorf("ready output should show blocks dep: %q", out)
	}
}

func TestCmdReadyNoDepsNoBrackets(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Standalone ready", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	err := cmdReady(env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()

	if strings.Contains(out, "[blocks:") || strings.Contains(out, "[blocked by:") {
		t.Errorf("standalone ready issue should not show dep brackets: %q", out)
	}
}

func TestCmdReadyGroupsChildrenUnderParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic task", issue.CreateOpts{})
	child1, _ := env.Store.Create("Child one", issue.CreateOpts{Parent: epic.ID})
	child2, _ := env.Store.Create("Child two", issue.CreateOpts{Parent: epic.ID})
	env.Repo.Commit("create epic with children")

	var buf bytes.Buffer
	err := cmdReady(env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()
	lines := strings.Split(out, "\n")

	// Find the epic line and verify children appear indented after it.
	// Match " epic.ID " with spaces to avoid substring matches on child IDs.
	epicIdx := -1
	child1Idx := -1
	child2Idx := -1
	for i, line := range lines {
		if strings.Contains(line, " "+epic.ID+" ") {
			epicIdx = i
		}
		if strings.Contains(line, child1.ID) {
			child1Idx = i
		}
		if strings.Contains(line, child2.ID) {
			child2Idx = i
		}
	}
	if epicIdx < 0 {
		t.Fatalf("epic %s not found in output:\n%s", epic.ID, out)
	}
	if child1Idx < 0 || child2Idx < 0 {
		t.Fatalf("children not found in output:\n%s", out)
	}
	// Children must appear after their parent.
	if child1Idx <= epicIdx || child2Idx <= epicIdx {
		t.Errorf("children should appear after parent; epicIdx=%d child1Idx=%d child2Idx=%d", epicIdx, child1Idx, child2Idx)
	}
	// Children must be indented (leading spaces).
	if !strings.HasPrefix(lines[child1Idx], "  ") {
		t.Errorf("child1 line should be indented: %q", lines[child1Idx])
	}
	if !strings.HasPrefix(lines[child2Idx], "  ") {
		t.Errorf("child2 line should be indented: %q", lines[child2Idx])
	}
	// Epic line must NOT be indented.
	if strings.HasPrefix(lines[epicIdx], "  ") {
		t.Errorf("epic line should not be indented: %q", lines[epicIdx])
	}
}

func TestCmdReadyChildUnderNonReadyParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create an epic and close it, but leave a child open.
	epic, _ := env.Store.Create("Closed epic", issue.CreateOpts{})
	child, _ := env.Store.Create("Orphaned child", issue.CreateOpts{Parent: epic.ID})
	env.Store.Close(epic.ID, "done")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	err := cmdReady(env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()
	lines := strings.Split(out, "\n")

	// The child should still appear, grouped under its parent as a header.
	// Use " epic.ID " (with spaces) to match the epic line specifically,
	// not the child line which contains the epic ID as a prefix.
	epicIdx := -1
	childIdx := -1
	for i, line := range lines {
		if strings.Contains(line, " "+epic.ID+" ") {
			epicIdx = i
		}
		if strings.Contains(line, child.ID) {
			childIdx = i
		}
	}
	if childIdx < 0 {
		t.Fatalf("child %s not found in output:\n%s", child.ID, out)
	}
	// Parent should appear as a group header even though it's not ready itself.
	if epicIdx < 0 {
		t.Fatalf("parent %s should appear as group header:\n%s", epic.ID, out)
	}
	if childIdx <= epicIdx {
		t.Errorf("child should appear after parent header; epicIdx=%d childIdx=%d", epicIdx, childIdx)
	}
	if !strings.HasPrefix(lines[childIdx], "  ") {
		t.Errorf("child line should be indented: %q", lines[childIdx])
	}
}

func TestCmdReadyStandaloneIssuesFlat(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Standalone A", issue.CreateOpts{})
	b, _ := env.Store.Create("Standalone B", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	err := cmdReady(env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()
	lines := strings.Split(out, "\n")

	// Both should appear without indentation.
	for _, line := range lines {
		if strings.Contains(line, a.ID) || strings.Contains(line, b.ID) {
			if strings.HasPrefix(line, "  ") {
				t.Errorf("standalone issue should not be indented: %q", line)
			}
		}
	}
}

func TestCmdReadyJSONFlatWithParents(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{})
	env.Store.Create("Child", issue.CreateOpts{Parent: epic.ID})
	env.Store.Create("Standalone", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	err := cmdReady(env.Store, []string{"--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdReady --json: %v", err)
	}

	var issues []issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &issues); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	// JSON should be a flat array — no nesting.
	if len(issues) != 3 {
		t.Errorf("expected 3 issues in flat JSON, got %d", len(issues))
	}
}

func TestCmdReadyGroupsNotSeparatedByBlankLine(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic1, _ := env.Store.Create("Epic one", issue.CreateOpts{})
	env.Store.Create("Child of one", issue.CreateOpts{Parent: epic1.ID})
	epic2, _ := env.Store.Create("Epic two", issue.CreateOpts{})
	env.Store.Create("Child of two", issue.CreateOpts{Parent: epic2.ID})
	env.Repo.Commit("create epics")

	var buf bytes.Buffer
	err := cmdReady(env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()

	// Find the issue lines (before the separator).
	sepIdx := strings.Index(out, "---")
	if sepIdx < 0 {
		t.Fatalf("separator not found in output:\n%s", out)
	}
	issueSection := out[:sepIdx]

	// No blank lines between groups in the issue section
	// (trim trailing whitespace to exclude the blank line before the separator).
	issueSection = strings.TrimRight(issueSection, " \n")
	if strings.Contains(issueSection, "\n\n") {
		t.Errorf("groups should not be separated by blank lines, got:\n%s", issueSection)
	}
}
