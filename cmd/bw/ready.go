package main

import "fmt"

func cmdReady(args []string) {
	_, store := mustInitialized()

	issues, err := store.Ready()
	if err != nil {
		fatal(err.Error())
	}

	if hasFlag(args, "--json") {
		printJSON(issues)
	} else {
		if len(issues) == 0 {
			fmt.Println("no ready issues")
			return
		}
		for _, iss := range issues {
			fmt.Printf("%-14s p%d %-12s %-12s %s\n", iss.ID, iss.Priority, iss.Status, iss.Type, iss.Title)
		}
	}
}
