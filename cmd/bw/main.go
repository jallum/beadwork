package main

import (
	"fmt"
	"os"
)

const version = "0.4.1"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]
	w := os.Stdout

	var err error
	switch cmd {
	case "--version", "-v":
		fmt.Fprintln(w, "bw "+version)
	case "init":
		err = cmdInit(args, w)
	case "create":
		err = cmdCreate(args, w)
	case "show":
		err = cmdShow(args, w)
	case "list":
		err = cmdList(args, w)
	case "update":
		err = cmdUpdate(args, w)
	case "close":
		err = cmdClose(args, w)
	case "reopen":
		err = cmdReopen(args, w)
	case "ready":
		err = cmdReady(args, w)
	case "graph":
		err = cmdGraph(args, w)
	case "label":
		err = cmdLabel(args, w)
	case "link":
		err = cmdLink(args, w)
	case "unlink":
		err = cmdUnlink(args, w)
	case "config":
		err = cmdConfig(args, w)
	case "export":
		err = cmdExport(args, w)
	case "import":
		err = cmdImport(args, w)
	case "sync":
		err = cmdSync(args, w)
	case "onboard":
		err = cmdOnboard(w)
	case "prime":
		err = cmdPrime(w)
	case "upgrade":
		err = cmdUpgrade(args, w)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
	if err != nil {
		fatal(err.Error())
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
