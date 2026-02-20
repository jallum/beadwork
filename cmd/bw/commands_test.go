package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func intPtr(n int) *int { return &n }

// --- Create ---

func TestCmdCreateBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate([]string{"Test issue"}, &buf)
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}
	if !strings.Contains(buf.String(), "created") {
		t.Errorf("output = %q, want 'created'", buf.String())
	}
}

func TestCmdCreateWithFlags(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate([]string{"Bug report", "--priority", "1", "--type", "bug", "--assignee", "alice"}, &buf)
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}

	// Verify the issue was created with correct attributes
	issues, _ := env.Store.List(issue.Filter{})
	found := false
	for _, iss := range issues {
		if iss.Title == "Bug report" {
			found = true
			if iss.Priority != 1 {
				t.Errorf("priority = %d, want 1", iss.Priority)
			}
			if iss.Type != "bug" {
				t.Errorf("type = %q, want bug", iss.Type)
			}
			if iss.Assignee != "alice" {
				t.Errorf("assignee = %q, want alice", iss.Assignee)
			}
		}
	}
	if !found {
		t.Error("issue 'Bug report' not found")
	}
}

func TestCmdCreateJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate([]string{"JSON test", "--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}

	var iss issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &iss); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if iss.Title != "JSON test" {
		t.Errorf("title = %q, want 'JSON test'", iss.Title)
	}
}

func TestCmdCreateWithLabels(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate([]string{"Labeled task", "--labels", "frontend,urgent"}, &buf)
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Labeled task") {
		t.Errorf("output missing title: %q", out)
	}

	// Verify labels were applied by re-reading via store
	issues, _ := env.Store.List(issue.Filter{})
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	iss := issues[0]
	if len(iss.Labels) != 2 {
		t.Fatalf("Labels = %v, want 2 labels", iss.Labels)
	}
	if iss.Labels[0] != "frontend" || iss.Labels[1] != "urgent" {
		t.Errorf("Labels = %v, want [frontend urgent]", iss.Labels)
	}
}

func TestCmdCreateWithLabelsJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate([]string{"Labeled JSON", "--labels", "backend", "--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}

	var iss issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &iss); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(iss.Labels) != 1 || iss.Labels[0] != "backend" {
		t.Errorf("Labels = %v, want [backend]", iss.Labels)
	}
}

// --- Show ---

func TestCmdShowBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Show me", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdShow([]string{iss.ID}, &buf)
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
	err := cmdShow([]string{iss.ID, "--json"}, &buf)
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
	err := cmdShow([]string{a.ID, b.ID}, &buf)
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
	err := cmdShow([]string{a.ID, b.ID, "--json"}, &buf)
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

func TestCmdShowShort(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Short show", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdShow([]string{iss.ID, "--short"}, &buf)
	if err != nil {
		t.Fatalf("cmdShow --short: %v", err)
	}
	out := buf.String()
	// Should be a compact one-liner, not the full multi-line display
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line for --short, got %d: %q", len(lines), out)
	}
	if !strings.Contains(out, iss.ID) {
		t.Errorf("missing ID: %q", out)
	}
	if !strings.Contains(out, "Short show") {
		t.Errorf("missing title: %q", out)
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
	err := cmdShow([]string{b.ID}, &buf)
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	out := buf.String()
	// Should show the blocker's title, not just its ID
	if !strings.Contains(out, "Blocker issue") {
		t.Errorf("should show blocker title: %q", out)
	}
	if !strings.Contains(out, "DEPENDS ON") {
		t.Errorf("should show DEPENDS ON section: %q", out)
	}
}

func TestCmdShowNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdShow([]string{"nonexistent"}, &buf)
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

// --- List ---

func TestCmdListBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Issue A", issue.CreateOpts{})
	env.Store.Create("Issue B", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	err := cmdList([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	if !strings.Contains(buf.String(), "Issue A") || !strings.Contains(buf.String(), "Issue B") {
		t.Errorf("output missing issues: %q", buf.String())
	}
}

func TestCmdListJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("JSON list", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	err := cmdList([]string{"--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}

	var issues []issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &issues); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(issues) == 0 {
		t.Error("expected issues in JSON output")
	}
}

func TestCmdListEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdList([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	if !strings.Contains(buf.String(), "no issues found") {
		t.Errorf("output = %q, want 'no issues found'", buf.String())
	}
}

func TestCmdListFilterByStatus(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To close", issue.CreateOpts{})
	env.Store.Close(iss.ID, "")
	env.Repo.Commit("create and close")

	var buf bytes.Buffer
	err := cmdList([]string{"--status", "closed"}, &buf)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	if !strings.Contains(buf.String(), "To close") {
		t.Errorf("output missing closed issue: %q", buf.String())
	}
}

// --- Close / Reopen ---

func TestCmdCloseBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Close me", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdClose([]string{iss.ID}, &buf)
	if err != nil {
		t.Fatalf("cmdClose: %v", err)
	}
	if !strings.Contains(buf.String(), "closed") {
		t.Errorf("output = %q", buf.String())
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "closed" {
		t.Errorf("status = %q, want closed", got.Status)
	}
}

func TestCmdCloseWithReason(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Close with reason", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdClose([]string{iss.ID, "--reason", "duplicate"}, &buf)
	if err != nil {
		t.Fatalf("cmdClose: %v", err)
	}
}

func TestCmdCloseJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Close JSON", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdClose([]string{iss.ID, "--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdClose --json: %v", err)
	}

	var got issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got.Status != "closed" {
		t.Errorf("status = %q, want closed", got.Status)
	}
}

func TestCmdCloseNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdClose([]string{"nonexistent"}, &buf)
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

func TestCmdReopenBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Reopen me", issue.CreateOpts{})
	env.Store.Close(iss.ID, "")
	env.Repo.Commit("create and close " + iss.ID)

	var buf bytes.Buffer
	err := cmdReopen([]string{iss.ID}, &buf)
	if err != nil {
		t.Fatalf("cmdReopen: %v", err)
	}
	if !strings.Contains(buf.String(), "reopened") {
		t.Errorf("output = %q", buf.String())
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
}

func TestCmdReopenJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Reopen JSON", issue.CreateOpts{})
	env.Store.Close(iss.ID, "")
	env.Repo.Commit("create and close " + iss.ID)

	var buf bytes.Buffer
	err := cmdReopen([]string{iss.ID, "--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdReopen --json: %v", err)
	}

	var got issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
}

func TestCmdReopenNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdReopen([]string{"nonexistent"}, &buf)
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

// --- Update ---

func TestCmdUpdatePriority(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Update me", issue.CreateOpts{Priority: intPtr(3)})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUpdate([]string{iss.ID, "--priority", "1"}, &buf)
	if err != nil {
		t.Fatalf("cmdUpdate: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Priority != 1 {
		t.Errorf("priority = %d, want 1", got.Priority)
	}
}

func TestCmdUpdateMultipleFields(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Multi update", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUpdate([]string{iss.ID, "--assignee", "bob", "--type", "bug", "--title", "Updated title"}, &buf)
	if err != nil {
		t.Fatalf("cmdUpdate: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Assignee != "bob" {
		t.Errorf("assignee = %q", got.Assignee)
	}
	if got.Type != "bug" {
		t.Errorf("type = %q", got.Type)
	}
	if got.Title != "Updated title" {
		t.Errorf("title = %q", got.Title)
	}
}

func TestCmdUpdateJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("JSON update", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUpdate([]string{iss.ID, "--assignee", "alice", "--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdUpdate: %v", err)
	}

	var got issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got.Assignee != "alice" {
		t.Errorf("assignee = %q", got.Assignee)
	}
}

func TestCmdUpdateDescription(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Desc update", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUpdate([]string{iss.ID, "--description", "new description"}, &buf)
	if err != nil {
		t.Fatalf("cmdUpdate: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Description != "new description" {
		t.Errorf("description = %q, want %q", got.Description, "new description")
	}
}

func TestCmdUpdateStatus(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Status update", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUpdate([]string{iss.ID, "--status", "in_progress"}, &buf)
	if err != nil {
		t.Fatalf("cmdUpdate: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "in_progress" {
		t.Errorf("status = %q, want in_progress", got.Status)
	}
}

func TestCmdUpdateNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdUpdate([]string{"nonexistent", "--title", "x"}, &buf)
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

// --- Label ---

func TestCmdLabelAdd(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Label me", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdLabel([]string{iss.ID, "+bug", "+urgent"}, &buf)
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
	err := cmdLabel([]string{iss.ID, "-bug"}, &buf)
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
	err := cmdLabel([]string{iss.ID, "+bug", "--json"}, &buf)
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
	err := cmdLabel([]string{iss.ID, "feature"}, &buf)
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
	err := cmdLabel([]string{"nonexistent", "+bug"}, &buf)
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

// --- Dep Add / Dep Remove ---

func TestCmdDepAddBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	err := cmdDepAdd([]string{a.ID, "blocks", b.ID}, &buf)
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
	err := cmdDepAdd([]string{"nonexistent", "blocks", "also-missing"}, &buf)
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
	err := cmdDepRemove([]string{a.ID, "blocks", b.ID}, &buf)
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
	err := cmdDepRemove([]string{"nonexistent", "blocks", "also-missing"}, &buf)
	if err == nil {
		t.Error("expected error for nonexistent issues")
	}
}

// --- Ready ---

func TestCmdReadyBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Ready issue", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	err := cmdReady([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	if !strings.Contains(buf.String(), "Ready issue") {
		t.Errorf("output missing issue: %q", buf.String())
	}
}

func TestCmdReadyJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Ready JSON", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	err := cmdReady([]string{"--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}

	var issues []issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &issues); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(issues) == 0 {
		t.Error("expected issues in JSON output")
	}
}

func TestCmdReadyEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdReady([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	if !strings.Contains(buf.String(), "no ready issues") {
		t.Errorf("output = %q", buf.String())
	}
}

// --- Graph ---

func TestCmdGraphAll(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Graph A", issue.CreateOpts{})
	b, _ := env.Store.Create("Graph B", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("create and link")

	var buf bytes.Buffer
	err := cmdGraph([]string{"--all"}, &buf)
	if err != nil {
		t.Fatalf("cmdGraph: %v", err)
	}
	if !strings.Contains(buf.String(), "Graph A") {
		t.Errorf("output missing Graph A: %q", buf.String())
	}
}

func TestCmdGraphJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Graph JSON", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	err := cmdGraph([]string{"--all", "--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdGraph: %v", err)
	}

	var nodes []issue.GraphNode
	if err := json.Unmarshal(buf.Bytes(), &nodes); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
}

func TestCmdGraphRooted(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Root", issue.CreateOpts{})
	b, _ := env.Store.Create("Child", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("create and link")

	var buf bytes.Buffer
	err := cmdGraph([]string{a.ID}, &buf)
	if err != nil {
		t.Fatalf("cmdGraph: %v", err)
	}
	if !strings.Contains(buf.String(), "Root") {
		t.Errorf("output missing Root: %q", buf.String())
	}
}

func TestCmdGraphEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdGraph([]string{"--all"}, &buf)
	if err != nil {
		t.Fatalf("cmdGraph --all: %v", err)
	}
	if !strings.Contains(buf.String(), "no issues") {
		t.Errorf("output = %q, want 'no issues'", buf.String())
	}
}

func TestCmdGraphMultipleRoots(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create a graph with two independent trees
	a, _ := env.Store.Create("Root A", issue.CreateOpts{})
	b, _ := env.Store.Create("Child B", issue.CreateOpts{})
	c, _ := env.Store.Create("Root C", issue.CreateOpts{})
	d, _ := env.Store.Create("Child D", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Link(c.ID, d.ID)
	env.Repo.Commit("setup graph")

	var buf bytes.Buffer
	err := cmdGraph([]string{"--all"}, &buf)
	if err != nil {
		t.Fatalf("cmdGraph: %v", err)
	}
	// Should show all 4 nodes
	out := buf.String()
	if !strings.Contains(out, "Root A") || !strings.Contains(out, "Child B") {
		t.Errorf("missing tree A in output: %q", out)
	}
	if !strings.Contains(out, "Root C") || !strings.Contains(out, "Child D") {
		t.Errorf("missing tree C in output: %q", out)
	}
}

func TestCmdGraphAllShowsClosedIssues(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Closed A", issue.CreateOpts{})
	b, _ := env.Store.Create("Closed B", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(a.ID, "")
	env.Store.Close(b.ID, "")
	env.Repo.Commit("create, link, and close")

	var buf bytes.Buffer
	err := cmdGraph([]string{"--all"}, &buf)
	if err != nil {
		t.Fatalf("cmdGraph --all: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "no issues") {
		t.Error("--all should show closed issues, got 'no issues'")
	}
	if !strings.Contains(out, "Closed A") || !strings.Contains(out, "Closed B") {
		t.Errorf("output missing closed issues: %q", out)
	}
}

func TestCmdGraphNotFound(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdGraph([]string{"nonexistent"}, &buf)
	if err == nil {
		t.Error("expected error for nonexistent root ID")
	}
}

// --- Config ---

func TestCmdConfigSetAndGet(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdConfig([]string{"set", "default.priority", "2"}, &buf)
	if err != nil {
		t.Fatalf("config set: %v", err)
	}
	if !strings.Contains(buf.String(), "default.priority=2") {
		t.Errorf("set output = %q", buf.String())
	}

	buf.Reset()
	err = cmdConfig([]string{"get", "default.priority"}, &buf)
	if err != nil {
		t.Fatalf("config get: %v", err)
	}
	if !strings.Contains(buf.String(), "2") {
		t.Errorf("get output = %q", buf.String())
	}
}

func TestCmdConfigList(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdConfig([]string{"list"}, &buf)
	if err != nil {
		t.Fatalf("config list: %v", err)
	}
	if !strings.Contains(buf.String(), "prefix=test") {
		t.Errorf("output = %q, want prefix=test", buf.String())
	}
}

func TestCmdConfigGetMissing(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdConfig([]string{"get", "nonexistent"}, &buf)
	if err == nil {
		t.Error("expected error for missing key")
	}
}




// --- Export ---

func TestCmdExportBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Export me", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	err := cmdExport([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdExport: %v", err)
	}

	// Should be valid JSONL
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 {
		t.Error("expected at least one line of JSONL output")
	}
	var rec beadsRecord
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("JSONL parse: %v", err)
	}
	if rec.Title != "Export me" {
		t.Errorf("title = %q", rec.Title)
	}
}

func TestDateConversionHelpers(t *testing.T) {
	// toRFC3339Date
	if got := toRFC3339Date(""); got != "" {
		t.Errorf("toRFC3339Date empty = %q", got)
	}
	if got := toRFC3339Date("2027-06-01"); got != "2027-06-01T00:00:00Z" {
		t.Errorf("toRFC3339Date bare = %q", got)
	}
	if got := toRFC3339Date("2027-06-01T12:00:00Z"); got != "2027-06-01T12:00:00Z" {
		t.Errorf("toRFC3339Date passthrough = %q", got)
	}

	// fromRFC3339Date
	if got := fromRFC3339Date(""); got != "" {
		t.Errorf("fromRFC3339Date empty = %q", got)
	}
	if got := fromRFC3339Date("2027-06-01T00:00:00Z"); got != "2027-06-01" {
		t.Errorf("fromRFC3339Date rfc3339 = %q", got)
	}
	if got := fromRFC3339Date("2027-06-01"); got != "2027-06-01" {
		t.Errorf("fromRFC3339Date bare = %q", got)
	}
}

func TestCmdExportWithParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Parent", issue.CreateOpts{})
	env.Store.Import(&issue.Issue{
		ID:       "test-child",
		Title:    "Child",
		Status:   "open",
		Priority: 3,
		Type:     "task",
		Created:  "2024-01-01",
		Parent:   parent.ID,
		Labels:   []string{},
		Blocks:   []string{},
		BlockedBy: []string{},
	})
	env.Repo.Commit("create with parent")

	var buf bytes.Buffer
	if err := cmdExport([]string{}, &buf); err != nil {
		t.Fatalf("cmdExport: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	for _, line := range lines {
		var rec beadsRecord
		json.Unmarshal([]byte(line), &rec)
		if rec.ID == "test-child" {
			foundParent := false
			for _, dep := range rec.Dependencies {
				if dep.Type == "parent-child" && dep.DependsOnID == parent.ID {
					foundParent = true
				}
			}
			if !foundParent {
				t.Errorf("child deps=%v, missing parent-child dep on %s", rec.Dependencies, parent.ID)
			}
			return
		}
	}
	t.Error("child issue not found in export")
}

// --- Import ---

func TestCmdImportBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create a JSONL file
	jsonl := `{"id":"test-imp1","title":"Imported","status":"open","priority":2,"issue_type":"task","created_at":"2024-01-01T00:00:00Z"}`
	tmpFile := env.Dir + "/import.jsonl"
	os.WriteFile(tmpFile, []byte(jsonl+"\n"), 0644)

	var buf bytes.Buffer
	err := cmdImport([]string{tmpFile}, &buf)
	if err != nil {
		t.Fatalf("cmdImport: %v", err)
	}
	if !strings.Contains(buf.String(), "imported 1") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestCmdImportDryRun(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	jsonl := `{"id":"test-dry1","title":"Dry run","status":"open","priority":2,"issue_type":"task","created_at":"2024-01-01T00:00:00Z"}`
	tmpFile := env.Dir + "/import.jsonl"
	os.WriteFile(tmpFile, []byte(jsonl+"\n"), 0644)

	var buf bytes.Buffer
	err := cmdImport([]string{tmpFile, "--dry-run"}, &buf)
	if err != nil {
		t.Fatalf("cmdImport: %v", err)
	}
	if !strings.Contains(buf.String(), "dry run") {
		t.Errorf("output = %q", buf.String())
	}

	// Issue should not exist
	_, getErr := env.Store.Get("test-dry1")
	if getErr == nil {
		t.Error("issue should not exist after dry run")
	}
}

func TestCmdImportPriorityZero(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// P0 should be preserved, not treated as "unset"
	jsonl := `{"id":"test-p0","title":"Critical","status":"open","priority":0,"issue_type":"bug","created_at":"2024-01-01T00:00:00Z"}`
	tmpFile := env.Dir + "/import.jsonl"
	os.WriteFile(tmpFile, []byte(jsonl+"\n"), 0644)

	var buf bytes.Buffer
	if err := cmdImport([]string{tmpFile}, &buf); err != nil {
		t.Fatalf("cmdImport: %v", err)
	}

	iss, err := env.Store.Get("test-p0")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if iss.Priority != 0 {
		t.Errorf("priority = %d, want 0 (P0)", iss.Priority)
	}
}

func TestCmdImportPriorityAbsent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// No priority field — should default to 2
	jsonl := `{"id":"test-nopr","title":"No priority","status":"open","issue_type":"task","created_at":"2024-01-01T00:00:00Z"}`
	tmpFile := env.Dir + "/import.jsonl"
	os.WriteFile(tmpFile, []byte(jsonl+"\n"), 0644)

	var buf bytes.Buffer
	if err := cmdImport([]string{tmpFile}, &buf); err != nil {
		t.Fatalf("cmdImport: %v", err)
	}

	iss, err := env.Store.Get("test-nopr")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if iss.Priority != 2 {
		t.Errorf("priority = %d, want 2 (default)", iss.Priority)
	}
}

// --- Sync ---

