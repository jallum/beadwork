package md

import (
	"strings"
	"testing"
)

func TestResolveMarkdownStatusTokens(t *testing.T) {
	tests := []struct {
		token string
		want  string
	}{
		{"{status:open}", "○"},
		{"{status:closed}", "✓"},
		{"{status:in_progress}", "◐"},
		{"{status:blocked}", "⊘"},
		{"{status:deferred}", "❄"},
	}
	for _, tt := range tests {
		got := ResolveMarkdown(tt.token)
		if got != tt.want {
			t.Errorf("ResolveMarkdown(%q) = %q, want %q", tt.token, got, tt.want)
		}
	}
}

func TestResolveMarkdownPriorityTokens(t *testing.T) {
	for p := 0; p <= 4; p++ {
		token := "{p:" + string(rune('0'+p)) + "}"
		want := "P" + string(rune('0'+p))
		got := ResolveMarkdown(token)
		if got != want {
			t.Errorf("ResolveMarkdown(%q) = %q, want %q", token, got, want)
		}
	}
}

func TestResolveMarkdownIDToken(t *testing.T) {
	got := ResolveMarkdown("{id:bw-xyz}")
	if got != "bw-xyz" {
		t.Errorf("ResolveMarkdown({id:bw-xyz}) = %q, want bw-xyz", got)
	}
}

func TestResolveMarkdownTypeToken(t *testing.T) {
	tests := []struct {
		token string
		want  string
	}{
		{"{type:epic}", "[EPIC]"},
		{"{type:bug}", "[BUG]"},
	}
	for _, tt := range tests {
		got := ResolveMarkdown(tt.token)
		if got != tt.want {
			t.Errorf("ResolveMarkdown(%q) = %q, want %q", tt.token, got, tt.want)
		}
	}
}

func TestResolveMarkdownCheckTokens(t *testing.T) {
	if got := ResolveMarkdown("{check:done}"); got != "[x]" {
		t.Errorf("got %q, want [x]", got)
	}
	if got := ResolveMarkdown("{check:open}"); got != "[ ]" {
		t.Errorf("got %q, want [ ]", got)
	}
}

func TestResolveMarkdownDepTokens(t *testing.T) {
	got := ResolveMarkdown("{dep:blocks:bw-abc}")
	if got != "[blocks: bw-abc]" {
		t.Errorf("single: got %q, want [blocks: bw-abc]", got)
	}
	got = ResolveMarkdown("{dep:blocked_by:bw-xyz}")
	if got != "[blocked by: bw-xyz]" {
		t.Errorf("single: got %q, want [blocked by: bw-xyz]", got)
	}
	got = ResolveMarkdown("{dep:blocked_by:bw-a,bw-b}")
	if got != "[blocked by: bw-a, bw-b]" {
		t.Errorf("multi: got %q, want [blocked by: bw-a, bw-b]", got)
	}
}

func TestResolveMarkdownMultipleTokens(t *testing.T) {
	input := "{status:open} {id:bw-abc} {p:1} Fix the bug"
	got := ResolveMarkdown(input)
	want := "○ bw-abc P1 Fix the bug"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveMarkdownEscapedBraces(t *testing.T) {
	input := Escape("user text with {curly} braces")
	got := ResolveMarkdown(input)
	if !strings.Contains(got, "{curly}") {
		t.Errorf("escaped braces should be restored: got %q", got)
	}
}

func TestResolveTTYStatusTokens(t *testing.T) {
	// TTY output should contain the icon and ANSI codes
	got := ResolveTTY("{status:closed}", 80)
	if !strings.Contains(got, "✓") {
		t.Errorf("ResolveTTY status:closed should contain ✓: got %q", got)
	}
	// Should contain ANSI escape (green for closed)
	if !strings.Contains(got, "\033[") {
		t.Errorf("ResolveTTY should contain ANSI codes: got %q", got)
	}
}

func TestResolveTTYPriorityTokens(t *testing.T) {
	got := ResolveTTY("{p:1}", 80)
	// TTY adds ● dot before priority
	if !strings.Contains(got, "●") {
		t.Errorf("ResolveTTY priority should have ● dot: got %q", got)
	}
	if !strings.Contains(got, "P1") {
		t.Errorf("ResolveTTY priority should contain P1: got %q", got)
	}
}

func TestResolveTTYIDToken(t *testing.T) {
	got := ResolveTTY("{id:bw-xyz}", 80)
	if !strings.Contains(got, "bw-xyz") {
		t.Errorf("ResolveTTY should contain ID text: got %q", got)
	}
	if !strings.Contains(got, "\033[") {
		t.Errorf("ResolveTTY ID should be ANSI colored: got %q", got)
	}
}

func TestResolveTTYWrapping(t *testing.T) {
	long := "{id:bw-abc} " + strings.Repeat("word ", 30)
	got := ResolveTTY(long, 40)
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Errorf("expected wrapping at width 40, got %d lines", len(lines))
	}
}

