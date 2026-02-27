package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KarakuriAgent/clawdroid/pkg/config"
)

// --- Server lifecycle tests ---

func TestNewServer_FieldsInitialized(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgPath := "/tmp/config.json"
	var restarted bool

	s := NewServer(cfg, cfgPath, func() {
		restarted = true
	})

	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if s.cfg != cfg {
		t.Error("server should keep config pointer")
	}
	if s.configPath != cfgPath {
		t.Errorf("configPath = %q, want %q", s.configPath, cfgPath)
	}
	if s.onRestart == nil {
		t.Fatal("onRestart should be set")
	}

	s.onRestart()
	if !restarted {
		t.Error("onRestart callback should be callable")
	}
}

func TestServerStop_NoServer_NoError(t *testing.T) {
	s := &Server{}
	if err := s.Stop(context.Background()); err != nil {
		t.Errorf("Stop() without server should not error, got %v", err)
	}
}

func TestServerStart_RegistersRoutesAndAuth(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = -1 // invalid port prevents real listen but Start still builds handler
	cfg.Gateway.APIKey = "test-key"

	s := NewServer(cfg, "/tmp/config.json", nil)
	if err := s.Start(); err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
	}
	if s.server == nil {
		t.Fatal("server should be initialized after Start")
	}

	// Missing auth header should be rejected by middleware.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s.server.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/config without token status = %d, want 401", rr.Code)
	}

	// Correct token should allow handler execution.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/config/schema", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	s.server.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("GET /api/config/schema with token status = %d, want 200", rr.Code)
	}

	// Method pattern registration should reject unsupported methods.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/config", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	s.server.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/config status = %d, want 405", rr.Code)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := s.Stop(ctx); err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}

// --- Schema tests ---

func TestBuildSchema_SectionCount(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())
	// Config has exported fields: LLM, Agents, Channels, Gateway, Tools, Heartbeat, RateLimits
	if len(schema.Sections) < 7 {
		t.Errorf("expected at least 7 sections, got %d", len(schema.Sections))
	}
}

func TestBuildSchema_FieldKeys(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	// Find LLM section
	var llm *SchemaSection
	for i := range schema.Sections {
		if schema.Sections[i].Key == "llm" {
			llm = &schema.Sections[i]
			break
		}
	}
	if llm == nil {
		t.Fatal("LLM section not found")
	}

	keys := map[string]bool{}
	for _, f := range llm.Fields {
		keys[f.Key] = true
	}

	for _, k := range []string{"model", "api_key", "base_url"} {
		if !keys[k] {
			t.Errorf("expected LLM field %q", k)
		}
	}
}

func TestBuildSchema_Types(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	fieldMap := allFields(schema)

	tests := []struct {
		key      string
		wantType string
	}{
		{"model", "string"},
		{"api_key", "string"},
		{"enabled", "bool"},
		{"max_tokens", "int"},
		{"temperature", "float"},
		{"allow_from", "[]string"},
	}

	for _, tc := range tests {
		f, ok := fieldMap[tc.key]
		if !ok {
			// Try with prefix
			continue
		}
		if f.Type != tc.wantType {
			t.Errorf("field %q: want type %q, got %q", tc.key, tc.wantType, f.Type)
		}
	}
}

func TestBuildSchema_SecretFlag(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())
	fieldMap := allFields(schema)

	secrets := []string{"api_key", "token", "bot_token", "app_token", "channel_secret", "channel_access_token"}
	for _, k := range secrets {
		f, ok := fieldMap[k]
		if !ok {
			continue
		}
		if !f.Secret {
			t.Errorf("field %q should be marked as secret", k)
		}
	}

	nonSecrets := []string{"model", "base_url", "enabled", "host", "port"}
	for _, k := range nonSecrets {
		f, ok := fieldMap[k]
		if !ok {
			continue
		}
		if f.Secret {
			t.Errorf("field %q should NOT be marked as secret", k)
		}
	}
}

func TestBuildSchema_Labels(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	wantLabels := map[string]string{
		"api_key":              "API Key",
		"base_url":             "Base URL",
		"defaults.max_tokens":  "Max Tokens",
		"slack.bot_token":      "Bot Token",
	}

	for _, sec := range schema.Sections {
		for _, f := range sec.Fields {
			if want, ok := wantLabels[f.Key]; ok {
				if f.Label != want {
					t.Errorf("field %q label = %q, want %q", f.Key, f.Label, want)
				}
			}
		}
	}
}

func TestBuildSchema_MapType(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	// Tools section should have an mcp field of type "map"
	var toolsSection *SchemaSection
	for i := range schema.Sections {
		if schema.Sections[i].Key == "tools" {
			toolsSection = &schema.Sections[i]
			break
		}
	}
	if toolsSection == nil {
		t.Fatal("tools section not found")
	}

	found := false
	for _, f := range toolsSection.Fields {
		if f.Key == "mcp" {
			found = true
			if f.Type != "map" {
				t.Errorf("mcp field type: want %q, got %q", "map", f.Type)
			}
		}
	}
	if !found {
		t.Error("mcp field not found in tools section")
	}
}

// --- Schema: unexported fields and omitempty ---

func TestBuildSchema_UnexportedFieldsSkipped(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	// Config has an unexported field `mu sync.RWMutex` — it must not appear
	for _, sec := range schema.Sections {
		if sec.Key == "mu" {
			t.Error("unexported field 'mu' should not appear as a section")
		}
		for _, f := range sec.Fields {
			if f.Key == "mu" || strings.HasSuffix(f.Key, ".mu") {
				t.Errorf("unexported field 'mu' should not appear as a field, got key %q", f.Key)
			}
		}
	}
}

func TestBuildSchema_OmitemptyTagHandled(t *testing.T) {
	// MCPServerConfig has json:"command,omitempty" — jsonKey should return "command"
	got := jsonKey(reflect.TypeOf(config.MCPServerConfig{}).Field(0)) // Command field
	if got != "command" {
		t.Errorf("jsonKey for omitempty field = %q, want %q", got, "command")
	}
}

// --- Schema: nested keys and defaults ---

