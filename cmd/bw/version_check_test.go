package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jallum/beadwork/internal/issue"
	"github.com/jallum/beadwork/internal/testutil"
)

func mockVersionCheck(t *testing.T) {
	t.Helper()
	origNow := vcheckNow
	origFetch := vcheckFetchRelease
	origChangelog := vcheckFetchChangelog
	origCacheDir := vcheckCacheDir
	origVersion := vcheckCurrentVersion
	t.Cleanup(func() {
		vcheckNow = origNow
		vcheckFetchRelease = origFetch
		vcheckFetchChangelog = origChangelog
		vcheckCacheDir = origCacheDir
		vcheckCurrentVersion = origVersion
	})
	// Default: changelog fetch fails silently
	vcheckFetchChangelog = func(version string) (string, error) {
		return "", fmt.Errorf("mock: no changelog")
	}
}

func TestCheckForNewerVersion_CacheMiss(t *testing.T) {
	mockVersionCheck(t)
	dir := t.TempDir()
	vcheckCacheDir = func() string { return dir }
	vcheckCurrentVersion = func() string { return "0.9.0" }
	vcheckNow = func() time.Time { return time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC) }

	fetched := false
	vcheckFetchRelease = func() (*ghRelease, error) {
		fetched = true
		return &ghRelease{TagName: "v1.0.0"}, nil
	}

	got := checkForNewerVersion()
	if !fetched {
		t.Error("expected API call on cache miss")
	}
	if got != "1.0.0" {
		t.Errorf("got %q, want %q", got, "1.0.0")
	}

	// Verify cache was written
	data, err := os.ReadFile(filepath.Join(dir, "version-check.json"))
	if err != nil {
		t.Fatalf("cache not written: %v", err)
	}
	var c versionCache
	json.Unmarshal(data, &c)
	if c.LatestVersion != "1.0.0" {
		t.Errorf("cached version = %q, want %q", c.LatestVersion, "1.0.0")
	}
}

func TestCheckForNewerVersion_CacheHitSkipsAPI(t *testing.T) {
	mockVersionCheck(t)
	dir := t.TempDir()
	vcheckCacheDir = func() string { return dir }
	vcheckCurrentVersion = func() string { return "0.9.0" }

	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	vcheckNow = func() time.Time { return now }

	// Pre-populate cache from 1 hour ago
	cache := versionCache{
		LastCheck:     now.Add(-1 * time.Hour).Format(time.RFC3339),
		LatestVersion: "1.0.0",
	}
	data, _ := json.Marshal(cache)
	os.WriteFile(filepath.Join(dir, "version-check.json"), data, 0644)

	fetched := false
	vcheckFetchRelease = func() (*ghRelease, error) {
		fetched = true
		return &ghRelease{TagName: "v1.0.0"}, nil
	}

	got := checkForNewerVersion()
	if fetched {
		t.Error("should not call API when cache is fresh")
	}
	if got != "1.0.0" {
		t.Errorf("got %q, want %q from cache", got, "1.0.0")
	}
}

func TestCheckForNewerVersion_UpToDate(t *testing.T) {
	mockVersionCheck(t)
	dir := t.TempDir()
	vcheckCacheDir = func() string { return dir }
	vcheckCurrentVersion = func() string { return "1.0.0" }
	vcheckNow = func() time.Time { return time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC) }
	vcheckFetchRelease = func() (*ghRelease, error) {
		return &ghRelease{TagName: "v1.0.0"}, nil
	}

	got := checkForNewerVersion()
	if got != "" {
		t.Errorf("got %q, want empty (up to date)", got)
	}
}

func TestCheckForNewerVersion_APIFailureWritesTimestamp(t *testing.T) {
	mockVersionCheck(t)
	dir := t.TempDir()
	vcheckCacheDir = func() string { return dir }
	vcheckCurrentVersion = func() string { return "0.9.0" }
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	vcheckNow = func() time.Time { return now }
	vcheckFetchRelease = func() (*ghRelease, error) {
		return nil, fmt.Errorf("network timeout")
	}

	got := checkForNewerVersion()
	if got != "" {
		t.Errorf("got %q, want empty on failure", got)
	}

	// Cache timestamp should still be written
	data, err := os.ReadFile(filepath.Join(dir, "version-check.json"))
	if err != nil {
		t.Fatal("cache not written after failure")
	}
	var c versionCache
	json.Unmarshal(data, &c)
	if c.LastCheck == "" {
		t.Error("lastCheck not set after failure")
	}

	// Next call within 24h should NOT retry
	fetched := false
	vcheckFetchRelease = func() (*ghRelease, error) {
		fetched = true
		return nil, fmt.Errorf("should not be called")
	}
	checkForNewerVersion()
	if fetched {
		t.Error("should not retry API after recent failure")
	}
}

func TestCheckForNewerVersion_StaleCache(t *testing.T) {
	mockVersionCheck(t)
	dir := t.TempDir()
	vcheckCacheDir = func() string { return dir }
	vcheckCurrentVersion = func() string { return "0.9.0" }

	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	vcheckNow = func() time.Time { return now }

	// Pre-populate cache from 25 hours ago
	cache := versionCache{
		LastCheck:     now.Add(-25 * time.Hour).Format(time.RFC3339),
		LatestVersion: "1.0.0",
	}
	data, _ := json.Marshal(cache)
	os.WriteFile(filepath.Join(dir, "version-check.json"), data, 0644)

	fetched := false
	vcheckFetchRelease = func() (*ghRelease, error) {
		fetched = true
		return &ghRelease{TagName: "v1.1.0"}, nil
	}

	got := checkForNewerVersion()
	if !fetched {
		t.Error("expected API call when cache is stale")
	}
	if got != "1.1.0" {
		t.Errorf("got %q, want %q", got, "1.1.0")
	}
}

