package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
)

// nilIfEmpty returns nil for empty slices so omitempty works.
func nilIfEmpty(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

// toRFC3339Date converts YYYY-MM-DD to RFC3339 for bd compat.
// Returns empty string if input is empty.
func toRFC3339Date(s string) string {
	if s == "" {
		return ""
	}
	// Already RFC3339? Return as-is.
	if strings.Contains(s, "T") {
		return s
	}
	return s + "T00:00:00Z"
}

// fromRFC3339Date extracts YYYY-MM-DD from an RFC3339 or bare date string.
func fromRFC3339Date(s string) string {
	if s == "" {
		return ""
	}
	if idx := strings.IndexByte(s, 'T'); idx >= 0 {
		return s[:idx]
	}
	return s
}

type beadsDep struct {
	IssueID     string `json:"issue_id"`
	DependsOnID string `json:"depends_on_id"`
	Type        string `json:"type"`
	CreatedAt   string `json:"created_at"`
	CreatedBy   string `json:"created_by"`
	Metadata    string `json:"metadata"`
}

type beadsRecord struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description,omitempty"`
	Status       string     `json:"status"`
	Priority     int        `json:"priority"`
	IssueType    string     `json:"issue_type"`
	Owner        string     `json:"owner,omitempty"`
	CreatedAt    string     `json:"created_at"`
	Labels       []string   `json:"labels,omitempty"`
	DeferUntil   string     `json:"defer_until,omitempty"`
	Blocks       []string   `json:"blocks,omitempty"`
	BlockedBy    []string   `json:"blocked_by,omitempty"`
	Dependencies []beadsDep `json:"dependencies,omitempty"`
}

type ExportArgs struct {
	Status string
	JSON   bool
}

func parseExportArgs(raw []string) (ExportArgs, error) {
	a, err := ParseArgs(raw, []string{"--status"}, []string{"--json"})
	if err != nil {
		return ExportArgs{}, err
	}
	return ExportArgs{
		Status: a.String("--status"),
		JSON:   a.JSON(),
	}, nil
}

func cmdExport(args []string, w io.Writer) error {
	ea, err := parseExportArgs(args)
	if err != nil {
		return err
	}

	_, store, err := getInitialized()
	if err != nil {
		return err
	}

	filter := issue.Filter{Status: ea.Status}

	issues, err := store.List(filter)
	if err != nil {
		return err
	}

	for _, iss := range issues {
		rec := beadsRecord{
			ID:          iss.ID,
			Title:       iss.Title,
			Description: iss.Description,
			Status:      iss.Status,
			Priority:    iss.Priority,
			IssueType:   iss.Type,
			Owner:       iss.Assignee,
			CreatedAt:   iss.Created,
			Labels:      nilIfEmpty(iss.Labels),
			DeferUntil:  toRFC3339Date(iss.DeferUntil),
			Blocks:      nilIfEmpty(iss.Blocks),
			BlockedBy:   nilIfEmpty(iss.BlockedBy),
		}

		// Build dependencies from BlockedBy and Parent
		for _, blockerID := range iss.BlockedBy {
			rec.Dependencies = append(rec.Dependencies, beadsDep{
				IssueID:     iss.ID,
				DependsOnID: blockerID,
				Type:        "blocks",
				CreatedAt:   iss.Created,
				CreatedBy:   iss.Assignee,
				Metadata:    "{}",
			})
		}
		if iss.Parent != "" {
			rec.Dependencies = append(rec.Dependencies, beadsDep{
				IssueID:     iss.ID,
				DependsOnID: iss.Parent,
				Type:        "parent-child",
				CreatedAt:   iss.Created,
				CreatedBy:   iss.Assignee,
				Metadata:    "{}",
			})
		}

		data, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(data))
	}
	return nil
}
