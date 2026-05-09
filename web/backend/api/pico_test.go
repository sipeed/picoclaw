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
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/memory"
	ppid "github.com/sipeed/picoclaw/pkg/pid"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func newPicoProxyRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, "http://launcher.local:18800"+path, nil)
	req.Header.Set("Origin", "http://launcher.local:18800")
	return req
}

func TestEnsurePicoChannel_FreshConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	changed, err := h.EnsurePicoChannel()
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

	bc := cfg.Channels["pico"]
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	picoCfg := decoded.(*config.PicoSettings)
	if !bc.Enabled {
		t.Error("expected Pico to be enabled after setup")
	}
	if picoCfg.Token.String() == "" {
		t.Error("expected a non-empty token after setup")
	}
}

func TestEnsurePicoChannel_DoesNotEnableTokenQuery(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	if _, err := h.EnsurePicoChannel(); err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	bc := cfg.Channels["pico"]
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	picoCfg := decoded.(*config.PicoSettings)
	if picoCfg.AllowTokenQuery {
		t.Error("setup must not enable allow_token_query by default")
	}
}

func TestEnsurePicoChannel_LeavesAllowOriginsEmptyByDefault(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	if _, err := h.EnsurePicoChannel(); err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	bc := cfg.Channels["pico"]
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	picoCfg := decoded.(*config.PicoSettings)
	if len(picoCfg.AllowOrigins) != 0 {
		t.Errorf("allow_origins = %v, want empty", picoCfg.AllowOrigins)
	}
}

func TestEnsurePicoChannel_NoOriginConfigurationRequired(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	if _, err := h.EnsurePicoChannel(); err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	bc := cfg.Channels["pico"]
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	picoCfg := decoded.(*config.PicoSettings)
	if len(picoCfg.AllowOrigins) != 0 {
		t.Errorf("allow_origins = %v, want empty", picoCfg.AllowOrigins)
	}
}

