package issue_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func issueIDs(issues []*issue.Issue) []string {
	ids := make([]string, len(issues))
	for i, iss := range issues {
		ids[i] = iss.ID
	}
	return ids
}

func TestArchiveClosedIssue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Done thing", issue.CreateOpts{})
	env.Store.Label(iss.ID, []string{"bug"}, nil)
	env.Store.Close(iss.ID, "completed")
	env.CommitIntent("setup")

	got, err := env.Store.Archive(iss.ID, issue.ArchiveOpts{})
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if got.ArchivedAt == "" {
		t.Error("ArchivedAt not stamped")
	}
	if !env.MarkerExists("archive/" + iss.ID + ".json") {
		t.Error("archive/<id>.json missing")
	}
	if env.IssueFileExists(iss.ID) {
		t.Error("issues/<id>.json should be gone")
	}
	if env.MarkerExists("status/closed/" + iss.ID) {
		t.Error("status/closed marker should be gone")
	}
	if env.MarkerExists("labels/bug/" + iss.ID) {
		t.Error("label marker should be gone")
	}
	if _, err := env.Store.Get(iss.ID); err == nil {
		t.Error("archived issue should no longer resolve via Get")
	}
	a, err := env.Store.ArchivedIssue(iss.ID)
	if err != nil {
		t.Fatalf("ArchivedIssue: %v", err)
	}
	if a.Title != "Done thing" {
		t.Errorf("archived title = %q, want %q", a.Title, "Done thing")
	}
}

func TestArchiveExcludedFromListAndReady(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Done thing", issue.CreateOpts{})
	env.Store.Close(iss.ID, "")
	env.CommitIntent("setup")

	if _, err := env.Store.Archive(iss.ID, issue.ArchiveOpts{}); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	all, _ := env.Store.List(issue.Filter{})
	for _, l := range all {
		if l.ID == iss.ID {
			t.Error("archived issue appeared in List")
		}
	}
	closed, _ := env.Store.List(issue.Filter{Status: "closed"})
	for _, l := range closed {
		if l.ID == iss.ID {
			t.Error("archived issue appeared in List(status=closed)")
		}
	}
}

func TestArchiveRefusesOpenIssue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Active", issue.CreateOpts{})
	env.CommitIntent("setup")

	_, err := env.Store.Archive(iss.ID, issue.ArchiveOpts{})
	if err == nil {
		t.Fatal("expected error archiving an open issue")
	}
	var nce *issue.NotClosedError
	if !errors.As(err, &nce) {
		t.Errorf("want *NotClosedError, got %T: %v", err, err)
	}
	if !env.IssueFileExists(iss.ID) {
		t.Error("issue file should remain after refused archive")
	}
}

func TestArchiveClosesWithCloseOpt(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Active", issue.CreateOpts{})
	env.CommitIntent("setup")

	got, err := env.Store.Archive(iss.ID, issue.ArchiveOpts{Close: true})
	if err != nil {
		t.Fatalf("Archive --close: %v", err)
	}
	if got.Status != "closed" || got.ClosedAt == "" {
		t.Errorf("want closed with ClosedAt, got status=%q closedAt=%q", got.Status, got.ClosedAt)
	}
	if !env.MarkerExists("archive/" + iss.ID + ".json") {
		t.Error("archive/<id>.json missing")
	}
	if env.MarkerExists("status/open/" + iss.ID) {
		t.Error("status/open marker should be gone")
	}
}

func TestArchiveRefusesOpenDependent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	blocker, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	blocked, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(blocker.ID, blocked.ID) // blocker blocks blocked
	env.Store.Close(blocker.ID, "")
	env.CommitIntent("setup")

	_, err := env.Store.Archive(blocker.ID, issue.ArchiveOpts{})
	if err == nil {
		t.Fatal("expected refusal: open dependent")
	}
	var abe *issue.ArchiveBlockedError
	if !errors.As(err, &abe) {
		t.Fatalf("want *ArchiveBlockedError, got %T: %v", err, err)
	}
	if len(abe.OpenDependents) != 1 || abe.OpenDependents[0] != blocked.ID {
		t.Errorf("OpenDependents = %v, want [%s]", abe.OpenDependents, blocked.ID)
	}
	if !env.IssueFileExists(blocker.ID) {
		t.Error("blocker should remain after refused archive")
	}
}

