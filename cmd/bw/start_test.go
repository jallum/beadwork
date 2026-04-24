package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdStartBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Start me", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdStart: %v", err)
	}
	out := buf.String()

	// Should show issue context (like show)
	if !strings.Contains(out, "Start me") {
		t.Errorf("output missing title: %q", out)
	}
	// Assignee is set on the issue but not displayed in the summary heading.

	// Should show landing instructions
	if !strings.Contains(out, "LANDING THE WORK") {
		t.Errorf("output missing LANDING THE WORK: %q", out)
	}
	if !strings.Contains(out, "bw close") {
		t.Errorf("output missing close instruction: %q", out)
	}
	if !strings.Contains(out, iss.ID) {
		t.Errorf("output missing issue ID in instructions: %q", out)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "in_progress" {
		t.Errorf("status = %q, want in_progress", got.Status)
	}
	if got.Assignee != "alice" {
		t.Errorf("assignee = %q, want alice", got.Assignee)
	}
}

func TestCmdStartJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Start JSON", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{iss.ID, "--assignee", "bob", "--json"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdStart --json: %v", err)
	}

	var got issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got.Status != "in_progress" {
		t.Errorf("status = %q, want in_progress", got.Status)
	}
	if got.Assignee != "bob" {
		t.Errorf("assignee = %q, want bob", got.Assignee)
	}
}

func TestCmdStartShowsDescription(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Described issue", issue.CreateOpts{
		Description: "This is the full description of the work.",
	})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdStart: %v", err)
	}
	if !strings.Contains(buf.String(), "full description") {
		t.Errorf("output missing description: %q", buf.String())
	}
}

func TestCmdStartShowsComments(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Commented issue", issue.CreateOpts{})
	env.Store.Comment(iss.ID, "Previous session note", "bot")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdStart: %v", err)
	}
	if !strings.Contains(buf.String(), "Previous session note") {
		t.Errorf("output missing comment: %q", buf.String())
	}
}

func TestCmdStartPRHint(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Configure PR review workflow
	env.Repo.SetConfig("workflow.review", "pr")
	env.Repo.Commit("config workflow.review=pr")

	iss, _ := env.Store.Create("PR issue", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdStart: %v", err)
	}
	if !strings.Contains(buf.String(), "PR") {
		t.Errorf("output missing PR hint: %q", buf.String())
	}
}

func TestCmdStartNoPRHintByDefault(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Simple issue", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdStart: %v", err)
	}
	if strings.Contains(buf.String(), "open a PR") {
		t.Errorf("output should not mention PR without config: %q", buf.String())
	}
}

func TestCmdStartEpicLanding(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Epic issue", issue.CreateOpts{Type: "epic"})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdStart: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "children") {
		t.Errorf("epic output should mention children: %q", out)
	}
	if strings.Contains(out, "Commit only") {
		t.Errorf("epic output should not mention committing changes: %q", out)
	}
}

func TestCmdStartTaskLanding(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Task issue", issue.CreateOpts{Type: "task"})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdStart: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Commit only") {
		t.Errorf("task output should mention committing: %q", out)
	}
	if !strings.Contains(out, "Land this ticket") {
		t.Errorf("task output should say land this ticket: %q", out)
	}
}

func TestCmdStartBlocked(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{b.ID, "--assignee", "alice"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Fatal("expected error for blocked issue")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("error = %q, want mention of blocked", err.Error())
	}
	if !strings.Contains(err.Error(), a.ID) {
		t.Errorf("error = %q, want blocker ID %s", err.Error(), a.ID)
	}
}

func TestCmdStartAlreadyInProgress(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Already started", issue.CreateOpts{})
	status := "in_progress"
	env.Store.Update(iss.ID, issue.UpdateOpts{Status: &status})
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Fatal("expected error for in_progress issue")
	}
}

func TestCmdStartClosed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Closed issue", issue.CreateOpts{})
	env.Store.Close(iss.ID, "")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Fatal("expected error for closed issue")
	}
}

func TestCmdStartNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{"nonexistent", "--assignee", "alice"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

func TestCmdStartNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, nil, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for missing args")
	}
}

func TestCmdStartEpicShowsChildren(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic parent", issue.CreateOpts{Type: "epic"})
	env.Store.Create("Child task A", issue.CreateOpts{Type: "task", Parent: epic.ID})
	env.Store.Create("Child task B", issue.CreateOpts{Type: "task", Parent: epic.ID})
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{epic.ID, "--assignee", "alice"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdStart: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Child task A") {
		t.Errorf("epic output missing child A: %q", out)
	}
	if !strings.Contains(out, "Child task B") {
		t.Errorf("epic output missing child B: %q", out)
	}
}

func TestCmdStartTaskShowsMap(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("First task", issue.CreateOpts{})
	b, _ := env.Store.Create("Second task", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID) // a blocks b
	env.Repo.Commit("setup")

	// Start "a" — it unblocks "b", so the map should mention "b".
	var buf bytes.Buffer
	_, err := cmdStart(env.Store, []string{a.ID, "--assignee", "alice"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdStart: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, b.ID) {
		t.Errorf("task output missing unblocked issue %s: %q", b.ID, out)
	}
	if !strings.Contains(out, "Second task") {
		t.Errorf("task output missing unblocked title: %q", out)
	}
}
