package main

import (
	"encoding/json"
	"fmt"

	"github.com/jallum/beadwork/internal/recap"
)

// renderRecap renders a single-repo recap to w.
func renderRecap(w Writer, r recap.Recap, ra recapArgs) error {
	if ra.JSON {
		return renderRecapJSON(w, r)
	}
	return renderRecapTree(w, r, ra.ASCII)
}

func renderRecapJSON(w Writer, r recap.Recap) error {
	out := struct {
		Scope string      `json:"scope"`
		Recap recap.Recap `json:"recap"`
	}{Scope: "single", Recap: r}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(data))
	return nil
}

func renderRecapTree(w Writer, r recap.Recap, ascii bool) error {
	fmt.Fprintf(w, "Recap: %s\n", r.Window.Label)
	if len(r.Sections) == 0 {
		fmt.Fprintln(w, "  (nothing to report — you're caught up)")
		return nil
	}

	branch := "├─"
	last := "└─"
	vbar := "│"
	if ascii {
		branch = "|-"
		last = "`-"
		vbar = "|"
	}

	for i, s := range r.Sections {
		isLast := i == len(r.Sections)-1
		marker := branch
		if isLast {
			marker = last
		}
		title := s.Title
		if title == "" {
			title = "(deleted)"
		}
		fmt.Fprintf(w, "%s %s  %s\n", marker, s.ID, title)

		// Indent prefix for leaves.
		indent := vbar + "  "
		if isLast {
			indent = "   "
		}

		for j, leaf := range s.Leaves {
			leafMarker := branch
			if j == len(s.Leaves)-1 {
				leafMarker = last
			}
			detail := leaf.Type
			if leaf.Detail != "" {
				detail += " " + leaf.Detail
			}
			fmt.Fprintf(w, "%s%s %s  %s\n", indent, leafMarker, leaf.Time, detail)
		}
	}
	return nil
}

// repoRecap pairs a repository path with its computed recap.
// Used for cross-repo fan-out rendering.
type repoRecap struct {
	Path  string      `json:"path"`
	Recap recap.Recap `json:"recap"`
}

// renderCrossRepo renders the output for `bw recap --all`.
func renderCrossRepo(w Writer, all []repoRecap, ra recapArgs) error {
	if ra.JSON {
		out := struct {
			Scope string      `json:"scope"`
			Repos []repoRecap `json:"repos"`
		}{Scope: "cross", Repos: all}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(data))
		return nil
	}

	totalSections := 0
	for _, r := range all {
		totalSections += len(r.Recap.Sections)
	}
	fmt.Fprintf(w, "Cross-repo recap: %d repo(s), %d active issue(s)\n\n", len(all), totalSections)

	for _, rr := range all {
		fmt.Fprintf(w, "=== %s ===\n", rr.Path)
		if err := renderRecapTree(w, rr.Recap, ra.ASCII); err != nil {
			return err
		}
		fmt.Fprintln(w)
	}
	return nil
}
