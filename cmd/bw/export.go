package main

import (
	"encoding/json"
	"fmt"

	"github.com/j5n/beadwork/internal/issue"
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
	ID           string      `json:"id"`
	Title        string      `json:"title"`
	Description  string      `json:"description,omitempty"`
	Status       string      `json:"status"`
	Priority     int         `json:"priority"`
	IssueType    string      `json:"issue_type"`
	Owner        string      `json:"owner,omitempty"`
	CreatedAt    string      `json:"created_at"`
	Dependencies []beadsDep  `json:"dependencies,omitempty"`
}

func cmdExport(args []string) {
	_, store := mustInitialized()

	filter := issue.Filter{}
	for i := 0; i < len(args); i++ {
		if args[i] == "--status" && i+1 < len(args) {
			filter.Status = args[i+1]
			i++
		}
	}

	issues, err := store.List(filter)
	if err != nil {
		fatal(err.Error())
	}

	for _, iss := range issues {
		rec := beadsRecord{
			ID:        iss.ID,
			Title:     iss.Title,
			Description: iss.Description,
			Status:    iss.Status,
			Priority:  iss.Priority,
			IssueType: iss.Type,
			Owner:     iss.Assignee,
			CreatedAt: iss.Created,
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
			fatal(err.Error())
		}
		fmt.Println(string(data))
	}
}
