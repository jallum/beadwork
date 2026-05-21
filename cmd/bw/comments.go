package main

import (
	"fmt"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
)

type CommentArgs struct {
	ID     string
	Text   string
	Author string
	JSON   bool
}

func parseCommentArgs(raw []string) (CommentArgs, error) {
	if len(raw) == 0 {
		return CommentArgs{}, fmt.Errorf("usage: bw comment <id> <text> [--author NAME] [--json]")
	}

	rest := expandAliases(raw, []Flag{
		{Long: "--author", Short: "-a", Value: "NAME"},
		{Long: "--json"},
	})
	a, err := ParseArgs(rest, []string{"--author"}, []string{"--json"})
	if err != nil {
		return CommentArgs{}, err
	}
	pos := a.Pos()
	if len(pos) < 1 {
		return CommentArgs{}, fmt.Errorf("usage: bw comment <id> <text> [--author NAME] [--json]")
	}
	if len(pos) < 2 {
		return CommentArgs{}, fmt.Errorf("usage: bw comment <id> <text> [--author NAME] [--json]")
	}

	return CommentArgs{
		ID:     pos[0],
		Text:   pos[1],
		Author: a.String("--author"),
		JSON:   a.JSON(),
	}, nil
}

func cmdComment(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	ca, err := parseCommentArgs(args)
	if err != nil {
		return nil, err
	}

	var iss *issue.Issue
	err = commitWithRetry(store, commitMaxRetries, func() (string, error) {
		var cerr error
		iss, cerr = store.Comment(ca.ID, ca.Text, ca.Author)
		if cerr != nil {
			return "", cerr
		}
		return fmt.Sprintf("comment %s %q", iss.ID, ca.Text), nil
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
