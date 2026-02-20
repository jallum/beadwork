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

	var got issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got.Title != "JSON show" {
		t.Errorf("title = %q", got.Title)
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
	env.Store.Close(iss.ID)
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
	env.Store.Close(iss.ID)
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
	env.Store.Close(iss.ID)
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

	iss, _ := env.Store.Create("Update me", issue.CreateOpts{Priority: 3})
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
	env.Store.Close(a.ID)
	env.Store.Close(b.ID)
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
	a := ParseArgs([]string{"--json", "--all", "positional"})
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
	a := ParseArgs([]string{"--status", "open", "--priority", "1"}, "--status", "--priority")
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
	a := ParseArgs([]string{"-p", "2", "-t", "bug", "-a", "alice"}, "--priority", "--type", "--assignee")
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
	a := ParseArgs([]string{"my", "title", "here", "--json"})
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
	a := ParseArgs([]string{})
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
	a := ParseArgs([]string{"--status"}, "--status")
	if a.Has("--status") {
		t.Error("expected Has(--status) to be false when no value follows")
	}
	if a.String("--status") != "" {
		t.Errorf("status = %q, want empty", a.String("--status"))
	}
}

func TestParseArgsIntErr(t *testing.T) {
	a := ParseArgs([]string{"--priority", "abc"}, "--priority")
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
	a2 := ParseArgs([]string{"--priority", "3"}, "--priority")
	n, set, err := a2.IntErr("--priority")
	if !set || err != nil || n != 3 {
		t.Errorf("expected (3, true, nil), got (%d, %v, %v)", n, set, err)
	}
}

func TestParseArgsMixedFlagsAndPositionals(t *testing.T) {
	a := ParseArgs([]string{"id-123", "--json", "--status", "open"}, "--status")
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
	if !strings.Contains(out, "Blocks: test-aaaa") {
		t.Errorf("missing Blocks in output: %q", out)
	}
	if !strings.Contains(out, "Blocked by: test-bbbb") {
		t.Errorf("missing BlockedBy in output: %q", out)
	}
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

func TestGetInitializedWithDefaultPriority(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Set a default priority config
	env.Repo.SetConfig("default.priority", "2")
	env.Repo.Commit("config default.priority=2")

	_, store, err := getInitialized()
	if err != nil {
		t.Fatalf("getInitialized: %v", err)
	}
	if store.DefaultPriority != 2 {
		t.Errorf("DefaultPriority = %d, want 2", store.DefaultPriority)
	}
}

func TestGetInitializedReturnsError(t *testing.T) {
	// getRepo / getInitialized should return errors, not crash
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Verify it works when initialized
	r, s, err := getInitialized()
	if err != nil {
		t.Fatalf("getInitialized: %v", err)
	}
	if r == nil || s == nil {
		t.Error("expected non-nil repo and store")
	}
}

func init() {
	os.Setenv("GIT_AUTHOR_NAME", "Test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@test.com")
	os.Setenv("GIT_COMMITTER_NAME", "Test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@test.com")
}
