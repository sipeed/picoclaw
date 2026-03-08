package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
