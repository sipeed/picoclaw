package config

import (
	"strings"
)

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

// GetEnabledChannels returns the list of channels that are enabled in the config.
func GetEnabledChannels(c *Config) []string {
	var res []string
	if c == nil {
		return res
	}
	if c.Channels.WhatsApp.Enabled {
		res = append(res, "whatsapp")
	}
	if c.Channels.Telegram.Enabled {
		res = append(res, "telegram")
	}
	if c.Channels.Feishu.Enabled {
		res = append(res, "feishu")
	}
	if c.Channels.Discord.Enabled {
		res = append(res, "discord")
	}
	if c.Channels.MaixCam.Enabled {
		res = append(res, "maixcam")
	}
	if c.Channels.QQ.Enabled {
		res = append(res, "qq")
	}
	if c.Channels.DingTalk.Enabled {
		res = append(res, "dingtalk")
	}
	if c.Channels.Slack.Enabled {
		res = append(res, "slack")
	}
	if c.Channels.LINE.Enabled {
		res = append(res, "line")
	}
	if c.Channels.OneBot.Enabled {
		res = append(res, "onebot")
	}
	if c.Channels.WeCom.Enabled {
		res = append(res, "wecom")
	}
	if c.Channels.WeComApp.Enabled {
		res = append(res, "wecom_app")
	}
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

// GetSupportedProviders returns a deduplicated list of provider identifiers.
// It includes known provider fields and protocols discovered in model_list entries
// and the agents default provider if set.
func GetSupportedProviders(c *Config) []string {
	known := []string{
		"anthropic", "openai", "openrouter", "groq", "zhipu", "vllm",
		"gemini", "nvidia", "ollama", "moonshot", "shengsuanyun",
		"deepseek", "cerebras", "volcengine", "github_copilot", "antigravity",
		"qwen",
	}

	set := make(map[string]struct{})
	for _, k := range known {
		set[k] = struct{}{}
	}

	// Add provider from agents defaults if present
	if c != nil && c.Agents.Defaults.Provider != "" {
		set[strings.ToLower(c.Agents.Defaults.Provider)] = struct{}{}
	}

	// Discover protocols used in model_list (protocol/model format)
	if c != nil {
		for _, m := range c.ModelList {
			if m.Model == "" {
				continue
			}
			p := parseProtocol(m.Model)
			if p != "" {
				set[p] = struct{}{}
			}
		}
	}

	res := make([]string, 0, len(set))
	for k := range set {
		res = append(res, k)
	}
	return res
}

// GetProvidersWithCredentials returns providers that have API key or API base set
// in the legacy ProvidersConfig, as well as protocols with credentials defined
// in model_list entries (ModelConfig.APIKey/APIBase).
func GetProvidersWithCredentials(c *Config) []string {
	if c == nil {
		return nil
	}
	set := make(map[string]struct{})

	// Legacy providers
	if c.Providers.Anthropic.APIKey != "" || c.Providers.Anthropic.APIBase != "" {
		set["anthropic"] = struct{}{}
	}
	if c.Providers.OpenAI.APIKey != "" || c.Providers.OpenAI.APIBase != "" {
		set["openai"] = struct{}{}
	}
	if c.Providers.OpenRouter.APIKey != "" || c.Providers.OpenRouter.APIBase != "" {
		set["openrouter"] = struct{}{}
	}
	if c.Providers.Groq.APIKey != "" || c.Providers.Groq.APIBase != "" {
		set["groq"] = struct{}{}
	}
	if c.Providers.Zhipu.APIKey != "" || c.Providers.Zhipu.APIBase != "" {
		set["zhipu"] = struct{}{}
	}
	if c.Providers.VLLM.APIKey != "" || c.Providers.VLLM.APIBase != "" {
		set["vllm"] = struct{}{}
	}
	if c.Providers.Gemini.APIKey != "" || c.Providers.Gemini.APIBase != "" {
		set["gemini"] = struct{}{}
	}
	if c.Providers.Nvidia.APIKey != "" || c.Providers.Nvidia.APIBase != "" {
		set["nvidia"] = struct{}{}
	}
	if c.Providers.Ollama.APIKey != "" || c.Providers.Ollama.APIBase != "" {
		set["ollama"] = struct{}{}
	}
	if c.Providers.Moonshot.APIKey != "" || c.Providers.Moonshot.APIBase != "" {
		set["moonshot"] = struct{}{}
	}
	if c.Providers.ShengSuanYun.APIKey != "" || c.Providers.ShengSuanYun.APIBase != "" {
		set["shengsuanyun"] = struct{}{}
	}
	if c.Providers.DeepSeek.APIKey != "" || c.Providers.DeepSeek.APIBase != "" {
		set["deepseek"] = struct{}{}
	}
	if c.Providers.Cerebras.APIKey != "" || c.Providers.Cerebras.APIBase != "" {
		set["cerebras"] = struct{}{}
	}
	if c.Providers.VolcEngine.APIKey != "" || c.Providers.VolcEngine.APIBase != "" {
		set["volcengine"] = struct{}{}
	}
	if c.Providers.GitHubCopilot.APIKey != "" || c.Providers.GitHubCopilot.APIBase != "" {
		set["github_copilot"] = struct{}{}
	}
	if c.Providers.Antigravity.APIKey != "" || c.Providers.Antigravity.APIBase != "" {
		set["antigravity"] = struct{}{}
	}
	if c.Providers.Qwen.APIKey != "" || c.Providers.Qwen.APIBase != "" {
		set["qwen"] = struct{}{}
	}

	// Model-list based providers
	for _, m := range c.ModelList {
		if m.APIKey != "" || m.APIBase != "" {
			p := parseProtocol(m.Model)
			if p == "" {
				p = "openai"
			}
			set[p] = struct{}{}
		}
	}

	res := make([]string, 0, len(set))
	for k := range set {
		res = append(res, k)
	}
	return res
}

// GetWebTools returns the names of web tools configured (those present in Tools.Web).
func GetWebTools(c *Config) []string {
	if c == nil {
		return nil
	}
	var res []string
	if c.Tools.Web.Brave.Enabled {
		res = append(res, "brave")
	}
	if c.Tools.Web.Tavily.Enabled {
		res = append(res, "tavily")
	}
	if c.Tools.Web.DuckDuckGo.Enabled {
		res = append(res, "duckduckgo")
	}
	if c.Tools.Web.Perplexity.Enabled {
		res = append(res, "perplexity")
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
