package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdCloseBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Close me", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdClose(env.Store, []string{iss.ID}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdClose: %v", err)
	}
	if !strings.Contains(buf.String(), "closed") {
		t.Errorf("output = %q", buf.String())
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "closed" {
		t.Errorf("status = %q, want closed", got.Status)
	}
}

func TestCmdCloseWithReason(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Close with reason", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdClose(env.Store, []string{iss.ID, "--reason", "duplicate"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdClose: %v", err)
	}
}

func TestCmdCloseJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Close JSON", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdClose(env.Store, []string{iss.ID, "--json"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdClose --json: %v", err)
	}

	var got issue.CloseResult
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got.Issue.Status != "closed" {
		t.Errorf("status = %q, want closed", got.Issue.Status)
	}
	if got.Unblocked == nil {
		t.Error("unblocked should be [] not null")
	}
}

func TestCmdCloseNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdClose(env.Store, []string{"nonexistent"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

func TestCmdCloseRecursive(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	c, _ := env.Store.Create("Child", issue.CreateOpts{Parent: root.ID})
	gc, _ := env.Store.Create("Grandchild", issue.CreateOpts{Parent: c.ID})
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdClose(env.Store, []string{root.ID, "--recursive"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdClose --recursive: %v", err)
	}
	if !strings.Contains(buf.String(), "closed 3 issue") {
		t.Errorf("output = %q", buf.String())
	}

	for _, id := range []string{root.ID, c.ID, gc.ID} {
		got, _ := env.Store.Get(id)
		if got.Status != "closed" {
			t.Errorf("%s status = %q, want closed", id, got.Status)
		}
	}
}

func TestCmdCloseRecursiveAlias(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	c, _ := env.Store.Create("Child", issue.CreateOpts{Parent: root.ID})
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	if _, err := cmdClose(env.Store, []string{root.ID, "-r"}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdClose -r: %v", err)
	}
	got, _ := env.Store.Get(c.ID)
	if got.Status != "closed" {
		t.Errorf("child status = %q, want closed", got.Status)
	}
}

func TestCmdCloseRecursiveSkipsAlreadyClosed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	c, _ := env.Store.Create("Child", issue.CreateOpts{Parent: root.ID})
	env.Store.Close(c.ID, "")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	if _, err := cmdClose(env.Store, []string{root.ID, "-r"}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdClose -r: %v", err)
	}
	if !strings.Contains(buf.String(), "already closed, skipped") {
		t.Errorf("output = %q", buf.String())
	}
	rootGot, _ := env.Store.Get(root.ID)
	if rootGot.Status != "closed" {
		t.Errorf("root status = %q, want closed", rootGot.Status)
	}
}

func TestCmdCloseRecursiveJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	root, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	env.Store.Create("Child", issue.CreateOpts{Parent: root.ID})
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdClose(env.Store, []string{root.ID, "-r", "--json"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdClose -r --json: %v", err)
	}

	var got issue.SubtreeCloseResult
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(got.Closed) != 2 {
		t.Errorf("closed = %d, want 2", len(got.Closed))
	}
	if got.Skipped == nil || got.Unblocked == nil {
		t.Error("skipped/unblocked should be [] not null")
	}
}

func TestCmdReopenBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Reopen me", issue.CreateOpts{})
	env.Store.Close(iss.ID, "")
	env.Repo.Commit("create and close " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdReopen(env.Store, []string{iss.ID}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdReopen: %v", err)
	}
	if !strings.Contains(buf.String(), "reopened") {
		t.Errorf("output = %q", buf.String())
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
}

func TestCmdReopenJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Reopen JSON", issue.CreateOpts{})
	env.Store.Close(iss.ID, "")
	env.Repo.Commit("create and close " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdReopen(env.Store, []string{iss.ID, "--json"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdReopen --json: %v", err)
	}

	var got issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
}

func TestCmdReopenInProgress(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Unclaim me", issue.CreateOpts{})
	env.Store.Start(iss.ID, "alice")
	env.Repo.Commit("create and start " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdReopen(env.Store, []string{iss.ID}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdReopen in_progress: %v", err)
	}
	if !strings.Contains(buf.String(), "reopened") {
		t.Errorf("output = %q", buf.String())
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
	if got.Assignee != "" {
		t.Errorf("assignee = %q, want empty", got.Assignee)
	}
}

func TestCmdReopenNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdReopen(env.Store, []string{"nonexistent"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

func TestCmdCloseUnblocked(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Dependent", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdClose(env.Store, []string{a.ID}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdClose: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "closed") {
		t.Errorf("missing 'closed' in output: %q", out)
	}
	if !strings.Contains(out, "unblocked") {
		t.Errorf("missing 'unblocked' in output: %q", out)
	}
	if !strings.Contains(out, b.ID) {
		t.Errorf("missing dependent ID %s in output: %q", b.ID, out)
	}
}

func TestCmdCloseUnblockedJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Dependent", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdClose(env.Store, []string{a.ID, "--json"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdClose --json: %v", err)
	}

	var got issue.CloseResult
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(got.Unblocked) != 1 {
		t.Fatalf("unblocked count = %d, want 1", len(got.Unblocked))
	}
	if got.Unblocked[0].ID != b.ID {
		t.Errorf("unblocked[0].ID = %q, want %q", got.Unblocked[0].ID, b.ID)
	}
}
