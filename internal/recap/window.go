package recap

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Window represents a time range for filtering events.
type Window struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Label string    `json:"label"`
}

// ParseWindow parses a time window from tokens. Accepted tokens:
//   - "today"      — start of today in local TZ to now
//   - "yesterday"  — start of yesterday to start of today (local TZ)
//   - "week"       — start of this week (Monday) to now
//   - "24h"        — 24 hours ago to now
//   - "7d"         — 7 days ago to now
//
// The --since flag overrides with an explicit RFC3339 or date start.
// now is passed explicitly so tests can be deterministic.
func ParseWindow(tokens []string, since string, now time.Time) (Window, error) {
	loc := now.Location()

	if since != "" {
		start, err := parseSinceDate(since, loc)
		if err != nil {
			return Window{}, fmt.Errorf("invalid --since: %w", err)
		}
		return Window{Start: start, End: now, Label: "since " + since}, nil
	}

	token := "24h"
	if len(tokens) > 0 {
		token = strings.ToLower(strings.Join(tokens, " "))
	}

	switch token {
	case "today":
		start := startOfDay(now, loc)
		return Window{Start: start, End: now, Label: "today"}, nil
	case "yesterday":
		todayStart := startOfDay(now, loc)
		yesterdayStart := todayStart.AddDate(0, 0, -1)
		return Window{Start: yesterdayStart, End: todayStart, Label: "yesterday"}, nil
	case "week", "this week":
		start := startOfWeek(now, loc)
		return Window{Start: start, End: now, Label: "this week"}, nil
	}

	if d, ok := parseDurationToken(token); ok {
		return Window{Start: now.Add(-d), End: now, Label: "last " + formatDuration(d)}, nil
	}

	return Window{}, fmt.Errorf("unknown window %q (expected: today, yesterday, week, or duration like 15m, 1h, 24h, 7d, 2w)", token)
}

// durationPartRe matches a single duration component like "15m", "3h", "2d", "1w".
var durationPartRe = regexp.MustCompile(`^(\d+)(m|h|d|w)$`)

// parseDurationToken parses tokens like "15m", "1h", "3h30m", "24h", "7d", "2w".
// Returns (duration, true) on success. Accepts Go's time.ParseDuration syntax
// for minute/hour combinations (so "3h30m" works) and extends with d/w for
// days and weeks.
func parseDurationToken(s string) (time.Duration, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}

	// First try Go's built-in parser (handles combinations like 3h30m).
	if d, err := time.ParseDuration(s); err == nil && d > 0 {
		return d, true
	}

	// Fall back to our extended parser that supports d (day) and w (week).
	// Split into consecutive <digits><unit> chunks.
	var total time.Duration
	rest := s
	for rest != "" {
		m := durationPartRe.FindStringSubmatch(rest)
		if m == nil {
			// Try a multi-part match by scanning.
			i := 0
			for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
				i++
			}
			if i == 0 || i == len(rest) {
				return 0, false
			}
			n, err := strconv.Atoi(rest[:i])
			if err != nil {
				return 0, false
			}
			unit := rest[i]
			rest = rest[i+1:]
			switch unit {
			case 'm':
				total += time.Duration(n) * time.Minute
			case 'h':
				total += time.Duration(n) * time.Hour
			case 'd':
				total += time.Duration(n) * 24 * time.Hour
			case 'w':
				total += time.Duration(n) * 7 * 24 * time.Hour
			default:
				return 0, false
			}
			continue
		}
		n, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "m":
			total += time.Duration(n) * time.Minute
		case "h":
			total += time.Duration(n) * time.Hour
		case "d":
			total += time.Duration(n) * 24 * time.Hour
		case "w":
			total += time.Duration(n) * 7 * 24 * time.Hour
		}
		rest = ""
	}

	if total <= 0 {
		return 0, false
	}
	return total, true
}

// formatDuration renders a Duration as a compact label ("15m", "3h", "2d", "1w").
func formatDuration(d time.Duration) string {
	switch {
	case d%(7*24*time.Hour) == 0 && d >= 7*24*time.Hour:
		return fmt.Sprintf("%dw", d/(7*24*time.Hour))
	case d%(24*time.Hour) == 0 && d >= 24*time.Hour:
		return fmt.Sprintf("%dd", d/(24*time.Hour))
	case d%time.Hour == 0 && d >= time.Hour:
		return fmt.Sprintf("%dh", d/time.Hour)
	case d%time.Minute == 0 && d >= time.Minute:
		return fmt.Sprintf("%dm", d/time.Minute)
	}
	return d.String()
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

func startOfWeek(t time.Time, loc *time.Location) time.Time {
	local := t.In(loc)
	weekday := local.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	daysBack := int(weekday) - int(time.Monday)
	y, m, d := local.AddDate(0, 0, -daysBack).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

func parseSinceDate(s string, loc *time.Location) (time.Time, error) {
	// Try RFC3339 first.
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try YYYY-MM-DD in local TZ.
	if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("expected RFC3339 or YYYY-MM-DD, got %q", s)
}
