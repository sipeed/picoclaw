package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/minio/selfupdate"
	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// httpClient is a shared HTTP client used for release checks and downloads.
// The Timeout value applies to the entire HTTP request: dialing, TLS
// handshake, redirects, and reading the response body. It is NOT only
// a connection (dial) timeout. To control lower-level timeouts (dial,
// TLS handshake, response header wait), supply a custom Transport with
// an appropriately configured net.Dialer.
var httpClient = &http.Client{Timeout: 2 * time.Minute}

func getWithRetry(rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	return utils.DoRequestWithRetry(httpClient, req)
}

// DownloadAndExtractRelease downloads a release archive (or uses a direct
// asset URL) and extracts it to a temporary directory. It returns the
// extraction directory on success. If releaseURL is empty, the latest
// release of the current project is used. platform/arch can be used to
// select the correct asset (e.g. "linux", "amd64").
func DownloadAndExtractRelease(releaseURL, platform, arch string) (string, error) {
	assetURL, checksum, err := findAssetInfo(releaseURL, platform, arch)
	if err != nil {
		return "", err
	}

	// Download asset to temp file. Use the asset URL extension so
	// extractArchive can detect the archive format (zip/tar.gz/tar).
	tmpPattern := "picoclaw-release-*"
	if u, perr := url.Parse(assetURL); perr == nil {
		base := filepath.Base(u.Path)
		lbase := strings.ToLower(base)
		switch {
		case strings.HasSuffix(lbase, ".zip"):
			tmpPattern += ".zip"
		case strings.HasSuffix(lbase, ".tar.gz") || strings.HasSuffix(lbase, ".tgz"):
			tmpPattern += ".tar.gz"
		case strings.HasSuffix(lbase, ".tar"):
			tmpPattern += ".tar"
		default:
			tmpPattern += ".archive"
		}
	} else {
		tmpPattern += ".archive"
	}

	tmpFile, err := os.CreateTemp("", tmpPattern)
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer tmpFile.Close()

	resp, err := getWithRetry(assetURL)
	if err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to download asset: status %d", resp.StatusCode)
	}

	// Stream download while computing SHA256 to avoid a second download.
	// Also show a simple progress line to stderr so users see activity.
	h := sha256.New()
	pw := &progressWriter{total: resp.ContentLength}
	mw := io.MultiWriter(tmpFile, h, pw)
	if _, err = io.Copy(mw, resp.Body); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	// ensure final progress line ends with newline
	pw.Finish()

	// verify checksum if available
	if checksum != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if !strings.EqualFold(got, checksum) {
			_ = os.Remove(tmpPath)
			return "", fmt.Errorf("checksum mismatch: got %s expected %s", got, checksum)
		}
	}

	// Extract
	destDir, err := os.MkdirTemp("", "picoclaw-extract-*")
	if err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	if err := extractArchive(tmpPath, destDir); err != nil {
		os.Remove(tmpPath)
		os.RemoveAll(destDir)
		return "", err
	}

	// cleanup archive file; keep extracted contents
	_ = os.Remove(tmpPath)
	return destDir, nil
}

