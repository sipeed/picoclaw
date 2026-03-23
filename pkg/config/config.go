package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/caarlos0/env/v11"

	"github.com/sipeed/picoclaw/pkg"
	"github.com/sipeed/picoclaw/pkg/credential"
	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// rrCounter is a global counter for round-robin load balancing across models.
var rrCounter atomic.Uint64

// FlexibleStringSlice is a []string that also accepts JSON numbers,
// so allow_from can contain both "123" and 123.
// It also supports parsing comma-separated strings from environment variables,
// including both English (,) and Chinese (，) commas.
type FlexibleStringSlice []string

func (f *FlexibleStringSlice) UnmarshalJSON(data []byte) error {
	// Try []string first
	var ss []string
	if err := json.Unmarshal(data, &ss); err == nil {
		*f = ss
		return nil
	}

	// Try []interface{} to handle mixed types
	var raw []any
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

// UnmarshalText implements encoding.TextUnmarshaler to support env variable parsing.
// It handles comma-separated values with both English (,) and Chinese (，) commas.
func (f *FlexibleStringSlice) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*f = nil
		return nil
	}

	s := string(text)
	// Replace Chinese comma with English comma, then split
	s = strings.ReplaceAll(s, "，", ",")
	parts := strings.Split(s, ",")

	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	*f = result
	return nil
}

// CurrentVersion is the latest config schema version
const CurrentVersion = 1

// Config is the current config structure with version support
type Config struct {
	Version   int             `json:"version"` // Config schema version for migration
	Agents    AgentsConfig    `json:"agents"`
	Bindings  []AgentBinding  `json:"bindings,omitempty"`
	Session   SessionConfig   `json:"session,omitempty"`
	Channels  ChannelsConfig  `json:"channels"`
	ModelList []*ModelConfig  `json:"model_list"` // New model-centric provider configuration
	Gateway   GatewayConfig   `json:"gateway"`
	Hooks     HooksConfig     `json:"hooks,omitempty"`
	Tools     ToolsConfig     `json:"tools"`
	Heartbeat HeartbeatConfig `json:"heartbeat"`
	Devices   DevicesConfig   `json:"devices"`
	Voice     VoiceConfig     `json:"voice"`
	// BuildInfo contains build-time version information
	BuildInfo BuildInfo `json:"build_info,omitempty"`

	security *SecurityConfig
}

func (c *Config) WithSecurity(sec *SecurityConfig) *Config {
	if sec == nil {
		c.security = sec
		return c
	}
	err := applySecurityConfig(c, sec)
	if err != nil {
		return nil
	}
	c.security = sec
	return c
}

type HooksConfig struct {
	Enabled   bool                         `json:"enabled"`
	Defaults  HookDefaultsConfig           `json:"defaults,omitempty"`
	Builtins  map[string]BuiltinHookConfig `json:"builtins,omitempty"`
	Processes map[string]ProcessHookConfig `json:"processes,omitempty"`
}

type HookDefaultsConfig struct {
	ObserverTimeoutMS    int `json:"observer_timeout_ms,omitempty"`
	InterceptorTimeoutMS int `json:"interceptor_timeout_ms,omitempty"`
	ApprovalTimeoutMS    int `json:"approval_timeout_ms,omitempty"`
}

type BuiltinHookConfig struct {
	Enabled  bool            `json:"enabled"`
	Priority int             `json:"priority,omitempty"`
	Config   json.RawMessage `json:"config,omitempty"`
}

type ProcessHookConfig struct {
	Enabled   bool              `json:"enabled"`
	Priority  int               `json:"priority,omitempty"`
	Transport string            `json:"transport,omitempty"`
	Command   []string          `json:"command,omitempty"`
	Dir       string            `json:"dir,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Observe   []string          `json:"observe,omitempty"`
	Intercept []string          `json:"intercept,omitempty"`
}

// BuildInfo contains build-time version information
type BuildInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

// MarshalJSON implements custom JSON marshaling for Config
// to omit providers section when empty and session when empty
func (c *Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	aux := &struct {
		Session *SessionConfig `json:"session,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	// Only include session if not empty
	if c.Session.DMScope != "" || len(c.Session.IdentityLinks) > 0 {
		aux.Session = &c.Session
	}

	return json.Marshal(aux)
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
	List     []AgentConfig `json:"list,omitempty"`
}

// AgentModelConfig supports both string and structured model config.
// String format: "gpt-4" (just primary, no fallbacks)
// Object format: {"primary": "gpt-4", "fallbacks": ["claude-haiku"]}
type AgentModelConfig struct {
	Primary   string   `json:"primary,omitempty"`
	Fallbacks []string `json:"fallbacks,omitempty"`
}

func (m *AgentModelConfig) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		m.Primary = s
		m.Fallbacks = nil
		return nil
	}
	type raw struct {
		Primary   string   `json:"primary"`
		Fallbacks []string `json:"fallbacks"`
	}
	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	m.Primary = r.Primary
	m.Fallbacks = r.Fallbacks
	return nil
}

func (m AgentModelConfig) MarshalJSON() ([]byte, error) {
	if len(m.Fallbacks) == 0 && m.Primary != "" {
		return json.Marshal(m.Primary)
	}
	type raw struct {
		Primary   string   `json:"primary,omitempty"`
		Fallbacks []string `json:"fallbacks,omitempty"`
	}
	return json.Marshal(raw{Primary: m.Primary, Fallbacks: m.Fallbacks})
}

type AgentConfig struct {
	ID        string            `json:"id"`
	Default   bool              `json:"default,omitempty"`
	Name      string            `json:"name,omitempty"`
	Workspace string            `json:"workspace,omitempty"`
	Model     *AgentModelConfig `json:"model,omitempty"`
	Skills    []string          `json:"skills,omitempty"`
	Subagents *SubagentsConfig  `json:"subagents,omitempty"`
}

type SubagentsConfig struct {
	AllowAgents []string          `json:"allow_agents,omitempty"`
	Model       *AgentModelConfig `json:"model,omitempty"`
}

type PeerMatch struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type BindingMatch struct {
	Channel   string     `json:"channel"`
	AccountID string     `json:"account_id,omitempty"`
	Peer      *PeerMatch `json:"peer,omitempty"`
	GuildID   string     `json:"guild_id,omitempty"`
	TeamID    string     `json:"team_id,omitempty"`
}

type AgentBinding struct {
	AgentID string       `json:"agent_id"`
	Match   BindingMatch `json:"match"`
}

type SessionConfig struct {
	DMScope       string              `json:"dm_scope,omitempty"`
	IdentityLinks map[string][]string `json:"identity_links,omitempty"`
}

// RoutingConfig controls the intelligent model routing feature.
// When enabled, each incoming message is scored against structural features
// (message length, code blocks, tool call history, conversation depth, attachments).
// Messages scoring below Threshold are sent to LightModel; all others use the
// agent's primary model. This reduces cost and latency for simple tasks without
// requiring any keyword matching — all scoring is language-agnostic.
type RoutingConfig struct {
	Enabled    bool    `json:"enabled"`
	LightModel string  `json:"light_model"` // model_name from model_list to use for simple tasks
	Threshold  float64 `json:"threshold"`   // complexity score in [0,1]; score >= threshold → primary model
}

// SubTurnConfig configures the SubTurn execution system.
type SubTurnConfig struct {
	MaxDepth              int `json:"max_depth"               env:"PICOCLAW_AGENTS_DEFAULTS_SUBTURN_MAX_DEPTH"`
	MaxConcurrent         int `json:"max_concurrent"          env:"PICOCLAW_AGENTS_DEFAULTS_SUBTURN_MAX_CONCURRENT"`
	DefaultTimeoutMinutes int `json:"default_timeout_minutes" env:"PICOCLAW_AGENTS_DEFAULTS_SUBTURN_DEFAULT_TIMEOUT_MINUTES"`
	DefaultTokenBudget    int `json:"default_token_budget"    env:"PICOCLAW_AGENTS_DEFAULTS_SUBTURN_DEFAULT_TOKEN_BUDGET"`
	ConcurrencyTimeoutSec int `json:"concurrency_timeout_sec" env:"PICOCLAW_AGENTS_DEFAULTS_SUBTURN_CONCURRENCY_TIMEOUT_SEC"`
}

type ToolFeedbackConfig struct {
	Enabled       bool `json:"enabled"         env:"PICOCLAW_AGENTS_DEFAULTS_TOOL_FEEDBACK_ENABLED"`
	MaxArgsLength int  `json:"max_args_length" env:"PICOCLAW_AGENTS_DEFAULTS_TOOL_FEEDBACK_MAX_ARGS_LENGTH"`
}

type AgentDefaults struct {
	Workspace                 string             `json:"workspace"                       env:"PICOCLAW_AGENTS_DEFAULTS_WORKSPACE"`
	RestrictToWorkspace       bool               `json:"restrict_to_workspace"           env:"PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE"`
	AllowReadOutsideWorkspace bool               `json:"allow_read_outside_workspace"    env:"PICOCLAW_AGENTS_DEFAULTS_ALLOW_READ_OUTSIDE_WORKSPACE"`
	Provider                  string             `json:"provider"                        env:"PICOCLAW_AGENTS_DEFAULTS_PROVIDER"`
	ModelName                 string             `json:"model_name"                      env:"PICOCLAW_AGENTS_DEFAULTS_MODEL_NAME"`
	ModelFallbacks            []string           `json:"model_fallbacks,omitempty"`
	ImageModel                string             `json:"image_model,omitempty"           env:"PICOCLAW_AGENTS_DEFAULTS_IMAGE_MODEL"`
	ImageModelFallbacks       []string           `json:"image_model_fallbacks,omitempty"`
	MaxTokens                 int                `json:"max_tokens"                      env:"PICOCLAW_AGENTS_DEFAULTS_MAX_TOKENS"`
	ContextWindow             int                `json:"context_window,omitempty"        env:"PICOCLAW_AGENTS_DEFAULTS_CONTEXT_WINDOW"`
	Temperature               *float64           `json:"temperature,omitempty"           env:"PICOCLAW_AGENTS_DEFAULTS_TEMPERATURE"`
	MaxToolIterations         int                `json:"max_tool_iterations"             env:"PICOCLAW_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS"`
	SummarizeMessageThreshold int                `json:"summarize_message_threshold"     env:"PICOCLAW_AGENTS_DEFAULTS_SUMMARIZE_MESSAGE_THRESHOLD"`
	SummarizeTokenPercent     int                `json:"summarize_token_percent"         env:"PICOCLAW_AGENTS_DEFAULTS_SUMMARIZE_TOKEN_PERCENT"`
	MaxMediaSize              int                `json:"max_media_size,omitempty"        env:"PICOCLAW_AGENTS_DEFAULTS_MAX_MEDIA_SIZE"`
	Routing                   *RoutingConfig     `json:"routing,omitempty"`
	SteeringMode              string             `json:"steering_mode,omitempty"         env:"PICOCLAW_AGENTS_DEFAULTS_STEERING_MODE"` // "one-at-a-time" (default) or "all"
	SubTurn                   SubTurnConfig      `json:"subturn"                                                                                     envPrefix:"PICOCLAW_AGENTS_DEFAULTS_SUBTURN_"`
	ToolFeedback              ToolFeedbackConfig `json:"tool_feedback,omitempty"`
}

