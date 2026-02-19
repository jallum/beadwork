package main

import (
	"fmt"
	"os"
)

func cmdClose(args []string) {
	r, store := mustInitialized()

	if len(args) == 0 {
		fatal("usage: bw close <id> [--reason <reason>]")
	}
	id := args[0]
	reason := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--reason" && i+1 < len(args) {
			reason = args[i+1]
			i++
		}
	}

	iss, err := store.Close(id)
	if err != nil {
		fatal(err.Error())
	}

	intent := fmt.Sprintf("close %s", iss.ID)
	if reason != "" {
		intent += fmt.Sprintf(" reason=%q", reason)
	}
	if err := r.Commit(intent); err != nil {
		fatal("commit failed: " + err.Error())
	}

	if hasFlag(os.Args, "--json") {
		printJSON(iss)
	} else {
		fmt.Printf("closed %s: %s\n", iss.ID, iss.Title)
	}
}

func cmdReopen(args []string) {
	r, store := mustInitialized()

	if len(args) == 0 {
		fatal("usage: bw reopen <id>")
	}
	id := args[0]

	iss, err := store.Reopen(id)
	if err != nil {
		fatal(err.Error())
	}

	intent := fmt.Sprintf("reopen %s", iss.ID)
	if err := r.Commit(intent); err != nil {
		fatal("commit failed: " + err.Error())
	}

	if hasFlag(os.Args, "--json") {
		printJSON(iss)
	} else {
		fmt.Printf("reopened %s: %s\n", iss.ID, iss.Title)
	}
}
