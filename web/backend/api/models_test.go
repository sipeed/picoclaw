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
	"github.com/sipeed/picoclaw/pkg/providers"
)

func resetModelProbeHooks(t *testing.T) {
	t.Helper()

	origTCPProbe := probeTCPServiceFunc
	origOllamaProbe := probeOllamaModelFunc
	origOpenAIProbe := probeOpenAICompatibleModelFunc
	origCommandProbe := probeCommandAvailableFunc
	origNow := modelProbeNowFunc
	resetModelProbeCache()
	t.Cleanup(func() {
		probeTCPServiceFunc = origTCPProbe
		probeOllamaModelFunc = origOllamaProbe
		probeOpenAICompatibleModelFunc = origOpenAIProbe
		probeCommandAvailableFunc = origCommandProbe
		modelProbeNowFunc = origNow
		resetModelProbeCache()
	})
}

func addModelAndLoadLatest(t *testing.T, configPath string, body string) *config.ModelConfig {
	t.Helper()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.ModelList) == 0 {
		t.Fatal("model_list should contain the newly added model")
	}

	return cfg.ModelList[len(cfg.ModelList)-1]
}

func TestHandleListModels_AvailabilityUsesRuntimeProbesForLocalModels(t *testing.T) {
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
			APIKeys:   config.SimpleSecureStrings("remote-key"),
		},
		{
			ModelName:  "copilot-gpt-5.4",
			Model:      "github-copilot/gpt-5.4",
			APIBase:    "http://127.0.0.1:4321",
			AuthMethod: "oauth",
		},
	}
	cfg.Agents.Defaults.ModelName = "openai-oauth"
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
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
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	gotAvailable := make(map[string]bool, len(resp.Models))
	gotStatus := make(map[string]string, len(resp.Models))
	for _, model := range resp.Models {
		gotAvailable[model.ModelName] = model.Available
		gotStatus[model.ModelName] = model.Status
	}

	if gotAvailable["openai-oauth"] {
		t.Fatalf("openai oauth model available = true, want false without stored credential")
	}
	if !gotAvailable["vllm-local"] {
		t.Fatalf("vllm local model available = false, want true when local probe succeeds")
	}
	if !gotAvailable["ollama-default"] {
		t.Fatalf("ollama default model available = false, want true when default local probe succeeds")
	}
	if !gotAvailable["vllm-remote"] {
		t.Fatalf("remote vllm model available = false, want true with api_key")
	}
	if !gotAvailable["copilot-gpt-5.4"] {
		t.Fatalf("copilot model available = false, want true when local bridge probe succeeds")
	}
	if gotStatus["openai-oauth"] != modelStatusUnconfigured {
		t.Fatalf("openai oauth model status = %q, want %q", gotStatus["openai-oauth"], modelStatusUnconfigured)
	}
	if gotStatus["vllm-local"] != modelStatusAvailable {
		t.Fatalf("vllm local model status = %q, want %q", gotStatus["vllm-local"], modelStatusAvailable)
	}
	if gotStatus["ollama-default"] != modelStatusAvailable {
		t.Fatalf("ollama default model status = %q, want %q", gotStatus["ollama-default"], modelStatusAvailable)
	}
	if gotStatus["vllm-remote"] != modelStatusAvailable {
		t.Fatalf("remote vllm model status = %q, want %q", gotStatus["vllm-remote"], modelStatusAvailable)
	}
	if gotStatus["copilot-gpt-5.4"] != modelStatusAvailable {
		t.Fatalf("copilot model status = %q, want %q", gotStatus["copilot-gpt-5.4"], modelStatusAvailable)
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

func TestHandleListModels_AvailabilityForOAuthModelWithCredential(t *testing.T) {
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
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if setCredentialErr := auth.SetCredential(oauthProviderAnthropic, &auth.AuthCredential{
		AccessToken: "anthropic-token",
		Provider:    oauthProviderAnthropic,
		AuthMethod:  "oauth",
	}); setCredentialErr != nil {
		t.Fatalf("SetCredential() error = %v", setCredentialErr)
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
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(resp.Models))
	}
	if !resp.Models[0].Available {
		t.Fatalf("oauth model available = false, want true with stored credential")
	}
}

func TestHasModelConfiguration_OAuthWithoutMappedCredentialFallsBackToAPIKey(t *testing.T) {
	noKey := &config.ModelConfig{
		Provider:   "gemini",
		Model:      "gemini-2.5-flash",
		AuthMethod: "oauth",
	}
	if hasModelConfiguration(noKey) {
		t.Fatal("oauth model without credential mapping and api key should be unconfigured")
	}

	withKey := &config.ModelConfig{
		Provider:   "gemini",
		Model:      "gemini-2.5-flash",
		AuthMethod: "oauth",
		APIKeys:    config.SimpleSecureStrings("gemini-key"),
	}
	if !hasModelConfiguration(withKey) {
		t.Fatal("oauth model without credential mapping should fall back to api key configuration")
	}
}

