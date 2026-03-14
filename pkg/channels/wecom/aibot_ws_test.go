package wecom

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
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

// TestStoreWSImage_NilStore verifies that storeWSImage returns an error when no
// MediaStore has been injected.
func TestStoreWSImage_NilStore(t *testing.T) {
	ch := newTestWSChannel(t)
	_, err := ch.storeWSImage(context.Background(), "chat1", "msg1", "http://any", "")
	if err == nil {
		t.Fatal("expected error when no MediaStore is set")
	}
}

// TestStoreWSImage_HTTPError verifies that storeWSImage propagates HTTP errors
// from the image server.
func TestStoreWSImage_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	ch := newTestWSChannel(t)
	ch.SetMediaStore(media.NewFileMediaStore())

	_, err := ch.storeWSImage(context.Background(), "chat1", "msg1", srv.URL, "")
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
}

// TestStoreWSImage_ServerUnavailable verifies that storeWSImage returns a clear
// error when the image server cannot be reached.
func TestStoreWSImage_ServerUnavailable(t *testing.T) {
	ch := newTestWSChannel(t)
	ch.SetMediaStore(media.NewFileMediaStore())

	// Port 1 is reserved and will refuse the connection immediately.
	_, err := ch.storeWSImage(context.Background(), "chat1", "msg1", "http://127.0.0.1:1", "")
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
}

// TestStoreWSImage_Success_NoAES verifies the happy path: the image is downloaded,
// a media ref is returned, and the file persists and is readable via Resolve until
// ReleaseAll is called.
func TestStoreWSImage_Success_NoAES(t *testing.T) {
	imageData := bytes.Repeat([]byte("x"), 256)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer srv.Close()

	ch := newTestWSChannel(t)
	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	ref, err := ch.storeWSImage(context.Background(), "chat1", "msg1", srv.URL, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ref == "" {
		t.Fatal("expected non-empty ref")
	}

	// File must be accessible after storeWSImage returns (no premature deletion).
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

// TestStoreWSImage_MultipleMessages verifies that concurrent image messages with
// different msgIDs do not collide and each resolve to distinct files.
func TestStoreWSImage_MultipleMessages(t *testing.T) {
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

	refA, err := ch.storeWSImage(context.Background(), "chat1", "msgA", srvA.URL, "")
	if err != nil {
		t.Fatalf("storeWSImage A: %v", err)
	}
	refB, err := ch.storeWSImage(context.Background(), "chat1", "msgB", srvB.URL, "")
	if err != nil {
		t.Fatalf("storeWSImage B: %v", err)
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
