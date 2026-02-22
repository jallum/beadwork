package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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

	// -- Rich output: issue context + template-driven briefing --

	fprintIssueSummary(w, iss)
	fprintDescription(w, iss)

	// Template controls everything after summary+description.
	tmpl := strings.ReplaceAll(prompts.Start, "{id}", iss.ID)
	cfg := r.ListConfig()
	resolve := func(key string) string {
		switch key {
		case "type":
			return iss.Type
		case "status":
			return iss.Status
		case "parent":
			return iss.Parent
		default:
			return cfg[key]
		}
	}

	var buf bytes.Buffer
	flush := func() {
		s := strings.Trim(buf.String(), "\n")
		buf.Reset()
		if s == "" {
			return
		}
		fmt.Fprintln(w)
		for _, line := range strings.Split(s, "\n") {
			if isAllCaps(line) {
				fmt.Fprintln(w, w.Style(line, Bold))
			} else {
				fmt.Fprintln(w, line)
			}
		}
	}

	cmdFn := func(args []string, _ io.Writer) {
		if cmd := commandMap[args[0]]; cmd != nil {
			flush()
			cmd.Run(store, args[1:], w)
		}
	}

	template.ProcessWithCommands(&buf, tmpl, resolve, nil, cmdFn)
	flush()

	return nil
}

// isAllCaps reports whether s is non-empty and contains only uppercase
// letters and spaces (matching section headers like "STARTING THE WORK").
func isAllCaps(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r != ' ' && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}
