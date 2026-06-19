package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func archiveFileExists(env *testutil.Env, id string) bool {
	return env.MarkerExists("archive/" + id + ".json")
}

func TestCmdArchiveClosed(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Done", issue.CreateOpts{})
	env.Store.Close(iss.ID, "")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdArchive(env.Store, []string{iss.ID}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdArchive: %v", err)
	}
	if !archiveFileExists(env, iss.ID) {
		t.Error("archive file missing")
	}
	if env.IssueFileExists(iss.ID) {
		t.Error("issues/<id>.json should be gone")
	}
}

func TestCmdArchiveOpenRefusedThenClose(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Active", issue.CreateOpts{})
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdArchive(env.Store, []string{iss.ID}, PlainWriter(&buf), nil)
	if err == nil {
		t.Fatal("expected error archiving an open issue")
	}
	if !strings.Contains(err.Error(), "--close") {
		t.Errorf("error should hint at --close, got: %v", err)
	}

	_, err = cmdArchive(env.Store, []string{iss.ID, "--close"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdArchive --close: %v", err)
	}
	if !archiveFileExists(env, iss.ID) {
		t.Error("archive file missing after --close")
	}
}

func TestCmdArchiveOpenDependentRefusedThenDetach(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	blocker, _ := env.Store.Create("Blocker", issue.CreateOpts{})
	blocked, _ := env.Store.Create("Blocked", issue.CreateOpts{})
	env.Store.Link(blocker.ID, blocked.ID)
	env.Store.Close(blocker.ID, "")
	env.Repo.Commit("setup")

	var buf bytes.Buffer
	_, err := cmdArchive(env.Store, []string{blocker.ID}, PlainWriter(&buf), nil)
	if err == nil {
		t.Fatal("expected refusal: open dependent")
	}
	if !strings.Contains(err.Error(), "--detach") {
		t.Errorf("error should hint at --detach, got: %v", err)
	}

	_, err = cmdArchive(env.Store, []string{blocker.ID, "--detach"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdArchive --detach: %v", err)
	}
	if !archiveFileExists(env, blocker.ID) {
		t.Error("archive file missing after --detach")
	}
}

func TestCmdArchiveBeforePreviewThenForce(t *testing.T) {
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
	env.Repo.Commit("setup")

	// Preview (no --force): reports the old one, mutates nothing.
	var preview bytes.Buffer
	_, err := cmdArchive(env.Store, []string{"--before", "2026-01-01"}, PlainWriter(&preview), nil)
	if err != nil {
		t.Fatalf("cmdArchive --before (preview): %v", err)
	}
	if !strings.Contains(preview.String(), "test-old") {
		t.Errorf("preview should mention test-old, got: %q", preview.String())
	}
	if archiveFileExists(env, "test-old") {
		t.Error("preview must not archive anything")
	}

	// Execute with --force.
	var run bytes.Buffer
	_, err = cmdArchive(env.Store, []string{"--before", "2026-01-01", "--force"}, PlainWriter(&run), nil)
	if err != nil {
		t.Fatalf("cmdArchive --before --force: %v", err)
	}
	if !archiveFileExists(env, "test-old") {
		t.Error("test-old should be archived")
	}
	if archiveFileExists(env, "test-new") {
		t.Error("test-new is newer than cutoff; must not be archived")
	}
}

func TestCmdArchiveNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdArchive(env.Store, []string{}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected usage error with no args")
	}
}

func TestCmdArchiveBeforeWithIDsRejected(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdArchive(env.Store, []string{"test-aaaa", "--before", "2026-01-01"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error combining --before with explicit IDs")
	}
}
