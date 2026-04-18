package issue_test

import (
	"strings"
	"testing"

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