func TestBuildSchema_NestedDotKeys(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	// Find channels section
	var channels *SchemaSection
	for i := range schema.Sections {
		if schema.Sections[i].Key == "channels" {
			channels = &schema.Sections[i]
			break
		}
	}
	if channels == nil {
		t.Fatal("channels section not found")
	}

	// Expect dot-separated keys like "telegram.enabled", "telegram.token"
	wantKeys := []string{
		"telegram.enabled",
		"telegram.token",
		"slack.bot_token",
		"line.channel_secret",
		"websocket.host",
		"websocket.port",
	}

	keySet := map[string]bool{}
	for _, f := range channels.Fields {
		keySet[f.Key] = true
	}

	for _, k := range wantKeys {
		if !keySet[k] {
			t.Errorf("expected channels field %q, available keys: %v", k, keys(keySet))
		}
	}
}

func TestBuildSchema_DefaultValues(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	// Collect all fields with full keys (section.field)
	type fullField struct {
		section string
		field   SchemaField
	}
	var all []fullField
	for _, sec := range schema.Sections {
		for _, f := range sec.Fields {
			all = append(all, fullField{sec.Key, f})
		}
	}

	findField := func(section, key string) *SchemaField {
		for _, ff := range all {
			if ff.section == section && ff.field.Key == key {
				return &ff.field
			}
		}
		return nil
	}

	tests := []struct {
		section string
		key     string
		want    interface{}
	}{
		{"agents", "defaults.max_tokens", float64(8192)},        // JSON numbers → float64
		{"agents", "defaults.context_window", float64(128000)},
		{"agents", "defaults.restrict_to_workspace", true},
		{"gateway", "host", "127.0.0.1"},
		{"gateway", "port", float64(18790)},
		{"heartbeat", "enabled", true},
		{"heartbeat", "interval", float64(30)},
	}

	for _, tc := range tests {
		f := findField(tc.section, tc.key)
		if f == nil {
			t.Errorf("field %s.%s not found", tc.section, tc.key)
			continue
		}
		// Compare via JSON round-trip since reflect values may differ
		gotJSON, _ := json.Marshal(f.Default)
		wantJSON, _ := json.Marshal(tc.want)
		if string(gotJSON) != string(wantJSON) {
			t.Errorf("field %s.%s default = %s, want %s", tc.section, tc.key, gotJSON, wantJSON)
		}
	}
}

// --- GET /api/config/schema handler test ---

func TestHandleGetSchema_Response(t *testing.T) {
	cfg := config.DefaultConfig()
	s := newTestServer(cfg)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config/schema", nil)
	s.handleGetSchema(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var schema SchemaResponse
	if err := json.NewDecoder(rr.Body).Decode(&schema); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(schema.Sections) < 7 {
		t.Errorf("expected at least 7 sections, got %d", len(schema.Sections))
	}

	// Verify each section has at least one field
	for _, sec := range schema.Sections {
		if sec.Key == "" {
			t.Error("section key should not be empty")
		}
		if sec.Label == "" {
			t.Errorf("section %q label should not be empty", sec.Key)
		}
	}
}

// --- GET /api/config tests ---

func TestHandleGetConfig_SecretsReturnedAsIs(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.APIKey = "secret-key-123"
	cfg.Channels.Telegram.Token = "telegram-token-456"
	cfg.Channels.Slack.BotToken = "slack-bot-token"
	cfg.Channels.Slack.AppToken = "slack-app-token"
	cfg.Channels.LINE.ChannelSecret = "line-secret"
	cfg.Channels.LINE.ChannelAccessToken = "line-access-token"

	s := newTestServer(cfg)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s.handleGetConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)

	// Secrets are returned as-is (no masking)
	llm := result["llm"].(map[string]interface{})
	if llm["api_key"] != "secret-key-123" {
		t.Errorf("LLM api_key = %v, want %q", llm["api_key"], "secret-key-123")
	}

	telegram := result["channels"].(map[string]interface{})["telegram"].(map[string]interface{})
	if telegram["token"] != "telegram-token-456" {
		t.Errorf("Telegram token = %v, want %q", telegram["token"], "telegram-token-456")
	}

	slack := result["channels"].(map[string]interface{})["slack"].(map[string]interface{})
	if slack["bot_token"] != "slack-bot-token" {
		t.Errorf("Slack bot_token = %v, want %q", slack["bot_token"], "slack-bot-token")
	}

	line := result["channels"].(map[string]interface{})["line"].(map[string]interface{})
	if line["channel_secret"] != "line-secret" {
		t.Errorf("LINE channel_secret = %v, want %q", line["channel_secret"], "line-secret")
	}
}

func TestHandleGetConfig_NonSecretValues(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.Model = "test-model"
	cfg.LLM.BaseURL = "https://example.com"
	cfg.Gateway.Host = "0.0.0.0"
	cfg.Gateway.Port = 9999

	s := newTestServer(cfg)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s.handleGetConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)

	llm := result["llm"].(map[string]interface{})
	if llm["model"] != "test-model" {
		t.Errorf("model = %v, want %q", llm["model"], "test-model")
	}
	if llm["base_url"] != "https://example.com" {
		t.Errorf("base_url = %v, want %q", llm["base_url"], "https://example.com")
	}

	gw := result["gateway"].(map[string]interface{})
	if gw["host"] != "0.0.0.0" {
		t.Errorf("gateway host = %v, want %q", gw["host"], "0.0.0.0")
	}
	if gw["port"] != float64(9999) {
		t.Errorf("gateway port = %v, want %v", gw["port"], 9999)
	}
}

func TestHandleGetConfig_GatewayAPIKeyReturnedAsIs(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.APIKey = "my-gateway-secret"

	s := newTestServer(cfg)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s.handleGetConfig(rr, req)

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)

	gw := result["gateway"].(map[string]interface{})
	if gw["api_key"] != "my-gateway-secret" {
		t.Errorf("gateway api_key = %v, want %q", gw["api_key"], "my-gateway-secret")
	}
}

func TestHandleGetConfig_EmptySecretsReturnedEmpty(t *testing.T) {
	cfg := config.DefaultConfig()
	// All secrets are empty by default

	s := newTestServer(cfg)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s.handleGetConfig(rr, req)

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)

	llm := result["llm"].(map[string]interface{})
	if llm["api_key"] != "" {
		t.Errorf("Empty api_key should be returned as empty string, got %v", llm["api_key"])
	}
}

