package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/md"
)

type ListArgs struct {
	Status   string
	Assignee string
	Priority *int
	Type     string
	Label    string
	Grep     string
	Parent   string
	Limit    int
	LimitSet bool
	All      bool
	Deferred bool
	Overdue  bool
	JSON     bool
}

func parseListArgs(raw []string) (ListArgs, error) {
	a, err := ParseArgs(raw,
		[]string{"--status", "--assignee", "--priority", "--type", "--label", "--limit", "--grep", "--parent"},
		[]string{"--all", "--deferred", "--overdue", "--json"},
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
		Parent:   a.String("--parent"),
		All:      a.Bool("--all"),
		Deferred: a.Bool("--deferred"),
		Overdue:  a.Bool("--overdue"),
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
	if a.missingValue == "--limit" {
		return la, fmt.Errorf("--limit requires a value")
	}
	limit, limitSet, err := a.IntErr("--limit")
	if err != nil {
		return la, err
	}
	if limitSet {
		la.Limit = limit
		la.LimitSet = true
	}
	return la, nil
}

func cmdList(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	la, err := parseListArgs(args)
	if err != nil {
		return nil, err
	}

	filter := issue.Filter{
		Status:   la.Status,
		Assignee: la.Assignee,
		Priority: la.Priority,
		Type:     la.Type,
		Label:    la.Label,
		Grep:     la.Grep,
		Parent:   la.Parent,
	}

	limit := la.Limit

	// Defaults: open status, limit 10. --all overrides both.
	// --deferred overrides status to "deferred".
	// --overdue filters to overdue issues across actionable statuses.
	if la.Overdue {
		filter.Overdue = true
		if la.Status == "" {
			filter.Statuses = []string{"open", "in_progress", "deferred"}
			filter.IncludeExpiredDeferred = true
		}
	} else if la.Deferred {
		filter.Status = "deferred"
	} else if la.All {
		if !la.LimitSet {
			limit = 0
		}
	} else if la.Status == "" {
		filter.Statuses = []string{"open", "in_progress"}
		filter.IncludeExpiredDeferred = true
	}

	issues, err := store.List(filter)
	if err != nil {
		return nil, err
	}

	total := len(issues)
	if limit > 0 && total > limit {
		issues = issues[:limit]
	}
	shown := len(issues)

	if la.JSON {
		fprintJSON(w, issues)
		if shown < total {
			diagnostic := struct {
				Truncated bool `json:"truncated"`
				Shown     int  `json:"shown"`
				Total     int  `json:"total"`
			}{Truncated: true, Shown: shown, Total: total}
			return nil, json.NewEncoder(os.Stderr).Encode(diagnostic)
		}
		return nil, nil
	}

	if shown == 0 {
		fmt.Fprintln(w, "no issues found")
		return nil, nil
	}

	closedBlockers := store.ClosedBlockerSet(issues)
	now := store.Now()
	for _, iss := range issues {
		fmt.Fprintln(w, md.IssueOneLinerWithDue(iss, now, closedBlockers))
	}
	if shown < total {
		fmt.Fprintf(w, "... and %d more (use --limit 0 for all matches)\n", total-shown)
	}
	return nil, nil
}
