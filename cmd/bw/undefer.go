package main

import (
	"fmt"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
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

func cmdUndefer(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	ua, err := parseUndeferArgs(args)
	if err != nil {
		return nil, err
	}

	status := "open"
	emptyDefer := ""
	var iss *issue.Issue
	err = commitWithRetry(store, commitMaxRetries, func() (string, error) {
		var uerr error
		iss, uerr = store.Update(ua.ID, issue.UpdateOpts{
			Status:     &status,
			DeferUntil: &emptyDefer,
		})
		if uerr != nil {
			return "", uerr
		}
		return fmt.Sprintf("undefer %s", iss.ID), nil
	})
	if err != nil {
		return nil, err
	}

	if ua.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "undeferred %s\n", iss.ID)
	}
	return nil, nil
}
