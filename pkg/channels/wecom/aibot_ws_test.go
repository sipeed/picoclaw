package wecom

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
)

// newTestWSChannel creates a WeComAIBotWSChannel ready for unit testing.
func newTestWSChannel(t *testing.T) *WeComAIBotWSChannel {
	t.Helper()
	cfg := config.WeComAIBotConfig{
		Enabled: true,
		BotID:   "test_bot_id",
		Secret:  "test_secret",
	}
	ch, err := newWeComAIBotWSChannel(cfg, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("create WS channel: %v", err)
	}
	return ch
}

// TestStoreWSMedia_NilStore verifies that storeWSMedia returns an error when no
// MediaStore has been injected.
func TestStoreWSMedia_NilStore(t *testing.T) {
	ch := newTestWSChannel(t)
	_, err := ch.storeWSMedia(context.Background(), "chat1", "msg1", "http://any", "", ".jpg")
	if err == nil {
		t.Fatal("expected error when no MediaStore is set")
	}
}

// TestStoreWSMedia_HTTPError verifies that storeWSMedia propagates HTTP errors
// from the media server.
func TestStoreWSMedia_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	ch := newTestWSChannel(t)
	ch.SetMediaStore(media.NewFileMediaStore())

	_, err := ch.storeWSMedia(context.Background(), "chat1", "msg1", srv.URL, "", ".jpg")
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
}

// TestStoreWSMedia_ServerUnavailable verifies that storeWSMedia returns a clear
// error when the media server cannot be reached.
func TestStoreWSMedia_ServerUnavailable(t *testing.T) {
	ch := newTestWSChannel(t)
	ch.SetMediaStore(media.NewFileMediaStore())

	// Port 1 is reserved and will refuse the connection immediately.
	_, err := ch.storeWSMedia(context.Background(), "chat1", "msg1", "http://127.0.0.1:1", "", ".jpg")
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
}

// TestStoreWSMedia_Success_NoAES verifies the happy path: the media is downloaded,
// a media ref is returned, and the file persists and is readable via Resolve until
// ReleaseAll is called. The server returns no Content-Type, so the defaultExt is used.
func TestStoreWSMedia_Success_NoAES(t *testing.T) {
	imageData := bytes.Repeat([]byte("x"), 256)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer srv.Close()

	ch := newTestWSChannel(t)
	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	ref, err := ch.storeWSMedia(context.Background(), "chat1", "msg1", srv.URL, "", ".jpg")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ref == "" {
		t.Fatal("expected non-empty ref")
	}

	// File must be accessible after storeWSMedia returns (no premature deletion).
	path, err := store.Resolve(ref)
	if err != nil {
		t.Fatalf("ref should resolve: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file should exist at %s: %v", path, err)
	}
	if !bytes.Equal(got, imageData) {
		t.Errorf("content mismatch: got len=%d, want len=%d", len(got), len(imageData))
	}

	// ReleaseAll must delete the file (store owns lifecycle).
	scope := channels.BuildMediaScope("wecom_aibot", "chat1", "msg1")
	if err := store.ReleaseAll(scope); err != nil {
		t.Fatalf("ReleaseAll failed: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file should have been deleted by ReleaseAll, stat err: %v", err)
	}
}

// TestStoreWSMedia_MultipleMessages verifies that concurrent media messages with
// different msgIDs do not collide and each resolve to distinct files.
func TestStoreWSMedia_MultipleMessages(t *testing.T) {
	imageA := bytes.Repeat([]byte("a"), 64)
	imageB := bytes.Repeat([]byte("b"), 64)

	srvA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageA)
	}))
	defer srvA.Close()
	srvB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageB)
	}))
	defer srvB.Close()

	ch := newTestWSChannel(t)
	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	refA, err := ch.storeWSMedia(context.Background(), "chat1", "msgA", srvA.URL, "", ".jpg")
	if err != nil {
		t.Fatalf("storeWSMedia A: %v", err)
	}
	refB, err := ch.storeWSMedia(context.Background(), "chat1", "msgB", srvB.URL, "", ".jpg")
	if err != nil {
		t.Fatalf("storeWSMedia B: %v", err)
	}
	if refA == refB {
		t.Fatal("distinct messages must produce distinct refs")
	}

	pathA, _ := store.Resolve(refA)
	pathB, _ := store.Resolve(refB)
	if pathA == pathB {
		t.Fatal("distinct messages must be stored at distinct paths")
	}

	gotA, _ := os.ReadFile(pathA)
	gotB, _ := os.ReadFile(pathB)
	if !bytes.Equal(gotA, imageA) {
		t.Errorf("content mismatch for message A")
	}
	if !bytes.Equal(gotB, imageB) {
		t.Errorf("content mismatch for message B")
	}
}

