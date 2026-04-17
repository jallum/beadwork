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

func TestFindRepoAtExplicitDir(t *testing.T) {
	dir := t.TempDir()

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	// No chdir — discover repo purely from the explicit path.
	r, err := repo.FindRepoAt(dir)
	if err != nil {
		t.Fatalf("FindRepoAt: %v", err)
	}
	if r.CWD != dir {
		t.Errorf("CWD = %q, want %q", r.CWD, dir)
	}
	if r.RepoDir() != dir {
		t.Errorf("RepoDir() = %q, want %q", r.RepoDir(), dir)
	}
}

func TestFindRepoAtNestedDir(t *testing.T) {
	dir := t.TempDir()

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	nested := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(nested, 0755)

	r, err := repo.FindRepoAt(nested)
	if err != nil {
		t.Fatalf("FindRepoAt: %v", err)
	}
	if r.CWD != nested {
		t.Errorf("CWD = %q, want %q", r.CWD, nested)
	}
	if r.RepoDir() != dir {
		t.Errorf("RepoDir() = %q, want %q", r.RepoDir(), dir)
	}
}

func TestFindRepoAtNotGit(t *testing.T) {
	dir := t.TempDir()

	_, err := repo.FindRepoAt(dir)
	if err == nil {
		t.Error("expected error for non-git directory")
	}
}

func TestFindRepoAtEmpty(t *testing.T) {
	// Empty string should fall back to cwd, same as FindRepo.
	dir, _ := filepath.EvalSymlinks(t.TempDir())

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	r, err := repo.FindRepoAt("")
	if err != nil {
		t.Fatalf("FindRepoAt empty: %v", err)
	}
	if r.RepoDir() != dir {
		t.Errorf("RepoDir() = %q, want %q", r.RepoDir(), dir)
	}
}

func TestFindRepoAtWithWorktreeConfigExtension(t *testing.T) {
	dir := t.TempDir()

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	// Simulate the state real git creates after `git config --worktree ...`:
	// format version bumped to 1, worktreeConfig extension enabled.
	gitRun(t, dir, "config", "core.repositoryFormatVersion", "1")
	gitRun(t, dir, "config", "extensions.worktreeConfig", "true")

	if _, err := repo.FindRepoAt(dir); err != nil {
		t.Fatalf("FindRepoAt with worktreeConfig extension: %v", err)
	}
}

func TestFindRepoAtFromWorktreeWithWorktreeConfig(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	main := filepath.Join(base, "main")
	if err := os.MkdirAll(main, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	gitRun(t, main, "init")
	gitRun(t, main, "config", "user.email", "test@test.com")
	gitRun(t, main, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(main, "README"), []byte("test"), 0644)
	gitRun(t, main, "add", ".")
	gitRun(t, main, "commit", "-m", "initial")

	wt := filepath.Join(base, "wt")
	gitRun(t, main, "worktree", "add", wt)

	// Enable worktreeConfig extension (mirrors what git does automatically
	// when a user runs `git config --worktree ...`).
	gitRun(t, main, "config", "core.repositoryFormatVersion", "1")
	gitRun(t, main, "config", "extensions.worktreeConfig", "true")
	gitRun(t, wt, "config", "--worktree", "core.sparseCheckout", "true")

	r, err := repo.FindRepoAt(wt)
	if err != nil {
		t.Fatalf("FindRepoAt from worktree: %v", err)
	}
	if r.RepoDir() != main {
		t.Errorf("RepoDir() = %q, want %q", r.RepoDir(), main)
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
	r.TreeFS().WriteFile("issues/test.json", []byte(`{"id":"test"}`))
	if err := r.Commit("test commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Commit with no changes should be a noop (not an error)
	if err := r.Commit("noop"); err != nil {
		t.Fatalf("Commit noop: %v", err)
	}
}
