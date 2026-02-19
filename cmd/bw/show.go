package main

func cmdShow(args []string) {
	_, store := mustInitialized()

	if len(args) == 0 {
		fatal("usage: bw show <id>")
	}
	id := args[0]

	iss, err := store.Get(id)
	if err != nil {
		fatal(err.Error())
	}

	if hasFlag(args, "--json") {
		printJSON(iss)
	} else {
		printIssue(iss)
	}
}