func TestHandleListModels_AntigravityImplicitOAuthAvailability(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetOAuthHooks(t)
	resetModelProbeHooks(t)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "gemini-flash",
		Provider:  "antigravity",
		Model:     "gemini-3-flash",
	}}
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if err := auth.SetCredential(oauthProviderGoogleAntigravity, &auth.AuthCredential{
		AccessToken: "antigravity-token",
		Provider:    oauthProviderGoogleAntigravity,
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
	if unmarshalErr := json.Unmarshal(rec.Body.Bytes(), &resp); unmarshalErr != nil {
		t.Fatalf("Unmarshal() error = %v", unmarshalErr)
	}
	if len(resp.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(resp.Models))
	}
	if !resp.Models[0].Available {
		t.Fatal("antigravity model available = false, want true with stored credential even without auth_method")
	}
}

func TestHandleListModels_BedrockUsesAmbientCredentialStatus(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetOAuthHooks(t)
	resetModelProbeHooks(t)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "bedrock-claude",
		Provider:  "bedrock",
		Model:     "us.anthropic.claude-sonnet-4-20250514-v1:0",
	}}
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
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
	if unmarshalErr := json.Unmarshal(rec.Body.Bytes(), &resp); unmarshalErr != nil {
		t.Fatalf("Unmarshal() error = %v", unmarshalErr)
	}
	if len(resp.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(resp.Models))
	}
	if !resp.Models[0].Available {
		t.Fatal("bedrock model available = false, want true because Bedrock uses ambient AWS credentials")
	}
	if resp.Models[0].Status != modelStatusAvailable {
		t.Fatalf("bedrock model status = %q, want %q", resp.Models[0].Status, modelStatusAvailable)
	}
}

func TestHandleListModels_CLIProvidersRequireInstalledCommands(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetOAuthHooks(t)
	resetModelProbeHooks(t)

	probeCommandAvailableFunc = func(command string) bool {
		switch command {
		case "claude":
			return false
		case "codex":
			return true
		default:
			return false
		}
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "claude-cli-model",
			Provider:  "claude-cli",
			Model:     "claude-cli",
		},
		{
			ModelName: "codex-cli-model",
			Provider:  "codex-cli",
			Model:     "codex-cli",
		},
	}
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
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
		Models          []modelResponse                 `json:"models"`
		ProviderOptions []providers.ModelProviderOption `json:"provider_options"`
	}
	if unmarshalErr := json.Unmarshal(rec.Body.Bytes(), &resp); unmarshalErr != nil {
		t.Fatalf("Unmarshal() error = %v", unmarshalErr)
	}

	modelsByName := make(map[string]modelResponse, len(resp.Models))
	for _, model := range resp.Models {
		modelsByName[model.ModelName] = model
	}
	if model := modelsByName["claude-cli-model"]; model.Available || model.Status != modelStatusUnreachable {
		t.Fatalf(
			"claude-cli status = (%t, %q), want (%t, %q)",
			model.Available,
			model.Status,
			false,
			modelStatusUnreachable,
		)
	}
	if model := modelsByName["codex-cli-model"]; !model.Available || model.Status != modelStatusAvailable {
		t.Fatalf(
			"codex-cli status = (%t, %q), want (%t, %q)",
			model.Available,
			model.Status,
			true,
			modelStatusAvailable,
		)
	}

	optionsByID := make(map[string]providers.ModelProviderOption, len(resp.ProviderOptions))
	for _, option := range resp.ProviderOptions {
		optionsByID[option.ID] = option
	}
	if option, ok := optionsByID["claude-cli"]; !ok {
		t.Fatal("claude-cli provider option missing")
	} else if option.CreateAllowed {
		t.Fatal("claude-cli should not be creatable when the claude command is missing")
	}
	if option, ok := optionsByID["codex-cli"]; !ok {
		t.Fatal("codex-cli provider option missing")
	} else if !option.CreateAllowed {
		t.Fatal("codex-cli should be creatable when the codex command is available")
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
	if unmarshalErr := json.Unmarshal(rec.Body.Bytes(), &resp); unmarshalErr != nil {
		t.Fatalf("Unmarshal() error = %v", unmarshalErr)
	}
	if len(resp.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(resp.Models))
	}
	if !resp.Models[0].Available {
		t.Fatal("wildcard-bound local model available = false, want true after probe host normalization")
	}
	if gotProbe != "http://127.0.0.1:8000/v1|custom-model|" {
		t.Fatalf("probe api base = %q, want %q", gotProbe, "http://127.0.0.1:8000/v1|custom-model|")
	}
}

