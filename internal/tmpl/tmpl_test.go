package tmpl

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecuteBasic(t *testing.T) {
	var buf bytes.Buffer
	err := Execute(&buf, "test", "Hello {{ .Name }}!", struct{ Name string }{"world"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "Hello world!" {
		t.Errorf("got %q", buf.String())
	}
}

func TestExecuteBwFunction(t *testing.T) {
	var buf bytes.Buffer
	bwFn := func(args ...string) string {
		return "[" + strings.Join(args, " ") + "]"
	}
	err := Execute(&buf, "test", "before\n{{ bw \"ready\" }}\nafter", nil, bwFn)
	if err != nil {
		t.Fatal(err)
	}
	want := "before\n[ready]\nafter"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestExecuteBwFunctionWithArgs(t *testing.T) {
	var buf bytes.Buffer
	bwFn := func(args ...string) string {
		return "(" + strings.Join(args, ",") + ")"
	}
	err := Execute(&buf, "test", `{{ bw "show" .ID "--only" "children" }}`, struct{ ID string }{"bw-123"}, bwFn)
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "(show,bw-123,--only,children)" {
		t.Errorf("got %q", buf.String())
	}
}

func TestExecuteConditional(t *testing.T) {
	src := `before
{{ if eq .Type "epic" }}epic content{{ end }}
after`
	var buf bytes.Buffer
	err := Execute(&buf, "test", src, struct{ Type string }{"epic"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "epic content") {
		t.Errorf("should contain epic content: %q", buf.String())
	}

	buf.Reset()
	err = Execute(&buf, "test", src, struct{ Type string }{"task"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "epic content") {
		t.Errorf("should not contain epic content: %q", buf.String())
	}
}

func TestExecuteCollapsesBlanks(t *testing.T) {
	var buf bytes.Buffer
	err := Execute(&buf, "test", "a\n\n\n\nb", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "a\n\nb" {
		t.Errorf("got %q", buf.String())
	}
}

func TestExecuteNilBwFn(t *testing.T) {
	var buf bytes.Buffer
	err := Execute(&buf, "test", "plain text", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "plain text" {
		t.Errorf("got %q", buf.String())
	}
}

func TestExecuteParseError(t *testing.T) {
	var buf bytes.Buffer
	err := Execute(&buf, "test", "{{ .Bad", nil, nil)
	if err == nil {
		t.Error("expected parse error")
	}
}
