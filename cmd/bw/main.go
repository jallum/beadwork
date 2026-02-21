package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

const version = "0.6.1"

func main() {
	var w Writer
	if term.IsTerminal(int(os.Stdout.Fd())) && os.Getenv("NO_COLOR") == "" {
		width, _, _ := term.GetSize(int(os.Stdout.Fd()))
		w = ColorWriter(os.Stdout, width)
	} else {
		w = PlainWriter(os.Stdout)
	}

	if len(os.Args) < 2 {
		printUsage(w)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

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

	if err := c.Run(args, w); err != nil {
		fatal(err.Error())
	}
}
