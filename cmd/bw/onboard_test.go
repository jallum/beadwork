package main

import (
	"bytes"
	"testing"
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
