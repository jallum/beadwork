package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdDeletePreview(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Delete me", issue.CreateOpts{Priority: intPtr(2), Type: "task"})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdDelete(env.Store, []string{iss.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdDelete preview: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "DELETE PREVIEW") {
		t.Errorf("expected DELETE PREVIEW in output, got: %q", out)
	}
	if !strings.Contains(out, iss.Title) {
		t.Errorf("expected issue title in output, got: %q", out)
	}
}

func TestCmdDeletePreviewDoesNotModify(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Keep me", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	cmdDelete(env.Store, []string{iss.ID}, PlainWriter(&buf))

	// Issue should still exist
	got, err := env.Store.Get(iss.ID)
	if err != nil {
		t.Fatalf("issue should still exist after preview: %v", err)
	}
	if got.Title != "Keep me" {
		t.Errorf("issue title changed: %q", got.Title)
	}
}

func TestCmdDeleteForce(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Delete me", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdDelete(env.Store, []string{iss.ID, "--force"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdDelete --force: %v", err)
	}
	if !strings.Contains(buf.String(), "deleted") {
		t.Errorf("expected 'deleted' in output, got: %q", buf.String())
	}

	// Issue should be gone
	_, err = env.Store.Get(iss.ID)
	if err == nil {
		t.Error("issue should not exist after --force delete")
	}
}

func TestCmdDeleteForceJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("JSON delete", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdDelete(env.Store, []string{iss.ID, "--force", "--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdDelete --force --json: %v", err)
	}

	var got issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got.Title != "JSON delete" {
		t.Errorf("title = %q, want 'JSON delete'", got.Title)
	}
}

func TestCmdDeleteNonexistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdDelete(env.Store, []string{"test-zzzz", "--force"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}
