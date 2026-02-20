package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
)

type BlockedArgs struct {
	JSON bool
}

func parseBlockedArgs(raw []string) (BlockedArgs, error) {
	a, err := ParseArgs(raw, nil, []string{"--json"})
	if err != nil {
		return BlockedArgs{}, err
	}
	return BlockedArgs{JSON: a.JSON()}, nil
}

func cmdBlocked(args []string, w io.Writer) error {
	ba, err := parseBlockedArgs(args)
	if err != nil {
		return err
	}

	_, store, err := getInitializedRepo()
	if err != nil {
		return err
	}

	blocked, err := store.Blocked()
	if err != nil {
		return err
	}

	if ba.JSON {
		fprintJSON(w, blocked)
		return nil
	}

	if len(blocked) == 0 {
		fmt.Fprintln(w, "no blocked issues")
		return nil
	}

	fmt.Fprintf(w, "\n● Blocked (%d):\n", len(blocked))

	for _, bi := range blocked {
		fmt.Fprintf(w, "\n[%s P%d] %s: %s\n",
			issue.PriorityDot(bi.Priority),
			bi.Priority,
			bi.ID,
			bi.Title,
		)
		fmt.Fprintf(w, "  Blocked by: %s\n", strings.Join(bi.OpenBlockers, ", "))
	}
	return nil
}
