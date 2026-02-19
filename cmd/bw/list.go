package main

import (
	"fmt"
	"strconv"

	"github.com/j5n/beadwork/internal/issue"
)

func cmdList(args []string) {
	_, store := mustInitialized()

	filter := issue.Filter{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--status":
			if i+1 < len(args) {
				filter.Status = args[i+1]
				i++
			}
		case "--assignee":
			if i+1 < len(args) {
				filter.Assignee = args[i+1]
				i++
			}
		case "--priority":
			if i+1 < len(args) {
				p, _ := strconv.Atoi(args[i+1])
				filter.Priority = p
				i++
			}
		case "--type":
			if i+1 < len(args) {
				filter.Type = args[i+1]
				i++
			}
		case "--label":
			if i+1 < len(args) {
				filter.Label = args[i+1]
				i++
			}
		}
	}

	issues, err := store.List(filter)
	if err != nil {
		fatal(err.Error())
	}

	if hasFlag(args, "--json") {
		printJSON(issues)
	} else {
		if len(issues) == 0 {
			fmt.Println("no issues found")
			return
		}
		for _, iss := range issues {
			fmt.Printf("%-14s p%d %-12s %-12s %s\n", iss.ID, iss.Priority, iss.Status, iss.Type, iss.Title)
		}
	}
}
