package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
)

func cmdUpdate(args []string, w io.Writer) error {
	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: bw update <id> [flags]")
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
					return fmt.Errorf("invalid priority: %s", rest[i+1])
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
		return err
	}

	intent := fmt.Sprintf("update %s %s", iss.ID, strings.Join(changes, " "))
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if hasFlag(args, "--json") {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "updated %s\n", iss.ID)
	}
	return nil
}
