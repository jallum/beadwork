package repo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jallum/beadwork/internal/repo"
)

func TestUserNameFromRepoConfig(t *testing.T) {
	dir := t.TempDir()

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Repo User")
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

	got := r.UserName()
	if got != "Repo User" {
		t.Errorf("UserName() = %q, want %q", got, "Repo User")
	}
}

func TestUserNameUnsetFallback(t *testing.T) {
	dir := t.TempDir()

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	// Do NOT set user.name in local config.
	// The init() in config_test.go sets GIT_COMMITTER_NAME env var
	// so commits succeed even without local user.name.
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

	got := r.UserName()
	// When local user.name is unset, ConfigScoped reads global config.
	// The test runner's global config may or may not have user.name.
	// Either way, the method should return a non-empty string (either
	// the global user.name or "unknown").
	if got == "" {
		t.Error("UserName() should not return empty string")
	}
}

func TestUserNameReturnsNonEmpty(t *testing.T) {
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

	got := r.UserName()
	if got == "" {
		t.Error("UserName() should not return empty string")
	}
	if got == "unknown" {
		t.Error("UserName() should not be 'unknown' when user.name is set")
	}
}
