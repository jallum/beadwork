package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/j5n/beadwork/internal/intent"
	"github.com/j5n/beadwork/internal/issue"
	"github.com/j5n/beadwork/internal/repo"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
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
	case "sync":
		cmdSync(args)
	case "onboard":
		cmdOnboard()
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
  onboard                     Print tool description for agents
`)
}

func mustRepo() *repo.Repo {
	r, err := repo.FindRepo()
	if err != nil {
		fatal(err.Error())
	}
	return r
}

func mustInitialized() (*repo.Repo, *issue.Store) {
	r := mustRepo()
	if !r.IsInitialized() {
		fatal("beadwork not initialized. Run: bw init")
	}
	return r, issue.NewStore(r.WorkTree, r.Prefix)
}

func fatal(msg string) {
	fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	os.Exit(1)
}

// --- Commands ---

func cmdInit(args []string) {
	prefix := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--prefix" && i+1 < len(args) {
			prefix = args[i+1]
			i++
		}
	}

	r := mustRepo()
	if r.IsInitialized() {
		fatal("beadwork already initialized")
	}
	if err := r.Init(prefix); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("initialized beadwork (prefix: %s)\n", r.Prefix)
}

func cmdCreate(args []string) {
	r, store := mustInitialized()
	opts := issue.CreateOpts{}
	var title string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--priority", "-p":
			if i+1 < len(args) {
				p, err := strconv.Atoi(args[i+1])
				if err != nil {
					fatal("invalid priority: " + args[i+1])
				}
				opts.Priority = p
				i++
			}
		case "--type", "-t":
			if i+1 < len(args) {
				opts.Type = args[i+1]
				i++
			}
		case "--assignee", "-a":
			if i+1 < len(args) {
				opts.Assignee = args[i+1]
				i++
			}
		case "--description", "-d":
			if i+1 < len(args) {
				opts.Description = args[i+1]
				i++
			}
		default:
			if title == "" {
				title = args[i]
			} else {
				title += " " + args[i]
			}
		}
	}

	if title == "" {
		fatal("title is required")
	}

	iss, err := store.Create(title, opts)
	if err != nil {
		fatal(err.Error())
	}

	intent := fmt.Sprintf("create %s p%d %s %q", iss.ID, iss.Priority, iss.Type, iss.Title)
	if err := r.Commit(intent); err != nil {
		fatal("commit failed: " + err.Error())
	}

	if hasFlag(os.Args, "--json") {
		printJSON(iss)
	} else {
		fmt.Printf("created %s: %s\n", iss.ID, iss.Title)
	}
}

func cmdShow(args []string) {
	_, store := mustInitialized()

	if len(args) == 0 {
		fatal("usage: bw show <id>")
	}
	id := args[0]

	iss, err := store.Get(id)
	if err != nil {
		fatal(err.Error())
	}

	if hasFlag(args, "--json") {
		printJSON(iss)
	} else {
		printIssue(iss)
	}
}

func cmdList(args []string) {
	_, store := mustInitialized()

	filter := issue.Filter{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--status":
			if i+1 < len(args) {
				filter.Status = args[i+1]
				i++
			}
		case "--assignee":
			if i+1 < len(args) {
				filter.Assignee = args[i+1]
				i++
			}
		case "--priority":
			if i+1 < len(args) {
				p, _ := strconv.Atoi(args[i+1])
				filter.Priority = p
				i++
			}
		case "--type":
			if i+1 < len(args) {
				filter.Type = args[i+1]
				i++
			}
		case "--label":
			if i+1 < len(args) {
				filter.Label = args[i+1]
				i++
			}
		}
	}

	issues, err := store.List(filter)
	if err != nil {
		fatal(err.Error())
	}

	if hasFlag(args, "--json") {
		printJSON(issues)
	} else {
		if len(issues) == 0 {
			fmt.Println("no issues found")
			return
		}
		for _, iss := range issues {
			fmt.Printf("%-14s p%d %-12s %-12s %s\n", iss.ID, iss.Priority, iss.Status, iss.Type, iss.Title)
		}
	}
}

func cmdUpdate(args []string) {
	r, store := mustInitialized()

	if len(args) == 0 {
		fatal("usage: bw update <id> [flags]")
	}
	id := args[0]
	rest := args[1:]

	opts := issue.UpdateOpts{}
	var changes []string

	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--title":
			if i+1 < len(rest) {
				opts.Title = &rest[i+1]
				changes = append(changes, "title="+rest[i+1])
				i++
			}
		case "--description", "-d":
			if i+1 < len(rest) {
				opts.Description = &rest[i+1]
				changes = append(changes, "description=...")
				i++
			}
		case "--priority", "-p":
			if i+1 < len(rest) {
				p, err := strconv.Atoi(rest[i+1])
				if err != nil {
					fatal("invalid priority: " + rest[i+1])
				}
				opts.Priority = &p
				changes = append(changes, "priority="+rest[i+1])
				i++
			}
		case "--assignee", "-a":
			if i+1 < len(rest) {
				opts.Assignee = &rest[i+1]
				changes = append(changes, "assignee="+rest[i+1])
				i++
			}
		case "--type", "-t":
			if i+1 < len(rest) {
				opts.Type = &rest[i+1]
				changes = append(changes, "type="+rest[i+1])
				i++
			}
		case "--status", "-s":
			if i+1 < len(rest) {
				opts.Status = &rest[i+1]
				changes = append(changes, "status="+rest[i+1])
				i++
			}
		}
	}

	iss, err := store.Update(id, opts)
	if err != nil {
		fatal(err.Error())
	}

	intent := fmt.Sprintf("update %s %s", iss.ID, strings.Join(changes, " "))
	if err := r.Commit(intent); err != nil {
		fatal("commit failed: " + err.Error())
	}

	if hasFlag(os.Args, "--json") {
		printJSON(iss)
	} else {
		fmt.Printf("updated %s\n", iss.ID)
	}
}

func cmdClose(args []string) {
	r, store := mustInitialized()

	if len(args) == 0 {
		fatal("usage: bw close <id> [--reason <reason>]")
	}
	id := args[0]
	reason := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--reason" && i+1 < len(args) {
			reason = args[i+1]
			i++
		}
	}

	iss, err := store.Close(id)
	if err != nil {
		fatal(err.Error())
	}

	intent := fmt.Sprintf("close %s", iss.ID)
	if reason != "" {
		intent += fmt.Sprintf(" reason=%q", reason)
	}
	if err := r.Commit(intent); err != nil {
		fatal("commit failed: " + err.Error())
	}

	if hasFlag(os.Args, "--json") {
		printJSON(iss)
	} else {
		fmt.Printf("closed %s: %s\n", iss.ID, iss.Title)
	}
}

func cmdReopen(args []string) {
	r, store := mustInitialized()

	if len(args) == 0 {
		fatal("usage: bw reopen <id>")
	}
	id := args[0]

	iss, err := store.Reopen(id)
	if err != nil {
		fatal(err.Error())
	}

	intent := fmt.Sprintf("reopen %s", iss.ID)
	if err := r.Commit(intent); err != nil {
		fatal("commit failed: " + err.Error())
	}

	if hasFlag(os.Args, "--json") {
		printJSON(iss)
	} else {
		fmt.Printf("reopened %s: %s\n", iss.ID, iss.Title)
	}
}

func cmdGraph(args []string) {
	_, store := mustInitialized()

	rootID := ""
	for _, arg := range args {
		if arg != "--json" {
			rootID = arg
			break
		}
	}

	nodes, err := store.Graph(rootID)
	if err != nil {
		fatal(err.Error())
	}

	if hasFlag(args, "--json") {
		printJSON(nodes)
		return
	}

	if len(nodes) == 0 {
		fmt.Println("no issues in graph")
		return
	}

	// Build adjacency for ASCII rendering
	blocked := make(map[string][]string) // blocker -> blocked
	hasParent := make(map[string]bool)
	nodeMap := make(map[string]issue.GraphNode)
	for _, n := range nodes {
		nodeMap[n.ID] = n
		for _, b := range n.Blocks {
			blocked[n.ID] = append(blocked[n.ID], b)
			hasParent[b] = true
		}
	}

	// Find roots (nodes with no incoming edges in this graph)
	var roots []string
	for _, n := range nodes {
		if !hasParent[n.ID] {
			roots = append(roots, n.ID)
		}
	}

	// Render tree
	visited := make(map[string]bool)
	for i, root := range roots {
		last := i == len(roots)-1
		printTree(root, "", last, true, blocked, nodeMap, visited)
	}
}

func printTree(id, prefix string, last bool, isRoot bool, children map[string][]string, nodes map[string]issue.GraphNode, visited map[string]bool) {
	if visited[id] {
		return
	}
	visited[id] = true

	connector := "├── "
	if last {
		connector = "└── "
	}
	if isRoot {
		connector = ""
	}

	n, ok := nodes[id]
	status := ""
	if ok {
		status = fmt.Sprintf(" [%s]", n.Status)
	}
	title := ""
	if ok {
		title = n.Title
	}
	fmt.Printf("%s%s%s%s %s\n", prefix, connector, id, status, title)

	childPrefix := prefix
	if !isRoot {
		if last {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	kids := children[id]
	for i, kid := range kids {
		printTree(kid, childPrefix, i == len(kids)-1, false, children, nodes, visited)
	}
}

func cmdLabel(args []string) {
	r, store := mustInitialized()

	// bw label <id> +bug +frontend -wontfix
	if len(args) < 2 {
		fatal("usage: bw label <id> +label [-label] ...")
	}
	id := args[0]

	var add, remove []string
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "+") {
			add = append(add, strings.TrimPrefix(arg, "+"))
		} else if strings.HasPrefix(arg, "-") {
			remove = append(remove, strings.TrimPrefix(arg, "-"))
		} else {
			// bare label name = add
			add = append(add, arg)
		}
	}

	iss, err := store.Label(id, add, remove)
	if err != nil {
		fatal(err.Error())
	}

	var parts []string
	for _, l := range add {
		parts = append(parts, "+"+l)
	}
	for _, l := range remove {
		parts = append(parts, "-"+l)
	}
	intent := fmt.Sprintf("label %s %s", iss.ID, strings.Join(parts, " "))
	if err := r.Commit(intent); err != nil {
		fatal("commit failed: " + err.Error())
	}

	if hasFlag(os.Args, "--json") {
		printJSON(iss)
	} else {
		fmt.Printf("labeled %s: %s\n", iss.ID, strings.Join(iss.Labels, ", "))
	}
}

func cmdLink(args []string) {
	r, store := mustInitialized()

	// bw link <id1> blocks <id2>
	if len(args) < 3 || args[1] != "blocks" {
		fatal("usage: bw link <id> blocks <id>")
	}
	blockerID := args[0]
	blockedID := args[2]

	if err := store.Link(blockerID, blockedID); err != nil {
		fatal(err.Error())
	}

	// Resolve full IDs for the commit message
	blocker, _ := store.Get(blockerID)
	blocked, _ := store.Get(blockedID)
	intent := fmt.Sprintf("link %s blocks %s", blocker.ID, blocked.ID)
	if err := r.Commit(intent); err != nil {
		fatal("commit failed: " + err.Error())
	}

	fmt.Printf("linked %s blocks %s\n", blocker.ID, blocked.ID)
}

func cmdUnlink(args []string) {
	r, store := mustInitialized()

	// bw unlink <id1> blocks <id2>
	if len(args) < 3 || args[1] != "blocks" {
		fatal("usage: bw unlink <id> blocks <id>")
	}
	blockerID := args[0]
	blockedID := args[2]

	if err := store.Unlink(blockerID, blockedID); err != nil {
		fatal(err.Error())
	}

	blocker, _ := store.Get(blockerID)
	blocked, _ := store.Get(blockedID)
	intent := fmt.Sprintf("unlink %s blocks %s", blocker.ID, blocked.ID)
	if err := r.Commit(intent); err != nil {
		fatal("commit failed: " + err.Error())
	}

	fmt.Printf("unlinked %s blocks %s\n", blocker.ID, blocked.ID)
}

func cmdReady(args []string) {
	_, store := mustInitialized()

	issues, err := store.Ready()
	if err != nil {
		fatal(err.Error())
	}

	if hasFlag(args, "--json") {
		printJSON(issues)
	} else {
		if len(issues) == 0 {
			fmt.Println("no ready issues")
			return
		}
		for _, iss := range issues {
			fmt.Printf("%-14s p%d %-12s %-12s %s\n", iss.ID, iss.Priority, iss.Status, iss.Type, iss.Title)
		}
	}
}

func cmdSync(args []string) {
	r, store := mustInitialized()
	_ = args

	status, intents, err := r.Sync()
	if err != nil {
		fatal(err.Error())
	}

	if status == "needs replay" {
		fmt.Printf("rebase conflict — replaying %d intent(s)...\n", len(intents))
		errs := intent.Replay(r, store, intents)
		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "  warning: %s\n", e)
			}
		}
		if err := r.Push(); err != nil {
			fatal("push after replay failed: " + err.Error())
		}
		fmt.Println("replayed and pushed")
	} else {
		fmt.Println(status)
	}
}

func cmdOnboard() {
	fmt.Print(`# Beadwork — Agent Memory System

