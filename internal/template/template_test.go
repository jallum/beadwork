package template

import (
	"bytes"
	"io"
	"testing"
)

func mapResolver(m map[string]string) func(string) string {
	if m == nil {
		return nil
	}
	return func(key string) string { return m[key] }
}

func process(t *testing.T, input string, config map[string]string, sections map[string]func(io.Writer)) string {
	t.Helper()
	var buf bytes.Buffer
	Process(&buf, input, mapResolver(config), sections)
	return buf.String()
}

func processWithCommands(t *testing.T, input string, config map[string]string, sections map[string]func(io.Writer), cmdFn func([]string, io.Writer)) string {
	t.Helper()
	var buf bytes.Buffer
	ProcessWithCommands(&buf, input, mapResolver(config), sections, cmdFn)
	return buf.String()
}

func TestProcessNoDirectives(t *testing.T) {
	input := "line one\nline two\nline three"
	got := process(t, input, nil, nil)
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestProcessTrueCondition(t *testing.T) {
	input := "before\n<!-- IF workflow.agents == multi -->\nagent content\n<!-- END -->\nafter"
	got := process(t, input, map[string]string{"workflow.agents": "multi"}, nil)
	want := "before\nagent content\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessFalseCondition(t *testing.T) {
	input := "before\n<!-- IF workflow.agents == multi -->\nagent content\n<!-- END -->\nafter"
	got := process(t, input, map[string]string{}, nil)
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessFalseConditionWrongValue(t *testing.T) {
	input := "before\n<!-- IF workflow.agents == multi -->\nagent content\n<!-- END -->\nafter"
	got := process(t, input, map[string]string{"workflow.agents": "single"}, nil)
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessDirectiveLinesStripped(t *testing.T) {
	input := "a\n<!-- IF x == y -->\nb\n<!-- END -->\nc"
	got := process(t, input, map[string]string{"x": "y"}, nil)
	want := "a\nb\nc"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessMultipleBlocks(t *testing.T) {
	input := "start\n<!-- IF a == 1 -->\nfirst\n<!-- END -->\nmiddle\n<!-- IF b == 2 -->\nsecond\n<!-- END -->\nend"
	cfg := map[string]string{"a": "1"}
	got := process(t, input, cfg, nil)
	want := "start\nfirst\nmiddle\nend"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessNestedFalseBlock(t *testing.T) {
	input := "before\n<!-- IF outer == yes -->\nouter content\n<!-- IF inner == yes -->\ninner content\n<!-- END -->\n<!-- END -->\nafter"
	got := process(t, input, map[string]string{}, nil)
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessNilConfig(t *testing.T) {
	input := "a\n<!-- IF x == y -->\nb\n<!-- END -->\nc"
	got := process(t, input, nil, nil)
	want := "a\nc"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessEmptyInput(t *testing.T) {
	got := process(t, "", nil, nil)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestProcessUnmatchedEnd(t *testing.T) {
	input := "a\n<!-- END -->\nb"
	got := process(t, input, nil, nil)
	want := "a\nb"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessSingleLineComment(t *testing.T) {
	input := "before\n<!-- this is a comment -->\nafter"
	got := process(t, input, nil, nil)
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessMultiLineComment(t *testing.T) {
	input := "before\n<!--\nDesign notes:\n1. Be concise\n2. Be clear\n-->\nafter"
	got := process(t, input, nil, nil)
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessCommentDoesNotAffectDirectives(t *testing.T) {
	input := "<!--\nA comment block\n-->\nbefore\n<!-- IF x == y -->\nconditional\n<!-- END -->\nafter"
	got := process(t, input, map[string]string{"x": "y"}, nil)
	want := "before\nconditional\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessMixedContent(t *testing.T) {
	input := "# Header\n\nPlain text.\n\n<!-- IF workflow.review == pr -->\n## Code Review\n\nPush branches.\n<!-- END -->\n\n## Footer\n"
	cfg := map[string]string{"workflow.review": "pr"}
	got := process(t, input, cfg, nil)
	want := "# Header\n\nPlain text.\n\n## Code Review\n\nPush branches.\n\n## Footer\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Section marker tests

func TestProcessSectionCallback(t *testing.T) {
	input := "before\n<!-- STATE -->\nafter"
	sections := map[string]func(io.Writer){
		"STATE": func(w io.Writer) {
			io.WriteString(w, "injected")
		},
	}
	got := process(t, input, nil, sections)
	want := "before\ninjected\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessSectionCallbackWithDirectives(t *testing.T) {
	input := "header\n<!-- IF x == y -->\nconditional\n<!-- END -->\n<!-- STATE -->\nfooter"
	sections := map[string]func(io.Writer){
		"STATE": func(w io.Writer) {
			io.WriteString(w, "injected state")
		},
	}
	got := process(t, input, map[string]string{"x": "y"}, sections)
	want := "header\nconditional\ninjected state\nfooter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessSectionSkippedInFalseBlock(t *testing.T) {
	input := "before\n<!-- IF x == y -->\n<!-- STATE -->\n<!-- END -->\nafter"
	called := false
	sections := map[string]func(io.Writer){
		"STATE": func(w io.Writer) {
			called = true
			io.WriteString(w, "injected")
		},
	}
	got := process(t, input, map[string]string{}, sections)
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if called {
		t.Error("section callback should not be called inside skipped block")
	}
}

func TestProcessUnregisteredSectionStripped(t *testing.T) {
	input := "before\n<!-- STATE -->\nafter"
	got := process(t, input, nil, nil)
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessSectionAtStart(t *testing.T) {
	input := "<!-- STATE -->\nafter"
	sections := map[string]func(io.Writer){
		"STATE": func(w io.Writer) {
			io.WriteString(w, "first")
		},
	}
	got := process(t, input, nil, sections)
	want := "first\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessSectionAtEnd(t *testing.T) {
	input := "before\n<!-- STATE -->"
	sections := map[string]func(io.Writer){
		"STATE": func(w io.Writer) {
			io.WriteString(w, "last")
		},
	}
	got := process(t, input, nil, sections)
	want := "before\nlast"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessMultipleSections(t *testing.T) {
	input := "top\n<!-- HEADER -->\nmiddle\n<!-- FOOTER -->\nbottom"
	sections := map[string]func(io.Writer){
		"HEADER": func(w io.Writer) { io.WriteString(w, "=head=") },
		"FOOTER": func(w io.Writer) { io.WriteString(w, "=foot=") },
	}
	got := process(t, input, nil, sections)
	want := "top\n=head=\nmiddle\n=foot=\nbottom"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Blank line collapsing tests

func TestProcessCollapsesConsecutiveBlanks(t *testing.T) {
	input := "before\n\n\n\nafter"
	got := process(t, input, nil, nil)
	want := "before\n\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestProcessCollapsesBlanksBetweenFalseBlocks(t *testing.T) {
	input := "header\n<!-- IF type == epic -->\nepic\n<!-- END -->\n\n<!-- IF type == task -->\ntask\n<!-- END -->\nfooter"
	got := process(t, input, map[string]string{"type": "bug"}, nil)
	want := "header\n\nfooter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Command marker tests

func TestProcessCommand(t *testing.T) {
	input := "before\n<!-- bw ready -->\nafter"
	var gotArgs []string
	cmdFn := func(args []string, w io.Writer) {
		gotArgs = args
		io.WriteString(w, "ready-output")
	}
	got := processWithCommands(t, input, nil, nil, cmdFn)
	want := "before\nready-output\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if len(gotArgs) != 1 || gotArgs[0] != "ready" {
		t.Errorf("args = %v, want [ready]", gotArgs)
	}
}

func TestProcessCommandWithArgs(t *testing.T) {
	input := "before\n<!-- bw list --status in_progress -->\nafter"
	var gotArgs []string
	cmdFn := func(args []string, w io.Writer) {
		gotArgs = args
		io.WriteString(w, "list-output")
	}
	got := processWithCommands(t, input, nil, nil, cmdFn)
	want := "before\nlist-output\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	wantArgs := []string{"list", "--status", "in_progress"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", gotArgs, wantArgs)
	}
	for i, a := range wantArgs {
		if gotArgs[i] != a {
			t.Errorf("args[%d] = %q, want %q", i, gotArgs[i], a)
		}
	}
}

func TestProcessCommandSkippedInFalseBlock(t *testing.T) {
	input := "before\n<!-- IF x == y -->\n<!-- bw ready -->\n<!-- END -->\nafter"
	called := false
	cmdFn := func(args []string, w io.Writer) {
		called = true
	}
	got := processWithCommands(t, input, nil, nil, cmdFn)
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if called {
		t.Error("command callback should not be called inside skipped block")
	}
}

func TestProcessCommandNilCallback(t *testing.T) {
	input := "before\n<!-- bw ready -->\nafter"
	got := processWithCommands(t, input, nil, nil, nil)
	want := "before\nafter"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
