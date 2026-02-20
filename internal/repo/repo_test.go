package repo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jallum/beadwork/internal/repo"
)

func TestFindRepoNotInitialized(t *testing.T) {
	dir := t.TempDir()

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	r, err := repo.FindRepo()
	if err != nil {
		t.Fatalf("FindRepo: %v", err)
	}
	if r.IsInitialized() {
		t.Error("should not be initialized before Init")
	}
	if r.Prefix != "" {
		t.Errorf("prefix = %q, want empty before init", r.Prefix)
	}
}

func TestFindRepoNotGit(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	_, err := repo.FindRepo()
	if err == nil {
		t.Error("expected error outside git repo")
	}
}

func TestInitInvalidPrefix(t *testing.T) {
	dir := t.TempDir()

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	r, _ := repo.FindRepo()
	err := r.Init("has space")
	if err == nil {
		t.Error("expected error for invalid prefix")
	}
}

func TestDerivePrefixFallback(t *testing.T) {
	// Create a repo in a directory named with only special chars
	base := t.TempDir()
	dir := filepath.Join(base, "...")
	os.Mkdir(dir, 0755)

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	r, _ := repo.FindRepo()
	if err := r.Init(""); err != nil {
		t.Fatalf("Init: %v", err)
	}
	// Should fallback to "bw" when dir name has no valid chars
	if r.Prefix != "bw" {
		t.Errorf("prefix = %q, want bw", r.Prefix)
	}
}

func TestCommitWithChanges(t *testing.T) {
	dir := t.TempDir()

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	r, _ := repo.FindRepo()
	r.Init("test")

	// Create a file and commit
	os.WriteFile(filepath.Join(r.WorkTree, "issues", "test.json"), []byte(`{"id":"test"}`), 0644)
	if err := r.Commit("test commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Commit with no changes should be a noop (not an error)
	if err := r.Commit("noop"); err != nil {
		t.Fatalf("Commit noop: %v", err)
	}
}
