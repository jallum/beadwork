package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdDefer(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Defer me", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdDefer(env.Store, []string{iss.ID, "2027-06-01"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdDefer: %v", err)
	}
	if !strings.Contains(buf.String(), "deferred") {
		t.Errorf("output = %q, want 'deferred'", buf.String())
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "deferred" {
		t.Errorf("status = %q, want deferred", got.Status)
	}
	if got.DeferUntil != "2027-06-01" {
		t.Errorf("defer_until = %q, want 2027-06-01", got.DeferUntil)
	}
}

func TestCmdDeferJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Defer JSON", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdDefer(env.Store, []string{iss.ID, "2027-06-01", "--json"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdDefer --json: %v", err)
	}

	var got issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got.Status != "deferred" {
		t.Errorf("status = %q, want deferred", got.Status)
	}
	if got.DeferUntil != "2027-06-01" {
		t.Errorf("defer_until = %q, want 2027-06-01", got.DeferUntil)
	}
}

func TestCmdDeferInvalidDate(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Bad date", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdDefer(env.Store, []string{iss.ID, "not-a-date"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestCmdUndefer(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Undefer me", issue.CreateOpts{DeferUntil: "2027-06-01"})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdUndefer(env.Store, []string{iss.ID}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdUndefer: %v", err)
	}
	if !strings.Contains(buf.String(), "undeferred") {
		t.Errorf("output = %q, want 'undeferred'", buf.String())
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
	if got.DeferUntil != "" {
		t.Errorf("defer_until = %q, want empty", got.DeferUntil)
	}
}

func TestCmdUndeferJSON(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Undefer JSON", issue.CreateOpts{DeferUntil: "2027-06-01"})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdUndefer(env.Store, []string{iss.ID, "--json"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdUndefer --json: %v", err)
	}

	var got issue.Issue
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
}

func TestCmdCreateWithDefer(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdCreate(env.Store, []string{"Deferred task", "--defer", "2027-03-15"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdCreate --defer: %v", err)
	}

	issues, _ := env.Store.List(issue.Filter{Status: "deferred"})
	if len(issues) != 1 {
		t.Fatalf("expected 1 deferred issue, got %d", len(issues))
	}
	if issues[0].DeferUntil != "2027-03-15" {
		t.Errorf("defer_until = %q, want 2027-03-15", issues[0].DeferUntil)
	}
}

func TestCmdUpdateWithDefer(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Update defer", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdUpdate(env.Store, []string{iss.ID, "--defer", "2027-09-01"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdUpdate --defer: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "deferred" {
		t.Errorf("status = %q, want deferred", got.Status)
	}
	if got.DeferUntil != "2027-09-01" {
		t.Errorf("defer_until = %q, want 2027-09-01", got.DeferUntil)
	}
}

func TestCmdListDeferred(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Store.Create("Open task", issue.CreateOpts{})
	env.Store.Create("Deferred task", issue.CreateOpts{DeferUntil: "2027-01-01"})
	env.Repo.Commit("create issues")

	// Default list should show only open
	var buf bytes.Buffer
	_, err := cmdList(env.Store, []string{}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	if !strings.Contains(buf.String(), "Open task") {
		t.Errorf("default list should show open task: %q", buf.String())
	}
	if strings.Contains(buf.String(), "Deferred task") {
		t.Error("default list should NOT show deferred task")
	}

	// --deferred should show only deferred
	buf.Reset()
	_, err = cmdList(env.Store, []string{"--deferred"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdList --deferred: %v", err)
	}
	if !strings.Contains(buf.String(), "Deferred task") {
		t.Errorf("--deferred should show deferred task: %q", buf.String())
	}
	if strings.Contains(buf.String(), "Open task") {
		t.Error("--deferred should NOT show open task")
	}
}

func TestCmdShowDeferred(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Deferred show", issue.CreateOpts{DeferUntil: "2027-04-15"})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdShow(env.Store, []string{iss.ID}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Deferred: 2027-04-15") {
		t.Errorf("output should contain 'Deferred: 2027-04-15': %q", out)
	}
	if !strings.Contains(out, "❄") {
		t.Errorf("output should contain deferred icon: %q", out)
	}
}

func TestCmdDeferNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdDefer(env.Store, []string{}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for no args")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q, want usage message", err)
	}
}

