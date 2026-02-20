package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

const version = "0.5.3"

func main() {
	if len(os.Args) < 2 {
		printUsage(PlainWriter(os.Stderr))
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var w Writer
	if term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("NO_COLOR") == "" {
		w = ColorWriter(os.Stdout)
	} else {
		w = PlainWriter(os.Stdout)
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
		printUsage(PlainWriter(os.Stderr))
		os.Exit(1)
	}

	if hasFlag(args, "--help") || hasFlag(args, "-h") {
		printCommandHelp(w, c)
		return
	}

	if err := c.Run(args, w); err != nil {
		fatal(err.Error())
	}
}