func TestHandleGetConfig_MarshalError_500(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Temperature = math.NaN() // json.Marshal should fail

	s := newTestServer(cfg)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s.handleGetConfig(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if _, ok := result["error"]; !ok {
		t.Error("expected error field in response")
	}
}

// --- PUT /api/config tests ---

func TestHandlePutConfig_UpdateNonSecret(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.Model = "old-model"

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{"llm":{"model":"new-model"}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	// Verify saved config
	saved, _ := config.LoadConfig(cfgPath)
	if saved.LLM.Model != "new-model" {
		t.Errorf("model = %q, want %q", saved.LLM.Model, "new-model")
	}
}

func TestHandlePutConfig_SecretPreservedViaPartialUpdate(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.APIKey = "real-secret"

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	// Only send model — api_key should be preserved via partial update
	body := `{"llm":{"model":"updated"}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, _ := config.LoadConfig(cfgPath)
	if saved.LLM.APIKey != "real-secret" {
		t.Errorf("api_key = %q, want %q", saved.LLM.APIKey, "real-secret")
	}
	if saved.LLM.Model != "updated" {
		t.Errorf("model = %q, want %q", saved.LLM.Model, "updated")
	}
}

func TestHandlePutConfig_SecretUpdatedWithNewValue(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.APIKey = "old-secret"

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	// Send a new secret value — should save the new value
	body := `{"llm":{"api_key":"brand-new-secret"}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, _ := config.LoadConfig(cfgPath)
	if saved.LLM.APIKey != "brand-new-secret" {
		t.Errorf("api_key = %q, want %q", saved.LLM.APIKey, "brand-new-secret")
	}
}

func TestHandlePutConfig_NilOnRestart_NoPanic(t *testing.T) {
	cfg := config.DefaultConfig()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	// onRestart is nil
	s := &Server{cfg: cfg, configPath: cfgPath, onRestart: nil}
	body := `{"llm":{"model":"test"}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))

	// Should not panic
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

func TestHandlePutConfig_InvalidJSON_400(t *testing.T) {
	cfg := config.DefaultConfig()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{invalid json`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if _, ok := result["error"]; !ok {
		t.Error("expected error field in response")
	}
}

func TestHandlePutConfig_InvalidFieldType_400(t *testing.T) {
	cfg := config.DefaultConfig()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	// gateway.port expects int, so string should fail during unmarshal to Config
	body := `{"gateway":{"port":"not-an-int"}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	errMsg, ok := result["error"].(string)
	if !ok || errMsg == "" {
		t.Fatal("expected non-empty error message")
	}
	if !strings.Contains(errMsg, "invalid config") {
		t.Errorf("error = %q, want message containing %q", errMsg, "invalid config")
	}
}

func TestHandlePutConfig_SaveError_500(t *testing.T) {
	cfg := config.DefaultConfig()

	// Using a directory as file path makes SaveConfig fail on WriteFile.
	cfgPath := t.TempDir()
	s := &Server{cfg: cfg, configPath: cfgPath}

	body := `{"llm":{"model":"new-model"}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	errMsg, ok := result["error"].(string)
	if !ok || errMsg == "" {
		t.Fatal("expected non-empty error message")
	}
	if !strings.Contains(errMsg, "failed to save config") {
		t.Errorf("error = %q, want message containing %q", errMsg, "failed to save config")
	}
}

func TestHandlePutConfig_PartialUpdatePreservesOtherSections(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.Model = "original-model"
	cfg.Gateway.Host = "10.0.0.1"
	cfg.Gateway.Port = 12345
	cfg.Channels.Telegram.Enabled = true
	cfg.Heartbeat.Interval = 60

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	// Only send LLM update — everything else should be preserved
	body := `{"llm":{"model":"new-model"}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if saved.LLM.Model != "new-model" {
		t.Errorf("LLM model = %q, want %q", saved.LLM.Model, "new-model")
	}
	if saved.Gateway.Host != "10.0.0.1" {
		t.Errorf("Gateway host = %q, want %q (should be preserved)", saved.Gateway.Host, "10.0.0.1")
	}
	if saved.Gateway.Port != 12345 {
		t.Errorf("Gateway port = %d, want %d (should be preserved)", saved.Gateway.Port, 12345)
	}
	if !saved.Channels.Telegram.Enabled {
		t.Error("Telegram enabled should be preserved as true")
	}
	if saved.Heartbeat.Interval != 60 {
		t.Errorf("Heartbeat interval = %d, want %d (should be preserved)", saved.Heartbeat.Interval, 60)
	}
}

func TestHandlePutConfig_ResponseBody(t *testing.T) {
	cfg := config.DefaultConfig()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{"llm":{"model":"test"}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)

	if result["status"] != "ok" {
		t.Errorf("status = %v, want %q", result["status"], "ok")
	}
	if result["restart"] != true {
		t.Errorf("restart = %v, want true", result["restart"])
	}
}

func TestHandlePutConfig_OnRestartCalled(t *testing.T) {
	cfg := config.DefaultConfig()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	var mu sync.Mutex
	restarted := false
	s := &Server{
		cfg:        cfg,
		configPath: cfgPath,
		onRestart: func() {
			mu.Lock()
			restarted = true
			mu.Unlock()
		},
	}

	body := `{"llm":{"model":"test"}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	// Wait for async restart callback
	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	if !restarted {
		t.Error("onRestart should have been called")
	}
	mu.Unlock()
}

// --- Auth middleware tests ---

func TestAuthMiddleware_NoKey_Skips(t *testing.T) {
	cfg := config.DefaultConfig()
	// api_key is empty by default — auth should be skipped
	s := newTestServer(cfg)

	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (auth skipped)", rr.Code)
	}
}

func TestAuthMiddleware_MissingHeader_401(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.APIKey = "test-key"
	s := newTestServer(cfg)

	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestAuthMiddleware_WrongToken_403(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.APIKey = "test-key"
	s := newTestServer(cfg)

	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rr.Code)
	}
}

func TestAuthMiddleware_BasicScheme_401(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.APIKey = "test-key"
	s := newTestServer(cfg)

	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for Basic scheme", rr.Code)
	}
}

func TestAuthMiddleware_ErrorResponseJSON(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.APIKey = "test-key"
	s := newTestServer(cfg)

	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test 401 response
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler(rr, req)

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("401 Content-Type = %q, want %q", ct, "application/json")
	}
	var body401 map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&body401)
	if _, ok := body401["error"]; !ok {
		t.Error("401 response should have error field")
	}

	// Test 403 response
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	handler(rr, req)

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("403 Content-Type = %q, want %q", ct, "application/json")
	}
	var body403 map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&body403)
	if _, ok := body403["error"]; !ok {
		t.Error("403 response should have error field")
	}
}

