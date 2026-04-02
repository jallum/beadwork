package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
	"github.com/jallum/beadwork/internal/tmpl"
	"github.com/jallum/beadwork/prompts"
)

type StartArgs struct {
	ID       string
	Assignee string
	JSON     bool
}

func parseStartArgs(raw []string) (StartArgs, error) {
	if len(raw) == 0 {
		return StartArgs{}, fmt.Errorf("usage: bw start <id> [--assignee <name>]")
	}
	a, err := ParseArgs(raw[1:], []string{"--assignee"}, []string{"--json"})
	if err != nil {
		return StartArgs{}, err
	}
	return StartArgs{
		ID:       raw[0],
		Assignee: a.String("--assignee"),
		JSON:     a.JSON(),
	}, nil
}

type StartData struct {
	ID             string
	Type           string
	Status         string
	Parent         string
	WorkflowReview string
}

func cmdStart(store *issue.Store, args []string, w Writer) error {
	sa, err := parseStartArgs(args)
	if err != nil {
		return err
	}

	r := store.Committer.(*repo.Repo)

	// Default assignee to git user.name
	assignee := sa.Assignee
	if assignee == "" {
		assignee = r.UserName()
	}

	iss, err := store.Start(sa.ID, assignee)
	if err != nil {
		var be *issue.BlockedError
		if errors.As(err, &be) {
			lines := []string{fmt.Sprintf("%s is blocked by:", be.ID)}
			for _, id := range be.Blockers {
				dep, derr := store.Get(id)
				if derr != nil {
					lines = append(lines, fmt.Sprintf("  %s %s", issue.StatusIcon("open"), id))
				} else {
					lines = append(lines, fmt.Sprintf("  %s %s: %s", issue.StatusIcon(dep.Status), id, dep.Title))
				}
			}
			lines = append(lines, "\nuse bw ready to find available work")
			return fmt.Errorf("%s", strings.Join(lines, "\n"))
		}
		return err
	}

	intent := fmt.Sprintf("start %s assignee=%q", iss.ID, assignee)
	if err := store.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if sa.JSON {
		fprintJSON(w, iss)
		return nil
	}

	// -- Rich output: issue context + template-driven briefing --

	fprintIssueSummary(w, iss, store.Now())
	fprintDescription(w, iss)
	fprintComments(w, iss)

	cfg := r.ListConfig()
	data := StartData{
		ID:             iss.ID,
		Type:           iss.Type,
		Status:         iss.Status,
		Parent:         iss.Parent,
		WorkflowReview: cfg["workflow.review"],
	}

	bwFn := func(args ...string) string {
		if cmd := commandMap[args[0]]; cmd != nil {
			var buf bytes.Buffer
			cmd.Run(store, args[1:], TokenWriter(&buf))
			return strings.TrimRight(buf.String(), "\n")
		}
		return ""
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, "start", prompts.Start, data, bwFn); err != nil {
		return err
	}

	out := strings.Trim(buf.String(), "\n")
	if out != "" {
		fmt.Fprintln(w)
		fmt.Fprint(w, out)
		fmt.Fprintln(w)
	}

	return nil
}
