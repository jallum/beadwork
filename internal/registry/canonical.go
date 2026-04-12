package registry

import (
	"os"
	"path/filepath"
	"strings"
)

// CanonicalRepoPath resolves a directory to its main repository root.
// If dir is inside a git worktree, it follows the .git file → commondir
// chain to find the shared .git directory, then returns its parent.
// For normal repositories, it walks up looking for a .git directory.
func CanonicalRepoPath(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return findMainRepo(dir)
}

func findMainRepo(dir string) (string, error) {
	cur := dir
	for {
		dotGit := filepath.Join(cur, ".git")
		fi, err := os.Stat(dotGit)
		if err == nil {
			if fi.IsDir() {
				return cur, nil
			}
			// Worktree: .git is a file pointing to the real git dir.
			return resolveWorktreeRoot(dotGit)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return dir, nil // not a git repo; return as-is
		}
		cur = parent
	}
}

func resolveWorktreeRoot(dotGitFile string) (string, error) {
	data, err := os.ReadFile(dotGitFile)
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir: ") {
		return filepath.Dir(dotGitFile), nil
	}

	gitdir := strings.TrimPrefix(line, "gitdir: ")
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(filepath.Dir(dotGitFile), gitdir)
	}
	gitdir = filepath.Clean(gitdir)

	// Read commondir to find the shared .git directory.
	cdData, err := os.ReadFile(filepath.Join(gitdir, "commondir"))
	if err != nil {
		return filepath.Dir(gitdir), nil
	}

	commondir := strings.TrimSpace(string(cdData))
	if !filepath.IsAbs(commondir) {
		commondir = filepath.Join(gitdir, commondir)
	}
	commondir, err = filepath.Abs(commondir)
	if err != nil {
		return "", err
	}
	// The repo root is the parent of the .git directory.
	return filepath.Dir(commondir), nil
}
