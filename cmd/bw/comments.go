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

// foldCommentOptionValues groups a comment option with its value so global
// flags such as --help and -C cannot consume a filename or author.
func foldCommentOptionValues(args []string) []string {
	folded := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--":
			return append(folded, args[i:]...)
		case "-a", "--author", "-F", "--file":
			if i+1 < len(args) {
				folded = append(folded, arg+"="+args[i+1])
				i++
				continue
			}
		case "-C", "--x-render-as":
			if i+1 < len(args) {
				folded = append(folded, args[i:i+2]...)
				i++
				continue
			}
		}
		folded = append(folded, arg)
	}
	return folded
}

func parseCommentArgs(raw []string) (CommentArgs, error) {
	var ca CommentArgs
	var pos []string
	hasFile := false
	args := foldCommentOptionValues(raw)
	for i, arg := range args {
		if arg == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		name, value, attached := strings.Cut(arg, "=")
		switch {
		case name == "-F" || name == "--file":
			if !attached || value == "" {
				return ca, fmt.Errorf("--file requires a path (use - for stdin)")
			}
			ca.File = value
			hasFile = true
		case name == "-a" || name == "--author":
			if !attached {
				return ca, fmt.Errorf("--author requires a value")
			}
			ca.Author = value
		case arg == "--json":
			ca.JSON = true
		case strings.HasPrefix(name, "-") && name != "-" && !strings.ContainsAny(name, " \t\r\n"):
			return ca, fmt.Errorf("unknown flag: %s", arg)
		default:
			pos = append(pos, arg)
		}
	}

	switch {
	case len(pos) == 0 || (len(pos) == 1 && !hasFile):
		return ca, errors.New(commentUsage)
	case len(pos) > 2:
		return ca, fmt.Errorf("too many arguments: quote multi-word comment text, or pass it with --file")
	case len(pos) == 2 && hasFile:
		return ca, fmt.Errorf("provide the comment body as <text> or --file, not both")
	}
	ca.ID = pos[0]
	if len(pos) == 2 {
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
