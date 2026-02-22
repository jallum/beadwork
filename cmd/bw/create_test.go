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
	err := cmdCreate(env.Store, []string{"Test issue"}, PlainWriter(&buf))
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
	err := cmdCreate(env.Store, []string{"Bug report", "--priority", "1", "--type", "bug"}, PlainWriter(&buf))
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
	err := cmdCreate(env.Store, []string{"JSON test", "--json"}, PlainWriter(&buf))
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
	err := cmdCreate(env.Store, []string{"Labeled task", "--labels", "frontend,urgent"}, PlainWriter(&buf))
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

func TestCmdCreateSilent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate(env.Store, []string{"Silent test", "--silent"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}
	out := strings.TrimSpace(buf.String())

	// Silent output should be the bare issue ID only
	issues, _ := env.Store.List(issue.Filter{})
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if out != issues[0].ID {
		t.Errorf("silent output = %q, want bare ID %q", out, issues[0].ID)
	}

	// Should NOT contain the human-readable "created" prefix
	if strings.Contains(buf.String(), "created") {
		t.Errorf("silent output should not contain 'created': %q", buf.String())
	}
}

func TestCmdCreateSilentNoExtraOutput(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate(env.Store, []string{"Silent only ID", "--silent"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}
	out := buf.String()

	// Output should be exactly one line: the issue ID followed by a newline
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line of output, got %d: %q", len(lines), out)
	}

	// The line should match the issue ID prefix pattern
	if !strings.HasPrefix(lines[0], env.Store.Prefix+"-") {
		t.Errorf("silent output %q does not start with prefix %q", lines[0], env.Store.Prefix+"-")
	}
}

func TestCmdCreateWithExplicitID(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate(env.Store, []string{"Explicit ID", "--id", "test-myid"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "test-myid") {
		t.Errorf("output = %q, want 'test-myid'", out)
	}

	// Verify the issue was created with the explicit ID
	iss, err := env.Store.Get("test-myid")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if iss.Title != "Explicit ID" {
		t.Errorf("title = %q, want %q", iss.Title, "Explicit ID")
	}
}

func TestCmdCreateWithExplicitIDSilent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate(env.Store, []string{"Silent explicit", "--id", "test-silent-id", "--silent"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if out != "test-silent-id" {
		t.Errorf("silent output = %q, want %q", out, "test-silent-id")
	}
}

func TestCmdCreateWithExplicitIDJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate(env.Store, []string{"JSON explicit", "--id", "test-json-id", "--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}

	var iss issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &iss); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if iss.ID != "test-json-id" {
		t.Errorf("ID = %q, want %q", iss.ID, "test-json-id")
	}
}

func TestCmdCreateWithLabelsJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdCreate(env.Store, []string{"Labeled JSON", "--labels", "backend", "--json"}, PlainWriter(&buf))
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
