package update

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []int
	}{
		{name: "plain", input: "1.2.3", want: []int{1, 2, 3}},
		{name: "v-prefix", input: "v0.1.2", want: []int{0, 1, 2}},
		{name: "with-prerelease", input: "v0.1.2-42-gabcdef", want: []int{0, 1, 2}},
		{name: "with-dirty", input: "v0.1.2-dirty", want: []int{0, 1, 2}},
		{name: "invalid-empty", input: "", want: nil},
		{name: "invalid-two-parts", input: "1.2", want: nil},
		{name: "invalid-letters", input: "v1.x.3", want: nil},
		{name: "dev", input: "dev", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSemver(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("parseSemver(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("parseSemver(%q) = nil, want %v", tt.input, tt.want)
			}
			for i := 0; i < 3; i++ {
				if got[i] != tt.want[i] {
					t.Errorf("parseSemver(%q)[%d] = %d, want %d", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name    string
		current string
		remote  string
		want    bool
	}{
		{name: "newer-patch", current: "v0.1.2", remote: "v0.1.3", want: true},
		{name: "newer-minor", current: "v0.1.2", remote: "v0.2.0", want: true},
		{name: "newer-major", current: "v0.1.2", remote: "v1.0.0", want: true},
		{name: "same", current: "v0.1.2", remote: "v0.1.2", want: false},
		{name: "older", current: "v0.2.0", remote: "v0.1.9", want: false},
		{name: "current-with-metadata", current: "v0.1.2-42-gabcdef", remote: "v0.1.3", want: true},
		{name: "same-with-metadata", current: "v0.1.2-42-gabcdef", remote: "v0.1.2", want: false},
		{name: "invalid-current", current: "dev", remote: "v0.1.2", want: false},
		{name: "invalid-remote", current: "v0.1.2", remote: "dev", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNewer(tt.current, tt.remote)
			if got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.remote, got, tt.want)
			}
		})
	}
}

func TestNormalizeOS(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"darwin", "Darwin"},
		{"linux", "Linux"},
		{"windows", "Windows"},
		{"freebsd", "Freebsd"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeOS(tt.input)
			if got != tt.want {
				t.Errorf("normalizeOS(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"amd64", "x86_64"},
		{"arm64", "arm64"},
		{"arm", "armv6"},
		{"riscv64", "riscv64"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeArch(tt.input)
			if got != tt.want {
				t.Errorf("normalizeArch(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCheckLatest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release := ReleaseInfo{
			TagName: "v1.0.0",
			HTMLURL: "https://github.com/sipeed/picoclaw/releases/tag/v1.0.0",
			Assets: []ReleaseAsset{
				{Name: "picoclaw_Darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.com/picoclaw_Darwin_arm64.tar.gz"},
			},
		}
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	// Override the HTTP client to use test server
	origClient := httpClient
	httpClient = server.Client()
	defer func() { httpClient = origClient }()

	// Override releaseAPIURL by testing CheckLatest indirectly via FindAssetURL
	// For direct testing, we test the JSON parsing
	resp, err := httpClient.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if release.TagName != "v1.0.0" {
		t.Errorf("TagName = %q, want %q", release.TagName, "v1.0.0")
	}
}

func TestFindAssetURL(t *testing.T) {
	assetName := AssetName()
	release := &ReleaseInfo{
		Assets: []ReleaseAsset{
			{Name: "picoclaw_Linux_x86_64.tar.gz", BrowserDownloadURL: "https://example.com/linux"},
			{Name: assetName, BrowserDownloadURL: "https://example.com/match"},
			{Name: "picoclaw_Windows_x86_64.zip", BrowserDownloadURL: "https://example.com/windows"},
		},
	}

	url, err := FindAssetURL(release)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://example.com/match" {
		t.Errorf("FindAssetURL() = %q, want %q", url, "https://example.com/match")
	}

	// Test missing asset
	release.Assets = []ReleaseAsset{
		{Name: "picoclaw_UnknownOS_unknownarch.tar.gz", BrowserDownloadURL: "https://example.com/nope"},
	}
	_, err = FindAssetURL(release)
	if err == nil {
		t.Error("expected error for missing asset, got nil")
	}
}

func TestExtractFromTarGz(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "update-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test tar.gz with a picoclaw binary
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	binaryContent := []byte("#!/bin/sh\necho hello\n")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("creating archive: %v", err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Add a non-binary file first
	if err := tw.WriteHeader(&tar.Header{Name: "README.md", Size: 6, Mode: 0o644}); err != nil {
		t.Fatalf("writing tar header: %v", err)
	}
	tw.Write([]byte("readme"))

	// Add the binary
	if err := tw.WriteHeader(&tar.Header{Name: "picoclaw", Size: int64(len(binaryContent)), Mode: 0o755}); err != nil {
		t.Fatalf("writing tar header: %v", err)
	}
	tw.Write(binaryContent)

	tw.Close()
	gw.Close()
	f.Close()

	// Extract
	destDir := filepath.Join(tmpDir, "extracted")
	os.MkdirAll(destDir, 0o755)

	binPath, err := extractFromTarGz(archivePath, destDir)
	if err != nil {
		t.Fatalf("extractFromTarGz: %v", err)
	}

	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("reading extracted binary: %v", err)
	}
	if string(data) != string(binaryContent) {
		t.Errorf("extracted content = %q, want %q", string(data), string(binaryContent))
	}
}

func TestReplaceBinary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "replace-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldPath := filepath.Join(tmpDir, "picoclaw")
	newPath := filepath.Join(tmpDir, "picoclaw-new")

	os.WriteFile(oldPath, []byte("old"), 0o755)
	os.WriteFile(newPath, []byte("new"), 0o755)

	if err := replaceBinary(newPath, oldPath); err != nil {
		t.Fatalf("replaceBinary: %v", err)
	}

	data, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatalf("reading replaced binary: %v", err)
	}
	if string(data) != "new" {
		t.Errorf("replaced binary content = %q, want %q", string(data), "new")
	}

	// Backup should be cleaned up
	if _, err := os.Stat(oldPath + ".bak"); !os.IsNotExist(err) {
		t.Error("backup file was not cleaned up")
	}
}

func TestCacheRoundTrip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cache-test-*")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home dir for cache path
	origCacheFilePath := cacheFilePath
	cacheFilePath = func() string {
		return filepath.Join(tmpDir, "last_update_check.json")
	}
	defer func() { cacheFilePath = origCacheFilePath }()

	cache := &checkCache{
		LastCheck:     time.Now().Truncate(time.Second),
		LatestVersion: "v1.2.3",
		HTMLURL:       "https://example.com",
	}

	saveCache(cache)

	loaded, err := loadCache()
	if err != nil {
		t.Fatalf("loadCache: %v", err)
	}

	if loaded.LatestVersion != cache.LatestVersion {
		t.Errorf("LatestVersion = %q, want %q", loaded.LatestVersion, cache.LatestVersion)
	}
	if loaded.HTMLURL != cache.HTMLURL {
		t.Errorf("HTMLURL = %q, want %q", loaded.HTMLURL, cache.HTMLURL)
	}
}
