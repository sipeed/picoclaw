package status

import (
	"fmt"
	"os"
	"strings"

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

		// Show model_list entries
		if len(cfg.ModelList) > 0 {
			fmt.Println()
			fmt.Println("--- Model List ---")
			for i, mc := range cfg.ModelList {
				apiStatus := ""
				if mc.APIKey != "" {
					apiStatus = " [API key set]"
				} else if mc.APIBase != "" {
					if strings.Contains(mc.APIBase, "localhost") || strings.Contains(mc.APIBase, "127.0.0.1") {
						apiStatus = " [local: " + mc.APIBase + "]"
					} else {
						apiStatus = " [" + mc.APIBase + "]"
					}
				} else if mc.AuthMethod == "oauth" {
					apiStatus = " [OAuth]"
				}
				fmt.Printf("%d. %s -> %s%s\n", i+1, mc.ModelName, mc.Model, apiStatus)
			}
		}

		// Show fallback chain
		if len(cfg.Agents.Defaults.ModelFallbacks) > 0 {
			fmt.Println()
			fmt.Println("--- Fallback Chain ---")
			fmt.Printf("Primary: %s\n", cfg.Agents.Defaults.GetModelName())
			fmt.Print("Fallbacks: ")
			for i, fb := range cfg.Agents.Defaults.ModelFallbacks {
				if i > 0 {
					fmt.Print(" -> ")
				}
				fmt.Print(fb)
			}
			fmt.Println()
		}

		// Check model_list for API keys/bases
		modelListHasOpenRouter := false
		modelListHasOpenAI := false
		modelListHasOllama := false
		modelListHasLocal := false
		modelListHasAntigravity := false

		for _, mc := range cfg.ModelList {
			// Check provider from model string
			if strings.HasPrefix(mc.Model, "openrouter/") {
				modelListHasOpenRouter = true
			}
			if strings.HasPrefix(mc.Model, "openai/") || strings.HasPrefix(mc.Model, "gpt") {
				modelListHasOpenAI = true
			}
			if strings.HasPrefix(mc.Model, "antigravity/") {
				modelListHasAntigravity = true
			}
			if strings.HasPrefix(mc.Model, "ollama/") {
				modelListHasOllama = true
			}
			if mc.APIBase != "" {
				if strings.Contains(mc.APIBase, "ollama") {
					modelListHasOllama = true
				}
				// Local endpoints (not remote APIs)
				if strings.Contains(mc.APIBase, "localhost") || strings.Contains(mc.APIBase, "127.0.0.1") {
					modelListHasLocal = true
				}
			}
		}

		// Check old providers config (legacy)
		hasOpenRouter := cfg.Providers.OpenRouter.APIKey != "" || modelListHasOpenRouter
		hasAnthropic := cfg.Providers.Anthropic.APIKey != ""
		hasOpenAI := cfg.Providers.OpenAI.APIKey != "" || modelListHasOpenAI
		hasGemini := cfg.Providers.Gemini.APIKey != ""
		hasZhipu := cfg.Providers.Zhipu.APIKey != ""
		hasQwen := cfg.Providers.Qwen.APIKey != ""
		hasGroq := cfg.Providers.Groq.APIKey != ""
		hasVLLM := cfg.Providers.VLLM.APIBase != "" || modelListHasLocal
		hasMoonshot := cfg.Providers.Moonshot.APIKey != ""
		hasDeepSeek := cfg.Providers.DeepSeek.APIKey != ""
		hasVolcEngine := cfg.Providers.VolcEngine.APIKey != ""
		hasNvidia := cfg.Providers.Nvidia.APIKey != ""
		hasOllama := cfg.Providers.Ollama.APIBase != "" || modelListHasOllama
		hasAntigravityOAuth := cfg.Providers.Antigravity.APIKey != "" || modelListHasAntigravity

		status := func(enabled bool) string {
			if enabled {
				return "✓"
			}
			return "not set"
		}

		fmt.Println()
		fmt.Println("--- API Keys ---")
		fmt.Println("OpenRouter API:", status(hasOpenRouter))
		fmt.Println("Anthropic API:", status(hasAnthropic))
		fmt.Println("OpenAI API:", status(hasOpenAI))
		fmt.Println("Gemini API:", status(hasGemini))
		fmt.Println("Zhipu API:", status(hasZhipu))
		fmt.Println("Qwen API:", status(hasQwen))
		fmt.Println("Groq API:", status(hasGroq))
		fmt.Println("Moonshot API:", status(hasMoonshot))
		fmt.Println("DeepSeek API:", status(hasDeepSeek))
		fmt.Println("VolcEngine API:", status(hasVolcEngine))
		fmt.Println("Nvidia API:", status(hasNvidia))

		// Show local/ollama status
		if cfg.Providers.VLLM.APIBase != "" {
			fmt.Printf("vLLM/Local: ✓ %s\n", cfg.Providers.VLLM.APIBase)
		} else if modelListHasLocal {
			// Find and show the local endpoint
			for _, mc := range cfg.ModelList {
				if mc.APIBase != "" && (strings.Contains(mc.APIBase, "localhost") || strings.Contains(mc.APIBase, "127.0.0.1")) {
					fmt.Printf("Local: ✓ %s\n", mc.APIBase)
					break
				}
			}
		} else {
			fmt.Println("vLLM/Local:", status(hasVLLM))
		}

		if cfg.Providers.Ollama.APIBase != "" {
			fmt.Printf("Ollama: ✓ %s\n", cfg.Providers.Ollama.APIBase)
		} else if modelListHasOllama {
			fmt.Println("Ollama: ✓ (via model_list)")
		} else {
			fmt.Println("Ollama:", status(hasOllama))
		}

		if hasAntigravityOAuth {
			fmt.Println("Antigravity: ✓ (OAuth)")
		}

		store, _ := auth.LoadStore()
		if store != nil && len(store.Credentials) > 0 {
			fmt.Println()
			fmt.Println("--- OAuth/Token Auth ---")
			for provider, cred := range store.Credentials {
				authStatus := "authenticated"
				if cred.IsExpired() {
					authStatus = "expired"
				} else if cred.NeedsRefresh() {
					authStatus = "needs refresh"
				}
				fmt.Printf("  %s (%s): %s\n", provider, cred.AuthMethod, authStatus)
			}
		}
	}
}
