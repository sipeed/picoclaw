package server

import (
	"log"
	"strings"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
)

// updateConfigAfterLogin updates config.json after a successful provider login.
func updateConfigAfterLogin(configPath, provider string, cred *auth.AuthCredential) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("Warning: could not load config to update auth_method: %v", err)
		return
	}

	switch provider {
	case "openai":
		cfg.Providers.OpenAI.AuthMethod = "oauth"
		found := false
		for i := range cfg.ModelList {
			if isOpenAIModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = "oauth"
				found = true
				break
			}
		}
		if !found {
			cfg.ModelList = append(cfg.ModelList, config.ModelConfig{
				ModelName:  "gpt-5.2",
				Model:      "openai/gpt-5.2",
				AuthMethod: "oauth",
			})
		}
		cfg.Agents.Defaults.ModelName = "gpt-5.2"

	case "anthropic":
		cfg.Providers.Anthropic.AuthMethod = "token"
		found := false
		for i := range cfg.ModelList {
			if isAnthropicModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = "token"
				found = true
				break
			}
		}
		if !found {
			cfg.ModelList = append(cfg.ModelList, config.ModelConfig{
				ModelName:  "claude-sonnet-4.6",
				Model:      "anthropic/claude-sonnet-4.6",
				AuthMethod: "token",
			})
		}
		cfg.Agents.Defaults.ModelName = "claude-sonnet-4.6"

	case "google-antigravity":
		cfg.Providers.Antigravity.AuthMethod = "oauth"
		found := false
		for i := range cfg.ModelList {
			if isAntigravityModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = "oauth"
				found = true
				break
			}
		}
		if !found {
			cfg.ModelList = append(cfg.ModelList, config.ModelConfig{
				ModelName:  "gemini-flash",
				Model:      "antigravity/gemini-3-flash",
				AuthMethod: "oauth",
			})
		}
		cfg.Agents.Defaults.ModelName = "gemini-flash"
	}

	if err := config.SaveConfig(configPath, cfg); err != nil {
		log.Printf("Warning: could not update config: %v", err)
	}
}

// clearAuthMethodInConfig clears auth_method for a specific provider in config.json.
func clearAuthMethodInConfig(configPath, provider string) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return
	}

	for i := range cfg.ModelList {
		switch provider {
		case "openai":
			if isOpenAIModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = ""
			}
		case "anthropic":
			if isAnthropicModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = ""
			}
		case "google-antigravity", "antigravity":
			if isAntigravityModel(cfg.ModelList[i].Model) {
				cfg.ModelList[i].AuthMethod = ""
			}
		}
	}

	switch provider {
	case "openai":
		cfg.Providers.OpenAI.AuthMethod = ""
	case "anthropic":
		cfg.Providers.Anthropic.AuthMethod = ""
	case "google-antigravity", "antigravity":
		cfg.Providers.Antigravity.AuthMethod = ""
	}

	config.SaveConfig(configPath, cfg)
}

// clearAllAuthMethodsInConfig clears auth_method for all providers in config.json.
func clearAllAuthMethodsInConfig(configPath string) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return
	}
	for i := range cfg.ModelList {
		cfg.ModelList[i].AuthMethod = ""
	}
	cfg.Providers.OpenAI.AuthMethod = ""
	cfg.Providers.Anthropic.AuthMethod = ""
	cfg.Providers.Antigravity.AuthMethod = ""
	config.SaveConfig(configPath, cfg)
}

// ── Model identification helpers ─────────────────────────────────

func isOpenAIModel(model string) bool {
	return model == "openai" || strings.HasPrefix(model, "openai/")
}

func isAnthropicModel(model string) bool {
	return model == "anthropic" || strings.HasPrefix(model, "anthropic/")
}

func isAntigravityModel(model string) bool {
	return model == "antigravity" || model == "google-antigravity" ||
		strings.HasPrefix(model, "antigravity/") || strings.HasPrefix(model, "google-antigravity/")
}
