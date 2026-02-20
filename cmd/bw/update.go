package main

import (
	"fmt"
	"io"
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

	a := ParseArgs(args[1:], "--title", "--description", "--priority", "--assignee", "--type", "--status")

	opts := issue.UpdateOpts{}
	var changes []string

	if a.Has("--title") {
		v := a.String("--title")
		opts.Title = &v
		changes = append(changes, "title="+v)
	}
	if a.Has("--description") {
		v := a.String("--description")
		opts.Description = &v
		changes = append(changes, "description=...")
	}
	if a.Has("--priority") {
		p, _, err := a.IntErr("--priority")
		if err != nil {
			return err
		}
		opts.Priority = &p
		changes = append(changes, "priority="+a.String("--priority"))
	}
	if a.Has("--assignee") {
		v := a.String("--assignee")
		opts.Assignee = &v
		changes = append(changes, "assignee="+v)
	}
	if a.Has("--type") {
		v := a.String("--type")
		opts.Type = &v
		changes = append(changes, "type="+v)
	}
	if a.Has("--status") {
		v := a.String("--status")
		opts.Status = &v
		changes = append(changes, "status="+v)
	}

	iss, err := store.Update(id, opts)
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("update %s %s", iss.ID, strings.Join(changes, " "))
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if a.JSON() {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "updated %s\n", iss.ID)
	}
	return nil
}
