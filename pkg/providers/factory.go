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

func createClaudeAuthProvider(apiBase string) (LLMProvider, error) {
	if apiBase == "" {
		apiBase = defaultAnthropicAPIBase
	}
	cred, err := getCredential("anthropic")
	if err != nil {
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("no credentials for anthropic. Run: picoclaw auth login --provider anthropic")
	}
	return NewClaudeProviderWithTokenSourceAndBaseURL(cred.AccessToken, createClaudeTokenSource(), apiBase), nil
}

func createCodexAuthProvider(enableWebSearch bool) (LLMProvider, error) {
	cred, err := getCredential("openai")
	if err != nil {
		return nil, fmt.Errorf("loading auth credentials: %w", err)
	}
	if cred == nil {
		return nil, fmt.Errorf("no credentials for openai. Run: picoclaw auth login --provider openai")
	}
	p := NewCodexProviderWithTokenSource(cred.AccessToken, cred.AccountID, createCodexTokenSource())
	p.enableWebSearch = enableWebSearch
	return p, nil
}

func resolveProviderSelection(cfg *config.Config) (providerSelection, error) {
	model := cfg.Agents.Defaults.Model
	providerName := strings.ToLower(cfg.Agents.Defaults.Provider)
	lowerModel := strings.ToLower(model)

	sel := providerSelection{
		providerType: providerTypeHTTPCompat,
		model:        model,
	}

	// First, prefer explicit provider configuration.
	if providerName != "" {
		switch providerName {
		case "groq":
			if cfg.Providers["groq"].APIKey != "" {
				sel.apiKey = cfg.Providers["groq"].APIKey
				sel.apiBase = cfg.Providers["groq"].APIBase
				sel.proxy = cfg.Providers["groq"].Proxy
				if sel.apiBase == "" {
					sel.apiBase = "https://api.groq.com/openai/v1"
				}
			}
		case "openai", "gpt":
			if cfg.Providers["openai"].APIKey != "" || cfg.Providers["openai"].AuthMethod != "" {
				sel.enableWebSearch = cfg.Providers["openai"].WebSearch
				if cfg.Providers["openai"].AuthMethod == "codex-cli" {
					sel.providerType = providerTypeCodexCLIToken
					return sel, nil
				}
				if cfg.Providers["openai"].AuthMethod == "oauth" || cfg.Providers["openai"].AuthMethod == "token" {
					sel.providerType = providerTypeCodexAuth
					return sel, nil
				}
				sel.apiKey = cfg.Providers["openai"].APIKey
				sel.apiBase = cfg.Providers["openai"].APIBase
				sel.proxy = cfg.Providers["openai"].Proxy
				if sel.apiBase == "" {
					sel.apiBase = "https://api.openai.com/v1"
				}
			}
		case "anthropic", "claude":
			if cfg.Providers["anthropic"].APIKey != "" || cfg.Providers["anthropic"].AuthMethod != "" {
				if cfg.Providers["anthropic"].AuthMethod == "oauth" || cfg.Providers["anthropic"].AuthMethod == "token" {
					sel.apiBase = cfg.Providers["anthropic"].APIBase
					if sel.apiBase == "" {
						sel.apiBase = defaultAnthropicAPIBase
					}
					sel.providerType = providerTypeClaudeAuth
					return sel, nil
				}
				sel.apiKey = cfg.Providers["anthropic"].APIKey
				sel.apiBase = cfg.Providers["anthropic"].APIBase
				sel.proxy = cfg.Providers["anthropic"].Proxy
				if sel.apiBase == "" {
					sel.apiBase = defaultAnthropicAPIBase
				}
			}
		case "openrouter":
			if cfg.Providers["openrouter"].APIKey != "" {
				sel.apiKey = cfg.Providers["openrouter"].APIKey
				sel.proxy = cfg.Providers["openrouter"].Proxy
				if cfg.Providers["openrouter"].APIBase != "" {
					sel.apiBase = cfg.Providers["openrouter"].APIBase
				} else {
					sel.apiBase = "https://openrouter.ai/api/v1"
				}
			}
		case "zhipu", "glm":
			if cfg.Providers["zhipu"].APIKey != "" {
				sel.apiKey = cfg.Providers["zhipu"].APIKey
				sel.apiBase = cfg.Providers["zhipu"].APIBase
				sel.proxy = cfg.Providers["zhipu"].Proxy
				if sel.apiBase == "" {
					sel.apiBase = "https://open.bigmodel.cn/api/paas/v4"
				}
			}
		case "gemini", "google":
			if cfg.Providers["gemini"].APIKey != "" {
				sel.apiKey = cfg.Providers["gemini"].APIKey
				sel.apiBase = cfg.Providers["gemini"].APIBase
				sel.proxy = cfg.Providers["gemini"].Proxy
				if sel.apiBase == "" {
					sel.apiBase = "https://generativelanguage.googleapis.com/v1beta"
				}
			}
		case "vllm":
			if cfg.Providers["vllm"].APIBase != "" {
				sel.apiKey = cfg.Providers["vllm"].APIKey
				sel.apiBase = cfg.Providers["vllm"].APIBase
				sel.proxy = cfg.Providers["vllm"].Proxy
			}
		case "shengsuanyun":
			if cfg.Providers["shengsuanyun"].APIKey != "" {
				sel.apiKey = cfg.Providers["shengsuanyun"].APIKey
				sel.apiBase = cfg.Providers["shengsuanyun"].APIBase
				sel.proxy = cfg.Providers["shengsuanyun"].Proxy
				if sel.apiBase == "" {
					sel.apiBase = "https://router.shengsuanyun.com/api/v1"
				}
			}
		case "nvidia":
			if cfg.Providers["nvidia"].APIKey != "" {
				sel.apiKey = cfg.Providers["nvidia"].APIKey
				sel.apiBase = cfg.Providers["nvidia"].APIBase
				sel.proxy = cfg.Providers["nvidia"].Proxy
				if sel.apiBase == "" {
					sel.apiBase = "https://integrate.api.nvidia.com/v1"
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
		case "deepseek":
			if cfg.Providers["deepseek"].APIKey != "" {
				sel.apiKey = cfg.Providers["deepseek"].APIKey
				sel.apiBase = cfg.Providers["deepseek"].APIBase
				sel.proxy = cfg.Providers["deepseek"].Proxy
				if sel.apiBase == "" {
					sel.apiBase = "https://api.deepseek.com/v1"
				}
				if model != "deepseek-chat" && model != "deepseek-reasoner" {
					sel.model = "deepseek-chat"
				}
			}
		case "github_copilot", "copilot":
			sel.providerType = providerTypeGitHubCopilot
			if cfg.Providers["github_copilot"].APIBase != "" {
				sel.apiBase = cfg.Providers["github_copilot"].APIBase
			} else {
				sel.apiBase = "localhost:4321"
			}
			sel.connectMode = cfg.Providers["github_copilot"].ConnectMode
			return sel, nil
		}
	}

	// Fallback: infer provider from model and configured keys.
	if sel.apiKey == "" && sel.apiBase == "" {
		switch {
		case (strings.Contains(lowerModel, "kimi") || strings.Contains(lowerModel, "moonshot") || strings.HasPrefix(model, "moonshot/")) && cfg.Providers["moonshot"].APIKey != "":
			sel.apiKey = cfg.Providers["moonshot"].APIKey
			sel.apiBase = cfg.Providers["moonshot"].APIBase
			sel.proxy = cfg.Providers["moonshot"].Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://api.moonshot.cn/v1"
			}
		case strings.HasPrefix(model, "openrouter/") ||
			strings.HasPrefix(model, "anthropic/") ||
			strings.HasPrefix(model, "openai/") ||
			strings.HasPrefix(model, "meta-llama/") ||
			strings.HasPrefix(model, "deepseek/") ||
			strings.HasPrefix(model, "google/"):
			sel.apiKey = cfg.Providers["openrouter"].APIKey
			sel.proxy = cfg.Providers["openrouter"].Proxy
			if cfg.Providers["openrouter"].APIBase != "" {
				sel.apiBase = cfg.Providers["openrouter"].APIBase
			} else {
				sel.apiBase = "https://openrouter.ai/api/v1"
			}
		case (strings.Contains(lowerModel, "claude") || strings.HasPrefix(model, "anthropic/")) &&
			(cfg.Providers["anthropic"].APIKey != "" || cfg.Providers["anthropic"].AuthMethod != ""):
			if cfg.Providers["anthropic"].AuthMethod == "oauth" || cfg.Providers["anthropic"].AuthMethod == "token" {
				sel.apiBase = cfg.Providers["anthropic"].APIBase
				if sel.apiBase == "" {
					sel.apiBase = defaultAnthropicAPIBase
				}
				sel.providerType = providerTypeClaudeAuth
				return sel, nil
			}
			sel.apiKey = cfg.Providers["anthropic"].APIKey
			sel.apiBase = cfg.Providers["anthropic"].APIBase
			sel.proxy = cfg.Providers["anthropic"].Proxy
			if sel.apiBase == "" {
				sel.apiBase = defaultAnthropicAPIBase
			}
		case (strings.Contains(lowerModel, "gpt") || strings.HasPrefix(model, "openai/")) &&
			(cfg.Providers["openai"].APIKey != "" || cfg.Providers["openai"].AuthMethod != ""):
			sel.enableWebSearch = cfg.Providers["openai"].WebSearch
			if cfg.Providers["openai"].AuthMethod == "codex-cli" {
				sel.providerType = providerTypeCodexCLIToken
				return sel, nil
			}
			if cfg.Providers["openai"].AuthMethod == "oauth" || cfg.Providers["openai"].AuthMethod == "token" {
				sel.providerType = providerTypeCodexAuth
				return sel, nil
			}
			sel.apiKey = cfg.Providers["openai"].APIKey
			sel.apiBase = cfg.Providers["openai"].APIBase
			sel.proxy = cfg.Providers["openai"].Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://api.openai.com/v1"
			}
		case (strings.Contains(lowerModel, "gemini") || strings.HasPrefix(model, "google/")) && cfg.Providers["gemini"].APIKey != "":
			sel.apiKey = cfg.Providers["gemini"].APIKey
			sel.apiBase = cfg.Providers["gemini"].APIBase
			sel.proxy = cfg.Providers["gemini"].Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://generativelanguage.googleapis.com/v1beta"
			}
		case (strings.Contains(lowerModel, "glm") || strings.Contains(lowerModel, "zhipu") || strings.Contains(lowerModel, "zai")) && cfg.Providers["zhipu"].APIKey != "":
			sel.apiKey = cfg.Providers["zhipu"].APIKey
			sel.apiBase = cfg.Providers["zhipu"].APIBase
			sel.proxy = cfg.Providers["zhipu"].Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://open.bigmodel.cn/api/paas/v4"
			}
		case (strings.Contains(lowerModel, "groq") || strings.HasPrefix(model, "groq/")) && cfg.Providers["groq"].APIKey != "":
			sel.apiKey = cfg.Providers["groq"].APIKey
			sel.apiBase = cfg.Providers["groq"].APIBase
			sel.proxy = cfg.Providers["groq"].Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://api.groq.com/openai/v1"
			}
		case (strings.Contains(lowerModel, "nvidia") || strings.HasPrefix(model, "nvidia/")) && cfg.Providers["nvidia"].APIKey != "":
			sel.apiKey = cfg.Providers["nvidia"].APIKey
			sel.apiBase = cfg.Providers["nvidia"].APIBase
			sel.proxy = cfg.Providers["nvidia"].Proxy
			if sel.apiBase == "" {
				sel.apiBase = "https://integrate.api.nvidia.com/v1"
			}
		case (strings.Contains(lowerModel, "ollama") || strings.HasPrefix(model, "ollama/")) && cfg.Providers["ollama"].APIKey != "":
			sel.apiKey = cfg.Providers["ollama"].APIKey
			sel.apiBase = cfg.Providers["ollama"].APIBase
			sel.proxy = cfg.Providers["ollama"].Proxy
			if sel.apiBase == "" {
				sel.apiBase = "http://localhost:11434/v1"
			}
		case cfg.Providers["vllm"].APIBase != "":
			sel.apiKey = cfg.Providers["vllm"].APIKey
			sel.apiBase = cfg.Providers["vllm"].APIBase
			sel.proxy = cfg.Providers["vllm"].Proxy
		default:
			if cfg.Providers["openrouter"].APIKey != "" {
				sel.apiKey = cfg.Providers["openrouter"].APIKey
				sel.proxy = cfg.Providers["openrouter"].Proxy
				if cfg.Providers["openrouter"].APIBase != "" {
					sel.apiBase = cfg.Providers["openrouter"].APIBase
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

func CreateProvider(cfg *config.Config) (LLMProvider, error) {
	sel, err := resolveProviderSelection(cfg)
	if err != nil {
		return nil, err
	}

	switch sel.providerType {
	case providerTypeClaudeAuth:
		return createClaudeAuthProvider(sel.apiBase)
	case providerTypeCodexAuth:
		return createCodexAuthProvider(sel.enableWebSearch)
	case providerTypeCodexCLIToken:
		c := NewCodexProviderWithTokenSource("", "", CreateCodexCliTokenSource())
		c.enableWebSearch = sel.enableWebSearch
		return c, nil
	case providerTypeClaudeCLI:
		return NewClaudeCliProvider(sel.workspace), nil
	case providerTypeCodexCLI:
		return NewCodexCliProvider(sel.workspace), nil
	case providerTypeGitHubCopilot:
		return NewGitHubCopilotProvider(sel.apiBase, sel.connectMode, sel.model)
	default:
		return NewHTTPProvider(sel.apiKey, sel.apiBase, sel.proxy), nil
	}
}
