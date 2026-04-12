// Package recap builds structured activity summaries from beadwork commit
// history. The data model (Recap/Section/Leaf) is renderer-agnostic: a single
// Build produces one model that both tree and JSON renderers consume.
package recap

import (
	"sort"
	"time"

	"github.com/jallum/beadwork/internal/treefs"
)

// IssueLookup resolves an issue ID to its title. Implementations may return
// "" if the issue has been deleted or is otherwise unavailable.
type IssueLookup interface {
	Title(id string) string
}

// Event represents a single parsed activity from a commit message.
type Event struct {
	Type string    // "create", "close", "start", "update", "reopen", "defer", "undefer", "comment", "link", "unlink", "unblocked", "delete", "label"
	ID   string    // primary issue ID
	Time time.Time // commit timestamp
	Detail string  // additional context (title, reason, etc.)
}

// Leaf is a single event in the recap tree.
type Leaf struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Time   string `json:"time"`
	Detail string `json:"detail,omitempty"`
}

// Section groups events for a single issue.
type Section struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Leaves []Leaf `json:"events"`
}

// Recap is the top-level activity summary.
type Recap struct {
	Window   Window    `json:"window"`
	Sections []Section `json:"sections"`
}

// Build constructs a Recap from commits within the given window.
// It parses each commit's intent, deduplicates events, groups by issue,
// and resolves titles via the lookup.
func Build(commits []treefs.CommitInfo, w Window, lookup IssueLookup) Recap {
	var events []Event
	seen := make(map[string]bool) // dedup key: "type:id:time"

	for _, c := range commits {
		// Window is [Start, End] inclusive on both ends so that events
		// at "now" are captured.
		if c.Time.Before(w.Start) || c.Time.After(w.End) {
			continue
		}
		parsed := ParseIntent(c.Message, c.Time)
		for _, e := range parsed {
			key := e.Type + ":" + e.ID + ":" + e.Time.Format(time.RFC3339)
			if seen[key] {
				continue
			}
			seen[key] = true
			events = append(events, e)
		}
	}

	// Group by issue ID.
	grouped := make(map[string][]Event)
	var order []string
	for _, e := range events {
		if _, exists := grouped[e.ID]; !exists {
			order = append(order, e.ID)
		}
		grouped[e.ID] = append(grouped[e.ID], e)
	}

	// Build sections.
	sections := make([]Section, 0, len(order))
	for _, id := range order {
		evts := grouped[id]
		title := ""
		if lookup != nil {
			title = lookup.Title(id)
		}

		leaves := make([]Leaf, 0, len(evts))
		for _, e := range evts {
			leaves = append(leaves, Leaf{
				Type:   e.Type,
				ID:     e.ID,
				Time:   e.Time.UTC().Format(time.RFC3339),
				Detail: e.Detail,
			})
		}

		sections = append(sections, Section{
			ID:     id,
			Title:  title,
			Leaves: leaves,
		})
	}

	// Sort sections by first event time.
	sort.Slice(sections, func(i, j int) bool {
		if len(sections[i].Leaves) == 0 || len(sections[j].Leaves) == 0 {
			return false
		}
		return sections[i].Leaves[0].Time < sections[j].Leaves[0].Time
	})

	return Recap{
		Window:   w,
		Sections: sections,
	}
}