func TestHandleListModels_StatusMarksUnreachableLocalModel(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetOAuthHooks(t)
	resetModelProbeHooks(t)

	probeOpenAICompatibleModelFunc = func(apiBase, modelID, apiKey string) bool {
		return false
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "vllm-local-down",
		Model:     "vllm/custom-model",
		APIBase:   "http://127.0.0.1:8000/v1",
		APIKeys:   config.SimpleSecureStrings("test-key"),
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

	if resp.Models[0].Available {
		t.Fatal("unreachable local model available = true, want false")
	}
	if resp.Models[0].Status != modelStatusUnreachable {
		t.Fatalf("unreachable local model status = %q, want %q", resp.Models[0].Status, modelStatusUnreachable)
	}
	if resp.Models[0].APIKey == "" {
		t.Fatal("masked API key preview should still be returned when API key is configured")
	}
}

func TestHandleListModels_RuntimeProbeUsesExplicitProviderField(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetOAuthHooks(t)
	resetModelProbeHooks(t)

	var gotProbe string
	probeOpenAICompatibleModelFunc = func(apiBase, modelID, apiKey string) bool {
		gotProbe = apiBase + "|" + modelID + "|" + apiKey
		return true
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "vllm-local",
		Provider:  "vllm",
		Model:     "custom-model",
		APIBase:   "http://127.0.0.1:8000/v1",
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

	if gotProbe != "http://127.0.0.1:8000/v1|custom-model|" {
		t.Fatalf("probe = %q, want %q", gotProbe, "http://127.0.0.1:8000/v1|custom-model|")
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

func TestHandleAddModel_PersistsProvider(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(`{
		"model_name":"nvidia-glm",
		"provider":"nvidia",
		"model":"z-ai/glm-5.1",
		"api_key":"nv-key"
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
	added := cfg.ModelList[len(cfg.ModelList)-1]
	if added.Provider != "nvidia" {
		t.Fatalf("provider = %q, want %q", added.Provider, "nvidia")
	}
	if added.Model != "z-ai/glm-5.1" {
		t.Fatalf("model = %q, want %q", added.Model, "z-ai/glm-5.1")
	}
}

func TestHandleAddModel_RejectsUnsupportedProvider(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(`{
		"model_name":"bad-provider",
		"provider":"not-supported",
		"model":"gpt-4o-mini"
	}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `provider "not-supported" is not supported`) {
		t.Fatalf("body = %q, want unsupported provider error", rec.Body.String())
	}
}

func TestHandleAddModel_AllowsBedrockProvider(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(`{
		"model_name":"bedrock-claude",
		"provider":"bedrock",
		"model":"us.anthropic.claude-sonnet-4-20250514-v1:0"
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
	added := cfg.ModelList[len(cfg.ModelList)-1]
	if got := added.Provider; got != "bedrock" {
		t.Fatalf("provider = %q, want %q", got, "bedrock")
	}
	if got := added.Model; got != "us.anthropic.claude-sonnet-4-20250514-v1:0" {
		t.Fatalf("model = %q, want bedrock model ID", got)
	}
}

func TestHandleAddModel_NormalizesLegacyElevenLabsASRConfig(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "elevenlabs-asr",
		Model:     "elevenlabs/scribe_v1",
		APIKeys:   config.SimpleSecureStrings("sk_elevenlabs_test"),
	}}
	if err = config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(`{
		"model_name":"new-model",
		"provider":"openai",
		"model":"gpt-4o-mini",
		"api_key":"sk-new-model-key"
	}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	updated, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(updated.ModelList) != 2 {
		t.Fatalf("len(model_list) = %d, want 2", len(updated.ModelList))
	}
	if got := updated.ModelList[0].Provider; got != "elevenlabs" {
		t.Fatalf("provider = %q, want %q after normalization", got, "elevenlabs")
	}
	if got := updated.ModelList[0].Model; got != "scribe_v1" {
		t.Fatalf("model = %q, want %q after normalization", got, "scribe_v1")
	}
}

func TestHandleAddModel_NormalizesExplicitElevenLabsUnsupportedModelID(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "elevenlabs-asr",
		Provider:  "elevenlabs",
		Model:     "scribe_v2",
		APIKeys:   config.SimpleSecureStrings("sk_elevenlabs_test"),
	}}
	if err = config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(`{
		"model_name":"new-model",
		"provider":"openai",
		"model":"gpt-4o-mini",
		"api_key":"sk-new-model-key"
	}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	updated, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := updated.ModelList[0].Provider; got != "elevenlabs" {
		t.Fatalf("provider = %q, want %q after normalization", got, "elevenlabs")
	}
	if got := updated.ModelList[0].Model; got != "scribe_v1" {
		t.Fatalf("model = %q, want %q after normalization", got, "scribe_v1")
	}
}

func TestHandleAddModel_RejectsMissingCLIProviderCommand(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()
	resetOAuthHooks(t)
	resetModelProbeHooks(t)

	probeCommandAvailableFunc = func(command string) bool {
		return false
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(`{
		"model_name":"claude-cli-model",
		"provider":"claude-cli",
		"model":"claude-cli"
	}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `provider "claude-cli" is not available for new models`) {
		t.Fatalf("body = %q, want missing cli command error", rec.Body.String())
	}
}

func TestHandleAddModel_DefaultsAntigravityToOAuth(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	added := addModelAndLoadLatest(t, configPath, `{
		"model_name":"gemini-flash",
		"provider":"antigravity",
		"model":"gemini-3-flash"
	}`)
	if got := added.AuthMethod; got != "oauth" {
		t.Fatalf("auth_method = %q, want %q", got, "oauth")
	}
}

func TestHandleAddModel_NormalizesMixedCaseAuthMethod(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	added := addModelAndLoadLatest(t, configPath, `{
		"model_name":"openai-oauth",
		"provider":"openai",
		"model":"gpt-5.4",
		"auth_method":"OAuth"
	}`)
	if got := added.AuthMethod; got != "oauth" {
		t.Fatalf("auth_method = %q, want %q", got, "oauth")
	}
}

func TestHandleAddModel_PreservesExplicitProviderPrefixedModel(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(`{
		"model_name":"openai-gpt",
		"provider":"openai",
		"model":"openai/gpt-4o-mini",
		"api_key":"sk-openai"
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
	added := cfg.ModelList[len(cfg.ModelList)-1]
	if got := added.Provider; got != "openai" {
		t.Fatalf("provider = %q, want %q", got, "openai")
	}
	if got := added.Model; got != "openai/gpt-4o-mini" {
		t.Fatalf("model = %q, want %q", got, "openai/gpt-4o-mini")
	}
}

func TestHandleAddModel_PersistsCustomHeaders(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(`{
		"model_name":"new-model-headers",
		"model":"openai/gpt-4o-mini",
		"custom_headers":{"X-Source":"coding-plan","X-Agent":"openclaw"}
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
	if added.CustomHeaders == nil {
		t.Fatal("custom_headers should not be nil")
	}
	if got := added.CustomHeaders["X-Source"]; got != "coding-plan" {
		t.Fatalf("custom_headers[X-Source] = %q, want %q", got, "coding-plan")
	}
	if got := added.CustomHeaders["X-Agent"]; got != "openclaw" {
		t.Fatalf("custom_headers[X-Agent] = %q, want %q", got, "openclaw")
	}
}

func TestHandleAddModel_PersistsToolSchemaTransform(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(`{
		"model_name":"new-model-transform",
		"model":"openai/gpt-4o-mini",
		"tool_schema_transform":"simple"
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
	added := cfg.ModelList[len(cfg.ModelList)-1]
	if got := added.ToolSchemaTransform; got != "simple" {
		t.Fatalf("tool_schema_transform = %q, want %q", got, "simple")
	}
}

func TestHandleUpdateModel_CustomHeadersPreserveAndClear(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName:     "editable",
		Model:         "openai/gpt-4o-mini",
		APIKeys:       config.SimpleSecureStrings("sk-existing"),
		CustomHeaders: map[string]string{"X-Source": "coding-plan"},
	}}
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Omitted custom_headers should preserve existing value.
	recPreserve := httptest.NewRecorder()
	reqPreserve := httptest.NewRequest(http.MethodPut, "/api/models/0", bytes.NewBufferString(`{
		"model_name":"editable",
		"model":"openai/gpt-4o-mini"
	}`))
	reqPreserve.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recPreserve, reqPreserve)
	if recPreserve.Code != http.StatusOK {
		t.Fatalf("preserve status = %d, want %d, body=%s", recPreserve.Code, http.StatusOK, recPreserve.Body.String())
	}

	afterPreserve, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() after preserve error = %v", err)
	}
	if got := afterPreserve.ModelList[0].CustomHeaders["X-Source"]; got != "coding-plan" {
		t.Fatalf("preserved custom_headers[X-Source] = %q, want %q", got, "coding-plan")
	}

	// Empty object should clear custom_headers.
	recClear := httptest.NewRecorder()
	reqClear := httptest.NewRequest(http.MethodPut, "/api/models/0", bytes.NewBufferString(`{
		"model_name":"editable",
		"model":"openai/gpt-4o-mini",
		"custom_headers":{}
	}`))
	reqClear.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recClear, reqClear)
	if recClear.Code != http.StatusOK {
		t.Fatalf("clear status = %d, want %d, body=%s", recClear.Code, http.StatusOK, recClear.Body.String())
	}

	afterClear, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() after clear error = %v", err)
	}
	if afterClear.ModelList[0].CustomHeaders != nil {
		t.Fatalf("custom_headers = %#v, want nil", afterClear.ModelList[0].CustomHeaders)
	}
}

func TestHandleUpdateModel_ToolSchemaTransformPreserveAndClear(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName:           "editable",
		Model:               "openai/gpt-4o-mini",
		APIKeys:             config.SimpleSecureStrings("sk-existing"),
		ToolSchemaTransform: "simple",
	}}
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	recPreserve := httptest.NewRecorder()
	reqPreserve := httptest.NewRequest(http.MethodPut, "/api/models/0", bytes.NewBufferString(`{
		"model_name":"editable",
		"model":"openai/gpt-4o-mini"
	}`))
	reqPreserve.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recPreserve, reqPreserve)
	if recPreserve.Code != http.StatusOK {
		t.Fatalf("preserve status = %d, want %d, body=%s", recPreserve.Code, http.StatusOK, recPreserve.Body.String())
	}

	afterPreserve, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() after preserve error = %v", err)
	}
	if got := afterPreserve.ModelList[0].ToolSchemaTransform; got != "simple" {
		t.Fatalf("preserved tool_schema_transform = %q, want %q", got, "simple")
	}

	recClear := httptest.NewRecorder()
	reqClear := httptest.NewRequest(http.MethodPut, "/api/models/0", bytes.NewBufferString(`{
		"model_name":"editable",
		"model":"openai/gpt-4o-mini",
		"tool_schema_transform":""
	}`))
	reqClear.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recClear, reqClear)
	if recClear.Code != http.StatusOK {
		t.Fatalf("clear status = %d, want %d, body=%s", recClear.Code, http.StatusOK, recClear.Body.String())
	}

	afterClear, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() after clear error = %v", err)
	}
	if afterClear.ModelList[0].ToolSchemaTransform != "" {
		t.Fatalf("tool_schema_transform = %q, want empty", afterClear.ModelList[0].ToolSchemaTransform)
	}
}

func TestHandleUpdateModel_PersistsProvider(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "editable",
		Model:     "gpt-4o",
		Provider:  "openai",
	}}
	if err = config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/models/0", bytes.NewBufferString(`{
		"model_name":"editable",
		"provider":"openrouter",
		"model":"openai/gpt-4o"
	}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	updated, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := updated.ModelList[0].Provider; got != "openrouter" {
		t.Fatalf("provider = %q, want %q", got, "openrouter")
	}
}

func TestHandleUpdateModel_PreservesExplicitProviderPrefixedModel(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "editable",
		Model:     "gpt-4o",
		Provider:  "openai",
	}}
	if err = config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/models/0", bytes.NewBufferString(`{
		"model_name":"editable",
		"provider":"openai",
		"model":"openai/gpt-5.4"
	}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	updated, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := updated.ModelList[0].Provider; got != "openai" {
		t.Fatalf("provider = %q, want %q", got, "openai")
	}
	if got := updated.ModelList[0].Model; got != "openai/gpt-5.4" {
		t.Fatalf("model = %q, want %q", got, "openai/gpt-5.4")
	}
}

func TestHandleListModels_PreservesExplicitProviderPrefixedModel(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "openrouter-auto-explicit",
		Provider:  "openrouter",
		Model:     "openrouter/auto",
	}}
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
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
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(resp.Models))
	}
	if got := resp.Models[0].Provider; got != "openrouter" {
		t.Fatalf("provider = %q, want %q", got, "openrouter")
	}
	if got := resp.Models[0].Model; got != "openrouter/auto" {
		t.Fatalf("model = %q, want %q", got, "openrouter/auto")
	}
}