func TestAuthMiddleware_CorrectToken_200(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.APIKey = "test-key"
	s := newTestServer(cfg)

	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestSecretKeys(t *testing.T) {
	wantSecret := []string{"api_key", "token", "bot_token", "app_token", "channel_secret", "channel_access_token"}
	for _, k := range wantSecret {
		if !secretKeys[k] {
			t.Errorf("secretKeys[%q] = false, want true", k)
		}
	}

	notSecret := []string{"model", "host", "port", "enabled"}
	for _, k := range notSecret {
		if secretKeys[k] {
			t.Errorf("secretKeys[%q] = true, want false", k)
		}
	}
}

// --- PUT /api/config: nested secret preservation via partial update ---

func TestHandlePutConfig_NestedSecretPreservedViaPartialUpdate(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Token = "real-tg-token"
	cfg.Channels.Telegram.Enabled = false
	cfg.Channels.Slack.BotToken = "real-slack-bot"
	cfg.Channels.LINE.ChannelSecret = "real-line-secret"

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	// Only send non-secret fields — secrets should be preserved via partial update
	body := `{"channels":{"telegram":{"enabled":true}}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if saved.Channels.Telegram.Token != "real-tg-token" {
		t.Errorf("telegram token = %q, want %q", saved.Channels.Telegram.Token, "real-tg-token")
	}
	if saved.Channels.Slack.BotToken != "real-slack-bot" {
		t.Errorf("slack bot_token = %q, want %q", saved.Channels.Slack.BotToken, "real-slack-bot")
	}
	if saved.Channels.LINE.ChannelSecret != "real-line-secret" {
		t.Errorf("line channel_secret = %q, want %q", saved.Channels.LINE.ChannelSecret, "real-line-secret")
	}
}

// --- PUT /api/config: empty body ---

func TestHandlePutConfig_EmptyBody(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.Model = "keep-this-model"
	cfg.Gateway.Port = 12345

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	// All existing values should be preserved with empty update
	if saved.LLM.Model != "keep-this-model" {
		t.Errorf("model = %q, want %q", saved.LLM.Model, "keep-this-model")
	}
	if saved.Gateway.Port != 12345 {
		t.Errorf("gateway port = %d, want %d", saved.Gateway.Port, 12345)
	}
}

// --- GET /api/config: Content-Type verification ---

func TestHandleGetConfig_ContentType(t *testing.T) {
	cfg := config.DefaultConfig()
	s := newTestServer(cfg)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s.handleGetConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

// --- PUT /api/config: Content-Type verification ---

func TestHandlePutConfig_ContentType(t *testing.T) {
	cfg := config.DefaultConfig()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{"llm":{"model":"test"}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

// --- Schema: goTypeToSchema edge cases ---

func TestBuildSchema_BoolDefaultValues(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	// Find heartbeat section
	var heartbeat *SchemaSection
	for i := range schema.Sections {
		if schema.Sections[i].Key == "heartbeat" {
			heartbeat = &schema.Sections[i]
			break
		}
	}
	if heartbeat == nil {
		t.Fatal("heartbeat section not found")
	}

	for _, f := range heartbeat.Fields {
		if f.Key == "enabled" {
			if f.Default != true {
				t.Errorf("heartbeat enabled default = %v, want true", f.Default)
			}
			if f.Type != "bool" {
				t.Errorf("heartbeat enabled type = %q, want %q", f.Type, "bool")
			}
			return
		}
	}
	t.Error("heartbeat enabled field not found")
}

// --- Auth middleware: Bearer with extra whitespace ---

func TestAuthMiddleware_BearerExtraSpace_401(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.APIKey = "test-key"
	s := newTestServer(cfg)

	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// "Bearer  test-key" (double space) — prefix check passes but token won't match
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer  test-key")
	handler(rr, req)

	if rr.Code == http.StatusOK {
		t.Error("double-space Bearer should not authenticate successfully")
	}
}

// --- Auth middleware: empty Bearer token ---

func TestAuthMiddleware_EmptyBearerToken_403(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.APIKey = "test-key"
	s := newTestServer(cfg)

	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 for empty bearer token", rr.Code)
	}
}

// --- Schema: section labels ---

func TestBuildSchema_SectionLabels(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	wantLabels := map[string]string{
		"llm":         "LLM",
		"agents":      "Agent Defaults",
		"channels":    "Messaging Channels",
		"gateway":     "Gateway",
		"tools":       "Tool Settings",
		"heartbeat":   "Heartbeat",
		"rate_limits": "Rate Limits",
	}

	for _, sec := range schema.Sections {
		if want, ok := wantLabels[sec.Key]; ok {
			if sec.Label != want {
				t.Errorf("section %q label = %q, want %q", sec.Key, sec.Label, want)
			}
		}
	}
}

// --- PUT /api/config via routing with auth ---

func TestServerRouting_PutConfig_AuthRequired(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = -1
	cfg.Gateway.APIKey = "route-test-key"

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := NewServer(cfg, cfgPath, nil)
	if err := s.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer s.Stop(context.Background())

	// PUT without auth should be rejected
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(`{"llm":{"model":"x"}}`))
	s.server.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("PUT /api/config without token status = %d, want 401", rr.Code)
	}

	// PUT with wrong token should be rejected
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(`{"llm":{"model":"x"}}`))
	req.Header.Set("Authorization", "Bearer wrong-key")
	s.server.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("PUT /api/config with wrong token status = %d, want 403", rr.Code)
	}

	// PUT with correct token should succeed
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(`{"llm":{"model":"x"}}`))
	req.Header.Set("Authorization", "Bearer route-test-key")
	s.server.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("PUT /api/config with correct token status = %d, want 200", rr.Code)
	}
}

// --- PUT /api/config: secret clearing ---

func TestHandlePutConfig_SecretClearedWithEmptyString(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.APIKey = "existing-secret"

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	// Send empty string to clear the secret
	body := `{"llm":{"api_key":""}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if saved.LLM.APIKey != "" {
		t.Errorf("api_key = %q, want empty (should be cleared)", saved.LLM.APIKey)
	}
}

// --- PUT /api/config: multiple sections in one request ---

func TestHandlePutConfig_MultipleSectionsSimultaneous(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.Model = "old-model"
	cfg.Gateway.Port = 18790
	cfg.Heartbeat.Interval = 30

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{"llm":{"model":"new-model"},"gateway":{"port":9999},"heartbeat":{"interval":60}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if saved.LLM.Model != "new-model" {
		t.Errorf("LLM model = %q, want %q", saved.LLM.Model, "new-model")
	}
	if saved.Gateway.Port != 9999 {
		t.Errorf("Gateway port = %d, want %d", saved.Gateway.Port, 9999)
	}
	if saved.Heartbeat.Interval != 60 {
		t.Errorf("Heartbeat interval = %d, want %d", saved.Heartbeat.Interval, 60)
	}
}

// --- PUT /api/config: in-memory config updated immediately ---

func TestHandlePutConfig_InMemoryConfigUpdated(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.Model = "original"

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{"llm":{"model":"updated"}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	// In-memory config should be updated immediately
	if s.cfg.LLM.Model != "updated" {
		t.Errorf("in-memory model = %q, want %q", s.cfg.LLM.Model, "updated")
	}

	// Disk should have new value
	saved, _ := config.LoadConfig(cfgPath)
	if saved.LLM.Model != "updated" {
		t.Errorf("disk model = %q, want %q", saved.LLM.Model, "updated")
	}
}

// --- GET /api/config: Discord token returned as-is ---

func TestHandleGetConfig_DiscordTokenReturnedAsIs(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Discord.Token = "discord-secret-token"

	s := newTestServer(cfg)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s.handleGetConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)

	discord := result["channels"].(map[string]interface{})["discord"].(map[string]interface{})
	if discord["token"] != "discord-secret-token" {
		t.Errorf("Discord token = %v, want %q", discord["token"], "discord-secret-token")
	}
}

// --- Schema: every section has at least one field ---

func TestBuildSchema_AllSectionsHaveFields(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	for _, sec := range schema.Sections {
		if len(sec.Fields) == 0 {
			t.Errorf("section %q has 0 fields, expected at least 1", sec.Key)
		}
	}
}

// --- Schema: deeply nested fields (3+ levels) ---

func TestBuildSchema_DeeplyNestedFields(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	// tools section has web.brave.api_key, web.duckduckgo.enabled, etc.
	var toolsSection *SchemaSection
	for i := range schema.Sections {
		if schema.Sections[i].Key == "tools" {
			toolsSection = &schema.Sections[i]
			break
		}
	}
	if toolsSection == nil {
		t.Fatal("tools section not found")
	}

	keySet := map[string]bool{}
	for _, f := range toolsSection.Fields {
		keySet[f.Key] = true
	}

	wantKeys := []string{
		"web.brave.api_key",
		"web.brave.enabled",
		"web.brave.max_results",
		"web.duckduckgo.enabled",
		"web.duckduckgo.max_results",
	}

	for _, k := range wantKeys {
		if !keySet[k] {
			t.Errorf("expected tools field %q, available: %v", k, keys(keySet))
		}
	}
}

func TestBuildSchema_DeeplyNestedSecretFlag(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	// tools.web.brave.api_key should be marked as secret
	var toolsSection *SchemaSection
	for i := range schema.Sections {
		if schema.Sections[i].Key == "tools" {
			toolsSection = &schema.Sections[i]
			break
		}
	}
	if toolsSection == nil {
		t.Fatal("tools section not found")
	}

	for _, f := range toolsSection.Fields {
		if f.Key == "web.brave.api_key" {
			if !f.Secret {
				t.Error("web.brave.api_key should be marked as secret")
			}
			return
		}
	}
	t.Error("web.brave.api_key field not found in tools section")
}

// --- Schema: []string default value ---

func TestBuildSchema_StringSliceDefault(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	// Find a []string field (e.g., channels.telegram.allow_from)
	var channels *SchemaSection
	for i := range schema.Sections {
		if schema.Sections[i].Key == "channels" {
			channels = &schema.Sections[i]
			break
		}
	}
	if channels == nil {
		t.Fatal("channels section not found")
	}

	for _, f := range channels.Fields {
		if f.Key == "telegram.allow_from" {
			if f.Type != "[]string" {
				t.Errorf("allow_from type = %q, want %q", f.Type, "[]string")
			}
			// Default should be an empty slice (marshals to [])
			defJSON, _ := json.Marshal(f.Default)
			if string(defJSON) != "[]" {
				t.Errorf("allow_from default = %s, want []", defJSON)
			}
			return
		}
	}
	t.Error("telegram.allow_from field not found")
}

// =============================================================================
// Additional tests: High priority
// =============================================================================

// --- #13: Intra-section partial update (fields not sent should be preserved) ---

func TestHandlePutConfig_IntraSectionPartialUpdate(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.Model = "gpt-4"
	cfg.LLM.APIKey = "secret-key"
	cfg.LLM.BaseURL = "https://example.com"

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	// Only send model — api_key and base_url should be preserved
	body := `{"llm":{"model":"claude"}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if saved.LLM.Model != "claude" {
		t.Errorf("model = %q, want %q", saved.LLM.Model, "claude")
	}
	if saved.LLM.APIKey != "secret-key" {
		t.Errorf("api_key = %q, want %q (should be preserved)", saved.LLM.APIKey, "secret-key")
	}
	if saved.LLM.BaseURL != "https://example.com" {
		t.Errorf("base_url = %q, want %q (should be preserved)", saved.LLM.BaseURL, "https://example.com")
	}
}

// --- #14: bool true → false via PUT ---

func TestHandlePutConfig_BoolTrueToFalse(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Enabled = true
	cfg.Heartbeat.Enabled = true

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{"channels":{"telegram":{"enabled":false}},"heartbeat":{"enabled":false}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if saved.Channels.Telegram.Enabled {
		t.Error("telegram enabled should be false after update")
	}
	if saved.Heartbeat.Enabled {
		t.Error("heartbeat enabled should be false after update")
	}
}

// --- #15: int set to zero ---

func TestHandlePutConfig_IntSetToZero(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.RateLimits.MaxToolCallsPerMinute = 30

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{"rate_limits":{"max_tool_calls_per_minute":0}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if saved.RateLimits.MaxToolCallsPerMinute != 0 {
		t.Errorf("max_tool_calls_per_minute = %d, want 0", saved.RateLimits.MaxToolCallsPerMinute)
	}
}

// --- #16: float set to zero ---

func TestHandlePutConfig_FloatSetToZero(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Temperature = 0.7

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{"agents":{"defaults":{"temperature":0}}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if saved.Agents.Defaults.Temperature != 0 {
		t.Errorf("temperature = %v, want 0", saved.Agents.Defaults.Temperature)
	}
}

// --- #18: MCP map PUT update ---

func TestHandlePutConfig_MCPMapUpdate(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.MCP = map[string]config.MCPServerConfig{
		"server1": {Command: "cmd1", Enabled: true, Description: "first"},
	}

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{"tools":{"mcp":{"server1":{"command":"cmd1","enabled":false,"description":"first"},"server2":{"command":"cmd2","enabled":true,"description":"second"}}}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if len(saved.Tools.MCP) != 2 {
		t.Fatalf("MCP count = %d, want 2", len(saved.Tools.MCP))
	}
	s1, ok := saved.Tools.MCP["server1"]
	if !ok {
		t.Fatal("server1 not found in saved MCP")
	}
	if s1.Enabled {
		t.Error("server1 enabled should be false")
	}
	s2, ok := saved.Tools.MCP["server2"]
	if !ok {
		t.Fatal("server2 not found in saved MCP")
	}
	if !s2.Enabled || s2.Command != "cmd2" {
		t.Errorf("server2 = %+v, want enabled=true command=cmd2", s2)
	}
}

func TestHandlePutConfig_MCPMapInMemoryUpdated(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.MCP = map[string]config.MCPServerConfig{
		"original": {Command: "orig", Enabled: true},
	}

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{"tools":{"mcp":{"original":{"command":"orig","enabled":true},"added":{"command":"new","enabled":true}}}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	// In-memory config should now contain the added server
	if _, found := s.cfg.Tools.MCP["added"]; !found {
		t.Error("in-memory MCP should contain 'added' after PUT")
	}
	if len(s.cfg.Tools.MCP) != 2 {
		t.Errorf("in-memory MCP count = %d, want 2", len(s.cfg.Tools.MCP))
	}
}

// --- #23: Concurrent PUT requests ---

func TestHandlePutConfig_ConcurrentPUT(t *testing.T) {
	cfg := config.DefaultConfig()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			body := fmt.Sprintf(`{"llm":{"model":"model-%d"}}`, n)
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
			s.handlePutConfig(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("concurrent PUT #%d status = %d, want 200", n, rr.Code)
			}
		}(i)
	}
	wg.Wait()

	// Verify the file is valid JSON with one of the models
	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error after concurrent PUTs: %v", err)
	}
	if !strings.HasPrefix(saved.LLM.Model, "model-") {
		t.Errorf("model = %q, want model-N", saved.LLM.Model)
	}
}

// --- #24: Concurrent GET + PUT ---

func TestHandlePutConfig_ConcurrentGETAndPUT(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.Model = "initial"

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}

	var wg sync.WaitGroup
	// Concurrent GETs
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
			s.handleGetConfig(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("concurrent GET status = %d, want 200", rr.Code)
			}
		}()
	}
	// Concurrent PUTs
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			body := fmt.Sprintf(`{"llm":{"model":"put-%d"}}`, n)
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
			s.handlePutConfig(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("concurrent PUT #%d status = %d, want 200", n, rr.Code)
			}
		}(i)
	}
	wg.Wait()
}

// =============================================================================
// Additional tests: Medium priority
// =============================================================================

// --- #8: Brave api_key returned as-is in GET ---

func TestHandleGetConfig_BraveAPIKeyReturnedAsIs(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Web.Brave.APIKey = "brave-secret"

	s := newTestServer(cfg)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s.handleGetConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)

	tools := result["tools"].(map[string]interface{})
	web := tools["web"].(map[string]interface{})
	brave := web["brave"].(map[string]interface{})
	if brave["api_key"] != "brave-secret" {
		t.Errorf("brave api_key = %v, want %q", brave["api_key"], "brave-secret")
	}
}

// --- #10: GET with MCP servers configured ---

func TestHandleGetConfig_WithMCPServers(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.MCP = map[string]config.MCPServerConfig{
		"test-server": {
			Command:     "test-cmd",
			Args:        []string{"--flag"},
			Env:         map[string]string{"KEY": "value"},
			Description: "test server",
			Enabled:     true,
		},
	}

	s := newTestServer(cfg)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s.handleGetConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)

	tools := result["tools"].(map[string]interface{})
	mcp, ok := tools["mcp"].(map[string]interface{})
	if !ok {
		t.Fatal("mcp should be a map in response")
	}
	srv, ok := mcp["test-server"].(map[string]interface{})
	if !ok {
		t.Fatal("test-server not found in mcp response")
	}
	if srv["command"] != "test-cmd" {
		t.Errorf("command = %v, want %q", srv["command"], "test-cmd")
	}
	if srv["enabled"] != true {
		t.Errorf("enabled = %v, want true", srv["enabled"])
	}
}

// --- #11: FlexibleStringSlice serialization in GET ---

func TestHandleGetConfig_FlexibleStringSliceSerialization(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.AllowFrom = config.FlexibleStringSlice{"user1", "12345"}

	s := newTestServer(cfg)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s.handleGetConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)

	channels := result["channels"].(map[string]interface{})
	tg := channels["telegram"].(map[string]interface{})
	af, ok := tg["allow_from"].([]interface{})
	if !ok {
		t.Fatal("allow_from should be a JSON array")
	}
	if len(af) != 2 {
		t.Fatalf("allow_from length = %d, want 2", len(af))
	}
	if af[0] != "user1" || af[1] != "12345" {
		t.Errorf("allow_from = %v, want [user1, 12345]", af)
	}
}

// --- #17: FlexibleStringSlice PUT round-trip ---

func TestHandlePutConfig_FlexibleStringSliceUpdate(t *testing.T) {
	cfg := config.DefaultConfig()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{"channels":{"telegram":{"allow_from":["user1",456,"user3"]}}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	af := saved.Channels.Telegram.AllowFrom
	if len(af) != 3 {
		t.Fatalf("allow_from length = %d, want 3", len(af))
	}
	if af[0] != "user1" {
		t.Errorf("allow_from[0] = %q, want %q", af[0], "user1")
	}
	// 456 (number) should be converted to "456"
	if af[1] != "456" {
		t.Errorf("allow_from[1] = %q, want %q", af[1], "456")
	}
	if af[2] != "user3" {
		t.Errorf("allow_from[2] = %q, want %q", af[2], "user3")
	}
}

// --- #19: Deep nested partial update (tools.web.brave) ---

func TestHandlePutConfig_DeepNestedPartialUpdate(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Web.Brave.Enabled = false
	cfg.Tools.Web.Brave.APIKey = "brave-key"
	cfg.Tools.Web.Brave.MaxResults = 5
	cfg.Tools.Web.DuckDuckGo.Enabled = true
	cfg.Tools.Web.DuckDuckGo.MaxResults = 10

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	// Only update brave.api_key — everything else should be preserved
	body := `{"tools":{"web":{"brave":{"api_key":"new-brave-key"}}}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if saved.Tools.Web.Brave.APIKey != "new-brave-key" {
		t.Errorf("brave api_key = %q, want %q", saved.Tools.Web.Brave.APIKey, "new-brave-key")
	}
	if saved.Tools.Web.Brave.Enabled != false {
		t.Error("brave enabled should be preserved as false")
	}
	if saved.Tools.Web.Brave.MaxResults != 5 {
		t.Errorf("brave max_results = %d, want 5 (preserved)", saved.Tools.Web.Brave.MaxResults)
	}
	if !saved.Tools.Web.DuckDuckGo.Enabled {
		t.Error("duckduckgo enabled should be preserved as true")
	}
	if saved.Tools.Web.DuckDuckGo.MaxResults != 10 {
		t.Errorf("duckduckgo max_results = %d, want 10 (preserved)", saved.Tools.Web.DuckDuckGo.MaxResults)
	}
}

// --- #20: agents.defaults intra-section partial update ---

func TestHandlePutConfig_AgentsDefaultsPartialUpdate(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.MaxTokens = 8192
	cfg.Agents.Defaults.Workspace = "/my/workspace"
	cfg.Agents.Defaults.Temperature = 0.5
	cfg.Agents.Defaults.ContextWindow = 128000

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	body := `{"agents":{"defaults":{"max_tokens":4096}}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(body))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	saved, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if saved.Agents.Defaults.MaxTokens != 4096 {
		t.Errorf("max_tokens = %d, want 4096", saved.Agents.Defaults.MaxTokens)
	}
	if saved.Agents.Defaults.Workspace != "/my/workspace" {
		t.Errorf("workspace = %q, want preserved", saved.Agents.Defaults.Workspace)
	}
	if saved.Agents.Defaults.Temperature != 0.5 {
		t.Errorf("temperature = %v, want 0.5 (preserved)", saved.Agents.Defaults.Temperature)
	}
	if saved.Agents.Defaults.ContextWindow != 128000 {
		t.Errorf("context_window = %d, want 128000 (preserved)", saved.Agents.Defaults.ContextWindow)
	}
}

// --- #21: Empty body (0 bytes, not {}) ---

func TestHandlePutConfig_EmptyBodyZeroBytes(t *testing.T) {
	cfg := config.DefaultConfig()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(""))
	s.handlePutConfig(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for empty body", rr.Code)
	}
}

// --- #22: PUT → GET round-trip ---

func TestHandlePutConfig_ThenGetRoundTrip(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.APIKey = "my-secret"

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	config.SaveConfig(cfgPath, cfg)

	s := &Server{cfg: cfg, configPath: cfgPath}

	// PUT to change model (only send model — api_key preserved via partial update)
	putBody := `{"llm":{"model":"new-model"}}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/config", bytes.NewBufferString(putBody))
	s.handlePutConfig(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200", rr.Code)
	}

	// Load saved config into a new server to simulate restart
	savedCfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	s2 := &Server{cfg: savedCfg, configPath: cfgPath}

	// GET the config
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s2.handleGetConfig(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", rr.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	llm := result["llm"].(map[string]interface{})
	if llm["model"] != "new-model" {
		t.Errorf("GET model = %v, want %q", llm["model"], "new-model")
	}
	if llm["api_key"] != "my-secret" {
		t.Errorf("GET api_key = %v, want %q", llm["api_key"], "my-secret")
	}
}

// =============================================================================
// Additional tests: Low priority
// =============================================================================

// --- #2: Stop() called twice ---

func TestServerStop_CalledTwice(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = -1

	s := NewServer(cfg, "/tmp/config.json", nil)
	s.Start()

	ctx := context.Background()
	if err := s.Stop(ctx); err != nil {
		t.Errorf("first Stop() error: %v", err)
	}
	// Second stop — server is already shut down
	if err := s.Stop(ctx); err != nil {
		// Acceptable: may return ErrServerClosed or nil
		t.Logf("second Stop() returned: %v (acceptable)", err)
	}
}

// --- #3: DELETE and PATCH → 405 ---

func TestServerRouting_UnsupportedMethods(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = -1

	s := NewServer(cfg, "/tmp/config.json", nil)
	s.Start()
	defer s.Stop(context.Background())

	methods := []string{http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/api/config", nil)
		s.server.Handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /api/config status = %d, want 405", method, rr.Code)
		}
	}
}

// --- #4: Unknown path → 404 ---

func TestServerRouting_UnknownPath(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = -1

	s := NewServer(cfg, "/tmp/config.json", nil)
	s.Start()
	defer s.Stop(context.Background())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/unknown", nil)
	s.server.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("GET /api/unknown status = %d, want 404", rr.Code)
	}
}

// --- #5: "bearer " lowercase → 401 ---

func TestAuthMiddleware_LowercaseBearer_401(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.APIKey = "test-key"
	s := newTestServer(cfg)

	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "bearer test-key")
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for lowercase 'bearer'", rr.Code)
	}
}

// --- #6: Token with surrounding whitespace ---

func TestAuthMiddleware_TokenWithWhitespace_403(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.APIKey = "test-key"
	s := newTestServer(cfg)

	handler := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer test-key ")
	handler(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 for token with trailing space", rr.Code)
	}
}

// --- #7: Auth applied to GET /api/config/schema via routing ---

func TestServerRouting_SchemaAuthRequired(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = -1
	cfg.Gateway.APIKey = "schema-auth-key"

	s := NewServer(cfg, "/tmp/config.json", nil)
	s.Start()
	defer s.Stop(context.Background())

	// Without auth
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config/schema", nil)
	s.server.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/config/schema without auth status = %d, want 401", rr.Code)
	}

	// With correct auth
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/config/schema", nil)
	req.Header.Set("Authorization", "Bearer schema-auth-key")
	s.server.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("GET /api/config/schema with auth status = %d, want 200", rr.Code)
	}
}