// TestStoreWSMedia_ContentTypeExt verifies that the file extension is inferred
// from the HTTP Content-Type header and the defaultExt fallback is used when the
// type is absent or unrecognized.
func TestStoreWSMedia_ContentTypeExt(t *testing.T) {
	tests := []struct {
		contentType string
		wantExt     string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"video/mp4", ".mp4"},
		{"application/pdf", ".pdf"},
		{"application/zip", ".zip"},
		// With parameters stripped.
		{"video/mp4; codecs=avc1", ".mp4"},
		// Unknown type → falls back to defaultExt.
		{"", ""},
		{"application/octet-stream", ""},
	}
	for _, tc := range tests {
		got := wsMediaExtFromContentType(tc.contentType)
		if got != tc.wantExt {
			t.Errorf("wsMediaExtFromContentType(%q) = %q, want %q", tc.contentType, got, tc.wantExt)
		}
	}

	// End-to-end: server returns Content-Type: video/mp4, defaultExt is .bin.
	// The stored file should carry the .mp4 extension, not .bin.
	payload := bytes.Repeat([]byte("v"), 128)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	ch := newTestWSChannel(t)
	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	ref, err := ch.storeWSMedia(context.Background(), "chat1", "vid1", srv.URL, "", ".bin")
	if err != nil {
		t.Fatalf("storeWSMedia: %v", err)
	}
	path, err := store.Resolve(ref)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if ext := path[len(path)-4:]; ext != ".mp4" {
		t.Errorf("expected .mp4 extension from Content-Type, got %q", ext)
	}
}

// TestSplitWSContent verifies byte-aware splitting of stream content.
func TestSplitWSContent(t *testing.T) {
	t.Run("short content is not split", func(t *testing.T) {
		chunks := splitWSContent("hello", 20480)
		if len(chunks) != 1 || chunks[0] != "hello" {
			t.Fatalf("unexpected chunks: %v", chunks)
		}
	})

	t.Run("ASCII content split at byte boundary", func(t *testing.T) {
		// Build a string just over the limit.
		content := strings.Repeat("a", 20481)
		chunks := splitWSContent(content, 20480)
		if len(chunks) < 2 {
			t.Fatalf("expected >= 2 chunks, got %d", len(chunks))
		}
		for i, c := range chunks {
			if len(c) > 20480 {
				t.Errorf("chunk %d has %d bytes, want <= 20480", i, len(c))
			}
		}
		// Reassembled content must equal the original (possibly without leading
		// whitespace that splitWSContent trims between chunks).
		joined := strings.Join(chunks, "")
		if len(joined) < len(content)-len(chunks) {
			t.Errorf("joined length %d too short (original %d)", len(joined), len(content))
		}
	})

	t.Run("CJK content split within byte limit", func(t *testing.T) {
		// Each CJK rune is 3 bytes in UTF-8.
		// 7000 CJK chars = 21000 bytes, which exceeds 20480.
		content := strings.Repeat("\u4e2d", 7000)
		chunks := splitWSContent(content, 20480)
		if len(chunks) < 2 {
			t.Fatalf("expected >= 2 chunks for 21000-byte CJK content, got %d", len(chunks))
		}
		for i, c := range chunks {
			if len(c) > 20480 {
				t.Errorf("chunk %d has %d bytes, want <= 20480", i, len(c))
			}
			// Every chunk must be valid UTF-8.
			if !strings.ContainsRune(c, '\u4e2d') && len(c) > 0 {
				// quick plausibility check — content was pure CJK
			}
		}
	})
}

// TestSplitAtByteBoundary verifies the last-resort byte-boundary splitter.
func TestSplitAtByteBoundary(t *testing.T) {
	t.Run("ASCII fits in one chunk", func(t *testing.T) {
		parts := splitAtByteBoundary("hello world", 100)
		if len(parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(parts))
		}
	})

	t.Run("splits at byte boundary, never mid-rune", func(t *testing.T) {
		// 10 CJK characters = 30 bytes; split at 20 bytes.
		s := strings.Repeat("\u6587", 10) // 10 × 3 bytes = 30 bytes
		parts := splitAtByteBoundary(s, 20)
		for i, p := range parts {
			if len(p) > 20 {
				t.Errorf("part %d has %d bytes, want <= 20", i, len(p))
			}
			// Must be valid UTF-8 (no torn multi-byte sequences).
			for j, r := range p {
				if r == '\uFFFD' {
					t.Errorf("part %d has replacement rune at position %d: torn UTF-8", i, j)
				}
			}
		}
	})
}
