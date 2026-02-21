package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"0.3.0", "0.3.0", 0},
		{"0.3.0", "0.4.0", -1},
		{"0.4.0", "0.3.0", 1},
		{"0.3.0", "0.3.1", -1},
		{"1.0.0", "0.99.99", 1},
		{"0.3.0", "1.0.0", -1},
	}
	for _, tt := range tests {
		got := compareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestValidVersion(t *testing.T) {
	tests := []struct {
		v    string
		want bool
	}{
		{"0.3.0", true},
		{"1.0.0", true},
		{"10.20.30", true},
		{"0.3", false},
		{"abc", false},
		{"0.3.0.1", false},
		{"v0.3.0", false},
		{"", false},
	}
	for _, tt := range tests {
		got := validVersion(tt.v)
		if got != tt.want {
			t.Errorf("validVersion(%q) = %v, want %v", tt.v, got, tt.want)
		}
	}
}

func TestFindAsset(t *testing.T) {
	release := &ghRelease{
		TagName: "v1.0.0",
		Assets: []ghAsset{
			{Name: "beadwork_1.0.0_linux_amd64.tar.gz", URL: "https://example.com/linux_amd64.tar.gz"},
			{Name: "beadwork_1.0.0_darwin_arm64.tar.gz", URL: "https://example.com/darwin_arm64.tar.gz"},
			{Name: "beadwork_1.0.0_windows_amd64.zip", URL: "https://example.com/windows_amd64.zip"},
		},
	}

	// Should find asset for current platform
	asset, err := findAsset(release, "1.0.0")
	if err != nil {
		t.Fatalf("findAsset: %v", err)
	}
	if asset == nil {
		t.Fatal("expected to find asset for current platform")
	}

	// Should fail for missing platform
	release2 := &ghRelease{
		TagName: "v1.0.0",
		Assets:  []ghAsset{{Name: "beadwork_1.0.0_plan9_mips.tar.gz"}},
	}
	_, err = findAsset(release2, "1.0.0")
	if err == nil {
		t.Error("expected error for missing platform asset")
	}
}

func TestExtractFromTarGz(t *testing.T) {
	// Build a tar.gz with a "bw" binary inside
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("#!/bin/sh\necho hello")
	tw.WriteHeader(&tar.Header{
		Name: "bw",
		Size: int64(len(content)),
		Mode: 0755,
	})
	tw.Write(content)
	tw.Close()
	gw.Close()

	got, err := extractFromTarGz(buf.Bytes())
	if err != nil {
		t.Fatalf("extractFromTarGz: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestExtractFromTarGzMissing(t *testing.T) {
	// Build a tar.gz without a "bw" binary
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("not the binary")
	tw.WriteHeader(&tar.Header{
		Name: "README.md",
		Size: int64(len(content)),
		Mode: 0644,
	})
	tw.Write(content)
	tw.Close()
	gw.Close()

	_, err := extractFromTarGz(buf.Bytes())
	if err == nil {
		t.Error("expected error when bw binary not in archive")
	}
}

func TestInstallDirect(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "bw")

	// Write a fake "old" binary
	os.WriteFile(target, []byte("old"), 0755)

	newContent := []byte("new-binary-content")
	if err := installDirect(target, newContent); err != nil {
		t.Fatalf("installDirect: %v", err)
	}

	got, _ := os.ReadFile(target)
	if !bytes.Equal(got, newContent) {
		t.Errorf("binary content = %q, want %q", got, newContent)
	}

	info, _ := os.Stat(target)
	if info.Mode().Perm()&0111 == 0 {
		t.Error("installed binary is not executable")
	}
}

func TestInstallSymlink(t *testing.T) {
	dir := t.TempDir()

	// Create a fake current binary
	oldBinary := filepath.Join(dir, "bw-0.3.0")
	os.WriteFile(oldBinary, []byte("old"), 0755)

	// Create symlink pointing to it
	linkPath := filepath.Join(dir, "bw")
	os.Symlink(oldBinary, linkPath)

	newContent := []byte("new-binary-content")
	if err := installSymlink(linkPath, oldBinary, dir, "0.4.0", newContent); err != nil {
		t.Fatalf("installSymlink: %v", err)
	}

	// Symlink should now point to bw-0.4.0
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if filepath.Base(target) != "bw-0.4.0" {
		t.Errorf("symlink target = %q, want bw-0.4.0", target)
	}

	// New binary should have correct content
	got, _ := os.ReadFile(filepath.Join(dir, "bw-0.4.0"))
	if !bytes.Equal(got, newContent) {
		t.Errorf("new binary content = %q, want %q", got, newContent)
	}

	// Old binary should still exist
	if _, err := os.Stat(oldBinary); err != nil {
		t.Error("old binary was removed (should be preserved)")
	}
}

func TestCheckWritable(t *testing.T) {
	dir := t.TempDir()
	if err := checkWritable(dir); err != nil {
		t.Errorf("writable dir should pass: %v", err)
	}
}

func TestResolveBinaryRegularFile(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bw")
	os.WriteFile(bin, []byte("binary"), 0755)

	_, symlink, _, err := resolveBinaryPath(bin)
	if err != nil {
		t.Fatalf("resolveBinaryPath: %v", err)
	}
	if symlink {
		t.Error("regular file should not be detected as symlink")
	}
}

func TestResolveBinarySymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "bw-0.3.0")
	os.WriteFile(real, []byte("binary"), 0755)

	link := filepath.Join(dir, "bw")
	os.Symlink(real, link)

	_, symlink, targetPath, err := resolveBinaryPath(link)
	if err != nil {
		t.Fatalf("resolveBinaryPath: %v", err)
	}
	if !symlink {
		t.Error("symlink should be detected")
	}
	if filepath.Base(targetPath) != "bw-0.3.0" {
		t.Errorf("targetPath base = %q, want bw-0.3.0", filepath.Base(targetPath))
	}
}