const (
	DefaultMaxMediaSize                = 20 * 1024 * 1024 // 20 MB
	DefaultWeComAIBotProcessingMessage = "⏳ Processing, please wait. The results will be sent shortly."
)

func (d *AgentDefaults) GetMaxMediaSize() int {
	if d.MaxMediaSize > 0 {
		return d.MaxMediaSize
	}
	return DefaultMaxMediaSize
}

// GetToolFeedbackMaxArgsLength returns the max args preview length for tool feedback messages.
func (d *AgentDefaults) GetToolFeedbackMaxArgsLength() int {
	if d.ToolFeedback.MaxArgsLength > 0 {
		return d.ToolFeedback.MaxArgsLength
	}
	return 300
}

// IsToolFeedbackEnabled returns true when tool feedback messages should be sent to the chat.
func (d *AgentDefaults) IsToolFeedbackEnabled() bool {
	return d.ToolFeedback.Enabled
}

// GetModelName returns the effective model name for the agent defaults.
// It prefers the new "model_name" field but falls back to "model" for backward compatibility.
func (d *AgentDefaults) GetModelName() string {
	return d.ModelName
}

type ChannelsConfig struct {
	WhatsApp   WhatsAppConfig   `json:"whatsapp"`
	Telegram   TelegramConfig   `json:"telegram"`
	Feishu     FeishuConfig     `json:"feishu"`
	Discord    DiscordConfig    `json:"discord"`
	MaixCam    MaixCamConfig    `json:"maixcam"`
	QQ         QQConfig         `json:"qq"`
	DingTalk   DingTalkConfig   `json:"dingtalk"`
	Slack      SlackConfig      `json:"slack"`
	Matrix     MatrixConfig     `json:"matrix"`
	LINE       LINEConfig       `json:"line"`
	OneBot     OneBotConfig     `json:"onebot"`
	WeCom      WeComConfig      `json:"wecom"`
	WeComApp   WeComAppConfig   `json:"wecom_app"`
	WeComAIBot WeComAIBotConfig `json:"wecom_aibot"`
	Weixin     WeixinConfig     `json:"weixin"`
	Pico       PicoConfig       `json:"pico"`
	PicoClient PicoClientConfig `json:"pico_client"`
	IRC        IRCConfig        `json:"irc"`
}

// GroupTriggerConfig controls when the bot responds in group chats.
type GroupTriggerConfig struct {
	MentionOnly bool     `json:"mention_only,omitempty"`
	Prefixes    []string `json:"prefixes,omitempty"`
}

// TypingConfig controls typing indicator behavior (Phase 10).
type TypingConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

// PlaceholderConfig controls placeholder message behavior (Phase 10).
type PlaceholderConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Text    string `json:"text,omitempty"`
}

type StreamingConfig struct {
	Enabled         bool `json:"enabled,omitempty"          env:"PICOCLAW_CHANNELS_TELEGRAM_STREAMING_ENABLED"`
	ThrottleSeconds int  `json:"throttle_seconds,omitempty" env:"PICOCLAW_CHANNELS_TELEGRAM_STREAMING_THROTTLE_SECONDS"`
	MinGrowthChars  int  `json:"min_growth_chars,omitempty" env:"PICOCLAW_CHANNELS_TELEGRAM_STREAMING_MIN_GROWTH_CHARS"`
}

type WhatsAppConfig struct {
	Enabled            bool                `json:"enabled"              env:"PICOCLAW_CHANNELS_WHATSAPP_ENABLED"`
	BridgeURL          string              `json:"bridge_url"           env:"PICOCLAW_CHANNELS_WHATSAPP_BRIDGE_URL"`
	UseNative          bool                `json:"use_native"           env:"PICOCLAW_CHANNELS_WHATSAPP_USE_NATIVE"`
	SessionStorePath   string              `json:"session_store_path"   env:"PICOCLAW_CHANNELS_WHATSAPP_SESSION_STORE_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"           env:"PICOCLAW_CHANNELS_WHATSAPP_ALLOW_FROM"`
	ReasoningChannelID string              `json:"reasoning_channel_id" env:"PICOCLAW_CHANNELS_WHATSAPP_REASONING_CHANNEL_ID"`
}

type TelegramConfig struct {
	Enabled            bool `json:"enabled"                 env:"PICOCLAW_CHANNELS_TELEGRAM_ENABLED"`
	token              string
	BaseURL            string              `json:"base_url"                env:"PICOCLAW_CHANNELS_TELEGRAM_BASE_URL"`
	Proxy              string              `json:"proxy"                   env:"PICOCLAW_CHANNELS_TELEGRAM_PROXY"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_TELEGRAM_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	Streaming          StreamingConfig     `json:"streaming,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_TELEGRAM_REASONING_CHANNEL_ID"`
	UseMarkdownV2      bool                `json:"use_markdown_v2"         env:"PICOCLAW_CHANNELS_TELEGRAM_USE_MARKDOWN_V2"`
	secDirty           bool
}

// Token returns the Telegram bot token
func (c *TelegramConfig) Token() string {
	return c.token
}

// SetToken sets the Telegram bot token
func (c *TelegramConfig) SetToken(token string) {
	c.token = token
	c.secDirty = true
}

type FeishuConfig struct {
	Enabled             bool   `json:"enabled"                 env:"PICOCLAW_CHANNELS_FEISHU_ENABLED"`
	AppID               string `json:"app_id"                  env:"PICOCLAW_CHANNELS_FEISHU_APP_ID"`
	appSecret           string
	encryptKey          string
	verificationToken   string
	AllowFrom           FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_FEISHU_ALLOW_FROM"`
	GroupTrigger        GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Placeholder         PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID  string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_FEISHU_REASONING_CHANNEL_ID"`
	RandomReactionEmoji FlexibleStringSlice `json:"random_reaction_emoji"   env:"PICOCLAW_CHANNELS_FEISHU_RANDOM_REACTION_EMOJI"`
	IsLark              bool                `json:"is_lark"                 env:"PICOCLAW_CHANNELS_FEISHU_IS_LARK"`
	secDirty            bool
}

// AppSecret returns the Feishu app secret
func (c *FeishuConfig) AppSecret() string {
	return c.appSecret
}

// SetAppSecret sets the Feishu app secret
func (c *FeishuConfig) SetAppSecret(secret string) {
	c.appSecret = secret
	c.secDirty = true
}

// EncryptKey returns the Feishu encrypt key
func (c *FeishuConfig) EncryptKey() string {
	return c.encryptKey
}

// SetEncryptKey sets the Feishu encrypt key
func (c *FeishuConfig) SetEncryptKey(key string) {
	c.encryptKey = key
	c.secDirty = true
}

// VerificationToken returns the Feishu verification token
func (c *FeishuConfig) VerificationToken() string {
	return c.verificationToken
}

// SetVerificationToken sets the Feishu verification token
func (c *FeishuConfig) SetVerificationToken(token string) {
	c.verificationToken = token
	c.secDirty = true
}

type DiscordConfig struct {
	Enabled            bool `json:"enabled"                 env:"PICOCLAW_CHANNELS_DISCORD_ENABLED"`
	token              string
	Proxy              string              `json:"proxy"                   env:"PICOCLAW_CHANNELS_DISCORD_PROXY"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_DISCORD_ALLOW_FROM"`
	MentionOnly        bool                `json:"mention_only"            env:"PICOCLAW_CHANNELS_DISCORD_MENTION_ONLY"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_DISCORD_REASONING_CHANNEL_ID"`
	secDirty           bool
}

// Token returns the Discord bot token
func (c *DiscordConfig) Token() string {
	return c.token
}

// SetToken sets the Discord bot token
func (c *DiscordConfig) SetToken(token string) {
	c.token = token
	c.secDirty = true
}

type MaixCamConfig struct {
	Enabled            bool                `json:"enabled"              env:"PICOCLAW_CHANNELS_MAIXCAM_ENABLED"`
	Host               string              `json:"host"                 env:"PICOCLAW_CHANNELS_MAIXCAM_HOST"`
	Port               int                 `json:"port"                 env:"PICOCLAW_CHANNELS_MAIXCAM_PORT"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"           env:"PICOCLAW_CHANNELS_MAIXCAM_ALLOW_FROM"`
	ReasoningChannelID string              `json:"reasoning_channel_id" env:"PICOCLAW_CHANNELS_MAIXCAM_REASONING_CHANNEL_ID"`
}

type QQConfig struct {
	Enabled              bool   `json:"enabled"                  env:"PICOCLAW_CHANNELS_QQ_ENABLED"`
	AppID                string `json:"app_id"                   env:"PICOCLAW_CHANNELS_QQ_APP_ID"`
	appSecret            string
	AllowFrom            FlexibleStringSlice `json:"allow_from"               env:"PICOCLAW_CHANNELS_QQ_ALLOW_FROM"`
	GroupTrigger         GroupTriggerConfig  `json:"group_trigger,omitempty"`
	MaxMessageLength     int                 `json:"max_message_length"       env:"PICOCLAW_CHANNELS_QQ_MAX_MESSAGE_LENGTH"`
	MaxBase64FileSizeMiB int64               `json:"max_base64_file_size_mib" env:"PICOCLAW_CHANNELS_QQ_MAX_BASE64_FILE_SIZE_MIB"`
	SendMarkdown         bool                `json:"send_markdown"            env:"PICOCLAW_CHANNELS_QQ_SEND_MARKDOWN"`
	ReasoningChannelID   string              `json:"reasoning_channel_id"     env:"PICOCLAW_CHANNELS_QQ_REASONING_CHANNEL_ID"`
	secDirty             bool
}

// AppSecret returns the QQ app secret
func (c *QQConfig) AppSecret() string {
	return c.appSecret
}

// SetAppSecret sets the QQ app secret
func (c *QQConfig) SetAppSecret(secret string) {
	c.appSecret = secret
	c.secDirty = true
}

type DingTalkConfig struct {
	Enabled            bool   `json:"enabled"                 env:"PICOCLAW_CHANNELS_DINGTALK_ENABLED"`
	ClientID           string `json:"client_id"               env:"PICOCLAW_CHANNELS_DINGTALK_CLIENT_ID"`
	clientSecret       string
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_DINGTALK_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_DINGTALK_REASONING_CHANNEL_ID"`
	secDirty           bool
}

// ClientSecret returns the DingTalk client secret
func (c *DingTalkConfig) ClientSecret() string {
	return c.clientSecret
}

// SetClientSecret sets the DingTalk client secret
func (c *DingTalkConfig) SetClientSecret(secret string) {
	c.clientSecret = secret
	c.secDirty = true
}

type SlackConfig struct {
	Enabled            bool `json:"enabled"                 env:"PICOCLAW_CHANNELS_SLACK_ENABLED"`
	botToken           string
	appToken           string
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_SLACK_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_SLACK_REASONING_CHANNEL_ID"`
	secDirty           bool
}

// BotToken returns the Slack bot token
func (c *SlackConfig) BotToken() string {
	return c.botToken
}

