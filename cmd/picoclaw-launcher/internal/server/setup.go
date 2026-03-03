package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// RegisterSetupAPI registers endpoints for the initial setup flow.
// These are used when the user hasn't run `picoclaw init` yet.
func RegisterSetupAPI(mux *http.ServeMux, absPath string) {
	mux.HandleFunc("GET /api/setup/status", func(w http.ResponseWriter, r *http.Request) {
		handleSetupStatus(w, absPath)
	})
	mux.HandleFunc("POST /api/setup/test-llm", func(w http.ResponseWriter, r *http.Request) {
		handleTestLLM(w, r)
	})
	mux.HandleFunc("POST /api/setup/save", func(w http.ResponseWriter, r *http.Request) {
		handleSetupSave(w, r, absPath)
	})
}

// NeedsSetup returns true if config is missing or has no usable LLM configured.
func NeedsSetup(absPath string) bool {
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return true
	}
	cfg, err := config.LoadConfig(absPath)
	if err != nil {
		return true
	}
	if cfg.Agents.Defaults.GetModelName() == "" {
		return true
	}
	if len(cfg.ModelList) == 0 && cfg.Providers.IsEmpty() {
		return true
	}
	return false
}

// handleSetupStatus returns whether initial setup is needed.
func handleSetupStatus(w http.ResponseWriter, absPath string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"needs_setup": NeedsSetup(absPath),
		"config_path": absPath,
	})
}

// setupTestRequest is the request body for POST /api/setup/test-llm.
type setupTestRequest struct {
	APIKey  string `json:"api_key"`
	APIBase string `json:"api_base"`
	Model   string `json:"model"`
}

// handleTestLLM tests an LLM connection without saving config.
func handleTestLLM(w http.ResponseWriter, r *http.Request) {
	var req setupTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.APIKey == "" || req.Model == "" {
		http.Error(w, "api_key and model are required", http.StatusBadRequest)
		return
	}
	if req.APIBase == "" {
		req.APIBase = "https://api.openai.com/v1"
	}

	protocol := DetectProtocol(req.APIBase)
	modelID := protocol + "/" + req.Model

	modelCfg := &config.ModelConfig{
		ModelName: req.Model,
		Model:     modelID,
		APIBase:   req.APIBase,
		APIKey:    req.APIKey,
	}

	provider, resolvedModel, err := providers.CreateProviderFromConfig(modelCfg)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Failed to create provider: %v", err),
		})
		return
	}

	if resolvedModel == "" {
		resolvedModel = req.Model
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp, err := provider.Chat(ctx, []providers.Message{
		{Role: "user", Content: "Reply with exactly one word: PONG"},
	}, nil, resolvedModel, nil)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   fmt.Sprintf("LLM call failed: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success":  true,
		"response": strings.TrimSpace(resp.Content),
		"model":    resolvedModel,
		"protocol": protocol,
	})
}

// setupSaveRequest is the request body for POST /api/setup/save.
type setupSaveRequest struct {
	APIKey  string `json:"api_key"`
	APIBase string `json:"api_base"`
	Model   string `json:"model"`
}

// handleSetupSave saves a minimal config from the setup form.
func handleSetupSave(w http.ResponseWriter, r *http.Request, absPath string) {
	var req setupSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.APIKey == "" || req.Model == "" {
		http.Error(w, "api_key and model are required", http.StatusBadRequest)
		return
	}
	if req.APIBase == "" {
		req.APIBase = "https://api.openai.com/v1"
	}

	protocol := DetectProtocol(req.APIBase)
	modelID := protocol + "/" + req.Model

	defaults := config.DefaultConfig()
	workspace := defaults.Agents.Defaults.Workspace

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:           workspace,
				RestrictToWorkspace: true,
				ModelName:           req.Model,
				MaxTokens:           32768,
				MaxToolIterations:   50,
			},
		},
		ModelList: []config.ModelConfig{
			{
				ModelName: req.Model,
				Model:     modelID,
				APIBase:   req.APIBase,
				APIKey:    req.APIKey,
			},
		},
		Gateway: defaults.Gateway,
		Tools: config.ToolsConfig{
			Exec: config.ExecConfig{EnableDenyPatterns: true},
			Web: config.WebToolsConfig{
				DuckDuckGo: config.DuckDuckGoConfig{Enabled: true, MaxResults: 5},
			},
		},
		Providers: config.ProvidersConfig{
			OpenAI: config.OpenAIProviderConfig{WebSearch: true},
		},
	}

	// Ensure directories exist.
	os.MkdirAll(filepath.Dir(absPath), 0755)
	os.MkdirAll(workspace, 0755)

	if err := config.SaveConfig(absPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success":     true,
		"config_path": absPath,
		"workspace":   workspace,
	})
}

// DetectProtocol guesses the provider protocol from the API base URL.
func DetectProtocol(baseURL string) string {
	lower := strings.ToLower(baseURL)
	switch {
	case strings.Contains(lower, "anthropic"):
		return "anthropic"
	case strings.Contains(lower, "generativelanguage.googleapis"):
		return "gemini"
	case strings.Contains(lower, "dashscope.aliyuncs"):
		return "qwen"
	case strings.Contains(lower, "open.bigmodel.cn"):
		return "zhipu"
	case strings.Contains(lower, "moonshot"):
		return "moonshot"
	case strings.Contains(lower, "deepseek"):
		return "deepseek"
	case strings.Contains(lower, "openrouter"):
		return "openrouter"
	case strings.Contains(lower, "groq"):
		return "groq"
	case strings.Contains(lower, "localhost:11434"):
		return "ollama"
	case strings.Contains(lower, "volcengine") || strings.Contains(lower, "volces.com"):
		return "volcengine"
	case strings.Contains(lower, "cerebras"):
		return "cerebras"
	case strings.Contains(lower, "nvidia") || strings.Contains(lower, "integrate.api"):
		return "nvidia"
	case strings.Contains(lower, "mistral"):
		return "mistral"
	default:
		return "openai"
	}
}
