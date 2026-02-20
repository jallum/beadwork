package main

import (
	"fmt"
	"io"
)

type CloseArgs struct {
	ID     string
	Reason string
	JSON   bool
}

func parseCloseArgs(raw []string) (CloseArgs, error) {
	if len(raw) == 0 {
		return CloseArgs{}, fmt.Errorf("usage: bw close <id> [--reason <reason>]")
	}
	a := ParseArgs(raw[1:], "--reason")
	return CloseArgs{
		ID:     raw[0],
		Reason: a.String("--reason"),
		JSON:   a.JSON(),
	}, nil
}

func cmdClose(args []string, w io.Writer) error {
	ca, err := parseCloseArgs(args)
	if err != nil {
		return err
	}

	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	iss, err := store.Close(ca.ID)
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("close %s", iss.ID)
	if ca.Reason != "" {
		intent += fmt.Sprintf(" reason=%q", ca.Reason)
	}
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if ca.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "closed %s: %s\n", iss.ID, iss.Title)
	}
	return nil
}

type ReopenArgs struct {
	ID   string
	JSON bool
}

func parseReopenArgs(raw []string) (ReopenArgs, error) {
	a := ParseArgs(raw)
	id := a.PosFirst()
	if id == "" {
		return ReopenArgs{}, fmt.Errorf("usage: bw reopen <id>")
	}
	return ReopenArgs{ID: id, JSON: a.JSON()}, nil
}

func cmdReopen(args []string, w io.Writer) error {
	ra, err := parseReopenArgs(args)
	if err != nil {
		return err
	}

	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	iss, err := store.Reopen(ra.ID)
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("reopen %s", iss.ID)
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if ra.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "reopened %s: %s\n", iss.ID, iss.Title)
	}
	return nil
}
