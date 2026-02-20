package channels

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mellium.im/xmpp/jid"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newTestXMPPChannel(t *testing.T) *XMPPChannel {
	t.Helper()
	ch, err := NewXMPPChannel(config.XMPPConfig{
		JID:      "bot@example.com",
		Password: "pass",
	}, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("NewXMPPChannel error: %v", err)
	}
	return ch
}

func TestXMPPChannelDedupBySender(t *testing.T) {
	ch := newTestXMPPChannel(t)
	now := time.Now()

	if ch.shouldDropDuplicate("alice@example.com", "hi", now) {
		t.Fatalf("expected first message to be accepted")
	}
	if !ch.shouldDropDuplicate("alice@example.com", "hi", now.Add(time.Second)) {
		t.Fatalf("expected duplicate within window to be dropped")
	}
	if ch.shouldDropDuplicate("bob@example.com", "hi", now.Add(time.Second)) {
		t.Fatalf("expected different sender to be accepted")
	}
	if ch.shouldDropDuplicate("alice@example.com", "hi", now.Add(3*time.Second)) {
		t.Fatalf("expected message after window to be accepted")
	}
}

func TestXMPPChannelUploadRespectsServerMax(t *testing.T) {
	ch := newTestXMPPChannel(t)
	ch.uploadMax = 1

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "a.txt")
	if err := os.WriteFile(path, []byte("hi"), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	uploadJID, err := jid.Parse("upload.example.com")
	if err != nil {
		t.Fatalf("parse upload jid: %v", err)
	}

	_, err = ch.uploadFile(context.Background(), uploadJID, path)
	if err == nil {
		t.Fatal("expected upload to fail due to size limit")
	}
	if !strings.Contains(err.Error(), "exceeds server limit") {
		t.Fatalf("unexpected error: %v", err)
	}
}
