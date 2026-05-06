package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/audio/asr"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// registerModelRoutes binds model list management endpoints to the ServeMux.
func (h *Handler) registerModelRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/models", h.handleListModels)
	mux.HandleFunc("POST /api/models", h.handleAddModel)
	mux.HandleFunc("POST /api/models/default", h.handleSetDefaultModel)
	mux.HandleFunc("PUT /api/models/{index}", h.handleUpdateModel)
	mux.HandleFunc("DELETE /api/models/{index}", h.handleDeleteModel)
	mux.HandleFunc("POST /api/models/{index}/test", h.handleTestModel)
	mux.HandleFunc("POST /api/models/fetch", h.handleFetchModels)
	mux.HandleFunc("GET /api/models/catalog", h.handleListCatalogs)
	mux.HandleFunc("DELETE /api/models/catalog/{id}", h.handleDeleteCatalog)
}

// modelResponse is the JSON structure returned for each model in the list.
// All ModelConfig fields are included so the frontend can display and edit them.
type modelResponse struct {
	Index      int    `json:"index"`
	ModelName  string `json:"model_name"`
	Provider   string `json:"provider,omitempty"`
	Model      string `json:"model"`
	APIBase    string `json:"api_base,omitempty"`
	APIKey     string `json:"api_key"`
	Proxy      string `json:"proxy,omitempty"`
	AuthMethod string `json:"auth_method,omitempty"`
	// Advanced fields
	ConnectMode         string            `json:"connect_mode,omitempty"`
	Workspace           string            `json:"workspace,omitempty"`
	RPM                 int               `json:"rpm,omitempty"`
	MaxTokensField      string            `json:"max_tokens_field,omitempty"`
	RequestTimeout      int               `json:"request_timeout,omitempty"`
	ThinkingLevel       string            `json:"thinking_level,omitempty"`
	ToolSchemaTransform string            `json:"tool_schema_transform,omitempty"`
	ExtraBody           map[string]any    `json:"extra_body,omitempty"`
	CustomHeaders       map[string]string `json:"custom_headers,omitempty"`
	// Meta
	Enabled             bool   `json:"enabled"`
	Available           bool   `json:"available"`
	Status              string `json:"status"`
	IsDefault           bool   `json:"is_default"`
	IsVirtual           bool   `json:"is_virtual"`
	DefaultModelAllowed bool   `json:"default_model_allowed"`
}

func normalizeStoredModelConfig(mc *config.ModelConfig) bool {
	if mc == nil {
		return false
	}

	changed := false
	model := strings.TrimSpace(mc.Model)
	if model != mc.Model {
		mc.Model = model
		changed = true
	}
	provider := strings.TrimSpace(mc.Provider)
	if provider != mc.Provider {
		mc.Provider = provider
		changed = true
	}
	authMethod := strings.ToLower(strings.TrimSpace(mc.AuthMethod))
	if authMethod != mc.AuthMethod {
		mc.AuthMethod = authMethod
		changed = true
	}

	if provider != "" {
		normalizedProvider := providers.NormalizeProvider(provider)
		if providers.IsSupportedModelProvider(normalizedProvider) && normalizedProvider != provider {
			mc.Provider = normalizedProvider
			changed = true
		}
		if mc.Provider == "elevenlabs" {
			if _, strippedModel, found := strings.Cut(
				model,
				"/",
			); found &&
				providers.NormalizeProvider(strings.TrimSpace(provider)) == "elevenlabs" {
				strippedModel = strings.TrimSpace(strippedModel)
				if strippedModel != "" && strippedModel != mc.Model {
					mc.Model = strippedModel
					changed = true
				}
			}
			if strings.TrimSpace(mc.Model) != asr.ElevenLabsSupportedModelID() {
				mc.Model = asr.ElevenLabsSupportedModelID()
				changed = true
			}
		}
		return changed
	}

	effectiveProvider, modelID := providers.SplitModelProviderAndID(model, "openai")
	if effectiveProvider == "" {
		return changed
	}
	if mc.Provider != effectiveProvider {
		mc.Provider = effectiveProvider
		changed = true
	}
	if mc.Model != modelID {
		mc.Model = modelID
		changed = true
	}
	return changed
}

