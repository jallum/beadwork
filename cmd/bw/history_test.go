package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

// --- parseHistoryArgs ---

func TestParseHistoryArgsBasic(t *testing.T) {
	ha, err := parseHistoryArgs([]string{"bw-1234"})
	if err != nil {
		t.Fatal(err)
	}
	if ha.ID != "bw-1234" {
		t.Errorf("ID = %q, want bw-1234", ha.ID)
	}
	if ha.Limit != 0 {
		t.Errorf("Limit = %d, want 0", ha.Limit)
	}
	if ha.JSON {
		t.Error("expected JSON = false")
	}
}

func TestParseHistoryArgsWithLimit(t *testing.T) {
	ha, err := parseHistoryArgs([]string{"bw-1234", "--limit", "5"})
	if err != nil {
		t.Fatal(err)
	}
	if ha.ID != "bw-1234" {
		t.Errorf("ID = %q", ha.ID)
	}
	if ha.Limit != 5 {
		t.Errorf("Limit = %d, want 5", ha.Limit)
	}
}

func TestParseHistoryArgsWithJSON(t *testing.T) {
	ha, err := parseHistoryArgs([]string{"bw-1234", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !ha.JSON {
		t.Error("expected JSON = true")
	}
}

func TestParseHistoryArgsWithAll(t *testing.T) {
	ha, err := parseHistoryArgs([]string{"bw-abcd", "--limit", "3", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if ha.ID != "bw-abcd" {
		t.Errorf("ID = %q", ha.ID)
	}
	if ha.Limit != 3 {
		t.Errorf("Limit = %d, want 3", ha.Limit)
	}
	if !ha.JSON {
		t.Error("expected JSON = true")
	}
}

func TestParseHistoryArgsMissingID(t *testing.T) {
	_, err := parseHistoryArgs([]string{})
	if err == nil {
		t.Error("expected error for missing id")
	}
}

// --- cmdHistory ---

func TestCmdHistoryShowsMatchingCommits(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create an issue so we have a valid ID
	iss, err := env.Store.Create("Test issue", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := env.Repo.Commit(iss.ID + ": create issue"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Make another commit mentioning this issue
	env.Repo.TreeFS().WriteFile("dummy.txt", []byte("x"))
	if err := env.Repo.Commit(iss.ID + ": update something"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	var buf bytes.Buffer
	err = cmdHistory(env.Store, []string{iss.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdHistory: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, iss.ID) {
		t.Errorf("output should contain issue ID %q, got: %s", iss.ID, out)
	}
}

func TestCmdHistoryFiltersOutOtherIssues(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Create two issues
	iss1, err := env.Store.Create("Issue one", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := env.Repo.Commit(iss1.ID + ": create issue one"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	iss2, err := env.Store.Create("Issue two", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := env.Repo.Commit(iss2.ID + ": create issue two"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	var buf bytes.Buffer
	err = cmdHistory(env.Store, []string{iss1.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdHistory: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, iss1.ID) {
		t.Errorf("output should contain %q", iss1.ID)
	}
	if strings.Contains(out, iss2.ID) {
		t.Errorf("output should NOT contain %q, got: %s", iss2.ID, out)
	}
}

func TestCmdHistoryChronologicalOrder(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Chrono test", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := env.Repo.Commit(iss.ID + ": first action"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	env.Repo.TreeFS().WriteFile("dummy2.txt", []byte("y"))
	if err := env.Repo.Commit(iss.ID + ": second action"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	var buf bytes.Buffer
	err = cmdHistory(env.Store, []string{iss.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdHistory: %v", err)
	}

	lines := nonEmptyLines(buf.String())
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %s", len(lines), buf.String())
	}

	// First line should mention "first", last line should mention "second"
	if !strings.Contains(lines[0], "first action") {
		t.Errorf("first line should contain 'first action', got: %s", lines[0])
	}
	if !strings.Contains(lines[len(lines)-1], "second action") {
		t.Errorf("last line should contain 'second action', got: %s", lines[len(lines)-1])
	}
}

func TestCmdHistoryLimit(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("Limit test", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Create 3 commits for this issue
	for i, msg := range []string{"alpha", "beta", "gamma"} {
		if i > 0 {
			env.Repo.TreeFS().WriteFile("f"+msg+".txt", []byte(msg))
		}
		if err := env.Repo.Commit(iss.ID + ": " + msg); err != nil {
			t.Fatalf("Commit: %v", err)
		}
	}

	var buf bytes.Buffer
	err = cmdHistory(env.Store, []string{iss.ID, "--limit", "2"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdHistory: %v", err)
	}

	lines := nonEmptyLines(buf.String())
	if len(lines) != 2 {
		t.Errorf("expected 2 lines with --limit 2, got %d: %s", len(lines), buf.String())
	}

	// --limit shows the most recent N, so we should see beta and gamma
	if len(lines) == 2 {
		if !strings.Contains(lines[0], "beta") {
			t.Errorf("line 0 should contain 'beta', got: %s", lines[0])
		}
		if !strings.Contains(lines[1], "gamma") {
			t.Errorf("line 1 should contain 'gamma', got: %s", lines[1])
		}
	}
}

func TestCmdHistoryJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, err := env.Store.Create("JSON test", issue.CreateOpts{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := env.Repo.Commit(iss.ID + ": json commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	var buf bytes.Buffer
	err = cmdHistory(env.Store, []string{iss.ID, "--json"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdHistory: %v", err)
	}

	var entries []struct {
		Hash      string `json:"hash"`
		Timestamp string `json:"timestamp"`
		Author    string `json:"author"`
		Intent    string `json:"intent"`
	}
	if err := json.Unmarshal(buf.Bytes(), &entries); err != nil {
		t.Fatalf("JSON parse: %v\nraw: %s", err, buf.String())
	}
	if len(entries) == 0 {
		t.Fatal("expected at least 1 entry")
	}
	for i, e := range entries {
		if e.Hash == "" {
			t.Errorf("entry[%d].Hash is empty", i)
		}
		if e.Timestamp == "" {
			t.Errorf("entry[%d].Timestamp is empty", i)
		}
		if e.Author == "" {
			t.Errorf("entry[%d].Author is empty", i)
		}
		if e.Intent == "" {
			t.Errorf("entry[%d].Intent is empty", i)
		}
	}
}

func TestCmdHistoryNonexistentID(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdHistory(env.Store, []string{"test-zzzz"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for nonexistent ID")
	}
}

func nonEmptyLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
