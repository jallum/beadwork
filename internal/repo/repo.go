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

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/jallum/beadwork/internal/treefs"
)

const BranchName = "beadwork"

// CurrentVersion is the highest repo schema version this binary understands.
const CurrentVersion = 2

const refLocal = "refs/heads/" + BranchName

type Repo struct {
	GitDir      string
	CWD         string // working directory this repo was discovered from
	Prefix      string
	tfs         *treefs.TreeFS
	initialized bool

	// preReplayHash, when non-zero, is the local ref hash captured just
	// before Sync reset to the remote tip in the "needs replay" branch.
	// Attachment replay walks this commit's tree (via Store.SourceHash)
	// to recover blobs that are still in the ODB but no longer reachable
	// from the current local ref. Cleared by callers once replay is
	// complete via ClearPreReplayHash.
	preReplayHash plumbing.Hash
}

// PreReplayHash returns the local ref hash captured by the most recent
// Sync call on its "needs replay" path. Zero if Sync has not yet taken
// that path (or it has been cleared). Callers should use this to set
// Store.SourceHash before invoking intent.Replay so the attach handler
// can recover pre-reset blobs.
func (r *Repo) PreReplayHash() plumbing.Hash {
	return r.preReplayHash
}

// ClearPreReplayHash resets the captured pre-replay hash. Callers use
// this after replay + push succeeds so a subsequent Sync starts fresh.
func (r *Repo) ClearPreReplayHash() {
	r.preReplayHash = plumbing.ZeroHash
}

// FindRepo discovers the repository from the current working directory.
func FindRepo() (*Repo, error) {
	return FindRepoAt("")
}

