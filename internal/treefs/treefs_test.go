package treefs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// initGitRepo creates a git repo with one commit on the default branch.
// Used as the base for both initTestRepo and initEmptyRepo.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "initial")
	return dir
}

// initTestRepo creates a git repo with one commit on the "beadwork" orphan
// branch, populated via TreeFS (no worktree checkout needed).
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := initGitRepo(t)

	// Use TreeFS to create the orphan beadwork branch
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open for init: %v", err)
	}
	tfs.WriteFile("issues/test-1234.json", []byte(`{"id":"test-1234"}`))
	tfs.WriteFile(".bwconfig", []byte("prefix=test\n"))
	tfs.WriteFile("status/open/test-1234", []byte{})
	tfs.WriteFile("status/open/.gitkeep", []byte{})
	if err := tfs.Commit("init beadwork"); err != nil {
		t.Fatalf("Commit beadwork branch: %v", err)
	}

	return dir
}

// initEmptyRepo creates a git repo with no beadwork branch.
func initEmptyRepo(t *testing.T) string {
	t.Helper()
	return initGitRepo(t)
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %s: %v", name, args, out, err)
	}
}

func TestOpenExistingBranch(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !tfs.HasRef() {
		t.Fatal("expected HasRef to be true")
	}
}

func TestOpenNonexistentBranch(t *testing.T) {
	dir := initEmptyRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if tfs.HasRef() {
		t.Fatal("expected HasRef to be false")
	}
}

func TestReadFileFromBase(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	data, err := tfs.ReadFile("issues/test-1234.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != `{"id":"test-1234"}` {
		t.Fatalf("unexpected content: %s", data)
	}
}

func TestReadFileNotFound(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	_, err = tfs.ReadFile("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestWriteAndReadFile(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := tfs.WriteFile("issues/test-5678.json", []byte(`{"id":"test-5678"}`)); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	data, err := tfs.ReadFile("issues/test-5678.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != `{"id":"test-5678"}` {
		t.Fatalf("unexpected content: %s", data)
	}
}

func TestRemoveFile(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := tfs.Remove("issues/test-1234.json"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	_, err = tfs.ReadFile("issues/test-1234.json")
	if err == nil {
		t.Fatal("expected error after Remove")
	}
}

func TestReadDir(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	entries, err := tfs.ReadDir("status/open")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}

	if !names[".gitkeep"] {
		t.Error("expected .gitkeep in status/open")
	}
	if !names["test-1234"] {
		t.Error("expected test-1234 in status/open")
	}
}

func TestReadDirRoot(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	entries, err := tfs.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir root: %v", err)
	}

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}

	if !names[".bwconfig"] {
		t.Error("expected .bwconfig at root")
	}
	if !names["issues"] {
		t.Error("expected issues dir at root")
	}
	if !names["status"] {
		t.Error("expected status dir at root")
	}
}

func TestReadDirWithOverlay(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Add a file
	tfs.WriteFile("status/open/test-aaaa", []byte{})
	// Remove a file
	tfs.Remove("status/open/test-1234")

	entries, err := tfs.ReadDir("status/open")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}

	if !names["test-aaaa"] {
		t.Error("expected test-aaaa after overlay write")
	}
	if names["test-1234"] {
		t.Error("test-1234 should have been removed by overlay")
	}
}

