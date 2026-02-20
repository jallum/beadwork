package issue_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCreateAndGet(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Fix auth bug", issue.CreateOpts{
		Priority:    1,
		Type:        "bug",
		Description: "Tokens expire too fast",
		Assignee:    "agent-1",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	env.CommitIntent("create " + iss.ID)

	// Verify the issue file exists
	if !env.IssueFileExists(iss.ID) {
		t.Fatal("issue file not created")
	}

	// Verify status marker
	if !env.MarkerExists(filepath.Join("status", "open", iss.ID)) {
		t.Fatal("status marker not created")
	}

	// Get it back
	got, err := env.Store.Get(iss.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Fix auth bug" {
		t.Errorf("title = %q, want %q", got.Title, "Fix auth bug")
	}
	if got.Priority != 1 {
		t.Errorf("priority = %d, want 1", got.Priority)
	}
	if got.Type != "bug" {
		t.Errorf("type = %q, want %q", got.Type, "bug")
	}
	if got.Status != "open" {
		t.Errorf("status = %q, want %q", got.Status, "open")
	}
	if got.Assignee != "agent-1" {
		t.Errorf("assignee = %q, want %q", got.Assignee, "agent-1")
	}
	if got.Description != "Tokens expire too fast" {
		t.Errorf("description = %q, want %q", got.Description, "Tokens expire too fast")
	}
}

func TestCreateDefaults(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Simple task", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if iss.Type != "task" {
		t.Errorf("default type = %q, want %q", iss.Type, "task")
	}
	if iss.Priority != 3 {
		t.Errorf("default priority = %d, want 3", iss.Priority)
	}
}

