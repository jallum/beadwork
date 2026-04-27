package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
)

type CreateArgs struct {
	Title       string
	ID          string
	Parent      string
	Priority    *int
	Type        string
	Description string
	DeferUntil  string
	Due         string
	Labels      []string
	JSON        bool
	Silent      bool
}

func parseCreateArgs(raw []string) (CreateArgs, error) {
	a, err := ParseArgs(raw,
		[]string{"--priority", "--type", "--description", "--defer", "--due", "--labels", "--parent", "--id"},
		[]string{"--json", "--silent"},
	)
	if err != nil {
		return CreateArgs{}, err
	}
	var ca CreateArgs
	pos := a.Pos()
	if len(pos) == 0 {
		return ca, fmt.Errorf("title is required")
	}
	if len(pos) > 1 {
		return ca, fmt.Errorf("unexpected arguments after title: %s\nhint: use quotes for multi-word titles, e.g. bw create \"Fix the bug\"", strings.Join(pos[1:], ", "))
	}
	ca.Title = pos[0]
	ca.ID = a.String("--id")
	ca.Type = a.String("--type")
	ca.Description = a.String("--description")
	ca.DeferUntil = a.String("--defer")
	ca.Due = a.String("--due")
	ca.Parent = a.String("--parent")
	ca.JSON = a.JSON()
	ca.Silent = a.Bool("--silent")
	if a.Has("--priority") {
		p, err := parsePriority(a.String("--priority"))
		if err != nil {
			return ca, err
		}
		ca.Priority = &p
	}
	if a.Has("--labels") {
		for _, l := range strings.Split(a.String("--labels"), ",") {
			l = strings.TrimSpace(l)
			if l != "" {
				ca.Labels = append(ca.Labels, l)
			}
		}
	}
	return ca, nil
}

func cmdCreate(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	ca, err := parseCreateArgs(args)
	if err != nil {
		return nil, err
	}

	now := store.Now()
	if ca.DeferUntil != "" {
		resolved, err := resolveDate(ca.DeferUntil, now)
		if err != nil {
			return nil, err
		}
		ca.DeferUntil = resolved
	}
	if ca.Due != "" {
		resolved, err := resolveDate(ca.Due, now)
		if err != nil {
			return nil, err
		}
		ca.Due = resolved
	}

	opts := issue.CreateOpts{
		ID:          ca.ID,
		Priority:    ca.Priority,
		Type:        ca.Type,
		Description: ca.Description,
		DeferUntil:  ca.DeferUntil,
		Due:         ca.Due,
		Parent:      ca.Parent,
	}

	iss, err := store.Create(ca.Title, opts)
	if err != nil {
		return nil, err
	}

	if len(ca.Labels) > 0 {
		iss, err = store.Label(iss.ID, ca.Labels, nil)
		if err != nil {
			return nil, err
		}
	}

	intent := fmt.Sprintf("create %s p%d %s %q", iss.ID, iss.Priority, iss.Type, iss.Title)
	if iss.Description != "" {
		intent += fmt.Sprintf(" description=%q", iss.Description)
	}
	if iss.Due != "" {
		intent += " due=" + iss.Due
	}
	if iss.Parent != "" {
		intent += " parent=" + iss.Parent
	}
	if err := store.Commit(intent); err != nil {
		return nil, fmt.Errorf("commit failed: %w", err)
	}

	if ca.Silent {
		fmt.Fprintln(w, iss.ID)
	} else if ca.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "created %s: %s\n", iss.ID, iss.Title)
	}
	return nil, nil
}
