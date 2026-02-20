package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
)

type CreateArgs struct {
	Title       string
	Priority    int
	PrioritySet bool
	Type        string
	Assignee    string
	Description string
	DeferUntil  string
	JSON        bool
}

func parseCreateArgs(raw []string) (CreateArgs, error) {
	a := ParseArgs(raw, "--priority", "--type", "--assignee", "--description", "--defer")
	var ca CreateArgs
	pos := a.Pos()
	if len(pos) == 0 {
		return ca, fmt.Errorf("title is required")
	}
	if len(pos) > 1 {
		return ca, fmt.Errorf("unexpected arguments after title: %s\nhint: use quotes for multi-word titles, e.g. bw create \"Fix the bug\"", strings.Join(pos[1:], ", "))
	}
	ca.Title = pos[0]
	ca.Type = a.String("--type")
	ca.Assignee = a.String("--assignee")
	ca.Description = a.String("--description")
	ca.DeferUntil = a.String("--defer")
	ca.JSON = a.JSON()
	if p, set, err := a.IntErr("--priority"); err != nil {
		return ca, err
	} else if set {
		ca.Priority = p
		ca.PrioritySet = true
	}
	if ca.DeferUntil != "" {
		if err := validateDate(ca.DeferUntil); err != nil {
			return ca, err
		}
	}
	return ca, nil
}

func cmdCreate(args []string, w io.Writer) error {
	ca, err := parseCreateArgs(args)
	if err != nil {
		return err
	}

	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	opts := issue.CreateOpts{
		Type:        ca.Type,
		Assignee:    ca.Assignee,
		Description: ca.Description,
		DeferUntil:  ca.DeferUntil,
	}
	if ca.PrioritySet {
		opts.Priority = ca.Priority
	}

	iss, err := store.Create(ca.Title, opts)
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("create %s p%d %s %q", iss.ID, iss.Priority, iss.Type, iss.Title)
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if ca.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "created %s: %s\n", iss.ID, iss.Title)
	}
	return nil
}
