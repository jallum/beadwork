package issue_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

// TestAttachWritesBlobAndStagesEntry covers the internal write helper:
// Attach stages a blob at attachments/<ticketID>/<storedPath>, the
// subsequent commit makes it a real tree entry, and the commit message
// carries the matching `attach <id> <path>` intent line (see
// docs/design.md for the grammar).
func TestAttachWritesBlobAndStagesEntry(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	if err := env.Store.Attach("bw-x", "foo.txt", []byte("hi")); err != nil {
		t.Fatalf("Attach: %v", err)
	}
	env.CommitIntent("attach bw-x foo.txt")

	got, err := env.Repo.TreeFS().ReadFile("attachments/bw-x/foo.txt")
	if err != nil {
		t.Fatalf("ReadFile attachments/bw-x/foo.txt: %v", err)
	}
	if string(got) != "hi" {
		t.Errorf("contents = %q, want %q", got, "hi")
	}

	commits, err := env.Repo.AllCommits()
	if err != nil {
		t.Fatalf("AllCommits: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("no commits")
	}
	if !strings.Contains(commits[0].Message, "attach bw-x foo.txt") {
		t.Errorf("latest commit msg = %q, want it to contain %q",
			commits[0].Message, "attach bw-x foo.txt")
	}
}

// TestAttachSupportsNestedPaths confirms the helper stores the path
// verbatim, without normalization or basename flattening.
func TestAttachSupportsNestedPaths(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	if err := env.Store.Attach("bw-y", "dir/sub/file.ex", []byte("payload")); err != nil {
		t.Fatalf("Attach: %v", err)
	}
	env.CommitIntent("attach bw-y dir/sub/file.ex")

	got, err := env.Repo.TreeFS().ReadFile("attachments/bw-y/dir/sub/file.ex")
	if err != nil {
		t.Fatalf("ReadFile nested: %v", err)
	}
	if string(got) != "payload" {
		t.Errorf("nested contents = %q, want %q", got, "payload")
	}
}

// TestGetAttachmentRoundTrip covers the reader API: writing through the
// internal Attach helper + Commit, then reading back through the public
// GetAttachment, returns the same bytes.
func TestGetAttachmentRoundTrip(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	want := []byte("hello world")
	if err := env.Store.Attach("bw-rt", "notes.txt", want); err != nil {
		t.Fatalf("Attach: %v", err)
	}
	env.CommitIntent("attach bw-rt notes.txt")

	got, err := env.Store.GetAttachment("bw-rt", "notes.txt")
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("contents = %q, want %q", got, want)
	}
}

// TestGetAttachmentMissingReturnsSentinel covers the sentinel error
// contract: a missing attachment returns an error that matches
// ErrAttachmentNotFound via errors.Is.
func TestGetAttachmentMissingReturnsSentinel(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, err := env.Store.GetAttachment("bw-nope", "missing.bin")
	if err == nil {
		t.Fatal("expected error for missing attachment")
	}
	if !errors.Is(err, issue.ErrAttachmentNotFound) {
		t.Errorf("err = %v, want errors.Is(err, ErrAttachmentNotFound)", err)
	}
}
