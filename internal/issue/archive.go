package issue

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ArchiveOpts controls how Archive treats an issue.
type ArchiveOpts struct {
	// Close closes the issue first if it is not already closed. Without it,
	// archiving a non-closed issue fails with a *NotClosedError.
	Close bool
	// Detach severs live graph edges to OPEN dependents/children instead of
	// refusing. Without it, archiving an issue that open work still depends on
	// fails with an *ArchiveBlockedError.
	Detach bool
}

// NotClosedError reports that an issue cannot be archived because it is not
// closed (and Close was not requested).
type NotClosedError struct {
	ID     string
	Status string
}

func (e *NotClosedError) Error() string {
	return fmt.Sprintf("%s is %s, not closed", e.ID, e.Status)
}

// ArchiveBlockedError reports that archiving an issue would orphan open work:
// open issues it still blocks, or open children. Pass ArchiveOpts.Detach to
// sever those edges and proceed.
type ArchiveBlockedError struct {
	ID             string
	OpenDependents []string // open issues this one blocks
	OpenChildren   []string // open child issues
}

func (e *ArchiveBlockedError) Error() string {
	var parts []string
	if len(e.OpenDependents) > 0 {
		parts = append(parts, "open dependents: "+strings.Join(e.OpenDependents, ", "))
	}
	if len(e.OpenChildren) > 0 {
		parts = append(parts, "open children: "+strings.Join(e.OpenChildren, ", "))
	}
	return fmt.Sprintf("%s still has live work attached (%s); pass --detach to sever and archive anyway",
		e.ID, strings.Join(parts, "; "))
}

// archiveAttachments returns the open dependents (issues iss blocks that are
// not closed) and open children of iss, sorted. These are the live edges that
// archiving without --detach would orphan.
func (s *Store) archiveAttachments(iss *Issue) (openDeps, openKids []string) {
	for _, blockedID := range iss.Blocks {
		if !s.IsClosed(blockedID) {
			openDeps = append(openDeps, blockedID)
		}
	}
	if children, err := s.Children(iss.ID); err == nil {
		for _, c := range children {
			if c.Status != "closed" {
				openKids = append(openKids, c.ID)
			}
		}
	}
	sort.Strings(openDeps)
	sort.Strings(openKids)
	return openDeps, openKids
}

// ArchiveCheck reports the open dependents and open children that a no-detach
// archive of id would orphan. Empty results mean the issue can be archived
// cleanly. It does not mutate anything.
func (s *Store) ArchiveCheck(id string) (openDeps, openKids []string, err error) {
	id, err = s.resolveID(id)
	if err != nil {
		return nil, nil, err
	}
	iss, err := s.readIssue(id)
	if err != nil {
		return nil, nil, err
	}
	openDeps, openKids = s.archiveAttachments(iss)
	return openDeps, openKids, nil
}

// Archive moves a closed issue out of the live tree into archive/<id>.json,
// detaching it from the dependency graph. Archived issues leave the live ID
// space and the status/labels/blocks indexes, so they no longer surface in
// ready, blocked, list, or ID resolution. The move is one-way; recovery is via
// git history or `bw import`.
//
// Eligibility: the issue must be closed. ArchiveOpts.Close closes it first if
// it is still open. If open issues still depend on it (it blocks them, or they
// are open children), Archive refuses with an *ArchiveBlockedError unless
// ArchiveOpts.Detach is set, in which case those edges are severed (mirroring
// Delete: dependents lose the blocker from BlockedBy, open children are
// orphaned).
func (s *Store) Archive(id string, opts ArchiveOpts) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	iss, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}

	now := s.nowRFC3339()

	// Eligibility: must be closed (or closable via opts.Close).
	if iss.Status != "closed" {
		if !opts.Close {
			return nil, &NotClosedError{ID: id, Status: iss.Status}
		}
		original := iss.Status
		if err := s.moveStatus(id, original, "closed"); err != nil {
			return nil, err
		}
		iss.Status = "closed"
		iss.ClosedAt = now
	}

	// Graph guard: open dependents (issues this one blocks) and open children.
	openDeps, openKids := s.archiveAttachments(iss)
	children, _ := s.Children(id)
	if (len(openDeps) > 0 || len(openKids) > 0) && !opts.Detach {
		return nil, &ArchiveBlockedError{ID: id, OpenDependents: openDeps, OpenChildren: openKids}
	}

	// Sever this issue's outgoing block edges.
	for _, blockedID := range iss.Blocks {
		s.FS.Remove("blocks/" + id + "/" + blockedID)
		if other, err := s.readIssue(blockedID); err == nil {
			other.BlockedBy = removeStr(other.BlockedBy, id)
			other.UpdatedAt = now
			s.writeIssue(other)
		}
	}
	s.FS.Remove("blocks/" + id + "/.gitkeep")
	s.FS.Remove("blocks/" + id)

	// Sever incoming block edges.
	for _, blockerID := range iss.BlockedBy {
		s.FS.Remove("blocks/" + blockerID + "/" + id)
		if other, err := s.readIssue(blockerID); err == nil {
			other.Blocks = removeStr(other.Blocks, id)
			other.UpdatedAt = now
			s.writeIssue(other)
		}
	}

	// Orphan children (clear their Parent pointer to the archived issue).
	for _, c := range children {
		c.Parent = ""
		c.UpdatedAt = now
		s.writeIssue(c)
	}

	// Remove label markers.
	for _, label := range iss.Labels {
		s.FS.Remove("labels/" + label + "/" + id)
	}

	// Remove the status marker.
	s.FS.Remove("status/" + iss.Status + "/" + id)

	// Detach the archived record's own edge arrays; they are no longer live.
	iss.Blocks = []string{}
	iss.BlockedBy = []string{}
	iss.ArchivedAt = now
	iss.UpdatedAt = now

	// Write to the archive tree and remove the live issue JSON.
	data, err := json.MarshalIndent(iss, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if err := s.FS.WriteFile("archive/"+id+".json", data); err != nil {
		return nil, err
	}
	s.FS.Remove("issues/" + id + ".json")

	// Evict from caches and the live ID space.
	delete(s.cache, id)
	s.untrackID(id)

	return iss, nil
}

// ArchivedIssue reads an archived issue by its exact ID from archive/<id>.json.
// Archived issues are intentionally outside the live ID space, so this does not
// go through ID resolution. Returns an error if the ID is not archived.
func (s *Store) ArchivedIssue(id string) (*Issue, error) {
	data, err := s.FS.ReadFile("archive/" + id + ".json")
	if err != nil {
		return nil, fmt.Errorf("archived issue %s not found", id)
	}
	var iss Issue
	if err := json.Unmarshal(data, &iss); err != nil {
		return nil, fmt.Errorf("corrupt archived issue %s: %w", id, err)
	}
	return &iss, nil
}

// ClosedBefore returns closed issues whose closed_at timestamp precedes cutoff,
// sorted like other listings. Issues without a parseable closed_at are skipped.
// Only closed issues are considered — open work is never selected by date.
func (s *Store) ClosedBefore(cutoff time.Time) ([]*Issue, error) {
	var result []*Issue
	for _, id := range s.IDsWithStatus("closed") {
		iss, err := s.readIssue(id)
		if err != nil {
			continue
		}
		if iss.ClosedAt == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, iss.ClosedAt)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			result = append(result, iss)
		}
	}
	sortIssues(result, s.Now())
	return result, nil
}
