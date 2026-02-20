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

// --- Comments Add ---

func TestCmdCommentsAddBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdComments([]string{"add", iss.ID, "This is a comment"}, &buf)
	if err != nil {
		t.Fatalf("cmdComments add: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "comment added") {
		t.Errorf("output = %q, want 'comment added'", output)
	}

	// Verify comment was stored
	got, _ := env.Store.Get(iss.ID)
	if len(got.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(got.Comments))
	}
	if got.Comments[0].Text != "This is a comment" {
		t.Errorf("text = %q, want %q", got.Comments[0].Text, "This is a comment")
	}
}

func TestCmdCommentsAddNoText(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdComments([]string{"add", iss.ID}, &buf)
	if err == nil {
		t.Fatal("expected error when no text provided")
	}
}

func TestCmdCommentsAddWithAuthor(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdComments([]string{"add", iss.ID, "Authored comment", "--author", "alice"}, &buf)
	if err != nil {
		t.Fatalf("cmdComments add: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Comments[0].Author != "alice" {
		t.Errorf("author = %q, want %q", got.Comments[0].Author, "alice")
	}
}

func TestCmdCommentsAddWithShortAuthor(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdComments([]string{"add", iss.ID, "Short flag", "-a", "bob"}, &buf)
	if err != nil {
		t.Fatalf("cmdComments add: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Comments[0].Author != "bob" {
		t.Errorf("author = %q, want %q", got.Comments[0].Author, "bob")
	}
}

// --- Comments List ---

func TestCmdCommentsListPlain(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	env.Store.Comment(iss.ID, "First comment", "alice")
	env.CommitIntent("comment 1")
	env.Store.Comment(iss.ID, "Second comment", "")
	env.CommitIntent("comment 2")

	var buf bytes.Buffer
	err := cmdComments([]string{iss.ID}, &buf)
	if err != nil {
		t.Fatalf("cmdComments list: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "First comment") {
		t.Errorf("output missing 'First comment': %q", output)
	}
	if !strings.Contains(output, "Second comment") {
		t.Errorf("output missing 'Second comment': %q", output)
	}
	if !strings.Contains(output, "alice") {
		t.Errorf("output missing author 'alice': %q", output)
	}
}

func TestCmdCommentsListJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	env.Store.Comment(iss.ID, "JSON comment", "alice")
	env.CommitIntent("comment")

	var buf bytes.Buffer
	err := cmdComments([]string{iss.ID, "--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdComments list --json: %v", err)
	}

	var comments []issue.Comment
	if err := json.Unmarshal(buf.Bytes(), &comments); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Text != "JSON comment" {
		t.Errorf("text = %q, want %q", comments[0].Text, "JSON comment")
	}
}

func TestCmdCommentsListNonexistent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdComments([]string{"test-nonexistent"}, &buf)
	if err == nil {
		t.Fatal("expected error for nonexistent issue")
	}
}

func TestCmdCommentsListEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdComments([]string{iss.ID}, &buf)
	if err != nil {
		t.Fatalf("cmdComments list: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "no comments") {
		t.Errorf("output for empty comments = %q, want 'no comments'", output)
	}
}

// --- Show integration ---

func TestFprintIssueWithComments(t *testing.T) {
	iss := &issue.Issue{
		ID:      "test-abc",
		Title:   "Test",
		Status:  "open",
		Type:    "task",
		Created: "2026-02-20T00:00:00Z",
		Comments: []issue.Comment{
			{Text: "First note", Author: "alice", Timestamp: "2026-02-20T10:00:00Z"},
			{Text: "Second note", Author: "", Timestamp: "2026-02-20T11:00:00Z"},
		},
	}

	var buf bytes.Buffer
	fprintIssue(&buf, iss)
	output := buf.String()

	if !strings.Contains(output, "COMMENTS") {
		t.Errorf("output missing COMMENTS section: %q", output)
	}
	if !strings.Contains(output, "First note") {
		t.Errorf("output missing 'First note': %q", output)
	}
	if !strings.Contains(output, "Second note") {
		t.Errorf("output missing 'Second note': %q", output)
	}
	if !strings.Contains(output, "alice") {
		t.Errorf("output missing author 'alice': %q", output)
	}
}

func TestFprintIssueWithoutComments(t *testing.T) {
	iss := &issue.Issue{
		ID:      "test-abc",
		Title:   "Test",
		Status:  "open",
		Type:    "task",
		Created: "2026-02-20T00:00:00Z",
	}

	var buf bytes.Buffer
	fprintIssue(&buf, iss)
	output := buf.String()

	if strings.Contains(output, "COMMENTS") {
		t.Errorf("output should not contain COMMENTS when no comments: %q", output)
	}
}

func TestCmdShowJSONIncludesComments(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Test issue", issue.CreateOpts{})
	env.CommitIntent("create " + iss.ID)

	env.Store.Comment(iss.ID, "Show JSON comment", "alice")
	env.CommitIntent("comment")

	var buf bytes.Buffer
	err := cmdShow([]string{iss.ID, "--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdShow --json: %v", err)
	}

	// show --json returns an array of issues
	var issues []issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &issues); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if len(issues[0].Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(issues[0].Comments))
	}
	if issues[0].Comments[0].Text != "Show JSON comment" {
		t.Errorf("text = %q, want %q", issues[0].Comments[0].Text, "Show JSON comment")
	}
}

