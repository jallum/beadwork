package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestCommandMapContainsAllCommands(t *testing.T) {
	expected := []string{
		"init", "create", "show", "list", "update",
		"close", "reopen", "label",
		"dep", "ready", "graph",
		"sync", "export", "import",
		"config", "upgrade", "onboard", "prime",
	}
	for _, name := range expected {
		if _, ok := commandMap[name]; !ok {
			t.Errorf("commandMap missing command %q", name)
		}
	}
}

func TestCommandMapHasNoNilRun(t *testing.T) {
	for _, cmd := range commands {
		if cmd.Run == nil {
			t.Errorf("command %q has nil Run", cmd.Name)
		}
	}
}

func TestCommandsHaveSummary(t *testing.T) {
	for _, cmd := range commands {
		if cmd.Summary == "" {
			t.Errorf("command %q has empty Summary", cmd.Name)
		}
	}
}

func TestPrintUsageContainsAllCommands(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	out := buf.String()

	for _, cmd := range commands {
		if !strings.Contains(out, cmd.Name) {
			t.Errorf("printUsage output missing command %q", cmd.Name)
		}
		if !strings.Contains(out, cmd.Summary) {
			t.Errorf("printUsage output missing summary for %q: %q", cmd.Name, cmd.Summary)
		}
	}
}

func TestPrintUsageLayout(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	out := buf.String()

	// Header line with app name
	if !strings.Contains(out, "bw") {
		t.Error("missing 'bw' in header")
	}

	// Usage: section
	if !strings.Contains(out, "Usage:") {
		t.Error("missing 'Usage:' section")
	}
	if !strings.Contains(out, "bw <command>") {
		t.Error("missing 'bw <command>' in Usage section")
	}

	// Footer with help hint
	if !strings.Contains(out, `bw <command> --help`) {
		t.Error("missing help hint footer")
	}
}

func TestPrintCommandHelp(t *testing.T) {
	cmd := commandMap["create"]
	if cmd == nil {
		t.Fatal("create command not found")
	}

	var buf bytes.Buffer
	printCommandHelp(&buf, cmd)
	out := buf.String()

	// Should contain usage line
	if !strings.Contains(out, "bw create") {
		t.Error("missing 'bw create' in help output")
	}

	// Should contain flag descriptions
	for _, f := range cmd.Flags {
		if !strings.Contains(out, f.Long) {
			t.Errorf("missing flag %q in help output", f.Long)
		}
		if f.Help != "" && !strings.Contains(out, f.Help) {
			t.Errorf("missing help text for %q: %q", f.Long, f.Help)
		}
	}

	// Should contain positional descriptions
	for _, p := range cmd.Positionals {
		if !strings.Contains(out, p.Name) {
			t.Errorf("missing positional %q in help output", p.Name)
		}
	}
}

func TestPrintCommandHelpLayout(t *testing.T) {
	cmd := &Command{
		Name:        "test",
		Summary:     "Test summary",
		Description: "Longer description of the test command.\nIt can span multiple lines.",
		Positionals: []Positional{
			{Name: "<id>", Required: true, Help: "The issue ID"},
		},
		Flags: []Flag{
			{Long: "--verbose", Short: "-v", Help: "Enable verbose output"},
			{Long: "--format", Value: "FMT", Help: "Output format"},
		},
		Examples: []Example{
			{Cmd: "bw test abc123", Help: "Show test for abc123"},
			{Cmd: "bw test abc123 --verbose", Help: "Verbose output"},
		},
		Run: func([]string, io.Writer) error { return nil },
	}

	var buf bytes.Buffer
	printCommandHelp(&buf, cmd)
	out := buf.String()

	// Description appears first (before Usage:)
	descIdx := strings.Index(out, "Longer description")
	usageIdx := strings.Index(out, "Usage:")
	argsIdx := strings.Index(out, "Arguments:")
	flagsIdx := strings.Index(out, "Flags:")
	exIdx := strings.Index(out, "Examples:")

	if descIdx < 0 {
		t.Fatal("missing description in output")
	}
	if usageIdx < 0 {
		t.Fatal("missing Usage: section")
	}
	if argsIdx < 0 {
		t.Fatal("missing Arguments: section")
	}
	if flagsIdx < 0 {
		t.Fatal("missing Flags: section")
	}
	if exIdx < 0 {
		t.Fatal("missing Examples: section")
	}

	// Order: description < Usage < Arguments < Flags < Examples
	if descIdx >= usageIdx {
		t.Error("description should appear before Usage:")
	}
	if usageIdx >= argsIdx {
		t.Error("Usage: should appear before Arguments:")
	}
	if argsIdx >= flagsIdx {
		t.Error("Arguments: should appear before Flags:")
	}
	if flagsIdx >= exIdx {
		t.Error("Flags: should appear before Examples:")
	}

	// Examples content
	if !strings.Contains(out, "bw test abc123") {
		t.Error("missing example command")
	}
	if !strings.Contains(out, "Show test for abc123") {
		t.Error("missing example help text")
	}
}

