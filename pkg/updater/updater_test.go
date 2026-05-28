package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

type testReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest,omitempty"`
}

type testReleasePayload struct {
	TagName string             `json:"tag_name"`
	Assets  []testReleaseAsset `json:"assets"`
}

const testReleaseAPIPath = "/api.github.com/repos/sipeed/picoclaw/releases/latest"

// TestDownloadAndExtractRelease_IntegrationLatestRelease downloads the latest
// public release for a single platform as an opt-in smoke test.
func TestDownloadAndExtractRelease_IntegrationLatestRelease(t *testing.T) {
	if os.Getenv("PICOCLAW_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test (set PICOCLAW_INTEGRATION_TESTS=1 to enable)")
	}
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	const platform = "linux"
	const arch = "amd64"
	apiURL := GetProdReleaseAPIURL()
	assetURL, checksum, err := findAssetInfo(apiURL, platform, arch)
	if err != nil {
		t.Fatalf("findAssetInfo failed for %s/%s: %v", platform, arch, err)
	}
	t.Logf("asset URL: %s checksum: %s", assetURL, checksum)

	dir, err := DownloadAndExtractRelease(apiURL, platform, arch)
	if err != nil {
		t.Fatalf("DownloadAndExtractRelease failed for %s/%s: %v", platform, arch, err)
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
		ok, err := matchesMagic(path, platform)
		if err != nil {
			return err
		}
		if ok {
			found = true
			t.Logf("found artifact: %s (size=%d)", path, info.Size())
		}
		return nil
	})
	if !found {
		t.Fatalf("no binary-like artifact found for %s/%s", platform, arch)
	}
}

