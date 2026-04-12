package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
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
	os.WriteFile(txtFile, []byte("hello"), 0o644)

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
	os.WriteFile(bigFile, data, 0o644)

	store := media.NewFileMediaStore()
	tool := NewLoadImageTool(dir, false, 512, store) // maxSize = 512
	ctx := WithToolContext(context.Background(), "test", "chat1")
	result := tool.Execute(ctx, map[string]any{"path": bigFile})
	if !result.IsError {
		t.Fatal("expected error for oversized file")
	}
}

func TestSubagentManager_SetMediaResolver_StoresResolver(t *testing.T) {
	manager := NewSubagentManager(nil, "gpt-test", "/tmp")

	called := false
	manager.SetMediaResolver(func(msgs []providers.Message) []providers.Message {
		called = true
		return msgs
	})

	manager.mu.RLock()
	got := manager.mediaResolver
	manager.mu.RUnlock()

	if got == nil {
		t.Fatal("expected mediaResolver to be set")
	}

	if called {
		t.Fatal("resolver should not be called during SetMediaResolver")
	}
}

func TestLoadImage_SuccessPath(t *testing.T) {
	dir := t.TempDir()

	// Create a minimal valid PNG file (8-byte signature + minimal IHDR + IEND).
	// The PNG spec requires the 8-byte magic header: 0x89 P N G \r \n 0x1a \n
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	// IHDR chunk: length(13) + "IHDR" + 1x1 px, 8-bit RGB, no interlace + CRC
	ihdr := []byte{
		0x00, 0x00, 0x00, 0x0D, // chunk length = 13
		0x49, 0x48, 0x44, 0x52, // "IHDR"
		0x00, 0x00, 0x00, 0x01, // width = 1
		0x00, 0x00, 0x00, 0x01, // height = 1
		0x08,             // bit depth = 8
		0x02,             // color type = RGB
		0x00, 0x00, 0x00, // compression, filter, interlace
		0x90, 0x77, 0x53, 0xDE, // CRC (valid for this IHDR)
	}
	// IEND chunk
	iend := []byte{
		0x00, 0x00, 0x00, 0x00, // chunk length = 0
		0x49, 0x45, 0x4E, 0x44, // "IEND"
		0xAE, 0x42, 0x60, 0x82, // CRC
	}

	pngData := make([]byte, 0, len(pngSignature)+len(ihdr)+len(iend))
	pngData = append(pngData, pngSignature...)
	pngData = append(pngData, ihdr...)
	pngData = append(pngData, iend...)

	imgPath := filepath.Join(dir, "test_image.png")
	if err := os.WriteFile(imgPath, pngData, 0o644); err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}

	store := media.NewFileMediaStore()
	tool := NewLoadImageTool(dir, false, 0, store)
	ctx := WithToolContext(context.Background(), "test", "chat1")

	result := tool.Execute(ctx, map[string]any{"path": imgPath})

	// 1. Must not be an error
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}

	// 2. Media must contain exactly one media:// ref
	if len(result.Media) != 1 {
		t.Fatalf("expected 1 media ref, got %d", len(result.Media))
	}
	if !strings.HasPrefix(result.Media[0], "media://") {
		t.Errorf("expected media ref to start with 'media://', got: %s", result.Media[0])
	}

	// 3. ForLLM must contain the [image: marker
	if !strings.Contains(result.ForLLM, "[image:") {
		t.Errorf("expected ForLLM to contain '[image:' marker, got: %s", result.ForLLM)
	}

	// 4. ForLLM should also contain the media:// ref
	if !strings.Contains(result.ForLLM, result.Media[0]) {
		t.Errorf("expected ForLLM to contain media ref %q, got: %s", result.Media[0], result.ForLLM)
	}

	// 5. Verify the ref is resolvable in the store
	resolved, err := store.Resolve(result.Media[0])
	if err != nil {
		t.Fatalf("media ref not resolvable: %v", err)
	}
	if resolved != imgPath {
		t.Errorf("expected resolved path %q, got %q", imgPath, resolved)
	}
}
