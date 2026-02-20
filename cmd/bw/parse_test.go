package main

import (
	"strings"
	"testing"
)

// --- parseShowArgs ---

func TestParseShowArgs(t *testing.T) {
	a, err := parseShowArgs([]string{"bw-1234"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "bw-1234" {
		t.Errorf("ID = %q, want bw-1234", a.ID)
	}
	if a.JSON {
		t.Error("expected JSON = false")
	}
}

func TestParseShowArgsJSON(t *testing.T) {
	a, err := parseShowArgs([]string{"bw-1234", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "bw-1234" {
		t.Errorf("ID = %q", a.ID)
	}
	if !a.JSON {
		t.Error("expected JSON = true")
	}
}

func TestParseShowArgsMissingID(t *testing.T) {
	_, err := parseShowArgs([]string{})
	if err == nil {
		t.Error("expected error for missing id")
	}
}

// --- parseReopenArgs ---

func TestParseReopenArgs(t *testing.T) {
	a, err := parseReopenArgs([]string{"bw-5678"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "bw-5678" {
		t.Errorf("ID = %q", a.ID)
	}
	if a.JSON {
		t.Error("expected JSON = false")
	}
}

func TestParseReopenArgsJSON(t *testing.T) {
	a, err := parseReopenArgs([]string{"bw-5678", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.JSON {
		t.Error("expected JSON = true")
	}
}

func TestParseReopenArgsMissingID(t *testing.T) {
	_, err := parseReopenArgs([]string{})
	if err == nil {
		t.Error("expected error for missing id")
	}
}

// --- parseReadyArgs ---

func TestParseReadyArgs(t *testing.T) {
	a, err := parseReadyArgs([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if a.JSON {
		t.Error("expected JSON = false")
	}
}

func TestParseReadyArgsJSON(t *testing.T) {
	a, err := parseReadyArgs([]string{"--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.JSON {
		t.Error("expected JSON = true")
	}
}

// --- parseCloseArgs ---

func TestParseCloseArgs(t *testing.T) {
	a, err := parseCloseArgs([]string{"bw-abcd"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "bw-abcd" {
		t.Errorf("ID = %q", a.ID)
	}
	if a.Reason != "" {
		t.Errorf("Reason = %q, want empty", a.Reason)
	}
	if a.JSON {
		t.Error("expected JSON = false")
	}
}

func TestParseCloseArgsWithReason(t *testing.T) {
	a, err := parseCloseArgs([]string{"bw-abcd", "--reason", "duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "bw-abcd" {
		t.Errorf("ID = %q", a.ID)
	}
	if a.Reason != "duplicate" {
		t.Errorf("Reason = %q, want duplicate", a.Reason)
	}
}

func TestParseCloseArgsJSON(t *testing.T) {
	a, err := parseCloseArgs([]string{"bw-abcd", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.JSON {
		t.Error("expected JSON = true")
	}
}

func TestParseCloseArgsMissingID(t *testing.T) {
	_, err := parseCloseArgs([]string{})
	if err == nil {
		t.Error("expected error for missing id")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q, want usage message", err.Error())
	}
}

// --- parseCreateArgs ---

func TestParseCreateArgsSingleWord(t *testing.T) {
	a, err := parseCreateArgs([]string{"Fix"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Title != "Fix" {
		t.Errorf("Title = %q, want %q", a.Title, "Fix")
	}
	if a.JSON {
		t.Error("expected JSON = false")
	}
}

func TestParseCreateArgsQuotedMultiWord(t *testing.T) {
	a, err := parseCreateArgs([]string{"Fix the bug"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Title != "Fix the bug" {
		t.Errorf("Title = %q, want %q", a.Title, "Fix the bug")
	}
}

func TestParseCreateArgsExtraPositionalsError(t *testing.T) {
	_, err := parseCreateArgs([]string{"Fix", "the", "bug"})
	if err == nil {
		t.Fatal("expected error for extra positional args")
	}
	if !strings.Contains(err.Error(), "unexpected arguments") {
		t.Errorf("error = %q, want mention of unexpected arguments", err.Error())
	}
	if !strings.Contains(err.Error(), "the") || !strings.Contains(err.Error(), "bug") {
		t.Errorf("error = %q, want mention of extra args", err.Error())
	}
}

func TestParseCreateArgsWithFlags(t *testing.T) {
	a, err := parseCreateArgs([]string{"Bug report", "-p", "1", "-t", "bug", "-a", "alice", "-d", "desc text"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Title != "Bug report" {
		t.Errorf("Title = %q", a.Title)
	}
	if a.Priority == nil || *a.Priority != 1 {
		t.Errorf("Priority = %v, want 1", a.Priority)
	}
	if a.Type != "bug" {
		t.Errorf("Type = %q", a.Type)
	}
	if a.Assignee != "alice" {
		t.Errorf("Assignee = %q", a.Assignee)
	}
	if a.Description != "desc text" {
		t.Errorf("Description = %q", a.Description)
	}
}

func TestParseCreateArgsMissingTitle(t *testing.T) {
	_, err := parseCreateArgs([]string{"--priority", "1"})
	if err == nil {
		t.Error("expected error for missing title")
	}
}

func TestParseCreateArgsInvalidPriority(t *testing.T) {
	_, err := parseCreateArgs([]string{"Title", "--priority", "abc"})
	if err == nil {
		t.Error("expected error for non-numeric priority")
	}
}

func TestParseCreateArgsPrefixedPriority(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"P0", 0},
		{"P1", 1},
		{"P2", 2},
		{"P3", 3},
		{"P4", 4},
		{"p0", 0},
		{"p2", 2},
		{"p4", 4},
		{"0", 0},
		{"4", 4},
	}
	for _, tt := range tests {
		a, err := parseCreateArgs([]string{"Title", "--priority", tt.input})
		if err != nil {
			t.Errorf("parseCreateArgs(--priority %q): %v", tt.input, err)
			continue
		}
		if a.Priority == nil || *a.Priority != tt.want {
			t.Errorf("--priority %q: Priority=%v, want %d", tt.input, a.Priority, tt.want)
		}
	}
}

func TestParseCreateArgsPriorityOutOfRange(t *testing.T) {
	invalid := []string{"5", "-1", "P5", "P6", "high", "P-1", "10"}
	for _, v := range invalid {
		_, err := parseCreateArgs([]string{"Title", "--priority", v})
		if err == nil {
			t.Errorf("expected error for --priority %q", v)
		}
		if err != nil && !strings.Contains(err.Error(), "invalid priority") {
			t.Errorf("--priority %q: error = %q, want 'invalid priority' message", v, err.Error())
		}
	}
}

func TestParseUpdateArgsPrefixedPriority(t *testing.T) {
	a, err := parseUpdateArgs([]string{"bw-1234", "--priority", "P0"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Priority == nil || *a.Priority != 0 {
		t.Errorf("Priority=%v, want 0", a.Priority)
	}
}

func TestParseUpdateArgsPriorityOutOfRange(t *testing.T) {
	invalid := []string{"5", "-1", "P5", "high"}
	for _, v := range invalid {
		_, err := parseUpdateArgs([]string{"bw-1234", "--priority", v})
		if err == nil {
			t.Errorf("expected error for --priority %q", v)
		}
	}
}

func TestParseCreateArgsJSON(t *testing.T) {
	a, err := parseCreateArgs([]string{"Title", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.JSON {
		t.Error("expected JSON = true")
	}
}

func TestParseCreateArgsWithLabels(t *testing.T) {
	a, err := parseCreateArgs([]string{"Title", "--labels", "frontend,urgent"})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Labels) != 2 || a.Labels[0] != "frontend" || a.Labels[1] != "urgent" {
		t.Errorf("Labels = %v, want [frontend urgent]", a.Labels)
	}
}

func TestParseCreateArgsWithLabelsAlias(t *testing.T) {
	a, err := parseCreateArgs([]string{"Title", "-l", "backend"})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Labels) != 1 || a.Labels[0] != "backend" {
		t.Errorf("Labels = %v, want [backend]", a.Labels)
	}
}

// --- parseExportArgs ---

func TestParseExportArgs(t *testing.T) {
	a, err := parseExportArgs([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != "" {
		t.Errorf("Status = %q, want empty", a.Status)
	}
}

func TestParseExportArgsWithStatus(t *testing.T) {
	a, err := parseExportArgs([]string{"--status", "open"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != "open" {
		t.Errorf("Status = %q, want open", a.Status)
	}
}

func TestParseExportArgsAlias(t *testing.T) {
	a, err := parseExportArgs([]string{"-s", "closed"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != "closed" {
		t.Errorf("Status = %q, want closed", a.Status)
	}
}

// --- parseImportArgs ---

func TestParseImportArgs(t *testing.T) {
	a, err := parseImportArgs([]string{"/tmp/issues.jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	if a.FilePath != "/tmp/issues.jsonl" {
		t.Errorf("FilePath = %q", a.FilePath)
	}
	if a.DryRun {
		t.Error("expected DryRun = false")
	}
}

func TestParseImportArgsDryRun(t *testing.T) {
	a, err := parseImportArgs([]string{"/tmp/issues.jsonl", "--dry-run"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.DryRun {
		t.Error("expected DryRun = true")
	}
}

func TestParseImportArgsMissingFile(t *testing.T) {
	_, err := parseImportArgs([]string{})
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// --- parseGraphArgs ---

func TestParseGraphArgsWithID(t *testing.T) {
	a, err := parseGraphArgs([]string{"bw-1234"})
	if err != nil {
		t.Fatal(err)
	}
	if a.RootID != "bw-1234" {
		t.Errorf("RootID = %q", a.RootID)
	}
	if a.All {
		t.Error("expected All = false")
	}
}

func TestParseGraphArgsAll(t *testing.T) {
	a, err := parseGraphArgs([]string{"--all"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.All {
		t.Error("expected All = true")
	}
	if a.RootID != "" {
		t.Errorf("RootID = %q, want empty", a.RootID)
	}
}

func TestParseGraphArgsMissing(t *testing.T) {
	_, err := parseGraphArgs([]string{})
	if err == nil {
		t.Error("expected error when no ID and no --all")
	}
}

func TestParseGraphArgsJSON(t *testing.T) {
	a, err := parseGraphArgs([]string{"--all", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.JSON {
		t.Error("expected JSON = true")
	}
}

// --- parseListArgs ---

func TestParseListArgsDefaults(t *testing.T) {
	a, err := parseListArgs([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Limit != 10 {
		t.Errorf("Limit = %d, want 10", a.Limit)
	}
	if a.All {
		t.Error("expected All = false")
	}
}

func TestParseListArgsFilters(t *testing.T) {
	a, err := parseListArgs([]string{"--status", "open", "-p", "1", "-t", "bug", "-a", "alice", "--label", "urgent"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != "open" {
		t.Errorf("Status = %q", a.Status)
	}
	if a.Priority == nil || *a.Priority != 1 {
		t.Errorf("Priority = %v, want 1", a.Priority)
	}
	if a.Type != "bug" {
		t.Errorf("Type = %q", a.Type)
	}
	if a.Assignee != "alice" {
		t.Errorf("Assignee = %q", a.Assignee)
	}
	if a.Label != "urgent" {
		t.Errorf("Label = %q", a.Label)
	}
}

func TestParseListArgsLimit(t *testing.T) {
	a, err := parseListArgs([]string{"--limit", "5"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Limit != 5 {
		t.Errorf("Limit = %d, want 5", a.Limit)
	}
	if !a.LimitSet {
		t.Error("expected LimitSet = true")
	}
}

func TestParseListArgsAll(t *testing.T) {
	a, err := parseListArgs([]string{"--all"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.All {
		t.Error("expected All = true")
	}
}

// --- parseUpdateArgs ---

func TestParseUpdateArgs(t *testing.T) {
	a, err := parseUpdateArgs([]string{"bw-1234", "--title", "New title", "--priority", "2"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "bw-1234" {
		t.Errorf("ID = %q", a.ID)
	}
	if a.Title != "New title" || !a.TitleSet {
		t.Errorf("Title = %q, TitleSet = %v", a.Title, a.TitleSet)
	}
	if a.Priority == nil || *a.Priority != 2 {
		t.Errorf("Priority = %v, want 2", a.Priority)
	}
}

func TestParseUpdateArgsAllFields(t *testing.T) {
	a, err := parseUpdateArgs([]string{"bw-1234",
		"--title", "T", "--description", "D", "--priority", "1",
		"--assignee", "bob", "--type", "bug", "--status", "in_progress",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !a.TitleSet || !a.DescSet || a.Priority == nil || !a.AssigneeSet || !a.TypeSet || !a.StatusSet {
		t.Error("expected all fields to be set")
	}
	if a.Description != "D" {
		t.Errorf("Description = %q", a.Description)
	}
	if a.Assignee != "bob" {
		t.Errorf("Assignee = %q", a.Assignee)
	}
	if a.Type != "bug" {
		t.Errorf("Type = %q", a.Type)
	}
	if a.Status != "in_progress" {
		t.Errorf("Status = %q", a.Status)
	}
}

func TestParseUpdateArgsMissingID(t *testing.T) {
	_, err := parseUpdateArgs([]string{})
	if err == nil {
		t.Error("expected error for missing id")
	}
}

func TestParseUpdateArgsInvalidPriority(t *testing.T) {
	_, err := parseUpdateArgs([]string{"bw-1234", "--priority", "abc"})
	if err == nil {
		t.Error("expected error for non-numeric priority")
	}
}

func TestParseUpdateArgsAliases(t *testing.T) {
	a, err := parseUpdateArgs([]string{"bw-1234", "-p", "3", "-a", "carol", "-t", "task"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Priority == nil || *a.Priority != 3 {
		t.Errorf("Priority = %v, want 3", a.Priority)
	}
	if a.Assignee != "carol" {
		t.Errorf("Assignee = %q", a.Assignee)
	}
	if a.Type != "task" {
		t.Errorf("Type = %q", a.Type)
	}
}

// --- parseLabelArgs ---

func TestParseLabelArgs(t *testing.T) {
	a, err := parseLabelArgs([]string{"bw-1234", "+bug", "+urgent"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "bw-1234" {
		t.Errorf("ID = %q", a.ID)
	}
	if len(a.Add) != 2 || a.Add[0] != "bug" || a.Add[1] != "urgent" {
		t.Errorf("Add = %v", a.Add)
	}
}

func TestParseLabelArgsRemove(t *testing.T) {
	a, err := parseLabelArgs([]string{"bw-1234", "-bug"})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Remove) != 1 || a.Remove[0] != "bug" {
		t.Errorf("Remove = %v", a.Remove)
	}
}

func TestParseLabelArgsBareAdd(t *testing.T) {
	a, err := parseLabelArgs([]string{"bw-1234", "feature"})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Add) != 1 || a.Add[0] != "feature" {
		t.Errorf("Add = %v, want [feature]", a.Add)
	}
}

func TestParseLabelArgsJSON(t *testing.T) {
	a, err := parseLabelArgs([]string{"bw-1234", "+bug", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.JSON {
		t.Error("expected JSON = true")
	}
	if len(a.Add) != 1 || a.Add[0] != "bug" {
		t.Errorf("Add = %v", a.Add)
	}
}

func TestParseLabelArgsMissing(t *testing.T) {
	_, err := parseLabelArgs([]string{"bw-1234"})
	if err == nil {
		t.Error("expected error for missing label args")
	}
}

func TestParseLabelArgsMissingID(t *testing.T) {
	_, err := parseLabelArgs([]string{})
	if err == nil {
		t.Error("expected error for missing args")
	}
}

func TestParseLabelArgsUnknownFlag(t *testing.T) {
	_, err := parseLabelArgs([]string{"bw-1234", "+bug", "--verbose"})
	if err == nil {
		t.Error("expected error for unknown flag --verbose")
	}
	if err != nil && !strings.Contains(err.Error(), "unknown flag: --verbose") {
		t.Errorf("error = %q, want 'unknown flag: --verbose'", err.Error())
	}
}

// --- parseDepArgs / parseDepAddArgs / parseDepRemoveArgs ---

func TestParseDepArgs(t *testing.T) {
	a, err := parseDepArgs([]string{"add", "bw-aaaa", "blocks", "bw-bbbb"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Subcmd != "add" || a.BlockerID != "bw-aaaa" || a.BlockedID != "bw-bbbb" {
		t.Errorf("got %+v", a)
	}
}

func TestParseDepArgsMissing(t *testing.T) {
	_, err := parseDepArgs([]string{})
	if err == nil {
		t.Error("expected error for missing subcommand")
	}
}

func TestParseDepArgsUnknown(t *testing.T) {
	_, err := parseDepArgs([]string{"list"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestParseDepAddArgs(t *testing.T) {
	a, err := parseDepAddArgs([]string{"bw-aaaa", "blocks", "bw-bbbb"})
	if err != nil {
		t.Fatal(err)
	}
	if a.BlockerID != "bw-aaaa" {
		t.Errorf("BlockerID = %q", a.BlockerID)
	}
	if a.BlockedID != "bw-bbbb" {
		t.Errorf("BlockedID = %q", a.BlockedID)
	}
}

func TestParseDepAddArgsBadSyntax(t *testing.T) {
	_, err := parseDepAddArgs([]string{"a", "b"})
	if err == nil {
		t.Error("expected error for bad syntax")
	}
}

func TestParseDepAddArgsMissing(t *testing.T) {
	_, err := parseDepAddArgs([]string{})
	if err == nil {
		t.Error("expected error for missing args")
	}
}

func TestParseDepRemoveArgs(t *testing.T) {
	a, err := parseDepRemoveArgs([]string{"bw-aaaa", "blocks", "bw-bbbb"})
	if err != nil {
		t.Fatal(err)
	}
	if a.BlockerID != "bw-aaaa" || a.BlockedID != "bw-bbbb" {
		t.Errorf("got %+v", a)
	}
}

func TestParseDepRemoveArgsBadSyntax(t *testing.T) {
	_, err := parseDepRemoveArgs([]string{"a", "b"})
	if err == nil {
		t.Error("expected error for bad syntax")
	}
}

// --- parseConfigArgs ---

func TestParseConfigArgsGet(t *testing.T) {
	a, err := parseConfigArgs([]string{"get", "default.priority"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Subcmd != "get" || a.Key != "default.priority" {
		t.Errorf("got %+v", a)
	}
}

func TestParseConfigArgsSet(t *testing.T) {
	a, err := parseConfigArgs([]string{"set", "default.priority", "2"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Subcmd != "set" || a.Key != "default.priority" || a.Value != "2" {
		t.Errorf("got %+v", a)
	}
}

func TestParseConfigArgsList(t *testing.T) {
	a, err := parseConfigArgs([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Subcmd != "list" {
		t.Errorf("Subcmd = %q", a.Subcmd)
	}
}

func TestParseConfigArgsMissing(t *testing.T) {
	_, err := parseConfigArgs([]string{})
	if err == nil {
		t.Error("expected error for missing subcommand")
	}
}

func TestParseConfigArgsUnknown(t *testing.T) {
	_, err := parseConfigArgs([]string{"delete"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestParseConfigArgsGetNoKey(t *testing.T) {
	_, err := parseConfigArgs([]string{"get"})
	if err == nil {
		t.Error("expected error for get without key")
	}
}

func TestParseConfigArgsSetNoValue(t *testing.T) {
	_, err := parseConfigArgs([]string{"set", "key"})
	if err == nil {
		t.Error("expected error for set without value")
	}
}

// --- parseInitArgs ---

func TestParseInitArgs(t *testing.T) {
	a, err := parseInitArgs([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Prefix != "" || a.Force {
		t.Errorf("got %+v, want empty defaults", a)
	}
}

func TestParseInitArgsWithFlags(t *testing.T) {
	a, err := parseInitArgs([]string{"--prefix", "test", "--force"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Prefix != "test" {
		t.Errorf("Prefix = %q", a.Prefix)
	}
	if !a.Force {
		t.Error("expected Force = true")
	}
}

// --- parseUpgradeArgs ---

func TestParseUpgradeArgs(t *testing.T) {
	a, err := parseUpgradeArgs([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Check || a.Yes {
		t.Errorf("got %+v, want empty defaults", a)
	}
}

func TestParseUpgradeArgsCheck(t *testing.T) {
	a, err := parseUpgradeArgs([]string{"--check"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.Check {
		t.Error("expected Check = true")
	}
}

func TestParseUpgradeArgsYes(t *testing.T) {
	a, err := parseUpgradeArgs([]string{"--yes"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.Yes {
		t.Error("expected Yes = true")
	}
}

// --- ParseArgs unknown flag rejection ---

func TestParseArgsUnknownFlag(t *testing.T) {
	_, err := ParseArgs([]string{"--foo"}, nil, []string{"--json"})
	if err == nil {
		t.Error("expected error for unknown flag --foo")
	}
	if err != nil && !strings.Contains(err.Error(), "unknown flag: --foo") {
		t.Errorf("error = %q, want 'unknown flag: --foo'", err.Error())
	}
}

func TestParseArgsUnknownValueFlag(t *testing.T) {
	_, err := ParseArgs([]string{"--labels", "bug"}, nil, []string{"--json"})
	if err == nil {
		t.Error("expected error for unknown flag --labels")
	}
	if err != nil && !strings.Contains(err.Error(), "unknown flag: --labels") {
		t.Errorf("error = %q, want 'unknown flag: --labels'", err.Error())
	}
}

func TestParseArgsKnownFlagsPass(t *testing.T) {
	a, err := ParseArgs(
		[]string{"positional", "--json", "--status", "open", "-p", "2"},
		[]string{"--status", "--priority"},
		[]string{"--json"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.JSON() {
		t.Error("expected JSON() to be true")
	}
	if a.String("--status") != "open" {
		t.Errorf("status = %q, want open", a.String("--status"))
	}
	if a.Int("--priority") != 2 {
		t.Errorf("priority = %d, want 2", a.Int("--priority"))
	}
	if a.PosFirst() != "positional" {
		t.Errorf("PosFirst() = %q, want positional", a.PosFirst())
	}
}

// --- Command-level unknown flag rejection ---

func TestParseCreateArgsUnknownFlag(t *testing.T) {
	_, err := parseCreateArgs([]string{"My task", "--verbose"})
	if err == nil {
		t.Error("expected error for unknown flag --verbose")
	}
	if err != nil && !strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("error = %q, want unknown flag message", err.Error())
	}
}

func TestParseListArgsUnknownFlag(t *testing.T) {
	_, err := parseListArgs([]string{"--foo"})
	if err == nil {
		t.Error("expected error for unknown flag --foo")
	}
}

func TestParseUpdateArgsUnknownFlag(t *testing.T) {
	_, err := parseUpdateArgs([]string{"bw-1234", "--unknown", "val"})
	if err == nil {
		t.Error("expected error for unknown flag --unknown")
	}
}

func TestParseShowArgsUnknownFlag(t *testing.T) {
	_, err := parseShowArgs([]string{"bw-1234", "--verbose"})
	if err == nil {
		t.Error("expected error for unknown flag --verbose")
	}
}

func TestParseCloseArgsUnknownFlag(t *testing.T) {
	_, err := parseCloseArgs([]string{"bw-1234", "--force"})
	if err == nil {
		t.Error("expected error for unknown flag --force")
	}
}

func TestParseReadyArgsUnknownFlag(t *testing.T) {
	_, err := parseReadyArgs([]string{"--verbose"})
	if err == nil {
		t.Error("expected error for unknown flag --verbose")
	}
}

func TestParseImportArgsUnknownFlag(t *testing.T) {
	_, err := parseImportArgs([]string{"file.jsonl", "--force"})
	if err == nil {
		t.Error("expected error for unknown flag --force")
	}
}

func TestParseInitArgsUnknownFlag(t *testing.T) {
	_, err := parseInitArgs([]string{"--verbose"})
	if err == nil {
		t.Error("expected error for unknown flag --verbose")
	}
}
