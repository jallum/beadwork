package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
	"github.com/jallum/beadwork/internal/treefs"
)

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %s: %v", name, args, out, err)
	}
}

func runOut(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %s: %v", name, args, out, err)
	}
	return string(out)
}

// TestCommitWithRetryRecoversFromRefMove proves the retry helper refreshes
// and replays when another writer moves the ref between mutation and commit.
func TestCommitWithRetryRecoversFromRefMove(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To start", issue.CreateOpts{})
	if err := env.Repo.Commit("create " + iss.ID); err != nil {
		t.Fatalf("seed commit: %v", err)
	}

	// Open a second TreeFS on the same ref to simulate a concurrent writer.
	racer, err := treefs.Open(env.Dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("open racer: %v", err)
	}

	calls := 0
	err = commitWithRetry(env.Store, 3, func() (string, error) {
		calls++

		// Mutate via the store on every attempt.
		if _, serr := env.Store.Start(iss.ID, "alice"); serr != nil {
			return "", serr
		}

		// On the first attempt only, advance the ref behind the store's
		// back so its Commit will fail with ErrRefMoved.
		if calls == 1 {
			racer.WriteFile("racer.txt", []byte("racer"))
			if cerr := racer.Commit("racer commit"); cerr != nil {
				return "", fmt.Errorf("racer commit: %w", cerr)
			}
		}

		return "start " + iss.ID, nil
	})
	if err != nil {
		t.Fatalf("commitWithRetry: %v", err)
	}
	if calls < 2 {
		t.Errorf("apply called %d times; expected at least 2 (one retry)", calls)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "in_progress" {
		t.Errorf("final status = %q, want in_progress", got.Status)
	}
}

// TestCommitWithRetryGivesUpAfterMax proves the helper bounds its retries.
func TestCommitWithRetryGivesUpAfterMax(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To start", issue.CreateOpts{})
	if err := env.Repo.Commit("create " + iss.ID); err != nil {
		t.Fatalf("seed commit: %v", err)
	}

	racer, err := treefs.Open(env.Dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("open racer: %v", err)
	}

	racerCounter := 0
	err = commitWithRetry(env.Store, 2, func() (string, error) {
		if _, serr := env.Store.Start(iss.ID, "alice"); serr != nil {
			return "", serr
		}
		// Advance the ref on every attempt so we always lose the CAS.
		racerCounter++
		racer.WriteFile(fmt.Sprintf("r%d.txt", racerCounter), []byte("x"))
		if cerr := racer.Commit(fmt.Sprintf("racer %d", racerCounter)); cerr != nil {
			return "", cerr
		}
		return "start " + iss.ID, nil
	})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if !errors.Is(err, treefs.ErrRefMoved) {
		t.Errorf("expected ErrRefMoved in error chain, got: %v", err)
	}
}

// TestCommitWithRetryRecoversFromRawGitRefMove proves the helper picks up
// state when the ref is moved by a raw git process (not go-git) — defending
// against any in-process caching the existing TreeFS might be holding.
// This is the closest analogue to the user's reported failure mode: a
// concurrent `bw` invocation in a different process advances the ref via
// the on-disk loose ref file.
func TestCommitWithRetryRecoversFromRawGitRefMove(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To start", issue.CreateOpts{})
	if err := env.Repo.Commit("create " + iss.ID); err != nil {
		t.Fatalf("seed commit: %v", err)
	}

	// Build a fully-independent commit object that we can advance the ref
	// to via raw git plumbing (no go-git involvement). We pre-create a
	// dangling commit referencing the current ref's tree so the ref move
	// is valid history.
	cur := strings.TrimSpace(runOut(t, env.Dir, "git", "rev-parse", "refs/heads/beadwork"))
	tree := strings.TrimSpace(runOut(t, env.Dir, "git", "rev-parse", "refs/heads/beadwork^{tree}"))
	newHash := strings.TrimSpace(runOut(t, env.Dir,
		"sh", "-c",
		"GIT_AUTHOR_NAME=racer GIT_AUTHOR_EMAIL=r@r GIT_COMMITTER_NAME=racer GIT_COMMITTER_EMAIL=r@r "+
			"git commit-tree "+tree+" -p "+cur+" -m racer"))
	if newHash == "" {
		t.Fatal("failed to build racer commit")
	}

	calls := 0
	err := commitWithRetry(env.Store, 3, func() (string, error) {
		calls++
		if _, serr := env.Store.Start(iss.ID, "alice"); serr != nil {
			return "", serr
		}
		if calls == 1 {
			// Advance the ref via raw git on the filesystem. The
			// in-process TreeFS's go-git Repository handle is now
			// looking at a stale view of refs/heads/beadwork.
			run(t, env.Dir, "git", "update-ref", "refs/heads/beadwork", newHash)
		}
		return "start " + iss.ID, nil
	})
	if err != nil {
		t.Fatalf("commitWithRetry: %v", err)
	}
	if calls < 2 {
		t.Errorf("apply called %d times; expected at least 2 (one retry)", calls)
	}
	got, _ := env.Store.Get(iss.ID)
	if got.Status != "in_progress" {
		t.Errorf("final status = %q, want in_progress", got.Status)
	}
}

// TestCommitWithRetrySurvivesSustainedContention proves the helper can
// recover when the ref is being moved on every retry until eventually
// the racer stops. This is the closest analogue to the user-reported
// failure: "commit failed after 3 attempts" because the race kept
// winning. With more retries (and ideally backoff), we should succeed.
func TestCommitWithRetrySurvivesSustainedContention(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("To start", issue.CreateOpts{})
	if err := env.Repo.Commit("create " + iss.ID); err != nil {
		t.Fatalf("seed commit: %v", err)
	}

	racer, err := treefs.Open(env.Dir, "refs/heads/beadwork")
	if err != nil {
		t.Fatalf("open racer: %v", err)
	}

	// Move the ref on the first 5 attempts; let the 6th succeed.
	calls := 0
	err = commitWithRetry(env.Store, 10, func() (string, error) {
		calls++
		if _, serr := env.Store.Start(iss.ID, "alice"); serr != nil {
			return "", serr
		}
		if calls <= 5 {
			racer.WriteFile(fmt.Sprintf("r%d.txt", calls), []byte("x"))
			if cerr := racer.Commit(fmt.Sprintf("racer %d", calls)); cerr != nil {
				return "", cerr
			}
		}
		return "start " + iss.ID, nil
	})
	if err != nil {
		t.Fatalf("commitWithRetry: %v", err)
	}
	if calls < 6 {
		t.Errorf("apply called %d times; expected at least 6", calls)
	}
}

// TestCommitMaxRetriesIsGenerous documents the chosen retry budget.
// Three was too few in practice; bump it.
func TestCommitMaxRetriesIsGenerous(t *testing.T) {
	if commitMaxRetries < 10 {
		t.Errorf("commitMaxRetries = %d, want >= 10 (got user-reported failure at 3)", commitMaxRetries)
	}
}

// TestCommitWithRetryPropagatesNonRetryableErrors proves we don't loop on
// errors unrelated to CAS.
func TestCommitWithRetryPropagatesNonRetryableErrors(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	sentinel := errors.New("sentinel apply failure")
	calls := 0
	err := commitWithRetry(env.Store, 3, func() (string, error) {
		calls++
		return "", sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got: %v", err)
	}
	if calls != 1 {
		t.Errorf("apply called %d times; expected 1", calls)
	}
}
