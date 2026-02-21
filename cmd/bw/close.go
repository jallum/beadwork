package main

import (
	"fmt"

	"github.com/jallum/beadwork/internal/issue"
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
	a, err := ParseArgs(raw[1:], []string{"--reason"}, []string{"--json"})
	if err != nil {
		return CloseArgs{}, err
	}
	return CloseArgs{
		ID:     raw[0],
		Reason: a.String("--reason"),
		JSON:   a.JSON(),
	}, nil
}

func cmdClose(args []string, w Writer) error {
	ca, err := parseCloseArgs(args)
	if err != nil {
		return err
	}

	r, store, err := getInitializedRepo()
	if err != nil {
		return err
	}

	iss, err := store.Close(ca.ID, ca.Reason)
	if err != nil {
		return err
	}

	unblocked, err := store.NewlyUnblocked(iss.ID)
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
		result := issue.CloseResult{
			Issue:     iss,
			Unblocked: unblocked,
		}
		if result.Unblocked == nil {
			result.Unblocked = []*issue.Issue{}
		}
		fprintJSON(w, result)
	} else {
		fmt.Fprintf(w, "closed %s: %s\n", iss.ID, iss.Title)
		if len(unblocked) > 0 {
			w.Push(2)
			for _, u := range unblocked {
				fmt.Fprintf(w, "unblocked %s: %s\n", u.ID, u.Title)
			}
			w.Pop()
		}
	}
	return nil
}

type ReopenArgs struct {
	ID   string
	JSON bool
}

func parseReopenArgs(raw []string) (ReopenArgs, error) {
	a, err := ParseArgs(raw, nil, []string{"--json"})
	if err != nil {
		return ReopenArgs{}, err
	}
	id := a.PosFirst()
	if id == "" {
		return ReopenArgs{}, fmt.Errorf("usage: bw reopen <id>")
	}
	return ReopenArgs{ID: id, JSON: a.JSON()}, nil
}

func cmdReopen(args []string, w Writer) error {
	ra, err := parseReopenArgs(args)
	if err != nil {
		return err
	}

	r, store, err := getInitializedRepo()
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
