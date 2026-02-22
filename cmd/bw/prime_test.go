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
	err := cmdPrime(env.Store, nil, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Prime issue") {
		t.Errorf("output missing ready issue: %q", out)
	}
	if !strings.Contains(out, "Ready: 1") {
		t.Errorf("output missing ready count: %q", out)
	}
	if !strings.Contains(out, "Work In Progress") {
		t.Errorf("output missing 'Work In Progress' section header: %q", out)
	}
}

func TestCmdPrimeEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdPrime(env.Store, nil, PlainWriter(&buf))
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
	err := cmdPrime(env.Store, nil, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "WIP task") {
		t.Errorf("output missing in-progress issue: %q", out)
	}
}

func TestCmdPrimeTemplateProcessing(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Prime no longer has conditional sections (moved to land_the_work.md).
	// Verify it renders core sections cleanly.
	var buf bytes.Buffer
	err := cmdPrime(env.Store, nil, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Beadwork") {
		t.Errorf("output missing 'Beadwork' header: %q", out)
	}
	if !strings.Contains(out, "Starting the Work") {
		t.Errorf("output missing 'Starting the Work' section: %q", out)
	}
	// No IF/END template artifacts should leak through
	if strings.Contains(out, "<!-- IF") || strings.Contains(out, "<!-- END") {
		t.Errorf("output contains unprocessed template directives: %q", out)
	}
}
