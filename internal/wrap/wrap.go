// Package wrap provides text line-wrapping utilities.
package wrap

import (
	"strings"
	"unicode"
)

// Text wraps s to fit within the given width. It preserves existing
// line breaks and detects leading whitespace on each input line,
// maintaining that indent when a line is wrapped. Words longer than
// the available width are broken to fit.
//
// A width of zero or less disables wrapping and returns s unchanged.
func Text(s string, width int) string {
	if width <= 0 {
		return s
	}

	lines := strings.Split(s, "\n")
	var out strings.Builder

	for i, line := range lines {
		if i > 0 {
			out.WriteByte('\n')
		}

		// Preserve blank lines.
		if line == "" {
			continue
		}

		indent := leadingWhitespace(line)
		body := line[len(indent):]

		// If the body is empty (line is all whitespace), keep the original.
		if body == "" {
			out.WriteString(line)
			continue
		}

		wrapLine(&out, indent, body, width)
	}

	return out.String()
}

// wrapLine word-wraps body into out, prefixing every output line
// (including continuations) with indent. It breaks words that exceed
// the available space.
func wrapLine(out *strings.Builder, indent, body string, width int) {
	indentLen := visibleLen(indent)
	avail := width - indentLen
	if avail <= 0 {
		// Indent alone exceeds width; use at least 1 character.
		avail = 1
	}

	words := strings.Fields(body)
	col := 0 // current column (relative to indent)
	first := true

	for _, w := range words {
		wLen := visibleLen(w)

		if first {
			out.WriteString(indent)
			// Word wider than available space: break it.
			if wLen > avail {
				breakWord(out, indent, w, avail, true)
				col = visibleLen(lastSegment(w, avail))
			} else {
				out.WriteString(w)
				col = wLen
			}
			first = false
			continue
		}

		// Fits on current line with a leading space?
		if col+1+wLen <= avail {
			out.WriteByte(' ')
			out.WriteString(w)
			col += 1 + wLen
		} else {
			// Start a new continuation line.
			out.WriteByte('\n')
			out.WriteString(indent)
			if wLen > avail {
				breakWord(out, indent, w, avail, true)
				col = visibleLen(lastSegment(w, avail))
			} else {
				out.WriteString(w)
				col = wLen
			}
		}
	}
}

// breakWord writes word into out, splitting it across lines so that
// no line (after indent) exceeds avail visible characters.
// If firstLine is true the first chunk is written directly (indent is
// already in the buffer); subsequent chunks are preceded by
// newline+indent.
func breakWord(out *strings.Builder, indent string, word string, avail int, firstLine bool) {
	runes := []rune(word)
	pos := 0
	for pos < len(runes) {
		end := pos + avail
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[pos:end])

		if pos == 0 && firstLine {
			out.WriteString(chunk)
		} else {
			out.WriteByte('\n')
			out.WriteString(indent)
			out.WriteString(chunk)
		}
		pos = end
	}
}

// lastSegment returns the tail of word that would appear on the last
// line after breaking it into avail-wide chunks.
func lastSegment(word string, avail int) string {
	runes := []rune(word)
	rem := len(runes) % avail
	if rem == 0 && len(runes) > 0 {
		return string(runes[len(runes)-avail:])
	}
	return string(runes[len(runes)-rem:])
}

// leadingWhitespace returns the leading whitespace prefix of s.
func leadingWhitespace(s string) string {
	for i, r := range s {
		if !unicode.IsSpace(r) {
			return s[:i]
		}
	}
	return s
}

// visibleLen returns the number of visible (non-ANSI-escape) rune positions
// in s. For now it simply counts runes; this can be extended later to
// skip ANSI escape sequences.
func visibleLen(s string) int {
	return len([]rune(s))
}
