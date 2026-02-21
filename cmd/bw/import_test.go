package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdImportBasic(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create a JSONL file
	jsonl := `{"id":"test-imp1","title":"Imported","status":"open","priority":2,"issue_type":"task","created_at":"2024-01-01T00:00:00Z"}`
	tmpFile := env.Dir + "/import.jsonl"
	os.WriteFile(tmpFile, []byte(jsonl+"\n"), 0644)

	var buf bytes.Buffer
	err := cmdImport(env.Store, []string{tmpFile}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdImport: %v", err)
	}
	if !strings.Contains(buf.String(), "imported 1") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestCmdImportDryRun(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	jsonl := `{"id":"test-dry1","title":"Dry run","status":"open","priority":2,"issue_type":"task","created_at":"2024-01-01T00:00:00Z"}`
	tmpFile := env.Dir + "/import.jsonl"
	os.WriteFile(tmpFile, []byte(jsonl+"\n"), 0644)

	var buf bytes.Buffer
	err := cmdImport(env.Store, []string{tmpFile, "--dry-run"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdImport: %v", err)
	}
	if !strings.Contains(buf.String(), "dry run") {
		t.Errorf("output = %q", buf.String())
	}

	// Issue should not exist
	_, getErr := env.Store.Get("test-dry1")
	if getErr == nil {
		t.Error("issue should not exist after dry run")
	}
}

func TestCmdImportPriorityZero(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// P0 should be preserved, not treated as "unset"
	jsonl := `{"id":"test-p0","title":"Critical","status":"open","priority":0,"issue_type":"bug","created_at":"2024-01-01T00:00:00Z"}`
	tmpFile := env.Dir + "/import.jsonl"
	os.WriteFile(tmpFile, []byte(jsonl+"\n"), 0644)

	var buf bytes.Buffer
	if err := cmdImport(env.Store, []string{tmpFile}, PlainWriter(&buf)); err != nil {
		t.Fatalf("cmdImport: %v", err)
	}

	iss, err := env.Store.Get("test-p0")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if iss.Priority != 0 {
		t.Errorf("priority = %d, want 0 (P0)", iss.Priority)
	}
}

func TestCmdImportPriorityAbsent(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// No priority field — should default to 2
	jsonl := `{"id":"test-nopr","title":"No priority","status":"open","issue_type":"task","created_at":"2024-01-01T00:00:00Z"}`
	tmpFile := env.Dir + "/import.jsonl"
	os.WriteFile(tmpFile, []byte(jsonl+"\n"), 0644)

	var buf bytes.Buffer
	if err := cmdImport(env.Store, []string{tmpFile}, PlainWriter(&buf)); err != nil {
		t.Fatalf("cmdImport: %v", err)
	}

	iss, err := env.Store.Get("test-nopr")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if iss.Priority != 2 {
		t.Errorf("priority = %d, want 2 (default)", iss.Priority)
	}
}

func TestCmdImportStdin(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	jsonl := `{"id":"stdin-1","title":"From stdin","status":"open","issue_type":"task","created_at":"2024-01-01T00:00:00Z"}`

	orig := importStdin
	importStdin = bytes.NewReader([]byte(jsonl + "\n"))
	t.Cleanup(func() { importStdin = orig })

	var buf bytes.Buffer
	if err := cmdImport(env.Store, []string{"-"}, PlainWriter(&buf)); err != nil {
		t.Fatalf("cmdImport stdin: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("imported 1 issues")) {
		t.Errorf("output = %q, want 'imported 1 issues'", buf.String())
	}

	iss, err := env.Store.Get("stdin-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if iss.Title != "From stdin" {
		t.Errorf("title = %q, want %q", iss.Title, "From stdin")
	}
}
