//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package channels

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitFeishuOutboundContent_MarkdownImageOnly(t *testing.T) {
	imagePath := createFeishuTempImageFile(t, "test_image.png")

	text, images := splitFeishuOutboundContent("![](" + imagePath + ")")
	if text != "" {
		t.Fatalf("expected empty text, got %q", text)
	}
	if len(images) != 1 || images[0] != imagePath {
		t.Fatalf("expected one image path %q, got %#v", imagePath, images)
	}
}

func TestSplitFeishuOutboundContent_TextAndMarkdownImage(t *testing.T) {
	imagePath := createFeishuTempImageFile(t, "test_image.png")

	text, images := splitFeishuOutboundContent("请查看图片\n![img](" + imagePath + ")")
	if text != "请查看图片" {
		t.Fatalf("expected text to be cleaned, got %q", text)
	}
	if len(images) != 1 || images[0] != imagePath {
		t.Fatalf("expected one image path %q, got %#v", imagePath, images)
	}
}

func TestSplitFeishuOutboundContent_FileURI(t *testing.T) {
	imagePath := createFeishuTempImageFile(t, "test_image.jpeg")

	text, images := splitFeishuOutboundContent("file://" + imagePath)
	if text != "" {
		t.Fatalf("expected empty text, got %q", text)
	}
	if len(images) != 1 || images[0] != imagePath {
		t.Fatalf("expected one image path %q, got %#v", imagePath, images)
	}
}

func TestSplitFeishuOutboundContent_UnsupportedFileAsText(t *testing.T) {
	nonImagePath := createFeishuTempImageFile(t, "notes.txt")

	text, images := splitFeishuOutboundContent(nonImagePath)
	if text != nonImagePath {
		t.Fatalf("expected text to keep original non-image path, got %q", text)
	}
	if len(images) != 0 {
		t.Fatalf("expected no images, got %#v", images)
	}
}

func createFeishuTempImageFile(t *testing.T, name string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}