func TestEnsureUpgradeBead_CreatesNew(t *testing.T) {
	mockVersionCheck(t)
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	vcheckCurrentVersion = func() string { return "0.9.0" }

	var buf bytes.Buffer
	ensureUpgradeBead(env.Store, "1.0.0", PlainWriter(&buf))

	// Should have created a bead with the upgrade label
	issues, _ := env.Store.List(issue.Filter{Label: upgradeLabel})
	if len(issues) != 1 {
		t.Fatalf("expected 1 upgrade bead, got %d", len(issues))
	}
	iss := issues[0]
	if iss.Title != "Upgrade bw to 1.0.0" {
		t.Errorf("title = %q, want %q", iss.Title, "Upgrade bw to 1.0.0")
	}
	if iss.Priority != 1 {
		t.Errorf("priority = %d, want 1", iss.Priority)
	}
	if !strings.Contains(iss.Description, "bw upgrade") {
		t.Error("description should contain upgrade instructions")
	}
}

func TestEnsureUpgradeBead_UpdatesExisting(t *testing.T) {
	mockVersionCheck(t)
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	vcheckCurrentVersion = func() string { return "0.9.0" }

	var buf bytes.Buffer
	w := PlainWriter(&buf)

	// Create initial upgrade bead
	ensureUpgradeBead(env.Store, "1.0.0", w)

	// Now a newer version comes along
	ensureUpgradeBead(env.Store, "1.1.0", w)

	// Should still be just one bead, updated
	issues, _ := env.Store.List(issue.Filter{Label: upgradeLabel})
	if len(issues) != 1 {
		t.Fatalf("expected 1 upgrade bead, got %d", len(issues))
	}
	if issues[0].Title != "Upgrade bw to 1.1.0" {
		t.Errorf("title = %q, want updated to 1.1.0", issues[0].Title)
	}
}

func TestEnsureUpgradeBead_SameVersionNoOp(t *testing.T) {
	mockVersionCheck(t)
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	vcheckCurrentVersion = func() string { return "0.9.0" }

	var buf bytes.Buffer
	w := PlainWriter(&buf)

	ensureUpgradeBead(env.Store, "1.0.0", w)
	// Call again with same version — should be a no-op
	ensureUpgradeBead(env.Store, "1.0.0", w)

	issues, _ := env.Store.List(issue.Filter{Label: upgradeLabel})
	if len(issues) != 1 {
		t.Fatalf("expected 1 upgrade bead, got %d", len(issues))
	}
}

func TestFindUpgradeBead_IgnoresClosed(t *testing.T) {
	mockVersionCheck(t)
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	vcheckCurrentVersion = func() string { return "0.9.0" }

	var buf bytes.Buffer
	ensureUpgradeBead(env.Store, "1.0.0", PlainWriter(&buf))

	// Close the bead
	issues, _ := env.Store.List(issue.Filter{Label: upgradeLabel})
	env.Store.Close(issues[0].ID, "upgraded")
	env.Repo.Commit("close upgrade bead")

	// Should not find it
	found := findUpgradeBead(env.Store)
	if found != nil {
		t.Errorf("expected nil, got %s", found.ID)
	}
}

func TestMaybeCheckForUpgrade_EndToEnd(t *testing.T) {
	mockVersionCheck(t)
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	dir := t.TempDir()
	vcheckCacheDir = func() string { return dir }
	vcheckCurrentVersion = func() string { return "0.9.0" }
	vcheckNow = func() time.Time { return time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC) }
	vcheckFetchRelease = func() (*ghRelease, error) {
		return &ghRelease{TagName: "v1.0.0"}, nil
	}

	var buf bytes.Buffer
	maybeCheckForUpgrade(env.Store, PlainWriter(&buf))

	issues, _ := env.Store.List(issue.Filter{Label: upgradeLabel})
	if len(issues) != 1 {
		t.Fatalf("expected upgrade bead to be created, got %d", len(issues))
	}
	if issues[0].Title != "Upgrade bw to 1.0.0" {
		t.Errorf("title = %q", issues[0].Title)
	}
}

func TestMaybeCheckForUpgrade_NoOpWhenUpToDate(t *testing.T) {
	mockVersionCheck(t)
	env := testutil.NewEnv(t)
	defer env.Cleanup()

	dir := t.TempDir()
	vcheckCacheDir = func() string { return dir }
	vcheckCurrentVersion = func() string { return "1.0.0" }
	vcheckNow = func() time.Time { return time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC) }
	vcheckFetchRelease = func() (*ghRelease, error) {
		return &ghRelease{TagName: "v1.0.0"}, nil
	}

	var buf bytes.Buffer
	maybeCheckForUpgrade(env.Store, PlainWriter(&buf))

	issues, _ := env.Store.List(issue.Filter{Label: upgradeLabel})
	if len(issues) != 0 {
		t.Errorf("expected no upgrade bead, got %d", len(issues))
	}
}
