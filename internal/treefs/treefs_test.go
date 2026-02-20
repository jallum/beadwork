package treefs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// initTestRepo creates a bare git repo with one commit on the "beadwork" branch.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create a normal repo, commit, then we'll use it.
	run(t, dir, "git", "init", "-b", "main")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "initial")

	// Create orphan beadwork branch with a file
	run(t, dir, "git", "checkout", "--orphan", "beadwork")
	run(t, dir, "git", "rm", "-rf", ".")
	os.MkdirAll(filepath.Join(dir, "issues"), 0755)
	os.WriteFile(filepath.Join(dir, "issues", "test-1234.json"), []byte(`{"id":"test-1234"}`), 0644)
	os.WriteFile(filepath.Join(dir, ".bwconfig"), []byte("prefix=test\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "status", "open"), 0755)
	os.WriteFile(filepath.Join(dir, "status", "open", "test-1234"), []byte{}, 0644)
	os.WriteFile(filepath.Join(dir, "status", "open", ".gitkeep"), []byte{}, 0644)
	run(t, dir, "git", "add", "-A")
	run(t, dir, "git", "commit", "-m", "init beadwork")

	// Switch back to main
	run(t, dir, "git", "checkout", "main")

	return dir
}

// initEmptyRepo creates a git repo with no beadwork branch.
func initEmptyRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init", "-b", "main")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "initial")
	return dir
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
	if !containsStr(err.Error(), "conflict") {
		t.Fatalf("expected conflict error, got: %v", err)
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

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
