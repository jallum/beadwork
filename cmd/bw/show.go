package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
	"github.com/jallum/beadwork/internal/wrap"
)

// validShowSections lists section names accepted by --only.
var validShowSections = map[string]bool{
	"summary":     true,
	"description": true,
	"blockedby":   true,
	"unblocks":    true,
	"comments":    true,
	"labels":      true,
	"parent":      true,
}

type ShowArgs struct {
	IDs      []string
	JSON     bool
	Sections map[string]bool // nil means show all
}

func parseShowArgs(raw []string) (ShowArgs, error) {
	a, err := ParseArgs(raw, []string{"--only"}, []string{"--json"})
	if err != nil {
		return ShowArgs{}, err
	}
	ids := a.Pos()
	if len(ids) == 0 {
		return ShowArgs{}, fmt.Errorf("usage: bw show <id> [<id>...]")
	}

	sa := ShowArgs{IDs: ids, JSON: a.JSON()}

	if only := a.String("--only"); only != "" {
		sa.Sections = make(map[string]bool)
		for _, s := range strings.Split(only, ",") {
			s = strings.TrimSpace(s)
			if !validShowSections[s] {
				return ShowArgs{}, fmt.Errorf("unknown section %q; valid: %s", s, showSectionNames())
			}
			sa.Sections[s] = true
		}
	}

	return sa, nil
}

func showSectionNames() string {
	names := make([]string, 0, len(validShowSections))
	for k := range validShowSections {
		names = append(names, k)
	}
	return strings.Join(names, ", ")
}

func (sa ShowArgs) showSection(name string) bool {
	if sa.Sections == nil {
		return true // no filter, show all
	}
	return sa.Sections[name]
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
		if sa.showSection("summary") {
			fprintIssueSummary(w, iss)
		}
		if sa.showSection("description") {
			fprintDescription(w, iss)
		}
		if sa.showSection("blockedby") || sa.showSection("unblocks") {
			fprintMap(w, iss, store)
		}
		if sa.showSection("comments") {
			fprintComments(w, iss)
		}
	}
	return nil
}

// fprintIssueSummary renders the header, metadata, labels, and parent.
func fprintIssueSummary(w Writer, iss *issue.Issue) {
	fprintIssue(w, iss)
}

// fprintDescription renders just the DESCRIPTION section.
func fprintDescription(w Writer, iss *issue.Issue) {
	if iss.Description != "" {
		fmt.Fprintf(w, "\n%s\n\n", w.Style("DESCRIPTION", Bold))
		w.Push(2)
		desc := iss.Description
		if ww := w.Width(); ww > 0 {
			desc = wrap.Text(desc, ww)
		}
		fmt.Fprintln(w, desc)
		w.Pop()
	}
}
