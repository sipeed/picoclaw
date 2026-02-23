package config

import (
	"strings"
)

// ChannelInfo describes a channel and the credential keys it expects.
type ChannelInfo struct {
	Name                string   `json:"name"`
	Enabled             bool     `json:"enabled"`
	RequiredCredentials []string `json:"required_credentials"` // config keys needed (tokens/secrets)
}

// ProviderInfo describes a provider and its credential requirements.
type ProviderInfo struct {
	Name                string   `json:"name"`
	RequiredCredentials []string `json:"required_credentials"`
	OptionalCredentials []string `json:"optional_credentials"`
	HasCredentials      bool     `json:"has_credentials"`
}

// GetAllChannelNames returns all channel keys available in ChannelsConfig.
func GetAllChannelNames() []string {
	return []string{
		"whatsapp",
		"telegram",
		"feishu",
		"discord",
		"maixcam",
		"qq",
		"dingtalk",
		"slack",
		"line",
		"onebot",
		"wecom",
		"wecom_app",
	}
}

var channelRequirements = map[string][]string{
	"whatsapp":  {"bridge_url"},
	"telegram":  {"token"},
	"feishu":    {"app_id", "app_secret"},
	"discord":   {"token"},
	"maixcam":   {"host", "port"},
	"qq":        {"app_id", "app_secret"},
	"dingtalk":  {"client_id", "client_secret"},
	"slack":     {"bot_token", "app_token"},
	"line":      {"channel_secret", "channel_access_token"},
	"onebot":    {"ws_url", "access_token"},
	"wecom":     {"token", "encoding_aes_key"},
	"wecom_app": {"corp_id", "corp_secret", "agent_id", "token"},
}

// GetChannelRequirements returns the list of credential keys expected for that channel.
// These are the config field names (not values) that are typically required to operate
// the channel (e.g. "token", "bot_token"). Returns nil if the channel is unknown.
func GetChannelRequirements(channel string) []string {
	return channelRequirements[strings.ToLower(channel)]
}

// GetEnabledChannelsInfo returns structured information about all known channels
// including whether they are enabled in the provided config and which credentials
// they expect.
func GetEnabledChannelsInfo(c *Config) []ChannelInfo {
	var res []ChannelInfo
	if c == nil {
		// return requirements only (not enabled) if config not provided
		for _, name := range GetAllChannelNames() {
			res = append(res, ChannelInfo{Name: name, Enabled: false, RequiredCredentials: GetChannelRequirements(name)})
		}
		return res
	}

	// helper to append
	add := func(name string, enabled bool) {
		res = append(res, ChannelInfo{Name: name, Enabled: enabled, RequiredCredentials: GetChannelRequirements(name)})
	}

	add("whatsapp", c.Channels.WhatsApp.Enabled)
	add("telegram", c.Channels.Telegram.Enabled)
	add("feishu", c.Channels.Feishu.Enabled)
	add("discord", c.Channels.Discord.Enabled)
	add("maixcam", c.Channels.MaixCam.Enabled)
	add("qq", c.Channels.QQ.Enabled)
	add("dingtalk", c.Channels.DingTalk.Enabled)
	add("slack", c.Channels.Slack.Enabled)
	add("line", c.Channels.LINE.Enabled)
	add("onebot", c.Channels.OneBot.Enabled)
	add("wecom", c.Channels.WeCom.Enabled)
	add("wecom_app", c.Channels.WeComApp.Enabled)

	return res
}

// GetSupportedModelNames returns all model_name entries from model_list and
// gathers model names referenced in agents defaults and agent entries.
func GetSupportedModelNames(c *Config) []string {
	if c == nil {
		return nil
	}
	set := make(map[string]struct{})
	for _, m := range c.ModelList {
		if m.ModelName != "" {
			set[m.ModelName] = struct{}{}
		}
	}

	// Agents defaults
	if c.Agents.Defaults.Model != "" {
		set[c.Agents.Defaults.Model] = struct{}{}
	}
	for _, f := range c.Agents.Defaults.ModelFallbacks {
		if f != "" {
			set[f] = struct{}{}
		}
	}

	// Agent entries
	for _, a := range c.Agents.List {
		if a.Model != nil {
			if a.Model.Primary != "" {
				set[a.Model.Primary] = struct{}{}
			}
			for _, fb := range a.Model.Fallbacks {
				if fb != "" {
					set[fb] = struct{}{}
				}
			}
		}
	}

	res := make([]string, 0, len(set))
	for k := range set {
		res = append(res, k)
	}
	return res
}