Beadwork is a filesystem-native issue tracker that lives on a git orphan branch.
Use it to track tasks, bugs, and epics for the current project.

## Commands

  bw create <title> [flags]      Create an issue
    --priority, -p <1-5>           Priority (1=highest, default 3)
    --type, -t <type>              Type: task, bug, epic (default: task)
    --assignee, -a <name>          Assign to someone
    --description, -d <text>       Description

  bw show <id> [--json]           Show issue details
  bw list [filters] [--json]      List issues
    --status <s>                   Filter: open, in_progress, closed
    --assignee <name>              Filter by assignee
    --priority <n>                 Filter by priority
    --type <type>                  Filter by type
    --label <label>                Filter by label

  bw update <id> [flags]          Update an issue
    --title <text>                 New title
    --description, -d <text>       New description
    --priority, -p <n>             New priority
    --assignee, -a <name>          New assignee
    --type, -t <type>              New type
    --status, -s <status>          New status

  bw close <id> [--reason <r>]    Close an issue
  bw reopen <id>                  Reopen a closed issue
  bw ready [--json]               List issues with no open blockers
  bw link <id> blocks <id>        Create dependency link
  bw unlink <id> blocks <id>      Remove dependency link
  bw graph [<id>] [--json]        Show dependency graph
  bw sync                         Fetch, rebase (or replay), push

