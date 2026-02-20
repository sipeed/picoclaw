// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"fmt"
	"os"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show picoclaw status",
		RunE:  runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return nil
	}

	configPath := getConfigPath()

	fmt.Printf("%s picoclaw Status\n", logo)
	fmt.Printf("Version: %s\n", formatVersion())
	build, _ := formatBuildInfo()
	if build != "" {
		fmt.Printf("Build: %s\n", build)
	}
	fmt.Println()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("Config:", configPath, "\u2713")
	} else {
		fmt.Println("Config:", configPath, "\u2717")
	}

	workspace := cfg.WorkspacePath()
	if _, err := os.Stat(workspace); err == nil {
		fmt.Println("Workspace:", workspace, "\u2713")
	} else {
		fmt.Println("Workspace:", workspace, "\u2717")
	}

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Model: %s\n", cfg.Agents.Defaults.Model)

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
				return "\u2713"
			}
			return "not set"
		}
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
		if hasVLLM {
			fmt.Printf("vLLM/Local: \u2713 %s\n", cfg.Providers.VLLM.APIBase)
		} else {
			fmt.Println("vLLM/Local: not set")
		}
		if hasOllama {
			fmt.Printf("Ollama: \u2713 %s\n", cfg.Providers.Ollama.APIBase)
		} else {
			fmt.Println("Ollama: not set")
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
	return nil
}
