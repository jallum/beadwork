package intent_test

import (
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/jallum/beadwork/internal/intent"
	"github.com/jallum/beadwork/internal/testutil"
)

// commitHashFromString converts a hex hash string to plumbing.Hash.
// The env parameter is unused but retained for symmetry with other helpers.
func commitHashFromString(t *testing.T, _ *testutil.Env, hex string) plumbing.Hash {
	t.Helper()
	return plumbing.NewHash(hex)
}

// TestParseIntentAttach covers the parser-level grammar for the new
// attach verb. Tokens are a single-space-separated list: the verb,
// the ticket id, then the path-verbatim. The path may contain "/"
// and "." since it is simply the rest of the line.
func TestParseIntentAttach(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"attach bw-x foo.txt", []string{"attach", "bw-x", "foo.txt"}},
		{"attach bw-x dir/sub/file.ex", []string{"attach", "bw-x", "dir/sub/file.ex"}},
	}
	for _, tt := range tests {
		got := intent.ParseIntent(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("ParseIntent(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ParseIntent(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

// TestReplayAttachReusesPreResetBlob simulates the conflict-replay path.
// After a local attachment commit the ref is Reset() to wipe the tip
// (mimicking what repo.Sync does on conflict); the blob object is still
// in the ODB. The Store exposes the pre-reset commit hash via SourceHash
// so replayAttach can walk that tree, find the blob at
// attachments/<id>/<path>, and re-stage the entry at the same path.
func TestReplayAttachReusesPreResetBlob(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	if err := env.Store.Attach("bw-x", "foo.txt", []byte("hi")); err != nil {
		t.Fatalf("Attach: %v", err)
	}
	env.CommitIntent("attach bw-x foo.txt")

	// Capture the pre-reset hash and wipe the local ref to simulate the
	// "reset to remote" step of a conflict sync.
	preReset := env.Repo.TreeFS().RefHash()
	if preReset.IsZero() {
		t.Fatal("expected non-zero ref hash before reset")
	}

	// Reset to the root commit (the init commit). After this the
	// attachments/ tree path should no longer be reachable from the ref.
	commits, err := env.Repo.AllCommits()
	if err != nil || len(commits) < 2 {
		t.Fatalf("AllCommits: %v (got %d)", err, len(commits))
	}
	resetTo := commitHashFromString(t, env, commits[len(commits)-1].Hash)
	if err := env.Repo.TreeFS().Reset(resetTo); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	env.Store.ClearCache()

	// Confirm attachment is no longer reachable from the current tree.
	if _, err := env.Repo.TreeFS().ReadFile("attachments/bw-x/foo.txt"); err == nil {
		t.Fatal("attachment should not be reachable after reset")
	}

	// Tell the store where to find source blobs for attach replay.
	env.Store.SourceHash = preReset

	errs := intent.Replay(env.Store, []string{"attach bw-x foo.txt"})
	if len(errs) != 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	got, err := env.Repo.TreeFS().ReadFile("attachments/bw-x/foo.txt")
	if err != nil {
		t.Fatalf("ReadFile after replay: %v", err)
	}
	if string(got) != "hi" {
		t.Errorf("contents = %q, want %q", got, "hi")
	}
}

// TestReplayAttachMissingBlobFailsLoudly confirms that a missing blob
// produces an error — attachments are never silently dropped.
func TestReplayAttachMissingBlobFailsLoudly(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// No SourceHash set and no attach on disk: the blob simply does
	// not exist anywhere reachable.
	errs := intent.Replay(env.Store, []string{"attach bw-missing nowhere.bin"})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), "attach") {
		t.Errorf("error %q should mention attach verb", errs[0])
	}
}

// TestReplayAttachMultilineCommitMessage exercises the multi-attach case:
// a commit whose message has a primary intent line plus two attach lines.
// Replay must process each attach line, not just the first.
func TestReplayAttachMultilineCommitMessage(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Attach("bw-y", "a.txt", []byte("alpha"))
	env.Store.Attach("bw-y", "b.txt", []byte("beta"))
	env.CommitIntent("review bw-y\nattach bw-y a.txt\nattach bw-y b.txt")

	preReset := env.Repo.TreeFS().RefHash()

	// Reset to init commit.
	commits, _ := env.Repo.AllCommits()
	resetTo := commitHashFromString(t, env, commits[len(commits)-1].Hash)
	env.Repo.TreeFS().Reset(resetTo)
	env.Store.ClearCache()
	env.Store.SourceHash = preReset

	errs := intent.Replay(env.Store, []string{
		"review bw-y\nattach bw-y a.txt\nattach bw-y b.txt",
	})
	if len(errs) != 0 {
		t.Fatalf("Replay errors: %v", errs)
	}

	for _, tc := range []struct {
		path string
		want string
	}{
		{"attachments/bw-y/a.txt", "alpha"},
		{"attachments/bw-y/b.txt", "beta"},
	} {
		got, err := env.Repo.TreeFS().ReadFile(tc.path)
		if err != nil {
			t.Errorf("ReadFile %s: %v", tc.path, err)
			continue
		}
		if string(got) != tc.want {
			t.Errorf("%s = %q, want %q", tc.path, got, tc.want)
		}
	}
}
