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

	a := ParseArgs(args)

	id := a.PosFirst()
	if id == "" {
		return fmt.Errorf("usage: bw show <id>")
	}

	iss, err := store.Get(id)
	if err != nil {
		return err
	}

	if a.JSON() {
		fprintJSON(w, iss)
	} else {
		fprintIssue(w, iss)
	}
	return nil
}
