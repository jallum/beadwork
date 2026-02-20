package main

import (
	"fmt"
	"io"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/prompts"
)

func cmdPrime(w io.Writer) error {
	_, store, err := getInitialized()
	if err != nil {
		return err
	}

	fmt.Fprint(w, prompts.Prime)

	// Dynamic section: current project state
	ready, _ := store.Ready()
	all, _ := store.List(issue.Filter{})

	openCount := 0
	ipCount := 0
	closedCount := 0
	for _, iss := range all {
		switch iss.Status {
		case "open":
			openCount++
		case "in_progress":
			ipCount++
		case "closed":
			closedCount++
		}
	}

	fmt.Fprintln(w, "## Current State")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %d open, %d in progress, %d closed\n", openCount, ipCount, closedCount)
	fmt.Fprintf(w, "  %d ready (unblocked)\n", len(ready))
	fmt.Fprintln(w)

	if ipCount > 0 {
		fmt.Fprintln(w, "  In progress:")
		for _, iss := range all {
			if iss.Status == "in_progress" {
				fmt.Fprintf(w, "    %-14s p%d %s\n", iss.ID, iss.Priority, iss.Title)
			}
		}
		fmt.Fprintln(w)
	}

	if len(ready) > 0 {
		fmt.Fprintln(w, "  Ready for work:")
		for _, iss := range ready {
			fmt.Fprintf(w, "    %-14s p%d %s\n", iss.ID, iss.Priority, iss.Title)
		}
		fmt.Fprintln(w)
	}
	return nil
}