## Workflow

1. Run "bw onboard" at the start of each task to refresh your understanding
2. Run "bw list" or "bw ready" to see available work
3. Pick a task, update its status: bw update <id> --status in_progress
4. Do the work
5. Close when done: bw close <id>

## Notes

- Use --json on any command for structured output
- IDs support prefix matching (e.g., "a1b2" matches "canon-a1b2")
- All data lives on the "beadwork" git orphan branch
- Issues are JSON files in .git/beadwork/issues/
`)
}

// --- Helpers ---

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func printJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func printIssue(iss *issue.Issue) {
	fmt.Printf("ID:          %s\n", iss.ID)
	fmt.Printf("Title:       %s\n", iss.Title)
	fmt.Printf("Status:      %s\n", iss.Status)
	fmt.Printf("Priority:    p%d\n", iss.Priority)
	fmt.Printf("Type:        %s\n", iss.Type)
	if iss.Assignee != "" {
		fmt.Printf("Assignee:    %s\n", iss.Assignee)
	}
	if iss.Description != "" {
		fmt.Printf("Description: %s\n", iss.Description)
	}
	if len(iss.Labels) > 0 {
		fmt.Printf("Labels:      %s\n", strings.Join(iss.Labels, ", "))
	}
	if len(iss.Blocks) > 0 {
		fmt.Printf("Blocks:      %s\n", strings.Join(iss.Blocks, ", "))
	}
	if len(iss.BlockedBy) > 0 {
		fmt.Printf("Blocked by:  %s\n", strings.Join(iss.BlockedBy, ", "))
	}
	if iss.Parent != "" {
		fmt.Printf("Parent:      %s\n", iss.Parent)
	}
	fmt.Printf("Created:     %s\n", iss.Created)
}
