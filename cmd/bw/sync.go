package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jallum/beadwork/internal/config"

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

func cmdSync(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	_ = args
	r := store.Committer.(*repo.Repo)

	resolver := makeRemoteResolver(r, w)

	status, intents, err := r.Sync(resolver)
	if err != nil {
		return nil, err
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
			return nil, fmt.Errorf("push after replay failed: %w", err)
		}
		fmt.Fprintln(w, "replayed and pushed")
	} else {
		fmt.Fprintln(w, status)
	}
	return nil, nil
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
		chosen, err := promptForRemote(candidates, w)
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

// promptForRemote presents a numbered menu and reads a selection from
// syncStdin. Re-prompts up to 3 times on invalid input.
func promptForRemote(candidates []string, w Writer) (string, error) {
	return promptForRemoteWithReader(candidates, w, syncStdin)
}

// promptForRemoteWithReader is the shared implementation behind both the
// sync and init prompts. The caller supplies the input source so init
// can use initStdin and sync can use syncStdin without either depending
// on the other's package-level variable.
func promptForRemoteWithReader(candidates []string, w Writer, source io.Reader) (string, error) {
	reader := bufio.NewReader(source)
	for attempt := 0; attempt < 3; attempt++ {
		fmt.Fprintln(w, "multiple remotes — pick one for bw to sync with:")
		for i, name := range candidates {
			fmt.Fprintf(w, "  %d) %s\n", i+1, name)
		}
		fmt.Fprint(w, "select [1]: ")

		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return "", fmt.Errorf("read input: %w", err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			line = "1"
		}
		n, convErr := strconv.Atoi(line)
		if convErr != nil || n < 1 || n > len(candidates) {
			fmt.Fprintf(w, "invalid selection %q; expected a number between 1 and %d\n", line, len(candidates))
			continue
		}
		return candidates[n-1], nil
	}
	return "", fmt.Errorf("too many invalid selections")
}

func gitConfigSet(repoDir, key, value string) error {
	cmd := exec.Command("git", "config", key, value)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git config %s %s: %s: %w", key, value, strings.TrimSpace(string(out)), err)
	}
	return nil
}
