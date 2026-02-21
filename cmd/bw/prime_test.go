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

func TestCmdPrimeInProgressWithComment(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Commented task", issue.CreateOpts{})
	statusIP := "in_progress"
	env.Store.Update(iss.ID, issue.UpdateOpts{Status: &statusIP})
	env.Store.Comment(iss.ID, "Left off here", "")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	err := cmdPrime(PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "└") {
		t.Errorf("output missing comment indicator: %q", out)
	}
	if !strings.Contains(out, "Left off here") {
		t.Errorf("output missing comment text: %q", out)
	}
}

func TestCmdPrimeCommentTruncation(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Long comment task", issue.CreateOpts{})
	statusIP := "in_progress"
	env.Store.Update(iss.ID, issue.UpdateOpts{Status: &statusIP})
	longText := "This is a very long comment that exceeds sixty characters and should be truncated"
	env.Store.Comment(iss.ID, longText, "")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	err := cmdPrime(PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "...") {
		t.Errorf("output missing truncation: %q", out)
	}
}

func TestCmdPrimeTemplateProcessing(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Without config: no conditional section
	var buf bytes.Buffer
	err := cmdPrime(PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	if strings.Contains(buf.String(), "Parallel Agents") {
		t.Error("should not contain 'Parallel Agents' without config")
	}

	// With config: conditional section appears
	env.Repo.SetConfig("workflow.agents", "multi")
	env.Repo.Commit("set config")

	buf.Reset()
	err = cmdPrime(PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	if !strings.Contains(buf.String(), "Parallel Agents") {
		t.Errorf("output missing 'Parallel Agents': %q", buf.String())
	}
}
