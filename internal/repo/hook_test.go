package repo_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/j5n/beadwork/internal/repo"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s: %v", args, out, err)
	}
}

func TestInitWithPreCommitHook(t *testing.T) {
	dir := t.TempDir()

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	// Install a pre-commit hook that always fails
	hookDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(hookDir, 0755)
	hook := "#!/bin/sh\nexit 1\n"
	os.WriteFile(filepath.Join(hookDir, "pre-commit"), []byte(hook), 0755)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	r, err := repo.FindRepo()
	if err != nil {
		t.Fatalf("FindRepo: %v", err)
	}
	if err := r.Init("test"); err != nil {
		t.Fatalf("Init should succeed despite pre-commit hook: %v", err)
	}
}

func TestCommitWithPreCommitHook(t *testing.T) {
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
	if err := r.Init("test"); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Install a pre-commit hook that always fails AFTER init
	hookDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(hookDir, 0755)
	hook := "#!/bin/sh\nexit 1\n"
	os.WriteFile(filepath.Join(hookDir, "pre-commit"), []byte(hook), 0755)

	// Commit should still work (beadwork uses --no-verify)
	os.WriteFile(filepath.Join(r.WorkTree, "issues", "dummy.json"), []byte("{}"), 0644)
	if err := r.Commit("test commit"); err != nil {
		t.Fatalf("Commit should succeed despite pre-commit hook: %v", err)
	}
}
