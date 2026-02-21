package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

type ReadyArgs struct {
	JSON bool
}

func parseReadyArgs(raw []string) (ReadyArgs, error) {
	a, err := ParseArgs(raw, nil, []string{"--json"})
	if err != nil {
		return ReadyArgs{}, err
	}
	return ReadyArgs{JSON: a.JSON()}, nil
}

func cmdReady(_ *repo.Repo, store *issue.Store, args []string, w Writer) error {
	ra, err := parseReadyArgs(args)
	if err != nil {
		return err
	}

	issues, err := store.Ready()
	if err != nil {
		return err
	}

	if ra.JSON {
		fprintJSON(w, issues)
		return nil
	}

	if len(issues) == 0 {
		fmt.Fprintln(w, "no ready issues")
		return nil
	}

	for _, iss := range issues {
		ps := PriorityStyle(iss.Priority)
		fmt.Fprintf(w, "%s %s %s %s %s%s\n",
			issue.StatusIcon(iss.Status),
			iss.ID,
			w.Style("●", ps),
			w.Style(fmt.Sprintf("P%d", iss.Priority), ps),
			iss.Title,
			formatDeps(w, iss),
		)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, strings.Repeat("-", 80))
	fmt.Fprintf(w, "Ready: %d issues with no blockers\n", len(issues))
	fmt.Fprintln(w)

	var legend []string
	for _, s := range issue.Statuses {
		legend = append(legend, s.Icon+" "+s.Name)
	}
	fmt.Fprintf(w, "Status: %s\n", strings.Join(legend, "  "))
	return nil
}
