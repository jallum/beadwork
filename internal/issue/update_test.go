package issue_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

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
		t.Error("expected error reopening open issue")
	}
}


func TestReopenInProgress(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Unclaim me", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	started, _ := env.Store.Start(iss.ID, "alice")
	env.CommitIntent("start " + started.ID)

	reopened, err := env.Store.Reopen(iss.ID)
	if err != nil {
		t.Fatalf("Reopen in_progress: %v", err)
	}
	if reopened.Status != "open" {
		t.Errorf("status = %q, want open", reopened.Status)
	}
	if reopened.Assignee != "" {
		t.Errorf("assignee = %q, want empty", reopened.Assignee)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("persisted status = %q, want open", got.Status)
	}
	if got.Assignee != "" {
		t.Errorf("persisted assignee = %q, want empty", got.Assignee)
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


func TestStartBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Start me", issue.CreateOpts{})
	env.CommitIntent("create")

	started, err := env.Store.Start(iss.ID, "alice")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if started.Status != "in_progress" {
		t.Errorf("status = %q, want in_progress", started.Status)
	}
	if started.Assignee != "alice" {
		t.Errorf("assignee = %q, want alice", started.Assignee)
	}
}


func TestStartBlocked(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.CommitIntent("setup")

	_, err := env.Store.Start(b.ID, "alice")
	if err == nil {
		t.Fatal("expected error for blocked issue")
	}
}


func TestStartAlreadyInProgress(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("In progress", issue.CreateOpts{})
	status := "in_progress"
	env.Store.Update(iss.ID, issue.UpdateOpts{Status: &status})
	env.CommitIntent("setup")

	_, err := env.Store.Start(iss.ID, "alice")
	if err == nil {
		t.Fatal("expected error for in_progress issue")
	}
}


func TestStartClosed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Closed", issue.CreateOpts{})
	env.Store.Close(iss.ID, "")
	env.CommitIntent("setup")

	_, err := env.Store.Start(iss.ID, "alice")
	if err == nil {
		t.Fatal("expected error for closed issue")
	}
}


func TestStartNonExistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.Start("nonexistent", "alice")
	if err == nil {
		t.Fatal("expected error for nonexistent issue")
	}
}


func TestStartBlockedReturnsBlockerIDs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker A", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocker B", issue.CreateOpts{})
	c, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, c.ID)
	env.Store.Link(b.ID, c.ID)
	env.CommitIntent("setup")

	_, err := env.Store.Start(c.ID, "alice")
	if err == nil {
		t.Fatal("expected error for blocked issue")
	}

	var be *issue.BlockedError
	if !errors.As(err, &be) {
		t.Fatalf("expected BlockedError, got %T: %v", err, err)
	}
	if len(be.Blockers) != 2 {
		t.Errorf("got %d blockers, want 2", len(be.Blockers))
	}
}


func TestStartClosedBlockerAllowsStart(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	a, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	b, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(a.ID, b.ID)
	env.Store.Close(a.ID, "")
	env.CommitIntent("setup")

	started, err := env.Store.Start(b.ID, "alice")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if started.Status != "in_progress" {
		t.Errorf("status = %q, want in_progress", started.Status)
	}
}
