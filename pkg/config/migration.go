// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"slices"
	"strings"
)

// buildModelWithProtocol constructs a model string with protocol prefix.
// If the model already contains a "/" (indicating it has a protocol prefix), it is returned as-is.
// Otherwise, the protocol prefix is added.
func buildModelWithProtocol(protocol, model string) string {
	if strings.Contains(model, "/") {
		// Model already has a protocol prefix, return as-is
		return model
	}
	return protocol + "/" + model
}

// providerMigrationConfig defines how to migrate a provider from old config to new format.
type providerMigrationConfig struct {
	// providerNames are the possible names used in agents.defaults.provider
	providerNames []string
	// protocol is the protocol prefix for the model field
	protocol string
	// buildConfig creates the ModelConfig from ProviderConfig
	buildConfig func(p ProvidersConfig) (ModelConfig, bool)
}

// ConvertProvidersToModelList converts the old ProvidersConfig to a slice of ModelConfig.
// This enables backward compatibility with existing configurations.
// It preserves the user's configured model from agents.defaults.model when possible.
func ConvertProvidersToModelList(cfg *Config) []ModelConfig {
	if cfg == nil {
		return nil
	}

	// Get user's configured provider and model
	userProvider := strings.ToLower(cfg.Agents.Defaults.Provider)
	userModel := cfg.Agents.Defaults.GetModelName()

	p := cfg.Providers

	var result []ModelConfig

	// Track if we've applied the legacy model name fix (only for first provider)
	legacyModelNameApplied := false

	// Define migration rules for each provider
	migrations := []providerMigrationConfig{
		{
			providerNames: []string{"openai", "gpt"},
			protocol:      "openai",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.OpenAI.APIKey == "" && p.OpenAI.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName:  "openai",
					Model:      "openai/gpt-5.2",
					APIKey:     p.OpenAI.APIKey,
					APIBase:    p.OpenAI.APIBase,
					Proxy:      p.OpenAI.Proxy,
					AuthMethod: p.OpenAI.AuthMethod,
				}, true
			},
		},
		{
			providerNames: []string{"anthropic", "claude"},
			protocol:      "anthropic",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.Anthropic.APIKey == "" && p.Anthropic.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName:  "anthropic",
					Model:      "anthropic/claude-sonnet-4.6",
					APIKey:     p.Anthropic.APIKey,
					APIBase:    p.Anthropic.APIBase,
					Proxy:      p.Anthropic.Proxy,
					AuthMethod: p.Anthropic.AuthMethod,
				}, true
			},
		},
		{
			providerNames: []string{"openrouter"},
			protocol:      "openrouter",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.OpenRouter.APIKey == "" && p.OpenRouter.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "openrouter",
					Model:     "openrouter/auto",
					APIKey:    p.OpenRouter.APIKey,
					APIBase:   p.OpenRouter.APIBase,
					Proxy:     p.OpenRouter.Proxy,
				}, true
			},
		},
		{
			providerNames: []string{"groq"},
			protocol:      "groq",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.Groq.APIKey == "" && p.Groq.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "groq",
					Model:     "groq/llama-3.1-70b-versatile",
					APIKey:    p.Groq.APIKey,
					APIBase:   p.Groq.APIBase,
					Proxy:     p.Groq.Proxy,
				}, true
			},
		},
		{
			providerNames: []string{"zhipu", "glm"},
			protocol:      "zhipu",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.Zhipu.APIKey == "" && p.Zhipu.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "zhipu",
					Model:     "zhipu/glm-4",
					APIKey:    p.Zhipu.APIKey,
					APIBase:   p.Zhipu.APIBase,
					Proxy:     p.Zhipu.Proxy,
				}, true
			},
		},
		{
			providerNames: []string{"vllm"},
			protocol:      "vllm",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.VLLM.APIKey == "" && p.VLLM.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "vllm",
					Model:     "vllm/auto",
					APIKey:    p.VLLM.APIKey,
					APIBase:   p.VLLM.APIBase,
					Proxy:     p.VLLM.Proxy,
				}, true
			},
		},
		{
			providerNames: []string{"gemini", "google"},
			protocol:      "gemini",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.Gemini.APIKey == "" && p.Gemini.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "gemini",
					Model:     "gemini/gemini-pro",
					APIKey:    p.Gemini.APIKey,
					APIBase:   p.Gemini.APIBase,
					Proxy:     p.Gemini.Proxy,
				}, true
			},
		},
		{
			providerNames: []string{"nvidia"},
			protocol:      "nvidia",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.Nvidia.APIKey == "" && p.Nvidia.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "nvidia",
					Model:     "nvidia/meta/llama-3.1-8b-instruct",
					APIKey:    p.Nvidia.APIKey,
					APIBase:   p.Nvidia.APIBase,
					Proxy:     p.Nvidia.Proxy,
				}, true
			},
		},
		{
			providerNames: []string{"ollama"},
			protocol:      "ollama",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.Ollama.APIKey == "" && p.Ollama.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "ollama",
					Model:     "ollama/llama3",
					APIKey:    p.Ollama.APIKey,
					APIBase:   p.Ollama.APIBase,
					Proxy:     p.Ollama.Proxy,
				}, true
			},
		},
		{
			providerNames: []string{"moonshot", "kimi"},
			protocol:      "moonshot",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.Moonshot.APIKey == "" && p.Moonshot.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "moonshot",
					Model:     "moonshot/kimi",
					APIKey:    p.Moonshot.APIKey,
					APIBase:   p.Moonshot.APIBase,
					Proxy:     p.Moonshot.Proxy,
				}, true
			},
		},
		{
			providerNames: []string{"shengsuanyun"},
			protocol:      "shengsuanyun",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.ShengSuanYun.APIKey == "" && p.ShengSuanYun.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "shengsuanyun",
					Model:     "shengsuanyun/auto",
					APIKey:    p.ShengSuanYun.APIKey,
					APIBase:   p.ShengSuanYun.APIBase,
					Proxy:     p.ShengSuanYun.Proxy,
				}, true
			},
		},
		{
			providerNames: []string{"deepseek"},
			protocol:      "deepseek",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.DeepSeek.APIKey == "" && p.DeepSeek.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "deepseek",
					Model:     "deepseek/deepseek-chat",
					APIKey:    p.DeepSeek.APIKey,
					APIBase:   p.DeepSeek.APIBase,
					Proxy:     p.DeepSeek.Proxy,
				}, true
			},
		},
		{
			providerNames: []string{"cerebras"},
			protocol:      "cerebras",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.Cerebras.APIKey == "" && p.Cerebras.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "cerebras",
					Model:     "cerebras/llama-3.3-70b",
					APIKey:    p.Cerebras.APIKey,
					APIBase:   p.Cerebras.APIBase,
					Proxy:     p.Cerebras.Proxy,
				}, true
			},
		},
		{
			providerNames: []string{"volcengine", "doubao"},
			protocol:      "volcengine",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.VolcEngine.APIKey == "" && p.VolcEngine.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "volcengine",
					Model:     "volcengine/doubao-pro",
					APIKey:    p.VolcEngine.APIKey,
					APIBase:   p.VolcEngine.APIBase,
					Proxy:     p.VolcEngine.Proxy,
				}, true
			},
		},
		{
			providerNames: []string{"github_copilot", "copilot"},
			protocol:      "github-copilot",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.GitHubCopilot.APIKey == "" && p.GitHubCopilot.APIBase == "" && p.GitHubCopilot.ConnectMode == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName:   "github-copilot",
					Model:       "github-copilot/gpt-5.2",
					APIBase:     p.GitHubCopilot.APIBase,
					ConnectMode: p.GitHubCopilot.ConnectMode,
				}, true
			},
		},
		{
			providerNames: []string{"antigravity"},
			protocol:      "antigravity",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.Antigravity.APIKey == "" && p.Antigravity.AuthMethod == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName:  "antigravity",
					Model:      "antigravity/gemini-2.0-flash",
					APIKey:     p.Antigravity.APIKey,
					AuthMethod: p.Antigravity.AuthMethod,
				}, true
			},
		},
		{
			providerNames: []string{"qwen", "tongyi"},
			protocol:      "qwen",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.Qwen.APIKey == "" && p.Qwen.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "qwen",
					Model:     "qwen/qwen-max",
					APIKey:    p.Qwen.APIKey,
					APIBase:   p.Qwen.APIBase,
					Proxy:     p.Qwen.Proxy,
				}, true
			},
		},
		{
			providerNames: []string{"mistral"},
			protocol:      "mistral",
			buildConfig: func(p ProvidersConfig) (ModelConfig, bool) {
				if p.Mistral.APIKey == "" && p.Mistral.APIBase == "" {
					return ModelConfig{}, false
				}
				return ModelConfig{
					ModelName: "mistral",
					Model:     "mistral/mistral-small-latest",
					APIKey:    p.Mistral.APIKey,
					APIBase:   p.Mistral.APIBase,
					Proxy:     p.Mistral.Proxy,
				}, true
			},
		},
	}

	// Process each provider migration
	for _, m := range migrations {
		mc, ok := m.buildConfig(p)
		if !ok {
			continue
		}

		// Check if this is the user's configured provider
		if slices.Contains(m.providerNames, userProvider) && userModel != "" {
			// Use the user's configured model instead of default
			mc.Model = buildModelWithProtocol(m.protocol, userModel)
		} else if userProvider == "" && userModel != "" && !legacyModelNameApplied {
			// Legacy config: no explicit provider field but model is specified
			// Use userModel as ModelName for the FIRST provider so GetModelConfig(model) can find it
			// This maintains backward compatibility with old configs that relied on implicit provider selection
			mc.ModelName = userModel
			mc.Model = buildModelWithProtocol(m.protocol, userModel)
			legacyModelNameApplied = true
		}

		result = append(result, mc)
	}

	return result
}
