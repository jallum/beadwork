package main

import (
	"fmt"
	"io"
)

func cmdInit(args []string, w io.Writer) error {
	prefix := ""
	force := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--prefix":
			if i+1 < len(args) {
				prefix = args[i+1]
				i++
			}
		case "--force":
			force = true
		}
	}

	r, err := getRepo()
	if err != nil {
		return err
	}
	if force {
		if err := r.ForceReinit(prefix); err != nil {
			return err
		}
		fmt.Fprintf(w, "reinitialized beadwork (prefix: %s)\n", r.Prefix)
		return nil
	}
	if r.IsInitialized() {
		return fmt.Errorf("beadwork already initialized")
	}
	if err := r.Init(prefix); err != nil {
		return err
	}
	fmt.Fprintf(w, "initialized beadwork (prefix: %s)\n", r.Prefix)
	return nil
}
