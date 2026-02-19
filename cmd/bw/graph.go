package main

import (
	"fmt"

	"github.com/j5n/beadwork/internal/issue"
)

func cmdGraph(args []string) {
	_, store := mustInitialized()

	showAll := hasFlag(args, "--all")
	jsonOut := hasFlag(args, "--json")

	rootID := ""
	for _, arg := range args {
		if arg == "--json" || arg == "--all" {
			continue
		}
		rootID = arg
		break
	}

	if rootID == "" && !showAll {
		fatal("issue ID required (or use --all for all open issues)")
	}

	nodes, err := store.Graph(rootID)
	if err != nil {
		fatal(err.Error())
	}

	// For --all without a root, filter to non-closed only
	if showAll && rootID == "" {
		var filtered []issue.GraphNode
		for _, n := range nodes {
			if n.Status != "closed" {
				filtered = append(filtered, n)
			}
		}
		nodes = filtered
	}

	if jsonOut {
		printJSON(nodes)
		return
	}

	if len(nodes) == 0 {
		fmt.Println("no issues in graph")
		return
	}

	// Build adjacency for tree rendering
	blocked := make(map[string][]string) // blocker -> blocked
	hasParent := make(map[string]bool)
	nodeMap := make(map[string]issue.GraphNode)
	for _, n := range nodes {
		nodeMap[n.ID] = n
		for _, b := range n.Blocks {
			if _, ok := nodeMap[b]; showAll || ok || rootID != "" {
				blocked[n.ID] = append(blocked[n.ID], b)
				hasParent[b] = true
			}
		}
	}

	// Rebuild hasParent after all nodes are in nodeMap
	hasParent = make(map[string]bool)
	for _, n := range nodes {
		for _, b := range n.Blocks {
			if _, ok := nodeMap[b]; ok {
				hasParent[b] = true
			}
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

	connector := "\u251c\u2500\u2500 "
	if last {
		connector = "\u2514\u2500\u2500 "
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
			childPrefix += "\u2502   "
		}
	}

	kids := children[id]
	for i, kid := range kids {
		printTree(kid, childPrefix, i == len(kids)-1, false, children, nodes, visited)
	}
}
