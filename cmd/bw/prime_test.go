package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdPrimeBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Prime issue", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	_, err := cmdPrime(env.Store, nil, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Prime issue") {
		t.Errorf("output missing ready issue: %q", out)
	}
	if !strings.Contains(out, "Work In Progress") {
		t.Errorf("output missing 'Work In Progress' section header: %q", out)
	}
}

func TestCmdPrimeEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdPrime(env.Store, nil, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "no ready issues") {
		t.Errorf("output missing 'no ready issues': %q", out)
	}
}

func TestCmdPrimeInProgress(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("WIP task", issue.CreateOpts{})
	env.Repo.Commit("create issue")
	status := "in_progress"
	env.Store.Update(iss.ID, issue.UpdateOpts{Status: &status})
	env.Repo.Commit("start issue")

	var buf bytes.Buffer
	_, err := cmdPrime(env.Store, nil, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "WIP task") {
		t.Errorf("output missing in-progress issue: %q", out)
	}
}

// The "Currently available work" section groups ready children under their
// in_progress epic header — the legend tells readers not to try to start the
// ◐ entries.
func TestCmdPrimeIncludesIconLegend(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	if _, err := cmdPrime(env.Store, nil, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "○") || !strings.Contains(out, "◐") {
		t.Errorf("output missing icon legend glyphs:\n%s", out)
	}
	if !strings.Contains(out, "group header") {
		t.Errorf("output missing group-header explanation:\n%s", out)
	}
}

// Regression: when every top-level epic is in_progress, "Currently available
// work" went silent. Ready() should drill past claimed ancestors and surface
// the ready children.
func TestCmdPrimeSurfacesChildrenOfInProgressEpic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{Type: "epic"})
	child, _ := env.Store.Create("Next step", issue.CreateOpts{Parent: epic.ID})
	env.Repo.Commit("setup")
	if _, err := env.Store.Start(epic.ID, ""); err != nil {
		t.Fatalf("Start: %v", err)
	}
	env.Repo.Commit("start epic")

	var buf bytes.Buffer
	if _, err := cmdPrime(env.Store, nil, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, child.ID) {
		t.Errorf("output missing child %s under in_progress epic:\n%s", child.ID, out)
	}
	if strings.Contains(out, "no ready issues") {
		t.Errorf("output claims no ready issues despite ready child:\n%s", out)
	}
}

func TestCmdPrimeTemplateProcessing(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Prime no longer has conditional sections (moved to start.md).
	// Verify it renders core sections cleanly.
	var buf bytes.Buffer
	_, err := cmdPrime(env.Store, nil, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Beadwork") {
		t.Errorf("output missing 'Beadwork' header: %q", out)
	}
	if !strings.Contains(out, "Workflow") {
		t.Errorf("output missing 'Workflow' section: %q", out)
	}
	// No IF/END template artifacts should leak through
	if strings.Contains(out, "<!-- IF") || strings.Contains(out, "<!-- END") {
		t.Errorf("output contains unprocessed template directives: %q", out)
	}
}
