package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
	"github.com/jallum/beadwork/internal/template"

	"github.com/jallum/beadwork/prompts"
)

func cmdPrime(store *issue.Store, _ []string, w Writer) error {
	r := store.Committer.(*repo.Repo)
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

	md := func(s string) { fmt.Fprint(w, styleMD(w, s)) }

	md("\n## Current State\n\n")
	md(fmt.Sprintf("%d open, %d in progress, %d closed\n", openCount, ipCount, closedCount))
	md(fmt.Sprintf("%d ready (unblocked)\n", len(ready)))

	if ipCount > 0 {
		md("\n**In progress:**\n")
		w.Push(2)
		for _, iss := range all {
			if iss.Status == "in_progress" {
				md(fmt.Sprintf("`%-*s`  P%d  %s\n", idw, iss.ID, iss.Priority, iss.Title))
				if n := len(iss.Comments); n > 0 {
					last := iss.Comments[n-1]
					text := last.Text
					if len(text) > 60 {
						text = text[:57] + "..."
					}
					w.Push(2)
					md(fmt.Sprintf("└ %q (%s)\n", text, relativeTime(last.Timestamp)))
					w.Pop()
				}
			}
		}
		w.Pop()
	}

	if len(ready) > 0 {
		md("\n**Ready for work:**\n")
		w.Push(2)
		for _, iss := range ready {
			md(fmt.Sprintf("`%-*s`  P%d  %s\n", idw, iss.ID, iss.Priority, iss.Title))
		}
		w.Pop()
	}

	return nil
}
