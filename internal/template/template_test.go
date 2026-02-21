package template

import "testing"

func TestProcessNoDirectives(t *testing.T) {
	input := "line one\nline two\nline three"
	got := Process(input, nil)
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestProcessTrueCondition(t *testing.T) {
	input := "before\n<!-- IF workflow.agents == multi -->\nagent content\n<!-- END -->\nafter"
	got := Process(input, map[string]string{"workflow.agents": "multi"})
	want := "before\nagent content\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessFalseCondition(t *testing.T) {
	input := "before\n<!-- IF workflow.agents == multi -->\nagent content\n<!-- END -->\nafter"
	got := Process(input, map[string]string{})
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessFalseConditionWrongValue(t *testing.T) {
	input := "before\n<!-- IF workflow.agents == multi -->\nagent content\n<!-- END -->\nafter"
	got := Process(input, map[string]string{"workflow.agents": "single"})
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessDirectiveLinesStripped(t *testing.T) {
	input := "a\n<!-- IF x == y -->\nb\n<!-- END -->\nc"
	got := Process(input, map[string]string{"x": "y"})
	want := "a\nb\nc"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessMultipleBlocks(t *testing.T) {
	input := "start\n<!-- IF a == 1 -->\nfirst\n<!-- END -->\nmiddle\n<!-- IF b == 2 -->\nsecond\n<!-- END -->\nend"
	cfg := map[string]string{"a": "1"}
	got := Process(input, cfg)
	want := "start\nfirst\nmiddle\nend"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessNestedFalseBlock(t *testing.T) {
	input := "before\n<!-- IF outer == yes -->\nouter content\n<!-- IF inner == yes -->\ninner content\n<!-- END -->\n<!-- END -->\nafter"
	got := Process(input, map[string]string{})
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessNilConfig(t *testing.T) {
	input := "a\n<!-- IF x == y -->\nb\n<!-- END -->\nc"
	got := Process(input, nil)
	want := "a\nc"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessEmptyInput(t *testing.T) {
	got := Process("", nil)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestProcessUnmatchedEnd(t *testing.T) {
	input := "a\n<!-- END -->\nb"
	got := Process(input, nil)
	want := "a\nb"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessSingleLineComment(t *testing.T) {
	input := "before\n<!-- this is a comment -->\nafter"
	got := Process(input, nil)
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessMultiLineComment(t *testing.T) {
	input := "before\n<!--\nDesign notes:\n1. Be concise\n2. Be clear\n-->\nafter"
	got := Process(input, nil)
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessCommentDoesNotAffectDirectives(t *testing.T) {
	input := "<!--\nA comment block\n-->\nbefore\n<!-- IF x == y -->\nconditional\n<!-- END -->\nafter"
	got := Process(input, map[string]string{"x": "y"})
	want := "before\nconditional\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessMixedContent(t *testing.T) {
	input := "# Header\n\nPlain text.\n\n<!-- IF workflow.review == pr -->\n## Code Review\n\nPush branches.\n<!-- END -->\n\n## Footer\n"
	cfg := map[string]string{"workflow.review": "pr"}
	got := Process(input, cfg)
	want := "# Header\n\nPlain text.\n\n## Code Review\n\nPush branches.\n\n## Footer\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
