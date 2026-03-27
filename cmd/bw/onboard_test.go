package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jallum/beadwork/prompts"
)

func TestCmdOnboardBasic(t *testing.T) {
	var buf bytes.Buffer
	err := cmdOnboard(PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdOnboard: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}

func TestCmdOnboardTTY(t *testing.T) {
	var buf bytes.Buffer
	err := cmdOnboard(ColorWriter(&buf, 80))
	if err != nil {
		t.Fatalf("cmdOnboard: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Add this to your agent instructions file") {
		t.Error("TTY output should include decoration")
	}
	if !strings.Contains(out, "How it works") {
		t.Error("TTY output should include 'How it works' section")
	}
}

func TestCmdOnboardPiped(t *testing.T) {
	var buf bytes.Buffer
	err := cmdOnboard(PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdOnboard: %v", err)
	}
	out := buf.String()

	// Should contain just the snippet content
	snippet := strings.TrimSpace(prompts.AgentsMD)
	if !strings.Contains(out, "Work Management") {
		t.Error("piped output should contain the snippet")
	}

	// Should NOT contain the decoration
	if strings.Contains(out, "Add this to your agent instructions file") {
		t.Error("piped output should not include decoration")
	}
	if strings.Contains(out, "How it works") {
		t.Error("piped output should not include 'How it works' section")
	}

	// Should be exactly the snippet plus a trailing newline
	if strings.TrimSpace(out) != snippet {
		t.Errorf("piped output should be exactly the snippet\ngot:  %q\nwant: %q", strings.TrimSpace(out), snippet)
	}
}
