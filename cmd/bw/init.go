package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
)

// initStdin is the input source for the init remote-selection prompt.
// Tests override it with a strings.Reader. Kept separate from syncStdin
// so init and sync tests can drive their prompts independently.
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

func cmdInit(_ *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	ia, err := parseInitArgs(args)
	if err != nil {
		return nil, err
	}

	r, err := getRepo()
	if err != nil {
		return nil, err
	}
	resolver := makeRemoteResolver(r, w, initStdin)
	if ia.Force {
		if err := r.ForceReinit(ia.Prefix, resolver); err != nil {
			return nil, err
		}
		fmt.Fprintln(w, initMessage("reinitialized", r.Prefix))
		return nil, nil
	}
	if r.IsInitialized() {
		return nil, fmt.Errorf("beadwork already initialized")
	}
	if err := r.Init(ia.Prefix, resolver); err != nil {
		return nil, err
	}
	fmt.Fprintln(w, initMessage("initialized", r.Prefix))
	return nil, nil
}

func initMessage(verb, prefix string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s beadwork (prefix: %s)\n", verb, prefix)
	b.WriteString("\nNext step: run `bw onboard` to set up agent integration.")
	return b.String()
}
