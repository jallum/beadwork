package main

import (
	"fmt"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/md"
)

type CloseArgs struct {
	ID        string
	Reason    string
	Recursive bool
	JSON      bool
}

func parseCloseArgs(raw []string) (CloseArgs, error) {
	a, err := ParseArgs(raw, []string{"--reason"}, []string{"--recursive", "--json"})
	if err != nil {
		return CloseArgs{}, err
	}
	id := a.PosFirst()
	if id == "" {
		return CloseArgs{}, fmt.Errorf("usage: bw close <id> [--reason <reason>] [--recursive]")
	}
	return CloseArgs{
		ID:        id,
		Reason:    a.String("--reason"),
		Recursive: a.Bool("--recursive"),
		JSON:      a.JSON(),
	}, nil
}

func cmdClose(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	ca, err := parseCloseArgs(args)
	if err != nil {
		return nil, err
	}

	if ca.Recursive {
		return cmdCloseRecursive(store, ca, w)
	}

	var iss *issue.Issue
	var unblocked []*issue.Issue

	err = commitWithRetry(store, commitMaxRetries, func() (string, error) {
		var cerr error
		iss, cerr = store.Close(ca.ID, ca.Reason)
		if cerr != nil {
			return "", cerr
		}
		unblocked, cerr = store.NewlyUnblocked(iss.ID)
		if cerr != nil {
			return "", cerr
		}
		intent := fmt.Sprintf("close %s", iss.ID)
		if ca.Reason != "" {
			intent += fmt.Sprintf(" reason=%q", ca.Reason)
		}
		for _, u := range unblocked {
			intent += fmt.Sprintf("\nunblocked %s", u.ID)
		}
		return intent, nil
	})
	if err != nil {
		return nil, err
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
	return nil, nil
}

// cmdCloseRecursive closes an issue and its entire subtree in a single commit.
func cmdCloseRecursive(store *issue.Store, ca CloseArgs, w Writer) (*config.Config, error) {
	var result *issue.SubtreeCloseResult

	err := commitWithRetry(store, commitMaxRetries, func() (string, error) {
		var cerr error
		result, cerr = store.CloseSubtree(ca.ID, ca.Reason)
		if cerr != nil {
			return "", cerr
		}
		if len(result.Closed) == 0 {
			return "", fmt.Errorf("nothing to close: %s and its subtree are already closed", ca.ID)
		}

		intent := ""
		for i, iss := range result.Closed {
			if i > 0 {
				intent += "\n"
			}
			intent += fmt.Sprintf("close %s", iss.ID)
			if iss.CloseReason != "" {
				intent += fmt.Sprintf(" reason=%q", iss.CloseReason)
			}
		}
		for _, u := range result.Unblocked {
			intent += fmt.Sprintf("\nunblocked %s", u.ID)
		}
		return intent, nil
	})
	if err != nil {
		return nil, err
	}

	if ca.JSON {
		if result.Closed == nil {
			result.Closed = []*issue.Issue{}
		}
		if result.Skipped == nil {
			result.Skipped = []*issue.Issue{}
		}
		if result.Unblocked == nil {
			result.Unblocked = []*issue.Issue{}
		}
		fprintJSON(w, result)
		return nil, nil
	}

	fmt.Fprintf(w, "closed %d issue(s) under {id:%s}:\n", len(result.Closed), ca.ID)
	w.Push(2)
	for _, iss := range result.Closed {
		fmt.Fprintf(w, "{id:%s}: ~~%s~~\n", iss.ID, md.Escape(iss.Title))
	}
	w.Pop()
	if len(result.Skipped) > 0 {
		fmt.Fprintf(w, "\n%d already closed, skipped.\n", len(result.Skipped))
	}
	if len(result.Unblocked) > 0 {
		fmt.Fprintln(w)
		w.Push(2)
		for _, u := range result.Unblocked {
			fmt.Fprintf(w, "unblocked {id:%s}: %s\n", u.ID, md.Escape(u.Title))
		}
		w.Pop()
		fmt.Fprintln(w)
		if len(result.Unblocked) == 1 {
			fmt.Fprintf(w, "Next: `bw start %s` to begin, or `bw ready` for all options.\n", result.Unblocked[0].ID)
		} else {
			fmt.Fprintln(w, "Next: `bw ready` to see available work.")
		}
	}
	return nil, nil
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

func cmdReopen(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	ra, err := parseReopenArgs(args)
	if err != nil {
		return nil, err
	}

	var iss *issue.Issue
	err = commitWithRetry(store, commitMaxRetries, func() (string, error) {
		var rerr error
		iss, rerr = store.Reopen(ra.ID)
		if rerr != nil {
			return "", rerr
		}
		return fmt.Sprintf("reopen %s", iss.ID), nil
	})
	if err != nil {
		return nil, err
	}

	if ra.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "reopened %s: %s\n", iss.ID, iss.Title)
	}
	return nil, nil
}
