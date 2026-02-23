package setup

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// StepDef describes a single prompt step in the setup wizard.
type StepDef struct {
	Kind    string // "text", "select", "yesno"
	Prompt  string
	Info    string
	Options []string // for select/yesno
	ID      string   // logical identifier for mapping answers into config
}

// BuildDefs creates the per-step tui definitions based on the current config.
// It returns a modular list of steps for workspace, provider, model, and channel setup.
func (s *Setup) BuildDefs() []StepDef {
	defs := []StepDef{}

	// Step 1: Workspace setup
	defs = append(defs, StepDef{ID: "workspace", Kind: "text", Prompt: "1. Workspace path", Info: "Directory for agent workspace files"})
	defs = append(defs, StepDef{ID: "restrict_workspace", Kind: "select", Prompt: "1b. Restrict to workspace?", Options: []string{"yes", "no"}})

	// Step 2: Provider setup (ordered by popularity)
	defs = append(defs, buildProviderSelectStep(s.Cfg)...)

	// Step 3: Model selection (popular suggestions + custom option)
	defs = append(defs, buildModelSelectStep(s.Cfg)...)

	// Step 4: Channel setup
	defs = append(defs, buildChannelSelectStep(s.Cfg)...)

	// Final confirmation step
	defs = append(defs, StepDef{ID: "confirm", Kind: "yesno", Prompt: "6. Confirm and save configuration?", Options: []string{"yes", "no"}})

	return defs
}

// buildProviderSelectStep returns provider selection step and credential prompts for missing required fields.
func buildProviderSelectStep(cfg *config.Config) []StepDef {
	var steps []StepDef

	provInfo := config.GetProvidersInfo(cfg)
	ordered := config.GetOrderedProviderNames()

	provOptions := []string{}
	added := map[string]struct{}{}

	// Merge ordered list with available provider info
	for _, name := range ordered {
		if _, ok := provInfoLookup(provInfo, name); ok {
			provOptions = append(provOptions, name)
			added[name] = struct{}{}
		}
	}
	// Add any remaining providers not in priority list
	for _, p := range provInfo {
		if _, ok := added[p.Name]; !ok {
			provOptions = append(provOptions, p.Name)
		}
	}

	// Provider selection
	steps = append(steps, StepDef{ID: "provider", Kind: "select", Prompt: "2. Choose provider", Options: provOptions})

	// Determine selected provider (from config or first in list)
	selProvider := cfg.Agents.Defaults.Provider
	if selProvider == "" && len(provOptions) > 0 {
		selProvider = provOptions[0]
	}

	// Add credential prompts for missing required fields
	if selProvider != "" {
		provInfoItem, ok := provInfoLookup(provInfo, selProvider)
		if ok {
			for _, cred := range provInfoItem.RequiredCredentials {
				if !ProviderCredentialPresent(cfg, selProvider, cred) {
					steps = append(steps, StepDef{
						ID:     "provider_" + cred,
						Kind:   "text",
						Prompt: fmt.Sprintf("2b. %s for %s", cred, selProvider),
					})
				}
			}
		}
	}

	return steps
}

// buildModelSelectStep returns model selection step and custom model input.
func buildModelSelectStep(cfg *config.Config) []StepDef {
	var steps []StepDef

	selProvider := cfg.Agents.Defaults.Provider
	modelSuggestions := config.GetPopularModels(selProvider)

	modelOptions := make([]string, len(modelSuggestions)+1)
	copy(modelOptions, modelSuggestions)
	modelOptions[len(modelSuggestions)] = "custom"

	steps = append(steps, StepDef{
		ID:      "model_select",
		Kind:    "select",
		Prompt:  "3. Choose model",
		Options: modelOptions,
	})

	steps = append(steps, StepDef{
		ID:     "default_model",
		Kind:   "text",
		Prompt: "3b. Enter custom model name",
	})

	return steps
}

// buildChannelSelectStep returns channel selection step and credential prompts.
func buildChannelSelectStep(cfg *config.Config) []StepDef {
	var steps []StepDef

	channels := config.GetAllChannelNames()

	steps = append(steps, StepDef{
		ID:      "channel_select",
		Kind:    "select",
		Prompt:  "4. Choose channel",
		Options: channels,
	})

	// Add placeholder for channel credentials (filled dynamically based on selection)
	steps = append(steps, StepDef{
		ID:     "channel_token",
		Kind:   "text",
		Prompt: "4b. Channel credential/token",
	})

	return steps
}

// provInfoLookup returns (ProviderInfo, true) if provider name exists in list.
func provInfoLookup(list []config.ProviderInfo, name string) (config.ProviderInfo, bool) {
	for _, p := range list {
		if p.Name == name {
			return p, true
		}
	}
	return config.ProviderInfo{}, false
}

