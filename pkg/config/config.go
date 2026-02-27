package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/caarlos0/env/v11"
)

// FlexibleStringSlice is a []string that also accepts JSON numbers,
// so allow_from can contain both "123" and 123.
type FlexibleStringSlice []string

func (f *FlexibleStringSlice) UnmarshalJSON(data []byte) error {
	// Try []string first
	var ss []string
	if err := json.Unmarshal(data, &ss); err == nil {
		*f = ss
		return nil
	}

	// Try []interface{} to handle mixed types
	var raw []interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	result := make([]string, 0, len(raw))
	for _, v := range raw {
		switch val := v.(type) {
		case string:
			result = append(result, val)
		case float64:
			result = append(result, fmt.Sprintf("%.0f", val))
		default:
			result = append(result, fmt.Sprintf("%v", val))
		}
	}
	*f = result
	return nil
}

type LLMConfig struct {
	Model   string `json:"model" label:"Model" env:"CLAWDROID_LLM_MODEL"`
	APIKey  string `json:"api_key" label:"API Key" env:"CLAWDROID_LLM_API_KEY"`
	BaseURL string `json:"base_url" label:"Base URL" env:"CLAWDROID_LLM_BASE_URL"`
}

type Config struct {
	LLM        LLMConfig        `json:"llm" label:"LLM"`
	Agents     AgentsConfig     `json:"agents" label:"Agent Defaults"`
	Channels   ChannelsConfig   `json:"channels" label:"Messaging Channels"`
	Gateway    GatewayConfig    `json:"gateway" label:"Gateway"`
	Tools      ToolsConfig      `json:"tools" label:"Tool Settings"`
	Heartbeat  HeartbeatConfig  `json:"heartbeat" label:"Heartbeat"`
	RateLimits RateLimitsConfig `json:"rate_limits" label:"Rate Limits"`
	mu         sync.RWMutex
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults" label:"Defaults"`
}

type AgentDefaults struct {
	Workspace           string  `json:"workspace" label:"Workspace" env:"CLAWDROID_AGENTS_DEFAULTS_WORKSPACE"`
	DataDir             string  `json:"data_dir" label:"Data Directory" env:"CLAWDROID_AGENTS_DEFAULTS_DATA_DIR"`
	RestrictToWorkspace bool    `json:"restrict_to_workspace" label:"Restrict to Workspace" env:"CLAWDROID_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE"`
	MaxTokens           int     `json:"max_tokens" label:"Max Tokens" env:"CLAWDROID_AGENTS_DEFAULTS_MAX_TOKENS"`
	ContextWindow       int     `json:"context_window" label:"Context Window" env:"CLAWDROID_AGENTS_DEFAULTS_CONTEXT_WINDOW"`
	Temperature         float64 `json:"temperature" label:"Temperature" env:"CLAWDROID_AGENTS_DEFAULTS_TEMPERATURE"`
	MaxToolIterations   int     `json:"max_tool_iterations" label:"Max Tool Iterations" env:"CLAWDROID_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS"`
}

type ChannelsConfig struct {
	WhatsApp  WhatsAppConfig  `json:"whatsapp" label:"WhatsApp"`
	Telegram  TelegramConfig  `json:"telegram" label:"Telegram"`
	Discord   DiscordConfig   `json:"discord" label:"Discord"`
	Slack     SlackConfig     `json:"slack" label:"Slack"`
	LINE      LINEConfig      `json:"line" label:"LINE"`
	WebSocket WebSocketConfig `json:"websocket" label:"WebSocket"`
}

type WhatsAppConfig struct {
	Enabled   bool                `json:"enabled" label:"Enabled" env:"CLAWDROID_CHANNELS_WHATSAPP_ENABLED"`
	BridgeURL string              `json:"bridge_url" label:"Bridge URL" env:"CLAWDROID_CHANNELS_WHATSAPP_BRIDGE_URL"`
	AllowFrom FlexibleStringSlice `json:"allow_from" label:"Allow From" env:"CLAWDROID_CHANNELS_WHATSAPP_ALLOW_FROM"`
}

type TelegramConfig struct {
	Enabled   bool                `json:"enabled" label:"Enabled" env:"CLAWDROID_CHANNELS_TELEGRAM_ENABLED"`
	Token     string              `json:"token" label:"Token" env:"CLAWDROID_CHANNELS_TELEGRAM_TOKEN"`
	Proxy     string              `json:"proxy" label:"Proxy" env:"CLAWDROID_CHANNELS_TELEGRAM_PROXY"`
	AllowFrom FlexibleStringSlice `json:"allow_from" label:"Allow From" env:"CLAWDROID_CHANNELS_TELEGRAM_ALLOW_FROM"`
}

