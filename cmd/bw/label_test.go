package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdLabelAdd(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Label me", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdLabel(env.Repo, env.Store, []string{iss.ID, "+bug", "+urgent"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdLabel: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if len(got.Labels) != 2 {
		t.Errorf("labels = %v, want 2", got.Labels)
	}
}

func TestCmdLabelRemove(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Unlabel me", issue.CreateOpts{})
	env.Store.Label(iss.ID, []string{"bug", "urgent"}, nil)
	env.Repo.Commit("create and label " + iss.ID)

	var buf bytes.Buffer
	err := cmdLabel(env.Repo, env.Store, []string{iss.ID, "-bug"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdLabel: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	for _, l := range got.Labels {
		if l == "bug" {
			t.Error("label 'bug' should have been removed")
		}
	}
}

func TestCmdLabelJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Label JSON", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdLabel(env.Repo, env.Store, []string{iss.ID, "+bug", "--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdLabel --json: %v", err)
	}

	var got issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(got.Labels) == 0 || got.Labels[0] != "bug" {
		t.Errorf("labels = %v, want [bug]", got.Labels)
	}
}

func TestCmdLabelBareAdd(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Bare label", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdLabel(env.Repo, env.Store, []string{iss.ID, "feature"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdLabel bare: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if len(got.Labels) != 1 || got.Labels[0] != "feature" {
		t.Errorf("labels = %v, want [feature]", got.Labels)
	}
}

func TestCmdLabelNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdLabel(env.Repo, env.Store, []string{"nonexistent", "+bug"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}