// SetBotToken sets the Slack bot token
func (c *SlackConfig) SetBotToken(token string) {
	c.botToken = token
	c.secDirty = true
}

// AppToken returns the Slack app token
func (c *SlackConfig) AppToken() string {
	return c.appToken
}

// SetAppToken sets the Slack app token
func (c *SlackConfig) SetAppToken(token string) {
	c.appToken = token
	c.secDirty = true
}

type MatrixConfig struct {
	Enabled            bool   `json:"enabled"                  env:"PICOCLAW_CHANNELS_MATRIX_ENABLED"`
	Homeserver         string `json:"homeserver"               env:"PICOCLAW_CHANNELS_MATRIX_HOMESERVER"`
	UserID             string `json:"user_id"                  env:"PICOCLAW_CHANNELS_MATRIX_USER_ID"`
	accessToken        string
	DeviceID           string              `json:"device_id,omitempty"      env:"PICOCLAW_CHANNELS_MATRIX_DEVICE_ID"`
	JoinOnInvite       bool                `json:"join_on_invite"           env:"PICOCLAW_CHANNELS_MATRIX_JOIN_ON_INVITE"`
	MessageFormat      string              `json:"message_format,omitempty" env:"PICOCLAW_CHANNELS_MATRIX_MESSAGE_FORMAT"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"               env:"PICOCLAW_CHANNELS_MATRIX_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"     env:"PICOCLAW_CHANNELS_MATRIX_REASONING_CHANNEL_ID"`
	secDirty           bool
}

// AccessToken returns the Matrix access token
func (c *MatrixConfig) AccessToken() string {
	return c.accessToken
}

// SetAccessToken sets the Matrix access token
func (c *MatrixConfig) SetAccessToken(token string) {
	c.accessToken = token
	c.secDirty = true
}

type LINEConfig struct {
	Enabled            bool `json:"enabled"                 env:"PICOCLAW_CHANNELS_LINE_ENABLED"`
	channelSecret      string
	channelAccessToken string
	WebhookHost        string              `json:"webhook_host"            env:"PICOCLAW_CHANNELS_LINE_WEBHOOK_HOST"`
	WebhookPort        int                 `json:"webhook_port"            env:"PICOCLAW_CHANNELS_LINE_WEBHOOK_PORT"`
	WebhookPath        string              `json:"webhook_path"            env:"PICOCLAW_CHANNELS_LINE_WEBHOOK_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_LINE_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_LINE_REASONING_CHANNEL_ID"`
	secDirty           bool
}

// ChannelSecret returns the LINE channel secret
func (c *LINEConfig) ChannelSecret() string {
	return c.channelSecret
}

// SetChannelSecret sets the LINE channel secret
func (c *LINEConfig) SetChannelSecret(secret string) {
	c.channelSecret = secret
	c.secDirty = true
}

// ChannelAccessToken returns the LINE channel access token
func (c *LINEConfig) ChannelAccessToken() string {
	return c.channelAccessToken
}

// SetChannelAccessToken sets the LINE channel access token
func (c *LINEConfig) SetChannelAccessToken(token string) {
	c.channelAccessToken = token
	c.secDirty = true
}

type OneBotConfig struct {
	Enabled            bool   `json:"enabled"                 env:"PICOCLAW_CHANNELS_ONEBOT_ENABLED"`
	WSUrl              string `json:"ws_url"                  env:"PICOCLAW_CHANNELS_ONEBOT_WS_URL"`
	accessToken        string
	ReconnectInterval  int                 `json:"reconnect_interval"      env:"PICOCLAW_CHANNELS_ONEBOT_RECONNECT_INTERVAL"`
	GroupTriggerPrefix []string            `json:"group_trigger_prefix"    env:"PICOCLAW_CHANNELS_ONEBOT_GROUP_TRIGGER_PREFIX"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_ONEBOT_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_ONEBOT_REASONING_CHANNEL_ID"`
	secDirty           bool
}

// AccessToken returns the OneBot access token
func (c *OneBotConfig) AccessToken() string {
	return c.accessToken
}

// SetAccessToken sets the OneBot access token
func (c *OneBotConfig) SetAccessToken(token string) {
	c.accessToken = token
	c.secDirty = true
}

type WeComConfig struct {
	Enabled            bool `json:"enabled"                 env:"PICOCLAW_CHANNELS_WECOM_ENABLED"`
	token              string
	encodingAESKey     string
	WebhookURL         string              `json:"webhook_url"             env:"PICOCLAW_CHANNELS_WECOM_WEBHOOK_URL"`
	WebhookHost        string              `json:"webhook_host"            env:"PICOCLAW_CHANNELS_WECOM_WEBHOOK_HOST"`
	WebhookPort        int                 `json:"webhook_port"            env:"PICOCLAW_CHANNELS_WECOM_WEBHOOK_PORT"`
	WebhookPath        string              `json:"webhook_path"            env:"PICOCLAW_CHANNELS_WECOM_WEBHOOK_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_WECOM_ALLOW_FROM"`
	ReplyTimeout       int                 `json:"reply_timeout"           env:"PICOCLAW_CHANNELS_WECOM_REPLY_TIMEOUT"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_WECOM_REASONING_CHANNEL_ID"`
	secDirty           bool
}

// Token returns the WeCom token
func (c *WeComConfig) Token() string {
	return c.token
}

// SetToken sets the WeCom token
func (c *WeComConfig) SetToken(token string) {
	c.token = token
	c.secDirty = true
}

// EncodingAESKey returns the WeCom encoding AES key
func (c *WeComConfig) EncodingAESKey() string {
	return c.encodingAESKey
}

// SetEncodingAESKey sets the WeCom encoding AES key
func (c *WeComConfig) SetEncodingAESKey(key string) {
	c.encodingAESKey = key
	c.secDirty = true
}

type WeComAppConfig struct {
	Enabled            bool   `json:"enabled"                 env:"PICOCLAW_CHANNELS_WECOM_APP_ENABLED"`
	CorpID             string `json:"corp_id"                 env:"PICOCLAW_CHANNELS_WECOM_APP_CORP_ID"`
	corpSecret         string
	AgentID            int64 `json:"agent_id"                env:"PICOCLAW_CHANNELS_WECOM_APP_AGENT_ID"`
	token              string
	encodingAESKey     string
	WebhookHost        string              `json:"webhook_host"            env:"PICOCLAW_CHANNELS_WECOM_APP_WEBHOOK_HOST"`
	WebhookPort        int                 `json:"webhook_port"            env:"PICOCLAW_CHANNELS_WECOM_APP_WEBHOOK_PORT"`
	WebhookPath        string              `json:"webhook_path"            env:"PICOCLAW_CHANNELS_WECOM_APP_WEBHOOK_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_WECOM_APP_ALLOW_FROM"`
	ReplyTimeout       int                 `json:"reply_timeout"           env:"PICOCLAW_CHANNELS_WECOM_APP_REPLY_TIMEOUT"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_WECOM_APP_REASONING_CHANNEL_ID"`
	secDirty           bool
}

// CorpSecret returns the corporate secret for WeCom app
func (c *WeComAppConfig) CorpSecret() string {
	return c.corpSecret
}

// SetCorpSecret sets the corporate secret for WeCom app
func (c *WeComAppConfig) SetCorpSecret(secret string) {
	c.corpSecret = secret
	c.secDirty = true
}

// Token returns the webhook token for WeCom app
func (c *WeComAppConfig) Token() string {
	return c.token
}

// SetToken sets the webhook token for WeCom app
func (c *WeComAppConfig) SetToken(token string) {
	c.token = token
	c.secDirty = true
}

// EncodingAESKey returns the encoding AES key for WeCom app
func (c *WeComAppConfig) EncodingAESKey() string {
	return c.encodingAESKey
}

// SetEncodingAESKey sets the encoding AES key for WeCom app
func (c *WeComAppConfig) SetEncodingAESKey(key string) {
	c.encodingAESKey = key
	c.secDirty = true
}

type WeComAIBotConfig struct {
	Enabled            bool   `json:"enabled"                      env:"PICOCLAW_CHANNELS_WECOM_AIBOT_ENABLED"`
	BotID              string `json:"bot_id,omitempty"             env:"PICOCLAW_CHANNELS_WECOM_AIBOT_BOT_ID"`
	secret             string
	token              string
	encodingAESKey     string
	WebhookPath        string              `json:"webhook_path,omitempty"       env:"PICOCLAW_CHANNELS_WECOM_AIBOT_WEBHOOK_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"                   env:"PICOCLAW_CHANNELS_WECOM_AIBOT_ALLOW_FROM"`
	ReplyTimeout       int                 `json:"reply_timeout"                env:"PICOCLAW_CHANNELS_WECOM_AIBOT_REPLY_TIMEOUT"`
	MaxSteps           int                 `json:"max_steps"                    env:"PICOCLAW_CHANNELS_WECOM_AIBOT_MAX_STEPS"`       // Maximum streaming steps
	WelcomeMessage     string              `json:"welcome_message"              env:"PICOCLAW_CHANNELS_WECOM_AIBOT_WELCOME_MESSAGE"` // Sent on enter_chat event; empty = no welcome
	ProcessingMessage  string              `json:"processing_message,omitempty" env:"PICOCLAW_CHANNELS_WECOM_AIBOT_PROCESSING_MESSAGE"`
	ReasoningChannelID string              `json:"reasoning_channel_id"         env:"PICOCLAW_CHANNELS_WECOM_AIBOT_REASONING_CHANNEL_ID"`
	secDirty           bool
}

// Token returns the webhook token for WeCom AI bot
func (c *WeComAIBotConfig) Token() string {
	return c.token
}

// EncodingAESKey returns the encoding AES key for WeCom AI bot
func (c *WeComAIBotConfig) EncodingAESKey() string {
	return c.encodingAESKey
}

// SetToken sets the token for WeCom AI bot
func (c *WeComAIBotConfig) SetToken(token string) {
	c.token = token
	c.secDirty = true
}

// SetEncodingAESKey sets the encoding AES key for WeCom AI bot
func (c *WeComAIBotConfig) SetEncodingAESKey(key string) {
	c.encodingAESKey = key
	c.secDirty = true
}

func (c *WeComAIBotConfig) Secret() string {
	return c.secret
}

func (c *WeComAIBotConfig) SetSecret(secret string) {
	c.secret = secret
	c.secDirty = true
}

type WeixinConfig struct {
	Enabled            bool `json:"enabled"              env:"PICOCLAW_CHANNELS_WEIXIN_ENABLED"`
	token              string
	BaseURL            string              `json:"base_url"             env:"PICOCLAW_CHANNELS_WEIXIN_BASE_URL"`
	CDNBaseURL         string              `json:"cdn_base_url"         env:"PICOCLAW_CHANNELS_WEIXIN_CDN_BASE_URL"`
	Proxy              string              `json:"proxy"                env:"PICOCLAW_CHANNELS_WEIXIN_PROXY"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"           env:"PICOCLAW_CHANNELS_WEIXIN_ALLOW_FROM"`
	ReasoningChannelID string              `json:"reasoning_channel_id" env:"PICOCLAW_CHANNELS_WEIXIN_REASONING_CHANNEL_ID"`
	secDirty           bool
}

