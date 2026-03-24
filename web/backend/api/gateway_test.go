package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/web/backend/utils"
)

func startLongRunningProcess(t *testing.T) *exec.Cmd {
	t.Helper()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-NoProfile", "-Command", "Start-Sleep -Seconds 30")
	} else {
		cmd = exec.Command("sleep", "30")
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	return cmd
}

func mockGatewayHealthResponse(statusCode, pid int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body: io.NopCloser(strings.NewReader(
			`{"status":"ok","uptime":"1s","pid":` + strconv.Itoa(pid) + `}`,
		)),
	}
}

func startIgnoringTermProcess(t *testing.T) *exec.Cmd {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("TERM handling differs on Windows")
	}

	cmd := exec.Command("sh", "-c", "trap '' TERM; sleep 30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	return cmd
}

func resetGatewayTestState(t *testing.T) {
	t.Helper()

	originalHealthGet := gatewayHealthGet
	originalRestartGracePeriod := gatewayRestartGracePeriod
	originalRestartForceKillWindow := gatewayRestartForceKillWindow
	originalRestartPollInterval := gatewayRestartPollInterval
	t.Cleanup(func() {
		gatewayHealthGet = originalHealthGet
		gatewayRestartGracePeriod = originalRestartGracePeriod
		gatewayRestartForceKillWindow = originalRestartForceKillWindow
		gatewayRestartPollInterval = originalRestartPollInterval

		gateway.mu.Lock()
		gateway.cmd = nil
		gateway.bootDefaultModel = ""
		gateway.bootConfigSignature = ""
		setGatewayRuntimeStatusLocked("stopped")
		gateway.mu.Unlock()
	})
}

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
	cfg.Agents.Defaults.ModelName = "missing-model"
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
	cfg.ModelList[0].SetAPIKey("test-key")
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
	cfg.ModelList[0].SetAPIKey("")
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

	probeOpenAICompatibleModelFunc = func(apiBase, modelID, apiKey string) bool {
		return false
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
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

	probeOpenAICompatibleModelFunc = func(apiBase, modelID, apiKey string) bool {
		return apiBase == "http://127.0.0.1:8000/v1" && modelID == "custom-model" && apiKey == ""
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
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

	probeOpenAICompatibleModelFunc = func(apiBase, modelID, apiKey string) bool {
		t.Fatalf("unexpected OpenAI-compatible probe for %q (%q)", apiBase, modelID)
		return false
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "remote-vllm",
		Model:     "vllm/custom-model",
		APIBase:   "https://models.example.com/v1",
	}}
	cfg.ModelList[0o0].SetAPIKey("remote-key")
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
	cfg.ModelList = []*config.ModelConfig{{
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
	cfg.ModelList = []*config.ModelConfig{{
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

func TestGatewayStatusKeepsRunningWhenHealthProbeFailsAfterRunning(t *testing.T) {
	resetGatewayTestState(t)

	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	cmd := startLongRunningProcess(t)
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	gateway.mu.Lock()
	gateway.cmd = cmd
	gateway.bootDefaultModel = "existing-model"
	// Simulate a process that has already reached the running state.
	setGatewayRuntimeStatusLocked("running")
	gateway.mu.Unlock()

	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		return nil, errors.New("probe failed")
	}

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

	if got := body["gateway_status"]; got != "running" {
		t.Fatalf("gateway_status = %#v, want %q", got, "running")
	}
}

func TestGatewayStatusReportsRunningFromHealthProbe(t *testing.T) {
	resetGatewayTestState(t)

	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	cmd := startLongRunningProcess(t)
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	gateway.mu.Lock()
	setGatewayRuntimeStatusLocked("stopped")
	gateway.mu.Unlock()

	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		return mockGatewayHealthResponse(http.StatusOK, cmd.Process.Pid), nil
	}

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

	if got := body["gateway_status"]; got != "running" {
		t.Fatalf("gateway_status = %#v, want %q", got, "running")
	}
	if got := body["pid"]; got != float64(cmd.Process.Pid) {
		t.Fatalf("pid = %#v, want %d", got, cmd.Process.Pid)
	}
	if got := body["gateway_restart_required"]; got != false {
		t.Fatalf("gateway_restart_required = %#v, want false", got)
	}
}

func TestGatewayStatusRequiresRestartAfterDefaultModelChange(t *testing.T) {
	resetGatewayTestState(t)

	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = cfg.ModelList[0].ModelName
	cfg.ModelList[0].SetAPIKey("test-key")
	cfg.ModelList = append(cfg.ModelList, &config.ModelConfig{
		ModelName: "second-model",
		Model:     "openai/gpt-4.1",
	})
	cfg.ModelList[len(cfg.ModelList)-1].SetAPIKey("second-key")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess() error = %v", err)
	}

	bootSignature := computeConfigSignature(cfg)
	gateway.mu.Lock()
	gateway.cmd = &exec.Cmd{Process: process}
	gateway.bootDefaultModel = cfg.ModelList[0].ModelName
	gateway.bootConfigSignature = bootSignature
	setGatewayRuntimeStatusLocked("running")
	gateway.mu.Unlock()

	updatedCfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	updatedCfg.Agents.Defaults.ModelName = "second-model"
	if err := config.SaveConfig(configPath, updatedCfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		return mockGatewayHealthResponse(http.StatusOK, os.Getpid()), nil
	}

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

	if got := body["gateway_status"]; got != "running" {
		t.Fatalf("gateway_status = %#v, want %q", got, "running")
	}
	if got := body["boot_default_model"]; got != cfg.ModelList[0].ModelName {
		t.Fatalf("boot_default_model = %#v, want %q", got, cfg.ModelList[0].ModelName)
	}
	if got := body["config_default_model"]; got != "second-model" {
		t.Fatalf("config_default_model = %#v, want %q", got, "second-model")
	}
	if got := body["gateway_restart_required"]; got != true {
		t.Fatalf("gateway_restart_required = %#v, want true", got)
	}
}

func TestGatewayStatusRequiresRestartAfterToolChange(t *testing.T) {
	resetGatewayTestState(t)

	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = cfg.ModelList[0].ModelName
	cfg.ModelList[0].SetAPIKey("test-key")
	cfg.Tools.WriteFile.Enabled = true
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess() error = %v", err)
	}

	bootSignature := computeConfigSignature(cfg)
	gateway.mu.Lock()
	gateway.cmd = &exec.Cmd{Process: process}
	gateway.bootDefaultModel = cfg.ModelList[0].ModelName
	gateway.bootConfigSignature = bootSignature
	setGatewayRuntimeStatusLocked("running")
	gateway.mu.Unlock()

	updatedCfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	updatedCfg.Tools.WriteFile.Enabled = false
	if err := config.SaveConfig(configPath, updatedCfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		return mockGatewayHealthResponse(http.StatusOK, os.Getpid()), nil
	}

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

	if got := body["gateway_status"]; got != "running" {
		t.Fatalf("gateway_status = %#v, want %q", got, "running")
	}
	if got := body["gateway_restart_required"]; got != true {
		t.Fatalf("gateway_restart_required = %#v, want true", got)
	}
}

func TestGatewayStatusNoRestartRequiredForNonSensitiveChanges(t *testing.T) {
	resetGatewayTestState(t)

	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = cfg.ModelList[0].ModelName
	cfg.ModelList[0].SetAPIKey("test-key")
	cfg.Agents.Defaults.MaxTokens = 1000
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess() error = %v", err)
	}

	bootSignature := computeConfigSignature(cfg)
	gateway.mu.Lock()
	gateway.cmd = &exec.Cmd{Process: process}
	gateway.bootDefaultModel = cfg.ModelList[0].ModelName
	gateway.bootConfigSignature = bootSignature
	setGatewayRuntimeStatusLocked("running")
	gateway.mu.Unlock()

	updatedCfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	updatedCfg.Agents.Defaults.MaxTokens = 2000
	if err := config.SaveConfig(configPath, updatedCfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		return mockGatewayHealthResponse(http.StatusOK, os.Getpid()), nil
	}

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

	if got := body["gateway_status"]; got != "running" {
		t.Fatalf("gateway_status = %#v, want %q", got, "running")
	}
	if got := body["gateway_restart_required"]; got != false {
		t.Fatalf("gateway_restart_required = %#v, want false", got)
	}
}

