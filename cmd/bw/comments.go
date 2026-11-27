package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
)

// commentStdin is the reader used when the comment body is read from stdin
// (--file -). It is a package var so tests can substitute a fixed reader.
var commentStdin io.Reader = os.Stdin

const commentUsage = "usage: bw comment <id> <text> [--author NAME] [--json]\n" +
	"   or: bw comment <id> --file <path> [--author NAME] [--json]   (use --file - to read the body from stdin)"

type CommentArgs struct {
	ID     string
	Text   string // body from the <text> positional; empty when --file is used
	File   string // path from --file ("-" means stdin); empty when <text> is used
	Author string
	JSON   bool
}

func parseCommentArgs(raw []string) (CommentArgs, error) {
	if len(raw) == 0 {
		return CommentArgs{}, errors.New(commentUsage)
	}

	rest := expandAliases(raw, []Flag{
		{Long: "--author", Short: "-a", Value: "NAME"},
		{Long: "--file", Short: "-F", Value: "PATH"},
		{Long: "--json"},
	})
	a, err := ParseArgs(rest, []string{"--author", "--file"}, []string{"--json"})
	if err != nil {
		return CommentArgs{}, err
	}
	pos := a.Pos()
	if len(pos) < 1 {
		return CommentArgs{}, errors.New(commentUsage)
	}
	if len(pos) > 2 {
		return CommentArgs{}, fmt.Errorf("too many arguments: quote multi-word comment text, or pass it with --file")
	}

	hasFile := a.Has("--file")
	hasText := len(pos) == 2

	switch {
	case hasFile && hasText:
		return CommentArgs{}, fmt.Errorf("provide the comment body as <text> or --file, not both")
	case !hasFile && !hasText:
		return CommentArgs{}, errors.New(commentUsage)
	case hasFile && a.String("--file") == "":
		return CommentArgs{}, fmt.Errorf("--file requires a path (use - for stdin)")
	}

	ca := CommentArgs{
		ID:     pos[0],
		File:   a.String("--file"),
		Author: a.String("--author"),
		JSON:   a.JSON(),
	}
	if hasText {
		ca.Text = pos[1]
	}
	return ca, nil
}

// readCommentBody returns the comment body from a file path, or from stdin when
// path is "-". Trailing newlines (commonly appended by editors and heredocs)
// are trimmed; internal blank lines and leading whitespace are preserved.
func readCommentBody(path string) (string, error) {
	var (
		data []byte
		err  error
	)
	if path == "-" {
		data, err = io.ReadAll(commentStdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\n"), nil
}

func cmdComment(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	ca, err := parseCommentArgs(args)
	if err != nil {
		return nil, err
	}

	body := ca.Text
	if ca.File != "" {
		body, err = readCommentBody(ca.File)
		if err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("comment body is empty")
	}

	var iss *issue.Issue
	err = commitWithRetry(store, commitMaxRetries, func() (string, error) {
		var cerr error
		iss, cerr = store.Comment(ca.ID, body, ca.Author)
		if cerr != nil {
			return "", cerr
		}
		return fmt.Sprintf("comment %s %q", iss.ID, body), nil
	})
	if err != nil {
		return nil, err
	}

	if ca.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "comment added to %s\n", w.Style(iss.ID, Cyan))
	}
	return nil, nil
}
