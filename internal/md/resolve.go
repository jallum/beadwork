package md

import (
	"regexp"
	"strings"

	"github.com/jallum/beadwork/internal/wrap"
)

// ANSI escape codes.
const (
	ansiBold          = "\033[1m"
	ansiDim           = "\033[2m"
	ansiStrikethrough = "\033[9m"
	ansiRed           = "\033[31m"
	ansiBrightRed     = "\033[91m"
	ansiGreen         = "\033[32m"
	ansiYellow        = "\033[33m"
	ansiCyan          = "\033[36m"
	ansiReset         = "\033[0m"
)

var statusIcons = map[string]string{
	"open":        "○",
	"closed":      "✓",
	"in_progress": "◐",
	"blocked":     "⊘",
	"deferred":    "❄",
}

var priorityColors = map[string]string{
	"0": ansiBrightRed,
	"1": ansiRed,
	"2": ansiYellow,
	"3": ansiCyan,
	"4": ansiDim,
}

var statusColors = map[string]string{
	"closed":      ansiGreen,
	"in_progress": ansiYellow,
	"blocked":     ansiRed,
	"deferred":    ansiDim,
}

// tokenRe matches all token forms: {kind:value} and {dep:kind:value}.
var tokenRe = regexp.MustCompile(`\{(status|p|id|type|check|dep):([^}]+)\}`)

// ResolveMarkdown transforms tokenized markdown into clean markdown for agents.
func ResolveMarkdown(s string) string {
	s = tokenRe.ReplaceAllStringFunc(s, func(tok string) string {
		m := tokenRe.FindStringSubmatch(tok)
		if m == nil {
			return tok
		}
		kind, val := m[1], m[2]
		switch kind {
		case "status":
			if icon, ok := statusIcons[val]; ok {
				return icon
			}
			return val
		case "p":
			return "P" + val
		case "id":
			return val
		case "type":
			return "[" + strings.ToUpper(val) + "]"
		case "check":
			if val == "done" {
				return "[x]"
			}
			return "[ ]"
		case "dep":
			return resolveDepMarkdown(val)
		}
		return tok
	})
	return unescape(s)
}

// ResolveTTY transforms tokenized markdown into ANSI-colored terminal output.
// Pipeline: resolve tokens → plain text with TTY extras → wrap → colorize.
func ResolveTTY(s string, width int) string {
	// Stage 1: Resolve tokens to plain TTY text (no ANSI yet).
	plain := tokenRe.ReplaceAllStringFunc(s, func(tok string) string {
		m := tokenRe.FindStringSubmatch(tok)
		if m == nil {
			return tok
		}
		kind, val := m[1], m[2]
		switch kind {
		case "status":
			if icon, ok := statusIcons[val]; ok {
				if _, hasColor := statusColors[val]; hasColor {
					return "\x01status:" + val + "\x02" + icon + "\x01end\x02"
				}
				return icon
			}
			return val
		case "p":
			return "\x01p:" + val + "\x02" + "● P" + val + "\x01end\x02"
		case "id":
			return "\x01id\x02" + val + "\x01end\x02"
		case "type":
			return "[" + strings.ToUpper(val) + "]"
		case "check":
			if val == "done" {
				return "\x01check:done\x02" + "✓" + "\x01end\x02"
			}
			return "[ ]"
		case "dep":
			return resolveDepTTYPlain(val)
		}
		return tok
	})
	plain = unescape(plain)

	// Stage 2: Wrap at terminal width.
	if width > 0 {
		plain = wrap.Text(plain, width)
	}

	// Stage 3: Colorize.
	return colorize(plain)
}

func resolveDepMarkdown(val string) string {
	parts := strings.SplitN(val, ":", 2)
	if len(parts) != 2 {
		return val
	}
	kind, id := parts[0], parts[1]
	switch kind {
	case "blocks":
		return "[blocks: " + id + "]"
	case "blocked_by":
		return "[blocked by: " + id + "]"
	}
	return val
}

