package issue

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jallum/beadwork/internal/treefs"
)

// Status definitions

type StatusInfo struct {
	Name string
	Icon string
}

var Statuses = []StatusInfo{
	{"open", "○"},
	{"in_progress", "◐"},
	{"deferred", "❄"},
	{"closed", "✓"},
}

func StatusNames() []string {
	names := make([]string, len(Statuses))
	for i, s := range Statuses {
		names[i] = s.Name
	}
	return names
}

func StatusIcon(status string) string {
	for _, s := range Statuses {
		if s.Name == status {
			return s.Icon
		}
	}
	return "?"
}

// Priority definitions

var PriorityColors = map[int]string{
	0: "\033[91m", // bright red (P0 — critical)
	1: "\033[31m", // red
	2: "\033[33m", // yellow
	3: "\033[36m", // cyan
	4: "\033[2m",  // dim
}

const ColorReset = "\033[0m"

func PriorityDot(priority int) string {
	color, ok := PriorityColors[priority]
	if !ok {
		color = ""
	}
	return color + "●" + ColorReset
}

type Comment struct {
	Text      string `json:"text"`
	Author    string `json:"author,omitempty"`
	Timestamp string `json:"timestamp"`
}

type Issue struct {
	Assignee    string   `json:"assignee"`
	BlockedBy   []string `json:"blocked_by"`
	Blocks      []string `json:"blocks"`
	ClosedAt    string   `json:"closed_at,omitempty"`
	CloseReason string   `json:"close_reason,omitempty"`
	Created     string   `json:"created"`
	DeferUntil  string   `json:"defer_until,omitempty"`
	Description string   `json:"description"`
	ID          string   `json:"id"`
	Labels      []string `json:"labels"`
	Parent      string    `json:"parent,omitempty"`
	Comments    []Comment `json:"comments,omitempty"`
	Priority    int      `json:"priority"`
	Status      string   `json:"status"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

type Store struct {
	FS              *treefs.TreeFS
	Prefix          string
	DefaultPriority *int
	IDRetries       int    // retries per length before bumping; 0 means 10
	RandReader      io.Reader // random source; nil means crypto/rand.Reader
}

func NewStore(fs *treefs.TreeFS, prefix string) *Store {
	return &Store{FS: fs, Prefix: prefix}
}

func (s *Store) Create(title string, opts CreateOpts) (*Issue, error) {
	// Resolve parent first (needed for dotted child ID generation).
	var parentID string
	if opts.Parent != "" {
		var err error
		parentID, err = s.resolveID(opts.Parent)
		if err != nil {
			return nil, fmt.Errorf("parent: %w", err)
		}
	}

	var id string
	var err error
	if parentID != "" {
		id, err = s.generateChildID(parentID)
	} else {
		id, err = s.generateID()
	}
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	status := "open"
	if opts.DeferUntil != "" {
		status = "deferred"
	}
	issue := &Issue{
		Assignee:    opts.Assignee,
		BlockedBy:   []string{},
		Blocks:      []string{},
		Created:     now,
		DeferUntil:  opts.DeferUntil,
		Description: opts.Description,
		ID:          id,
		Labels:      []string{},
		Parent:      parentID,
		Status:      status,
		Title:       title,
		Type:        opts.Type,
		UpdatedAt:   now,
	}
	if issue.Type == "" {
		issue.Type = "task"
	}
	if opts.Priority != nil {
		issue.Priority = *opts.Priority
	} else if s.DefaultPriority != nil {
		issue.Priority = *s.DefaultPriority
	} else {
		issue.Priority = 2
	}

	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	if err := s.setStatus(id, status); err != nil {
		return nil, err
	}
	return issue, nil
}

// Import writes an issue with a caller-provided ID and fields.
// Used for importing from external sources where the ID is already known.
func (s *Store) Import(iss *Issue) error {
	if err := s.writeIssue(iss); err != nil {
		return err
	}
	return s.setStatus(iss.ID, iss.Status)
}

func (s *Store) Get(id string) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	return s.readIssue(id)
}

func (s *Store) List(filter Filter) ([]*Issue, error) {
	statuses := StatusNames()
	if len(filter.Statuses) > 0 {
		statuses = filter.Statuses
	} else if filter.Status != "" {
		statuses = []string{filter.Status}
	}

	var ids []string
	for _, status := range statuses {
		dir := "status/" + status
		entries, err := s.FS.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.Name() == ".gitkeep" {
				continue
			}
			ids = append(ids, e.Name())
		}
	}

	var issues []*Issue
	for _, id := range ids {
		issue, err := s.readIssue(id)
		if err != nil {
			continue
		}
		if filter.Assignee != "" && issue.Assignee != filter.Assignee {
			continue
		}
		if filter.Priority != nil && issue.Priority != *filter.Priority {
			continue
		}
		if filter.Type != "" && issue.Type != filter.Type {
			continue
		}
		if filter.Label != "" && !containsStr(issue.Labels, filter.Label) {
			continue
		}
		if filter.Grep != "" {
			needle := strings.ToLower(filter.Grep)
			if !strings.Contains(strings.ToLower(issue.Title), needle) &&
				!strings.Contains(strings.ToLower(issue.Description), needle) {
				continue
			}
		}
		issues = append(issues, issue)
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Priority != issues[j].Priority {
			return issues[i].Priority < issues[j].Priority
		}
		return issues[i].Created < issues[j].Created
	})

	return issues, nil
}

func (s *Store) Update(id string, opts UpdateOpts) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	issue, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}

	if opts.Title != nil {
		issue.Title = *opts.Title
	}
	if opts.Description != nil {
		issue.Description = *opts.Description
	}
	if opts.Priority != nil {
		issue.Priority = *opts.Priority
	}
	if opts.Assignee != nil {
		issue.Assignee = *opts.Assignee
	}
	if opts.Type != nil {
		issue.Type = *opts.Type
	}
	if opts.DeferUntil != nil {
		issue.DeferUntil = *opts.DeferUntil
	}
	if opts.Parent != nil {
		if *opts.Parent != "" {
			if *opts.Parent == id {
				return nil, fmt.Errorf("cannot set issue as its own parent")
			}
			parentID, err := s.resolveID(*opts.Parent)
			if err != nil {
				return nil, fmt.Errorf("parent: %w", err)
			}
			// Cycle detection: walk ancestor chain
			current := parentID
			for current != "" {
				if current == id {
					return nil, fmt.Errorf("circular parent reference: %s is an ancestor of %s", id, parentID)
				}
				ancestor, err := s.readIssue(current)
				if err != nil {
					break
				}
				current = ancestor.Parent
			}
			issue.Parent = parentID
		} else {
			issue.Parent = ""
		}
	}
	if opts.Status != nil {
		oldStatus := issue.Status
		newStatus := *opts.Status
		if oldStatus != newStatus {
			if err := s.moveStatus(id, oldStatus, newStatus); err != nil {
				return nil, err
			}
			issue.Status = newStatus
		}
	}

	issue.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	return issue, nil
}

func (s *Store) Close(id, reason string) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	issue, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}
	if issue.Status == "closed" {
		return nil, fmt.Errorf("%s is already closed", id)
	}

	if err := s.moveStatus(id, issue.Status, "closed"); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	issue.Status = "closed"
	issue.ClosedAt = now
	issue.CloseReason = reason
	issue.UpdatedAt = now
	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	return issue, nil
}

func (s *Store) Reopen(id string) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	issue, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}
	if issue.Status != "closed" {
		return nil, fmt.Errorf("%s is not closed", id)
	}

	if err := s.moveStatus(id, "closed", "open"); err != nil {
		return nil, err
	}
	issue.Status = "open"
	issue.ClosedAt = ""
	issue.CloseReason = ""
	issue.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	return issue, nil
}

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
	now := time.Now().UTC().Format(time.RFC3339)
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
	now := time.Now().UTC().Format(time.RFC3339)
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

func (s *Store) Label(id string, add, remove []string) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	issue, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}

	// Add labels
	for _, label := range add {
		s.FS.MkdirAll("labels/" + label)
		if err := s.FS.WriteFile("labels/"+label+"/"+id, []byte{}); err != nil {
			return nil, err
		}
		if !containsStr(issue.Labels, label) {
			issue.Labels = append(issue.Labels, label)
		}
	}

	// Remove labels
	for _, label := range remove {
		s.FS.Remove("labels/" + label + "/" + id)
		// Clean up empty label directory
		entries, _ := s.FS.ReadDir("labels/" + label)
		empty := true
		for _, e := range entries {
			if e.Name() != ".gitkeep" {
				empty = false
				break
			}
		}
		if empty {
			s.FS.Remove("labels/" + label + "/.gitkeep")
		}
		issue.Labels = removeStr(issue.Labels, label)
	}

	sort.Strings(issue.Labels)
	issue.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	return issue, nil
}

func (s *Store) Comment(id, text, author string) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	issue, err := s.readIssue(id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	issue.Comments = append(issue.Comments, Comment{
		Text:      text,
		Author:    author,
		Timestamp: now,
	})
	issue.UpdatedAt = now
	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	return issue, nil
}

type GraphNode struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Blocks    []string `json:"blocks"`
	BlockedBy []string `json:"blocked_by"`
}

func (s *Store) Graph(rootID string) ([]GraphNode, error) {
	if rootID != "" {
		var err error
		rootID, err = s.resolveID(rootID)
		if err != nil {
			return nil, err
		}
	}

	// Read all block relationships
	entries, err := s.FS.ReadDir("blocks")
	if err != nil {
		entries = nil
	}

	// Collect all issue IDs involved in any relationship
	involved := make(map[string]bool)
	edges := make(map[string][]string) // blocker -> []blocked

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
			edges[blockerID] = append(edges[blockerID], blockedID)
			involved[blockerID] = true
			involved[blockedID] = true
		}
	}

	// If root specified, walk only reachable nodes
	if rootID != "" {
		reachable := make(map[string]bool)
		s.walkGraph(rootID, edges, reachable)
		involved = reachable
	}

	// If no relationships exist, show all open issues
	if len(involved) == 0 {
		issues, err := s.List(Filter{})
		if err != nil {
			return nil, err
		}
		var nodes []GraphNode
		for _, iss := range issues {
			if iss.Status == "closed" {
				continue
			}
			nodes = append(nodes, GraphNode{
				ID:        iss.ID,
				Title:     iss.Title,
				Status:    iss.Status,
				Blocks:    iss.Blocks,
				BlockedBy: iss.BlockedBy,
			})
		}
		return nodes, nil
	}

	var nodes []GraphNode
	for id := range involved {
		iss, err := s.readIssue(id)
		if err != nil {
			continue
		}
		nodes = append(nodes, GraphNode{
			ID:        iss.ID,
			Title:     iss.Title,
			Status:    iss.Status,
			Blocks:    iss.Blocks,
			BlockedBy: iss.BlockedBy,
		})
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	return nodes, nil
}

func (s *Store) walkGraph(id string, edges map[string][]string, visited map[string]bool) {
	if visited[id] {
		return
	}
	visited[id] = true
	for _, child := range edges[id] {
		s.walkGraph(child, edges, visited)
	}
}

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

	// Find children by scanning all issues
	var children []string
	all, _ := s.List(Filter{})
	for _, other := range all {
		if other.Parent == id {
			children = append(children, other.ID)
		}
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

	// Orphan children: clear Parent field on child issues
	all, _ := s.List(Filter{})
	for _, other := range all {
		if other.Parent == id {
			other.Parent = ""
			s.writeIssue(other)
		}
	}

	// Remove status marker
	s.FS.Remove("status/" + iss.Status + "/" + id)

	// Remove issue JSON file
	s.FS.Remove("issues/" + id + ".json")

	return iss, nil
}

func (s *Store) Ready() ([]*Issue, error) {
	issues, err := s.List(Filter{})
	if err != nil {
		return nil, err
	}

	var ready []*Issue
	for _, issue := range issues {
		if issue.Status != "open" {
			continue
		}
		if len(issue.BlockedBy) == 0 {
			ready = append(ready, issue)
			continue
		}
		allResolved := true
		for _, blockerID := range issue.BlockedBy {
			blocker, err := s.readIssue(blockerID)
			if err != nil {
				allResolved = false
				break
			}
			if blocker.Status != "closed" {
				allResolved = false
				break
			}
		}
		if allResolved {
			ready = append(ready, issue)
		}
	}
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

// Blocked returns open issues that have at least one open blocker.
func (s *Store) Blocked() ([]BlockedIssue, error) {
	issues, err := s.List(Filter{})
	if err != nil {
		return nil, err
	}

	var blocked []BlockedIssue
	for _, iss := range issues {
		if iss.Status == "closed" {
			continue
		}
		if len(iss.BlockedBy) == 0 {
			continue
		}
		var open []string
		for _, blockerID := range iss.BlockedBy {
			blocker, err := s.readIssue(blockerID)
			if err != nil {
				open = append(open, blockerID)
				continue
			}
			if blocker.Status != "closed" {
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

// --- Internal helpers ---

const base36 = "0123456789abcdefghijklmnopqrstuvwxyz"

func (s *Store) generateID() (string, error) {
	existing := s.ExistingIDs()
	retries := s.IDRetries
	if retries <= 0 {
		retries = 10
	}
	rr := s.RandReader
	if rr == nil {
		rr = rand.Reader
	}
	for length := 3; length <= 8; length++ {
		for attempt := 0; attempt < retries; attempt++ {
			b := make([]byte, length)
			if _, err := io.ReadFull(rr, b); err != nil {
				return "", err
			}
			var suffix strings.Builder
			for _, v := range b {
				suffix.WriteByte(base36[int(v)%36])
			}
			id := s.Prefix + "-" + suffix.String()
			if !existing[id] {
				return id, nil
			}
		}
	}
	return "", fmt.Errorf("failed to generate unique ID after trying lengths 3-8")
}

// generateChildID returns the next sequential child ID for the given parent.
// Format: parentID.N where N is max(existing child numbers) + 1.
func (s *Store) generateChildID(parentID string) (string, error) {
	existing := s.ExistingIDs()
	prefix := parentID + "."
	maxN := 0
	for id := range existing {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		rest := id[len(prefix):]
		n, err := strconv.Atoi(rest)
		if err != nil {
			continue // skip grandchildren (e.g. "parent.1.1")
		}
		if n > maxN {
			maxN = n
		}
	}
	return fmt.Sprintf("%s.%d", parentID, maxN+1), nil
}

func (s *Store) ExistingIDs() map[string]bool {
	ids := make(map[string]bool)
	entries, err := s.FS.ReadDir("issues")
	if err != nil {
		return ids
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".json") {
			ids[strings.TrimSuffix(name, ".json")] = true
		}
	}
	return ids
}

func (s *Store) resolveID(partial string) (string, error) {
	entries, err := s.FS.ReadDir("issues")
	if err != nil {
		return "", fmt.Errorf("cannot read issues: %w", err)
	}
	var matches []string
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".json")
		if name == partial {
			return name, nil
		}
		if strings.HasPrefix(name, partial) || strings.HasSuffix(name, partial) {
			matches = append(matches, name)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous ID %q: matches %s", partial, strings.Join(matches, ", "))
	}
	return "", fmt.Errorf("no issue found matching %q", partial)
}

func (s *Store) readIssue(id string) (*Issue, error) {
	data, err := s.FS.ReadFile("issues/" + id + ".json")
	if err != nil {
		return nil, fmt.Errorf("issue %s not found", id)
	}
	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("corrupt issue %s: %w", id, err)
	}
	return &issue, nil
}

func (s *Store) writeIssue(issue *Issue) error {
	data, err := json.MarshalIndent(issue, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return s.FS.WriteFile("issues/"+issue.ID+".json", data)
}

func (s *Store) setStatus(id, status string) error {
	s.FS.MkdirAll("status/" + status)
	return s.FS.WriteFile("status/"+status+"/"+id, []byte{})
}

func (s *Store) moveStatus(id, from, to string) error {
	s.FS.Remove("status/" + from + "/" + id)
	return s.setStatus(id, to)
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func removeStr(slice []string, s string) []string {
	var result []string
	for _, v := range slice {
		if v != s {
			result = append(result, v)
		}
	}
	if result == nil {
		return []string{}
	}
	return result
}

type CreateOpts struct {
	Parent      string
	Description string
	Priority    *int
	Type        string
	Assignee    string
	DeferUntil  string
}

type UpdateOpts struct {
	Parent      *string
	Title       *string
	Description *string
	Priority    *int
	Assignee    *string
	Type        *string
	Status      *string
	DeferUntil  *string
}

type Filter struct {
	Status   string
	Statuses []string
	Assignee string
	Priority *int
	Type     string
	Label    string
	Grep     string
}