func TestCreateDefaultPriorityFromStore(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.DefaultPriority = 2

	iss, err := env.Store.Create("Custom default", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if iss.Priority != 2 {
		t.Errorf("priority = %d, want 2 (from store default)", iss.Priority)
	}

	// Explicit priority should still override
	iss2, err := env.Store.Create("Explicit priority", issue.CreateOpts{Priority: 1})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if iss2.Priority != 1 {
		t.Errorf("priority = %d, want 1 (explicit override)", iss2.Priority)
	}
}

func TestIDPrefixMatch(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Test issue", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	env.CommitIntent("create " + iss.ID)

	// Should resolve by suffix (last 4 chars)
	suffix := iss.ID[len("test-"):]
	got, err := env.Store.Get(suffix)
	if err != nil {
		t.Fatalf("Get by suffix %q: %v", suffix, err)
	}
	if got.ID != iss.ID {
		t.Errorf("resolved to %q, want %q", got.ID, iss.ID)
	}
}

func TestListFilters(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Bug one", issue.CreateOpts{Priority: 1, Type: "bug"})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Task two", issue.CreateOpts{Priority: 2, Type: "task", Assignee: "agent-1"})
	env.CommitIntent("create " + b.ID)
	c, _ := env.Store.Create("Bug three", issue.CreateOpts{Priority: 1, Type: "bug"})
	env.CommitIntent("create " + c.ID)

	// Filter by type
	bugs, err := env.Store.List(issue.Filter{Type: "bug"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(bugs) != 2 {
		t.Errorf("got %d bugs, want 2", len(bugs))
	}

	// Filter by priority
	p1, err := env.Store.List(issue.Filter{Priority: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(p1) != 2 {
		t.Errorf("got %d p1 issues, want 2", len(p1))
	}

	// Filter by assignee
	assigned, err := env.Store.List(issue.Filter{Assignee: "agent-1"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(assigned) != 1 {
		t.Errorf("got %d assigned, want 1", len(assigned))
	}

	// List sorted by priority then created
	all, _ := env.Store.List(issue.Filter{})
	if len(all) != 3 {
		t.Fatalf("got %d total, want 3", len(all))
	}
	if all[0].Priority != 1 {
		t.Error("first issue should be p1")
	}
}

func TestUpdateFields(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Original title", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	newTitle := "Updated title"
	newPriority := 1
	newAssignee := "agent-2"
	updated, err := env.Store.Update(iss.ID, issue.UpdateOpts{
		Title:    &newTitle,
		Priority: &newPriority,
		Assignee: &newAssignee,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if updated.Title != "Updated title" {
		t.Errorf("title = %q", updated.Title)
	}
	if updated.Priority != 1 {
		t.Errorf("priority = %d", updated.Priority)
	}
	if updated.Assignee != "agent-2" {
		t.Errorf("assignee = %q", updated.Assignee)
	}
}

func TestStatusTransitions(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Lifecycle test", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// open -> in_progress
	status := "in_progress"
	iss, _ = env.Store.Update(iss.ID, issue.UpdateOpts{Status: &status})
	env.CommitIntent("update " + iss.ID)

	if !env.MarkerExists(filepath.Join("status", "in_progress", iss.ID)) {
		t.Error("in_progress marker missing")
	}
	if env.MarkerExists(filepath.Join("status", "open", iss.ID)) {
		t.Error("open marker should be gone")
	}

	// in_progress -> closed
	iss, err := env.Store.Close(iss.ID)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	env.CommitIntent("close " + iss.ID)

	if !env.MarkerExists(filepath.Join("status", "closed", iss.ID)) {
		t.Error("closed marker missing")
	}
	if env.MarkerExists(filepath.Join("status", "in_progress", iss.ID)) {
		t.Error("in_progress marker should be gone")
	}

	// closed -> open (reopen)
	iss, err = env.Store.Reopen(iss.ID)
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	env.CommitIntent("reopen " + iss.ID)

	if !env.MarkerExists(filepath.Join("status", "open", iss.ID)) {
		t.Error("open marker missing after reopen")
	}
	if env.MarkerExists(filepath.Join("status", "closed", iss.ID)) {
		t.Error("closed marker should be gone after reopen")
	}
	if iss.Status != "open" {
		t.Errorf("status = %q, want open", iss.Status)
	}
}

func TestCloseAlreadyClosed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test", issue.CreateOpts{})
	env.Store.Close(iss.ID)

	_, err := env.Store.Close(iss.ID)
	if err == nil {
		t.Error("expected error closing already-closed issue")
	}
}

func TestReopenNotClosed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test", issue.CreateOpts{})

	_, err := env.Store.Reopen(iss.ID)
	if err == nil {
		t.Error("expected error reopening non-closed issue")
	}
}

func TestLink(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.CommitIntent("create " + b.ID)

	if err := env.Store.Link(a.ID, b.ID); err != nil {
		t.Fatalf("Link: %v", err)
	}
	env.CommitIntent("link " + a.ID + " blocks " + b.ID)

	// Check marker file
	if !env.MarkerExists(filepath.Join("blocks", a.ID, b.ID)) {
		t.Error("blocks marker missing")
	}

	// Check JSON updated on both sides
	aGot, _ := env.Store.Get(a.ID)
	if len(aGot.Blocks) != 1 || aGot.Blocks[0] != b.ID {
		t.Errorf("blocker.Blocks = %v, want [%s]", aGot.Blocks, b.ID)
	}
	bGot, _ := env.Store.Get(b.ID)
	if len(bGot.BlockedBy) != 1 || bGot.BlockedBy[0] != a.ID {
		t.Errorf("blocked.BlockedBy = %v, want [%s]", bGot.BlockedBy, a.ID)
	}
}

func TestUnlink(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("link " + a.ID + " blocks " + b.ID)

	if err := env.Store.Unlink(a.ID, b.ID); err != nil {
		t.Fatalf("Unlink: %v", err)
	}
	env.CommitIntent("unlink " + a.ID + " blocks " + b.ID)

	// Marker should be gone
	if env.MarkerExists(filepath.Join("blocks", a.ID, b.ID)) {
		t.Error("blocks marker should be gone")
	}

	// JSON updated
	aGot, _ := env.Store.Get(a.ID)
	if len(aGot.Blocks) != 0 {
		t.Errorf("blocker.Blocks = %v, want empty", aGot.Blocks)
	}
	bGot, _ := env.Store.Get(b.ID)
	if len(bGot.BlockedBy) != 0 {
		t.Errorf("blocked.BlockedBy = %v, want empty", bGot.BlockedBy)
	}
}

func TestLinkSelfBlocking(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Self", issue.CreateOpts{})
	err := env.Store.Link(a.ID, a.ID)
	if err == nil {
		t.Error("expected error for self-blocking")
	}
}

func TestReady(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Blocked task", issue.CreateOpts{})
	env.CommitIntent("create " + b.ID)
	c, _ := env.Store.Create("Free task", issue.CreateOpts{})
	env.CommitIntent("create " + c.ID)

	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("link " + a.ID + " blocks " + b.ID)

	// B is blocked, A and C are ready
	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[a.ID] {
		t.Error("blocker should be ready")
	}
	if ids[b.ID] {
		t.Error("blocked task should NOT be ready")
	}
	if !ids[c.ID] {
		t.Error("free task should be ready")
	}

	// Close the blocker -> B becomes ready
	env.Store.Close(a.ID)
	env.CommitIntent("close " + a.ID)

	ready, _ = env.Store.Ready()
	ids = make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[b.ID] {
		t.Error("blocked task should be ready after blocker closed")
	}
}

func TestLabel(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Labeled issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// Add labels
	iss, err := env.Store.Label(iss.ID, []string{"bug", "frontend"}, nil)
	if err != nil {
		t.Fatalf("Label add: %v", err)
	}
	env.CommitIntent("label " + iss.ID + " +bug +frontend")

	// Check markers
	if !env.MarkerExists(filepath.Join("labels", "bug", iss.ID)) {
		t.Error("bug label marker missing")
	}
	if !env.MarkerExists(filepath.Join("labels", "frontend", iss.ID)) {
		t.Error("frontend label marker missing")
	}

	// Check JSON
	got, _ := env.Store.Get(iss.ID)
	if len(got.Labels) != 2 {
		t.Fatalf("labels = %v, want 2", got.Labels)
	}

	// Filter by label
	filtered, _ := env.Store.List(issue.Filter{Label: "bug"})
	if len(filtered) != 1 {
		t.Errorf("filtered by bug: got %d, want 1", len(filtered))
	}

	// Remove label
	iss, err = env.Store.Label(iss.ID, nil, []string{"frontend"})
	if err != nil {
		t.Fatalf("Label remove: %v", err)
	}
	env.CommitIntent("label " + iss.ID + " -frontend")

	if env.MarkerExists(filepath.Join("labels", "frontend", iss.ID)) {
		t.Error("frontend label marker should be gone")
	}
	got, _ = env.Store.Get(iss.ID)
	if len(got.Labels) != 1 || got.Labels[0] != "bug" {
		t.Errorf("labels = %v, want [bug]", got.Labels)
	}
}

func TestLabelDirectoryCleanup(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test", issue.CreateOpts{})
	env.Store.Label(iss.ID, []string{"temp"}, nil)
	env.Store.Label(iss.ID, nil, []string{"temp"})

	// The temp/ directory should be cleaned up (no entry in TreeFS)
	_, err := env.Repo.TreeFS().Stat("labels/temp")
	if err == nil {
		t.Error("empty label directory should be removed")
	}
}

func TestGraph(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Root", issue.CreateOpts{})
	b, _ := env.Store.Create("Middle", issue.CreateOpts{})
	c, _ := env.Store.Create("Leaf", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Link(b.ID, c.ID)
	env.CommitIntent("setup graph")

	// Full graph
	nodes, err := env.Store.Graph("")
	if err != nil {
		t.Fatalf("Graph: %v", err)
	}
	if len(nodes) != 3 {
		t.Errorf("got %d nodes, want 3", len(nodes))
	}

	// Rooted graph
	nodes, err = env.Store.Graph(a.ID)
	if err != nil {
		t.Fatalf("Graph rooted: %v", err)
	}
	if len(nodes) != 3 {
		t.Errorf("rooted got %d nodes, want 3 (a->b->c)", len(nodes))
	}

	// Rooted at middle
	nodes, err = env.Store.Graph(b.ID)
	if err != nil {
		t.Fatalf("Graph rooted at middle: %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("middle-rooted got %d nodes, want 2 (b->c)", len(nodes))
	}
}

func TestListByStatus(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open one", issue.CreateOpts{})
	env.Store.Create("Open two", issue.CreateOpts{})
	env.Store.Close(a.ID)

	open, _ := env.Store.List(issue.Filter{Status: "open"})
	if len(open) != 1 {
		t.Errorf("open = %d, want 1", len(open))
	}

	closed, _ := env.Store.List(issue.Filter{Status: "closed"})
	if len(closed) != 1 {
		t.Errorf("closed = %d, want 1", len(closed))
	}
}

func TestStatusNames(t *testing.T) {
	names := issue.StatusNames()
	if len(names) != len(issue.Statuses) {
		t.Fatalf("StatusNames() returned %d, want %d", len(names), len(issue.Statuses))
	}
	for i, s := range issue.Statuses {
		if names[i] != s.Name {
			t.Errorf("StatusNames()[%d] = %q, want %q", i, names[i], s.Name)
		}
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"open", "○"},
		{"in_progress", "◐"},
		{"closed", "✓"},
		{"unknown", "?"},
	}
	for _, tt := range tests {
		got := issue.StatusIcon(tt.status)
		if got != tt.want {
			t.Errorf("StatusIcon(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}

	// Deferred status should have snowflake icon
	got := issue.StatusIcon("deferred")
	if got != "❄" {
		t.Errorf("StatusIcon(deferred) = %q, want ❄", got)
	}
}

func TestPriorityDot(t *testing.T) {
	// Each priority should produce a colored ● with reset
	for p := 0; p <= 5; p++ {
		dot := issue.PriorityDot(p)
		if dot == "" {
			t.Errorf("PriorityDot(%d) returned empty string", p)
		}
		color, ok := issue.PriorityColors[p]
		if !ok {
			t.Errorf("PriorityColors missing key %d", p)
			continue
		}
		want := color + "●" + issue.ColorReset
		if dot != want {
			t.Errorf("PriorityDot(%d) = %q, want %q", p, dot, want)
		}
	}

	// Unknown priority should still return a dot
	dot := issue.PriorityDot(99)
	if dot == "" {
		t.Error("PriorityDot(99) returned empty string")
	}
	if dot != "●"+issue.ColorReset {
		t.Errorf("PriorityDot(99) = %q, want uncolored dot", dot)
	}
}

func TestListJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create issues with different properties
	bug, _ := env.Store.Create("Fix crash", issue.CreateOpts{Priority: 1, Type: "bug", Assignee: "alice"})
	env.CommitIntent("create " + bug.ID)
	task, _ := env.Store.Create("Add search", issue.CreateOpts{Priority: 3, Type: "task"})
	env.CommitIntent("create " + task.ID)

	// Close the bug
	env.Store.Close(bug.ID)
	env.CommitIntent("close " + bug.ID)

	issues, err := env.Store.List(issue.Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	data, err := json.MarshalIndent(issues, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}

	// Unmarshal back to verify round-trip and field presence
	var parsed []issue.Issue
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(parsed) != 2 {
		t.Fatalf("got %d issues, want 2", len(parsed))
	}

	// Issues are sorted by priority, so P1 bug comes first
	assertJSONFields(t, parsed[0], "closed", 1, "bug", "Fix crash", "alice")
	assertJSONFields(t, parsed[1], "open", 3, "task", "Add search", "")
}

func TestReadyJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Unblocked task", issue.CreateOpts{Priority: 2, Type: "task"})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Blocked task", issue.CreateOpts{Priority: 3, Type: "task"})
	env.CommitIntent("create " + b.ID)

	// a blocks b
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("link " + a.ID + " blocks " + b.ID)

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}

	data, err := json.MarshalIndent(ready, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}

	var parsed []issue.Issue
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Only the unblocked task should be ready
	if len(parsed) != 1 {
		t.Fatalf("got %d ready issues, want 1", len(parsed))
	}
	assertJSONFields(t, parsed[0], "open", 2, "task", "Unblocked task", "")

	// Verify blocks/blocked_by are present in JSON
	if len(parsed[0].Blocks) != 1 || parsed[0].Blocks[0] != b.ID {
		t.Errorf("blocks = %v, want [%s]", parsed[0].Blocks, b.ID)
	}
}

func TestListJSONFilterByStatus(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open issue", issue.CreateOpts{Priority: 2})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Closed issue", issue.CreateOpts{Priority: 2})
	env.CommitIntent("create " + b.ID)
	env.Store.Close(b.ID)
	env.CommitIntent("close " + b.ID)

	issues, err := env.Store.List(issue.Filter{Status: "open"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	data, _ := json.MarshalIndent(issues, "", "  ")
	var parsed []issue.Issue
	json.Unmarshal(data, &parsed)

	if len(parsed) != 1 {
		t.Fatalf("got %d issues, want 1", len(parsed))
	}
	if parsed[0].Status != "open" {
		t.Errorf("status = %q, want open", parsed[0].Status)
	}
}

func assertJSONFields(t *testing.T, iss issue.Issue, status string, priority int, typ, title, assignee string) {
	t.Helper()
	if iss.Status != status {
		t.Errorf("status = %q, want %q", iss.Status, status)
	}
	if iss.Priority != priority {
		t.Errorf("priority = %d, want %d", iss.Priority, priority)
	}
	if iss.Type != typ {
		t.Errorf("type = %q, want %q", iss.Type, typ)
	}
	if iss.Title != title {
		t.Errorf("title = %q, want %q", iss.Title, title)
	}
	if iss.Assignee != assignee {
		t.Errorf("assignee = %q, want %q", iss.Assignee, assignee)
	}
	if iss.ID == "" {
		t.Error("id is empty")
	}
	if iss.Created == "" {
		t.Error("created is empty")
	}
}

func TestImportDirect(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss := &issue.Issue{
		ID:          "ext-001",
		Title:       "Imported directly",
		Description: "From external source",
		Status:      "open",
		Priority:    2,
		Type:        "task",
		Assignee:    "someone",
		Created:     "2026-01-01T00:00:00Z",
		Labels:      []string{},
		Blocks:      []string{},
		BlockedBy:   []string{},
	}

	if err := env.Store.Import(iss); err != nil {
		t.Fatalf("Import: %v", err)
	}
	env.CommitIntent("import ext-001")

	got, err := env.Store.Get("ext-001")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Imported directly" {
		t.Errorf("title = %q", got.Title)
	}
	if got.Status != "open" {
		t.Errorf("status = %q", got.Status)
	}
	if got.Assignee != "someone" {
		t.Errorf("assignee = %q", got.Assignee)
	}
}

func TestImportClosedStatus(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss := &issue.Issue{
		ID:        "ext-closed",
		Title:     "Closed import",
		Status:    "closed",
		Priority:  1,
		Type:      "bug",
		Created:   "2026-01-01T00:00:00Z",
		Labels:    []string{},
		Blocks:    []string{},
		BlockedBy: []string{},
	}

	if err := env.Store.Import(iss); err != nil {
		t.Fatalf("Import: %v", err)
	}
	env.CommitIntent("import closed")

	got, err := env.Store.Get("ext-closed")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != "closed" {
		t.Errorf("status = %q, want closed", got.Status)
	}
	if !env.MarkerExists(filepath.Join("status", "closed", "ext-closed")) {
		t.Error("closed status marker missing")
	}
}

func TestAmbiguousID(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create issues with similar IDs that share a prefix
	for _, id := range []string{"test-ab01", "test-ab02"} {
		env.Store.Import(&issue.Issue{
			ID:        id,
			Title:     "Issue " + id,
			Status:    "open",
			Priority:  3,
			Type:      "task",
			Created:   "2026-01-01T00:00:00Z",
			Labels:    []string{},
			Blocks:    []string{},
			BlockedBy: []string{},
		})
	}
	env.CommitIntent("import two similar issues")

	// Searching for "test-ab0" should be ambiguous (matches both via prefix)
	_, err := env.Store.Get("test-ab0")
	if err == nil {
		t.Error("expected ambiguous ID error")
	}
	if err != nil && !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' in error, got: %v", err)
	}

	// Exact match should work
	got, err := env.Store.Get("test-ab01")
	if err != nil {
		t.Fatalf("Get exact: %v", err)
	}
	if got.ID != "test-ab01" {
		t.Errorf("id = %q", got.ID)
	}
}

func TestLinkIdempotent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)

	// Link again — should be a noop, not create duplicate entries
	if err := env.Store.Link(a.ID, b.ID); err != nil {
		t.Fatalf("second Link: %v", err)
	}

	aGot, _ := env.Store.Get(a.ID)
	if len(aGot.Blocks) != 1 {
		t.Errorf("blocks = %v, want exactly 1 entry", aGot.Blocks)
	}
	bGot, _ := env.Store.Get(b.ID)
	if len(bGot.BlockedBy) != 1 {
		t.Errorf("blockedBy = %v, want exactly 1 entry", bGot.BlockedBy)
	}
}

func TestUnlinkIdempotent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Unlink(a.ID, b.ID)

	// Unlink again — should be a noop
	if err := env.Store.Unlink(a.ID, b.ID); err != nil {
		t.Fatalf("second Unlink: %v", err)
	}

	aGot, _ := env.Store.Get(a.ID)
	if len(aGot.Blocks) != 0 {
		t.Errorf("blocks = %v, want empty", aGot.Blocks)
	}
}

func TestLabelIdempotent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Label test", issue.CreateOpts{})
	env.Store.Label(iss.ID, []string{"bug"}, nil)

	// Add same label again — should not duplicate
	iss2, err := env.Store.Label(iss.ID, []string{"bug"}, nil)
	if err != nil {
		t.Fatalf("second Label: %v", err)
	}
	if len(iss2.Labels) != 1 {
		t.Errorf("labels = %v, want exactly 1 entry", iss2.Labels)
	}
}

func TestRemoveNonexistentLabel(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("No labels", issue.CreateOpts{})

	// Remove a label that doesn't exist — should be fine
	iss2, err := env.Store.Label(iss.ID, nil, []string{"nonexistent"})
	if err != nil {
		t.Fatalf("Label remove nonexistent: %v", err)
	}
	if len(iss2.Labels) != 0 {
		t.Errorf("labels = %v, want empty", iss2.Labels)
	}
}

func TestGetNonExistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.Get("test-zzzz")
	if err == nil {
		t.Error("expected error for non-existent issue")
	}
}

func TestGraphEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// No issues at all — graph should return empty
	nodes, err := env.Store.Graph("")
	if err != nil {
		t.Fatalf("Graph: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("got %d nodes, want 0", len(nodes))
	}
}

func TestGraphRootedSubset(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Root A", issue.CreateOpts{})
	b, _ := env.Store.Create("Child B", issue.CreateOpts{})
	c, _ := env.Store.Create("Independent C", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	// Rooted at A should only return A and B, not C
	nodes, err := env.Store.Graph(a.ID)
	if err != nil {
		t.Fatalf("Graph: %v", err)
	}
	ids := make(map[string]bool)
	for _, n := range nodes {
		ids[n.ID] = true
	}
	if !ids[a.ID] || !ids[b.ID] {
		t.Error("expected A and B in rooted graph")
	}
	if ids[c.ID] {
		t.Error("C should not be in graph rooted at A")
	}
	_ = c
}

func TestLinkNonExistentBlocker(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})

	err := env.Store.Link("test-zzzz", b.ID)
	if err == nil {
		t.Error("expected error for non-existent blocker")
	}
}

