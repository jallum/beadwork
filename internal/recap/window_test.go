package recap

import (
	"testing"
	"time"
)

func TestWindowToday(t *testing.T) {
	// Wednesday 2026-01-15 at 10:00 UTC
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	w, err := ParseWindow([]string{"today"}, "", now)
	if err != nil {
		t.Fatal(err)
	}
	if w.Label != "today" {
		t.Errorf("label = %q", w.Label)
	}
	wantStart := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	if !w.Start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", w.Start, wantStart)
	}
	if !w.End.Equal(now) {
		t.Errorf("end = %v, want %v", w.End, now)
	}
}

func TestWindowYesterday(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	w, err := ParseWindow([]string{"yesterday"}, "", now)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	if !w.Start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", w.Start, wantStart)
	}
	if !w.End.Equal(wantEnd) {
		t.Errorf("end = %v, want %v", w.End, wantEnd)
	}
}

func TestWindowWeek(t *testing.T) {
	// Wednesday 2026-01-15 → Monday 2026-01-12
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	w, err := ParseWindow([]string{"week"}, "", now)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC)
	if !w.Start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", w.Start, wantStart)
	}
}

func TestWindow24h(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	w, err := ParseWindow([]string{"24h"}, "", now)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := now.Add(-24 * time.Hour)
	if !w.Start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", w.Start, wantStart)
	}
}

func TestWindow7d(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	w, err := ParseWindow([]string{"7d"}, "", now)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := now.AddDate(0, 0, -7)
	if !w.Start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", w.Start, wantStart)
	}
}

func TestWindowDefault24h(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	w, err := ParseWindow(nil, "", now)
	if err != nil {
		t.Fatal(err)
	}
	if w.Label != "last 24h" {
		t.Errorf("default label = %q, want 'last 24h'", w.Label)
	}
}

func TestWindowSinceRFC3339(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	w, err := ParseWindow(nil, "2026-01-10T00:00:00Z", now)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	if !w.Start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", w.Start, wantStart)
	}
}

func TestWindowSinceDate(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	w, err := ParseWindow(nil, "2026-01-10", now)
	if err != nil {
		t.Fatal(err)
	}
	if w.Start.Year() != 2026 || w.Start.Month() != 1 || w.Start.Day() != 10 {
		t.Errorf("start = %v", w.Start)
	}
}

func TestWindowSinceInvalid(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	_, err := ParseWindow(nil, "not-a-date", now)
	if err == nil {
		t.Error("expected error for invalid --since")
	}
}

func TestWindowUnknownToken(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	_, err := ParseWindow([]string{"fortnight"}, "", now)
	if err == nil {
		t.Error("expected error for unknown token")
	}
}

func TestWindowTodayLocalTZ(t *testing.T) {
	// Ensure "today" uses the local timezone of now, not UTC.
	eastern := time.FixedZone("EST", -5*3600)
	// 2026-01-15 01:00 EST = 2026-01-15 06:00 UTC
	now := time.Date(2026, 1, 15, 1, 0, 0, 0, eastern)
	w, err := ParseWindow([]string{"today"}, "", now)
	if err != nil {
		t.Fatal(err)
	}
	// Start of today in EST should be 2026-01-15 00:00 EST
	wantStart := time.Date(2026, 1, 15, 0, 0, 0, 0, eastern)
	if !w.Start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", w.Start, wantStart)
	}
}

func TestWindowYesterdayLocalTZ(t *testing.T) {
	// At 2026-01-15 01:00 EST, yesterday starts at 2026-01-14 00:00 EST
	eastern := time.FixedZone("EST", -5*3600)
	now := time.Date(2026, 1, 15, 1, 0, 0, 0, eastern)
	w, err := ParseWindow([]string{"yesterday"}, "", now)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 1, 14, 0, 0, 0, 0, eastern)
	wantEnd := time.Date(2026, 1, 15, 0, 0, 0, 0, eastern)
	if !w.Start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", w.Start, wantStart)
	}
	if !w.End.Equal(wantEnd) {
		t.Errorf("end = %v, want %v", w.End, wantEnd)
	}
}

func FuzzParseWindow(f *testing.F) {
	f.Add("today")
	f.Add("yesterday")
	f.Add("week")
	f.Add("24h")
	f.Add("7d")
	f.Add("garbage")
	f.Fuzz(func(t *testing.T, token string) {
		now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
		ParseWindow([]string{token}, "", now)
	})
}
