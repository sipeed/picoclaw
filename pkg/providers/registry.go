// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"fmt"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
)

// ProviderRegistry manages provider instances for different provider names.
// It lazily creates providers on-demand and caches them for reuse.
// This is essential for proper fallback behavior where different candidates
// may use different providers (e.g., cerebras, ollama, openai).
type ProviderRegistry struct {
	cfg       *config.Config
	modelList []config.ModelConfig
	providers map[string]LLMProvider
	mu        sync.RWMutex
}

// NewProviderRegistry creates a new provider registry with the given config.
func NewProviderRegistry(cfg *config.Config) *ProviderRegistry {
	var modelList []config.ModelConfig
	if cfg != nil {
		modelList = cfg.ModelList
	}
	return &ProviderRegistry{
		cfg:       cfg,
		modelList: modelList,
		providers: make(map[string]LLMProvider),
	}
}

// GetProvider returns a provider for the given provider name (e.g., "openai", "cerebras", "ollama").
// If the provider has already been created, it returns the cached instance.
// Otherwise, it creates a new provider from the model list config.
func (pr *ProviderRegistry) GetProvider(providerName string) (LLMProvider, error) {
	// Normalize provider name (lowercase)
	providerName = strings.ToLower(providerName)

	// Check cache first
	pr.mu.RLock()
	if provider, ok := pr.providers[providerName]; ok {
		pr.mu.RUnlock()
		return provider, nil
	}
	pr.mu.RUnlock()

	// Find the model config for this provider
	var modelCfg *config.ModelConfig
	for i := range pr.modelList {
		cfg := &pr.modelList[i]
		// Extract provider from model string (e.g., "cerebras/gpt-oss-120b" -> "cerebras")
		protocol, _ := ExtractProtocol(cfg.Model)
		if strings.EqualFold(protocol, providerName) {
			modelCfg = cfg
			break
		}
	}

	// If not found in model list, try creating with protocol-only config
	if modelCfg == nil {
		modelCfg = &config.ModelConfig{
			Model: providerName + "/dummy", // Protocol is what matters
		}
	}

	// Create the provider
	provider, _, err := CreateProviderFromConfig(modelCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider for %q: %w", providerName, err)
	}

	// Cache the provider
	pr.mu.Lock()
	pr.providers[providerName] = provider
	pr.mu.Unlock()

	return provider, nil
}

// GetDefaultProvider returns the provider for the default/primary protocol.
// This is used for backward compatibility with code that expects a single provider.
func (pr *ProviderRegistry) GetDefaultProvider() (LLMProvider, error) {
	if len(pr.modelList) == 0 {
		// No model list configured, return error
		// Caller should handle this by using a default provider
		return nil, fmt.Errorf("no model list configured")
	}

	// Use the first model's provider as default
	protocol, _ := ExtractProtocol(pr.modelList[0].Model)
	return pr.GetProvider(protocol)
}
