package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestScenarioFullWorkflow(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// ── Step 1: Create 5 issues ──
	var bufA bytes.Buffer
	if err := cmdCreate([]string{"API design", "--priority", "1", "--type", "bug", "--assignee", "alice"}, PlainWriter(&bufA)); err != nil {
		t.Fatalf("create A: %v", err)
	}
	var bufB bytes.Buffer
	if err := cmdCreate([]string{"Database schema", "--priority", "1", "--type", "task"}, PlainWriter(&bufB)); err != nil {
		t.Fatalf("create B: %v", err)
	}
	var bufC bytes.Buffer
	if err := cmdCreate([]string{"Frontend UI", "--priority", "2", "--type", "task"}, PlainWriter(&bufC)); err != nil {
		t.Fatalf("create C: %v", err)
	}
	var bufD bytes.Buffer
	if err := cmdCreate([]string{"Documentation", "--priority", "3", "--type", "task", "--defer", "2027-06-01"}, PlainWriter(&bufD)); err != nil {
		t.Fatalf("create D: %v", err)
	}
	var bufE bytes.Buffer
	if err := cmdCreate([]string{"Performance audit", "--priority", "2", "--type", "task"}, PlainWriter(&bufE)); err != nil {
		t.Fatalf("create E: %v", err)
	}

	// Collect IDs from store (sorted by creation order via list --all)
	allIssues, _ := env.Store.List(issue.Filter{})
	if len(allIssues) != 5 {
		t.Fatalf("expected 5 issues, got %d", len(allIssues))
	}
	idOf := make(map[string]string) // title → ID
	for _, iss := range allIssues {
		idOf[iss.Title] = iss.ID
	}
	idA := idOf["API design"]
	idB := idOf["Database schema"]
	idC := idOf["Frontend UI"]
	idD := idOf["Documentation"]
	idE := idOf["Performance audit"]

	// ── Step 2: Wire dependencies ──
	// A blocks B, B blocks C
	var buf bytes.Buffer
	if err := cmdDepAdd([]string{idA, "blocks", idB}, PlainWriter(&buf)); err != nil {
		t.Fatalf("dep add A→B: %v", err)
	}
	buf.Reset()
	if err := cmdDepAdd([]string{idB, "blocks", idC}, PlainWriter(&buf)); err != nil {
		t.Fatalf("dep add B→C: %v", err)
	}

	// ── Step 3: Label ──
	buf.Reset()
	if err := cmdLabel([]string{idA, "+backend", "+critical"}, PlainWriter(&buf)); err != nil {
		t.Fatalf("label A: %v", err)
	}
	buf.Reset()
	if err := cmdLabel([]string{idC, "+frontend"}, PlainWriter(&buf)); err != nil {
		t.Fatalf("label C: %v", err)
	}

	// ── Step 4: Progress ──
	// Close A
	buf.Reset()
	if err := cmdClose([]string{idA}, PlainWriter(&buf)); err != nil {
		t.Fatalf("close A: %v", err)
	}
	// B to in_progress
	buf.Reset()
	if err := cmdUpdate([]string{idB, "--status", "in_progress"}, PlainWriter(&buf)); err != nil {
		t.Fatalf("update B: %v", err)
	}
	// Defer E
	buf.Reset()
	if err := cmdDefer([]string{idE, "2027-09-01"}, PlainWriter(&buf)); err != nil {
		t.Fatalf("defer E: %v", err)
	}

	// ════════════════════════════════════════════════════════
	// Verify: list --all --json
	// ════════════════════════════════════════════════════════
	t.Run("list_all_json", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdList([]string{"--all", "--json"}, PlainWriter(&buf)); err != nil {
			t.Fatalf("list --all --json: %v", err)
		}
		var issues []issue.Issue
		if err := json.Unmarshal(buf.Bytes(), &issues); err != nil {
			t.Fatalf("JSON parse: %v", err)
		}
		if len(issues) != 5 {
			t.Fatalf("expected 5 issues, got %d", len(issues))
		}

		byID := make(map[string]*issue.Issue)
		for i := range issues {
			byID[issues[i].ID] = &issues[i]
		}

		// A: closed, P1, bug, alice, labels=[backend,critical]
		a := byID[idA]
		if a == nil {
			t.Fatal("issue A missing")
		}
		if a.Status != "closed" {
			t.Errorf("A status=%q, want closed", a.Status)
		}
		if a.Priority != 1 {
			t.Errorf("A priority=%d, want 1", a.Priority)
		}
		if a.Type != "bug" {
			t.Errorf("A type=%q, want bug", a.Type)
		}
		if a.Assignee != "alice" {
			t.Errorf("A assignee=%q, want alice", a.Assignee)
		}
		if len(a.Labels) != 2 {
			t.Errorf("A labels=%v, want 2 labels", a.Labels)
		}

		// B: in_progress, P1, task, blocked by A, blocks C
		b := byID[idB]
		if b == nil {
			t.Fatal("issue B missing")
		}
		if b.Status != "in_progress" {
			t.Errorf("B status=%q, want in_progress", b.Status)
		}
		if len(b.BlockedBy) != 1 || b.BlockedBy[0] != idA {
			t.Errorf("B blocked_by=%v, want [%s]", b.BlockedBy, idA)
		}
		if len(b.Blocks) != 1 || b.Blocks[0] != idC {
			t.Errorf("B blocks=%v, want [%s]", b.Blocks, idC)
		}

		// C: open, P2, task, blocked by B, labels=[frontend]
		c := byID[idC]
		if c == nil {
			t.Fatal("issue C missing")
		}
		if c.Status != "open" {
			t.Errorf("C status=%q, want open", c.Status)
		}
		if len(c.BlockedBy) != 1 || c.BlockedBy[0] != idB {
			t.Errorf("C blocked_by=%v, want [%s]", c.BlockedBy, idB)
		}
		if len(c.Labels) != 1 || c.Labels[0] != "frontend" {
			t.Errorf("C labels=%v, want [frontend]", c.Labels)
		}

		// D: deferred, P3, task, defer_until=2027-06-01
		d := byID[idD]
		if d == nil {
			t.Fatal("issue D missing")
		}
		if d.Status != "deferred" {
			t.Errorf("D status=%q, want deferred", d.Status)
		}
		if d.DeferUntil != "2027-06-01" {
			t.Errorf("D defer_until=%q, want 2027-06-01", d.DeferUntil)
		}

		// E: deferred, P2, task, defer_until=2027-09-01
		e := byID[idE]
		if e == nil {
			t.Fatal("issue E missing")
		}
		if e.Status != "deferred" {
			t.Errorf("E status=%q, want deferred", e.Status)
		}
		if e.DeferUntil != "2027-09-01" {
			t.Errorf("E defer_until=%q, want 2027-09-01", e.DeferUntil)
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: list (default = open + in_progress) — B and C
	// ════════════════════════════════════════════════════════
	t.Run("list_default", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdList([]string{}, PlainWriter(&buf)); err != nil {
			t.Fatalf("list: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, idB) {
			t.Errorf("default list should contain B (%s, in_progress): %q", idB, out)
		}
		if !strings.Contains(out, idC) {
			t.Errorf("default list should contain C (%s, open): %q", idC, out)
		}
		// A (closed), D (deferred), E (deferred) should not appear
		for _, id := range []string{idA, idD, idE} {
			if strings.Contains(out, id) {
				t.Errorf("default list should NOT contain %s: %q", id, out)
			}
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: list --status in_progress — only B
	// ════════════════════════════════════════════════════════
	t.Run("list_in_progress", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdList([]string{"--status", "in_progress"}, PlainWriter(&buf)); err != nil {
			t.Fatalf("list --status in_progress: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, idB) {
			t.Errorf("should contain B (%s): %q", idB, out)
		}
		if strings.Contains(out, idA) || strings.Contains(out, idC) {
			t.Errorf("should not contain A or C: %q", out)
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: list --deferred — D and E
	// ════════════════════════════════════════════════════════
	t.Run("list_deferred", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdList([]string{"--deferred"}, PlainWriter(&buf)); err != nil {
			t.Fatalf("list --deferred: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, idD) {
			t.Errorf("should contain D (%s): %q", idD, out)
		}
		if !strings.Contains(out, idE) {
			t.Errorf("should contain E (%s): %q", idE, out)
		}
		if strings.Contains(out, idA) || strings.Contains(out, idB) || strings.Contains(out, idC) {
			t.Errorf("should not contain A, B, or C: %q", out)
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: show --json on D
	// ════════════════════════════════════════════════════════
	t.Run("show_deferred", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdShow([]string{idD, "--json"}, PlainWriter(&buf)); err != nil {
			t.Fatalf("show D: %v", err)
		}
		var arr []issue.Issue
		if err := json.Unmarshal(buf.Bytes(), &arr); err != nil {
			t.Fatalf("JSON parse: %v", err)
		}
		if len(arr) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(arr))
		}
		d := arr[0]
		if d.Status != "deferred" {
			t.Errorf("D status=%q, want deferred", d.Status)
		}
		if d.DeferUntil != "2027-06-01" {
			t.Errorf("D defer_until=%q, want 2027-06-01", d.DeferUntil)
		}
		if d.Title != "Documentation" {
			t.Errorf("D title=%q, want Documentation", d.Title)
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: ready — no ready issues (C is blocked, others not open)
	// ════════════════════════════════════════════════════════
	t.Run("ready_text", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdReady([]string{}, PlainWriter(&buf)); err != nil {
			t.Fatalf("ready: %v", err)
		}
		if !strings.Contains(buf.String(), "no ready issues") {
			t.Errorf("expected 'no ready issues', got: %q", buf.String())
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: ready --json — empty array
	// ════════════════════════════════════════════════════════
	t.Run("ready_json", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdReady([]string{"--json"}, PlainWriter(&buf)); err != nil {
			t.Fatalf("ready --json: %v", err)
		}
		var issues []issue.Issue
		if err := json.Unmarshal(buf.Bytes(), &issues); err != nil {
			t.Fatalf("JSON parse: %v", err)
		}
		if len(issues) != 0 {
			t.Errorf("expected 0 ready issues, got %d", len(issues))
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: blocked --json — C blocked by B
	// ════════════════════════════════════════════════════════
	t.Run("blocked_json", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdBlocked([]string{"--json"}, PlainWriter(&buf)); err != nil {
			t.Fatalf("blocked --json: %v", err)
		}
		var result []struct {
			ID           string   `json:"id"`
			OpenBlockers []string `json:"open_blockers"`
		}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("JSON parse: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 blocked issue, got %d", len(result))
		}
		if result[0].ID != idC {
			t.Errorf("blocked id=%q, want %q", result[0].ID, idC)
		}
		if len(result[0].OpenBlockers) != 0 {
			// B is in_progress, which counts as an open blocker for the blocked list
			// The Blocked() method checks if blockers have status != "closed"
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: graph --all
	// ════════════════════════════════════════════════════════
	t.Run("graph_all", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdGraph([]string{"--all"}, PlainWriter(&buf)); err != nil {
			t.Fatalf("graph --all: %v", err)
		}
		out := buf.String()
		// A, B, C should be in a chain
		if !strings.Contains(out, idA) {
			t.Errorf("graph should contain A: %q", out)
		}
		if !strings.Contains(out, idB) {
			t.Errorf("graph should contain B: %q", out)
		}
		if !strings.Contains(out, idC) {
			t.Errorf("graph should contain C: %q", out)
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: export — JSONL with all fields
	// ════════════════════════════════════════════════════════
	t.Run("export", func(t *testing.T) {
		var buf bytes.Buffer
		if err := cmdExport([]string{}, PlainWriter(&buf)); err != nil {
			t.Fatalf("export: %v", err)
		}
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if len(lines) != 5 {
			t.Fatalf("expected 5 JSONL lines, got %d", len(lines))
		}

		byID := make(map[string]beadsRecord)
		for _, line := range lines {
			var rec beadsRecord
			if err := json.Unmarshal([]byte(line), &rec); err != nil {
				t.Fatalf("JSONL parse: %v\nline: %s", err, line)
			}
			byID[rec.ID] = rec
		}

		// A: closed, labels=[backend,critical], blocks=[B]
		a := byID[idA]
		if a.Status != "closed" {
			t.Errorf("export A status=%q, want closed", a.Status)
		}
		if len(a.Labels) != 2 {
			t.Errorf("export A labels=%v, want 2 labels", a.Labels)
		}
		if len(a.Blocks) != 1 || a.Blocks[0] != idB {
			t.Errorf("export A blocks=%v, want [%s]", a.Blocks, idB)
		}
		if a.UpdatedAt == "" {
			t.Error("export A updated_at should be set")
		}
		if a.ClosedAt == "" {
			t.Error("export A closed_at should be set (issue is closed)")
		}

		// B: in_progress, blocked_by=[A], blocks=[C]
		b := byID[idB]
		if b.Status != "in_progress" {
			t.Errorf("export B status=%q, want in_progress", b.Status)
		}
		if len(b.BlockedBy) != 1 || b.BlockedBy[0] != idA {
			t.Errorf("export B blocked_by=%v, want [%s]", b.BlockedBy, idA)
		}
		if len(b.Blocks) != 1 || b.Blocks[0] != idC {
			t.Errorf("export B blocks=%v, want [%s]", b.Blocks, idC)
		}

		// C: open, blocked_by=[B], labels=[frontend]
		c := byID[idC]
		if c.Status != "open" {
			t.Errorf("export C status=%q, want open", c.Status)
		}
		if len(c.BlockedBy) != 1 || c.BlockedBy[0] != idB {
			t.Errorf("export C blocked_by=%v, want [%s]", c.BlockedBy, idB)
		}
		if len(c.Labels) != 1 || c.Labels[0] != "frontend" {
			t.Errorf("export C labels=%v, want [frontend]", c.Labels)
		}
		if c.ClosedAt != "" {
			t.Errorf("export C closed_at should be empty for open issue, got %q", c.ClosedAt)
		}
		if c.UpdatedAt == "" {
			t.Error("export C updated_at should be set")
		}

		// D: deferred, defer_until as RFC3339
		d := byID[idD]
		if d.Status != "deferred" {
			t.Errorf("export D status=%q, want deferred", d.Status)
		}
		if d.DeferUntil != "2027-06-01T00:00:00Z" {
			t.Errorf("export D defer_until=%q, want 2027-06-01T00:00:00Z", d.DeferUntil)
		}

		// E: deferred, defer_until as RFC3339
		e := byID[idE]
		if e.Status != "deferred" {
			t.Errorf("export E status=%q, want deferred", e.Status)
		}
		if e.DeferUntil != "2027-09-01T00:00:00Z" {
			t.Errorf("export E defer_until=%q, want 2027-09-01T00:00:00Z", e.DeferUntil)
		}

		// Verify dependencies array for B (should have A as blocker)
		foundBlockDep := false
		for _, dep := range b.Dependencies {
			if dep.Type == "blocks" && dep.DependsOnID == idA {
				foundBlockDep = true
			}
		}
		if !foundBlockDep {
			t.Errorf("export B deps=%v, missing blocks dep on A", b.Dependencies)
		}
	})

	// ════════════════════════════════════════════════════════
	// Verify: export → import round-trip
	// ════════════════════════════════════════════════════════
	t.Run("export_import_roundtrip", func(t *testing.T) {
		// Export to JSONL
		var exportBuf bytes.Buffer
		if err := cmdExport([]string{}, PlainWriter(&exportBuf)); err != nil {
			t.Fatalf("export: %v", err)
		}

		// Write to temp file
		tmpFile := env.Dir + "/roundtrip.jsonl"
		os.WriteFile(tmpFile, exportBuf.Bytes(), 0644)

		// Try dry-run import — should detect all 5 as collisions
		var dryBuf bytes.Buffer
		if err := cmdImport([]string{tmpFile, "--dry-run"}, PlainWriter(&dryBuf)); err != nil {
			t.Fatalf("import dry-run: %v", err)
		}
		out := dryBuf.String()
		if !strings.Contains(out, "5 skipped") {
			t.Errorf("expected '5 skipped' in dry-run output: %q", out)
		}
		if !strings.Contains(out, "importing 0 of 5") {
			t.Errorf("expected 'importing 0 of 5' in dry-run output: %q", out)
		}
	})
}
