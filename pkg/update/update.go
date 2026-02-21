// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	// GitHubRepo is the upstream repository for release checks
	GitHubRepo = "sipeed/picoclaw"

	// releaseAPIURL is the GitHub API endpoint for the latest release
	releaseAPIURL = "https://api.github.com/repos/" + GitHubRepo + "/releases/latest"

	// checkInterval is how often the periodic hint checks for updates
	checkInterval = 24 * time.Hour

	// httpTimeout is the timeout for GitHub API and download requests
	httpTimeout = 30 * time.Second
)

// httpClient is the HTTP client used for all requests. Tests can replace it.
var httpClient = &http.Client{Timeout: httpTimeout}

// ReleaseInfo holds information about a GitHub release
type ReleaseInfo struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
	HTMLURL string         `json:"html_url"`
}

// ReleaseAsset holds information about a release asset
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// checkCache stores the last update check result
type checkCache struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version"`
	HTMLURL       string    `json:"html_url"`
}

// CheckLatest fetches the latest release info from GitHub
func CheckLatest() (*ReleaseInfo, error) {
	req, err := http.NewRequest("GET", releaseAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "picoclaw-updater")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parsing release info: %w", err)
	}

	return &release, nil
}

// IsNewer returns true if the remote version is newer than the current version.
// Both versions should be semver strings, optionally prefixed with "v".
func IsNewer(current, remote string) bool {
	curParts := parseSemver(current)
	remParts := parseSemver(remote)

	if curParts == nil || remParts == nil {
		return false
	}

	for i := 0; i < 3; i++ {
		if remParts[i] > curParts[i] {
			return true
		}
		if remParts[i] < curParts[i] {
			return false
		}
	}
	return false
}

// parseSemver extracts [major, minor, patch] from a version string.
// Accepts "v1.2.3", "1.2.3", "v0.1.2-42-gabcdef", etc.
func parseSemver(v string) []int {
	v = strings.TrimPrefix(v, "v")

	// Strip anything after a hyphen (pre-release/build metadata)
	if idx := strings.Index(v, "-"); idx != -1 {
		v = v[:idx]
	}

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return nil
	}

	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		nums[i] = n
	}
	return nums
}

// AssetName returns the expected asset filename for the current platform.
// Asset naming convention: picoclaw_{OS}_{Arch}.tar.gz (or .zip for Windows)
func AssetName() string {
	osName := normalizeOS(runtime.GOOS)
	archName := normalizeArch(runtime.GOARCH)

	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("picoclaw_%s_%s.%s", osName, archName, ext)
}

func normalizeOS(goos string) string {
	switch goos {
	case "darwin":
		return "Darwin"
	case "linux":
		return "Linux"
	case "windows":
		return "Windows"
	case "freebsd":
		return "Freebsd"
	default:
		// Title case the first letter
		return strings.ToUpper(goos[:1]) + goos[1:]
	}
}

func normalizeArch(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "arm64"
	case "arm":
		return "armv6"
	case "riscv64":
		return "riscv64"
	case "mips64":
		return "mips64"
	case "s390x":
		return "s390x"
	default:
		return goarch
	}
}

// FindAssetURL finds the download URL for the current platform in a release
func FindAssetURL(release *ReleaseInfo) (string, error) {
	want := AssetName()
	for _, asset := range release.Assets {
		if asset.Name == want {
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("no release asset found for %s", want)
}

// DownloadAndReplace downloads the release asset and replaces the current binary.
// It writes progress to the provided writer.
func DownloadAndReplace(downloadURL string, progress io.Writer) error {
	// 1. Download the archive
	fmt.Fprintf(progress, "Downloading %s...\n", filepath.Base(downloadURL))
	resp, err := httpClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("downloading release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "picoclaw-update-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, filepath.Base(downloadURL))
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	written, err := io.Copy(archiveFile, resp.Body)
	archiveFile.Close()
	if err != nil {
		return fmt.Errorf("saving download: %w", err)
	}
	fmt.Fprintf(progress, "Downloaded %.1f MB\n", float64(written)/1024/1024)

	// 2. Extract the binary
	fmt.Fprintf(progress, "Extracting...\n")
	binaryPath, err := extractBinary(archivePath, tmpDir)
	if err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	// 3. Replace the current binary
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return fmt.Errorf("resolving binary path: %w", err)
	}

	fmt.Fprintf(progress, "Replacing %s...\n", currentBinary)
	if err := replaceBinary(binaryPath, currentBinary); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	// 4. Recreate pico symlink if it exists next to the binary
	picoSymlink := filepath.Join(filepath.Dir(currentBinary), "pico")
	if target, err := os.Readlink(picoSymlink); err == nil {
		// Only recreate if it was pointing at a picoclaw binary
		if strings.Contains(target, "picoclaw") {
			os.Remove(picoSymlink)
			os.Symlink(filepath.Base(currentBinary), picoSymlink)
		}
	}

	return nil
}

// extractBinary extracts the picoclaw binary from a tar.gz or zip archive
func extractBinary(archivePath, destDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractFromZip(archivePath, destDir)
	}
	return extractFromTarGz(archivePath, destDir)
}

