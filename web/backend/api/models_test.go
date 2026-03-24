package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

func resetModelProbeHooks(t *testing.T) {
	t.Helper()

	origTCPProbe := probeTCPServiceFunc
	origOllamaProbe := probeOllamaModelFunc
	origOpenAIProbe := probeOpenAICompatibleModelFunc
	t.Cleanup(func() {
		probeTCPServiceFunc = origTCPProbe
		probeOllamaModelFunc = origOllamaProbe
		probeOpenAICompatibleModelFunc = origOpenAIProbe
	})
}

func TestHandleListModels_ConfiguredStatusUsesRuntimeProbesForLocalModels(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetOAuthHooks(t)
	resetModelProbeHooks(t)

	var mu sync.Mutex
	var openAIProbes []string
	var ollamaProbes []string
	var tcpProbes []string

	probeOpenAICompatibleModelFunc = func(apiBase, modelID, apiKey string) bool {
		mu.Lock()
		openAIProbes = append(openAIProbes, apiBase+"|"+modelID+"|"+apiKey)
		mu.Unlock()
		return apiBase == "http://127.0.0.1:8000/v1" && modelID == "custom-model" && apiKey == ""
	}
	probeOllamaModelFunc = func(apiBase, modelID string) bool {
		mu.Lock()
		ollamaProbes = append(ollamaProbes, apiBase+"|"+modelID)
		mu.Unlock()
		return apiBase == "http://localhost:11434/v1" && modelID == "llama3"
	}
	probeTCPServiceFunc = func(apiBase string) bool {
		mu.Lock()
		tcpProbes = append(tcpProbes, apiBase)
		mu.Unlock()
		return apiBase == "http://127.0.0.1:4321"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName:  "openai-oauth",
			Model:      "openai/gpt-5.4",
			AuthMethod: "oauth",
		},
		{
			ModelName: "vllm-local",
			Model:     "vllm/custom-model",
			APIBase:   "http://127.0.0.1:8000/v1",
		},
		{
			ModelName: "ollama-default",
			Model:     "ollama/llama3",
		},
		{
			ModelName: "vllm-remote",
			Model:     "vllm/custom-model",
			APIBase:   "https://models.example.com/v1",
		},
		{
			ModelName:  "copilot-gpt-5.4",
			Model:      "github-copilot/gpt-5.4",
			APIBase:    "http://127.0.0.1:4321",
			AuthMethod: "oauth",
		},
	}
	cfg.WithSecurity(&config.SecurityConfig{ModelList: map[string]config.ModelSecurityEntry{
		"vllm-remote": {
			APIKeys: []string{"remote-key"},
		},
	}})
	cfg.Agents.Defaults.ModelName = "openai-oauth"
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Models []modelResponse `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	got := make(map[string]bool, len(resp.Models))
	for _, model := range resp.Models {
		got[model.ModelName] = model.Configured
	}

	if got["openai-oauth"] {
		t.Fatalf("openai oauth model configured = true, want false without stored credential")
	}
	if !got["vllm-local"] {
		t.Fatalf("vllm local model configured = false, want true when local probe succeeds")
	}
	if !got["ollama-default"] {
		t.Fatalf("ollama default model configured = false, want true when default local probe succeeds")
	}
	if !got["vllm-remote"] {
		t.Fatalf("remote vllm model configured = false, want true with api_key")
	}
	if !got["copilot-gpt-5.4"] {
		t.Fatalf("copilot model configured = false, want true when local bridge probe succeeds")
	}
	if len(openAIProbes) != 1 || openAIProbes[0] != "http://127.0.0.1:8000/v1|custom-model|" {
		t.Fatalf("openAI probes = %#v, want only local vllm probe", openAIProbes)
	}
	if len(ollamaProbes) != 1 || ollamaProbes[0] != "http://localhost:11434/v1|llama3" {
		t.Fatalf("ollama probes = %#v, want default local probe", ollamaProbes)
	}
	if len(tcpProbes) != 1 || tcpProbes[0] != "http://127.0.0.1:4321" {
		t.Fatalf("tcp probes = %#v, want only local copilot probe", tcpProbes)
	}
}

func TestHandleListModels_ConfiguredStatusForOAuthModelWithCredential(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetOAuthHooks(t)
	resetModelProbeHooks(t)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName:  "claude-oauth",
		Model:      "anthropic/claude-sonnet-4.6",
		AuthMethod: "oauth",
	}}
	cfg.Agents.Defaults.ModelName = "claude-oauth"
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if err := auth.SetCredential(oauthProviderAnthropic, &auth.AuthCredential{
		AccessToken: "anthropic-token",
		Provider:    oauthProviderAnthropic,
		AuthMethod:  "oauth",
	}); err != nil {
		t.Fatalf("SetCredential() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Models []modelResponse `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(resp.Models))
	}
	if !resp.Models[0].Configured {
		t.Fatalf("oauth model configured = false, want true with stored credential")
	}
}

func TestHandleListModels_ProbesLocalModelsConcurrently(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetOAuthHooks(t)
	resetModelProbeHooks(t)

	started := make(chan string, 2)
	release := make(chan struct{})

	probeOpenAICompatibleModelFunc = func(apiBase, modelID, apiKey string) bool {
		started <- apiBase + "|" + modelID
		<-release
		return true
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "local-vllm-a",
			Model:     "vllm/custom-a",
			APIBase:   "http://127.0.0.1:8000/v1",
		},
		{
			ModelName: "local-vllm-b",
			Model:     "vllm/custom-b",
			APIBase:   "http://127.0.0.1:8001/v1",
		},
	}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	recCh := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
		mux.ServeHTTP(rec, req)
		recCh <- rec
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(200 * time.Millisecond):
			t.Fatal("expected both local probes to start before the first one completed")
		}
	}
	close(release)

	rec := <-recCh
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestHandleListModels_NormalizesWildcardLocalAPIBaseForProbe(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetOAuthHooks(t)
	resetModelProbeHooks(t)

	var gotProbe string
	probeOpenAICompatibleModelFunc = func(apiBase, modelID, apiKey string) bool {
		gotProbe = apiBase + "|" + modelID + "|" + apiKey
		return apiBase == "http://127.0.0.1:8000/v1" && modelID == "custom-model" && apiKey == ""
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "vllm-local",
		Model:     "vllm/custom-model",
		APIBase:   "http://0.0.0.0:8000/v1",
	}}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Models []modelResponse `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(resp.Models))
	}
	if !resp.Models[0].Configured {
		t.Fatal("wildcard-bound local model configured = false, want true after probe host normalization")
	}
	if gotProbe != "http://127.0.0.1:8000/v1|custom-model|" {
		t.Fatalf("probe api base = %q, want %q", gotProbe, "http://127.0.0.1:8000/v1|custom-model|")
	}
}
