package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/j5n/beadwork/internal/issue"
)

func cmdUpdate(args []string) {
	r, store := mustInitialized()

	if len(args) == 0 {
		fatal("usage: bw update <id> [flags]")
	}
	id := args[0]
	rest := args[1:]

	opts := issue.UpdateOpts{}
	var changes []string

	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--title":
			if i+1 < len(rest) {
				opts.Title = &rest[i+1]
				changes = append(changes, "title="+rest[i+1])
				i++
			}
		case "--description", "-d":
			if i+1 < len(rest) {
				opts.Description = &rest[i+1]
				changes = append(changes, "description=...")
				i++
			}
		case "--priority", "-p":
			if i+1 < len(rest) {
				p, err := strconv.Atoi(rest[i+1])
				if err != nil {
					fatal("invalid priority: " + rest[i+1])
				}
				opts.Priority = &p
				changes = append(changes, "priority="+rest[i+1])
				i++
			}
		case "--assignee", "-a":
			if i+1 < len(rest) {
				opts.Assignee = &rest[i+1]
				changes = append(changes, "assignee="+rest[i+1])
				i++
			}
		case "--type", "-t":
			if i+1 < len(rest) {
				opts.Type = &rest[i+1]
				changes = append(changes, "type="+rest[i+1])
				i++
			}
		case "--status", "-s":
			if i+1 < len(rest) {
				opts.Status = &rest[i+1]
				changes = append(changes, "status="+rest[i+1])
				i++
			}
		}
	}

	iss, err := store.Update(id, opts)
	if err != nil {
		fatal(err.Error())
	}

	intent := fmt.Sprintf("update %s %s", iss.ID, strings.Join(changes, " "))
	if err := r.Commit(intent); err != nil {
		fatal("commit failed: " + err.Error())
	}

	if hasFlag(os.Args, "--json") {
		printJSON(iss)
	} else {
		fmt.Printf("updated %s\n", iss.ID)
	}
}
