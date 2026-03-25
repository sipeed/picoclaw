package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestHandleAddModel_PersistsAPIKey(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(`{
		"model_name":"new-model",
		"model":"openai/gpt-4o-mini",
		"api_key":"sk-new-model-key"
	}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.ModelList) != 2 {
		t.Fatalf("len(model_list) = %d, want 2", len(cfg.ModelList))
	}

	added := cfg.ModelList[1]
	if added.ModelName != "new-model" {
		t.Fatalf("model_name = %q, want %q", added.ModelName, "new-model")
	}
	if added.APIKey() != "sk-new-model-key" {
		t.Fatalf("api_key = %q, want %q", added.APIKey(), "sk-new-model-key")
	}
}

// TestHandleSetDefaultModel_RejectsNonexistentModel tests that setting a non-existent
// model as default returns 404. This covers the case where virtual models (which are
// filtered by SaveConfig) cannot be set as default.
func TestHandleSetDefaultModel_RejectsNonexistentModel(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	// First save a valid config with a primary model
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{
		{ModelName: "gpt-4", Model: "openai/gpt-4o"},
	}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Try to set a non-existent model (like a virtual model name) as default
	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models/default", bytes.NewBufferString(`{
		"model_name": "gpt-4__key_1"
	}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	// Should return 404 because the virtual model doesn't exist in the persisted config
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "not found") {
		t.Fatalf("error message should mention 'not found', got: %s", rec.Body.String())
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "empty key",
			key:  "",
			want: "",
		},
		{
			name: "short key fully masked",
			key:  "abcd",
			want: "****",
		},
		{
			name: "length 8 boundary fully masked",
			key:  "12345678",
			want: "****",
		},
		{
			name: "length 9 boundary shows last 2",
			key:  "123456789",
			want: "123****89",
		},
		{
			name: "length 12 boundary shows last 2",
			key:  "abcdefghijkl",
			want: "abc****kl",
		},
		{
			name: "length 13 boundary shows last 4",
			key:  "abcdefghijklm",
			want: "abc****jklm",
		},
		{
			name: "typical api key",
			key:  "sk-1234567890abcd",
			want: "sk-****abcd",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := maskAPIKey(tc.key)
			if got != tc.want {
				t.Fatalf("maskAPIKey(%q) = %q, want %q", tc.key, got, tc.want)
			}

			if tc.key != "" {
				displayed := strings.Replace(tc.want, "****", "", 1)
				if len(tc.key) <= 8 {
					if displayed != "" {
						t.Fatalf("maskAPIKey(%q) displayed part = %q, want empty", tc.key, displayed)
					}
				} else {
					if len(displayed)*10 > len(tc.key)*6 {
						t.Fatalf(
							"maskAPIKey(%q) displayed length = %d, want at most 60%% of %d",
							tc.key,
							len(displayed),
							len(tc.key),
						)
					}
				}
			}
		})
	}
}