// --- #9: WhatsApp bridge_url not masked ---

func TestHandleGetConfig_WhatsAppBridgeURLNotMasked(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.WhatsApp.BridgeURL = "ws://localhost:3001"

	s := newTestServer(cfg)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s.handleGetConfig(rr, req)

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)

	wa := result["channels"].(map[string]interface{})["whatsapp"].(map[string]interface{})
	if wa["bridge_url"] != "ws://localhost:3001" {
		t.Errorf("bridge_url = %v, should NOT be masked", wa["bridge_url"])
	}
}

// --- #12: GET returns all 7 sections ---

func TestHandleGetConfig_AllSectionsPresent(t *testing.T) {
	cfg := config.DefaultConfig()
	s := newTestServer(cfg)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	s.handleGetConfig(rr, req)

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)

	wantSections := []string{"llm", "agents", "channels", "gateway", "tools", "heartbeat", "rate_limits"}
	for _, sec := range wantSections {
		if _, ok := result[sec]; !ok {
			t.Errorf("section %q missing from GET /api/config response", sec)
		}
	}
}

// --- #28: agents section all fields in schema ---

func TestBuildSchema_AgentsSectionFields(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	var agents *SchemaSection
	for i := range schema.Sections {
		if schema.Sections[i].Key == "agents" {
			agents = &schema.Sections[i]
			break
		}
	}
	if agents == nil {
		t.Fatal("agents section not found")
	}

	keySet := map[string]bool{}
	for _, f := range agents.Fields {
		keySet[f.Key] = true
	}

	wantKeys := []string{
		"defaults.workspace",
		"defaults.data_dir",
		"defaults.restrict_to_workspace",
		"defaults.max_tokens",
		"defaults.context_window",
		"defaults.temperature",
		"defaults.max_tool_iterations",
	}
	for _, k := range wantKeys {
		if !keySet[k] {
			t.Errorf("agents field %q not found, available: %v", k, keys(keySet))
		}
	}
}

