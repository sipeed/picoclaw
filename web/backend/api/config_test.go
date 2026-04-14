package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func assertGatewayLogLevelApplied(t *testing.T, method, body string, want logger.LogLevel) {
	t.Helper()

	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	initialLevel := logger.GetLevel()
	logger.SetLevel(logger.INFO)
	t.Cleanup(func() {
		logger.SetLevel(initialLevel)
	})

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(method, "/api/config", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s /api/config status = %d, want %d, body=%s", method, rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := logger.GetLevel(); got != want {
		t.Fatalf("logger.GetLevel() = %v, want %v", got, want)
	}
}

func TestHandleUpdateConfig_PreservesExecAllowRemoteDefaultWhenOmitted(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(`{
"version": 3,
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace"
			}
		},
		"model_list": [
			{
				"model_name": "custom-default",
				"model": "openai/gpt-4o",
				"api_keys": ["sk-default"]
			}
		]
	}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if !cfg.Tools.Exec.AllowRemote {
		t.Fatal("tools.exec.allow_remote should remain true when omitted from PUT /api/config")
	}
}

func TestHandleUpdateConfig_DoesNotInheritDefaultModelFields(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(`{
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace"
			}
		},
		"model_list": [
			{
				"model_name": "custom-default",
				"model": "openai/gpt-4o",
				"api_key": "sk-default"
			}
		]
	}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := cfg.ModelList[0].APIBase; got != "" {
		t.Fatalf("model_list[0].api_base = %q, want empty string", got)
	}
}

func TestHandlePatchConfig_RejectsInvalidExecRegexPatterns(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/config", bytes.NewBufferString(`{
		"tools": {
			"exec": {
				"custom_deny_patterns": ["("]
			}
		}
	}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("custom_deny_patterns")) {
		t.Fatalf("expected validation error mentioning custom_deny_patterns, body=%s", rec.Body.String())
	}
}

func TestHandlePatchConfig_AllowsInvalidExecRegexPatternsWhenExecDisabled(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/config", bytes.NewBufferString(`{
		"tools": {
			"exec": {
				"enabled": false,
				"custom_deny_patterns": ["("],
				"custom_allow_patterns": ["("]
			}
		}
	}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// setupPicoEnabledEnv creates a test environment with Pico channel enabled and
// its token stored only in .security.yml (not in the JSON payload).
func setupPicoEnabledEnv(t *testing.T) (string, func()) {
	t.Helper()

	tmp := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldPicoHome := os.Getenv("PICOCLAW_HOME")

	if err := os.Setenv("HOME", tmp); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	if err := os.Setenv("PICOCLAW_HOME", filepath.Join(tmp, ".picoclaw")); err != nil {
		t.Fatalf("set PICOCLAW_HOME: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "custom-default",
		Model:     "openai/gpt-4o",
		APIKeys:   config.SimpleSecureStrings("sk-default"),
	}}
	cfg.Agents.Defaults.ModelName = "custom-default"
	bc := cfg.Channels["pico"]
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	picoCfg := decoded.(*config.PicoSettings)
	bc.Enabled = true
	picoCfg.Token = *config.NewSecureString("test-pico-token")

	configPath := filepath.Join(tmp, "config.json")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig error: %v", err)
	}

	cleanup := func() {
		_ = os.Setenv("HOME", oldHome)
		if oldPicoHome == "" {
			_ = os.Unsetenv("PICOCLAW_HOME")
		} else {
			_ = os.Setenv("PICOCLAW_HOME", oldPicoHome)
		}
	}
	return configPath, cleanup
}

