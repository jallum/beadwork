package prompts

import (
	"strings"
	"testing"
)

func TestPrimePriorityRange(t *testing.T) {
	if !strings.Contains(Prime, "Priority: 0-4 (0=critical, 2=default, 4=backlog)") {
		t.Error("prime.md should document priority range as 0-4 (0=critical, 2=default, 4=backlog)")
	}
	if !strings.Contains(Prime, "Also accepts P0-P4") {
		t.Error("prime.md should mention P0-P4 notation")
	}
}

func TestPrimeStatuses(t *testing.T) {
	if !strings.Contains(Prime, "Statuses: open, in_progress, deferred, closed") {
		t.Error("prime.md should list all four statuses: open, in_progress, deferred, closed")
	}
}
