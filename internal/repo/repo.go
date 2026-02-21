package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/jallum/beadwork/internal/treefs"
)

const BranchName = "beadwork"

// CurrentVersion is the highest repo schema version this binary understands.
const CurrentVersion = 2

const refLocal = "refs/heads/" + BranchName
const refRemote = "refs/remotes/origin/" + BranchName

type Repo struct {
	GitDir      string
	Prefix      string
	tfs         *treefs.TreeFS
	initialized bool
}

func FindRepo() (*Repo, error) {
	gitDir, err := findGitDir()
	if err != nil {
		return nil, fmt.Errorf("not a git repository")
	}

	repoDir := filepath.Dir(gitDir)
	goRepo, err := git.PlainOpen(repoDir)
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	tfs, err := treefs.OpenFromRepo(goRepo, refLocal)
	if err != nil {
		return nil, fmt.Errorf("open treefs: %w", err)
	}

	r := &Repo{
		GitDir: gitDir,
		tfs:    tfs,
	}

	if tfs.HasRef() {
		r.initialized = true
		r.Prefix = r.readPrefix()
	}
	return r, nil
}

// TreeFS returns the underlying TreeFS. Used by Store and tests.
func (r *Repo) TreeFS() *treefs.TreeFS {
	return r.tfs
}

func (r *Repo) IsInitialized() bool {
	return r.initialized
}

var prefixRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

func ValidatePrefix(prefix string) error {
	if prefix == "" {
		return nil
	}
	if len(prefix) > 16 {
		return fmt.Errorf("prefix too long (max 16 characters)")
	}
	if !prefixRe.MatchString(prefix) {
		return fmt.Errorf("prefix must be alphanumeric (hyphens and underscores allowed)")
	}
	return nil
}

// ForceReinit destroys the existing beadwork branch and reinitializes.
func (r *Repo) ForceReinit(prefix string) error {
	if err := ValidatePrefix(prefix); err != nil {
		return err
	}

	// Clean up any legacy worktree at .git/beadwork/
	legacyWt := filepath.Join(r.GitDir, BranchName)
	if _, err := os.Stat(legacyWt); err == nil {
		execGit(filepath.Dir(r.GitDir), "worktree", "remove", "--force", legacyWt)
		os.RemoveAll(legacyWt)
	}

	// Delete local branch ref
	if r.localBranchExists() {
		r.tfs.DeleteRef(refLocal)
	}

	r.initialized = false

	// Re-open TreeFS (old ref is gone)
	tfs, err := treefs.OpenFromRepo(r.tfs.Repo(), refLocal)
	if err != nil {
		return fmt.Errorf("reopen treefs: %w", err)
	}
	r.tfs = tfs

	return r.Init(prefix)
}

func (r *Repo) Init(prefix string) error {
	if r.initialized {
		return fmt.Errorf("beadwork already initialized")
	}

	if err := ValidatePrefix(prefix); err != nil {
		return err
	}

	remoteExists := r.remoteBranchExists()
	localExists := r.localBranchExists()

	if remoteExists {
		// Fetch remote branch
		refSpec := config.RefSpec(fmt.Sprintf("+%s:%s", refLocal, refRemote))
		if err := r.fetch("origin", refSpec); err != nil {
			return fmt.Errorf("fetch failed: %w", err)
		}

		if !localExists {
			// Create local branch from remote
			remoteHash, err := r.tfs.LookupRef(refRemote)
			if err != nil {
				return fmt.Errorf("lookup remote ref: %w", err)
			}
			if err := r.tfs.SetRef(refLocal, remoteHash); err != nil {
				return fmt.Errorf("create local branch: %w", err)
			}
		}

		// Reopen TreeFS to pick up the new ref
		tfs, err := treefs.OpenFromRepo(r.tfs.Repo(), refLocal)
		if err != nil {
			return fmt.Errorf("reopen treefs: %w", err)
		}
		r.tfs = tfs
		r.Prefix = r.readPrefix()
	} else if !localExists {
		// No branch anywhere — TreeFS will create it on first Commit
		// (baseRef is zero, so Commit creates the ref)
		if prefix == "" {
			prefix = r.derivePrefix()
		}
		r.Prefix = prefix

		// Write skeleton
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
			r.tfs.WriteFile(d+"/.gitkeep", []byte{})
		}
		r.tfs.WriteFile(".bwconfig", []byte("prefix="+prefix+"\nversion="+strconv.Itoa(CurrentVersion)+"\n"))

		if err := r.tfs.Commit("init beadwork"); err != nil {
			return fmt.Errorf("init commit: %w", err)
		}
	}

	if prefix != "" {
		r.Prefix = prefix
	}
	r.initialized = true
	return nil
}

// AllCommits returns all commits on the beadwork branch, newest-first.
func (r *Repo) AllCommits() ([]treefs.CommitInfo, error) {
	return r.tfs.AllCommits()
}

func (r *Repo) Commit(message string) error {
	return r.tfs.Commit(message)
}

