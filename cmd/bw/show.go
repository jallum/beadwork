package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/jallum/beadwork/internal/config"

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
}

type ShowArgs struct {
	ID       string
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
		return ShowArgs{}, fmt.Errorf("usage: bw show <id>")
	}
	if len(ids) > 1 {
		return ShowArgs{}, fmt.Errorf("show accepts a single issue ID")
	}

	sa := ShowArgs{ID: ids[0], JSON: a.JSON()}

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

func cmdShow(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	sa, err := parseShowArgs(args)
	if err != nil {
		return nil, err
	}

	iss, err := store.Get(sa.ID)
	if err != nil {
		return nil, err
	}

	if sa.JSON {
		fprintJSON(w, iss)
		return nil, nil
	}

	if sa.showSection("summary") {
		fmt.Fprintln(w, md.IssueSummary(iss, store.Now()))
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
	return nil, nil
}

func showDescription(w Writer, iss *issue.Issue) {
	if iss.Description != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, md.Description(iss.Description))
	}
}

func showChildren(w Writer, iss *issue.Issue, store *issue.Store) {
	children, err := store.Children(iss.ID)
	if err != nil || len(children) == 0 {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, md.Children(children))
}

// showMap renders BLOCKED BY and UNBLOCKS sections using the md package.
func showMap(w Writer, iss *issue.Issue, store *issue.Store) {
	if openBlockers := openDirectBlockers(iss, store); len(openBlockers) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, md.BlockedBy(openBlockers))
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
			fmt.Fprintln(w, md.Unblocks(deps))
		}
	}
}

// openDirectBlockers resolves iss.BlockedBy to issues, dropping closed
// ones — the same set `bw list` and `bw blocked` render. (--json's
// blocked_by is the raw, unfiltered field and can still include closed
// blockers.)
func openDirectBlockers(iss *issue.Issue, store *issue.Store) []*issue.Issue {
	var open []*issue.Issue
	for _, id := range iss.BlockedBy {
		dep, err := store.Get(id)
		if err != nil || dep.Status == "closed" {
			continue
		}
		open = append(open, dep)
	}
	return open
}

func showComments(w Writer, iss *issue.Issue) {
	if len(iss.Comments) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, md.Comments(iss.Comments))
	}
}

// fprintIssueSummary renders the summary for use by start.go (Phase 3).
func fprintIssueSummary(w Writer, iss *issue.Issue, now time.Time) {
	fmt.Fprintln(w, md.IssueSummary(iss, now))
}

// fprintDescription renders description for use by start.go (Phase 3).
func fprintDescription(w Writer, iss *issue.Issue) {
	showDescription(w, iss)
}

// fprintComments renders comments for use by start.go (Phase 3).
func fprintComments(w Writer, iss *issue.Issue) {
	showComments(w, iss)
}
