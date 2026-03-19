package openclaw

import (
	"encoding/json"
	"strings"
)

type OpenClawConfig struct {
	Auth     *OpenClawAuth     `json:"auth"`
	Models   *OpenClawModels   `json:"models"`
	Agents   *OpenClawAgents   `json:"agents"`
	Tools    *OpenClawTools    `json:"tools"`
	Channels *OpenClawChannels `json:"channels"`
	Cron     json.RawMessage   `json:"cron"`
	Hooks    json.RawMessage   `json:"hooks"`
	Skills   *OpenClawSkills   `json:"skills"`
	Memory   json.RawMessage   `json:"memory"`
	Session  json.RawMessage   `json:"session"`
}

type OpenClawAuth struct {
	Profiles json.RawMessage `json:"profiles"`
	Order    json.RawMessage `json:"order"`
}

type OpenClawModels struct {
	Providers map[string]json.RawMessage `json:"providers"`
}

type ProviderConfig struct {
	BaseUrl string        `json:"baseUrl"`
	Api     string        `json:"api"`
	Models  []ModelConfig `json:"models"`
	ApiKey  string        `json:"apiKey"`
}

type OpenClawModelConfig struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Reasoning     bool     `json:"reasoning"`
	Input         []string `json:"input"`
	Cost          Cost     `json:"cost"`
	ContextWindow int      `json:"contextWindow"`
	MaxTokens     int      `json:"maxTokens"`
	Api           string   `json:"api,omitempty"`
}

type Cost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

type OpenClawTools struct {
	Profile *string  `json:"profile"`
	Allow   []string `json:"allow"`
	Deny    []string `json:"deny"`
}

type OpenClawAgents struct {
	Defaults *OpenClawAgentDefaults `json:"defaults"`
	List     []OpenClawAgentEntry   `json:"list"`
}

type OpenClawAgentDefaults struct {
	Model     *OpenClawAgentModel `json:"model"`
	Workspace *string             `json:"workspace"`
	Tools     *OpenClawAgentTools `json:"tools"`
	Identity  *string             `json:"identity"`
}

type OpenClawAgentModel struct {
	Simple    string   `json:"-"`
	Primary   *string  `json:"primary"`
	Fallbacks []string `json:"fallbacks"`
}

func (m *OpenClawAgentModel) GetPrimary() string {
	if m.Simple != "" {
		return m.Simple
	}
	if m.Primary != nil {
		return *m.Primary
	}
	return ""
}

func (m *OpenClawAgentModel) GetFallbacks() []string {
	return m.Fallbacks
}

type OpenClawAgentEntry struct {
	ID        string              `json:"id"`
	Name      *string             `json:"name"`
	Model     *OpenClawAgentModel `json:"model"`
	Tools     *OpenClawAgentTools `json:"tools"`
	Workspace *string             `json:"workspace"`
	Skills    []string            `json:"skills"`
	Identity  *string             `json:"identity"`
}

type OpenClawAgentTools struct {
	Profile   *string  `json:"profile"`
	Allow     []string `json:"allow"`
	Deny      []string `json:"deny"`
	AlsoAllow []string `json:"alsoAllow"`
}

type OpenClawChannels struct {
	Telegram    *OpenClawTelegramConfig    `json:"telegram"`
	Discord     *OpenClawDiscordConfig     `json:"discord"`
	Slack       *OpenClawSlackConfig       `json:"slack"`
	WhatsApp    *OpenClawWhatsAppConfig    `json:"whatsapp"`
	Signal      *OpenClawSignalConfig      `json:"signal"`
	Matrix      *OpenClawMatrixConfig      `json:"matrix"`
	GoogleChat  *OpenClawGoogleChatConfig  `json:"googlechat"`
	Teams       *OpenClawTeamsConfig       `json:"msteams"`
	IRC         *OpenClawIrcConfig         `json:"irc"`
	Mattermost  *OpenClawMattermostConfig  `json:"mattermost"`
	IMessage    *OpenClawIMessageConfig    `json:"imessage"`
	BlueBubbles *OpenClawBlueBubblesConfig `json:"bluebubbles"`
	QQ          *OpenClawQQConfig          `json:"qq"`
	DingTalk    *OpenClawDingTalkConfig    `json:"dingtalk"`
	MaixCam     *OpenClawMaixCamConfig     `json:"maixcam"`
}

