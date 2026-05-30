package issue

import "fmt"

// SubtreeCloseResult reports the outcome of a recursive close: the issues
// newly closed (leaf-up order), those skipped because they were already
// closed, and the external issues that became unblocked as a result.
type SubtreeCloseResult struct {
	Closed    []*Issue `json:"closed"`
	Skipped   []*Issue `json:"skipped"`
	Unblocked []*Issue `json:"unblocked"`
}

// CloseSubtree closes id and every descendant under it. Members that are
// already closed are skipped rather than erroring, and an already-closed
// root is tolerated so a re-run mops up any stragglers. The root records the
// given reason verbatim; each descendant records an auto-generated reason
// noting the cascade. Issues outside the subtree that become fully unblocked
// are returned, deduplicated, in close order.
//
// Traversal is built from all issues (including closed ones) so the walk can
// pass through an already-closed node to reach still-open descendants beneath
// it. Closing proceeds leaf-up: a parent is closed only after its children.
func (s *Store) CloseSubtree(id, reason string) (*SubtreeCloseResult, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}

	all, err := s.List(Filter{})
	if err != nil {
		return nil, err
	}
	childrenOf := make(map[string][]string)
	for _, iss := range all {
		if iss.Parent != "" {
			childrenOf[iss.Parent] = append(childrenOf[iss.Parent], iss.ID)
		}
	}

	subtree := buildSubtreeSet(id, childrenOf)
	ordered := subtreePostOrder(id, childrenOf)

	result := &SubtreeCloseResult{}
	for _, memberID := range ordered {
		iss, err := s.readIssue(memberID)
		if err != nil {
			continue
		}
		if iss.Status == "closed" {
			result.Skipped = append(result.Skipped, iss)
			continue
		}

		memberReason := reason
		if memberID != id {
			memberReason = fmt.Sprintf("closed with parent %s", id)
		}
		if err := s.moveStatus(memberID, iss.Status, "closed"); err != nil {
			return nil, err
		}
		now := s.nowRFC3339()
		iss.Status = "closed"
		iss.ClosedAt = now
		iss.CloseReason = memberReason
		iss.UpdatedAt = now
		if err := s.writeIssue(iss); err != nil {
			return nil, err
		}
		result.Closed = append(result.Closed, iss)
	}

	// With the whole subtree closed, surface only external unblocks: issues
	// outside the subtree whose blockers are now all closed. Internal
	// unblocks are noise since those issues are closed too.
	seen := make(map[string]bool)
	for _, closed := range result.Closed {
		unblocked, err := s.NewlyUnblocked(closed.ID)
		if err != nil {
			return nil, err
		}
		for _, u := range unblocked {
			if subtree[u.ID] || seen[u.ID] {
				continue
			}
			seen[u.ID] = true
			result.Unblocked = append(result.Unblocked, u)
		}
	}

	return result, nil
}

// subtreePostOrder returns the subtree rooted at rootID in post-order, so
// every node appears after all of its descendants (root last).
func subtreePostOrder(rootID string, children map[string][]string) []string {
	var out []string
	var walk func(id string)
	walk = func(id string) {
		for _, c := range children[id] {
			walk(c)
		}
		out = append(out, id)
	}
	walk(rootID)
	return out
}
