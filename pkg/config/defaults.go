// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"os"
	"path/filepath"
)

// DefaultConfig returns the default configuration for PicoClaw.
func DefaultConfig() *Config {
	// Determine the base path for the workspace.
	// Priority: $PICOCLAW_HOME > ~/.picoclaw
	var homePath string
	if picoclawHome := os.Getenv("PICOCLAW_HOME"); picoclawHome != "" {
		homePath = picoclawHome
	} else {
		userHome, _ := os.UserHomeDir()
		homePath = filepath.Join(userHome, ".picoclaw")
	}
	workspacePath := filepath.Join(homePath, "workspace")

	return &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace:                 workspacePath,
				RestrictToWorkspace:       true,
				Provider:                  "",
				Model:                     "",
				MaxTokens:                 32768,
				Temperature:               nil, // nil means use provider default
				MaxToolIterations:         50,
				SummarizeMessageThreshold: 20,
				SummarizeTokenPercent:     75,
			},
		},
		Bindings: []AgentBinding{},
		Session: SessionConfig{
			DMScope: "per-channel-peer",
		},
		Channels: ChannelsConfig{
			WhatsApp: WhatsAppConfig{
				Enabled:          false,
				BridgeURL:        "ws://localhost:3001",
				UseNative:        false,
				SessionStorePath: "",
				AllowFrom:        FlexibleStringSlice{},
			},
			Telegram: TelegramConfig{
				Enabled:   false,
				Token:     "",
				AllowFrom: FlexibleStringSlice{},
				Typing:    TypingConfig{Enabled: true},
				Placeholder: PlaceholderConfig{
					Enabled: true,
					Text:    "Thinking... 💭",
				},
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
				Enabled:          false,
				AppID:            "",
				AppSecret:        "",
				AllowFrom:        FlexibleStringSlice{},
				MaxMessageLength: 2000,
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
			Matrix: MatrixConfig{
				Enabled:      false,
				Homeserver:   "https://matrix.org",
				UserID:       "",
				AccessToken:  "",
				DeviceID:     "",
				JoinOnInvite: true,
				AllowFrom:    FlexibleStringSlice{},
				GroupTrigger: GroupTriggerConfig{
					MentionOnly: true,
				},
				Placeholder: PlaceholderConfig{
					Enabled: true,
					Text:    "Thinking... 💭",
				},
			},
			LINE: LINEConfig{
				Enabled:            false,
				ChannelSecret:      "",
				ChannelAccessToken: "",
				WebhookHost:        "0.0.0.0",
				WebhookPort:        18791,
				WebhookPath:        "/webhook/line",
				AllowFrom:          FlexibleStringSlice{},
				GroupTrigger:       GroupTriggerConfig{MentionOnly: true},
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
			WeComAIBot: WeComAIBotConfig{
				Enabled:        false,
				Token:          "",
				EncodingAESKey: "",
				WebhookPath:    "/webhook/wecom-aibot",
				AllowFrom:      FlexibleStringSlice{},
				ReplyTimeout:   5,
				MaxSteps:       10,
				WelcomeMessage: "Hello! I'm your AI assistant. How can I help you today?",
			},
			Pico: PicoConfig{
				Enabled:        false,
				Token:          "",
				PingInterval:   30,
				ReadTimeout:    60,
				WriteTimeout:   10,
				MaxConnections: 100,
				AllowFrom:      FlexibleStringSlice{},
			},
		},
		Providers: ProvidersConfig{
			OpenAI: OpenAIProviderConfig{WebSearch: true},
		},
		ModelList: []ModelConfig{
			// ============================================
			// Add your API key to the model you want to use
			// ============================================

			// Zhipu AI (智谱) - https://open.bigmodel.cn/usercenter/apikeys
			{
				ModelName: "glm-4.7",
				Model:     "zhipu/glm-4.7",
				APIBase:   "https://open.bigmodel.cn/api/paas/v4",
				APIKey:    "",
			},

			// OpenAI - https://platform.openai.com/api-keys
			{
				ModelName: "gpt-5.4",
				Model:     "openai/gpt-5.4",
				APIBase:   "https://api.openai.com/v1",
				APIKey:    "",
			},

			// Anthropic Claude - https://console.anthropic.com/settings/keys
			{
				ModelName: "claude-sonnet-4.6",
				Model:     "anthropic/claude-sonnet-4.6",
				APIBase:   "https://api.anthropic.com/v1",
				APIKey:    "",
			},

			// DeepSeek - https://platform.deepseek.com/
			{
				ModelName: "deepseek-chat",
				Model:     "deepseek/deepseek-chat",
				APIBase:   "https://api.deepseek.com/v1",
				APIKey:    "",
			},

			// Google Gemini - https://ai.google.dev/
			{
				ModelName: "gemini-2.0-flash",
				Model:     "gemini/gemini-2.0-flash-exp",
				APIBase:   "https://generativelanguage.googleapis.com/v1beta",
				APIKey:    "",
			},

			// Qwen (通义千问) - https://dashscope.console.aliyun.com/apiKey
			{
				ModelName: "qwen-plus",
				Model:     "qwen/qwen-plus",
				APIBase:   "https://dashscope.aliyuncs.com/compatible-mode/v1",
				APIKey:    "",
			},

			// Moonshot (月之暗面) - https://platform.moonshot.cn/console/api-keys
			{
				ModelName: "moonshot-v1-8k",
				Model:     "moonshot/moonshot-v1-8k",
				APIBase:   "https://api.moonshot.cn/v1",
				APIKey:    "",
			},

			// Groq - https://console.groq.com/keys
			{
				ModelName: "llama-3.3-70b",
				Model:     "groq/llama-3.3-70b-versatile",
				APIBase:   "https://api.groq.com/openai/v1",
				APIKey:    "",
			},

			// OpenRouter (100+ models) - https://openrouter.ai/keys
			{
				ModelName: "openrouter-auto",
				Model:     "openrouter/auto",
				APIBase:   "https://openrouter.ai/api/v1",
				APIKey:    "",
			},
			{
				ModelName: "openrouter-gpt-5.4",
				Model:     "openrouter/openai/gpt-5.4",
				APIBase:   "https://openrouter.ai/api/v1",
				APIKey:    "",
			},

			// NVIDIA - https://build.nvidia.com/
			{
				ModelName: "nemotron-4-340b",
				Model:     "nvidia/nemotron-4-340b-instruct",
				APIBase:   "https://integrate.api.nvidia.com/v1",
				APIKey:    "",
			},

			// Cerebras - https://inference.cerebras.ai/
			{
				ModelName: "cerebras-llama-3.3-70b",
				Model:     "cerebras/llama-3.3-70b",
				APIBase:   "https://api.cerebras.ai/v1",
				APIKey:    "",
			},

			// Vivgrid - https://vivgrid.com
			{
				ModelName: "vivgrid-auto",
				Model:     "vivgrid/auto",
				APIBase:   "https://api.vivgrid.com/v1",
				APIKey:    "",
			},

			// Volcengine (火山引擎) - https://console.volcengine.com/ark
			{
				ModelName: "ark-code-latest",
				Model:     "volcengine/ark-code-latest",
				APIBase:   "https://ark.cn-beijing.volces.com/api/v3",
				APIKey:    "",
			},
			{
				ModelName: "doubao-pro",
				Model:     "volcengine/doubao-pro-32k",
				APIBase:   "https://ark.cn-beijing.volces.com/api/v3",
				APIKey:    "",
			},

			// ShengsuanYun (神算云)
			{
				ModelName: "deepseek-v3",
				Model:     "shengsuanyun/deepseek-v3",
				APIBase:   "https://api.shengsuanyun.com/v1",
				APIKey:    "",
			},

			// Antigravity (Google Cloud Code Assist) - OAuth only
			{
				ModelName:  "gemini-flash",
				Model:      "antigravity/gemini-3-flash",
				AuthMethod: "oauth",
			},

			// GitHub Copilot - https://github.com/settings/tokens
			{
				ModelName:  "copilot-gpt-5.4",
				Model:      "github-copilot/gpt-5.4",
				APIBase:    "http://localhost:4321",
				AuthMethod: "oauth",
			},

			// Ollama (local) - https://ollama.com
			{
				ModelName: "llama3",
				Model:     "ollama/llama3",
				APIBase:   "http://localhost:11434/v1",
				APIKey:    "ollama",
			},

			// Mistral AI - https://console.mistral.ai/api-keys
			{
				ModelName: "mistral-small",
				Model:     "mistral/mistral-small-latest",
				APIBase:   "https://api.mistral.ai/v1",
				APIKey:    "",
			},

			// Avian - https://avian.io
			{
				ModelName: "deepseek-v3.2",
				Model:     "avian/deepseek/deepseek-v3.2",
				APIBase:   "https://api.avian.io/v1",
				APIKey:    "",
			},
			{
				ModelName: "kimi-k2.5",
				Model:     "avian/moonshotai/kimi-k2.5",
				APIBase:   "https://api.avian.io/v1",
				APIKey:    "",
			},

			// Minimax - https://api.minimaxi.com/
			{
				ModelName: "MiniMax-M2.5",
				Model:     "minimax/MiniMax-M2.5",
				APIBase:   "https://api.minimaxi.com/v1",
				APIKey:    "",
			},

			// LongCat - https://longcat.chat/platform
			{
				ModelName: "LongCat-Flash-Thinking",
				Model:     "longcat/LongCat-Flash-Thinking",
				APIBase:   "https://api.longcat.chat/openai",
				APIKey:    "",
			},

			// ModelScope (魔搭社区) - https://modelscope.cn/my/tokens
			{
				ModelName: "modelscope-qwen",
				Model:     "modelscope/Qwen/Qwen3-235B-A22B-Instruct-2507",
				APIBase:   "https://api-inference.modelscope.cn/v1",
				APIKey:    "",
			},

			// VLLM (local) - http://localhost:8000
			{
				ModelName: "local-model",
				Model:     "vllm/custom-model",
				APIBase:   "http://localhost:8000/v1",
				APIKey:    "",
			},

			// Azure OpenAI - https://portal.azure.com
			// model_name is a user-friendly alias; the model field's path after "azure/" is your deployment name
			{
				ModelName: "azure-gpt5",
				Model:     "azure/my-gpt5-deployment",
				APIBase:   "https://your-resource.openai.azure.com",
				APIKey:    "",
			},
		},
		Gateway: GatewayConfig{
			Host: "127.0.0.1",
			Port: 18790,
		},
		Tools: ToolsConfig{
			MediaCleanup: MediaCleanupConfig{
				ToolConfig: ToolConfig{
					Enabled: true,
				},
				MaxAge:   30,
				Interval: 5,
			},
			Web: WebToolsConfig{
				ToolConfig: ToolConfig{
					Enabled: true,
				},
				Proxy:           "",
				FetchLimitBytes: 10 * 1024 * 1024, // 10MB by default
				Brave: BraveConfig{
					Enabled:    false,
					APIKey:     "",
					APIKeys:    nil,
					MaxResults: 5,
				},
				Tavily: TavilyConfig{
					Enabled:    false,
					APIKey:     "",
					APIKeys:    nil,
					MaxResults: 5,
				},
				DuckDuckGo: DuckDuckGoConfig{
					Enabled:    true,
					MaxResults: 5,
				},
				Perplexity: PerplexityConfig{
					Enabled:    false,
					APIKey:     "",
					APIKeys:    nil,
					MaxResults: 5,
				},
				SearXNG: SearXNGConfig{
					Enabled:    false,
					BaseURL:    "",
					MaxResults: 5,
				},
				GLMSearch: GLMSearchConfig{
					Enabled:      false,
					APIKey:       "",
					BaseURL:      "https://open.bigmodel.cn/api/paas/v4/web_search",
					SearchEngine: "search_std",
					MaxResults:   5,
				},
			},
			Cron: CronToolsConfig{
				ToolConfig: ToolConfig{
					Enabled: true,
				},
				ExecTimeoutMinutes: 5,
			},
			Exec: ExecConfig{
				ToolConfig: ToolConfig{
					Enabled: true,
				},
				EnableDenyPatterns: true,
				AllowRemote:        true,
				TimeoutSeconds:     60,
			},
			Skills: SkillsToolsConfig{
				ToolConfig: ToolConfig{
					Enabled: true,
				},
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
			SendFile: ToolConfig{
				Enabled: true,
			},
			MCP: MCPConfig{
				ToolConfig: ToolConfig{
					Enabled: false,
				},
				Discovery: ToolDiscoveryConfig{
					Enabled:          false,
					TTL:              5,
					MaxSearchResults: 5,
					UseBM25:          true,
					UseRegex:         false,
				},
				Servers: map[string]MCPServerConfig{},
			},
			AppendFile: ToolConfig{
				Enabled: true,
			},
			EditFile: ToolConfig{
				Enabled: true,
			},
			FindSkills: ToolConfig{
				Enabled: true,
			},
			I2C: ToolConfig{
				Enabled: false, // Hardware tool - Linux only
			},
			InstallSkill: ToolConfig{
				Enabled: true,
			},
			ListDir: ToolConfig{
				Enabled: true,
			},
			Message: ToolConfig{
				Enabled: true,
			},
			ReadFile: ReadFileToolConfig{
				Enabled:         true,
				MaxReadFileSize: 64 * 1024, // 64KB
			},
			Spawn: ToolConfig{
				Enabled: true,
			},
			SPI: ToolConfig{
				Enabled: false, // Hardware tool - Linux only
			},
			Subagent: ToolConfig{
				Enabled: true,
			},
			WebFetch: ToolConfig{
				Enabled: true,
			},
			WriteFile: ToolConfig{
				Enabled: true,
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
		Voice: VoiceConfig{
			EchoTranscription: false,
		},
		BuildInfo: BuildInfo{
			Version:   Version,
			GitCommit: GitCommit,
			BuildTime: BuildTime,
			GoVersion: GoVersion,
		},
	}
}
