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
	bc := cfg.Channels[config.ChannelFeishu]
	bc.Enabled = true
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	bcfg := decoded.(*config.FeishuSettings)
	bcfg.AppID = "cli_test_app"
	bcfg.AppSecret = *config.NewSecureString("feishu-secret-from-security")
	bc.AllowFrom = config.FlexibleStringSlice{"ou_test_user"}
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
	if got := resp.Config["enabled"]; got != true {
		t.Fatalf("config.enabled = %#v, want true", got)
	}
	allowFrom, ok := resp.Config["allow_from"].([]any)
	if !ok || len(allowFrom) != 1 || allowFrom[0] != "ou_test_user" {
		t.Fatalf("config.allow_from = %#v, want [\"ou_test_user\"]", resp.Config["allow_from"])
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

func TestHandleGetChannelConfig_ReturnsCommonFieldsWhenSettingsEmpty(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	bc := cfg.Channels[config.ChannelFeishu]
	bc.Enabled = true
	bc.AllowFrom = config.FlexibleStringSlice{"ou_common_user"}
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

	var resp struct {
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got := resp.Config["enabled"]; got != true {
		t.Fatalf("config.enabled = %#v, want true", got)
	}
	allowFrom, ok := resp.Config["allow_from"].([]any)
	if !ok || len(allowFrom) != 1 || allowFrom[0] != "ou_common_user" {
		t.Fatalf("config.allow_from = %#v, want [\"ou_common_user\"]", resp.Config["allow_from"])
	}
}

func TestHandleGetChannelConfig_ReturnsDefaultShapeForMissingChannel(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	delete(cfg.Channels, config.ChannelIRC)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/channels/irc/config", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"GET /api/channels/irc/config status = %d, want %d, body=%s",
			rec.Code,
			http.StatusOK,
			rec.Body.String(),
		)
	}

	var resp struct {
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got := resp.Config["server"]; got != "" {
		t.Fatalf("config.server = %#v, want empty string", got)
	}
	if got := resp.Config["nick"]; got != "picoclaw" {
		t.Fatalf("config.nick = %#v, want %q", got, "picoclaw")
	}
	if got := resp.Config["enabled"]; got != false {
		t.Fatalf("config.enabled = %#v, want false", got)
	}
}

func TestHandleListChannelCatalog_IncludesExistingWeixinInstances(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.Channels["weixin_work"] = &config.Channel{
		Enabled: true,
		Type:    config.ChannelWeixin,
	}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/channels/catalog", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/channels/catalog status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Channels []struct {
			Name             string `json:"name"`
			Type             string `json:"type"`
			ConfigKey        string `json:"config_key"`
			SupportsMultiple bool   `json:"supports_multiple"`
		} `json:"channels"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	foundStatic := false
	foundDynamic := false
	for _, item := range resp.Channels {
		if item.Name == "weixin" {
			foundStatic = item.Type == config.ChannelWeixin && item.SupportsMultiple
		}
		if item.Name == "weixin_work" {
			foundDynamic = item.Type == config.ChannelWeixin &&
				item.ConfigKey == "weixin_work" &&
				item.SupportsMultiple
		}
	}
	if !foundStatic {
		t.Fatal("expected static weixin catalog entry")
	}
	if !foundDynamic {
		t.Fatal("expected dynamic weixin instance in catalog")
	}
}

func TestHandleGetChannelConfig_ReturnsDynamicWeixinInstance(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	bc := &config.Channel{Enabled: true, Type: config.ChannelWeixin}
	wxCfg := &config.WeixinSettings{
		AccountID: "work-account",
		BaseURL:   "https://ilinkai.weixin.qq.com/",
	}
	wxCfg.SetToken("secret-token")
	if err := bc.Decode(wxCfg); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	cfg.Channels["weixin_work"] = bc
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/channels/weixin_work/config", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/channels/weixin_work/config status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp struct {
		Config            map[string]any `json:"config"`
		ConfiguredSecrets []string       `json:"configured_secrets"`
		ConfigKey         string         `json:"config_key"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got := resp.ConfigKey; got != "weixin_work" {
		t.Fatalf("config_key = %q, want weixin_work", got)
	}
	if got := resp.Config["account_id"]; got != "work-account" {
		t.Fatalf("config.account_id = %#v, want work-account", got)
	}
	if _, exists := resp.Config["token"]; exists {
		t.Fatalf("config should omit token, got %#v", resp.Config["token"])
	}
	if len(resp.ConfiguredSecrets) != 1 || resp.ConfiguredSecrets[0] != "token" {
		t.Fatalf("configured_secrets = %#v, want [token]", resp.ConfiguredSecrets)
	}
}
