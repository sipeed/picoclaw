package status

import (
	"fmt"
	"os"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/auth"
)

func statusCmd() {
	cfg, err := internal.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	configPath := internal.GetConfigPath()

	fmt.Printf("%s picoclaw Status\n", internal.Logo)
	fmt.Printf("Version: %s\n", internal.FormatVersion())
	build, _ := internal.FormatBuildInfo()
	if build != "" {
		fmt.Printf("Build: %s\n", build)
	}
	fmt.Println()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("Config:", configPath, "✓")
	} else {
		fmt.Println("Config:", configPath, "✗")
	}

	workspace := cfg.WorkspacePath()
	if _, err := os.Stat(workspace); err == nil {
		fmt.Println("Workspace:", workspace, "✓")
	} else {
		fmt.Println("Workspace:", workspace, "✗")
	}

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Model: %s\n", cfg.Agents.Defaults.GetModelName())

		apiKeyProviders := []struct {
			name   string
			hasKey bool
		}{
			{"OpenRouter API", cfg.Providers.OpenRouter.APIKey != ""},
			{"Anthropic API", cfg.Providers.Anthropic.APIKey != ""},
			{"OpenAI API", cfg.Providers.OpenAI.APIKey != ""},
			{"Gemini API", cfg.Providers.Gemini.APIKey != ""},
			{"Zhipu API", cfg.Providers.Zhipu.APIKey != ""},
			{"Qwen API", cfg.Providers.Qwen.APIKey != ""},
			{"Groq API", cfg.Providers.Groq.APIKey != ""},
			{"Moonshot API", cfg.Providers.Moonshot.APIKey != ""},
			{"DeepSeek API", cfg.Providers.DeepSeek.APIKey != ""},
			{"VolcEngine API", cfg.Providers.VolcEngine.APIKey != ""},
			{"Nvidia API", cfg.Providers.Nvidia.APIKey != ""},
		}
		for _, p := range apiKeyProviders {
			if p.hasKey {
				fmt.Printf("%s: ✓\n", p.name)
			} else {
				fmt.Printf("%s: not set\n", p.name)
			}
		}

		urlProviders := []struct {
			name    string
			apiBase string
		}{
			{"vLLM/Local", cfg.Providers.VLLM.APIBase},
			{"Ollama", cfg.Providers.Ollama.APIBase},
		}
		for _, p := range urlProviders {
			if p.apiBase != "" {
				fmt.Printf("%s: ✓ %s\n", p.name, p.apiBase)
			} else {
				fmt.Printf("%s: not set\n", p.name)
			}
		}

		store, _ := auth.LoadStore()
		if store != nil && len(store.Credentials) > 0 {
			fmt.Println("\nOAuth/Token Auth:")
			for provider, cred := range store.Credentials {
				status := "authenticated"
				if cred.IsExpired() {
					status = "expired"
				} else if cred.NeedsRefresh() {
					status = "needs refresh"
				}
				fmt.Printf("  %s (%s): %s\n", provider, cred.AuthMethod, status)
			}
		}
	}
}
