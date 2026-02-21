package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdUpdatePriority(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Update me", issue.CreateOpts{Priority: intPtr(3)})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUpdate(env.Store, []string{iss.ID, "--priority", "1"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpdate: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Priority != 1 {
		t.Errorf("priority = %d, want 1", got.Priority)
	}
}

func TestCmdUpdateMultipleFields(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Multi update", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUpdate(env.Store, []string{iss.ID, "--assignee", "bob", "--type", "bug", "--title", "Updated title"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpdate: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Assignee != "bob" {
		t.Errorf("assignee = %q", got.Assignee)
	}
	if got.Type != "bug" {
		t.Errorf("type = %q", got.Type)
	}
	if got.Title != "Updated title" {
		t.Errorf("title = %q", got.Title)
	}
}

func TestCmdUpdateJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("JSON update", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUpdate(env.Store, []string{iss.ID, "--assignee", "alice", "--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpdate: %v", err)
	}

	var got issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got.Assignee != "alice" {
		t.Errorf("assignee = %q", got.Assignee)
	}
}

func TestCmdUpdateDescription(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Desc update", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUpdate(env.Store, []string{iss.ID, "--description", "new description"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpdate: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Description != "new description" {
		t.Errorf("description = %q, want %q", got.Description, "new description")
	}
}

func TestCmdUpdateStatus(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Status update", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUpdate(env.Store, []string{iss.ID, "--status", "in_progress"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpdate: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "in_progress" {
		t.Errorf("status = %q, want in_progress", got.Status)
	}
}

func TestCmdUpdateNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdUpdate(env.Store, []string{"nonexistent", "--title", "x"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}
