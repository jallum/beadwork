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

func TestCmdExportBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Export me", issue.CreateOpts{})
	env.Repo.Commit("create issue")

	var buf bytes.Buffer
	_, err := cmdExport(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdExport: %v", err)
	}

	// Should be valid JSONL
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 {
		t.Error("expected at least one line of JSONL output")
	}
	var rec beadsRecord
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("JSONL parse: %v", err)
	}
	if rec.Title != "Export me" {
		t.Errorf("title = %q", rec.Title)
	}
}

func TestDateConversionHelpers(t *testing.T) {
	// toRFC3339Date
	if got := toRFC3339Date(""); got != "" {
		t.Errorf("toRFC3339Date empty = %q", got)
	}
	if got := toRFC3339Date("2027-06-01"); got != "2027-06-01T00:00:00Z" {
		t.Errorf("toRFC3339Date bare = %q", got)
	}
	if got := toRFC3339Date("2027-06-01T12:00:00Z"); got != "2027-06-01T12:00:00Z" {
		t.Errorf("toRFC3339Date passthrough = %q", got)
	}

	// fromRFC3339Date
	if got := fromRFC3339Date(""); got != "" {
		t.Errorf("fromRFC3339Date empty = %q", got)
	}
	if got := fromRFC3339Date("2027-06-01T00:00:00Z"); got != "2027-06-01" {
		t.Errorf("fromRFC3339Date rfc3339 = %q", got)
	}
	if got := fromRFC3339Date("2027-06-01"); got != "2027-06-01" {
		t.Errorf("fromRFC3339Date bare = %q", got)
	}
}

func TestCmdExportWithParent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	parent, _ := env.Store.Create("Parent", issue.CreateOpts{})
	env.Store.Import(&issue.Issue{
		ID:        "test-child",
		Title:     "Child",
		Status:    "open",
		Priority:  3,
		Type:      "task",
		Created:   "2024-01-01",
		Parent:    parent.ID,
		Labels:    []string{},
		Blocks:    []string{},
		BlockedBy: []string{},
	})
	env.Repo.Commit("create with parent")

	var buf bytes.Buffer
	if _, err := cmdExport(env.Store, []string{}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdExport: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	for _, line := range lines {
		var rec beadsRecord
		json.Unmarshal([]byte(line), &rec)
		if rec.ID == "test-child" {
			foundParent := false
			for _, dep := range rec.Dependencies {
				if dep.Type == "parent-child" && dep.DependsOnID == parent.ID {
					foundParent = true
				}
			}
			if !foundParent {
				t.Errorf("child deps=%v, missing parent-child dep on %s", rec.Dependencies, parent.ID)
			}
			return
		}
	}
	t.Error("child issue not found in export")
}

func TestExportDeferUntilPassthrough(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Import(&issue.Issue{
		ID:         "test-defer-pt",
		Title:      "Deferred date-only",
		Status:     "deferred",
		Priority:   2,
		Type:       "task",
		Created:    "2024-01-01",
		DeferUntil: "2027-06-01",
		Labels:     []string{},
		Blocks:     []string{},
		BlockedBy:  []string{},
	})
	env.Repo.Commit("create deferred issue")

	var buf bytes.Buffer
	if _, err := cmdExport(env.Store, []string{}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdExport: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"defer_until":"2027-06-01"`) {
		t.Errorf("expected date-only passthrough, got: %s", output)
	}
	if strings.Contains(output, `"defer_until":"2027-06-01T00:00:00Z"`) {
		t.Error("defer_until should NOT be padded to RFC3339")
	}
}

func TestExportDeferUntilRFC3339Passthrough(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Import(&issue.Issue{
		ID:         "test-defer-rfc",
		Title:      "Deferred RFC3339",
		Status:     "deferred",
		Priority:   2,
		Type:       "task",
		Created:    "2024-01-01",
		DeferUntil: "2027-06-01T14:00:00-04:00",
		Labels:     []string{},
		Blocks:     []string{},
		BlockedBy:  []string{},
	})
	env.Repo.Commit("create deferred issue rfc3339")

	var buf bytes.Buffer
	if _, err := cmdExport(env.Store, []string{}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdExport: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"defer_until":"2027-06-01T14:00:00-04:00"`) {
		t.Errorf("expected RFC3339 passthrough, got: %s", output)
	}
}

func TestExportImportWithDue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Import(&issue.Issue{
		ID:        "test-due-rt",
		Title:     "Has due date",
		Status:    "open",
		Priority:  1,
		Type:      "task",
		Created:   "2024-01-01",
		Due:       "2027-12-25",
		Labels:    []string{},
		Blocks:    []string{},
		BlockedBy: []string{},
	})
	env.Repo.Commit("create issue with due")

	// Export
	var buf bytes.Buffer
	if _, err := cmdExport(env.Store, []string{}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdExport: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"due":"2027-12-25"`) {
		t.Errorf("export missing due field, got: %s", output)
	}

	// Write exported JSONL to a file and reimport into a fresh env
	env2 := testutil.NewEnv(t)
	defer env2.Cleanup()

	tmpFile := env2.Dir + "/due-export.jsonl"
	os.WriteFile(tmpFile, buf.Bytes(), 0644)

	var buf2 bytes.Buffer
	if _, err := cmdImport(env2.Store, []string{tmpFile}, PlainWriter(&buf2), nil); err != nil {
		t.Fatalf("cmdImport: %v", err)
	}

	iss, err := env2.Store.Get("test-due-rt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if iss.Due != "2027-12-25" {
		t.Errorf("due = %q, want 2027-12-25", iss.Due)
	}
}

func TestImportOldFormatDeferUntil(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	jsonl := `{"id":"test-old-defer","title":"Old format","status":"deferred","priority":2,"issue_type":"task","created_at":"2024-01-01T00:00:00Z","defer_until":"2027-06-01T00:00:00Z"}`
	tmpFile := env.Dir + "/old-format.jsonl"
	os.WriteFile(tmpFile, []byte(jsonl+"\n"), 0644)

	var buf bytes.Buffer
	if _, err := cmdImport(env.Store, []string{tmpFile}, PlainWriter(&buf), nil); err != nil {
		t.Fatalf("cmdImport: %v", err)
	}

	iss, err := env.Store.Get("test-old-defer")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if iss.DeferUntil != "2027-06-01T00:00:00Z" {
		t.Errorf("defer_until = %q, want 2027-06-01T00:00:00Z (stored as-is)", iss.DeferUntil)
	}
}
