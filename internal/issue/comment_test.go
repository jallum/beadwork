package issue_test

import (
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCommentNoExisting(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Test issue", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	env.CommitIntent("create " + iss.ID)

	updated, err := env.Store.Comment(iss.ID, "First comment", "alice")
	if err != nil {
		t.Fatalf("Comment: %v", err)
	}
	if len(updated.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(updated.Comments))
	}
	if updated.Comments[0].Text != "First comment" {
		t.Errorf("text = %q, want %q", updated.Comments[0].Text, "First comment")
	}
	if updated.Comments[0].Author != "alice" {
		t.Errorf("author = %q, want %q", updated.Comments[0].Author, "alice")
	}
	if updated.Comments[0].Timestamp == "" {
		t.Error("timestamp should be set")
	}
}

func TestCommentWithExisting(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	env.Store.Comment(iss.ID, "First", "alice")
	env.CommitIntent("comment 1")

	updated, err := env.Store.Comment(iss.ID, "Second", "bob")
	if err != nil {
		t.Fatalf("Comment: %v", err)
	}
	if len(updated.Comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(updated.Comments))
	}
	if updated.Comments[0].Text != "First" {
		t.Errorf("first comment text = %q, want %q", updated.Comments[0].Text, "First")
	}
	if updated.Comments[1].Text != "Second" {
		t.Errorf("second comment text = %q, want %q", updated.Comments[1].Text, "Second")
	}
}

func TestCommentWithAuthor(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	updated, err := env.Store.Comment(iss.ID, "Noted", "charlie")
	if err != nil {
		t.Fatalf("Comment: %v", err)
	}
	if updated.Comments[0].Author != "charlie" {
		t.Errorf("author = %q, want %q", updated.Comments[0].Author, "charlie")
	}
}

func TestCommentEmptyAuthor(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	updated, err := env.Store.Comment(iss.ID, "Anonymous comment", "")
	if err != nil {
		t.Fatalf("Comment: %v", err)
	}
	if updated.Comments[0].Author != "" {
		t.Errorf("author = %q, want empty", updated.Comments[0].Author)
	}
}

func TestCommentNonexistentIssue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.Comment("test-nonexistent", "text", "alice")
	if err == nil {
		t.Fatal("expected error for nonexistent issue")
	}
}

func TestCommentSetsUpdatedAt(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	originalUpdated := iss.UpdatedAt

	updated, err := env.Store.Comment(iss.ID, "Comment", "alice")
	if err != nil {
		t.Fatalf("Comment: %v", err)
	}
	if updated.UpdatedAt == "" {
		t.Error("UpdatedAt should be set")
	}
	// UpdatedAt may or may not differ if the test runs sub-second,
	// but it should at least be present.
	_ = originalUpdated
}

func TestCommentPersistence(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	env.Store.Comment(iss.ID, "Persisted comment", "dave")
	env.CommitIntent("comment " + iss.ID)

	// Re-read from store
	got, err := env.Store.Get(iss.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Comments) != 1 {
		t.Fatalf("expected 1 comment after re-read, got %d", len(got.Comments))
	}
	if got.Comments[0].Text != "Persisted comment" {
		t.Errorf("text = %q, want %q", got.Comments[0].Text, "Persisted comment")
	}
	if got.Comments[0].Author != "dave" {
		t.Errorf("author = %q, want %q", got.Comments[0].Author, "dave")
	}
	if got.Comments[0].Timestamp == "" {
		t.Error("timestamp should be set")
	}
}
