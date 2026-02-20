package main

import (
	"fmt"
	"os"
)

const version = "0.3.3"

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
	case "export":
		cmdExport(args)
	case "import":
		cmdImport(args)
	case "sync":
		cmdSync(args)
	case "onboard":
		cmdOnboard()
	case "prime":
		cmdPrime()
	case "upgrade":
		cmdUpgrade(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: bw <command> [args]

Issues:
  create <title> [flags]      Create an issue
  show <id> [--json]          Show issue details
  list [filters] [--json]     List issues
  update <id> [flags]         Update an issue
  close <id> [--reason <r>]   Close an issue
  reopen <id>                 Reopen a closed issue
  label <id> +lab [-lab] ...  Add/remove labels

Dependencies:
  link <id> blocks <id>       Create dependency link
  unlink <id> blocks <id>     Remove dependency link
  ready [--json]              List unblocked issues
  graph <id>|--all [--json]   Dependency graph

Collaboration:
  sync                        Fetch, rebase/replay, push
  export [--status <s>]       Export issues as JSONL
  import <file> [--dry-run]   Import issues from JSONL

Setup & Config:
  init [--prefix] [--force]   Initialize beadwork
  config get|set|list         View/set config options
  upgrade [--check] [--yes]   Check for / install updates
  onboard                     Print AGENTS.md snippet
  prime                       Print workflow context for agents
`)
}