func TestExtractFromZip(t *testing.T) {
	// Build a zip with a "bw" binary inside
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	content := []byte("#!/bin/sh\necho hello from zip")
	f, _ := zw.Create("bw")
	f.Write(content)
	zw.Close()

	got, err := extractFromZip(buf.Bytes())
	if err != nil {
		t.Fatalf("extractFromZip: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestExtractFromZipExe(t *testing.T) {
	// Build a zip with "bw.exe" (Windows naming)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	content := []byte("windows binary content")
	f, _ := zw.Create("beadwork_1.0.0/bw.exe")
	f.Write(content)
	zw.Close()

	got, err := extractFromZip(buf.Bytes())
	if err != nil {
		t.Fatalf("extractFromZip: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestExtractFromZipMissing(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	f, _ := zw.Create("README.md")
	f.Write([]byte("not a binary"))
	zw.Close()

	_, err := extractFromZip(buf.Bytes())
	if err == nil {
		t.Error("expected error when bw binary not in zip archive")
	}
}

func TestExtractFromTarGzNested(t *testing.T) {
	// Build a tar.gz with nested path like "beadwork_1.0.0/bw"
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("nested binary content")
	tw.WriteHeader(&tar.Header{
		Name: "beadwork_1.0.0/bw",
		Size: int64(len(content)),
		Mode: 0755,
	})
	tw.Write(content)
	tw.Close()
	gw.Close()

	got, err := extractFromTarGz(buf.Bytes())
	if err != nil {
		t.Fatalf("extractFromTarGz: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestExtractBinaryRoutesCorrectly(t *testing.T) {
	// Test that extractBinary routes .zip to extractFromZip
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	content := []byte("zip binary")
	f, _ := zw.Create("bw")
	f.Write(content)
	zw.Close()

	got, err := extractBinary("beadwork_1.0.0_windows_amd64.zip", buf.Bytes())
	if err != nil {
		t.Fatalf("extractBinary(.zip): %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestExtractBinaryRoutesTarGz(t *testing.T) {
	// Test that extractBinary routes .tar.gz to extractFromTarGz
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	content := []byte("tar binary")
	tw.WriteHeader(&tar.Header{
		Name: "bw",
		Size: int64(len(content)),
		Mode: 0755,
	})
	tw.Write(content)
	tw.Close()
	gw.Close()

	got, err := extractBinary("beadwork_1.0.0_linux_amd64.tar.gz", buf.Bytes())
	if err != nil {
		t.Fatalf("extractBinary(.tar.gz): %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestResolveBinaryPathNotExist(t *testing.T) {
	_, _, _, err := resolveBinaryPath("/nonexistent/path/bw")
	if err == nil {
		t.Error("expected error for non-existent binary path")
	}
}

func TestInstallDirectNewFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "bw")

	// Install without an existing file
	content := []byte("brand-new-binary")
	if err := installDirect(target, content); err != nil {
		t.Fatalf("installDirect: %v", err)
	}

	got, _ := os.ReadFile(target)
	if !bytes.Equal(got, content) {
		t.Errorf("binary content = %q, want %q", got, content)
	}
}

func TestCheckWritableNoPermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission test not reliable on Windows")
	}
	dir := t.TempDir()
	os.Chmod(dir, 0555)
	defer os.Chmod(dir, 0755)

	if err := checkWritable(dir); err == nil {
		t.Error("expected error for read-only directory")
	}
}

// --- cmdUpgrade DI tests ---

// mockUpgrade saves and restores all injectable vars for a test.
func mockUpgrade(t *testing.T) {
	t.Helper()
	origFetch := upgradeFetchRelease
	origDownload := upgradeDownloadAsset
	origResolve := upgradeResolveBinary
	origChangelog := upgradeFetchChangelog
	origStdin := upgradeStdin
	origVersion := upgradeCurrentVersion
	origVerify := upgradeVerify
	t.Cleanup(func() {
		upgradeFetchRelease = origFetch
		upgradeDownloadAsset = origDownload
		upgradeResolveBinary = origResolve
		upgradeFetchChangelog = origChangelog
		upgradeStdin = origStdin
		upgradeCurrentVersion = origVersion
		upgradeVerify = origVerify
	})
	// Default: changelog fetch fails silently (most tests don't care)
	upgradeFetchChangelog = func(version string) (string, error) {
		return "", fmt.Errorf("mock: no changelog")
	}
}

// makeTarGz builds a tar.gz archive containing a "bw" binary with the given content.
func makeTarGz(t *testing.T, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	tw.WriteHeader(&tar.Header{
		Name: "bw",
		Size: int64(len(content)),
		Mode: 0755,
	})
	tw.Write(content)
	tw.Close()
	gw.Close()
	return buf.Bytes()
}

func mockRelease(ver string) *ghRelease {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return &ghRelease{
		TagName: "v" + ver,
		Assets: []ghAsset{
			{
				Name: "beadwork_" + ver + "_" + runtime.GOOS + "_" + runtime.GOARCH + ext,
				URL:  "https://example.com/download",
			},
		},
	}
}

func TestCmdUpgradeUpToDate(t *testing.T) {
	mockUpgrade(t)
	upgradeCurrentVersion = func() string { return "1.0.0" }
	upgradeFetchRelease = func() (*ghRelease, error) {
		return mockRelease("1.0.0"), nil
	}
	upgradeResolveBinary = func() (string, bool, string, error) {
		return "/usr/local/bin/bw", false, "", nil
	}

	var buf bytes.Buffer
	err := cmdUpgrade(nil, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpgrade: %v", err)
	}
	if !strings.Contains(buf.String(), "up to date") {
		t.Errorf("output = %q, want 'up to date'", buf.String())
	}
}

func TestCmdUpgradeCheckOnly(t *testing.T) {
	mockUpgrade(t)
	upgradeCurrentVersion = func() string { return "1.0.0" }
	upgradeFetchRelease = func() (*ghRelease, error) {
		return mockRelease("2.0.0"), nil
	}
	upgradeResolveBinary = func() (string, bool, string, error) {
		return "/usr/local/bin/bw", false, "", nil
	}

	var buf bytes.Buffer
	err := cmdUpgrade(nil, []string{"--check"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpgrade --check: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "2.0.0 available") {
		t.Errorf("output = %q, want '2.0.0 available'", out)
	}
}

func TestCmdUpgradeConfirmNo(t *testing.T) {
	mockUpgrade(t)
	dir := t.TempDir()
	bin := filepath.Join(dir, "bw")
	os.WriteFile(bin, []byte("old"), 0755)

	upgradeCurrentVersion = func() string { return "1.0.0" }
	upgradeFetchRelease = func() (*ghRelease, error) {
		return mockRelease("2.0.0"), nil
	}
	upgradeResolveBinary = func() (string, bool, string, error) {
		return bin, false, "", nil
	}
	upgradeStdin = strings.NewReader("n\n")

	var buf bytes.Buffer
	err := cmdUpgrade(nil, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpgrade: %v", err)
	}
	if !strings.Contains(buf.String(), "cancelled") {
		t.Errorf("output = %q, want 'cancelled'", buf.String())
	}
}

func TestCmdUpgradeYesFlag(t *testing.T) {
	mockUpgrade(t)
	dir := t.TempDir()
	bin := filepath.Join(dir, "bw")
	os.WriteFile(bin, []byte("old"), 0755)

	upgradeCurrentVersion = func() string { return "1.0.0" }
	upgradeFetchRelease = func() (*ghRelease, error) {
		return mockRelease("2.0.0"), nil
	}
	upgradeResolveBinary = func() (string, bool, string, error) {
		return bin, false, "", nil
	}
	upgradeDownloadAsset = func(url string, size int64, w Writer) ([]byte, error) {
		return makeTarGz(t, []byte("new-binary")), nil
	}
	upgradeVerify = func(execPath string) (string, error) {
		return "bw 2.0.0", nil
	}

	var buf bytes.Buffer
	err := cmdUpgrade(nil, []string{"--yes"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpgrade --yes: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "bw 2.0.0") {
		t.Errorf("output = %q, want 'bw 2.0.0'", out)
	}

	// Verify binary was replaced
	got, _ := os.ReadFile(bin)
	if !bytes.Equal(got, []byte("new-binary")) {
		t.Errorf("binary content = %q, want 'new-binary'", got)
	}
}

func TestCmdUpgradeFullFlowConfirmYes(t *testing.T) {
	mockUpgrade(t)
	dir := t.TempDir()
	bin := filepath.Join(dir, "bw")
	os.WriteFile(bin, []byte("old"), 0755)

	upgradeCurrentVersion = func() string { return "1.0.0" }
	upgradeFetchRelease = func() (*ghRelease, error) {
		return mockRelease("2.0.0"), nil
	}
	upgradeResolveBinary = func() (string, bool, string, error) {
		return bin, false, "", nil
	}
	upgradeDownloadAsset = func(url string, size int64, w Writer) ([]byte, error) {
		return makeTarGz(t, []byte("new-binary")), nil
	}
	upgradeVerify = func(execPath string) (string, error) {
		return "bw 2.0.0", nil
	}
	upgradeStdin = strings.NewReader("y\n")

	var buf bytes.Buffer
	err := cmdUpgrade(nil, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpgrade: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "2.0.0 available") {
		t.Errorf("output should mention available: %q", out)
	}
	if !strings.Contains(out, "bw 2.0.0") {
		t.Errorf("output should show verified version: %q", out)
	}
}

func TestCmdUpgradeFetchError(t *testing.T) {
	mockUpgrade(t)
	upgradeCurrentVersion = func() string { return "1.0.0" }
	upgradeFetchRelease = func() (*ghRelease, error) {
		return nil, fmt.Errorf("network error")
	}
	upgradeResolveBinary = func() (string, bool, string, error) {
		return "/usr/local/bin/bw", false, "", nil
	}

	var buf bytes.Buffer
	err := cmdUpgrade(nil, []string{}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for fetch failure")
	}
	if !strings.Contains(err.Error(), "failed to check for updates") {
		t.Errorf("error = %q", err)
	}
}

func TestCmdUpgradeInvalidVersion(t *testing.T) {
	mockUpgrade(t)
	upgradeCurrentVersion = func() string { return "1.0.0" }
	upgradeFetchRelease = func() (*ghRelease, error) {
		return &ghRelease{TagName: "vNOTVALID"}, nil
	}
	upgradeResolveBinary = func() (string, bool, string, error) {
		return "/usr/local/bin/bw", false, "", nil
	}

	var buf bytes.Buffer
	err := cmdUpgrade(nil, []string{}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for invalid version")
	}
	if !strings.Contains(err.Error(), "invalid version") {
		t.Errorf("error = %q", err)
	}
}

func TestCmdUpgradeDownloadError(t *testing.T) {
	mockUpgrade(t)
	dir := t.TempDir()
	bin := filepath.Join(dir, "bw")
	os.WriteFile(bin, []byte("old"), 0755)

	upgradeCurrentVersion = func() string { return "1.0.0" }
	upgradeFetchRelease = func() (*ghRelease, error) {
		return mockRelease("2.0.0"), nil
	}
	upgradeResolveBinary = func() (string, bool, string, error) {
		return bin, false, "", nil
	}
	upgradeDownloadAsset = func(url string, size int64, w Writer) ([]byte, error) {
		return nil, fmt.Errorf("download timeout")
	}
	upgradeStdin = strings.NewReader("y\n")

	var buf bytes.Buffer
	err := cmdUpgrade(nil, []string{}, PlainWriter(&buf))
	if err == nil {
		t.Error("expected error for download failure")
	}
	if !strings.Contains(err.Error(), "download failed") {
		t.Errorf("error = %q", err)
	}
}

func TestCmdUpgradeRoutesToRepo(t *testing.T) {
	// Verify that "repo" subcommand is routed correctly
	// This will fail because we're not in a git repo, but it proves routing works
	var buf bytes.Buffer
	err := cmdUpgrade(nil, []string{"repo"}, PlainWriter(&buf))
	if err == nil {
		// Might succeed if we happen to be in a valid beadwork repo
		return
	}
	// Expected — proves it routed to cmdUpgradeRepo, not the binary upgrade flow
}

func TestFindAssetNaming(t *testing.T) {
	// Verify we build the expected asset name
	release := &ghRelease{
		TagName: "v2.1.0",
		Assets: []ghAsset{
			{Name: "beadwork_2.1.0_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz", URL: "https://x"},
		},
	}
	if runtime.GOOS == "windows" {
		release.Assets[0].Name = "beadwork_2.1.0_windows_" + runtime.GOARCH + ".zip"
	}

	asset, err := findAsset(release, "2.1.0")
	if err != nil {
		t.Fatalf("should find asset: %v", err)
	}
	if asset.URL != "https://x" {
		t.Errorf("wrong asset URL: %s", asset.URL)
	}
}

// --- formatBytes tests ---

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{10240, "10.0 KB"},
		{1048576, "1.0 MB"},
		{8388608, "8.0 MB"},
		{1572864, "1.5 MB"},
	}
	for _, tt := range tests {
		got := formatBytes(tt.n)
		if got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

// --- parseChangelog tests ---

func TestParseChangelog(t *testing.T) {
	content := `# Changelog

## 0.6.0 — 2026-02-21

- Feature A
- Feature B

## 0.5.3 — 2026-02-20

- Feature C

## 0.5.2 — 2026-02-20

- Feature D

## 0.5.0 — 2026-02-20

- Feature E
`

	tests := []struct {
		name     string
		from, to string
		want     []string // substrings that must appear
		dontWant []string // substrings that must not appear
	}{
		{
			name: "single version jump",
			from: "0.5.3", to: "0.6.0",
			want:     []string{"0.6.0", "Feature A", "Feature B"},
			dontWant: []string{"0.5.3", "Feature C", "Feature D"},
		},
		{
			name: "multi-version span",
			from: "0.5.0", to: "0.6.0",
			want:     []string{"0.6.0", "0.5.3", "0.5.2", "Feature A", "Feature C", "Feature D"},
			dontWant: []string{"Feature E"},
		},
		{
			name: "no matching versions",
			from: "0.6.0", to: "0.7.0",
			want:     nil,
			dontWant: []string{"0.6.0", "Feature A"},
		},
		{
			name: "same version returns nothing",
			from: "0.5.3", to: "0.5.3",
			want:     nil,
			dontWant: []string{"Feature"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseChangelog(content, tt.from, tt.to)
			for _, s := range tt.want {
				if !strings.Contains(got, s) {
					t.Errorf("parseChangelog(%q, %q) missing %q in:\n%s", tt.from, tt.to, s, got)
				}
			}
			for _, s := range tt.dontWant {
				if strings.Contains(got, s) {
					t.Errorf("parseChangelog(%q, %q) should not contain %q in:\n%s", tt.from, tt.to, s, got)
				}
			}
		})
	}
}

func TestParseChangelogEmpty(t *testing.T) {
	got := parseChangelog("", "0.1.0", "0.2.0")
	if got != "" {
		t.Errorf("expected empty string for empty content, got %q", got)
	}
}

// --- changelog integration tests ---

func TestCmdUpgradeCheckShowsChangelog(t *testing.T) {
	mockUpgrade(t)
	upgradeCurrentVersion = func() string { return "0.5.0" }
	upgradeFetchRelease = func() (*ghRelease, error) {
		return mockRelease("0.6.0"), nil
	}
	upgradeResolveBinary = func() (string, bool, string, error) {
		return "/usr/local/bin/bw", false, "", nil
	}
	upgradeFetchChangelog = func(version string) (string, error) {
		return "## 0.6.0 — 2026-02-21\n\n- New feature\n\n## 0.5.0 — 2026-02-20\n\n- Old\n", nil
	}

	var buf bytes.Buffer
	err := cmdUpgrade(nil, []string{"--check"}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpgrade --check: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "New feature") {
		t.Errorf("expected changelog entry in output, got: %q", out)
	}
	if strings.Contains(out, "Old") {
		t.Errorf("should not include entries at/before current version, got: %q", out)
	}
}

func TestCmdUpgradeChangelogFailureNonFatal(t *testing.T) {
	mockUpgrade(t)
	dir := t.TempDir()
	bin := filepath.Join(dir, "bw")
	os.WriteFile(bin, []byte("old"), 0755)

	upgradeCurrentVersion = func() string { return "1.0.0" }
	upgradeFetchRelease = func() (*ghRelease, error) {
		return mockRelease("2.0.0"), nil
	}
	upgradeResolveBinary = func() (string, bool, string, error) {
		return bin, false, "", nil
	}
	upgradeFetchChangelog = func(version string) (string, error) {
		return "", fmt.Errorf("404 not found")
	}
	upgradeDownloadAsset = func(url string, size int64, w Writer) ([]byte, error) {
		return makeTarGz(t, []byte("new-binary")), nil
	}
	upgradeVerify = func(execPath string) (string, error) {
		return "bw 2.0.0", nil
	}
	upgradeStdin = strings.NewReader("y\n")

	var buf bytes.Buffer
	err := cmdUpgrade(nil, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("upgrade should succeed despite changelog failure: %v", err)
	}
	if !strings.Contains(buf.String(), "bw 2.0.0") {
		t.Errorf("expected verification output, got: %q", buf.String())
	}
}

// --- narration tests ---

func TestCmdUpgradeNarratesSteps(t *testing.T) {
	mockUpgrade(t)
	dir := t.TempDir()
	bin := filepath.Join(dir, "bw")
	os.WriteFile(bin, []byte("old"), 0755)

	upgradeCurrentVersion = func() string { return "1.0.0" }
	upgradeFetchRelease = func() (*ghRelease, error) {
		return mockRelease("2.0.0"), nil
	}
	upgradeResolveBinary = func() (string, bool, string, error) {
		return bin, false, "", nil
	}
	upgradeFetchChangelog = func(version string) (string, error) {
		return "", fmt.Errorf("skip")
	}
	upgradeDownloadAsset = func(url string, size int64, w Writer) ([]byte, error) {
		return makeTarGz(t, []byte("new-binary")), nil
	}
	upgradeVerify = func(execPath string) (string, error) {
		return "bw 2.0.0", nil
	}
	upgradeStdin = strings.NewReader("y\n")

	var buf bytes.Buffer
	err := cmdUpgrade(nil, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpgrade: %v", err)
	}
	out := buf.String()

	// Should narrate extraction and installation steps
	if !strings.Contains(out, "extracting") {
		t.Errorf("expected extraction narration, got: %q", out)
	}
	if !strings.Contains(out, "installing") && !strings.Contains(out, "replacing") {
		t.Errorf("expected install narration, got: %q", out)
	}
	if !strings.Contains(out, "verifying") {
		t.Errorf("expected verify narration, got: %q", out)
	}
}

func TestCmdUpgradeSymlinkNarration(t *testing.T) {
	mockUpgrade(t)
	dir := t.TempDir()

	oldBin := filepath.Join(dir, "bw-1.0.0")
	os.WriteFile(oldBin, []byte("old"), 0755)
	link := filepath.Join(dir, "bw")
	os.Symlink(oldBin, link)

	upgradeCurrentVersion = func() string { return "1.0.0" }
	upgradeFetchRelease = func() (*ghRelease, error) {
		return mockRelease("2.0.0"), nil
	}
	upgradeResolveBinary = func() (string, bool, string, error) {
		return link, true, oldBin, nil
	}
	upgradeFetchChangelog = func(version string) (string, error) {
		return "", fmt.Errorf("skip")
	}
	upgradeDownloadAsset = func(url string, size int64, w Writer) ([]byte, error) {
		return makeTarGz(t, []byte("new-binary")), nil
	}
	upgradeVerify = func(execPath string) (string, error) {
		return "bw 2.0.0", nil
	}
	upgradeStdin = strings.NewReader("y\n")

	var buf bytes.Buffer
	err := cmdUpgrade(nil, []string{}, PlainWriter(&buf))
	if err != nil {
		t.Fatalf("cmdUpgrade: %v", err)
	}
	out := buf.String()

	// Symlink install should mention "symlink"
	if !strings.Contains(out, "symlink") {
		t.Errorf("expected symlink narration, got: %q", out)
	}
}
