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
	_, err := cmdReady(env.Store, []string{}, PlainWriter(&buf), nil)
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
	_, err := cmdReady(env.Store, []string{"--json"}, PlainWriter(&buf), nil)
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
	_, err := cmdReady(env.Store, []string{}, PlainWriter(&buf), nil)
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
	_, err := cmdReady(env.Store, []string{}, PlainWriter(&buf), nil)
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
	_, err := cmdReady(env.Store, []string{}, PlainWriter(&buf), nil)
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
	_, err := cmdReady(env.Store, []string{}, PlainWriter(&buf), nil)
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
	_, err := cmdReady(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()

	if strings.Contains(out, "[blocks:") || strings.Contains(out, "[blocked by:") {
		t.Errorf("standalone ready issue should not show dep brackets: %q", out)
	}
}

func TestCmdReadyEpicAppearsChildrenSuppressed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic task", issue.CreateOpts{})
	child1, _ := env.Store.Create("Child one", issue.CreateOpts{Parent: epic.ID})
	child2, _ := env.Store.Create("Child two", issue.CreateOpts{Parent: epic.ID})
	env.Repo.Commit("create epic with children")

	var buf bytes.Buffer
	_, err := cmdReady(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, epic.ID) {
		t.Errorf("epic %s should appear in ready output:\n%s", epic.ID, out)
	}
	// Children are part of the root ticket and should not appear individually.
	if strings.Contains(out, child1.ID) {
		t.Errorf("child1 %s should NOT appear in ready output:\n%s", child1.ID, out)
	}
	if strings.Contains(out, child2.ID) {
		t.Errorf("child2 %s should NOT appear in ready output:\n%s", child2.ID, out)
	}
}

func TestCmdReadyChildOfClosedParentStillShown(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create an epic and close it, but leave a child open.
	// The closed parent has no children loaded in analyzeSubtrees (it's closed),
	// so the child is not a descendant — it appears as a standalone ready item.
	epic, _ := env.Store.Create("Closed epic", issue.CreateOpts{})
	child, _ := env.Store.Create("Orphaned child", issue.CreateOpts{Parent: epic.ID})
	env.Store.Close(epic.ID, "done")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdReady(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, child.ID) {
		t.Fatalf("child %s should appear in ready output:\n%s", child.ID, out)
	}
}

func TestCmdReadyStandaloneIssuesFlat(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Standalone A", issue.CreateOpts{})
	b, _ := env.Store.Create("Standalone B", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	_, err := cmdReady(env.Store, []string{}, PlainWriter(&buf), nil)
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
	_, err := cmdReady(env.Store, []string{"--json"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdReady --json: %v", err)
	}

	var issues []issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &issues); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	// JSON should be a flat array — no nesting.
	// Children are suppressed as descendants, so only epic + standalone = 2.
	if len(issues) != 2 {
		t.Errorf("expected 2 issues in flat JSON (epic + standalone), got %d", len(issues))
	}
}

func TestCmdReadyNoBlankLinesBetweenIssues(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Issue one", issue.CreateOpts{})
	env.Store.Create("Issue two", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	_, err := cmdReady(env.Store, []string{"--no-context"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := strings.TrimRight(buf.String(), " \n")

	// No blank lines between issues (PlainWriter has no footer, context suppressed)
	if strings.Contains(out, "\n\n") {
		t.Errorf("issues should not be separated by blank lines, got:\n%s", out)
	}
}

func TestCmdReadyTTYHasFooter(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Ready task", issue.CreateOpts{})
	env.Repo.Commit("create")

	var buf bytes.Buffer
	_, err := cmdReady(env.Store, []string{}, ColorWriter(&buf, 80), nil)
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := stripANSI(buf.String())
	if !strings.Contains(out, "---") {
		t.Errorf("TTY output should have separator: %q", out)
	}
	if !strings.Contains(out, "Ready:") {
		t.Errorf("TTY output should have count: %q", out)
	}
}

func TestCmdReadyPlainNoFooter(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Ready task", issue.CreateOpts{})
	env.Repo.Commit("create")

	var buf bytes.Buffer
	_, err := cmdReady(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "---") {
		t.Errorf("PlainWriter output should NOT have separator: %q", out)
	}
}

func TestCmdReadyScopedToParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic task", issue.CreateOpts{})
	childA, _ := env.Store.Create("Child A", issue.CreateOpts{Parent: epic.ID})
	childB, _ := env.Store.Create("Child B", issue.CreateOpts{Parent: epic.ID})
	childC, _ := env.Store.Create("Child C", issue.CreateOpts{Parent: epic.ID})
	env.Store.Link(childA.ID, childC.ID)
	env.Repo.Commit("create epic with children")

	var buf bytes.Buffer
	_, err := cmdReady(env.Store, []string{epic.ID}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdReady scoped: %v", err)
	}
	out := buf.String()

	// Parent should NOT appear
	if strings.Contains(out, "Epic task") {
		t.Errorf("parent should NOT appear in scoped output: %q", out)
	}
	// childA and childB are ready
	if !strings.Contains(out, childA.ID) {
		t.Errorf("childA should appear: %q", out)
	}
	if !strings.Contains(out, childB.ID) {
		t.Errorf("childB should appear: %q", out)
	}
	// childC blocked — its title should not appear as its own line.
	// (childC.ID may appear in childA's [blocks: ...] annotation, so check title.)
	if strings.Contains(out, "Child C") {
		t.Errorf("childC should NOT appear as a ready issue (blocked): %q", out)
	}
}

func TestCmdReadyScopedJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{})
	env.Store.Create("Child A", issue.CreateOpts{Parent: epic.ID})
	env.Store.Create("Child B", issue.CreateOpts{Parent: epic.ID})
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdReady(env.Store, []string{epic.ID, "--json"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdReady scoped --json: %v", err)
	}

	var issues []issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &issues); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
}

func TestCmdReadyScopedEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{})
	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: epic.ID})
	env.Store.Close(child.ID, "done")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdReady(env.Store, []string{epic.ID}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdReady scoped: %v", err)
	}
	if !strings.Contains(buf.String(), "no ready issues") {
		t.Errorf("expected 'no ready issues', got: %q", buf.String())
	}
}
