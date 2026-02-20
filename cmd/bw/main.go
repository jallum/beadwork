package main

import (
	"fmt"
	"os"
)

const version = "0.4.3"

func main() {
	if len(os.Args) < 2 {
		printUsage(os.Stderr)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]
	w := os.Stdout

	switch cmd {
	case "--version", "-v":
		fmt.Fprintln(w, "bw "+version)
		return
	case "--help", "-h":
		printUsage(os.Stdout)
		return
	}

	c, ok := commandMap[cmd]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage(os.Stderr)
		os.Exit(1)
	}

	if hasFlag(args, "--help") || hasFlag(args, "-h") {
		printCommandHelp(os.Stdout, c)
		return
	}

	if err := c.Run(args, w); err != nil {
		fatal(err.Error())
	}
}
