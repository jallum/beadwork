package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/jallum/beadwork/internal/repo"
)

const releaseURL = "https://api.github.com/repos/jallum/beadwork/releases/latest"

// Injectable dependencies for testing. Production code uses the defaults.
var (
	upgradeFetchRelease   = fetchLatestRelease
	upgradeDownloadAsset  = downloadAsset
	upgradeResolveBinary  = resolveBinary
	upgradeFetchChangelog = fetchChangelog
	upgradeStdin          io.Reader = os.Stdin
	upgradeCurrentVersion           = func() string { return version }
	upgradeVerify                   = func(execPath string) (string, error) {
		out, err := exec.Command(execPath, "--version").Output()
		return strings.TrimSpace(string(out)), err
	}
)

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

type UpgradeArgs struct {
	Check bool
	Yes   bool
}

func parseUpgradeArgs(raw []string) (UpgradeArgs, error) {
	a, err := ParseArgs(raw, nil, []string{"--check", "--yes"})
	if err != nil {
		return UpgradeArgs{}, err
	}
	return UpgradeArgs{
		Check: a.Bool("--check"),
		Yes:   a.Bool("--yes"),
	}, nil
}

func cmdUpgrade(args []string, w Writer) error {
	if len(args) > 0 && args[0] == "repo" {
		return cmdUpgradeRepo(args[1:], w)
	}

	ua, err := parseUpgradeArgs(args)
	if err != nil {
		return err
	}
	check := ua.Check
	yes := ua.Yes

	// Resolve our binary location
	execPath, symlink, targetPath, err := upgradeResolveBinary()
	if err != nil {
		return err
	}

	// Fetch latest release info
	release, err := upgradeFetchRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	if !validVersion(latest) {
		return fmt.Errorf("invalid version from release: %s", release.TagName)
	}

	cur := upgradeCurrentVersion()
	if compareVersions(cur, latest) >= 0 {
		fmt.Fprintf(w, "bw %s (up to date)\n", w.Style(cur, Dim))
		return nil
	}

	fmt.Fprintf(w, "bw %s %s %s available\n",
		w.Style(cur, Dim), w.Style("→", Dim), w.Style(latest, Bold))

	// Fetch and display changelog (non-fatal on failure)
	if changelogContent, cerr := upgradeFetchChangelog(latest); cerr == nil {
		if parsed := parseChangelog(changelogContent, cur, latest); parsed != "" {
			fmt.Fprintln(w)
			for _, line := range strings.Split(parsed, "\n") {
				if strings.HasPrefix(line, "## ") {
					fmt.Fprintf(w, "  %s\n", w.Style(strings.TrimPrefix(line, "## "), Bold))
				} else {
					fmt.Fprintf(w, "  %s\n", line)
				}
			}
		}
	}

	if check {
		return nil
	}

	// Find matching asset
	asset, err := findAsset(release, latest)
	if err != nil {
		return err
	}

	// Determine where we'll write
	var installDir string
	if symlink {
		installDir = filepath.Dir(targetPath)
	} else {
		installDir = filepath.Dir(execPath)
	}

	// Precheck: write permission
	if err := checkWritable(installDir); err != nil {
		return fmt.Errorf("no write permission to %s: %v", installDir, err)
	}

	// Prompt unless --yes
	if !yes {
		fmt.Fprintf(w, "\ndownload and install? [y/N] ")
		reader := bufio.NewReader(upgradeStdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(w, "cancelled")
			return nil
		}
	}

	// Download
	fmt.Fprintf(w, "downloading %s...\n", w.Style(asset.Name, Cyan))
	archiveData, err := upgradeDownloadAsset(asset.URL, asset.Size, w)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Extract binary from archive
	fmt.Fprintln(w, "extracting binary from archive...")
	binaryData, err := extractBinary(asset.Name, archiveData)
	if err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}

	// Install
	if symlink {
		fmt.Fprintf(w, "installing %s → %s (symlink)\n",
			w.Style("bw-"+latest, Bold), w.Style(execPath, Cyan))
		err = installSymlink(execPath, targetPath, installDir, latest, binaryData)
	} else {
		fmt.Fprintf(w, "replacing %s...\n", w.Style(execPath, Cyan))
		err = installDirect(execPath, binaryData)
	}
	if err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	// Verify
	fmt.Fprintf(w, "verifying... ")
	verOut, verr := upgradeVerify(execPath)
	if verr != nil {
		return fmt.Errorf("installed binary failed verification: %w", verr)
	}
	fmt.Fprintln(w, w.Style(verOut, Green))
	return nil
}

// resolveBinary returns the executable path, whether it's a symlink, and
// the resolved target path.
func resolveBinary() (execPath string, symlink bool, targetPath string, err error) {
	execPath, err = os.Executable()
	if err != nil {
		return "", false, "", fmt.Errorf("cannot determine binary path: %w", err)
	}
	execPath, symlink, targetPath, err = resolveBinaryPath(execPath)
	return
}

// resolveBinaryPath is the testable core of resolveBinary.
func resolveBinaryPath(execPath string) (string, bool, string, error) {
	targetPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", false, "", fmt.Errorf("cannot resolve binary path: %w", err)
	}
	// Check if execPath itself is a symlink (not just path canonicalization)
	fi, err := os.Lstat(execPath)
	if err != nil {
		return "", false, "", fmt.Errorf("cannot stat binary: %w", err)
	}
	isSymlink := fi.Mode()&os.ModeSymlink != 0
	return execPath, isSymlink, targetPath, nil
}