func (r *Repo) remoteBranchExists() bool {
	_, err := r.tfs.LookupRef(refRemote)
	if err == nil {
		return true
	}
	// Also check via ls-remote for freshly cloned repos where we haven't
	// fetched yet but the remote has the branch
	has, _ := r.tfs.HasRemotes()
	if !has {
		return false
	}
	// Use git CLI for ls-remote since go-git remote.List requires network
	out, err := execGit(filepath.Dir(r.GitDir), "ls-remote", "--heads", "origin", BranchName)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

func (r *Repo) localBranchExists() bool {
	_, err := r.tfs.LookupRef(refLocal)
	return err == nil
}

func (r *Repo) hasRemote() bool {
	has, err := r.tfs.HasRemotes()
	if err != nil {
		return false
	}
	return has
}

func (r *Repo) derivePrefix() string {
	parent := filepath.Dir(r.GitDir)
	name := filepath.Base(parent)
	var clean []byte
	for i := range name {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			clean = append(clean, c)
		}
	}
	if len(clean) == 0 {
		return "bw"
	}
	if len(clean) > 8 {
		clean = clean[:8]
	}
	return string(clean)
}

// Version returns the repo schema version (0 if unset or invalid).
func (r *Repo) Version() int {
	v, ok := r.GetConfig("version")
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

// GetConfig reads a single key from .bwconfig.
func (r *Repo) GetConfig(key string) (string, bool) {
	cfg := r.ListConfig()
	val, ok := cfg[key]
	return val, ok
}

// SetConfig writes or updates a key in .bwconfig, preserving other entries.
func (r *Repo) SetConfig(key, value string) error {
	cfg := r.ListConfig()
	cfg[key] = value

	var lines []string
	for k, v := range cfg {
		lines = append(lines, k+"="+v)
	}
	sort.Strings(lines)
	data := strings.Join(lines, "\n") + "\n"
	return r.tfs.WriteFile(".bwconfig", []byte(data))
}

// ListConfig reads all key=value pairs from .bwconfig.
func (r *Repo) ListConfig() map[string]string {
	cfg := make(map[string]string)
	data, err := r.tfs.ReadFile(".bwconfig")
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
	data, err := r.tfs.ReadFile(".bwconfig")
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

// Sync fetches from origin, identifies local-only commits, and pushes or
// returns intents for replay.
func (r *Repo) Sync() (status string, replayed []string, err error) {
	if !r.hasRemote() {
		return "no remote configured", nil, nil
	}

	// Try to fetch
	refSpec := config.RefSpec(fmt.Sprintf("+%s:%s", refLocal, refRemote))
	if fetchErr := r.fetch("origin", refSpec); fetchErr != nil {
		// Remote branch may not exist — just push
		if err := r.push(); err != nil {
			return "", nil, fmt.Errorf("push failed: %w", err)
		}
		return "pushed", nil, nil
	}

	remoteHash, err := r.tfs.LookupRef(refRemote)
	if err != nil {
		// No remote ref after fetch — just push
		if err := r.push(); err != nil {
			return "", nil, fmt.Errorf("push failed: %w", err)
		}
		return "pushed", nil, nil
	}

	localHash := r.tfs.RefHash()

	// Check if we have local commits ahead of remote
	localCommits, err := r.tfs.CommitsBetween(localHash, remoteHash)
	if err != nil {
		return "", nil, err
	}

	if len(localCommits) == 0 {
		// Nothing local — fast-forward to remote
		if localHash != remoteHash {
			if err := r.tfs.Reset(remoteHash); err != nil {
				return "", nil, fmt.Errorf("fast-forward: %w", err)
			}
		}
		return "up to date", nil, nil
	}

	// Check if remote has diverged from local
	remoteCommits, err := r.tfs.CommitsBetween(remoteHash, localHash)
	if err != nil {
		return "", nil, err
	}

	if len(remoteCommits) == 0 {
		// Remote hasn't diverged — local is strictly ahead, just push
		if err := r.push(); err != nil {
			return "", nil, fmt.Errorf("push failed: %w", err)
		}
		return "pushed", nil, nil
	}

	// Diverged: try 3-way tree merge first
	localMsgs := make([]string, 0, len(localCommits))
	for _, c := range localCommits {
		localMsgs = append(localMsgs, c.Message)
	}

	merged, err := r.tfs.MergeCommit(localHash, remoteHash, localMsgs)
	if err != nil {
		return "", nil, fmt.Errorf("merge: %w", err)
	}

	if merged {
		if err := r.push(); err != nil {
			return "", nil, fmt.Errorf("push after merge: %w", err)
		}
		return "rebased and pushed", nil, nil
	}

	// Merge had conflicts — fall back to intent replay
	if err := r.tfs.Reset(remoteHash); err != nil {
		return "", nil, fmt.Errorf("reset to remote: %w", err)
	}
	return "needs replay", localMsgs, nil
}

// Push pushes the beadwork branch to origin.
func (r *Repo) Push() error {
	return r.push()
}

func (r *Repo) push() error {
	refSpec := config.RefSpec(refLocal + ":" + refLocal)
	return r.gitPush("origin", refSpec)
}

func (r *Repo) repoDir() string {
	return filepath.Dir(r.GitDir)
}

func (r *Repo) fetch(remoteName string, refSpec config.RefSpec) error {
	_, err := execGit(r.repoDir(), "fetch", remoteName, string(refSpec))
	return err
}

func (r *Repo) gitPush(remoteName string, refSpec config.RefSpec) error {
	_, err := execGit(r.repoDir(), "push", remoteName, string(refSpec))
	return err
}

// findGitDir walks up from cwd to find the .git directory.
func findGitDir() (string, error) {
	// Use git CLI for reliable detection (handles worktrees, submodules, etc.)
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return filepath.Abs(strings.TrimSpace(string(out)))
}

// execGit is kept only for ls-remote in remoteBranchExists and legacy cleanup.
func execGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(out)), err)
	}
	return string(out), nil
}