func TestHandleListModels_ExposesElevenLabsASRProvider(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "elevenlabs-asr",
		Model:     "elevenlabs/scribe_v1",
		APIKeys:   config.SimpleSecureStrings("sk_elevenlabs_test"),
	}}
	if err = config.SaveConfig(configPath, cfg); err != nil {
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
	if err = json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(resp.Models))
	}
	if got := resp.Models[0].Provider; got != "elevenlabs" {
		t.Fatalf("provider = %q, want %q", got, "elevenlabs")
	}
	if got := resp.Models[0].Model; got != "scribe_v1" {
		t.Fatalf("model = %q, want %q", got, "scribe_v1")
	}
	if resp.Models[0].DefaultModelAllowed {
		t.Fatal("elevenlabs ASR model should not be allowed as the default chat model")
	}
}

func TestHandleUpdateModel_PreservesLegacyModelPrefixWhenProviderOmitted(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "legacy-openrouter",
		Model:     "openrouter/openai/gpt-5.4",
	}}
	if err = config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Simulate an older client: it reads GET /api/models, ignores the new
	// provider field, then PUTs the visible model string back unchanged.
	recList := httptest.NewRecorder()
	reqList := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	mux.ServeHTTP(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body=%s", recList.Code, http.StatusOK, recList.Body.String())
	}

	var listResp struct {
		Models []modelResponse `json:"models"`
	}
	if err = json.Unmarshal(recList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(listResp.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(listResp.Models))
	}
	if got := listResp.Models[0].Provider; got != "openrouter" {
		t.Fatalf("provider = %q, want %q", got, "openrouter")
	}
	if got := listResp.Models[0].Model; got != "openai/gpt-5.4" {
		t.Fatalf("model = %q, want %q", got, "openai/gpt-5.4")
	}

	recUpdate := httptest.NewRecorder()
	reqUpdate := httptest.NewRequest(http.MethodPut, "/api/models/0", bytes.NewBufferString(`{
		"model_name":"legacy-openrouter",
		"model":"openai/gpt-5.4"
	}`))
	reqUpdate.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recUpdate, reqUpdate)

	if recUpdate.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d, body=%s", recUpdate.Code, http.StatusOK, recUpdate.Body.String())
	}

	updated, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := updated.ModelList[0].Provider; got != "openrouter" {
		t.Fatalf("provider = %q, want %q", got, "openrouter")
	}
	if got := updated.ModelList[0].Model; got != "openai/gpt-5.4" {
		t.Fatalf("model = %q, want %q", got, "openai/gpt-5.4")
	}
}