func TestFindAssetInfo_SelectsPreferredAsset(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testReleaseAPIPath:
			writeReleasePayload(w, testReleasePayload{
				TagName: "v0.2.6",
				Assets: []testReleaseAsset{
					{
						Name:               "picoclaw_Linux_x86_64.zip",
						BrowserDownloadURL: server.URL + "/assets/picoclaw_Linux_x86_64.zip",
						Digest:             "sha256:" + strings.Repeat("1", 64),
					},
					{
						Name:               "picoclaw_Linux_x86_64.tar.gz",
						BrowserDownloadURL: server.URL + "/assets/picoclaw_Linux_x86_64.tar.gz",
						Digest:             "sha256:" + strings.Repeat("2", 64),
					},
					{
						Name:               "picoclaw_Windows_x86_64.zip",
						BrowserDownloadURL: server.URL + "/assets/picoclaw_Windows_x86_64.zip",
						Digest:             "sha256:" + strings.Repeat("3", 64),
					},
					{
						Name:               "picoclaw_Windows_arm64.zip",
						BrowserDownloadURL: server.URL + "/assets/picoclaw_Windows_arm64.zip",
						Digest:             "sha256:" + strings.Repeat("4", 64),
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	withTestHTTPClient(t, server.Client())

	tests := []struct {
		name         string
		platform     string
		arch         string
		wantURL      string
		wantChecksum string
	}{
		{
			name:         "linux prefers tar.gz over zip",
			platform:     "linux",
			arch:         "amd64",
			wantURL:      server.URL + "/assets/picoclaw_Linux_x86_64.tar.gz",
			wantChecksum: strings.Repeat("2", 64),
		},
		{
			name:         "windows amd64 matches x86_64 zip",
			platform:     "windows",
			arch:         "amd64",
			wantURL:      server.URL + "/assets/picoclaw_Windows_x86_64.zip",
			wantChecksum: strings.Repeat("3", 64),
		},
		{
			name:         "windows arm64 matches arm64 zip",
			platform:     "windows",
			arch:         "arm64",
			wantURL:      server.URL + "/assets/picoclaw_Windows_arm64.zip",
			wantChecksum: strings.Repeat("4", 64),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotURL, gotChecksum, err := findAssetInfo(server.URL+testReleaseAPIPath, tc.platform, tc.arch)
			if err != nil {
				t.Fatalf(
					"findAssetInfo(%q, %q, %q) error: %v",
					server.URL+testReleaseAPIPath,
					tc.platform,
					tc.arch,
					err,
				)
			}
			if gotURL != tc.wantURL {
				t.Fatalf("assetURL = %q, want %q", gotURL, tc.wantURL)
			}
			if gotChecksum != tc.wantChecksum {
				t.Fatalf("checksum = %q, want %q", gotChecksum, tc.wantChecksum)
			}
		})
	}
}

func TestFindAssetInfo_UsesChecksumAssetWhenDigestMissing(t *testing.T) {
	const checksum = "77b564f36da6d1e02169d0ecc837728eecb9ef983c317d9186ac9651798b924c"

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testReleaseAPIPath:
			writeReleasePayload(w, testReleasePayload{
				TagName: "v0.2.6",
				Assets: []testReleaseAsset{
					{
						Name:               "picoclaw_Windows_x86_64.zip",
						BrowserDownloadURL: server.URL + "/assets/picoclaw_Windows_x86_64.zip",
					},
					{
						Name:               "checksums.txt",
						BrowserDownloadURL: server.URL + "/assets/checksums.txt",
					},
				},
			})
		case "/assets/checksums.txt":
			_, _ = io.WriteString(w, checksum+"  picoclaw_Windows_x86_64.zip\n")
		case "/assets/picoclaw_Windows_x86_64.zip":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	withTestHTTPClient(t, server.Client())

	gotURL, gotChecksum, err := findAssetInfo(server.URL+testReleaseAPIPath, "windows", "amd64")
	if err != nil {
		t.Fatalf("findAssetInfo returned error: %v", err)
	}
	if gotURL != server.URL+"/assets/picoclaw_Windows_x86_64.zip" {
		t.Fatalf("assetURL = %q, want %q", gotURL, server.URL+"/assets/picoclaw_Windows_x86_64.zip")
	}
	if gotChecksum != checksum {
		t.Fatalf("checksum = %q, want %q", gotChecksum, checksum)
	}
}

func TestDownloadAndExtractRelease_ExtractsTarGz(t *testing.T) {
	tarGzContent := buildTestTarGz(t, map[string]string{
		"picoclaw_Linux_x86_64/picoclaw": "test linux binary payload",
	})
	sum := sha256.Sum256(tarGzContent)
	checksum := hex.EncodeToString(sum[:])

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testReleaseAPIPath:
			writeReleasePayload(w, testReleasePayload{
				TagName: "v0.2.6",
				Assets: []testReleaseAsset{
					{
						Name:               "picoclaw_Linux_x86_64.tar.gz",
						BrowserDownloadURL: server.URL + "/assets/picoclaw_Linux_x86_64.tar.gz",
						Digest:             "sha256:" + checksum,
					},
				},
			})
		case "/assets/picoclaw_Linux_x86_64.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(tarGzContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	withTestHTTPClient(t, server.Client())

	dir, err := DownloadAndExtractRelease(server.URL+testReleaseAPIPath, "linux", "amd64")
	if err != nil {
		t.Fatalf("DownloadAndExtractRelease returned error: %v", err)
	}
	defer os.RemoveAll(dir)

	binPath, err := findBinaryInDir(dir, "picoclaw")
	if err != nil {
		t.Fatalf("findBinaryInDir returned error: %v", err)
	}

	bs, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("ReadFile extracted asset: %v", err)
	}
	if got := string(bs); got != "test linux binary payload" {
		t.Fatalf("extracted content = %q, want %q", got, "test linux binary payload")
	}
}

func TestDownloadAndExtractRelease_RetriesTransientAssetFailure(t *testing.T) {
	zipContent := buildTestZip(t, map[string]string{
		"picoclaw.exe": "test windows binary payload",
	})
	sum := sha256.Sum256(zipContent)
	checksum := hex.EncodeToString(sum[:])

	var assetAttempts int
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api.github.com/repos/sipeed/picoclaw/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(
				w,
				`{"tag_name":"v0.2.6","assets":[{"name":"picoclaw_Windows_x86_64.zip","browser_download_url":%q,"digest":"sha256:%s"}]}`,
				server.URL+"/assets/picoclaw_Windows_x86_64.zip",
				checksum,
			)
		case "/assets/picoclaw_Windows_x86_64.zip":
			assetAttempts++
			if assetAttempts == 1 {
				w.WriteHeader(http.StatusGatewayTimeout)
				return
			}
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	withTestHTTPClient(t, server.Client())

	dir, err := DownloadAndExtractRelease(
		server.URL+"/api.github.com/repos/sipeed/picoclaw/releases/latest",
		"windows",
		"amd64",
	)
	if err != nil {
		t.Fatalf("DownloadAndExtractRelease returned error: %v", err)
	}
	defer os.RemoveAll(dir)

	if assetAttempts != 2 {
		t.Fatalf("asset attempts = %d, want 2", assetAttempts)
	}

	bs, err := os.ReadFile(filepath.Join(dir, "picoclaw.exe"))
	if err != nil {
		t.Fatalf("ReadFile extracted asset: %v", err)
	}
	if got := string(bs); got != "test windows binary payload" {
		t.Fatalf("extracted content = %q, want %q", got, "test windows binary payload")
	}
}

func buildTestZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("Create zip entry %q: %v", name, err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatalf("Write zip entry %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close zip writer: %v", err)
	}
	return buf.Bytes()
}

func buildTestTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(content)),
		}); err != nil {
			t.Fatalf("Write tar header %q: %v", name, err)
		}
		if _, err := io.WriteString(tw, content); err != nil {
			t.Fatalf("Write tar entry %q: %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("Close tar writer: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("Close gzip writer: %v", err)
	}
	return buf.Bytes()
}

func writeReleasePayload(w http.ResponseWriter, payload testReleasePayload) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func withTestHTTPClient(t *testing.T, client *http.Client) {
	t.Helper()

	origClient := httpClient
	httpClient = client
	httpClient.Timeout = 5 * time.Second
	t.Cleanup(func() {
		httpClient = origClient
	})
}