func normalizeIncomingModelConfig(mc *config.ModelConfig) {
	if mc == nil {
		return
	}

	mc.Model = strings.TrimSpace(mc.Model)
	mc.Provider = strings.TrimSpace(mc.Provider)
	mc.AuthMethod = strings.ToLower(strings.TrimSpace(mc.AuthMethod))
	if mc.Provider == "" {
		mc.Provider, mc.Model = providers.SplitModelProviderAndID(mc.Model, "openai")
	} else {
		mc.Provider = providers.NormalizeProvider(mc.Provider)
		if mc.Provider == "elevenlabs" {
			if _, strippedModel, found := strings.Cut(mc.Model, "/"); found {
				strippedModel = strings.TrimSpace(strippedModel)
				if strippedModel != "" {
					mc.Model = strippedModel
				}
			}
		}
	}
	if mc.Provider == "antigravity" && mc.AuthMethod == "" {
		mc.AuthMethod = "oauth"
	}
}

func createAllowedForProvider(provider string) bool {
	normalized := providers.NormalizeProvider(provider)
	switch normalized {
	case "bedrock":
		// Bedrock currently authenticates through the AWS SDK credential chain
		// (env vars, shared profiles, IAM roles, etc.), and this Web layer does
		// not yet have a reliable preflight check for those credential sources.
		// Keep it creatable in the catalog and let provider construction/runtime
		// return the concrete AWS error when the environment is incomplete.
		return true
	case "claude-cli", "codex-cli":
		return cliProviderCreateAllowedFromCurrentStatus(normalized)
	default:
		return providers.IsCreatableModelProvider(normalized)
	}
}

// cliProviderCreateAllowedFromCurrentStatus intentionally reuses the existing
// local model status pipeline so provider catalog gating follows the same CLI
// executable probe used by launcher readiness.
func cliProviderCreateAllowedFromCurrentStatus(provider string) bool {
	status := modelConfigurationStatus(&config.ModelConfig{
		Provider: provider,
		Model:    provider,
	})
	return status.Available
}

func modelProviderOptionsForResponse() []providers.ModelProviderOption {
	options := providers.ModelProviderOptions()
	for i := range options {
		options[i].CreateAllowed = createAllowedForProvider(options[i].ID)
	}
	return options
}

func defaultModelAllowedForModelConfig(mc *config.ModelConfig) bool {
	provider, _ := providers.ExtractProtocol(mc)
	return providers.IsDefaultModelProvider(provider)
}

func validateIncomingModelConfig(mc *config.ModelConfig, existing *config.ModelConfig) error {
	if mc == nil {
		return fmt.Errorf("model config is required")
	}
	if err := mc.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(mc.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if !providers.IsSupportedModelProvider(mc.Provider) {
		return fmt.Errorf("provider %q is not supported", mc.Provider)
	}
	if mc.Provider == "elevenlabs" && strings.TrimSpace(mc.Model) != asr.ElevenLabsSupportedModelID() {
		return fmt.Errorf("provider %q only supports model %q", mc.Provider, asr.ElevenLabsSupportedModelID())
	}
	if !createAllowedForProvider(mc.Provider) {
		if existing == nil {
			return fmt.Errorf("provider %q is not available for new models", mc.Provider)
		}
		existingProvider, _ := providers.ExtractProtocol(existing)
		if providers.NormalizeProvider(existingProvider) != mc.Provider {
			return fmt.Errorf("provider %q is not available for selection", mc.Provider)
		}
	}
	return nil
}

func normalizeStoredModelProviders(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}

	changed := false
	for _, model := range cfg.ModelList {
		if normalizeStoredModelConfig(model) {
			changed = true
		}
	}
	return changed
}