// ProviderCredentialPresent checks whether a given provider credential is present
// in the legacy ProvidersConfig or within any ModelConfig for that provider.
func ProviderCredentialPresent(c *config.Config, provider, cred string) bool {
	if c == nil {
		return false
	}
	p := strings.ToLower(provider)
	switch p {
	case "anthropic":
		if cred == "api_key" {
			return c.Providers.Anthropic.APIKey != ""
		}
		if cred == "api_base" {
			return c.Providers.Anthropic.APIBase != ""
		}
	case "openai":
		if cred == "api_key" {
			return c.Providers.OpenAI.APIKey != ""
		}
		if cred == "api_base" {
			return c.Providers.OpenAI.APIBase != ""
		}
	case "openrouter":
		if cred == "api_key" {
			return c.Providers.OpenRouter.APIKey != ""
		}
		if cred == "api_base" {
			return c.Providers.OpenRouter.APIBase != ""
		}
	case "groq":
		if cred == "api_key" {
			return c.Providers.Groq.APIKey != ""
		}
	case "zhipu":
		if cred == "api_key" {
			return c.Providers.Zhipu.APIKey != ""
		}
	case "vllm":
		if cred == "api_key" {
			return c.Providers.VLLM.APIKey != ""
		}
	case "gemini":
		if cred == "api_key" {
			return c.Providers.Gemini.APIKey != ""
		}
	case "nvidia":
		if cred == "api_key" {
			return c.Providers.Nvidia.APIKey != ""
		}
	case "ollama":
		if cred == "api_key" {
			return c.Providers.Ollama.APIKey != ""
		}
	case "moonshot":
		if cred == "api_key" {
			return c.Providers.Moonshot.APIKey != ""
		}
	case "shengsuanyun":
		if cred == "api_key" {
			return c.Providers.ShengSuanYun.APIKey != ""
		}
	case "deepseek":
		if cred == "api_key" {
			return c.Providers.DeepSeek.APIKey != ""
		}
	case "cerebras":
		if cred == "api_key" {
			return c.Providers.Cerebras.APIKey != ""
		}
	case "volcengine":
		if cred == "api_key" {
			return c.Providers.VolcEngine.APIKey != ""
		}
	case "github_copilot":
		if cred == "api_key" {
			return c.Providers.GitHubCopilot.APIKey != ""
		}
	case "antigravity":
		if cred == "api_key" {
			return c.Providers.Antigravity.APIKey != ""
		}
	case "qwen":
		if cred == "api_key" {
			return c.Providers.Qwen.APIKey != ""
		}
	}

	// Check model_list entries for provider credentials
	for _, m := range c.ModelList {
		pfx := config.ParseProtocol(m.Model)
		if pfx == "" {
			pfx = "openai"
		}
		if pfx == p {
			if cred == "api_key" && m.APIKey != "" {
				return true
			}
			if cred == "api_base" && m.APIBase != "" {
				return true
			}
		}
	}

	return false
}

// SetProviderCredential sets a credential value for the named provider into the
// appropriate place in the config (legacy ProvidersConfig or model_list fallback).
func SetProviderCredential(c *config.Config, provider, cred, value string) {
	if c == nil || provider == "" || cred == "" {
		return
	}
	p := strings.ToLower(provider)
	switch p {
	case "anthropic":
		if cred == "api_key" {
			c.Providers.Anthropic.APIKey = value
		} else if cred == "api_base" {
			c.Providers.Anthropic.APIBase = value
		}
	case "openai":
		if cred == "api_key" {
			c.Providers.OpenAI.APIKey = value
		} else if cred == "api_base" {
			c.Providers.OpenAI.APIBase = value
		}
	case "openrouter":
		if cred == "api_key" {
			c.Providers.OpenRouter.APIKey = value
		} else if cred == "api_base" {
			c.Providers.OpenRouter.APIBase = value
		}
	case "groq":
		if cred == "api_key" {
			c.Providers.Groq.APIKey = value
		}
	case "zhipu":
		if cred == "api_key" {
			c.Providers.Zhipu.APIKey = value
		}
	case "vllm":
		if cred == "api_key" {
			c.Providers.VLLM.APIKey = value
		}
	case "gemini":
		if cred == "api_key" {
			c.Providers.Gemini.APIKey = value
		}
	case "nvidia":
		if cred == "api_key" {
			c.Providers.Nvidia.APIKey = value
		}
	case "ollama":
		if cred == "api_key" {
			c.Providers.Ollama.APIKey = value
		}
	case "moonshot":
		if cred == "api_key" {
			c.Providers.Moonshot.APIKey = value
		}
	case "shengsuanyun":
		if cred == "api_key" {
			c.Providers.ShengSuanYun.APIKey = value
		}
	case "deepseek":
		if cred == "api_key" {
			c.Providers.DeepSeek.APIKey = value
		}
	case "cerebras":
		if cred == "api_key" {
			c.Providers.Cerebras.APIKey = value
		}
	case "volcengine":
		if cred == "api_key" {
			c.Providers.VolcEngine.APIKey = value
		}
	case "github_copilot":
		if cred == "api_key" {
			c.Providers.GitHubCopilot.APIKey = value
		}
	case "antigravity":
		if cred == "api_key" {
			c.Providers.Antigravity.APIKey = value
		}
	case "qwen":
		if cred == "api_key" {
			c.Providers.Qwen.APIKey = value
		}
	default:
		// If provider not in legacy ProvidersConfig, try to set on a matching ModelConfig
		for i := range c.ModelList {
			pfx := config.ParseProtocol(c.ModelList[i].Model)
			if pfx == "" {
				pfx = "openai"
			}
			if pfx == p {
				if cred == "api_key" {
					c.ModelList[i].APIKey = value
				} else if cred == "api_base" {
					c.ModelList[i].APIBase = value
				}
				return
			}
		}
	}
}

// BuildSummary returns a human-readable summary of the current configuration.
func BuildSummary(cfg *config.Config) []string {
	summary := []string{}
	summary = append(summary, "Configuration summary:")
	summary = append(summary, fmt.Sprintf("Workspace: %s", cfg.Agents.Defaults.Workspace))
	summary = append(summary, fmt.Sprintf("Restrict to workspace: %v", cfg.Agents.Defaults.RestrictToWorkspace))
	summary = append(summary, fmt.Sprintf("Provider: %s", cfg.Agents.Defaults.Provider))
	if cfg.Agents.Defaults.Model != "" {
		summary = append(summary, fmt.Sprintf("Model: %s", cfg.Agents.Defaults.Model))
	}
	return summary
}
