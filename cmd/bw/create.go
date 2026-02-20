package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/jallum/beadwork/internal/issue"
)

func cmdCreate(args []string) {
	r, store := mustInitialized()
	opts := issue.CreateOpts{}
	var title string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--priority", "-p":
			if i+1 < len(args) {
				p, err := strconv.Atoi(args[i+1])
				if err != nil {
					fatal("invalid priority: " + args[i+1])
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
		default:
			if title == "" {
				title = args[i]
			} else {
				title += " " + args[i]
			}
		}
	}

	if title == "" {
		fatal("title is required")
	}

	iss, err := store.Create(title, opts)
	if err != nil {
		fatal(err.Error())
	}

	intent := fmt.Sprintf("create %s p%d %s %q", iss.ID, iss.Priority, iss.Type, iss.Title)
	if err := r.Commit(intent); err != nil {
		fatal("commit failed: " + err.Error())
	}

	if hasFlag(os.Args, "--json") {
		printJSON(iss)
	} else {
		fmt.Printf("created %s: %s\n", iss.ID, iss.Title)
	}
}
