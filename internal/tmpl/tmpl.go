// Package tmpl wraps Go text/template for beadwork prompt rendering.
package tmpl

import (
	"io"
	"strings"
	"text/template"
)

// Execute parses src as a Go text/template, executes it with data, and writes
// the result to w. The optional bwFn is registered as a "bw" template function
// that runs beadwork commands and returns their output inline.
func Execute(w io.Writer, name, src string, data any, bwFn func(args ...string) string) error {
	funcMap := template.FuncMap{}
	if bwFn != nil {
		funcMap["bw"] = bwFn
	}

	tmpl, err := template.New(name).Funcs(funcMap).Parse(src)
	if err != nil {
		return err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	out := collapseBlankLines(buf.String())
	_, err = io.WriteString(w, out)
	return err
}

// collapseBlankLines reduces runs of 3+ consecutive newlines to 2 (one blank line).
func collapseBlankLines(s string) string {
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}
