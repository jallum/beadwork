package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/jallum/beadwork/internal/issue"
)

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
	Dependencies []beadsDep `json:"dependencies,omitempty"`
}

type ExportArgs struct {
	Status string
	JSON   bool
}

func parseExportArgs(raw []string) (ExportArgs, error) {
	a := ParseArgs(raw, "--status")
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
