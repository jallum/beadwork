package main

import (
	"bytes"
	"fmt"
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
	w := ColorWriter(&buf, 0)

	got := w.Style("hello", Red)
	want := "\033[31mhello\033[0m"
	if got != want {
		t.Errorf("ColorWriter.Style(Red) = %q, want %q", got, want)
	}
}

func TestColorWriterStyleMultiple(t *testing.T) {
	var buf bytes.Buffer
	w := ColorWriter(&buf, 0)

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
	w := ColorWriter(&buf, 0)

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

func TestPlainWriterWidth(t *testing.T) {
	var buf bytes.Buffer
	w := PlainWriter(&buf)
	if w.Width() != 0 {
		t.Errorf("PlainWriter.Width() = %d, want 0", w.Width())
	}
}

func TestColorWriterWidth(t *testing.T) {
	var buf bytes.Buffer
	w := ColorWriter(&buf, 80)
	if w.Width() != 80 {
		t.Errorf("ColorWriter.Width() = %d, want 80", w.Width())
	}
}

func TestPushPopBasic(t *testing.T) {
	var buf bytes.Buffer
	w := PlainWriter(&buf)

	fmt.Fprintln(w, "no indent")
	w.Push(2)
	fmt.Fprintln(w, "indented")
	w.Pop()
	fmt.Fprintln(w, "no indent again")

	want := "no indent\n  indented\nno indent again\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestPushPopNested(t *testing.T) {
	var buf bytes.Buffer
	w := PlainWriter(&buf)

	w.Push(2)
	fmt.Fprintln(w, "level 1")
	w.Push(2)
	fmt.Fprintln(w, "level 2")
	w.Pop()
	fmt.Fprintln(w, "back to 1")
	w.Pop()
	fmt.Fprintln(w, "back to 0")

	want := "  level 1\n    level 2\n  back to 1\nback to 0\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestPushPopMultiline(t *testing.T) {
	var buf bytes.Buffer
	w := PlainWriter(&buf)

	w.Push(2)
	fmt.Fprint(w, "line1\nline2\n")
	w.Pop()

	want := "  line1\n  line2\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestPushPopPartialWrite(t *testing.T) {
	var buf bytes.Buffer
	w := PlainWriter(&buf)

	w.Push(2)
	fmt.Fprint(w, "hello ")
	fmt.Fprintln(w, "world")
	w.Pop()

	want := "  hello world\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWidthWithIndent(t *testing.T) {
	var buf bytes.Buffer
	w := ColorWriter(&buf, 80)

	if w.Width() != 80 {
		t.Errorf("Width() = %d, want 80", w.Width())
	}
	w.Push(4)
	if w.Width() != 76 {
		t.Errorf("Width() after Push(4) = %d, want 76", w.Width())
	}
	w.Push(2)
	if w.Width() != 74 {
		t.Errorf("Width() after Push(4)+Push(2) = %d, want 74", w.Width())
	}
	w.Pop()
	if w.Width() != 76 {
		t.Errorf("Width() after Pop = %d, want 76", w.Width())
	}
	w.Pop()
	if w.Width() != 80 {
		t.Errorf("Width() after all Pops = %d, want 80", w.Width())
	}
}

func TestWidthClampToOne(t *testing.T) {
	var buf bytes.Buffer
	w := ColorWriter(&buf, 3)
	w.Push(10)
	if w.Width() != 1 {
		t.Errorf("Width() = %d, want 1 (clamped)", w.Width())
	}
}

func TestWidthZeroParentUnchanged(t *testing.T) {
	var buf bytes.Buffer
	w := PlainWriter(&buf) // Width()=0
	w.Push(4)
	if w.Width() != 0 {
		t.Errorf("Width() = %d, want 0 (plain writer stays 0)", w.Width())
	}
}

func TestPopOnEmptyStack(t *testing.T) {
	var buf bytes.Buffer
	w := PlainWriter(&buf)
	w.Pop() // should not panic
	fmt.Fprintln(w, "ok")
	if buf.String() != "ok\n" {
		t.Errorf("got %q after Pop on empty stack", buf.String())
	}
}
