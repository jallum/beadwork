package main

import (
	"fmt"
	"strings"
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

func cmdBlocked(args []string, w Writer) error {
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
		ps := PriorityStyle(bi.Priority)
		fmt.Fprintf(w, "\n[%s %s] %s: %s\n",
			w.Style("●", ps),
			w.Style(fmt.Sprintf("P%d", bi.Priority), ps),
			bi.ID,
			bi.Title,
		)
		styledBlockers := make([]string, len(bi.OpenBlockers))
		for i, id := range bi.OpenBlockers {
			styledBlockers[i] = w.Style(id, Red)
		}
		fmt.Fprintf(w, "  Blocked by: %s\n", strings.Join(styledBlockers, ", "))
		if len(bi.Blocks) > 0 {
			styledBlocks := make([]string, len(bi.Blocks))
			for i, id := range bi.Blocks {
				styledBlocks[i] = w.Style(id, Red)
			}
			fmt.Fprintf(w, "  Blocks: %s\n", strings.Join(styledBlocks, ", "))
		}
	}
	return nil
}
