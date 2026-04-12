package registry

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalRepoPathNormalRepo(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)

	got, err := CanonicalRepoPath(dir)
	if err != nil {
		t.Fatalf("CanonicalRepoPath: %v", err)
	}
	if got != dir {
		t.Errorf("got %q, want %q", got, dir)
	}
}

func TestCanonicalRepoPathSubdir(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)

	sub := filepath.Join(dir, "a", "b")
	os.MkdirAll(sub, 0755)

	got, err := CanonicalRepoPath(sub)
	if err != nil {
		t.Fatalf("CanonicalRepoPath: %v", err)
	}
	if got != dir {
		t.Errorf("got %q, want %q", got, dir)
	}
}

func TestCanonicalRepoPathWorktree(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "initial")

	wtDir := filepath.Join(t.TempDir(), "worktree")
	gitRun(t, dir, "worktree", "add", wtDir, "-b", "wt-branch")
	t.Cleanup(func() {
		gitRun(t, dir, "worktree", "remove", "--force", wtDir)
	})

	got, err := CanonicalRepoPath(wtDir)
	if err != nil {
		t.Fatalf("CanonicalRepoPath: %v", err)
	}

	// Resolve symlinks for comparison (macOS /private/tmp vs /tmp).
	wantReal, _ := filepath.EvalSymlinks(dir)
	gotReal, _ := filepath.EvalSymlinks(got)
	if gotReal != wantReal {
		t.Errorf("worktree resolved to %q, want %q", gotReal, wantReal)
	}
}

func TestCanonicalRepoPathNotGitRepo(t *testing.T) {
	dir := t.TempDir()
	got, err := CanonicalRepoPath(dir)
	if err != nil {
		t.Fatalf("CanonicalRepoPath: %v", err)
	}
	// For non-git dirs, returns the dir as-is.
	if !strings.HasPrefix(got, dir) {
		t.Errorf("got %q, want prefix %q", got, dir)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %s: %v", strings.Join(args, " "), out, err)
	}
}