func extractFromTarGz(archivePath, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("opening gzip: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("reading tar: %w", err)
		}

		name := filepath.Base(header.Name)
		if name == "picoclaw" || name == "picoclaw.exe" {
			outPath := filepath.Join(destDir, name)
			outFile, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return "", err
			}
			outFile.Close()
			return outPath, nil
		}
	}
	return "", fmt.Errorf("picoclaw binary not found in archive")
}

func extractFromZip(archivePath, destDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name == "picoclaw" || name == "picoclaw.exe" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			outPath := filepath.Join(destDir, name)
			outFile, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
			if err != nil {
				rc.Close()
				return "", err
			}
			if _, err := io.Copy(outFile, rc); err != nil {
				outFile.Close()
				rc.Close()
				return "", err
			}
			outFile.Close()
			rc.Close()
			return outPath, nil
		}
	}
	return "", fmt.Errorf("picoclaw binary not found in archive")
}

// replaceBinary atomically replaces the old binary with the new one
func replaceBinary(newPath, oldPath string) error {
	// Rename old binary to .bak
	bakPath := oldPath + ".bak"
	if err := os.Rename(oldPath, bakPath); err != nil {
		return fmt.Errorf("backing up current binary: %w", err)
	}

	// Copy new binary into place (can't rename across filesystems)
	src, err := os.Open(newPath)
	if err != nil {
		// Restore backup
		os.Rename(bakPath, oldPath)
		return fmt.Errorf("opening new binary: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(oldPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		os.Rename(bakPath, oldPath)
		return fmt.Errorf("writing new binary: %w", err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Rename(bakPath, oldPath)
		return fmt.Errorf("copying new binary: %w", err)
	}
	dst.Close()

	// Remove backup
	os.Remove(bakPath)
	return nil
}

// --- Periodic hint support ---

// cacheFilePath returns the path to the update check cache file.
// Tests can override this variable.
var cacheFilePath = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "last_update_check.json")
}

// CheckHint checks if a newer version is available, using a cached result
// if the last check was within the check interval. Returns the latest version
// string if an update is available, or empty string if up-to-date or on error.
// This is designed to be called from a goroutine and never block.
func CheckHint(currentVersion string) string {
	// 1. Try to read cache
	cache, err := loadCache()
	if err == nil && time.Since(cache.LastCheck) < checkInterval {
		// Cache is fresh, use it
		if IsNewer(currentVersion, cache.LatestVersion) {
			return cache.LatestVersion
		}
		return ""
	}

	// 2. Cache is stale or missing, check GitHub
	release, err := CheckLatest()
	if err != nil {
		return ""
	}

	// 3. Update cache
	newCache := checkCache{
		LastCheck:     time.Now(),
		LatestVersion: release.TagName,
		HTMLURL:       release.HTMLURL,
	}
	saveCache(&newCache)

	if IsNewer(currentVersion, release.TagName) {
		return release.TagName
	}
	return ""
}

func loadCache() (*checkCache, error) {
	data, err := os.ReadFile(cacheFilePath())
	if err != nil {
		return nil, err
	}
	var cache checkCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

func saveCache(cache *checkCache) {
	data, err := json.Marshal(cache)
	if err != nil {
		return
	}
	dir := filepath.Dir(cacheFilePath())
	os.MkdirAll(dir, 0o755)
	os.WriteFile(cacheFilePath(), data, 0o644)
}
