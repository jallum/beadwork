package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPlainWriterStyle(t *testing.T) {
	var buf bytes.Buffer
	w := PlainWriter(&buf)

	got := w.Style("hello", Bold, Red)
	if got != "hello" {
		t.Errorf("PlainWriter.Style() = %q, want %q", got, "hello")
	}
}

func TestPlainWriterStyleNoArgs(t *testing.T) {
	var buf bytes.Buffer
	w := PlainWriter(&buf)

	got := w.Style("hello")
	if got != "hello" {
		t.Errorf("PlainWriter.Style() = %q, want %q", got, "hello")
	}
}

func TestColorWriterStyle(t *testing.T) {
	var buf bytes.Buffer
	w := ColorWriter(&buf)

	got := w.Style("hello", Red)
	want := "\033[31mhello\033[0m"
	if got != want {
		t.Errorf("ColorWriter.Style(Red) = %q, want %q", got, want)
	}
}

func TestColorWriterStyleMultiple(t *testing.T) {
	var buf bytes.Buffer
	w := ColorWriter(&buf)

	got := w.Style("hello", Bold, Red)
	if !strings.HasPrefix(got, "\033[1m\033[31m") {
		t.Errorf("ColorWriter.Style(Bold, Red) = %q, want prefix \\033[1m\\033[31m", got)
	}
	if !strings.HasSuffix(got, "\033[0m") {
		t.Errorf("ColorWriter.Style(Bold, Red) = %q, want suffix \\033[0m", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("ColorWriter.Style(Bold, Red) = %q, want to contain 'hello'", got)
	}
}

func TestColorWriterStyleNoArgs(t *testing.T) {
	var buf bytes.Buffer
	w := ColorWriter(&buf)

	got := w.Style("hello")
	if got != "hello" {
		t.Errorf("ColorWriter.Style() with no styles = %q, want %q", got, "hello")
	}
}

func TestPriorityStyle(t *testing.T) {
	expected := map[int]Style{
		0: BrightRed,
		1: Red,
		2: Yellow,
		3: Cyan,
		4: Dim,
	}
	for p, want := range expected {
		got := PriorityStyle(p)
		if got != want {
			t.Errorf("PriorityStyle(%d) = %d, want %d", p, got, want)
		}
	}

	// Unknown priority falls back to Dim
	got := PriorityStyle(99)
	if got != Dim {
		t.Errorf("PriorityStyle(99) = %d, want Dim(%d)", got, Dim)
	}
}
