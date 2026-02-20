package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
)

func mustRepo() *repo.Repo {
	r, err := repo.FindRepo()
	if err != nil {
		fatal(err.Error())
	}
	return r
}

func mustInitialized() (*repo.Repo, *issue.Store) {
	r := mustRepo()
	if !r.IsInitialized() {
		fatal("beadwork not initialized. Run: bw init")
	}
	store := issue.NewStore(r.WorkTree, r.Prefix)
	if val, ok := r.GetConfig("default.priority"); ok {
		if p, err := strconv.Atoi(val); err == nil && p > 0 {
			store.DefaultPriority = p
		}
	}
	return r, store
}

func fatal(msg string) {
	fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	os.Exit(1)
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func printJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func printIssue(iss *issue.Issue) {
	// Header: ○ bw-f0ae · Title   [● P3 · OPEN]
	fmt.Printf("%s %s · %s   [%s P%d · %s]\n",
		issue.StatusIcon(iss.Status),
		iss.ID,
		iss.Title,
		issue.PriorityDot(iss.Priority),
		iss.Priority,
		strings.ToUpper(iss.Status),
	)

	// Metadata line: Assignee · Type
	assignee := iss.Assignee
	if assignee == "" {
		assignee = "—"
	}
	fmt.Printf("Assignee: %s · Type: %s\n", assignee, iss.Type)

	// Created date (trim to date only)
	created := iss.Created
	if len(created) >= 10 {
		created = created[:10]
	}
	fmt.Printf("Created: %s\n", created)

	// Optional metadata
	if len(iss.Labels) > 0 {
		fmt.Printf("Labels: %s\n", strings.Join(iss.Labels, ", "))
	}

	var deps []string
	if len(iss.Blocks) > 0 {
		deps = append(deps, "Blocks: "+strings.Join(iss.Blocks, ", "))
	}
	if len(iss.BlockedBy) > 0 {
		deps = append(deps, "Blocked by: "+strings.Join(iss.BlockedBy, ", "))
	}
	if len(deps) > 0 {
		fmt.Println(strings.Join(deps, " · "))
	}
	if iss.Parent != "" {
		fmt.Printf("Parent: %s\n", iss.Parent)
	}

	// Description
	if iss.Description != "" {
		fmt.Printf("\nDESCRIPTION\n\n")
		for _, line := range strings.Split(iss.Description, "\n") {
			fmt.Printf("  %s\n", line)
		}
		fmt.Println()
	}
}
