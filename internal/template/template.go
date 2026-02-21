package template

import "strings"

// Process evaluates conditional directives in text against config values.
// Directives are HTML comments: <!-- IF key == value --> and <!-- END -->.
// Lines containing only a directive are stripped from output.
// Content between IF/END is included only when the config matches.
func Process(text string, config map[string]string) string {
	lines := strings.Split(text, "\n")
	var out []string
	skipDepth := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

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

		if trimmed == "<!-- END -->" {
			if skipDepth > 0 {
				skipDepth--
			}
			continue
		}

		if skipDepth == 0 {
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n")
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
