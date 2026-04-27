package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
)

// AttachArgs holds parsed `bw attach` arguments.
type AttachArgs struct {
	TicketID   string
	FilePath   string
	StoredPath string // value of --name; empty means use basename(FilePath)
}

func parseAttachArgs(raw []string) (AttachArgs, error) {
	rest := expandAliases(raw, []Flag{
		{Long: "--name", Value: "PATH"},
	})
	a, err := ParseArgs(rest, []string{"--name"}, nil)
	if err != nil {
		return AttachArgs{}, err
	}
	pos := a.Pos()
	if len(pos) < 1 {
		return AttachArgs{}, fmt.Errorf("usage: bw attach <ticket-id> <file-path> [--name <stored-path>]")
	}
	if len(pos) < 2 {
		return AttachArgs{}, fmt.Errorf("usage: bw attach <ticket-id> <file-path> [--name <stored-path>]")
	}
	return AttachArgs{
		TicketID:   pos[0],
		FilePath:   pos[1],
		StoredPath: a.String("--name"),
	}, nil
}

// cmdAttach implements `bw attach <ticket-id> <file-path> [--name <stored-path>]`.
//
// It reads <file-path> from disk, writes the bytes as a blob under
// attachments/<ticket-id>/<stored-path>, and commits with a message
// containing exactly one line: "attach <ticket-id> <stored-path>". When
// --name is not given the stored path defaults to filepath.Base of the
// source file. See docs/design.md for the on-disk layout and the
// matching intent grammar.
func cmdAttach(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	aa, err := parseAttachArgs(args)
	if err != nil {
		return nil, err
	}
	storedPath := aa.StoredPath
	if storedPath == "" {
		storedPath = filepath.Base(aa.FilePath)
	}

	data, err := os.ReadFile(aa.FilePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", aa.FilePath, err)
	}

	if err := store.Attach(aa.TicketID, storedPath, data); err != nil {
		return nil, err
	}
	intent := fmt.Sprintf("attach %s %s", aa.TicketID, storedPath)
	if err := store.Commit(intent); err != nil {
		return nil, fmt.Errorf("commit failed: %w", err)
	}

	fmt.Fprintf(w, "attached %s to %s\n", w.Style(storedPath, Cyan), w.Style(aa.TicketID, Cyan))
	return nil, nil
}
