package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

// initStdin is the input source for the init remote-selection prompt.
// Tests override it with a strings.Reader. Unlike sync's syncStdin, the
// init prompt is never gated on a TTY check: `bw init` is a
// user-initiated setup command, so we always prompt when there's an
// ambiguous seed-remote choice.
var initStdin io.Reader = os.Stdin

type InitArgs struct {
	Prefix string
	Force  bool
}

func parseInitArgs(raw []string) (InitArgs, error) {
	a, err := ParseArgs(raw, []string{"--prefix"}, []string{"--force"})
	if err != nil {
		return InitArgs{}, err
	}
	return InitArgs{
		Prefix: a.String("--prefix"),
		Force:  a.Bool("--force"),
	}, nil
}

func cmdInit(_ *issue.Store, args []string, w Writer) error {
	ia, err := parseInitArgs(args)
	if err != nil {
		return err
	}

	r, err := getRepo()
	if err != nil {
		return err
	}
	resolver := makeInitRemoteResolver(r, w)
	if ia.Force {
		if err := r.ForceReinit(ia.Prefix, resolver); err != nil {
			return err
		}
		fmt.Fprintln(w, initMessage("reinitialized", r.Prefix))
		return nil
	}
	if r.IsInitialized() {
		return fmt.Errorf("beadwork already initialized")
	}
	if err := r.Init(ia.Prefix, resolver); err != nil {
		return err
	}
	fmt.Fprintln(w, initMessage("initialized", r.Prefix))
	return nil
}

// makeInitRemoteResolver returns a RemoteResolver that unconditionally
// prompts (no TTY check) and persists the choice via git config. Init
// is a human-driven setup step, so there's no non-interactive path to
// guard against — if someone really can't interact, they can set
// git config beadwork.remote <name> before running bw init and the
// resolver will never be called.
func makeInitRemoteResolver(r *repo.Repo, w Writer) repo.RemoteResolver {
	return func(candidates []string) (string, error) {
		chosen, err := promptForRemoteWithReader(candidates, w, initStdin)
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


func initMessage(verb, prefix string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s beadwork (prefix: %s)\n", verb, prefix)
	b.WriteString("\nNext step: run `bw onboard` to set up agent integration.")
	return b.String()
}
