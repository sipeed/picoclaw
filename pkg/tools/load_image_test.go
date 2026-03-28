package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
)

func TestLoadImage_PathRequired(t *testing.T) {
	tool := NewLoadImageTool("/tmp", false, 0, nil)
	ctx := WithToolContext(context.Background(), "test", "chat1")
	result := tool.Execute(ctx, map[string]any{})
	if !result.IsError {
		t.Fatal("expected error for missing path")
	}
}

func TestLoadImage_NilMediaStore(t *testing.T) {
	tool := NewLoadImageTool("/tmp", false, 0, nil)
	ctx := WithToolContext(context.Background(), "test", "chat1")
	result := tool.Execute(ctx, map[string]any{"path": "test.png"})
	if !result.IsError || result.ForLLM != "media store not configured" {
		t.Fatalf("expected media store error, got: %s", result.ForLLM)
	}
}

func TestLoadImage_NoChannelContext(t *testing.T) {
	store := media.NewFileMediaStore()
	tool := NewLoadImageTool("/tmp", false, 0, store)
	// No WithToolContext — should fail
	result := tool.Execute(context.Background(), map[string]any{"path": "test.png"})
	if !result.IsError || result.ForLLM != "no target channel/chat available" {
		t.Fatalf("expected channel error, got: %s", result.ForLLM)
	}
}

func TestLoadImage_NonImageFile(t *testing.T) {
	dir := t.TempDir()
	txtFile := filepath.Join(dir, "readme.txt")
	os.WriteFile(txtFile, []byte("hello"), 0644)

	store := media.NewFileMediaStore()
	tool := NewLoadImageTool(dir, false, 0, store)
	ctx := WithToolContext(context.Background(), "test", "chat1")
	result := tool.Execute(ctx, map[string]any{"path": txtFile})
	if !result.IsError {
		t.Fatal("expected error for non-image file")
	}
}

func TestLoadImage_DefaultMaxSize(t *testing.T) {
	tool := NewLoadImageTool("/tmp", false, 0, nil)
	if tool.maxFileSize != config.DefaultMaxMediaSize {
		t.Errorf("expected default max size %d, got %d", config.DefaultMaxMediaSize, tool.maxFileSize)
	}
}

func TestLoadImage_FileTooLarge(t *testing.T) {
	dir := t.TempDir()
	bigFile := filepath.Join(dir, "big.png")
	// Create a file with PNG header but exceeding max size
	data := make([]byte, 1024)
	copy(data, []byte{0x89, 0x50, 0x4E, 0x47}) // PNG magic bytes
	os.WriteFile(bigFile, data, 0644)

	store := media.NewFileMediaStore()
	tool := NewLoadImageTool(dir, false, 512, store) // maxSize = 512
	ctx := WithToolContext(context.Background(), "test", "chat1")
	result := tool.Execute(ctx, map[string]any{"path": bigFile})
	if !result.IsError {
		t.Fatal("expected error for oversized file")
	}
}