func TestHandleUpdateModel_MigratesLegacyElevenLabsASRWhenProviderOmitted(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "elevenlabs-asr",
		Model:     "elevenlabs/scribe_v1",
		APIKeys:   config.SimpleSecureStrings("sk_elevenlabs_test"),
	}}
	if err = config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	recList := httptest.NewRecorder()
	reqList := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	mux.ServeHTTP(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body=%s", recList.Code, http.StatusOK, recList.Body.String())
	}

	var listResp struct {
		Models []modelResponse `json:"models"`
	}
	if err = json.Unmarshal(recList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(listResp.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(listResp.Models))
	}
	if got := listResp.Models[0].Provider; got != "elevenlabs" {
		t.Fatalf("provider = %q, want %q", got, "elevenlabs")
	}
	if got := listResp.Models[0].Model; got != "scribe_v1" {
		t.Fatalf("model = %q, want %q", got, "scribe_v1")
	}

	recUpdate := httptest.NewRecorder()
	reqUpdate := httptest.NewRequest(http.MethodPut, "/api/models/0", bytes.NewBufferString(`{
		"model_name":"elevenlabs-asr",
		"model":"scribe_v1",
		"api_base":"https://api.elevenlabs.io"
	}`))
	reqUpdate.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recUpdate, reqUpdate)

	if recUpdate.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d, body=%s", recUpdate.Code, http.StatusOK, recUpdate.Body.String())
	}

	updated, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := updated.ModelList[0].Provider; got != "elevenlabs" {
		t.Fatalf("provider = %q, want %q", got, "elevenlabs")
	}
	if got := updated.ModelList[0].Model; got != "scribe_v1" {
		t.Fatalf("model = %q, want %q", got, "scribe_v1")
	}
	if got := updated.ModelList[0].APIBase; got != "https://api.elevenlabs.io" {
		t.Fatalf("api_base = %q, want %q", got, "https://api.elevenlabs.io")
	}
}

