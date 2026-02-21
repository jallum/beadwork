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
	err := cmdPrime(PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	if !strings.Contains(buf.String(), "Current State") {
		t.Errorf("output missing 'Current State': %q", buf.String())
	}
	if !strings.Contains(buf.String(), "1 open") {
		t.Errorf("output missing '1 open': %q", buf.String())
	}
}

func TestCmdPrimeInProgress(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("In progress task", issue.CreateOpts{})
	statusIP := "in_progress"
	env.Store.Update(iss.ID, issue.UpdateOpts{Status: &statusIP})
	env.Repo.Commit("create and update " + iss.ID)

	var buf bytes.Buffer
	err := cmdPrime(PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "In progress:") {
		t.Errorf("output missing 'In progress:': %q", out)
	}
	if !strings.Contains(out, "In progress task") {
		t.Errorf("output missing in_progress task: %q", out)
	}
}

func TestCmdPrimeEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdPrime(PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "0 open") {
		t.Errorf("output missing '0 open': %q", out)
	}
}
