//go:build whatsapp_native

package whatsapp

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestHandleIncoming_DoesNotConsumeGenericCommandsLocally(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &WhatsAppNativeChannel{
		BaseChannel: channels.NewBaseChannel("whatsapp_native", config.WhatsAppConfig{}, messageBus, nil),
		runCtx:      context.Background(),
	}

	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Sender: types.NewJID("1001", types.DefaultUserServer),
				Chat:   types.NewJID("1001", types.DefaultUserServer),
			},
			ID:       "mid1",
			PushName: "Alice",
		},
		Message: &waE2E.Message{
			Conversation: proto.String("/new"),
		},
	}

	ch.handleIncoming(evt)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		t.Fatal("timeout waiting for message to be forwarded")
		return
	case inbound, ok := <-messageBus.InboundChan():
		if !ok {
			t.Fatal("expected inbound message to be forwarded")
		}
		if inbound.Channel != "whatsapp_native" {
			t.Fatalf("channel=%q", inbound.Channel)
		}
		if inbound.Content != "/new" {
			t.Fatalf("content=%q", inbound.Content)
		}
	}
}

func TestImageFilenameFromMime(t *testing.T) {
	tests := []struct {
		mime string
		want string
	}{
		{"image/png", "photo.png"},
		{"image/jpeg", "photo.jpg"},
		{"image/gif", "photo.gif"},
		{"image/webp", "photo.webp"},
		{"", "photo.jpg"},
	}
	for _, tt := range tests {
		if got := imageFilenameFromMime(tt.mime); got != tt.want {
			t.Errorf("imageFilenameFromMime(%q) = %q, want %q", tt.mime, got, tt.want)
		}
	}
}

// whatsappTestClient returns a disconnected whatsmeow client (valid store) for exercising handleIncoming media paths.
func whatsappTestClient(t *testing.T) *whatsmeow.Client {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	container := sqlstore.NewWithDB(db, "sqlite", waLog.Noop)
	if err := container.Upgrade(ctx); err != nil {
		t.Fatal(err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return whatsmeow.NewClient(device, waLog.Noop)
}

func TestHandleIncoming_ImageCaptionDownloadFailsStillForwardsCaption(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &WhatsAppNativeChannel{
		BaseChannel: channels.NewBaseChannel("whatsapp_native", config.WhatsAppConfig{}, messageBus, nil),
		runCtx:      context.Background(),
		client:      whatsappTestClient(t),
	}

	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Sender: types.NewJID("1001", types.DefaultUserServer),
				Chat:   types.NewJID("1001", types.DefaultUserServer),
			},
			ID:       "mid-img",
			PushName: "Bob",
		},
		Message: &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption: proto.String("look at this"),
			},
		},
	}

	ch.handleIncoming(evt)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		t.Fatal("timeout waiting for inbound message")
	case inbound, ok := <-messageBus.InboundChan():
		if !ok {
			t.Fatal("expected inbound message")
		}
		if inbound.Content != "look at this" {
			t.Fatalf("content=%q, want caption only when download fails", inbound.Content)
		}
		if len(inbound.Media) != 0 {
			t.Fatalf("expected no media refs when download fails, got %v", inbound.Media)
		}
	}
}

func TestHandleIncoming_ImageOnlyNoCaptionDownloadFails_NoInbound(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &WhatsAppNativeChannel{
		BaseChannel: channels.NewBaseChannel("whatsapp_native", config.WhatsAppConfig{}, messageBus, nil),
		runCtx:      context.Background(),
		client:      whatsappTestClient(t),
	}

	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Sender: types.NewJID("1001", types.DefaultUserServer),
				Chat:   types.NewJID("1001", types.DefaultUserServer),
			},
			ID:       "mid-img2",
			PushName: "Bob",
		},
		Message: &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{},
		},
	}

	ch.handleIncoming(evt)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	select {
	case inbound, ok := <-messageBus.InboundChan():
		if ok {
			t.Fatalf("unexpected inbound when media-only and download fails: %+v", inbound)
		}
	case <-ctx.Done():
		// expected: no message
	}
}
