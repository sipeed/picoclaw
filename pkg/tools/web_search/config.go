package web_search

import (
	"github.com/sipeed/picoclaw/pkg/config"
)

type WebToolsConfig struct {
	Brave      BraveConfig
	Tavily     TavilyConfig
	DuckDuckGo DuckDuckGoConfig
	Perplexity PerplexityConfig
	Proxy      string
	Enabled    bool
}

type BraveConfig struct {
	Enabled    bool
	APIKey     string
	MaxResults int
}

type TavilyConfig struct {
	Enabled    bool
	APIKey     string
	BaseURL    string
	MaxResults int
}

type DuckDuckGoConfig struct {
	Enabled    bool
	MaxResults int
}

type PerplexityConfig struct {
	Enabled    bool
	APIKey     string
	MaxResults int
}

func GetWebToolsConfig(cfg *config.Config) WebToolsConfig {
	tc := cfg.GetTool("web")
	if tc == nil || !tc.Enabled {
		return WebToolsConfig{}
	}
	extra := tc.Extra
	if extra == nil {
		return WebToolsConfig{Enabled: true}
	}
	webCfg := WebToolsConfig{Enabled: true}

	if v, ok := extra["brave"]; ok {
		if m, ok := v.(map[string]any); ok {
			webCfg.Brave = BraveConfig{
				Enabled:    config.GetBoolOrDefault(m, "enabled", false),
				APIKey:     config.GetStringOrDefault(m, "api_key", ""),
				MaxResults: config.GetIntOrDefault(m, "max_results", 5),
			}
		}
	}
	if v, ok := extra["tavily"]; ok {
		if m, ok := v.(map[string]any); ok {
			webCfg.Tavily = TavilyConfig{
				Enabled:    config.GetBoolOrDefault(m, "enabled", false),
				APIKey:     config.GetStringOrDefault(m, "api_key", ""),
				MaxResults: config.GetIntOrDefault(m, "max_results", 5),
				BaseURL:    config.GetStringOrDefault(m, "base_url", ""),
			}
		}
	}
	if v, ok := extra["duckduckgo"]; ok {
		if m, ok := v.(map[string]any); ok {
			webCfg.DuckDuckGo = DuckDuckGoConfig{
				Enabled:    config.GetBoolOrDefault(m, "enabled", false),
				MaxResults: config.GetIntOrDefault(m, "max_results", 5),
			}
		}
	}
	if v, ok := extra["perplexity"]; ok {
		if m, ok := v.(map[string]any); ok {
			webCfg.Perplexity = PerplexityConfig{
				Enabled:    config.GetBoolOrDefault(m, "enabled", false),
				APIKey:     config.GetStringOrDefault(m, "api_key", ""),
				MaxResults: config.GetIntOrDefault(m, "max_results", 5),
			}
		}
	}
	if v, ok := extra["proxy"]; ok {
		if s, ok := v.(string); ok {
			webCfg.Proxy = s
		}
	}
	return webCfg
}
