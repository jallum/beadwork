package main

import (
	"fmt"

	"github.com/j5n/beadwork/internal/issue"
	"github.com/j5n/beadwork/prompts"
)

func cmdPrime() {
	_, store := mustInitialized()

	fmt.Print(prompts.Prime)

	// Dynamic section: current project state
	ready, _ := store.Ready()
	all, _ := store.List(issue.Filter{})

	openCount := 0
	ipCount := 0
	closedCount := 0
	for _, iss := range all {
		switch iss.Status {
		case "open":
			openCount++
		case "in_progress":
			ipCount++
		case "closed":
			closedCount++
		}
	}

	fmt.Println("## Current State")
	fmt.Println()
	fmt.Printf("  %d open, %d in progress, %d closed\n", openCount, ipCount, closedCount)
	fmt.Printf("  %d ready (unblocked)\n", len(ready))
	fmt.Println()

	if ipCount > 0 {
		fmt.Println("  In progress:")
		for _, iss := range all {
			if iss.Status == "in_progress" {
				fmt.Printf("    %-14s p%d %s\n", iss.ID, iss.Priority, iss.Title)
			}
		}
		fmt.Println()
	}

	if len(ready) > 0 {
		fmt.Println("  Ready for work:")
		for _, iss := range ready {
			fmt.Printf("    %-14s p%d %s\n", iss.ID, iss.Priority, iss.Title)
		}
		fmt.Println()
	}
}
