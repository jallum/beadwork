package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdDepAddBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	err := cmdDepAdd([]string{a.ID, "blocks", b.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdDepAdd: %v", err)
	}
	if !strings.Contains(buf.String(), "added dep") {
		t.Errorf("output = %q", buf.String())
	}

	got, _ := env.Store.Get(b.ID)
	if len(got.BlockedBy) == 0 {
		t.Error("expected BlockedBy to be set")
	}
}

func TestCmdDepAddNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdDepAdd([]string{"nonexistent", "blocks", "also-missing"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for nonexistent issues")
	}
}

func TestCmdDepRemoveBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("create and link")

	var buf bytes.Buffer
	err := cmdDepRemove([]string{a.ID, "blocks", b.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdDepRemove: %v", err)
	}
	if !strings.Contains(buf.String(), "removed dep") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestCmdDepRemoveNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdDepRemove([]string{"nonexistent", "blocks", "also-missing"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for nonexistent issues")
	}
}
