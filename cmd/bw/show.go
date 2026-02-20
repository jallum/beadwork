package main

import (
	"fmt"
	"io"
)

type ShowArgs struct {
	ID   string
	JSON bool
}

func parseShowArgs(raw []string) (ShowArgs, error) {
	a := ParseArgs(raw)
	id := a.PosFirst()
	if id == "" {
		return ShowArgs{}, fmt.Errorf("usage: bw show <id>")
	}
	return ShowArgs{ID: id, JSON: a.JSON()}, nil
}

func cmdShow(args []string, w io.Writer) error {
	sa, err := parseShowArgs(args)
	if err != nil {
		return err
	}

	_, store, err := getInitialized()
	if err != nil {
		return err
	}

	iss, err := store.Get(sa.ID)
	if err != nil {
		return err
	}

	if sa.JSON {
		fprintJSON(w, iss)
	} else {
		fprintIssue(w, iss)
	}
	return nil
}