// Known providers and their typical credential requirements.
var knownProviders = map[string]ProviderInfo{
	"anthropic":      {Name: "anthropic", RequiredCredentials: []string{"api_key"}, OptionalCredentials: []string{"api_base"}},
	"openai":         {Name: "openai", RequiredCredentials: []string{"api_key"}, OptionalCredentials: []string{"api_base"}},
	"openrouter":     {Name: "openrouter", RequiredCredentials: []string{"api_key"}, OptionalCredentials: []string{"api_base"}},
	"groq":           {Name: "groq", RequiredCredentials: []string{"api_key"}},
	"zhipu":          {Name: "zhipu", RequiredCredentials: []string{"api_key"}},
	"vllm":           {Name: "vllm", RequiredCredentials: []string{"api_key"}},
	"gemini":         {Name: "gemini", RequiredCredentials: []string{"api_key"}},
	"nvidia":         {Name: "nvidia", RequiredCredentials: []string{"api_key"}},
	"ollama":         {Name: "ollama", RequiredCredentials: []string{"api_key"}},
	"moonshot":       {Name: "moonshot", RequiredCredentials: []string{"api_key"}},
	"shengsuanyun":   {Name: "shengsuanyun", RequiredCredentials: []string{"api_key"}},
	"deepseek":       {Name: "deepseek", RequiredCredentials: []string{"api_key"}},
	"cerebras":       {Name: "cerebras", RequiredCredentials: []string{"api_key"}},
	"volcengine":     {Name: "volcengine", RequiredCredentials: []string{"api_key"}},
	"github_copilot": {Name: "github_copilot", RequiredCredentials: []string{"api_key"}},
	"antigravity":    {Name: "antigravity", RequiredCredentials: []string{"api_key"}},
	"qwen":           {Name: "qwen", RequiredCredentials: []string{"api_key"}},
}

// GetProvidersInfo returns provider metadata including required/optional fields
// and whether credentials are present in the current config.
func GetProvidersInfo(c *Config) []ProviderInfo {
	// start from known providers map
	res := make([]ProviderInfo, 0, len(knownProviders))
	has := map[string]bool{}

	if c != nil {
		// record legacy providers that have keys
		if c.Providers.Anthropic.APIKey != "" || c.Providers.Anthropic.APIBase != "" {
			has["anthropic"] = true
		}
		if c.Providers.OpenAI.APIKey != "" || c.Providers.OpenAI.APIBase != "" {
			has["openai"] = true
		}
		if c.Providers.OpenRouter.APIKey != "" || c.Providers.OpenRouter.APIBase != "" {
			has["openrouter"] = true
		}
		if c.Providers.Groq.APIKey != "" || c.Providers.Groq.APIBase != "" {
			has["groq"] = true
		}
		if c.Providers.Zhipu.APIKey != "" || c.Providers.Zhipu.APIBase != "" {
			has["zhipu"] = true
		}
		if c.Providers.VLLM.APIKey != "" || c.Providers.VLLM.APIBase != "" {
			has["vllm"] = true
		}
		if c.Providers.Gemini.APIKey != "" || c.Providers.Gemini.APIBase != "" {
			has["gemini"] = true
		}
		if c.Providers.Nvidia.APIKey != "" || c.Providers.Nvidia.APIBase != "" {
			has["nvidia"] = true
		}
		if c.Providers.Ollama.APIKey != "" || c.Providers.Ollama.APIBase != "" {
			has["ollama"] = true
		}
		if c.Providers.Moonshot.APIKey != "" || c.Providers.Moonshot.APIBase != "" {
			has["moonshot"] = true
		}
		if c.Providers.ShengSuanYun.APIKey != "" || c.Providers.ShengSuanYun.APIBase != "" {
			has["shengsuanyun"] = true
		}
		if c.Providers.DeepSeek.APIKey != "" || c.Providers.DeepSeek.APIBase != "" {
			has["deepseek"] = true
		}
		if c.Providers.Cerebras.APIKey != "" || c.Providers.Cerebras.APIBase != "" {
			has["cerebras"] = true
		}
		if c.Providers.VolcEngine.APIKey != "" || c.Providers.VolcEngine.APIBase != "" {
			has["volcengine"] = true
		}
		if c.Providers.GitHubCopilot.APIKey != "" || c.Providers.GitHubCopilot.APIBase != "" {
			has["github_copilot"] = true
		}
		if c.Providers.Antigravity.APIKey != "" || c.Providers.Antigravity.APIBase != "" {
			has["antigravity"] = true
		}
		if c.Providers.Qwen.APIKey != "" || c.Providers.Qwen.APIBase != "" {
			has["qwen"] = true
		}

		// Model-list based credentials: map model protocol -> present
		for _, m := range c.ModelList {
			if m.APIKey != "" || m.APIBase != "" {
				p := parseProtocol(m.Model)
				if p == "" {
					p = "openai"
				}
				has[p] = true
			}
		}
	}

	for k, info := range knownProviders {
		pi := info
		// if knownProviders entry didn't set optional/required copies, ensure Name set
		if pi.Name == "" {
			pi.Name = k
		}
		if has[k] {
			pi.HasCredentials = true
		}
		res = append(res, pi)
	}
	return res
}

// parseProtocol extracts the protocol prefix from a model string of the form
// "protocol/model-identifier". If no prefix exists, returns empty string.
func parseProtocol(model string) string {
	if model == "" {
		return ""
	}
	if !strings.Contains(model, "/") {
		return ""
	}
	parts := strings.SplitN(model, "/", 2)
	p := strings.ToLower(strings.TrimSpace(parts[0]))
	return p
}
