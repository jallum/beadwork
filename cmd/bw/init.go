package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
)

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
	if ia.Force {
		if err := r.ForceReinit(ia.Prefix); err != nil {
			return err
		}
		emitln(w, initMessage("reinitialized", r.Prefix))
		return nil
	}
	if r.IsInitialized() {
		return fmt.Errorf("beadwork already initialized")
	}
	if err := r.Init(ia.Prefix); err != nil {
		return err
	}
	emitln(w, initMessage("initialized", r.Prefix))
	return nil
}

func initMessage(verb, prefix string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s beadwork (prefix: %s)\n", verb, prefix)
	b.WriteString("\nNext step: run `bw onboard` to set up agent integration.")
	return b.String()
}
