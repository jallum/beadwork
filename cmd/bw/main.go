package main

import (
	"fmt"
	"os"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "--version", "-v":
		fmt.Println("bw " + version)
	case "init":
		cmdInit(args)
	case "create":
		cmdCreate(args)
	case "show":
		cmdShow(args)
	case "list":
		cmdList(args)
	case "update":
		cmdUpdate(args)
	case "close":
		cmdClose(args)
	case "reopen":
		cmdReopen(args)
	case "ready":
		cmdReady(args)
	case "graph":
		cmdGraph(args)
	case "label":
		cmdLabel(args)
	case "link":
		cmdLink(args)
	case "unlink":
		cmdUnlink(args)
	case "config":
		cmdConfig(args)
	case "sync":
		cmdSync(args)
	case "onboard":
		cmdOnboard()
	case "prime":
		cmdPrime()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: bw <command> [args]

Commands:
  init [--prefix <name>]      Initialize beadwork
  create <title> [flags]      Create an issue
  show <id> [--json]          Show issue details
  list [filters] [--json]     List issues
  update <id> [flags]         Update an issue
  close <id> [--reason <r>]   Close an issue
  reopen <id>                 Reopen a closed issue
  ready [--json]              List issues with no open blockers
  label <id> +lab [-lab] ...  Add/remove labels
  link <id> blocks <id>       Create dependency link
  unlink <id> blocks <id>     Remove dependency link
  graph <id>|--all [--json]   Dependency graph
  config get|set|list         View/set config options
  sync                        Fetch, rebase/replay, push
  onboard                     Print AGENTS.md snippet
  prime                       Print workflow context for agents
`)
}
