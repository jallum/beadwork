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

func TestCmdPrimeTemplateProcessing(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Without config: no conditional section
	var buf bytes.Buffer
	err := cmdPrime(env.Store, nil, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	if strings.Contains(buf.String(), "Working in Parallel") {
		t.Error("should not contain 'Working in Parallel' without config")
	}

	// With config: conditional section appears
	env.Repo.SetConfig("workflow.agents", "multi")
	env.Repo.Commit("set config")

	buf.Reset()
	err = cmdPrime(env.Store, nil, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	if !strings.Contains(buf.String(), "Working in Parallel") {
		t.Errorf("output missing 'Working in Parallel': %q", buf.String())
	}
}
