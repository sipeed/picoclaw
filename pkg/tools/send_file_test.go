package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
)

func TestSendFileTool_MissingPath(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewSendFileTool("/tmp", false, 0, store)
	tool.SetContext("feishu", "chat123")

	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error for missing path")
	}
}

func TestSendFileTool_NoContext(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewSendFileTool("/tmp", false, 0, store)
	// no SetContext call

	result := tool.Execute(context.Background(), map[string]any{"path": "/tmp/test.txt"})
	if !result.IsError {
		t.Fatal("expected error when no channel context")
	}
}

func TestSendFileTool_NoMediaStore(t *testing.T) {
	tool := NewSendFileTool("/tmp", false, 0, nil)
	tool.SetContext("feishu", "chat123")

	result := tool.Execute(context.Background(), map[string]any{"path": "/tmp/test.txt"})
	if !result.IsError {
		t.Fatal("expected error when no media store")
	}
}

func TestSendFileTool_Directory(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewSendFileTool("/tmp", false, 0, store)
	tool.SetContext("feishu", "chat123")

	result := tool.Execute(context.Background(), map[string]any{"path": "/tmp"})
	if !result.IsError {
		t.Fatal("expected error for directory path")
	}
}

func TestSendFileTool_FileTooLarge(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "big.bin")
	// Create a file larger than the limit
	if err := os.WriteFile(testFile, make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	store := media.NewFileMediaStore()
	tool := NewSendFileTool(dir, false, 512, store) // 512 byte limit
	tool.SetContext("feishu", "chat123")

	result := tool.Execute(context.Background(), map[string]any{"path": testFile})
	if !result.IsError {
		t.Fatal("expected error for oversized file")
	}
	if !strings.Contains(result.ForLLM, "too large") {
		t.Errorf("expected 'too large' in error, got %q", result.ForLLM)
	}
}

func TestSendFileTool_DefaultMaxSize(t *testing.T) {
	tool := NewSendFileTool("/tmp", false, 0, nil)
	if tool.maxFileSize != config.DefaultMaxMediaSize {
		t.Errorf("expected default max size %d, got %d", config.DefaultMaxMediaSize, tool.maxFileSize)
	}
}

func TestSendFileTool_Success(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "photo.png")
	if err := os.WriteFile(testFile, []byte("fake png"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := media.NewFileMediaStore()
	tool := NewSendFileTool(dir, false, 0, store)
	tool.SetContext("feishu", "chat123")

	result := tool.Execute(context.Background(), map[string]any{"path": testFile})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if len(result.Media) != 1 {
		t.Fatalf("expected 1 media ref, got %d", len(result.Media))
	}
	if result.Media[0][:8] != "media://" {
		t.Errorf("expected media:// ref, got %q", result.Media[0])
	}
}

func TestSendFileTool_CustomFilename(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "img.jpg")
	if err := os.WriteFile(testFile, []byte("fake jpg"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := media.NewFileMediaStore()
	tool := NewSendFileTool(dir, false, 0, store)
	tool.SetContext("telegram", "chat456")

	result := tool.Execute(context.Background(), map[string]any{
		"path":     testFile,
		"filename": "my-photo.jpg",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if len(result.Media) != 1 {
		t.Fatalf("expected 1 media ref, got %d", len(result.Media))
	}
}

func TestSendFileTool_AllowsWhitelistedMediaTempPath(t *testing.T) {
	workspace := t.TempDir()
	mediaDir := media.TempDir()
	if err := os.MkdirAll(mediaDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(mediaDir) error = %v", err)
	}

	testFile, err := os.CreateTemp(mediaDir, "send-file-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp(mediaDir) error = %v", err)
	}
	testPath := testFile.Name()
	if _, err := testFile.WriteString("forward me"); err != nil {
		testFile.Close()
		t.Fatalf("WriteString(testFile) error = %v", err)
	}
	if err := testFile.Close(); err != nil {
		t.Fatalf("Close(testFile) error = %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(testPath) })

	pattern := regexp.MustCompile(
		"^" + regexp.QuoteMeta(filepath.Clean(mediaDir)) + "(?:" + regexp.QuoteMeta(string(os.PathSeparator)) + "|$)",
	)

	store := media.NewFileMediaStore()
	tool := NewSendFileTool(workspace, true, 0, store, []*regexp.Regexp{pattern})
	tool.SetContext("feishu", "chat123")

	result := tool.Execute(context.Background(), map[string]any{"path": testPath})
	if result.IsError {
		t.Fatalf("expected whitelisted temp media file to be sendable, got: %s", result.ForLLM)
	}
	if len(result.Media) != 1 {
		t.Fatalf("expected 1 media ref, got %d", len(result.Media))
	}
}

func TestSendFileTool_URLDownload(t *testing.T) {
	// Start a test HTTP server serving a fake image
	fakeImage := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(fakeImage)
	}))
	defer srv.Close()

	store := media.NewFileMediaStore()
	tool := NewSendFileTool(t.TempDir(), false, 0, store)
	tool.SetContext("telegram", "chat123")

	result := tool.Execute(context.Background(), map[string]any{
		"path":     srv.URL + "/photos/42.png",
		"filename": "cat.png",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if len(result.Media) != 1 {
		t.Fatalf("expected 1 media ref, got %d", len(result.Media))
	}
}

func TestSendFileTool_URLDownloadDefaultFilename(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data"))
	}))
	defer srv.Close()

	store := media.NewFileMediaStore()
	tool := NewSendFileTool(t.TempDir(), false, 0, store)
	tool.SetContext("telegram", "chat123")

	result := tool.Execute(context.Background(), map[string]any{
		"path": srv.URL + "/mcp/photos/42",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	// filename should be derived from URL path: "42"
	if !strings.Contains(result.ForLLM, `"42"`) {
		t.Errorf("expected filename '42' in result, got %q", result.ForLLM)
	}
}

func TestSendFileTool_URLDownloadHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	store := media.NewFileMediaStore()
	tool := NewSendFileTool(t.TempDir(), false, 0, store)
	tool.SetContext("telegram", "chat123")

	result := tool.Execute(context.Background(), map[string]any{
		"path": srv.URL + "/missing.jpg",
	})
	if !result.IsError {
		t.Fatal("expected error for HTTP 404")
	}
	if !strings.Contains(result.ForLLM, "404") {
		t.Errorf("expected 404 in error, got %q", result.ForLLM)
	}
}

func TestSendFileTool_URLTempFilePersistsForMediaStore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("data"))
	}))
	defer srv.Close()

	store := media.NewFileMediaStore()
	tool := NewSendFileTool(t.TempDir(), false, 0, store)
	tool.SetContext("telegram", "chat123")

	result := tool.Execute(context.Background(), map[string]any{
		"path": srv.URL + "/photo.jpg",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	// The temp file must still exist so MediaStore can resolve and read it
	ref := result.Media[0]
	resolved, err := store.Resolve(ref)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if _, err := os.Stat(resolved); err != nil {
		t.Errorf("temp file should persist for MediaStore, but got: %v", err)
	}
	t.Cleanup(func() { os.Remove(resolved) })
}

func TestCleanupTempFiles(t *testing.T) {
	if err := os.MkdirAll(sendFileTempDir(), 0o700); err != nil {
		t.Fatal(err)
	}

	// Create an "old" temp file
	oldFile := filepath.Join(sendFileTempDir(), "dl_test_old.tmp")
	if err := os.WriteFile(oldFile, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Set mtime to 2 hours ago
	past := time.Now().Add(-2 * time.Hour)
	os.Chtimes(oldFile, past, past)

	// Create a "new" temp file
	newFile := filepath.Join(sendFileTempDir(), "dl_test_new.tmp")
	if err := os.WriteFile(newFile, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(newFile) })

	CleanupTempFiles()

	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("expected old temp file to be deleted")
	}
	if _, err := os.Stat(newFile); err != nil {
		t.Error("expected new temp file to remain")
	}
}

func TestFilenameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/photos/cat.jpg", "cat.jpg"},
		{"https://example.com/mcp/photos/42", "42"},
		{"https://example.com/", "download"},
		{"https://example.com", "download"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := filenameFromURL(tt.url)
			if got != tt.want {
				t.Errorf("filenameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestIsHTTPURL(t *testing.T) {
	if !isHTTPURL("https://example.com/photo.jpg") {
		t.Error("expected true for https URL")
	}
	if !isHTTPURL("http://localhost:8080/file") {
		t.Error("expected true for http URL")
	}
	if isHTTPURL("/tmp/file.jpg") {
		t.Error("expected false for local path")
	}
	if isHTTPURL("file:///tmp/file.jpg") {
		t.Error("expected false for file:// URL")
	}
}

func TestSendFileTool_URLDownloadTooLarge(t *testing.T) {
	// maxFileSize を小さくして、ダウンロード自体は成功するがファイルサイズチェックで弾かれるケース
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 1024))
	}))
	defer srv.Close()

	store := media.NewFileMediaStore()
	tool := NewSendFileTool(t.TempDir(), false, 512, store) // 512 byte limit
	tool.SetContext("telegram", "chat123")

	result := tool.Execute(context.Background(), map[string]any{
		"path": fmt.Sprintf("%s/big.bin", srv.URL),
	})
	if !result.IsError {
		t.Fatal("expected error for oversized downloaded file")
	}
	if !strings.Contains(result.ForLLM, "too large") {
		t.Errorf("expected 'too large' in error, got %q", result.ForLLM)
	}
}

func TestDetectMediaType_MagicBytes(t *testing.T) {
	dir := t.TempDir()

	// Minimal valid PNG header
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	pngFile := filepath.Join(dir, "image.dat") // wrong extension, but valid PNG bytes
	if err := os.WriteFile(pngFile, pngHeader, 0o644); err != nil {
		t.Fatal(err)
	}

	got := detectMediaType(pngFile)
	if got != "image/png" {
		t.Errorf("expected image/png from magic bytes, got %q", got)
	}
}

func TestDetectMediaType_FallbackToExtension(t *testing.T) {
	dir := t.TempDir()

	// File with unrecognizable content but known extension
	txtFile := filepath.Join(dir, "readme.txt")
	if err := os.WriteFile(txtFile, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := detectMediaType(txtFile)
	// text/plain or similar — just verify it's not application/octet-stream
	if got == "application/octet-stream" {
		t.Errorf("expected extension-based MIME for .txt, got %q", got)
	}
}

func TestDetectMediaType_UnknownFallsToOctetStream(t *testing.T) {
	dir := t.TempDir()

	// File with no extension and random bytes
	unknownFile := filepath.Join(dir, "mystery")
	if err := os.WriteFile(unknownFile, []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatal(err)
	}

	got := detectMediaType(unknownFile)
	if got != "application/octet-stream" {
		t.Errorf("expected application/octet-stream, got %q", got)
	}
}
