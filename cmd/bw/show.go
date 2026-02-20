package main

import (
	"fmt"
	"io"
)

func cmdShow(args []string, w io.Writer) error {
	_, store, err := getInitialized()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: bw show <id>")
	}
	id := args[0]

	iss, err := store.Get(id)
	if err != nil {
		return err
	}

	if hasFlag(args, "--json") {
		fprintJSON(w, iss)
	} else {
		fprintIssue(w, iss)
	}
	return nil
}
