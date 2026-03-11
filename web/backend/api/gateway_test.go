package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestGatewayStartReady_NoDefaultModel(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)

	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		t.Fatalf("gatewayStartReady() error = %v", err)
	}
	if ready {
		t.Fatalf("gatewayStartReady() ready = true, want false")
	}
	if reason != "no default model configured" {
		t.Fatalf("gatewayStartReady() reason = %q, want %q", reason, "no default model configured")
	}
}

func TestGatewayStartReady_InvalidDefaultModel(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "missing-model"
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		t.Fatalf("gatewayStartReady() error = %v", err)
	}
	if ready {
		t.Fatalf("gatewayStartReady() ready = true, want false")
	}
	if reason == "" {
		t.Fatalf("gatewayStartReady() reason is empty")
	}
}

func TestGatewayStartReady_ValidDefaultModel(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = cfg.ModelList[0].ModelName
	cfg.ModelList[0].APIKey = "test-key"
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		t.Fatalf("gatewayStartReady() error = %v", err)
	}
	if !ready {
		t.Fatalf("gatewayStartReady() ready = false, want true (reason=%q)", reason)
	}
}

func TestGatewayStartReady_DefaultModelWithoutCredential(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = cfg.ModelList[0].ModelName
	cfg.ModelList[0].APIKey = ""
	cfg.ModelList[0].AuthMethod = ""
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		t.Fatalf("gatewayStartReady() error = %v", err)
	}
	if ready {
		t.Fatalf("gatewayStartReady() ready = true, want false")
	}
	if !strings.Contains(reason, "no credentials configured") {
		t.Fatalf("gatewayStartReady() reason = %q, want contains %q", reason, "no credentials configured")
	}
}

func TestGatewayStatusIncludesStartConditionWhenNotReady(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/gateway/status", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	allowed, ok := body["gateway_start_allowed"].(bool)
	if !ok {
		t.Fatalf("gateway_start_allowed missing or not bool: %#v", body["gateway_start_allowed"])
	}
	if allowed {
		t.Fatalf("gateway_start_allowed = true, want false")
	}
	if _, ok := body["gateway_start_reason"].(string); !ok {
		t.Fatalf("gateway_start_reason missing or not string: %#v", body["gateway_start_reason"])
	}
}

func TestFindPicoclawBinary_EnvOverride(t *testing.T) {
	// Create a temporary file to act as the mock binary
	tmpDir := t.TempDir()
	mockBinary := filepath.Join(tmpDir, "picoclaw-mock")
	if err := os.WriteFile(mockBinary, []byte("mock"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PICOCLAW_BINARY", mockBinary)

	got := findPicoclawBinary()
	if got != mockBinary {
		t.Errorf("findPicoclawBinary() = %q, want %q", got, mockBinary)
	}
}

func TestFindPicoclawBinary_EnvOverride_InvalidPath(t *testing.T) {
	// When PICOCLAW_BINARY points to a non-existent path, fall through to next strategy
	t.Setenv("PICOCLAW_BINARY", "/nonexistent/picoclaw-binary")

	got := findPicoclawBinary()
	// Should not return the invalid path; falls back to "picoclaw" or another found path
	if got == "/nonexistent/picoclaw-binary" {
		t.Errorf("findPicoclawBinary() returned invalid env path %q, expected fallback", got)
	}
}
