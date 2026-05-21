package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
)

type LabelArgs struct {
	ID     string
	Add    []string
	Remove []string
	JSON   bool
}

func parseLabelArgs(raw []string) (LabelArgs, error) {
	if len(raw) < 2 {
		return LabelArgs{}, fmt.Errorf("usage: bw label <id> +label [-label]")
	}
	la := LabelArgs{ID: raw[0], JSON: hasFlag(raw, "--json")}
	for _, arg := range raw[1:] {
		if arg == "--json" {
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return LabelArgs{}, fmt.Errorf("unknown flag: %s", arg)
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

func cmdLabel(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	la, err := parseLabelArgs(args)
	if err != nil {
		return nil, err
	}

	var iss *issue.Issue
	err = commitWithRetry(store, commitMaxRetries, func() (string, error) {
		var lerr error
		iss, lerr = store.Label(la.ID, la.Add, la.Remove)
		if lerr != nil {
			return "", lerr
		}
		var parts []string
		for _, l := range la.Add {
			parts = append(parts, "+"+l)
		}
		for _, l := range la.Remove {
			parts = append(parts, "-"+l)
		}
		return fmt.Sprintf("label %s %s", iss.ID, strings.Join(parts, " ")), nil
	})
	if err != nil {
		return nil, err
	}

	if la.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "labeled %s: %s\n", iss.ID, strings.Join(iss.Labels, ", "))
	}
	return nil, nil
}
