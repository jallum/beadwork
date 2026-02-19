package issue

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Issue struct {
	Assignee    string   `json:"assignee"`
	BlockedBy   []string `json:"blocked_by"`
	Blocks      []string `json:"blocks"`
	Created     string   `json:"created"`
	Description string   `json:"description"`
	ID          string   `json:"id"`
	Labels      []string `json:"labels"`
	Parent      string   `json:"parent,omitempty"`
	Priority    int      `json:"priority"`
	Status      string   `json:"status"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
}

type Store struct {
	WorkTree string
	Prefix   string
}

func NewStore(workTree, prefix string) *Store {
	return &Store{WorkTree: workTree, Prefix: prefix}
}

func (s *Store) Create(title string, opts CreateOpts) (*Issue, error) {
	id, err := s.generateID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	issue := &Issue{
		Assignee:    opts.Assignee,
		BlockedBy:   []string{},
		Blocks:      []string{},
		Created:     now,
		Description: opts.Description,
		ID:          id,
		Labels:      []string{},
		Priority:    opts.Priority,
		Status:      "open",
		Title:       title,
		Type:        opts.Type,
	}
	if issue.Type == "" {
		issue.Type = "task"
	}
	if issue.Priority == 0 {
		issue.Priority = 3
	}

	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	if err := s.setStatus(id, "open"); err != nil {
		return nil, err
	}
	return issue, nil
}

func (s *Store) Get(id string) (*Issue, error) {
	id, err := s.resolveID(id)
	if err != nil {
		return nil, err
	}
	return s.readIssue(id)
}

func (s *Store) List(filter Filter) ([]*Issue, error) {
	statuses := []string{"open", "in_progress", "closed"}
	if filter.Status != "" {
		statuses = []string{filter.Status}
	}

	var ids []string
	for _, status := range statuses {
		dir := filepath.Join(s.WorkTree, "status", status)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
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
		if filter.Priority != 0 && issue.Priority != filter.Priority {
			continue
		}
		if filter.Type != "" && issue.Type != filter.Type {
			continue
		}
		if filter.Label != "" && !containsStr(issue.Labels, filter.Label) {
			continue
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

	if err := s.writeIssue(issue); err != nil {
		return nil, err
	}
	return issue, nil
}

func (s *Store) Close(id string) (*Issue, error) {
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
	issue.Status = "closed"
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
	dir := filepath.Join(s.WorkTree, "blocks", blockerID)
	os.MkdirAll(dir, 0755)
	if err := os.WriteFile(filepath.Join(dir, blockedID), []byte{}, 0644); err != nil {
		return err
	}

	// Update blocker's JSON
	blocker, err := s.readIssue(blockerID)
	if err != nil {
		return err
	}
	if !containsStr(blocker.Blocks, blockedID) {
		blocker.Blocks = append(blocker.Blocks, blockedID)
		sort.Strings(blocker.Blocks)
		if err := s.writeIssue(blocker); err != nil {
			return err
		}
	}

	// Update blocked's JSON
	blocked, err := s.readIssue(blockedID)
	if err != nil {
		return err
	}
	if !containsStr(blocked.BlockedBy, blockerID) {
		blocked.BlockedBy = append(blocked.BlockedBy, blockerID)
		sort.Strings(blocked.BlockedBy)
		if err := s.writeIssue(blocked); err != nil {
			return err
		}
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
	markerPath := filepath.Join(s.WorkTree, "blocks", blockerID, blockedID)
	os.Remove(markerPath)

	// Clean up empty directory
	dir := filepath.Join(s.WorkTree, "blocks", blockerID)
	entries, _ := os.ReadDir(dir)
	if len(entries) == 0 {
		os.Remove(dir)
	}

	// Update blocker's JSON
	blocker, err := s.readIssue(blockerID)
	if err != nil {
		return err
	}
	blocker.Blocks = removeStr(blocker.Blocks, blockedID)
	if err := s.writeIssue(blocker); err != nil {
		return err
	}

	// Update blocked's JSON
	blocked, err := s.readIssue(blockedID)
	if err != nil {
		return err
	}
	blocked.BlockedBy = removeStr(blocked.BlockedBy, blockerID)
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
		dir := filepath.Join(s.WorkTree, "labels", label)
		os.MkdirAll(dir, 0755)
		if err := os.WriteFile(filepath.Join(dir, id), []byte{}, 0644); err != nil {
			return nil, err
		}
		if !containsStr(issue.Labels, label) {
			issue.Labels = append(issue.Labels, label)
		}
	}

	// Remove labels
	for _, label := range remove {
		os.Remove(filepath.Join(s.WorkTree, "labels", label, id))
		// Clean up empty label directory
		dir := filepath.Join(s.WorkTree, "labels", label)
		entries, _ := os.ReadDir(dir)
		empty := true
		for _, e := range entries {
			if e.Name() != ".gitkeep" {
				empty = false
				break
			}
		}
		if empty {
			os.RemoveAll(dir)
		}
		issue.Labels = removeStr(issue.Labels, label)
	}

	sort.Strings(issue.Labels)
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
		rootID, _ = s.resolveID(rootID)
	}

	// Read all block relationships from the filesystem
	blocksDir := filepath.Join(s.WorkTree, "blocks")
	entries, err := os.ReadDir(blocksDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Collect all issue IDs involved in any relationship
	involved := make(map[string]bool)
	edges := make(map[string][]string) // blocker -> []blocked

	for _, e := range entries {
		if !e.IsDir() || e.Name() == ".gitkeep" {
			continue
		}
		blockerID := e.Name()
		children, err := os.ReadDir(filepath.Join(blocksDir, blockerID))
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

func (s *Store) Ready() ([]*Issue, error) {
	issues, err := s.List(Filter{})
	if err != nil {
		return nil, err
	}

	var ready []*Issue
	for _, issue := range issues {
		if issue.Status == "closed" {
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

// --- Internal helpers ---

func (s *Store) generateID() (string, error) {
	existing := s.existingIDs()
	for attempts := 0; attempts < 100; attempts++ {
		b := make([]byte, 3)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		suffix := hex.EncodeToString(b)[:4]
		id := s.Prefix + "-" + suffix
		if !existing[id] {
			return id, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique ID after 100 attempts")
}

func (s *Store) existingIDs() map[string]bool {
	ids := make(map[string]bool)
	entries, err := os.ReadDir(filepath.Join(s.WorkTree, "issues"))
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
	entries, err := os.ReadDir(filepath.Join(s.WorkTree, "issues"))
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
	path := filepath.Join(s.WorkTree, "issues", id+".json")
	data, err := os.ReadFile(path)
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
	path := filepath.Join(s.WorkTree, "issues", issue.ID+".json")
	return os.WriteFile(path, data, 0644)
}

func (s *Store) setStatus(id, status string) error {
	dir := filepath.Join(s.WorkTree, "status", status)
	os.MkdirAll(dir, 0755)
	return os.WriteFile(filepath.Join(dir, id), []byte{}, 0644)
}

func (s *Store) moveStatus(id, from, to string) error {
	oldPath := filepath.Join(s.WorkTree, "status", from, id)
	os.Remove(oldPath) // best effort remove from old status
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
	Description string
	Priority    int
	Type        string
	Assignee    string
}

type UpdateOpts struct {
	Title       *string
	Description *string
	Priority    *int
	Assignee    *string
	Type        *string
	Status      *string
}

type Filter struct {
	Status   string
	Assignee string
	Priority int
	Type     string
	Label    string
}
