package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/j5n/beadwork/internal/issue"
	"github.com/j5n/beadwork/internal/repo"
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
	fmt.Printf("ID:          %s\n", iss.ID)
	fmt.Printf("Title:       %s\n", iss.Title)
	fmt.Printf("Status:      %s\n", iss.Status)
	fmt.Printf("Priority:    p%d\n", iss.Priority)
	fmt.Printf("Type:        %s\n", iss.Type)
	if iss.Assignee != "" {
		fmt.Printf("Assignee:    %s\n", iss.Assignee)
	}
	if iss.Description != "" {
		fmt.Printf("Description: %s\n", iss.Description)
	}
	if len(iss.Labels) > 0 {
		fmt.Printf("Labels:      %s\n", strings.Join(iss.Labels, ", "))
	}
	if len(iss.Blocks) > 0 {
		fmt.Printf("Blocks:      %s\n", strings.Join(iss.Blocks, ", "))
	}
	if len(iss.BlockedBy) > 0 {
		fmt.Printf("Blocked by:  %s\n", strings.Join(iss.BlockedBy, ", "))
	}
	if iss.Parent != "" {
		fmt.Printf("Parent:      %s\n", iss.Parent)
	}
	fmt.Printf("Created:     %s\n", iss.Created)
}
