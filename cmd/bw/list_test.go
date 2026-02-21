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
	err := cmdList([]string{}, PlainWriter(&buf))
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
	err := cmdList([]string{"--json"}, PlainWriter(&buf))
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
	err := cmdList([]string{}, PlainWriter(&buf))
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
	err := cmdList([]string{"--status", "closed"}, PlainWriter(&buf))
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
	err := cmdList([]string{}, PlainWriter(&buf))
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
	err := cmdList([]string{"--grep", "login"}, PlainWriter(&buf))
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
	err := cmdList([]string{"-g", "login"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdList -g: %v", err)
	}
	if !strings.Contains(buf.String(), "Login page broken") {
		t.Errorf("-g login should match: %q", buf.String())
	}
}
