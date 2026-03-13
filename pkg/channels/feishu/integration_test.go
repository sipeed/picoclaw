//go:build integration

package feishu

import (
	"context"
	"os"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newIntegrationChannel(t *testing.T) *FeishuChannel {
	t.Helper()
	cfg := config.FeishuConfig{
		Enabled:   true,
		AppID:     os.Getenv("FEISHU_APP_ID"),
		AppSecret: os.Getenv("FEISHU_APP_SECRET"),
	}
	if cfg.AppID == "" || cfg.AppSecret == "" {
		t.Skip("FEISHU_APP_ID or FEISHU_APP_SECRET is not set")
	}
	ch, err := NewFeishuChannel(cfg, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("new channel: %v", err)
	}
	return ch
}

func TestIntegrationGetMessageRequiresMessageID(t *testing.T) {
	ch := newIntegrationChannel(t)
	if _, err := ch.GetMessage(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty message id")
	}
}

func TestIntegrationDriveRequiresFileToken(t *testing.T) {
	ch := newIntegrationChannel(t)
	if _, err := ch.GetDriveFile(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty file token")
	}
}
