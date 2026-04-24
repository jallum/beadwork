package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/md"
	"github.com/jallum/beadwork/internal/repo"
)

type ReadyArgs struct {
	ParentID  string
	JSON      bool
	NoContext bool
}

func parseReadyArgs(raw []string) (ReadyArgs, error) {
	a, err := ParseArgs(raw, nil, []string{"--json", "--no-context"})
	if err != nil {
		return ReadyArgs{}, err
	}
	return ReadyArgs{
		ParentID:  a.PosFirst(),
		JSON:      a.JSON(),
		NoContext: a.Bool("--no-context"),
	}, nil
}

func cmdReady(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	ra, err := parseReadyArgs(args)
	if err != nil {
		return nil, err
	}

	var issues []*issue.Issue
	if ra.ParentID != "" {
		issues, err = store.ReadyScoped(ra.ParentID)
	} else {
		issues, err = store.Ready()
	}
	if err != nil {
		return nil, err
	}

	if ra.JSON {
		fprintJSON(w, issues)
		return nil, nil
	}

	if len(issues) == 0 {
		fmt.Fprintln(w, "no ready issues")
		return nil, nil
	}

	closedBlockers := store.HiddenBlockerSet(issues)
	now := store.Now()

	if ra.ParentID != "" {
		// Scoped mode: flat list, no parent grouping.
		for _, iss := range issues {
			fmt.Fprintln(w, md.IssueOneLinerWithDue(iss, now, closedBlockers))
		}
	} else {
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
			fmt.Fprintln(w, md.IssueOneLinerWithDue(iss, now, closedBlockers))
		}

		// Print groups.
		for _, parentID := range parentOrder {
			// Print parent as group header.
			parent, err := store.Get(parentID)
			if err == nil {
				fmt.Fprintln(w, md.IssueOneLinerWithDue(parent, now, closedBlockers))
			}
			// Print children indented.
			w.Push(2)
			for _, child := range groups[parentID] {
				fmt.Fprintln(w, md.IssueOneLinerWithDue(child, now, closedBlockers))
			}
			w.Pop()
		}
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

	return nil, nil
}
