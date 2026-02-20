package main

import (
	"fmt"
	"io"

	"github.com/jallum/beadwork/internal/issue"
)

func cmdList(args []string, w io.Writer) error {
	_, store, err := getInitialized()
	if err != nil {
		return err
	}

	a := ParseArgs(args, "--status", "--assignee", "--priority", "--type", "--label", "--limit")

	filter := issue.Filter{
		Status:   a.String("--status"),
		Assignee: a.String("--assignee"),
		Priority: a.Int("--priority"),
		Type:     a.String("--type"),
		Label:    a.String("--label"),
	}

	limit := 10
	if a.Has("--limit") {
		limit = a.Int("--limit")
	}

	showAll := a.Bool("--all")

	// Defaults: open status, limit 10. --all overrides both.
	if showAll {
		if !a.Has("--limit") {
			limit = 0
		}
	} else if !a.Has("--status") {
		filter.Status = "open"
	}

	issues, err := store.List(filter)
	if err != nil {
		return err
	}

	if a.JSON() {
		if limit > 0 && len(issues) > limit {
			issues = issues[:limit]
		}
		fprintJSON(w, issues)
	} else {
		if len(issues) == 0 {
			fmt.Fprintln(w, "no issues found")
			return nil
		}
		displayed := issues
		if limit > 0 && len(displayed) > limit {
			displayed = displayed[:limit]
		}
		for _, iss := range displayed {
			fmt.Fprintf(w, "%s %s [%s P%d] [%s] - %s\n",
				issue.StatusIcon(iss.Status),
				iss.ID,
				issue.PriorityDot(iss.Priority),
				iss.Priority,
				iss.Type,
				iss.Title,
			)
		}
		if limit > 0 && len(issues) > limit {
			fmt.Fprintf(w, "... and %d more (use --limit or --all)\n", len(issues)-limit)
		}
	}
	return nil
}
