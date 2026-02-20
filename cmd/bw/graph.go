package main

import (
	"fmt"
	"io"

	"github.com/jallum/beadwork/internal/issue"
)

type GraphArgs struct {
	RootID string
	All    bool
	JSON   bool
}

func parseGraphArgs(raw []string) (GraphArgs, error) {
	a := ParseArgs(raw)
	ga := GraphArgs{
		RootID: a.PosFirst(),
		All:    a.Bool("--all"),
		JSON:   a.JSON(),
	}
	if ga.RootID == "" && !ga.All {
		return ga, fmt.Errorf("issue ID required (or use --all for all open issues)")
	}
	return ga, nil
}

func cmdGraph(args []string, w io.Writer) error {
	ga, err := parseGraphArgs(args)
	if err != nil {
		return err
	}

	_, store, err := getInitialized()
	if err != nil {
		return err
	}

	showAll := ga.All
	rootID := ga.RootID

	nodes, err := store.Graph(rootID)
	if err != nil {
		return err
	}

	if ga.JSON {
		fprintJSON(w, nodes)
		return nil
	}

	if len(nodes) == 0 {
		fmt.Fprintln(w, "no issues in graph")
		return nil
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
		fprintTree(w, root, "", last, true, blocked, nodeMap, visited)
	}
	return nil
}

func fprintTree(w io.Writer, id, prefix string, last bool, isRoot bool, children map[string][]string, nodes map[string]issue.GraphNode, visited map[string]bool) {
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
	fmt.Fprintf(w, "%s%s%s%s %s\n", prefix, connector, id, status, title)

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
		fprintTree(w, kid, childPrefix, i == len(kids)-1, false, children, nodes, visited)
	}
}