func TestGatewayStatusNoRestartRequiredWhenNotRunning(t *testing.T) {
	resetGatewayTestState(t)

	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = cfg.ModelList[0].ModelName
	cfg.ModelList[0].SetAPIKey("test-key")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	gateway.mu.Lock()
	gateway.cmd = nil
	gateway.bootDefaultModel = ""
	gateway.bootConfigSignature = ""
	setGatewayRuntimeStatusLocked("stopped")
	gateway.mu.Unlock()

	updatedCfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	updatedCfg.Agents.Defaults.ModelName = "different-model"
	if err := config.SaveConfig(configPath, updatedCfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		return nil, errors.New("no gateway running")
	}

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

	if got := body["gateway_status"]; got != "stopped" {
		t.Fatalf("gateway_status = %#v, want %q", got, "stopped")
	}
	if got := body["gateway_restart_required"]; got != false {
		t.Fatalf("gateway_restart_required = %#v, want false", got)
	}
}

func TestGatewayStatusReturnsErrorAfterStartupWindowExpires(t *testing.T) {
	resetGatewayTestState(t)

	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	cmd := startLongRunningProcess(t)
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	gateway.mu.Lock()
	gateway.cmd = cmd
	gateway.bootDefaultModel = "existing-model"
	setGatewayRuntimeStatusLocked("starting")
	gateway.startupDeadline = time.Now().Add(-time.Second)
	gateway.mu.Unlock()

	gatewayHealthGet = func(string, time.Duration) (*http.Response, error) {
		return nil, errors.New("probe failed")
	}

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

	if got := body["gateway_status"]; got != "error" {
		t.Fatalf("gateway_status = %#v, want %q", got, "error")
	}
}

