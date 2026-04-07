package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/sipeed/picoclaw/pkg/channels/pico"
	"github.com/sipeed/picoclaw/pkg/config"
	ppid "github.com/sipeed/picoclaw/pkg/pid"
)

func TestEnsurePicoChannel_FreshConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	changed, err := h.EnsurePicoChannel("")
	if err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
	}
	if !changed {
		t.Fatal("EnsurePicoChannel() should report changed on a fresh config")
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.Channels.Pico.Enabled {
		t.Error("expected Pico to be enabled after setup")
	}
	if cfg.Channels.Pico.Token.String() == "" {
		t.Error("expected a non-empty token after setup")
	}
}

func TestEnsurePicoChannel_DoesNotEnableTokenQuery(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	if _, err := h.EnsurePicoChannel(""); err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
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

	if _, err := h.EnsurePicoChannel("http://localhost:18800"); err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
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

	if _, err := h.EnsurePicoChannel(""); err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
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
	if _, err := h.EnsurePicoChannel(lanOrigin); err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
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
	cfg.Channels.Pico.SetToken("user-custom-token")
	cfg.Channels.Pico.AllowTokenQuery = true
	cfg.Channels.Pico.AllowOrigins = []string{"https://myapp.example.com"}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)

	changed, err := h.EnsurePicoChannel("")
	if err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
	}
	if changed {
		t.Error("EnsurePicoChannel() should not change a fully configured config")
	}

	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Channels.Pico.Token.String() != "user-custom-token" {
		t.Errorf("token = %q, want %q", cfg.Channels.Pico.Token.String(), "user-custom-token")
	}
	if !cfg.Channels.Pico.AllowTokenQuery {
		t.Error("user's allow_token_query=true must be preserved")
	}
	if len(cfg.Channels.Pico.AllowOrigins) != 1 || cfg.Channels.Pico.AllowOrigins[0] != "https://myapp.example.com" {
		t.Errorf("allow_origins = %v, want [https://myapp.example.com]", cfg.Channels.Pico.AllowOrigins)
	}
}

func TestEnsurePicoChannel_ExistingConfigWithoutSecurityFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")

	cfg := config.DefaultConfig()
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err = os.WriteFile(configPath, raw, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	h := NewHandler(configPath)

	changed, err := h.EnsurePicoChannel("")
	if err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
	}
	if !changed {
		t.Fatal("EnsurePicoChannel() should report changed when pico is missing")
	}

	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.Channels.Pico.Enabled {
		t.Error("expected Pico to be enabled after setup")
	}
	if cfg.Channels.Pico.Token.String() == "" {
		t.Error("expected a non-empty token after setup")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(configPath), config.SecurityConfigFile)); err != nil {
		t.Fatalf("expected .security.yml to be created: %v", err)
	}
}

func TestEnsurePicoChannel_ConfiguresPicoWithoutGateway(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = ""
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	if _, err := h.EnsurePicoChannel(""); err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !cfg.Channels.Pico.Enabled {
		t.Error("expected Pico to be enabled after launcher startup setup")
	}
	if cfg.Channels.Pico.Token.String() == "" {
		t.Error("expected a non-empty token after launcher startup setup")
	}
}

func TestEnsurePicoChannel_Idempotent(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	origin := "http://localhost:18800"

	// First call sets things up
	if _, err := h.EnsurePicoChannel(origin); err != nil {
		t.Fatalf("first EnsurePicoChannel() error = %v", err)
	}

	cfg1, _ := config.LoadConfig(configPath)
	token1 := cfg1.Channels.Pico.Token.String()

	// Second call should be a no-op
	changed, err := h.EnsurePicoChannel(origin)
	if err != nil {
		t.Fatalf("second EnsurePicoChannel() error = %v", err)
	}
	if changed {
		t.Error("second EnsurePicoChannel() should not report changed")
	}

	cfg2, _ := config.LoadConfig(configPath)
	if cfg2.Channels.Pico.Token.String() != token1 {
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

func TestHandleWebSocketProxyReloadsGatewayTargetFromConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PICOCLAW_HOME", home)

	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	handler := h.handleWebSocketProxy()

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pico/ws" {
			t.Fatalf("server1 path = %q, want %q", r.URL.Path, "/pico/ws")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "server1")
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pico/ws" {
			t.Fatalf("server2 path = %q, want %q", r.URL.Path, "/pico/ws")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "server2")
	}))
	defer server2.Close()

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = mustGatewayTestPort(t, server1.URL)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	if _, err := ppid.WritePidFile(globalConfigDir(), cfg.Gateway.Host, cfg.Gateway.Port); err != nil {
		t.Fatalf("WritePidFile() error = %v", err)
	}
	origPidData := gateway.pidData
	origPicoToken := gateway.picoToken
	t.Cleanup(func() {
		ppid.RemovePidFile(globalConfigDir())
		gateway.pidData = origPidData
		gateway.picoToken = origPicoToken
	})

	gateway.pidData = &ppid.PidFileData{}
	gateway.picoToken = "pico"
	req1 := httptest.NewRequest(http.MethodGet, "/pico/ws", nil)
	req1.Header.Set(protocolKey, tokenPrefix+"wrong_token")
	rec1 := httptest.NewRecorder()
	handler(rec1, req1)

	if rec1.Code != http.StatusForbidden {
		t.Fatalf("first status = %d, want %d", rec1.Code, http.StatusForbidden)
	}

	req1 = httptest.NewRequest(http.MethodGet, "/pico/ws", nil)
	req1.Header.Set(protocolKey, tokenPrefix+"pico")
	rec1 = httptest.NewRecorder()
	handler(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", rec1.Code, http.StatusOK)
	}
	if body := rec1.Body.String(); body != "server1" {
		t.Fatalf("first body = %q, want %q", body, "server1")
	}

	cfg.Gateway.Port = mustGatewayTestPort(t, server2.URL)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/pico/ws", nil)
	req2.Header.Set(protocolKey, tokenPrefix+"pico")
	rec2 := httptest.NewRecorder()
	handler(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", rec2.Code, http.StatusOK)
	}
	if body := rec2.Body.String(); body != "server2" {
		t.Fatalf("second body = %q, want %q", body, "server2")
	}
}