func TestLinkNonExistentBlocked(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})

	err := env.Store.Link(a.ID, "test-zzzz")
	if err == nil {
		t.Error("expected error for non-existent blocked")
	}
}

func TestReadCorruptJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create a valid issue, then corrupt it
	iss, _ := env.Store.Create("Valid", issue.CreateOpts{})
	env.Repo.TreeFS().WriteFile("issues/"+iss.ID+".json", []byte("{invalid json"))

	_, err := env.Store.Get(iss.ID)
	if err == nil {
		t.Error("expected error for corrupt JSON")
	}
	if !strings.Contains(err.Error(), "corrupt") {
		t.Errorf("error = %q, want 'corrupt'", err.Error())
	}
}

func TestGraphNonexistentRootReturnsError(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Exists", issue.CreateOpts{})
	env.CommitIntent("create")

	_, err := env.Store.Graph("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent root ID")
	}
}

func TestGraphNoRelationshipsShowsOpen(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create issues without any links
	env.Store.Create("Standalone A", issue.CreateOpts{})
	env.Store.Create("Standalone B", issue.CreateOpts{})
	iss3, _ := env.Store.Create("Closed one", issue.CreateOpts{})
	env.Store.Close(iss3.ID)
	env.CommitIntent("setup")

	// Graph with no root and no edges: should show all open issues
	nodes, err := env.Store.Graph("")
	if err != nil {
		t.Fatalf("Graph: %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("got %d nodes, want 2 (only open issues)", len(nodes))
	}
	for _, n := range nodes {
		if n.Status == "closed" {
			t.Error("closed issue should not appear in graph without edges")
		}
	}
}

func TestUpdateStatusChange(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Status test", issue.CreateOpts{})

	// Update status via UpdateOpts
	newStatus := "in_progress"
	updated, err := env.Store.Update(iss.ID, issue.UpdateOpts{Status: &newStatus})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Status != "in_progress" {
		t.Errorf("status = %q, want in_progress", updated.Status)
	}

	// Update to same status (noop)
	sameStatus := "in_progress"
	updated2, err := env.Store.Update(iss.ID, issue.UpdateOpts{Status: &sameStatus})
	if err != nil {
		t.Fatalf("Update same status: %v", err)
	}
	if updated2.Status != "in_progress" {
		t.Errorf("status = %q", updated2.Status)
	}
}

func TestUnlinkNonExistentBlocker(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	err := env.Store.Unlink("test-zzzz", b.ID)
	if err == nil {
		t.Error("expected error for non-existent blocker")
	}
}

func TestUnlinkNonExistentBlocked(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	err := env.Store.Unlink(a.ID, "test-zzzz")
	if err == nil {
		t.Error("expected error for non-existent blocked")
	}
}

func TestLabelNonExistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.Label("test-zzzz", []string{"bug"}, nil)
	if err == nil {
		t.Error("expected error for non-existent issue")
	}
}

func TestCloseNonExistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.Close("test-zzzz")
	if err == nil {
		t.Error("expected error for non-existent issue")
	}
}

func TestReopenNonExistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.Reopen("test-zzzz")
	if err == nil {
		t.Error("expected error for non-existent issue")
	}
}

func TestUpdateNonExistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	title := "x"
	_, err := env.Store.Update("test-zzzz", issue.UpdateOpts{Title: &title})
	if err == nil {
		t.Error("expected error for non-existent issue")
	}
}

func TestMultipleIDsNeverCollide(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	ids := make(map[string]bool)
	for i := 0; i < 50; i++ {
		iss, err := env.Store.Create("Issue", issue.CreateOpts{})
		if err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
		if ids[iss.ID] {
			t.Fatalf("duplicate ID: %s", iss.ID)
		}
		ids[iss.ID] = true
	}
}

func TestBlockedSingle(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("link")

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	if len(blocked) != 1 {
		t.Fatalf("got %d blocked, want 1", len(blocked))
	}
	if blocked[0].ID != b.ID {
		t.Errorf("blocked ID = %q, want %q", blocked[0].ID, b.ID)
	}
	if len(blocked[0].OpenBlockers) != 1 || blocked[0].OpenBlockers[0] != a.ID {
		t.Errorf("open blockers = %v, want [%s]", blocked[0].OpenBlockers, a.ID)
	}
}

