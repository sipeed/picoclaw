package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestSaveWeixinBindingReturnsSuccessWhenRestartFails(t *testing.T) {
	resetGatewayTestState(t)

	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	originalHealthGet := gatewayHealthGet
	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{"status":"ok","uptime":"1s","pid":` + strconv.Itoa(os.Getpid()) + `}`,
			)),
		}, nil
	}
	t.Cleanup(func() {
		gatewayHealthGet = originalHealthGet
	})

	h := NewHandler(configPath)
	if err := h.saveWeixinBinding("weixin", "bot-token", "bot-account"); err != nil {
		t.Fatalf("saveWeixinBinding() error = %v, want nil after config save succeeds", err)
	}

	savedCfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	bc := savedCfg.Channels["weixin"]
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	wxCfg := decoded.(*config.WeixinSettings)
	if got := wxCfg.Token.String(); got != "bot-token" {
		t.Fatalf("Weixin.Token() = %q, want %q", got, "bot-token")
	}
	if got := wxCfg.AccountID; got != "bot-account" {
		t.Fatalf("Weixin.AccountID = %q, want %q", got, "bot-account")
	}
	if !bc.Enabled {
		t.Fatalf("Weixin.Enabled = false, want true")
	}
}

func TestSaveWeixinBindingSavesNamedChannel(t *testing.T) {
	resetGatewayTestState(t)

	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	if err := h.saveWeixinBinding("weixin_work", "bot-token", "bot-account"); err != nil {
		t.Fatalf("saveWeixinBinding() error = %v", err)
	}

	savedCfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	bc := savedCfg.Channels["weixin_work"]
	if bc == nil {
		t.Fatal("expected weixin_work channel to be created")
	}
	if bc.Type != config.ChannelWeixin {
		t.Fatalf("channel type = %q, want %q", bc.Type, config.ChannelWeixin)
	}
	decoded, err := bc.GetDecoded()
	if err != nil {
		t.Fatalf("GetDecoded() error = %v", err)
	}
	wxCfg := decoded.(*config.WeixinSettings)
	if got := wxCfg.Token.String(); got != "bot-token" {
		t.Fatalf("Weixin.Token() = %q, want %q", got, "bot-token")
	}
	if got := wxCfg.AccountID; got != "bot-account" {
		t.Fatalf("Weixin.AccountID = %q, want %q", got, "bot-account")
	}
}

func TestHandleStartWeixinFlowRejectsInvalidChannel(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/weixin/flows",
		bytes.NewBufferString(`{"channel":"bad name"}`),
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandleStartWeixinFlowRejectsConflictingChannelType(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Channels["weixin_conflict"] = &config.Channel{Enabled: true, Type: config.ChannelTelegram}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/weixin/flows",
		bytes.NewBufferString(`{"channel":"weixin_conflict"}`),
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestParseWeixinFlowChannelDefaultsToWeixin(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodPost, "/api/weixin/flows", nil)
	channel, err := h.parseWeixinFlowChannel(req)
	if err != nil {
		t.Fatalf("parseWeixinFlowChannel() error = %v", err)
	}
	if channel != config.ChannelWeixin {
		t.Fatalf("channel = %q, want %q", channel, config.ChannelWeixin)
	}
}

func TestParseWeixinFlowChannelReadsBody(t *testing.T) {
	h := &Handler{}

	body, err := json.Marshal(map[string]string{"channel": "weixin_work"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/weixin/flows", bytes.NewReader(body))
	channel, err := h.parseWeixinFlowChannel(req)
	if err != nil {
		t.Fatalf("parseWeixinFlowChannel() error = %v", err)
	}
	if channel != "weixin_work" {
		t.Fatalf("channel = %q, want weixin_work", channel)
	}
}
