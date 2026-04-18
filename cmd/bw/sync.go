package main

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/jallum/beadwork/internal/intent"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

func cmdSync(store *issue.Store, args []string, w Writer) error {
	_ = args
	r := store.Committer.(*repo.Repo)

	status, intents, err := r.Sync()
	if err != nil {
		return err
	}

	// After sync the underlying tree may have changed; discard stale cache.
	store.ClearCache()

	if status == "needs replay" {
		// Expose the pre-reset local commit to attachment replay so the
		// attach intent can re-stage blobs whose objects still live in
		// the ODB. See docs/design.md for the replay semantics.
		store.SourceHash = r.PreReplayHash()
		defer func() {
			store.SourceHash = plumbing.ZeroHash
			r.ClearPreReplayHash()
		}()

		fmt.Fprintf(w, "rebase conflict — replaying %d intent(s)...\n", len(intents))
		errs := intent.Replay(store, intents)
		if len(errs) > 0 {
			w.Push(2)
			for _, e := range errs {
				fmt.Fprintf(w, "warning: %s\n", e)
			}
			w.Pop()
		}
		if err := r.Push(); err != nil {
			return fmt.Errorf("push after replay failed: %w", err)
		}
		fmt.Fprintln(w, "replayed and pushed")
	} else {
		fmt.Fprintln(w, status)
	}
	return nil
}
