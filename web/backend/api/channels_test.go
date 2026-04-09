package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestHandleGetChannelConfig_ReturnsSecretPresenceWithoutLeakingSecrets(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.Channels.Feishu.Enabled = true
	cfg.Channels.Feishu.AppID = "cli_test_app"
	cfg.Channels.Feishu.AppSecret = *config.NewSecureString("feishu-secret-from-security")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/channels/feishu/config", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"GET /api/channels/feishu/config status = %d, want %d, body=%s",
			rec.Code,
			http.StatusOK,
			rec.Body.String(),
		)
	}
	if strings.Contains(rec.Body.String(), "feishu-secret-from-security") {
		t.Fatalf("response leaked secret value: %s", rec.Body.String())
	}

	var resp struct {
		Config            map[string]any `json:"config"`
		ConfiguredSecrets []string       `json:"configured_secrets"`
		ConfigKey         string         `json:"config_key"`
		Variant           string         `json:"variant"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got := resp.ConfigKey; got != "feishu" {
		t.Fatalf("config_key = %q, want %q", got, "feishu")
	}
	if got := resp.Config["app_id"]; got != "cli_test_app" {
		t.Fatalf("config.app_id = %#v, want %q", got, "cli_test_app")
	}
	if _, exists := resp.Config["app_secret"]; exists {
		t.Fatalf("config should omit app_secret, got %#v", resp.Config["app_secret"])
	}
	if len(resp.ConfiguredSecrets) != 1 || resp.ConfiguredSecrets[0] != "app_secret" {
		t.Fatalf("configured_secrets = %#v, want [\"app_secret\"]", resp.ConfiguredSecrets)
	}
}

func TestHandleGetChannelConfig_ReturnsNotFoundForUnknownChannel(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/channels/not-a-channel/config", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /api/channels/not-a-channel/config status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
