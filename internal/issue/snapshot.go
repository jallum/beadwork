package issue

// Snapshot is a read-only, point-in-time view of all issues and edges.
// It loads everything once (List + LoadEdges) so callers can run
// multiple graph queries without repeated filesystem reads.
type Snapshot struct {
	Issues  map[string]*Issue
	Forward map[string][]string // blocker → []blocked
	Reverse map[string][]string // blocked → []blocker
	Closed  map[string]bool
	IDs     map[string]bool
}

// NewSnapshot loads all issues and edges from the store, returning
// a frozen view suitable for read-only graph queries.
func NewSnapshot(s *Store) (*Snapshot, error) {
	all, err := s.List(Filter{})
	if err != nil {
		return nil, err
	}

	fwd, rev := s.LoadEdges()

	issues := make(map[string]*Issue, len(all))
	closed := make(map[string]bool)
	ids := make(map[string]bool, len(all))
	for _, iss := range all {
		issues[iss.ID] = iss
		ids[iss.ID] = true
		if iss.Status == "closed" {
			closed[iss.ID] = true
		}
	}

	return &Snapshot{
		Issues:  issues,
		Forward: fwd,
		Reverse: rev,
		Closed:  closed,
		IDs:     ids,
	}, nil
}

// Tips walks from roots following edges, returning the leaf issues --
// nodes with no further outgoing edges in the map. Closed intermediary
// nodes are walked through but not returned. Unlike Store.Tips, this
// operates entirely on snapshot data with no filesystem reads.
func (snap *Snapshot) Tips(roots []string, edges map[string][]string) []*Issue {
	if len(roots) == 0 {
		return nil
	}

	visited := make(map[string]bool)
	var tips []*Issue

	var walk func(id string)
	walk = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true

		children := edges[id]
		if len(children) == 0 {
			if iss, ok := snap.Issues[id]; ok {
				tips = append(tips, iss)
			}
			return
		}

		for _, child := range children {
			walk(child)
		}
	}

	for _, root := range roots {
		walk(root)
	}

	return tips
}

// Ready returns open issues whose blockers are all closed.
func (snap *Snapshot) Ready() []*Issue {
	var ready []*Issue
	for _, iss := range snap.Issues {
		if iss.Status != "open" {
			continue
		}
		blockers := snap.Reverse[iss.ID]
		if len(blockers) == 0 {
			ready = append(ready, iss)
			continue
		}
		allResolved := true
		for _, blockerID := range blockers {
			if !snap.Closed[blockerID] {
				allResolved = false
				break
			}
		}
		if allResolved {
			ready = append(ready, iss)
		}
	}
	return ready
}

// Blocked returns open issues that have at least one open blocker.
func (snap *Snapshot) Blocked() []BlockedIssue {
	var blocked []BlockedIssue
	for _, iss := range snap.Issues {
		if iss.Status == "closed" {
			continue
		}
		if len(iss.BlockedBy) == 0 {
			continue
		}
		var open []string
		for _, blockerID := range iss.BlockedBy {
			if !snap.Closed[blockerID] {
				open = append(open, blockerID)
			}
		}
		if len(open) > 0 {
			blocked = append(blocked, BlockedIssue{Issue: iss, OpenBlockers: open})
		}
	}
	return blocked
}

// Children returns issues whose Parent field matches parentID.
func (snap *Snapshot) Children(parentID string) []*Issue {
	var children []*Issue
	for _, iss := range snap.Issues {
		if iss.Parent == parentID {
			children = append(children, iss)
		}
	}
	return children
}
