package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jallum/beadwork/internal/issue"
)

const (
	upgradeLabel  = "beadwork-upgrade"
	checkInterval = 24 * time.Hour
)

// Injectable dependencies for testing.
var (
	vcheckNow          = func() time.Time { return time.Now() }
	vcheckFetchRelease = fetchLatestRelease
	vcheckFetchChangelog = fetchChangelog
	vcheckCacheDir     = func() string {
		if d, err := os.UserCacheDir(); err == nil {
			return filepath.Join(d, "bw")
		}
		return ""
	}
	vcheckCurrentVersion = func() string { return version }
)

type versionCache struct {
	LastCheck     string `json:"last_check"`
	LatestVersion string `json:"latest_version,omitempty"`
}

func readCache() *versionCache {
	dir := vcheckCacheDir()
	if dir == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(dir, "version-check.json"))
	if err != nil {
		return nil
	}
	var c versionCache
	if json.Unmarshal(data, &c) != nil {
		return nil
	}
	return &c
}

func writeCache(c *versionCache) {
	dir := vcheckCacheDir()
	if dir == "" {
		return
	}
	os.MkdirAll(dir, 0755)
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	os.WriteFile(filepath.Join(dir, "version-check.json"), data, 0644)
}

// checkForNewerVersion returns the latest version string if newer than
// the running binary, or "" if up-to-date or on any error. It writes
// the cache timestamp unconditionally to throttle checks to once per day.
func checkForNewerVersion() string {
	now := vcheckNow()

	// Read cache; skip if checked recently.
	if c := readCache(); c != nil {
		if t, err := time.Parse(time.RFC3339, c.LastCheck); err == nil {
			if now.Sub(t) < checkInterval {
				// Return cached result.
				cur := vcheckCurrentVersion()
				if c.LatestVersion != "" && compareVersions(cur, c.LatestVersion) < 0 {
					return c.LatestVersion
				}
				return ""
			}
		}
	}

	// Write timestamp now — success or failure, we tried today.
	cache := &versionCache{LastCheck: now.Format(time.RFC3339)}
	defer writeCache(cache)

	release, err := vcheckFetchRelease()

	if err != nil || release == nil {
		return ""
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	if !validVersion(latest) {
		return ""
	}

	cache.LatestVersion = latest

	cur := vcheckCurrentVersion()
	if compareVersions(cur, latest) < 0 {
		return latest
	}
	return ""
}

// maybeCheckForUpgrade runs a throttled version check and, if a newer
// release exists, creates or updates an upgrade bead in the store.
// All errors are silently swallowed — this must never break normal commands.
func maybeCheckForUpgrade(store *issue.Store, w Writer) {
	latest := checkForNewerVersion()
	if latest == "" {
		return
	}
	ensureUpgradeBead(store, latest, w)
}

// ensureUpgradeBead creates or updates a bead labeled "beadwork-upgrade".
func ensureUpgradeBead(store *issue.Store, latest string, w Writer) {
	cur := vcheckCurrentVersion()
	title := fmt.Sprintf("Upgrade bw to %s", latest)
	desc := buildUpgradeDescription(cur, latest)

	// Look for an existing open upgrade bead.
	existing := findUpgradeBead(store)

	if existing != nil {
		// Already tracking this version — nothing to do.
		if existing.Title == title {
			return
		}
		// Update to reflect newer version.
		t := title
		opts := issue.UpdateOpts{Title: &t, Description: &desc}
		if _, err := store.Update(existing.ID, opts); err != nil {
			return
		}
		intent := fmt.Sprintf("update %s title=%q description=...", existing.ID, title)
		store.Commit(intent)
		return
	}

	// Create new upgrade bead.
	p := 1
	iss, err := store.Create(title, issue.CreateOpts{
		Priority:    &p,
		Type:        "task",
		Description: desc,
	})
	if err != nil {
		return
	}
	if _, err := store.Label(iss.ID, []string{upgradeLabel}, nil); err != nil {
		return
	}
	intent := fmt.Sprintf("create %s p1 task %q", iss.ID, title)
	store.Commit(intent)
}

func findUpgradeBead(store *issue.Store) *issue.Issue {
	issues, err := store.List(issue.Filter{
		Label:    upgradeLabel,
		Statuses: []string{"open", "in_progress"},
	})
	if err != nil || len(issues) == 0 {
		return nil
	}
	return issues[0]
}

func buildUpgradeDescription(cur, latest string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "bw %s → %s is available.\n", cur, latest)

	// Try to include changelog; non-fatal on failure.
	if content, err := vcheckFetchChangelog(latest); err == nil {
		if parsed := parseChangelog(content, cur, latest); parsed != "" {
			b.WriteString("\n## What's new\n\n")
			b.WriteString(parsed)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n## To upgrade\n\nRun `bw upgrade --yes` or `bw upgrade` for interactive mode.\n")
	return b.String()
}
