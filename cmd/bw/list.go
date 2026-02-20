package main

import (
	"fmt"
	"strconv"

	"github.com/j5n/beadwork/internal/issue"
)

func cmdList(args []string) {
	_, store := mustInitialized()

	filter := issue.Filter{}
	limit := 10
	statusSet := false
	limitSet := false
	showAll := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--status":
			if i+1 < len(args) {
				filter.Status = args[i+1]
				statusSet = true
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
		case "--limit":
			if i+1 < len(args) {
				limit, _ = strconv.Atoi(args[i+1])
				limitSet = true
				i++
			}
		case "--all":
			showAll = true
		}
	}

	// Defaults: open status, limit 10. --all overrides both.
	if showAll {
		if !limitSet {
			limit = 0
		}
	} else if !statusSet {
		filter.Status = "open"
	}

	issues, err := store.List(filter)
	if err != nil {
		fatal(err.Error())
	}

	if hasFlag(args, "--json") {
		if limit > 0 && len(issues) > limit {
			issues = issues[:limit]
		}
		printJSON(issues)
	} else {
		if len(issues) == 0 {
			fmt.Println("no issues found")
			return
		}
		displayed := issues
		if limit > 0 && len(displayed) > limit {
			displayed = displayed[:limit]
		}
		for _, iss := range displayed {
			fmt.Printf("%s %s [%s P%d] [%s] - %s\n",
				issue.StatusIcon(iss.Status),
				iss.ID,
				issue.PriorityDot(iss.Priority),
				iss.Priority,
				iss.Type,
				iss.Title,
			)
		}
		if limit > 0 && len(issues) > limit {
			fmt.Printf("... and %d more (use --limit or --all)\n", len(issues)-limit)
		}
	}
}