// --- Export/Import roundtrip ---

func TestExportIncludesComments(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("With comments", issue.CreateOpts{})
	env.Store.Comment(iss.ID, "Export this comment", "alice")
	env.Repo.Commit("create and comment")

	var buf bytes.Buffer
	if err := cmdExport([]string{}, &buf); err != nil {
		t.Fatalf("cmdExport: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	found := false
	for _, line := range lines {
		var rec beadsRecord
		json.Unmarshal([]byte(line), &rec)
		if rec.ID == iss.ID && len(rec.Comments) == 1 {
			found = true
			if rec.Comments[0].Text != "Export this comment" {
				t.Errorf("exported comment text = %q", rec.Comments[0].Text)
			}
			if rec.Comments[0].CreatedAt == "" {
				t.Error("exported comment created_at should be set")
			}
		}
	}
	if !found {
		t.Error("issue with comment not found in export output")
	}
}

func TestImportWithComments(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	jsonl := `{"id":"test-cmt1","title":"Has comments","status":"open","priority":2,"issue_type":"task","created_at":"2024-01-01T00:00:00Z","comments":[{"text":"Imported comment","author":"bob","created_at":"2024-01-02T10:00:00Z"}]}`
	tmpFile := env.Dir + "/import.jsonl"
	os.WriteFile(tmpFile, []byte(jsonl+"\n"), 0644)

	var buf bytes.Buffer
	if err := cmdImport([]string{tmpFile}, &buf); err != nil {
		t.Fatalf("cmdImport: %v", err)
	}

	got, err := env.Store.Get("test-cmt1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(got.Comments))
	}
	if got.Comments[0].Text != "Imported comment" {
		t.Errorf("text = %q, want %q", got.Comments[0].Text, "Imported comment")
	}
	if got.Comments[0].Author != "bob" {
		t.Errorf("author = %q, want %q", got.Comments[0].Author, "bob")
	}
}

func TestExportImportRoundtrip(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Roundtrip", issue.CreateOpts{})
	env.Store.Comment(iss.ID, "Roundtrip comment", "charlie")
	env.Repo.Commit("create and comment")

	// Export
	var exportBuf bytes.Buffer
	if err := cmdExport([]string{}, &exportBuf); err != nil {
		t.Fatalf("cmdExport: %v", err)
	}

	// Create a new env for import
	env2 := testutil.NewEnv(t)
	defer env2.Cleanup()

	tmpFile := env2.Dir + "/roundtrip.jsonl"
	os.WriteFile(tmpFile, exportBuf.Bytes(), 0644)

	var importBuf bytes.Buffer
	if err := cmdImport([]string{tmpFile}, &importBuf); err != nil {
		t.Fatalf("cmdImport: %v", err)
	}

	got, err := env2.Store.Get(iss.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Comments) != 1 {
		t.Fatalf("expected 1 comment after roundtrip, got %d", len(got.Comments))
	}
	if got.Comments[0].Text != "Roundtrip comment" {
		t.Errorf("text = %q", got.Comments[0].Text)
	}
	if got.Comments[0].Author != "charlie" {
		t.Errorf("author = %q", got.Comments[0].Author)
	}
	if got.Comments[0].Timestamp == "" {
		t.Error("timestamp should be preserved")
	}
}
