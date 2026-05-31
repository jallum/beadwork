package issue

import "sort"

// SubtreeDeleteResult reports the outcome of a recursive delete: the issues
// removed (leaf-up order, descendants before parents) and the external issues
// that became fully unblocked once the subtree was gone.
type SubtreeDeleteResult struct {
	Deleted   []*Issue `json:"deleted"`
	Unblocked []*Issue `json:"unblocked"`
}

// subtreeDeletePlan computes, without mutating anything, the leaf-up delete
// order for the subtree rooted at id and the external dependents — issues
// outside the subtree that a subtree member blocks. Those externals are the
// candidates that may become unblocked once the subtree is removed.
//
// Like CloseSubtree, the traversal is built from all issues (including closed
// ones) so it can pass through a closed node to reach descendants beneath it.
func (s *Store) subtreeDeletePlan(id string) (ordered, extDeps []string, err error) {
	id, err = s.resolveID(id)
	if err != nil {
		return nil, nil, err
	}

	all, err := s.List(Filter{})
	if err != nil {
		return nil, nil, err
	}
	childrenOf := make(map[string][]string)
	for _, iss := range all {
		if iss.Parent != "" {
			childrenOf[iss.Parent] = append(childrenOf[iss.Parent], iss.ID)
		}
	}

	subtree := buildSubtreeSet(id, childrenOf)
	ordered = subtreePostOrder(id, childrenOf)

	seen := make(map[string]bool)
	for _, memberID := range ordered {
		iss, err := s.readIssue(memberID)
		if err != nil {
			continue
		}
		for _, blockedID := range iss.Blocks {
			if subtree[blockedID] || seen[blockedID] {
				continue
			}
			seen[blockedID] = true
			extDeps = append(extDeps, blockedID)
		}
	}
	sort.Strings(extDeps)
	return ordered, extDeps, nil
}

// DeleteSubtreePreview returns the subtree members that a recursive delete of
// id would remove (leaf-up order) and the external dependents that would become
// fully unblocked. It does not mutate anything.
func (s *Store) DeleteSubtreePreview(id string) (*SubtreeDeleteResult, error) {
	ordered, extDeps, err := s.subtreeDeletePlan(id)
	if err != nil {
		return nil, err
	}

	result := &SubtreeDeleteResult{}
	for _, memberID := range ordered {
		if iss, err := s.readIssue(memberID); err == nil {
			result.Deleted = append(result.Deleted, iss)
		}
	}
	// Predict which external dependents lose their last open blocker. The
	// subtree members are still live here, so exclude them when judging
	// whether the dependent's remaining blockers are all resolved.
	subtree := make(map[string]bool, len(ordered))
	for _, m := range ordered {
		subtree[m] = true
	}
	for _, depID := range extDeps {
		iss, err := s.readIssue(depID)
		if err != nil || iss.Status == "closed" {
			continue
		}
		if remainingBlockersResolved(s, iss.BlockedBy, subtree) {
			result.Unblocked = append(result.Unblocked, iss)
		}
	}
	return result, nil
}

// DeleteSubtree deletes id and every descendant under it, leaf-up (a parent is
// removed only after its children). Edges to issues outside the subtree are
// severed (mirroring Delete: dependents lose the blocker, external children are
// orphaned). External issues that become fully unblocked are returned,
// deduplicated, in delete order. The removal is permanent.
func (s *Store) DeleteSubtree(id string) (*SubtreeDeleteResult, error) {
	ordered, extDeps, err := s.subtreeDeletePlan(id)
	if err != nil {
		return nil, err
	}

	result := &SubtreeDeleteResult{}
	for _, memberID := range ordered {
		iss, err := s.Delete(memberID)
		if err != nil {
			return nil, err
		}
		result.Deleted = append(result.Deleted, iss)
	}

	// With the subtree gone and its edges severed, surface external dependents
	// whose remaining blockers are now all closed.
	for _, depID := range extDeps {
		iss, err := s.readIssue(depID)
		if err != nil || iss.Status == "closed" {
			continue
		}
		if allResolved(s, iss.BlockedBy) {
			result.Unblocked = append(result.Unblocked, iss)
		}
	}
	return result, nil
}

// remainingBlockersResolved reports whether every blocker outside the excluded
// set is closed. Used to predict unblocking while the excluded (about-to-be-
// deleted) issues are still live.
func remainingBlockersResolved(s *Store, blockerIDs []string, excluded map[string]bool) bool {
	for _, id := range blockerIDs {
		if excluded[id] {
			continue
		}
		if !s.IsClosed(id) {
			return false
		}
	}
	return true
}
