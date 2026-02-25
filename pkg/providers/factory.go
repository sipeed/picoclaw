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

// providerDefaults holds the default API base URL and a config accessor for a standard HTTP-compatible provider.
type providerDefaults struct {
	defaultBase string
	getConfig   func(cfg *config.Config) (apiKey, apiBase, proxy string)
	// hasKey returns true if the provider has credentials configured.
	// If nil, checks that getConfig returns a non-empty apiKey.
	hasKey func(cfg *config.Config) bool
}

// standardProviderRegistry maps provider names to their defaults.
// Only covers providers that follow the standard pattern: apiKey + apiBase + proxy.
// Special-case providers (CLI, OAuth, Copilot) are handled separately.
var standardProviderRegistry = map[string]providerDefaults{
	"groq": {
		defaultBase: "https://api.groq.com/openai/v1",
		getConfig: func(cfg *config.Config) (string, string, string) {
			return cfg.Providers.Groq.APIKey, cfg.Providers.Groq.APIBase, cfg.Providers.Groq.Proxy
		},
	},
	"openrouter": {
		defaultBase: "https://openrouter.ai/api/v1",
		getConfig: func(cfg *config.Config) (string, string, string) {
			return cfg.Providers.OpenRouter.APIKey, cfg.Providers.OpenRouter.APIBase, cfg.Providers.OpenRouter.Proxy
		},
	},
	"zhipu": {
		defaultBase: "https://open.bigmodel.cn/api/paas/v4",
		getConfig: func(cfg *config.Config) (string, string, string) {
			return cfg.Providers.Zhipu.APIKey, cfg.Providers.Zhipu.APIBase, cfg.Providers.Zhipu.Proxy
		},
	},
	"gemini": {
		defaultBase: "https://generativelanguage.googleapis.com/v1beta",
		getConfig: func(cfg *config.Config) (string, string, string) {
			return cfg.Providers.Gemini.APIKey, cfg.Providers.Gemini.APIBase, cfg.Providers.Gemini.Proxy
		},
	},
	"vllm": {
		defaultBase: "", // no default base; requires explicit config
		getConfig: func(cfg *config.Config) (string, string, string) {
			return cfg.Providers.VLLM.APIKey, cfg.Providers.VLLM.APIBase, cfg.Providers.VLLM.Proxy
		},
		hasKey: func(cfg *config.Config) bool {
			return cfg.Providers.VLLM.APIBase != ""
		},
	},
	"shengsuanyun": {
		defaultBase: "https://router.shengsuanyun.com/api/v1",
		getConfig: func(cfg *config.Config) (string, string, string) {
			return cfg.Providers.ShengSuanYun.APIKey, cfg.Providers.ShengSuanYun.APIBase, cfg.Providers.ShengSuanYun.Proxy
		},
	},
	"nvidia": {
		defaultBase: "https://integrate.api.nvidia.com/v1",
		getConfig: func(cfg *config.Config) (string, string, string) {
			return cfg.Providers.Nvidia.APIKey, cfg.Providers.Nvidia.APIBase, cfg.Providers.Nvidia.Proxy
		},
	},
	"deepseek": {
		defaultBase: "https://api.deepseek.com/v1",
		getConfig: func(cfg *config.Config) (string, string, string) {
			return cfg.Providers.DeepSeek.APIKey, cfg.Providers.DeepSeek.APIBase, cfg.Providers.DeepSeek.Proxy
		},
	},
	"mistral": {
		defaultBase: "https://api.mistral.ai/v1",
		getConfig: func(cfg *config.Config) (string, string, string) {
			return cfg.Providers.Mistral.APIKey, cfg.Providers.Mistral.APIBase, cfg.Providers.Mistral.Proxy
		},
	},
	"ollama": {
		defaultBase: "http://localhost:11434/v1",
		getConfig: func(cfg *config.Config) (string, string, string) {
			return cfg.Providers.Ollama.APIKey, cfg.Providers.Ollama.APIBase, cfg.Providers.Ollama.Proxy
		},
	},
	"moonshot": {
		defaultBase: "https://api.moonshot.cn/v1",
		getConfig: func(cfg *config.Config) (string, string, string) {
			return cfg.Providers.Moonshot.APIKey, cfg.Providers.Moonshot.APIBase, cfg.Providers.Moonshot.Proxy
		},
	},
}

// providerNameAliases maps alternative provider names to their canonical names in the registry.
var providerNameAliases = map[string]string{
	"glm":    "zhipu",
	"google": "gemini",
}

