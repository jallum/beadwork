package main

import (
	"fmt"
	"strings"

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
	Labels      []string
	JSON        bool
	Silent      bool
}

func parseCreateArgs(raw []string) (CreateArgs, error) {
	a, err := ParseArgs(raw,
		[]string{"--priority", "--type", "--description", "--defer", "--labels", "--parent", "--id"},
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
	if ca.DeferUntil != "" {
		if err := validateDate(ca.DeferUntil); err != nil {
			return ca, err
		}
	}
	return ca, nil
}

func cmdCreate(store *issue.Store, args []string, w Writer) error {
	ca, err := parseCreateArgs(args)
	if err != nil {
		return err
	}

	opts := issue.CreateOpts{
		ID:          ca.ID,
		Priority:    ca.Priority,
		Type:        ca.Type,
		Description: ca.Description,
		DeferUntil:  ca.DeferUntil,
		Parent:      ca.Parent,
	}

	iss, err := store.Create(ca.Title, opts)
	if err != nil {
		return err
	}

	if len(ca.Labels) > 0 {
		iss, err = store.Label(iss.ID, ca.Labels, nil)
		if err != nil {
			return err
		}
	}

	intent := fmt.Sprintf("create %s p%d %s %q", iss.ID, iss.Priority, iss.Type, iss.Title)
	if iss.Parent != "" {
		intent += " parent=" + iss.Parent
	}
	if err := store.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if ca.Silent {
		fmt.Fprintln(w, iss.ID)
	} else if ca.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "created %s: %s\n", iss.ID, iss.Title)
	}
	return nil
}
