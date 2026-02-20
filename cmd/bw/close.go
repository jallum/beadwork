package main

import (
	"fmt"
	"io"
)

func cmdClose(args []string, w io.Writer) error {
	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: bw close <id> [--reason <reason>]")
	}
	id := args[0]
	a := ParseArgs(args[1:], "--reason")

	iss, err := store.Close(id)
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("close %s", iss.ID)
	if reason := a.String("--reason"); reason != "" {
		intent += fmt.Sprintf(" reason=%q", reason)
	}
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if a.JSON() {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "closed %s: %s\n", iss.ID, iss.Title)
	}
	return nil
}

func cmdReopen(args []string, w io.Writer) error {
	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	a := ParseArgs(args)

	id := a.PosFirst()
	if id == "" {
		return fmt.Errorf("usage: bw reopen <id>")
	}

	iss, err := store.Reopen(id)
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("reopen %s", iss.ID)
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if a.JSON() {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "reopened %s: %s\n", iss.ID, iss.Title)
	}
	return nil
}
