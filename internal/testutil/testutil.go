package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/j5n/beadwork/internal/issue"
	"github.com/j5n/beadwork/internal/repo"
)

// Env is a self-contained test environment with a real git repo.
type Env struct {
	T        *testing.T
	Dir      string
	Repo     *repo.Repo
	Store    *issue.Store
	Cleanup  func()
}

// NewEnv creates a temp directory with a git repo, initializes beadwork,
// and returns everything needed to test against it.
func NewEnv(t *testing.T) *Env {
	t.Helper()
	dir := t.TempDir()

	// Initialize a git repo with one commit
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "initial")

	// cd into the dir so repo.FindRepo works
	orig, _ := os.Getwd()
	os.Chdir(dir)

	r, err := repo.FindRepo()
	if err != nil {
		t.Fatalf("FindRepo: %v", err)
	}
	if err := r.Init("test"); err != nil {
		t.Fatalf("Init: %v", err)
	}

	store := issue.NewStore(r.WorkTree, r.Prefix)

	return &Env{
		T:     t,
		Dir:   dir,
		Repo:  r,
		Store: store,
		Cleanup: func() {
			os.Chdir(orig)
		},
	}
}

// NewBareRemote creates a bare git repo to use as a remote, adds it as
// "origin" to the test env, and returns the bare repo path.
func (e *Env) NewBareRemote() string {
	e.T.Helper()
	bare := filepath.Join(e.Dir, "remote.git")
	run(e.T, e.Dir, "git", "init", "--bare", bare)
	run(e.T, e.Dir, "git", "remote", "add", "origin", bare)
	return bare
}

// CloneEnv clones the remote into a new working directory and initializes
// beadwork from the existing remote branch. Returns a second Env.
func (e *Env) CloneEnv(barePath string) *Env {
	e.T.Helper()
	cloneDir := filepath.Join(e.Dir, "clone")
	run(e.T, e.Dir, "git", "clone", barePath, cloneDir)

	orig, _ := os.Getwd()
	os.Chdir(cloneDir)

	// Configure git in the clone
	run(e.T, cloneDir, "git", "config", "user.email", "clone@test.com")
	run(e.T, cloneDir, "git", "config", "user.name", "Clone Agent")

	r, err := repo.FindRepo()
	if err != nil {
		e.T.Fatalf("FindRepo in clone: %v", err)
	}
	if err := r.Init("test"); err != nil {
		e.T.Fatalf("Init in clone: %v", err)
	}

	store := issue.NewStore(r.WorkTree, r.Prefix)

	return &Env{
		T:     e.T,
		Dir:   cloneDir,
		Repo:  r,
		Store: store,
		Cleanup: func() {
			os.Chdir(orig)
		},
	}
}

// SwitchTo changes the working directory to this env's dir.
func (e *Env) SwitchTo() {
	os.Chdir(e.Dir)
}

// CommitIntent creates and commits a change with the given intent message.
func (e *Env) CommitIntent(msg string) {
	e.T.Helper()
	if err := e.Repo.Commit(msg); err != nil {
		e.T.Fatalf("Commit(%q): %v", msg, err)
	}
}

// MarkerExists checks if a zero-byte marker file exists at the given
// path relative to the beadwork worktree.
func (e *Env) MarkerExists(relPath string) bool {
	_, err := os.Stat(filepath.Join(e.Repo.WorkTree, relPath))
	return err == nil
}

// IssueFileExists checks if an issue JSON file exists.
func (e *Env) IssueFileExists(id string) bool {
	_, err := os.Stat(filepath.Join(e.Repo.WorkTree, "issues", id+".json"))
	return err == nil
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %s: %v", name, args, out, err)
	}
}
