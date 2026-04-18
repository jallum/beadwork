package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
