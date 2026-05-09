package recap

import (
	"testing"
	"time"
)

var testTime = time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

func TestParseIntentCreate(t *testing.T) {
	events := ParseIntent(`create bw-abc p2 task "Fix login"`, testTime)
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	e := events[0]
	if e.Type != "create" || e.ID != "bw-abc" {
		t.Errorf("type=%q id=%q", e.Type, e.ID)
	}
	if e.Detail == "" {
		t.Error("expected detail for create")
	}
}

func TestParseIntentClose(t *testing.T) {
	events := ParseIntent("close bw-xyz", testTime)
	if len(events) != 1 || events[0].Type != "close" || events[0].ID != "bw-xyz" {
		t.Errorf("got %v", events)
	}
}

func TestParseIntentCloseWithReason(t *testing.T) {
	events := ParseIntent(`close bw-xyz reason="duplicate"`, testTime)
	if len(events) != 1 || events[0].Type != "close" {
		t.Fatalf("got %v", events)
	}
	if events[0].Detail == "" {
		t.Error("expected detail with reason")
	}
}

func TestParseIntentStart(t *testing.T) {
	events := ParseIntent(`start bw-1 assignee="alice"`, testTime)
	if len(events) != 1 || events[0].Type != "start" || events[0].ID != "bw-1" {
		t.Errorf("got %v", events)
	}
}

func TestParseIntentUpdate(t *testing.T) {
	events := ParseIntent(`update bw-1 priority=1`, testTime)
	if len(events) != 1 || events[0].Type != "update" {
		t.Errorf("got %v", events)
	}
}

func TestParseIntentReopen(t *testing.T) {
	events := ParseIntent("reopen bw-1", testTime)
	if len(events) != 1 || events[0].Type != "reopen" {
		t.Errorf("got %v", events)
	}
}

func TestParseIntentDefer(t *testing.T) {
	events := ParseIntent("defer bw-1 until 2026-06-01", testTime)
	if len(events) != 1 || events[0].Type != "defer" {
		t.Errorf("got %v", events)
	}
}

func TestParseIntentUndefer(t *testing.T) {
	events := ParseIntent("undefer bw-1", testTime)
	if len(events) != 1 || events[0].Type != "undefer" {
		t.Errorf("got %v", events)
	}
}

func TestParseIntentComment(t *testing.T) {
	events := ParseIntent(`comment bw-1 "Fixed it"`, testTime)
	if len(events) != 1 || events[0].Type != "comment" {
		t.Errorf("got %v", events)
	}
}

func TestParseIntentLink(t *testing.T) {
	events := ParseIntent("link bw-1 blocks bw-2", testTime)
	if len(events) != 1 || events[0].Type != "link" || events[0].Detail != "blocks bw-2" {
		t.Errorf("got %v", events)
	}
}

func TestParseIntentUnlink(t *testing.T) {
	events := ParseIntent("unlink bw-1 blocks bw-2", testTime)
	if len(events) != 1 || events[0].Type != "unlink" {
		t.Errorf("got %v", events)
	}
}

func TestParseIntentDelete(t *testing.T) {
	events := ParseIntent("delete bw-1", testTime)
	if len(events) != 1 || events[0].Type != "delete" {
		t.Errorf("got %v", events)
	}
}

func TestParseIntentLabel(t *testing.T) {
	events := ParseIntent("label bw-1 +bug +urgent", testTime)
	if len(events) != 1 || events[0].Type != "label" {
		t.Errorf("got %v", events)
	}
}

func TestParseIntentUnblockedSecondary(t *testing.T) {
	msg := "close bw-1\nunblocked bw-2\nunblocked bw-3"
	events := ParseIntent(msg, testTime)
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	if events[0].Type != "close" || events[0].ID != "bw-1" {
		t.Errorf("event[0] = %v", events[0])
	}
	if events[1].Type != "unblocked" || events[1].ID != "bw-2" {
		t.Errorf("event[1] = %v", events[1])
	}
	if events[2].Type != "unblocked" || events[2].ID != "bw-3" {
		t.Errorf("event[2] = %v", events[2])
	}
}

func TestParseIntentReasonContainingUnblocked(t *testing.T) {
	// A close reason that contains "unblocked" as a word should NOT
	// produce a spurious unblocked event.
	msg := `close bw-1 reason="unblocked by external team"`
	events := ParseIntent(msg, testTime)
	for _, e := range events {
		if e.Type == "unblocked" {
			t.Errorf("spurious unblocked event from reason text: %v", e)
		}
	}
}

func TestParseIntentEmpty(t *testing.T) {
	events := ParseIntent("", testTime)
	if len(events) != 0 {
		t.Errorf("got %d events for empty message", len(events))
	}
}

func TestParseIntentUnknown(t *testing.T) {
	events := ParseIntent("init beadwork", testTime)
	if len(events) != 0 {
		t.Errorf("got %d events for unknown intent", len(events))
	}
}

func FuzzParseIntent(f *testing.F) {
	f.Add("create bw-1 p2 task \"test\"")
	f.Add("close bw-1\nunblocked bw-2")
	f.Add("start bw-1 assignee=\"alice\"")
	f.Add("update bw-1 priority=0")
	f.Add("")
	f.Add("garbage input!!!")
	f.Fuzz(func(t *testing.T, msg string) {
		// Should never panic.
		ParseIntent(msg, testTime)
	})
}