// --- #28b: directory type override ---

func TestBuildSchema_DirectoryType(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	// Find the agents section which contains the defaults sub-fields
	var agentsFields []SchemaField
	for _, sec := range schema.Sections {
		if sec.Key == "agents" {
			agentsFields = sec.Fields
			break
		}
	}
	if agentsFields == nil {
		t.Fatal("agents section not found in schema")
	}

	fieldByKey := make(map[string]SchemaField)
	for _, f := range agentsFields {
		fieldByKey[f.Key] = f
	}

	dirFields := []string{"defaults.workspace", "defaults.data_dir"}
	for _, k := range dirFields {
		f, ok := fieldByKey[k]
		if !ok {
			t.Errorf("field %q not found in agents section", k)
			continue
		}
		if f.Type != "directory" {
			t.Errorf("field %q: want type %q, got %q", k, "directory", f.Type)
		}
	}
}

// --- #29: rate_limits section fields in schema ---

func TestBuildSchema_RateLimitsSectionFields(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	var rl *SchemaSection
	for i := range schema.Sections {
		if schema.Sections[i].Key == "rate_limits" {
			rl = &schema.Sections[i]
			break
		}
	}
	if rl == nil {
		t.Fatal("rate_limits section not found")
	}

	keySet := map[string]bool{}
	for _, f := range rl.Fields {
		keySet[f.Key] = true
	}

	wantKeys := []string{"max_tool_calls_per_minute", "max_requests_per_minute"}
	for _, k := range wantKeys {
		if !keySet[k] {
			t.Errorf("rate_limits field %q not found", k)
		}
	}
}

