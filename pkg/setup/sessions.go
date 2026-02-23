package setup

import (
	"fmt"
	"net"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

type ChannelField struct {
	ID          string
	Prompt      string
	Placeholder string
	Info        string
}

type ChannelInfo struct {
	Fields []ChannelField
}

var channelInfoMap = map[string]ChannelInfo{
	"telegram": {
		Fields: []ChannelField{
			{
				ID: "token", Prompt: "Bot Token", Placeholder: "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz",
				Info: "Get from @BotFather on Telegram",
			},
		},
	},
	"slack": {
		Fields: []ChannelField{
			{
				ID: "bot_token", Prompt: "Bot Token", Placeholder: "xoxb-xxxxx-xxxxx",
				Info: "Bot token from Slack App settings (starts with xoxb-)",
			},
			{
				ID: "app_token", Prompt: "App Token", Placeholder: "xoxa-xxxxx-xxxxx",
				Info: "App token from Slack App settings (starts with xoxa-)",
			},
		},
	},
	"discord": {
		Fields: []ChannelField{
			{
				ID:          "token",
				Prompt:      "Bot Token",
				Placeholder: "MTExxxxxxxxxxxx.MDExxxxxxxx.xxxxx",
				Info:        "Get from Discord Developer Portal > Bot",
			},
		},
	},
	"whatsapp": {
		Fields: []ChannelField{
			{
				ID:          "bridge_url",
				Prompt:      "Bridge URL",
				Placeholder: "http://localhost:9080",
				Info:        "URL of your WhatsApp bridge service",
			},
		},
	},
	"feishu": {
		Fields: []ChannelField{
			{ID: "app_id", Prompt: "App ID", Placeholder: "cli_xxxxx", Info: "App ID from Feishu Open Platform > App"},
			{
				ID:          "app_secret",
				Prompt:      "App Secret",
				Placeholder: "xxxxxxxxxxxxxxxx",
				Info:        "App Secret from Feishu Open Platform > App",
			},
		},
	},
	"dingtalk": {
		Fields: []ChannelField{
			{
				ID:          "client_id",
				Prompt:      "Client ID",
				Placeholder: "dingxxxxx",
				Info:        "Client ID from DingTalk Admin > OAuth",
			},
			{
				ID:          "client_secret",
				Prompt:      "Client Secret",
				Placeholder: "xxxxxxxxxxxxxxxx",
				Info:        "Client Secret from DingTalk Admin > OAuth",
			},
		},
	},
	"line": {
		Fields: []ChannelField{
			{
				ID:          "channel_access_token",
				Prompt:      "Channel Access Token",
				Placeholder: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
				Info:        "Get from LINE Developers > Channel Access Token",
			},
		},
	},
	"qq": {
		Fields: []ChannelField{
			{ID: "app_id", Prompt: "App ID", Placeholder: "1234567890", Info: "App ID from QQ Open Platform"},
		},
	},
	"onebot": {
		Fields: []ChannelField{
			{
				ID:          "access_token",
				Prompt:      "Access Token",
				Placeholder: "your_access_token_here",
				Info:        "Access token configured in your OneBot server",
			},
		},
	},
	"wecom": {
		Fields: []ChannelField{
			{ID: "token", Prompt: "Token", Placeholder: "your_token", Info: "Token from Wecom Admin"},
			{
				ID:          "encoding_aes_key",
				Prompt:      "Encoding AES Key",
				Placeholder: "your_encoding_aes_key",
				Info:        "Encoding AES Key from Wecom Admin",
			},
		},
	},
	"wecom_app": {
		Fields: []ChannelField{
			{ID: "corp_id", Prompt: "Corp ID", Placeholder: "wwxxxxx", Info: "Corp ID from Wecom Admin"},
			{ID: "agent_id", Prompt: "Agent ID", Placeholder: "1000001", Info: "Agent ID from Wecom Admin"},
		},
	},
	"maixcam": {
		Fields: []ChannelField{
			{
				ID:          "device_address",
				Prompt:      "Device Address",
				Placeholder: "http://192.168.1.100:8080",
				Info:        "MaixCam device HTTP address",
			},
		},
	},
}

func GetChannelInfo(channel string) (ChannelInfo, bool) {
	info, ok := channelInfoMap[strings.ToLower(channel)]
	return info, ok
}

func isOAuthProvider(provider string) bool {
	switch strings.ToLower(provider) {
	case "openai", "anthropic", "google-antigravity", "antigravity":
		return true
	}
	return false
}

// QuestionType defines the type of response expected for a question.
type QuestionType string

const (
	QuestionTypeText   QuestionType = "text"
	QuestionTypeSelect QuestionType = "select"
	QuestionTypeYesNo  QuestionType = "yesno"
)

// Question represents a single question in a setup session.
type Question struct {
	ID           string       // unique identifier
	Type         QuestionType // response type: text, select, yesno
	Prompt       string       // question text shown to user
	Info         string       // optional helper text
	Options      []string     // options for select/yesno types
	DefaultValue string       // default value to prefill
	DependsOn    string       // question ID this depends on (optional)
	DependsValue string       // value that must be matched to show this question
}

// Session represents a group of questions shown together in one step.
type Session struct {
	ID          string                          // session identifier (e.g., "workspace", "provider", "model", "channel", "confirm")
	Title       string                          // step title shown to user
	Questions   []Question                      // questions in this session
	Transformer func(answers map[string]string) // optional: transform answers before applying to config
}

// AllSessions returns the complete list of setup sessions.
// Each session can have multiple questions shown together.
var AllSessions = []Session{
	{
		ID:    "workspace",
		Title: "Workspace Setup",
		Questions: []Question{
			{
				ID:           "workspace",
				Type:         QuestionTypeText,
				Prompt:       "Workspace path",
				Info:         "Directory for agent workspace files",
				DefaultValue: "~/.picoclaw/workspace",
			},
			{
				ID:           "restrict_workspace",
				Type:         QuestionTypeYesNo,
				Prompt:       "Restrict to workspace?",
				Options:      []string{"yes", "no"},
				DefaultValue: "no",
			},
		},
	},
	{
		ID:    "provider",
		Title: "Provider Setup",
		Questions: []Question{
			{
				ID:     "provider",
				Type:   QuestionTypeSelect,
				Prompt: "Choose provider",
			},
			{
				ID:        "provider_auth_method",
				Type:      QuestionTypeSelect,
				Prompt:    "Authentication method",
				Options:   []string{"api_key", "oauth_login"},
				DependsOn: "provider",
			},
			{
				ID:           "provider_api_key",
				Type:         QuestionTypeText,
				Prompt:       "API key",
				Info:         "API key for authentication",
				DependsOn:    "provider_auth_method",
				DependsValue: "api_key",
			},
		},
	},
	{
		ID:    "model",
		Title: "Model Setup",
		Questions: []Question{
			{
				ID:     "model_select",
				Type:   QuestionTypeSelect,
				Prompt: "Choose model",
			},
			{
				ID:           "custom_model",
				Type:         QuestionTypeText,
				Prompt:       "Custom model name",
				Info:         "Enter the model identifier",
				DependsOn:    "model_select",
				DependsValue: "custom",
			},
		},
	},
	{
		ID:    "channel",
		Title: "Channel Setup",
		Questions: []Question{
			{
				ID:     "channel_select",
				Type:   QuestionTypeSelect,
				Prompt: "Choose channel to connect",
				Info:   "Select a channel to enable",
			},
			{
				ID:           "channel_enable",
				Type:         QuestionTypeYesNo,
				Prompt:       "Enable this channel?",
				Options:      []string{"yes", "no"},
				DefaultValue: "yes",
				DependsOn:    "channel_select",
			},
			{
				ID:           "telegram_token",
				Type:         QuestionTypeText,
				Prompt:       "Bot Token",
				Info:         "Get from @BotFather on Telegram",
				DependsOn:    "channel_select",
				DependsValue: "telegram",
			},
			{
				ID:           "slack_bot_token",
				Type:         QuestionTypeText,
				Prompt:       "Bot Token",
				Info:         "Bot token from Slack App settings (starts with xoxb-)",
				DependsOn:    "channel_select",
				DependsValue: "slack",
			},
			{
				ID:           "slack_app_token",
				Type:         QuestionTypeText,
				Prompt:       "App Token",
				Info:         "App token from Slack App settings (starts with xoxa-)",
				DependsOn:    "channel_select",
				DependsValue: "slack",
			},
			{
				ID:           "discord_token",
				Type:         QuestionTypeText,
				Prompt:       "Bot Token",
				Info:         "Get from Discord Developer Portal > Bot",
				DependsOn:    "channel_select",
				DependsValue: "discord",
			},
			{
				ID:           "whatsapp_bridge_url",
				Type:         QuestionTypeText,
				Prompt:       "Bridge URL",
				Info:         "URL of your WhatsApp bridge service",
				DependsOn:    "channel_select",
				DependsValue: "whatsapp",
			},
			{
				ID:           "feishu_app_id",
				Type:         QuestionTypeText,
				Prompt:       "App ID",
				Info:         "App ID from Feishu Open Platform > App",
				DependsOn:    "channel_select",
				DependsValue: "feishu",
			},
			{
				ID:           "feishu_app_secret",
				Type:         QuestionTypeText,
				Prompt:       "App Secret",
				Info:         "App Secret from Feishu Open Platform > App",
				DependsOn:    "channel_select",
				DependsValue: "feishu",
			},
			{
				ID:           "dingtalk_client_id",
				Type:         QuestionTypeText,
				Prompt:       "Client ID",
				Info:         "Client ID from DingTalk Admin > OAuth",
				DependsOn:    "channel_select",
				DependsValue: "dingtalk",
			},
			{
				ID:           "dingtalk_client_secret",
				Type:         QuestionTypeText,
				Prompt:       "Client Secret",
				Info:         "Client Secret from DingTalk Admin > OAuth",
				DependsOn:    "channel_select",
				DependsValue: "dingtalk",
			},
			{
				ID:           "line_channel_access_token",
				Type:         QuestionTypeText,
				Prompt:       "Channel Access Token",
				Info:         "Get from LINE Developers > Channel Access Token",
				DependsOn:    "channel_select",
				DependsValue: "line",
			},
			{
				ID:           "qq_app_id",
				Type:         QuestionTypeText,
				Prompt:       "App ID",
				Info:         "App ID from QQ Open Platform",
				DependsOn:    "channel_select",
				DependsValue: "qq",
			},
			{
				ID:           "onebot_access_token",
				Type:         QuestionTypeText,
				Prompt:       "Access Token",
				Info:         "Access token configured in your OneBot server",
				DependsOn:    "channel_select",
				DependsValue: "onebot",
			},
			{
				ID:           "wecom_token",
				Type:         QuestionTypeText,
				Prompt:       "Token",
				Info:         "Token from Wecom Admin",
				DependsOn:    "channel_select",
				DependsValue: "wecom",
			},
			{
				ID:           "wecom_encoding_aes_key",
				Type:         QuestionTypeText,
				Prompt:       "Encoding AES Key",
				Info:         "Encoding AES Key from Wecom Admin",
				DependsOn:    "channel_select",
				DependsValue: "wecom",
			},
			{
				ID:           "wecom_app_corp_id",
				Type:         QuestionTypeText,
				Prompt:       "Corp ID",
				Info:         "Corp ID from Wecom Admin",
				DependsOn:    "channel_select",
				DependsValue: "wecom_app",
			},
			{
				ID:           "wecom_app_agent_id",
				Type:         QuestionTypeText,
				Prompt:       "Agent ID",
				Info:         "Agent ID from Wecom Admin",
				DependsOn:    "channel_select",
				DependsValue: "wecom_app",
			},
			{
				ID:           "maixcam_device_address",
				Type:         QuestionTypeText,
				Prompt:       "Device Address",
				Info:         "MaixCam device HTTP address",
				DependsOn:    "channel_select",
				DependsValue: "maixcam",
			},
		},
	},
	{
		ID:    "confirm",
		Title: "Confirmation",
		Questions: []Question{
			{
				ID:           "confirm",
				Type:         QuestionTypeYesNo,
				Prompt:       "Confirm and save configuration?",
				Options:      []string{"yes", "no"},
				DefaultValue: "yes",
			},
		},
	},
}

// SessionRegistry holds resolved sessions for a specific config state.
type SessionRegistry struct {
	Sessions []Session
	Answers  map[string]string // questionID -> answer
}

// BuildSessionRegistry creates the session registry with resolved options from config.
func BuildSessionRegistry(cfg *config.Config) SessionRegistry {
	registry := SessionRegistry{
		Sessions: make([]Session, len(AllSessions)),
		Answers:  map[string]string{},
	}

	// Collect provider options
	provInfo := config.GetProvidersInfo(cfg)
	ordered := config.GetOrderedProviderNames()
	provOptions := []string{}
	added := map[string]struct{}{}
	for _, name := range ordered {
		if _, ok := provInfoLookup(provInfo, name); ok {
			provOptions = append(provOptions, name)
			added[name] = struct{}{}
		}
	}
	for _, p := range provInfo {
		if _, ok := added[p.Name]; !ok {
			provOptions = append(provOptions, p.Name)
		}
	}

	// Collect channel options
	channelOptions := config.GetAllChannelNames()

	// Collect model options based on selected provider
	selProvider := cfg.Agents.Defaults.Provider
	var modelOptions []string
	if models := config.GetModelsForProvider(selProvider); len(models) > 0 {
		for _, m := range models {
			modelOptions = append(modelOptions, m.ID)
		}
	} else {
		modelSuggestions := config.GetPopularModels(selProvider)
		modelOptions = make([]string, 0, len(modelSuggestions))
		copy(modelOptions, modelSuggestions)
	}
	modelOptions = append(modelOptions, "custom")

	// Deep copy sessions and fill in options
	for i, sess := range AllSessions {
		registry.Sessions[i] = Session{
			ID:        sess.ID,
			Title:     sess.Title,
			Questions: make([]Question, len(sess.Questions)),
		}

		for j, q := range sess.Questions {
			registry.Sessions[i].Questions[j] = q

			// Fill in options for select/yesno types
			if q.Type == QuestionTypeSelect || q.Type == QuestionTypeYesNo {
				switch q.ID {
				case "provider":
					registry.Sessions[i].Questions[j].Options = provOptions
				case "provider_auth_method":
					// Determine auth options based on selected provider
					selectedProvider := cfg.Agents.Defaults.Provider
					if selectedProvider == "" {
						selectedProvider = registry.Answers["provider"]
					}
					if isOAuthProvider(selectedProvider) {
						registry.Sessions[i].Questions[j].Options = []string{"oauth_login", "api_key"}
					} else {
						registry.Sessions[i].Questions[j].Options = []string{"api_key"}
					}
				case "channel_select":
					registry.Sessions[i].Questions[j].Options = channelOptions
				case "model_select":
					registry.Sessions[i].Questions[j].Options = modelOptions
				case "restrict_workspace", "confirm":
					// Ensure yesno options are set
					if len(q.Options) == 0 {
						registry.Sessions[i].Questions[j].Options = []string{"yes", "no"}
					}
				}
			}

			// Pre-fill defaults from config
			if q.DefaultValue == "" {
				switch q.ID {
				case "workspace":
					if cfg.Agents.Defaults.Workspace != "" {
						registry.Sessions[i].Questions[j].DefaultValue = cfg.Agents.Defaults.Workspace
					}
				case "restrict_workspace":
					if cfg.Agents.Defaults.RestrictToWorkspace {
						registry.Sessions[i].Questions[j].DefaultValue = "yes"
					} else {
						registry.Sessions[i].Questions[j].DefaultValue = "no"
					}
				case "provider":
					if cfg.Agents.Defaults.Provider != "" {
						registry.Sessions[i].Questions[j].DefaultValue = cfg.Agents.Defaults.Provider
					}
				case "model_select":
					if cfg.Agents.Defaults.Model != "" {
						registry.Sessions[i].Questions[j].DefaultValue = cfg.Agents.Defaults.Model
					}
				case "custom_model":
					if cfg.Agents.Defaults.Model != "" {
						registry.Sessions[i].Questions[j].DefaultValue = cfg.Agents.Defaults.Model
					}
				}
			}
		}
	}

	return registry
}

// ShouldShowQuestion checks if a question should be visible based on current answers.
func (r *SessionRegistry) ShouldShowQuestion(questionID string) bool {
	for _, sess := range r.Sessions {
		for _, q := range sess.Questions {
			if q.ID == questionID {
				if q.DependsOn == "" {
					return true
				}
				depValue := r.Answers[q.DependsOn]
				if q.DependsValue != "" {
					return depValue == q.DependsValue
				}
				return depValue != ""
			}
		}
	}
	return false
}

// GetSessionByID returns a session by its ID.
func (r *SessionRegistry) GetSessionByID(id string) *Session {
	for i := range r.Sessions {
		if r.Sessions[i].ID == id {
			return &r.Sessions[i]
		}
	}
	return nil
}

// provInfoLookup is a helper to find provider info by name.
func provInfoLookup(list []config.ProviderInfo, name string) (config.ProviderInfo, bool) {
	for _, p := range list {
		if p.Name == name {
			return p, true
		}
	}
	return config.ProviderInfo{}, false
}

// BuildSummary returns a human-readable summary of the current configuration.
func BuildSummary(cfg *config.Config) []string {
	summary := []string{}
	summary = append(summary, "Configuration summary:")
	summary = append(summary, "─────────────────────")
	summary = append(summary, fmt.Sprintf("Workspace: %s", cfg.Agents.Defaults.Workspace))
	summary = append(summary, fmt.Sprintf("Restrict to workspace: %v", cfg.Agents.Defaults.RestrictToWorkspace))
	summary = append(summary, fmt.Sprintf("Provider: %s", cfg.Agents.Defaults.Provider))
	if cfg.Agents.Defaults.Model != "" {
		summary = append(summary, fmt.Sprintf("Model: %s", cfg.Agents.Defaults.Model))
	}
	return summary
}

// SetProviderCredential sets a credential value for the named provider into the
// appropriate place in the config (legacy ProvidersConfig or model_list fallback).
func SetProviderCredential(c *config.Config, provider, cred, value string) {
	if c == nil || provider == "" || cred == "" {
		return
	}
	p := strings.ToLower(provider)
	switch p {
	case "anthropic":
		if cred == "api_key" {
			c.Providers.Anthropic.APIKey = value
		} else if cred == "api_base" {
			c.Providers.Anthropic.APIBase = value
		}
	case "openai":
		if cred == "api_key" {
			c.Providers.OpenAI.APIKey = value
		} else if cred == "api_base" {
			c.Providers.OpenAI.APIBase = value
		}
	case "openrouter":
		if cred == "api_key" {
			c.Providers.OpenRouter.APIKey = value
		} else if cred == "api_base" {
			c.Providers.OpenRouter.APIBase = value
		}
	case "groq":
		if cred == "api_key" {
			c.Providers.Groq.APIKey = value
		}
	case "zhipu":
		if cred == "api_key" {
			c.Providers.Zhipu.APIKey = value
		}
	case "vllm":
		if cred == "api_key" {
			c.Providers.VLLM.APIKey = value
		}
	case "gemini":
		if cred == "api_key" {
			c.Providers.Gemini.APIKey = value
		}
	case "nvidia":
		if cred == "api_key" {
			c.Providers.Nvidia.APIKey = value
		}
	case "ollama":
		if cred == "api_key" {
			c.Providers.Ollama.APIKey = value
		}
	case "moonshot":
		if cred == "api_key" {
			c.Providers.Moonshot.APIKey = value
		}
	case "shengsuanyun":
		if cred == "api_key" {
			c.Providers.ShengSuanYun.APIKey = value
		}
	case "deepseek":
		if cred == "api_key" {
			c.Providers.DeepSeek.APIKey = value
		}
	case "cerebras":
		if cred == "api_key" {
			c.Providers.Cerebras.APIKey = value
		}
	case "volcengine":
		if cred == "api_key" {
			c.Providers.VolcEngine.APIKey = value
		}
	case "github_copilot":
		if cred == "api_key" {
			c.Providers.GitHubCopilot.APIKey = value
		}
	case "antigravity":
		if cred == "api_key" {
			c.Providers.Antigravity.APIKey = value
		}
	case "qwen":
		if cred == "api_key" {
			c.Providers.Qwen.APIKey = value
		}
	default:
		// If provider not in legacy ProvidersConfig, try to set on a matching ModelConfig
		for i := range c.ModelList {
			pfx := config.ParseProtocol(c.ModelList[i].Model)
			if pfx == "" {
				pfx = "openai"
			}
			if pfx == p {
				if cred == "api_key" {
					c.ModelList[i].APIKey = value
				} else if cred == "api_base" {
					c.ModelList[i].APIBase = value
				}
				return
			}
		}
	}
}

// GetProviderAPIKey retrieves the API key for a provider from config.
func GetProviderAPIKey(cfg *config.Config, provider string) string {
	if cfg == nil || provider == "" {
		return ""
	}
	p := strings.ToLower(provider)
	switch p {
	case "anthropic":
		return cfg.Providers.Anthropic.APIKey
	case "openai":
		return cfg.Providers.OpenAI.APIKey
	case "openrouter":
		return cfg.Providers.OpenRouter.APIKey
	case "groq":
		return cfg.Providers.Groq.APIKey
	case "zhipu":
		return cfg.Providers.Zhipu.APIKey
	case "vllm":
		return cfg.Providers.VLLM.APIKey
	case "gemini":
		return cfg.Providers.Gemini.APIKey
	case "nvidia":
		return cfg.Providers.Nvidia.APIKey
	case "ollama":
		return cfg.Providers.Ollama.APIKey
	case "moonshot":
		return cfg.Providers.Moonshot.APIKey
	case "shengsuanyun":
		return cfg.Providers.ShengSuanYun.APIKey
	case "deepseek":
		return cfg.Providers.DeepSeek.APIKey
	case "cerebras":
		return cfg.Providers.Cerebras.APIKey
	case "volcengine":
		return cfg.Providers.VolcEngine.APIKey
	case "github_copilot":
		return cfg.Providers.GitHubCopilot.APIKey
	case "antigravity":
		return cfg.Providers.Antigravity.APIKey
	case "qwen":
		return cfg.Providers.Qwen.APIKey
	default:
		for _, m := range cfg.ModelList {
			pfx := config.ParseProtocol(m.Model)
			if pfx == "" {
				pfx = "openai"
			}
			if pfx == p {
				return m.APIKey
			}
		}
	}
	return ""
}

// GetProviderAPIBase retrieves the API base URL for a provider from config.
func GetProviderAPIBase(cfg *config.Config, provider string) string {
	if cfg == nil || provider == "" {
		return ""
	}
	p := strings.ToLower(provider)
	switch p {
	case "anthropic":
		return cfg.Providers.Anthropic.APIBase
	case "openai":
		return cfg.Providers.OpenAI.APIBase
	case "openrouter":
		return cfg.Providers.OpenRouter.APIBase
	case "vllm":
		return cfg.Providers.VLLM.APIBase
	default:
		for _, m := range cfg.ModelList {
			pfx := config.ParseProtocol(m.Model)
			if pfx == "" {
				pfx = "openai"
			}
			if pfx == p {
				return m.APIBase
			}
		}
	}
	return ""
}

// GetChannelFieldToken retrieves a specific token field for a channel from config.
func GetChannelFieldToken(cfg *config.Config, questionID string) string {
	if cfg == nil || questionID == "" {
		return ""
	}
	switch questionID {
	case "telegram_token":
		return cfg.Channels.Telegram.Token
	case "slack_bot_token":
		return cfg.Channels.Slack.BotToken
	case "slack_app_token":
		return cfg.Channels.Slack.AppToken
	case "discord_token":
		return cfg.Channels.Discord.Token
	case "whatsapp_bridge_url":
		return cfg.Channels.WhatsApp.BridgeURL
	case "feishu_app_id":
		return cfg.Channels.Feishu.AppID
	case "feishu_app_secret":
		return cfg.Channels.Feishu.AppSecret
	case "dingtalk_client_id":
		return cfg.Channels.DingTalk.ClientID
	case "dingtalk_client_secret":
		return cfg.Channels.DingTalk.ClientSecret
	case "line_channel_access_token":
		return cfg.Channels.LINE.ChannelAccessToken
	case "qq_app_id":
		return cfg.Channels.QQ.AppID
	case "onebot_access_token":
		return cfg.Channels.OneBot.AccessToken
	case "wecom_token":
		return cfg.Channels.WeCom.Token
	case "wecom_encoding_aes_key":
		return cfg.Channels.WeCom.EncodingAESKey
	case "wecom_app_corp_id":
		return cfg.Channels.WeComApp.CorpID
	case "wecom_app_agent_id":
		return fmt.Sprintf("%d", cfg.Channels.WeComApp.AgentID)
	case "maixcam_device_address":
		return "http://" + net.JoinHostPort(cfg.Channels.MaixCam.Host, fmt.Sprintf("%d", cfg.Channels.MaixCam.Port))
	}
	return ""
}

// GetChannelToken retrieves the token/credential for a channel from config.
func GetChannelToken(cfg *config.Config, channel string) string {
	if cfg == nil || channel == "" {
		return ""
	}
	ch := strings.ToLower(channel)
	switch ch {
	case "telegram":
		return cfg.Channels.Telegram.Token
	case "slack":
		return cfg.Channels.Slack.BotToken
	case "discord":
		return cfg.Channels.Discord.Token
	case "whatsapp":
		return cfg.Channels.WhatsApp.BridgeURL
	case "feishu":
		return cfg.Channels.Feishu.AppID
	case "dingtalk":
		return cfg.Channels.DingTalk.ClientID
	case "line":
		return cfg.Channels.LINE.ChannelAccessToken
	case "qq":
		return cfg.Channels.QQ.AppID
	case "onebot":
		return cfg.Channels.OneBot.AccessToken
	case "wecom":
		return cfg.Channels.WeCom.Token
	case "wecom_app":
		return cfg.Channels.WeComApp.CorpID
	}
	return ""
}

// GetChannelTokenFromConfig retrieves any enabled channel's token from config.
func GetChannelTokenFromConfig(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	if cfg.Channels.Telegram.Enabled {
		return cfg.Channels.Telegram.Token
	}
	if cfg.Channels.Slack.Enabled {
		return cfg.Channels.Slack.BotToken
	}
	if cfg.Channels.Discord.Enabled {
		return cfg.Channels.Discord.Token
	}
	if cfg.Channels.WhatsApp.Enabled {
		return cfg.Channels.WhatsApp.BridgeURL
	}
	if cfg.Channels.Feishu.Enabled {
		return cfg.Channels.Feishu.AppID
	}
	if cfg.Channels.DingTalk.Enabled {
		return cfg.Channels.DingTalk.ClientID
	}
	if cfg.Channels.LINE.Enabled {
		return cfg.Channels.LINE.ChannelAccessToken
	}
	if cfg.Channels.QQ.Enabled {
		return cfg.Channels.QQ.AppID
	}
	if cfg.Channels.OneBot.Enabled {
		return cfg.Channels.OneBot.AccessToken
	}
	if cfg.Channels.WeCom.Enabled {
		return cfg.Channels.WeCom.Token
	}
	if cfg.Channels.WeComApp.Enabled {
		return cfg.Channels.WeComApp.CorpID
	}
	return ""
}
