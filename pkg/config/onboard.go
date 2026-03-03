package config

// MinimalOnboardConfig produces a stripped-down config for initial onboarding.
// It keeps only the essentials, omitting empty channels, providers, and
// model_list entries without API keys.
func MinimalOnboardConfig(full *Config) *Config {
	// Filter model_list: only keep entries that have an API key set,
	// or special auth (e.g., OAuth, Ollama local).
	var models []ModelConfig
	for _, m := range full.ModelList {
		if m.APIKey != "" || m.AuthMethod != "" {
			models = append(models, m)
		}
	}

	return &Config{
		Agents: AgentsConfig{
			Defaults: full.Agents.Defaults,
		},
		Session:   full.Session,
		ModelList:  models,
		Gateway:   full.Gateway,
		Tools: ToolsConfig{
			Exec: full.Tools.Exec,
			Web: WebToolsConfig{
				DuckDuckGo: full.Tools.Web.DuckDuckGo,
			},
		},
	}
}
