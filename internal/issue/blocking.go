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

	// Ancestor check: a descendant cannot block its ancestor.
	if s.isAncestor(blockerID, blockedID) {
		return fmt.Errorf("child cannot block ancestor: %s is a descendant of %s", blockerID, blockedID)
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
	overlay := s.buildSubtreeOverlay()

	var ready []*Issue

	for _, id := range s.IDsWithStatus("open") {
		if overlay.descendants[id] {
			continue
		}
		iss, err := s.readIssue(id)
		if err != nil {
			continue
		}
		if allResolved(s, overlay.effectiveBlockedBy(iss)) {
			ready = append(ready, iss)
		}
	}

	for _, id := range s.IDsWithStatus("deferred") {
		if overlay.descendants[id] {
			continue
		}
		iss, err := s.readIssue(id)
		if err != nil {
			continue
		}
		if IsDeferralExpired(iss.DeferUntil, now) && allResolved(s, overlay.effectiveBlockedBy(iss)) {
			ready = append(ready, iss)
		}
	}

	sortIssues(ready, now)
	return ready, nil
}

// ReadyScoped returns the ready descendants of parentID, excluding the parent
// itself. It uses the same blocked-by and deferred logic as Ready, but scoped
// to the parent's subtree.
func (s *Store) ReadyScoped(parentID string) ([]*Issue, error) {
	parentID, err := s.resolveID(parentID)
	if err != nil {
		return nil, fmt.Errorf("parent: %w", err)
	}

	now := s.Now()

	// Load all non-closed issues and build children map.
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

	// Build the subtree of the parent.
	subtree := buildSubtreeSet(parentID, children)

	var ready []*Issue
	for id := range subtree {
		if id == parentID {
			continue // exclude parent
		}
		iss := issues[id]
		if iss == nil {
			continue
		}

		switch iss.Status {
		case "open":
			if allResolved(s, iss.BlockedBy) {
				ready = append(ready, iss)
			}
		case "deferred":
			if IsDeferralExpired(iss.DeferUntil, now) && allResolved(s, iss.BlockedBy) {
				ready = append(ready, iss)
			}
		}
	}

	sortIssues(ready, now)
	return ready, nil
}

func allResolved(s *Store, blockerIDs []string) bool {
	for _, id := range blockerIDs {
		if !s.IsClosed(id) {
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
	overlay := s.buildSubtreeOverlay()

	var ids []string
	ids = append(ids, s.IDsWithStatus("open")...)
	ids = append(ids, s.IDsWithStatus("in_progress")...)

	var blocked []BlockedIssue
	for _, id := range ids {
		if overlay.descendants[id] {
			continue
		}
		iss, err := s.readIssue(id)
		if err != nil {
			continue
		}
		effective := overlay.effectiveBlockedBy(iss)
		var open []string
		for _, blockerID := range effective {
			if !s.IsClosed(blockerID) {
				open = append(open, blockerID)
			}
		}
		if len(open) > 0 {
			blocked = append(blocked, BlockedIssue{Issue: iss, OpenBlockers: open})
		}
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

// HiddenBlockerSet returns blocker IDs that should be hidden from ready
// display: closed blockers plus internal (within-subtree) blockers.
func (s *Store) HiddenBlockerSet(issues []*Issue) map[string]bool {
	set := s.ClosedBlockerSet(issues)
	overlay := s.buildSubtreeOverlay()
	for id := range overlay.descendants {
		set[id] = true
	}
	return set
}

// subtreeOverlay maps each subtree root to its effective (external-only)
// blockers and tracks all non-root subtree members as descendants.
type subtreeOverlay struct {
	descendants      map[string]bool     // non-root subtree members
	externalBlockers map[string][]string // root ID → external blocker IDs (empty = no external blockers)
}

// effectiveBlockedBy returns the external blockers for a subtree root,
// or the issue's raw BlockedBy for non-root issues.
func (o *subtreeOverlay) effectiveBlockedBy(iss *Issue) []string {
	if ext, ok := o.externalBlockers[iss.ID]; ok {
		return ext
	}
	return iss.BlockedBy
}

// buildSubtreeOverlay builds subtrees from parent fields, collects
// external blockers per root, and marks all descendants.
func (s *Store) buildSubtreeOverlay() *subtreeOverlay {
	overlay := &subtreeOverlay{
		descendants:      make(map[string]bool),
		externalBlockers: make(map[string][]string),
	}

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

	// Find roots and process each subtree
	for parentID := range children {
		iss, ok := issues[parentID]
		if !ok {
			continue
		}
		if iss.Parent != "" && issues[iss.Parent] != nil {
			continue // not a root
		}

		subtree := buildSubtreeSet(parentID, children)

		for id := range subtree {
			if id != parentID {
				overlay.descendants[id] = true
			}
		}

		var ext []string
		for id := range subtree {
			member := issues[id]
			if member == nil {
				continue
			}
			for _, blockerID := range member.BlockedBy {
				if !subtree[blockerID] && !s.IsClosed(blockerID) {
					ext = append(ext, blockerID)
				}
			}
		}
		overlay.externalBlockers[parentID] = uniqueStrings(ext)
	}

	return overlay
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

// isAncestor reports whether candidateAncestor is an ancestor of id
// by walking up the Parent chain.
func (s *Store) isAncestor(id, candidateAncestor string) bool {
	current := id
	for {
		iss, err := s.readIssue(current)
		if err != nil || iss.Parent == "" {
			return false
		}
		if iss.Parent == candidateAncestor {
			return true
		}
		current = iss.Parent
	}
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
