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

    // Download asset to temp file
    tmpFile, err := os.CreateTemp("", "picoclaw-release-*.archive")
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
    // Default to test repo for now. Use GetProdReleaseAPIURL() for production.
    return UpdateSelfFromRelease(GetTestReleaseAPIURL(), runtime.GOOS, runtime.GOARCH, programName)
}

// GetReleaseAPIURL returns the GitHub Releases API URL for the given repo owner.
// Example: owner="sky5454" -> https://api.github.com/repos/sky5454/picoclaw/releases/latest
func GetReleaseAPIURL(owner string) string {
    return fmt.Sprintf("https://api.github.com/repos/%s/picoclaw/releases/latest", owner)
}

// GetTestReleaseAPIURL returns the test release API URL (user fork).
func GetTestReleaseAPIURL() string {
    return GetReleaseAPIURL("sky5454")
}

// GetProdReleaseAPIURL returns the production release API URL (upstream).
func GetProdReleaseAPIURL() string {
    return GetReleaseAPIURL("sipeed")
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
        // If caller provided an empty releaseURL, default to test repo.
        // Otherwise fall back to production repo API URL.
        if strings.TrimSpace(releaseURL) == "" {
            apiURL = GetTestReleaseAPIURL()
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

    // prefer exact platform+arch match
    var fallback string
    for _, a := range data.Assets {
        n := strings.ToLower(a.Name)
        if platform != "" && arch != "" {
            if strings.Contains(n, strings.ToLower(platform)) && strings.Contains(n, strings.ToLower(arch)) {
                return a.BrowserDownloadURL, nil
            }
        }
        if platform != "" && strings.Contains(n, strings.ToLower(platform)) {
            if fallback == "" {
                fallback = a.BrowserDownloadURL
            }
        }
        if fallback == "" {
            fallback = a.BrowserDownloadURL
        }
    }

    if fallback != "" {
        return fallback, nil
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
