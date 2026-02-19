package repo_test

import (
	"os"
	"testing"

	"github.com/j5n/beadwork/internal/testutil"
)

func TestGetConfigExistingKey(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// prefix is set during Init
	val, ok := env.Repo.GetConfig("prefix")
	if !ok {
		t.Fatal("expected prefix to exist in config")
	}
	if val != "test" {
		t.Errorf("prefix = %q, want %q", val, "test")
	}
}

func TestGetConfigMissingKey(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	_, ok := env.Repo.GetConfig("nonexistent")
	if ok {
		t.Error("expected ok=false for missing key")
	}
}

func TestSetConfigNewKey(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	if err := env.Repo.SetConfig("default.priority", "2"); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}

	val, ok := env.Repo.GetConfig("default.priority")
	if !ok {
		t.Fatal("expected default.priority to exist after set")
	}
	if val != "2" {
		t.Errorf("default.priority = %q, want %q", val, "2")
	}

	// Existing keys should be preserved
	val, ok = env.Repo.GetConfig("prefix")
	if !ok {
		t.Fatal("prefix should still exist")
	}
	if val != "test" {
		t.Errorf("prefix = %q, want %q", val, "test")
	}
}

func TestSetConfigOverwriteKey(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	if err := env.Repo.SetConfig("default.priority", "2"); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	if err := env.Repo.SetConfig("default.priority", "1"); err != nil {
		t.Fatalf("SetConfig overwrite: %v", err)
	}

	val, ok := env.Repo.GetConfig("default.priority")
	if !ok {
		t.Fatal("expected default.priority to exist")
	}
	if val != "1" {
		t.Errorf("default.priority = %q, want %q", val, "1")
	}
}

func TestListConfig(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	env.Repo.SetConfig("default.priority", "2")
	env.Repo.SetConfig("default.type", "bug")

	cfg := env.Repo.ListConfig()
	if len(cfg) != 3 {
		t.Errorf("got %d config entries, want 3 (prefix + 2 new)", len(cfg))
	}
	if cfg["prefix"] != "test" {
		t.Errorf("prefix = %q", cfg["prefix"])
	}
	if cfg["default.priority"] != "2" {
		t.Errorf("default.priority = %q", cfg["default.priority"])
	}
	if cfg["default.type"] != "bug" {
		t.Errorf("default.type = %q", cfg["default.type"])
	}
}

func TestListConfigEmpty(t *testing.T) {
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	// Remove the .bwconfig file to test empty case
	os.Remove(env.Repo.WorkTree + "/.bwconfig")

	cfg := env.Repo.ListConfig()
	if len(cfg) != 0 {
		t.Errorf("got %d config entries, want 0", len(cfg))
	}
}

func init() {
	os.Setenv("GIT_AUTHOR_NAME", "Test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@test.com")
	os.Setenv("GIT_COMMITTER_NAME", "Test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@test.com")
}