func TestArchiveDetachDropsEdgeAndUnblocks(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	blocker, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	blocked, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(blocker.ID, blocked.ID)
	env.Store.Close(blocker.ID, "")
	env.CommitIntent("setup")

	if _, err := env.Store.Archive(blocker.ID, issue.ArchiveOpts{Detach: true}); err != nil {
		t.Fatalf("Archive --detach: %v", err)
	}

	b, _ := env.Store.Get(blocked.ID)
	if len(b.BlockedBy) != 0 {
		t.Errorf("blocked.BlockedBy = %v, want empty", b.BlockedBy)
	}
	if env.MarkerExists("blocks/" + blocker.ID + "/" + blocked.ID) {
		t.Error("blocks marker should be gone")
	}
	ready, _ := env.Store.Ready()
	found := false
	for _, r := range ready {
		if r.ID == blocked.ID {
			found = true
		}
	}
	if !found {
		t.Error("blocked should be ready after its only blocker is archived")
	}
}

func TestArchiveClosedDependentNeedsNoDetach(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	blocker, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	blocked, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(blocker.ID, blocked.ID)
	env.Store.Close(blocked.ID, "")
	env.Store.Close(blocker.ID, "")
	env.CommitIntent("setup")

	if _, err := env.Store.Archive(blocker.ID, issue.ArchiveOpts{}); err != nil {
		t.Fatalf("Archive (closed dependent): %v", err)
	}
	b, _ := env.Store.Get(blocked.ID)
	if len(b.BlockedBy) != 0 {
		t.Errorf("blocked.BlockedBy = %v, want empty after archive", b.BlockedBy)
	}
}

func TestArchiveRefusesOpenChildThenDetachOrphans(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Parent", issue.CreateOpts{})
	child, _ := env.Store.Create("Child", issue.CreateOpts{Parent: parent.ID})
	env.Store.Close(parent.ID, "")
	env.CommitIntent("setup")

	_, err := env.Store.Archive(parent.ID, issue.ArchiveOpts{})
	if err == nil {
		t.Fatal("expected refusal: open child")
	}
	var abe *issue.ArchiveBlockedError
	if !errors.As(err, &abe) || len(abe.OpenChildren) != 1 || abe.OpenChildren[0] != child.ID {
		t.Fatalf("want ArchiveBlockedError with open child %s, got %v", child.ID, err)
	}

	if _, err := env.Store.Archive(parent.ID, issue.ArchiveOpts{Detach: true}); err != nil {
		t.Fatalf("Archive --detach: %v", err)
	}
	c, _ := env.Store.Get(child.ID)
	if c.Parent != "" {
		t.Errorf("child.Parent = %q, want cleared", c.Parent)
	}
}

func TestArchiveAlreadyArchivedFails(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Done", issue.CreateOpts{})
	env.Store.Close(iss.ID, "")
	env.CommitIntent("setup")
	if _, err := env.Store.Archive(iss.ID, issue.ArchiveOpts{}); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if _, err := env.Store.Archive(iss.ID, issue.ArchiveOpts{}); err == nil {
		t.Error("archiving an already-archived issue should fail")
	}
}

func TestArchivedIDCannotBeReused(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Done", issue.CreateOpts{})
	id := iss.ID
	env.Store.Close(id, "")
	env.CommitIntent("setup")
	if _, err := env.Store.Archive(id, issue.ArchiveOpts{}); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	// An archived ID is out of the live space but must not be recyclable —
	// reuse would collide with the archived record and break recovery.
	if _, err := env.Store.Create("Imposter", issue.CreateOpts{ID: id}); err == nil {
		t.Errorf("creating an issue with archived ID %q should fail", id)
	}
}

func TestClosedBefore(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	mk := func(id, closedAt string) {
		env.Store.Import(&issue.Issue{
			ID: id, Title: id, Status: "closed", Priority: 2, Type: "task",
			Created: "2024-01-01T00:00:00Z", ClosedAt: closedAt,
			Labels: []string{}, Blocks: []string{}, BlockedBy: []string{},
		})
	}
	mk("test-old", "2025-06-01T00:00:00Z")
	mk("test-new", "2026-03-01T00:00:00Z")
	// An open issue is never returned, regardless of age.
	env.Store.Import(&issue.Issue{
		ID: "test-open", Title: "open", Status: "open", Priority: 2, Type: "task",
		Created: "2020-01-01T00:00:00Z",
		Labels:  []string{}, Blocks: []string{}, BlockedBy: []string{},
	})
	env.CommitIntent("setup")

	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got, err := env.Store.ClosedBefore(cutoff)
	if err != nil {
		t.Fatalf("ClosedBefore: %v", err)
	}
	if len(got) != 1 || got[0].ID != "test-old" {
		t.Errorf("ClosedBefore = %v, want [test-old]", issueIDs(got))
	}
}
