package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdShowBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Show me", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdShow(env.Repo, env.Store, []string{iss.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	if !strings.Contains(buf.String(), "Show me") {
		t.Errorf("output missing title: %q", buf.String())
	}
}

func TestCmdShowJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("JSON show", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdShow(env.Repo, env.Store, []string{iss.ID, "--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}

	// JSON output should be an array
	var got []issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(got))
	}
	if got[0].Title != "JSON show" {
		t.Errorf("title = %q", got[0].Title)
	}
}

func TestCmdShowMultiID(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("First", issue.CreateOpts{})
	b, _ := env.Store.Create("Second", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	err := cmdShow(env.Repo, env.Store, []string{a.ID, b.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow multi: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "First") {
		t.Errorf("missing First: %q", out)
	}
	if !strings.Contains(out, "Second") {
		t.Errorf("missing Second: %q", out)
	}
}

func TestCmdShowMultiIDJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Alpha", issue.CreateOpts{})
	b, _ := env.Store.Create("Beta", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	err := cmdShow(env.Repo, env.Store, []string{a.ID, b.ID, "--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow multi --json: %v", err)
	}

	var got []issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(got))
	}
}

func TestCmdShowShort(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Short show", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdShow(env.Repo, env.Store, []string{iss.ID, "--short"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow --short: %v", err)
	}
	out := buf.String()
	// Should be a compact one-liner, not the full multi-line display
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line for --short, got %d: %q", len(lines), out)
	}
	if !strings.Contains(out, iss.ID) {
		t.Errorf("missing ID: %q", out)
	}
	if !strings.Contains(out, "Short show") {
		t.Errorf("missing title: %q", out)
	}
}

func TestCmdShowRichDeps(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker issue", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked issue", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("create and link")

	// Show b — should display rich dep info for its blocker
	var buf bytes.Buffer
	err := cmdShow(env.Repo, env.Store, []string{b.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	out := buf.String()
	// Should show the blocker's title, not just its ID
	if !strings.Contains(out, "Blocker issue") {
		t.Errorf("should show blocker title: %q", out)
	}
	if !strings.Contains(out, "DEPENDS ON") {
		t.Errorf("should show DEPENDS ON section: %q", out)
	}
}

func TestCmdShowNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdShow(env.Repo, env.Store, []string{"nonexistent"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}