func TestResolveTTYHeadings(t *testing.T) {
	got := ResolveTTY("# My Heading", 80)
	// Should contain bold+cyan ANSI for heading
	if !strings.Contains(got, "\033[1m") {
		t.Errorf("heading should be bold: got %q", got)
	}
	if !strings.Contains(got, "\033[36m") {
		t.Errorf("heading should be cyan: got %q", got)
	}
}

func TestResolveTTYBold(t *testing.T) {
	got := ResolveTTY("some **bold** text", 80)
	if !strings.Contains(got, "\033[1m") {
		t.Errorf("**bold** should have ANSI bold: got %q", got)
	}
}

func TestResolveTTYInlineCode(t *testing.T) {
	got := ResolveTTY("use `bw ready` now", 80)
	if !strings.Contains(got, "\033[2m") || !strings.Contains(got, "\033[33m") {
		t.Errorf("`code` should be dim+yellow: got %q", got)
	}
}

func TestResolveTTYStrikethrough(t *testing.T) {
	got := ResolveTTY("~~struck~~", 80)
	if !strings.Contains(got, "\033[9m") {
		t.Errorf("~~strike~~ should have ANSI strikethrough: got %q", got)
	}
}

func TestResolveTTYDepTokens(t *testing.T) {
	got := ResolveTTY("{dep:blocks:bw-abc}", 80)
	if !strings.Contains(got, "blocks:") {
		t.Errorf("dep token should contain label: got %q", got)
	}
	if !strings.Contains(got, "bw-abc") {
		t.Errorf("dep token should contain ID: got %q", got)
	}
	// Multiple IDs should resolve into a single bracket group.
	got = ResolveTTY("{dep:blocked_by:bw-a,bw-b}", 80)
	if !strings.Contains(got, "blocked by:") {
		t.Errorf("multi dep should contain label: got %q", got)
	}
	if !strings.Contains(got, "bw-a") || !strings.Contains(got, "bw-b") {
		t.Errorf("multi dep should contain both IDs: got %q", got)
	}
}

func TestResolveTTYZeroWidth(t *testing.T) {
	// Width 0 should disable wrapping
	long := strings.Repeat("word ", 50)
	got := ResolveTTY(long, 0)
	if strings.Contains(got, "\n") {
		t.Errorf("width 0 should not wrap: got %q", got)
	}
}

func TestEscapeAndResolve(t *testing.T) {
	input := "config {foo} and {bar}"
	escaped := Escape(input)
	// Escaped text should differ from input (escape prefix added)
	if escaped == input {
		t.Errorf("Escape should modify input: got %q", escaped)
	}
	// After resolving, original text restored
	got := ResolveMarkdown(escaped)
	if got != input {
		t.Errorf("round-trip failed: got %q, want %q", got, input)
	}
}

func TestResolveTTYBullets(t *testing.T) {
	got := ResolveTTY("- item one", 80)
	// Bullet should be cyan
	if !strings.Contains(got, "\033[36m") {
		t.Errorf("bullet should have cyan ANSI: got %q", got)
	}
}

func TestResolveTTYCheckboxes(t *testing.T) {
	got := ResolveTTY("- {check:done} done item", 80)
	if !strings.Contains(got, "✓") || !strings.Contains(got, "\033[32m") {
		t.Errorf("check:done in TTY should be green ✓: got %q", got)
	}

	got = ResolveTTY("- {check:open} open item", 80)
	if !strings.Contains(got, "☐") {
		t.Errorf("check:open in TTY should contain ☐: got %q", got)
	}
	if !strings.Contains(got, "\033[2m") {
		t.Errorf("check:open in TTY should be dim: got %q", got)
	}
}

