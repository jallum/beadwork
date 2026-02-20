package main

import (
	"fmt"
	"io"
)

// DepArgs holds the parsed subcommand and IDs for "bw dep add|remove".
type DepArgs struct {
	Subcmd    string // "add" or "remove"
	BlockerID string
	BlockedID string
}

func parseDepArgs(raw []string) (DepArgs, error) {
	if len(raw) == 0 {
		return DepArgs{}, fmt.Errorf("usage: bw dep add|remove <id> blocks <id>")
	}
	da := DepArgs{Subcmd: raw[0]}
	switch da.Subcmd {
	case "add":
		a, err := parseDepAddArgs(raw[1:])
		if err != nil {
			return da, err
		}
		da.BlockerID = a.BlockerID
		da.BlockedID = a.BlockedID
	case "remove":
		a, err := parseDepRemoveArgs(raw[1:])
		if err != nil {
			return da, err
		}
		da.BlockerID = a.BlockerID
		da.BlockedID = a.BlockedID
	default:
		return da, fmt.Errorf("usage: bw dep add|remove <id> blocks <id>")
	}
	return da, nil
}

func cmdDep(args []string, w io.Writer) error {
	da, err := parseDepArgs(args)
	if err != nil {
		return err
	}
	switch da.Subcmd {
	case "add":
		return cmdDepAdd([]string{da.BlockerID, "blocks", da.BlockedID}, w)
	case "remove":
		return cmdDepRemove([]string{da.BlockerID, "blocks", da.BlockedID}, w)
	}
	return nil
}

type DepAddArgs struct {
	BlockerID string
	BlockedID string
}

func parseDepAddArgs(raw []string) (DepAddArgs, error) {
	if len(raw) < 3 || raw[1] != "blocks" {
		return DepAddArgs{}, fmt.Errorf("usage: bw dep add <id> blocks <id>")
	}
	return DepAddArgs{BlockerID: raw[0], BlockedID: raw[2]}, nil
}

func cmdDepAdd(args []string, w io.Writer) error {
	la, err := parseDepAddArgs(args)
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

	// Resolve full IDs for the commit message.
	// Intent verb stays "link" for replay compatibility.
	blocker, _ := store.Get(la.BlockerID)
	blocked, _ := store.Get(la.BlockedID)
	intent := fmt.Sprintf("link %s blocks %s", blocker.ID, blocked.ID)
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	fmt.Fprintf(w, "added dep %s blocks %s\n", blocker.ID, blocked.ID)
	return nil
}

type DepRemoveArgs struct {
	BlockerID string
	BlockedID string
}

func parseDepRemoveArgs(raw []string) (DepRemoveArgs, error) {
	if len(raw) < 3 || raw[1] != "blocks" {
		return DepRemoveArgs{}, fmt.Errorf("usage: bw dep remove <id> blocks <id>")
	}
	return DepRemoveArgs{BlockerID: raw[0], BlockedID: raw[2]}, nil
}

func cmdDepRemove(args []string, w io.Writer) error {
	ua, err := parseDepRemoveArgs(args)
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
	// Intent verb stays "unlink" for replay compatibility.
	intent := fmt.Sprintf("unlink %s blocks %s", blocker.ID, blocked.ID)
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	fmt.Fprintf(w, "removed dep %s blocks %s\n", blocker.ID, blocked.ID)
	return nil
}
