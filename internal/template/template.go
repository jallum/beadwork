package template

import (
	"io"
	"strings"
)

// Process evaluates conditional directives and section markers in text,
// writing the result to w. Config values control IF/END conditionals.
// Section markers (<!-- NAME -->) invoke the corresponding callback from
// sections; unregistered markers are stripped. Plain HTML comments are
// stripped entirely.
func Process(w io.Writer, text string, config map[string]string, sections map[string]func(io.Writer)) {
	lines := strings.Split(text, "\n")
	skipDepth := 0
	inComment := false
	first := true

	emit := func(s string) {
		if !first {
			io.WriteString(w, "\n")
		}
		io.WriteString(w, s)
		first = false
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Continue multi-line comment.
		if inComment {
			if strings.Contains(trimmed, "-->") {
				inComment = false
			}
			continue
		}

		// IF directive.
		if cond, ok := parseIF(trimmed); ok {
			if skipDepth > 0 {
				skipDepth++
				continue
			}
			if !evalCondition(cond, config) {
				skipDepth = 1
			}
			continue
		}

		// END directive.
		if trimmed == "<!-- END -->" {
			if skipDepth > 0 {
				skipDepth--
			}
			continue
		}

		// Section marker.
		if name, ok := parseSection(trimmed); ok {
			if skipDepth == 0 {
				if fn := sections[name]; fn != nil {
					if !first {
						io.WriteString(w, "\n")
					}
					fn(w)
					first = false
				}
			}
			continue
		}

		// Plain HTML comment (single or multi-line).
		if strings.HasPrefix(trimmed, "<!--") {
			if !strings.Contains(trimmed, "-->") {
				inComment = true
			}
			continue
		}

		// Content line.
		if skipDepth == 0 {
			emit(line)
		}
	}
}

// parseIF checks if a line is an IF directive and returns the condition.
func parseIF(trimmed string) (string, bool) {
	after, ok := strings.CutPrefix(trimmed, "<!-- IF ")
	if !ok {
		return "", false
	}
	cond, ok := strings.CutSuffix(after, " -->")
	if !ok {
		return "", false
	}
	return cond, true
}

// parseSection checks if a line is a section marker (<!-- NAME -->)
// where NAME is a single word with no spaces.
func parseSection(trimmed string) (string, bool) {
	after, ok := strings.CutPrefix(trimmed, "<!-- ")
	if !ok {
		return "", false
	}
	name, ok := strings.CutSuffix(after, " -->")
	if !ok {
		return "", false
	}
	if strings.ContainsRune(name, ' ') {
		return "", false
	}
	return name, true
}

// evalCondition checks "key == value" against the config map.
func evalCondition(cond string, config map[string]string) bool {
	parts := strings.SplitN(cond, "==", 2)
	if len(parts) != 2 {
		return false
	}
	key := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])
	return config[key] == val
}
