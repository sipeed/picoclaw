package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestEnsurePicoChannel_FreshConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	changed, err := h.ensurePicoChannel("")
	if err != nil {
		t.Fatalf("ensurePicoChannel() error = %v", err)
	}
	if !changed {
		t.Fatal("ensurePicoChannel() should report changed on a fresh config")
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.Channels.Pico.Enabled {
		t.Error("expected Pico to be enabled after setup")
	}
	if cfg.Channels.Pico.Token == "" {
		t.Error("expected a non-empty token after setup")
	}
}

func TestEnsurePicoChannel_DoesNotEnableTokenQuery(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	if _, err := h.ensurePicoChannel(""); err != nil {
		t.Fatalf("ensurePicoChannel() error = %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Channels.Pico.AllowTokenQuery {
		t.Error("setup must not enable allow_token_query by default")
	}
}

func TestEnsurePicoChannel_DoesNotSetWildcardOrigins(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	if _, err := h.ensurePicoChannel("http://localhost:18800"); err != nil {
		t.Fatalf("ensurePicoChannel() error = %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	for _, origin := range cfg.Channels.Pico.AllowOrigins {
		if origin == "*" {
			t.Error("setup must not set wildcard origin '*'")
		}
	}
}

func TestEnsurePicoChannel_NoOriginWithoutCaller(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	if _, err := h.ensurePicoChannel(""); err != nil {
		t.Fatalf("ensurePicoChannel() error = %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Without a caller origin, allow_origins stays empty (CheckOrigin
	// allows all when the list is empty, so the channel still works).
	if len(cfg.Channels.Pico.AllowOrigins) != 0 {
		t.Errorf("allow_origins = %v, want empty when no caller origin", cfg.Channels.Pico.AllowOrigins)
	}
}

func TestEnsurePicoChannel_SetsCallerOrigin(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	lanOrigin := "http://192.168.1.9:18800"
	if _, err := h.ensurePicoChannel(lanOrigin); err != nil {
		t.Fatalf("ensurePicoChannel() error = %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Channels.Pico.AllowOrigins) != 1 || cfg.Channels.Pico.AllowOrigins[0] != lanOrigin {
		t.Errorf("allow_origins = %v, want [%s]", cfg.Channels.Pico.AllowOrigins, lanOrigin)
	}
}

func TestEnsurePicoChannel_PreservesUserSettings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")

	// Pre-configure with custom user settings
	cfg := config.DefaultConfig()
	cfg.Channels.Pico.Enabled = true
	cfg.Channels.Pico.Token = "user-custom-token"
	cfg.Channels.Pico.AllowTokenQuery = true
	cfg.Channels.Pico.AllowOrigins = []string{"https://myapp.example.com"}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)

	changed, err := h.ensurePicoChannel("")
	if err != nil {
		t.Fatalf("ensurePicoChannel() error = %v", err)
	}
	if changed {
		t.Error("ensurePicoChannel() should not change a fully configured config")
	}

	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Channels.Pico.Token != "user-custom-token" {
		t.Errorf("token = %q, want %q", cfg.Channels.Pico.Token, "user-custom-token")
	}
	if !cfg.Channels.Pico.AllowTokenQuery {
		t.Error("user's allow_token_query=true must be preserved")
	}
	if len(cfg.Channels.Pico.AllowOrigins) != 1 || cfg.Channels.Pico.AllowOrigins[0] != "https://myapp.example.com" {
		t.Errorf("allow_origins = %v, want [https://myapp.example.com]", cfg.Channels.Pico.AllowOrigins)
	}
}

func TestEnsurePicoChannel_Idempotent(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	origin := "http://localhost:18800"

	// First call sets things up
	if _, err := h.ensurePicoChannel(origin); err != nil {
		t.Fatalf("first ensurePicoChannel() error = %v", err)
	}

	cfg1, _ := config.LoadConfig(configPath)
	token1 := cfg1.Channels.Pico.Token

	// Second call should be a no-op
	changed, err := h.ensurePicoChannel(origin)
	if err != nil {
		t.Fatalf("second ensurePicoChannel() error = %v", err)
	}
	if changed {
		t.Error("second ensurePicoChannel() should not report changed")
	}

	cfg2, _ := config.LoadConfig(configPath)
	if cfg2.Channels.Pico.Token != token1 {
		t.Error("token should not change on subsequent calls")
	}
}

func TestHandlePicoSetup_IncludesRequestOrigin(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	req := httptest.NewRequest("POST", "/api/pico/setup", nil)
	req.Header.Set("Origin", "http://10.0.0.5:3000")
	rec := httptest.NewRecorder()

	h.handlePicoSetup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Channels.Pico.AllowOrigins) != 1 || cfg.Channels.Pico.AllowOrigins[0] != "http://10.0.0.5:3000" {
		t.Errorf("allow_origins = %v, want [http://10.0.0.5:3000]", cfg.Channels.Pico.AllowOrigins)
	}
}

func TestHandlePicoSetup_Response(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	req := httptest.NewRequest("POST", "/api/pico/setup", nil)
	rec := httptest.NewRecorder()

	h.handlePicoSetup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["token"] == nil || resp["token"] == "" {
		t.Error("response should contain a non-empty token")
	}
	if resp["ws_url"] == nil || resp["ws_url"] == "" {
		t.Error("response should contain ws_url")
	}
	if resp["enabled"] != true {
		t.Error("response should have enabled=true")
	}
	if resp["changed"] != true {
		t.Error("response should have changed=true on first setup")
	}
}
