package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/template"
	"github.com/jallum/beadwork/prompts"
)

func cmdPrime(w Writer) error {
	r, store, err := getInitializedRepo()
	if err != nil {
		return err
	}

	out := template.Process(prompts.Prime, r.ListConfig())
	fmt.Fprint(w, styleMD(w, strings.TrimRight(out, "\n")))
	fmt.Fprintln(w)

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

	// Find max ID length for column alignment.
	idw := 0
	for _, iss := range all {
		if iss.Status == "in_progress" && len(iss.ID) > idw {
			idw = len(iss.ID)
		}
	}
	for _, iss := range ready {
		if len(iss.ID) > idw {
			idw = len(iss.ID)
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "\n## Current State\n\n")
	fmt.Fprintf(&sb, "%d open, %d in progress, %d closed\n", openCount, ipCount, closedCount)
	fmt.Fprintf(&sb, "%d ready (unblocked)\n", len(ready))

	if ipCount > 0 {
		fmt.Fprintf(&sb, "\n**In progress:**\n")
		for _, iss := range all {
			if iss.Status == "in_progress" {
				fmt.Fprintf(&sb, "  `%-*s`  P%d  %s\n", idw, iss.ID, iss.Priority, iss.Title)
				if n := len(iss.Comments); n > 0 {
					last := iss.Comments[n-1]
					text := last.Text
					if len(text) > 60 {
						text = text[:57] + "..."
					}
					fmt.Fprintf(&sb, "    └ %q (%s)\n", text, relativeTime(last.Timestamp))
				}
			}
		}
	}

	if len(ready) > 0 {
		fmt.Fprintf(&sb, "\n**Ready for work:**\n")
		for _, iss := range ready {
			fmt.Fprintf(&sb, "  `%-*s`  P%d  %s\n", idw, iss.ID, iss.Priority, iss.Title)
		}
	}

	fmt.Fprintln(w, styleMD(w, sb.String()))
	return nil
}