func TestEnsurePicoChannel_PreservesUserSettings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")

	// Pre-configure with custom user settings
	cfg := config.DefaultConfig()
	bc := cfg.Channels["pico"]
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	picoCfg := decoded.(*config.PicoSettings)
	bc.Enabled = true
	picoCfg.SetToken("user-custom-token")
	picoCfg.AllowTokenQuery = true
	picoCfg.AllowOrigins = []string{"https://myapp.example.com"}
	if err = config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)

	changed, err := h.EnsurePicoChannel()
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

	bc = cfg.Channels["pico"]
	decoded, err = bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	picoCfg = decoded.(*config.PicoSettings)
	if picoCfg.Token.String() != "user-custom-token" {
		t.Errorf("token = %q, want %q", picoCfg.Token.String(), "user-custom-token")
	}
	if !picoCfg.AllowTokenQuery {
		t.Error("user's allow_token_query=true must be preserved")
	}
	if len(picoCfg.AllowOrigins) != 1 || picoCfg.AllowOrigins[0] != "https://myapp.example.com" {
		t.Errorf("allow_origins = %v, want [https://myapp.example.com]", picoCfg.AllowOrigins)
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

	changed, err := h.EnsurePicoChannel()
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

	bc := cfg.Channels["pico"]
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	picoCfg := decoded.(*config.PicoSettings)
	if !bc.Enabled {
		t.Error("expected Pico to be enabled after setup")
	}
	if picoCfg.Token.String() == "" {
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
	if _, err := h.EnsurePicoChannel(); err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	bc := cfg.Channels["pico"]
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	picoCfg := decoded.(*config.PicoSettings)
	if !bc.Enabled {
		t.Error("expected Pico to be enabled after launcher startup setup")
	}
	if picoCfg.Token.String() == "" {
		t.Error("expected a non-empty token after launcher startup setup")
	}
}

func TestEnsurePicoChannel_Idempotent(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	// First call sets things up
	if _, err := h.EnsurePicoChannel(); err != nil {
		t.Fatalf("first EnsurePicoChannel() error = %v", err)
	}

	cfg1, _ := config.LoadConfig(configPath)
	bc := cfg1.Channels["pico"]
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	picoCfg := decoded.(*config.PicoSettings)
	token1 := picoCfg.Token.String()

	// Second call should be a no-op
	changed, err := h.EnsurePicoChannel()
	if err != nil {
		t.Fatalf("second EnsurePicoChannel() error = %v", err)
	}
	if changed {
		t.Error("second EnsurePicoChannel() should not report changed")
	}

	cfg2, _ := config.LoadConfig(configPath)
	bc = cfg2.Channels["pico"]
	decoded, err = bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	picoCfg = decoded.(*config.PicoSettings)
	if picoCfg.Token.String() != token1 {
		t.Error("token should not change on subsequent calls")
	}
}

func TestHandlePicoSetup_DoesNotPersistRequestOrigin(t *testing.T) {
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

	bc := cfg.Channels["pico"]
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	picoCfg := decoded.(*config.PicoSettings)
	if len(picoCfg.AllowOrigins) != 0 {
		t.Errorf("allow_origins = %v, want empty", picoCfg.AllowOrigins)
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

	if _, ok := resp["token"]; ok {
		t.Error("response must not expose the raw pico token")
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
	if resp["configured"] != true {
		t.Error("response should have configured=true")
	}
}

func TestHandleGetPicoInfo_OmitsToken(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	if _, err := h.EnsurePicoChannel(); err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://launcher.local/api/pico/info", nil)
	rec := httptest.NewRecorder()

	h.handleGetPicoInfo(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := resp["token"]; ok {
		t.Fatal("info response must not expose the raw pico token")
	}
	if resp["enabled"] != true {
		t.Fatalf("enabled = %#v, want true", resp["enabled"])
	}
	if resp["configured"] != true {
		t.Fatalf("configured = %#v, want true", resp["configured"])
	}
	if resp["ws_url"] == nil || resp["ws_url"] == "" {
		t.Fatal("response should contain ws_url")
	}
}

func TestHandleRegenPicoToken_RefreshesGatewayTokenCache(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	if _, err := h.EnsurePicoChannel(); err != nil {
		t.Fatalf("EnsurePicoChannel() error = %v", err)
	}

	origPicoToken := gateway.picoToken
	t.Cleanup(func() {
		gateway.mu.Lock()
		gateway.picoToken = origPicoToken
		gateway.mu.Unlock()
	})

	gateway.mu.Lock()
	gateway.picoToken = "stale-token"
	gateway.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "http://launcher.local/api/pico/token", nil)
	rec := httptest.NewRecorder()
	h.handleRegenPicoToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	bc := cfg.Channels["pico"]
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	token := decoded.(*config.PicoSettings).Token.String()
	if token == "" {
		t.Fatal("expected regenerated pico token to be persisted")
	}
	if token == "stale-token" {
		t.Fatal("expected regenerated pico token to differ from stale cache")
	}

	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	if gateway.picoToken != token {
		t.Fatalf("gateway.picoToken = %q, want %q", gateway.picoToken, token)
	}
}

func TestHandleWebSocketProxyReloadsGatewayTargetFromConfig(t *testing.T) {
	origMatcher := gatewayProcessMatcher
	gatewayProcessMatcher = func(int) (bool, bool) { return true, true }
	t.Cleanup(func() { gatewayProcessMatcher = origMatcher })

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
	cmd := startGatewayLikeProcess(t)
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})
	writeTestPidFile(t, ppid.PidFileData{
		PID:   cmd.Process.Pid,
		Token: "test-token",
		Host:  cfg.Gateway.Host,
		Port:  cfg.Gateway.Port,
	})
	origPidData := gateway.pidData
	origPicoToken := gateway.picoToken
	t.Cleanup(func() {
		ppid.RemovePidFile(globalConfigDir())
		gateway.pidData = origPidData
		gateway.picoToken = origPicoToken
	})

	gateway.pidData = &ppid.PidFileData{}
	gateway.picoToken = "pico"
	req1 := newPicoProxyRequest(http.MethodGet, "/pico/ws")
	rec1 := httptest.NewRecorder()
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

	req2 := newPicoProxyRequest(http.MethodGet, "/pico/ws")
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
	origMatcher := gatewayProcessMatcher
	gatewayProcessMatcher = func(int) (bool, bool) { return true, true }
	t.Cleanup(func() { gatewayProcessMatcher = origMatcher })

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
	bc := cfg.Channels["pico"]
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	picoCfg := decoded.(*config.PicoSettings)
	bc.Enabled = true
	picoCfg.SetToken("cached-token")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	cmd := startGatewayLikeProcess(t)
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})
	writeTestPidFile(t, ppid.PidFileData{
		PID:   cmd.Process.Pid,
		Token: "test-token",
		Host:  cfg.Gateway.Host,
		Port:  cfg.Gateway.Port,
	})
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

	req := newPicoProxyRequest(http.MethodGet, "/pico/ws?session_id=test-session")
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
	origMatcher := gatewayProcessMatcher
	gatewayProcessMatcher = func(int) (bool, bool) { return true, true }
	t.Cleanup(func() { gatewayProcessMatcher = origMatcher })

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
	bc := cfg.Channels["pico"]
	bc.Enabled = true
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	decoded.(*config.PicoSettings).SetToken("ui-token")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := startGatewayLikeProcess(t)
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})
	pidData := ppid.PidFileData{
		PID:   cmd.Process.Pid,
		Token: "test-token",
		Host:  cfg.Gateway.Host,
		Port:  cfg.Gateway.Port,
	}
	writeTestPidFile(t, pidData)
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

	req := newPicoProxyRequest(http.MethodGet, "/pico/ws?session_id=test-session")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	expected := tokenPrefix + "ui-token"
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