func TestHandleUpdateModel_RoundTripsExplicitLegacyElevenLabsModelID(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "elevenlabs-asr",
		Provider:  "elevenlabs",
		Model:     "scribe_v2",
		APIKeys:   config.SimpleSecureStrings("sk_elevenlabs_test"),
	}}
	if err = config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	recList := httptest.NewRecorder()
	reqList := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	mux.ServeHTTP(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body=%s", recList.Code, http.StatusOK, recList.Body.String())
	}

	var listResp struct {
		Models []modelResponse `json:"models"`
	}
	if err = json.Unmarshal(recList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(listResp.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(listResp.Models))
	}
	if got := listResp.Models[0].Provider; got != "elevenlabs" {
		t.Fatalf("provider = %q, want %q", got, "elevenlabs")
	}
	if got := listResp.Models[0].Model; got != "scribe_v1" {
		t.Fatalf("model = %q, want %q after GET normalization", got, "scribe_v1")
	}

	recUpdate := httptest.NewRecorder()
	reqUpdate := httptest.NewRequest(http.MethodPut, "/api/models/0", bytes.NewBufferString(`{
		"model_name":"elevenlabs-asr",
		"provider":"elevenlabs",
		"model":"scribe_v1",
		"api_base":"https://api.elevenlabs.io"
	}`))
	reqUpdate.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recUpdate, reqUpdate)

	if recUpdate.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d, body=%s", recUpdate.Code, http.StatusOK, recUpdate.Body.String())
	}

	updated, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := updated.ModelList[0].Provider; got != "elevenlabs" {
		t.Fatalf("provider = %q, want %q", got, "elevenlabs")
	}
	if got := updated.ModelList[0].Model; got != "scribe_v1" {
		t.Fatalf("model = %q, want %q", got, "scribe_v1")
	}
	if got := updated.ModelList[0].APIBase; got != "https://api.elevenlabs.io" {
		t.Fatalf("api_base = %q, want %q", got, "https://api.elevenlabs.io")
	}
}

func TestHandleUpdateModel_ClearsDefaultWhenSavingASROnlyModel(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "elevenlabs-asr",
		Provider:  "elevenlabs",
		Model:     "scribe_v1",
		APIKeys:   config.SimpleSecureStrings("sk_elevenlabs_test"),
	}}
	cfg.Agents.Defaults.ModelName = "elevenlabs-asr"
	if err = config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/models/0", bytes.NewBufferString(`{
		"model_name":"elevenlabs-asr",
		"provider":"elevenlabs",
		"model":"scribe_v1",
		"api_base":"https://api.elevenlabs.io"
	}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	updated, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := updated.Agents.Defaults.ModelName; got != "" {
		t.Fatalf("default model = %q, want cleared default", got)
	}
}

