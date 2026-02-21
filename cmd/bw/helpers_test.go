package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

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
}

func TestParseArgsEmpty(t *testing.T) {
	a, err := ParseArgs([]string{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.PosFirst() != "" {
		t.Errorf("PosFirst() = %q, want empty", a.PosFirst())
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

func TestFprintIssueWrapsDescription(t *testing.T) {
	iss := &issue.Issue{
		ID:          "test-wrap",
		Title:       "Wrap test",
		Status:      "open",
		Priority:    2,
		Type:        "task",
		Created:     "2024-01-15T12:00:00Z",
		Description: "This is a long description that should be wrapped when the terminal width is narrow enough to trigger wrapping behavior",
	}

	var buf bytes.Buffer
	w := ColorWriter(&buf, 40)
	fprintIssue(w, iss)

	// Strip ANSI codes for line-length checking.
	out := stripANSI(buf.String())

	// With width=40 and 2-char indent, lines wrap at 38 chars.
	inDesc := false
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "DESCRIPTION") {
			inDesc = true
			continue
		}
		if !inDesc || line == "" {
			continue
		}
		content := strings.TrimPrefix(line, "  ")
		if len(content) > 38 {
			t.Errorf("description line exceeds width: %d chars: %q", len(content), content)
		}
	}
}

func TestFprintCommentsWrapsText(t *testing.T) {
	iss := &issue.Issue{
		ID:       "test-cwrap",
		Title:    "Comment wrap test",
		Status:   "open",
		Priority: 2,
		Type:     "task",
		Created:  "2024-01-15T12:00:00Z",
		Comments: []issue.Comment{
			{
				Timestamp: "2024-01-16T10:00:00Z",
				Text:      "This is a long comment that should be wrapped when the terminal width is narrow enough to trigger wrapping",
			},
		},
	}

	var buf bytes.Buffer
	w := ColorWriter(&buf, 40)
	fprintComments(w, iss)

	// Strip ANSI codes for line-length checking.
	out := stripANSI(buf.String())

	// With width=40 and 4-char indent, content lines wrap at 36 chars.
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "    ") {
			continue
		}
		content := strings.TrimPrefix(line, "    ")
		if len(content) > 36 {
			t.Errorf("comment line exceeds width: %d chars: %q", len(content), content)
		}
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
	w := PlainWriter(&buf)
	fprintIssue(w, iss)
	fprintDescription(w, iss)
	out := buf.String()
	if !strings.Contains(out, "[BUG]") {
		t.Errorf("missing [BUG] type tag: %q", out)
	}
	// Deps are now rendered by fprintMap (not fprintIssue)
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

func TestFormatDepsEmpty(t *testing.T) {
	w := PlainWriter(&bytes.Buffer{})
	iss := &issue.Issue{
		Blocks:    []string{},
		BlockedBy: []string{},
	}
	if got := formatDeps(w, iss); got != "" {
		t.Errorf("formatDeps(empty) = %q, want empty", got)
	}
}

func TestFormatDepsBlocksOnly(t *testing.T) {
	w := PlainWriter(&bytes.Buffer{})
	iss := &issue.Issue{
		Blocks:    []string{"bw-abc", "bw-def"},
		BlockedBy: []string{},
	}
	got := formatDeps(w, iss)
	want := " [blocks: bw-abc, bw-def]"
	if got != want {
		t.Errorf("formatDeps = %q, want %q", got, want)
	}
}

func TestFormatDepsBlockedByOnly(t *testing.T) {
	w := PlainWriter(&bytes.Buffer{})
	iss := &issue.Issue{
		Blocks:    []string{},
		BlockedBy: []string{"bw-xyz"},
	}
	got := formatDeps(w, iss)
	want := " [blocked by: bw-xyz]"
	if got != want {
		t.Errorf("formatDeps = %q, want %q", got, want)
	}
}

func TestFormatDepsBoth(t *testing.T) {
	w := PlainWriter(&bytes.Buffer{})
	iss := &issue.Issue{
		Blocks:    []string{"bw-abc"},
		BlockedBy: []string{"bw-xyz"},
	}
	got := formatDeps(w, iss)
	want := " [blocks: bw-abc] [blocked by: bw-xyz]"
	if got != want {
		t.Errorf("formatDeps = %q, want %q", got, want)
	}
}

func TestFormatDepsNilSlices(t *testing.T) {
	w := PlainWriter(&bytes.Buffer{})
	iss := &issue.Issue{}
	if got := formatDeps(w, iss); got != "" {
		t.Errorf("formatDeps(nil slices) = %q, want empty", got)
	}
}

func TestGetInitializedWithDefaultPriority(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Set a default priority config
	env.Repo.SetConfig("default.priority", "2")
	env.Repo.Commit("config default.priority=2")

	store, err := getInitializedStore()
	if err != nil {
		t.Fatalf("getInitializedStore: %v", err)
	}
	if store.DefaultPriority == nil || *store.DefaultPriority != 2 {
		t.Errorf("DefaultPriority = %v, want 2", store.DefaultPriority)
	}
}

func TestGetInitializedReturnsError(t *testing.T) {
	// getRepo / getInitializedStore should return errors, not crash
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Verify it works when initialized
	s, err := getInitializedStore()
	if err != nil {
		t.Fatalf("getInitializedStore: %v", err)
	}
	if s == nil {
		t.Error("expected non-nil store")
	}
	_ = env
}

func TestRelativeTimeSince(t *testing.T) {
	now := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		t    time.Time
		want string
	}{
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "5m ago"},
		{now.Add(-2 * time.Hour), "2h ago"},
		{now.Add(-3 * 24 * time.Hour), "3d ago"},
		{now.Add(-45 * 24 * time.Hour), "1mo ago"},
		{now.Add(-90 * 24 * time.Hour), "3mo ago"},
		{now.Add(10 * time.Minute), "just now"}, // future
	}
	for _, tt := range tests {
		got := relativeTimeSince(tt.t, now)
		if got != tt.want {
			t.Errorf("relativeTimeSince(%v, now) = %q, want %q", tt.t, got, tt.want)
		}
	}
}

func TestRelativeTimeInvalidTimestamp(t *testing.T) {
	got := relativeTime("not-a-timestamp")
	if got != "not-a-timestamp" {
		t.Errorf("relativeTime(invalid) = %q, want raw string", got)
	}
}
