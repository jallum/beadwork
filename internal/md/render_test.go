package md

import (
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
)

func TestEscape(t *testing.T) {
	tests := []struct {
		input string
		want  string // should not contain raw unescaped {
	}{
		{"no braces", "no braces"},
		{"has {brace}", "has \x00{brace}"},
		{"{start} and {end}", "\x00{start} and \x00{end}"},
		{"empty", "empty"},
	}
	for _, tt := range tests {
		got := Escape(tt.input)
		if got != tt.want {
			t.Errorf("Escape(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIssueSummary(t *testing.T) {
	iss := &issue.Issue{
		ID:       "bw-abc",
		Title:    "Fix the widget",
		Status:   "open",
		Priority: 1,
		Type:     "task",
	}
	got := IssueSummary(iss)
	// Should start with # heading
	if !strings.HasPrefix(got, "# ") {
		t.Errorf("IssueSummary should start with '# ': got %q", got)
	}
	// Should contain status, id tokens
	if !strings.Contains(got, "{status:open}") {
		t.Errorf("should contain {status:open}: got %q", got)
	}
	if !strings.Contains(got, "{id:bw-abc}") {
		t.Errorf("should contain {id:bw-abc}: got %q", got)
	}
	// Task type should NOT have type tag
	if strings.Contains(got, "{type:") {
		t.Errorf("task type should not have type tag: got %q", got)
	}
	// Title should be escaped
	if !strings.Contains(got, "Fix the widget") {
		t.Errorf("should contain title: got %q", got)
	}
}

func TestIssueSummaryEpic(t *testing.T) {
	iss := &issue.Issue{
		ID:     "bw-xyz",
		Title:  "Big project",
		Status: "in_progress",
		Type:   "epic",
	}
	got := IssueSummary(iss)
	if !strings.Contains(got, "{type:epic}") {
		t.Errorf("epic should have type tag: got %q", got)
	}
}

func TestIssueSummaryWithParentAndLabels(t *testing.T) {
	iss := &issue.Issue{
		ID:     "bw-123",
		Title:  "Child task",
		Status: "open",
		Type:   "task",
		Parent: "bw-abc",
		Labels: []string{"v2", "frontend"},
	}
	got := IssueSummary(iss)
	if !strings.Contains(got, "Parent: bw-abc") {
		t.Errorf("should contain Parent line: got %q", got)
	}
	if !strings.Contains(got, "Labels: v2, frontend") {
		t.Errorf("should contain Labels line: got %q", got)
	}
}

func TestIssueSummaryNoParentNoLabels(t *testing.T) {
	iss := &issue.Issue{
		ID:     "bw-456",
		Title:  "Simple task",
		Status: "open",
		Type:   "task",
	}
	got := IssueSummary(iss)
	if strings.Contains(got, "Parent:") {
		t.Errorf("should not contain Parent line when empty: got %q", got)
	}
	if strings.Contains(got, "Labels:") {
		t.Errorf("should not contain Labels line when empty: got %q", got)
	}
}

func TestIssueOneLiner(t *testing.T) {
	iss := &issue.Issue{
		ID:       "bw-xyz",
		Title:    "Do something",
		Status:   "open",
		Priority: 2,
		Type:     "task",
	}
	got := IssueOneLiner(iss)
	if !strings.Contains(got, "{status:open}") {
		t.Errorf("should contain status token: got %q", got)
	}
	if !strings.Contains(got, "{id:bw-xyz}") {
		t.Errorf("should contain id token: got %q", got)
	}
	if !strings.Contains(got, "{p:2}") {
		t.Errorf("should contain priority token: got %q", got)
	}
	if !strings.Contains(got, "Do something") {
		t.Errorf("should contain title: got %q", got)
	}
}

func TestIssueOneLinerTypeTag(t *testing.T) {
	iss := &issue.Issue{
		ID:     "bw-e1",
		Title:  "Epic thing",
		Status: "open",
		Type:   "epic",
	}
	got := IssueOneLiner(iss)
	if !strings.Contains(got, "{type:epic}") {
		t.Errorf("non-task type should have type tag: got %q", got)
	}

	iss.Type = "task"
	got = IssueOneLiner(iss)
	if strings.Contains(got, "{type:") {
		t.Errorf("task should not have type tag: got %q", got)
	}
}

func TestIssueOneLinerWithDeps(t *testing.T) {
	iss := &issue.Issue{
		ID:        "bw-dep",
		Title:     "Has deps",
		Status:    "open",
		Priority:  1,
		Type:      "task",
		Blocks:    []string{"bw-other"},
		BlockedBy: []string{"bw-blocker"},
	}
	got := IssueOneLiner(iss)
	if !strings.Contains(got, "{dep:blocks:bw-other}") {
		t.Errorf("should contain blocks dep: got %q", got)
	}
	if !strings.Contains(got, "{dep:blocked_by:bw-blocker}") {
		t.Errorf("should contain blocked_by dep: got %q", got)
	}
}

func TestIssueOneLinerBlockedIcon(t *testing.T) {
	iss := &issue.Issue{
		ID:        "bw-blk",
		Title:     "Blocked task",
		Status:    "open",
		Priority:  2,
		Type:      "task",
		BlockedBy: []string{"bw-other"},
	}
	got := IssueOneLinerBlocked(iss, []string{"bw-other"})
	if !strings.Contains(got, "{status:blocked}") {
		t.Errorf("blocked issue should use blocked status icon: got %q", got)
	}
}

func TestDescription(t *testing.T) {
	got := Description("Hello world\nSecond line")
	// Should be escaped
	resolved := ResolveMarkdown(got)
	if resolved != "Hello world\nSecond line" {
		t.Errorf("Description should preserve text: got %q", resolved)
	}
}

func TestDescriptionWithBraces(t *testing.T) {
	got := Description("config {key} = {value}")
	resolved := ResolveMarkdown(got)
	if resolved != "config {key} = {value}" {
		t.Errorf("braces should survive round-trip: got %q", resolved)
	}
}

func TestChildren(t *testing.T) {
	children := []*issue.Issue{
		{ID: "bw-c1", Title: "First child", Status: "closed", Priority: 1, Type: "task",
			Blocks: []string{"bw-c2"}},
		{ID: "bw-c2", Title: "Second child", Status: "open", Priority: 2, Type: "task",
			BlockedBy: []string{"bw-c1"}},
	}
	got := Children(children)
	if !strings.Contains(got, "## CHILDREN") {
		t.Errorf("should have CHILDREN header: got %q", got)
	}
	if !strings.Contains(got, "{check:done}") {
		t.Errorf("closed child should have check:done: got %q", got)
	}
	if !strings.Contains(got, "{check:open}") {
		t.Errorf("open child should have check:open: got %q", got)
	}
	if !strings.Contains(got, "{id:bw-c1}") {
		t.Errorf("should contain child ID token: got %q", got)
	}
	if !strings.Contains(got, "{dep:blocks:bw-c2}") {
		t.Errorf("should contain dep token: got %q", got)
	}
}

func TestChildrenEmpty(t *testing.T) {
	got := Children(nil)
	if got != "" {
		t.Errorf("Children(nil) should be empty: got %q", got)
	}
}

func TestBlockedBy(t *testing.T) {
	blockers := []*issue.Issue{
		{ID: "bw-b1", Title: "Blocker one", Status: "open", Priority: 1, Type: "task"},
	}
	got := BlockedBy(blockers)
	if !strings.Contains(got, "## BLOCKED BY") {
		t.Errorf("should have BLOCKED BY header: got %q", got)
	}
	if !strings.Contains(got, "{id:bw-b1}") {
		t.Errorf("should contain blocker ID: got %q", got)
	}
}

func TestBlockedByEmpty(t *testing.T) {
	got := BlockedBy(nil)
	if got != "" {
		t.Errorf("BlockedBy(nil) should be empty: got %q", got)
	}
}

func TestUnblocks(t *testing.T) {
	deps := []*issue.Issue{
		{ID: "bw-u1", Title: "Unblocked one", Status: "open", Priority: 2, Type: "task"},
	}
	got := Unblocks(deps)
	if !strings.Contains(got, "## UNBLOCKS") {
		t.Errorf("should have UNBLOCKS header: got %q", got)
	}
	if !strings.Contains(got, "{id:bw-u1}") {
		t.Errorf("should contain unblocked ID: got %q", got)
	}
}

func TestUnblocksEmpty(t *testing.T) {
	got := Unblocks(nil)
	if got != "" {
		t.Errorf("Unblocks(nil) should be empty: got %q", got)
	}
}

func TestComments(t *testing.T) {
	comments := []issue.Comment{
		{Timestamp: "2024-01-15T10:00:00Z", Author: "alice", Text: "First comment"},
		{Timestamp: "2024-01-16T11:00:00Z", Text: "Anonymous comment"},
	}
	got := Comments(comments)
	if !strings.Contains(got, "## COMMENTS") {
		t.Errorf("should have COMMENTS header: got %q", got)
	}
	if !strings.Contains(got, "alice") {
		t.Errorf("should contain author: got %q", got)
	}
	if !strings.Contains(got, "First comment") {
		t.Errorf("should contain comment text: got %q", got)
	}
}

func TestCommentsEmpty(t *testing.T) {
	got := Comments(nil)
	if got != "" {
		t.Errorf("Comments(nil) should be empty: got %q", got)
	}
}

func TestCommentsWithBraces(t *testing.T) {
	comments := []issue.Comment{
		{Timestamp: "2024-01-15T10:00:00Z", Text: "config {foo} bar"},
	}
	got := Comments(comments)
	resolved := ResolveMarkdown(got)
	if !strings.Contains(resolved, "{foo}") {
		t.Errorf("braces in comments should be preserved: got %q", resolved)
	}
}

func TestFormatDeps(t *testing.T) {
	iss := &issue.Issue{
		Blocks:    []string{"bw-a", "bw-b"},
		BlockedBy: []string{"bw-c"},
	}
	got := FormatDeps(iss)
	// Multiple IDs of the same kind should be grouped into one token.
	if !strings.Contains(got, "{dep:blocks:bw-a,bw-b}") {
		t.Errorf("should group blocks into single token: got %q", got)
	}
	if !strings.Contains(got, "{dep:blocked_by:bw-c}") {
		t.Errorf("should contain blocked_by token: got %q", got)
	}
}

func TestFormatDepsEmpty(t *testing.T) {
	iss := &issue.Issue{}
	got := FormatDeps(iss)
	if got != "" {
		t.Errorf("FormatDeps(empty) should be empty: got %q", got)
	}
}

func TestFormatDepsFiltersClosed(t *testing.T) {
	iss := &issue.Issue{
		BlockedBy: []string{"bw-open", "bw-closed"},
	}
	closedSet := map[string]bool{"bw-closed": true}
	got := FormatDepsFiltered(iss, closedSet)
	if strings.Contains(got, "bw-closed") {
		t.Errorf("should filter closed blockers: got %q", got)
	}
	if !strings.Contains(got, "bw-open") {
		t.Errorf("should keep open blockers: got %q", got)
	}
}

func TestFullRoundTrip(t *testing.T) {
	iss := &issue.Issue{
		ID:          "bw-trip",
		Title:       "Round trip {test}",
		Status:      "in_progress",
		Priority:    1,
		Type:        "epic",
		Parent:      "bw-parent",
		Labels:      []string{"v1"},
		Description: "A description with {braces}",
	}
	summary := IssueSummary(iss)
	desc := Description(iss.Description)
	full := summary + "\n" + desc

	md := ResolveMarkdown(full)
	if !strings.Contains(md, "◐") {
		t.Errorf("resolved markdown should contain in_progress icon: got %q", md)
	}
	if !strings.Contains(md, "bw-trip") {
		t.Errorf("resolved markdown should contain ID: got %q", md)
	}
	if !strings.Contains(md, "[EPIC]") {
		t.Errorf("resolved markdown should contain type tag: got %q", md)
	}
	if !strings.Contains(md, "{braces}") {
		t.Errorf("escaped braces should survive: got %q", md)
	}
	if !strings.Contains(md, "{test}") {
		t.Errorf("escaped braces in title should survive: got %q", md)
	}

	tty := ResolveTTY(full, 80)
	if !strings.Contains(tty, "bw-trip") {
		t.Errorf("TTY should contain ID: got %q", tty)
	}
}