func TestHandleAddModel_RejectsUnsupportedElevenLabsModelID(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models", bytes.NewBufferString(`{
		"model_name":"elevenlabs-asr",
		"provider":"elevenlabs",
		"model":"scribe_v2"
	}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `provider "elevenlabs" only supports model "scribe_v1"`) {
		t.Fatalf("body = %q, want elevenlabs model validation error", rec.Body.String())
	}
}

func TestHandleUpdateModel_PreservesLegacyModelPrefixWhenProviderOmittedAndModelChanges(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "legacy-openrouter",
		Model:     "openrouter/openai/gpt-5.4",
	}}
	if err = config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/models/0", bytes.NewBufferString(`{
		"model_name":"legacy-openrouter",
		"model":"openai/gpt-5.5"
	}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	updated, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := updated.ModelList[0].Provider; got != "openrouter" {
		t.Fatalf("provider = %q, want %q", got, "openrouter")
	}
	if got := updated.ModelList[0].Model; got != "openai/gpt-5.5" {
		t.Fatalf("model = %q, want %q", got, "openai/gpt-5.5")
	}
}

func TestHandleListModels_ReturnsProviderOptionsWithoutPersistingLegacyMigration(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "legacy-openrouter",
		Model:     "openrouter/openai/gpt-5.4",
	}}
	err = config.SaveConfig(configPath, cfg)
	if err != nil {
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
		Models          []modelResponse                 `json:"models"`
		ProviderOptions []providers.ModelProviderOption `json:"provider_options"`
	}
	if unmarshalErr := json.Unmarshal(rec.Body.Bytes(), &resp); unmarshalErr != nil {
		t.Fatalf("Unmarshal() error = %v", unmarshalErr)
	}
	if len(resp.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(resp.Models))
	}
	if got := resp.Models[0].Provider; got != "openrouter" {
		t.Fatalf("provider = %q, want %q", got, "openrouter")
	}
	if got := resp.Models[0].Model; got != "openai/gpt-5.4" {
		t.Fatalf("model = %q, want %q", got, "openai/gpt-5.4")
	}

	optionsByID := make(map[string]providers.ModelProviderOption, len(resp.ProviderOptions))
	for _, option := range resp.ProviderOptions {
		optionsByID[option.ID] = option
	}
	if len(optionsByID) == 0 {
		t.Fatal("provider_options should not be empty")
	}
	if option, ok := optionsByID["openai"]; !ok {
		t.Fatal("openai provider option missing")
	} else if option.DefaultAPIBase != "https://api.openai.com/v1" {
		t.Fatalf("openai default_api_base = %q, want %q", option.DefaultAPIBase, "https://api.openai.com/v1")
	}
	if option, ok := optionsByID["anthropic"]; !ok {
		t.Fatal("anthropic provider option missing")
	} else if option.DefaultAPIBase != "https://api.anthropic.com/v1" {
		t.Fatalf("anthropic default_api_base = %q, want %q", option.DefaultAPIBase, "https://api.anthropic.com/v1")
	}
	if _, ok := optionsByID["azure"]; !ok {
		t.Fatal("azure provider option missing")
	}
	if option, ok := optionsByID["github-copilot"]; !ok {
		t.Fatal("github-copilot provider option missing")
	} else if option.DefaultAPIBase != "localhost:4321" {
		t.Fatalf("github-copilot default_api_base = %q, want %q", option.DefaultAPIBase, "localhost:4321")
	}
	if option, ok := optionsByID["elevenlabs"]; !ok {
		t.Fatal("elevenlabs provider option missing")
	} else {
		if option.DefaultAPIBase != "https://api.elevenlabs.io" {
			t.Fatalf("elevenlabs default_api_base = %q, want %q", option.DefaultAPIBase, "https://api.elevenlabs.io")
		}
		if option.DefaultModelAllowed {
			t.Fatal("elevenlabs should be marked as not allowed for default chat model selection")
		}
	}
	if option, ok := optionsByID["lmstudio"]; !ok {
		t.Fatal("lmstudio provider option missing")
	} else if !option.EmptyAPIKeyAllowed {
		t.Fatal("lmstudio should allow empty api keys")
	}
	if option, ok := optionsByID["bedrock"]; !ok {
		t.Fatal("bedrock provider option missing")
	} else if !option.CreateAllowed {
		t.Fatal("bedrock should stay creatable and defer AWS credential failures to runtime")
	}
	if option, ok := optionsByID["antigravity"]; !ok {
		t.Fatal("antigravity provider option missing")
	} else {
		if option.DefaultAuthMethod != "oauth" {
			t.Fatalf("antigravity default_auth_method = %q, want %q", option.DefaultAuthMethod, "oauth")
		}
		if !option.AuthMethodLocked {
			t.Fatal("antigravity auth method should be locked")
		}
	}

	updated, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := updated.ModelList[0].Provider; got != "" {
		t.Fatalf("persisted provider = %q, want unchanged empty provider", got)
	}
	if got := updated.ModelList[0].Model; got != "openrouter/openai/gpt-5.4" {
		t.Fatalf("persisted model = %q, want unchanged legacy model", got)
	}
}

