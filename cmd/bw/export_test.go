package main

import (
	"bytes"
	"encoding/json"
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
	err := cmdExport(env.Repo, env.Store, []string{}, PlainWriter(&buf))
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
	if err := cmdExport(env.Repo, env.Store, []string{}, PlainWriter(&buf)); err != nil {
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
