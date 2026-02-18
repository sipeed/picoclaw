// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

// createClaudeAuthProvider creates a Claude provider using OAuth credentials.
func createClaudeAuthProvider() (LLMProvider, error) {
	cred, err := auth.GetCredential("anthropic")
	if err != nil {
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("no credentials for anthropic. Run: picoclaw auth login --provider anthropic")
	}
	return NewClaudeProviderWithTokenSource(cred.AccessToken, createClaudeTokenSource()), nil
}

// createCodexAuthProvider creates a Codex provider using OAuth credentials.
func createCodexAuthProvider() (LLMProvider, error) {
	cred, err := auth.GetCredential("openai")
	if err != nil {
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("no credentials for openai. Run: picoclaw auth login --provider openai")
	}
	return NewCodexProviderWithTokenSource(cred.AccessToken, cred.AccountID, createCodexTokenSource()), nil
}

// CreateProvider creates a provider based on the configuration.
// It supports both the new model_list configuration and the legacy providers configuration.
// Returns the provider, the model ID to use, and any error.
func CreateProvider(cfg *config.Config) (LLMProvider, string, error) {
	model := cfg.Agents.Defaults.Model

	// First, try to use model_list configuration
	if len(cfg.ModelList) > 0 {
		// Try to get config by model name first
		modelCfg, err := cfg.GetModelConfig(model)
		if err == nil {
			// Found in model_list, use factory to create provider
			provider, modelID, err := CreateProviderFromConfig(modelCfg)
			if err != nil {
				return nil, "", fmt.Errorf("failed to create provider from model_list: %w", err)
			}
			return provider, modelID, nil
		}
		// Model not found in model_list, fall through to providers config
	}

	// Log deprecation warning if using old providers config
	if cfg.HasProvidersConfig() && len(cfg.ModelList) == 0 {
		fmt.Println("WARNING: providers config is deprecated, please migrate to model_list")
	}

	providerName := strings.ToLower(cfg.Agents.Defaults.Provider)

	var apiKey, apiBase, proxy string

	lowerModel := strings.ToLower(model)

	// First, try to use explicitly configured provider
	if providerName != "" {
		switch providerName {
		case "groq":
			if cfg.Providers.Groq.APIKey != "" {
				apiKey = cfg.Providers.Groq.APIKey
				apiBase = cfg.Providers.Groq.APIBase
				if apiBase == "" {
					apiBase = "https://api.groq.com/openai/v1"
				}
			}
		case "openai", "gpt":
			if cfg.Providers.OpenAI.APIKey != "" || cfg.Providers.OpenAI.AuthMethod != "" {
				if cfg.Providers.OpenAI.AuthMethod == "codex-cli" {
					return NewCodexProviderWithTokenSource("", "", CreateCodexCliTokenSource()), model, nil
				}
				if cfg.Providers.OpenAI.AuthMethod == "oauth" || cfg.Providers.OpenAI.AuthMethod == "token" {
					provider, err := createCodexAuthProvider()
					return provider, model, err
				}
				apiKey = cfg.Providers.OpenAI.APIKey
				apiBase = cfg.Providers.OpenAI.APIBase
				if apiBase == "" {
					apiBase = "https://api.openai.com/v1"
				}
			}
		case "anthropic", "claude":
			if cfg.Providers.Anthropic.APIKey != "" || cfg.Providers.Anthropic.AuthMethod != "" {
				if cfg.Providers.Anthropic.AuthMethod == "oauth" || cfg.Providers.Anthropic.AuthMethod == "token" {
					provider, err := createClaudeAuthProvider()
					return provider, model, err
				}
				apiKey = cfg.Providers.Anthropic.APIKey
				apiBase = cfg.Providers.Anthropic.APIBase
				if apiBase == "" {
					apiBase = "https://api.anthropic.com/v1"
				}
			}
		case "openrouter":
			if cfg.Providers.OpenRouter.APIKey != "" {
				apiKey = cfg.Providers.OpenRouter.APIKey
				if cfg.Providers.OpenRouter.APIBase != "" {
					apiBase = cfg.Providers.OpenRouter.APIBase
				} else {
					apiBase = "https://openrouter.ai/api/v1"
				}
			}
		case "zhipu", "glm":
			if cfg.Providers.Zhipu.APIKey != "" {
				apiKey = cfg.Providers.Zhipu.APIKey
				apiBase = cfg.Providers.Zhipu.APIBase
				if apiBase == "" {
					apiBase = "https://open.bigmodel.cn/api/paas/v4"
				}
			}
		case "gemini", "google":
			if cfg.Providers.Gemini.APIKey != "" {
				apiKey = cfg.Providers.Gemini.APIKey
				apiBase = cfg.Providers.Gemini.APIBase
				if apiBase == "" {
					apiBase = "https://generativelanguage.googleapis.com/v1beta"
				}
			}
		case "vllm":
			if cfg.Providers.VLLM.APIBase != "" {
				apiKey = cfg.Providers.VLLM.APIKey
				apiBase = cfg.Providers.VLLM.APIBase
			}
		case "shengsuanyun":
			if cfg.Providers.ShengSuanYun.APIKey != "" {
				apiKey = cfg.Providers.ShengSuanYun.APIKey
				apiBase = cfg.Providers.ShengSuanYun.APIBase
				if apiBase == "" {
					apiBase = "https://router.shengsuanyun.com/api/v1"
				}
			}
		case "claude-cli", "claudecode", "claude-code":
			workspace := cfg.WorkspacePath()
			if workspace == "" {
				workspace = "."
			}
			return NewClaudeCliProvider(workspace), model, nil
		case "codex-cli", "codex-code":
			workspace := cfg.WorkspacePath()
			if workspace == "" {
				workspace = "."
			}
			return NewCodexCliProvider(workspace), model, nil
		case "cerebras":
			if cfg.Providers.Cerebras.APIKey != "" {
				apiKey = cfg.Providers.Cerebras.APIKey
				apiBase = cfg.Providers.Cerebras.APIBase
				if apiBase == "" {
					apiBase = "https://api.cerebras.ai/v1"
				}
			}
		case "deepseek":
			if cfg.Providers.DeepSeek.APIKey != "" {
				apiKey = cfg.Providers.DeepSeek.APIKey
				apiBase = cfg.Providers.DeepSeek.APIBase
				if apiBase == "" {
					apiBase = "https://api.deepseek.com/v1"
				}
				if model != "deepseek-chat" && model != "deepseek-reasoner" {
					model = "deepseek-chat"
				}
			}
		case "qwen":
			if cfg.Providers.Qwen.APIKey != "" {
				apiKey = cfg.Providers.Qwen.APIKey
				apiBase = cfg.Providers.Qwen.APIBase
				if apiBase == "" {
					apiBase = "https://dashscope.aliyuncs.com/compatible-mode/v1"
				}
			}
		case "github_copilot", "copilot":
			if cfg.Providers.GitHubCopilot.APIBase != "" {
				apiBase = cfg.Providers.GitHubCopilot.APIBase
			} else {
				apiBase = "localhost:4321"
			}
			provider, err := NewGitHubCopilotProvider(apiBase, cfg.Providers.GitHubCopilot.ConnectMode, model)
			return provider, model, err
		case "antigravity", "google-antigravity":
			return NewAntigravityProvider(), model, nil

		case "volcengine", "doubao":
			if cfg.Providers.VolcEngine.APIKey != "" {
				apiKey = cfg.Providers.VolcEngine.APIKey
				apiBase = cfg.Providers.VolcEngine.APIBase
				if apiBase == "" {
					apiBase = "https://ark.cn-beijing.volces.com/api/v3"
				}
			}

		}

	}

	// Fallback: detect provider from model name
	if apiKey == "" && apiBase == "" {
		switch {
		case (strings.Contains(lowerModel, "kimi") || strings.Contains(lowerModel, "moonshot") || strings.HasPrefix(model, "moonshot/")) && cfg.Providers.Moonshot.APIKey != "":
			apiKey = cfg.Providers.Moonshot.APIKey
			apiBase = cfg.Providers.Moonshot.APIBase
			proxy = cfg.Providers.Moonshot.Proxy
			if apiBase == "" {
				apiBase = "https://api.moonshot.cn/v1"
			}

		case strings.HasPrefix(model, "openrouter/") || strings.HasPrefix(model, "anthropic/") || strings.HasPrefix(model, "openai/") || strings.HasPrefix(model, "meta-llama/") || strings.HasPrefix(model, "deepseek/") || strings.HasPrefix(model, "google/"):
			apiKey = cfg.Providers.OpenRouter.APIKey
			proxy = cfg.Providers.OpenRouter.Proxy
			if cfg.Providers.OpenRouter.APIBase != "" {
				apiBase = cfg.Providers.OpenRouter.APIBase
			} else {
				apiBase = "https://openrouter.ai/api/v1"
			}

		case (strings.Contains(lowerModel, "claude") || strings.HasPrefix(model, "anthropic/")) && (cfg.Providers.Anthropic.APIKey != "" || cfg.Providers.Anthropic.AuthMethod != ""):
			if cfg.Providers.Anthropic.AuthMethod == "oauth" || cfg.Providers.Anthropic.AuthMethod == "token" {
				provider, err := createClaudeAuthProvider()
				return provider, model, err
			}
			apiKey = cfg.Providers.Anthropic.APIKey
			apiBase = cfg.Providers.Anthropic.APIBase
			proxy = cfg.Providers.Anthropic.Proxy
			if apiBase == "" {
				apiBase = "https://api.anthropic.com/v1"
			}

		case (strings.Contains(lowerModel, "gpt") || strings.HasPrefix(model, "openai/")) && (cfg.Providers.OpenAI.APIKey != "" || cfg.Providers.OpenAI.AuthMethod != ""):
			if cfg.Providers.OpenAI.AuthMethod == "oauth" || cfg.Providers.OpenAI.AuthMethod == "token" {
				provider, err := createCodexAuthProvider()
				return provider, model, err
			}
			apiKey = cfg.Providers.OpenAI.APIKey
			apiBase = cfg.Providers.OpenAI.APIBase
			proxy = cfg.Providers.OpenAI.Proxy
			if apiBase == "" {
				apiBase = "https://api.openai.com/v1"
			}

		case (strings.Contains(lowerModel, "gemini") || strings.HasPrefix(model, "google/")) && cfg.Providers.Gemini.APIKey != "":
			apiKey = cfg.Providers.Gemini.APIKey
			apiBase = cfg.Providers.Gemini.APIBase
			proxy = cfg.Providers.Gemini.Proxy
			if apiBase == "" {
				apiBase = "https://generativelanguage.googleapis.com/v1beta"
			}

		case (strings.Contains(lowerModel, "glm") || strings.Contains(lowerModel, "zhipu") || strings.Contains(lowerModel, "zai")) && cfg.Providers.Zhipu.APIKey != "":
			apiKey = cfg.Providers.Zhipu.APIKey
			apiBase = cfg.Providers.Zhipu.APIBase
			proxy = cfg.Providers.Zhipu.Proxy
			if apiBase == "" {
				apiBase = "https://open.bigmodel.cn/api/paas/v4"
			}

		case (strings.Contains(lowerModel, "groq") || strings.HasPrefix(model, "groq/")) && cfg.Providers.Groq.APIKey != "":
			apiKey = cfg.Providers.Groq.APIKey
			apiBase = cfg.Providers.Groq.APIBase
			proxy = cfg.Providers.Groq.Proxy
			if apiBase == "" {
				apiBase = "https://api.groq.com/openai/v1"
			}

		case (strings.Contains(lowerModel, "qwen") || strings.HasPrefix(model, "qwen/")) && cfg.Providers.Qwen.APIKey != "":
			apiKey = cfg.Providers.Qwen.APIKey
			apiBase = cfg.Providers.Qwen.APIBase
			proxy = cfg.Providers.Qwen.Proxy
			if apiBase == "" {
				apiBase = "https://dashscope.aliyuncs.com/compatible-mode/v1"
			}

		case (strings.Contains(lowerModel, "nvidia") || strings.HasPrefix(model, "nvidia/")) && cfg.Providers.Nvidia.APIKey != "":
			apiKey = cfg.Providers.Nvidia.APIKey
			apiBase = cfg.Providers.Nvidia.APIBase
			proxy = cfg.Providers.Nvidia.Proxy
			if apiBase == "" {
				apiBase = "https://integrate.api.nvidia.com/v1"
			}
		case (strings.Contains(lowerModel, "cerebras") || strings.HasPrefix(model, "cerebras/")) && cfg.Providers.Cerebras.APIKey != "":
			apiKey = cfg.Providers.Cerebras.APIKey
			apiBase = cfg.Providers.Cerebras.APIBase
			proxy = cfg.Providers.Cerebras.Proxy
			if apiBase == "" {
				apiBase = "https://api.cerebras.ai/v1"
			}

		case (strings.Contains(lowerModel, "ollama") || strings.HasPrefix(model, "ollama/")) && cfg.Providers.Ollama.APIKey != "":
			fmt.Println("Ollama provider selected based on model name prefix")
			apiKey = cfg.Providers.Ollama.APIKey
			apiBase = cfg.Providers.Ollama.APIBase
			proxy = cfg.Providers.Ollama.Proxy
			if apiBase == "" {
				apiBase = "http://localhost:11434/v1"
			}
			fmt.Println("Ollama apiBase:", apiBase)

		case (strings.Contains(lowerModel, "doubao") || strings.HasPrefix(lowerModel, "doubao") || strings.Contains(lowerModel, "volcengine")) && cfg.Providers.VolcEngine.APIKey != "":
			apiKey = cfg.Providers.VolcEngine.APIKey
			apiBase = cfg.Providers.VolcEngine.APIBase
			proxy = cfg.Providers.VolcEngine.Proxy
			if apiBase == "" {
				apiBase = "https://ark.cn-beijing.volces.com/api/v3"
			}

		case cfg.Providers.VLLM.APIBase != "":
			apiKey = cfg.Providers.VLLM.APIKey
			apiBase = cfg.Providers.VLLM.APIBase
			proxy = cfg.Providers.VLLM.Proxy

		default:
			if cfg.Providers.OpenRouter.APIKey != "" {
				apiKey = cfg.Providers.OpenRouter.APIKey
				proxy = cfg.Providers.OpenRouter.Proxy
				if cfg.Providers.OpenRouter.APIBase != "" {
					apiBase = cfg.Providers.OpenRouter.APIBase
				} else {
					apiBase = "https://openrouter.ai/api/v1"
				}
			} else {
				return nil, "", fmt.Errorf("no API key configured for model: %s", model)
			}
		}
	}

	if apiKey == "" && !strings.HasPrefix(model, "bedrock/") {
		return nil, "", fmt.Errorf("no API key configured for provider (model: %s)", model)
	}

	if apiBase == "" {
		return nil, "", fmt.Errorf("no API base configured for provider (model: %s)", model)
	}

	return NewHTTPProvider(apiKey, apiBase, proxy), model, nil
}