func TestCreatePicoHTTPProxyInjectsGatewayAuth(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = 18790
	bc := cfg.Channels["pico"]
	bc.Enabled = true
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	decoded.(*config.PicoSettings).SetToken("ui-token")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	proxy := h.createPicoHTTPProxy("ui-token")
	var capturedPath string
	var capturedAuth string
	proxy.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		capturedPath = req.URL.Path
		capturedAuth = req.Header.Get("Authorization")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("proxied")),
			Request:    req,
		}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/pico/media/attachment-1", nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if capturedPath != "/pico/media/attachment-1" {
		t.Fatalf("capturedPath = %q, want %q", capturedPath, "/pico/media/attachment-1")
	}
	expected := "Bearer ui-token"
	if capturedAuth != expected {
		t.Fatalf("Authorization = %q, want %q", capturedAuth, expected)
	}
}

func TestHandlePicoMediaProxyUsesRawBearerToken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PICOCLAW_HOME", home)

	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	handler := h.handlePicoMediaProxy()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pico/media/attachment-1" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/pico/media/attachment-1")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ui-token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer ui-token")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "proxied-media")
	}))
	defer server.Close()

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = mustGatewayTestPort(t, server.URL)
	bc := cfg.Channels["pico"]
	bc.Enabled = true
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	decoded.(*config.PicoSettings).SetToken("ui-token")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := startGatewayLikeProcess(t)
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	origPidData := gateway.pidData
	origPicoToken := gateway.picoToken
	origCmd := gateway.cmd
	t.Cleanup(func() {
		gateway.mu.Lock()
		gateway.pidData = origPidData
		gateway.picoToken = origPicoToken
		gateway.cmd = origCmd
		gateway.mu.Unlock()
	})

	gateway.mu.Lock()
	gateway.pidData = &ppid.PidFileData{PID: cmd.Process.Pid}
	gateway.picoToken = "ui-token"
	gateway.cmd = cmd
	gateway.mu.Unlock()

	req := newPicoProxyRequest(http.MethodGet, "/pico/media/attachment-1")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); body != "proxied-media" {
		t.Fatalf("body = %q, want %q", body, "proxied-media")
	}
}