func TestCmdSyncNoRemote(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdSync([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdSync: %v", err)
	}
	if !strings.Contains(buf.String(), "no remote") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestCmdSyncPush(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.NewBareRemote()

	env.Store.Create("Sync test", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	err := cmdSync([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdSync: %v", err)
	}
	if !strings.Contains(buf.String(), "pushed") {
		t.Errorf("output = %q, want 'pushed'", buf.String())
	}
}

func TestCmdSyncUpToDate(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.NewBareRemote()

	env.Store.Create("Sync test", issue.CreateOpts{})
	env.Repo.Commit("create issue")
	env.Repo.Sync() // initial push

	var buf bytes.Buffer
	err := cmdSync([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdSync: %v", err)
	}
	if !strings.Contains(buf.String(), "up to date") {
		t.Errorf("output = %q, want 'up to date'", buf.String())
	}
}

func TestCmdSyncReplay(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()

	// Create a shared issue and push
	shared, _ := env.Store.Create("Shared", issue.CreateOpts{})
	env.Repo.Commit("create " + shared.ID + " p3 task \"Shared\"")
	env.Repo.Sync()

	// Clone and modify the same issue
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	statusIP := "in_progress"
	env2.Store.Update(shared.ID, issue.UpdateOpts{Status: &statusIP})
	env2.Repo.Commit("update " + shared.ID + " status=in_progress")
	env2.Repo.Sync()

	// Back to original, modify the same issue (potential conflict)
	env.SwitchTo()
	assignee := "agent"
	env.Store.Update(shared.ID, issue.UpdateOpts{Assignee: &assignee})
	env.Repo.Commit("update " + shared.ID + " assignee=agent")

	var buf bytes.Buffer
	err := cmdSync([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdSync: %v", err)
	}
	out := buf.String()
	// Should either replay or clean rebase
	if !strings.Contains(out, "replayed") && !strings.Contains(out, "rebased") {
		t.Errorf("output = %q, want replay or rebase", out)
	}
}

// --- Prime ---

func TestCmdPrimeBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Prime issue", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	err := cmdPrime(&buf)
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
	err := cmdPrime(&buf)
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
	err := cmdPrime(&buf)
	if err != nil {
		t.Fatalf("cmdPrime: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "0 open") {
		t.Errorf("output missing '0 open': %q", out)
	}
}

// --- Onboard ---

func TestCmdOnboardBasic(t *testing.T) {
	var buf bytes.Buffer
	err := cmdOnboard(&buf)
	if err != nil {
		t.Fatalf("cmdOnboard: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}

// --- Init ---

func TestCmdInitFresh(t *testing.T) {
	// Create a bare git repo (not initialized with beadwork)
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")
	os.WriteFile(dir+"/README", []byte("test"), 0644)
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	var buf bytes.Buffer
	err := cmdInit([]string{"--prefix", "fresh"}, &buf)
	if err != nil {
		t.Fatalf("cmdInit: %v", err)
	}
	if !strings.Contains(buf.String(), "initialized") {
		t.Errorf("output = %q, want 'initialized'", buf.String())
	}
	if !strings.Contains(buf.String(), "fresh") {
		t.Errorf("output = %q, want prefix 'fresh'", buf.String())
	}
}

func TestCmdInitAlreadyInitialized(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdInit([]string{}, &buf)
	if err == nil {
		t.Error("expected error for already initialized")
	}
	if !strings.Contains(err.Error(), "already initialized") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestCmdInitForce(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdInit([]string{"--force", "--prefix", "newpfx"}, &buf)
	if err != nil {
		t.Fatalf("cmdInit --force: %v", err)
	}
	if !strings.Contains(buf.String(), "reinitialized") {
		t.Errorf("output = %q", buf.String())
	}
}

// --- Helpers ---

func TestParseArgsBooleans(t *testing.T) {
	a, err := ParseArgs([]string{"--json", "--all", "positional"}, nil, []string{"--json", "--all"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.Bool("--json") {
		t.Error("expected --json to be true")
	}
	if !a.Bool("--all") {
		t.Error("expected --all to be true")
	}
	if a.Bool("--missing") {
		t.Error("expected --missing to be false")
	}
	if !a.JSON() {
		t.Error("expected JSON() to be true")
	}
}

func TestParseArgsValueFlags(t *testing.T) {
	a, err := ParseArgs([]string{"--status", "open", "--priority", "1"}, []string{"--status", "--priority"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.String("--status") != "open" {
		t.Errorf("status = %q, want open", a.String("--status"))
	}
	if a.Int("--priority") != 1 {
		t.Errorf("priority = %d, want 1", a.Int("--priority"))
	}
	if !a.Has("--status") {
		t.Error("expected Has(--status) to be true")
	}
	if a.Has("--missing") {
		t.Error("expected Has(--missing) to be false")
	}
}

func TestParseArgsAliases(t *testing.T) {
	a, err := ParseArgs([]string{"-p", "2", "-t", "bug", "-a", "alice"}, []string{"--priority", "--type", "--assignee"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.Int("--priority") != 2 {
		t.Errorf("priority = %d, want 2", a.Int("--priority"))
	}
	if a.String("--type") != "bug" {
		t.Errorf("type = %q, want bug", a.String("--type"))
	}
	if a.String("--assignee") != "alice" {
		t.Errorf("assignee = %q, want alice", a.String("--assignee"))
	}
}

func TestParseArgsPositionals(t *testing.T) {
	a, err := ParseArgs([]string{"my", "title", "here", "--json"}, nil, []string{"--json"})
	if err != nil {
		t.Fatal(err)
	}
	pos := a.Pos()
	if len(pos) != 3 {
		t.Fatalf("pos = %v, want 3 items", pos)
	}
	if a.PosFirst() != "my" {
		t.Errorf("PosFirst() = %q, want my", a.PosFirst())
	}
	if a.PosJoined() != "my title here" {
		t.Errorf("PosJoined() = %q", a.PosJoined())
	}
}

func TestParseArgsEmpty(t *testing.T) {
	a, err := ParseArgs([]string{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.PosFirst() != "" {
		t.Errorf("PosFirst() = %q, want empty", a.PosFirst())
	}
	if a.PosJoined() != "" {
		t.Errorf("PosJoined() = %q, want empty", a.PosJoined())
	}
	if a.JSON() {
		t.Error("expected JSON() to be false")
	}
}

func TestParseArgsValueFlagAtEnd(t *testing.T) {
	// Value flag with no following token should be silently ignored
	a, err := ParseArgs([]string{"--status"}, []string{"--status"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.Has("--status") {
		t.Error("expected Has(--status) to be false when no value follows")
	}
	if a.String("--status") != "" {
		t.Errorf("status = %q, want empty", a.String("--status"))
	}
}

func TestParseArgsIntErr(t *testing.T) {
	a, parseErr := ParseArgs([]string{"--priority", "abc"}, []string{"--priority"}, nil)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	_, set, err := a.IntErr("--priority")
	if !set {
		t.Error("expected set to be true")
	}
	if err == nil {
		t.Error("expected error for non-numeric priority")
	}

	// Missing flag
	_, set, err = a.IntErr("--missing")
	if set || err != nil {
		t.Errorf("expected set=false, err=nil for missing flag, got set=%v, err=%v", set, err)
	}

	// Valid int
	a2, parseErr := ParseArgs([]string{"--priority", "3"}, []string{"--priority"}, nil)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	n, set, err := a2.IntErr("--priority")
	if !set || err != nil || n != 3 {
		t.Errorf("expected (3, true, nil), got (%d, %v, %v)", n, set, err)
	}
}

func TestParseArgsMixedFlagsAndPositionals(t *testing.T) {
	a, err := ParseArgs([]string{"id-123", "--json", "--status", "open"}, []string{"--status"}, []string{"--json"})
	if err != nil {
		t.Fatal(err)
	}
	if a.PosFirst() != "id-123" {
		t.Errorf("PosFirst() = %q, want id-123", a.PosFirst())
	}
	if !a.JSON() {
		t.Error("expected JSON() to be true")
	}
	if a.String("--status") != "open" {
		t.Errorf("status = %q, want open", a.String("--status"))
	}
}

func TestHasFlag(t *testing.T) {
	tests := []struct {
		args []string
		flag string
		want bool
	}{
		{[]string{"--yes"}, "--yes", true},
		{[]string{"show", "id", "--json"}, "--json", true},
		{[]string{"show", "id"}, "--json", false},
		{[]string{}, "--json", false},
	}
	for _, tt := range tests {
		got := hasFlag(tt.args, tt.flag)
		if got != tt.want {
			t.Errorf("hasFlag(%v, %q) = %v, want %v", tt.args, tt.flag, got, tt.want)
		}
	}
}

func TestFprintJSON(t *testing.T) {
	var buf bytes.Buffer
	fprintJSON(&buf, map[string]string{"key": "value"})
	if !strings.Contains(buf.String(), `"key": "value"`) {
		t.Errorf("output = %q", buf.String())
	}
}

func TestFprintIssue(t *testing.T) {
	iss := &issue.Issue{
		ID:       "test-1234",
		Title:    "Test issue",
		Status:   "open",
		Priority: 2,
		Type:     "task",
		Assignee: "alice",
		Created:  "2024-01-15T12:00:00Z",
		Labels:   []string{"bug"},
	}

	var buf bytes.Buffer
	fprintIssue(&buf, iss)
	out := buf.String()
	if !strings.Contains(out, "test-1234") {
		t.Errorf("missing ID in output: %q", out)
	}
	if !strings.Contains(out, "Test issue") {
		t.Errorf("missing title in output: %q", out)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("missing assignee in output: %q", out)
	}
	if !strings.Contains(out, "Labels: bug") {
		t.Errorf("missing labels in output: %q", out)
	}
}

func TestFprintIssueFull(t *testing.T) {
	iss := &issue.Issue{
		ID:          "test-5678",
		Title:       "Full issue",
		Status:      "in_progress",
		Priority:    1,
		Type:        "bug",
		Assignee:    "",
		Created:     "2024-06-15T12:00:00Z",
		Labels:      []string{},
		Blocks:      []string{"test-aaaa"},
		BlockedBy:   []string{"test-bbbb"},
		Parent:      "test-cccc",
		Description: "Line one\nLine two",
	}

	var buf bytes.Buffer
	fprintIssue(&buf, iss)
	out := buf.String()
	if !strings.Contains(out, "[BUG]") {
		t.Errorf("missing [BUG] type tag: %q", out)
	}
	// Deps are now rendered by fprintDeps (not fprintIssue)
	if !strings.Contains(out, "Parent: test-cccc") {
		t.Errorf("missing Parent in output: %q", out)
	}
	if !strings.Contains(out, "DESCRIPTION") {
		t.Errorf("missing DESCRIPTION in output: %q", out)
	}
	if !strings.Contains(out, "Line one") || !strings.Contains(out, "Line two") {
		t.Errorf("missing description lines in output: %q", out)
	}
	// No assignee → should show "—"
	if !strings.Contains(out, "—") {
		t.Errorf("missing dash for empty assignee: %q", out)
	}
}

func TestFprintIssueTypeTag(t *testing.T) {
	iss := &issue.Issue{
		ID:       "test-9999",
		Title:    "Bug report",
		Status:   "open",
		Priority: 1,
		Type:     "bug",
		Created:  "2024-01-15T12:00:00Z",
	}

	var buf bytes.Buffer
	fprintIssue(&buf, iss)
	out := buf.String()
	if !strings.Contains(out, "[BUG]") {
		t.Errorf("should contain [BUG] tag: %q", out)
	}

	// Task type should NOT have a tag
	iss.Type = "task"
	buf.Reset()
	fprintIssue(&buf, iss)
	out = buf.String()
	if strings.Contains(out, "[TASK]") {
		t.Errorf("task should not have type tag: %q", out)
	}
}

func TestFprintIssueDateLine(t *testing.T) {
	iss := &issue.Issue{
		ID:        "test-date",
		Title:     "Date test",
		Status:    "open",
		Priority:  2,
		Type:      "task",
		Created:   "2024-01-15T12:00:00Z",
		UpdatedAt: "2024-02-20T14:00:00Z",
	}

	var buf bytes.Buffer
	fprintIssue(&buf, iss)
	out := buf.String()
	if !strings.Contains(out, "Created: 2024-01-15") {
		t.Errorf("missing Created date: %q", out)
	}
	if !strings.Contains(out, "Updated: 2024-02-20") {
		t.Errorf("missing Updated date: %q", out)
	}

	// Deferred should be on same line
	iss.DeferUntil = "2027-06-01"
	buf.Reset()
	fprintIssue(&buf, iss)
	out = buf.String()
	if !strings.Contains(out, "Deferred: 2027-06-01") {
		t.Errorf("missing Deferred date: %q", out)
	}
}

func TestFprintIssueCloseReason(t *testing.T) {
	iss := &issue.Issue{
		ID:          "test-closed",
		Title:       "Closed issue",
		Status:      "closed",
		Priority:    2,
		Type:        "task",
		Created:     "2024-01-15T12:00:00Z",
		ClosedAt:    "2024-03-01T10:00:00Z",
		CloseReason: "duplicate",
	}

	var buf bytes.Buffer
	fprintIssue(&buf, iss)
	out := buf.String()
	if !strings.Contains(out, "Close reason: duplicate") {
		t.Errorf("should contain close reason: %q", out)
	}
}

func TestGetInitializedWithDefaultPriority(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Set a default priority config
	env.Repo.SetConfig("default.priority", "2")
	env.Repo.Commit("config default.priority=2")

	_, store, err := getInitializedRepo()
	if err != nil {
		t.Fatalf("getInitializedRepo: %v", err)
	}
	if store.DefaultPriority == nil || *store.DefaultPriority != 2 {
		t.Errorf("DefaultPriority = %v, want 2", store.DefaultPriority)
	}
}

func TestGetInitializedReturnsError(t *testing.T) {
	// getRepo / getInitializedRepo should return errors, not crash
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Verify it works when initialized
	r, s, err := getInitializedRepo()
	if err != nil {
		t.Fatalf("getInitializedRepo: %v", err)
	}
	if r == nil || s == nil {
		t.Error("expected non-nil repo and store")
	}
}

// --- Blocked ---

func TestCmdBlockedBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{Priority: intPtr(1)})
	b, _ := env.Store.Create("Blocked task", issue.CreateOpts{Priority: intPtr(2)})
	env.Store.Link(a.ID, b.ID)
	env.Repo.Commit("link")

	var buf bytes.Buffer
	err := cmdBlocked([]string{}, &buf)
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
	err := cmdBlocked([]string{}, &buf)
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
	err := cmdBlocked([]string{"--json"}, &buf)
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
	err := cmdBlocked([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdBlocked: %v", err)
	}
	if !strings.Contains(buf.String(), "no blocked issues") {
		t.Errorf("expected 'no blocked issues', got: %q", buf.String())
	}
}

// --- Defer / Undefer ---

func TestCmdDefer(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Defer me", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdDefer([]string{iss.ID, "2027-06-01"}, &buf)
	if err != nil {
		t.Fatalf("cmdDefer: %v", err)
	}
	if !strings.Contains(buf.String(), "deferred") {
		t.Errorf("output = %q, want 'deferred'", buf.String())
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "deferred" {
		t.Errorf("status = %q, want deferred", got.Status)
	}
	if got.DeferUntil != "2027-06-01" {
		t.Errorf("defer_until = %q, want 2027-06-01", got.DeferUntil)
	}
}

func TestCmdDeferJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Defer JSON", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdDefer([]string{iss.ID, "2027-06-01", "--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdDefer --json: %v", err)
	}

	var got issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got.Status != "deferred" {
		t.Errorf("status = %q, want deferred", got.Status)
	}
	if got.DeferUntil != "2027-06-01" {
		t.Errorf("defer_until = %q, want 2027-06-01", got.DeferUntil)
	}
}

func TestCmdDeferInvalidDate(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Bad date", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdDefer([]string{iss.ID, "not-a-date"}, &buf)
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestCmdUndefer(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Undefer me", issue.CreateOpts{DeferUntil: "2027-06-01"})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUndefer([]string{iss.ID}, &buf)
	if err != nil {
		t.Fatalf("cmdUndefer: %v", err)
	}
	if !strings.Contains(buf.String(), "undeferred") {
		t.Errorf("output = %q, want 'undeferred'", buf.String())
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
	if got.DeferUntil != "" {
		t.Errorf("defer_until = %q, want empty", got.DeferUntil)
	}
}

func TestCmdUndeferJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Undefer JSON", issue.CreateOpts{DeferUntil: "2027-06-01"})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUndefer([]string{iss.ID, "--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdUndefer --json: %v", err)
	}

	var got issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
}

func TestCmdCreateWithDefer(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate([]string{"Deferred task", "--defer", "2027-03-15"}, &buf)
	if err != nil {
		t.Fatalf("cmdCreate --defer: %v", err)
	}

	issues, _ := env.Store.List(issue.Filter{Status: "deferred"})
	if len(issues) != 1 {
		t.Fatalf("expected 1 deferred issue, got %d", len(issues))
	}
	if issues[0].DeferUntil != "2027-03-15" {
		t.Errorf("defer_until = %q, want 2027-03-15", issues[0].DeferUntil)
	}
}

func TestCmdUpdateWithDefer(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Update defer", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUpdate([]string{iss.ID, "--defer", "2027-09-01"}, &buf)
	if err != nil {
		t.Fatalf("cmdUpdate --defer: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "deferred" {
		t.Errorf("status = %q, want deferred", got.Status)
	}
	if got.DeferUntil != "2027-09-01" {
		t.Errorf("defer_until = %q, want 2027-09-01", got.DeferUntil)
	}
}

func TestCmdListDeferred(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Open task", issue.CreateOpts{})
	env.Store.Create("Deferred task", issue.CreateOpts{DeferUntil: "2027-01-01"})
	env.Repo.Commit("create issues")

	// Default list should show only open
	var buf bytes.Buffer
	err := cmdList([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	if !strings.Contains(buf.String(), "Open task") {
		t.Errorf("default list should show open task: %q", buf.String())
	}
	if strings.Contains(buf.String(), "Deferred task") {
		t.Error("default list should NOT show deferred task")
	}

	// --deferred should show only deferred
	buf.Reset()
	err = cmdList([]string{"--deferred"}, &buf)
	if err != nil {
		t.Fatalf("cmdList --deferred: %v", err)
	}
	if !strings.Contains(buf.String(), "Deferred task") {
		t.Errorf("--deferred should show deferred task: %q", buf.String())
	}
	if strings.Contains(buf.String(), "Open task") {
		t.Error("--deferred should NOT show open task")
	}
}

func TestCmdReadyExcludesInProgress(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open task", issue.CreateOpts{})
	b, _ := env.Store.Create("WIP task", issue.CreateOpts{})
	statusIP := "in_progress"
	env.Store.Update(b.ID, issue.UpdateOpts{Status: &statusIP})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	err := cmdReady([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, a.ID) {
		t.Errorf("output should contain open task %s: %q", a.ID, out)
	}
	if strings.Contains(out, b.ID) {
		t.Errorf("output should NOT contain in_progress task %s: %q", b.ID, out)
	}
}

func TestCmdReadyExcludesDeferred(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open task", issue.CreateOpts{})
	env.Store.Create("Deferred task", issue.CreateOpts{DeferUntil: "2027-06-01"})
	env.Repo.Commit("create issues")

	var buf bytes.Buffer
	err := cmdReady([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdReady: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, a.ID) {
		t.Errorf("output should contain open task %s: %q", a.ID, out)
	}
	if strings.Contains(out, "Deferred task") {
		t.Error("output should NOT contain deferred task")
	}
}

func TestCmdShowDeferred(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Deferred show", issue.CreateOpts{DeferUntil: "2027-04-15"})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdShow([]string{iss.ID}, &buf)
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Deferred: 2027-04-15") {
		t.Errorf("output should contain 'Deferred: 2027-04-15': %q", out)
	}
	if !strings.Contains(out, "DEFERRED") {
		t.Errorf("output should contain DEFERRED status: %q", out)
	}
}

// --- Scenario: Full Workflow ---
// Exercises create, label, dep, close, update, defer across 5 issues,
// then verifies state through list, show, ready, blocked, graph, and export.

func TestScenarioFullWorkflow(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// ── Step 1: Create 5 issues ──
	var bufA bytes.Buffer
	if err := cmdCreate([]string{"API design", "--priority", "1", "--type", "bug", "--assignee", "alice"}, &bufA); err != nil {
		t.Fatalf("create A: %v", err)
	}
	var bufB bytes.Buffer
	if err := cmdCreate([]string{"Database schema", "--priority", "1", "--type", "task"}, &bufB); err != nil {
		t.Fatalf("create B: %v", err)
	}
	var bufC bytes.Buffer
	if err := cmdCreate([]string{"Frontend UI", "--priority", "2", "--type", "task"}, &bufC); err != nil {
		t.Fatalf("create C: %v", err)
	}
	var bufD bytes.Buffer
	if err := cmdCreate([]string{"Documentation", "--priority", "3", "--type", "task", "--defer", "2027-06-01"}, &bufD); err != nil {
		t.Fatalf("create D: %v", err)
	}
	var bufE bytes.Buffer
	if err := cmdCreate([]string{"Performance audit", "--priority", "2", "--type", "task"}, &bufE); err != nil {
		t.Fatalf("create E: %v", err)
	}

	// Collect IDs from store (sorted by creation order via list --all)
	allIssues, _ := env.Store.List(issue.Filter{})
	if len(allIssues) != 5 {
		t.Fatalf("expected 5 issues, got %d", len(allIssues))
	}
	idOf := make(map[string]string) // title → ID
	for _, iss := range allIssues {
		idOf[iss.Title] = iss.ID
	}
	idA := idOf["API design"]
	idB := idOf["Database schema"]
	idC := idOf["Frontend UI"]
	idD := idOf["Documentation"]
	idE := idOf["Performance audit"]

	// ── Step 2: Wire dependencies ──
	// A blocks B, B blocks C
	var buf bytes.Buffer
	if err := cmdDepAdd([]string{idA, "blocks", idB}, &buf); err != nil {
		t.Fatalf("dep add A→B: %v", err)
	}
	buf.Reset()
	if err := cmdDepAdd([]string{idB, "blocks", idC}, &buf); err != nil {
		t.Fatalf("dep add B→C: %v", err)
	}

	// ── Step 3: Label ──
	buf.Reset()
	if err := cmdLabel([]string{idA, "+backend", "+critical"}, &buf); err != nil {
		t.Fatalf("label A: %v", err)
	}
	buf.Reset()
	if err := cmdLabel([]string{idC, "+frontend"}, &buf); err != nil {
		t.Fatalf("label C: %v", err)
	}

	// ── Step 4: Progress ──
	// Close A
	buf.Reset()
	if err := cmdClose([]string{idA}, &buf); err != nil {
		t.Fatalf("close A: %v", err)
	}
	// B to in_progress
	buf.Reset()
	if err := cmdUpdate([]string{idB, "--status", "in_progress"}, &buf); err != nil {
		t.Fatalf("update B: %v", err)
	}
	// Defer E
	buf.Reset()
	if err := cmdDefer([]string{idE, "2027-09-01"}, &buf); err != nil {
		t.Fatalf("defer E: %v", err)
	}

	// ════════════════════════════════════════════════════════
	// Verify: list --all --json
	// ════════════════════════════════════════════════════════
	t.Run("list_all_json", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdList([]string{"--all", "--json"}, &buf); err != nil {
			t.Fatalf("list --all --json: %v", err)
		}
		var issues []issue.Issue
		if err := json.Unmarshal(buf.Bytes(), &issues); err != nil {
			t.Fatalf("JSON parse: %v", err)
		}
		if len(issues) != 5 {
			t.Fatalf("expected 5 issues, got %d", len(issues))
		}

		byID := make(map[string]*issue.Issue)
		for i := range issues {
			byID[issues[i].ID] = &issues[i]
		}

		// A: closed, P1, bug, alice, labels=[backend,critical]
		a := byID[idA]
		if a == nil {
			t.Fatal("issue A missing")
		}
		if a.Status != "closed" {
			t.Errorf("A status=%q, want closed", a.Status)
		}
		if a.Priority != 1 {
			t.Errorf("A priority=%d, want 1", a.Priority)
		}
		if a.Type != "bug" {
			t.Errorf("A type=%q, want bug", a.Type)
		}
		if a.Assignee != "alice" {
			t.Errorf("A assignee=%q, want alice", a.Assignee)
		}
		if len(a.Labels) != 2 {
			t.Errorf("A labels=%v, want 2 labels", a.Labels)
		}

		// B: in_progress, P1, task, blocked by A, blocks C
		b := byID[idB]
		if b == nil {
			t.Fatal("issue B missing")
		}
		if b.Status != "in_progress" {
			t.Errorf("B status=%q, want in_progress", b.Status)
		}
		if len(b.BlockedBy) != 1 || b.BlockedBy[0] != idA {
			t.Errorf("B blocked_by=%v, want [%s]", b.BlockedBy, idA)
		}
		if len(b.Blocks) != 1 || b.Blocks[0] != idC {
			t.Errorf("B blocks=%v, want [%s]", b.Blocks, idC)
		}

		// C: open, P2, task, blocked by B, labels=[frontend]
		c := byID[idC]
		if c == nil {
			t.Fatal("issue C missing")
		}
		if c.Status != "open" {
			t.Errorf("C status=%q, want open", c.Status)
		}
		if len(c.BlockedBy) != 1 || c.BlockedBy[0] != idB {
			t.Errorf("C blocked_by=%v, want [%s]", c.BlockedBy, idB)
		}
		if len(c.Labels) != 1 || c.Labels[0] != "frontend" {
			t.Errorf("C labels=%v, want [frontend]", c.Labels)
		}

		// D: deferred, P3, task, defer_until=2027-06-01
		d := byID[idD]
		if d == nil {
			t.Fatal("issue D missing")
		}
		if d.Status != "deferred" {
			t.Errorf("D status=%q, want deferred", d.Status)
		}
		if d.DeferUntil != "2027-06-01" {
			t.Errorf("D defer_until=%q, want 2027-06-01", d.DeferUntil)
		}

		// E: deferred, P2, task, defer_until=2027-09-01
		e := byID[idE]
		if e == nil {
			t.Fatal("issue E missing")
		}
		if e.Status != "deferred" {
			t.Errorf("E status=%q, want deferred", e.Status)
		}
		if e.DeferUntil != "2027-09-01" {
			t.Errorf("E defer_until=%q, want 2027-09-01", e.DeferUntil)
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: list (default = open) — only C
	// ════════════════════════════════════════════════════════
	t.Run("list_default", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdList([]string{}, &buf); err != nil {
			t.Fatalf("list: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, idC) {
			t.Errorf("default list should contain C (%s): %q", idC, out)
		}
		// A (closed), B (in_progress), D (deferred), E (deferred) should not appear
		for _, id := range []string{idA, idB, idD, idE} {
			if strings.Contains(out, id) {
				t.Errorf("default list should NOT contain %s: %q", id, out)
			}
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: list --status in_progress — only B
	// ════════════════════════════════════════════════════════
	t.Run("list_in_progress", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdList([]string{"--status", "in_progress"}, &buf); err != nil {
			t.Fatalf("list --status in_progress: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, idB) {
			t.Errorf("should contain B (%s): %q", idB, out)
		}
		if strings.Contains(out, idA) || strings.Contains(out, idC) {
			t.Errorf("should not contain A or C: %q", out)
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: list --deferred — D and E
	// ════════════════════════════════════════════════════════
	t.Run("list_deferred", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdList([]string{"--deferred"}, &buf); err != nil {
			t.Fatalf("list --deferred: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, idD) {
			t.Errorf("should contain D (%s): %q", idD, out)
		}
		if !strings.Contains(out, idE) {
			t.Errorf("should contain E (%s): %q", idE, out)
		}
		if strings.Contains(out, idA) || strings.Contains(out, idB) || strings.Contains(out, idC) {
			t.Errorf("should not contain A, B, or C: %q", out)
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: show --json on D
	// ════════════════════════════════════════════════════════
	t.Run("show_deferred", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdShow([]string{idD, "--json"}, &buf); err != nil {
			t.Fatalf("show D: %v", err)
		}
		var arr []issue.Issue
		if err := json.Unmarshal(buf.Bytes(), &arr); err != nil {
			t.Fatalf("JSON parse: %v", err)
		}
		if len(arr) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(arr))
		}
		d := arr[0]
		if d.Status != "deferred" {
			t.Errorf("D status=%q, want deferred", d.Status)
		}
		if d.DeferUntil != "2027-06-01" {
			t.Errorf("D defer_until=%q, want 2027-06-01", d.DeferUntil)
		}
		if d.Title != "Documentation" {
			t.Errorf("D title=%q, want Documentation", d.Title)
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: ready — no ready issues (C is blocked, others not open)
	// ════════════════════════════════════════════════════════
	t.Run("ready_text", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdReady([]string{}, &buf); err != nil {
			t.Fatalf("ready: %v", err)
		}
		if !strings.Contains(buf.String(), "no ready issues") {
			t.Errorf("expected 'no ready issues', got: %q", buf.String())
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: ready --json — empty array
	// ════════════════════════════════════════════════════════
	t.Run("ready_json", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdReady([]string{"--json"}, &buf); err != nil {
			t.Fatalf("ready --json: %v", err)
		}
		var issues []issue.Issue
		if err := json.Unmarshal(buf.Bytes(), &issues); err != nil {
			t.Fatalf("JSON parse: %v", err)
		}
		if len(issues) != 0 {
			t.Errorf("expected 0 ready issues, got %d", len(issues))
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: blocked --json — C blocked by B
	// ════════════════════════════════════════════════════════
	t.Run("blocked_json", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdBlocked([]string{"--json"}, &buf); err != nil {
			t.Fatalf("blocked --json: %v", err)
		}
		var result []struct {
			ID           string   `json:"id"`
			OpenBlockers []string `json:"open_blockers"`
		}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("JSON parse: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 blocked issue, got %d", len(result))
		}
		if result[0].ID != idC {
			t.Errorf("blocked id=%q, want %q", result[0].ID, idC)
		}
		if len(result[0].OpenBlockers) != 0 {
			// B is in_progress, which counts as an open blocker for the blocked list
			// The Blocked() method checks if blockers have status != "closed"
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: graph --all
	// ════════════════════════════════════════════════════════
	t.Run("graph_all", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdGraph([]string{"--all"}, &buf); err != nil {
			t.Fatalf("graph --all: %v", err)
		}
		out := buf.String()
		// A, B, C should be in a chain
		if !strings.Contains(out, idA) {
			t.Errorf("graph should contain A: %q", out)
		}
		if !strings.Contains(out, idB) {
			t.Errorf("graph should contain B: %q", out)
		}
		if !strings.Contains(out, idC) {
			t.Errorf("graph should contain C: %q", out)
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: export — JSONL with all fields
	// ════════════════════════════════════════════════════════
	t.Run("export", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdExport([]string{}, &buf); err != nil {
			t.Fatalf("export: %v", err)
		}
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if len(lines) != 5 {
			t.Fatalf("expected 5 JSONL lines, got %d", len(lines))
		}

		byID := make(map[string]beadsRecord)
		for _, line := range lines {
			var rec beadsRecord
			if err := json.Unmarshal([]byte(line), &rec); err != nil {
				t.Fatalf("JSONL parse: %v\nline: %s", err, line)
			}
			byID[rec.ID] = rec
		}

		// A: closed, labels=[backend,critical], blocks=[B]
		a := byID[idA]
		if a.Status != "closed" {
			t.Errorf("export A status=%q, want closed", a.Status)
		}
		if len(a.Labels) != 2 {
			t.Errorf("export A labels=%v, want 2 labels", a.Labels)
		}
		if len(a.Blocks) != 1 || a.Blocks[0] != idB {
			t.Errorf("export A blocks=%v, want [%s]", a.Blocks, idB)
		}
		if a.UpdatedAt == "" {
			t.Error("export A updated_at should be set")
		}
		if a.ClosedAt == "" {
			t.Error("export A closed_at should be set (issue is closed)")
		}

		// B: in_progress, blocked_by=[A], blocks=[C]
		b := byID[idB]
		if b.Status != "in_progress" {
			t.Errorf("export B status=%q, want in_progress", b.Status)
		}
		if len(b.BlockedBy) != 1 || b.BlockedBy[0] != idA {
			t.Errorf("export B blocked_by=%v, want [%s]", b.BlockedBy, idA)
		}
		if len(b.Blocks) != 1 || b.Blocks[0] != idC {
			t.Errorf("export B blocks=%v, want [%s]", b.Blocks, idC)
		}

		// C: open, blocked_by=[B], labels=[frontend]
		c := byID[idC]
		if c.Status != "open" {
			t.Errorf("export C status=%q, want open", c.Status)
		}
		if len(c.BlockedBy) != 1 || c.BlockedBy[0] != idB {
			t.Errorf("export C blocked_by=%v, want [%s]", c.BlockedBy, idB)
		}
		if len(c.Labels) != 1 || c.Labels[0] != "frontend" {
			t.Errorf("export C labels=%v, want [frontend]", c.Labels)
		}
		if c.ClosedAt != "" {
			t.Errorf("export C closed_at should be empty for open issue, got %q", c.ClosedAt)
		}
		if c.UpdatedAt == "" {
			t.Error("export C updated_at should be set")
		}

		// D: deferred, defer_until as RFC3339
		d := byID[idD]
		if d.Status != "deferred" {
			t.Errorf("export D status=%q, want deferred", d.Status)
		}
		if d.DeferUntil != "2027-06-01T00:00:00Z" {
			t.Errorf("export D defer_until=%q, want 2027-06-01T00:00:00Z", d.DeferUntil)
		}

		// E: deferred, defer_until as RFC3339
		e := byID[idE]
		if e.Status != "deferred" {
			t.Errorf("export E status=%q, want deferred", e.Status)
		}
		if e.DeferUntil != "2027-09-01T00:00:00Z" {
			t.Errorf("export E defer_until=%q, want 2027-09-01T00:00:00Z", e.DeferUntil)
		}

		// Verify dependencies array for B (should have A as blocker)
		foundBlockDep := false
		for _, dep := range b.Dependencies {
			if dep.Type == "blocks" && dep.DependsOnID == idA {
				foundBlockDep = true
			}
		}
		if !foundBlockDep {
			t.Errorf("export B deps=%v, missing blocks dep on A", b.Dependencies)
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: export → import round-trip
	// ════════════════════════════════════════════════════════
	t.Run("export_import_roundtrip", func(t *testing.T) {
		// Export to JSONL
		var exportBuf bytes.Buffer
		if err := cmdExport([]string{}, &exportBuf); err != nil {
			t.Fatalf("export: %v", err)
		}

		// Write to temp file
		tmpFile := env.Dir + "/roundtrip.jsonl"
		os.WriteFile(tmpFile, exportBuf.Bytes(), 0644)

		// Try dry-run import — should detect all 5 as collisions
		var dryBuf bytes.Buffer
		if err := cmdImport([]string{tmpFile, "--dry-run"}, &dryBuf); err != nil {
			t.Fatalf("import dry-run: %v", err)
		}
		out := dryBuf.String()
		if !strings.Contains(out, "5 skipped") {
			t.Errorf("expected '5 skipped' in dry-run output: %q", out)
		}
		if !strings.Contains(out, "importing 0 of 5") {
			t.Errorf("expected 'importing 0 of 5' in dry-run output: %q", out)
		}
	})
}

// --- Version Gate ---

func TestVersionGateBlocksOlderRepo(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Downgrade to v0 by removing version key
	env.Repo.SetConfig("version", "0")
	env.Repo.Commit("downgrade to v0")

	var buf bytes.Buffer
	err := cmdList([]string{}, &buf)
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

	var buf bytes.Buffer
	err := cmdList([]string{}, &buf)
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
	err := cmdUpgradeRepo([]string{}, &buf)
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
	err = cmdList([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdList after upgrade: %v", err)
	}
}

func TestCmdUpgradeRepoAlreadyCurrent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdUpgradeRepo([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdUpgradeRepo: %v", err)
	}
	if !strings.Contains(buf.String(), "up to date") {
		t.Errorf("output should contain 'up to date': %q", buf.String())
	}
}

// --- Defer (additional coverage) ---

func TestCmdDeferNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdDefer([]string{}, &buf)
	if err == nil {
		t.Error("expected error for no args")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q, want usage message", err)
	}
}

func TestCmdDeferOneArg(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdDefer([]string{"bw-1234"}, &buf)
	if err == nil {
		t.Error("expected error for missing date arg")
	}
}

func TestCmdDeferNonExistentIssue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdDefer([]string{"bw-0000", "2027-06-01"}, &buf)
	if err == nil {
		t.Error("expected error for non-existent issue")
	}
}

func TestCmdDeferVariousInvalidDates(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Date test", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	invalidDates := []string{
		"2027/06/01",
		"06-01-2027",
		"tomorrow",
		"2027-13-01",
		"2027-06-32",
		"",
	}
	for _, d := range invalidDates {
		var buf bytes.Buffer
		err := cmdDefer([]string{iss.ID, d}, &buf)
		if err == nil {
			t.Errorf("expected error for invalid date %q", d)
		}
	}
}

func TestCmdDeferAlreadyDeferred(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Already deferred", issue.CreateOpts{DeferUntil: "2027-01-01"})
	env.Repo.Commit("create " + iss.ID)

	// Defer again with a new date
	var buf bytes.Buffer
	err := cmdDefer([]string{iss.ID, "2027-12-01"}, &buf)
	if err != nil {
		t.Fatalf("cmdDefer on already deferred: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.DeferUntil != "2027-12-01" {
		t.Errorf("defer_until = %q, want 2027-12-01", got.DeferUntil)
	}
}

func TestCmdDeferUnknownFlag(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Flag test", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdDefer([]string{iss.ID, "2027-06-01", "--unknown"}, &buf)
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestValidateDate(t *testing.T) {
	valid := []string{"2027-01-01", "2026-12-31", "2030-06-15"}
	for _, d := range valid {
		if err := validateDate(d); err != nil {
			t.Errorf("validateDate(%q) = %v, want nil", d, err)
		}
	}

	invalid := []string{"not-a-date", "2027/01/01", "01-01-2027", "2027-13-01", ""}
	for _, d := range invalid {
		if err := validateDate(d); err == nil {
			t.Errorf("validateDate(%q) = nil, want error", d)
		}
	}
}

// --- Undefer (additional coverage) ---

func TestCmdUndeferNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdUndefer([]string{}, &buf)
	if err == nil {
		t.Error("expected error for no args")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q, want usage message", err)
	}
}

func TestCmdUndeferNonExistentIssue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdUndefer([]string{"bw-0000"}, &buf)
	if err == nil {
		t.Error("expected error for non-existent issue")
	}
}

func TestCmdUndeferAlreadyOpen(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Already open", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	// Undefer an issue that's already open (no defer_until set)
	var buf bytes.Buffer
	err := cmdUndefer([]string{iss.ID}, &buf)
	if err != nil {
		t.Fatalf("cmdUndefer on open issue: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
	if got.DeferUntil != "" {
		t.Errorf("defer_until = %q, want empty", got.DeferUntil)
	}
}

func TestCmdUndeferUnknownFlag(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Flag test", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUndefer([]string{iss.ID, "--unknown"}, &buf)
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestCmdUndeferVerifyCommit(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Verify commit", issue.CreateOpts{DeferUntil: "2027-06-01"})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdUndefer([]string{iss.ID}, &buf)
	if err != nil {
		t.Fatalf("cmdUndefer: %v", err)
	}
	if !strings.Contains(buf.String(), "undeferred") {
		t.Errorf("output = %q, want 'undeferred'", buf.String())
	}

	// Verify the issue was updated and committed
	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
	if got.DeferUntil != "" {
		t.Errorf("defer_until = %q, want empty", got.DeferUntil)
	}
}

func TestParseUndeferArgs(t *testing.T) {
	ua, err := parseUndeferArgs([]string{"bw-1234"})
	if err != nil {
		t.Fatalf("parseUndeferArgs: %v", err)
	}
	if ua.ID != "bw-1234" {
		t.Errorf("ID = %q, want bw-1234", ua.ID)
	}
	if ua.JSON {
		t.Error("expected JSON=false")
	}
}

func TestParseUndeferArgsJSON(t *testing.T) {
	ua, err := parseUndeferArgs([]string{"bw-1234", "--json"})
	if err != nil {
		t.Fatalf("parseUndeferArgs: %v", err)
	}
	if ua.ID != "bw-1234" {
		t.Errorf("ID = %q, want bw-1234", ua.ID)
	}
	if !ua.JSON {
		t.Error("expected JSON=true")
	}
}

// --- Init (additional coverage) ---

func TestCmdInitDefaultPrefix(t *testing.T) {
	// Init without --prefix should derive prefix from repo name
	dir := t.TempDir()
	runInDir := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}
	runInDir("git", "init")
	runInDir("git", "config", "user.email", "test@test.com")
	runInDir("git", "config", "user.name", "Test")
	os.WriteFile(dir+"/README", []byte("test"), 0644)
	runInDir("git", "add", ".")
	runInDir("git", "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	var buf bytes.Buffer
	err := cmdInit([]string{}, &buf)
	if err != nil {
		t.Fatalf("cmdInit: %v", err)
	}
	if !strings.Contains(buf.String(), "initialized") {
		t.Errorf("output = %q, want 'initialized'", buf.String())
	}
}

func TestCmdInitForceInvalidPrefix(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdInit([]string{"--force", "--prefix", "has space"}, &buf)
	if err == nil {
		t.Error("expected error for invalid prefix")
	}
}

func TestCmdInitUnknownFlag(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdInit([]string{"--verbose"}, &buf)
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestCmdInitForceDefaultPrefix(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Force reinit without specifying prefix (uses empty string -> derives from repo)
	var buf bytes.Buffer
	err := cmdInit([]string{"--force"}, &buf)
	if err != nil {
		t.Fatalf("cmdInit --force: %v", err)
	}
	if !strings.Contains(buf.String(), "reinitialized") {
		t.Errorf("output = %q, want 'reinitialized'", buf.String())
	}
}

func TestCmdInitInvalidPrefix(t *testing.T) {
	dir := t.TempDir()
	runInDir := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}
	runInDir("git", "init")
	runInDir("git", "config", "user.email", "test@test.com")
	runInDir("git", "config", "user.name", "Test")
	os.WriteFile(dir+"/README", []byte("test"), 0644)
	runInDir("git", "add", ".")
	runInDir("git", "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	var buf bytes.Buffer
	err := cmdInit([]string{"--prefix", "invalid prefix!"}, &buf)
	if err == nil {
		t.Error("expected error for invalid prefix")
	}
}

func init() {
	os.Setenv("GIT_AUTHOR_NAME", "Test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@test.com")
	os.Setenv("GIT_COMMITTER_NAME", "Test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@test.com")
}
