package issue_test

import (
	"testing"

	"github.com/jallum/beadwork/internal/issue"
)

func TestStatusNames(t *testing.T) {
	names := issue.StatusNames()
	if len(names) != len(issue.Statuses) {
		t.Fatalf("StatusNames() returned %d, want %d", len(names), len(issue.Statuses))
	}
	for i, s := range issue.Statuses {
		if names[i] != s.Name {
			t.Errorf("StatusNames()[%d] = %q, want %q", i, names[i], s.Name)
		}
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"open", "○"},
		{"in_progress", "◐"},
		{"closed", "✓"},
		{"unknown", "?"},
	}
	for _, tt := range tests {
		got := issue.StatusIcon(tt.status)
		if got != tt.want {
			t.Errorf("StatusIcon(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}

	// Deferred status should have snowflake icon
	got := issue.StatusIcon("deferred")
	if got != "❄" {
		t.Errorf("StatusIcon(deferred) = %q, want ❄", got)
	}
}

func TestPriorityDot(t *testing.T) {
	// Each priority should produce a colored ● with reset
	for p := 0; p <= 4; p++ {
		dot := issue.PriorityDot(p)
		if dot == "" {
			t.Errorf("PriorityDot(%d) returned empty string", p)
		}
		color, ok := issue.PriorityColors[p]
		if !ok {
			t.Errorf("PriorityColors missing key %d", p)
			continue
		}
		want := color + "●" + issue.ColorReset
		if dot != want {
			t.Errorf("PriorityDot(%d) = %q, want %q", p, dot, want)
		}
	}

	// Unknown priority should still return a dot
	dot := issue.PriorityDot(99)
	if dot == "" {
		t.Error("PriorityDot(99) returned empty string")
	}
	if dot != "●"+issue.ColorReset {
		t.Errorf("PriorityDot(99) = %q, want uncolored dot", dot)
	}
}