// UpdateSelfFromRelease downloads the release matching the given parameters,
// extracts it and applies the binary named programName to update the
// currently running executable using minio/selfupdate.
// If releaseURL is empty, the latest release is used. If platform or arch
// is empty, runtime values are used.
func UpdateSelfFromRelease(releaseURL, platform, arch, programName string) error {
	if platform == "" {
		platform = runtime.GOOS
	}
	if arch == "" {
		arch = runtime.GOARCH
	}

	dir, err := DownloadAndExtractRelease(releaseURL, platform, arch)
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	binPath, err := findBinaryInDir(dir, programName)
	if err != nil {
		return err
	}

	// ensure executable bit on non-windows
	if runtime.GOOS != "windows" {
		_ = os.Chmod(binPath, 0o755)
	}

	f, err := os.Open(binPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Backup current executable so we can roll back if needed.
	var opts selfupdate.Options
	if exePath, err := os.Executable(); err == nil {
		opts.OldSavePath = exePath + ".old"
	}

	if err := selfupdate.Apply(f, opts); err != nil {
		return fmt.Errorf("apply update: %w", err)
	}

	return nil
}

// UpdateSelf updates the running executable by fetching the latest release
// and applying the binary matching programName.
func UpdateSelf(programName string) error {
	// By default, select the latest stable release when no explicit
	// release URL is provided. Use --nightly or a custom URL to override.
	return UpdateSelfFromRelease("", runtime.GOOS, runtime.GOARCH, programName)
}

// GetReleaseAPIURL returns the GitHub Releases API URL for the given repo owner.
// Example: owner="sky5454" -> https://api.github.com/repos/sky5454/picoclaw/releases/latest
func GetReleaseAPIURL(owner string) string {
	return fmt.Sprintf("https://api.github.com/repos/%s/picoclaw/releases/latest", owner)
}

// GetProdReleaseAPIURL returns the production release API URL (upstream).
func GetProdReleaseAPIURL() string {
	return GetReleaseAPIURL("sipeed")
}

// GetReleaseTagAPIURL returns the GitHub Releases API URL for a specific tag.
// Example: owner="sipeed", tag="nightly" -> https://api.github.com/repos/sipeed/picoclaw/releases/tags/nightly
func GetReleaseTagAPIURL(owner, tag string) string {
	return fmt.Sprintf("https://api.github.com/repos/%s/picoclaw/releases/tags/%s", owner, tag)
}

// GetNightlyReleaseAPIURL returns the nightly release API URL for the production repo.
func GetNightlyReleaseAPIURL() string {
	return GetReleaseTagAPIURL("sipeed", "nightly")
}

// findAssetURL resolves the appropriate asset URL for the given release
// selector. It accepts direct archive URLs as well as GitHub release URLs
// or empty (latest release for the project).
func findAssetInfo(releaseURL, platform, arch string) (string, string, error) {
	// returns (assetURL, sha256ChecksumHex, error)
	if looksLikeDirectAssetURL(releaseURL) {
		return "", "", fmt.Errorf("no checksum found for asset %s", releaseURL)
	}

	apiURL := buildReleaseAPIURL(releaseURL)
	if apiURL == "" {
		// If caller provided an empty releaseURL, default to the
		// production latest release API URL (stable release).
		apiURL = GetProdReleaseAPIURL()
	}

	resp, err := getWithRetry(apiURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to query releases: status %d", resp.StatusCode)
	}

	var data struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
			Digest             string `json:"digest"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", err
	}

	// Selection order: platform -> arch -> extension.
	platformLower := strings.ToLower(platform)
	archLower := strings.ToLower(arch)

	isZip := func(name string) bool {
		return strings.HasSuffix(name, ".zip")
	}
	isTarGz := func(name string) bool {
		return strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz")
	}
	isTar := func(name string) bool { return strings.HasSuffix(name, ".tar") }

	// collect indices of assets that contain platform (if provided)
	var platformIdx []int
	for i, a := range data.Assets {
		n := strings.ToLower(a.Name)
		if platform == "" || strings.Contains(n, platformLower) {
			platformIdx = append(platformIdx, i)
		}
	}

	pickBest := func(idxs []int) (string, int, bool) {
		if len(idxs) == 0 {
			return "", -1, false
		}
		// prefer arch matches within idxs; if arch was specified but
		// no arch match exists among idxs, treat as no candidate.
		var archIdx []int
		if arch != "" {
			aliases := archAliases(archLower)
			for _, i := range idxs {
				n := strings.ToLower(data.Assets[i].Name)
				for _, ali := range aliases {
					if strings.Contains(n, ali) {
						archIdx = append(archIdx, i)
						break
					}
				}
			}
			if len(archIdx) == 0 {
				return "", -1, false
			}
		}
		candidates := archIdx
		if len(candidates) == 0 {
			candidates = idxs
		}

		// extension preference
		if platformLower == "windows" {
			// prefer .zip only
			for _, i := range candidates {
				if isZip(strings.ToLower(data.Assets[i].Name)) {
					return data.Assets[i].BrowserDownloadURL, i, true
				}
			}
			// if no zip found, fallthrough to first candidate
			return data.Assets[candidates[0]].BrowserDownloadURL, candidates[0], true
		}

		// non-windows: prefer tar.gz/tgz, then tar, then zip
		for _, i := range candidates {
			if isTarGz(strings.ToLower(data.Assets[i].Name)) {
				return data.Assets[i].BrowserDownloadURL, i, true
			}
		}
		for _, i := range candidates {
			if isTar(strings.ToLower(data.Assets[i].Name)) {
				return data.Assets[i].BrowserDownloadURL, i, true
			}
		}
		for _, i := range candidates {
			if isZip(strings.ToLower(data.Assets[i].Name)) {
				return data.Assets[i].BrowserDownloadURL, i, true
			}
		}
		// fallback to first candidate
		return data.Assets[candidates[0]].BrowserDownloadURL, candidates[0], true
	}

	// Try platform matches first
	if url, idx, ok := pickBest(platformIdx); ok {
		// attempt to find checksum: prefer asset digest from API if present
		if d := strings.TrimSpace(data.Assets[idx].Digest); d != "" {
			dLower := strings.ToLower(d)
			if strings.HasPrefix(dLower, "sha256:") {
				hexpart := strings.TrimPrefix(dLower, "sha256:")
				return url, hexpart, nil
			}
			// If digest already looks like a 64-hex, return it
			if ok, _ := regexp.MatchString("(?i)^[a-f0-9]{64}$", dLower); ok {
				return url, dLower, nil
			}
		}
		// Look for checksum assets and verify by computing the asset's sha256.
		for j, a := range data.Assets {
			n := strings.ToLower(a.Name)
			if strings.Contains(n, "sha256") ||
				strings.Contains(n, "sha256sum") ||
				strings.Contains(n, "checksums") ||
				strings.HasSuffix(n, ".sha256") ||
				strings.HasSuffix(n, ".sha256sum") {
				resp2, err := getWithRetry(data.Assets[j].BrowserDownloadURL)
				if err != nil {
					continue
				}
				bs, err := io.ReadAll(resp2.Body)
				resp2.Body.Close()
				if err != nil {
					continue
				}
				if h, ok := findHashInChecksumContent(bs, url); ok {
					return url, h, nil
				}
			}
		}
		// No checksum found for the selected platform asset -> error
		return "", "", fmt.Errorf("no checksum found for asset %s", url)
	}

	// No platform match — require explicit platform+arch; fail fast.
	return "", "", fmt.Errorf("no release asset matching platform %q and arch %q", platform, arch)
}

func looksLikeDirectAssetURL(u string) bool {
	if u == "" {
		return false
	}
	lower := strings.ToLower(u)
	if strings.HasSuffix(lower, ".zip") ||
		strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".tgz") ||
		strings.HasSuffix(lower, ".tar") {
		return true
	}
	if strings.Contains(lower, "/releases/download/") {
		return true
	}
	return false
}

func buildReleaseAPIURL(releaseURL string) string {
	if releaseURL == "" {
		return ""
	}
	if strings.Contains(releaseURL, "api.github.com") {
		return releaseURL
	}
	u, err := url.Parse(releaseURL)
	if err != nil {
		return ""
	}
	if u.Host != "github.com" {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	owner := parts[0]
	repo := parts[1]
	// if tag specified
	if len(parts) >= 5 && parts[2] == "releases" && parts[3] == "tag" {
		tag := parts[4]
		return fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, tag)
	}
	// default to latest
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
}

// NOTE: helper functions to compute SHA256 from URL/path were removed
// after refactoring to stream the download and verify the checksum
// during the single download to avoid double-transfer.

// findHashInChecksumContent attempts to locate a 64-hex SHA256 in the
// checksum file content that corresponds to assetURL. It returns the
// found hash (lowercase) and true, or "", false if not found.
func findHashInChecksumContent(bs []byte, assetURL string) (string, bool) {
	s := strings.ToLower(string(bs))
	var assetBase string
	if u, err := url.Parse(assetURL); err == nil {
		assetBase = strings.ToLower(filepath.Base(u.Path))
	} else {
		assetBase = strings.ToLower(filepath.Base(assetURL))
	}
	re := regexp.MustCompile(`(?i)\b([a-f0-9]{64})\b`)
	// prefer a line containing the asset filename
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, assetBase) {
			if m := re.FindString(line); m != "" {
				return m, true
			}
		}
	}
	// fallback: if there's exactly one unique 64-hex value, return it
	matches := re.FindAllString(s, -1)
	uniq := map[string]struct{}{}
	for _, m := range matches {
		uniq[m] = struct{}{}
	}
	if len(uniq) == 1 {
		for k := range uniq {
			return k, true
		}
	}
	return "", false
}

// progressWriter implements io.Writer and prints a simple progress
// line to stderr while bytes are written. It is intended to be used
// as one writer in an io.MultiWriter so we can stream-to-disk, compute
// the sha256, and update the progress display in a single pass.
type progressWriter struct {
	total   int64
	written int64
	last    time.Time
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)
	now := time.Now()
	if pw.last.IsZero() || now.Sub(pw.last) >= 200*time.Millisecond || (pw.total > 0 && pw.written == pw.total) {
		pw.print()
		pw.last = now
	}
	return n, nil
}

func (pw *progressWriter) print() {
	if pw.total > 0 {
		pct := float64(pw.written) * 100.0 / float64(pw.total)
		fmt.Fprintf(os.Stderr, "\rDownloading: %s / %s (%.1f%%)", humanBytes(pw.written), humanBytes(pw.total), pct)
	} else {
		fmt.Fprintf(os.Stderr, "\rDownloading: %s", humanBytes(pw.written))
	}
}

func (pw *progressWriter) Finish() {
	pw.print()
	fmt.Fprintln(os.Stderr, "")
}

func humanBytes(n int64) string {
	f := float64(n)
	const (
		KB = 1024.0
		MB = KB * 1024.0
		GB = MB * 1024.0
	)
	switch {
	case f >= GB:
		return fmt.Sprintf("%.2f GB", f/GB)
	case f >= MB:
		return fmt.Sprintf("%.2f MB", f/MB)
	case f >= KB:
		return fmt.Sprintf("%.2f KB", f/KB)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// archAliases returns common name variants for an architecture string
// so we can match release asset names like "x86_64" vs Go's "amd64".
// archAliases returns name variants for an architecture string.
// If `arch` is empty or matches the local runtime.GOARCH, prefer the
// compile-time architecture aliases provided by archAliasesForLocal
// (implemented per-architecture via build tags). For other `arch`
// values we use a small synonyms map.
func archAliases(arch string) []string {
	a := strings.ToLower(arch)
	if syns, ok := archSynonyms[a]; ok {
		return syns
	}
	return []string{a}
}

var archSynonyms = map[string][]string{
	"amd64":   {"amd64", "x86_64", "x64"},
	"x86_64":  {"amd64", "x86_64", "x64"},
	"x64":     {"amd64", "x86_64", "x64"},
	"386":     {"386", "x86"},
	"x86":     {"386", "x86"},
	"arm64":   {"arm64", "aarch64"},
	"aarch64": {"arm64", "aarch64"},
	"arm":     {"arm"},
}

func extractArchive(archivePath, destDir string) error {
	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".zip") {
		return extractZip(archivePath, destDir)
	}
	// treat .tar.gz and .tgz as gzip+tar
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return extractTarGz(archivePath, destDir)
	}
	if strings.HasSuffix(lower, ".tar") {
		return extractTar(archivePath, destDir)
	}
	// fallback: try tar.gz
	return extractTarGz(archivePath, destDir)
}

func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()
	destClean := filepath.Clean(destDir)
	for _, f := range r.File {
		target := filepath.Clean(filepath.Join(destClean, f.Name))
		if !strings.HasPrefix(target, destClean+string(os.PathSeparator)) && target != destClean {
			return fmt.Errorf("path traversal detected: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.FileInfo().Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			return err
		}
		rc.Close()
		out.Close()
	}
	return nil
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	return extractTarFromReader(tr, destDir)
}

func extractTar(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	tr := tar.NewReader(f)
	return extractTarFromReader(tr, destDir)
}

// extractTarFromReader contains logic common to extracting entries from a
// tar.Reader and is used by both extractTarGz and extractTar to avoid
// duplicated code (golangci-lint: dupl).
func extractTarFromReader(tr *tar.Reader, destDir string) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Clean(filepath.Join(filepath.Clean(destDir), hdr.Name))
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) &&
			target != filepath.Clean(destDir) {
			return fmt.Errorf("path traversal detected: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

func findBinaryInDir(dir, programName string) (string, error) {
	wanted := []string{programName}
	if runtime.GOOS == "windows" {
		wanted = append([]string{programName + ".exe"}, wanted...)
	} else {
		// also accept programs with .exe in archives targeting windows
		wanted = append(wanted, programName+".exe")
	}

	var found string
	if err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := filepath.Base(p)
		for _, w := range wanted {
			if base == w {
				found = p
				return io.EOF // use EOF to stop walking early
			}
		}
		return nil
	}); err != nil && err != io.EOF {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("binary %q not found in archive", programName)
	}
	return found, nil
}

// NewUpdateCommand returns a cobra command that triggers UpdateSelfFromRelease.
func NewUpdateCommand(binaryName string) *cobra.Command {
	var urlStr, platform, arch string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check and apply updates from GitHub releases",
		RunE: func(cmd *cobra.Command, args []string) error {
			if platform == "" {
				platform = runtime.GOOS
			}
			if arch == "" {
				arch = runtime.GOARCH
			}
			fmt.Printf("Current version: %s\n", config.FormatVersion())
			if err := UpdateSelfFromRelease(urlStr, platform, arch, binaryName); err != nil {
				return err
			}
			fmt.Println("Update applied; restart to use the new version.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&urlStr, "url", "u", "", "Direct URL to download release asset or release page")
	cmd.Flags().StringVar(&platform, "platform", "", "Target platform (default: runtime.GOOS)")
	cmd.Flags().StringVar(&arch, "arch", "", "Target arch (default: runtime.GOARCH)")
	return cmd
}