func TestPrintCommandHelpNoDescription(t *testing.T) {
	cmd := &Command{
		Name:    "simple",
		Summary: "A simple command",
		Flags: []Flag{
			{Long: "--json", Help: "Output as JSON"},
		},
		Run: func([]string, io.Writer) error { return nil },
	}

	var buf bytes.Buffer
	printCommandHelp(&buf, cmd)
	out := buf.String()

	// Falls back to Summary when Description is empty
	if !strings.Contains(out, "A simple command") {
		t.Error("missing summary fallback in output")
	}
}

func TestPrintCommandHelpNoExamples(t *testing.T) {
	cmd := &Command{
		Name:    "bare",
		Summary: "Bare command",
		Run:     func([]string, io.Writer) error { return nil },
	}

	var buf bytes.Buffer
	printCommandHelp(&buf, cmd)
	out := buf.String()

	if strings.Contains(out, "Examples:") {
		t.Error("should not show Examples: section when there are no examples")
	}
}

func TestCommandsWithExamplesHaveDescriptions(t *testing.T) {
	for _, cmd := range commands {
		if len(cmd.Examples) > 0 && cmd.Description == "" {
			t.Errorf("command %q has examples but no description", cmd.Name)
		}
	}
}

func TestExamplesStartWithBw(t *testing.T) {
	for _, cmd := range commands {
		for _, ex := range cmd.Examples {
			if !strings.HasPrefix(ex.Cmd, "bw ") {
				t.Errorf("command %q example %q does not start with 'bw '", cmd.Name, ex.Cmd)
			}
		}
	}
}

func TestExpandAliases(t *testing.T) {
	flags := []Flag{
		{Long: "--priority", Short: "-p", Value: "N"},
		{Long: "--type", Short: "-t", Value: "TYPE"},
		{Long: "--json"},
	}

	raw := []string{"-p", "2", "-t", "bug", "--json", "title"}
	got := expandAliases(raw, flags)

	if got[0] != "--priority" {
		t.Errorf("got[0] = %q, want --priority", got[0])
	}
	if got[1] != "2" {
		t.Errorf("got[1] = %q, want 2", got[1])
	}
	if got[2] != "--type" {
		t.Errorf("got[2] = %q, want --type", got[2])
	}
	if got[4] != "--json" {
		t.Errorf("got[4] = %q, want --json (should be unchanged)", got[4])
	}
	if got[5] != "title" {
		t.Errorf("got[5] = %q, want title (should be unchanged)", got[5])
	}
}

func TestCommandValueFlags(t *testing.T) {
	// Verify that Command.valueFlags() returns the right set
	cmd := commandMap["create"]
	if cmd == nil {
		t.Fatal("create command not found")
	}
	vf := cmd.valueFlags()
	// --priority, --type, --assignee, --description should be value flags
	for _, name := range []string{"--priority", "--type", "--assignee", "--description"} {
		found := false
		for _, f := range vf {
			if f == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("valueFlags() missing %q for create command", name)
		}
	}
	// --json should NOT be a value flag
	for _, f := range vf {
		if f == "--json" {
			t.Error("--json should not be a value flag")
		}
	}
}

func TestPriorityHelpText(t *testing.T) {
	want := "Priority (0-4 or P0-P4, 0=highest)"
	for _, name := range []string{"create", "list", "update"} {
		cmd := commandMap[name]
		if cmd == nil {
			t.Fatalf("command %q not found", name)
		}
		found := false
		for _, f := range cmd.Flags {
			if f.Long == "--priority" {
				found = true
				if f.Help != want {
					t.Errorf("command %q --priority help = %q, want %q", name, f.Help, want)
				}
			}
		}
		if !found {
			t.Errorf("command %q missing --priority flag", name)
		}
	}
}
