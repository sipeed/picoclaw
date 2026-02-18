// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

// ConvertProvidersToModelList converts the old ProvidersConfig to a slice of ModelConfig.
// This enables backward compatibility with existing configurations.
func ConvertProvidersToModelList(cfg *Config) []ModelConfig {
	if cfg == nil {
		return nil
	}

	var result []ModelConfig
	p := cfg.Providers

	// OpenAI
	if p.OpenAI.APIKey != "" || p.OpenAI.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName:  "openai",
			Model:      "openai/gpt-4o",
			APIKey:     p.OpenAI.APIKey,
			APIBase:    p.OpenAI.APIBase,
			Proxy:      p.OpenAI.Proxy,
			AuthMethod: p.OpenAI.AuthMethod,
		})
	}

	// Anthropic
	if p.Anthropic.APIKey != "" || p.Anthropic.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName:  "anthropic",
			Model:      "anthropic/claude-3-sonnet",
			APIKey:     p.Anthropic.APIKey,
			APIBase:    p.Anthropic.APIBase,
			Proxy:      p.Anthropic.Proxy,
			AuthMethod: p.Anthropic.AuthMethod,
		})
	}

	// OpenRouter
	if p.OpenRouter.APIKey != "" || p.OpenRouter.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName: "openrouter",
			Model:     "openrouter/auto",
			APIKey:    p.OpenRouter.APIKey,
			APIBase:   p.OpenRouter.APIBase,
			Proxy:     p.OpenRouter.Proxy,
		})
	}

	// Groq
	if p.Groq.APIKey != "" || p.Groq.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName: "groq",
			Model:     "groq/llama-3.1-70b-versatile",
			APIKey:    p.Groq.APIKey,
			APIBase:   p.Groq.APIBase,
			Proxy:     p.Groq.Proxy,
		})
	}

	// Zhipu
	if p.Zhipu.APIKey != "" || p.Zhipu.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName: "zhipu",
			Model:     "openai/glm-4",
			APIKey:    p.Zhipu.APIKey,
			APIBase:   p.Zhipu.APIBase,
			Proxy:     p.Zhipu.Proxy,
		})
	}

	// VLLM
	if p.VLLM.APIKey != "" || p.VLLM.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName: "vllm",
			Model:     "openai/auto",
			APIKey:    p.VLLM.APIKey,
			APIBase:   p.VLLM.APIBase,
			Proxy:     p.VLLM.Proxy,
		})
	}

	// Gemini
	if p.Gemini.APIKey != "" || p.Gemini.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName: "gemini",
			Model:     "openai/gemini-pro",
			APIKey:    p.Gemini.APIKey,
			APIBase:   p.Gemini.APIBase,
			Proxy:     p.Gemini.Proxy,
		})
	}

	// Nvidia
	if p.Nvidia.APIKey != "" || p.Nvidia.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName: "nvidia",
			Model:     "nvidia/meta/llama-3.1-8b-instruct",
			APIKey:    p.Nvidia.APIKey,
			APIBase:   p.Nvidia.APIBase,
			Proxy:     p.Nvidia.Proxy,
		})
	}

	// Ollama
	if p.Ollama.APIKey != "" || p.Ollama.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName: "ollama",
			Model:     "ollama/llama3",
			APIKey:    p.Ollama.APIKey,
			APIBase:   p.Ollama.APIBase,
			Proxy:     p.Ollama.Proxy,
		})
	}

	// Moonshot
	if p.Moonshot.APIKey != "" || p.Moonshot.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName: "moonshot",
			Model:     "moonshot/kimi",
			APIKey:    p.Moonshot.APIKey,
			APIBase:   p.Moonshot.APIBase,
			Proxy:     p.Moonshot.Proxy,
		})
	}

	// ShengSuanYun
	if p.ShengSuanYun.APIKey != "" || p.ShengSuanYun.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName: "shengsuanyun",
			Model:     "openai/auto",
			APIKey:    p.ShengSuanYun.APIKey,
			APIBase:   p.ShengSuanYun.APIBase,
			Proxy:     p.ShengSuanYun.Proxy,
		})
	}

	// DeepSeek
	if p.DeepSeek.APIKey != "" || p.DeepSeek.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName: "deepseek",
			Model:     "openai/deepseek-chat",
			APIKey:    p.DeepSeek.APIKey,
			APIBase:   p.DeepSeek.APIBase,
			Proxy:     p.DeepSeek.Proxy,
		})
	}

	// Cerebras
	if p.Cerebras.APIKey != "" || p.Cerebras.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName: "cerebras",
			Model:     "cerebras/llama-3.3-70b",
			APIKey:    p.Cerebras.APIKey,
			APIBase:   p.Cerebras.APIBase,
			Proxy:     p.Cerebras.Proxy,
		})
	}

	// VolcEngine (Doubao)
	if p.VolcEngine.APIKey != "" || p.VolcEngine.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName: "volcengine",
			Model:     "openai/doubao-pro",
			APIKey:    p.VolcEngine.APIKey,
			APIBase:   p.VolcEngine.APIBase,
			Proxy:     p.VolcEngine.Proxy,
		})
	}

	// GitHub Copilot
	if p.GitHubCopilot.APIKey != "" || p.GitHubCopilot.APIBase != "" || p.GitHubCopilot.ConnectMode != "" {
		result = append(result, ModelConfig{
			ModelName:   "github-copilot",
			Model:       "github-copilot/gpt-4o",
			APIBase:     p.GitHubCopilot.APIBase,
			ConnectMode: p.GitHubCopilot.ConnectMode,
		})
	}

	// Antigravity
	if p.Antigravity.APIKey != "" || p.Antigravity.AuthMethod != "" {
		result = append(result, ModelConfig{
			ModelName:  "antigravity",
			Model:      "antigravity/gemini-2.0-flash",
			APIKey:     p.Antigravity.APIKey,
			AuthMethod: p.Antigravity.AuthMethod,
		})
	}

	// Qwen
	if p.Qwen.APIKey != "" || p.Qwen.APIBase != "" {
		result = append(result, ModelConfig{
			ModelName: "qwen",
			Model:     "qwen/qwen-max",
			APIKey:    p.Qwen.APIKey,
			APIBase:   p.Qwen.APIBase,
			Proxy:     p.Qwen.Proxy,
		})
	}

	return result
}