type OpenClawTelegramConfig struct {
	BotToken    *string  `json:"botToken"`
	AllowFrom   []string `json:"allowFrom"`
	GroupPolicy *string  `json:"groupPolicy"`
	DmPolicy    *string  `json:"dmPolicy"`
	Enabled     *bool    `json:"enabled"`
}

type OpenClawDiscordConfig struct {
	Token       *string         `json:"token"`
	Guilds      json.RawMessage `json:"guilds"`
	DmPolicy    *string         `json:"dmPolicy"`
	GroupPolicy *string         `json:"groupPolicy"`
	AllowFrom   []string        `json:"allowFrom"`
	Enabled     *bool           `json:"enabled"`
}

type OpenClawSlackConfig struct {
	BotToken    *string  `json:"botToken"`
	AppToken    *string  `json:"appToken"`
	DmPolicy    *string  `json:"dmPolicy"`
	GroupPolicy *string  `json:"groupPolicy"`
	AllowFrom   []string `json:"allowFrom"`
	Enabled     *bool    `json:"enabled"`
}

type OpenClawWhatsAppConfig struct {
	AuthDir     *string  `json:"authDir"`
	DmPolicy    *string  `json:"dmPolicy"`
	AllowFrom   []string `json:"allowFrom"`
	GroupPolicy *string  `json:"groupPolicy"`
	Enabled     *bool    `json:"enabled"`
	BridgeURL   *string  `json:"bridgeUrl"`
}

type OpenClawSignalConfig struct {
	HttpUrl   *string  `json:"httpUrl"`
	HttpHost  *string  `json:"httpHost"`
	HttpPort  *int     `json:"httpPort"`
	Account   *string  `json:"account"`
	DmPolicy  *string  `json:"dmPolicy"`
	AllowFrom []string `json:"allowFrom"`
	Enabled   *bool    `json:"enabled"`
}

type OpenClawMatrixConfig struct {
	Homeserver  *string  `json:"homeserver"`
	UserID      *string  `json:"userId"`
	AccessToken *string  `json:"accessToken"`
	Rooms       []string `json:"rooms"`
	DmPolicy    *string  `json:"dmPolicy"`
	AllowFrom   []string `json:"allowFrom"`
	Enabled     *bool    `json:"enabled"`
}

type OpenClawGoogleChatConfig struct {
	ServiceAccountFile *string `json:"serviceAccountFile"`
	WebhookPath        *string `json:"webhookPath"`
	BotUser            *string `json:"botUser"`
	DmPolicy           *string `json:"dmPolicy"`
	Enabled            *bool   `json:"enabled"`
}

type OpenClawTeamsConfig struct {
	AppID       *string  `json:"appId"`
	AppPassword *string  `json:"appPassword"`
	TenantID    *string  `json:"tenantId"`
	DmPolicy    *string  `json:"dmPolicy"`
	AllowFrom   []string `json:"allowFrom"`
	Enabled     *bool    `json:"enabled"`
}

type OpenClawIrcConfig struct {
	Host      *string  `json:"host"`
	Port      *int     `json:"port"`
	TLS       *bool    `json:"tls"`
	Nick      *string  `json:"nick"`
	Password  *string  `json:"password"`
	Channels  []string `json:"channels"`
	DmPolicy  *string  `json:"dmPolicy"`
	AllowFrom []string `json:"allowFrom"`
	Enabled   *bool    `json:"enabled"`
}

type OpenClawMattermostConfig struct {
	BotToken  *string  `json:"botToken"`
	BaseURL   *string  `json:"baseUrl"`
	DmPolicy  *string  `json:"dmPolicy"`
	AllowFrom []string `json:"allowFrom"`
	Enabled   *bool    `json:"enabled"`
}

type OpenClawIMessageConfig struct {
	CliPath   *string  `json:"cliPath"`
	DbPath    *string  `json:"dbPath"`
	DmPolicy  *string  `json:"dmPolicy"`
	AllowFrom []string `json:"allowFrom"`
	Enabled   *bool    `json:"enabled"`
}

