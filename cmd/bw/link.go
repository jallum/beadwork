package main

import (
	"fmt"
	"io"
)

type LinkArgs struct {
	BlockerID string
	BlockedID string
}

func parseLinkArgs(raw []string) (LinkArgs, error) {
	if len(raw) < 3 || raw[1] != "blocks" {
		return LinkArgs{}, fmt.Errorf("usage: bw link <id> blocks <id>")
	}
	return LinkArgs{BlockerID: raw[0], BlockedID: raw[2]}, nil
}

func cmdLink(args []string, w io.Writer) error {
	la, err := parseLinkArgs(args)
	if err != nil {
		return err
	}

	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	if err := store.Link(la.BlockerID, la.BlockedID); err != nil {
		return err
	}

	// Resolve full IDs for the commit message
	blocker, _ := store.Get(la.BlockerID)
	blocked, _ := store.Get(la.BlockedID)
	intent := fmt.Sprintf("link %s blocks %s", blocker.ID, blocked.ID)
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	fmt.Fprintf(w, "linked %s blocks %s\n", blocker.ID, blocked.ID)
	return nil
}

type UnlinkArgs struct {
	BlockerID string
	BlockedID string
}

func parseUnlinkArgs(raw []string) (UnlinkArgs, error) {
	if len(raw) < 3 || raw[1] != "blocks" {
		return UnlinkArgs{}, fmt.Errorf("usage: bw unlink <id> blocks <id>")
	}
	return UnlinkArgs{BlockerID: raw[0], BlockedID: raw[2]}, nil
}

func cmdUnlink(args []string, w io.Writer) error {
	ua, err := parseUnlinkArgs(args)
	if err != nil {
		return err
	}

	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	if err := store.Unlink(ua.BlockerID, ua.BlockedID); err != nil {
		return err
	}

	blocker, _ := store.Get(ua.BlockerID)
	blocked, _ := store.Get(ua.BlockedID)
	intent := fmt.Sprintf("unlink %s blocks %s", blocker.ID, blocked.ID)
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	fmt.Fprintf(w, "unlinked %s blocks %s\n", blocker.ID, blocked.ID)
	return nil
}
