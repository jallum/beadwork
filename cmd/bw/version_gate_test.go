package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/testutil"
)

func TestVersionGateBlocksOlderRepo(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Downgrade to v0 by removing version key
	env.Repo.SetConfig("version", "0")
	env.Repo.Commit("downgrade to v0")

	_, _, err := getInitializedRepo()
	if err == nil {
		t.Fatal("expected error for v0 repo")
	}
	if !strings.Contains(err.Error(), "upgrade repo") {
		t.Errorf("error should mention 'upgrade repo': %v", err)
	}
}

func TestVersionGateBlocksNewerRepo(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Set to a future version
	env.Repo.SetConfig("version", "99")
	env.Repo.Commit("future version")

	_, _, err := getInitializedRepo()
	if err == nil {
		t.Fatal("expected error for v99 repo")
	}
	if !strings.Contains(err.Error(), "bw upgrade") {
		t.Errorf("error should mention 'bw upgrade': %v", err)
	}
	if strings.Contains(err.Error(), "upgrade repo") {
		t.Errorf("error for newer repo should NOT suggest 'upgrade repo': %v", err)
	}
}

func TestCmdUpgradeRepoFromV0(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Downgrade to v0
	env.Repo.SetConfig("version", "0")
	env.Repo.Commit("downgrade to v0")

	var buf bytes.Buffer
	err := cmdUpgradeRepo([]string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpgradeRepo: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "upgrading") {
		t.Errorf("output should contain 'upgrading': %q", out)
	}
	if !strings.Contains(out, "upgraded to v2") {
		t.Errorf("output should contain 'upgraded to v2': %q", out)
	}

	// Commands should work now
	buf.Reset()
	err = cmdList(env.Repo, env.Store, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdList after upgrade: %v", err)
	}
}

func TestCmdUpgradeRepoAlreadyCurrent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdUpgradeRepo([]string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpgradeRepo: %v", err)
	}
	if !strings.Contains(buf.String(), "up to date") {
		t.Errorf("output should contain 'up to date': %q", buf.String())
	}
}
