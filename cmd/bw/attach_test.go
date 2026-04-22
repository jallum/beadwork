package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

// TestCmdAttachBasic covers the happy path: running `bw attach <id>
// <file>` reads <file> from disk, stores it at
// attachments/<id>/<basename>, and writes a commit whose message is
// exactly `attach <id> <basename>\n`.
func TestCmdAttachBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	src := filepath.Join(env.Dir, "some.txt")
	if err := os.WriteFile(src, []byte("payload"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	var buf bytes.Buffer
	if err := cmdAttach(env.Store, []string{"bw-x", src}, PlainWriter(&buf)); err != nil {
		t.Fatalf("cmdAttach: %v", err)
	}

	got, err := env.Store.GetAttachment("bw-x", "some.txt")
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}
	if string(got) != "payload" {
		t.Errorf("contents = %q, want %q", got, "payload")
	}

	commits, err := env.Repo.AllCommits()
	if err != nil {
		t.Fatalf("AllCommits: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("no commits")
	}
	wantMsg := "attach bw-x some.txt"
	if strings.TrimRight(commits[0].Message, "\n") != wantMsg {
		t.Errorf("commit msg = %q, want exactly %q", commits[0].Message, wantMsg)
	}
}

// TestCmdAttachWithName covers the --name override: the stored path is
// taken verbatim from --name instead of from the source file's basename.
// Nested paths are allowed.
func TestCmdAttachWithName(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	src := filepath.Join(env.Dir, "local-name.bin")
	if err := os.WriteFile(src, []byte("blob"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	var buf bytes.Buffer
	if err := cmdAttach(env.Store, []string{"bw-y", src, "--name", "dir/sub/stored.bin"}, PlainWriter(&buf)); err != nil {
		t.Fatalf("cmdAttach: %v", err)
	}

	got, err := env.Store.GetAttachment("bw-y", "dir/sub/stored.bin")
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}
	if string(got) != "blob" {
		t.Errorf("contents = %q, want %q", got, "blob")
	}

	commits, _ := env.Repo.AllCommits()
	wantMsg := "attach bw-y dir/sub/stored.bin"
	if strings.TrimRight(commits[0].Message, "\n") != wantMsg {
		t.Errorf("commit msg = %q, want exactly %q", commits[0].Message, wantMsg)
	}
}

// TestCmdAttachUsage ensures the command surfaces a usage error when
// required positional args are missing.
func TestCmdAttachUsage(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	if err := cmdAttach(env.Store, []string{}, PlainWriter(&buf)); err == nil {
		t.Fatal("expected error for empty args")
	}
	if err := cmdAttach(env.Store, []string{"bw-x"}, PlainWriter(&buf)); err == nil {
		t.Fatal("expected error when file path missing")
	}
}

// TestCmdAttachMissingFile ensures the command fails clearly when the
// source file does not exist on disk.
func TestCmdAttachMissingFile(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdAttach(env.Store, []string{"bw-x", filepath.Join(env.Dir, "no-such-file")}, PlainWriter(&buf))
	if err == nil {
		t.Fatal("expected error for missing source file")
	}
}

// TestAttachSurvivesSyncConflictReplay is the end-to-end integration
// test for Step 7: make a tree with attachments via `bw attach`, force a
// diverged remote so `bw sync` falls back to conflict replay, and
// verify the attachment bytes survive untouched at their original path.
//
// The conflict is forced by mutating the same issue JSON on both sides
// (local adds an assignee; remote moves to in_progress). That mutation
// makes MergeCommit return false, the local ref is Reset to the remote
// tip, and Replay runs. During Replay the "attach" intent line must
// recover the blob from the ODB via Store.SourceHash — the very path
// exercised by this test.
func TestAttachSurvivesSyncConflictReplay(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()

	// Seed a shared issue and push so both sides agree on a base.
	shared, _ := env.Store.Create("Shared work", issue.CreateOpts{})
	env.Repo.Commit(`create ` + shared.ID + ` p2 task "Shared work"`)
	env.Repo.Sync(nil)

	// Remote side: update the shared issue and push.
	env2 := env.CloneEnv(bare)
	defer env2.Cleanup()

	env2.SwitchTo()
	statusIP := "in_progress"
	env2.Store.Update(shared.ID, issue.UpdateOpts{Status: &statusIP})
	env2.Repo.Commit("update " + shared.ID + " status=in_progress")
	env2.Repo.Sync(nil)

	// Local side: same issue gets a different assignee, AND we attach
	// a file with a distinctive payload that must survive the replay.
	env.SwitchTo()
	assignee := "agent-42"
	env.Store.Update(shared.ID, issue.UpdateOpts{Assignee: &assignee})
	env.Repo.Commit("update " + shared.ID + " assignee=agent-42")

	src := filepath.Join(env.Dir, "survives.txt")
	payload := []byte("attachment payload must survive replay")
	if err := os.WriteFile(src, payload, 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	var buf bytes.Buffer
	if err := cmdAttach(env.Store, []string{shared.ID, src}, PlainWriter(&buf)); err != nil {
		t.Fatalf("cmdAttach: %v", err)
	}

	// Sync — this must take the conflict-replay path.
	buf.Reset()
	if err := cmdSync(env.Store, []string{}, PlainWriter(&buf)); err != nil {
		t.Fatalf("cmdSync: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "replayed") {
		t.Fatalf("expected sync to replay; got output: %q", out)
	}

	// The attachment must be reachable from the fresh tree after replay.
	env.Store.ClearCache()
	got, err := env.Store.GetAttachment(shared.ID, "survives.txt")
	if err != nil {
		t.Fatalf("GetAttachment after replay: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("contents after replay = %q, want %q", got, payload)
	}
}
