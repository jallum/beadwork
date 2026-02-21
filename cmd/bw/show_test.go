package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdShowBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Show me", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{iss.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	if !strings.Contains(buf.String(), "Show me") {
		t.Errorf("output missing title: %q", buf.String())
	}
}

func TestCmdShowJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("JSON show", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{iss.ID, "--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}

	// JSON output should be an array
	var got []issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(got))
	}
	if got[0].Title != "JSON show" {
		t.Errorf("title = %q", got[0].Title)
	}
}

func TestCmdShowMultiID(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("First", issue.CreateOpts{})
	b, _ := env.Store.Create("Second", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{a.ID, b.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow multi: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "First") {
		t.Errorf("missing First: %q", out)
	}
	if !strings.Contains(out, "Second") {
		t.Errorf("missing Second: %q", out)
	}
}

func TestCmdShowMultiIDJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Alpha", issue.CreateOpts{})
	b, _ := env.Store.Create("Beta", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{a.ID, b.ID, "--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow multi --json: %v", err)
	}

	var got []issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(got))
	}
}

func TestCmdShowOnlySummary(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Summary only", issue.CreateOpts{
		Description: "Should not appear",
	})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{iss.ID, "--only", "summary"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow --only summary: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, iss.ID) {
		t.Errorf("missing ID: %q", out)
	}
	if !strings.Contains(out, "Summary only") {
		t.Errorf("missing title: %q", out)
	}
	if strings.Contains(out, "DESCRIPTION") {
		t.Errorf("should NOT show description section: %q", out)
	}
	if strings.Contains(out, "Should not appear") {
		t.Errorf("should NOT show description text: %q", out)
	}
}

func TestCmdShowOnlyDescription(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Desc test", issue.CreateOpts{
		Description: "The description text",
	})
	env.Store.Comment(iss.ID, "A comment", "")
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{iss.ID, "--only", "description"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow --only description: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "The description text") {
		t.Errorf("missing description: %q", out)
	}
	// Should NOT show header, comments, etc.
	if strings.Contains(out, "Assignee:") {
		t.Errorf("should NOT show metadata: %q", out)
	}
	if strings.Contains(out, "COMMENTS") {
		t.Errorf("should NOT show comments: %q", out)
	}
}

func TestCmdShowOnlyMultiple(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{
		Description: "Some desc",
	})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{b.ID, "--only", "summary,blockedby"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow --only: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Blocked") {
		t.Errorf("missing title: %q", out)
	}
	if !strings.Contains(out, "BLOCKED BY") {
		t.Errorf("missing BLOCKED BY: %q", out)
	}
	if strings.Contains(out, "DESCRIPTION") {
		t.Errorf("should NOT show description: %q", out)
	}
}

func TestCmdShowOnlyInvalid(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test", issue.CreateOpts{})
	env.Repo.Commit("create")

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{iss.ID, "--only", "nonsense"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for invalid section name")
	}
}

func TestCmdShowShortRemoved(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test", issue.CreateOpts{})
	env.Repo.Commit("create")

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{iss.ID, "--short"}, PlainWriter(&buf))
	if err == nil {
		t.Error("--short should be removed, expected error")
	}
}

