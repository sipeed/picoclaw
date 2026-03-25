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

	var authMethod, modelName, model string
	var matchFn func(string) bool

	switch provider {
	case "openai":
		authMethod, modelName, model = "oauth", "gpt-5.2", "openai/gpt-5.2"
		matchFn = isOpenAIModel
	case "anthropic":
		authMethod, modelName, model = "token", "claude-sonnet-4.6", "anthropic/claude-sonnet-4.6"
		matchFn = isAnthropicModel
	case "google-antigravity":
		authMethod, modelName, model = "oauth", "gemini-flash", "antigravity/gemini-3-flash"
		matchFn = isAntigravityModel
	default:
		return
	}

	found := false
	for _, m := range cfg.ModelList {
		if matchFn(m.Model) {
			m.AuthMethod = authMethod
			found = true
			break
		}
	}
	if !found {
		cfg.ModelList = append(cfg.ModelList, &config.ModelConfig{
			ModelName:  modelName,
			Model:      model,
			AuthMethod: authMethod,
		})
	}
	cfg.Agents.Defaults.ModelName = modelName

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

	for _, m := range cfg.ModelList {
		switch provider {
		case "openai":
			if isOpenAIModel(m.Model) {
				m.AuthMethod = ""
			}
		case "anthropic":
			if isAnthropicModel(m.Model) {
				m.AuthMethod = ""
			}
		case "google-antigravity", "antigravity":
			if isAntigravityModel(m.Model) {
				m.AuthMethod = ""
			}
		}
	}

	config.SaveConfig(configPath, cfg)
}

// clearAllAuthMethodsInConfig clears auth_method for all providers in config.json.
func clearAllAuthMethodsInConfig(configPath string) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return
	}
	for _, m := range cfg.ModelList {
		m.AuthMethod = ""
	}
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
