package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Path() != path {
		t.Errorf("Path() = %q, want %q", cfg.Path(), path)
	}
	if len(cfg.Data()) != 0 {
		t.Errorf("Data = %v, want empty", cfg.Data())
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yml")
	cfg, _ := Load(path)
	cfg.Set("registry.auto_register", true)
	cfg.Set("registry.repos", []string{"/a", "/b"})

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cfg2, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg2.Bool("registry.auto_register") {
		t.Error("auto_register not preserved")
	}
	repos := cfg2.StringSlice("registry.repos")
	if len(repos) != 2 || repos[0] != "/a" || repos[1] != "/b" {
		t.Errorf("repos = %v", repos)
	}
}

func TestGetSetNested(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yml")
	cfg, _ := Load(path)
	cfg.Set("a.b.c", "deep")

	if cfg.String("a.b.c") != "deep" {
		t.Errorf("String(a.b.c) = %q", cfg.String("a.b.c"))
	}
	if cfg.String("a.b.missing") != "" {
		t.Error("missing key should return empty string")
	}
}

func TestUnknownFieldPreservation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yml")
	os.WriteFile(path, []byte("future_field: hello\nregistry:\n  repos: []\n"), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Set("registry.auto_register", true)
	cfg.Save()

	cfg2, _ := Load(path)
	if cfg2.String("future_field") != "hello" {
		t.Error("future_field not preserved")
	}
}

func TestDefaultPathRespectsEnv(t *testing.T) {
	t.Setenv("BW_CONFIG", "/custom/path")
	if got := DefaultPath(); got != "/custom/path" {
		t.Errorf("DefaultPath() = %q", got)
	}
}

func TestSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yml")
	cfg, _ := Load(path)
	cfg.Set("registry.repos", []string{"/a"})
	cfg.Set("registry.auto_register", true)

	sec := cfg.Section("registry")
	if sec == nil {
		t.Fatal("Section(registry) = nil")
	}
	if _, ok := sec["repos"]; !ok {
		t.Error("section missing repos key")
	}
}
