package feishu

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
)

func TestUploadMultipartFromReaderValidation(t *testing.T) {
	ch := &FeishuChannel{}
	if _, err := ch.uploadMultipartFromReader(context.Background(), "", bytes.NewReader([]byte("x")), 4); err == nil {
		t.Fatal("expected error for empty upload id")
	}
	if _, err := ch.uploadMultipartFromReader(context.Background(), "upload_1", nil, 4); err == nil {
		t.Fatal("expected error for nil reader")
	}
	if _, err := ch.uploadMultipartFromReader(context.Background(), "upload_1", bytes.NewReader([]byte("x")), 0); err == nil {
		t.Fatal("expected error for invalid block size")
	}
}

func TestUploadMultipartReadPattern(t *testing.T) {
	data := []byte("abcdefghij")
	reader := bytes.NewReader(data)
	buf := make([]byte, 4)
	var chunks [][]byte
	for {
		n, err := io.ReadFull(reader, buf)
		if err == io.EOF {
			break
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			t.Fatalf("unexpected read error: %v", err)
		}
		if n <= 0 {
			break
		}
		chunks = append(chunks, append([]byte(nil), buf[:n]...))
		if err == io.ErrUnexpectedEOF {
			break
		}
	}
	if len(chunks) != 3 {
		t.Fatalf("chunk count = %d, want 3", len(chunks))
	}
	if string(chunks[0]) != "abcd" || string(chunks[1]) != "efgh" || string(chunks[2]) != "ij" {
		t.Fatalf("unexpected chunks: %q %q %q", chunks[0], chunks[1], chunks[2])
	}
}

func TestUploadLargeDriveFileValidation(t *testing.T) {
	ch := &FeishuChannel{}
	if _, err := ch.UploadLargeDriveFile(context.Background(), "", "", bytes.NewReader([]byte("x")), 1); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := ch.UploadLargeDriveFile(context.Background(), "", "demo.bin", nil, 1); err == nil {
		t.Fatal("expected error for nil reader")
	}
	if _, err := ch.UploadLargeDriveFile(context.Background(), "", "demo.bin", bytes.NewReader([]byte("x")), 0); err == nil {
		t.Fatal("expected error for invalid size")
	}
}

func TestUploadLargeDriveFileFromPathValidation(t *testing.T) {
	ch := &FeishuChannel{}
	if _, err := ch.UploadLargeDriveFileFromPath(context.Background(), "", ""); err == nil {
		t.Fatal("expected error for empty path")
	}
	dir := t.TempDir()
	if _, err := ch.UploadLargeDriveFileFromPath(context.Background(), "", dir); err == nil {
		t.Fatal("expected error for directory path")
	}
}

func TestUploadLargeDriveFileFromPathMissingFile(t *testing.T) {
	ch := &FeishuChannel{}
	_, err := ch.UploadLargeDriveFileFromPath(context.Background(), "", "missing.bin")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got: %v", err)
	}
}
