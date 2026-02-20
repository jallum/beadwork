package main

import (
	"fmt"
	"io"
	"strconv"

	"github.com/jallum/beadwork/internal/issue"
)

func cmdCreate(args []string, w io.Writer) error {
	r, store, err := getInitialized()
	if err != nil {
		return err
	}
	opts := issue.CreateOpts{}
	var title string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--priority", "-p":
			if i+1 < len(args) {
				p, err := strconv.Atoi(args[i+1])
				if err != nil {
					return fmt.Errorf("invalid priority: %s", args[i+1])
				}
				opts.Priority = p
				i++
			}
		case "--type", "-t":
			if i+1 < len(args) {
				opts.Type = args[i+1]
				i++
			}
		case "--assignee", "-a":
			if i+1 < len(args) {
				opts.Assignee = args[i+1]
				i++
			}
		case "--description", "-d":
			if i+1 < len(args) {
				opts.Description = args[i+1]
				i++
			}
		case "--json":
			// handled after creation
		default:
			if title == "" {
				title = args[i]
			} else {
				title += " " + args[i]
			}
		}
	}

	if title == "" {
		return fmt.Errorf("title is required")
	}

	iss, err := store.Create(title, opts)
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("create %s p%d %s %q", iss.ID, iss.Priority, iss.Type, iss.Title)
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if hasFlag(args, "--json") {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "created %s: %s\n", iss.ID, iss.Title)
	}
	return nil
}
