// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"fmt"

	"github.com/sipeed/picoclaw/pkg/config"
)

// ModelEntry holds the resolved provider and model ID for a configured model.
type ModelEntry struct {
	Provider    LLMProvider
	ModelID     string // Protocol-stripped model ID sent to Chat()
	ProviderKey string // Protocol string used for cooldown tracking (e.g. "openai", "anthropic")
}

// ModelRegistry is the single source of truth for all configured models.
// It maps model_name → (provider, modelID). Providers with identical configs
// (protocol + api_base + api_key + auth_method) share a single instance.
type ModelRegistry struct {
	models           map[string]*ModelEntry
	defaultModelName string
}

// NewModelRegistry builds a registry from config, creating providers for
// every model_list entry. It is built once at startup and passed through
// to the agent layer.
func NewModelRegistry(cfg *config.Config) (*ModelRegistry, error) {
	// Ensure model_list is populated from legacy providers config
	if cfg.HasProvidersConfig() {
		providerModels := config.ConvertProvidersToModelList(cfg)
		existingModelNames := make(map[string]bool)
		for _, m := range cfg.ModelList {
			existingModelNames[m.ModelName] = true
		}
		for _, pm := range providerModels {
			if !existingModelNames[pm.ModelName] {
				cfg.ModelList = append(cfg.ModelList, pm)
			}
		}
	}

	if len(cfg.ModelList) == 0 {
		return nil, fmt.Errorf("no providers configured. Please add entries to model_list in your config")
	}

	reg := &ModelRegistry{
		models:           make(map[string]*ModelEntry),
		defaultModelName: cfg.Agents.Defaults.GetModelName(),
	}

	// Provider cache: models sharing protocol+apiBase+apiKey+authMethod reuse one provider.
	type providerKey struct {
		protocol   string
		apiBase    string
		apiKey     string
		authMethod string
	}
	cache := make(map[providerKey]LLMProvider)

	for i := range cfg.ModelList {
		mc := &cfg.ModelList[i]
		if mc.Model == "" || mc.ModelName == "" {
			continue
		}

		protocol, modelID := ExtractProtocol(mc.Model)

		apiBase := mc.APIBase
		if apiBase == "" {
			apiBase = getDefaultAPIBase(protocol)
		}

		key := providerKey{
			protocol:   protocol,
			apiBase:    apiBase,
			apiKey:     mc.APIKey,
			authMethod: mc.AuthMethod,
		}

		provider, cached := cache[key]
		if !cached {
			var err error
			provider, _, err = CreateProviderFromConfig(mc)
			if err != nil {
				return nil, fmt.Errorf("failed to create provider for model %q: %w", mc.ModelName, err)
			}
			cache[key] = provider
		}

		providerKeyStr := protocol
		if providerKeyStr == "" {
			providerKeyStr = "openai"
		}
		reg.models[mc.ModelName] = &ModelEntry{
			Provider:    provider,
			ModelID:     modelID,
			ProviderKey: providerKeyStr,
		}
	}

	if len(reg.models) == 0 {
		return nil, fmt.Errorf("no valid models in model_list")
	}

	return reg, nil
}

// Get returns the entry for a model_name.
func (r *ModelRegistry) Get(modelName string) (*ModelEntry, bool) {
	entry, ok := r.models[modelName]
	return entry, ok
}

// GetDefault returns the entry for the configured default model.
func (r *ModelRegistry) GetDefault() (*ModelEntry, bool) {
	return r.Get(r.defaultModelName)
}

// DefaultModelName returns the configured default model name.
func (r *ModelRegistry) DefaultModelName() string {
	return r.defaultModelName
}

// ModelNames returns all registered model names.
func (r *ModelRegistry) ModelNames() []string {
	names := make([]string, 0, len(r.models))
	for name := range r.models {
		names = append(names, name)
	}
	return names
}

// NewModelRegistryFromProvider wraps a single provider into a minimal
// ModelRegistry. Useful for tests and the CLI agent path where a full
// config-driven registry isn't needed.
//
// modelID may include a protocol prefix (e.g. "anthropic/claude-3-5-sonnet").
// The protocol is stripped to form both the registry key (user-facing model name)
// and the ModelID passed to Chat().
func NewModelRegistryFromProvider(provider LLMProvider, modelID string) *ModelRegistry {
	protocol, stripped := ExtractProtocol(modelID)
	name := stripped
	if name == "" {
		name = "default"
	}
	providerKey := protocol
	if providerKey == "" {
		providerKey = "openai"
	}
	return &ModelRegistry{
		models: map[string]*ModelEntry{
			name: {Provider: provider, ModelID: stripped, ProviderKey: providerKey},
		},
		defaultModelName: name,
	}
}

// Close closes any stateful providers in the registry.
func (r *ModelRegistry) Close() {
	closed := make(map[LLMProvider]bool)
	for _, entry := range r.models {
		if closed[entry.Provider] {
			continue
		}
		if sp, ok := entry.Provider.(StatefulProvider); ok {
			sp.Close()
		}
		closed[entry.Provider] = true
	}
}
