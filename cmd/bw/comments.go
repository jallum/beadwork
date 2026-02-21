package main

import (
	"fmt"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/repo"
	"github.com/jallum/beadwork/internal/wrap"
)

type CommentsArgs struct {
	SubCmd string // "add" or "" (list)
	ID     string
	Text   string
	Author string
	JSON   bool
}

func parseCommentsArgs(raw []string) (CommentsArgs, error) {
	if len(raw) == 0 {
		return CommentsArgs{}, fmt.Errorf("usage: bw comments [add] <id> [text] [--author NAME] [--json]")
	}

	ca := CommentsArgs{}

	if raw[0] == "add" {
		ca.SubCmd = "add"
		rest := raw[1:]
		if len(rest) == 0 {
			return CommentsArgs{}, fmt.Errorf("usage: bw comments add <id> <text> [--author NAME]")
		}
		// Expand short flags
		rest = expandAliases(rest, []Flag{
			{Long: "--author", Short: "-a", Value: "NAME"},
			{Long: "--json"},
		})
		a, err := ParseArgs(rest, []string{"--author"}, []string{"--json"})
		if err != nil {
			return CommentsArgs{}, err
		}
		pos := a.Pos()
		if len(pos) < 1 {
			return CommentsArgs{}, fmt.Errorf("usage: bw comments add <id> <text> [--author NAME]")
		}
		ca.ID = pos[0]
		if len(pos) < 2 {
			return CommentsArgs{}, fmt.Errorf("usage: bw comments add <id> <text> [--author NAME]")
		}
		ca.Text = pos[1]
		ca.Author = a.String("--author")
		ca.JSON = a.JSON()
	} else {
		// List mode
		rest := expandAliases(raw, []Flag{
			{Long: "--json"},
		})
		a, err := ParseArgs(rest, nil, []string{"--json"})
		if err != nil {
			return CommentsArgs{}, err
		}
		pos := a.Pos()
		if len(pos) < 1 {
			return CommentsArgs{}, fmt.Errorf("usage: bw comments <id> [--json]")
		}
		ca.ID = pos[0]
		ca.JSON = a.JSON()
	}

	return ca, nil
}

func cmdComments(r *repo.Repo, store *issue.Store, args []string, w Writer) error {
	ca, err := parseCommentsArgs(args)
	if err != nil {
		return err
	}

	if ca.SubCmd == "add" {
		iss, err := store.Comment(ca.ID, ca.Text, ca.Author)
		if err != nil {
			return err
		}

		intent := fmt.Sprintf("comment %s %q", iss.ID, ca.Text)
		if err := r.Commit(intent); err != nil {
			return fmt.Errorf("commit failed: %w", err)
		}

		if ca.JSON {
			fprintJSON(w, iss)
		} else {
			fmt.Fprintf(w, "comment added to %s\n", w.Style(iss.ID, Cyan))
		}
		return nil
	}

	// List mode
	iss, err := store.Get(ca.ID)
	if err != nil {
		return err
	}

	if ca.JSON {
		fprintJSON(w, iss.Comments)
		return nil
	}

	if len(iss.Comments) == 0 {
		fmt.Fprintf(w, "no comments on %s\n", w.Style(iss.ID, Cyan))
		return nil
	}

	fmt.Fprintf(w, "Comments on %s:\n", w.Style(iss.ID, Cyan))
	w.Push(2)
	for _, c := range iss.Comments {
		ts := trimDate(c.Timestamp)
		if c.Author != "" {
			fmt.Fprintf(w, "\n%s %s\n", w.Style(ts, Dim), w.Style(c.Author, Bold))
		} else {
			fmt.Fprintf(w, "\n%s\n", w.Style(ts, Dim))
		}
		w.Push(2)
		text := c.Text
		if ww := w.Width(); ww > 0 {
			text = wrap.Text(text, ww)
		}
		fmt.Fprintln(w, text)
		w.Pop()
	}
	w.Pop()
	return nil
}
