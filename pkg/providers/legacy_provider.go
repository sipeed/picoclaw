// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// ResolveModelConfig resolves an alias or raw model reference to a configured
// provider. Raw references may reuse credentials and endpoint settings from
// another model entry for the same provider.
func ResolveModelConfig(cfg *config.Config, model string) (*config.ModelConfig, error) {
	model = strings.TrimSpace(model)
	if cfg == nil || model == "" {
		return nil, fmt.Errorf("model is required")
	}
	defaultProvider := strings.TrimSpace(cfg.Agents.Defaults.Provider)
	if defaultProvider == "" {
		defaultProvider = "openai"
	}
	if modelCfg, err := cfg.GetModelConfig(model); err == nil && modelCfg != nil {
		return cloneResolvedModelConfig(modelCfg, defaultProvider), nil
	}

	ref := ParseModelRef(model, defaultProvider)
	if ref == nil || strings.TrimSpace(ref.Provider) == "" || strings.TrimSpace(ref.Model) == "" {
		return nil, fmt.Errorf("model %q is invalid", model)
	}

	targetKey := ModelKey(ref.Provider, ref.Model)
	explicitRef := ParseModelRef(model, "")
	rawHasProvider := explicitRef != nil && strings.TrimSpace(explicitRef.Provider) != ""
	var exactMatch *config.ModelConfig
	exactMatchScore := -1
	var legacyMatch *config.ModelConfig
	legacyMatchScore := -1
	var canonicalMatch *config.ModelConfig
	canonicalMatchScore := -1
	var providerTemplate *config.ModelConfig
	providerTemplateScore := -1
	for _, candidate := range cfg.ModelList {
		if candidate == nil || candidate.IsVirtual() {
			continue
		}
		score := modelConfigTemplateScore(candidate, defaultProvider)
		if rawHasProvider && strings.TrimSpace(candidate.Model) == model && score > exactMatchScore {
			exactMatch = candidate
			exactMatchScore = score
		}
		if rawHasProvider &&
			strings.TrimSpace(candidate.Provider) == "" &&
			strings.TrimSpace(candidate.Model) == NormalizeProvider(defaultProvider)+"/"+model &&
			score > legacyMatchScore {
			legacyMatch = candidate
			legacyMatchScore = score
		}
		provider, modelID := modelConfigProviderAndID(candidate, defaultProvider)
		if ModelKey(provider, modelID) == targetKey && score > canonicalMatchScore {
			canonicalMatch = candidate
			canonicalMatchScore = score
		}
		if NormalizeProvider(provider) == NormalizeProvider(ref.Provider) {
			if score > providerTemplateScore {
				providerTemplate = candidate
				providerTemplateScore = score
			}
		}
	}
	if exactMatch != nil {
		return cloneResolvedModelConfig(exactMatch, defaultProvider), nil
	}
	if legacyMatch != nil {
		return cloneResolvedModelConfig(legacyMatch, defaultProvider), nil
	}
	if canonicalMatch != nil {
		return cloneResolvedModelConfig(canonicalMatch, defaultProvider), nil
	}
	if providerTemplate == nil {
		return nil, fmt.Errorf("model %q has no configured provider", model)
	}

	clone := *providerTemplate
	clone.ModelName = model
	clone.Provider = NormalizeProvider(ref.Provider)
	clone.Model = ref.Model
	clone.Fallbacks = nil
	clone.RPM = 0
	clone.MaxTokensField = ""
	clone.ThinkingLevel = ""
	clone.ToolSchemaTransform = ""
	clone.Streaming = config.ModelStreamingConfig{}
	clone.ExtraBody = nil
	return &clone, nil
}

func cloneResolvedModelConfig(modelCfg *config.ModelConfig, defaultProvider string) *config.ModelConfig {
	clone := *modelCfg
	if strings.TrimSpace(clone.Provider) == "" {
		clone.Provider, clone.Model = modelConfigProviderAndID(modelCfg, defaultProvider)
	}
	return &clone
}

func modelConfigProviderAndID(modelCfg *config.ModelConfig, defaultProvider string) (string, string) {
	if modelCfg == nil {
		return "", ""
	}
	if strings.TrimSpace(modelCfg.Provider) != "" {
		return ExtractProtocol(modelCfg)
	}
	return SplitModelProviderAndID(modelCfg.Model, defaultProvider)
}

func modelConfigTemplateScore(modelCfg *config.ModelConfig, defaultProvider string) int {
	if modelCfg == nil {
		return -1
	}
	score := 0
	if strings.TrimSpace(modelCfg.APIKey()) != "" {
		score += 32
	}
	if strings.TrimSpace(modelCfg.APIBase) != "" {
		authMethod := strings.ToLower(strings.TrimSpace(modelCfg.AuthMethod))
		if authMethod != "oauth" && authMethod != "token" {
			provider, _ := modelConfigProviderAndID(modelCfg, defaultProvider)
			defaultBase := strings.TrimRight(DefaultAPIBaseForProtocol(provider), "/")
			if defaultBase == "" || strings.TrimRight(strings.TrimSpace(modelCfg.APIBase), "/") != defaultBase {
				score += 16
			}
		}
	}
	if strings.TrimSpace(modelCfg.AuthMethod) != "" || strings.TrimSpace(modelCfg.ConnectMode) != "" {
		score += 8
	}
	if modelCfg.Enabled {
		score++
	}
	return score
}

// CreateProvider creates a provider based on the configuration.
// It uses the model_list configuration (new format) to create providers.
// The old providers config is automatically converted to model_list during config loading.
// Returns the provider, the model ID to use, and any error.
func CreateProvider(cfg *config.Config) (LLMProvider, string, error) {
	model := cfg.Agents.Defaults.GetModelName()

	// Must have model_list at this point
	if len(cfg.ModelList) == 0 {
		return nil, "", fmt.Errorf("no providers configured. Please add entries to model_list in your config")
	}

	modelCfg, err := ResolveModelConfig(cfg, model)
	if err != nil {
		return nil, "", fmt.Errorf("model %q not found in model_list: %w", model, err)
	}

	// Inject global workspace if not set in model config
	if modelCfg.Workspace == "" {
		modelCfg.Workspace = cfg.WorkspacePath()
	}

	// Use factory to create provider
	provider, modelID, err := CreateProviderFromConfig(modelCfg)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create provider for model %q: %w", model, err)
	}

	return provider, modelID, nil
}
