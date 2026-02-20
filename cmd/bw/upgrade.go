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
)

const releaseURL = "https://api.github.com/repos/jallum/beadwork/releases/latest"

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

func cmdUpgrade(args []string) {
	check := hasFlag(args, "--check")
	yes := hasFlag(args, "--yes")

	// Resolve our binary location
	execPath, symlink, targetPath := resolveBinary()

	// Fetch latest release info
	release, err := fetchLatestRelease()
	if err != nil {
		fatal("failed to check for updates: " + err.Error())
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	if !validVersion(latest) {
		fatal("invalid version from release: " + release.TagName)
	}

	if compareVersions(version, latest) >= 0 {
		fmt.Printf("bw %s (up to date)\n", version)
		return
	}

	fmt.Printf("bw %s → %s available\n", version, latest)

	if check {
		return
	}

	// Find matching asset
	asset, err := findAsset(release, latest)
	if err != nil {
		fatal(err.Error())
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
		fatal(fmt.Sprintf("no write permission to %s: %v", installDir, err))
	}

	// Prompt unless --yes
	if !yes {
		fmt.Printf("download and install? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("cancelled")
			return
		}
	}

	// Download
	fmt.Printf("downloading %s...\n", asset.Name)
	archiveData, err := downloadAsset(asset.URL)
	if err != nil {
		fatal("download failed: " + err.Error())
	}

	// Extract binary from archive
	binaryData, err := extractBinary(asset.Name, archiveData)
	if err != nil {
		fatal("extract failed: " + err.Error())
	}

	// Install
	if symlink {
		err = installSymlink(execPath, targetPath, installDir, latest, binaryData)
	} else {
		err = installDirect(execPath, binaryData)
	}
	if err != nil {
		fatal("install failed: " + err.Error())
	}

	// Verify
	out, verr := exec.Command(execPath, "--version").Output()
	if verr != nil {
		fatal("installed binary failed verification: " + verr.Error())
	}
	fmt.Print(strings.TrimSpace(string(out)) + "\n")
}

// resolveBinary returns the executable path, whether it's a symlink, and
// the resolved target path.
func resolveBinary() (execPath string, symlink bool, targetPath string) {
	execPath, err := os.Executable()
	if err != nil {
		fatal("cannot determine binary path: " + err.Error())
	}
	execPath, symlink, targetPath = resolveBinaryPath(execPath)
	return
}

// resolveBinaryPath is the testable core of resolveBinary.
func resolveBinaryPath(execPath string) (string, bool, string) {
	targetPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		fatal("cannot resolve binary path: " + err.Error())
	}
	// Check if execPath itself is a symlink (not just path canonicalization)
	fi, err := os.Lstat(execPath)
	if err != nil {
		fatal("cannot stat binary: " + err.Error())
	}
	isSymlink := fi.Mode()&os.ModeSymlink != 0
	return execPath, isSymlink, targetPath
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

func downloadAsset(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
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
