package main

import (
	"fmt"
	"os"

	"github.com/jallum/beadwork/internal/issue"
	"golang.org/x/term"
)

const version = "0.11.6"

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

	if len(os.Args) < 2 {
		printUsage(w)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

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
