package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jallum/beadwork/internal/config"

	"github.com/jallum/beadwork/internal/issue"
)

// timeOfDay represents a parsed time-of-day value.
type timeOfDay struct {
	hour, min int
}

type DeferArgs struct {
	ID   string
	Date string
	JSON bool
}

func parseDeferArgs(raw []string) (DeferArgs, error) {
	if len(raw) < 2 {
		return DeferArgs{}, fmt.Errorf("usage: bw defer <id> <date>")
	}
	a, err := ParseArgs(raw[1:], nil, []string{"--json"})
	if err != nil {
		return DeferArgs{}, err
	}
	dateExpr := strings.Join(a.Pos(), " ")
	if dateExpr == "" {
		return DeferArgs{}, fmt.Errorf("usage: bw defer <id> <date>")
	}
	return DeferArgs{ID: raw[0], Date: dateExpr, JSON: a.JSON()}, nil
}

// resolveDate converts a date or datetime expression to either YYYY-MM-DD
// (for date-only expressions) or RFC3339 with local timezone offset (for
// time-bearing expressions).
// The now parameter allows testing with a fixed time.
func resolveDate(expr string, now time.Time) (string, error) {
	original := strings.TrimSpace(expr)
	expr = strings.ToLower(original)

	// Try absolute date first (YYYY-MM-DD).
	if _, err := time.Parse("2006-01-02", expr); err == nil {
		return expr, nil
	}

	// Try RFC3339 passthrough (use original case to preserve T and Z).
	if _, err := time.Parse(time.RFC3339, original); err == nil {
		return original, nil
	}

	parts := strings.Fields(expr)
	local := now.In(time.Local)

	// Split on "at" for date-at-time expressions.
	dateExpr, timeExpr := splitAtKeyword(parts)

	// Handle bare time ("3pm", "14:00") — no date part.
	if dateExpr == "" && timeExpr != "" {
		tod, err := parseTimeOfDay(timeExpr)
		if err != nil {
			return "", err
		}
		result := time.Date(local.Year(), local.Month(), local.Day(),
			tod.hour, tod.min, 0, 0, local.Location())
		if !result.After(now) {
			result = result.AddDate(0, 0, 1)
			fmt.Fprintf(os.Stderr, "note: %s has passed; deferring until tomorrow at %s\n",
				time.Date(0, 1, 1, tod.hour, tod.min, 0, 0, time.UTC).Format("3:04 PM"),
				result.Format("3:04 PM"))
		}
		return result.Format(time.RFC3339), nil
	}

	// Handle "tomorrow" with optional time.
	if dateExpr == "tomorrow" || (len(parts) > 0 && parts[0] == "tomorrow" && timeExpr == "") {
		if timeExpr != "" {
			tod, err := parseTimeOfDay(timeExpr)
			if err != nil {
				return "", err
			}
			tomorrow := local.AddDate(0, 0, 1)
			result := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(),
				tod.hour, tod.min, 0, 0, local.Location())
			return result.Format(time.RFC3339), nil
		}
		return now.AddDate(0, 0, 1).Format("2006-01-02"), nil
	}

	// Handle "in N unit" or "N unit" patterns.
	offset, unit, ok := parseDurationExpr(parts)
	if ok {
		if offset < 0 {
			return "", fmt.Errorf("duration must be positive, got %d %s", offset, unit)
		}
		switch unit {
		case "minute":
			return now.Add(time.Duration(offset) * time.Minute).In(time.Local).Format(time.RFC3339), nil
		case "hour":
			return now.Add(time.Duration(offset) * time.Hour).In(time.Local).Format(time.RFC3339), nil
		case "day":
			return now.AddDate(0, 0, offset).Format("2006-01-02"), nil
		case "week":
			return now.AddDate(0, 0, offset*7).Format("2006-01-02"), nil
		case "month":
			return now.AddDate(0, offset, 0).Format("2006-01-02"), nil
		case "year":
			return now.AddDate(offset, 0, 0).Format("2006-01-02"), nil
		}
	}

	// Handle "next <weekday>" with optional time.
	if len(parts) >= 2 && parts[0] == "next" {
		day, ok := parseWeekday(parts[1])
		if ok {
			target := nextWeekday(now, day)
			if timeExpr != "" {
				tod, err := parseTimeOfDay(timeExpr)
				if err != nil {
					return "", err
				}
				targetLocal := target.In(time.Local)
				result := time.Date(targetLocal.Year(), targetLocal.Month(), targetLocal.Day(),
					tod.hour, tod.min, 0, 0, local.Location())
				return result.Format(time.RFC3339), nil
			}
			return target.Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("invalid date %q (expected YYYY-MM-DD, RFC3339, or relative like \"2 weeks\", \"next monday\", \"tomorrow\", \"in 15 minutes\", \"3pm\")", expr)
}

// splitAtKeyword splits parts on the "at" keyword, returning the date portion
// and time portion as joined strings. If no "at" is found, returns the full
// expression as dateExpr. If the expression starts with a time-like token
// (contains digits + am/pm or colon), it's treated as a bare time.
func splitAtKeyword(parts []string) (dateExpr, timeExpr string) {
	// Find the last "at" in parts.
	atIdx := -1
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "at" {
			atIdx = i
			break
		}
	}
	if atIdx >= 0 {
		dateExpr = strings.Join(parts[:atIdx], " ")
		timeExpr = strings.Join(parts[atIdx+1:], " ")
		return
	}
	// No "at" found. Check if entire expression is a bare time.
	joined := strings.Join(parts, " ")
	if len(parts) == 1 && looksLikeTime(parts[0]) {
		return "", joined
	}
	return joined, ""
}

