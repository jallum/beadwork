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