func fetchLatestRelease() (*ghRelease, error) {
	resp, err := http.Get(releaseURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	return &release, nil
}

func findAsset(release *ghRelease, ver string) (*ghAsset, error) {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	want := fmt.Sprintf("beadwork_%s_%s_%s%s", ver, runtime.GOOS, runtime.GOARCH, ext)
	for _, a := range release.Assets {
		if a.Name == want {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("no release asset for %s/%s (looking for %s)", runtime.GOOS, runtime.GOARCH, want)
}

func downloadAsset(url string, size int64, w Writer) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	total := size
	if resp.ContentLength > 0 {
		total = resp.ContentLength
	}

	var buf bytes.Buffer
	if total > 0 {
		buf.Grow(int(total))
	}

	// Stream with progress reporting
	isTTY := w.Width() > 0
	written := int64(0)
	chunk := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(chunk)
		if n > 0 {
			buf.Write(chunk[:n])
			written += int64(n)
			if isTTY && total > 0 {
				fmt.Fprintf(w, "\r  %s",
					w.Style(fmt.Sprintf("%s/%s (%d%%)",
						formatBytes(written), formatBytes(total),
						written*100/total), Dim))
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}

	if isTTY && total > 0 {
		fmt.Fprintf(w, "\r  %s %s\n", formatBytes(written), w.Style("done", Green))
	} else {
		fmt.Fprintf(w, "  %s %s\n", formatBytes(written), w.Style("done", Green))
	}

	return buf.Bytes(), nil
}

func extractBinary(assetName string, data []byte) ([]byte, error) {
	if strings.HasSuffix(assetName, ".zip") {
		return extractFromZip(data)
	}
	return extractFromTarGz(data)
}

func extractFromTarGz(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) == "bw" && hdr.Typeflag == tar.TypeReg {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("bw binary not found in archive")
}

func extractFromZip(data []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name == "bw" || name == "bw.exe" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("bw binary not found in archive")
}

func checkWritable(dir string) error {
	tmp, err := os.CreateTemp(dir, ".bw-upgrade-check-*")
	if err != nil {
		return err
	}
	tmp.Close()
	os.Remove(tmp.Name())
	return nil
}

func installDirect(execPath string, binaryData []byte) error {
	dir := filepath.Dir(execPath)
	tmp, err := os.CreateTemp(dir, ".bw-upgrade-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(binaryData); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Chmod(0755); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

func installSymlink(linkPath, currentTarget, targetDir, ver string, binaryData []byte) error {
	newBinary := filepath.Join(targetDir, "bw-"+ver)

	// Write new versioned binary atomically
	tmp, err := os.CreateTemp(targetDir, ".bw-upgrade-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(binaryData); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Chmod(0755); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, newBinary); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Atomically update symlink: create temp symlink, then rename over original
	tmpLink := linkPath + ".tmp"
	os.Remove(tmpLink)
	if err := os.Symlink(newBinary, tmpLink); err != nil {
		return fmt.Errorf("create symlink: %w", err)
	}
	if err := os.Rename(tmpLink, linkPath); err != nil {
		os.Remove(tmpLink)
		return fmt.Errorf("update symlink: %w", err)
	}
	return nil
}

func validVersion(v string) bool {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if _, err := strconv.Atoi(p); err != nil {
			return false
		}
	}
	return true
}

func cmdUpgradeRepo(args []string, w Writer) error {
	r, err := getRepo()
	if err != nil {
		return err
	}
	if !r.IsInitialized() {
		return fmt.Errorf("beadwork not initialized. Run: bw init")
	}

	from := r.Version()
	if from >= repo.CurrentVersion {
		fmt.Fprintf(w, "repo at version %d (up to date)\n", from)
		return nil
	}

	fmt.Fprintf(w, "upgrading repo v%d -> v%d...\n", from, repo.CurrentVersion)
	for v := from; v < repo.CurrentVersion; v++ {
		fmt.Fprintf(w, "  v%d -> v%d: %s\n", v, v+1, repo.Migrations[v].Description)
	}

	_, to, err := r.Upgrade()
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "repo upgraded to v%d\n", to)
	return nil
}

const changelogURL = "https://raw.githubusercontent.com/jallum/beadwork/v%s/CHANGELOG.md"

func fetchChangelog(version string) (string, error) {
	resp, err := http.Get(fmt.Sprintf(changelogURL, version))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// parseChangelog extracts changelog entries for versions in the range (from, to].
func parseChangelog(content, from, to string) string {
	var result strings.Builder
	lines := strings.Split(content, "\n")
	include := false

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// Extract version: "## 0.6.0 — 2026-02-21" -> "0.6.0"
			rest := strings.TrimPrefix(line, "## ")
			ver := strings.Fields(rest)[0]
			if !validVersion(ver) {
				include = false
				continue
			}
			// Include if from < ver <= to
			include = compareVersions(ver, from) > 0 && compareVersions(ver, to) <= 0
		}
		if include {
			result.WriteString(line)
			result.WriteByte('\n')
		}
	}
	return strings.TrimRight(result.String(), "\n")
}

func formatBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func compareVersions(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		ai, _ := strconv.Atoi(ap[i])
		bi, _ := strconv.Atoi(bp[i])
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}
	return 0
}
