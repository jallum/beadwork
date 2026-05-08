package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jallum/beadwork/internal/recap"
)

// renderRecap renders a single-repo recap to w.
func renderRecap(w Writer, r recap.Recap, ra recapArgs) error {
	if ra.JSON {
		return renderRecapJSON(w, r)
	}
	if ra.Verbose {
		return renderRecapTree(w, r, ra.ASCII)
	}
	return renderRecapCondensed(w, r)
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

// sectionSummary rolls up a section into a compact summary string and the
// marker character (open/in_progress/closed).
//
// Buckets:
//   - "state" events that matter for a headline: close, reopen, start, create
//   - "notes":   comment, label
//   - "edits":   update, link, unlink, defer, undefer
//   - "tangential": unblocked, delete
//
// The summary leads with the most-significant state change within the window,
// then appends quiet counts for everything else ("+ 2 comments, 3 edits").
func sectionSummary(s recap.Section) (marker, summary string, latest time.Time) {
	var closed, reopened, started, created bool
	var comments, edits, labels, unblocked int

	for _, l := range s.Leaves {
		t, _ := time.Parse(time.RFC3339, l.Time)
		if t.After(latest) {
			latest = t
		}
		switch l.Type {
		case "close":
			closed = true
		case "reopen":
			reopened = true
		case "start":
			started = true
		case "create":
			created = true
		case "comment":
			comments++
		case "label":
			labels++
		case "unblocked":
			unblocked++
		case "update", "link", "unlink", "defer", "undefer":
			edits++
		}
	}

	// Headline: the most significant state change.
	var parts []string
	switch {
	case closed:
		parts = append(parts, "closed")
		marker = "●"
	case started:
		parts = append(parts, "started")
		marker = "◐"
	case reopened:
		parts = append(parts, "reopened")
		marker = "◐"
	case created:
		parts = append(parts, "created")
		marker = "○"
	default:
		marker = "·"
	}

	// Quiet counts.
	if comments > 0 {
		parts = append(parts, pluralize(comments, "comment"))
	}
	if edits > 0 {
		parts = append(parts, pluralize(edits, "edit"))
	}
	if labels > 0 {
		parts = append(parts, pluralize(labels, "label change"))
	}
	if unblocked > 0 {
		parts = append(parts, pluralize(unblocked, "unblocked"))
	}

	if len(parts) == 0 {
		summary = fmt.Sprintf("%d event(s)", len(s.Leaves))
	} else {
		summary = strings.Join(parts, ", ")
	}
	return marker, summary, latest
}

func pluralize(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// markerStyle maps a summary marker to the ANSI style used to color it.
func markerStyle(marker string) Style {
	switch marker {
	case "●":
		return Green // closed
	case "◐":
		return Yellow // started / reopened
	case "○":
		return Cyan // created
	default:
		return Dim
	}
}

// colorizeSummary applies per-keyword color to the summary phrase so state
// changes jump out from counts.
func colorizeSummary(w Writer, summary string) string {
	// Split by ", " and re-assemble with styling.
	parts := strings.Split(summary, ", ")
	for i, p := range parts {
		switch {
		case p == "closed":
			parts[i] = w.Style(p, Green, Bold)
		case p == "started":
			parts[i] = w.Style(p, Yellow, Bold)
		case p == "reopened":
			parts[i] = w.Style(p, Yellow, Bold)
		case p == "created":
			parts[i] = w.Style(p, Cyan, Bold)
		default:
			parts[i] = w.Style(p, Dim)
		}
	}
	return strings.Join(parts, w.Style(", ", Dim))
}

func renderRecapCondensed(w Writer, r recap.Recap) error {
	if len(r.Sections) == 0 {
		fmt.Fprintf(w, "Recap: %s — %s\n",
			w.Style(r.Window.Label, Cyan),
			w.Style("nothing to report", Dim))
		return nil
	}

	// Build rows then sort by latest activity, most recent first.
	type row struct {
		marker, id, title, summary string
		latest                     time.Time
	}
	rows := make([]row, 0, len(r.Sections))
	for _, s := range r.Sections {
		m, summ, lat := sectionSummary(s)
		title := s.Title
		if title == "" {
			title = "(deleted)"
		}
		rows = append(rows, row{
			marker: m, id: s.ID, title: title,
			summary: summ, latest: lat,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].latest.After(rows[j].latest)
	})

	fmt.Fprintf(w, "Recap: %s %s %s\n",
		w.Style(r.Window.Label, Cyan),
		w.Style("·", Dim),
		w.Style(fmt.Sprintf("%d issue(s)", len(rows)), Dim))

	// Compute id column width for alignment (pre-style, so ANSI codes
	// don't bloat the padding).
	idWidth := 0
	for _, r := range rows {
		if len(r.id) > idWidth {
			idWidth = len(r.id)
		}
	}

	// Compute available width for the title column so long titles don't
	// force the Writer to hard-wrap mid-line. Budget per row:
	//   "  " (2) + marker (1) + " " (1) + id (idWidth) + "  " (2) +
	//   TITLE + "  — " (4) + summary + " " (1) + "(age)"
	// Reserve a small cushion so we never hit the right edge.
	width := w.Width()
	now := bwNow()
	for _, r := range rows {
		age := relativeTimeSince(r.latest, now)
		title := r.title
		if width > 0 {
			fixed := 2 + 1 + 1 + idWidth + 2 + 4 + visibleLen(r.summary) + 1 + len("("+age+")")
			budget := width - fixed - 1 // 1-char cushion
			if budget < 10 {
				budget = 10
			}
			if len(title) > budget {
				title = title[:budget-1] + "…"
			}
		}
		marker := w.Style(r.marker, markerStyle(r.marker))
		// Pad the id BEFORE styling so alignment is based on visible width.
		paddedID := fmt.Sprintf("%-*s", idWidth, r.id)
		styledID := w.Style(paddedID, Bold)
		summary := colorizeSummary(w, r.summary)
		fmt.Fprintf(w, "  %s %s  %s  %s %s %s\n",
			marker,
			styledID,
			title,
			w.Style("—", Dim),
			summary,
			w.Style("("+age+")", Dim),
		)
	}
	// Only print the interactive hint when output is going to a TTY.
	// Keep piped / LLM consumption free of chatty prompts.
	if w.IsTTY() {
		fmt.Fprintln(w, w.Style("\n  (use --verbose for per-event detail)", Dim))
	}
	return nil
}

// visibleLen returns the rune count of s, ignoring nothing — used for
// budget math on strings that have NOT been ANSI-styled yet.
func visibleLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// eventTypeStyle returns the color styling for an event type in the verbose
// tree. State changes stand out; low-signal events are dim.
func eventTypeStyle(t string) []Style {
	switch t {
	case "close":
		return []Style{Green, Bold}
	case "start", "reopen":
		return []Style{Yellow, Bold}
	case "create":
		return []Style{Cyan, Bold}
	case "unblocked":
		return []Style{Cyan}
	case "delete":
		return []Style{Red, Bold}
	case "comment", "label":
		return []Style{} // default color
	case "update", "link", "unlink", "defer", "undefer":
		return []Style{Dim}
	default:
		return []Style{Dim}
	}
}

func renderRecapTree(w Writer, r recap.Recap, ascii bool) error {
	fmt.Fprintf(w, "Recap: %s\n", w.Style(r.Window.Label, Cyan))
	if len(r.Sections) == 0 {
		fmt.Fprintln(w, w.Style("  (nothing to report — you're caught up)", Dim))
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
			title = w.Style("(deleted)", Dim)
		}
		fmt.Fprintf(w, "%s %s  %s\n", w.Style(marker, Dim), w.Style(s.ID, Bold), title)

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

			styles := eventTypeStyle(leaf.Type)
			styledType := w.Style(leaf.Type, styles...)
			line := styledType
			if leaf.Detail != "" {
				line += " " + w.Style(leaf.Detail, Dim)
			}
			fmt.Fprintf(w, "%s%s %s  %s\n",
				w.Style(indent, Dim),
				w.Style(leafMarker, Dim),
				w.Style(leaf.Time, Dim),
				line,
			)
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
		if err := renderRecap(w, rr.Recap, ra); err != nil {
			return err
		}
		fmt.Fprintln(w)
	}
	return nil
}
