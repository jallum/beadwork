package main

import (
	"fmt"
	"io"

	"github.com/jallum/beadwork/internal/issue"
)

func cmdCreate(args []string, w io.Writer) error {
	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	a := ParseArgs(args, "--priority", "--type", "--assignee", "--description")

	title := a.PosJoined()
	if title == "" {
		return fmt.Errorf("title is required")
	}

	opts := issue.CreateOpts{
		Type:        a.String("--type"),
		Assignee:    a.String("--assignee"),
		Description: a.String("--description"),
	}
	if p, set, err := a.IntErr("--priority"); err != nil {
		return err
	} else if set {
		opts.Priority = p
	}

	iss, err := store.Create(title, opts)
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("create %s p%d %s %q", iss.ID, iss.Priority, iss.Type, iss.Title)
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if a.JSON() {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "created %s: %s\n", iss.ID, iss.Title)
	}
	return nil
}
