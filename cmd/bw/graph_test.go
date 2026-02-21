package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdGraphAll(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Graph A", issue.CreateOpts{})
	b, _ := env.Store.Create("Graph B", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("create and link")

	var buf bytes.Buffer
	err := cmdGraph(env.Repo, env.Store, []string{"--all"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdGraph: %v", err)
	}
	if !strings.Contains(buf.String(), "Graph A") {
		t.Errorf("output missing Graph A: %q", buf.String())
	}
}

func TestCmdGraphJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Graph JSON", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	err := cmdGraph(env.Repo, env.Store, []string{"--all", "--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdGraph: %v", err)
	}

	var nodes []issue.GraphNode
	if err := json.Unmarshal(buf.Bytes(), &nodes); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
}

func TestCmdGraphRooted(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Root", issue.CreateOpts{})
	b, _ := env.Store.Create("Child", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("create and link")

	var buf bytes.Buffer
	err := cmdGraph(env.Repo, env.Store, []string{a.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdGraph: %v", err)
	}
	if !strings.Contains(buf.String(), "Root") {
		t.Errorf("output missing Root: %q", buf.String())
	}
}

func TestCmdGraphEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdGraph(env.Repo, env.Store, []string{"--all"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdGraph --all: %v", err)
	}
	if !strings.Contains(buf.String(), "no issues") {
		t.Errorf("output = %q, want 'no issues'", buf.String())
	}
}

func TestCmdGraphMultipleRoots(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create a graph with two independent trees
	a, _ := env.Store.Create("Root A", issue.CreateOpts{})
	b, _ := env.Store.Create("Child B", issue.CreateOpts{})
	c, _ := env.Store.Create("Root C", issue.CreateOpts{})
	d, _ := env.Store.Create("Child D", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Link(c.ID, d.ID)
	env.Repo.Commit("setup graph")

	var buf bytes.Buffer
	err := cmdGraph(env.Repo, env.Store, []string{"--all"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdGraph: %v", err)
	}
	// Should show all 4 nodes
	out := buf.String()
	if !strings.Contains(out, "Root A") || !strings.Contains(out, "Child B") {
		t.Errorf("missing tree A in output: %q", out)
	}
	if !strings.Contains(out, "Root C") || !strings.Contains(out, "Child D") {
		t.Errorf("missing tree C in output: %q", out)
	}
}

func TestCmdGraphAllShowsClosedIssues(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Closed A", issue.CreateOpts{})
	b, _ := env.Store.Create("Closed B", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(a.ID, "")
	env.Store.Close(b.ID, "")
	env.Repo.Commit("create, link, and close")

	var buf bytes.Buffer
	err := cmdGraph(env.Repo, env.Store, []string{"--all"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdGraph --all: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "no issues") {
		t.Error("--all should show closed issues, got 'no issues'")
	}
	if !strings.Contains(out, "Closed A") || !strings.Contains(out, "Closed B") {
		t.Errorf("output missing closed issues: %q", out)
	}
}

func TestCmdGraphNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdGraph(env.Repo, env.Store, []string{"nonexistent"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for nonexistent root ID")
	}
}
