package issue

// StatusInfo pairs a status name with its display icon.
type StatusInfo struct {
	Name string
	Icon string
}

var Statuses = []StatusInfo{
	{"open", "○"},
	{"in_progress", "◐"},
	{"deferred", "❄"},
	{"closed", "✓"},
}

func StatusNames() []string {
	names := make([]string, len(Statuses))
	for i, s := range Statuses {
		names[i] = s.Name
	}
	return names
}

func StatusIcon(status string) string {
	for _, s := range Statuses {
		if s.Name == status {
			return s.Icon
		}
	}
	return "?"
}

// Priority definitions

var PriorityColors = map[int]string{
	0: "\033[91m", // bright red (P0 — critical)
	1: "\033[31m", // red
	2: "\033[33m", // yellow
	3: "\033[36m", // cyan
	4: "\033[2m",  // dim
}

const ColorReset = "\033[0m"

func PriorityDot(priority int) string {
	color, ok := PriorityColors[priority]
	if !ok {
		color = ""
	}
	return color + "●" + ColorReset
}