func (c *WeixinConfig) Token() string {
	return c.token
}

func (c *WeixinConfig) SetToken(token string) *WeixinConfig {
	c.token = token
	c.secDirty = true
	return c
}

type PicoConfig struct {
	Enabled         bool `json:"enabled"                     env:"PICOCLAW_CHANNELS_PICO_ENABLED"`
	token           string
	AllowTokenQuery bool                `json:"allow_token_query,omitempty"`
	AllowOrigins    []string            `json:"allow_origins,omitempty"`
	PingInterval    int                 `json:"ping_interval,omitempty"`
	ReadTimeout     int                 `json:"read_timeout,omitempty"`
	WriteTimeout    int                 `json:"write_timeout,omitempty"`
	MaxConnections  int                 `json:"max_connections,omitempty"`
	AllowFrom       FlexibleStringSlice `json:"allow_from"                  env:"PICOCLAW_CHANNELS_PICO_ALLOW_FROM"`
	Placeholder     PlaceholderConfig   `json:"placeholder,omitempty"`
	secDirty        bool
}

// Token returns the Pico channel token
func (c *PicoConfig) Token() string {
	return c.token
}

// SetToken sets the Pico channel token
func (c *PicoConfig) SetToken(token string) {
	c.token = token
	c.secDirty = true
}

type PicoClientConfig struct {
	Enabled      bool                `json:"enabled"                 env:"PICOCLAW_CHANNELS_PICO_CLIENT_ENABLED"`
	URL          string              `json:"url"                     env:"PICOCLAW_CHANNELS_PICO_CLIENT_URL"`
	Token        string              `json:"token"                   env:"PICOCLAW_CHANNELS_PICO_CLIENT_TOKEN"`
	SessionID    string              `json:"session_id,omitempty"`
	PingInterval int                 `json:"ping_interval,omitempty"`
	ReadTimeout  int                 `json:"read_timeout,omitempty"`
	AllowFrom    FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_PICO_CLIENT_ALLOW_FROM"`
}

type IRCConfig struct {
	Enabled            bool   `json:"enabled"                 env:"PICOCLAW_CHANNELS_IRC_ENABLED"`
	Server             string `json:"server"                  env:"PICOCLAW_CHANNELS_IRC_SERVER"`
	TLS                bool   `json:"tls"                     env:"PICOCLAW_CHANNELS_IRC_TLS"`
	Nick               string `json:"nick"                    env:"PICOCLAW_CHANNELS_IRC_NICK"`
	User               string `json:"user,omitempty"          env:"PICOCLAW_CHANNELS_IRC_USER"`
	RealName           string `json:"real_name,omitempty"     env:"PICOCLAW_CHANNELS_IRC_REAL_NAME"`
	password           string
	nickServPassword   string
	SASLUser           string `json:"sasl_user"               env:"PICOCLAW_CHANNELS_IRC_SASL_USER"`
	saslPassword       string
	Channels           FlexibleStringSlice `json:"channels"                env:"PICOCLAW_CHANNELS_IRC_CHANNELS"`
	RequestCaps        FlexibleStringSlice `json:"request_caps,omitempty"  env:"PICOCLAW_CHANNELS_IRC_REQUEST_CAPS"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_IRC_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_IRC_REASONING_CHANNEL_ID"`
	secDirty           bool
}

// Password returns the IRC password
func (c *IRCConfig) Password() string {
	return c.password
}

// NickServPassword returns the NickServ password
func (c *IRCConfig) NickServPassword() string {
	return c.nickServPassword
}

// SASLPassword returns the SASL password
func (c *IRCConfig) SASLPassword() string {
	return c.saslPassword
}

func (c *IRCConfig) SetPassword(password string) {
	c.password = password
	c.secDirty = true
}

func (c *IRCConfig) SetNickServPassword(password string) {
	c.nickServPassword = password
	c.secDirty = true
}

func (c *IRCConfig) SetSASLPassword(password string) {
	c.saslPassword = password
	c.secDirty = true
}

type HeartbeatConfig struct {
	Enabled  bool `json:"enabled"  env:"PICOCLAW_HEARTBEAT_ENABLED"`
	Interval int  `json:"interval" env:"PICOCLAW_HEARTBEAT_INTERVAL"` // minutes, min 5
}

type DevicesConfig struct {
	Enabled    bool `json:"enabled"     env:"PICOCLAW_DEVICES_ENABLED"`
	MonitorUSB bool `json:"monitor_usb" env:"PICOCLAW_DEVICES_MONITOR_USB"`
}

type VoiceConfig struct {
	ModelName         string `json:"model_name,omitempty" env:"PICOCLAW_VOICE_MODEL_NAME"`
	EchoTranscription bool   `json:"echo_transcription"   env:"PICOCLAW_VOICE_ECHO_TRANSCRIPTION"`
}

// ModelConfig represents a model-centric provider configuration.
// It allows adding new providers (especially OpenAI-compatible ones) via configuration only.
// The model field uses protocol prefix format: [protocol/]model-identifier
// Supported protocols include openai, anthropic, antigravity, claude-cli,
// codex-cli, github-copilot, and named OpenAI-compatible protocols such as
// groq, deepseek, modelscope, and novita.
// Default protocol is "openai" if no prefix is specified.
type ModelConfig struct {
	// Required fields
	ModelName string `json:"model_name"` // User-facing alias for the model
	Model     string `json:"model"`      // Protocol/model-identifier (e.g., "openai/gpt-4o", "anthropic/claude-sonnet-4.6")

	// HTTP-based providers
	APIBase   string   `json:"api_base,omitempty"`  // API endpoint URL
	Proxy     string   `json:"proxy,omitempty"`     // HTTP proxy URL
	Fallbacks []string `json:"fallbacks,omitempty"` // Fallback model names for failover

	// Special providers (CLI-based, OAuth, etc.)
	AuthMethod  string `json:"auth_method,omitempty"`  // Authentication method: oauth, token
	ConnectMode string `json:"connect_mode,omitempty"` // Connection mode: stdio, grpc
	Workspace   string `json:"workspace,omitempty"`    // Workspace path for CLI-based providers

	// Optional optimizations
	RPM            int            `json:"rpm,omitempty"`              // Requests per minute limit
	MaxTokensField string         `json:"max_tokens_field,omitempty"` // Field name for max tokens (e.g., "max_completion_tokens")
	RequestTimeout int            `json:"request_timeout,omitempty"`
	ThinkingLevel  string         `json:"thinking_level,omitempty"` // Extended thinking: off|low|medium|high|xhigh|adaptive
	ExtraBody      map[string]any `json:"extra_body,omitempty"`     // Additional fields to inject into request body

	// from security
	secModelName string
	apiKeys      []string
	secDirty     bool
}

// APIKey returns the first API key from apiKeys
func (c *ModelConfig) APIKey() string {
	if len(c.apiKeys) > 0 {
		return c.apiKeys[0]
	}
	return ""
}