// handleListModels returns all model_list entries with masked API keys.
//
//	GET /api/models
func (h *Handler) handleListModels(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	// Normalize legacy provider/model storage in memory so GET can round-trip
	// through the current API shape without mutating the on-disk config.
	normalizeStoredModelProviders(cfg)

	defaultModel := cfg.Agents.Defaults.GetModelName()
	modelStatuses := make([]modelConfigurationSummary, len(cfg.ModelList))

	var wg sync.WaitGroup
	wg.Add(len(cfg.ModelList))
	for i, m := range cfg.ModelList {
		go func(i int, m *config.ModelConfig) {
			defer wg.Done()
			modelStatuses[i] = modelConfigurationStatus(m)
		}(i, m)
	}
	wg.Wait()

	models := make([]modelResponse, 0, len(cfg.ModelList))
	for i, m := range cfg.ModelList {
		provider, modelID := providers.ExtractProtocol(m)
		models = append(models, modelResponse{
			Index:               i,
			ModelName:           m.ModelName,
			Provider:            provider,
			Model:               modelID,
			APIBase:             m.APIBase,
			APIKey:              maskAPIKey(m.APIKey()),
			Proxy:               m.Proxy,
			AuthMethod:          m.AuthMethod,
			ConnectMode:         m.ConnectMode,
			Workspace:           m.Workspace,
			RPM:                 m.RPM,
			MaxTokensField:      m.MaxTokensField,
			RequestTimeout:      m.RequestTimeout,
			ThinkingLevel:       m.ThinkingLevel,
			ToolSchemaTransform: m.ToolSchemaTransform,
			ExtraBody:           m.ExtraBody,
			CustomHeaders:       m.CustomHeaders,
			Enabled:             m.Enabled,
			Available:           modelStatuses[i].Available,
			Status:              modelStatuses[i].Status,
			IsDefault:           m.ModelName == defaultModel,
			IsVirtual:           m.IsVirtual(),
			DefaultModelAllowed: defaultModelAllowedForModelConfig(m),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"models":           models,
		"total":            len(models),
		"default_model":    defaultModel,
		"provider_options": modelProviderOptionsForResponse(),
	})
}

// handleAddModel appends a new model configuration entry.
//
//	POST /api/models
func (h *Handler) handleAddModel(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	type custom struct {
		config.ModelConfig
		APIKey string `json:"api_key"`
	}

	var mc custom
	if err = json.Unmarshal(body, &mc); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	normalizeIncomingModelConfig(&mc.ModelConfig)

	if err = validateIncomingModelConfig(&mc.ModelConfig, nil); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	if mc.APIKey != "" {
		mc.ModelConfig.SetAPIKey(mc.APIKey)
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	cfg.ModelList = append(cfg.ModelList, &mc.ModelConfig)
	normalizeStoredModelProviders(cfg)

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"index":  len(cfg.ModelList) - 1,
	})
}