func TestStat(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// File
	fi, err := tfs.Stat("issues/test-1234.json")
	if err != nil {
		t.Fatalf("Stat file: %v", err)
	}
	if fi.IsDir() {
		t.Error("expected file, got dir")
	}
	if fi.Name() != "test-1234.json" {
		t.Errorf("expected name test-1234.json, got %s", fi.Name())
	}

	// Directory
	fi, err = tfs.Stat("issues")
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if !fi.IsDir() {
		t.Error("expected dir, got file")
	}

	// Nonexistent
	_, err = tfs.Stat("bogus")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestCommitCreatesGitObjects(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	oldRef := tfs.RefHash()

	if err := tfs.WriteFile("issues/test-new.json", []byte(`{"id":"test-new"}`)); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := tfs.Commit("add test-new"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Ref should have advanced
	if tfs.RefHash() == oldRef {
		t.Error("ref did not advance after commit")
	}

	// Verify we can read the new file from the committed tree
	data, err := tfs.ReadFile("issues/test-new.json")
	if err != nil {
		t.Fatalf("ReadFile after commit: %v", err)
	}
	if string(data) != `{"id":"test-new"}` {
		t.Fatalf("unexpected content after commit: %s", data)
	}

	// Verify the original file is still there
	data, err = tfs.ReadFile("issues/test-1234.json")
	if err != nil {
		t.Fatalf("ReadFile original after commit: %v", err)
	}
	if string(data) != `{"id":"test-1234"}` {
		t.Fatalf("original file changed after commit: %s", data)
	}
}

func TestCommitNoopWhenClean(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	oldRef := tfs.RefHash()
	if err := tfs.Commit("noop"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if tfs.RefHash() != oldRef {
		t.Error("ref should not advance on noop commit")
	}
}

func TestCommitOnNewBranch(t *testing.T) {
	dir := initEmptyRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if tfs.HasRef() {
		t.Fatal("expected no ref initially")
	}

	tfs.WriteFile(".bwconfig", []byte("prefix=test\n"))
	tfs.WriteFile("issues/.gitkeep", []byte{})

	if err := tfs.Commit("init beadwork"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	if !tfs.HasRef() {
		t.Fatal("expected HasRef after commit")
	}

	// Verify ref exists in git
	repo, _ := git.PlainOpen(dir)
	ref, err := repo.Reference(plumbing.ReferenceName("refs/heads/beadwork"), true)
	if err != nil {
		t.Fatalf("ref lookup: %v", err)
	}
	if ref.Hash().IsZero() {
		t.Fatal("ref hash is zero")
	}
}

func TestCASConflict(t *testing.T) {
	dir := initTestRepo(t)

	// Open two TreeFS instances against the same ref
	tfs1, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open tfs1: %v", err)
	}
	tfs2, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open tfs2: %v", err)
	}

	// tfs1 commits first
	tfs1.WriteFile("file1.txt", []byte("from tfs1"))
	if err := tfs1.Commit("commit from tfs1"); err != nil {
		t.Fatalf("tfs1 Commit: %v", err)
	}

	// tfs2 tries to commit — should fail because ref moved
	tfs2.WriteFile("file2.txt", []byte("from tfs2"))
	err = tfs2.Commit("commit from tfs2")
	if err == nil {
		t.Fatal("expected CAS conflict error")
	}
	if !errors.Is(err, ErrRefMoved) {
		t.Fatalf("expected ErrRefMoved, got: %v", err)
	}
}

func TestMultipleCommits(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// First commit
	tfs.WriteFile("file1.txt", []byte("hello"))
	if err := tfs.Commit("add file1"); err != nil {
		t.Fatalf("Commit 1: %v", err)
	}

	// Second commit
	tfs.WriteFile("file2.txt", []byte("world"))
	if err := tfs.Commit("add file2"); err != nil {
		t.Fatalf("Commit 2: %v", err)
	}

	// Both files should be readable
	if _, err := tfs.ReadFile("file1.txt"); err != nil {
		t.Fatalf("ReadFile file1: %v", err)
	}
	if _, err := tfs.ReadFile("file2.txt"); err != nil {
		t.Fatalf("ReadFile file2: %v", err)
	}
}

func TestRemoveAndCommit(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	tfs.Remove("issues/test-1234.json")
	if err := tfs.Commit("remove test-1234"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// File should be gone from committed tree
	_, err = tfs.ReadFile("issues/test-1234.json")
	if err == nil {
		t.Fatal("expected file to be removed after commit")
	}
}

func TestMkdirAll(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := tfs.MkdirAll("labels/bug"); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	fi, err := tfs.Stat("labels/bug")
	if err != nil {
		t.Fatalf("Stat after MkdirAll: %v", err)
	}
	if !fi.IsDir() {
		t.Error("expected directory")
	}
}

func TestWriteFileCreatesParentDirs(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	tfs.WriteFile("blocks/a/b", []byte{})

	fi, err := tfs.Stat("blocks/a")
	if err != nil {
		t.Fatalf("Stat parent after WriteFile: %v", err)
	}
	if !fi.IsDir() {
		t.Error("expected parent to be a directory")
	}
}

func TestCommitRespectsClockField(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	fixedTime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	tfs.Clock = func() time.Time { return fixedTime }

	tfs.WriteFile("clock-test.txt", []byte("hello"))
	if err := tfs.Commit("clock test"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	commits, err := tfs.AllCommits()
	if err != nil {
		t.Fatalf("AllCommits: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("no commits found")
	}
	// Most recent commit should have our fixed time
	if !commits[0].Time.Equal(fixedTime) {
		t.Errorf("commit time = %v, want %v", commits[0].Time, fixedTime)
	}
}

func TestCommitRespectsBWClockEnv(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	t.Setenv("BW_CLOCK", "2025-06-15T10:30:00Z")

	tfs.WriteFile("env-clock-test.txt", []byte("hello"))
	if err := tfs.Commit("env clock test"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	commits, err := tfs.AllCommits()
	if err != nil {
		t.Fatalf("AllCommits: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("no commits found")
	}
	expected := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	if !commits[0].Time.Equal(expected) {
		t.Errorf("commit time = %v, want %v", commits[0].Time, expected)
	}
}

// --- Snapshot-consistency tests ---
//
// These tests verify that operations which move the underlying ref also
// advance the TreeFS in-memory snapshot (baseRef). If any of these
// regress, a subsequent Commit() on the same TreeFS instance will fail
// with ErrRefMoved, surfacing to users as
//   "commit failed: ref moved: ref refs/heads/beadwork (expected X, got Y)"

func TestSnapshotConsistentAfterReset(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Create a second commit so we have somewhere to Reset to.
	tfs.WriteFile("a.txt", []byte("a"))
	if err := tfs.Commit("a"); err != nil {
		t.Fatalf("Commit a: %v", err)
	}
	target := tfs.RefHash()

	// Make another commit on top, then Reset back to the earlier target.
	tfs.WriteFile("b.txt", []byte("b"))
	if err := tfs.Commit("b"); err != nil {
		t.Fatalf("Commit b: %v", err)
	}
	if err := tfs.Reset(target); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	// A subsequent Commit must not see a stale baseRef.
	tfs.WriteFile("c.txt", []byte("c"))
	if err := tfs.Commit("c"); err != nil {
		t.Fatalf("Commit after Reset: %v", err)
	}
}

func TestSnapshotConsistentAfterRefresh(t *testing.T) {
	dir := initTestRepo(t)
	tfs1, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open tfs1: %v", err)
	}
	tfs2, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open tfs2: %v", err)
	}

	// tfs1 commits, advancing the ref behind tfs2's back.
	tfs1.WriteFile("from-tfs1.txt", []byte("x"))
	if err := tfs1.Commit("tfs1 commit"); err != nil {
		t.Fatalf("tfs1 Commit: %v", err)
	}

	// tfs2 refreshes, then commits — must succeed.
	if err := tfs2.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	tfs2.WriteFile("from-tfs2.txt", []byte("y"))
	if err := tfs2.Commit("tfs2 commit"); err != nil {
		t.Fatalf("Commit after Refresh: %v", err)
	}
}

func TestSnapshotConsistentAfterMergeCommit(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Establish a shared base.
	tfs.WriteFile("base.txt", []byte("base"))
	if err := tfs.Commit("base"); err != nil {
		t.Fatalf("Commit base: %v", err)
	}
	baseHash := tfs.RefHash()

	// Build a "remote" branch one commit ahead.
	if err := tfs.SetRef("refs/remotes/origin/beadwork", baseHash); err != nil {
		t.Fatalf("SetRef remote: %v", err)
	}
	// Make a divergent local commit, then reset remote ref forward via a
	// second TreeFS so the two refs diverge from the same base.
	tfs2, err := Open(dir, "refs/remotes/origin/beadwork")
	if err != nil {
		t.Fatalf("Open remote tfs: %v", err)
	}
	tfs2.WriteFile("remote-only.txt", []byte("r"))
	if err := tfs2.Commit("remote work"); err != nil {
		t.Fatalf("remote Commit: %v", err)
	}
	remoteHash := tfs2.RefHash()

	tfs.WriteFile("local-only.txt", []byte("l"))
	if err := tfs.Commit("local work"); err != nil {
		t.Fatalf("local Commit: %v", err)
	}
	localHash := tfs.RefHash()

	merged, err := tfs.MergeCommit(localHash, remoteHash, []string{"local work"})
	if err != nil {
		t.Fatalf("MergeCommit: %v", err)
	}
	if !merged {
		t.Fatal("expected non-conflicting merge")
	}

	// Subsequent Commit must not be stale.
	tfs.WriteFile("after-merge.txt", []byte("m"))
	if err := tfs.Commit("after merge"); err != nil {
		t.Fatalf("Commit after MergeCommit: %v", err)
	}
}

// SetRef must keep the in-memory snapshot consistent when it targets the
// ref this TreeFS is tracking. Otherwise callers that follow SetRef with
// a Commit (or with reads gated by maybeRefresh's overlay guard) see a
// silent ErrRefMoved.
func TestSnapshotConsistentAfterSetRefOnTrackedRef(t *testing.T) {
	dir := initTestRepo(t)
	tfs, err := Open(dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Create a second commit, capture its hash, then rewind via SetRef
	// on the tracked ref.
	original := tfs.RefHash()
	tfs.WriteFile("x.txt", []byte("x"))
	if err := tfs.Commit("x"); err != nil {
		t.Fatalf("Commit x: %v", err)
	}
	if err := tfs.SetRef("refs/heads/beadwork", original); err != nil {
		t.Fatalf("SetRef: %v", err)
	}

	tfs.WriteFile("y.txt", []byte("y"))
	if err := tfs.Commit("y"); err != nil {
		t.Fatalf("Commit after SetRef on tracked ref: %v", err)
	}
}
