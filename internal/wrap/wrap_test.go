package wrap

import (
	"strings"
	"testing"
)

func TestText_EmptyInput(t *testing.T) {
	if got := Text("", 80); got != "" {
		t.Errorf("Text(%q, 80) = %q, want %q", "", got, "")
	}
}

func TestText_ZeroWidth(t *testing.T) {
	s := "hello world"
	if got := Text(s, 0); got != s {
		t.Errorf("Text(%q, 0) = %q, want %q", s, got, s)
	}
}

func TestText_NegativeWidth(t *testing.T) {
	s := "hello world"
	if got := Text(s, -5); got != s {
		t.Errorf("Text(%q, -5) = %q, want %q", s, got, s)
	}
}

func TestText_ShortLine(t *testing.T) {
	s := "short"
	if got := Text(s, 80); got != s {
		t.Errorf("Text(%q, 80) = %q, want %q", s, got, s)
	}
}

func TestText_ExactWidth(t *testing.T) {
	s := "hello"
	if got := Text(s, 5); got != s {
		t.Errorf("Text(%q, 5) = %q, want %q", s, got, s)
	}
}

func TestText_BasicWrap(t *testing.T) {
	input := "the quick brown fox jumps over the lazy dog"
	got := Text(input, 20)
	want := "the quick brown fox\njumps over the lazy\ndog"
	if got != want {
		t.Errorf("Text(..., 20):\n got: %q\nwant: %q", got, want)
	}
}

func TestText_PreservesExistingLineBreaks(t *testing.T) {
	input := "first line\nsecond line\nthird line"
	got := Text(input, 80)
	if got != input {
		t.Errorf("Text(..., 80):\n got: %q\nwant: %q", got, input)
	}
}

func TestText_PreservesBlankLines(t *testing.T) {
	input := "first\n\nsecond"
	got := Text(input, 80)
	if got != input {
		t.Errorf("Text(..., 80):\n got: %q\nwant: %q", got, input)
	}
}

func TestText_MultipleBlankLines(t *testing.T) {
	input := "first\n\n\nsecond"
	got := Text(input, 80)
	if got != input {
		t.Errorf("Text(..., 80):\n got: %q\nwant: %q", got, input)
	}
}

func TestText_IndentedText(t *testing.T) {
	input := "  the quick brown fox jumps over the lazy dog"
	got := Text(input, 20)
	// With 2-char indent, available width is 18.
	want := "  the quick brown\n  fox jumps over the\n  lazy dog"
	if got != want {
		t.Errorf("Text(..., 20):\n got: %q\nwant: %q", got, want)
	}
}

func TestText_DeeperIndent(t *testing.T) {
	input := "    deeply indented text that needs wrapping here"
	got := Text(input, 25)
	// 4-char indent, 21 available.
	want := "    deeply indented text\n    that needs wrapping\n    here"
	if got != want {
		t.Errorf("Text(..., 25):\n got: %q\nwant: %q", got, want)
	}
}

func TestText_MixedIndentLevels(t *testing.T) {
	input := "no indent\n  two indent\n    four indent"
	got := Text(input, 80)
	if got != input {
		t.Errorf("expected no change for short lines:\n got: %q\nwant: %q", got, input)
	}
}

func TestText_TabIndent(t *testing.T) {
	input := "\tindented with tab"
	got := Text(input, 80)
	if got != input {
		t.Errorf("Text(..., 80):\n got: %q\nwant: %q", got, input)
	}
}

func TestText_LongWord(t *testing.T) {
	input := "abcdefghij"
	got := Text(input, 5)
	want := "abcde\nfghij"
	if got != want {
		t.Errorf("Text(%q, 5):\n got: %q\nwant: %q", input, got, want)
	}
}

func TestText_LongWordWithPrefix(t *testing.T) {
	input := "hello abcdefghijklmno world"
	got := Text(input, 10)
	want := "hello\nabcdefghij\nklmno\nworld"
	if got != want {
		t.Errorf("Text(..., 10):\n got: %q\nwant: %q", got, want)
	}
}

