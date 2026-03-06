package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/md"
)

// validShowSections lists section names accepted by --only.
var validShowSections = map[string]bool{
	"summary":     true,
	"description": true,
	"children":    true,
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

func cmdShow(store *issue.Store, args []string, w Writer) error {
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
			emitln(w, md.IssueSummary(iss))
		}
		if sa.showSection("description") {
			showDescription(w, iss)
		}
		if sa.showSection("children") {
			showChildren(w, iss, store)
		}
		if sa.showSection("blockedby") || sa.showSection("unblocks") {
			showMap(w, iss, store)
		}
		if sa.showSection("comments") {
			showComments(w, iss)
		}
	}
	return nil
}

func showDescription(w Writer, iss *issue.Issue) {
	if iss.Description != "" {
		fmt.Fprintln(w)
		emitln(w, md.Description(iss.Description))
	}
}

func showChildren(w Writer, iss *issue.Issue, store *issue.Store) {
	children, err := store.Children(iss.ID)
	if err != nil || len(children) == 0 {
		return
	}
	fmt.Fprintln(w)
	emitln(w, md.Children(children))
}

// showMap renders BLOCKED BY and UNBLOCKS sections using the md package.
func showMap(w Writer, iss *issue.Issue, store *issue.Store) {
	_, rev := store.LoadEdges()

	if len(iss.BlockedBy) > 0 {
		tips, _ := store.Tips(iss.BlockedBy, rev)
		actionable := nearestOpen(tips, iss.ID, store)
		if len(actionable) > 0 {
			fmt.Fprintln(w)
			emitln(w, md.BlockedBy(actionable))
		}
	}

	if len(iss.Blocks) > 0 {
		var deps []*issue.Issue
		for _, id := range iss.Blocks {
			dep, err := store.Get(id)
			if err != nil {
				continue
			}
			deps = append(deps, dep)
		}
		if len(deps) > 0 {
			fmt.Fprintln(w)
			emitln(w, md.Unblocks(deps))
		}
	}
}

func showComments(w Writer, iss *issue.Issue) {
	if len(iss.Comments) > 0 {
		fmt.Fprintln(w)
		emitln(w, md.Comments(iss.Comments))
	}
}

// fprintIssueSummary renders the summary for use by start.go (Phase 3).
func fprintIssueSummary(w Writer, iss *issue.Issue) {
	emitln(w, md.IssueSummary(iss))
}

// fprintDescription renders description for use by start.go (Phase 3).
func fprintDescription(w Writer, iss *issue.Issue) {
	showDescription(w, iss)
}

// fprintComments renders comments for use by start.go (Phase 3).
func fprintComments(w Writer, iss *issue.Issue) {
	showComments(w, iss)
}
