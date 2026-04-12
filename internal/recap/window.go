package recap

import (
	"fmt"
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
	case "24h":
		return Window{Start: now.Add(-24 * time.Hour), End: now, Label: "last 24h"}, nil
	case "7d":
		return Window{Start: now.AddDate(0, 0, -7), End: now, Label: "last 7 days"}, nil
	default:
		return Window{}, fmt.Errorf("unknown window %q (expected: today, yesterday, week, 24h, 7d)", token)
	}
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