func TestText_LongWordIndented(t *testing.T) {
	input := "  abcdefghijkl"
	got := Text(input, 8)
	// 2-char indent, 6 available per line.
	want := "  abcdef\n  ghijkl"
	if got != want {
		t.Errorf("Text(%q, 8):\n got: %q\nwant: %q", input, got, want)
	}
}

func TestText_OnlySpaces(t *testing.T) {
	input := "   "
	got := Text(input, 80)
	if got != input {
		t.Errorf("Text(%q, 80) = %q, want %q", input, got, input)
	}
}

func TestText_MultipleSpacesBetweenWords(t *testing.T) {
	// Fields splitting normalizes spaces, which is acceptable for wrapping.
	input := "hello   world"
	got := Text(input, 80)
	want := "hello world"
	if got != want {
		t.Errorf("Text(%q, 80) = %q, want %q", input, got, want)
	}
}

func TestText_WidthOne(t *testing.T) {
	input := "ab cd"
	got := Text(input, 1)
	want := "a\nb\nc\nd"
	if got != want {
		t.Errorf("Text(%q, 1):\n got: %q\nwant: %q", input, got, want)
	}
}

func TestText_MultilineWrap(t *testing.T) {
	input := "first line is short\nthis second line is much longer and should be wrapped at the limit"
	got := Text(input, 30)
	want := "first line is short\nthis second line is much\nlonger and should be wrapped\nat the limit"
	if got != want {
		t.Errorf("Text(..., 30):\n got: %q\nwant: %q", got, want)
	}
}

func TestText_IndentExceedsWidth(t *testing.T) {
	// When the indent alone exceeds width, we still make progress
	// (avail falls back to 1 character per line).
	input := "      abc"
	got := Text(input, 4)
	want := "      a\n      b\n      c"
	if got != want {
		t.Errorf("Text(%q, 4):\n got: %q\nwant: %q", input, got, want)
	}
}

func TestText_Unicode(t *testing.T) {
	input := "hello world"
	got := Text(input, 7)
	want := "hello\nworld"
	if got != want {
		t.Errorf("Text(%q, 7):\n got: %q\nwant: %q", input, got, want)
	}
}

func TestText_TrailingNewline(t *testing.T) {
	input := "hello world\n"
	got := Text(input, 80)
	if got != input {
		t.Errorf("Text(%q, 80) = %q, want %q", input, got, input)
	}
}

func TestText_DescriptionBlock(t *testing.T) {
	// Simulates a bw show description block (2-space indent).
	lines := []string{
		"  Add a reusable line-wrapping function that wraps long text to a configurable width.",
		"  Should handle indented blocks correctly.",
	}
	input := strings.Join(lines, "\n")
	got := Text(input, 40)

	// Every output line should start with "  " (the original indent).
	for i, line := range strings.Split(got, "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("line %d missing indent: %q", i, line)
		}
		if len(line) > 40 {
			t.Errorf("line %d exceeds width 40 (%d chars): %q", i, len(line), line)
		}
	}
}

func TestText_CommentBlock(t *testing.T) {
	// Simulates a bw show comment block (4-space indent).
	input := "    Implemented the wrapping utility in internal/wrap with full test coverage."
	got := Text(input, 40)
	for i, line := range strings.Split(got, "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "    ") {
			t.Errorf("line %d missing 4-space indent: %q", i, line)
		}
		if len(line) > 40 {
			t.Errorf("line %d exceeds width 40 (%d chars): %q", i, len(line), line)
		}
	}
}

func TestLeadingWhitespace(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"hello", ""},
		{"  hello", "  "},
		{"\thello", "\t"},
		{"  \t hello", "  \t "},
		{"   ", "   "},
	}
	for _, tt := range tests {
		got := leadingWhitespace(tt.in)
		if got != tt.want {
			t.Errorf("leadingWhitespace(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestVisibleLen(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"hello", 5},
		{"  hi", 4},
	}
	for _, tt := range tests {
		got := visibleLen(tt.in)
		if got != tt.want {
			t.Errorf("visibleLen(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
