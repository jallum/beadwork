package main

import (
	"fmt"

	"github.com/jallum/beadwork/internal/intent"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

func cmdSync(r *repo.Repo, store *issue.Store, args []string, w Writer) error {
	_ = args

	status, intents, err := r.Sync()
	if err != nil {
		return err
	}

	if status == "needs replay" {
		fmt.Fprintf(w, "rebase conflict — replaying %d intent(s)...\n", len(intents))
		errs := intent.Replay(r, store, intents)
		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintf(w, "  warning: %s\n", e)
			}
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
