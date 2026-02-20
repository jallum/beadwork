package main

import (
	"testing"
)

func TestHasFlag(t *testing.T) {
	tests := []struct {
		args []string
		flag string
		want bool
	}{
		{[]string{"--yes"}, "--yes", true},
		{[]string{"show", "id", "--json"}, "--json", true},
		{[]string{"show", "id"}, "--json", false},
		{[]string{}, "--json", false},
		{[]string{"--json", "--all"}, "--all", true},
		{[]string{"--json", "--all"}, "--verbose", false},
	}
	for _, tt := range tests {
		got := hasFlag(tt.args, tt.flag)
		if got != tt.want {
			t.Errorf("hasFlag(%v, %q) = %v, want %v", tt.args, tt.flag, got, tt.want)
		}
	}
}
