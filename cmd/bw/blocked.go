package main

import (
	"fmt"
	"strings"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/md"
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

func cmdBlocked(store *issue.Store, args []string, w Writer) error {
	ba, err := parseBlockedArgs(args)
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

	fmt.Fprintf(w, "\nBlocked (%d):\n", len(blocked))

	for _, bi := range blocked {
		fmt.Fprintln(w)
		emitln(w, fmt.Sprintf("{p:%d} {id:%s} %s",
			bi.Priority, bi.ID, md.Escape(bi.Title)))
		w.Push(2)

		blockerIDs := make([]string, len(bi.OpenBlockers))
		for i, id := range bi.OpenBlockers {
			blockerIDs[i] = id
		}
		fmt.Fprintf(w, "Blocked by: %s\n", strings.Join(blockerIDs, ", "))

		if len(bi.Blocks) > 0 {
			blockIDs := make([]string, len(bi.Blocks))
			for i, id := range bi.Blocks {
				blockIDs[i] = id
			}
			fmt.Fprintf(w, "Blocks: %s\n", strings.Join(blockIDs, ", "))
		}
		w.Pop()
	}
	return nil
}