func TestCmdDeferOneArg(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdDefer(env.Store, []string{"bw-1234"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for missing date arg")
	}
}

func TestCmdDeferNonExistentIssue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdDefer(env.Store, []string{"bw-0000", "2027-06-01"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for non-existent issue")
	}
}

func TestCmdDeferVariousInvalidDates(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Date test", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	invalidDates := []string{
		"2027/06/01",
		"06-01-2027",
		"2027-13-01",
		"2027-06-32",
		"",
		"next someday",
		"5 fortnights",
	}
	for _, d := range invalidDates {
		var buf bytes.Buffer
		_, err := cmdDefer(env.Store, []string{iss.ID, d}, PlainWriter(&buf), nil)
		if err == nil {
			t.Errorf("expected error for invalid date %q", d)
		}
	}
}

func TestCmdDeferAlreadyDeferred(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Already deferred", issue.CreateOpts{DeferUntil: "2027-01-01"})
	env.Repo.Commit("create " + iss.ID)

	// Defer again with a new date
	var buf bytes.Buffer
	_, err := cmdDefer(env.Store, []string{iss.ID, "2027-12-01"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdDefer on already deferred: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.DeferUntil != "2027-12-01" {
		t.Errorf("defer_until = %q, want 2027-12-01", got.DeferUntil)
	}
}

func TestCmdDeferUnknownFlag(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Flag test", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdDefer(env.Store, []string{iss.ID, "2027-06-01", "--unknown"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestValidateDate(t *testing.T) {
	valid := []string{"2027-01-01", "2026-12-31", "2030-06-15"}
	for _, d := range valid {
		if err := validateDate(d); err != nil {
			t.Errorf("validateDate(%q) = %v, want nil", d, err)
		}
	}

	invalid := []string{"not-a-date", "2027/01/01", "01-01-2027", "2027-13-01", ""}
	for _, d := range invalid {
		if err := validateDate(d); err == nil {
			t.Errorf("validateDate(%q) = nil, want error", d)
		}
	}
}

func TestResolveDate(t *testing.T) {
	// Fixed reference time: Wednesday 2027-03-10
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		input string
		want  string
	}{
		// Absolute dates pass through.
		{"2027-06-01", "2027-06-01"},
		// Relative: N days/weeks/months/years.
		{"3 days", "2027-03-13"},
		{"1 day", "2027-03-11"},
		{"2 weeks", "2027-03-24"},
		{"1 week", "2027-03-17"},
		{"1 month", "2027-04-10"},
		{"2 months", "2027-05-10"},
		{"1 year", "2028-03-10"},
		// Tomorrow.
		{"tomorrow", "2027-03-11"},
		// Next weekday (reference is Wednesday).
		{"next monday", "2027-03-15"},
		{"next friday", "2027-03-12"},
		{"next wednesday", "2027-03-17"}, // same day → next week
		{"next sun", "2027-03-14"},
		// Case insensitive.
		{"Next Monday", "2027-03-15"},
		{"TOMORROW", "2027-03-11"},
		{"2 Weeks", "2027-03-24"},
	}

	for _, tt := range tests {
		got, err := resolveDate(tt.input, now)
		if err != nil {
			t.Errorf("resolveDate(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("resolveDate(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveDateTimeExpressions(t *testing.T) {
	// Fixed reference: Wednesday 2027-03-10 at noon UTC.
	// In a test environment, time.Local may be UTC, so we set up accordingly.
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)

	t.Run("RFC3339 passthrough", func(t *testing.T) {
		got, err := resolveDate("2027-04-15T14:00:00-04:00", now)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if got != "2027-04-15T14:00:00-04:00" {
			t.Errorf("got %q, want passthrough", got)
		}
	})

	t.Run("in N minutes", func(t *testing.T) {
		got, err := resolveDate("in 15 minutes", now)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		parsed, err := time.Parse(time.RFC3339, got)
		if err != nil {
			t.Fatalf("result %q is not RFC3339: %v", got, err)
		}
		expected := now.Add(15 * time.Minute)
		if !parsed.Equal(expected) {
			t.Errorf("got instant %v, want %v", parsed, expected)
		}
	})

	t.Run("in 4 hours", func(t *testing.T) {
		got, err := resolveDate("in 4 hours", now)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		parsed, _ := time.Parse(time.RFC3339, got)
		expected := now.Add(4 * time.Hour)
		if !parsed.Equal(expected) {
			t.Errorf("got instant %v, want %v", parsed, expected)
		}
	})

	t.Run("N minutes without in", func(t *testing.T) {
		got, err := resolveDate("15 minutes", now)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		parsed, _ := time.Parse(time.RFC3339, got)
		expected := now.Add(15 * time.Minute)
		if !parsed.Equal(expected) {
			t.Errorf("got instant %v, want %v", parsed, expected)
		}
	})

	t.Run("2 hours without in", func(t *testing.T) {
		got, err := resolveDate("2 hours", now)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		parsed, _ := time.Parse(time.RFC3339, got)
		expected := now.Add(2 * time.Hour)
		if !parsed.Equal(expected) {
			t.Errorf("got instant %v, want %v", parsed, expected)
		}
	})

	t.Run("negative duration rejected", func(t *testing.T) {
		_, err := resolveDate("in -5 minutes", now)
		if err == nil {
			t.Error("expected error for negative duration")
		}
	})

	t.Run("in 0 minutes", func(t *testing.T) {
		got, err := resolveDate("in 0 minutes", now)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		parsed, _ := time.Parse(time.RFC3339, got)
		if !parsed.Equal(now) {
			t.Errorf("got %v, want %v (now)", parsed, now)
		}
	})

	t.Run("tomorrow at 2pm", func(t *testing.T) {
		got, err := resolveDate("tomorrow at 2pm", now)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		parsed, err := time.Parse(time.RFC3339, got)
		if err != nil {
			t.Fatalf("result %q is not RFC3339: %v", got, err)
		}
		if parsed.Hour() != 14 || parsed.Minute() != 0 {
			t.Errorf("got hour=%d min=%d, want 14:00", parsed.Hour(), parsed.Minute())
		}
	})

	t.Run("tomorrow at 14:00", func(t *testing.T) {
		got, err := resolveDate("tomorrow at 14:00", now)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		parsed, _ := time.Parse(time.RFC3339, got)
		if parsed.Hour() != 14 || parsed.Minute() != 0 {
			t.Errorf("got hour=%d min=%d, want 14:00", parsed.Hour(), parsed.Minute())
		}
	})

	t.Run("next monday at 9am", func(t *testing.T) {
		got, err := resolveDate("next monday at 9am", now)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		parsed, _ := time.Parse(time.RFC3339, got)
		if parsed.Weekday() != time.Monday {
			t.Errorf("got weekday %v, want Monday", parsed.Weekday())
		}
		if parsed.Hour() != 9 {
			t.Errorf("got hour %d, want 9", parsed.Hour())
		}
	})

	t.Run("next monday at 9:30am", func(t *testing.T) {
		got, err := resolveDate("next monday at 9:30am", now)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		parsed, _ := time.Parse(time.RFC3339, got)
		if parsed.Hour() != 9 || parsed.Minute() != 30 {
			t.Errorf("got %d:%d, want 9:30", parsed.Hour(), parsed.Minute())
		}
	})

	t.Run("date-only expressions still produce YYYY-MM-DD", func(t *testing.T) {
		for _, expr := range []string{"tomorrow", "2 weeks", "next monday", "2027-06-01"} {
			got, err := resolveDate(expr, now)
			if err != nil {
				t.Errorf("resolveDate(%q) error: %v", expr, err)
				continue
			}
			if strings.Contains(got, "T") {
				t.Errorf("resolveDate(%q) = %q, should be date-only (no T)", expr, got)
			}
		}
	})
}

func TestParseTimeOfDay(t *testing.T) {
	tests := []struct {
		input   string
		wantH   int
		wantM   int
		wantErr bool
	}{
		{"3pm", 15, 0, false},
		{"3PM", 15, 0, false},
		{"12pm", 12, 0, false}, // noon
		{"12am", 0, 0, false},  // midnight
		{"9am", 9, 0, false},
		{"3:30pm", 15, 30, false},
		{"11:45am", 11, 45, false},
		{"14:00", 14, 0, false},
		{"0:00", 0, 0, false},
		{"23:59", 23, 59, false},
		{"9:05", 9, 5, false},
		// Invalid
		{"24:00", 0, 0, true},
		{"25:00", 0, 0, true},
		{"13pm", 0, 0, true},
		{"0am", 0, 0, true},
		{"abc", 0, 0, true},
		{"", 0, 0, true},
	}

	for _, tt := range tests {
		got, err := parseTimeOfDay(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseTimeOfDay(%q) = %+v, want error", tt.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseTimeOfDay(%q) error: %v", tt.input, err)
			continue
		}
		if got.hour != tt.wantH || got.min != tt.wantM {
			t.Errorf("parseTimeOfDay(%q) = %d:%02d, want %d:%02d", tt.input, got.hour, got.min, tt.wantH, tt.wantM)
		}
	}
}

func TestValidateDateAcceptsBothFormats(t *testing.T) {
	valid := []string{
		"2027-06-01",
		"2027-04-15T14:00:00-04:00",
		"2027-04-15T14:00:00Z",
	}
	for _, d := range valid {
		if err := validateDate(d); err != nil {
			t.Errorf("validateDate(%q) = %v, want nil", d, err)
		}
	}

	invalid := []string{
		"not-a-date",
		"2027/06/01",
		"",
	}
	for _, d := range invalid {
		if err := validateDate(d); err == nil {
			t.Errorf("validateDate(%q) = nil, want error", d)
		}
	}
}

func TestResolveDateInvalid(t *testing.T) {
	now := time.Date(2027, 3, 10, 12, 0, 0, 0, time.UTC)

	invalid := []string{
		"not-a-date",
		"2027/06/01",
		"next someday",
		"5 fortnights",
		"",
	}
	for _, expr := range invalid {
		_, err := resolveDate(expr, now)
		if err == nil {
			t.Errorf("resolveDate(%q) = nil error, want error", expr)
		}
	}
}

func TestCmdDeferRelativeDate(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Relative defer", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	// "2 weeks" is two positional args after the ID.
	var buf bytes.Buffer
	_, err := cmdDefer(env.Store, []string{iss.ID, "2", "weeks"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdDefer with relative date: %v", err)
	}
	if !strings.Contains(buf.String(), "deferred") {
		t.Errorf("output = %q, want 'deferred'", buf.String())
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "deferred" {
		t.Errorf("status = %q, want deferred", got.Status)
	}
	// The resolved date should be a valid YYYY-MM-DD.
	if err := validateDate(got.DeferUntil); err != nil {
		t.Errorf("defer_until = %q, not a valid date: %v", got.DeferUntil, err)
	}
}

func TestCmdDeferTomorrow(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Tomorrow defer", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdDefer(env.Store, []string{iss.ID, "tomorrow"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdDefer with tomorrow: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "deferred" {
		t.Errorf("status = %q, want deferred", got.Status)
	}
	if err := validateDate(got.DeferUntil); err != nil {
		t.Errorf("defer_until = %q, not a valid date: %v", got.DeferUntil, err)
	}
}

func TestCmdDeferNextMonday(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Next monday defer", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	// "next monday" arrives as two tokens.
	var buf bytes.Buffer
	_, err := cmdDefer(env.Store, []string{iss.ID, "next", "monday"}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdDefer with next monday: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "deferred" {
		t.Errorf("status = %q, want deferred", got.Status)
	}
	if err := validateDate(got.DeferUntil); err != nil {
		t.Errorf("defer_until = %q, not a valid date: %v", got.DeferUntil, err)
	}
}

func TestCmdUndeferNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdUndefer(env.Store, []string{}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for no args")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q, want usage message", err)
	}
}

func TestCmdUndeferNonExistentIssue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	_, err := cmdUndefer(env.Store, []string{"bw-0000"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for non-existent issue")
	}
}

func TestCmdUndeferAlreadyOpen(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Already open", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	// Undefer an issue that's already open (no defer_until set)
	var buf bytes.Buffer
	_, err := cmdUndefer(env.Store, []string{iss.ID}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdUndefer on open issue: %v", err)
	}

	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
	if got.DeferUntil != "" {
		t.Errorf("defer_until = %q, want empty", got.DeferUntil)
	}
}

func TestCmdUndeferUnknownFlag(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Flag test", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdUndefer(env.Store, []string{iss.ID, "--unknown"}, PlainWriter(&buf), nil)
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestCmdUndeferVerifyCommit(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Verify commit", issue.CreateOpts{DeferUntil: "2027-06-01"})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	_, err := cmdUndefer(env.Store, []string{iss.ID}, PlainWriter(&buf), nil)
	if err != nil {
		t.Fatalf("cmdUndefer: %v", err)
	}
	if !strings.Contains(buf.String(), "undeferred") {
		t.Errorf("output = %q, want 'undeferred'", buf.String())
	}

	// Verify the issue was updated and committed
	got, _ := env.Store.Get(iss.ID)
	if got.Status != "open" {
		t.Errorf("status = %q, want open", got.Status)
	}
	if got.DeferUntil != "" {
		t.Errorf("defer_until = %q, want empty", got.DeferUntil)
	}
}

func TestParseUndeferArgs(t *testing.T) {
	ua, err := parseUndeferArgs([]string{"bw-1234"})
	if err != nil {
		t.Fatalf("parseUndeferArgs: %v", err)
	}
	if ua.ID != "bw-1234" {
		t.Errorf("ID = %q, want bw-1234", ua.ID)
	}
	if ua.JSON {
		t.Error("expected JSON=false")
	}
}

func TestParseUndeferArgsJSON(t *testing.T) {
	ua, err := parseUndeferArgs([]string{"bw-1234", "--json"})
	if err != nil {
		t.Fatalf("parseUndeferArgs: %v", err)
	}
	if ua.ID != "bw-1234" {
		t.Errorf("ID = %q, want bw-1234", ua.ID)
	}
	if !ua.JSON {
		t.Error("expected JSON=true")
	}
}
