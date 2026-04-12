package main

import (
	"errors"
	"fmt"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/md"
	"github.com/jallum/beadwork/internal/treefs"
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

const closeMaxRetries = 3

func cmdClose(store *issue.Store, args []string, w Writer) error {
	ca, err := parseCloseArgs(args)
	if err != nil {
		return err
	}

	var iss *issue.Issue
	var unblocked []*issue.Issue

	// Retry loop: on CAS conflict, refresh the store and retry the close
	// operation up to closeMaxRetries times.
	for attempt := range closeMaxRetries {
		if attempt > 0 {
			store.ClearCache()
			if err := store.Refresh(); err != nil {
				return fmt.Errorf("refresh after conflict: %w", err)
			}
		}

		iss, err = store.Close(ca.ID, ca.Reason)
		if err != nil {
			return err
		}

		unblocked, err = store.NewlyUnblocked(iss.ID)
		if err != nil {
			return err
		}

		intent := fmt.Sprintf("close %s", iss.ID)
		if ca.Reason != "" {
			intent += fmt.Sprintf(" reason=%q", ca.Reason)
		}
		for _, u := range unblocked {
			intent += fmt.Sprintf("\nunblocked %s", u.ID)
		}

		err = store.Commit(intent)
		if err == nil {
			break
		}
		if !errors.Is(err, treefs.ErrRefMoved) {
			return fmt.Errorf("commit failed: %w", err)
		}
		// CAS conflict — retry
	}
	if err != nil {
		return fmt.Errorf("commit failed after %d attempts: %w", closeMaxRetries, err)
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
		fmt.Fprintf(w, "closed {id:%s}: ~~%s~~\n", iss.ID, md.Escape(iss.Title))
		if len(unblocked) > 0 {
			w.Push(2)
			for _, u := range unblocked {
				fmt.Fprintf(w, "unblocked {id:%s}: %s\n", u.ID, md.Escape(u.Title))
			}
			w.Pop()
			fmt.Fprintln(w)
			if len(unblocked) == 1 {
				fmt.Fprintf(w, "Next: `bw start %s` to begin, or `bw ready` for all options.\n", unblocked[0].ID)
			} else {
				fmt.Fprintln(w, "Next: `bw ready` to see available work.")
			}
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

func cmdReopen(store *issue.Store, args []string, w Writer) error {
	ra, err := parseReopenArgs(args)
	if err != nil {
		return err
	}

	iss, err := store.Reopen(ra.ID)
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("reopen %s", iss.ID)
	if err := store.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if ra.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "reopened %s: %s\n", iss.ID, iss.Title)
	}
	return nil
}
