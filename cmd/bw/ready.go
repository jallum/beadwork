package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
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

func cmdReady(store *issue.Store, args []string, w Writer) error {
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

	// Partition into standalone issues and groups keyed by parent ID.
	// Preserve insertion order for parents.
	var standalone []*issue.Issue
	groups := make(map[string][]*issue.Issue)
	var parentOrder []string
	for _, iss := range issues {
		if iss.Parent == "" {
			standalone = append(standalone, iss)
			continue
		}
		if _, seen := groups[iss.Parent]; !seen {
			parentOrder = append(parentOrder, iss.Parent)
		}
		groups[iss.Parent] = append(groups[iss.Parent], iss)
	}

	// Remove parents from standalone — they'll be printed as group headers.
	var filteredStandalone []*issue.Issue
	for _, iss := range standalone {
		if _, isParent := groups[iss.ID]; !isParent {
			filteredStandalone = append(filteredStandalone, iss)
		}
	}

	// Print standalone issues.
	for _, iss := range filteredStandalone {
		fprintReadyLine(w, iss)
	}

	// Print groups, separated by blank lines.
	for i, parentID := range parentOrder {
		if len(filteredStandalone) > 0 || i > 0 {
			fmt.Fprintln(w)
		}
		// Print parent as group header.
		parent, err := store.Get(parentID)
		if err == nil {
			fprintReadyLine(w, parent)
		}
		// Print children indented.
		w.Push(2)
		for _, child := range groups[parentID] {
			fprintReadyLine(w, child)
		}
		w.Pop()
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

func fprintReadyLine(w Writer, iss *issue.Issue) {
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
