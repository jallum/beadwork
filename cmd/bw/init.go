package main

import (
	"fmt"
	"io"
)

type InitArgs struct {
	Prefix string
	Force  bool
}

func parseInitArgs(raw []string) (InitArgs, error) {
	a := ParseArgs(raw, "--prefix")
	return InitArgs{
		Prefix: a.String("--prefix"),
		Force:  a.Bool("--force"),
	}, nil
}

func cmdInit(args []string, w io.Writer) error {
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
		fmt.Fprintf(w, "reinitialized beadwork (prefix: %s)\n", r.Prefix)
		return nil
	}
	if r.IsInitialized() {
		return fmt.Errorf("beadwork already initialized")
	}
	if err := r.Init(ia.Prefix); err != nil {
		return err
	}
	fmt.Fprintf(w, "initialized beadwork (prefix: %s)\n", r.Prefix)
	return nil
}
