package issue_test

import (
	"encoding/json"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

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

