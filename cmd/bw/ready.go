package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
)

func cmdReady(args []string) {
	_, store := mustInitialized()

	issues, err := store.Ready()
	if err != nil {
		fatal(err.Error())
	}

	if hasFlag(args, "--json") {
		printJSON(issues)
		return
	}

	if len(issues) == 0 {
		fmt.Println("no ready issues")
		return
	}

	for _, iss := range issues {
		fmt.Printf("%s %s %s P%d %s\n",
			issue.StatusIcon(iss.Status),
			iss.ID,
			issue.PriorityDot(iss.Priority),
			iss.Priority,
			iss.Title,
		)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Ready: %d issues with no blockers\n", len(issues))
	fmt.Println()

	var legend []string
	for _, s := range issue.Statuses {
		legend = append(legend, s.Icon+" "+s.Name)
	}
	fmt.Printf("Status: %s\n", strings.Join(legend, "  "))
}
