package issue

// DeletePlan describes the side-effects that deleting an issue would have.
type DeletePlan struct {
	Issue     *Issue   `json:"issue"`
	Blocks    []string `json:"blocks"`
	BlockedBy []string `json:"blocked_by"`
	Children  []string `json:"children"`
}

// DeletePreview returns a plan describing what would happen if the issue
// were deleted, without modifying any data.
func (s *Store) DeletePreview(id string) (*DeletePlan, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	iss, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}

	// Find children using snapshot instead of a second List call.
	snap, err := NewSnapshot(s)
	if err != nil {
		return nil, err
	}
	var children []string
	for _, child := range snap.Children(id) {
		children = append(children, child.ID)
	}

	return &DeletePlan{
		Issue:     iss,
		Blocks:    iss.Blocks,
		BlockedBy: iss.BlockedBy,
		Children:  children,
	}, nil
}

// Delete permanently removes an issue, cleaning up all references:
// blocks/blocked_by marker files, related issue JSON, status markers,
// and orphaning any children.
func (s *Store) Delete(id string) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	iss, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}

	// Remove blocks marker files (outgoing: this issue blocks others)
	for _, blocked := range iss.Blocks {
		s.FS.Remove("blocks/" + id + "/" + blocked)
	}
	// Remove the blocks/<id> directory marker
	s.FS.Remove("blocks/" + id + "/.gitkeep")
	s.FS.Remove("blocks/" + id)

	// Remove blocks marker files (incoming: other issues block this one)
	for _, blocker := range iss.BlockedBy {
		s.FS.Remove("blocks/" + blocker + "/" + id)
	}

	// Update related issues: remove deleted ID from their BlockedBy
	for _, blocked := range iss.Blocks {
		if other, err := s.readIssue(blocked); err == nil {
			other.BlockedBy = removeStr(other.BlockedBy, id)
			s.writeIssue(other)
		}
	}

	// Update related issues: remove deleted ID from their Blocks
	for _, blocker := range iss.BlockedBy {
		if other, err := s.readIssue(blocker); err == nil {
			other.Blocks = removeStr(other.Blocks, id)
			s.writeIssue(other)
		}
	}

	// Orphan children: clear Parent field on child issues.
	// Use snapshot for the child lookup to avoid a second List call.
	snap, err := NewSnapshot(s)
	if err != nil {
		return nil, err
	}
	for _, child := range snap.Children(id) {
		child.Parent = ""
		s.writeIssue(child)
	}

	// Remove status marker
	s.FS.Remove("status/" + iss.Status + "/" + id)

	// Remove issue JSON file
	s.FS.Remove("issues/" + id + ".json")

	// Evict from cache and ID set
	delete(s.cache, id)
	s.untrackID(id)

	return iss, nil
}
