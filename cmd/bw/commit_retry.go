package main

import (
	"errors"
	"fmt"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/treefs"
)

// commitMaxRetries bounds how many times mutating commands retry a commit
// after losing the CAS race against a concurrent writer.
const commitMaxRetries = 3

// commitWithRetry runs apply, then store.Commit(intent), retrying up to
// maxRetries times when the commit fails with treefs.ErrRefMoved. Between
// attempts the store cache is cleared and the underlying TreeFS is
// refreshed so apply observes the latest ref state.
//
// apply must re-derive all state from the store on every invocation —
// values captured from a previous attempt are stale after Refresh and
// must not be reused.
func commitWithRetry(store *issue.Store, maxRetries int, apply func() (intent string, err error)) error {
	var commitErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			store.ClearCache()
			if err := store.Refresh(); err != nil {
				return fmt.Errorf("refresh after conflict: %w", err)
			}
		}
		intent, aerr := apply()
		if aerr != nil {
			return aerr
		}
		commitErr = store.Commit(intent)
		if commitErr == nil {
			return nil
		}
		if !errors.Is(commitErr, treefs.ErrRefMoved) {
			return fmt.Errorf("commit failed: %w", commitErr)
		}
	}
	return fmt.Errorf("commit failed after %d attempts: %w", maxRetries, commitErr)
}