type DiscordConfig struct {
	Enabled   bool                `json:"enabled" label:"Enabled" env:"CLAWDROID_CHANNELS_DISCORD_ENABLED"`
	Token     string              `json:"token" label:"Token" env:"CLAWDROID_CHANNELS_DISCORD_TOKEN"`
	AllowFrom FlexibleStringSlice `json:"allow_from" label:"Allow From" env:"CLAWDROID_CHANNELS_DISCORD_ALLOW_FROM"`
}

type SlackConfig struct {
	Enabled   bool                `json:"enabled" label:"Enabled" env:"CLAWDROID_CHANNELS_SLACK_ENABLED"`
	BotToken  string              `json:"bot_token" label:"Bot Token" env:"CLAWDROID_CHANNELS_SLACK_BOT_TOKEN"`
	AppToken  string              `json:"app_token" label:"App Token" env:"CLAWDROID_CHANNELS_SLACK_APP_TOKEN"`
	AllowFrom FlexibleStringSlice `json:"allow_from" label:"Allow From" env:"CLAWDROID_CHANNELS_SLACK_ALLOW_FROM"`
}

type LINEConfig struct {
	Enabled            bool                `json:"enabled" label:"Enabled" env:"CLAWDROID_CHANNELS_LINE_ENABLED"`
	ChannelSecret      string              `json:"channel_secret" label:"Channel Secret" env:"CLAWDROID_CHANNELS_LINE_CHANNEL_SECRET"`
	ChannelAccessToken string              `json:"channel_access_token" label:"Channel Access Token" env:"CLAWDROID_CHANNELS_LINE_CHANNEL_ACCESS_TOKEN"`
	WebhookHost        string              `json:"webhook_host" label:"Webhook Host" env:"CLAWDROID_CHANNELS_LINE_WEBHOOK_HOST"`
	WebhookPort        int                 `json:"webhook_port" label:"Webhook Port" env:"CLAWDROID_CHANNELS_LINE_WEBHOOK_PORT"`
	WebhookPath        string              `json:"webhook_path" label:"Webhook Path" env:"CLAWDROID_CHANNELS_LINE_WEBHOOK_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from" label:"Allow From" env:"CLAWDROID_CHANNELS_LINE_ALLOW_FROM"`
}

type WebSocketConfig struct {
	Enabled   bool                `json:"enabled" label:"Enabled" env:"CLAWDROID_CHANNELS_WEBSOCKET_ENABLED"`
	Host      string              `json:"host" label:"Host" env:"CLAWDROID_CHANNELS_WEBSOCKET_HOST"`
	Port      int                 `json:"port" label:"Port" env:"CLAWDROID_CHANNELS_WEBSOCKET_PORT"`
	Path      string              `json:"path" label:"Path" env:"CLAWDROID_CHANNELS_WEBSOCKET_PATH"`
	APIKey    string              `json:"api_key" label:"API Key" env:"CLAWDROID_CHANNELS_WEBSOCKET_API_KEY"`
	AllowFrom FlexibleStringSlice `json:"allow_from" label:"Allow From" env:"CLAWDROID_CHANNELS_WEBSOCKET_ALLOW_FROM"`
}

type HeartbeatConfig struct {
	Enabled  bool `json:"enabled" label:"Enabled" env:"CLAWDROID_HEARTBEAT_ENABLED"`
	Interval int  `json:"interval" label:"Interval" env:"CLAWDROID_HEARTBEAT_INTERVAL"` // minutes, min 5
}

type RateLimitsConfig struct {
	MaxToolCallsPerMinute int `json:"max_tool_calls_per_minute" label:"Max Tool Calls Per Minute" env:"CLAWDROID_RATE_LIMITS_MAX_TOOL_CALLS_PER_MINUTE"` // 0 = unlimited
	MaxRequestsPerMinute  int `json:"max_requests_per_minute" label:"Max Requests Per Minute" env:"CLAWDROID_RATE_LIMITS_MAX_REQUESTS_PER_MINUTE"`       // 0 = unlimited
}

type GatewayConfig struct {
	Port   int    `json:"port" label:"Port" env:"CLAWDROID_GATEWAY_PORT"`
	APIKey string `json:"api_key" label:"API Key" env:"CLAWDROID_GATEWAY_API_KEY"`
}

type BraveConfig struct {
	Enabled    bool   `json:"enabled" label:"Enabled" env:"CLAWDROID_TOOLS_WEB_BRAVE_ENABLED"`
	APIKey     string `json:"api_key" label:"API Key" env:"CLAWDROID_TOOLS_WEB_BRAVE_API_KEY"`
	MaxResults int    `json:"max_results" label:"Max Results" env:"CLAWDROID_TOOLS_WEB_BRAVE_MAX_RESULTS"`
}