// handleUpdateModel replaces a model configuration entry at the given index.
// If the request body omits api_key (or sends an empty string), the existing
// stored key is preserved so callers can update only api_base / proxy without
// exposing or clearing the secret.
//
//	PUT /api/models/{index}
func (h *Handler) handleUpdateModel(w http.ResponseWriter, r *http.Request) {
	idx, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "Invalid index", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var rawFields map[string]json.RawMessage
	if err = json.Unmarshal(body, &rawFields); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	type custom struct {
		config.ModelConfig
		APIKey string `json:"api_key"`
	}

	var mc custom
	if err = json.Unmarshal(body, &mc); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	if idx < 0 || idx >= len(cfg.ModelList) {
		http.Error(w, fmt.Sprintf("Index %d out of range (0-%d)", idx, len(cfg.ModelList)-1), http.StatusNotFound)
		return
	}

	// Preserve the existing API key when the caller omits it (empty string).
	// This lets the UI update api_base / proxy without clearing the stored secret.
	if mc.APIKey == "" {
		mc.ModelConfig.SetAPIKey(cfg.ModelList[idx].APIKey())
	} else {
		mc.ModelConfig.SetAPIKey(mc.APIKey)
	}
	// Preserve existing ExtraBody when omitted (nil), but clear it when
	// the frontend sends an empty object {} to indicate the field should
	// be removed.
	if mc.ExtraBody == nil {
		mc.ExtraBody = cfg.ModelList[idx].ExtraBody
	} else if len(mc.ExtraBody) == 0 {
		mc.ExtraBody = nil
	}
	// Preserve existing CustomHeaders when omitted (nil), but clear it when
	// the frontend sends an empty object {} to indicate the field should
	// be removed.
	if mc.CustomHeaders == nil {
		mc.CustomHeaders = cfg.ModelList[idx].CustomHeaders
	} else if len(mc.CustomHeaders) == 0 {
		mc.CustomHeaders = nil
	}
	if _, ok := rawFields["tool_schema_transform"]; !ok {
		mc.ToolSchemaTransform = cfg.ModelList[idx].ToolSchemaTransform
	}
	// Preserve the existing Provider when the caller omits it. This keeps the
	// update API backward-compatible for clients that haven't started sending
	// the new field yet, while still allowing explicit clearing via "".
	if _, ok := rawFields["provider"]; !ok {
		mc.Provider = cfg.ModelList[idx].Provider
		// Older clients still round-trip the legacy model field only. When the
		// stored config encodes provider/model in Model and has no explicit
		// Provider field yet, continue preserving that hidden provider prefix.
		// This keeps provider-omitted updates backward-compatible even when an
		// older client edits the visible model ID.
		if strings.TrimSpace(cfg.ModelList[idx].Provider) == "" {
			existingRawModel := strings.TrimSpace(cfg.ModelList[idx].Model)
			incomingModel := strings.TrimSpace(mc.Model)
			existingProtocol, existingModelID := providers.ExtractProtocol(cfg.ModelList[idx])
			if existingRawModel != "" && existingRawModel != existingModelID && incomingModel != "" {
				if incomingModel == existingModelID {
					mc.Model = existingRawModel
				} else if strings.Contains(incomingModel, "/") && !strings.Contains(existingModelID, "/") {
					// Older clients never saw the hidden provider prefix for simple
					// legacy entries such as "openai/gpt-4o". If they now send an
					// explicit provider/model string, treat it as the caller's full
					// intent instead of re-applying the old hidden prefix.
					mc.Model = incomingModel
				} else if !strings.HasPrefix(incomingModel, existingProtocol+"/") {
					mc.Model = existingProtocol + "/" + incomingModel
				}
			}
		}
	}

	normalizeIncomingModelConfig(&mc.ModelConfig)
	if err = validateIncomingModelConfig(&mc.ModelConfig, cfg.ModelList[idx]); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}
	if cfg.Agents.Defaults.ModelName == cfg.ModelList[idx].ModelName &&
		!defaultModelAllowedForModelConfig(&mc.ModelConfig) {
		// Allow users to recover from legacy/invalid defaults by saving the model
		// and clearing the default chat model reference in the same write.
		cfg.Agents.Defaults.ModelName = ""
	}

	cfg.ModelList[idx] = &mc.ModelConfig
	normalizeStoredModelProviders(cfg)

	logger.Debugf("update model config: %#v", mc.ModelConfig)

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleDeleteModel removes a model configuration entry at the given index.
//
//	DELETE /api/models/{index}
func (h *Handler) handleDeleteModel(w http.ResponseWriter, r *http.Request) {
	idx, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "Invalid index", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	if idx < 0 || idx >= len(cfg.ModelList) {
		http.Error(w, fmt.Sprintf("Index %d out of range (0-%d)", idx, len(cfg.ModelList)-1), http.StatusNotFound)
		return
	}

	deletedModelName := cfg.ModelList[idx].ModelName

	cfg.ModelList = append(cfg.ModelList[:idx], cfg.ModelList[idx+1:]...)

	// If the deleted model was the default, clear it.
	if cfg.Agents.Defaults.ModelName == deletedModelName {
		cfg.Agents.Defaults.ModelName = ""
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleSetDefaultModel sets the default model for all agents.
//
//	POST /api/models/default
func (h *Handler) handleSetDefaultModel(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		ModelName string `json:"model_name"`
	}
	if err = json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.ModelName == "" {
		http.Error(w, "model_name is required", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	// Verify the model_name exists in model_list and is not a virtual model
	found := false
	isVirtual := false
	for _, m := range cfg.ModelList {
		if m.ModelName == req.ModelName {
			found = true
			isVirtual = m.IsVirtual()
			break
		}
	}
	if !found {
		http.Error(w, fmt.Sprintf("Model %q not found in model_list", req.ModelName), http.StatusNotFound)
		return
	}
	if isVirtual {
		http.Error(w, fmt.Sprintf("Cannot set virtual model %q as default", req.ModelName), http.StatusBadRequest)
		return
	}
	for _, m := range cfg.ModelList {
		if m.ModelName == req.ModelName {
			if !defaultModelAllowedForModelConfig(m) {
				http.Error(
					w,
					fmt.Sprintf("Model %q cannot be used as the default chat model", req.ModelName),
					http.StatusBadRequest,
				)
				return
			}
			break
		}
	}

	cfg.Agents.Defaults.ModelName = req.ModelName

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":        "ok",
		"default_model": req.ModelName,
	})
}

// maskAPIKey returns a masked version of an API key for safe display.
// Keys longer than 12 chars show prefix + last 4 chars: "sk-****abcd".
// Keys 9-12 chars show prefix + last 2 chars: "sk-****cd".
// Shorter keys are fully masked as "****".
// Empty keys return empty string.
// Ensure at least 40% of the key will not be displayed.
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}

	if len(key) <= 8 {
		return "****"
	}

	// Show first 3 chars and last 2 chars
	if len(key) <= 12 {
		return key[:3] + "****" + key[len(key)-2:]
	}

	// Show first 3 chars and last 4 chars
	return key[:3] + "****" + key[len(key)-4:]
}

