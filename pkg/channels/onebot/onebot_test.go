package onebot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/media"
)

func TestParseMessageSegments_BlocksLoopbackInboundMediaURL(t *testing.T) {
	ch := &OneBotChannel{}
	store := media.NewFileMediaStore()

	raw := json.RawMessage(`[
		{"type":"text","data":{"text":"see attachment"}},
		{"type":"image","data":{"url":"http://127.0.0.1:8080/evil.png","file":"evil.png"}}
	]`)

	result := ch.parseMessageSegments(raw, 0, store, "onebot:test:msg1")

	if got := result.Text; got != "see attachment" {
		t.Fatalf("Text = %q, want %q", got, "see attachment")
	}
	if len(result.Media) != 0 {
		t.Fatalf("Media count = %d, want 0", len(result.Media))
	}
}

func TestParseMessageSegments_BlocksInboundMediaRedirectToLoopback(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("secret"))
	}))
	defer target.Close()

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.String(), "http://example.com/evil.png"; got != want {
			t.Fatalf("proxy request URL = %q, want %q", got, want)
		}
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer proxy.Close()

	t.Setenv("HTTP_PROXY", proxy.URL)
	t.Setenv("http_proxy", proxy.URL)
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("https_proxy", "")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("all_proxy", "")
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")

	ch := &OneBotChannel{}
	store := media.NewFileMediaStore()

	raw := json.RawMessage(`[
		{"type":"text","data":{"text":"see attachment"}},
		{"type":"image","data":{"url":"http://example.com/evil.png","file":"evil.png"}}
	]`)

	result := ch.parseMessageSegments(raw, 0, store, "onebot:test:msg-redirect")

	if got := result.Text; got != "see attachment" {
		t.Fatalf("Text = %q, want %q", got, "see attachment")
	}
	if len(result.Media) != 0 {
		t.Fatalf("Media count = %d, want 0", len(result.Media))
	}
}

func TestParseMessageSegments_StoresDownloadedMediaRef(t *testing.T) {
	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "image.png")
	if err := os.WriteFile(localPath, []byte("fake-image"), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	ch := &OneBotChannel{
		downloadFn: func(urlStr, filename string) string {
			if urlStr != "https://cdn.example.com/image.png" {
				t.Fatalf("download url = %q, want %q", urlStr, "https://cdn.example.com/image.png")
			}
			if filename != "image.png" {
				t.Fatalf("download filename = %q, want %q", filename, "image.png")
			}
			return localPath
		},
	}
	store := media.NewFileMediaStore()

	raw := json.RawMessage(`[
		{"type":"text","data":{"text":"see attachment"}},
		{"type":"image","data":{"url":"https://cdn.example.com/image.png","file":"image.png"}}
	]`)

	result := ch.parseMessageSegments(raw, 0, store, "onebot:test:msg2")

	if got := result.Text; got != "see attachment[image]" {
		t.Fatalf("Text = %q, want %q", got, "see attachment[image]")
	}
	if len(result.Media) != 1 {
		t.Fatalf("Media count = %d, want 1", len(result.Media))
	}
	if !strings.HasPrefix(result.Media[0], "media://") {
		t.Fatalf("media ref = %q, want media:// prefix", result.Media[0])
	}

	resolvedPath, meta, err := store.ResolveWithMeta(result.Media[0])
	if err != nil {
		t.Fatalf("ResolveWithMeta() error = %v", err)
	}
	if resolvedPath != localPath {
		t.Fatalf("resolved path = %q, want %q", resolvedPath, localPath)
	}
	if meta.Source != "onebot" {
		t.Fatalf("meta.Source = %q, want %q", meta.Source, "onebot")
	}
	if meta.Filename != "image.png" {
		t.Fatalf("meta.Filename = %q, want %q", meta.Filename, "image.png")
	}
}
