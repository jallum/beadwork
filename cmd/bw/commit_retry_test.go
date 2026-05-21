package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
	"github.com/jallum/beadwork/internal/treefs"
)

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
