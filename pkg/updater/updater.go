package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/minio/selfupdate"
	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/pkg/config"
)

// DownloadAndExtractRelease downloads a release archive (or uses a direct
// asset URL) and extracts it to a temporary directory. It returns the
// extraction directory on success. If releaseURL is empty, the latest
// release of the current project is used. platform/arch can be used to
// select the correct asset (e.g. "linux", "amd64").
func DownloadAndExtractRelease(releaseURL, platform, arch string) (string, error) {
	assetURL, err := findAssetURL(releaseURL, platform, arch)
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
	defer func() { _ = tmpFile.Close() }()

	resp, err := http.Get(assetURL)
	if err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to download asset: status %d", resp.StatusCode)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpPath)
		return "", err
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
		_ = os.Chmod(binPath, 0755)
	}

	f, err := os.Open(binPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := selfupdate.Apply(f, selfupdate.Options{}); err != nil {
		return fmt.Errorf("apply update: %w", err)
	}

	return nil
}

// UpdateSelf updates the running executable by fetching the latest release
// and applying the binary matching programName.
func UpdateSelf(programName string) error {
	// By default, let findAssetURL select the nightly build when no explicit
	// release URL is provided. Passing an empty releaseURL triggers that path.
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
func findAssetURL(releaseURL, platform, arch string) (string, error) {
	if looksLikeDirectAssetURL(releaseURL) {
		return releaseURL, nil
	}

	apiURL := buildReleaseAPIURL(releaseURL)
	if apiURL == "" {
		// If caller provided an empty releaseURL, default to the nightly tag
		// from the production repo. Otherwise fall back to the production
		// latest release API URL.
		if strings.TrimSpace(releaseURL) == "" {
			apiURL = GetNightlyReleaseAPIURL()
		} else {
			apiURL = GetProdReleaseAPIURL()
		}
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to query releases: status %d", resp.StatusCode)
	}

	var data struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
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

	pickBest := func(idxs []int) (string, bool) {
		if len(idxs) == 0 {
			return "", false
		}
		// prefer arch matches within idxs
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
					return data.Assets[i].BrowserDownloadURL, true
				}
			}
			// if no zip found, fallthrough to first candidate
			return data.Assets[candidates[0]].BrowserDownloadURL, true
		}

		// non-windows: prefer tar.gz/tgz, then tar, then zip
		for _, i := range candidates {
			if isTarGz(strings.ToLower(data.Assets[i].Name)) {
				return data.Assets[i].BrowserDownloadURL, true
			}
		}
		for _, i := range candidates {
			if isTar(strings.ToLower(data.Assets[i].Name)) {
				return data.Assets[i].BrowserDownloadURL, true
			}
		}
		for _, i := range candidates {
			if isZip(strings.ToLower(data.Assets[i].Name)) {
				return data.Assets[i].BrowserDownloadURL, true
			}
		}
		// fallback to first candidate
		return data.Assets[candidates[0]].BrowserDownloadURL, true
	}

	// Try platform matches first
	if url, ok := pickBest(platformIdx); ok {
		return url, nil
	}

	// If no platform matches, try arch-only matches
	if arch != "" {
		var archOnlyIdx []int
		for i, a := range data.Assets {
			n := strings.ToLower(a.Name)
			if strings.Contains(n, archLower) {
				archOnlyIdx = append(archOnlyIdx, i)
			}
		}
		if url, ok := pickBest(archOnlyIdx); ok {
			return url, nil
		}
	}

	// Fallback to first asset
	if len(data.Assets) > 0 {
		return data.Assets[0].BrowserDownloadURL, nil
	}
	return "", errors.New("no suitable release asset found")
}

func looksLikeDirectAssetURL(u string) bool {
	if u == "" {
		return false
	}
	lower := strings.ToLower(u)
	if strings.HasSuffix(lower, ".zip") || strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") || strings.HasSuffix(lower, ".tar") {
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

	for _, f := range r.File {
		fp := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(fp, f.Mode())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
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
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, hdr.Name)
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

func extractTar(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, hdr.Name)
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
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
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
	})
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