// looksLikeTime returns true if s looks like a time expression (e.g., "3pm", "14:00").
func looksLikeTime(s string) bool {
	s = strings.ToLower(s)
	if strings.HasSuffix(s, "am") || strings.HasSuffix(s, "pm") {
		return true
	}
	if strings.Contains(s, ":") {
		return true
	}
	return false
}

// parseDurationExpr parses "in N unit", "N unit" patterns.
// Returns (n, normalizedUnit, true) or (0, "", false).
func parseDurationExpr(parts []string) (int, string, bool) {
	// Strip leading "in".
	if len(parts) >= 3 && parts[0] == "in" {
		parts = parts[1:]
	}
	if len(parts) != 2 {
		return 0, "", false
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", false
	}
	unit := strings.TrimSuffix(parts[1], "s") // normalize plural
	switch unit {
	case "minute", "hour", "day", "week", "month", "year":
		return n, unit, true
	}
	return 0, "", false
}

// parseTimeOfDay parses time expressions like "3pm", "14:00", "3:30am", "2:15pm".
func parseTimeOfDay(s string) (timeOfDay, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	// Try 24h format: "14:00", "9:30"
	if strings.Contains(s, ":") && !strings.HasSuffix(s, "am") && !strings.HasSuffix(s, "pm") {
		var h, m int
		n, _ := fmt.Sscanf(s, "%d:%d", &h, &m)
		if n != 2 {
			return timeOfDay{}, fmt.Errorf("invalid time %q", s)
		}
		if h < 0 || h > 23 || m < 0 || m > 59 {
			return timeOfDay{}, fmt.Errorf("invalid time %q: hour must be 0-23, minute 0-59", s)
		}
		return timeOfDay{h, m}, nil
	}

	// Try 12h format: "3pm", "3:30am", "12pm"
	var suffix string
	if strings.HasSuffix(s, "am") {
		suffix = "am"
		s = strings.TrimSuffix(s, "am")
	} else if strings.HasSuffix(s, "pm") {
		suffix = "pm"
		s = strings.TrimSuffix(s, "pm")
	} else {
		return timeOfDay{}, fmt.Errorf("invalid time %q", s)
	}

	var h, m int
	if strings.Contains(s, ":") {
		n, _ := fmt.Sscanf(s, "%d:%d", &h, &m)
		if n != 2 {
			return timeOfDay{}, fmt.Errorf("invalid time %q%s", s, suffix)
		}
	} else {
		n, _ := fmt.Sscanf(s, "%d", &h)
		if n != 1 {
			return timeOfDay{}, fmt.Errorf("invalid time %q%s", s, suffix)
		}
	}

	if h < 1 || h > 12 || m < 0 || m > 59 {
		return timeOfDay{}, fmt.Errorf("invalid time %q%s: hour must be 1-12", s, suffix)
	}

	// Convert to 24h.
	if suffix == "am" {
		if h == 12 {
			h = 0 // 12am = midnight
		}
	} else { // pm
		if h != 12 {
			h += 12 // 1pm-11pm
		}
		// 12pm stays 12 (noon)
	}

	return timeOfDay{h, m}, nil
}

// validateDate validates a date string as either YYYY-MM-DD or RFC3339.
func validateDate(s string) error {
	if _, err := time.Parse("2006-01-02", s); err == nil {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return nil
	}
	return fmt.Errorf("invalid date %q (expected YYYY-MM-DD or RFC3339)", s)
}

var weekdayNames = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
	"sun":       time.Sunday,
	"mon":       time.Monday,
	"tue":       time.Tuesday,
	"wed":       time.Wednesday,
	"thu":       time.Thursday,
	"fri":       time.Friday,
	"sat":       time.Saturday,
}

func parseWeekday(s string) (time.Weekday, bool) {
	d, ok := weekdayNames[strings.ToLower(s)]
	return d, ok
}

// nextWeekday returns the date of the next occurrence of the given weekday
// after now. If now is that weekday, it returns the following week.
func nextWeekday(now time.Time, day time.Weekday) time.Time {
	diff := int(day) - int(now.Weekday())
	if diff <= 0 {
		diff += 7
	}
	return now.AddDate(0, 0, diff)
}

func cmdDefer(store *issue.Store, args []string, w Writer, _ *config.Config) (*config.Config, error) {
	da, err := parseDeferArgs(args)
	if err != nil {
		return nil, err
	}

	resolved, err := resolveDate(da.Date, store.Now())
	if err != nil {
		return nil, err
	}
	da.Date = resolved

	status := "deferred"
	iss, err := store.Update(da.ID, issue.UpdateOpts{
		Status:     &status,
		DeferUntil: &da.Date,
	})
	if err != nil {
		return nil, err
	}

	intent := fmt.Sprintf("defer %s until %s", iss.ID, da.Date)
	if err := store.Commit(intent); err != nil {
		return nil, fmt.Errorf("commit failed: %w", err)
	}

	if da.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "deferred %s until %s\n", iss.ID, da.Date)
	}
	return nil, nil
}
