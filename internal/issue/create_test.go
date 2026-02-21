package issue_test

import (
	"path/filepath"
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


func TestCommitDelegatesToCommitter(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Wire repo as the store's Committer.
	env.Store.Committer = env.Repo

	iss, err := env.Store.Create("Test commit", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	intent := "create " + iss.ID
	if err := env.Store.Commit(intent); err != nil {
		t.Fatalf("store.Commit: %v", err)
	}

	// Verify commit landed in git history.
	commits, err := env.Repo.AllCommits()
	if err != nil {
		t.Fatalf("AllCommits: %v", err)
	}
	found := false
	for _, c := range commits {
		if c.Message == intent {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("commit with intent %q not found in history", intent)
	}
}


func TestCommitWithoutCommitterErrors(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Nil out the Committer that NewEnv wires up.
	env.Store.Committer = nil
	err := env.Store.Commit("should fail")
	if err == nil {
		t.Fatal("expected error from read-only store, got nil")
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