func TestHandleWebSocketProxyLoadsCachedPicoTokenWhenMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PICOCLAW_HOME", home)

	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	handler := h.handleWebSocketProxy()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pico/ws" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/pico/ws")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "proxied")
	}))
	defer server.Close()

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = mustGatewayTestPort(t, server.URL)
	cfg.Channels.Pico.Enabled = true
	cfg.Channels.Pico.SetToken("cached-token")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	if _, err := ppid.WritePidFile(globalConfigDir(), cfg.Gateway.Host, cfg.Gateway.Port); err != nil {
		t.Fatalf("WritePidFile() error = %v", err)
	}
	t.Cleanup(func() {
		ppid.RemovePidFile(globalConfigDir())
	})

	origPidData := gateway.pidData
	origPicoToken := gateway.picoToken
	t.Cleanup(func() {
		gateway.pidData = origPidData
		gateway.picoToken = origPicoToken
	})

	gateway.pidData = &ppid.PidFileData{}
	gateway.picoToken = ""

	req := httptest.NewRequest(http.MethodGet, "/pico/ws?session_id=test-session", nil)
	req.Header.Set(protocolKey, tokenPrefix+"cached-token")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); body != "proxied" {
		t.Fatalf("body = %q, want %q", body, "proxied")
	}
	if gateway.picoToken != "cached-token" {
		t.Fatalf("gateway.picoToken = %q, want %q", gateway.picoToken, "cached-token")
	}
}

func TestHandleWebSocketProxyLoadsPidDataOnDemand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PICOCLAW_HOME", home)

	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	handler := h.handleWebSocketProxy()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pico/ws" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/pico/ws")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, r.Header.Get(protocolKey))
	}))
	defer server.Close()

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = mustGatewayTestPort(t, server.URL)
	cfg.Channels.Pico.Enabled = true
	cfg.Channels.Pico.SetToken("ui-token")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	pidData, err := ppid.WritePidFile(globalConfigDir(), cfg.Gateway.Host, cfg.Gateway.Port)
	if err != nil {
		t.Fatalf("WritePidFile() error = %v", err)
	}
	t.Cleanup(func() {
		ppid.RemovePidFile(globalConfigDir())
	})

	origPidData := gateway.pidData
	origPicoToken := gateway.picoToken
	origStatus := gateway.runtimeStatus
	t.Cleanup(func() {
		gateway.mu.Lock()
		gateway.pidData = origPidData
		gateway.picoToken = origPicoToken
		gateway.runtimeStatus = origStatus
		gateway.mu.Unlock()
	})

	gateway.mu.Lock()
	gateway.pidData = nil
	gateway.picoToken = ""
	setGatewayRuntimeStatusLocked("stopped")
	gateway.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/pico/ws?session_id=test-session", nil)
	req.Header.Set(protocolKey, tokenPrefix+"ui-token")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	expected := tokenPrefix + pico.PicoTokenPrefix + pidData.Token + "ui-token"
	if got := rec.Body.String(); got != expected {
		t.Fatalf("forwarded protocol = %q, want %q", got, expected)
	}

	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	if gateway.pidData == nil {
		t.Fatal("gateway.pidData should be loaded from pid file")
	}
	if gateway.runtimeStatus != "running" {
		t.Fatalf("runtimeStatus = %q, want %q", gateway.runtimeStatus, "running")
	}
}

func TestHandleWebSocketProxyRejectsStalePidDataAfterProcessExit(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	handler := h.handleWebSocketProxy()

	cfg := config.DefaultConfig()
	cfg.Channels.Pico.Enabled = true
	cfg.Channels.Pico.SetToken("ui-token")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := startLongRunningProcess(t)
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	_ = cmd.Wait()

	origPidData := gateway.pidData
	origPicoToken := gateway.picoToken
	origCmd := gateway.cmd
	origStatus := gateway.runtimeStatus
	t.Cleanup(func() {
		gateway.mu.Lock()
		gateway.pidData = origPidData
		gateway.picoToken = origPicoToken
		gateway.cmd = origCmd
		gateway.runtimeStatus = origStatus
		gateway.mu.Unlock()
	})

	gateway.mu.Lock()
	gateway.pidData = &ppid.PidFileData{PID: cmd.Process.Pid, Token: "stale-token"}
	gateway.picoToken = "ui-token"
	gateway.cmd = cmd
	setGatewayRuntimeStatusLocked("running")
	gateway.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/pico/ws?session_id=test-session", nil)
	req.Header.Set(protocolKey, tokenPrefix+"ui-token")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	if gateway.pidData != nil {
		t.Fatal("gateway.pidData should be cleared after stale process exit is detected")
	}
}

func mustGatewayTestPort(t *testing.T, rawURL string) int {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("Atoi(%q) error = %v", parsed.Port(), err)
	}

	return port
}