type OpenClawBlueBubblesConfig struct {
	ServerURL *string  `json:"serverUrl"`
	Password  *string  `json:"password"`
	DmPolicy  *string  `json:"dmPolicy"`
	AllowFrom []string `json:"allowFrom"`
	Enabled   *bool    `json:"enabled"`
}

type OpenClawQQConfig struct {
	AppID     *string  `json:"appId"`
	AppSecret *string  `json:"appSecret"`
	DmPolicy  *string  `json:"dmPolicy"`
	AllowFrom []string `json:"allowFrom"`
	Enabled   *bool    `json:"enabled"`
}

type OpenClawDingTalkConfig struct {
	AppID     *string  `json:"appId"`
	AppSecret *string  `json:"appSecret"`
	DmPolicy  *string  `json:"dmPolicy"`
	AllowFrom []string `json:"allowFrom"`
	Enabled   *bool    `json:"enabled"`
}

type OpenClawMaixCamConfig struct {
	Host      *string  `json:"host"`
	Port      *int     `json:"port"`
	DmPolicy  *string  `json:"dmPolicy"`
	AllowFrom []string `json:"allowFrom"`
	Enabled   *bool    `json:"enabled"`
}

type OpenClawSkills struct {
	Entries map[string]json.RawMessage `json:"entries"`
	Load    json.RawMessage            `json:"load"`
}

type OpenClawProviderConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

func (c *OpenClawConfig) GetEnabled() bool {
	return true
}

func (c *OpenClawConfig) IsChannelEnabled(name string) bool {
	switch name {
	case "telegram":
		return c.Channels.Telegram == nil || c.Channels.Telegram.Enabled == nil || *c.Channels.Telegram.Enabled
	case "discord":
		return c.Channels.Discord == nil || c.Channels.Discord.Enabled == nil || *c.Channels.Discord.Enabled
	case "slack":
		return c.Channels.Slack == nil || c.Channels.Slack.Enabled == nil || *c.Channels.Slack.Enabled
	case "matrix":
		return c.Channels.Matrix == nil || c.Channels.Matrix.Enabled == nil || *c.Channels.Matrix.Enabled
	case "whatsapp":
		return c.Channels.WhatsApp == nil || c.Channels.WhatsApp.Enabled == nil || *c.Channels.WhatsApp.Enabled
	default:
		return false
	}
}

func (c *OpenClawConfig) GetDefaultModel() (provider, model string) {
	if c.Agents == nil || c.Agents.Defaults == nil || c.Agents.Defaults.Model == nil {
		return "anthropic", "claude-sonnet-4-20250514"
	}

	primary := c.Agents.Defaults.Model.GetPrimary()
	if primary == "" {
		return "anthropic", "claude-sonnet-4-20250514"
	}

	parts := strings.Split(primary, "/")
	if len(parts) > 1 {
		return mapProvider(parts[0]), parts[1]
	}

	return "anthropic", primary
}

func (c *OpenClawConfig) GetDefaultWorkspace() string {
	if c.Agents == nil || c.Agents.Defaults == nil || c.Agents.Defaults.Workspace == nil {
		return ""
	}
	return rewriteWorkspacePath(*c.Agents.Defaults.Workspace)
}

func (c *OpenClawConfig) GetAgents() []OpenClawAgentEntry {
	if c.Agents == nil {
		return nil
	}
	return c.Agents.List
}

func (c *OpenClawConfig) HasSkills() bool {
	return c.Skills != nil && c.Skills.Entries != nil && len(c.Skills.Entries) > 0
}

func (c *OpenClawConfig) HasMemory() bool {
	return c.Memory != nil && len(c.Memory) > 0
}

func (c *OpenClawConfig) HasCron() bool {
	return c.Cron != nil && len(c.Cron) > 0
}

func (c *OpenClawConfig) HasHooks() bool {
	return c.Hooks != nil && len(c.Hooks) > 0
}

func (c *OpenClawConfig) HasSession() bool {
	return c.Session != nil && len(c.Session) > 0
}

func (c *OpenClawConfig) HasAuthProfiles() bool {
	return c.Auth != nil && c.Auth.Profiles != nil && len(c.Auth.Profiles) > 0
}
