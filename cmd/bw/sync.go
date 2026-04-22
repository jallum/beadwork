package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/jallum/beadwork/internal/intent"
	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
	"golang.org/x/term"
)

// syncStdin is the input source for the interactive remote-selection
// prompt. Tests override it via a strings.Reader the same way upgrade.go
// uses upgradeStdin.
var syncStdin io.Reader = os.Stdin

// isInteractiveStdin decides whether the remote-selection prompt is
// allowed. It is a var (not a function) so tests can stub it without
// touching real stdin. When it returns false we must never prompt, even
// if a controlling terminal is reachable via /dev/tty.
var isInteractiveStdin = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func cmdSync(store *issue.Store, args []string, w Writer) error {
	_ = args
	r := store.Committer.(*repo.Repo)

	resolver := makeRemoteResolver(r, w)

	status, intents, err := r.Sync(resolver)
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
		if err := r.Push(resolver); err != nil {
			return fmt.Errorf("push after replay failed: %w", err)
		}
		fmt.Fprintln(w, "replayed and pushed")
	} else {
		fmt.Fprintln(w, status)
	}
	return nil
}

// makeRemoteResolver returns a RemoteResolver closed over the repo and
// the CLI writer. When stdin is interactive it prompts; otherwise it
// returns the same non-interactive error that a nil resolver would
// produce from the repo layer.
func makeRemoteResolver(r *repo.Repo, w Writer) repo.RemoteResolver {
	return func(candidates []string) (string, error) {
		if !isInteractiveStdin() {
			return "", fmt.Errorf("no default remote — multiple remotes, none have the %q branch, no remote is named \"origin\", and git config beadwork.remote is unset. Set one with: git config beadwork.remote <name> (remotes: %s)",
				"beadwork", strings.Join(candidates, ", "))
		}
		chosen, err := promptForRemoteWithReader(candidates, w, syncStdin)
		if err != nil {
			return "", err
		}
		if err := gitConfigSet(r.RepoDir(), "beadwork.remote", chosen); err != nil {
			return "", fmt.Errorf("set beadwork.remote: %w", err)
		}
		fmt.Fprintf(w, "saved: git config beadwork.remote=%s\n", chosen)
		return chosen, nil
	}
}
