package main

import (
	"bytes"
	"io"
	"strings"

	"github.com/jallum/beadwork/internal/md"
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
	Strikethrough
)

var styleCode = map[Style]string{
	Bold:          "\033[1m",
	Dim:           "\033[2m",
	Red:           "\033[31m",
	BrightRed:     "\033[91m",
	Yellow:        "\033[33m",
	Cyan:          "\033[36m",
	Green:         "\033[32m",
	Strikethrough: "\033[9m",
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
	// IsTTY returns true if the writer targets a terminal (color-capable).
	IsTTY() bool
	// IsRaw returns true if the writer should emit tokenized text without resolution.
	IsRaw() bool
}

// writer is the single concrete implementation of Writer.
type writer struct {
	out   io.Writer
	color bool
	raw   bool
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

func (w *writer) IsTTY() bool { return w.color }
func (w *writer) IsRaw() bool { return w.raw }

func (w *writer) rebuildPrefix() {
	total := 0
	for _, n := range w.stack {
		total += n
	}
	w.pfx = strings.Repeat(" ", total)
}

// resolvingWriter wraps a Writer and resolves tokenized markdown in Write().
// TTY mode resolves to ANSI-styled text, markdown mode resolves to plain
// markdown, and raw mode passes tokens through unchanged.
type resolvingWriter struct {
	Writer
}

func (rw *resolvingWriter) Write(p []byte) (int, error) {
	s := string(p)
	if !rw.IsRaw() {
		if rw.IsTTY() {
			s = md.ResolveTTY(s, rw.Width())
		} else {
			s = md.ResolveMarkdown(s)
		}
	}
	_, err := rw.Writer.Write([]byte(s))
	// Report the original length so fmt.Fprint doesn't think it short-wrote.
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// ResolvingWriter wraps a Writer so that all output is automatically resolved
// from tokenized markdown according to the writer's mode.
func ResolvingWriter(w Writer) Writer {
	return &resolvingWriter{Writer: w}
}

// PlainWriter returns a Writer that resolves tokenized markdown to plain text.
func PlainWriter(out io.Writer) Writer {
	return ResolvingWriter(plainWriter(out))
}

// ColorWriter returns a Writer that resolves tokenized markdown with ANSI styling.
func ColorWriter(out io.Writer, width int) Writer {
	return ResolvingWriter(colorWriter(out, width))
}

// RawWriter returns a Writer that passes tokenized text through without resolution.
func RawWriter(out io.Writer) Writer {
	return &writer{out: out, raw: true, bol: true}
}

// TokenWriter returns a non-resolving plain writer for capturing tokenized
// output (e.g. template expansion) before final resolution by an outer writer.
func TokenWriter(out io.Writer) Writer {
	return plainWriter(out)
}

func plainWriter(out io.Writer) Writer {
	return &writer{out: out, bol: true}
}

func colorWriter(out io.Writer, width int) Writer {
	return &writer{out: out, color: true, width: width, bol: true}
}
