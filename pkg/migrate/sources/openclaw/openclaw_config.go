package openclaw

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
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
	Feishu      *OpenClawFeishuConfig      `json:"feishu"`
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

type OpenClawFeishuConfig struct {
	AppID             *string  `json:"appId"`
	AppSecret         *string  `json:"appSecret"`
	Domain            *string  `json:"domain"`
	DmPolicy          *string  `json:"dmPolicy"`
	Enabled           *bool    `json:"enabled"`
	VerificationToken *string  `json:"verificationToken"`
	EncryptKey        *string  `json:"encryptKey"`
	AllowFrom         []string `json:"allowFrom"`
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

func LoadOpenClawConfig(path string) (*OpenClawConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config OpenClawConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &config, nil
}

func LoadOpenClawConfigFromDir(dir string) (*OpenClawConfig, error) {
	candidates := []string{
		filepath.Join(dir, "openclaw.json"),
		filepath.Join(dir, "config.json"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return LoadOpenClawConfig(p)
		}
	}

	return nil, fmt.Errorf("no config file found in %s", dir)
}

func GetProviderConfig(models *OpenClawModels) map[string]OpenClawProviderConfig {
	result := make(map[string]OpenClawProviderConfig)
	if models == nil || models.Providers == nil {
		return result
	}

	for name, raw := range models.Providers {
		var prov OpenClawProviderConfig
		if err := json.Unmarshal(raw, &prov); err != nil {
			continue
		}
		mappedName := mapProvider(name)
		result[mappedName] = prov
	}

	return result
}

func GetProviderConfigFromDir(dir string) map[string]ProviderConfig {
	result := make(map[string]ProviderConfig)
	p := filepath.Join(dir, "agents", "main", "agent", "models.json")

	if _, err := os.Stat(p); err != nil {
		return result
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return result
	}
	var models OpenClawModels
	if err := json.Unmarshal(data, &models); err != nil {
		return result
	}

	for name, raw := range models.Providers {
		var prov ProviderConfig
		if err := json.Unmarshal(raw, &prov); err != nil {
			continue
		}
		mappedName := mapProvider(name)
		result[mappedName] = prov
	}
	return result
}

func (c *OpenClawConfig) IsChannelEnabled(name string) bool {
	switch name {
	case "telegram":
		return c.Channels.Telegram == nil || c.Channels.Telegram.Enabled == nil || *c.Channels.Telegram.Enabled
	case "discord":
		return c.Channels.Discord == nil || c.Channels.Discord.Enabled == nil || *c.Channels.Discord.Enabled
	case "slack":
		return c.Channels.Slack == nil || c.Channels.Slack.Enabled == nil || *c.Channels.Slack.Enabled
	case "whatsapp":
		return c.Channels.WhatsApp == nil || c.Channels.WhatsApp.Enabled == nil || *c.Channels.WhatsApp.Enabled
	case "feishu":
		return c.Channels.Feishu == nil || c.Channels.Feishu.Enabled == nil || *c.Channels.Feishu.Enabled
	default:
		return false
	}
}

func GetChannelAllowFrom(ch any) []string {
	switch c := ch.(type) {
	case *OpenClawTelegramConfig:
		if c == nil {
			return nil
		}
		return c.AllowFrom
	case *OpenClawDiscordConfig:
		if c == nil {
			return nil
		}
		return c.AllowFrom
	case *OpenClawSlackConfig:
		if c == nil {
			return nil
		}
		return c.AllowFrom
	case *OpenClawWhatsAppConfig:
		if c == nil {
			return nil
		}
		return c.AllowFrom
	case *OpenClawFeishuConfig:
		if c == nil {
			return nil
		}
		return c.AllowFrom
	default:
		return nil
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

func (c *OpenClawConfig) ConvertToPicoClaw(sourceHome string) (*PicoClawConfig, []string, error) {
	cfg := &PicoClawConfig{}
	var warnings []string

	provider, modelName := c.GetDefaultModel()
	cfg.Agents.Defaults.Workspace = c.GetDefaultWorkspace()
	cfg.Agents.Defaults.ModelName = modelName

	providerConfigs := GetProviderConfigFromDir(sourceHome)
	defaultAPIKey := ""
	defaultBaseURL := ""

	if provCfg, ok := providerConfigs[provider]; ok {
		defaultAPIKey = provCfg.ApiKey
		defaultBaseURL = provCfg.BaseUrl
	}

	cfg.ModelList = []ModelConfig{
		{
			ModelName: modelName,
			Model:     fmt.Sprintf("%s/%s", provider, modelName),
			APIKey:    defaultAPIKey,
			APIBase:   defaultBaseURL,
		},
	}

	for provName, provCfg := range providerConfigs {
		if provName == provider {
			continue
		}
		if provCfg.ApiKey != "" {
			continue
		}
		cfg.ModelList = append(cfg.ModelList, ModelConfig{
			ModelName: fmt.Sprintf("%s", provName),
			Model:     fmt.Sprintf("%s/%s", provName, provName),
			APIKey:    provCfg.ApiKey,
			APIBase:   provCfg.BaseUrl,
		})
	}

	cfg.Channels = c.convertChannels(&warnings)

	agentList := c.convertAgents(&warnings)
	if len(agentList) > 0 {
		cfg.Agents.List = agentList
	}

	if c.HasSkills() {
		warnings = append(
			warnings,
			fmt.Sprintf(
				"Skills (%d entries) not automatically migrated - reinstall via picoclaw CLI",
				len(c.Skills.Entries),
			),
		)
	}
	if c.HasMemory() {
		warnings = append(warnings, "Memory backend config not migrated - PicoClaw uses SQLite with vector embeddings")
	}
	if c.HasCron() {
		warnings = append(
			warnings,
			"Cron job scheduling not supported in PicoClaw - consider using external schedulers",
		)
	}
	if c.HasHooks() {
		warnings = append(warnings, "Webhook hooks not supported in PicoClaw - use event system instead")
	}
	if c.HasSession() {
		warnings = append(warnings, "Session scope config differs - PicoClaw uses per-agent sessions by default")
	}
	if c.HasAuthProfiles() {
		warnings = append(
			warnings,
			"Auth profiles (API keys, OAuth tokens) not migrated for security - set env vars manually",
		)
	}

	return cfg, warnings, nil
}

type ModelConfig struct {
	ModelName string `json:"model_name"`
	Model     string `json:"model"`
	APIBase   string `json:"api_base,omitempty"`
	APIKey    string `json:"api_key"`
	Proxy     string `json:"proxy,omitempty"`
}

type PicoClawConfig struct {
	Agents    AgentsConfig   `json:"agents"`
	Bindings  []AgentBinding `json:"bindings,omitempty"`
	Channels  ChannelsConfig `json:"channels"`
	ModelList []ModelConfig  `json:"model_list"`
	Gateway   GatewayConfig  `json:"gateway"`
	Tools     ToolsConfig    `json:"tools"`
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
	List     []AgentConfig `json:"list,omitempty"`
}

type AgentDefaults struct {
	Workspace           string   `json:"workspace"`
	RestrictToWorkspace bool     `json:"restrict_to_workspace"`
	Provider            string   `json:"provider"`
	ModelName           string   `json:"model_name"`
	Model               string   `json:"model,omitempty"`
	ModelFallbacks      []string `json:"model_fallbacks,omitempty"`
	ImageModel          string   `json:"image_model,omitempty"`
	ImageModelFallbacks []string `json:"image_model_fallbacks,omitempty"`
	MaxTokens           int      `json:"max_tokens"`
	Temperature         *float64 `json:"temperature,omitempty"`
	MaxToolIterations   int      `json:"max_tool_iterations"`
}

type AgentConfig struct {
	ID        string            `json:"id"`
	Default   bool              `json:"default,omitempty"`
	Name      string            `json:"name,omitempty"`
	Workspace string            `json:"workspace,omitempty"`
	Model     *AgentModelConfig `json:"model,omitempty"`
	Skills    []string          `json:"skills,omitempty"`
}

type AgentModelConfig struct {
	Primary   string   `json:"primary,omitempty"`
	Fallbacks []string `json:"fallbacks,omitempty"`
}

type AgentBinding struct {
	AgentID string       `json:"agent_id"`
	Match   BindingMatch `json:"match"`
}

type BindingMatch struct {
	Channel   string     `json:"channel"`
	AccountID string     `json:"account_id,omitempty"`
	Peer      *PeerMatch `json:"peer,omitempty"`
	GuildID   string     `json:"guild_id,omitempty"`
	TeamID    string     `json:"team_id,omitempty"`
}

type PeerMatch struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type ChannelsConfig struct {
	WhatsApp WhatsAppConfig `json:"whatsapp"`
	Telegram TelegramConfig `json:"telegram"`
	Feishu   FeishuConfig   `json:"feishu"`
	Discord  DiscordConfig  `json:"discord"`
	MaixCam  MaixCamConfig  `json:"maixcam"`
	QQ       QQConfig       `json:"qq"`
	DingTalk DingTalkConfig `json:"dingtalk"`
	Slack    SlackConfig    `json:"slack"`
	LINE     LINEConfig     `json:"line"`
}

type WhatsAppConfig struct {
	Enabled   bool     `json:"enabled"`
	BridgeURL string   `json:"bridge_url"`
	AllowFrom []string `json:"allow_from"`
}

type TelegramConfig struct {
	Enabled   bool     `json:"enabled"`
	Token     string   `json:"token"`
	Proxy     string   `json:"proxy"`
	AllowFrom []string `json:"allow_from"`
}

type FeishuConfig struct {
	Enabled           bool     `json:"enabled"`
	AppID             string   `json:"app_id"`
	AppSecret         string   `json:"app_secret"`
	EncryptKey        string   `json:"encrypt_key"`
	VerificationToken string   `json:"verification_token"`
	AllowFrom         []string `json:"allow_from"`
}

type DiscordConfig struct {
	Enabled     bool     `json:"enabled"`
	Token       string   `json:"token"`
	MentionOnly bool     `json:"mention_only"`
	AllowFrom   []string `json:"allow_from"`
}

type MaixCamConfig struct {
	Enabled   bool     `json:"enabled"`
	Host      string   `json:"host"`
	Port      int      `json:"port"`
	AllowFrom []string `json:"allow_from"`
}

type QQConfig struct {
	Enabled   bool     `json:"enabled"`
	AppID     string   `json:"app_id"`
	AppSecret string   `json:"app_secret"`
	AllowFrom []string `json:"allow_from"`
}

type DingTalkConfig struct {
	Enabled      bool     `json:"enabled"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AllowFrom    []string `json:"allow_from"`
}

type SlackConfig struct {
	Enabled   bool     `json:"enabled"`
	BotToken  string   `json:"bot_token"`
	AppToken  string   `json:"app_token"`
	AllowFrom []string `json:"allow_from"`
}

type LINEConfig struct {
	Enabled            bool     `json:"enabled"`
	ChannelSecret      string   `json:"channel_secret"`
	ChannelAccessToken string   `json:"channel_access_token"`
	WebhookHost        string   `json:"webhook_host"`
	WebhookPort        int      `json:"webhook_port"`
	WebhookPath        string   `json:"webhook_path"`
	AllowFrom          []string `json:"allow_from"`
}

type GatewayConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type ToolsConfig struct {
	Web  WebToolsConfig `json:"web"`
	Cron CronConfig     `json:"cron"`
	Exec ExecConfig     `json:"exec"`
}

type WebToolsConfig struct {
	Brave      BraveConfig      `json:"brave"`
	Tavily     TavilyConfig     `json:"tavily"`
	DuckDuckGo DuckDuckGoConfig `json:"duckduckgo"`
	Perplexity PerplexityConfig `json:"perplexity"`
	Proxy      string           `json:"proxy,omitempty"`
}

type BraveConfig struct {
	Enabled    bool   `json:"enabled"`
	APIKey     string `json:"api_key"`
	MaxResults int    `json:"max_results"`
}

type TavilyConfig struct {
	Enabled    bool   `json:"enabled"`
	APIKey     string `json:"api_key"`
	BaseURL    string `json:"base_url"`
	MaxResults int    `json:"max_results"`
}

type DuckDuckGoConfig struct {
	Enabled    bool `json:"enabled"`
	MaxResults int  `json:"max_results"`
}

type PerplexityConfig struct {
	Enabled    bool   `json:"enabled"`
	APIKey     string `json:"api_key"`
	MaxResults int    `json:"max_results"`
}

type CronConfig struct {
	ExecTimeoutMinutes int `json:"exec_timeout_minutes"`
}

type ExecConfig struct {
	EnableDenyPatterns bool     `json:"enable_deny_patterns"`
	CustomDenyPatterns []string `json:"custom_deny_patterns"`
}

func (c *OpenClawConfig) convertChannels(warnings *[]string) ChannelsConfig {
	channels := ChannelsConfig{}

	if c.Channels == nil {
		return channels
	}

	if c.Channels.Telegram != nil {
		enabled := c.Channels.Telegram.Enabled == nil || *c.Channels.Telegram.Enabled
		channels.Telegram = TelegramConfig{
			Enabled:   enabled,
			AllowFrom: c.Channels.Telegram.AllowFrom,
		}
		if c.Channels.Telegram.BotToken != nil {
			channels.Telegram.Token = *c.Channels.Telegram.BotToken
		}
	}

	if c.Channels.Discord != nil {
		enabled := c.Channels.Discord.Enabled == nil || *c.Channels.Discord.Enabled
		channels.Discord = DiscordConfig{
			Enabled:   enabled,
			AllowFrom: c.Channels.Discord.AllowFrom,
		}
		if c.Channels.Discord.Token != nil {
			channels.Discord.Token = *c.Channels.Discord.Token
		}
	}

	if c.Channels.Slack != nil {
		enabled := c.Channels.Slack.Enabled == nil || *c.Channels.Slack.Enabled
		channels.Slack = SlackConfig{
			Enabled:   enabled,
			AllowFrom: c.Channels.Slack.AllowFrom,
		}
		if c.Channels.Slack.BotToken != nil {
			channels.Slack.BotToken = *c.Channels.Slack.BotToken
		}
		if c.Channels.Slack.AppToken != nil {
			channels.Slack.AppToken = *c.Channels.Slack.AppToken
		}
	}

	if c.Channels.WhatsApp != nil {
		enabled := c.Channels.WhatsApp.Enabled == nil || *c.Channels.WhatsApp.Enabled
		channels.WhatsApp = WhatsAppConfig{
			Enabled:   enabled,
			AllowFrom: c.Channels.WhatsApp.AllowFrom,
		}
		if c.Channels.WhatsApp.BridgeURL != nil {
			channels.WhatsApp.BridgeURL = *c.Channels.WhatsApp.BridgeURL
		}
	}

	if c.Channels.Feishu != nil {
		enabled := c.Channels.Feishu.Enabled == nil || *c.Channels.Feishu.Enabled
		channels.Feishu = FeishuConfig{
			Enabled:   enabled,
			AllowFrom: c.Channels.Feishu.AllowFrom,
		}
		if c.Channels.Feishu.AppID != nil {
			channels.Feishu.AppID = *c.Channels.Feishu.AppID
		}
		if c.Channels.Feishu.AppSecret != nil {
			channels.Feishu.AppSecret = *c.Channels.Feishu.AppSecret
		}
		if c.Channels.Feishu.EncryptKey != nil {
			channels.Feishu.EncryptKey = *c.Channels.Feishu.EncryptKey
		}
		if c.Channels.Feishu.VerificationToken != nil {
			channels.Feishu.VerificationToken = *c.Channels.Feishu.VerificationToken
		}
	}

	if c.Channels.QQ != nil && supportedChannels["qq"] {
		channels.QQ = QQConfig{
			Enabled:   true,
			AllowFrom: c.Channels.QQ.AllowFrom,
		}
		if c.Channels.QQ.AppID != nil {
			channels.QQ.AppID = *c.Channels.QQ.AppID
		}
		if c.Channels.QQ.AppSecret != nil {
			channels.QQ.AppSecret = *c.Channels.QQ.AppSecret
		}
	}

	if c.Channels.DingTalk != nil && supportedChannels["dingtalk"] {
		channels.DingTalk = DingTalkConfig{
			Enabled:   true,
			AllowFrom: c.Channels.DingTalk.AllowFrom,
		}
		if c.Channels.DingTalk.AppID != nil {
			channels.DingTalk.ClientID = *c.Channels.DingTalk.AppID
		}
		if c.Channels.DingTalk.AppSecret != nil {
			channels.DingTalk.ClientSecret = *c.Channels.DingTalk.AppSecret
		}
	}

	if c.Channels.MaixCam != nil && supportedChannels["maixcam"] {
		channels.MaixCam = MaixCamConfig{
			Enabled:   true,
			AllowFrom: c.Channels.MaixCam.AllowFrom,
		}
		if c.Channels.MaixCam.Host != nil {
			channels.MaixCam.Host = *c.Channels.MaixCam.Host
		}
		if c.Channels.MaixCam.Port != nil {
			channels.MaixCam.Port = *c.Channels.MaixCam.Port
		}
	}

	if c.Channels.Signal != nil {
		*warnings = append(*warnings, "Channel 'signal': No PicoClaw adapter available")
	}
	if c.Channels.Matrix != nil {
		*warnings = append(*warnings, "Channel 'matrix': No PicoClaw adapter available")
	}
	if c.Channels.IRC != nil {
		*warnings = append(*warnings, "Channel 'irc': No PicoClaw adapter available")
	}
	if c.Channels.Mattermost != nil {
		*warnings = append(*warnings, "Channel 'mattermost': No PicoClaw adapter available")
	}
	if c.Channels.IMessage != nil {
		*warnings = append(*warnings, "Channel 'imessage': macOS-only channel - requires manual setup")
	}
	if c.Channels.BlueBubbles != nil {
		*warnings = append(
			*warnings,
			"Channel 'bluebubbles': No PicoClaw adapter available - consider iMessage instead",
		)
	}

	return channels
}

func (c *OpenClawConfig) convertAgents(warnings *[]string) []AgentConfig {
	var agents []AgentConfig

	if c.Agents == nil {
		return agents
	}

	for _, entry := range c.Agents.List {
		agentID := entry.ID
		if agentID == "" {
			continue
		}

		agentName := agentID
		if entry.Name != nil {
			agentName = *entry.Name
		}

		agentCfg := AgentConfig{
			ID:      agentID,
			Name:    agentName,
			Default: len(agents) == 0,
		}

		if entry.Workspace != nil {
			agentCfg.Workspace = rewriteWorkspacePath(*entry.Workspace)
		}

		if entry.Model != nil {
			primary := entry.Model.GetPrimary()
			if primary != "" {
				agentCfg.Model = &AgentModelConfig{
					Primary:   primary,
					Fallbacks: entry.Model.GetFallbacks(),
				}
			}
		}

		if len(entry.Skills) > 0 {
			agentCfg.Skills = entry.Skills
		}

		agents = append(agents, agentCfg)
	}

	return agents
}

func (c *PicoClawConfig) ToStandardConfig() *config.Config {
	cfg := config.DefaultConfig()

	cfg.Agents.Defaults.Workspace = c.Agents.Defaults.Workspace
	cfg.Agents.Defaults.Provider = c.Agents.Defaults.Provider
	cfg.Agents.Defaults.ModelName = c.Agents.Defaults.ModelName
	cfg.Agents.Defaults.ModelFallbacks = c.Agents.Defaults.ModelFallbacks

	for _, m := range c.ModelList {
		cfg.ModelList = append(cfg.ModelList, config.ModelConfig{
			ModelName: m.ModelName,
			Model:     m.Model,
			APIBase:   m.APIBase,
			APIKey:    m.APIKey,
			Proxy:     m.Proxy,
		})
	}

	cfg.Channels = c.Channels.ToStandardChannels()
	cfg.Gateway = c.Gateway.ToStandardGateway()
	cfg.Tools = c.Tools.ToStandardTools()

	cfg.Agents.List = make([]config.AgentConfig, len(c.Agents.List))
	for i, a := range c.Agents.List {
		cfg.Agents.List[i] = config.AgentConfig{
			ID:        a.ID,
			Default:   a.Default,
			Name:      a.Name,
			Workspace: a.Workspace,
			Skills:    a.Skills,
		}
		if a.Model != nil {
			cfg.Agents.List[i].Model = &config.AgentModelConfig{
				Primary:   a.Model.Primary,
				Fallbacks: a.Model.Fallbacks,
			}
		}
	}

	return cfg
}

func (c ChannelsConfig) ToStandardChannels() config.ChannelsConfig {
	return config.ChannelsConfig{
		WhatsApp: config.WhatsAppConfig{
			Enabled:   c.WhatsApp.Enabled,
			BridgeURL: c.WhatsApp.BridgeURL,
		},
		Telegram: config.TelegramConfig{
			Enabled: c.Telegram.Enabled,
			Token:   c.Telegram.Token,
			Proxy:   c.Telegram.Proxy,
		},
		Feishu: config.FeishuConfig{
			Enabled:           c.Feishu.Enabled,
			AppID:             c.Feishu.AppID,
			AppSecret:         c.Feishu.AppSecret,
			EncryptKey:        c.Feishu.EncryptKey,
			VerificationToken: c.Feishu.VerificationToken,
		},
		Discord: config.DiscordConfig{
			Enabled:     c.Discord.Enabled,
			Token:       c.Discord.Token,
			MentionOnly: c.Discord.MentionOnly,
		},
		MaixCam: config.MaixCamConfig{
			Enabled: c.MaixCam.Enabled,
			Host:    c.MaixCam.Host,
			Port:    c.MaixCam.Port,
		},
		QQ: config.QQConfig{
			Enabled:   c.QQ.Enabled,
			AppID:     c.QQ.AppID,
			AppSecret: c.QQ.AppSecret,
		},
		DingTalk: config.DingTalkConfig{
			Enabled:      c.DingTalk.Enabled,
			ClientID:     c.DingTalk.ClientID,
			ClientSecret: c.DingTalk.ClientSecret,
		},
		Slack: config.SlackConfig{
			Enabled:  c.Slack.Enabled,
			BotToken: c.Slack.BotToken,
			AppToken: c.Slack.AppToken,
		},
		LINE: config.LINEConfig{
			Enabled:            c.LINE.Enabled,
			ChannelSecret:      c.LINE.ChannelSecret,
			ChannelAccessToken: c.LINE.ChannelAccessToken,
			WebhookHost:        c.LINE.WebhookHost,
			WebhookPort:        c.LINE.WebhookPort,
			WebhookPath:        c.LINE.WebhookPath,
		},
	}
}

func (c GatewayConfig) ToStandardGateway() config.GatewayConfig {
	return config.GatewayConfig{
		Host: c.Host,
		Port: c.Port,
	}
}

func (c ToolsConfig) ToStandardTools() config.ToolsConfig {
	return config.ToolsConfig{
		Web: config.WebToolsConfig{
			Brave: config.BraveConfig{
				Enabled:    c.Web.Brave.Enabled,
				APIKey:     c.Web.Brave.APIKey,
				MaxResults: c.Web.Brave.MaxResults,
			},
			Tavily: config.TavilyConfig{
				Enabled:    c.Web.Tavily.Enabled,
				APIKey:     c.Web.Tavily.APIKey,
				BaseURL:    c.Web.Tavily.BaseURL,
				MaxResults: c.Web.Tavily.MaxResults,
			},
			DuckDuckGo: config.DuckDuckGoConfig{
				Enabled:    c.Web.DuckDuckGo.Enabled,
				MaxResults: c.Web.DuckDuckGo.MaxResults,
			},
			Perplexity: config.PerplexityConfig{
				Enabled:    c.Web.Perplexity.Enabled,
				APIKey:     c.Web.Perplexity.APIKey,
				MaxResults: c.Web.Perplexity.MaxResults,
			},
			Proxy: c.Web.Proxy,
		},
		Cron: config.CronToolsConfig{
			ExecTimeoutMinutes: c.Cron.ExecTimeoutMinutes,
		},
		Exec: config.ExecConfig{
			EnableDenyPatterns: c.Exec.EnableDenyPatterns,
			CustomDenyPatterns: c.Exec.CustomDenyPatterns,
		},
	}
}
