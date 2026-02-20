package main

import (
	"io"
	"strings"
)

// Style represents an ANSI text style.
type Style int

const (
	Bold Style = iota
	Dim
	Red
	BrightRed
	Yellow
	Cyan
)

var styleCode = map[Style]string{
	Bold:      "\033[1m",
	Dim:       "\033[2m",
	Red:       "\033[31m",
	BrightRed: "\033[91m",
	Yellow:    "\033[33m",
	Cyan:      "\033[36m",
}

const reset = "\033[0m"

var priorityStyles = map[int]Style{
	0: BrightRed,
	1: Red,
	2: Yellow,
	3: Cyan,
	4: Dim,
}

// PriorityStyle returns the Style for a given priority level.
func PriorityStyle(priority int) Style {
	if s, ok := priorityStyles[priority]; ok {
		return s
	}
	return Dim
}

// Writer extends io.Writer with terminal styling.
// plainWriter returns strings unchanged; colorWriter wraps with ANSI codes.
type Writer interface {
	io.Writer
	Style(s string, styles ...Style) string
}

type plainWriter struct{ io.Writer }

func (w *plainWriter) Style(s string, _ ...Style) string { return s }

type colorWriter struct{ io.Writer }

func (w *colorWriter) Style(s string, styles ...Style) string {
	if len(styles) == 0 {
		return s
	}
	var b strings.Builder
	for _, st := range styles {
		if code, ok := styleCode[st]; ok {
			b.WriteString(code)
		}
	}
	b.WriteString(s)
	b.WriteString(reset)
	return b.String()
}

// PlainWriter returns a Writer that ignores all styling.
func PlainWriter(w io.Writer) Writer { return &plainWriter{w} }

// ColorWriter returns a Writer that applies ANSI styling.
func ColorWriter(w io.Writer) Writer { return &colorWriter{w} }
