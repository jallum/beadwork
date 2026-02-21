package main

import (
	"bytes"
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
	Green
)

var styleCode = map[Style]string{
	Bold:      "\033[1m",
	Dim:       "\033[2m",
	Red:       "\033[31m",
	BrightRed: "\033[91m",
	Yellow:    "\033[33m",
	Cyan:      "\033[36m",
	Green:     "\033[32m",
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

// Writer extends io.Writer with terminal styling, width awareness,
// and an indent stack. Push/Pop control indentation; Write automatically
// prefixes each new line with the current indent.
type Writer interface {
	io.Writer
	Style(s string, styles ...Style) string
	// Width returns the available width (terminal width minus current indent),
	// or 0 if unavailable (disables wrapping).
	Width() int
	// Push adds n spaces to the current indent level.
	Push(n int)
	// Pop removes the most recent indent level.
	Pop()
	// ClearLine returns a carriage return that also clears to end of line
	// on color-capable terminals; plain writers return a bare "\r".
	ClearLine() string
}

// writer is the single concrete implementation of Writer.
type writer struct {
	out   io.Writer
	color bool
	width int    // terminal width, 0 = no wrapping
	stack []int  // indent stack
	pfx   string // cached prefix (sum of stack as spaces)
	bol   bool   // at beginning of line
}

func (w *writer) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		if w.bol && w.pfx != "" {
			if _, err := io.WriteString(w.out, w.pfx); err != nil {
				return written, err
			}
			w.bol = false
		}
		idx := bytes.IndexByte(p, '\n')
		if idx < 0 {
			n, err := w.out.Write(p)
			written += n
			w.bol = false
			return written, err
		}
		n, err := w.out.Write(p[:idx+1])
		written += n
		if err != nil {
			return written, err
		}
		p = p[idx+1:]
		w.bol = true
	}
	return written, nil
}

func (w *writer) Style(s string, styles ...Style) string {
	if !w.color || len(styles) == 0 {
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

func (w *writer) Width() int {
	if w.width == 0 {
		return 0
	}
	avail := w.width - len(w.pfx)
	if avail < 1 {
		return 1
	}
	return avail
}

func (w *writer) Push(n int) {
	w.stack = append(w.stack, n)
	w.rebuildPrefix()
}

func (w *writer) Pop() {
	if len(w.stack) > 0 {
		w.stack = w.stack[:len(w.stack)-1]
		w.rebuildPrefix()
	}
}

func (w *writer) ClearLine() string {
	if w.color {
		return "\r\033[K"
	}
	return "\r"
}

func (w *writer) rebuildPrefix() {
	total := 0
	for _, n := range w.stack {
		total += n
	}
	w.pfx = strings.Repeat(" ", total)
}

// PlainWriter returns a Writer that ignores all styling and has no width (no wrapping).
func PlainWriter(out io.Writer) Writer {
	return &writer{out: out, bol: true}
}

// ColorWriter returns a Writer that applies ANSI styling with the given terminal width.
func ColorWriter(out io.Writer, width int) Writer {
	return &writer{out: out, color: true, width: width, bol: true}
}
