// Package tmpl wraps Go text/template for beadwork prompt rendering.
package tmpl

import (
	"io"
	"strings"
	"text/template"
)

// Template is a parsed prompt with Beadwork's output formatting.
type Template struct {
	tmpl *template.Template
}

// Parse parses src as a Go text/template. The optional bwFn is registered as a
// "bw" template function that runs beadwork commands and returns their output.
func Parse(name, src string, bwFn func(args ...string) string) (*Template, error) {
	funcMap := template.FuncMap{}
	if bwFn != nil {
		funcMap["bw"] = bwFn
	}

	tmpl, err := template.New(name).Funcs(funcMap).Parse(src)
	if err != nil {
		return nil, err
	}
	return &Template{tmpl: tmpl}, nil
}

// Execute renders the parsed template and collapses excess blank lines.
func (t *Template) Execute(w io.Writer, data any) error {
	var buf strings.Builder
	if err := t.tmpl.Execute(&buf, data); err != nil {
		return err
	}

	out := collapseBlankLines(buf.String())
	_, err := io.WriteString(w, out)
	return err
}

// Execute parses src and renders it with data.
func Execute(w io.Writer, name, src string, data any, bwFn func(args ...string) string) error {
	t, err := Parse(name, src, bwFn)
	if err != nil {
		return err
	}
	return t.Execute(w, data)
}

// collapseBlankLines reduces runs of 3+ consecutive newlines to 2 (one blank line).
func collapseBlankLines(s string) string {
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}
