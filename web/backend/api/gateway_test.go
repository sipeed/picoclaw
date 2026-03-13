package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/web/backend/utils"
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
	err := config.SaveConfig(configPath, cfg)
	if err != nil {
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
	err := config.SaveConfig(configPath, cfg)
	if err != nil {
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
	err := config.SaveConfig(configPath, cfg)
	if err != nil {
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

func TestGatewayStartReady_LocalModelWithoutAPIKey(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetModelProbeHooks(t)

	probeOpenAICompatibleModelFunc = func(apiBase, modelID string) bool {
		return false
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []config.ModelConfig{{
		ModelName: "local-vllm",
		Model:     "vllm/custom-model",
		APIBase:   "http://localhost:8000/v1",
	}}
	cfg.Agents.Defaults.ModelName = "local-vllm"
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		t.Fatalf("gatewayStartReady() error = %v", err)
	}
	if ready {
		t.Fatalf("gatewayStartReady() ready = true, want false without a running local service")
	}
	if !strings.Contains(reason, "not reachable") {
		t.Fatalf("gatewayStartReady() reason = %q, want contains %q", reason, "not reachable")
	}
}

func TestGatewayStartReady_LocalModelWithRunningService(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetModelProbeHooks(t)

	probeOpenAICompatibleModelFunc = func(apiBase, modelID string) bool {
		return apiBase == "http://127.0.0.1:8000/v1" && modelID == "custom-model"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []config.ModelConfig{{
		ModelName: "local-vllm",
		Model:     "vllm/custom-model",
		APIBase:   "http://127.0.0.1:8000/v1",
	}}
	cfg.Agents.Defaults.ModelName = "local-vllm"
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		t.Fatalf("gatewayStartReady() error = %v", err)
	}
	if !ready {
		t.Fatalf("gatewayStartReady() ready = false, want true with a running local service (reason=%q)", reason)
	}
}

func TestGatewayStartReady_RemoteVLLMWithAPIKeyDoesNotProbe(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetModelProbeHooks(t)

	probeOpenAICompatibleModelFunc = func(apiBase, modelID string) bool {
		t.Fatalf("unexpected OpenAI-compatible probe for %q (%q)", apiBase, modelID)
		return false
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []config.ModelConfig{{
		ModelName: "remote-vllm",
		Model:     "vllm/custom-model",
		APIBase:   "https://models.example.com/v1",
		APIKey:    "remote-key",
	}}
	cfg.Agents.Defaults.ModelName = "remote-vllm"
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		t.Fatalf("gatewayStartReady() error = %v", err)
	}
	if !ready {
		t.Fatalf("gatewayStartReady() ready = false, want true for remote vllm with api key (reason=%q)", reason)
	}
}

func TestGatewayStartReady_LocalOllamaUsesDefaultProbeBase(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetModelProbeHooks(t)

	probeOllamaModelFunc = func(apiBase, modelID string) bool {
		return apiBase == "http://localhost:11434/v1" && modelID == "llama3"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []config.ModelConfig{{
		ModelName: "local-ollama",
		Model:     "ollama/llama3",
	}}
	cfg.Agents.Defaults.ModelName = "local-ollama"
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		t.Fatalf("gatewayStartReady() error = %v", err)
	}
	if !ready {
		t.Fatalf("gatewayStartReady() ready = false, want true with default Ollama probe base (reason=%q)", reason)
	}
}

func TestGatewayStartReady_OAuthModelRequiresStoredCredential(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []config.ModelConfig{{
		ModelName:  "openai-oauth",
		Model:      "openai/gpt-5.4",
		AuthMethod: "oauth",
	}}
	cfg.Agents.Defaults.ModelName = "openai-oauth"
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	ready, reason, err := h.gatewayStartReady()
	if err != nil {
		t.Fatalf("gatewayStartReady() error = %v", err)
	}
	if ready {
		t.Fatalf("gatewayStartReady() ready = true, want false without stored credential")
	}
	if !strings.Contains(reason, "no credentials configured") {
		t.Fatalf("gatewayStartReady() reason = %q, want contains %q", reason, "no credentials configured")
	}

	err = auth.SetCredential(oauthProviderOpenAI, &auth.AuthCredential{
		AccessToken: "openai-token",
		Provider:    oauthProviderOpenAI,
		AuthMethod:  "oauth",
	})
	if err != nil {
		t.Fatalf("SetCredential() error = %v", err)
	}

	ready, reason, err = h.gatewayStartReady()
	if err != nil {
		t.Fatalf("gatewayStartReady() error = %v", err)
	}
	if !ready {
		t.Fatalf("gatewayStartReady() ready = false, want true with stored credential (reason=%q)", reason)
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

func TestGatewayClearLogsResetsBufferedHistory(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	gateway.logs.Clear()
	gateway.logs.Append("first line")
	gateway.logs.Append("second line")
	previousRunID := gateway.logs.RunID()

	clearRec := httptest.NewRecorder()
	clearReq := httptest.NewRequest(http.MethodPost, "/api/gateway/logs/clear", nil)
	mux.ServeHTTP(clearRec, clearReq)

	if clearRec.Code != http.StatusOK {
		t.Fatalf("clear status = %d, want %d", clearRec.Code, http.StatusOK)
	}

	var clearBody map[string]any
	if err := json.Unmarshal(clearRec.Body.Bytes(), &clearBody); err != nil {
		t.Fatalf("unmarshal clear response: %v", err)
	}

	if got := clearBody["status"]; got != "cleared" {
		t.Fatalf("clear status body = %#v, want %q", got, "cleared")
	}

	clearRunID, ok := clearBody["log_run_id"].(float64)
	if !ok {
		t.Fatalf("log_run_id missing or not number: %#v", clearBody["log_run_id"])
	}
	if int(clearRunID) <= previousRunID {
		t.Fatalf("log_run_id = %d, want > %d", int(clearRunID), previousRunID)
	}

	statusRec := httptest.NewRecorder()
	statusReq := httptest.NewRequest(
		http.MethodGet,
		"/api/gateway/status?log_offset=0&log_run_id="+strconv.Itoa(previousRunID),
		nil,
	)
	mux.ServeHTTP(statusRec, statusReq)

	if statusRec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", statusRec.Code, http.StatusOK)
	}

	var statusBody map[string]any
	if err := json.Unmarshal(statusRec.Body.Bytes(), &statusBody); err != nil {
		t.Fatalf("unmarshal status response: %v", err)
	}

	logs, ok := statusBody["logs"].([]any)
	if !ok {
		t.Fatalf("logs missing or not array: %#v", statusBody["logs"])
	}
	if len(logs) != 0 {
		t.Fatalf("logs len = %d, want 0", len(logs))
	}
	if got := statusBody["log_total"]; got != float64(0) {
		t.Fatalf("log_total = %#v, want 0", got)
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

	got := utils.FindPicoclawBinary()
	if got != mockBinary {
		t.Errorf("FindPicoclawBinary() = %q, want %q", got, mockBinary)
	}
}

func TestFindPicoclawBinary_EnvOverride_InvalidPath(t *testing.T) {
	// When PICOCLAW_BINARY points to a non-existent path, fall through to next strategy
	t.Setenv("PICOCLAW_BINARY", "/nonexistent/picoclaw-binary")

	got := utils.FindPicoclawBinary()
	// Should not return the invalid path; falls back to "picoclaw" or another found path
	if got == "/nonexistent/picoclaw-binary" {
		t.Errorf("FindPicoclawBinary() returned invalid env path %q, expected fallback", got)
	}
}
