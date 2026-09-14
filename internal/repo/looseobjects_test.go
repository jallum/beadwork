package repo_test

import (
	"os/exec"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

// runGit runs a real git command in dir, failing the test on error. testutil
// has its own unexported equivalent; this one is separate because it needs
// a subcommand — `maintenance run` — that testutil doesn't expose.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s: %v", args, out, err)
	}
}

// TestSyncAfterLooseObjectsMaintenance reproduces the failure described in
// go-git/go-git#2345: Git's `loose-objects` maintenance task batches loose
// objects into a pack named "loose-<hash>.pack" rather than the usual
// "pack-<hash>.pack". Its first run creates that pack but leaves the loose
// copies on disk; a second run prunes them, so from that point those
// objects exist only inside the loose-named pack.
//
// A go-git release without the go-git/go-git#2390 fix ignores any pack not
// named "pack-*", so once a repo reaches that state, Sync fails outright —
// typically "reopen after fetch: reopen treefs: read commit: object not
// found", since even the local branch's own history is now unreadable, or
// "walk remote commits: object not found" if it gets further before
// hitting an affected object. Either way the repository is perfectly
// healthy: `git fsck` and `git log` see it fine, because native git reads
// any packfile under objects/pack regardless of name. This needs no
// unusual git configuration to hit — `git maintenance start`/`register`
// (and Scalar) schedule this task daily by default.
func TestSyncAfterLooseObjectsMaintenance(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	bare := env.NewBareRemote()
	_ = bare

	// Build history that gets pushed, so those commits are known locally as
	// loose objects at push time (push does not repack or remove them).
	for i := 0; i < 5; i++ {
		iss, _ := env.Store.Create("Before push", issue.CreateOpts{})
		env.CommitIntent("create " + iss.ID)
	}
	if status, _, err := env.Repo.Sync(nil); err != nil || status != "pushed" {
		t.Fatalf("initial Sync: status=%q err=%v, want status=%q", status, err, "pushed")
	}

	// More commits after the push: local-only, and also still loose.
	for i := 0; i < 5; i++ {
		iss, _ := env.Store.Create("After push", issue.CreateOpts{})
		env.CommitIntent("create " + iss.ID)
	}

	// First run batches the current loose objects into loose-<hash>.pack
	// but leaves the loose copies in place.
	runGit(t, env.Dir, "maintenance", "run", "--task=loose-objects")
	// Second run prunes the now-duplicate loose copies, so those objects
	// exist only inside the loose-named pack from here on.
	runGit(t, env.Dir, "maintenance", "run", "--task=loose-objects")

	// Sync fetches the remote tip and walks from it (CommitsBetween), which
	// is exactly the walk that fails to see objects living only in a
	// loose-named pack.
	status, _, err := env.Repo.Sync(nil)
	if err != nil {
		t.Fatalf("Sync after loose-objects maintenance: %v", err)
	}
	if status != "pushed" {
		t.Errorf("status = %q, want %q", status, "pushed")
	}
}
