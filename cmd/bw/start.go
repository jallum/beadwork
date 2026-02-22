package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
	"github.com/jallum/beadwork/internal/template"
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

	// -- Rich output: issue context + workflow hints --

	// Summary (reuse show's display)
	fprintIssueSummary(w, iss)
	fprintDescription(w, iss)
	fprintChildren(w, iss, store)
	fprintMap(w, iss, store)
	fprintComments(w, iss)

	// Next steps from start.md template
	tmpl := strings.ReplaceAll(prompts.Start, "{id}", iss.ID)
	var buf bytes.Buffer
	template.Process(&buf, tmpl, r.ListConfig(), nil)

	text := strings.TrimSpace(buf.String())
	if text != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, w.Style("LANDING THE WORK", Bold))
		w.Push(2)
		fmt.Fprintln(w, text)
		w.Pop()
	}

	return nil
}
