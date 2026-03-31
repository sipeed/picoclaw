package status

import (
	"fmt"
	"os"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/cliui"
	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

func statusCmd() {
	cfg, err := internal.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	configPath := internal.GetConfigPath()
	build, _ := config.FormatBuildInfo()

	_, configStatErr := os.Stat(configPath)
	configOK := configStatErr == nil

	workspace := cfg.WorkspacePath()
	_, wsErr := os.Stat(workspace)
	wsOK := wsErr == nil

	report := cliui.StatusReport{
		Logo:          internal.Logo,
		Version:       config.FormatVersion(),
		Build:         build,
		ConfigPath:    configPath,
		ConfigOK:      configOK,
		WorkspacePath: workspace,
		WorkspaceOK:   wsOK,
		Model:         cfg.Agents.Defaults.GetModelName(),
	}

	if configOK {
		hasOpenRouter := cfg.Providers.OpenRouter.APIKey != ""
		hasAnthropic := cfg.Providers.Anthropic.APIKey != ""
		hasOpenAI := cfg.Providers.OpenAI.APIKey != ""
		hasGemini := cfg.Providers.Gemini.APIKey != ""
		hasZhipu := cfg.Providers.Zhipu.APIKey != ""
		hasQwen := cfg.Providers.Qwen.APIKey != ""
		hasGroq := cfg.Providers.Groq.APIKey != ""
		hasVLLM := cfg.Providers.VLLM.APIBase != ""
		hasMoonshot := cfg.Providers.Moonshot.APIKey != ""
		hasDeepSeek := cfg.Providers.DeepSeek.APIKey != ""
		hasVolcEngine := cfg.Providers.VolcEngine.APIKey != ""
		hasNvidia := cfg.Providers.Nvidia.APIKey != ""
		hasOllama := cfg.Providers.Ollama.APIBase != ""

		status := func(enabled bool) string {
			if enabled {
				return "✓"
			}
			return "not set"
		}

		report.ProviderNames = []string{
			"OpenRouter API",
			"Anthropic API",
			"OpenAI API",
			"Gemini API",
			"Zhipu API",
			"Qwen API",
			"Groq API",
			"Moonshot API",
			"DeepSeek API",
			"VolcEngine API",
			"Nvidia API",
		}
		report.ProviderVals = []string{
			status(hasOpenRouter),
			status(hasAnthropic),
			status(hasOpenAI),
			status(hasGemini),
			status(hasZhipu),
			status(hasQwen),
			status(hasGroq),
			status(hasMoonshot),
			status(hasDeepSeek),
			status(hasVolcEngine),
			status(hasNvidia),
		}

		if hasVLLM {
			report.ProviderNames = append(report.ProviderNames, "vLLM / local")
			report.ProviderVals = append(report.ProviderVals, "✓ "+cfg.Providers.VLLM.APIBase)
		} else {
			report.ProviderNames = append(report.ProviderNames, "vLLM / local")
			report.ProviderVals = append(report.ProviderVals, "not set")
		}
		if hasOllama {
			report.ProviderNames = append(report.ProviderNames, "Ollama")
			report.ProviderVals = append(report.ProviderVals, "✓ "+cfg.Providers.Ollama.APIBase)
		} else {
			report.ProviderNames = append(report.ProviderNames, "Ollama")
			report.ProviderVals = append(report.ProviderVals, "not set")
		}

		store, _ := auth.LoadStore()
		if store != nil && len(store.Credentials) > 0 {
			for provider, cred := range store.Credentials {
				st := "authenticated"
				if cred.IsExpired() {
					st = "expired"
				} else if cred.NeedsRefresh() {
					st = "needs refresh"
				}
				report.OAuthLines = append(report.OAuthLines,
					fmt.Sprintf("%s (%s): %s", provider, cred.AuthMethod, st))
			}
		}
	}

	cliui.PrintStatus(report)
}
