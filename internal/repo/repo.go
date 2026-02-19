package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const BranchName = "beadwork"

type Repo struct {
	GitDir      string // path to .git/
	WorkTree    string // path to .git/beadwork/ (the worktree)
	Prefix      string
	initialized bool
}

func FindRepo() (*Repo, error) {
	out, err := execGit(".", "rev-parse", "--git-dir")
	if err != nil {
		return nil, fmt.Errorf("not a git repository")
	}
	gitDir, err := filepath.Abs(strings.TrimSpace(out))
	if err != nil {
		return nil, err
	}
	r := &Repo{
		GitDir:   gitDir,
		WorkTree: filepath.Join(gitDir, BranchName),
	}
	if _, err := os.Stat(r.WorkTree); err == nil {
		r.initialized = true
		r.Prefix = r.readPrefix()
	}
	return r, nil
}

func (r *Repo) IsInitialized() bool {
	return r.initialized
}

func (r *Repo) Init(prefix string) error {
	if r.initialized {
		return fmt.Errorf("beadwork already initialized")
	}

	// Check if remote branch exists
	remoteExists := r.remoteBranchExists()
	localExists := r.localBranchExists()

	if remoteExists {
		// Path A: branch exists on remote, just set up worktree
		if _, err := r.git("fetch", "origin", BranchName); err != nil {
			return fmt.Errorf("fetch failed: %w", err)
		}
		if !localExists {
			if _, err := r.git("branch", BranchName, "origin/"+BranchName); err != nil {
				return fmt.Errorf("create local branch: %w", err)
			}
		}
	} else if !localExists {
		// Path B: no branch anywhere, create orphan
		if err := r.createOrphanBranch(); err != nil {
			return fmt.Errorf("create orphan branch: %w", err)
		}
	}

	// Create worktree
	if _, err := r.git("worktree", "add", r.WorkTree, BranchName); err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}

	// Create directory skeleton
	dirs := []string{
		"issues",
		"status/open",
		"status/in_progress",
		"status/closed",
		"labels",
		"blocks",
		"parent",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(r.WorkTree, d), 0755); err != nil {
			return err
		}
	}

	// Write prefix config
	if prefix == "" {
		prefix = r.derivePrefix()
	}
	r.Prefix = prefix
	if err := os.WriteFile(filepath.Join(r.WorkTree, ".bwconfig"), []byte("prefix="+prefix+"\n"), 0644); err != nil {
		return err
	}

	// Commit initial structure with .gitkeep files to preserve empty dirs
	for _, d := range dirs {
		keepFile := filepath.Join(r.WorkTree, d, ".gitkeep")
		os.WriteFile(keepFile, []byte{}, 0644)
	}

	if _, err := r.gitWt("add", "-A"); err != nil {
		return err
	}
	if _, err := r.gitWt("commit", "-m", "init beadwork"); err != nil {
		return err
	}

	r.initialized = true
	return nil
}

func (r *Repo) Commit(message string) error {
	if _, err := r.gitWt("add", "-A"); err != nil {
		return err
	}
	if _, err := r.gitWt("commit", "-m", message); err != nil {
		return err
	}
	return nil
}

func (r *Repo) remoteBranchExists() bool {
	out, err := r.git("ls-remote", "--heads", "origin", BranchName)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

func (r *Repo) localBranchExists() bool {
	_, err := r.git("rev-parse", "--verify", BranchName)
	return err == nil
}

func (r *Repo) createOrphanBranch() error {
	// Get current branch to return to
	origBranch, err := r.git("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		origBranch = "main"
	}
	origBranch = strings.TrimSpace(origBranch)

	if _, err := r.git("checkout", "--orphan", BranchName); err != nil {
		return err
	}
	// Remove all files from the orphan branch index
	r.git("rm", "-rf", "--quiet", ".")
	if _, err := r.git("commit", "--allow-empty", "-m", "init beadwork branch"); err != nil {
		r.git("checkout", origBranch)
		return err
	}
	if _, err := r.git("checkout", origBranch); err != nil {
		return fmt.Errorf("failed to return to %s: %w", origBranch, err)
	}
	return nil
}

func (r *Repo) derivePrefix() string {
	// Use the repo directory name as default prefix
	parent := filepath.Dir(r.GitDir)
	name := filepath.Base(parent)
	// Shorten to max 6 chars
	if len(name) > 6 {
		name = name[:6]
	}
	return strings.ToLower(name)
}

func (r *Repo) readPrefix() string {
	data, err := os.ReadFile(filepath.Join(r.WorkTree, ".bwconfig"))
	if err != nil {
		return r.derivePrefix()
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "prefix=") {
			return strings.TrimPrefix(line, "prefix=")
		}
	}
	return r.derivePrefix()
}

// git runs a git command from the main repo.
func (r *Repo) git(args ...string) (string, error) {
	return execGit(filepath.Dir(r.GitDir), args...)
}

// gitWt runs a git command from the beadwork worktree.
func (r *Repo) gitWt(args ...string) (string, error) {
	return execGit(r.WorkTree, args...)
}

func execGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(out)), err)
	}
	return string(out), nil
}
