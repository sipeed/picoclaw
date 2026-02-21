package utils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadFile_WithDefaultOptions(t *testing.T) {
	t.Parallel()

	const body = "hello from media download"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	path := DownloadFile(srv.URL, "sample.txt", DownloadOptions{})
	if path == "" {
		t.Fatal("DownloadFile() returned empty path")
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	if got := filepath.Base(path); got == "sample.txt" {
		t.Fatalf("expected uuid-prefixed filename, got %q", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error: %v", err)
	}
	if string(data) != body {
		t.Fatalf("downloaded content = %q, want %q", string(data), body)
	}
}
