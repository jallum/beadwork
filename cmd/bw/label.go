package main

import (
	"fmt"
	"io"
	"strings"
)

type LabelArgs struct {
	ID     string
	Add    []string
	Remove []string
	JSON   bool
}

func parseLabelArgs(raw []string) (LabelArgs, error) {
	if len(raw) < 2 {
		return LabelArgs{}, fmt.Errorf("usage: bw label <id> +label [-label] ...")
	}
	la := LabelArgs{ID: raw[0], JSON: hasFlag(raw, "--json")}
	for _, arg := range raw[1:] {
		if arg == "--json" {
			continue
		}
		if strings.HasPrefix(arg, "+") {
			la.Add = append(la.Add, strings.TrimPrefix(arg, "+"))
		} else if strings.HasPrefix(arg, "-") {
			la.Remove = append(la.Remove, strings.TrimPrefix(arg, "-"))
		} else {
			// bare label name = add
			la.Add = append(la.Add, arg)
		}
	}
	return la, nil
}

func cmdLabel(args []string, w io.Writer) error {
	la, err := parseLabelArgs(args)
	if err != nil {
		return err
	}

	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	iss, err := store.Label(la.ID, la.Add, la.Remove)
	if err != nil {
		return err
	}

	var parts []string
	for _, l := range la.Add {
		parts = append(parts, "+"+l)
	}
	for _, l := range la.Remove {
		parts = append(parts, "-"+l)
	}
	intent := fmt.Sprintf("label %s %s", iss.ID, strings.Join(parts, " "))
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if la.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "labeled %s: %s\n", iss.ID, strings.Join(iss.Labels, ", "))
	}
	return nil
}