func TestCmdShowRichDeps(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker issue", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked issue", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("create and link")

	// Show b — should display rich dep info for its blocker
	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{b.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	out := buf.String()
	// Should show the blocker's title in BLOCKED BY section
	if !strings.Contains(out, "Blocker issue") {
		t.Errorf("should show blocker title: %q", out)
	}
	if !strings.Contains(out, "BLOCKED BY") {
		t.Errorf("should show BLOCKED BY section: %q", out)
	}

	// Show a — should display BLOCKS section
	buf.Reset()
	err = cmdShow(env.Store, []string{a.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	out = buf.String()
	if !strings.Contains(out, "UNBLOCKS") {
		t.Errorf("should show UNBLOCKS section: %q", out)
	}
	if !strings.Contains(out, "Blocked issue") {
		t.Errorf("should show blocked issue title: %q", out)
	}
}

func TestCmdShowTipsDeepChain(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// C blocks B blocks A — showing A should display C (the tip), not B
	a, _ := env.Store.Create("Target", issue.CreateOpts{})
	b, _ := env.Store.Create("Middle", issue.CreateOpts{})
	c, _ := env.Store.Create("Leaf tip", issue.CreateOpts{})
	env.Store.Link(c.ID, b.ID)
	env.Store.Link(b.ID, a.ID)
	env.Repo.Commit("setup chain")

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{a.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "BLOCKED BY") {
		t.Errorf("should show BLOCKED BY section: %q", out)
	}
	if !strings.Contains(out, "Leaf tip") {
		t.Errorf("should show leaf tip title: %q", out)
	}
	if !strings.Contains(out, c.ID) {
		t.Errorf("should show leaf tip ID %s: %q", c.ID, out)
	}
	// Middle node should NOT appear in BLOCKED BY
	if strings.Contains(out, "Middle") {
		t.Errorf("should NOT show middle node: %q", out)
	}
}

func TestCmdShowBlockedByClosedTipWalksBack(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// D blocks C blocks B blocks A — D is closed.
	// Showing A should display C (nearest open ancestor of the closed tip).
	a, _ := env.Store.Create("Target", issue.CreateOpts{})
	b, _ := env.Store.Create("Middle", issue.CreateOpts{})
	c, _ := env.Store.Create("Workable", issue.CreateOpts{})
	d, _ := env.Store.Create("Done leaf", issue.CreateOpts{})
	env.Store.Link(d.ID, c.ID)
	env.Store.Link(c.ID, b.ID)
	env.Store.Link(b.ID, a.ID)
	env.Store.Close(d.ID, "")
	env.Repo.Commit("setup chain")

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{a.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Workable") {
		t.Errorf("should show nearest open blocker C: %q", out)
	}
	if !strings.Contains(out, c.ID) {
		t.Errorf("should show C's ID %s: %q", c.ID, out)
	}
	// Closed leaf D should NOT appear
	if strings.Contains(out, "Done leaf") {
		t.Errorf("should NOT show closed leaf D: %q", out)
	}
}

func TestCmdShowBlockedByClosedTipsDedup(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// D blocks C, E blocks C, C blocks A — D and E both closed.
	// Showing A should display C once (deduped).
	a, _ := env.Store.Create("Target", issue.CreateOpts{})
	c, _ := env.Store.Create("Shared blocker", issue.CreateOpts{})
	d, _ := env.Store.Create("Done D", issue.CreateOpts{})
	e, _ := env.Store.Create("Done E", issue.CreateOpts{})
	env.Store.Link(d.ID, c.ID)
	env.Store.Link(e.ID, c.ID)
	env.Store.Link(c.ID, a.ID)
	env.Store.Close(d.ID, "")
	env.Store.Close(e.ID, "")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{a.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "Shared blocker") {
		t.Errorf("should show C: %q", out)
	}
	// Should only appear once in BLOCKED BY
	idx1 := strings.Index(out, c.ID)
	if idx1 < 0 {
		t.Fatalf("C's ID not found: %q", out)
	}
	// Check no second occurrence after BLOCKED BY header
	blockedByIdx := strings.Index(out, "BLOCKED BY")
	afterHeader := out[blockedByIdx:]
	count := strings.Count(afterHeader, c.ID)
	if count != 1 {
		t.Errorf("C's ID should appear once in BLOCKED BY, got %d: %q", count, afterHeader)
	}
}

func TestCmdShowBlockedByAllResolved(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// B blocks A, B is closed — no BLOCKED BY section (all resolved).
	a, _ := env.Store.Create("Target", issue.CreateOpts{})
	b, _ := env.Store.Create("Resolved", issue.CreateOpts{})
	env.Store.Link(b.ID, a.ID)
	env.Store.Close(b.ID, "")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{a.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	out := buf.String()

	if strings.Contains(out, "BLOCKED BY") {
		t.Errorf("should NOT show BLOCKED BY when all blockers resolved: %q", out)
	}
}

func TestCmdShowUnblocksImmediate(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// A blocks B blocks C — showing A should display B (immediate), not C
	a, _ := env.Store.Create("Root blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Middle", issue.CreateOpts{})
	c, _ := env.Store.Create("Downstream leaf", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Link(b.ID, c.ID)
	env.Repo.Commit("setup chain")

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{a.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "UNBLOCKS") {
		t.Errorf("should show UNBLOCKS section: %q", out)
	}
	if !strings.Contains(out, "Middle") {
		t.Errorf("should show immediate dep Middle: %q", out)
	}
	// Downstream leaf should NOT appear (not an immediate dep)
	if strings.Contains(out, "Downstream leaf") {
		t.Errorf("should NOT show downstream leaf: %q", out)
	}
}

func TestCmdShowNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdShow(env.Store, []string{"nonexistent"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}
