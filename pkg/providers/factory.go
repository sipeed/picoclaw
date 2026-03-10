package providers

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

const defaultAnthropicAPIBase = "https://api.anthropic.com/v1"

var getCredential = auth.GetCredential

type providerType int

const (
	providerTypeHTTPCompat providerType = iota
	providerTypeClaudeAuth
	providerTypeCodexAuth
	providerTypeCodexCLIToken
	providerTypeClaudeCLI
	providerTypeCodexCLI
	providerTypeGitHubCopilot
)

type providerSelection struct {
	providerType    providerType
	apiKey          string
	apiBase         string
	proxy           string
	model           string
	workspace       string
	connectMode     string
	enableWebSearch bool
}

func resolveProviderSelection(cfg *config.Config) (providerSelection, error) {
	model := cfg.Agents.Defaults.GetModelName()
	providerName := strings.ToLower(cfg.Agents.Defaults.Provider)
	lowerModel := strings.ToLower(model)

	sel := providerSelection{
		providerType: providerTypeHTTPCompat,
		model:        model,
	}

	// First, prefer explicit provider configuration.
	if providerName != "" {
		if resolver, ok := registry[providerName]; ok {
			s, resolved, err := resolver(cfg, providerName, model)
			if err != nil {
				return providerSelection{}, err
			}
			if resolved {
				return s, nil
			}
		}
	}

	// Fallback: infer provider from model and configured keys.
	if sel.apiKey == "" && sel.apiBase == "" {
		switch {
		case (strings.Contains(lowerModel, "kimi") || strings.Contains(lowerModel, "moonshot") || strings.HasPrefix(model, "moonshot/")) && cfg.Providers.Moonshot.APIKey != "":
			sel.apiKey = cfg.Providers.Moonshot.APIKey
			sel.apiBase = cfg.Providers.Moonshot.APIBase
			sel.proxy = cfg.Providers.Moonshot.Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://api.moonshot.cn/v1"
			}
		case strings.HasPrefix(model, "openrouter/") ||
			strings.HasPrefix(model, "anthropic/") ||
			strings.HasPrefix(model, "openai/") ||
			strings.HasPrefix(model, "meta-llama/") ||
			strings.HasPrefix(model, "deepseek/") ||
			strings.HasPrefix(model, "google/"):
			sel.apiKey = cfg.Providers.OpenRouter.APIKey
			sel.proxy = cfg.Providers.OpenRouter.Proxy
			if cfg.Providers.OpenRouter.APIBase != "" {
				sel.apiBase = cfg.Providers.OpenRouter.APIBase
			} else {
				sel.apiBase = "https://openrouter.ai/api/v1"
			}
		case (strings.Contains(lowerModel, "claude") || strings.HasPrefix(model, "anthropic/")) &&
			(cfg.Providers.Anthropic.APIKey != "" || cfg.Providers.Anthropic.AuthMethod != ""):
			if cfg.Providers.Anthropic.AuthMethod == "oauth" || cfg.Providers.Anthropic.AuthMethod == "token" {
				sel.apiBase = cfg.Providers.Anthropic.APIBase
				if sel.apiBase == "" {
					sel.apiBase = defaultAnthropicAPIBase
				}
				sel.providerType = providerTypeClaudeAuth
				return sel, nil
			}
			sel.apiKey = cfg.Providers.Anthropic.APIKey
			sel.apiBase = cfg.Providers.Anthropic.APIBase
			sel.proxy = cfg.Providers.Anthropic.Proxy
			if sel.apiBase == "" {
				sel.apiBase = defaultAnthropicAPIBase
			}
		case (strings.Contains(lowerModel, "gpt") || strings.HasPrefix(model, "openai/")) &&
			(cfg.Providers.OpenAI.APIKey != "" || cfg.Providers.OpenAI.AuthMethod != ""):
			sel.enableWebSearch = cfg.Providers.OpenAI.WebSearch
			if cfg.Providers.OpenAI.AuthMethod == "codex-cli" {
				sel.providerType = providerTypeCodexCLIToken
				return sel, nil
			}
			if cfg.Providers.OpenAI.AuthMethod == "oauth" || cfg.Providers.OpenAI.AuthMethod == "token" {
				sel.providerType = providerTypeCodexAuth
				return sel, nil
			}
			sel.apiKey = cfg.Providers.OpenAI.APIKey
			sel.apiBase = cfg.Providers.OpenAI.APIBase
			sel.proxy = cfg.Providers.OpenAI.Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://api.openai.com/v1"
			}
		case (strings.Contains(lowerModel, "gemini") || strings.HasPrefix(model, "google/")) && cfg.Providers.Gemini.APIKey != "":
			sel.apiKey = cfg.Providers.Gemini.APIKey
			sel.apiBase = cfg.Providers.Gemini.APIBase
			sel.proxy = cfg.Providers.Gemini.Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://generativelanguage.googleapis.com/v1beta"
			}
		case (strings.Contains(lowerModel, "glm") || strings.Contains(lowerModel, "zhipu") || strings.Contains(lowerModel, "zai")) && cfg.Providers.Zhipu.APIKey != "":
			sel.apiKey = cfg.Providers.Zhipu.APIKey
			sel.apiBase = cfg.Providers.Zhipu.APIBase
			sel.proxy = cfg.Providers.Zhipu.Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://open.bigmodel.cn/api/paas/v4"
			}
		case (strings.Contains(lowerModel, "groq") || strings.HasPrefix(model, "groq/")) && cfg.Providers.Groq.APIKey != "":
			sel.apiKey = cfg.Providers.Groq.APIKey
			sel.apiBase = cfg.Providers.Groq.APIBase
			sel.proxy = cfg.Providers.Groq.Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://api.groq.com/openai/v1"
			}
		case (strings.Contains(lowerModel, "nvidia") || strings.HasPrefix(model, "nvidia/")) && cfg.Providers.Nvidia.APIKey != "":
			sel.apiKey = cfg.Providers.Nvidia.APIKey
			sel.apiBase = cfg.Providers.Nvidia.APIBase
			sel.proxy = cfg.Providers.Nvidia.Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://integrate.api.nvidia.com/v1"
			}
		case strings.HasPrefix(model, "vivgrid/") && cfg.Providers.Vivgrid.APIKey != "":
			sel.apiKey = cfg.Providers.Vivgrid.APIKey
			sel.apiBase = cfg.Providers.Vivgrid.APIBase
			sel.proxy = cfg.Providers.Vivgrid.Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://api.vivgrid.com/v1"
			}
		case (strings.Contains(lowerModel, "ollama") || strings.HasPrefix(model, "ollama/")) && cfg.Providers.Ollama.APIKey != "":
			sel.apiKey = cfg.Providers.Ollama.APIKey
			sel.apiBase = cfg.Providers.Ollama.APIBase
			sel.proxy = cfg.Providers.Ollama.Proxy
			if sel.apiBase == "" {
				sel.apiBase = "http://localhost:11434/v1"
			}
		case (strings.Contains(lowerModel, "mistral") || strings.HasPrefix(model, "mistral/")) && cfg.Providers.Mistral.APIKey != "":
			sel.apiKey = cfg.Providers.Mistral.APIKey
			sel.apiBase = cfg.Providers.Mistral.APIBase
			sel.proxy = cfg.Providers.Mistral.Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://api.mistral.ai/v1"
			}
		case (strings.Contains(lowerModel, "minimax") || strings.HasPrefix(model, "minimax/")) && cfg.Providers.Minimax.APIKey != "":
			sel.apiKey = cfg.Providers.Minimax.APIKey
			sel.apiBase = cfg.Providers.Minimax.APIBase
			sel.proxy = cfg.Providers.Minimax.Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://api.minimaxi.com/v1"
			}
		case strings.HasPrefix(model, "avian/") && cfg.Providers.Avian.APIKey != "":
			sel.apiKey = cfg.Providers.Avian.APIKey
			sel.apiBase = cfg.Providers.Avian.APIBase
			sel.proxy = cfg.Providers.Avian.Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://api.avian.io/v1"
			}
		case cfg.Providers.VLLM.APIBase != "":
			sel.apiKey = cfg.Providers.VLLM.APIKey
			sel.apiBase = cfg.Providers.VLLM.APIBase
			sel.proxy = cfg.Providers.VLLM.Proxy
		default:
			if cfg.Providers.OpenRouter.APIKey != "" {
				sel.apiKey = cfg.Providers.OpenRouter.APIKey
				sel.proxy = cfg.Providers.OpenRouter.Proxy
				if cfg.Providers.OpenRouter.APIBase != "" {
					sel.apiBase = cfg.Providers.OpenRouter.APIBase
				} else {
					sel.apiBase = "https://openrouter.ai/api/v1"
				}
			} else {
				return providerSelection{}, fmt.Errorf("no API key configured for model: %s", model)
			}
		}
	}

	if sel.providerType == providerTypeHTTPCompat {
		if sel.apiKey == "" && !strings.HasPrefix(model, "bedrock/") {
			return providerSelection{}, fmt.Errorf("no API key configured for provider (model: %s)", model)
		}
		if sel.apiBase == "" {
			return providerSelection{}, fmt.Errorf("no API base configured for provider (model: %s)", model)
		}
	}

	return sel, nil
}