func TestHandleUpdateConfig_SucceedsWhenPicoTokenInSecurityOnly(t *testing.T) {
	configPath, cleanup := setupPicoEnabledEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// PUT request with pico enabled but no token in JSON — token is in .security.yml
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(`{
		"version": 1,
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace",
				"model_name": "custom-default"
			}
		},
		"channels": {
			"pico": {
				"enabled": true,
				"ping_interval": 30,
				"read_timeout": 60,
				"write_timeout": 10,
				"max_connections": 100
			}
		},
		"model_list": [
			{
				"model_name": "custom-default",
				"model": "openai/gpt-4o",
				"api_keys": ["sk-default"]
			}
		]
	}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /api/config status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestHandlePatchConfig_SucceedsWhenPicoTokenInSecurityOnly(t *testing.T) {
	configPath, cleanup := setupPicoEnabledEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// PATCH request changing an unrelated field — pico token still in .security.yml
	req := httptest.NewRequest(http.MethodPatch, "/api/config", bytes.NewBufferString(`{
		"gateway": {
			"log_level": "info"
		}
	}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /api/config status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestHandleUpdateConfig_AppliesGatewayLogLevel(t *testing.T) {
	assertGatewayLogLevelApplied(t, http.MethodPut, `{
		"version": 1,
		"agents": {
			"defaults": {
				"workspace": "~/.picoclaw/workspace",
				"model_name": "custom-default"
			}
		},
		"gateway": {
			"log_level": "error"
		},
		"model_list": [
			{
				"model_name": "custom-default",
				"model": "openai/gpt-4o",
				"api_keys": ["sk-default"]
			}
		]
	}`, logger.ERROR)
}

func TestHandlePatchConfig_AppliesGatewayLogLevel(t *testing.T) {
	assertGatewayLogLevelApplied(t, http.MethodPatch, `{
		"gateway": {
			"log_level": "debug"
		}
	}`, logger.DEBUG)
}

func TestHandlePatchConfig_PreservesDebugFlagOverride(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	initialLevel := logger.GetLevel()
	logger.SetLevel(logger.INFO)
	t.Cleanup(func() {
		logger.SetLevel(initialLevel)
	})

	h := NewHandler(configPath)
	h.SetDebug(true)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/config", bytes.NewBufferString(`{
		"gateway": {
			"log_level": "error"
		}
	}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /api/config status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := logger.GetLevel(); got != logger.DEBUG {
		t.Fatalf("logger.GetLevel() = %v, want %v", got, logger.DEBUG)
	}
}

func TestHandlePatchConfig_SavesDiscordTokenFromPayload(t *testing.T) {
	t.Skip("TODO: fix this test")
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/config", bytes.NewBufferString(`{
		"channel_list": [
			{
				"name":"discord",
				"enabled": true,
				"token": "discord-test-token"
			}
		]
	}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /api/config status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	bc := cfg.Channels[config.ChannelDiscord]
	if !bc.Enabled {
		t.Fatal("discord should be enabled after PATCH")
	}
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	if got := decoded.(*config.DiscordSettings).Token.String(); got != "discord-test-token" {
		t.Fatalf("discord token = %q, want %q", got, "discord-test-token")
	}
}

func TestHandlePatchConfig_DoesNotPersistShadowRegistryAuthTokenField(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/config", bytes.NewBufferString(`{
		"tools": {
			"skills": {
				"registries": {
					"github": {
						"_auth_token": "ghp-shadow-token"
					}
				}
			}
		}
	}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /api/config status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	githubRegistry, ok := cfg.Tools.Skills.Registries.Get("github")
	if !ok {
		t.Fatal("github registry missing after PATCH")
	}
	if got := githubRegistry.AuthToken.String(); got != "ghp-shadow-token" {
		t.Fatalf("github registry auth token = %q, want %q", got, "ghp-shadow-token")
	}
	if got := githubRegistry.BaseURL; got != "https://github.com" {
		t.Fatalf("github registry base_url = %q, want %q", got, "https://github.com")
	}

	rawConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(configPath) error = %v", err)
	}
	if strings.Contains(string(rawConfig), "_auth_token") {
		t.Fatalf("config.json should not persist _auth_token shadow field, got:\n%s", string(rawConfig))
	}
}

func TestHandlePatchConfig_AllowsInvalidDenyRegexPatternsWhenDenyPatternsDisabled(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/config", bytes.NewBufferString(`{
		"tools": {
			"exec": {
				"enabled": true,
				"enable_deny_patterns": false,
				"custom_deny_patterns": ["("]
			}
		}
	}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