// --- #30: gateway section all fields in schema ---

func TestBuildSchema_GatewaySectionFields(t *testing.T) {
	schema := BuildSchema(config.DefaultConfig())

	var gw *SchemaSection
	for i := range schema.Sections {
		if schema.Sections[i].Key == "gateway" {
			gw = &schema.Sections[i]
			break
		}
	}
	if gw == nil {
		t.Fatal("gateway section not found")
	}

	keySet := map[string]bool{}
	for _, f := range gw.Fields {
		keySet[f.Key] = true
	}

	wantKeys := []string{"host", "port", "api_key"}
	for _, k := range wantKeys {
		if !keySet[k] {
			t.Errorf("gateway field %q not found", k)
		}
	}

	// api_key should be secret
	for _, f := range gw.Fields {
		if f.Key == "api_key" && !f.Secret {
			t.Error("gateway api_key should be marked as secret")
		}
	}
}


// --- #34: goTypeToSchema unknown type ---

func TestGoTypeToSchema_UnknownType(t *testing.T) {
	// chan int should return "any"
	ct := reflect.TypeOf(make(chan int))
	got := goTypeToSchema(ct)
	if got != "any" {
		t.Errorf("goTypeToSchema(chan int) = %q, want %q", got, "any")
	}
}

// --- #35: goTypeToSchema []int → "[]any" ---

func TestGoTypeToSchema_IntSlice(t *testing.T) {
	ct := reflect.TypeOf([]int{})
	got := goTypeToSchema(ct)
	if got != "[]any" {
		t.Errorf("goTypeToSchema([]int) = %q, want %q", got, "[]any")
	}
}

// --- helpers ---

func newTestServer(cfg *config.Config) *Server {
	return &Server{
		cfg:        cfg,
		configPath: "/tmp/test-config.json",
	}
}

// allFields collects all fields from all sections, using the leaf key for lookup.
func allFields(schema SchemaResponse) map[string]SchemaField {
	m := make(map[string]SchemaField)
	for _, sec := range schema.Sections {
		for _, f := range sec.Fields {
			// Use the leaf key (after last dot)
			key := f.Key
			if idx := lastDot(key); idx >= 0 {
				key = key[idx+1:]
			}
			m[key] = f
		}
	}
	return m
}

func keys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func lastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}
