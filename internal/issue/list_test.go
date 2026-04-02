package issue_test

import (
	"encoding/json"
	"testing"
	"time"

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

// --- Parent filter ---

func TestListFilterByParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	epic, _ := env.Store.Create("Epic", issue.CreateOpts{})
	env.CommitIntent("create " + epic.ID)
	child1, _ := env.Store.Create("Child 1", issue.CreateOpts{Parent: epic.ID})
	env.CommitIntent("create " + child1.ID)
	child2, _ := env.Store.Create("Child 2", issue.CreateOpts{Parent: epic.ID})
	env.CommitIntent("create " + child2.ID)
	env.Store.Create("Standalone", issue.CreateOpts{})
	env.CommitIntent("create standalone")

	children, err := env.Store.List(issue.Filter{Parent: epic.ID})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("got %d children, want 2", len(children))
	}
	for _, c := range children {
		if c.Parent != epic.ID {
			t.Errorf("issue %s has parent %q, want %q", c.ID, c.Parent, epic.ID)
		}
	}
}

// --- Due dates and overdue ---

func TestIsOverdue(t *testing.T) {
	// Fixed reference: 2027-04-16 at noon UTC
	now := time.Date(2027, 4, 16, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		due  string
		want bool
	}{
		{"", false},                                    // no due date
		{"2027-04-17", false},                          // future date
		{"2027-04-16", false},                          // today (end-of-day: not overdue)
		{"2027-04-15", true},                           // yesterday
		{"2027-03-01", true},                           // long past
		{"2027-04-16T13:00:00Z", false},                // future RFC3339
		{"2027-04-16T11:00:00Z", true},                 // past RFC3339
		{"2027-04-16T12:00:00Z", false},                // exactly now (not After)
		{"2027-04-16T08:00:00-04:00", false},           // noon UTC = same instant
		{"2027-04-16T07:00:00-04:00", true},            // 11am UTC = past
		{"not-a-date", false},                          // invalid format
		{"2027-04-16T99:99:99Z", false},                // invalid RFC3339
	}

	for _, tt := range tests {
		got := issue.IsOverdue(tt.due, now)
		if got != tt.want {
			t.Errorf("IsOverdue(%q, now) = %v, want %v", tt.due, got, tt.want)
		}
	}
}

func TestSortIssuesOverdueFirst(t *testing.T) {
	now := time.Date(2027, 4, 16, 12, 0, 0, 0, time.UTC)

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// P2 overdue
	a, _ := env.Store.Create("Overdue task", issue.CreateOpts{Priority: intPtr(2), Due: "2027-04-01"})
	env.CommitIntent("create " + a.ID)

	// P2 not overdue
	b, _ := env.Store.Create("Normal task", issue.CreateOpts{Priority: intPtr(2), Due: "2027-12-01"})
	env.CommitIntent("create " + b.ID)

	// P1 not overdue (higher priority)
	c, _ := env.Store.Create("Urgent task", issue.CreateOpts{Priority: intPtr(1)})
	env.CommitIntent("create " + c.ID)

	// P2 no due date
	d, _ := env.Store.Create("No due task", issue.CreateOpts{Priority: intPtr(2)})
	env.CommitIntent("create " + d.ID)

	t.Setenv("BW_CLOCK", now.Format(time.RFC3339))

	issues, err := env.Store.List(issue.Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(issues) != 4 {
		t.Fatalf("got %d issues, want 4", len(issues))
	}

	// P1 first (priority wins over overdue)
	if issues[0].ID != c.ID {
		t.Errorf("issues[0] = %s, want %s (P1)", issues[0].ID, c.ID)
	}
	// Then P2 overdue
	if issues[1].ID != a.ID {
		t.Errorf("issues[1] = %s, want %s (P2 overdue)", issues[1].ID, a.ID)
	}
}

func TestListOverdueFilter(t *testing.T) {
	now := time.Date(2027, 4, 16, 12, 0, 0, 0, time.UTC)

	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Overdue open issue
	a, _ := env.Store.Create("Overdue", issue.CreateOpts{Due: "2027-04-01"})
	env.CommitIntent("create " + a.ID)

	// Future due issue
	b, _ := env.Store.Create("Future", issue.CreateOpts{Due: "2027-12-01"})
	env.CommitIntent("create " + b.ID)

	// No due date
	env.Store.Create("No due", issue.CreateOpts{})
	env.CommitIntent("create no-due")

	// Closed overdue (should NOT appear)
	d, _ := env.Store.Create("Closed overdue", issue.CreateOpts{Due: "2027-03-01"})
	env.CommitIntent("create " + d.ID)
	env.Store.Close(d.ID, "")
	env.CommitIntent("close " + d.ID)

	t.Setenv("BW_CLOCK", now.Format(time.RFC3339))

	overdue, err := env.Store.List(issue.Filter{Overdue: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(overdue) != 1 {
		t.Fatalf("got %d overdue, want 1", len(overdue))
	}
	if overdue[0].ID != a.ID {
		t.Errorf("overdue[0] = %s, want %s", overdue[0].ID, a.ID)
	}
}

func TestCreateWithDue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("With due", issue.CreateOpts{Due: "2027-06-01"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if iss.Due != "2027-06-01" {
		t.Errorf("Due = %q, want 2027-06-01", iss.Due)
	}
	if iss.Status != "open" {
		t.Errorf("Status = %q, want open (due should not change status)", iss.Status)
	}
}

func TestUpdateDue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Update due", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	// Set due
	due := "2027-06-01"
	updated, err := env.Store.Update(iss.ID, issue.UpdateOpts{Due: &due})
	if err != nil {
		t.Fatalf("Update (set): %v", err)
	}
	if updated.Due != "2027-06-01" {
		t.Errorf("Due = %q, want 2027-06-01", updated.Due)
	}
	if updated.Status != "open" {
		t.Errorf("Status = %q, want open (due should not change status)", updated.Status)
	}

	// Change due
	due2 := "2027-09-01"
	updated2, err := env.Store.Update(iss.ID, issue.UpdateOpts{Due: &due2})
	if err != nil {
		t.Fatalf("Update (change): %v", err)
	}
	if updated2.Due != "2027-09-01" {
		t.Errorf("Due = %q, want 2027-09-01", updated2.Due)
	}

	// Clear due
	empty := ""
	updated3, err := env.Store.Update(iss.ID, issue.UpdateOpts{Due: &empty})
	if err != nil {
		t.Fatalf("Update (clear): %v", err)
	}
	if updated3.Due != "" {
		t.Errorf("Due = %q, want empty", updated3.Due)
	}
}

func TestStoreNowAlwaysUTC(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	now := env.Store.Now()
	if now.Location() != time.UTC {
		t.Errorf("Store.Now() location = %v, want UTC", now.Location())
	}

	// With BW_CLOCK set
	t.Setenv("BW_CLOCK", "2027-04-15T14:00:00-04:00")
	now2 := env.Store.Now()
	if now2.Location() != time.UTC {
		t.Errorf("Store.Now() with BW_CLOCK location = %v, want UTC", now2.Location())
	}
}

// --- DeletePreview tests ---

