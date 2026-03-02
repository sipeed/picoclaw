package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/media"
)

func TestSendFileTool_MissingPath(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewSendFileTool("/tmp", false, store)
	tool.SetContext("feishu", "chat123")

	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error for missing path")
	}
}

func TestSendFileTool_NoContext(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewSendFileTool("/tmp", false, store)
	// no SetContext call

	result := tool.Execute(context.Background(), map[string]any{"path": "/tmp/test.txt"})
	if !result.IsError {
		t.Fatal("expected error when no channel context")
	}
}

func TestSendFileTool_NoMediaStore(t *testing.T) {
	tool := NewSendFileTool("/tmp", false, nil)
	tool.SetContext("feishu", "chat123")

	result := tool.Execute(context.Background(), map[string]any{"path": "/tmp/test.txt"})
	if !result.IsError {
		t.Fatal("expected error when no media store")
	}
}

func TestSendFileTool_Directory(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewSendFileTool("/tmp", false, store)
	tool.SetContext("feishu", "chat123")

	result := tool.Execute(context.Background(), map[string]any{"path": "/tmp"})
	if !result.IsError {
		t.Fatal("expected error for directory path")
	}
}

func TestSendFileTool_Success(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "photo.png")
	if err := os.WriteFile(testFile, []byte("fake png"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := media.NewFileMediaStore()
	tool := NewSendFileTool(dir, false, store)
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
	tool := NewSendFileTool(dir, false, store)
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

func TestDetectMediaType(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"photo.png", "image/png"},
		{"anim.gif", "image/gif"},
		{"photo.webp", "image/webp"},
		{"doc.pdf", "application/pdf"},
		{"data.bin", "application/octet-stream"},
		{"noext", "application/octet-stream"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := detectMediaType(tt.path)
			if got != tt.want {
				t.Errorf("detectMediaType(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