// testCommandPatterns is a helper that sets up a handler and sends a test-command-patterns request.
func testCommandPatterns(t *testing.T, configPath string, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/config/test-command-patterns", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandleTestCommandPatterns_MatchesWhitelist(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	rec := testCommandPatterns(t, configPath, `{
		"allow_patterns": ["^echo\\s+hello"],
		"deny_patterns": ["^rm\\s+-rf"],
		"command": "echo hello world"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"allowed":true`)) {
		t.Fatalf("expected allowed=true, body=%s", rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(`"blocked":true`)) {
		t.Fatalf("expected blocked=false when whitelist matches, body=%s", rec.Body.String())
	}
}

func TestHandleTestCommandPatterns_MatchesBlacklistNotWhitelist(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	rec := testCommandPatterns(t, configPath, `{
		"allow_patterns": ["^echo\\s+hello"],
		"deny_patterns": ["^rm\\s+-rf"],
		"command": "rm -rf /tmp"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"blocked":true`)) {
		t.Fatalf("expected blocked=true, body=%s", rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(`"allowed":true`)) {
		t.Fatalf("expected allowed=false when blacklist matches but not whitelist, body=%s", rec.Body.String())
	}
}

func TestHandleTestCommandPatterns_MatchesNeither(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	rec := testCommandPatterns(t, configPath, `{
		"allow_patterns": ["^echo\\s+hello"],
		"deny_patterns": ["^rm\\s+-rf"],
		"command": "ls -la"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(`"allowed":true`)) {
		t.Fatalf("expected allowed=false, body=%s", rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(`"blocked":true`)) {
		t.Fatalf("expected blocked=false, body=%s", rec.Body.String())
	}
}

func TestHandleTestCommandPatterns_CaseInsensitiveWithGoFlag(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	rec := testCommandPatterns(t, configPath, `{
		"allow_patterns": ["(?i)^ECHO"],
		"deny_patterns": [],
		"command": "echo hello"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"allowed":true`)) {
		t.Fatalf("expected allowed=true with Go (?i) flag, body=%s", rec.Body.String())
	}
}

func TestHandleTestCommandPatterns_EmptyPatterns(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	rec := testCommandPatterns(t, configPath, `{
		"allow_patterns": [],
		"deny_patterns": [],
		"command": "rm -rf /tmp"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(`"allowed":true`)) {
		t.Fatalf("expected allowed=false with empty patterns, body=%s", rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(`"blocked":true`)) {
		t.Fatalf("expected blocked=false with empty patterns, body=%s", rec.Body.String())
	}
}

func TestHandleTestCommandPatterns_InvalidRegexSkipped(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	rec := testCommandPatterns(t, configPath, `{
		"allow_patterns": ["([[", "^echo"],
		"deny_patterns": [],
		"command": "echo hello"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"allowed":true`)) {
		t.Fatalf("expected allowed=true, invalid pattern skipped and valid one matched, body=%s", rec.Body.String())
	}
}

func TestHandleTestCommandPatterns_ReturnsMatchedPattern(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	rec := testCommandPatterns(t, configPath, `{
		"allow_patterns": [],
		"deny_patterns": ["\\$(?i)[a-zA-Z_]*(SECRET|KEY|PASSWORD|TOKEN|AUTH)[a-zA-Z0-9_]*"],
		"command": "echo $GITHUB_API_KEY"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"blocked":true`)) {
		t.Fatalf("expected blocked=true, body=%s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`matched_blacklist`)) {
		t.Fatalf("expected matched_blacklist field, body=%s", rec.Body.String())
	}
}

func TestHandleTestCommandPatterns_InvalidJSON(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/config/test-command-patterns",
		bytes.NewBufferString(`{invalid json}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestApplyConfigSecretsFromMap_TelegramToken(t *testing.T) {
	cfg := config.DefaultConfig()
	bc := cfg.Channels["telegram"]
	bc.Enabled = true
	// Pre-decode so extend is populated
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	tgCfg := decoded.(*config.TelegramSettings)
	tgCfg.Token = *config.NewSecureString("original-token")

	raw := map[string]any{
		"channel_list": map[string]any{
			"telegram": map[string]any{
				"enabled": true,
				"token":   "secret-from-api",
			},
		},
	}

	applyConfigSecretsFromMap(cfg, raw)

	if got := tgCfg.Token.String(); got != "secret-from-api" {
		t.Fatalf("telegram token = %q, want %q", got, "secret-from-api")
	}
}

func TestApplyConfigSecretsFromMap_TeamsWebhook(t *testing.T) {
	// applyConfigSecretsFromMap recurses into nested maps to find
	// SecureString fields at any depth (e.g. webhook_url inside webhooks map).
	cfg := config.DefaultConfig()
	bc := &config.Channel{Enabled: true, Type: config.ChannelTeamsWebHook}
	cfg.Channels["teams_webhook"] = bc
	target := &config.TeamsWebhookSettings{
		Webhooks: map[string]config.TeamsWebhookTarget{
			"default": {
				WebhookURL: *config.NewSecureString("https://example.com/hook1"),
				Title:      "Default",
			},
		},
	}
	if err := bc.Decode(target); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	raw := map[string]any{
		"channel_list": map[string]any{
			"teams_webhook": map[string]any{
				"enabled": true,
				"settings": map[string]any{
					"webhooks": map[string]any{
						"default": map[string]any{
							"webhook_url": "https://example.com/hook-updated",
							"title":       "Default Updated",
						},
					},
				},
			},
		},
	}

	applyConfigSecretsFromMap(cfg, raw)

	// Verify the decoded struct has the updated SecureString value
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	twCfg, ok := decoded.(*config.TeamsWebhookSettings)
	if !ok {
		t.Fatalf("expected *TeamsWebhookSettings, got %T", decoded)
	}

	hookURL := twCfg.Webhooks["default"].WebhookURL
	if got := hookURL.String(); got != "https://example.com/hook-updated" {
		t.Fatalf("webhook_url = %q, want %q", got, "https://example.com/hook-updated")
	}
	// Note: title is a plain string, not a SecureString, so it is NOT updated
	// by applyConfigSecretsFromMap (only secure fields are handled).
}

func TestApplyConfigSecretsFromMap_MultipleChannels(t *testing.T) {
	cfg := config.DefaultConfig()

	// Setup telegram
	bc := cfg.Channels["telegram"]
	bc.Enabled = true
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() telegram error = %v", err)
	}
	tgCfg := decoded.(*config.TelegramSettings)
	tgCfg.Token = *config.NewSecureString("old-telegram-token")

	// Setup discord
	bc = cfg.Channels["discord"]
	bc.Enabled = true
	decoded, err = bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() discord error = %v", err)
	}
	discCfg := decoded.(*config.DiscordSettings)
	discCfg.Token = *config.NewSecureString("old-discord-token")

	raw := map[string]any{
		"channel_list": map[string]any{
			"telegram": map[string]any{
				"enabled": true,
				"settings": map[string]any{
					"token": "new-telegram-token",
				},
			},
			"discord": map[string]any{
				"enabled": true,
				"settings": map[string]any{
					"token": "new-discord-token",
				},
			},
		},
	}

	applyConfigSecretsFromMap(cfg, raw)

	if got := tgCfg.Token.String(); got != "new-telegram-token" {
		t.Fatalf("telegram token = %q, want %q", got, "new-telegram-token")
	}
	if got := discCfg.Token.String(); got != "new-discord-token" {
		t.Fatalf("discord token = %q, want %q", got, "new-discord-token")
	}
}

func TestApplyConfigSecretsFromMap_SkipsNonStringValues(t *testing.T) {
	cfg := config.DefaultConfig()
	bc := cfg.Channels["telegram"]
	bc.Enabled = true
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	tgCfg := decoded.(*config.TelegramSettings)
	tgCfg.Token = *config.NewSecureString("original-token")

	raw := map[string]any{
		"channel_list": map[string]any{
			"telegram": map[string]any{
				"enabled": true,
				"token":   12345, // not a string, should be skipped
			},
		},
	}

	applyConfigSecretsFromMap(cfg, raw)

	if got := tgCfg.Token.String(); got != "original-token" {
		t.Fatalf("telegram token = %q, want %q", got, "original-token")
	}
}

func TestApplyConfigSecretsFromMap_ChannelNotDecodedYet(t *testing.T) {
	cfg := config.DefaultConfig()
	bc := cfg.Channels["telegram"]
	bc.Enabled = true
	// Don't decode — let the function handle lazy decoding
	bc.Type = config.ChannelTelegram

	raw := map[string]any{
		"channel_list": map[string]any{
			"telegram": map[string]any{
				"enabled": true,
				"token":   "lazy-decoded-token",
			},
		},
	}

	applyConfigSecretsFromMap(cfg, raw)

	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	tgCfg := decoded.(*config.TelegramSettings)
	if got := tgCfg.Token.String(); got != "lazy-decoded-token" {
		t.Fatalf("telegram token = %q, want %q", got, "lazy-decoded-token")
	}
}
