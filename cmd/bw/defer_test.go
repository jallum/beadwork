package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func TestCmdDefer(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	iss, _ := env.Store.Create("Defer me", issue.CreateOpts{})
	env.Repo.Commit("create " + iss.ID)

	var buf bytes.Buffer
	err := cmdDefer([]string{iss.ID, "2027-06-01"}, PlainWriter(&buf))
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
	err := cmdDefer([]string{iss.ID, "2027-06-01", "--json"}, PlainWriter(&buf))
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
	err := cmdDefer([]string{iss.ID, "not-a-date"}, PlainWriter(&buf))
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
	err := cmdUndefer([]string{iss.ID}, PlainWriter(&buf))
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
	err := cmdUndefer([]string{iss.ID, "--json"}, PlainWriter(&buf))
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
	err := cmdCreate([]string{"Deferred task", "--defer", "2027-03-15"}, PlainWriter(&buf))
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
	err := cmdUpdate([]string{iss.ID, "--defer", "2027-09-01"}, PlainWriter(&buf))
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
	err := cmdList([]string{}, PlainWriter(&buf))
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
	err = cmdList([]string{"--deferred"}, PlainWriter(&buf))
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
	err := cmdShow([]string{iss.ID}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Deferred: 2027-04-15") {
		t.Errorf("output should contain 'Deferred: 2027-04-15': %q", out)
	}
	if !strings.Contains(out, "DEFERRED") {
		t.Errorf("output should contain DEFERRED status: %q", out)
	}
}

func TestCmdDeferNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdDefer([]string{}, PlainWriter(&buf))
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
	err := cmdDefer([]string{"bw-1234"}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for missing date arg")
	}
}

func TestCmdDeferNonExistentIssue(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdDefer([]string{"bw-0000", "2027-06-01"}, PlainWriter(&buf))
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
		"tomorrow",
		"2027-13-01",
		"2027-06-32",
		"",
	}
	for _, d := range invalidDates {
		var buf bytes.Buffer
		err := cmdDefer([]string{iss.ID, d}, PlainWriter(&buf))
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
	err := cmdDefer([]string{iss.ID, "2027-12-01"}, PlainWriter(&buf))
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
	err := cmdDefer([]string{iss.ID, "2027-06-01", "--unknown"}, PlainWriter(&buf))
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

func TestCmdUndeferNoArgs(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	var buf bytes.Buffer
	err := cmdUndefer([]string{}, PlainWriter(&buf))
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
	err := cmdUndefer([]string{"bw-0000"}, PlainWriter(&buf))
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
	err := cmdUndefer([]string{iss.ID}, PlainWriter(&buf))
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
	err := cmdUndefer([]string{iss.ID, "--unknown"}, PlainWriter(&buf))
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
	err := cmdUndefer([]string{iss.ID}, PlainWriter(&buf))
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
