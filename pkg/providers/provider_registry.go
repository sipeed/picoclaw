package providers

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

type providerCreator func(cfg *config.Config, model string) (LLMProvider, bool, error)

type providerRegistration struct {
	Name          string
	Aliases       []string
	ModelPrefixes []string
	Creator       providerCreator
}

type providerRegistry struct {
	byName  map[string]providerRegistration
	ordered []*providerRegistration
}

func newProviderRegistry() *providerRegistry {
	return &providerRegistry{
		byName:  make(map[string]providerRegistration),
		ordered: make([]*providerRegistration, 0, 16),
	}
}

func (r *providerRegistry) Register(reg providerRegistration) {
	normalized := reg
	normalized.Name = strings.ToLower(strings.TrimSpace(normalized.Name))
	if normalized.Name == "" {
		return
	}

	r.byName[normalized.Name] = normalized
	for _, alias := range normalized.Aliases {
		a := strings.ToLower(strings.TrimSpace(alias))
		if a != "" {
			r.byName[a] = normalized
		}
	}

	normalizedCopy := normalized
	r.ordered = append(r.ordered, &normalizedCopy)
}

func (r *providerRegistry) Create(cfg *config.Config) (LLMProvider, error) {
	model := strings.TrimSpace(cfg.Agents.Defaults.Model)
	providerName := strings.ToLower(strings.TrimSpace(cfg.Agents.Defaults.Provider))

	if providerName != "" {
		reg, ok := r.byName[providerName]
		if !ok {
			return nil, fmt.Errorf("unknown provider: %s", providerName)
		}
		provider, configured, err := reg.Creator(cfg, model)
		if err != nil {
			return nil, err
		}
		if !configured {
			return nil, fmt.Errorf("provider '%s' is not configured", reg.Name)
		}
		return provider, nil
	}

	modelLower := strings.ToLower(model)
	for _, reg := range r.ordered {
		if !matchesModelPrefix(modelLower, reg.ModelPrefixes) {
			continue
		}
		provider, configured, err := reg.Creator(cfg, model)
		if err != nil {
			return nil, err
		}
		if configured {
			return provider, nil
		}
	}

	if provider, configured, err := openRouterCreator(cfg, model); err != nil {
		return nil, err
	} else if configured {
		return provider, nil
	}

	return nil, fmt.Errorf("no API key configured for model: %s", model)
}

func matchesModelPrefix(model string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(model, strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}

func createHTTPProviderFromConfig(pc config.ProviderConfig, defaultBase string, requireAPIKey bool, requireAPIBase bool) (LLMProvider, bool, error) {
	if requireAPIKey && pc.APIKey == "" {
		return nil, false, nil
	}
	if requireAPIBase && pc.APIBase == "" {
		return nil, false, nil
	}

	apiBase := pc.APIBase
	if apiBase == "" {
		apiBase = defaultBase
	}
	if apiBase == "" {
		return nil, false, fmt.Errorf("no API base configured for provider")
	}

	return NewHTTPProvider(pc.APIKey, apiBase, pc.Proxy), true, nil
}

func claudeCreator(cfg *config.Config, _ string) (LLMProvider, bool, error) {
	pc := cfg.Providers.Anthropic
	if pc.AuthMethod == "oauth" || pc.AuthMethod == "token" {
		p, err := createClaudeAuthProvider()
		if err != nil {
			return nil, false, err
		}
		return p, true, nil
	}
	return createHTTPProviderFromConfig(pc, "https://api.anthropic.com/v1", true, false)
}

func openAICreator(cfg *config.Config, _ string) (LLMProvider, bool, error) {
	pc := cfg.Providers.OpenAI
	if pc.AuthMethod == "oauth" || pc.AuthMethod == "token" {
		p, err := createCodexAuthProvider()
		if err != nil {
			return nil, false, err
		}
		return p, true, nil
	}
	return createHTTPProviderFromConfig(pc, "https://api.openai.com/v1", true, false)
}

func openRouterCreator(cfg *config.Config, _ string) (LLMProvider, bool, error) {
	return createHTTPProviderFromConfig(cfg.Providers.OpenRouter, "https://openrouter.ai/api/v1", true, false)
}

func groqCreator(cfg *config.Config, _ string) (LLMProvider, bool, error) {
	return createHTTPProviderFromConfig(cfg.Providers.Groq, "https://api.groq.com/openai/v1", true, false)
}

func zhipuCreator(cfg *config.Config, _ string) (LLMProvider, bool, error) {
	return createHTTPProviderFromConfig(cfg.Providers.Zhipu, "https://open.bigmodel.cn/api/paas/v4", true, false)
}

func zaiCreator(cfg *config.Config, _ string) (LLMProvider, bool, error) {
	return createHTTPProviderFromConfig(cfg.Providers.ZAI, "https://api.z.ai/api/paas/v4", true, false)
}

func geminiCreator(cfg *config.Config, _ string) (LLMProvider, bool, error) {
	return createHTTPProviderFromConfig(cfg.Providers.Gemini, "https://generativelanguage.googleapis.com/v1beta", true, false)
}

func nvidiaCreator(cfg *config.Config, _ string) (LLMProvider, bool, error) {
	return createHTTPProviderFromConfig(cfg.Providers.Nvidia, "https://integrate.api.nvidia.com/v1", true, false)
}

func moonshotCreator(cfg *config.Config, _ string) (LLMProvider, bool, error) {
	return createHTTPProviderFromConfig(cfg.Providers.Moonshot, "https://api.moonshot.cn/v1", true, false)
}

func vllmCreator(cfg *config.Config, _ string) (LLMProvider, bool, error) {
	return createHTTPProviderFromConfig(cfg.Providers.VLLM, "", false, true)
}

func zenCreator(cfg *config.Config, _ string) (LLMProvider, bool, error) {
	return createHTTPProviderFromConfig(cfg.Providers.Zen, "https://opencode.ai/zen/v1", true, false)
}

func claudeCLICreator(cfg *config.Config, _ string) (LLMProvider, bool, error) {
	workspace := cfg.Agents.Defaults.Workspace
	if workspace == "" {
		workspace = "."
	}
	return NewClaudeCliProvider(workspace), true, nil
}

var defaultProviderRegistry = newProviderRegistry()

func RegisterProvider(reg providerRegistration) {
	defaultProviderRegistry.Register(reg)
}
