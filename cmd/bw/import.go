package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/jallum/beadwork/internal/issue"
)

type importRecord struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Status       string     `json:"status"`
	Priority     int        `json:"priority"`
	IssueType    string     `json:"issue_type"`
	Owner        string     `json:"owner"`
	CreatedAt    string     `json:"created_at"`
	Labels       []string   `json:"labels"`
	DeferUntil   string     `json:"defer_until"`
	Dependencies []beadsDep `json:"dependencies"`
}

type ImportArgs struct {
	FilePath string
	DryRun   bool
}

func parseImportArgs(raw []string) (ImportArgs, error) {
	a := ParseArgs(raw)
	filePath := a.PosFirst()
	if filePath == "" {
		return ImportArgs{}, fmt.Errorf("usage: bw import <file> [--dry-run]")
	}
	return ImportArgs{
		FilePath: filePath,
		DryRun:   a.Bool("--dry-run"),
	}, nil
}

func cmdImport(args []string, w io.Writer) error {
	ia, err := parseImportArgs(args)
	if err != nil {
		return err
	}

	r, store, err := getInitialized()
	if err != nil {
		return err
	}

	dryRun := ia.DryRun
	filePath := ia.FilePath

	// Phase 1: Parse
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var records []importRecord
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}
		var rec importRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return fmt.Errorf("line %d: %v", lineNum, err)
		}
		records = append(records, rec)
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Phase 2: Collision check
	existing := store.ExistingIDs()
	var toImport []importRecord
	skipped := 0
	for _, rec := range records {
		if existing[rec.ID] {
			fmt.Fprintf(w, "skipping: %s already exists\n", rec.ID)
			skipped++
		} else {
			toImport = append(toImport, rec)
		}
	}

	fmt.Fprintf(w, "importing %d of %d issues", len(toImport), len(records))
	if skipped > 0 {
		fmt.Fprintf(w, " (%d skipped)", skipped)
	}
	fmt.Fprintln(w)

	if dryRun {
		fmt.Fprintln(w, "dry run: no changes made")
		return nil
	}

	if len(toImport) == 0 {
		return nil
	}

	// Phase 3: Write issues (first pass: set parent from deps, write all)
	for i := range toImport {
		rec := &toImport[i]
		labels := rec.Labels
		if labels == nil {
			labels = []string{}
		}
		iss := &issue.Issue{
			ID:          rec.ID,
			Title:       rec.Title,
			Description: rec.Description,
			Status:      rec.Status,
			Priority:    rec.Priority,
			Type:        rec.IssueType,
			Assignee:    rec.Owner,
			Created:     rec.CreatedAt,
			DeferUntil:  fromRFC3339Date(rec.DeferUntil),
			Labels:      labels,
			Blocks:      []string{},
			BlockedBy:   []string{},
		}
		if iss.Status == "" {
			iss.Status = "open"
		}
		if iss.Type == "" {
			iss.Type = "task"
		}
		if iss.Priority == 0 {
			iss.Priority = 2
		}
		// Set parent from dependencies
		for _, dep := range rec.Dependencies {
			if dep.Type == "parent-child" {
				iss.Parent = dep.DependsOnID
			}
		}
		if err := store.Import(iss); err != nil {
			return fmt.Errorf("import %s: %v", rec.ID, err)
		}
	}

	// Phase 3b: Process block dependencies (second pass, all issues exist)
	allIDs := make(map[string]bool)
	for _, rec := range toImport {
		allIDs[rec.ID] = true
	}
	for id := range existing {
		allIDs[id] = true
	}

	for _, rec := range toImport {
		for _, dep := range rec.Dependencies {
			if dep.Type != "blocks" || !allIDs[dep.DependsOnID] {
				continue
			}
			store.Link(dep.DependsOnID, rec.ID)
		}
	}

	intent := fmt.Sprintf("import %d issues", len(toImport))
	if err := r.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	// Count by status
	counts := map[string]int{}
	for _, rec := range toImport {
		s := rec.Status
		if s == "" {
			s = "open"
		}
		counts[s]++
	}
	fmt.Fprintf(w, "imported %d issues", len(toImport))
	parts := []string{}
	for _, s := range []string{"open", "in_progress", "deferred", "closed"} {
		if c := counts[s]; c > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c, s))
		}
	}
	if len(parts) > 0 {
		fmt.Fprintf(w, " (%s)", joinParts(parts))
	}
	fmt.Fprintln(w)
	return nil
}

func joinParts(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}