func TestBlockedMultipleBlockers(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker A", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocker B", issue.CreateOpts{})
	c, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, c.ID)
	env.Store.Link(b.ID, c.ID)
	env.Store.Close(b.ID) // close one blocker
	env.CommitIntent("setup")

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	if len(blocked) != 1 {
		t.Fatalf("got %d blocked, want 1", len(blocked))
	}
	// Only the open blocker should appear
	if len(blocked[0].OpenBlockers) != 1 || blocked[0].OpenBlockers[0] != a.ID {
		t.Errorf("open blockers = %v, want [%s]", blocked[0].OpenBlockers, a.ID)
	}
}

func TestBlockedResolves(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("link")

	// Close the blocker
	env.Store.Close(a.ID)
	env.CommitIntent("close")

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	if len(blocked) != 0 {
		t.Errorf("got %d blocked, want 0 after resolving blocker", len(blocked))
	}
}

func TestBlockedNoBlockers(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("No deps", issue.CreateOpts{})
	env.CommitIntent("create")

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	if len(blocked) != 0 {
		t.Errorf("got %d blocked, want 0", len(blocked))
	}
}

func TestBlockedClosedIssueExcluded(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked and closed", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(b.ID) // close the blocked issue itself
	env.CommitIntent("setup")

	blocked, err := env.Store.Blocked()
	if err != nil {
		t.Fatalf("Blocked: %v", err)
	}
	// Closed issues shouldn't appear even if they have open blockers
	for _, bi := range blocked {
		if bi.ID == b.ID {
			t.Error("closed issue should not appear in blocked list")
		}
	}
}

