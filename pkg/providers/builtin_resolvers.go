package providers

import (
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	RegisterProvider("groq", groqResolver)
	RegisterProvider("openai", openaiResolver)
	RegisterProvider("gpt", openaiResolver)
	RegisterProvider("anthropic", anthropicResolver)
	RegisterProvider("claude", anthropicResolver)
	RegisterProvider("openrouter", openrouterResolver)
	RegisterProvider("litellm", litellmResolver)
	RegisterProvider("zhipu", zhipuResolver)
	RegisterProvider("glm", zhipuResolver)
	RegisterProvider("gemini", geminiResolver)
	RegisterProvider("google", geminiResolver)
	RegisterProvider("vllm", vllmResolver)
	RegisterProvider("shengsuanyun", shengsuanyunResolver)
	RegisterProvider("nvidia", nvidiaResolver)
	RegisterProvider("vivgrid", vivgridResolver)
	RegisterProvider("deepseek", deepseekResolver)
	RegisterProvider("avian", avianResolver)
	RegisterProvider("mistral", mistralResolver)
	RegisterProvider("minimax", minimaxResolver)
	RegisterProvider("claude-cli", claudeCLIResolver)
	RegisterProvider("claude-code", claudeCLIResolver)
	RegisterProvider("claudecode", claudeCLIResolver)
	RegisterProvider("codex-cli", codexCLIResolver)
	RegisterProvider("codex-code", codexCLIResolver)
	RegisterProvider("github_copilot", copilotResolver)
	RegisterProvider("copilot", copilotResolver)
}

func groqResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.Groq.APIKey != "" {
		sel := providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
			apiKey:       cfg.Providers.Groq.APIKey,
			apiBase:      cfg.Providers.Groq.APIBase,
			proxy:        cfg.Providers.Groq.Proxy,
		}
		if sel.apiBase == "" {
			sel.apiBase = "https://api.groq.com/openai/v1"
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func openaiResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.OpenAI.APIKey != "" || cfg.Providers.OpenAI.AuthMethod != "" {
		sel := providerSelection{
			providerType:    providerTypeHTTPCompat,
			model:           model,
			enableWebSearch: cfg.Providers.OpenAI.WebSearch,
		}
		if cfg.Providers.OpenAI.AuthMethod == "codex-cli" {
			sel.providerType = providerTypeCodexCLIToken
			return sel, true, nil
		}
		if cfg.Providers.OpenAI.AuthMethod == "oauth" || cfg.Providers.OpenAI.AuthMethod == "token" {
			sel.providerType = providerTypeCodexAuth
			return sel, true, nil
		}
		sel.apiKey = cfg.Providers.OpenAI.APIKey
		sel.apiBase = cfg.Providers.OpenAI.APIBase
		sel.proxy = cfg.Providers.OpenAI.Proxy
		if sel.apiBase == "" {
			sel.apiBase = "https://api.openai.com/v1"
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func anthropicResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.Anthropic.APIKey != "" || cfg.Providers.Anthropic.AuthMethod != "" {
		sel := providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
		}
		if cfg.Providers.Anthropic.AuthMethod == "oauth" || cfg.Providers.Anthropic.AuthMethod == "token" {
			sel.apiBase = cfg.Providers.Anthropic.APIBase
			if sel.apiBase == "" {
				sel.apiBase = defaultAnthropicAPIBase
			}
			sel.providerType = providerTypeClaudeAuth
			return sel, true, nil
		}
		sel.apiKey = cfg.Providers.Anthropic.APIKey
		sel.apiBase = cfg.Providers.Anthropic.APIBase
		sel.proxy = cfg.Providers.Anthropic.Proxy
		if sel.apiBase == "" {
			sel.apiBase = defaultAnthropicAPIBase
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func openrouterResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.OpenRouter.APIKey != "" {
		sel := providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
			apiKey:       cfg.Providers.OpenRouter.APIKey,
			proxy:        cfg.Providers.OpenRouter.Proxy,
			apiBase:      cfg.Providers.OpenRouter.APIBase,
		}
		if sel.apiBase == "" {
			sel.apiBase = "https://openrouter.ai/api/v1"
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func litellmResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.LiteLLM.APIKey != "" || cfg.Providers.LiteLLM.APIBase != "" {
		sel := providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
			apiKey:       cfg.Providers.LiteLLM.APIKey,
			apiBase:      cfg.Providers.LiteLLM.APIBase,
			proxy:        cfg.Providers.LiteLLM.Proxy,
		}
		if sel.apiBase == "" {
			sel.apiBase = "http://localhost:4000/v1"
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func zhipuResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.Zhipu.APIKey != "" {
		sel := providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
			apiKey:       cfg.Providers.Zhipu.APIKey,
			apiBase:      cfg.Providers.Zhipu.APIBase,
			proxy:        cfg.Providers.Zhipu.Proxy,
		}
		if sel.apiBase == "" {
			sel.apiBase = "https://open.bigmodel.cn/api/paas/v4"
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func geminiResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.Gemini.APIKey != "" {
		sel := providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
			apiKey:       cfg.Providers.Gemini.APIKey,
			apiBase:      cfg.Providers.Gemini.APIBase,
			proxy:        cfg.Providers.Gemini.Proxy,
		}
		if sel.apiBase == "" {
			sel.apiBase = "https://generativelanguage.googleapis.com/v1beta"
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func vllmResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.VLLM.APIBase != "" {
		return providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
			apiKey:       cfg.Providers.VLLM.APIKey,
			apiBase:      cfg.Providers.VLLM.APIBase,
			proxy:        cfg.Providers.VLLM.Proxy,
		}, true, nil
	}
	return providerSelection{}, false, nil
}

func shengsuanyunResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.ShengSuanYun.APIKey != "" {
		sel := providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
			apiKey:       cfg.Providers.ShengSuanYun.APIKey,
			apiBase:      cfg.Providers.ShengSuanYun.APIBase,
			proxy:        cfg.Providers.ShengSuanYun.Proxy,
		}
		if sel.apiBase == "" {
			sel.apiBase = "https://router.shengsuanyun.com/api/v1"
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func nvidiaResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.Nvidia.APIKey != "" {
		sel := providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
			apiKey:       cfg.Providers.Nvidia.APIKey,
			apiBase:      cfg.Providers.Nvidia.APIBase,
			proxy:        cfg.Providers.Nvidia.Proxy,
		}
		if sel.apiBase == "" {
			sel.apiBase = "https://integrate.api.nvidia.com/v1"
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func vivgridResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.Vivgrid.APIKey != "" {
		sel := providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
			apiKey:       cfg.Providers.Vivgrid.APIKey,
			apiBase:      cfg.Providers.Vivgrid.APIBase,
			proxy:        cfg.Providers.Vivgrid.Proxy,
		}
		if sel.apiBase == "" {
			sel.apiBase = "https://api.vivgrid.com/v1"
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func claudeCLIResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	workspace := cfg.WorkspacePath()
	if workspace == "" {
		workspace = "."
	}
	return providerSelection{
		providerType: providerTypeClaudeCLI,
		model:        model,
		workspace:    workspace,
	}, true, nil
}

func codexCLIResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	workspace := cfg.WorkspacePath()
	if workspace == "" {
		workspace = "."
	}
	return providerSelection{
		providerType: providerTypeCodexCLI,
		model:        model,
		workspace:    workspace,
	}, true, nil
}

func deepseekResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.DeepSeek.APIKey != "" {
		sel := providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
			apiKey:       cfg.Providers.DeepSeek.APIKey,
			apiBase:      cfg.Providers.DeepSeek.APIBase,
			proxy:        cfg.Providers.DeepSeek.Proxy,
		}
		if sel.apiBase == "" {
			sel.apiBase = "https://api.deepseek.com/v1"
		}
		if model != "deepseek-chat" && model != "deepseek-reasoner" {
			sel.model = "deepseek-chat"
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func avianResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.Avian.APIKey != "" {
		sel := providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
			apiKey:       cfg.Providers.Avian.APIKey,
			apiBase:      cfg.Providers.Avian.APIBase,
			proxy:        cfg.Providers.Avian.Proxy,
		}
		if sel.apiBase == "" {
			sel.apiBase = "https://api.avian.io/v1"
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func mistralResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.Mistral.APIKey != "" {
		sel := providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
			apiKey:       cfg.Providers.Mistral.APIKey,
			apiBase:      cfg.Providers.Mistral.APIBase,
			proxy:        cfg.Providers.Mistral.Proxy,
		}
		if sel.apiBase == "" {
			sel.apiBase = "https://api.mistral.ai/v1"
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func minimaxResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	if cfg.Providers.Minimax.APIKey != "" {
		sel := providerSelection{
			providerType: providerTypeHTTPCompat,
			model:        model,
			apiKey:       cfg.Providers.Minimax.APIKey,
			apiBase:      cfg.Providers.Minimax.APIBase,
			proxy:        cfg.Providers.Minimax.Proxy,
		}
		if sel.apiBase == "" {
			sel.apiBase = "https://api.minimaxi.com/v1"
		}
		return sel, true, nil
	}
	return providerSelection{}, false, nil
}

func copilotResolver(cfg *config.Config, name, model string) (providerSelection, bool, error) {
	sel := providerSelection{
		providerType: providerTypeGitHubCopilot,
		model:        model,
		connectMode:  cfg.Providers.GitHubCopilot.ConnectMode,
	}
	if cfg.Providers.GitHubCopilot.APIBase != "" {
		sel.apiBase = cfg.Providers.GitHubCopilot.APIBase
	} else {
		sel.apiBase = "localhost:4321"
	}
	return sel, true, nil
}