// applyStandardProvider applies config from a standardProviderRegistry entry to sel.
// Returns true if the provider had credentials configured.
func applyStandardProvider(cfg *config.Config, sel *providerSelection, entry providerDefaults) bool {
	apiKey, apiBase, proxy := entry.getConfig(cfg)

	hasCredentials := apiKey != ""
	if entry.hasKey != nil {
		hasCredentials = entry.hasKey(cfg)
	}

	if !hasCredentials {
		return false
	}

	sel.apiKey = apiKey
	sel.apiBase = apiBase
	sel.proxy = proxy
	if sel.apiBase == "" && entry.defaultBase != "" {
		sel.apiBase = entry.defaultBase
	}
	return true
}

// modelInferenceEntry maps a model name pattern to a provider and optional match function.
type modelInferenceEntry struct {
	// matches returns true if this entry should handle the given model name (lowercase) and original model.
	matches func(lowerModel, model string, cfg *config.Config) bool
	// apply sets up the provider selection. Returns true on success.
	apply func(cfg *config.Config, sel *providerSelection) bool
}

// modelInferenceRegistry defines fallback model → provider inference rules.
// Order matters: first match wins.
var modelInferenceRegistry = []modelInferenceEntry{
	// Moonshot/Kimi
	{
		matches: func(lm, m string, cfg *config.Config) bool {
			return (strings.Contains(lm, "kimi") || strings.Contains(lm, "moonshot") || strings.HasPrefix(m, "moonshot/")) &&
				cfg.Providers.Moonshot.APIKey != ""
		},
		apply: func(cfg *config.Config, sel *providerSelection) bool {
			return applyStandardProvider(cfg, sel, standardProviderRegistry["moonshot"])
		},
	},
	// OpenRouter-prefixed models (openrouter/, anthropic/, openai/, meta-llama/, deepseek/, google/)
	{
		matches: func(_, m string, _ *config.Config) bool {
			for _, prefix := range []string{"openrouter/", "anthropic/", "openai/", "meta-llama/", "deepseek/", "google/"} {
				if strings.HasPrefix(m, prefix) {
					return true
				}
			}
			return false
		},
		apply: func(cfg *config.Config, sel *providerSelection) bool {
			return applyStandardProvider(cfg, sel, standardProviderRegistry["openrouter"])
		},
	},
	// Claude models → Anthropic (with OAuth support)
	{
		matches: func(lm, m string, cfg *config.Config) bool {
			return (strings.Contains(lm, "claude") || strings.HasPrefix(m, "anthropic/")) &&
				(cfg.Providers.Anthropic.APIKey != "" || cfg.Providers.Anthropic.AuthMethod != "")
		},
		apply: func(cfg *config.Config, sel *providerSelection) bool {
			if cfg.Providers.Anthropic.AuthMethod == "oauth" || cfg.Providers.Anthropic.AuthMethod == "token" {
				sel.apiBase = cfg.Providers.Anthropic.APIBase
				if sel.apiBase == "" {
					sel.apiBase = defaultAnthropicAPIBase
				}
				sel.providerType = providerTypeClaudeAuth
				return true
			}
			sel.apiKey = cfg.Providers.Anthropic.APIKey
			sel.apiBase = cfg.Providers.Anthropic.APIBase
			sel.proxy = cfg.Providers.Anthropic.Proxy
			if sel.apiBase == "" {
				sel.apiBase = defaultAnthropicAPIBase
			}
			return true
		},
	},
	// GPT models → OpenAI (with OAuth/codex-cli support)
	{
		matches: func(lm, m string, cfg *config.Config) bool {
			return (strings.Contains(lm, "gpt") || strings.HasPrefix(m, "openai/")) &&
				(cfg.Providers.OpenAI.APIKey != "" || cfg.Providers.OpenAI.AuthMethod != "")
		},
		apply: func(cfg *config.Config, sel *providerSelection) bool {
			sel.enableWebSearch = cfg.Providers.OpenAI.WebSearch
			if cfg.Providers.OpenAI.AuthMethod == "codex-cli" {
				sel.providerType = providerTypeCodexCLIToken
				return true
			}
			if cfg.Providers.OpenAI.AuthMethod == "oauth" || cfg.Providers.OpenAI.AuthMethod == "token" {
				sel.providerType = providerTypeCodexAuth
				return true
			}
			sel.apiKey = cfg.Providers.OpenAI.APIKey
			sel.apiBase = cfg.Providers.OpenAI.APIBase
			sel.proxy = cfg.Providers.OpenAI.Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://api.openai.com/v1"
			}
			return true
		},
	},
	// Gemini
	{
		matches: func(lm, m string, cfg *config.Config) bool {
			return (strings.Contains(lm, "gemini") || strings.HasPrefix(m, "google/")) && cfg.Providers.Gemini.APIKey != ""
		},
		apply: func(cfg *config.Config, sel *providerSelection) bool {
			return applyStandardProvider(cfg, sel, standardProviderRegistry["gemini"])
		},
	},
	// Zhipu/GLM
	{
		matches: func(lm, _ string, cfg *config.Config) bool {
			return (strings.Contains(lm, "glm") || strings.Contains(lm, "zhipu") || strings.Contains(lm, "zai")) && cfg.Providers.Zhipu.APIKey != ""
		},
		apply: func(cfg *config.Config, sel *providerSelection) bool {
			return applyStandardProvider(cfg, sel, standardProviderRegistry["zhipu"])
		},
	},
	// Groq
	{
		matches: func(lm, m string, cfg *config.Config) bool {
			return (strings.Contains(lm, "groq") || strings.HasPrefix(m, "groq/")) && cfg.Providers.Groq.APIKey != ""
		},
		apply: func(cfg *config.Config, sel *providerSelection) bool {
			return applyStandardProvider(cfg, sel, standardProviderRegistry["groq"])
		},
	},
	// Nvidia
	{
		matches: func(lm, m string, cfg *config.Config) bool {
			return (strings.Contains(lm, "nvidia") || strings.HasPrefix(m, "nvidia/")) && cfg.Providers.Nvidia.APIKey != ""
		},
		apply: func(cfg *config.Config, sel *providerSelection) bool {
			return applyStandardProvider(cfg, sel, standardProviderRegistry["nvidia"])
		},
	},
	// Ollama
	{
		matches: func(lm, m string, cfg *config.Config) bool {
			return (strings.Contains(lm, "ollama") || strings.HasPrefix(m, "ollama/")) && cfg.Providers.Ollama.APIKey != ""
		},
		apply: func(cfg *config.Config, sel *providerSelection) bool {
			return applyStandardProvider(cfg, sel, standardProviderRegistry["ollama"])
		},
	},
	// Mistral
	{
		matches: func(lm, m string, cfg *config.Config) bool {
			return (strings.Contains(lm, "mistral") || strings.HasPrefix(m, "mistral/")) && cfg.Providers.Mistral.APIKey != ""
		},
		apply: func(cfg *config.Config, sel *providerSelection) bool {
			return applyStandardProvider(cfg, sel, standardProviderRegistry["mistral"])
		},
	},
	// VLLM (fallback if API base is configured)
	{
		matches: func(_, _ string, cfg *config.Config) bool {
			return cfg.Providers.VLLM.APIBase != ""
		},
		apply: func(cfg *config.Config, sel *providerSelection) bool {
			return applyStandardProvider(cfg, sel, standardProviderRegistry["vllm"])
		},
	},
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
		// Handle special-case providers that have unique instantiation logic.
		switch providerName {
		case "openai", "gpt":
			if cfg.Providers.OpenAI.APIKey != "" || cfg.Providers.OpenAI.AuthMethod != "" {
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
			}
		case "anthropic", "claude":
			if cfg.Providers.Anthropic.APIKey != "" || cfg.Providers.Anthropic.AuthMethod != "" {
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
			}
		case "claude-cli", "claude-code", "claudecode":
			workspace := cfg.WorkspacePath()
			if workspace == "" {
				workspace = "."
			}
			sel.providerType = providerTypeClaudeCLI
			sel.workspace = workspace
			return sel, nil
		case "codex-cli", "codex-code":
			workspace := cfg.WorkspacePath()
			if workspace == "" {
				workspace = "."
			}
			sel.providerType = providerTypeCodexCLI
			sel.workspace = workspace
			return sel, nil
		case "github_copilot", "copilot":
			sel.providerType = providerTypeGitHubCopilot
			if cfg.Providers.GitHubCopilot.APIBase != "" {
				sel.apiBase = cfg.Providers.GitHubCopilot.APIBase
			} else {
				sel.apiBase = "localhost:4321"
			}
			sel.connectMode = cfg.Providers.GitHubCopilot.ConnectMode
			return sel, nil
		default:
			// Try standard provider registry lookup.
			canonicalName := providerName
			if alias, ok := providerNameAliases[providerName]; ok {
				canonicalName = alias
			}
			if entry, ok := standardProviderRegistry[canonicalName]; ok {
				if applyStandardProvider(cfg, &sel, entry) {
					// Handle deepseek model override.
					if canonicalName == "deepseek" && model != "deepseek-chat" && model != "deepseek-reasoner" {
						sel.model = "deepseek-chat"
					}
				}
			}
		}
	}

	// Fallback: infer provider from model name and configured keys.
	if sel.apiKey == "" && sel.apiBase == "" {
		matched := false
		for _, entry := range modelInferenceRegistry {
			if entry.matches(lowerModel, model, cfg) {
				entry.apply(cfg, &sel)
				matched = true
				break
			}
		}

		// Ultimate fallback: try OpenRouter if configured.
		if !matched {
			if cfg.Providers.OpenRouter.APIKey != "" {
				applyStandardProvider(cfg, &sel, standardProviderRegistry["openrouter"])
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

