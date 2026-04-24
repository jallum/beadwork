package registry

import (
	"testing"
)

func TestDefaultPathRespectsEnv(t *testing.T) {
	t.Setenv("BW_REGISTRY", "/custom/path")
	if got := DefaultPath(); got != "/custom/path" {
		t.Errorf("DefaultPath() = %q, want /custom/path", got)
	}
}

func TestDefaultPathFallsBackToHome(t *testing.T) {
	t.Setenv("BW_REGISTRY", "")
	got := DefaultPath()
	if got == "" || got == defaultFileName {
		t.Errorf("DefaultPath() = %q, expected a home-based path", got)
	}
}