func TestHandleGetPicoMemoryGraph_BuildsWorkspaceAndSessionGraph(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	workspaceDir := filepath.Join(t.TempDir(), "workspace")
	sessionsDir := filepath.Join(workspaceDir, "sessions")
	memoryDir := filepath.Join(workspaceDir, "memory")

	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(memoryDir) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(memoryDir, time.Now().Format("200601")), 0o755); err != nil {
		t.Fatalf("MkdirAll(dailyNoteDir) error = %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspaceDir
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	store, err := memory.NewJSONLStore(sessionsDir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error = %v", err)
	}

	sessionKey := legacyPicoSessionPrefix + "graph-session"
	for _, msg := range []providers.Message{
		{Role: "user", Content: "Remember the Nairobi deployment notes."},
		{Role: "assistant", Content: "Saved the deployment note and linked it to workspace memory."},
	} {
		if err := store.AddFullMessage(nil, sessionKey, msg); err != nil {
			t.Fatalf("AddFullMessage() error = %v", err)
		}
	}
	if err := store.SetSummary(nil, sessionKey, "Graph session"); err != nil {
		t.Fatalf("SetSummary() error = %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(memoryDir, "MEMORY.md"),
		[]byte("# Preferences\n- User prefers network graph views\n# Projects\n- PicoClaw cockpit integration"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(MEMORY.md) error = %v", err)
	}

	todayPath := filepath.Join(memoryDir, time.Now().Format("200601"), time.Now().Format("20060102")+".md")
	if err := os.WriteFile(
		todayPath,
		[]byte("# Daily\n- Reviewed active session memory graph"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(todayPath) error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/pico/memory-graph?session_id=graph-session", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp picoMemoryGraphResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if resp.SessionID != "graph-session" {
		t.Fatalf("resp.SessionID = %q, want %q", resp.SessionID, "graph-session")
	}
	if len(resp.Nodes) < 5 {
		t.Fatalf("len(resp.Nodes) = %d, want at least 5", len(resp.Nodes))
	}

	foundMemoryRoot := false
	foundSessionMessage := false
	for _, node := range resp.Nodes {
		if node.ID == "memory-root" {
			foundMemoryRoot = true
		}
		if strings.Contains(node.Label, "NAIROBI") || strings.Contains(node.Preview, "Nairobi") {
			foundSessionMessage = true
		}
	}
	if !foundMemoryRoot {
		t.Fatal("expected memory-root node in graph")
	}
	if !foundSessionMessage {
		t.Fatal("expected session content to appear in graph nodes")
	}
}

func TestHandleGetPicoSubagents_UsesScopedGatewayStatus(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PICOCLAW_HOME", home)

	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/subagents/status" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/internal/subagents/status")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer gateway-auth-token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer gateway-auth-token")
		}
		if got := r.URL.Query().Get("channel"); got != "pico" {
			t.Fatalf("channel query = %q, want %q", got, "pico")
		}
		if got := r.URL.Query().Get("chat_id"); got != "session-42" {
			t.Fatalf("chat_id query = %q, want %q", got, "session-42")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"channel": "pico",
			"chat_id": "session-42",
			"tasks": []map[string]any{
				{
					"id":      "subagent-1",
					"label":   "Research",
					"status":  "running",
					"created": int64(1710000000000),
					"result":  "This is a very long status summary that should still round-trip through the launcher API cleanly.",
				},
			},
		})
	}))
	defer server.Close()

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	cfg.Gateway.Port = mustGatewayTestPort(t, server.URL)
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := startGatewayLikeProcess(t)
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	origPidData := gateway.pidData
	origCmd := gateway.cmd
	t.Cleanup(func() {
		gateway.mu.Lock()
		gateway.pidData = origPidData
		gateway.cmd = origCmd
		gateway.mu.Unlock()
	})

	gateway.mu.Lock()
	gateway.pidData = &ppid.PidFileData{PID: cmd.Process.Pid, Token: "gateway-auth-token"}
	gateway.cmd = cmd
	gateway.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "http://launcher.local/api/pico/subagents?session_id=session-42", nil)
	rec := httptest.NewRecorder()
	h.handleGetPicoSubagents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp picoSubagentStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.SessionID != "session-42" || resp.ChatID != "session-42" || resp.Channel != "pico" {
		t.Fatalf("response = %#v, want scoped session metadata", resp)
	}
	if len(resp.Tasks) != 1 || resp.Tasks[0].ID != "subagent-1" {
		t.Fatalf("tasks = %#v, want one proxied task", resp.Tasks)
	}
}

func TestHandleWebSocketProxyRejectsStalePidDataAfterProcessExit(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("PICOCLAW_HOME", filepath.Join(tmpDir, ".picoclaw"))

	configPath := filepath.Join(tmpDir, "config.json")
	h := NewHandler(configPath)
	handler := h.handleWebSocketProxy()

	cfg := config.DefaultConfig()
	bc := cfg.Channels["pico"]
	bc.Enabled = true
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	decoded.(*config.PicoSettings).SetToken("ui-token")
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

	req := newPicoProxyRequest(http.MethodGet, "/pico/ws?session_id=test-session")
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

func TestHandleWebSocketProxy_AllowsArbitraryOrigin(t *testing.T) {
	origMatcher := gatewayProcessMatcher
	gatewayProcessMatcher = func(int) (bool, bool) { return true, true }
	t.Cleanup(func() { gatewayProcessMatcher = origMatcher })

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
	bc := cfg.Channels["pico"]
	bc.Enabled = true
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	decoded.(*config.PicoSettings).SetToken("ui-token")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cmd := startGatewayLikeProcess(t)
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})
	writeTestPidFile(t, ppid.PidFileData{
		PID:   cmd.Process.Pid,
		Token: "test-token",
		Host:  cfg.Gateway.Host,
		Port:  cfg.Gateway.Port,
	})
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
	gateway.picoToken = "ui-token"

	req := httptest.NewRequest(http.MethodGet, "http://launcher.local/pico/ws?session_id=test-session", nil)
	req.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
