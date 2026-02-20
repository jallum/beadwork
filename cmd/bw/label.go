package main

import (
	"fmt"
	"io"
	"strings"
)

func cmdLabel(args []string, w io.Writer) error {
	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	// bw label <id> +bug +frontend -wontfix
	if len(args) < 2 {
		return fmt.Errorf("usage: bw label <id> +label [-label] ...")
	}
	id := args[0]

	var add, remove []string
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "+") {
			add = append(add, strings.TrimPrefix(arg, "+"))
		} else if strings.HasPrefix(arg, "-") {
			remove = append(remove, strings.TrimPrefix(arg, "-"))
		} else {
			// bare label name = add
			add = append(add, arg)
		}
	}

	iss, err := store.Label(id, add, remove)
	if err != nil {
		return err
	}

	var parts []string
	for _, l := range add {
		parts = append(parts, "+"+l)
	}
	for _, l := range remove {
		parts = append(parts, "-"+l)
	}
	intent := fmt.Sprintf("label %s %s", iss.ID, strings.Join(parts, " "))
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if hasFlag(args, "--json") {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "labeled %s: %s\n", iss.ID, strings.Join(iss.Labels, ", "))
	}
	return nil
}
