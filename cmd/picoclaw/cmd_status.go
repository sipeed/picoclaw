// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/auth"
)

func statusCmd() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
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
		fmt.Printf("Model: %s\n", cfg.Agents.Defaults.Model)

		// Build a map of providers from model_list
		modelProviders := make(map[string]bool)
		for _, model := range cfg.ModelList {
			if model.APIKey == "" {
				continue
			}
			modelStr := strings.ToLower(model.Model)
			// Extract provider name (before "/")
			if idx := strings.Index(modelStr, "/"); idx > 0 {
				provider := modelStr[:idx]
				modelProviders[provider] = true
				// Add aliases
				switch provider {
				case "doubao":
					modelProviders["volcengine"] = true
				case "claude":
					modelProviders["anthropic"] = true
				case "gpt":
					modelProviders["openai"] = true
				case "tongyi":
					modelProviders["qwen"] = true
				case "kimi":
					modelProviders["moonshot"] = true
				case "glm":
					modelProviders["zhipu"] = true
				}
			}
		}

		// Check providers (legacy) or model_list (new)
		hasOpenRouter := cfg.Providers.OpenRouter.APIKey != "" || modelProviders["openrouter"]
		hasAnthropic := cfg.Providers.Anthropic.APIKey != "" || modelProviders["anthropic"]
		hasOpenAI := cfg.Providers.OpenAI.APIKey != "" || modelProviders["openai"]
		hasGemini := cfg.Providers.Gemini.APIKey != "" || modelProviders["gemini"]
		hasZhipu := cfg.Providers.Zhipu.APIKey != "" || modelProviders["zhipu"]
		hasQwen := cfg.Providers.Qwen.APIKey != "" || modelProviders["qwen"]
		hasGroq := cfg.Providers.Groq.APIKey != "" || modelProviders["groq"]
		hasVLLM := cfg.Providers.VLLM.APIBase != "" || modelProviders["vllm"]
		hasMoonshot := cfg.Providers.Moonshot.APIKey != "" || modelProviders["moonshot"]
		hasDeepSeek := cfg.Providers.DeepSeek.APIKey != "" || modelProviders["deepseek"]
		hasVolcEngine := cfg.Providers.VolcEngine.APIKey != "" || modelProviders["volcengine"]
		hasNvidia := cfg.Providers.Nvidia.APIKey != "" || modelProviders["nvidia"]
		hasOllama := cfg.Providers.Ollama.APIBase != "" || modelProviders["ollama"]

		status := func(enabled bool) string {
			if enabled {
				return "✓"
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
			fmt.Printf("vLLM/Local: ✓ %s\n", cfg.Providers.VLLM.APIBase)
		} else {
			fmt.Println("vLLM/Local: not set")
		}
		if hasOllama {
			fmt.Printf("Ollama: ✓ %s\n", cfg.Providers.Ollama.APIBase)
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

		// Display channel status
		fmt.Println("\nChannels:")
		channelStatus := func(enabled bool, name string, details ...string) {
			if enabled {
				if len(details) > 0 {
					fmt.Printf("  %s: ✓ %s\n", name, details[0])
				} else {
					fmt.Printf("  %s: ✓\n", name)
				}
			} else {
				fmt.Printf("  %s: disabled\n", name)
			}
		}

		channelStatus(cfg.Channels.Telegram.Enabled, "Telegram")
		channelStatus(cfg.Channels.Discord.Enabled, "Discord")
		channelStatus(cfg.Channels.Feishu.Enabled, "Feishu")
		channelStatus(cfg.Channels.DingTalk.Enabled, "DingTalk")
		channelStatus(cfg.Channels.Slack.Enabled, "Slack")
		channelStatus(cfg.Channels.WhatsApp.Enabled, "WhatsApp")
		channelStatus(cfg.Channels.QQ.Enabled, "QQ")
		channelStatus(cfg.Channels.LINE.Enabled, "LINE")
		channelStatus(cfg.Channels.OneBot.Enabled, "OneBot")
		channelStatus(cfg.Channels.MaixCam.Enabled, "MaixCam")
		if cfg.Channels.WebSocket.Enabled {
			hostPort := net.JoinHostPort(cfg.Channels.WebSocket.Host, strconv.Itoa(cfg.Channels.WebSocket.Port))
			addr := "http://" + hostPort
			channelStatus(true, "WebSocket", addr)
		} else {
			channelStatus(false, "WebSocket")
		}
	}
}
