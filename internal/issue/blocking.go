package issue

import (
	"fmt"
	"sort"
)

func (s *Store) Link(blockerID, blockedID string) error {
	blockerID, err := s.resolveID(blockerID)
	if err != nil {
		return fmt.Errorf("blocker: %w", err)
	}
	blockedID, err = s.resolveID(blockedID)
	if err != nil {
		return fmt.Errorf("blocked: %w", err)
	}
	if blockerID == blockedID {
		return fmt.Errorf("an issue cannot block itself")
	}

	// Cycle detection: walk from blockedID following Blocks edges.
	// If blockerID is reachable, adding this edge would create a cycle.
	if s.wouldCycle(blockedID, blockerID) {
		return fmt.Errorf("circular dependency: %s is already transitively blocked by %s", blockerID, blockedID)
	}

	// Create marker file: blocks/<blocker>/<blocked>
	s.FS.MkdirAll("blocks/" + blockerID)
	if err := s.FS.WriteFile("blocks/"+blockerID+"/"+blockedID, []byte{}); err != nil {
		return err
	}

	// Update blocker's JSON
	now := s.nowRFC3339()
	blocker, err := s.readIssue(blockerID)
	if err != nil {
		return err
	}
	if !containsStr(blocker.Blocks, blockedID) {
		blocker.Blocks = append(blocker.Blocks, blockedID)
		sort.Strings(blocker.Blocks)
	}
	blocker.UpdatedAt = now
	if err := s.writeIssue(blocker); err != nil {
		return err
	}

	// Update blocked's JSON
	blocked, err := s.readIssue(blockedID)
	if err != nil {
		return err
	}
	if !containsStr(blocked.BlockedBy, blockerID) {
		blocked.BlockedBy = append(blocked.BlockedBy, blockerID)
		sort.Strings(blocked.BlockedBy)
	}
	blocked.UpdatedAt = now
	if err := s.writeIssue(blocked); err != nil {
		return err
	}

	return nil
}

func (s *Store) Unlink(blockerID, blockedID string) error {
	blockerID, err := s.resolveID(blockerID)
	if err != nil {
		return fmt.Errorf("blocker: %w", err)
	}
	blockedID, err = s.resolveID(blockedID)
	if err != nil {
		return fmt.Errorf("blocked: %w", err)
	}

	// Remove marker file
	s.FS.Remove("blocks/" + blockerID + "/" + blockedID)

	// Check if directory is now empty (no non-.gitkeep files)
	entries, _ := s.FS.ReadDir("blocks/" + blockerID)
	empty := true
	for _, e := range entries {
		if e.Name() != ".gitkeep" {
			empty = false
			break
		}
	}
	if empty {
		s.FS.Remove("blocks/" + blockerID + "/.gitkeep")
	}

	// Update blocker's JSON
	now := s.nowRFC3339()
	blocker, err := s.readIssue(blockerID)
	if err != nil {
		return err
	}
	blocker.Blocks = removeStr(blocker.Blocks, blockedID)
	blocker.UpdatedAt = now
	if err := s.writeIssue(blocker); err != nil {
		return err
	}

	// Update blocked's JSON
	blocked, err := s.readIssue(blockedID)
	if err != nil {
		return err
	}
	blocked.BlockedBy = removeStr(blocked.BlockedBy, blockerID)
	blocked.UpdatedAt = now
	if err := s.writeIssue(blocked); err != nil {
		return err
	}

	return nil
}

// DepExists reports whether blockerID currently blocks blockedID.
func (s *Store) DepExists(blockerID, blockedID string) bool {
	blockerID, err := s.resolveID(blockerID)
	if err != nil {
		return false
	}
	blockedID, err = s.resolveID(blockedID)
	if err != nil {
		return false
	}
	_, err = s.FS.Stat("blocks/" + blockerID + "/" + blockedID)
	return err == nil
}

// LoadEdges reads the blocks/ directory and returns forward and reverse
// adjacency maps. Forward maps blocker → []blocked; reverse maps
// blocked → []blocker.
func (s *Store) LoadEdges() (forward, reverse map[string][]string) {
	forward = make(map[string][]string)
	reverse = make(map[string][]string)

	entries, err := s.FS.ReadDir("blocks")
	if err != nil {
		return forward, reverse
	}

	for _, e := range entries {
		if !e.IsDir() || e.Name() == ".gitkeep" {
			continue
		}
		blockerID := e.Name()
		children, err := s.FS.ReadDir("blocks/" + blockerID)
		if err != nil {
			continue
		}
		for _, c := range children {
			if c.Name() == ".gitkeep" {
				continue
			}
			blockedID := c.Name()
			forward[blockerID] = append(forward[blockerID], blockedID)
			reverse[blockedID] = append(reverse[blockedID], blockerID)
		}
	}
	return forward, reverse
}

