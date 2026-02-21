package issue_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func intPtr(n int) *int { return &n }

func TestCreateAndGet(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Fix auth bug", issue.CreateOpts{
		Priority:    intPtr(1),
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
	if iss.Priority != 2 {
		t.Errorf("default priority = %d, want 2", iss.Priority)
	}
}

func TestCreateDefaultPriorityFromStore(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.DefaultPriority = intPtr(3)

	iss, err := env.Store.Create("Custom default", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if iss.Priority != 3 {
		t.Errorf("priority = %d, want 3 (from store default)", iss.Priority)
	}

	// Explicit priority should still override
	iss2, err := env.Store.Create("Explicit priority", issue.CreateOpts{Priority: intPtr(1)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if iss2.Priority != 1 {
		t.Errorf("priority = %d, want 1 (explicit override)", iss2.Priority)
	}

	// Explicit P0 should work
	iss3, err := env.Store.Create("P0 priority", issue.CreateOpts{Priority: intPtr(0)})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if iss3.Priority != 0 {
		t.Errorf("priority = %d, want 0 (explicit P0)", iss3.Priority)
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

	// Should resolve by suffix
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

	a, _ := env.Store.Create("Bug one", issue.CreateOpts{Priority: intPtr(1), Type: "bug"})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Task two", issue.CreateOpts{Priority: intPtr(2), Type: "task", Assignee: "agent-1"})
	env.CommitIntent("create " + b.ID)
	c, _ := env.Store.Create("Bug three", issue.CreateOpts{Priority: intPtr(1), Type: "bug"})
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
	p1, err := env.Store.List(issue.Filter{Priority: intPtr(1)})
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
	iss, err := env.Store.Close(iss.ID, "")
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
	env.Store.Close(iss.ID, "")

	_, err := env.Store.Close(iss.ID, "")
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
	env.Store.Close(a.ID, "")
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

func TestListByStatus(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open one", issue.CreateOpts{})
	env.Store.Create("Open two", issue.CreateOpts{})
	env.Store.Close(a.ID, "")

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
	for p := 0; p <= 4; p++ {
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
	bug, _ := env.Store.Create("Fix crash", issue.CreateOpts{Priority: intPtr(1), Type: "bug", Assignee: "alice"})
	env.CommitIntent("create " + bug.ID)
	task, _ := env.Store.Create("Add search", issue.CreateOpts{Priority: intPtr(3), Type: "task"})
	env.CommitIntent("create " + task.ID)

	// Close the bug
	env.Store.Close(bug.ID, "")
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

	a, _ := env.Store.Create("Unblocked task", issue.CreateOpts{Priority: intPtr(2), Type: "task"})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Blocked task", issue.CreateOpts{Priority: intPtr(3), Type: "task"})
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

	a, _ := env.Store.Create("Open issue", issue.CreateOpts{Priority: intPtr(2)})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Closed issue", issue.CreateOpts{Priority: intPtr(2)})
	env.CommitIntent("create " + b.ID)
	env.Store.Close(b.ID, "")
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

	_, err := env.Store.Close("test-zzzz", "")
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

func TestIDIsBase36(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	for i := 0; i < 20; i++ {
		iss, err := env.Store.Create("Base36 test", issue.CreateOpts{})
		if err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
		suffix := iss.ID[len("test-"):]
		for _, c := range suffix {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z')) {
				t.Errorf("ID %q contains non-base36 char %q", iss.ID, string(c))
			}
		}
	}
}

func TestIDStartsAt3Chars(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Short ID", issue.CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	suffix := iss.ID[len("test-"):]
	if len(suffix) != 3 {
		t.Errorf("suffix %q length = %d, want 3", suffix, len(suffix))
	}
}

func TestIDDeterministicRand(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Inject a deterministic reader that always returns the same bytes.
	// Two creates with the same rand source should collide on the first
	// attempt at length 3, then succeed at length 3 on the second attempt
	// with different bytes.
	calls := 0
	env.Store.RandReader = readerFunc(func(b []byte) (int, error) {
		calls++
		for i := range b {
			b[i] = byte(calls) // deterministic, changes each call
		}
		return len(b), nil
	})

	iss1, err := env.Store.Create("First", issue.CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	iss2, err := env.Store.Create("Second", issue.CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if iss1.ID == iss2.ID {
		t.Errorf("expected different IDs, both got %q", iss1.ID)
	}
}

func TestIDAdaptiveLength(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Inject a reader that always returns zeros. Every attempt at a given
	// length produces the same ID, forcing a collision and length bump.
	env.Store.RandReader = readerFunc(func(b []byte) (int, error) {
		for i := range b {
			b[i] = 0
		}
		return len(b), nil
	})
	env.Store.IDRetries = 2

	// First create succeeds at length 3 (no collision yet).
	iss1, err := env.Store.Create("A", issue.CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	suffix1 := iss1.ID[len("test-"):]
	if len(suffix1) != 3 {
		t.Errorf("first suffix %q length = %d, want 3", suffix1, len(suffix1))
	}

	// Second create: all length-3 attempts collide (same bytes), bumps to 4.
	iss2, err := env.Store.Create("B", issue.CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	suffix2 := iss2.ID[len("test-"):]
	if len(suffix2) != 4 {
		t.Errorf("second suffix %q length = %d, want 4", suffix2, len(suffix2))
	}
}

// readerFunc adapts a function to io.Reader.
type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(b []byte) (int, error) { return f(b) }

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
	env.Store.Close(b.ID, "") // close one blocker
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
	env.Store.Close(a.ID, "")
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
	env.Store.Close(b.ID, "") // close the blocked issue itself
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

// --- updated_at, closed_at, close_reason ---

func TestUpdatedAtOnCreate(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("New issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	if iss.UpdatedAt == "" {
		t.Error("updated_at should be set on create")
	}
	if iss.UpdatedAt != iss.Created {
		t.Errorf("updated_at = %q, want same as created %q", iss.UpdatedAt, iss.Created)
	}
}

func TestUpdatedAtOnUpdate(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Update me", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)
	originalUpdated := iss.UpdatedAt

	title := "Updated title"
	updated, _ := env.Store.Update(iss.ID, issue.UpdateOpts{Title: &title})
	if updated.UpdatedAt == "" {
		t.Error("updated_at should be set after update")
	}
	if updated.UpdatedAt == originalUpdated {
		// updated_at should change (may be equal if test runs within same second)
		// At minimum it should be non-empty
	}

	got, _ := env.Store.Get(iss.ID)
	if got.UpdatedAt != updated.UpdatedAt {
		t.Errorf("persisted updated_at = %q, want %q", got.UpdatedAt, updated.UpdatedAt)
	}
}

func TestCloseReason(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Close me", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	closed, err := env.Store.Close(iss.ID, "duplicate")
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if closed.CloseReason != "duplicate" {
		t.Errorf("close_reason = %q, want duplicate", closed.CloseReason)
	}
	if closed.ClosedAt == "" {
		t.Error("closed_at should be set")
	}

	got, _ := env.Store.Get(iss.ID)
	if got.CloseReason != "duplicate" {
		t.Errorf("persisted close_reason = %q, want duplicate", got.CloseReason)
	}
	if got.ClosedAt == "" {
		t.Error("persisted closed_at should be set")
	}
}

func TestCloseWithoutReason(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Close no reason", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	closed, err := env.Store.Close(iss.ID, "")
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if closed.CloseReason != "" {
		t.Errorf("close_reason = %q, want empty", closed.CloseReason)
	}
	if closed.ClosedAt == "" {
		t.Error("closed_at should be set even without reason")
	}
}

func TestReopenClearsCloseFields(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Reopen me", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	env.Store.Close(iss.ID, "wontfix")
	env.CommitIntent("close " + iss.ID)

	reopened, err := env.Store.Reopen(iss.ID)
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if reopened.ClosedAt != "" {
		t.Errorf("closed_at = %q, want empty after reopen", reopened.ClosedAt)
	}
	if reopened.CloseReason != "" {
		t.Errorf("close_reason = %q, want empty after reopen", reopened.CloseReason)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.ClosedAt != "" || got.CloseReason != "" {
		t.Errorf("persisted close fields should be cleared: closed_at=%q close_reason=%q", got.ClosedAt, got.CloseReason)
	}
}

func TestUpdatedAtOnLabel(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Label me", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	labeled, _ := env.Store.Label(iss.ID, []string{"bug"}, nil)
	if labeled.UpdatedAt == "" {
		t.Error("updated_at should be set after label")
	}
}

func TestUpdatedAtOnLink(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.CommitIntent("create issues")

	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("link")

	gotA, _ := env.Store.Get(a.ID)
	gotB, _ := env.Store.Get(b.ID)
	if gotA.UpdatedAt == "" {
		t.Error("blocker updated_at should be set after link")
	}
	if gotB.UpdatedAt == "" {
		t.Error("blocked updated_at should be set after link")
	}
}

// --- Statuses filter (bw-17q) ---

func TestListStatuses(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Open task", issue.CreateOpts{})
	env.CommitIntent("create " + a.ID)

	b, _ := env.Store.Create("WIP task", issue.CreateOpts{})
	statusIP := "in_progress"
	env.Store.Update(b.ID, issue.UpdateOpts{Status: &statusIP})
	env.CommitIntent("create " + b.ID)

	c, _ := env.Store.Create("Closed task", issue.CreateOpts{})
	env.Store.Close(c.ID, "")
	env.CommitIntent("close " + c.ID)

	// Statuses=["open","in_progress"] should return both open and in_progress
	both, err := env.Store.List(issue.Filter{Statuses: []string{"open", "in_progress"}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(both) != 2 {
		t.Errorf("got %d issues, want 2 (open + in_progress)", len(both))
	}

	// Single Status="open" still works (backward compat)
	open, err := env.Store.List(issue.Filter{Status: "open"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(open) != 1 {
		t.Errorf("got %d open, want 1", len(open))
	}

	// Statuses takes precedence over Status
	precedence, err := env.Store.List(issue.Filter{
		Status:   "closed",
		Statuses: []string{"open"},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(precedence) != 1 {
		t.Errorf("got %d issues, want 1 (Statuses should override Status)", len(precedence))
	}
	if len(precedence) > 0 && precedence[0].Status != "open" {
		t.Errorf("got status %q, want open", precedence[0].Status)
	}
}

// --- Grep filter (bw-wdi) ---

func TestListGrep(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Login page broken", issue.CreateOpts{Description: "The form is blank"})
	env.CommitIntent("create " + a.ID)
	b, _ := env.Store.Create("Update readme", issue.CreateOpts{Description: "Add auth instructions"})
	env.CommitIntent("create " + b.ID)
	c, _ := env.Store.Create("Fix sidebar", issue.CreateOpts{})
	env.CommitIntent("create " + c.ID)

	// Grep matches title
	login, err := env.Store.List(issue.Filter{Grep: "login"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(login) != 1 {
		t.Errorf("grep 'login': got %d, want 1", len(login))
	}

	// Grep matches description
	auth, err := env.Store.List(issue.Filter{Grep: "auth"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(auth) != 1 {
		t.Errorf("grep 'auth': got %d, want 1", len(auth))
	}

	// Grep is case-insensitive
	upper, err := env.Store.List(issue.Filter{Grep: "LOGIN"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(upper) != 1 {
		t.Errorf("grep 'LOGIN' (case-insensitive): got %d, want 1", len(upper))
	}

	// Grep combined with Status filter
	closedStatus := "closed"
	env.Store.Update(a.ID, issue.UpdateOpts{Status: &closedStatus})
	openLogin, err := env.Store.List(issue.Filter{Grep: "login", Status: "open"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(openLogin) != 0 {
		t.Errorf("grep 'login' + status 'open': got %d, want 0", len(openLogin))
	}

	// Grep with no matches
	none, err := env.Store.List(issue.Filter{Grep: "nonexistent"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("grep 'nonexistent': got %d, want 0", len(none))
	}
}

// --- DeletePreview tests ---

func TestDeletePreviewNoDeps(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Solo issue", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	env.CommitIntent("create " + iss.ID)

	plan, err := env.Store.DeletePreview(iss.ID)
	if err != nil {
		t.Fatalf("DeletePreview: %v", err)
	}
	if plan.Issue.ID != iss.ID {
		t.Errorf("plan.Issue.ID = %q, want %q", plan.Issue.ID, iss.ID)
	}
	if len(plan.Blocks) != 0 {
		t.Errorf("plan.Blocks = %v, want empty", plan.Blocks)
	}
	if len(plan.BlockedBy) != 0 {
		t.Errorf("plan.BlockedBy = %v, want empty", plan.BlockedBy)
	}
	if len(plan.Children) != 0 {
		t.Errorf("plan.Children = %v, want empty", plan.Children)
	}
}

func TestDeletePreviewBlocks(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	plan, err := env.Store.DeletePreview(a.ID)
	if err != nil {
		t.Fatalf("DeletePreview: %v", err)
	}
	if len(plan.Blocks) != 1 || plan.Blocks[0] != b.ID {
		t.Errorf("plan.Blocks = %v, want [%s]", plan.Blocks, b.ID)
	}
}

func TestDeletePreviewBlockedBy(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	plan, err := env.Store.DeletePreview(b.ID)
	if err != nil {
		t.Fatalf("DeletePreview: %v", err)
	}
	if len(plan.BlockedBy) != 1 || plan.BlockedBy[0] != a.ID {
		t.Errorf("plan.BlockedBy = %v, want [%s]", plan.BlockedBy, a.ID)
	}
}

func TestDeletePreviewChildren(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Parent", issue.CreateOpts{})
	env.CommitIntent("create parent")
	env.Store.Import(&issue.Issue{
		ID:        "test-child",
		Title:     "Child",
		Status:    "open",
		Priority:  2,
		Type:      "task",
		Created:   "2024-01-01",
		Parent:    parent.ID,
		Labels:    []string{},
		Blocks:    []string{},
		BlockedBy: []string{},
	})
	env.CommitIntent("setup")

	plan, err := env.Store.DeletePreview(parent.ID)
	if err != nil {
		t.Fatalf("DeletePreview: %v", err)
	}
	if len(plan.Children) != 1 || plan.Children[0] != "test-child" {
		t.Errorf("plan.Children = %v, want [test-child]", plan.Children)
	}
}

func TestDeletePreviewNonexistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.DeletePreview("test-zzzz")
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

// --- Delete tests ---

func TestDeleteRemovesIssueFile(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To delete", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	_, err := env.Store.Delete(iss.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	env.CommitIntent("delete " + iss.ID)

	_, err = env.Store.Get(iss.ID)
	if err == nil {
		t.Error("expected error getting deleted issue")
	}
}

func TestDeleteRemovesStatusMarker(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To delete", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	env.Store.Delete(iss.ID)
	env.CommitIntent("delete " + iss.ID)

	issues, _ := env.Store.List(issue.Filter{})
	for _, i := range issues {
		if i.ID == iss.ID {
			t.Error("deleted issue still appears in list")
		}
	}
}

func TestDeleteCleansUpBlocksMarkers(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	env.Store.Delete(a.ID)
	env.CommitIntent("delete " + a.ID)

	// The blocks marker file should be gone
	if env.MarkerExists("blocks/" + a.ID + "/" + b.ID) {
		t.Error("blocks marker still exists after delete")
	}
}

func TestDeleteUpdatesRelatedBlockedBy(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	env.Store.Delete(a.ID)
	env.CommitIntent("delete " + a.ID)

	gotB, _ := env.Store.Get(b.ID)
	if len(gotB.BlockedBy) != 0 {
		t.Errorf("blocked.BlockedBy = %v, want empty", gotB.BlockedBy)
	}
}

func TestDeleteUpdatesRelatedBlocks(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	env.Store.Delete(b.ID)
	env.CommitIntent("delete " + b.ID)

	gotA, _ := env.Store.Get(a.ID)
	if len(gotA.Blocks) != 0 {
		t.Errorf("blocker.Blocks = %v, want empty", gotA.Blocks)
	}
}

func TestDeleteOrphansChildren(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Parent", issue.CreateOpts{})
	env.CommitIntent("create parent")
	env.Store.Import(&issue.Issue{
		ID:        "test-child",
		Title:     "Child",
		Status:    "open",
		Priority:  2,
		Type:      "task",
		Created:   "2024-01-01",
		Parent:    parent.ID,
		Labels:    []string{},
		Blocks:    []string{},
		BlockedBy: []string{},
	})
	env.CommitIntent("setup")

	env.Store.Delete(parent.ID)
	env.CommitIntent("delete " + parent.ID)

	gotChild, _ := env.Store.Get("test-child")
	if gotChild.Parent != "" {
		t.Errorf("child.Parent = %q, want empty", gotChild.Parent)
	}
}

func TestDeleteNoRelationshipsClean(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Solo", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	deleted, err := env.Store.Delete(iss.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted.ID != iss.ID {
		t.Errorf("deleted.ID = %q, want %q", deleted.ID, iss.ID)
	}
	if deleted.Title != "Solo" {
		t.Errorf("deleted.Title = %q, want Solo", deleted.Title)
	}
}

func TestDeleteNonexistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.Delete("test-zzzz")
	if err == nil {
		t.Error("expected error for nonexistent issue")
	}
}

// --- Parent field tests ---

func TestCreateWithParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, err := env.Store.Create("Parent issue", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create parent: %v", err)
	}
	env.CommitIntent("create " + parent.ID)

	child, err := env.Store.Create("Child issue", issue.CreateOpts{
		Parent: parent.ID,
	})
	if err != nil {
		t.Fatalf("Create child: %v", err)
	}
	env.CommitIntent("create " + child.ID)

	// Child should get a dotted ID
	want := parent.ID + ".1"
	if child.ID != want {
		t.Errorf("child.ID = %q, want %q", child.ID, want)
	}

	got, err := env.Store.Get(child.ID)
	if err != nil {
		t.Fatalf("Get child: %v", err)
	}
	if got.Parent != parent.ID {
		t.Errorf("child.Parent = %q, want %q", got.Parent, parent.ID)
	}
}

func TestCreateChildIDsAreSequential(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Parent", issue.CreateOpts{})
	env.CommitIntent("create " + parent.ID)

	c1, _ := env.Store.Create("Child 1", issue.CreateOpts{Parent: parent.ID})
	env.CommitIntent("create " + c1.ID)
	c2, _ := env.Store.Create("Child 2", issue.CreateOpts{Parent: parent.ID})
	env.CommitIntent("create " + c2.ID)
	c3, _ := env.Store.Create("Child 3", issue.CreateOpts{Parent: parent.ID})
	env.CommitIntent("create " + c3.ID)

	if c1.ID != parent.ID+".1" {
		t.Errorf("c1.ID = %q, want %s.1", c1.ID, parent.ID)
	}
	if c2.ID != parent.ID+".2" {
		t.Errorf("c2.ID = %q, want %s.2", c2.ID, parent.ID)
	}
	if c3.ID != parent.ID+".3" {
		t.Errorf("c3.ID = %q, want %s.3", c3.ID, parent.ID)
	}
}

func TestCreateGrandchild(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Parent", issue.CreateOpts{})
	env.CommitIntent("create " + parent.ID)

	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: parent.ID})
	env.CommitIntent("create " + child.ID)

	grandchild, _ := env.Store.Create("Grandchild", issue.CreateOpts{Parent: child.ID})
	env.CommitIntent("create " + grandchild.ID)

	// e.g., test-abc.1.1
	want := child.ID + ".1"
	if grandchild.ID != want {
		t.Errorf("grandchild.ID = %q, want %q", grandchild.ID, want)
	}
}

func TestDottedChildIDPermanentAfterOrphaning(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Parent", issue.CreateOpts{})
	env.CommitIntent("create " + parent.ID)

	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: parent.ID})
	env.CommitIntent("create " + child.ID)

	// Clear parent link
	empty := ""
	env.Store.Update(child.ID, issue.UpdateOpts{Parent: &empty})
	env.CommitIntent("clear parent")

	got, _ := env.Store.Get(child.ID)
	if got.Parent != "" {
		t.Errorf("Parent should be empty, got %q", got.Parent)
	}
	// ID should still be dotted
	if got.ID != parent.ID+".1" {
		t.Errorf("ID should remain %s.1 after orphaning, got %q", parent.ID, got.ID)
	}
}

func TestCreateWithNonexistentParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.Create("Orphan", issue.CreateOpts{
		Parent: "test-zzzz",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent parent")
	}
}

func TestUpdateSetParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, err := env.Store.Create("Parent", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create parent: %v", err)
	}
	child, err := env.Store.Create("Child", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create child: %v", err)
	}
	env.CommitIntent("create issues")

	parentID := parent.ID
	updated, err := env.Store.Update(child.ID, issue.UpdateOpts{
		Parent: &parentID,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Parent != parent.ID {
		t.Errorf("Parent = %q, want %q", updated.Parent, parent.ID)
	}

	got, err := env.Store.Get(child.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Parent != parent.ID {
		t.Errorf("persisted Parent = %q, want %q", got.Parent, parent.ID)
	}
}

func TestUpdateClearParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, err := env.Store.Create("Parent", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create parent: %v", err)
	}
	child, err := env.Store.Create("Child", issue.CreateOpts{
		Parent: parent.ID,
	})
	if err != nil {
		t.Fatalf("Create child: %v", err)
	}
	env.CommitIntent("create issues")

	empty := ""
	updated, err := env.Store.Update(child.ID, issue.UpdateOpts{
		Parent: &empty,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Parent != "" {
		t.Errorf("Parent = %q, want empty", updated.Parent)
	}
}

func TestUpdateSelfParentRejected(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Self", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	env.CommitIntent("create " + iss.ID)

	selfID := iss.ID
	_, err = env.Store.Update(iss.ID, issue.UpdateOpts{
		Parent: &selfID,
	})
	if err == nil {
		t.Fatal("expected error for self-parent")
	}
	if !strings.Contains(err.Error(), "own parent") {
		t.Errorf("error = %q, want mention of 'own parent'", err.Error())
	}
}

func TestUpdateDirectCycleRejected(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, err := env.Store.Create("A", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	b, err := env.Store.Create("B", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}
	env.CommitIntent("create issues")

	aID := a.ID
	_, err = env.Store.Update(b.ID, issue.UpdateOpts{Parent: &aID})
	if err != nil {
		t.Fatalf("Set A as parent of B: %v", err)
	}
	env.CommitIntent("update parent")

	bID := b.ID
	_, err = env.Store.Update(a.ID, issue.UpdateOpts{Parent: &bID})
	if err == nil {
		t.Fatal("expected error for direct cycle")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error = %q, want mention of 'circular'", err.Error())
	}
}

func TestUpdateDeepCycleRejected(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, err := env.Store.Create("A", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	b, err := env.Store.Create("B", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}
	c, err := env.Store.Create("C", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create C: %v", err)
	}
	env.CommitIntent("create issues")

	aID := a.ID
	_, err = env.Store.Update(b.ID, issue.UpdateOpts{Parent: &aID})
	if err != nil {
		t.Fatalf("Set A as parent of B: %v", err)
	}
	bID := b.ID
	_, err = env.Store.Update(c.ID, issue.UpdateOpts{Parent: &bID})
	if err != nil {
		t.Fatalf("Set B as parent of C: %v", err)
	}
	env.CommitIntent("setup chain")

	cID := c.ID
	_, err = env.Store.Update(a.ID, issue.UpdateOpts{Parent: &cID})
	if err == nil {
		t.Fatal("expected error for deep cycle")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error = %q, want mention of 'circular'", err.Error())
	}
}

func TestUpdateParentNonexistentRejected(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Issue", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	env.CommitIntent("create " + iss.ID)

	bad := "test-zzzz"
	_, err = env.Store.Update(iss.ID, issue.UpdateOpts{
		Parent: &bad,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent parent")
	}
}

func TestNewlyUnblockedSingle(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(a.ID, "")
	env.CommitIntent("close")

	unblocked, err := env.Store.NewlyUnblocked(a.ID)
	if err != nil {
		t.Fatalf("NewlyUnblocked: %v", err)
	}
	if len(unblocked) != 1 || unblocked[0].ID != b.ID {
		t.Errorf("got %d unblocked, want [%s]", len(unblocked), b.ID)
	}
}

func TestNewlyUnblockedMultipleBlockers(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker A", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	c, _ := env.Store.Create("Blocker C", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Link(c.ID, b.ID)
	env.Store.Close(a.ID, "")
	env.CommitIntent("close")

	unblocked, err := env.Store.NewlyUnblocked(a.ID)
	if err != nil {
		t.Fatalf("NewlyUnblocked: %v", err)
	}
	if len(unblocked) != 0 {
		t.Errorf("got %d unblocked, want 0 (C still open)", len(unblocked))
	}
}

func TestNewlyUnblockedNoBlocks(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("No deps", issue.CreateOpts{})
	env.Store.Close(a.ID, "")
	env.CommitIntent("close")

	unblocked, err := env.Store.NewlyUnblocked(a.ID)
	if err != nil {
		t.Fatalf("NewlyUnblocked: %v", err)
	}
	if len(unblocked) != 0 {
		t.Errorf("got %d unblocked, want 0", len(unblocked))
	}
}

func TestNewlyUnblockedSkipsClosed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Already closed", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(b.ID, "")
	env.Store.Close(a.ID, "")
	env.CommitIntent("close")

	unblocked, err := env.Store.NewlyUnblocked(a.ID)
	if err != nil {
		t.Fatalf("NewlyUnblocked: %v", err)
	}
	if len(unblocked) != 0 {
		t.Errorf("got %d unblocked, want 0 (B is closed)", len(unblocked))
	}
}

func TestNewlyUnblockedMultiple(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked B", issue.CreateOpts{})
	c, _ := env.Store.Create("Blocked C", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Link(a.ID, c.ID)
	env.Store.Close(a.ID, "")
	env.CommitIntent("close")

	unblocked, err := env.Store.NewlyUnblocked(a.ID)
	if err != nil {
		t.Fatalf("NewlyUnblocked: %v", err)
	}
	if len(unblocked) != 2 {
		t.Errorf("got %d unblocked, want 2", len(unblocked))
	}
}

// --- LoadEdges tests ---

func TestLoadEdges(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	c, _ := env.Store.Create("C", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID) // a blocks b
	env.Store.Link(b.ID, c.ID) // b blocks c
	env.CommitIntent("setup")

	fwd, rev := env.Store.LoadEdges()

	// Forward: a→[b], b→[c]
	if len(fwd[a.ID]) != 1 || fwd[a.ID][0] != b.ID {
		t.Errorf("forward[a] = %v, want [%s]", fwd[a.ID], b.ID)
	}
	if len(fwd[b.ID]) != 1 || fwd[b.ID][0] != c.ID {
		t.Errorf("forward[b] = %v, want [%s]", fwd[b.ID], c.ID)
	}

	// Reverse: b→[a], c→[b]
	if len(rev[b.ID]) != 1 || rev[b.ID][0] != a.ID {
		t.Errorf("reverse[b] = %v, want [%s]", rev[b.ID], a.ID)
	}
	if len(rev[c.ID]) != 1 || rev[c.ID][0] != b.ID {
		t.Errorf("reverse[c] = %v, want [%s]", rev[c.ID], b.ID)
	}

	// No edges from/to c in forward, no edges from/to a in reverse
	if len(fwd[c.ID]) != 0 {
		t.Errorf("forward[c] should be empty, got %v", fwd[c.ID])
	}
	if len(rev[a.ID]) != 0 {
		t.Errorf("reverse[a] should be empty, got %v", rev[a.ID])
	}
}

// --- Tips tests ---

func TestTipsSimpleChain(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// A blocked by B blocked by C
	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	c, _ := env.Store.Create("C", issue.CreateOpts{})
	env.Store.Link(c.ID, b.ID) // c blocks b
	env.Store.Link(b.ID, a.ID) // b blocks a
	env.CommitIntent("setup")

	// Re-read a to get updated BlockedBy
	a, _ = env.Store.Get(a.ID)

	// Walk reverse edges (blocked_by direction) from A's blockers
	_, rev := env.Store.LoadEdges()
	tips, err := env.Store.Tips(a.BlockedBy, rev)
	if err != nil {
		t.Fatalf("Tips: %v", err)
	}

	// Should find C (the leaf), not B
	if len(tips) != 1 {
		t.Fatalf("got %d tips, want 1", len(tips))
	}
	if tips[0].ID != c.ID {
		t.Errorf("tip = %s, want %s", tips[0].ID, c.ID)
	}
}

func TestTipsDiamond(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// A blocked by B and C, both blocked by D
	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	c, _ := env.Store.Create("C", issue.CreateOpts{})
	d, _ := env.Store.Create("D", issue.CreateOpts{})
	env.Store.Link(b.ID, a.ID) // b blocks a
	env.Store.Link(c.ID, a.ID) // c blocks a
	env.Store.Link(d.ID, b.ID) // d blocks b
	env.Store.Link(d.ID, c.ID) // d blocks c
	env.CommitIntent("setup")

	// Reload a to get updated BlockedBy
	a, _ = env.Store.Get(a.ID)

	_, rev := env.Store.LoadEdges()
	tips, err := env.Store.Tips(a.BlockedBy, rev)
	if err != nil {
		t.Fatalf("Tips: %v", err)
	}

	// Should find D (deduplicated)
	if len(tips) != 1 {
		t.Fatalf("got %d tips, want 1", len(tips))
	}
	if tips[0].ID != d.ID {
		t.Errorf("tip = %s, want %s", tips[0].ID, d.ID)
	}
}

func TestTipsClosedIntermediary(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// A blocked by B blocked by C, B is closed
	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	c, _ := env.Store.Create("C", issue.CreateOpts{})
	env.Store.Link(c.ID, b.ID)
	env.Store.Link(b.ID, a.ID)
	env.Store.Close(b.ID, "done")
	env.CommitIntent("setup")

	// Re-read a to get updated BlockedBy
	a, _ = env.Store.Get(a.ID)

	_, rev := env.Store.LoadEdges()
	tips, err := env.Store.Tips(a.BlockedBy, rev)
	if err != nil {
		t.Fatalf("Tips: %v", err)
	}

	// Should walk through closed B and find C
	if len(tips) != 1 {
		t.Fatalf("got %d tips, want 1", len(tips))
	}
	if tips[0].ID != c.ID {
		t.Errorf("tip = %s, want %s", tips[0].ID, c.ID)
	}
}

func TestTipsNoBlockers(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("A", issue.CreateOpts{})
	env.CommitIntent("setup")

	_, rev := env.Store.LoadEdges()
	tips, err := env.Store.Tips(nil, rev)
	if err != nil {
		t.Fatalf("Tips: %v", err)
	}

	if len(tips) != 0 {
		t.Errorf("got %d tips, want 0", len(tips))
	}
}

func TestTipsMultipleRoots(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Two independent chains: A←B and X←Y
	a, _ := env.Store.Create("A", issue.CreateOpts{})
	b, _ := env.Store.Create("B", issue.CreateOpts{})
	x, _ := env.Store.Create("X", issue.CreateOpts{})
	y, _ := env.Store.Create("Y", issue.CreateOpts{})
	env.Store.Link(b.ID, a.ID)
	env.Store.Link(y.ID, x.ID)
	env.CommitIntent("setup")

	// Re-read to get updated BlockedBy
	a, _ = env.Store.Get(a.ID)
	x, _ = env.Store.Get(x.ID)

	_, rev := env.Store.LoadEdges()
	tips, err := env.Store.Tips([]string{a.BlockedBy[0], x.BlockedBy[0]}, rev)
	if err != nil {
		t.Fatalf("Tips: %v", err)
	}

	// Should find B and Y
	if len(tips) != 2 {
		t.Fatalf("got %d tips, want 2", len(tips))
	}
	ids := map[string]bool{tips[0].ID: true, tips[1].ID: true}
	if !ids[b.ID] || !ids[y.ID] {
		t.Errorf("tips = %v, want {%s, %s}", ids, b.ID, y.ID)
	}
}

// TestReadyTipsChain verifies that in A←B←C (all open), only C (the tip) is ready.
func TestReadyTipsChain(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Target", issue.CreateOpts{})
	b, _ := env.Store.Create("Middle", issue.CreateOpts{})
	c, _ := env.Store.Create("Leaf", issue.CreateOpts{})
	env.Store.Link(c.ID, b.ID) // C blocks B
	env.Store.Link(b.ID, a.ID) // B blocks A
	env.CommitIntent("setup chain")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[c.ID] {
		t.Error("leaf C should be ready (no blockers)")
	}
	if ids[b.ID] {
		t.Error("middle B should NOT be ready (blocked by C)")
	}
	if ids[a.ID] {
		t.Error("target A should NOT be ready (blocked by B)")
	}
}

// TestReadyBlockerIsActionable verifies that a blocker with no blockers appears in ready.
func TestReadyBlockerIsActionable(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[a.ID] {
		t.Error("blocker A should be ready (it has no blockers itself)")
	}
	if ids[b.ID] {
		t.Error("blocked B should NOT be ready")
	}
}

// TestReadyChainPartialClose verifies that closing a leaf promotes the next in line.
func TestReadyChainPartialClose(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Target", issue.CreateOpts{})
	b, _ := env.Store.Create("Middle", issue.CreateOpts{})
	c, _ := env.Store.Create("Leaf", issue.CreateOpts{})
	env.Store.Link(c.ID, b.ID)
	env.Store.Link(b.ID, a.ID)
	env.CommitIntent("setup chain")

	// Close C → B should become ready
	env.Store.Close(c.ID, "")
	env.CommitIntent("close leaf")

	ready, err := env.Store.Ready()
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	ids := make(map[string]bool)
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[b.ID] {
		t.Error("middle B should be ready after leaf C closed")
	}
	if ids[a.ID] {
		t.Error("target A should NOT be ready (still blocked by B)")
	}
}
