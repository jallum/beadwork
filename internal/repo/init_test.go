package repo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/j5n/beadwork/internal/repo"
)

func TestForceReinit(t *testing.T) {
	dir := t.TempDir()

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	r, err := repo.FindRepo()
	if err != nil {
		t.Fatalf("FindRepo: %v", err)
	}

	// First init
	if err := r.Init("old"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if r.Prefix != "old" {
		t.Fatalf("prefix = %q, want old", r.Prefix)
	}

	// Regular init should fail
	r2, _ := repo.FindRepo()
	if err := r2.Init("new"); err == nil {
		t.Fatal("Init should fail when already initialized")
	}

	// Force reinit with new prefix
	r3, _ := repo.FindRepo()
	if err := r3.ForceReinit("new"); err != nil {
		t.Fatalf("ForceReinit: %v", err)
	}
	if r3.Prefix != "new" {
		t.Errorf("prefix = %q, want new", r3.Prefix)
	}
	if !r3.IsInitialized() {
		t.Error("should be initialized after force reinit")
	}

	// Worktree should exist with skeleton
	if _, err := os.Stat(filepath.Join(r3.WorkTree, "issues")); err != nil {
		t.Error("issues dir should exist after reinit")
	}
}

func TestForceReinitKeepsPrefix(t *testing.T) {
	dir := t.TempDir()

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	r, _ := repo.FindRepo()
	r.Init("keep")

	// Force reinit with empty prefix should derive a new one (not keep old)
	r2, _ := repo.FindRepo()
	if err := r2.ForceReinit(""); err != nil {
		t.Fatalf("ForceReinit: %v", err)
	}
	// Should derive prefix from dir name, not be empty
	if r2.Prefix == "" {
		t.Error("prefix should not be empty after reinit")
	}
}

func TestDerivePrefixLength(t *testing.T) {
	// Create a repo in a directory with a long name
	base := t.TempDir()
	dir := filepath.Join(base, "MyLongProjectName")
	os.Mkdir(dir, 0755)

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	r, _ := repo.FindRepo()
	if err := r.Init(""); err != nil {
		t.Fatalf("Init: %v", err)
	}
	// Should truncate to 8 chars max
	if len(r.Prefix) > 8 {
		t.Errorf("derived prefix %q longer than 8 chars", r.Prefix)
	}
	// Should be "MyLongPr" (8 chars, preserving case)
	if r.Prefix != "MyLongPr" {
		t.Errorf("prefix = %q, want MyLongPr", r.Prefix)
	}
}

func TestDerivePrefixPreservesCase(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "MyApp")
	os.Mkdir(dir, 0755)

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	r, _ := repo.FindRepo()
	if err := r.Init(""); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if r.Prefix != "MyApp" {
		t.Errorf("prefix = %q, want MyApp", r.Prefix)
	}
}

func TestDerivePrefixStripsInvalidChars(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "my.cool app")
	os.Mkdir(dir, 0755)

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	r, _ := repo.FindRepo()
	if err := r.Init(""); err != nil {
		t.Fatalf("Init: %v", err)
	}
	// Dots and spaces stripped, should be "mycoolap" (8 char truncation of "mycoolapp")
	if r.Prefix != "mycoolap" {
		t.Errorf("prefix = %q, want mycoolap", r.Prefix)
	}
}

func TestValidatePrefix(t *testing.T) {
	tests := []struct {
		prefix string
		ok     bool
	}{
		{"bw", true},
		{"myapp", true},
		{"MY-APP", true},
		{"my_app", true},
		{"a", true},
		{"abcdefghijklmnop", true},  // 16 chars, max
		{"abcdefghijklmnopq", false}, // 17 chars, too long
		{"", true},                   // empty is ok (will be derived)
		{"has space", false},
		{"has/slash", false},
		{"has.dot", false},
		{"café", false},
		{"a@b", false},
	}
	for _, tt := range tests {
		err := repo.ValidatePrefix(tt.prefix)
		if tt.ok && err != nil {
			t.Errorf("ValidatePrefix(%q) = %v, want nil", tt.prefix, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("ValidatePrefix(%q) = nil, want error", tt.prefix)
		}
	}
}
