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

func TestPriorityIcon(t *testing.T) {
	for p := 0; p <= 4; p++ {
		got := issue.PriorityIcon(p)
		if got != "●" {
			t.Errorf("PriorityIcon(%d) = %q, want %q", p, got, "●")
		}
	}

	// Unknown priority returns "?"
	if got := issue.PriorityIcon(99); got != "?" {
		t.Errorf("PriorityIcon(99) = %q, want %q", got, "?")
	}
	if got := issue.PriorityIcon(-1); got != "?" {
		t.Errorf("PriorityIcon(-1) = %q, want %q", got, "?")
	}
}
