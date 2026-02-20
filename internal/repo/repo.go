package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

	// Create directory skeleton (always, ensures dirs exist locally)
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

	// Only commit skeleton if this is a new branch (not cloned from remote)
	if !remoteExists {
		if err := os.WriteFile(filepath.Join(r.WorkTree, ".bwconfig"), []byte("prefix="+prefix+"\n"), 0644); err != nil {
			return err
		}
		for _, d := range dirs {
			keepFile := filepath.Join(r.WorkTree, d, ".gitkeep")
			os.WriteFile(keepFile, []byte{}, 0644)
		}
		if _, err := r.gitWt("add", "-A"); err != nil {
			return err
		}
		if _, err := r.gitWt("commit", "--no-verify", "-m", "init beadwork"); err != nil {
			return err
		}
	} else {
		// Read prefix from existing config
		r.Prefix = r.readPrefix()
	}

	r.initialized = true
	return nil
}

func (r *Repo) Commit(message string) error {
	if _, err := r.gitWt("add", "-A"); err != nil {
		return err
	}
	out, err := r.gitWt("commit", "--no-verify", "-m", message)
	if err != nil && strings.Contains(out, "nothing to commit") {
		return nil
	}
	return err
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
	// We need a commit to return to
	origRef, err := r.git("rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("repo must have at least one commit before initializing beadwork")
	}
	origRef = strings.TrimSpace(origRef)

	// Remember which branch we're on
	origBranch, _ := r.git("rev-parse", "--abbrev-ref", "HEAD")
	origBranch = strings.TrimSpace(origBranch)

	if _, err := r.git("checkout", "--orphan", BranchName); err != nil {
		return err
	}
	// Remove all files from the orphan branch index
	r.git("rm", "-rf", "--quiet", ".")
	if _, err := r.git("commit", "--allow-empty", "--no-verify", "-m", "init beadwork branch"); err != nil {
		r.git("checkout", origRef)
		return err
	}

	// Return to previous branch, fall back to detached HEAD at the commit
	if origBranch != "" && origBranch != "HEAD" {
		if _, err := r.git("checkout", origBranch); err != nil {
			r.git("checkout", origRef)
		}
	} else {
		r.git("checkout", origRef)
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

// GetConfig reads a single key from .bwconfig.
func (r *Repo) GetConfig(key string) (string, bool) {
	cfg := r.ListConfig()
	val, ok := cfg[key]
	return val, ok
}

// SetConfig writes or updates a key in .bwconfig, preserving other entries.
func (r *Repo) SetConfig(key, value string) error {
	path := filepath.Join(r.WorkTree, ".bwconfig")
	cfg := r.ListConfig()
	cfg[key] = value

	var lines []string
	for k, v := range cfg {
		lines = append(lines, k+"="+v)
	}
	sort.Strings(lines)
	data := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(data), 0644)
}

// ListConfig reads all key=value pairs from .bwconfig.
func (r *Repo) ListConfig() map[string]string {
	cfg := make(map[string]string)
	data, err := os.ReadFile(filepath.Join(r.WorkTree, ".bwconfig"))
	if err != nil {
		return cfg
	}
	for _, line := range strings.Split(string(data), "\n") {
		if i := strings.Index(line, "="); i > 0 {
			cfg[line[:i]] = line[i+1:]
		}
	}
	return cfg
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

func (r *Repo) hasRemote() bool {
	out, err := r.git("remote")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// Sync fetches from origin, rebases local commits, and pushes.
// Returns the list of intents that were replayed (if any), or nil if clean.
func (r *Repo) Sync() (status string, replayed []string, err error) {
	if !r.hasRemote() {
		return "no remote configured", nil, nil
	}

	// Check if remote branch exists
	if !r.remoteBranchExists() {
		// No remote branch — just push
		if _, err := r.gitWt("push", "-u", "origin", BranchName); err != nil {
			return "", nil, fmt.Errorf("push failed: %w", err)
		}
		return "pushed", nil, nil
	}

	// Fetch
	if _, err := r.gitWt("fetch", "origin", BranchName); err != nil {
		return "", nil, fmt.Errorf("fetch failed: %w", err)
	}

	// Check if we have local commits ahead of origin
	localCommits, err := r.localOnlyCommits()
	if err != nil {
		return "", nil, err
	}

	if len(localCommits) == 0 {
		// Nothing local to push, just fast-forward
		r.gitWt("merge", "--ff-only", "origin/"+BranchName)
		return "up to date", nil, nil
	}

	// Try rebase
	_, rebaseErr := r.gitWt("rebase", "origin/"+BranchName)
	if rebaseErr == nil {
		// Clean rebase — push
		if _, err := r.gitWt("push", "origin", BranchName); err != nil {
			return "", nil, fmt.Errorf("push after rebase failed: %w", err)
		}
		return "rebased and pushed", nil, nil
	}

	// Dirty rebase — abort and replay intents
	r.gitWt("rebase", "--abort")

	// Collect intents from local-only commits
	intents := make([]string, 0, len(localCommits))
	for _, commit := range localCommits {
		intents = append(intents, commit.Message)
	}

	// Reset to origin
	if _, err := r.gitWt("reset", "--hard", "origin/"+BranchName); err != nil {
		return "", nil, fmt.Errorf("reset failed: %w", err)
	}

	return "needs replay", intents, nil
}

type CommitInfo struct {
	Hash    string
	Message string
}

func (r *Repo) localOnlyCommits() ([]CommitInfo, error) {
	// Get commits on beadwork that aren't on origin/beadwork
	out, err := r.gitWt("log", "origin/"+BranchName+".."+BranchName, "--format=%H %s", "--reverse")
	if err != nil {
		// If origin/beadwork doesn't exist yet, all commits are local
		out, err = r.gitWt("log", BranchName, "--format=%H %s", "--reverse")
		if err != nil {
			return nil, err
		}
	}

	var commits []CommitInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			commits = append(commits, CommitInfo{Hash: parts[0], Message: parts[1]})
		}
	}
	return commits, nil
}

// Push pushes the beadwork branch to origin.
func (r *Repo) Push() error {
	_, err := r.gitWt("push", "origin", BranchName)
	return err
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
