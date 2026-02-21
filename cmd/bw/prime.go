package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
	"github.com/jallum/beadwork/internal/template"

	"github.com/jallum/beadwork/prompts"
)

func cmdPrime(store *issue.Store, _ []string, w Writer) error {
	r := store.Committer.(*repo.Repo)
	md := func(s string) { fmt.Fprint(w, styleMD(w, s)) }

	var buf bytes.Buffer
	firstFlush := true

	flush := func() {
		s := strings.Trim(buf.String(), "\n")
		buf.Reset()
		if s == "" {
			return
		}
		if !firstFlush {
			fmt.Fprintln(w)
		}
		fmt.Fprint(w, styleMD(w, s))
		fmt.Fprintln(w)
		firstFlush = false
	}

	sections := map[string]func(io.Writer){
		"STATE": func(_ io.Writer) {
			flush()
			primeState(store, w, md)
		},
	}

	template.Process(&buf, prompts.Prime, r.ListConfig(), sections)
	flush()

	return nil
}

func primeState(store *issue.Store, w Writer, md func(string)) {
	snap, err := issue.NewSnapshot(store)
	if err != nil {
		return
	}

	ready := snap.Ready()

	// Derive counts and in-progress list from the single snapshot.
	openCount := 0
	ipCount := 0
	closedCount := 0
	var inProgress []*issue.Issue
	for _, iss := range snap.Issues {
		switch iss.Status {
		case "open":
			openCount++
		case "in_progress":
			ipCount++
			inProgress = append(inProgress, iss)
		case "closed":
			closedCount++
		}
	}

	// Find max ID length for column alignment.
	idw := 0
	for _, iss := range inProgress {
		if len(iss.ID) > idw {
			idw = len(iss.ID)
		}
	}
	for _, iss := range ready {
		if len(iss.ID) > idw {
			idw = len(iss.ID)
		}
	}

	md(fmt.Sprintf("\n%d open, %d in progress, %d closed\n", openCount, ipCount, closedCount))
	md(fmt.Sprintf("%d ready (unblocked)\n", len(ready)))

	if ipCount > 0 {
		md("\n**In progress:**\n")
		w.Push(2)
		for _, iss := range inProgress {
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
}
