package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jallum/beadwork/internal/issue"
)

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

// resolveDate converts a date expression to YYYY-MM-DD format.
// It accepts absolute dates (YYYY-MM-DD) and relative expressions like
// "2 weeks", "3 days", "next monday", "tomorrow".
// The now parameter allows testing with a fixed time.
func resolveDate(expr string, now time.Time) (string, error) {
	expr = strings.TrimSpace(strings.ToLower(expr))

	// Try absolute date first.
	if _, err := time.Parse("2006-01-02", expr); err == nil {
		return expr, nil
	}

	// Handle "tomorrow".
	if expr == "tomorrow" {
		return now.AddDate(0, 0, 1).Format("2006-01-02"), nil
	}

	// Handle "<N> <unit>" patterns: "2 weeks", "3 days", "1 month".
	parts := strings.Fields(expr)
	if len(parts) == 2 {
		n, err := strconv.Atoi(parts[0])
		if err == nil {
			unit := strings.TrimSuffix(parts[1], "s") // normalize plural
			switch unit {
			case "day":
				return now.AddDate(0, 0, n).Format("2006-01-02"), nil
			case "week":
				return now.AddDate(0, 0, n*7).Format("2006-01-02"), nil
			case "month":
				return now.AddDate(0, n, 0).Format("2006-01-02"), nil
			case "year":
				return now.AddDate(n, 0, 0).Format("2006-01-02"), nil
			}
		}
	}

	// Handle "next <weekday>".
	if len(parts) == 2 && parts[0] == "next" {
		day, ok := parseWeekday(parts[1])
		if ok {
			return nextWeekday(now, day).Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("invalid date %q (expected YYYY-MM-DD or relative like \"2 weeks\", \"next monday\", \"tomorrow\")", expr)
}

// resolveDateNow is a convenience wrapper that uses time.Now().
func resolveDateNow(expr string) (string, error) {
	return resolveDate(expr, time.Now())
}

func validateDate(s string) error {
	_, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", s)
	}
	return nil
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

func cmdDefer(store *issue.Store, args []string, w Writer) error {
	da, err := parseDeferArgs(args)
	if err != nil {
		return err
	}

	resolved, err := resolveDateNow(da.Date)
	if err != nil {
		return err
	}
	da.Date = resolved

	status := "deferred"
	iss, err := store.Update(da.ID, issue.UpdateOpts{
		Status:     &status,
		DeferUntil: &da.Date,
	})
	if err != nil {
		return err
	}

	intent := fmt.Sprintf("defer %s until %s", iss.ID, da.Date)
	if err := store.Commit(intent); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	if da.JSON {
		fprintJSON(w, iss)
	} else {
		fmt.Fprintf(w, "deferred %s until %s\n", iss.ID, da.Date)
	}
	return nil
}
