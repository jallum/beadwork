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
func (r *Repo) ForceReinit(prefix string, resolve RemoteResolver) error {
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

	return r.Init(prefix, resolve)
}

// Init creates (or bootstraps from a remote) the beadwork branch.
//
// When any git remote already has the beadwork branch, Init fetches from
// one of them (selected via the same sync precedence: single-only, then
// git config beadwork.remote, then a remote named origin, then
// alphabetically first) and creates the local tracking branch from that
// tip.
//
// When no remote has the branch yet and multiple remotes exist, Init
// invokes resolve to let the caller pick which remote should be the
// project's default going forward; the selection is persisted to
// git config beadwork.remote. Passing nil skips that step — useful for
// tests and for the single-remote common case where no choice is
// needed.
func (r *Repo) Init(prefix string, resolve RemoteResolver) error {
	if r.initialized {
		return fmt.Errorf("beadwork already initialized")
	}

	if err := ValidatePrefix(prefix); err != nil {
		return err
	}

	allRemotes, err := r.tfs.RemoteNames()
	if err != nil {
		return fmt.Errorf("list remotes: %w", err)
	}
	var hasBW []string
	for _, name := range allRemotes {
		if r.remoteHasBeadwork(name) {
			hasBW = append(hasBW, name)
		}
	}
	sort.Strings(hasBW)

	localExists := r.localBranchExists()

	if len(hasBW) > 0 {
		// At least one remote has beadwork — bootstrap from it. Init
		// never prompts in this branch; it picks deterministically.
		fetchFrom := initFetchRemote(r, hasBW)
		remoteRef := "refs/remotes/" + fetchFrom + "/" + BranchName
		refSpec := config.RefSpec(fmt.Sprintf("+%s:%s", refLocal, remoteRef))
		if err := r.fetch(fetchFrom, refSpec); err != nil {
			return fmt.Errorf("fetch failed: %w", err)
		}

		if !localExists {
			remoteHash, err := r.tfs.LookupRef(remoteRef)
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
		// No remote has beadwork yet and we're about to seed a new
		// local branch. If multiple remotes exist AND sync's deterministic
		// short-circuits (existing git config, a remote named origin)
		// wouldn't already pick one, ask the user now so future
		// `bw sync` / `bw push` runs don't have to. Short-circuit cases
		// aren't persisted — sync re-applies the same rules on its own.
		if len(allRemotes) >= 2 && resolve != nil && initNeedsPrompt(r, allRemotes) {
			chosen, err := resolve(allRemotes)
			if err != nil {
				return err
			}
			if _, err := execGit(r.RepoDir(), "config", "beadwork.remote", chosen); err != nil {
				return fmt.Errorf("persist beadwork.remote: %w", err)
			}
		}

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

// initNeedsPrompt returns true when the "seed a fresh local branch in a
// multi-remote repo" path actually needs to ask the user. Returns false
// when git config beadwork.remote is already set to one of the remotes
// or when a remote named "origin" exists — in both cases sync will
// make the same pick on its own, so there's nothing useful to persist.
func initNeedsPrompt(r *Repo, allRemotes []string) bool {
	if out, err := execGit(r.RepoDir(), "config", "--get", "beadwork.remote"); err == nil {
		if cfg := strings.TrimSpace(out); cfg != "" {
			for _, name := range allRemotes {
				if name == cfg {
					return false
				}
			}
		}
	}
	for _, name := range allRemotes {
		if name == "origin" {
			return false
		}
	}
	return true
}

// initFetchRemote picks one remote from a non-empty list of remotes that
// have the beadwork branch, using the sync precedence but silently
// falling back to alphabetically-first when neither git config nor
// origin is a usable match. Never prompts.
func initFetchRemote(r *Repo, hasBW []string) string {
	if len(hasBW) == 1 {
		return hasBW[0]
	}
	if out, err := execGit(r.RepoDir(), "config", "--get", "beadwork.remote"); err == nil {
		cfg := strings.TrimSpace(out)
		for _, name := range hasBW {
			if name == cfg {
				return cfg
			}
		}
	}
	for _, name := range hasBW {
		if name == "origin" {
			return "origin"
		}
	}
	return hasBW[0]
}

// AllCommits returns all commits on the beadwork branch, newest-first.
func (r *Repo) AllCommits() ([]treefs.CommitInfo, error) {
	return r.tfs.AllCommits()
}

func (r *Repo) Commit(message string) error {
	return r.tfs.Commit(message)
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

// RemoteName returns the name of the git remote beadwork should use as a
// single-remote fallback (primarily for Init's bootstrap fetch). It reads
// git config beadwork.remote, defaulting to "origin" when unset. The
// richer multi-remote resolution for Sync/Push lives elsewhere.
func (r *Repo) RemoteName() string {
	if out, err := execGit(r.RepoDir(), "config", "--get", "beadwork.remote"); err == nil {
		if name := strings.TrimSpace(out); name != "" {
			return name
		}
	}
	return "origin"
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

// RemoteResolver picks a single remote name from a list of candidate
// remotes. It is invoked by Sync/Push in the "no remote has beadwork yet"
// fallback when the deterministic rules (single remote, git config
// beadwork.remote, a remote named "origin") don't produce an answer.
// Passing nil to Sync/Push causes that fallback to return a non-interactive
// error instead of prompting.
type RemoteResolver func(candidates []string) (string, error)

// remoteHasBeadwork returns true if the given remote has refs/heads/beadwork.
// Uses `git ls-remote --heads <name> beadwork` for a live probe.
func (r *Repo) remoteHasBeadwork(name string) bool {
	out, err := execGit(r.RepoDir(), "ls-remote", "--heads", name, BranchName)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// targetRemotes returns the list of remotes bw should act on.
// If any remote has the beadwork branch, only those are returned. Otherwise
// it resolves a single remote via resolveSingleRemote (short-circuit rules
// first, then the resolver callback).
func (r *Repo) targetRemotes(resolve RemoteResolver) ([]string, error) {
	all, err := r.tfs.RemoteNames()
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, nil
	}
	var hasBW []string
	for _, name := range all {
		if r.remoteHasBeadwork(name) {
			hasBW = append(hasBW, name)
		}
	}
	if len(hasBW) > 0 {
		sort.Strings(hasBW)
		return hasBW, nil
	}
	chosen, err := r.resolveSingleRemote(all, resolve)
	if err != nil {
		return nil, err
	}
	return []string{chosen}, nil
}

// resolveSingleRemote applies the precedence rules for picking exactly one
// remote when no remote has the beadwork branch yet.
func (r *Repo) resolveSingleRemote(all []string, resolve RemoteResolver) (string, error) {
	if len(all) == 1 {
		return all[0], nil
	}
	if out, err := execGit(r.RepoDir(), "config", "--get", "beadwork.remote"); err == nil {
		if cfg := strings.TrimSpace(out); cfg != "" {
			for _, name := range all {
				if name == cfg {
					return cfg, nil
				}
			}
			return "", fmt.Errorf("git config beadwork.remote is set to %q but no remote by that name exists (remotes: %s)", cfg, strings.Join(all, ", "))
		}
	}
	for _, name := range all {
		if name == "origin" {
			return "origin", nil
		}
	}
	if resolve != nil {
		return resolve(all)
	}
	return "", fmt.Errorf("no default remote — multiple remotes, none have the %q branch, no remote is named \"origin\", and git config beadwork.remote is unset. Set one with: git config beadwork.remote <name> (remotes: %s)", BranchName, strings.Join(all, ", "))
}

// Sync fetches from every remote that has the beadwork branch, merges
// their tips into local, and pushes the result back to every one of them.
// If no remote has the branch yet, it resolves a single remote via the
// precedence rules (single-remote auto-pick, git config beadwork.remote,
// "origin" by name, or the resolver callback for interactive selection)
// and pushes to just that one.
//
// On merge conflict against any remote, returns ("needs replay",
// conflicting-local-commits, nil) with preReplayHash captured. Callers run
// intent.Replay then re-invoke Sync to finish the fan-out.
func (r *Repo) Sync(resolve RemoteResolver) (status string, replayed []string, err error) {
	if !r.hasRemote() {
		return "no remote configured", nil, nil
	}

	remotes, err := r.targetRemotes(resolve)
	if err != nil {
		return "", nil, err
	}
	if len(remotes) == 0 {
		return "no remote configured", nil, nil
	}

	return r.syncTo(remotes)
}

// syncTo runs the three-phase multi-remote sync across the given remotes:
// fetch each, merge each into local in sorted order (stopping on conflict),
// then push local to each.
func (r *Repo) syncTo(remotes []string) (string, []string, error) {
	// Phase A: fetch every target remote into its own tracking ref.
	// A failure on one remote (e.g. network, missing beadwork branch)
	// is warned-and-skipped so the rest of the sync can proceed.
	fetched := make([]string, 0, len(remotes))
	for _, name := range remotes {
		refSpec := config.RefSpec(fmt.Sprintf("+%s:refs/remotes/%s/%s", refLocal, name, BranchName))
		if err := r.fetch(name, refSpec); err != nil {
			fmt.Fprintf(os.Stderr, "warning: fetch from %s failed: %v\n", name, err)
			continue
		}
		fetched = append(fetched, name)
	}
	sort.Strings(fetched)

	// Phase B: merge each fetched remote's tip into local.
	didMerge := false
	didFastForward := false
	for _, name := range fetched {
		remoteRef := "refs/remotes/" + name + "/" + BranchName
		remoteHash, err := r.tfs.LookupRef(remoteRef)
		if err != nil {
			// Tracking ref missing despite successful fetch — skip.
			continue
		}
		localHash := r.tfs.RefHash()

		localCommits, err := r.tfs.CommitsBetween(localHash, remoteHash)
		if err != nil {
			return "", nil, err
		}
		if len(localCommits) == 0 {
			if localHash != remoteHash {
				if err := r.tfs.Reset(remoteHash); err != nil {
					return "", nil, fmt.Errorf("fast-forward from %s: %w", name, err)
				}
				didFastForward = true
			}
			continue
		}

		remoteCommits, err := r.tfs.CommitsBetween(remoteHash, localHash)
		if err != nil {
			return "", nil, err
		}
		if len(remoteCommits) == 0 {
			// Local strictly ahead of this remote — Phase C will push.
			continue
		}

		// Diverged — attempt 3-way tree merge.
		localMsgs := make([]string, 0, len(localCommits))
		for _, c := range localCommits {
			localMsgs = append(localMsgs, c.Message)
		}
		merged, err := r.tfs.MergeCommit(localHash, remoteHash, localMsgs)
		if err != nil {
			return "", nil, fmt.Errorf("merge with %s: %w", name, err)
		}
		if merged {
			didMerge = true
			continue
		}

		// Conflict: capture pre-reset hash for attachment-blob recovery
		// (same as the single-remote path previously did), reset to this
		// remote's tip, and bubble the conflicting intents out for replay.
		r.preReplayHash = localHash
		if err := r.tfs.Reset(remoteHash); err != nil {
			return "", nil, fmt.Errorf("reset to %s: %w", name, err)
		}
		return "needs replay", localMsgs, nil
	}

	// Phase C: push local to every target remote. Only counts as "pushed"
	// when local is actually ahead of (or unrelated to) the remote's tip.
	pushRefSpec := config.RefSpec(refLocal + ":" + refLocal)
	pushedCount := 0
	for _, name := range remotes {
		remoteRef := "refs/remotes/" + name + "/" + BranchName
		localHash := r.tfs.RefHash()
		remoteHash, refErr := r.tfs.LookupRef(remoteRef)
		needsPush := refErr != nil || remoteHash != localHash
		if !needsPush {
			continue
		}
		if err := r.gitPush(name, pushRefSpec); err != nil {
			return "", nil, fmt.Errorf("push to %s failed: %w", name, err)
		}
		pushedCount++
	}

	switch {
	case didMerge && pushedCount > 0:
		return "rebased and pushed", nil, nil
	case pushedCount > 0:
		return "pushed", nil, nil
	case didFastForward:
		return "up to date", nil, nil
	default:
		return "up to date", nil, nil
	}
}

// Push pushes the beadwork branch to every remote that has the branch, or
// to a single resolved remote if none of them do. See Sync for the
// resolver semantics.
func (r *Repo) Push(resolve RemoteResolver) error {
	if !r.hasRemote() {
		return fmt.Errorf("no remote configured")
	}
	remotes, err := r.targetRemotes(resolve)
	if err != nil {
		return err
	}
	if len(remotes) == 0 {
		return fmt.Errorf("no remote configured")
	}
	return r.pushTo(remotes)
}

func (r *Repo) pushTo(remotes []string) error {
	refSpec := config.RefSpec(refLocal + ":" + refLocal)
	for _, name := range remotes {
		if err := r.gitPush(name, refSpec); err != nil {
			return fmt.Errorf("push to %s failed: %w", name, err)
		}
	}
	return nil
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
