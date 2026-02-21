package dashboard

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestChannelToggle(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Enabled = false

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	handler := channelToggleHandler(cfg, configPath)

	form := url.Values{}
	form.Set("name", "telegram")
	form.Set("enabled", "true")
	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/channels/toggle", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if !cfg.Channels.Telegram.Enabled {
		t.Error("expected Telegram to be enabled after toggle")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file should have been saved")
	}
}

func TestChannelUpdateTelegram(t *testing.T) {
	cfg := config.DefaultConfig()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	handler := channelUpdateHandler(cfg, configPath)

	form := url.Values{}
	form.Set("name", "telegram")
	form.Set("enabled", "true")
	form.Set("token", "bot123456:ABC-DEF")
	form.Set("proxy", "socks5://proxy:1080")
	form.Set("allow_from", "user1, user2, user3")
	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/channels/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if !cfg.Channels.Telegram.Enabled {
		t.Error("expected Telegram enabled")
	}
	if cfg.Channels.Telegram.Token != "bot123456:ABC-DEF" {
		t.Errorf("expected token 'bot123456:ABC-DEF', got %q", cfg.Channels.Telegram.Token)
	}
	if cfg.Channels.Telegram.Proxy != "socks5://proxy:1080" {
		t.Errorf("expected proxy 'socks5://proxy:1080', got %q", cfg.Channels.Telegram.Proxy)
	}
	if len(cfg.Channels.Telegram.AllowFrom) != 3 {
		t.Fatalf("expected 3 allow_from entries, got %d", len(cfg.Channels.Telegram.AllowFrom))
	}
	if cfg.Channels.Telegram.AllowFrom[0] != "user1" {
		t.Errorf("expected allow_from[0] = 'user1', got %q", cfg.Channels.Telegram.AllowFrom[0])
	}
}

func TestChannelUpdateDiscord(t *testing.T) {
	cfg := config.DefaultConfig()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	handler := channelUpdateHandler(cfg, configPath)

	form := url.Values{}
	form.Set("name", "discord")
	form.Set("enabled", "true")
	form.Set("token", "discord-bot-token-xyz")
	form.Set("mention_only", "true")
	form.Set("allow_from", "guild1")
	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/channels/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if cfg.Channels.Discord.Token != "discord-bot-token-xyz" {
		t.Errorf("expected discord token, got %q", cfg.Channels.Discord.Token)
	}
	if !cfg.Channels.Discord.MentionOnly {
		t.Error("expected MentionOnly to be true")
	}
}

func TestChannelUpdateUnknown(t *testing.T) {
	cfg := config.DefaultConfig()

	handler := channelUpdateHandler(cfg, "")

	form := url.Values{}
	form.Set("name", "nonexistent")
	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/channels/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown channel, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Unknown channel") {
		t.Error("expected 'Unknown channel' error message")
	}
}

func TestChannelEditFragment(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Token = "test-token-123"
	cfg.Channels.Telegram.Enabled = true

	handler := fragmentChannelEdit(cfg)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/fragments/channel-edit?name=telegram", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "<form") {
		t.Error("expected HTML form in response")
	}
	if !strings.Contains(body, "test-token-123") {
		t.Error("expected token value in form")
	}
	if !strings.Contains(body, "checked") {
		t.Error("expected checked attribute for enabled channel")
	}
}

func TestChannelEditFragmentUnknown(t *testing.T) {
	cfg := config.DefaultConfig()

	handler := fragmentChannelEdit(cfg)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/fragments/channel-edit?name=unknown", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestChannelToggleUnknown(t *testing.T) {
	cfg := config.DefaultConfig()

	handler := channelToggleHandler(cfg, "")

	form := url.Values{}
	form.Set("name", "nonexistent")
	form.Set("enabled", "true")
	req := httptest.NewRequest(http.MethodPost, "/dashboard/crud/channels/toggle", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown channel, got %d", w.Code)
	}
}

func TestChannelUpdateMethodNotAllowed(t *testing.T) {
	cfg := config.DefaultConfig()

	handler := channelUpdateHandler(cfg, "")

	req := httptest.NewRequest(http.MethodGet, "/dashboard/crud/channels/update", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}