func resolveDepTTYPlain(val string) string {
	parts := strings.SplitN(val, ":", 2)
	if len(parts) != 2 {
		return val
	}
	kind, id := parts[0], parts[1]
	switch kind {
	case "blocks":
		return "\x01dim\x02" + "[blocks: " + "\x01end\x02" + "\x01depid\x02" + id + "\x01end\x02" + "\x01dim\x02" + "]" + "\x01end\x02"
	case "blocked_by":
		return "\x01dim\x02" + "[blocked by: " + "\x01end\x02" + "\x01depid\x02" + id + "\x01end\x02" + "\x01dim\x02" + "]" + "\x01end\x02"
	}
	return val
}

// unescape restores escaped braces.
func unescape(s string) string {
	return strings.ReplaceAll(s, escChar+"{", "{")
}

// colorize applies ANSI coloring to the wrapped plain text.
// It processes inline color markers (\x01kind\x02) and markdown syntax.
var colorMarkerRe = regexp.MustCompile("\x01([^\x02]+)\x02")

func colorize(s string) string {
	var out strings.Builder
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(colorizeLine(line))
	}
	return out.String()
}

func colorizeLine(line string) string {
	// First resolve color markers from token resolution.
	line = resolveColorMarkers(line)

	// Then apply markdown syntax coloring.
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## "):
		return ansiBold + ansiCyan + line + ansiReset
	case strings.HasPrefix(trimmed, "- "):
		prefix := line[:strings.Index(line, "- ")]
		rest := line[strings.Index(line, "- ")+2:]
		return prefix + ansiCyan + "- " + ansiReset + styleInlineTTY(rest)
	default:
		return styleInlineTTY(line)
	}
}

// resolveColorMarkers replaces \x01kind\x02...\x01end\x02 with ANSI codes.
func resolveColorMarkers(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		start := strings.IndexByte(s[i:], '\x01')
		if start < 0 {
			out.WriteString(s[i:])
			break
		}
		out.WriteString(s[i : i+start])
		i += start + 1

		end := strings.IndexByte(s[i:], '\x02')
		if end < 0 {
			out.WriteByte('\x01')
			continue
		}
		kind := s[i : i+end]
		i += end + 1

		switch {
		case kind == "end":
			out.WriteString(ansiReset)
		case kind == "id":
			out.WriteString(ansiCyan)
		case kind == "dim":
			out.WriteString(ansiDim)
		case kind == "depid":
			out.WriteString(ansiRed)
		case kind == "check:done":
			out.WriteString(ansiGreen)
		case strings.HasPrefix(kind, "p:"):
			pLevel := kind[2:]
			if c, ok := priorityColors[pLevel]; ok {
				out.WriteString(c)
			}
		case strings.HasPrefix(kind, "status:"):
			status := kind[7:]
			if c, ok := statusColors[status]; ok {
				out.WriteString(c)
			}
		}
	}
	return out.String()
}

// styleInlineTTY applies markdown inline styling: **bold**, `code`, ~~strike~~.
func styleInlineTTY(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		// **bold**
		if i+1 < len(s) && s[i] == '*' && s[i+1] == '*' {
			end := strings.Index(s[i+2:], "**")
			if end >= 0 {
				out.WriteString(ansiBold)
				out.WriteString(s[i+2 : i+2+end])
				out.WriteString(ansiReset)
				i += 2 + end + 2
				continue
			}
		}
		// ~~strikethrough~~
		if i+1 < len(s) && s[i] == '~' && s[i+1] == '~' {
			end := strings.Index(s[i+2:], "~~")
			if end >= 0 {
				out.WriteString(ansiStrikethrough)
				out.WriteString(s[i+2 : i+2+end])
				out.WriteString(ansiReset)
				i += 2 + end + 2
				continue
			}
		}
		// `code`
		if s[i] == '`' {
			end := strings.IndexByte(s[i+1:], '`')
			if end >= 0 {
				out.WriteString(ansiDim)
				out.WriteString(ansiYellow)
				out.WriteString(s[i : i+1+end+1])
				out.WriteString(ansiReset)
				i += 1 + end + 1
				continue
			}
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}
