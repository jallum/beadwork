package main

import (
	"fmt"
	"io"

	"github.com/jallum/beadwork/internal/issue"
)

type ListArgs struct {
	Status   string
	Assignee string
	Priority int
	Type     string
	Label    string
	Limit    int
	LimitSet bool
	All      bool
	JSON     bool
}

func parseListArgs(raw []string) (ListArgs, error) {
	a := ParseArgs(raw, "--status", "--assignee", "--priority", "--type", "--label", "--limit")
	la := ListArgs{
		Status:   a.String("--status"),
		Assignee: a.String("--assignee"),
		Priority: a.Int("--priority"),
		Type:     a.String("--type"),
		Label:    a.String("--label"),
		All:      a.Bool("--all"),
		JSON:     a.JSON(),
		Limit:    10,
	}
	if a.Has("--limit") {
		la.Limit = a.Int("--limit")
		la.LimitSet = true
	}
	return la, nil
}

func cmdList(args []string, w io.Writer) error {
	la, err := parseListArgs(args)
	if err != nil {
		return err
	}

	_, store, err := getInitialized()
	if err != nil {
		return err
	}

	filter := issue.Filter{
		Status:   la.Status,
		Assignee: la.Assignee,
		Priority: la.Priority,
		Type:     la.Type,
		Label:    la.Label,
	}

	limit := la.Limit

	// Defaults: open status, limit 10. --all overrides both.
	if la.All {
		if !la.LimitSet {
			limit = 0
		}
	} else if la.Status == "" {
		filter.Status = "open"
	}

	issues, err := store.List(filter)
	if err != nil {
		return err
	}

	if la.JSON {
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
