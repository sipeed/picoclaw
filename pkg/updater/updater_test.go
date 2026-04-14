package updater

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// matchesMagic checks whether the file at path looks like a platform binary
// by inspecting magic bytes (ELF for linux, MZ for windows).
func matchesMagic(path, platform string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	buf := make([]byte, 4)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}
	if n >= 4 && buf[0] == 0x7f && buf[1] == 'E' && buf[2] == 'L' && buf[3] == 'F' {
		return strings.Contains(platform, "linux"), nil
	}
	if n >= 2 && buf[0] == 'M' && buf[1] == 'Z' {
		return strings.Contains(platform, "windows"), nil
	}
	return false, nil
}

// TestDownloadAndExtractRelease_RealPlatforms downloads the latest release
// asset for multiple platform/arch combos and inspects the extracted
// artifacts to ensure a binary-like file is present. This is a network test
// and is skipped in short mode.
func TestDownloadAndExtractRelease_RealPlatforms(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network tests in short mode")
	}

	combos := []struct{ platform, arch string }{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"windows", "amd64"},
		{"windows", "arm64"},
	}

	apiURL := GetProdReleaseAPIURL()
	for _, c := range combos {
		t.Run(c.platform+"_"+c.arch, func(t *testing.T) {
			assetURL, checksum, err := findAssetInfo(apiURL, c.platform, c.arch)
			if err != nil {
				// If no checksum could be located for this asset, skip this
				// combo rather than failing — we require signed/checksummed
				// releases for real-network tests.
				t.Skipf("skipping %s/%s: %v", c.platform, c.arch, err)
			}
			t.Logf("asset URL: %s checksum: %s", assetURL, checksum)

			// Pass the release API URL (not the direct asset URL) so
			// DownloadAndExtractRelease can locate and verify the asset.
			dir, err := DownloadAndExtractRelease(apiURL, c.platform, c.arch)
			if err != nil {
				t.Fatalf("DownloadAndExtractRelease failed for %s/%s: %v", c.platform, c.arch, err)
			}
			defer os.RemoveAll(dir)

			var found bool
			_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				info, err := d.Info()
				if err != nil {
					return err
				}
				if info.Size() < 64 {
					return nil
				}
				ok, err := matchesMagic(path, c.platform)
				if err != nil {
					return err
				}
				if ok {
					found = true
					t.Logf("found artifact: %s (size=%d)", path, info.Size())
					// continue walking to list all
				}
				return nil
			})
			if !found {
				t.Fatalf("no binary-like artifact found for %s/%s", c.platform, c.arch)
			}
		})
	}
}
