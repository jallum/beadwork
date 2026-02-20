package main

import (
	"fmt"
	"io"
)

func cmdLink(args []string, w io.Writer) error {
	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	// bw link <id1> blocks <id2>
	if len(args) < 3 || args[1] != "blocks" {
		return fmt.Errorf("usage: bw link <id> blocks <id>")
	}
	blockerID := args[0]
	blockedID := args[2]

	if err := store.Link(blockerID, blockedID); err != nil {
		return err
	}

	// Resolve full IDs for the commit message
	blocker, _ := store.Get(blockerID)
	blocked, _ := store.Get(blockedID)
	intent := fmt.Sprintf("link %s blocks %s", blocker.ID, blocked.ID)
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	fmt.Fprintf(w, "linked %s blocks %s\n", blocker.ID, blocked.ID)
	return nil
}

func cmdUnlink(args []string, w io.Writer) error {
	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	// bw unlink <id1> blocks <id2>
	if len(args) < 3 || args[1] != "blocks" {
		return fmt.Errorf("usage: bw unlink <id> blocks <id>")
	}
	blockerID := args[0]
	blockedID := args[2]

	if err := store.Unlink(blockerID, blockedID); err != nil {
		return err
	}

	blocker, _ := store.Get(blockerID)
	blocked, _ := store.Get(blockedID)
	intent := fmt.Sprintf("unlink %s blocks %s", blocker.ID, blocked.ID)
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	fmt.Fprintf(w, "unlinked %s blocks %s\n", blocker.ID, blocked.ID)
	return nil
}