func TestHandleListModels_ReturnsProviderField(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "nvidia-glm",
		Provider:  "nvidia",
		Model:     "z-ai/glm-5.1",
		APIKeys:   config.SimpleSecureStrings("nv-key"),
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
	if got := resp.Models[0].Provider; got != "nvidia" {
		t.Fatalf("provider = %q, want %q", got, "nvidia")
	}
}

func TestHandleListModels_PreservesKnownProviderInCatalog(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "bedrock-claude",
		Model:     "bedrock/us.anthropic.claude-sonnet-4-20250514-v1:0",
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
		Models          []modelResponse                 `json:"models"`
		ProviderOptions []providers.ModelProviderOption `json:"provider_options"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(resp.Models))
	}
	if got := resp.Models[0].Provider; got != "bedrock" {
		t.Fatalf("provider = %q, want %q", got, "bedrock")
	}
	if got := resp.Models[0].Model; got != "us.anthropic.claude-sonnet-4-20250514-v1:0" {
		t.Fatalf("model = %q, want %q", got, "us.anthropic.claude-sonnet-4-20250514-v1:0")
	}
	foundBedrock := false
	for _, option := range resp.ProviderOptions {
		if option.ID == "bedrock" {
			foundBedrock = true
			if !option.CreateAllowed {
				t.Fatal("bedrock should stay creatable in provider_options")
			}
		}
	}
	if !foundBedrock {
		t.Fatal("bedrock should be included in provider_options for compatibility")
	}
}

func TestHandleUpdateModel_AllowsExistingBedrockProvider(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "bedrock-claude",
		Provider:  "bedrock",
		Model:     "us.anthropic.claude-sonnet-4-20250514-v1:0",
		APIBase:   "us-west-2",
	}}
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatalf("SaveConfig() error = %v", saveErr)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/models/0", bytes.NewBufferString(`{
		"model_name":"bedrock-claude",
		"provider":"bedrock",
		"model":"us.anthropic.claude-3-7-sonnet-20250219-v1:0",
		"api_base":"us-east-1"
	}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	updated, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if got := updated.ModelList[0].Provider; got != "bedrock" {
		t.Fatalf("provider = %q, want %q", got, "bedrock")
	}
	if got := updated.ModelList[0].Model; got != "us.anthropic.claude-3-7-sonnet-20250219-v1:0" {
		t.Fatalf("model = %q, want updated bedrock model", got)
	}
	if got := updated.ModelList[0].APIBase; got != "us-east-1" {
		t.Fatalf("api_base = %q, want %q", got, "us-east-1")
	}
}

func TestHandleListModels_ReturnsEffectiveProviderField(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "plain-openai",
			Model:     "gpt-4o",
		},
		{
			ModelName: "explicit-google",
			Provider:  "google",
			Model:     "gemini-2.5-pro",
		},
		{
			ModelName: "explicit-qwen-intl",
			Provider:  "qwen-international",
			Model:     "qwen3-coder-plus",
		},
	}
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

	if len(resp.Models) != 3 {
		t.Fatalf("len(models) = %d, want 3", len(resp.Models))
	}

	if got := resp.Models[0].Provider; got != "openai" {
		t.Fatalf("provider[0] = %q, want %q", got, "openai")
	}
	if got := resp.Models[0].Model; got != "gpt-4o" {
		t.Fatalf("model[0] = %q, want %q", got, "gpt-4o")
	}
	if got := resp.Models[1].Provider; got != "gemini" {
		t.Fatalf("provider[1] = %q, want %q", got, "gemini")
	}
	if got := resp.Models[1].Model; got != "gemini-2.5-pro" {
		t.Fatalf("model[1] = %q, want %q", got, "gemini-2.5-pro")
	}
	if got := resp.Models[2].Provider; got != "qwen-intl" {
		t.Fatalf("provider[2] = %q, want %q", got, "qwen-intl")
	}
	if got := resp.Models[2].Model; got != "qwen3-coder-plus" {
		t.Fatalf("model[2] = %q, want %q", got, "qwen3-coder-plus")
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

func TestHandleSetDefaultModel_RejectsElevenLabsASRProvider(t *testing.T) {
	configPath, cleanup := setupOAuthTestEnv(t)
	defer cleanup()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "elevenlabs-asr",
			Provider:  "elevenlabs",
			Model:     "scribe_v1",
			APIKeys:   config.SimpleSecureStrings("sk_elevenlabs_test"),
		},
	}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	h := NewHandler(configPath)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/models/default", bytes.NewBufferString(`{
		"model_name": "elevenlabs-asr"
	}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "cannot be used as the default chat model") {
		t.Fatalf("body = %q, want default chat model rejection", rec.Body.String())
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
