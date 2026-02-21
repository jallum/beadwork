package main

import (
	"fmt"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

type ListArgs struct {
	Status   string
	Assignee string
	Priority *int
	Type     string
	Label    string
	Grep     string
	Limit    int
	LimitSet bool
	All      bool
	Deferred bool
	JSON     bool
}

func parseListArgs(raw []string) (ListArgs, error) {
	a, err := ParseArgs(raw,
		[]string{"--status", "--assignee", "--priority", "--type", "--label", "--limit", "--grep"},
		[]string{"--all", "--deferred", "--json"},
	)
	if err != nil {
		return ListArgs{}, err
	}
	la := ListArgs{
		Status:   a.String("--status"),
		Assignee: a.String("--assignee"),
		Type:     a.String("--type"),
		Label:    a.String("--label"),
		Grep:     a.String("--grep"),
		All:      a.Bool("--all"),
		Deferred: a.Bool("--deferred"),
		JSON:     a.JSON(),
		Limit:    10,
	}
	if a.Has("--priority") {
		p, err := parsePriority(a.String("--priority"))
		if err != nil {
			return la, err
		}
		la.Priority = &p
	}
	if a.Has("--limit") {
		la.Limit = a.Int("--limit")
		la.LimitSet = true
	}
	return la, nil
}

func cmdList(_ *repo.Repo, store *issue.Store, args []string, w Writer) error {
	la, err := parseListArgs(args)
	if err != nil {
		return err
	}

	filter := issue.Filter{
		Status:   la.Status,
		Assignee: la.Assignee,
		Priority: la.Priority,
		Type:     la.Type,
		Label:    la.Label,
		Grep:     la.Grep,
	}

	limit := la.Limit

	// Defaults: open status, limit 10. --all overrides both.
	// --deferred overrides status to "deferred".
	if la.Deferred {
		filter.Status = "deferred"
	} else if la.All {
		if !la.LimitSet {
			limit = 0
		}
	} else if la.Status == "" {
		filter.Statuses = []string{"open", "in_progress"}
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
			ps := PriorityStyle(iss.Priority)
			fmt.Fprintf(w, "%s %s [%s %s] [%s] - %s%s\n",
				issue.StatusIcon(iss.Status),
				iss.ID,
				w.Style("●", ps),
				w.Style(fmt.Sprintf("P%d", iss.Priority), ps),
				iss.Type,
				iss.Title,
				formatDeps(w, iss),
			)
		}
		if limit > 0 && len(issues) > limit {
			fmt.Fprintf(w, "... and %d more (use --limit or --all)\n", len(issues)-limit)
		}
	}
	return nil
}
