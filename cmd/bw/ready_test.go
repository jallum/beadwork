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
