package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jallum/beadwork/internal/issue"
	"golang.org/x/term"
)

const version = "0.12.3"

func resolveRenderMode(args []string) string {
	if mode, ok := flagValue(args, "--x-render-as"); ok && mode != "" {
		return mode
	}
	if hasFlag(args, "--x-raw") {
		return "raw"
	}
	if term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("NO_COLOR") == "" {
		return "tty"
	}
	return "markdown"
}

func main() {
	var w Writer
	switch resolveRenderMode(os.Args) {
	case "tty":
		width := 80
		if term.IsTerminal(int(os.Stdout.Fd())) {
			width, _, _ = term.GetSize(int(os.Stdout.Fd()))
		}
		w = ColorWriter(os.Stdout, width)
	case "raw":
		w = RawWriter(os.Stdout)
	default:
		w = PlainWriter(os.Stdout)
	}

	allArgs := os.Args[1:]
	allArgs = extractDirFlag(allArgs)

	if len(allArgs) < 1 {
		printUsage(w)
		os.Exit(1)
	}

	cmd := allArgs[0]
	args := allArgs[1:]

	args = removeFlag(args, "--x-raw")
	args, _ = removeFlagValue(args, "--x-render-as")

	dryRun := hasFlag(args, "--dry-run")
	if dryRun {
		args = removeFlag(args, "--dry-run")
	}

	switch cmd {
	case "--version", "-v":
		fmt.Fprintln(w, "bw "+version)
		return
	case "--help", "-h":
		printUsage(w)
		return
	}

	c, ok := commandMap[cmd]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage(w)
		os.Exit(1)
	}

	if hasFlag(args, "--help") || hasFlag(args, "-h") {
		printCommandHelp(w, c)
		return
	}

	var store *issue.Store
	if c.NeedsStore {
		var err error
		store, err = getInitializedStore()
		if err != nil {
			fatal(err.Error())
		}
		store.DryRun = dryRun
		maybeCheckForUpgrade(store, w)
	}

	if err := c.Run(store, args, w); err != nil {
		fatal(err.Error())
	}
}

// extractDirFlag removes all -C <dir> pairs from args and sets repoDir.
func extractDirFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "-C" {
			if i+1 >= len(args) {
				fatal("-C requires an argument")
			}
			abs, err := filepath.Abs(args[i+1])
			if err != nil {
				fatal(fmt.Sprintf("-C %s: %s", args[i+1], err))
			}
			repoDir = abs
			i++ // skip value
		} else {
			out = append(out, args[i])
		}
	}
	return out
}
