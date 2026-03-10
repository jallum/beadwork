// Package wrap provides text line-wrapping utilities.
package wrap

import (
	"regexp"
	"strings"
	"unicode"
)

// listPrefixRe matches a bullet or numbered-list prefix at the start of
// the body (after leading whitespace has been stripped). It captures the
// prefix so we can compute a content-aware continuation indent.
//
// Matches: "- ", "* ", "1. ", "12. ", "- ✓ ", "- ○ ", etc.
var listPrefixRe = regexp.MustCompile(`^(?:[-*]|[0-9]+\.)\s+(?:[✓○◐✗]\s+)?`)

// Text wraps s to fit within the given width. It preserves existing
// line breaks and detects leading whitespace on each input line,
// maintaining that indent when a line is wrapped. Words longer than
// the available width are broken to fit.
//
// For list items (bullets, numbered lists), continuation lines are
// indented to align with the content start after the list prefix.
//
// Dep annotations like [blocks: X] and [blocked by: X] are treated
// as non-breaking units.
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

		// Compute continuation indent: for list items, align to
		// content after the structural prefix.
		contIndent := indent
		if m := listPrefixRe.FindString(body); m != "" {
			contIndent = indent + strings.Repeat(" ", visibleLen(m))
		}

		wrapLine(&out, indent, contIndent, body, width)
	}

	return out.String()
}

// depAnnotationRe matches dependency annotations that should not be
// broken across lines, e.g. "[blocks: bw-abc]", "[blocked by: bw-xyz]".
// The pattern allows \x01...\x02 color markers anywhere inside the brackets
// so it works on both plain and TTY-resolved text.
var depAnnotationRe = regexp.MustCompile(`(?:\x01[^\x02]*\x02)*\[(?:blocks|blocked by):\s*(?:[^\]\x01]|\x01[^\x02]*\x02)*\](?:\x01[^\x02]*\x02)*`)

// tokenize splits body into words (like strings.Fields) but keeps dep
// annotations as single tokens even if they contain spaces.
func tokenize(body string) []string {
	var tokens []string
	for body != "" {
		body = strings.TrimLeft(body, " \t\r")
		if body == "" {
			break
		}
		// Check if the next token is a dep annotation.
		if loc := depAnnotationRe.FindStringIndex(body); loc != nil && loc[0] == 0 {
			tokens = append(tokens, body[:loc[1]])
			body = body[loc[1]:]
			continue
		}
		// Regular word: take until next whitespace.
		end := strings.IndexAny(body, " \t\r")
		if end == -1 {
			tokens = append(tokens, body)
			break
		}
		tokens = append(tokens, body[:end])
		body = body[end:]
	}
	return tokens
}

// wrapLine word-wraps body into out, prefixing the first output line
// with indent and continuation lines with contIndent. It breaks words
// that exceed the available space.
func wrapLine(out *strings.Builder, indent, contIndent, body string, width int) {
	indentLen := visibleLen(indent)
	contIndentLen := visibleLen(contIndent)
	avail := width - indentLen
	contAvail := width - contIndentLen
	if avail <= 0 {
		avail = 1
	}
	if contAvail <= 0 {
		contAvail = 1
	}

	words := tokenize(body)
	col := 0 // current column (relative to indent)
	first := true
	curAvail := avail

	for _, w := range words {
		wLen := visibleLen(w)

		if first {
			out.WriteString(indent)
			if wLen > curAvail {
				breakWord(out, indent, w, curAvail, true)
				col = visibleLen(lastSegment(w, curAvail))
			} else {
				out.WriteString(w)
				col = wLen
			}
			first = false
			continue
		}

		// Fits on current line with a leading space?
		if col+1+wLen <= curAvail {
			out.WriteByte(' ')
			out.WriteString(w)
			col += 1 + wLen
		} else {
			// Start a new continuation line.
			out.WriteByte('\n')
			out.WriteString(contIndent)
			curAvail = contAvail
			if wLen > curAvail {
				breakWord(out, contIndent, w, curAvail, true)
				col = visibleLen(lastSegment(w, curAvail))
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

// visibleLen returns the number of visible rune positions in s,
// excluding inline color markers (\x01...\x02 sequences).
func visibleLen(s string) int {
	n := 0
	inMarker := false
	for _, r := range s {
		switch {
		case r == '\x01':
			inMarker = true
		case r == '\x02':
			inMarker = false
		case !inMarker:
			n++
		}
	}
	return n
}