type DuckDuckGoConfig struct {
	Enabled    bool `json:"enabled" label:"Enabled" env:"CLAWDROID_TOOLS_WEB_DUCKDUCKGO_ENABLED"`
	MaxResults int  `json:"max_results" label:"Max Results" env:"CLAWDROID_TOOLS_WEB_DUCKDUCKGO_MAX_RESULTS"`
}

type WebToolsConfig struct {
	Brave      BraveConfig      `json:"brave" label:"Brave Search"`
	DuckDuckGo DuckDuckGoConfig `json:"duckduckgo" label:"DuckDuckGo"`
}

type ExecToolsConfig struct {
	Enabled bool `json:"enabled" label:"Enabled" env:"CLAWDROID_TOOLS_EXEC_ENABLED"`
}

type MCPServerConfig struct {
	// Stdio transport
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	// HTTP transport
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	// Common
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	IdleTimeout int    `json:"idle_timeout,omitempty"` // seconds, default 300
}

type AndroidToolsConfig struct {
	Enabled bool `json:"enabled" label:"Enabled" env:"CLAWDROID_TOOLS_ANDROID_ENABLED"`
}

type MemoryToolsConfig struct {
	Enabled bool `json:"enabled" label:"Enabled" env:"CLAWDROID_TOOLS_MEMORY_ENABLED"`
}

type ToolsConfig struct {
	Web     WebToolsConfig             `json:"web" label:"Web Search"`
	Exec    ExecToolsConfig            `json:"exec" label:"Shell Exec"`
	Android AndroidToolsConfig         `json:"android" label:"Android"`
	Memory  MemoryToolsConfig          `json:"memory" label:"Memory"`
	MCP     map[string]MCPServerConfig `json:"mcp,omitempty" label:"MCP Servers"`
}

func DefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			Model: "",
		},
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace:           "~/.clawdroid/workspace",
				DataDir:             "~/.clawdroid/data",
				RestrictToWorkspace: true,
				MaxTokens:           8192,
				ContextWindow:       128000,
				Temperature:         0,
				MaxToolIterations:   10,
			},
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
			Discord: DiscordConfig{
				Enabled:   false,
				Token:     "",
				AllowFrom: FlexibleStringSlice{},
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
				WebhookHost:        "127.0.0.1",
				WebhookPort:        18791,
				WebhookPath:        "/webhook/line",
				AllowFrom:          FlexibleStringSlice{},
			},
			WebSocket: WebSocketConfig{
				Enabled:   true,
				Host:      "127.0.0.1",
				Port:      18793,
				Path:      "/ws",
				AllowFrom: FlexibleStringSlice{},
			},
		},
		Gateway: GatewayConfig{
			Port: 18790,
		},
		Tools: ToolsConfig{
			Exec: ExecToolsConfig{
				Enabled: false,
			},
			Android: AndroidToolsConfig{
				Enabled: true,
			},
			Memory: MemoryToolsConfig{
				Enabled: true,
			},
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
			},
		},
		Heartbeat: HeartbeatConfig{
			Enabled:  true,
			Interval: 30, // default 30 minutes
		},
		RateLimits: RateLimitsConfig{
			MaxToolCallsPerMinute: 30,
			MaxRequestsPerMinute:  15,
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	return saveConfigLocked(path, cfg)
}

// SaveConfigLocked writes cfg to path without acquiring cfg's mutex.
// Use this when the caller manages synchronization externally.
func SaveConfigLocked(path string, cfg *Config) error {
	return saveConfigLocked(path, cfg)
}

func saveConfigLocked(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func (c *Config) Lock()    { c.mu.Lock() }
func (c *Config) Unlock()  { c.mu.Unlock() }
func (c *Config) RLock()   { c.mu.RLock() }
func (c *Config) RUnlock() { c.mu.RUnlock() }

// CopyFrom copies all configuration fields from src into c.
// The caller must hold c's write lock. src's mutex is not acquired.
func (c *Config) CopyFrom(src *Config) {
	c.LLM = src.LLM
	c.Agents = src.Agents
	c.Channels = src.Channels
	c.Gateway = src.Gateway
	c.Tools = src.Tools
	c.Heartbeat = src.Heartbeat
	c.RateLimits = src.RateLimits
}

func (c *Config) WorkspacePath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return expandHome(c.Agents.Defaults.Workspace)
}

func (c *Config) DataPath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return expandHome(c.Agents.Defaults.DataDir)
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:]
		}
		return home
	}
	return path
}
