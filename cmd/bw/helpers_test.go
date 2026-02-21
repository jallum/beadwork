package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestParseArgsBooleans(t *testing.T) {
	a, err := ParseArgs([]string{"--json", "--all", "positional"}, nil, []string{"--json", "--all"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.Bool("--json") {
		t.Error("expected --json to be true")
	}
	if !a.Bool("--all") {
		t.Error("expected --all to be true")
	}
	if a.Bool("--missing") {
		t.Error("expected --missing to be false")
	}
	if !a.JSON() {
		t.Error("expected JSON() to be true")
	}
}

func TestParseArgsValueFlags(t *testing.T) {
	a, err := ParseArgs([]string{"--status", "open", "--priority", "1"}, []string{"--status", "--priority"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.String("--status") != "open" {
		t.Errorf("status = %q, want open", a.String("--status"))
	}
	if a.Int("--priority") != 1 {
		t.Errorf("priority = %d, want 1", a.Int("--priority"))
	}
	if !a.Has("--status") {
		t.Error("expected Has(--status) to be true")
	}
	if a.Has("--missing") {
		t.Error("expected Has(--missing) to be false")
	}
}

func TestParseArgsAliases(t *testing.T) {
	a, err := ParseArgs([]string{"-p", "2", "-t", "bug", "-a", "alice"}, []string{"--priority", "--type", "--assignee"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.Int("--priority") != 2 {
		t.Errorf("priority = %d, want 2", a.Int("--priority"))
	}
	if a.String("--type") != "bug" {
		t.Errorf("type = %q, want bug", a.String("--type"))
	}
	if a.String("--assignee") != "alice" {
		t.Errorf("assignee = %q, want alice", a.String("--assignee"))
	}
}

func TestParseArgsPositionals(t *testing.T) {
	a, err := ParseArgs([]string{"my", "title", "here", "--json"}, nil, []string{"--json"})
	if err != nil {
		t.Fatal(err)
	}
	pos := a.Pos()
	if len(pos) != 3 {
		t.Fatalf("pos = %v, want 3 items", pos)
	}
	if a.PosFirst() != "my" {
		t.Errorf("PosFirst() = %q, want my", a.PosFirst())
	}
	if a.PosJoined() != "my title here" {
		t.Errorf("PosJoined() = %q", a.PosJoined())
	}
}

func TestParseArgsEmpty(t *testing.T) {
	a, err := ParseArgs([]string{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.PosFirst() != "" {
		t.Errorf("PosFirst() = %q, want empty", a.PosFirst())
	}
	if a.PosJoined() != "" {
		t.Errorf("PosJoined() = %q, want empty", a.PosJoined())
	}
	if a.JSON() {
		t.Error("expected JSON() to be false")
	}
}

func TestParseArgsValueFlagAtEnd(t *testing.T) {
	// Value flag with no following token should be silently ignored
	a, err := ParseArgs([]string{"--status"}, []string{"--status"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.Has("--status") {
		t.Error("expected Has(--status) to be false when no value follows")
	}
	if a.String("--status") != "" {
		t.Errorf("status = %q, want empty", a.String("--status"))
	}
}

func TestParseArgsIntErr(t *testing.T) {
	a, parseErr := ParseArgs([]string{"--priority", "abc"}, []string{"--priority"}, nil)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	_, set, err := a.IntErr("--priority")
	if !set {
		t.Error("expected set to be true")
	}
	if err == nil {
		t.Error("expected error for non-numeric priority")
	}

	// Missing flag
	_, set, err = a.IntErr("--missing")
	if set || err != nil {
		t.Errorf("expected set=false, err=nil for missing flag, got set=%v, err=%v", set, err)
	}

	// Valid int
	a2, parseErr := ParseArgs([]string{"--priority", "3"}, []string{"--priority"}, nil)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	n, set, err := a2.IntErr("--priority")
	if !set || err != nil || n != 3 {
		t.Errorf("expected (3, true, nil), got (%d, %v, %v)", n, set, err)
	}
}

func TestParseArgsMixedFlagsAndPositionals(t *testing.T) {
	a, err := ParseArgs([]string{"id-123", "--json", "--status", "open"}, []string{"--status"}, []string{"--json"})
	if err != nil {
		t.Fatal(err)
	}
	if a.PosFirst() != "id-123" {
		t.Errorf("PosFirst() = %q, want id-123", a.PosFirst())
	}
	if !a.JSON() {
		t.Error("expected JSON() to be true")
	}
	if a.String("--status") != "open" {
		t.Errorf("status = %q, want open", a.String("--status"))
	}
}

func TestHasFlag(t *testing.T) {
	tests := []struct {
		args []string
		flag string
		want bool
	}{
		{[]string{"--yes"}, "--yes", true},
		{[]string{"show", "id", "--json"}, "--json", true},
		{[]string{"show", "id"}, "--json", false},
		{[]string{}, "--json", false},
	}
	for _, tt := range tests {
		got := hasFlag(tt.args, tt.flag)
		if got != tt.want {
			t.Errorf("hasFlag(%v, %q) = %v, want %v", tt.args, tt.flag, got, tt.want)
		}
	}
}

func TestFprintJSON(t *testing.T) {
	var buf bytes.Buffer
	fprintJSON(PlainWriter(&buf), map[string]string{"key": "value"})
	if !strings.Contains(buf.String(), `"key": "value"`) {
		t.Errorf("output = %q", buf.String())
	}
}

func TestFprintIssue(t *testing.T) {
	iss := &issue.Issue{
		ID:       "test-1234",
		Title:    "Test issue",
		Status:   "open",
		Priority: 2,
		Type:     "task",
		Assignee: "alice",
		Created:  "2024-01-15T12:00:00Z",
		Labels:   []string{"bug"},
	}

	var buf bytes.Buffer
	fprintIssue(PlainWriter(&buf), iss)
	out := buf.String()
	if !strings.Contains(out, "test-1234") {
		t.Errorf("missing ID in output: %q", out)
	}
	if !strings.Contains(out, "Test issue") {
		t.Errorf("missing title in output: %q", out)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("missing assignee in output: %q", out)
	}
	if !strings.Contains(out, "Labels: bug") {
		t.Errorf("missing labels in output: %q", out)
	}
}

func TestFprintIssueFull(t *testing.T) {
	iss := &issue.Issue{
		ID:          "test-5678",
		Title:       "Full issue",
		Status:      "in_progress",
		Priority:    1,
		Type:        "bug",
		Assignee:    "",
		Created:     "2024-06-15T12:00:00Z",
		Labels:      []string{},
		Blocks:      []string{"test-aaaa"},
		BlockedBy:   []string{"test-bbbb"},
		Parent:      "test-cccc",
		Description: "Line one\nLine two",
	}

	var buf bytes.Buffer
	fprintIssue(PlainWriter(&buf), iss)
	out := buf.String()
	if !strings.Contains(out, "[BUG]") {
		t.Errorf("missing [BUG] type tag: %q", out)
	}
	// Deps are now rendered by fprintDeps (not fprintIssue)
	if !strings.Contains(out, "Parent: test-cccc") {
		t.Errorf("missing Parent in output: %q", out)
	}
	if !strings.Contains(out, "DESCRIPTION") {
		t.Errorf("missing DESCRIPTION in output: %q", out)
	}
	if !strings.Contains(out, "Line one") || !strings.Contains(out, "Line two") {
		t.Errorf("missing description lines in output: %q", out)
	}
	// No assignee → should show "—"
	if !strings.Contains(out, "\u2014") {
		t.Errorf("missing dash for empty assignee: %q", out)
	}
}

func TestFprintIssueTypeTag(t *testing.T) {
	iss := &issue.Issue{
		ID:       "test-9999",
		Title:    "Bug report",
		Status:   "open",
		Priority: 1,
		Type:     "bug",
		Created:  "2024-01-15T12:00:00Z",
	}

	var buf bytes.Buffer
	fprintIssue(PlainWriter(&buf), iss)
	out := buf.String()
	if !strings.Contains(out, "[BUG]") {
		t.Errorf("should contain [BUG] tag: %q", out)
	}

	// Task type should NOT have a tag
	iss.Type = "task"
	buf.Reset()
	fprintIssue(PlainWriter(&buf), iss)
	out = buf.String()
	if strings.Contains(out, "[TASK]") {
		t.Errorf("task should not have type tag: %q", out)
	}
}

func TestFprintIssueDateLine(t *testing.T) {
	iss := &issue.Issue{
		ID:        "test-date",
		Title:     "Date test",
		Status:    "open",
		Priority:  2,
		Type:      "task",
		Created:   "2024-01-15T12:00:00Z",
		UpdatedAt: "2024-02-20T14:00:00Z",
	}

	var buf bytes.Buffer
	fprintIssue(PlainWriter(&buf), iss)
	out := buf.String()
	if !strings.Contains(out, "Created: 2024-01-15") {
		t.Errorf("missing Created date: %q", out)
	}
	if !strings.Contains(out, "Updated: 2024-02-20") {
		t.Errorf("missing Updated date: %q", out)
	}

	// Deferred should be on same line
	iss.DeferUntil = "2027-06-01"
	buf.Reset()
	fprintIssue(PlainWriter(&buf), iss)
	out = buf.String()
	if !strings.Contains(out, "Deferred: 2027-06-01") {
		t.Errorf("missing Deferred date: %q", out)
	}
}

func TestFprintIssueCloseReason(t *testing.T) {
	iss := &issue.Issue{
		ID:          "test-closed",
		Title:       "Closed issue",
		Status:      "closed",
		Priority:    2,
		Type:        "task",
		Created:     "2024-01-15T12:00:00Z",
		ClosedAt:    "2024-03-01T10:00:00Z",
		CloseReason: "duplicate",
	}

	var buf bytes.Buffer
	fprintIssue(PlainWriter(&buf), iss)
	out := buf.String()
	if !strings.Contains(out, "Close reason: duplicate") {
		t.Errorf("should contain close reason: %q", out)
	}
}

func TestGetInitializedWithDefaultPriority(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Set a default priority config
	env.Repo.SetConfig("default.priority", "2")
	env.Repo.Commit("config default.priority=2")

	_, store, err := getInitializedRepo()
	if err != nil {
		t.Fatalf("getInitializedRepo: %v", err)
	}
	if store.DefaultPriority == nil || *store.DefaultPriority != 2 {
		t.Errorf("DefaultPriority = %v, want 2", store.DefaultPriority)
	}
}

func TestGetInitializedReturnsError(t *testing.T) {
	// getRepo / getInitializedRepo should return errors, not crash
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Verify it works when initialized
	r, s, err := getInitializedRepo()
	if err != nil {
		t.Fatalf("getInitializedRepo: %v", err)
	}
	if r == nil || s == nil {
		t.Error("expected non-nil repo and store")
	}
}
