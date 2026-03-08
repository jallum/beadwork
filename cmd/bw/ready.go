package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/md"
	"github.com/jallum/beadwork/internal/repo"
)

type ReadyArgs struct {
	JSON      bool
	NoContext bool
}

func parseReadyArgs(raw []string) (ReadyArgs, error) {
	a, err := ParseArgs(raw, nil, []string{"--json", "--no-context"})
	if err != nil {
		return ReadyArgs{}, err
	}
	return ReadyArgs{JSON: a.JSON(), NoContext: a.Bool("--no-context")}, nil
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

	closedBlockers := store.ClosedBlockerSet(issues)

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
		fmt.Fprintln(w, md.IssueOneLinerFiltered(iss, closedBlockers))
	}

	// Print groups.
	for _, parentID := range parentOrder {
		// Print parent as group header.
		parent, err := store.Get(parentID)
		if err == nil {
			fmt.Fprintln(w, md.IssueOneLinerFiltered(parent, closedBlockers))
		}
		// Print children indented.
		w.Push(2)
		for _, child := range groups[parentID] {
			fmt.Fprintln(w, md.IssueOneLinerFiltered(child, closedBlockers))
		}
		w.Pop()
	}

	// TTY-only footer
	if w.IsTTY() {
		fmt.Fprintln(w)
		fmt.Fprintln(w, strings.Repeat("-", 80))
		fmt.Fprintf(w, "Ready: %d issues with no blockers\n", len(issues))
		fmt.Fprintln(w)

		var legend []string
		for _, s := range issue.Statuses {
			legend = append(legend, s.Icon+" "+s.Name)
		}
		legend = append(legend, "⊘ blocked")
		fmt.Fprintf(w, "Status: %s\n", strings.Join(legend, "  "))
	}

	// Git context (visible in both TTY and piped output)
	if !ra.NoContext {
		if r, ok := store.Committer.(*repo.Repo); ok {
			fmt.Fprintln(w)
			fprintGitContext(w, r.GetGitContext())
		}
	}

	return nil
}
