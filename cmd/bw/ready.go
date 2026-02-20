package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
)

func cmdReady(args []string, w io.Writer) error {
	_, store, err := getInitialized()
	if err != nil {
		return err
	}

	issues, err := store.Ready()
	if err != nil {
		return err
	}

	a := ParseArgs(args)
	if a.JSON() {
		fprintJSON(w, issues)
		return nil
	}

	if len(issues) == 0 {
		fmt.Fprintln(w, "no ready issues")
		return nil
	}

	for _, iss := range issues {
		fmt.Fprintf(w, "%s %s %s P%d %s\n",
			issue.StatusIcon(iss.Status),
			iss.ID,
			issue.PriorityDot(iss.Priority),
			iss.Priority,
			iss.Title,
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
