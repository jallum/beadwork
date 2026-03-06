package prompts

import (
	"strings"
	"testing"
)

func TestPrimePriorityRange(t *testing.T) {
	if !strings.Contains(Prime, "P0") {
		t.Error("prime.md should document P0")
	}
	if !strings.Contains(Prime, "P4") {
		t.Error("prime.md should document P4")
	}
	if !strings.Contains(Prime, "P0-P4") {
		t.Error("prime.md should mention P0-P4 notation")
	}
}

func TestPrimeStatuses(t *testing.T) {
	for _, s := range []string{"open", "in_progress", "closed", "deferred"} {
		if !strings.Contains(Prime, s) {
			t.Errorf("prime.md should mention status %q", s)
		}
	}
}
