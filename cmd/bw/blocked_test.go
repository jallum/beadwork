package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdBlockedBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{Priority: intPtr(1)})
	b, _ := env.Store.Create("Blocked task", issue.CreateOpts{Priority: intPtr(2)})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("link")

	var buf bytes.Buffer
	err := cmdBlocked([]string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdBlocked: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, b.ID) {
		t.Errorf("output should contain blocked issue %s: %q", b.ID, out)
	}
	if !strings.Contains(out, "Blocked task") {
		t.Errorf("output should contain title: %q", out)
	}
	if !strings.Contains(out, a.ID) {
		t.Errorf("output should list blocker %s: %q", a.ID, out)
	}
}

func TestCmdBlockedResolves(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(a.ID, "")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	err := cmdBlocked([]string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdBlocked: %v", err)
	}
	if strings.Contains(buf.String(), b.ID) {
		t.Error("resolved issue should not appear in blocked output")
	}
}

func TestCmdBlockedJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("link")

	var buf bytes.Buffer
	err := cmdBlocked([]string{"--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdBlocked --json: %v", err)
	}

	var result []struct {
		ID           string   `json:"id"`
		OpenBlockers []string `json:"open_blockers"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("got %d results, want 1", len(result))
	}
	if result[0].ID != b.ID {
		t.Errorf("id = %q, want %q", result[0].ID, b.ID)
	}
	if len(result[0].OpenBlockers) != 1 || result[0].OpenBlockers[0] != a.ID {
		t.Errorf("open_blockers = %v, want [%s]", result[0].OpenBlockers, a.ID)
	}
}

func TestCmdBlockedEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("No deps", issue.CreateOpts{})
	env.Repo.Commit("create")

	var buf bytes.Buffer
	err := cmdBlocked([]string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdBlocked: %v", err)
	}
	if !strings.Contains(buf.String(), "no blocked issues") {
		t.Errorf("expected 'no blocked issues', got: %q", buf.String())
	}
}

func TestCmdBlockedMultipleBlockers(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker A", issue.CreateOpts{Priority: intPtr(1)})
	b, _ := env.Store.Create("Blocker B", issue.CreateOpts{Priority: intPtr(1)})
	c, _ := env.Store.Create("Blocked by two", issue.CreateOpts{Priority: intPtr(2)})
	env.Store.Link(a.ID, c.ID)
	env.Store.Link(b.ID, c.ID)
	env.Repo.Commit("link")

	var buf bytes.Buffer
	err := cmdBlocked([]string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdBlocked: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, c.ID) {
		t.Errorf("output should contain blocked issue %s: %q", c.ID, out)
	}
	if !strings.Contains(out, a.ID) {
		t.Errorf("output should list blocker %s: %q", a.ID, out)
	}
	if !strings.Contains(out, b.ID) {
		t.Errorf("output should list blocker %s: %q", b.ID, out)
	}
}

func TestCmdBlockedUnknownFlag(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdBlocked([]string{"--verbose"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestCmdBlockedJSONEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("No deps", issue.CreateOpts{})
	env.Repo.Commit("create")

	var buf bytes.Buffer
	err := cmdBlocked([]string{"--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdBlocked --json: %v", err)
	}

	var result []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestCmdBlockedPartiallyResolved(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker A", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocker B", issue.CreateOpts{})
	c, _ := env.Store.Create("Blocked by two", issue.CreateOpts{})
	env.Store.Link(a.ID, c.ID)
	env.Store.Link(b.ID, c.ID)
	// Close one blocker — issue should still be blocked by the other
	env.Store.Close(a.ID, "done")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	err := cmdBlocked([]string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdBlocked: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, c.ID) {
		t.Errorf("issue %s should still be blocked: %q", c.ID, out)
	}
	if !strings.Contains(out, b.ID) {
		t.Errorf("should list remaining blocker %s: %q", b.ID, out)
	}
}

func TestParseBlockedArgs(t *testing.T) {
	ba, err := parseBlockedArgs([]string{})
	if err != nil {
		t.Fatalf("parseBlockedArgs: %v", err)
	}
	if ba.JSON {
		t.Error("expected JSON=false")
	}
}

func TestParseBlockedArgsJSON(t *testing.T) {
	ba, err := parseBlockedArgs([]string{"--json"})
	if err != nil {
		t.Fatalf("parseBlockedArgs: %v", err)
	}
	if !ba.JSON {
		t.Error("expected JSON=true")
	}
}
