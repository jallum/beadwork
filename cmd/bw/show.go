package main

import (
	"fmt"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

type ShowArgs struct {
	IDs   []string
	JSON  bool
	Short bool
}

func parseShowArgs(raw []string) (ShowArgs, error) {
	a, err := ParseArgs(raw, nil, []string{"--json", "--short"})
	if err != nil {
		return ShowArgs{}, err
	}
	ids := a.Pos()
	if len(ids) == 0 {
		return ShowArgs{}, fmt.Errorf("usage: bw show <id> [<id>...]")
	}
	return ShowArgs{IDs: ids, JSON: a.JSON(), Short: a.Bool("--short")}, nil
}

func cmdShow(_ *repo.Repo, store *issue.Store, args []string, w Writer) error {
	sa, err := parseShowArgs(args)
	if err != nil {
		return err
	}

	var issues []*issue.Issue
	for _, id := range sa.IDs {
		iss, err := store.Get(id)
		if err != nil {
			return err
		}
		issues = append(issues, iss)
	}

	if sa.JSON {
		fprintJSON(w, issues)
		return nil
	}

	for i, iss := range issues {
		if i > 0 {
			fmt.Fprintln(w)
		}
		if sa.Short {
			ps := PriorityStyle(iss.Priority)
			fmt.Fprintf(w, "%s %s [%s %s] [%s] %s\n",
				issue.StatusIcon(iss.Status),
				w.Style(iss.ID, Cyan),
				w.Style("●", ps),
				w.Style(fmt.Sprintf("P%d", iss.Priority), ps),
				iss.Type,
				iss.Title,
			)
		} else {
			fprintIssue(w, iss)
			fprintDeps(w, iss, store)
			fprintComments(w, iss)
		}
	}
	return nil
}
