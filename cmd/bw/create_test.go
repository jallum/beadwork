package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdCreateBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate([]string{"Test issue"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}
	if !strings.Contains(buf.String(), "created") {
		t.Errorf("output = %q, want 'created'", buf.String())
	}
}

func TestCmdCreateWithFlags(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate([]string{"Bug report", "--priority", "1", "--type", "bug", "--assignee", "alice"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}

	// Verify the issue was created with correct attributes
	issues, _ := env.Store.List(issue.Filter{})
	found := false
	for _, iss := range issues {
		if iss.Title == "Bug report" {
			found = true
			if iss.Priority != 1 {
				t.Errorf("priority = %d, want 1", iss.Priority)
			}
			if iss.Type != "bug" {
				t.Errorf("type = %q, want bug", iss.Type)
			}
			if iss.Assignee != "alice" {
				t.Errorf("assignee = %q, want alice", iss.Assignee)
			}
		}
	}
	if !found {
		t.Error("issue 'Bug report' not found")
	}
}

func TestCmdCreateJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate([]string{"JSON test", "--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}

	var iss issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &iss); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if iss.Title != "JSON test" {
		t.Errorf("title = %q, want 'JSON test'", iss.Title)
	}
}

func TestCmdCreateWithLabels(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate([]string{"Labeled task", "--labels", "frontend,urgent"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Labeled task") {
		t.Errorf("output missing title: %q", out)
	}

	// Verify labels were applied by re-reading via store
	issues, _ := env.Store.List(issue.Filter{})
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	iss := issues[0]
	if len(iss.Labels) != 2 {
		t.Fatalf("Labels = %v, want 2 labels", iss.Labels)
	}
	if iss.Labels[0] != "frontend" || iss.Labels[1] != "urgent" {
		t.Errorf("Labels = %v, want [frontend urgent]", iss.Labels)
	}
}

func TestCmdCreateWithLabelsJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate([]string{"Labeled JSON", "--labels", "backend", "--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}

	var iss issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &iss); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(iss.Labels) != 1 || iss.Labels[0] != "backend" {
		t.Errorf("Labels = %v, want [backend]", iss.Labels)
	}
}