func TestResolveTTYFencedHeadingNotStyled(t *testing.T) {
	input := "before\n```\n# not a heading\n```\nafter"
	got := ResolveTTY(input, 80)
	// The heading inside the fence should NOT have bold+cyan
	lines := strings.Split(got, "\n")
	fencedLine := lines[2] // "# not a heading"
	if strings.Contains(fencedLine, "\033[1m") && strings.Contains(fencedLine, "\033[36m") {
		t.Errorf("heading inside fence should not be styled bold+cyan: got %q", fencedLine)
	}
	// Should be dim
	if !strings.Contains(fencedLine, "\033[2m") {
		t.Errorf("fenced content should be dim: got %q", fencedLine)
	}
}

func TestResolveTTYFencedBoldNotStyled(t *testing.T) {
	input := "```\n**not bold**\n```"
	got := ResolveTTY(input, 80)
	lines := strings.Split(got, "\n")
	fencedLine := lines[1]
	if strings.Contains(fencedLine, "\033[1m") {
		t.Errorf("bold inside fence should not be styled: got %q", fencedLine)
	}
	if !strings.Contains(fencedLine, "**not bold**") {
		t.Errorf("bold markers should be preserved literally: got %q", fencedLine)
	}
}

func TestResolveTTYFencedBulletNotStyled(t *testing.T) {
	input := "```\n- not a bullet\n```"
	got := ResolveTTY(input, 80)
	lines := strings.Split(got, "\n")
	fencedLine := lines[1]
	// Should NOT have cyan bullet styling
	if strings.Contains(fencedLine, "\033[36m") {
		t.Errorf("bullet inside fence should not have cyan: got %q", fencedLine)
	}
}

func TestResolveTTYFenceResumesNormalStyling(t *testing.T) {
	input := "```\ncode\n```\n# real heading"
	got := ResolveTTY(input, 80)
	lines := strings.Split(got, "\n")
	headingLine := lines[3]
	if !strings.Contains(headingLine, "\033[1m") || !strings.Contains(headingLine, "\033[36m") {
		t.Errorf("heading after fence close should be bold+cyan: got %q", headingLine)
	}
}

func TestResolveTTYNestedFences(t *testing.T) {
	input := "````\n```\nstill inside\n```\n````"
	got := ResolveTTY(input, 80)
	lines := strings.Split(got, "\n")
	// "still inside" (line 2) should be dim (inside outer fence)
	if !strings.Contains(lines[2], "\033[2m") {
		t.Errorf("content inside nested fence should be dim: got %q", lines[2])
	}
	// Inner ``` (line 1) should NOT close the outer fence
	// So line 3 (inner closing ```) should also be dim
	if !strings.Contains(lines[3], "\033[2m") {
		t.Errorf("inner closing fence should still be dim (inside outer): got %q", lines[3])
	}
}

func TestResolveTTYFenceClosingNeedsMatchingBackticks(t *testing.T) {
	input := "````\ncode\n```\nstill code\n````"
	got := ResolveTTY(input, 80)
	lines := strings.Split(got, "\n")
	// "still code" should be dim — ``` didn't close the ```` fence
	if !strings.Contains(lines[3], "\033[2m") {
		t.Errorf("line after short fence should still be dim: got %q", lines[3])
	}
}

func TestResolveTTYFenceClosingNoInfoString(t *testing.T) {
	input := "```go\ncode\n```go\nstill code\n```"
	got := ResolveTTY(input, 80)
	lines := strings.Split(got, "\n")
	// "```go" on line 2 has an info string — should NOT close the fence
	// "still code" should be dim
	if !strings.Contains(lines[3], "\033[2m") {
		t.Errorf("line after fence with info string should still be dim: got %q", lines[3])
	}
}

func TestResolveTTYFenceDelimitersDim(t *testing.T) {
	input := "```\ncode\n```"
	got := ResolveTTY(input, 80)
	lines := strings.Split(got, "\n")
	// Both fence delimiters should be dim
	if !strings.Contains(lines[0], "\033[2m") {
		t.Errorf("opening fence should be dim: got %q", lines[0])
	}
	if !strings.Contains(lines[2], "\033[2m") {
		t.Errorf("closing fence should be dim: got %q", lines[2])
	}
}
