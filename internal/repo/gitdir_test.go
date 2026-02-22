package repo

import (
	"os"
	"path/filepath"
	"testing"
)

// realPath resolves symlinks so tests work on macOS where /var -> /private/var.
func realPath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", p, err)
	}
	return r
}

func TestFindGitDirNormalRepo(t *testing.T) {
	dir := realPath(t, t.TempDir())

	// Create a .git directory (simulating a normal repo)
	gitDir := filepath.Join(dir, ".git")
	os.Mkdir(gitDir, 0755)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	got, err := findGitDir()
	if err != nil {
		t.Fatalf("findGitDir: %v", err)
	}
	if got != gitDir {
		t.Errorf("findGitDir() = %q, want %q", got, gitDir)
	}
}

func TestFindGitDirNestedDirectory(t *testing.T) {
	dir := realPath(t, t.TempDir())

	// Create a .git directory at the root
	gitDir := filepath.Join(dir, ".git")
	os.Mkdir(gitDir, 0755)

	// Create nested subdirectories
	nested := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(nested, 0755)

	orig, _ := os.Getwd()
	os.Chdir(nested)
	defer os.Chdir(orig)

	got, err := findGitDir()
	if err != nil {
		t.Fatalf("findGitDir: %v", err)
	}
	if got != gitDir {
		t.Errorf("findGitDir() = %q, want %q", got, gitDir)
	}
}

func TestFindGitDirNoRepo(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	_, err := findGitDir()
	if err == nil {
		t.Error("expected error outside git repo")
	}
}

func TestFindGitDirWorktreeGitFile(t *testing.T) {
	dir := realPath(t, t.TempDir())

	// Create the main repo's .git directory (the common dir)
	mainGitDir := filepath.Join(dir, "main-repo", ".git")
	os.MkdirAll(mainGitDir, 0755)

	// Create the worktree directory with a .git file
	wtDir := filepath.Join(dir, "worktree")
	os.MkdirAll(wtDir, 0755)

	// The worktree .git file points to a worktree-specific gitdir
	wtGitDir := filepath.Join(mainGitDir, "worktrees", "my-worktree")
	os.MkdirAll(wtGitDir, 0755)

	// Write .git file in the worktree
	gitFileContent := "gitdir: " + wtGitDir + "\n"
	os.WriteFile(filepath.Join(wtDir, ".git"), []byte(gitFileContent), 0644)

	// Write commondir file in the worktree-specific gitdir
	// commondir is relative to the worktree gitdir
	os.WriteFile(filepath.Join(wtGitDir, "commondir"), []byte("../..\n"), 0644)

	orig, _ := os.Getwd()
	os.Chdir(wtDir)
	defer os.Chdir(orig)

	got, err := findGitDir()
	if err != nil {
		t.Fatalf("findGitDir: %v", err)
	}
	// Should resolve to the main .git directory, not the worktree-specific one
	want, _ := filepath.Abs(mainGitDir)
	got, _ = filepath.Abs(got)
	if got != want {
		t.Errorf("findGitDir() = %q, want %q", got, want)
	}
}

func TestFindGitDirWorktreeNestedDirectory(t *testing.T) {
	dir := realPath(t, t.TempDir())

	// Create the main repo's .git directory
	mainGitDir := filepath.Join(dir, "main-repo", ".git")
	os.MkdirAll(mainGitDir, 0755)

	// Create worktree directory with nested subdirs
	wtDir := filepath.Join(dir, "worktree")
	os.MkdirAll(filepath.Join(wtDir, "src", "pkg"), 0755)

	// The worktree .git file points to a worktree-specific gitdir
	wtGitDir := filepath.Join(mainGitDir, "worktrees", "my-wt")
	os.MkdirAll(wtGitDir, 0755)

	// Write .git file
	os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: "+wtGitDir+"\n"), 0644)

	// Write commondir file
	os.WriteFile(filepath.Join(wtGitDir, "commondir"), []byte("../..\n"), 0644)

	orig, _ := os.Getwd()
	os.Chdir(filepath.Join(wtDir, "src", "pkg"))
	defer os.Chdir(orig)

	got, err := findGitDir()
	if err != nil {
		t.Fatalf("findGitDir: %v", err)
	}
	want, _ := filepath.Abs(mainGitDir)
	got, _ = filepath.Abs(got)
	if got != want {
		t.Errorf("findGitDir() = %q, want %q", got, want)
	}
}

func TestFindGitDirWorktreeAbsoluteCommondir(t *testing.T) {
	dir := realPath(t, t.TempDir())

	// Create the main repo's .git directory
	mainGitDir := filepath.Join(dir, "main-repo", ".git")
	os.MkdirAll(mainGitDir, 0755)

	// Create worktree
	wtDir := filepath.Join(dir, "worktree")
	os.MkdirAll(wtDir, 0755)

	wtGitDir := filepath.Join(mainGitDir, "worktrees", "my-wt")
	os.MkdirAll(wtGitDir, 0755)

	os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: "+wtGitDir+"\n"), 0644)

	// commondir with an absolute path
	os.WriteFile(filepath.Join(wtGitDir, "commondir"), []byte(mainGitDir+"\n"), 0644)

	orig, _ := os.Getwd()
	os.Chdir(wtDir)
	defer os.Chdir(orig)

	got, err := findGitDir()
	if err != nil {
		t.Fatalf("findGitDir: %v", err)
	}
	want, _ := filepath.Abs(mainGitDir)
	got, _ = filepath.Abs(got)
	if got != want {
		t.Errorf("findGitDir() = %q, want %q", got, want)
	}
}
