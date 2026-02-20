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
