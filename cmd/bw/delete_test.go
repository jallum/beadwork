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
	_, err := cmdDelete(env.Store, []string{iss.ID}, PlainWriter(&buf), nil)
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
	cmdDelete(env.Store, []string{iss.ID}, PlainWriter(&buf), nil)

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
	_, err := cmdDelete(env.Store, []string{iss.ID, "--force"}, PlainWriter(&buf), nil)
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
	_, err := cmdDelete(env.Store, []string{iss.ID, "--force", "--json"}, PlainWriter(&buf), nil)
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
	_, err := cmdDelete(env.Store, []string{"test-zzzz", "--force"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

func TestCmdDeleteRecursivePreviewThenForce(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	c, _ := env.Store.Create("Child", issue.CreateOpts{Parent: root.ID})
	gc, _ := env.Store.Create("Grandchild", issue.CreateOpts{Parent: c.ID})
	env.Repo.Commit("setup")

	// Preview (no --force): mentions the subtree, mutates nothing.
	var preview bytes.Buffer
	if _, err := cmdDelete(env.Store, []string{root.ID, "--recursive"}, PlainWriter(&preview), nil); err != nil {
		t.Fatalf("cmdDelete --recursive (preview): %v", err)
	}
	if !strings.Contains(preview.String(), "subtree") {
		t.Errorf("preview should describe the subtree, got: %q", preview.String())
	}
	if _, err := env.Store.Get(gc.ID); err != nil {
		t.Error("preview must not delete anything")
	}

	// Execute with --force.
	var run bytes.Buffer
	if _, err := cmdDelete(env.Store, []string{root.ID, "--recursive", "--force"}, PlainWriter(&run), nil); err != nil {
		t.Fatalf("cmdDelete --recursive --force: %v", err)
	}
	if !strings.Contains(run.String(), "deleted 3 issue") {
		t.Errorf("output = %q", run.String())
	}
	for _, id := range []string{root.ID, c.ID, gc.ID} {
		if _, err := env.Store.Get(id); err == nil {
			t.Errorf("%s should be gone after recursive delete", id)
		}
	}
}

func TestCmdDeleteRecursiveAlias(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	c, _ := env.Store.Create("Child", issue.CreateOpts{Parent: root.ID})
	env.Repo.Commit("setup")

	// -r aliases --recursive; with --force it should remove the whole subtree.
	var buf bytes.Buffer
	if _, err := cmdDelete(env.Store, []string{root.ID, "-r", "--force"}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdDelete -r --force: %v", err)
	}
	if _, err := env.Store.Get(c.ID); err == nil {
		t.Error("child should be gone after -r --force")
	}
}

func TestCmdDeleteRecursiveForceJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	env.Store.Create("Child", issue.CreateOpts{Parent: root.ID})
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	if _, err := cmdDelete(env.Store, []string{root.ID, "-r", "--force", "--json"}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdDelete -r --force --json: %v", err)
	}
	var got issue.SubtreeDeleteResult
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(got.Deleted) != 2 {
		t.Errorf("deleted = %d, want 2", len(got.Deleted))
	}
	if got.Unblocked == nil {
		t.Error("unblocked should be [] not null")
	}
}