// Validate checks if the ModelConfig has all required fields.
func (c *ModelConfig) Validate() error {
	if c.ModelName == "" {
		return fmt.Errorf("model_name is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

func (c *ModelConfig) SetAPIKey(value string) {
	if len(c.apiKeys) > 0 {
		c.apiKeys[0] = value
	} else {
		c.apiKeys = append(c.apiKeys, value)
	}
	c.secDirty = true
}

type GatewayConfig struct {
	Host      string `json:"host"                env:"PICOCLAW_GATEWAY_HOST"`
	Port      int    `json:"port"                env:"PICOCLAW_GATEWAY_PORT"`
	HotReload bool   `json:"hot_reload"          env:"PICOCLAW_GATEWAY_HOT_RELOAD"`
	LogLevel  string `json:"log_level,omitempty" env:"PICOCLAW_LOG_LEVEL"`
}

type ToolDiscoveryConfig struct {
	Enabled          bool `json:"enabled"            env:"PICOCLAW_TOOLS_DISCOVERY_ENABLED"`
	TTL              int  `json:"ttl"                env:"PICOCLAW_TOOLS_DISCOVERY_TTL"`
	MaxSearchResults int  `json:"max_search_results" env:"PICOCLAW_MAX_SEARCH_RESULTS"`
	UseBM25          bool `json:"use_bm25"           env:"PICOCLAW_TOOLS_DISCOVERY_USE_BM25"`
	UseRegex         bool `json:"use_regex"          env:"PICOCLAW_TOOLS_DISCOVERY_USE_REGEX"`
}

type ToolConfig struct {
	Enabled bool `json:"enabled" env:"ENABLED"`
}

type BraveConfig struct {
	Enabled    bool `json:"enabled"     env:"PICOCLAW_TOOLS_WEB_BRAVE_ENABLED"`
	apiKeys    []string
	secDirty   bool
	MaxResults int `json:"max_results" env:"PICOCLAW_TOOLS_WEB_BRAVE_MAX_RESULTS"`
}

// APIKey returns the Brave API key
func (c *BraveConfig) APIKey() string {
	if len(c.apiKeys) == 0 {
		return ""
	}
	return c.apiKeys[0]
}

// APIKeys returns the Brave API keys
func (c *BraveConfig) APIKeys() []string {
	return c.apiKeys
}

// SetAPIKey sets the Brave API key
func (c *BraveConfig) SetAPIKey(key string) {
	c.apiKeys = []string{key}
	c.secDirty = true
}

// SetAPIKeys sets the Brave API keys
func (c *BraveConfig) SetAPIKeys(keys []string) {
	c.apiKeys = keys
	c.secDirty = true
}

type TavilyConfig struct {
	Enabled    bool `json:"enabled"     env:"PICOCLAW_TOOLS_WEB_TAVILY_ENABLED"`
	apiKeys    []string
	secDirty   bool
	BaseURL    string `json:"base_url"    env:"PICOCLAW_TOOLS_WEB_TAVILY_BASE_URL"`
	MaxResults int    `json:"max_results" env:"PICOCLAW_TOOLS_WEB_TAVILY_MAX_RESULTS"`
}

// APIKey returns the Tavily API key
func (c *TavilyConfig) APIKey() string {
	if len(c.apiKeys) == 0 {
		return ""
	}
	return c.apiKeys[0]
}

// APIKeys returns the Tavily API keys
func (c *TavilyConfig) APIKeys() []string {
	return c.apiKeys
}

// SetAPIKey sets the Tavily API key
func (c *TavilyConfig) SetAPIKey(key string) {
	c.apiKeys = []string{key}
	c.secDirty = true
}

// SetAPIKeys sets the Tavily API keys
func (c *TavilyConfig) SetAPIKeys(keys []string) {
	c.apiKeys = keys
	c.secDirty = true
}

type DuckDuckGoConfig struct {
	Enabled    bool `json:"enabled"     env:"PICOCLAW_TOOLS_WEB_DUCKDUCKGO_ENABLED"`
	MaxResults int  `json:"max_results" env:"PICOCLAW_TOOLS_WEB_DUCKDUCKGO_MAX_RESULTS"`
}

type PerplexityConfig struct {
	Enabled    bool `json:"enabled"     env:"PICOCLAW_TOOLS_WEB_PERPLEXITY_ENABLED"`
	apiKeys    []string
	secDirty   bool
	MaxResults int `json:"max_results" env:"PICOCLAW_TOOLS_WEB_PERPLEXITY_MAX_RESULTS"`
}

// APIKey returns the Perplexity API key
func (c *PerplexityConfig) APIKey() string {
	if len(c.apiKeys) == 0 {
		return ""
	}
	return c.apiKeys[0]
}

// SetAPIKey sets the Perplexity API key
func (c *PerplexityConfig) SetAPIKey(key string) {
	c.apiKeys = []string{key}
	c.secDirty = true
}

// APIKeys returns the Perplexity API keys
func (c *PerplexityConfig) APIKeys() []string {
	return c.apiKeys
}

// SetAPIKeys sets the Perplexity API keys
func (c *PerplexityConfig) SetAPIKeys(keys []string) {
	c.apiKeys = keys
	c.secDirty = true
}

type SearXNGConfig struct {
	Enabled    bool   `json:"enabled"     env:"PICOCLAW_TOOLS_WEB_SEARXNG_ENABLED"`
	BaseURL    string `json:"base_url"    env:"PICOCLAW_TOOLS_WEB_SEARXNG_BASE_URL"`
	MaxResults int    `json:"max_results" env:"PICOCLAW_TOOLS_WEB_SEARXNG_MAX_RESULTS"`
}

type GLMSearchConfig struct {
	Enabled  bool `json:"enabled"  env:"PICOCLAW_TOOLS_WEB_GLM_ENABLED"`
	apiKey   string
	secDirty bool
	BaseURL  string `json:"base_url" env:"PICOCLAW_TOOLS_WEB_GLM_BASE_URL"`
	// SearchEngine specifies the search backend: "search_std" (default),
	// "search_pro", "search_pro_sogou", or "search_pro_quark".
	SearchEngine string `json:"search_engine" env:"PICOCLAW_TOOLS_WEB_GLM_SEARCH_ENGINE"`
	MaxResults   int    `json:"max_results"   env:"PICOCLAW_TOOLS_WEB_GLM_MAX_RESULTS"`
}

// APIKey returns the GLM search API key
func (c *GLMSearchConfig) APIKey() string {
	return c.apiKey
}

// SetAPIKey sets the GLM search API key (internal use only)
func (c *GLMSearchConfig) SetAPIKey(key string) {
	c.apiKey = key
	c.secDirty = true
}

type BaiduSearchConfig struct {
	Enabled    bool   `json:"enabled"     env:"PICOCLAW_TOOLS_WEB_BAIDU_ENABLED"`
	BaseURL    string `json:"base_url"    env:"PICOCLAW_TOOLS_WEB_BAIDU_BASE_URL"`
	MaxResults int    `json:"max_results" env:"PICOCLAW_TOOLS_WEB_BAIDU_MAX_RESULTS"`
	apiKey     string
	secDirty   bool
}

// APIKey returns the Baidu search API key
func (c *BaiduSearchConfig) APIKey() string {
	return c.apiKey
}

func (c *BaiduSearchConfig) SetAPIKey(key string) {
	c.apiKey = key
	c.secDirty = true
}

type WebToolsConfig struct {
	ToolConfig  `                  envPrefix:"PICOCLAW_TOOLS_WEB_"`
	Brave       BraveConfig       `                                json:"brave"`
	Tavily      TavilyConfig      `                                json:"tavily"`
	DuckDuckGo  DuckDuckGoConfig  `                                json:"duckduckgo"`
	Perplexity  PerplexityConfig  `                                json:"perplexity"`
	SearXNG     SearXNGConfig     `                                json:"searxng"`
	GLMSearch   GLMSearchConfig   `                                json:"glm_search"`
	BaiduSearch BaiduSearchConfig `                                json:"baidu_search"`
	// PreferNative controls whether to use provider-native web search when
	// the active LLM supports it (e.g. OpenAI web_search_preview). When true,
	// the client-side web_search tool is hidden to avoid duplicate search surfaces,
	// and the provider's built-in search is used instead. Falls back to client-side
	// search when the provider does not support native search.
	PreferNative bool `json:"prefer_native" env:"PICOCLAW_TOOLS_WEB_PREFER_NATIVE"`
	// Proxy is an optional proxy URL for web tools (http/https/socks5/socks5h).
	// For authenticated proxies, prefer HTTP_PROXY/HTTPS_PROXY env vars instead of embedding credentials in config.
	Proxy                string              `json:"proxy,omitempty"                  env:"PICOCLAW_TOOLS_WEB_PROXY"`
	FetchLimitBytes      int64               `json:"fetch_limit_bytes,omitempty"      env:"PICOCLAW_TOOLS_WEB_FETCH_LIMIT_BYTES"`
	Format               string              `json:"format,omitempty"                 env:"PICOCLAW_TOOLS_WEB_FORMAT"`
	PrivateHostWhitelist FlexibleStringSlice `json:"private_host_whitelist,omitempty" env:"PICOCLAW_TOOLS_WEB_PRIVATE_HOST_WHITELIST"`
}

type CronToolsConfig struct {
	ToolConfig         `     envPrefix:"PICOCLAW_TOOLS_CRON_"`
	ExecTimeoutMinutes int  `                                 env:"PICOCLAW_TOOLS_CRON_EXEC_TIMEOUT_MINUTES" json:"exec_timeout_minutes"` // 0 means no timeout
	AllowCommand       bool `                                 env:"PICOCLAW_TOOLS_CRON_ALLOW_COMMAND"        json:"allow_command"`
}

type ExecConfig struct {
	ToolConfig          `         envPrefix:"PICOCLAW_TOOLS_EXEC_"`
	EnableDenyPatterns  bool     `                                 env:"PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS"  json:"enable_deny_patterns"`
	AllowRemote         bool     `                                 env:"PICOCLAW_TOOLS_EXEC_ALLOW_REMOTE"          json:"allow_remote"`
	CustomDenyPatterns  []string `                                 env:"PICOCLAW_TOOLS_EXEC_CUSTOM_DENY_PATTERNS"  json:"custom_deny_patterns"`
	CustomAllowPatterns []string `                                 env:"PICOCLAW_TOOLS_EXEC_CUSTOM_ALLOW_PATTERNS" json:"custom_allow_patterns"`
	TimeoutSeconds      int      `                                 env:"PICOCLAW_TOOLS_EXEC_TIMEOUT_SECONDS"       json:"timeout_seconds"` // 0 means use default (60s)
}

type SkillsToolsConfig struct {
	ToolConfig            `                       envPrefix:"PICOCLAW_TOOLS_SKILLS_"`
	Registries            SkillsRegistriesConfig `                                   json:"registries"`
	Github                SkillsGithubConfig     `                                   json:"github"`
	MaxConcurrentSearches int                    `                                   json:"max_concurrent_searches" env:"PICOCLAW_TOOLS_SKILLS_MAX_CONCURRENT_SEARCHES"`
	SearchCache           SearchCacheConfig      `                                   json:"search_cache"`
}

type MediaCleanupConfig struct {
	ToolConfig `    envPrefix:"PICOCLAW_MEDIA_CLEANUP_"`
	MaxAge     int `                                    env:"PICOCLAW_MEDIA_CLEANUP_MAX_AGE"  json:"max_age_minutes"`
	Interval   int `                                    env:"PICOCLAW_MEDIA_CLEANUP_INTERVAL" json:"interval_minutes"`
}

type ReadFileToolConfig struct {
	Enabled         bool `json:"enabled"`
	MaxReadFileSize int  `json:"max_read_file_size"`
}

type ToolsConfig struct {
	AllowReadPaths  []string           `json:"allow_read_paths"  env:"PICOCLAW_TOOLS_ALLOW_READ_PATHS"`
	AllowWritePaths []string           `json:"allow_write_paths" env:"PICOCLAW_TOOLS_ALLOW_WRITE_PATHS"`
	Web             WebToolsConfig     `json:"web"`
	Cron            CronToolsConfig    `json:"cron"`
	Exec            ExecConfig         `json:"exec"`
	Skills          SkillsToolsConfig  `json:"skills"`
	MediaCleanup    MediaCleanupConfig `json:"media_cleanup"`
	MCP             MCPConfig          `json:"mcp"`
	AppendFile      ToolConfig         `json:"append_file"                                              envPrefix:"PICOCLAW_TOOLS_APPEND_FILE_"`
	EditFile        ToolConfig         `json:"edit_file"                                                envPrefix:"PICOCLAW_TOOLS_EDIT_FILE_"`
	FindSkills      ToolConfig         `json:"find_skills"                                              envPrefix:"PICOCLAW_TOOLS_FIND_SKILLS_"`
	I2C             ToolConfig         `json:"i2c"                                                      envPrefix:"PICOCLAW_TOOLS_I2C_"`
	InstallSkill    ToolConfig         `json:"install_skill"                                            envPrefix:"PICOCLAW_TOOLS_INSTALL_SKILL_"`
	ListDir         ToolConfig         `json:"list_dir"                                                 envPrefix:"PICOCLAW_TOOLS_LIST_DIR_"`
	Message         ToolConfig         `json:"message"                                                  envPrefix:"PICOCLAW_TOOLS_MESSAGE_"`
	ReadFile        ReadFileToolConfig `json:"read_file"                                                envPrefix:"PICOCLAW_TOOLS_READ_FILE_"`
	SendFile        ToolConfig         `json:"send_file"                                                envPrefix:"PICOCLAW_TOOLS_SEND_FILE_"`
	Spawn           ToolConfig         `json:"spawn"                                                    envPrefix:"PICOCLAW_TOOLS_SPAWN_"`
	SpawnStatus     ToolConfig         `json:"spawn_status"                                             envPrefix:"PICOCLAW_TOOLS_SPAWN_STATUS_"`
	SPI             ToolConfig         `json:"spi"                                                      envPrefix:"PICOCLAW_TOOLS_SPI_"`
	Subagent        ToolConfig         `json:"subagent"                                                 envPrefix:"PICOCLAW_TOOLS_SUBAGENT_"`
	WebFetch        ToolConfig         `json:"web_fetch"                                                envPrefix:"PICOCLAW_TOOLS_WEB_FETCH_"`
	WriteFile       ToolConfig         `json:"write_file"                                               envPrefix:"PICOCLAW_TOOLS_WRITE_FILE_"`
}

type SearchCacheConfig struct {
	MaxSize    int `json:"max_size"    env:"PICOCLAW_SKILLS_SEARCH_CACHE_MAX_SIZE"`
	TTLSeconds int `json:"ttl_seconds" env:"PICOCLAW_SKILLS_SEARCH_CACHE_TTL_SECONDS"`
}

type SkillsRegistriesConfig struct {
	ClawHub ClawHubRegistryConfig `json:"clawhub"`
}

type SkillsGithubConfig struct {
	token    string
	secDirty bool
	Proxy    string `json:"proxy,omitempty" env:"PICOCLAW_TOOLS_SKILLS_GITHUB_PROXY"`
}

// Token returns the GitHub token
func (c *SkillsGithubConfig) Token() string {
	return c.token
}

// SetToken sets the GitHub token
func (c *SkillsGithubConfig) SetToken(token string) {
	c.token = token
	c.secDirty = true
}

type ClawHubRegistryConfig struct {
	Enabled         bool   `json:"enabled"           env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_ENABLED"`
	BaseURL         string `json:"base_url"          env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_BASE_URL"`
	authToken       string
	secDirty        bool
	SearchPath      string `json:"search_path"       env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_SEARCH_PATH"`
	SkillsPath      string `json:"skills_path"       env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_SKILLS_PATH"`
	DownloadPath    string `json:"download_path"     env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_DOWNLOAD_PATH"`
	Timeout         int    `json:"timeout"           env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_TIMEOUT"`
	MaxZipSize      int    `json:"max_zip_size"      env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_MAX_ZIP_SIZE"`
	MaxResponseSize int    `json:"max_response_size" env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_MAX_RESPONSE_SIZE"`
}

// AuthToken returns the ClawHub auth token
func (c *ClawHubRegistryConfig) AuthToken() string {
	return c.authToken
}

// SetAuthToken sets the ClawHub auth token
func (c *ClawHubRegistryConfig) SetAuthToken(token string) {
	c.authToken = token
	c.secDirty = true
}

// MCPServerConfig defines configuration for a single MCP server
type MCPServerConfig struct {
	// Enabled indicates whether this MCP server is active
	Enabled bool `json:"enabled"`
	// Deferred controls whether this server's tools are registered as hidden (deferred/discovery mode).
	// When nil, the global Discovery.Enabled setting applies.
	// When explicitly set to true or false, it overrides the global setting for this server only.
	Deferred *bool `json:"deferred,omitempty"`
	// Command is the executable to run (e.g., "npx", "python", "/path/to/server")
	Command string `json:"command"`
	// Args are the arguments to pass to the command
	Args []string `json:"args,omitempty"`
	// Env are environment variables to set for the server process (stdio only)
	Env map[string]string `json:"env,omitempty"`
	// EnvFile is the path to a file containing environment variables (stdio only)
	EnvFile string `json:"env_file,omitempty"`
	// Type is "stdio", "sse", or "http" (default: stdio if command is set, sse if url is set)
	Type string `json:"type,omitempty"`
	// URL is used for SSE/HTTP transport
	URL string `json:"url,omitempty"`
	// Headers are HTTP headers to send with requests (sse/http only)
	Headers map[string]string `json:"headers,omitempty"`
}

// MCPConfig defines configuration for all MCP servers
type MCPConfig struct {
	ToolConfig `                    envPrefix:"PICOCLAW_TOOLS_MCP_"`
	Discovery  ToolDiscoveryConfig `                                json:"discovery"`
	// Servers is a map of server name to server configuration
	Servers map[string]MCPServerConfig `json:"servers,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	// First, try to detect config version by reading the version field
	var versionInfo struct {
		Version int `json:"version"`
	}
	if e := json.Unmarshal(data, &versionInfo); e != nil {
		return nil, fmt.Errorf("failed to detect config version: %w", e)
	}
	if len(data) <= 10 {
		return DefaultConfig().WithSecurity(&SecurityConfig{}), nil
	}

	// Load config based on detected version
	var cfg *Config
	switch versionInfo.Version {
	case 0:
		logger.InfoF("config migrate start", map[string]any{"from": versionInfo.Version, "to": CurrentVersion})
		// Legacy config (no version field)
		v, e := loadConfigV0(data)
		if e != nil {
			return nil, e
		}
		cfg, e = v.Migrate()
		if e != nil {
			logger.DebugF("config migrate fail", map[string]any{"from": versionInfo.Version, "to": CurrentVersion})
			return nil, e
		}
		logger.DebugF("config migrate success", map[string]any{"from": versionInfo.Version, "to": CurrentVersion})
		defer func() {
			_ = SaveConfig(path, cfg)
		}()
	case CurrentVersion:
		// Current version
		cfg, err = loadConfig(data)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported config version: %d", versionInfo.Version)
	}

	// Load security configuration
	securityPath := securityPath(path)
	sec, err := loadSecurityConfig(securityPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load security config: %w", err)
	}

	// Apply security references from .security.yml BEFORE resolveAPIKeys
	// This resolves ref: references to actual values
	if err := applySecurityConfig(cfg, sec); err != nil {
		return nil, fmt.Errorf("failed to apply security config: %w", err)
	}

	if passphrase := credential.PassphraseProvider(); passphrase != "" {
		for _, m := range cfg.ModelList {
			for _, k := range m.apiKeys {
				if k != "" && !strings.HasPrefix(k, "enc://") && !strings.HasPrefix(k, "file://") {
					fmt.Fprintf(os.Stderr,
						"picoclaw: warning: model %q has a plaintext api_key; call SaveConfig to encrypt it\n",
						m.ModelName)
					break // Only warn once per model
				}
			}
		}
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	if err := resolveAPIKeys(cfg.ModelList, filepath.Dir(path)); err != nil {
		return nil, err
	}

	// Resolve security fields like authToken that may contain file:// references
	if err := resolveSecurityFields(cfg, filepath.Dir(path)); err != nil {
		return nil, err
	}

	// Expand multi-key configs into separate entries for key-level failover
	cfg.ModelList = expandMultiKeyModels(cfg.ModelList)

	// Migrate legacy channel config fields to new unified structures
	cfg.migrateChannelConfigs()

	// Validate model_list for uniqueness and required fields
	if err := cfg.ValidateModelList(); err != nil {
		return nil, err
	}

	// Ensure Workspace has a default if not set
	if cfg.Agents.Defaults.Workspace == "" {
		homePath, _ := os.UserHomeDir()
		if picoclawHome := os.Getenv(EnvHome); picoclawHome != "" {
			homePath = picoclawHome
		} else if homePath != "" {
			homePath = filepath.Join(homePath, pkg.DefaultPicoClawHome)
		}
		cfg.Agents.Defaults.Workspace = filepath.Join(homePath, pkg.WorkspaceName)
	}

	return cfg, nil
}

func copyArray[T any](dst, src *[]T) {
	*dst = make([]T, len(*src))
	copy(*dst, *src)
}

// applySecurityConfig resolves all security references in config
// It checks each field for "ref:" prefixed values and resolves them from .security.yml
func applySecurityConfig(cfg *Config, sec *SecurityConfig) error {
	if sec == nil {
		return nil
	}

	if sec.Web.Brave != nil && len(sec.Web.Brave.APIKeys) > 0 {
		copyArray(&cfg.Tools.Web.Brave.apiKeys, &sec.Web.Brave.APIKeys)
	}

	if sec.Web.Tavily != nil && len(sec.Web.Tavily.APIKeys) > 0 {
		copyArray(&cfg.Tools.Web.Tavily.apiKeys, &sec.Web.Tavily.APIKeys)
	}

	if sec.Web.Perplexity != nil && len(sec.Web.Perplexity.APIKeys) > 0 {
		copyArray(&cfg.Tools.Web.Perplexity.apiKeys, &sec.Web.Perplexity.APIKeys)
	}

	if sec.Web.GLMSearch != nil && sec.Web.GLMSearch.APIKey != "" {
		cfg.Tools.Web.GLMSearch.apiKey = sec.Web.GLMSearch.APIKey
	}

	if sec.Web.BaiduSearch != nil && sec.Web.BaiduSearch.APIKey != "" {
		cfg.Tools.Web.BaiduSearch.apiKey = sec.Web.BaiduSearch.APIKey
	}

	if sec.Skills.Github != nil && sec.Skills.Github.Token != "" {
		cfg.Tools.Skills.Github.token = sec.Skills.Github.Token
	}

	if sec.Skills.ClawHub != nil && sec.Skills.ClawHub.AuthToken != "" {
		cfg.Tools.Skills.Registries.ClawHub.authToken = sec.Skills.ClawHub.AuthToken
	}

	names := toNameIndex(cfg.ModelList)
	for i, model := range cfg.ModelList {
		// Try exact match first (e.g., "abc:0" -> "abc:0")
		if entry, exists := sec.ModelList[names[i]]; exists {
			copyArray(&model.apiKeys, &entry.APIKeys)
			model.secModelName = names[i]
			continue
		}

		// Try match without index suffix (e.g., "abc" -> "abc")
		// This allows .security.yml to use simpler keys like "test-model" instead of "test-model:0"
		baseName := model.ModelName
		if entry, exists := sec.ModelList[baseName]; exists {
			copyArray(&model.apiKeys, &entry.APIKeys)
			model.secModelName = baseName
			continue
		}
	}

	// Handle Telegram token
	if sec.Channels.Telegram != nil && sec.Channels.Telegram.Token != "" {
		cfg.Channels.Telegram.token = sec.Channels.Telegram.Token
	}

	// Handle Feishu credentials
	if sec.Channels.Feishu != nil {
		if sec.Channels.Feishu.AppSecret != "" {
			cfg.Channels.Feishu.appSecret = sec.Channels.Feishu.AppSecret
		}
		if sec.Channels.Feishu.EncryptKey != "" {
			cfg.Channels.Feishu.encryptKey = sec.Channels.Feishu.EncryptKey
		}
		if sec.Channels.Feishu.VerificationToken != "" {
			cfg.Channels.Feishu.verificationToken = sec.Channels.Feishu.VerificationToken
		}
	}

	// Handle Discord token
	if sec.Channels.Discord != nil && sec.Channels.Discord.Token != "" {
		cfg.Channels.Discord.token = sec.Channels.Discord.Token
	}

	// Handle Weixin token
	if sec.Channels.Weixin != nil && sec.Channels.Weixin.Token != "" {
		cfg.Channels.Discord.token = sec.Channels.Discord.Token
	}

	// Handle DingTalk client secret
	if sec.Channels.DingTalk != nil && sec.Channels.DingTalk.ClientSecret != "" {
		cfg.Channels.DingTalk.clientSecret = sec.Channels.DingTalk.ClientSecret
	}

	// Handle Slack tokens
	if sec.Channels.Slack != nil {
		if sec.Channels.Slack.BotToken != "" {
			cfg.Channels.Slack.botToken = sec.Channels.Slack.BotToken
		}
		if sec.Channels.Slack.AppToken != "" {
			cfg.Channels.Slack.appToken = sec.Channels.Slack.AppToken
		}
	}

	// Handle Matrix access token
	if sec.Channels.Matrix != nil && sec.Channels.Matrix.AccessToken != "" {
		cfg.Channels.Matrix.accessToken = sec.Channels.Matrix.AccessToken
	}

	// Handle LINE credentials
	if sec.Channels.LINE != nil {
		if sec.Channels.LINE.ChannelSecret != "" {
			cfg.Channels.LINE.channelSecret = sec.Channels.LINE.ChannelSecret
		}
		if sec.Channels.LINE.ChannelAccessToken != "" {
			cfg.Channels.LINE.channelAccessToken = sec.Channels.LINE.ChannelAccessToken
		}
	}

	// Handle OneBot access token
	if sec.Channels.OneBot != nil && sec.Channels.OneBot.AccessToken != "" {
		cfg.Channels.OneBot.accessToken = sec.Channels.OneBot.AccessToken
	}

	// Handle WeCom token and encoding key
	if sec.Channels.WeCom != nil {
		if sec.Channels.WeCom.Token != "" {
			cfg.Channels.WeCom.token = sec.Channels.WeCom.Token
		}
		if sec.Channels.WeCom.EncodingAESKey != "" {
			cfg.Channels.WeCom.encodingAESKey = sec.Channels.WeCom.EncodingAESKey
		}
	}

	// Handle WeCom App credentials
	if sec.Channels.WeComApp != nil {
		if sec.Channels.WeComApp.CorpSecret != "" {
			cfg.Channels.WeComApp.corpSecret = sec.Channels.WeComApp.CorpSecret
		}
		if sec.Channels.WeComApp.Token != "" {
			cfg.Channels.WeComApp.token = sec.Channels.WeComApp.Token
		}
		if sec.Channels.WeComApp.EncodingAESKey != "" {
			cfg.Channels.WeComApp.encodingAESKey = sec.Channels.WeComApp.EncodingAESKey
		}
	}

	// Handle WeCom AI Bot credentials
	if sec.Channels.WeComAIBot != nil {
		if sec.Channels.WeComAIBot.Token != "" {
			cfg.Channels.WeComAIBot.token = sec.Channels.WeComAIBot.Token
		}
		if sec.Channels.WeComAIBot.EncodingAESKey != "" {
			cfg.Channels.WeComAIBot.encodingAESKey = sec.Channels.WeComAIBot.EncodingAESKey
		}
		if sec.Channels.WeComAIBot.Secret != "" {
			cfg.Channels.WeComAIBot.secret = sec.Channels.WeComAIBot.Secret
		}
	}

	// Handle Pico channel token
	if sec.Channels.Pico != nil && sec.Channels.Pico.Token != "" {
		cfg.Channels.Pico.token = sec.Channels.Pico.Token
	}

	// Handle IRC passwords
	if sec.Channels.IRC != nil {
		if sec.Channels.IRC.Password != "" {
			cfg.Channels.IRC.password = sec.Channels.IRC.Password
		}
		if sec.Channels.IRC.NickServPassword != "" {
			cfg.Channels.IRC.nickServPassword = sec.Channels.IRC.NickServPassword
		}
		if sec.Channels.IRC.SASLPassword != "" {
			cfg.Channels.IRC.saslPassword = sec.Channels.IRC.SASLPassword
		}
	}

	// Handle QQ app secret
	if sec.Channels.QQ != nil && sec.Channels.QQ.AppSecret != "" {
		cfg.Channels.QQ.appSecret = sec.Channels.QQ.AppSecret
	}

	cfg.security = sec

	return nil
}

func toNameIndex(list []*ModelConfig) []string {
	nameList := make([]string, 0, len(list))
	countMap := make(map[string]int)
	for _, model := range list {
		name := model.ModelName
		index := countMap[name]
		nameList = append(nameList, fmt.Sprintf("%s:%d", name, index))
		countMap[name]++
	}
	return nameList
}

// encryptPlaintextAPIKeys returns a copy of models with plaintext api_key values
// encrypted. Returns (nil, nil) when nothing changed (all keys already sealed or
// empty). Returns (nil, error) if any key fails to encrypt — callers must treat
// this as a hard failure to prevent a mixed plaintext/ciphertext state on disk.
// Symmetric counterpart of resolveAPIKeys: both operate purely on []ModelConfig
// and leave JSON marshaling to the caller.
func encryptPlaintextAPIKeys(
	models map[string]ModelSecurityEntry,
	passphrase string,
) (map[string]ModelSecurityEntry, error) {
	sealed := make(map[string]ModelSecurityEntry, len(models))
	changed := false
	for k, m := range models {
		sealedEntry := ModelSecurityEntry{APIKeys: make([]string, len(m.APIKeys))}

		// Encrypt each key in APIKeys
		for i, key := range m.APIKeys {
			if key == "" || strings.HasPrefix(key, "enc://") || strings.HasPrefix(key, "file://") {
				sealedEntry.APIKeys[i] = key
				continue
			}
			encrypted, err := credential.Encrypt(passphrase, "", key)
			if err != nil {
				return nil, fmt.Errorf("cannot seal api_key for model %q: %w", k, err)
			}
			sealedEntry.APIKeys[i] = encrypted
			changed = true
		}

		sealed[k] = sealedEntry
	}
	if !changed {
		return nil, nil
	}
	return sealed, nil
}

// resolveAPIKeys decrypts or dereferences each api_key in models in-place.
// Supports plaintext (no-op), file:// (read from configDir), and enc:// (AES-GCM decrypt).
func resolveAPIKeys(models []*ModelConfig, configDir string) error {
	cr := credential.NewResolver(configDir)
	for i := range models {
		// Resolve APIKeys array
		for j, key := range models[i].apiKeys {
			resolved, err := cr.Resolve(key)
			if err != nil {
				return fmt.Errorf(
					"model_list[%d] (%s): api_keys[%d]: %w",
					i,
					models[i].ModelName,
					j,
					err,
				)
			}
			models[i].apiKeys[j] = resolved
		}
	}
	return nil
}

func (c *Config) migrateChannelConfigs() {
	// Discord: mention_only -> group_trigger.mention_only
	if c.Channels.Discord.MentionOnly && !c.Channels.Discord.GroupTrigger.MentionOnly {
		c.Channels.Discord.GroupTrigger.MentionOnly = true
	}

	// OneBot: group_trigger_prefix -> group_trigger.prefixes
	if len(c.Channels.OneBot.GroupTriggerPrefix) > 0 &&
		len(c.Channels.OneBot.GroupTrigger.Prefixes) == 0 {
		c.Channels.OneBot.GroupTrigger.Prefixes = c.Channels.OneBot.GroupTriggerPrefix
	}
}

func SaveConfig(path string, cfg *Config) error {
	if cfg.security == nil {
		logger.Errorf("config %#v", *cfg)
		if len(cfg.ModelList) > 0 {
			logger.Errorf("model[0] %#v", cfg.ModelList[0])
		}
		logger.ErrorC("config", "security is nil")
		return fmt.Errorf("security is nil")
	}
	// Ensure version is always set when saving
	if cfg.Version == 0 {
		cfg.Version = CurrentVersion
	}
	names := toNameIndex(cfg.ModelList)
	for i, m := range cfg.ModelList {
		if m.secDirty {
			if m.secModelName == "" {
				m.secModelName = names[i]
			}
			cfg.security.ModelList[m.secModelName] = ModelSecurityEntry{
				APIKeys: m.apiKeys,
			}
			m.secDirty = false
		}
	}
	if cfg.Channels.Pico.secDirty {
		cfg.security.Channels.Pico = &PicoSecurity{
			Token: cfg.Channels.Pico.Token(),
		}
		cfg.Channels.Pico.secDirty = false
	}
	if cfg.Channels.IRC.secDirty {
		cfg.security.Channels.IRC = &IRCSecurity{
			Password:         cfg.Channels.IRC.password,
			NickServPassword: cfg.Channels.IRC.nickServPassword,
			SASLPassword:     cfg.Channels.IRC.saslPassword,
		}
		cfg.Channels.IRC.secDirty = false
	}
	if cfg.Channels.Telegram.secDirty {
		cfg.security.Channels.Telegram = &TelegramSecurity{
			Token: cfg.Channels.Telegram.Token(),
		}
		cfg.Channels.Telegram.secDirty = false
	}
	if cfg.Channels.Feishu.secDirty {
		cfg.security.Channels.Feishu = &FeishuSecurity{
			AppSecret:         cfg.Channels.Feishu.AppSecret(),
			EncryptKey:        cfg.Channels.Feishu.EncryptKey(),
			VerificationToken: cfg.Channels.Feishu.VerificationToken(),
		}
		cfg.Channels.Feishu.secDirty = false
	}
	if cfg.Channels.Discord.secDirty {
		cfg.security.Channels.Discord = &DiscordSecurity{
			Token: cfg.Channels.Discord.Token(),
		}
		cfg.Channels.Discord.secDirty = false
	}
	if cfg.Channels.Weixin.secDirty {
		cfg.security.Channels.Weixin = &WeixinSecurity{
			Token: cfg.Channels.Weixin.Token(),
		}
		cfg.Channels.Discord.secDirty = false
	}
	if cfg.Channels.QQ.secDirty {
		cfg.security.Channels.QQ = &QQSecurity{
			AppSecret: cfg.Channels.QQ.AppSecret(),
		}
		cfg.Channels.QQ.secDirty = false
	}
	if cfg.Channels.DingTalk.secDirty {
		cfg.security.Channels.DingTalk = &DingTalkSecurity{
			ClientSecret: cfg.Channels.DingTalk.ClientSecret(),
		}
		cfg.Channels.DingTalk.secDirty = false
	}
	if cfg.Channels.Slack.secDirty {
		cfg.security.Channels.Slack = &SlackSecurity{
			BotToken: cfg.Channels.Slack.BotToken(),
			AppToken: cfg.Channels.Slack.AppToken(),
		}
		cfg.Channels.Slack.secDirty = false
	}
	if cfg.Channels.Matrix.secDirty {
		cfg.security.Channels.Matrix = &MatrixSecurity{
			AccessToken: cfg.Channels.Matrix.AccessToken(),
		}
		cfg.Channels.Matrix.secDirty = false
	}
	if cfg.Channels.LINE.secDirty {
		cfg.security.Channels.LINE = &LINESecurity{
			ChannelSecret:      cfg.Channels.LINE.ChannelSecret(),
			ChannelAccessToken: cfg.Channels.LINE.ChannelAccessToken(),
		}
		cfg.Channels.LINE.secDirty = false
	}
	if cfg.Channels.OneBot.secDirty {
		cfg.security.Channels.OneBot = &OneBotSecurity{
			AccessToken: cfg.Channels.OneBot.AccessToken(),
		}
		cfg.Channels.OneBot.secDirty = false
	}
	if cfg.Channels.WeCom.secDirty {
		cfg.security.Channels.WeCom = &WeComSecurity{
			Token:          cfg.Channels.WeCom.Token(),
			EncodingAESKey: cfg.Channels.WeCom.EncodingAESKey(),
		}
		cfg.Channels.WeCom.secDirty = false
	}
	if cfg.Channels.WeComApp.secDirty {
		cfg.security.Channels.WeComApp = &WeComAppSecurity{
			CorpSecret:     cfg.Channels.WeComApp.CorpSecret(),
			Token:          cfg.Channels.WeComApp.Token(),
			EncodingAESKey: cfg.Channels.WeComApp.EncodingAESKey(),
		}
		cfg.Channels.WeComApp.secDirty = false
	}
	if cfg.Channels.WeComAIBot.secDirty {
		cfg.security.Channels.WeComAIBot = &WeComAIBotSecurity{
			Token:          cfg.Channels.WeComAIBot.Token(),
			EncodingAESKey: cfg.Channels.WeComAIBot.EncodingAESKey(),
			Secret:         cfg.Channels.WeComAIBot.Secret(),
		}
		cfg.Channels.WeComAIBot.secDirty = false
	}
	if cfg.Tools.Web.Brave.secDirty {
		cfg.security.Web.Brave = &BraveSecurity{
			APIKeys: cfg.Tools.Web.Brave.APIKeys(),
		}
		cfg.Tools.Web.Brave.secDirty = false
	}
	if cfg.Tools.Web.Tavily.secDirty {
		cfg.security.Web.Tavily = &TavilySecurity{
			APIKeys: cfg.Tools.Web.Tavily.APIKeys(),
		}
		cfg.Tools.Web.Tavily.secDirty = false
	}
	if cfg.Tools.Web.Perplexity.secDirty {
		cfg.security.Web.Perplexity = &PerplexitySecurity{
			APIKeys: cfg.Tools.Web.Perplexity.APIKeys(),
		}
		cfg.Tools.Web.Perplexity.secDirty = false
	}
	if cfg.Tools.Web.GLMSearch.secDirty {
		cfg.security.Web.GLMSearch = &GLMSearchSecurity{
			APIKey: cfg.Tools.Web.GLMSearch.APIKey(),
		}
		cfg.Tools.Web.GLMSearch.secDirty = false
	}
	if cfg.Tools.Web.BaiduSearch.secDirty {
		cfg.security.Web.BaiduSearch = &BaiduSearchSecurity{
			APIKey: cfg.Tools.Web.BaiduSearch.APIKey(),
		}
		cfg.Tools.Web.BaiduSearch.secDirty = false
	}
	if cfg.Tools.Skills.Github.secDirty {
		cfg.security.Skills.Github = &GithubSecurity{
			Token: cfg.Tools.Skills.Github.Token(),
		}
		cfg.Tools.Skills.Github.secDirty = false
	}
	if cfg.Tools.Skills.Registries.ClawHub.secDirty {
		cfg.security.Skills.ClawHub = &ClawHubSecurity{
			AuthToken: cfg.Tools.Skills.Registries.ClawHub.AuthToken(),
		}
		cfg.Tools.Skills.Registries.ClawHub.secDirty = false
	}

	if passphrase := credential.PassphraseProvider(); passphrase != "" {
		sealed, err := encryptPlaintextAPIKeys(cfg.security.ModelList, passphrase)
		if err != nil {
			return err
		}
		if sealed != nil {
			cfg.security.ModelList = sealed
		}
	}
	if err := saveSecurityConfig(securityPath(path), cfg.security); err != nil {
		logger.ErrorCF("config", "cannot save .security.yml", map[string]any{"error": err})
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(path, data, 0o600)
}

func (c *Config) WorkspacePath() string {
	return expandHome(c.Agents.Defaults.Workspace)
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

// GetModelConfig returns the ModelConfig for the given model name.
// If multiple configs exist with the same model_name, it uses round-robin
// selection for load balancing. Returns an error if the model is not found.
func (c *Config) GetModelConfig(modelName string) (*ModelConfig, error) {
	matches := c.findMatches(modelName)
	if len(matches) == 0 {
		return nil, fmt.Errorf("model %q not found in model_list or providers", modelName)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}

	// Multiple configs - use round-robin for load balancing
	idx := (rrCounter.Add(1) - 1) % uint64(len(matches))
	return matches[idx], nil
}

// findMatches finds all ModelConfig entries with the given model_name.
func (c *Config) findMatches(modelName string) []*ModelConfig {
	var matches []*ModelConfig
	for i := range c.ModelList {
		if c.ModelList[i].ModelName == modelName {
			matches = append(matches, c.ModelList[i])
		}
	}
	return matches
}

// ValidateModelList validates all ModelConfig entries in the model_list.
// It checks that each model config is valid.
// Note: Multiple entries with the same model_name are allowed for load balancing.
func (c *Config) ValidateModelList() error {
	for i := range c.ModelList {
		if err := c.ModelList[i].Validate(); err != nil {
			return fmt.Errorf("model_list[%d]: %w", i, err)
		}
	}
	return nil
}

func (c *Config) SecurityCopyFrom(cfg *Config) {
	c.security = cfg.security
}

func MergeAPIKeys(apiKey string, apiKeys []string) []string {
	seen := make(map[string]struct{})
	var all []string

	if k := strings.TrimSpace(apiKey); k != "" {
		if _, exists := seen[k]; !exists {
			seen[k] = struct{}{}
			all = append(all, k)
		}
	}

	for _, k := range apiKeys {
		if trimmed := strings.TrimSpace(k); trimmed != "" {
			if _, exists := seen[trimmed]; !exists {
				seen[trimmed] = struct{}{}
				all = append(all, trimmed)
			}
		}
	}

	return all
}

// resolveSecurityFields resolves file:// and enc:// references in security-sensitive fields
// like authToken and token that are not part of ModelConfig's apiKeys
func resolveSecurityFields(cfg *Config, configDir string) error {
	cr := credential.NewResolver(configDir)

	// Resolve Web tool API keys - set apiKey field to first resolved apiKeys entry
	if len(cfg.Tools.Web.Brave.apiKeys) > 0 {
		keys := cfg.Tools.Web.Brave.apiKeys
		for i, key := range keys {
			resolved, err := cr.Resolve(key)
			if err != nil {
				return fmt.Errorf("brave api_keys[%d]: %w", i, err)
			}
			keys[i] = resolved
		}
	}

	if len(cfg.Tools.Web.Tavily.apiKeys) > 0 {
		keys := cfg.Tools.Web.Tavily.apiKeys
		for i, key := range keys {
			resolved, err := cr.Resolve(key)
			if err != nil {
				return fmt.Errorf("tavily api_keys[%d]: %w", i, err)
			}
			keys[i] = resolved
		}
	}

	if len(cfg.Tools.Web.Perplexity.apiKeys) > 0 {
		keys := cfg.Tools.Web.Perplexity.apiKeys
		for i, key := range keys {
			resolved, err := cr.Resolve(key)
			if err != nil {
				return fmt.Errorf("perplexity api_keys[%d]: %w", i, err)
			}
			keys[i] = resolved
		}
	}

	// GLMSearch has a private apiKey field
	if cfg.Tools.Web.GLMSearch.apiKey != "" {
		resolved, err := cr.Resolve(cfg.Tools.Web.GLMSearch.apiKey)
		if err != nil {
			return fmt.Errorf("glm api_key: %w", err)
		}
		cfg.Tools.Web.GLMSearch.apiKey = resolved
	}

	// Resolve Skills tokens
	if cfg.Tools.Skills.Github.token != "" {
		resolved, err := cr.Resolve(cfg.Tools.Skills.Github.token)
		if err != nil {
			return fmt.Errorf("github token: %w", err)
		}
		cfg.Tools.Skills.Github.token = resolved
	}

	if cfg.Tools.Skills.Registries.ClawHub.authToken != "" {
		resolved, err := cr.Resolve(cfg.Tools.Skills.Registries.ClawHub.authToken)
		if err != nil {
			return fmt.Errorf("clawhub auth_token: %w", err)
		}
		cfg.Tools.Skills.Registries.ClawHub.authToken = resolved
	}

	return nil
}

// expandMultiKeyModels expands ModelConfig entries with multiple API keys into
// separate entries for key-level failover. Each key gets its own ModelConfig entry,
// and the original entry's fallbacks are set up to chain through the expanded entries.
//
// Example: {"model_name": "gpt-4", "api_keys": ["k1", "k2", "k3"]}
// Becomes:
//   - {"model_name": "gpt-4", "api_keys": ["k1"], "fallbacks": ["gpt-4__key_1", "gpt-4__key_2"]}
//   - {"model_name": "gpt-4__key_1", "api_keys": {"k2"}}
//   - {"model_name": "gpt-4__key_2", "api_keys": {"k3"}}
func expandMultiKeyModels(models []*ModelConfig) []*ModelConfig {
	var expanded []*ModelConfig

	for _, m := range models {
		keys := MergeAPIKeys("", m.apiKeys)

		// Single key or no keys: keep as-is
		if len(keys) <= 1 {
			m.apiKeys = keys
			expanded = append(expanded, m)
			continue
		}

		// Multiple keys: expand
		originalName := m.ModelName

		// Create entries for additional keys (key_1, key_2, ...)
		var fallbackNames []string
		for i := 1; i < len(keys); i++ {
			suffix := fmt.Sprintf("__key_%d", i)
			expandedName := originalName + suffix

			// Create a copy for the additional key
			additionalEntry := &ModelConfig{
				ModelName:      expandedName,
				Model:          m.Model,
				APIBase:        m.APIBase,
				apiKeys:        []string{keys[i]},
				Proxy:          m.Proxy,
				AuthMethod:     m.AuthMethod,
				ConnectMode:    m.ConnectMode,
				Workspace:      m.Workspace,
				RPM:            m.RPM,
				MaxTokensField: m.MaxTokensField,
				RequestTimeout: m.RequestTimeout,
				ThinkingLevel:  m.ThinkingLevel,
				ExtraBody:      m.ExtraBody,
			}
			expanded = append(expanded, additionalEntry)
			fallbackNames = append(fallbackNames, expandedName)
		}

		// Create the primary entry with first key and fallbacks
		primaryEntry := &ModelConfig{
			ModelName:      originalName,
			Model:          m.Model,
			APIBase:        m.APIBase,
			Proxy:          m.Proxy,
			AuthMethod:     m.AuthMethod,
			ConnectMode:    m.ConnectMode,
			Workspace:      m.Workspace,
			RPM:            m.RPM,
			MaxTokensField: m.MaxTokensField,
			RequestTimeout: m.RequestTimeout,
			ThinkingLevel:  m.ThinkingLevel,
			ExtraBody:      m.ExtraBody,
			apiKeys:        []string{keys[0]},
		}

		// Prepend new fallbacks to existing ones
		if len(fallbackNames) > 0 {
			primaryEntry.Fallbacks = append(fallbackNames, m.Fallbacks...)
		} else if len(m.Fallbacks) > 0 {
			primaryEntry.Fallbacks = m.Fallbacks
		}

		expanded = append(expanded, primaryEntry)
	}

	return expanded
}

func (t *ToolsConfig) IsToolEnabled(name string) bool {
	switch name {
	case "web":
		return t.Web.Enabled
	case "cron":
		return t.Cron.Enabled
	case "exec":
		return t.Exec.Enabled
	case "skills":
		return t.Skills.Enabled
	case "media_cleanup":
		return t.MediaCleanup.Enabled
	case "append_file":
		return t.AppendFile.Enabled
	case "edit_file":
		return t.EditFile.Enabled
	case "find_skills":
		return t.FindSkills.Enabled
	case "i2c":
		return t.I2C.Enabled
	case "install_skill":
		return t.InstallSkill.Enabled
	case "list_dir":
		return t.ListDir.Enabled
	case "message":
		return t.Message.Enabled
	case "read_file":
		return t.ReadFile.Enabled
	case "spawn":
		return t.Spawn.Enabled
	case "spawn_status":
		return t.SpawnStatus.Enabled
	case "spi":
		return t.SPI.Enabled
	case "subagent":
		return t.Subagent.Enabled
	case "web_fetch":
		return t.WebFetch.Enabled
	case "send_file":
		return t.SendFile.Enabled
	case "write_file":
		return t.WriteFile.Enabled
	case "mcp":
		return t.MCP.Enabled
	default:
		return true
	}
}