func TestReadyExcludesInProgress(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open task", issue.CreateOpts{})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("In-progress task", issue.CreateOpts{})
	env.CommitIntent("create " + b.ID)

	// Move B to in_progress
	status := "in_progress"
	env.Store.Update(b.ID, issue.UpdateOpts{Status: &status})
	env.CommitIntent("update " + b.ID)

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}

	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[a.ID] {
		t.Error("open task should be ready")
	}
	if ids[b.ID] {
		t.Error("in_progress task should NOT be ready")
	}
}

func TestReadyExcludesDeferred(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open task", issue.CreateOpts{})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Deferred task", issue.CreateOpts{})
	env.CommitIntent("create " + b.ID)

	// Move B to deferred
	status := "deferred"
	deferDate := "2027-06-01"
	env.Store.Update(b.ID, issue.UpdateOpts{Status: &status, DeferUntil: &deferDate})
	env.CommitIntent("defer " + b.ID)

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}

	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[a.ID] {
		t.Error("open task should be ready")
	}
	if ids[b.ID] {
		t.Error("deferred task should NOT be ready")
	}
}

func TestDeferUntilPersistence(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Deferred at create", issue.CreateOpts{
		DeferUntil: "2027-03-15",
	})
	env.CommitIntent("create " + iss.ID)

	if iss.Status != "deferred" {
		t.Errorf("status = %q, want deferred", iss.Status)
	}
	if iss.DeferUntil != "2027-03-15" {
		t.Errorf("defer_until = %q, want 2027-03-15", iss.DeferUntil)
	}

	// Read back to verify persistence
	got, err := env.Store.Get(iss.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.DeferUntil != "2027-03-15" {
		t.Errorf("persisted defer_until = %q, want 2027-03-15", got.DeferUntil)
	}
	if got.Status != "deferred" {
		t.Errorf("persisted status = %q, want deferred", got.Status)
	}
}

