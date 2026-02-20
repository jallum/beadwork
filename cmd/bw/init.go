package main

import (
	"fmt"
	"io"
)

func cmdInit(args []string, w io.Writer) error {
	a := ParseArgs(args, "--prefix")

	r, err := getRepo()
	if err != nil {
		return err
	}
	if a.Bool("--force") {
		if err := r.ForceReinit(a.String("--prefix")); err != nil {
			return err
		}
		fmt.Fprintf(w, "reinitialized beadwork (prefix: %s)\n", r.Prefix)
		return nil
	}
	if r.IsInitialized() {
		return fmt.Errorf("beadwork already initialized")
	}
	if err := r.Init(a.String("--prefix")); err != nil {
		return err
	}
	fmt.Fprintf(w, "initialized beadwork (prefix: %s)\n", r.Prefix)
	return nil
}
