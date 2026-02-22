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
	openIDs := s.IDsWithStatus("open")
	var ready []*Issue
	for _, id := range openIDs {
		iss, err := s.readIssue(id)
		if err != nil {
			continue
		}
		allResolved := true
		for _, blockerID := range iss.BlockedBy {
			if !s.IsClosed(blockerID) {
				allResolved = false
				break
			}
		}
		if allResolved {
			ready = append(ready, iss)
		}
	}
	sort.Slice(ready, func(i, j int) bool {
		if ready[i].Priority != ready[j].Priority {
			return ready[i].Priority < ready[j].Priority
		}
		return ready[i].Created < ready[j].Created
	})
	return ready, nil
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
