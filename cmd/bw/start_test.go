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
	err := cmdStart([]string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdStart: %v", err)
	}
	if !strings.Contains(buf.String(), "started") {
		t.Errorf("output = %q", buf.String())
	}
	if !strings.Contains(buf.String(), "alice") {
		t.Errorf("output missing assignee: %q", buf.String())
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
	err := cmdStart([]string{iss.ID, "--assignee", "bob", "--json"}, PlainWriter(&buf))
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

func TestCmdStartBlocked(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	err := cmdStart([]string{b.ID, "--assignee", "alice"}, PlainWriter(&buf))
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
	err := cmdStart([]string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf))
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
	err := cmdStart([]string{iss.ID, "--assignee", "alice"}, PlainWriter(&buf))
	if err == nil {
		t.Fatal("expected error for closed issue")
	}
}

func TestCmdStartNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdStart([]string{"nonexistent", "--assignee", "alice"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

func TestCmdStartNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdStart(nil, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for missing args")
	}
}