func TestGatewayStatusReturnsRestartingDuringRestartGap(t *testing.T) {
	resetGatewayTestState(t)

	// Mock health check to return error, so it won't override our "restarting" status
	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		return nil, errors.New("mock health check error")
	}

	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	gateway.mu.Lock()
	setGatewayRuntimeStatusLocked("restarting")
	gateway.mu.Unlock()

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

	if got := body["gateway_status"]; got != "restarting" {
		t.Fatalf("gateway_status = %#v, want %q", got, "restarting")
	}
}

func TestGatewayRestartKeepsRunningProcessWhenPreconditionsFail(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = cfg.ModelList[0].ModelName
	cfg.ModelList[0].SetAPIKey("")
	cfg.ModelList[0].AuthMethod = ""
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	cmd := startLongRunningProcess(t)
	t.Cleanup(func() {
		gateway.mu.Lock()
		if gateway.cmd == cmd {
			gateway.cmd = nil
			gateway.bootDefaultModel = ""
		}
		gateway.mu.Unlock()

		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	gateway.mu.Lock()
	gateway.cmd = cmd
	gateway.bootDefaultModel = "existing-model"
	gateway.mu.Unlock()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/gateway/restart", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	gateway.mu.Lock()
	stillRunning := gateway.cmd == cmd && isCmdProcessAliveLocked(cmd)
	gateway.mu.Unlock()

	if !stillRunning {
		t.Fatalf("gateway process was stopped when restart preconditions failed")
	}
}

func TestGatewayRestartKeepsOldProcessWhenItDoesNotExitInTime(t *testing.T) {
	resetGatewayTestState(t)

	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = cfg.ModelList[0].ModelName
	cfg.ModelList[0].SetAPIKey("test-key")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	cmd := startIgnoringTermProcess(t)
	t.Cleanup(func() {
		gateway.mu.Lock()
		if gateway.cmd == cmd {
			gateway.cmd = nil
			gateway.bootDefaultModel = ""
		}
		gateway.mu.Unlock()

		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	gatewayRestartGracePeriod = 150 * time.Millisecond
	gatewayRestartForceKillWindow = 150 * time.Millisecond
	gatewayRestartPollInterval = 10 * time.Millisecond

	gateway.mu.Lock()
	gateway.cmd = cmd
	gateway.bootDefaultModel = "existing-model"
	setGatewayRuntimeStatusLocked("running")
	gateway.mu.Unlock()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/gateway/restart", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	gateway.mu.Lock()
	stillRunning := gateway.cmd == cmd && isCmdProcessAliveLocked(cmd)
	status := gateway.runtimeStatus
	gateway.mu.Unlock()

	if !stillRunning {
		t.Fatalf("gateway process was replaced before the old process exited")
	}
	if status != "running" {
		t.Fatalf("runtimeStatus = %q, want %q", status, "running")
	}
}

func TestGatewayRestartReturnsErrorStatusWhenReplacementFailsToStart(t *testing.T) {
	resetGatewayTestState(t)

	// Mock health check to return error, so it won't override our "error" status
	gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
		return nil, errors.New("mock health check error")
	}

	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = cfg.ModelList[0].ModelName
	cfg.ModelList[0].SetAPIKey("test-key")
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	invalidBinaryPath := filepath.Join(t.TempDir(), "fake-picoclaw")
	if err := os.WriteFile(invalidBinaryPath, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("PICOCLAW_BINARY", invalidBinaryPath)

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/gateway/restart", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("restart status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	statusRec := httptest.NewRecorder()
	statusReq := httptest.NewRequest(http.MethodGet, "/api/gateway/status", nil)
	mux.ServeHTTP(statusRec, statusReq)

	if statusRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", statusRec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(statusRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if got := body["gateway_status"]; got != "error" {
		t.Fatalf("gateway_status = %#v, want %q", got, "error")
	}
}

func TestGatewayStatusExcludesLogsFields(t *testing.T) {
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

	if _, ok := body["logs"]; ok {
		t.Fatalf("logs unexpectedly present in status response: %#v", body["logs"])
	}
	if _, ok := body["log_total"]; ok {
		t.Fatalf("log_total unexpectedly present in status response: %#v", body["log_total"])
	}
	if _, ok := body["log_run_id"]; ok {
		t.Fatalf("log_run_id unexpectedly present in status response: %#v", body["log_run_id"])
	}
}

func TestGatewayLogsReturnsIncrementalHistory(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	gateway.logs.Clear()
	gateway.logs.Append("first line")
	gateway.logs.Append("second line")
	runID := gateway.logs.RunID()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/gateway/logs?log_offset=1&log_run_id="+strconv.Itoa(runID),
		nil,
	)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("logs status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal logs response: %v", err)
	}

	logs, ok := body["logs"].([]any)
	if !ok {
		t.Fatalf("logs missing or not array: %#v", body["logs"])
	}
	if len(logs) != 1 || logs[0] != "second line" {
		t.Fatalf("logs = %#v, want [\"second line\"]", logs)
	}
	if got := body["log_total"]; got != float64(2) {
		t.Fatalf("log_total = %#v, want 2", got)
	}
	if got := body["log_run_id"]; got != float64(runID) {
		t.Fatalf("log_run_id = %#v, want %d", got, runID)
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

	logsRec := httptest.NewRecorder()
	logsReq := httptest.NewRequest(
		http.MethodGet,
		"/api/gateway/logs?log_offset=0&log_run_id="+strconv.Itoa(previousRunID),
		nil,
	)
	mux.ServeHTTP(logsRec, logsReq)

	if logsRec.Code != http.StatusOK {
		t.Fatalf("logs code = %d, want %d", logsRec.Code, http.StatusOK)
	}

	var logsBody map[string]any
	if err := json.Unmarshal(logsRec.Body.Bytes(), &logsBody); err != nil {
		t.Fatalf("unmarshal logs response: %v", err)
	}

	logs, ok := logsBody["logs"].([]any)
	if !ok {
		t.Fatalf("logs missing or not array: %#v", logsBody["logs"])
	}
	if len(logs) != 0 {
		t.Fatalf("logs len = %d, want 0", len(logs))
	}
	if got := logsBody["log_total"]; got != float64(0) {
		t.Fatalf("log_total = %#v, want 0", got)
	}
	if got := logsBody["log_run_id"]; got != clearBody["log_run_id"] {
		t.Fatalf("log_run_id = %#v, want %#v", got, clearBody["log_run_id"])
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