// Tips walks from roots following edges, returning the leaf issues —
// open nodes with no further outgoing edges in the map. Closed
// intermediary nodes are walked through but not returned. This is the
// shared primitive for surfacing actionable work in show and ready.
func (s *Store) Tips(roots []string, edges map[string][]string) ([]*Issue, error) {
	if len(roots) == 0 {
		return nil, nil
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
			// Leaf node — this is a tip
			iss, err := s.readIssue(id)
			if err != nil {
				return
			}
			tips = append(tips, iss)
			return
		}

		// Has children — walk deeper
		for _, child := range children {
			walk(child)
		}
	}

	for _, root := range roots {
		walk(root)
	}

	return tips, nil
}

func (s *Store) Ready() ([]*Issue, error) {
	now := s.Now()

	// Open issues with all blockers resolved.
	openIDs := s.IDsWithStatus("open")
	var ready []*Issue
	for _, id := range openIDs {
		iss, err := s.readIssue(id)
		if err != nil {
			continue
		}
		if allBlockersResolved(s, iss) {
			ready = append(ready, iss)
		}
	}

	// Deferred issues whose deferral has expired, with all blockers resolved.
	deferredIDs := s.IDsWithStatus("deferred")
	for _, id := range deferredIDs {
		iss, err := s.readIssue(id)
		if err != nil {
			continue
		}
		if IsDeferralExpired(iss.DeferUntil, now) && allBlockersResolved(s, iss) {
			ready = append(ready, iss)
		}
	}

	// Subtree-aware reclassification
	if adj := s.analyzeSubtrees(); adj != nil {
		readySet := make(map[string]bool, len(ready))
		var filtered []*Issue
		for _, iss := range ready {
			readySet[iss.ID] = true
			if adj.blockedRoots[iss.ID] != nil {
				continue // root has external blockers → not ready
			}
			filtered = append(filtered, iss)
		}
		// Add subtree roots that are only internally blocked
		for rootID := range adj.readyRoots {
			if readySet[rootID] {
				continue // already in ready
			}
			iss, err := s.readIssue(rootID)
			if err != nil || iss.Status != "open" {
				continue
			}
			filtered = append(filtered, iss)
		}
		ready = filtered
	}

	sortIssues(ready, now)
	return ready, nil
}

func allBlockersResolved(s *Store, iss *Issue) bool {
	for _, blockerID := range iss.BlockedBy {
		if !s.IsClosed(blockerID) {
			return false
		}
	}
	return true
}

// BlockedIssue pairs an issue with the IDs of its open (unresolved) blockers.
type BlockedIssue struct {
	*Issue
	OpenBlockers []string `json:"open_blockers"`
}

// CloseResult pairs a closed issue with any issues that became unblocked.
type CloseResult struct {
	Issue     *Issue   `json:"issue"`
	Unblocked []*Issue `json:"unblocked"`
}

// Blocked returns non-closed issues that have at least one open blocker.
func (s *Store) Blocked() ([]BlockedIssue, error) {
	var ids []string
	ids = append(ids, s.IDsWithStatus("open")...)
	ids = append(ids, s.IDsWithStatus("in_progress")...)

	var blocked []BlockedIssue
	for _, id := range ids {
		iss, err := s.readIssue(id)
		if err != nil {
			continue
		}
		if len(iss.BlockedBy) == 0 {
			continue
		}
		var open []string
		for _, blockerID := range iss.BlockedBy {
			if !s.IsClosed(blockerID) {
				open = append(open, blockerID)
			}
		}
		if len(open) > 0 {
			blocked = append(blocked, BlockedIssue{Issue: iss, OpenBlockers: open})
		}
	}

	// Subtree-aware reclassification
	if adj := s.analyzeSubtrees(); adj != nil {
		var filtered []BlockedIssue
		for _, bi := range blocked {
			if adj.descendants[bi.ID] {
				continue // suppress descendants
			}
			if adj.readyRoots[bi.ID] {
				continue // only internal blockers → ready, not blocked
			}
			if ext, ok := adj.blockedRoots[bi.ID]; ok {
				bi.OpenBlockers = uniqueStrings(append(bi.OpenBlockers, ext...))
			}
			filtered = append(filtered, bi)
		}
		// Add subtree roots with external blockers not already in blocked
		blockedSet := make(map[string]bool, len(filtered))
		for _, bi := range filtered {
			blockedSet[bi.ID] = true
		}
		for rootID, ext := range adj.blockedRoots {
			if blockedSet[rootID] {
				continue
			}
			iss, err := s.readIssue(rootID)
			if err != nil || iss.Status == "closed" {
				continue
			}
			filtered = append(filtered, BlockedIssue{Issue: iss, OpenBlockers: ext})
		}
		blocked = filtered
	}

	return blocked, nil
}