// FindRepoAt discovers the repository starting from dir.
// If dir is empty, the current working directory is used.
func FindRepoAt(dir string) (*Repo, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	gitDir, err := findGitDir(dir)
	if err != nil {
		return nil, fmt.Errorf("not a git repository")
	}

	repoDir := filepath.Dir(gitDir)
	goRepo, err := openGitRepo(repoDir)
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	tfs, err := treefs.OpenFromRepo(goRepo, refLocal)
	if err != nil {
		return nil, fmt.Errorf("open treefs: %w", err)
	}

	r := &Repo{
		GitDir: gitDir,
		CWD:    dir,
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

// UserName reads user.name from the git config (local, then global, then
// system) using go-git's ConfigScoped. Returns "unknown" if unset.
func (r *Repo) UserName() string {
	cfg, err := r.tfs.Repo().ConfigScoped(config.GlobalScope)
	if err == nil && cfg.User.Name != "" {
		return cfg.User.Name
	}
	return "unknown"
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
		refSpec := config.RefSpec(fmt.Sprintf("+%s:%s", refLocal, r.refRemote()))
		if err := r.fetch(r.RemoteName(), refSpec); err != nil {
			return fmt.Errorf("fetch failed: %w", err)
		}

		if !localExists {
			// Create local branch from remote
			remoteHash, err := r.tfs.LookupRef(r.refRemote())
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

		// Write skeleton. Only directories that listing code does NOT
		// tolerate missing get a .gitkeep placeholder. Status index
		// directories (status/open, status/in_progress, status/closed,
		// status/deferred) are created on demand by setStatus; list
		// functions treat a missing directory as empty.
		dirs := []string{
			"issues",
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
	_, err := r.tfs.LookupRef(r.refRemote())
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
	out, err := execGit(r.RepoDir(), "ls-remote", "--heads", r.RemoteName(), BranchName)
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
	name := filepath.Base(r.RepoDir())
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

// RemoteName returns the name of the git remote beadwork should fetch from
// and push to. Resolution order:
//  1. git config beadwork.remote (per-clone override; readable before a
//     beadwork branch exists locally)
//  2. .bwconfig key "remote" (team-shared default, travels with the branch)
//  3. "origin" (built-in default, preserves pre-configurable behavior)
func (r *Repo) RemoteName() string {
	if out, err := execGit(r.RepoDir(), "config", "--get", "beadwork.remote"); err == nil {
		if name := strings.TrimSpace(out); name != "" {
			return name
		}
	}
	if v, ok := r.GetConfig("remote"); ok && v != "" {
		return v
	}
	return "origin"
}

// refRemote returns the local ref path for the fetched remote beadwork
// branch, e.g. "refs/remotes/upstream/beadwork".
func (r *Repo) refRemote() string {
	return "refs/remotes/" + r.RemoteName() + "/" + BranchName
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

// UnsetConfig removes a key from .bwconfig. Returns true if the key was
// present and removed, false if it was not present.
func (r *Repo) UnsetConfig(key string) (bool, error) {
	cfg := r.ListConfig()
	if _, ok := cfg[key]; !ok {
		return false, nil
	}
	delete(cfg, key)

	var lines []string
	for k, v := range cfg {
		lines = append(lines, k+"="+v)
	}
	sort.Strings(lines)
	data := strings.Join(lines, "\n")
	if len(lines) > 0 {
		data += "\n"
	}
	return true, r.tfs.WriteFile(".bwconfig", []byte(data))
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

// Sync fetches from the configured remote, identifies local-only commits,
// and pushes or returns intents for replay.
func (r *Repo) Sync() (status string, replayed []string, err error) {
	if !r.hasRemote() {
		return "no remote configured", nil, nil
	}

	// Try to fetch
	refSpec := config.RefSpec(fmt.Sprintf("+%s:%s", refLocal, r.refRemote()))
	if fetchErr := r.fetch(r.RemoteName(), refSpec); fetchErr != nil {
		// Remote branch may not exist — just push
		if err := r.push(); err != nil {
			return "", nil, fmt.Errorf("push failed: %w", err)
		}
		return "pushed", nil, nil
	}

	remoteHash, err := r.tfs.LookupRef(r.refRemote())
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

	// Merge had conflicts — fall back to intent replay.
	// Capture the pre-reset local hash so the caller can expose it to
	// intent replay for attachment-blob recovery. Git keeps the blob
	// objects in the ODB until gc, even after the ref is reset.
	r.preReplayHash = localHash
	if err := r.tfs.Reset(remoteHash); err != nil {
		return "", nil, fmt.Errorf("reset to remote: %w", err)
	}
	return "needs replay", localMsgs, nil
}

// Push pushes the beadwork branch to the configured remote.
func (r *Repo) Push() error {
	return r.push()
}

func (r *Repo) push() error {
	refSpec := config.RefSpec(refLocal + ":" + refLocal)
	return r.gitPush(r.RemoteName(), refSpec)
}

// RepoDir returns the repository root (the parent of the .git directory).
func (r *Repo) RepoDir() string {
	return filepath.Dir(r.GitDir)
}

// WorktreeDirty returns true if the user's working tree has uncommitted changes.
// Uses `git diff --quiet HEAD` which short-circuits on the first changed file
// and skips untracked-file scanning, making it fast on large repos.
// Shells out rather than using go-git's Worktree.Status, which can disagree
// with real git on worktree boundaries, submodules, and file modes.
func (r *Repo) WorktreeDirty() bool {
	_, err := execGit(r.CWD, "diff", "--quiet", "HEAD")
	return err != nil
}

// GitContext holds information about the user's current git working state.
type GitContext struct {
	Branch     string // current branch name (e.g. "main", "bw-a1b/fix-auth-bug")
	LastCommit string // short hash + subject of HEAD commit
	IsWorktree bool   // true if cwd is a git worktree (not the main working tree)
	Dirty      bool   // true if there are uncommitted changes
}

// GetGitContext returns information about the user's current working directory.
// Uses git commands from cwd so it picks up worktree-specific state.
func (r *Repo) GetGitContext() GitContext {
	ctx := GitContext{Dirty: r.WorktreeDirty()}

	if out, err := execGit(r.CWD, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		ctx.Branch = strings.TrimSpace(out)
	}
	if out, err := execGit(r.CWD, "log", "-1", "--oneline"); err == nil {
		ctx.LastCommit = strings.TrimSpace(out)
	}

	// .git is a file in worktrees, a directory in the main working tree
	dotGit := filepath.Join(r.CWD, ".git")
	if fi, err := os.Stat(dotGit); err == nil && !fi.IsDir() {
		ctx.IsWorktree = true
	}

	return ctx
}

func (r *Repo) fetch(remoteName string, refSpec config.RefSpec) error {
	_, err := execGit(r.RepoDir(), "fetch", remoteName, string(refSpec))
	return err
}

func (r *Repo) gitPush(remoteName string, refSpec config.RefSpec) error {
	_, err := execGit(r.RepoDir(), "push", "--no-verify", remoteName, string(refSpec))
	return err
}

// findGitDir returns the common .git directory for the repository.
// It walks up from startDir looking for .git (file or directory).
// If .git is a file (worktree), it reads the gitdir path and
// then reads commondir to resolve the shared .git directory.
func findGitDir(startDir string) (string, error) {
	dir := startDir

	for {
		dotGit := filepath.Join(dir, ".git")
		fi, err := os.Stat(dotGit)
		if err == nil {
			if fi.IsDir() {
				// Normal repo — .git is the git dir
				return dotGit, nil
			}
			// Worktree — .git is a file containing "gitdir: <path>"
			return resolveWorktreeGitDir(dotGit)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding .git
			return "", fmt.Errorf("not a git repository")
		}
		dir = parent
	}
}

// resolveWorktreeGitDir reads a .git file (as found in worktrees),
// extracts the gitdir path, then reads commondir to find the shared
// .git directory.
func resolveWorktreeGitDir(dotGitFile string) (string, error) {
	data, err := os.ReadFile(dotGitFile)
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir: ") {
		return "", fmt.Errorf("invalid .git file: %s", dotGitFile)
	}

	gitdir := strings.TrimPrefix(line, "gitdir: ")
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(filepath.Dir(dotGitFile), gitdir)
	}
	gitdir = filepath.Clean(gitdir)

	// Read commondir to find the shared .git directory
	commondirFile := filepath.Join(gitdir, "commondir")
	cdData, err := os.ReadFile(commondirFile)
	if err != nil {
		// No commondir file — gitdir itself is the common dir
		return gitdir, nil
	}

	commondir := strings.TrimSpace(string(cdData))
	if !filepath.IsAbs(commondir) {
		commondir = filepath.Join(gitdir, commondir)
	}
	return filepath.Abs(commondir)
}

// execGit is kept for network operations: ls-remote, fetch, and push.
func execGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(out)), err)
	}
	return string(out), nil
}
