// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

// DefaultConfig returns the default configuration for PicoClaw.
func DefaultConfig() *Config {
	return &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace:           "~/.picoclaw/workspace",
				RestrictToWorkspace: true,
				Provider:            "",
				Model:               "llama3",
				MaxTokens:           8192,
				Temperature:         nil, // nil means use provider default
				MaxToolIterations:   20,
			},
		},
		Bindings: []AgentBinding{},
		Session: SessionConfig{
			DMScope: "main",
		},
		Channels: ChannelsConfig{
			WhatsApp: WhatsAppConfig{
				Enabled:   false,
				BridgeURL: "ws://localhost:3001",
				AllowFrom: FlexibleStringSlice{},
			},
			Telegram: TelegramConfig{
				Enabled:   false,
				Token:     "",
				AllowFrom: FlexibleStringSlice{},
			},
			Feishu: FeishuConfig{
				Enabled:           false,
				AppID:             "",
				AppSecret:         "",
				EncryptKey:        "",
				VerificationToken: "",
				AllowFrom:         FlexibleStringSlice{},
			},
			Discord: DiscordConfig{
				Enabled:     false,
				Token:       "",
				AllowFrom:   FlexibleStringSlice{},
				MentionOnly: false,
			},
			MaixCam: MaixCamConfig{
				Enabled:   false,
				Host:      "0.0.0.0",
				Port:      18790,
				AllowFrom: FlexibleStringSlice{},
			},
			QQ: QQConfig{
				Enabled:   false,
				AppID:     "",
				AppSecret: "",
				AllowFrom: FlexibleStringSlice{},
			},
			DingTalk: DingTalkConfig{
				Enabled:      false,
				ClientID:     "",
				ClientSecret: "",
				AllowFrom:    FlexibleStringSlice{},
			},
			Slack: SlackConfig{
				Enabled:   false,
				BotToken:  "",
				AppToken:  "",
				AllowFrom: FlexibleStringSlice{},
			},
			LINE: LINEConfig{
				Enabled:            false,
				ChannelSecret:      "",
				ChannelAccessToken: "",
				WebhookHost:        "0.0.0.0",
				WebhookPort:        18791,
				WebhookPath:        "/webhook/line",
				AllowFrom:          FlexibleStringSlice{},
			},
			OneBot: OneBotConfig{
				Enabled:            false,
				WSUrl:              "ws://127.0.0.1:3001",
				AccessToken:        "",
				ReconnectInterval:  5,
				GroupTriggerPrefix: []string{},
				AllowFrom:          FlexibleStringSlice{},
			},
			WeCom: WeComConfig{
				Enabled:        false,
				Token:          "",
				EncodingAESKey: "",
				WebhookURL:     "",
				WebhookHost:    "0.0.0.0",
				WebhookPort:    18793,
				WebhookPath:    "/webhook/wecom",
				AllowFrom:      FlexibleStringSlice{},
				ReplyTimeout:   5,
			},
			WeComApp: WeComAppConfig{
				Enabled:        false,
				CorpID:         "",
				CorpSecret:     "",
				AgentID:        0,
				Token:          "",
				EncodingAESKey: "",
				WebhookHost:    "0.0.0.0",
				WebhookPort:    18792,
				WebhookPath:    "/webhook/wecom-app",
				AllowFrom:      FlexibleStringSlice{},
				ReplyTimeout:   5,
			},
		},
		Providers: ProvidersConfig{
			OpenAI: OpenAIProviderConfig{WebSearch: true},
		},
		ModelList: []ModelConfig{
			// ============================================
			// Add your API key to the model you want to use
			// ============================================

			// Ollama (local) - https://ollama.com
			{
				ModelName: "llama3",
				Model:     "ollama/llama3",
				APIBase:   "http://localhost:11434/v1",
				APIKey:    "ollama",
			},

			// OpenRouter (100+ models) - https://openrouter.ai/keys
			{
				ModelName: "openrouter-auto",
				Model:     "openrouter/auto",
				APIBase:   "https://openrouter.ai/api/v1",
				APIKey:    "",
			},

			// Anthropic Claude - https://console.anthropic.com/settings/keys
			{
				ModelName: "claude-sonnet-4.6",
				Model:     "anthropic/claude-sonnet-4.6",
				APIBase:   "https://api.anthropic.com/v1",
				APIKey:    "",
			},

			// OpenAI - https://platform.openai.com/api-keys
			{
				ModelName: "gpt-5.2",
				Model:     "openai/gpt-5.2",
				APIBase:   "https://api.openai.com/v1",
				APIKey:    "",
			},

			// DeepSeek - https://platform.deepseek.com/
			{
				ModelName: "deepseek-chat",
				Model:     "deepseek/deepseek-chat",
				APIBase:   "https://api.deepseek.com/v1",
				APIKey:    "",
			},
		},
		Gateway: GatewayConfig{
			Host: "0.0.0.0",
			Port: 18790,
		},
		Tools: ToolsConfig{
			Web: WebToolsConfig{
				Brave: BraveConfig{
					Enabled:    false,
					APIKey:     "",
					MaxResults: 5,
				},
				DuckDuckGo: DuckDuckGoConfig{
					Enabled:    true,
					MaxResults: 5,
				},
				Perplexity: PerplexityConfig{
					Enabled:    false,
					APIKey:     "",
					MaxResults: 5,
				},
			},
			Cron: CronToolsConfig{
				ExecTimeoutMinutes: 5,
			},
			Exec: ExecConfig{
				EnableDenyPatterns: true,
			},
			Skills: SkillsToolsConfig{
				Registries: SkillsRegistriesConfig{
					ClawHub: ClawHubRegistryConfig{
						Enabled: true,
						BaseURL: "https://clawhub.ai",
					},
				},
				MaxConcurrentSearches: 2,
				SearchCache: SearchCacheConfig{
					MaxSize:    50,
					TTLSeconds: 300,
				},
			},
		},
		Heartbeat: HeartbeatConfig{
			Enabled:  true,
			Interval: 30,
		},
		Devices: DevicesConfig{
			Enabled:    false,
			MonitorUSB: true,
		},
	}
}