// NewlyUnblocked returns open issues from the given issue's Blocks list
// whose blockers are now all closed. Call after closing an issue to discover
// which downstream issues became actionable.
func (s *Store) NewlyUnblocked(id string) ([]*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	blocker, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}
	if len(blocker.Blocks) == 0 {
		return nil, nil
	}

	var unblocked []*Issue
	for _, blockedID := range blocker.Blocks {
		iss, err := s.readIssue(blockedID)
		if err != nil {
			continue
		}
		if iss.Status == "closed" {
			continue
		}
		allResolved := true
		for _, depID := range iss.BlockedBy {
			dep, err := s.readIssue(depID)
			if err != nil {
				allResolved = false
				break
			}
			if dep.Status != "closed" {
				allResolved = false
				break
			}
		}
		if allResolved {
			unblocked = append(unblocked, iss)
		}
	}
	return unblocked, nil
}

// ClosedBlockerSet returns the set of blocker IDs that are closed,
// collected from the BlockedBy lists of the given issues.
func (s *Store) ClosedBlockerSet(issues []*Issue) map[string]bool {
	set := make(map[string]bool)
	for _, iss := range issues {
		for _, bid := range iss.BlockedBy {
			if dep, err := s.Get(bid); err == nil && dep.Status == "closed" {
				set[bid] = true
			}
		}
	}
	return set
}

// subtreeAdjustment holds the result of analyzing parent-child subtrees
// for blocked/ready reclassification.
type subtreeAdjustment struct {
	blockedRoots map[string][]string // root ID → external blocker IDs
	readyRoots   map[string]bool     // roots with no external blockers
	descendants  map[string]bool     // all non-root subtree members
}

// analyzeSubtrees builds subtrees from parent fields and classifies each
// root as ready (no external blockers) or blocked (has external blockers).
// Returns nil when no subtrees exist.
func (s *Store) analyzeSubtrees() *subtreeAdjustment {
	// Load all non-closed issues
	var allIDs []string
	for _, status := range []string{"open", "in_progress", "in_review", "deferred"} {
		allIDs = append(allIDs, s.IDsWithStatus(status)...)
	}

	issues := make(map[string]*Issue, len(allIDs))
	children := make(map[string][]string)
	for _, id := range allIDs {
		iss, err := s.readIssue(id)
		if err != nil {
			continue
		}
		issues[id] = iss
		if iss.Parent != "" {
			children[iss.Parent] = append(children[iss.Parent], iss.ID)
		}
	}

	// Find roots: issues with children whose parent is absent or not in the set
	var roots []string
	for parentID := range children {
		iss, ok := issues[parentID]
		if !ok {
			continue
		}
		if iss.Parent == "" || issues[iss.Parent] == nil {
			roots = append(roots, parentID)
		}
	}
	if len(roots) == 0 {
		return nil
	}

	adj := &subtreeAdjustment{
		blockedRoots: make(map[string][]string),
		readyRoots:   make(map[string]bool),
		descendants:  make(map[string]bool),
	}

	for _, rootID := range roots {
		subtree := buildSubtreeSet(rootID, children)

		for id := range subtree {
			if id != rootID {
				adj.descendants[id] = true
			}
		}

		var ext []string
		for id := range subtree {
			iss := issues[id]
			if iss == nil {
				continue
			}
			for _, blockerID := range iss.BlockedBy {
				if !subtree[blockerID] && !s.IsClosed(blockerID) {
					ext = append(ext, blockerID)
				}
			}
		}
		ext = uniqueStrings(ext)

		if len(ext) > 0 {
			adj.blockedRoots[rootID] = ext
		} else {
			adj.readyRoots[rootID] = true
		}
	}

	return adj
}

func buildSubtreeSet(rootID string, children map[string][]string) map[string]bool {
	set := map[string]bool{rootID: true}
	for _, childID := range children[rootID] {
		for id := range buildSubtreeSet(childID, children) {
			set[id] = true
		}
	}
	return set
}

func uniqueStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	var result []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// wouldCycle reports whether target is reachable from start by following
// Blocks edges. If so, adding an edge from target→start would create a
// cycle in the dependency graph.
func (s *Store) wouldCycle(start, target string) bool {
	visited := make(map[string]bool)
	var walk func(id string) bool
	walk = func(id string) bool {
		if id == target {
			return true
		}
		if visited[id] {
			return false
		}
		visited[id] = true
		iss, err := s.readIssue(id)
		if err != nil {
			return false
		}
		for _, next := range iss.Blocks {
			if walk(next) {
				return true
			}
		}
		return false
	}
	return walk(start)
}