func TestUpdateDeferUntil(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Task", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// Defer the issue
	status := "deferred"
	deferDate := "2027-06-01"
	updated, err := env.Store.Update(iss.ID, issue.UpdateOpts{
		Status:     &status,
		DeferUntil: &deferDate,
	})
	if err != nil {
		t.Fatalf("Update defer: %v", err)
	}
	if updated.Status != "deferred" {
		t.Errorf("status = %q, want deferred", updated.Status)
	}
	if updated.DeferUntil != "2027-06-01" {
		t.Errorf("defer_until = %q, want 2027-06-01", updated.DeferUntil)
	}

	// Undefer: clear DeferUntil and restore to open
	openStatus := "open"
	emptyDefer := ""
	undeferred, err := env.Store.Update(iss.ID, issue.UpdateOpts{
		Status:     &openStatus,
		DeferUntil: &emptyDefer,
	})
	if err != nil {
		t.Fatalf("Update undefer: %v", err)
	}
	if undeferred.Status != "open" {
		t.Errorf("status = %q, want open", undeferred.Status)
	}
	if undeferred.DeferUntil != "" {
		t.Errorf("defer_until = %q, want empty", undeferred.DeferUntil)
	}
}

func TestListByDeferredStatus(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open task", issue.CreateOpts{})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Deferred task", issue.CreateOpts{DeferUntil: "2027-01-01"})
	env.CommitIntent("create " + b.ID)

	// List all — should include both
	all, _ := env.Store.List(issue.Filter{})
	if len(all) != 2 {
		t.Errorf("all = %d, want 2", len(all))
	}

	// Filter by deferred status
	deferred, _ := env.Store.List(issue.Filter{Status: "deferred"})
	if len(deferred) != 1 {
		t.Errorf("deferred = %d, want 1", len(deferred))
	}
	if len(deferred) > 0 && deferred[0].ID != b.ID {
		t.Errorf("deferred ID = %q, want %q", deferred[0].ID, b.ID)
	}

	// Filter by open — should not include deferred
	open, _ := env.Store.List(issue.Filter{Status: "open"})
	if len(open) != 1 {
		t.Errorf("open = %d, want 1", len(open))
	}
}
