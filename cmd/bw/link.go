package main

import "fmt"

func cmdLink(args []string) {
	r, store := mustInitialized()

	// bw link <id1> blocks <id2>
	if len(args) < 3 || args[1] != "blocks" {
		fatal("usage: bw link <id> blocks <id>")
	}
	blockerID := args[0]
	blockedID := args[2]

	if err := store.Link(blockerID, blockedID); err != nil {
		fatal(err.Error())
	}

	// Resolve full IDs for the commit message
	blocker, _ := store.Get(blockerID)
	blocked, _ := store.Get(blockedID)
	intent := fmt.Sprintf("link %s blocks %s", blocker.ID, blocked.ID)
	if err := r.Commit(intent); err != nil {
		fatal("commit failed: " + err.Error())
	}

	fmt.Printf("linked %s blocks %s\n", blocker.ID, blocked.ID)
}

func cmdUnlink(args []string) {
	r, store := mustInitialized()

	// bw unlink <id1> blocks <id2>
	if len(args) < 3 || args[1] != "blocks" {
		fatal("usage: bw unlink <id> blocks <id>")
	}
	blockerID := args[0]
	blockedID := args[2]

	if err := store.Unlink(blockerID, blockedID); err != nil {
		fatal(err.Error())
	}

	blocker, _ := store.Get(blockerID)
	blocked, _ := store.Get(blockedID)
	intent := fmt.Sprintf("unlink %s blocks %s", blocker.ID, blocked.ID)
	if err := r.Commit(intent); err != nil {
		fatal("commit failed: " + err.Error())
	}

	fmt.Printf("unlinked %s blocks %s\n", blocker.ID, blocked.ID)
}
