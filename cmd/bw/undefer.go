package main

import (
	"fmt"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

type UndeferArgs struct {
	ID   string
	JSON bool
}

func parseUndeferArgs(raw []string) (UndeferArgs, error) {
	if len(raw) == 0 {
		return UndeferArgs{}, fmt.Errorf("usage: bw undefer <id>")
	}
	a, err := ParseArgs(raw[1:], nil, []string{"--json"})
	if err != nil {
		return UndeferArgs{}, err
	}
	return UndeferArgs{ID: raw[0], JSON: a.JSON()}, nil
}

func cmdUndefer(r *repo.Repo, store *issue.Store, args []string, w Writer) error {
	ua, err := parseUndeferArgs(args)
	if err != nil {
		return err
	}

	status := "open"
	emptyDefer := ""
	iss, err := store.Update(ua.ID, issue.UpdateOpts{
		Status:     &status,
		DeferUntil: &emptyDefer,
	})
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("undefer %s", iss.ID)
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if ua.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "undeferred %s\n", iss.ID)
	}
	return nil
}