// handleTestModel tests connectivity to a model endpoint.
//
//	POST /api/models/{index}/test
func (h *Handler) handleTestModel(w http.ResponseWriter, r *http.Request) {
	idx, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "Invalid index", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	if idx < 0 || idx >= len(cfg.ModelList) {
		http.Error(w, fmt.Sprintf("Index %d out of range (0-%d)", idx, len(cfg.ModelList)-1), http.StatusNotFound)
		return
	}

	m := cfg.ModelList[idx]
	start := time.Now()
	summary := modelConfigurationStatus(m)
	latency := time.Since(start).Milliseconds()

	result := map[string]any{
		"success":    summary.Available,
		"latency_ms": latency,
		"status":     summary.Status,
	}

	if !summary.Available {
		if summary.Status == modelStatusUnconfigured {
			result["error"] = "API key not configured"
		} else {
			result["error"] = "Endpoint unreachable"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleFetchModels fetches available models from an upstream provider.
//
//	POST /api/models/fetch
func (h *Handler) handleFetchModels(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		Provider string `json:"provider"`
		APIKey   string `json:"api_key"`
		APIBase  string `json:"api_base"`
	}
	if err = json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.Provider == "" {
		http.Error(w, "provider is required", http.StatusBadRequest)
		return
	}

	apiBase := strings.TrimSpace(req.APIBase)
	if apiBase == "" {
		apiBase = providers.DefaultAPIBaseForProtocol(req.Provider)
	}
	if apiBase == "" {
		http.Error(w, fmt.Sprintf("No default API base for provider %q", req.Provider), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	models, err := fetchUpstreamModels(ctx, req.Provider, apiBase, req.APIKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch models: %v", err), http.StatusBadGateway)
		return
	}

	// Auto-save fetched models to catalog
	catalogModels := make([]CatalogModel, len(models))
	for i, m := range models {
		catalogModels[i] = CatalogModel{ID: m.ID, OwnedBy: m.OwnedBy}
	}
	if saveErr := SaveCatalog(req.Provider, apiBase, req.APIKey, catalogModels); saveErr != nil {
		// Log but don't fail the request — saving catalog is non-critical
		logger.Warnf("Failed to save model catalog: %v", saveErr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"models": models,
		"total":  len(models),
	})
}

type upstreamModel struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by,omitempty"`
}

func fetchUpstreamModels(ctx context.Context, provider, apiBase, apiKey string) ([]upstreamModel, error) {
	apiBase = strings.TrimRight(strings.TrimSpace(apiBase), "/")

	var fetchURL string
	switch strings.ToLower(provider) {
	case "ollama":
		// Strip /v1 suffix if present to get the Ollama root
		root := apiBase
		if strings.HasSuffix(root, "/v1") {
			root = root[:len(root)-3]
		}
		root = strings.TrimRight(root, "/")
		fetchURL = root + "/api/tags"
		return fetchOllamaModels(ctx, fetchURL)
	default:
		// OpenAI-compatible: /v1/models
		fetchURL = apiBase + "/models"
		return fetchOpenAICompatibleModels(ctx, fetchURL, apiKey)
	}
}

func fetchOpenAICompatibleModels(ctx context.Context, fetchURL, apiKey string) ([]upstreamModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, err
	}
	if apiKey = strings.TrimSpace(apiKey); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	var parsed struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	models := make([]upstreamModel, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		if m.ID != "" {
			models = append(models, upstreamModel{ID: m.ID, OwnedBy: m.OwnedBy})
		}
	}
	return models, nil
}

func fetchOllamaModels(ctx context.Context, fetchURL string) ([]upstreamModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var parsed struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	models := make([]upstreamModel, 0, len(parsed.Models))
	for _, m := range parsed.Models {
		id := m.Name
		if id == "" {
			id = m.Model
		}
		if id != "" {
			models = append(models, upstreamModel{ID: id})
		}
	}
	return models, nil
}
