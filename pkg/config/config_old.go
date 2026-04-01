// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package config

import (
	"encoding/json"
)

type agentDefaultsV0 struct {
	Workspace                 string         `json:"workspace"                       env:"PICOCLAW_AGENTS_DEFAULTS_WORKSPACE"`
	RestrictToWorkspace       bool           `json:"restrict_to_workspace"           env:"PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE"`
	AllowReadOutsideWorkspace bool           `json:"allow_read_outside_workspace"    env:"PICOCLAW_AGENTS_DEFAULTS_ALLOW_READ_OUTSIDE_WORKSPACE"`
	Provider                  string         `json:"provider"                        env:"PICOCLAW_AGENTS_DEFAULTS_PROVIDER"`
	ModelName                 string         `json:"model_name,omitempty"            env:"PICOCLAW_AGENTS_DEFAULTS_MODEL_NAME"`
	Model                     string         `json:"model"                           env:"PICOCLAW_AGENTS_DEFAULTS_MODEL"` // Deprecated: use model_name instead
	ModelFallbacks            []string       `json:"model_fallbacks,omitempty"`
	ImageModel                string         `json:"image_model,omitempty"           env:"PICOCLAW_AGENTS_DEFAULTS_IMAGE_MODEL"`
	ImageModelFallbacks       []string       `json:"image_model_fallbacks,omitempty"`
	MaxTokens                 int            `json:"max_tokens"                      env:"PICOCLAW_AGENTS_DEFAULTS_MAX_TOKENS"`
	Temperature               *float64       `json:"temperature,omitempty"           env:"PICOCLAW_AGENTS_DEFAULTS_TEMPERATURE"`
	MaxToolIterations         int            `json:"max_tool_iterations"             env:"PICOCLAW_AGENTS_DEFAULTS_MAX_TOOL_ITERATIONS"`
	SummarizeMessageThreshold int            `json:"summarize_message_threshold"     env:"PICOCLAW_AGENTS_DEFAULTS_SUMMARIZE_MESSAGE_THRESHOLD"`
	SummarizeTokenPercent     int            `json:"summarize_token_percent"         env:"PICOCLAW_AGENTS_DEFAULTS_SUMMARIZE_TOKEN_PERCENT"`
	MaxMediaSize              int            `json:"max_media_size,omitempty"        env:"PICOCLAW_AGENTS_DEFAULTS_MAX_MEDIA_SIZE"`
	Routing                   *RoutingConfig `json:"routing,omitempty"`
}

// GetModelName returns the effective model name for the agent defaults.
// It prefers the new "model_name" field but falls back to "model" for backward compatibility.
func (d *agentDefaultsV0) GetModelName() string {
	if d.ModelName != "" {
		return d.ModelName
	}
	return d.Model
}

type agentsConfigV0 struct {
	Defaults agentDefaultsV0 `json:"defaults"`
	List     []AgentConfig   `json:"list,omitempty"`
}

// configV0 represents the config structure before versioning was introduced.
// This struct is used for loading legacy config files (version 0).
// It is unexported since it's only used internally for migration.
type configV0 struct {
	Agents    agentsConfigV0    `json:"agents"`
	Bindings  []AgentBinding    `json:"bindings,omitempty"`
	Session   SessionConfig     `json:"session,omitempty"`
	Channels  channelsConfigV0  `json:"channels"`
	Providers providersConfigV0 `json:"providers,omitempty"`
	ModelList []modelConfigV0   `json:"model_list"`
	Gateway   GatewayConfig     `json:"gateway"`
	Tools     toolsConfigV0     `json:"tools"`
	Heartbeat HeartbeatConfig   `json:"heartbeat"`
	Devices   DevicesConfig     `json:"devices"`
}

type toolsConfigV0 struct {
	AllowReadPaths  []string            `json:"allow_read_paths"  env:"PICOCLAW_TOOLS_ALLOW_READ_PATHS"`
	AllowWritePaths []string            `json:"allow_write_paths" env:"PICOCLAW_TOOLS_ALLOW_WRITE_PATHS"`
	Web             webToolsConfigV0    `json:"web"`
	Cron            CronToolsConfig     `json:"cron"`
	Exec            ExecConfig          `json:"exec"`
	Skills          skillsToolsConfigV0 `json:"skills"`
	MediaCleanup    MediaCleanupConfig  `json:"media_cleanup"`
	MCP             MCPConfig           `json:"mcp"`
	AppendFile      ToolConfig          `json:"append_file"                                              envPrefix:"PICOCLAW_TOOLS_APPEND_FILE_"`
	EditFile        ToolConfig          `json:"edit_file"                                                envPrefix:"PICOCLAW_TOOLS_EDIT_FILE_"`
	FindSkills      ToolConfig          `json:"find_skills"                                              envPrefix:"PICOCLAW_TOOLS_FIND_SKILLS_"`
	I2C             ToolConfig          `json:"i2c"                                                      envPrefix:"PICOCLAW_TOOLS_I2C_"`
	InstallSkill    ToolConfig          `json:"install_skill"                                            envPrefix:"PICOCLAW_TOOLS_INSTALL_SKILL_"`
	ListDir         ToolConfig          `json:"list_dir"                                                 envPrefix:"PICOCLAW_TOOLS_LIST_DIR_"`
	Message         ToolConfig          `json:"message"                                                  envPrefix:"PICOCLAW_TOOLS_MESSAGE_"`
	ReadFile        ReadFileToolConfig  `json:"read_file"                                                envPrefix:"PICOCLAW_TOOLS_READ_FILE_"`
	SendFile        ToolConfig          `json:"send_file"                                                envPrefix:"PICOCLAW_TOOLS_SEND_FILE_"`
	Spawn           ToolConfig          `json:"spawn"                                                    envPrefix:"PICOCLAW_TOOLS_SPAWN_"`
	SpawnStatus     ToolConfig          `json:"spawn_status"                                             envPrefix:"PICOCLAW_TOOLS_SPAWN_STATUS_"`
	SPI             ToolConfig          `json:"spi"                                                      envPrefix:"PICOCLAW_TOOLS_SPI_"`
	Subagent        ToolConfig          `json:"subagent"                                                 envPrefix:"PICOCLAW_TOOLS_SUBAGENT_"`
	WebFetch        ToolConfig          `json:"web_fetch"                                                envPrefix:"PICOCLAW_TOOLS_WEB_FETCH_"`
	WriteFile       ToolConfig          `json:"write_file"                                               envPrefix:"PICOCLAW_TOOLS_WRITE_FILE_"`
}

type channelsConfigV0 struct {
	WhatsApp WhatsAppConfig   `json:"whatsapp"`
	Telegram telegramConfigV0 `json:"telegram"`
	Feishu   feishuConfigV0   `json:"feishu"`
	Discord  discordConfigV0  `json:"discord"`
	MaixCam  maixcamConfigV0  `json:"maixcam"`
	Weixin   weixinConfigV0   `json:"weixin"`
	QQ       qqConfigV0       `json:"qq"`
	DingTalk dingtalkConfigV0 `json:"dingtalk"`
	Slack    slackConfigV0    `json:"slack"`
	Matrix   matrixConfigV0   `json:"matrix"`
	LINE     lineConfigV0     `json:"line"`
	OneBot   onebotConfigV0   `json:"onebot"`
	WeCom    wecomConfigV0    `json:"wecom"    envPrefix:"PICOCLAW_CHANNELS_WECOM_"`
	Pico     picoConfigV0     `json:"pico"`
	IRC      ircConfigV0      `json:"irc"`
}

func (v *channelsConfigV0) ToChannelsConfig() ChannelsConfig {
	telegram := v.Telegram.ToTelegramConfig()
	feishu := v.Feishu.ToFeishuConfig()
	discord := v.Discord.ToDiscordConfig()
	maixcam := v.MaixCam.ToMaixCamConfig()
	qq := v.QQ.ToQQConfig()
	weixin := v.Weixin.ToWeiXinConfig()
	dingtalk := v.DingTalk.ToDingTalkConfig()
	slack := v.Slack.ToSlackConfig()
	matrix := v.Matrix.ToMatrixConfig()
	line := v.LINE.ToLINEConfig()
	onebot := v.OneBot.ToOneBotConfig()
	wecom := v.WeCom.ToWeComConfig()
	pico := v.Pico.ToPicoConfig()
	irc := v.IRC.ToIRCConfig()

	return ChannelsConfig{
		WhatsApp: v.WhatsApp,
		Telegram: telegram,
		Feishu:   feishu,
		Discord:  discord,
		MaixCam:  maixcam,
		QQ:       qq,
		Weixin:   weixin,
		DingTalk: dingtalk,
		Slack:    slack,
		Matrix:   matrix,
		LINE:     line,
		OneBot:   onebot,
		WeCom:    wecom,
		Pico:     pico,
		IRC:      irc,
	}
}

type qqConfigV0 struct {
	Enabled              bool                `json:"enabled"                  env:"PICOCLAW_CHANNELS_QQ_ENABLED"`
	AppID                string              `json:"app_id"                   env:"PICOCLAW_CHANNELS_QQ_APP_ID"`
	AppSecret            string              `json:"app_secret"               env:"PICOCLAW_CHANNELS_QQ_APP_SECRET"`
	AllowFrom            FlexibleStringSlice `json:"allow_from"               env:"PICOCLAW_CHANNELS_QQ_ALLOW_FROM"`
	GroupTrigger         GroupTriggerConfig  `json:"group_trigger,omitempty"`
	MaxMessageLength     int                 `json:"max_message_length"       env:"PICOCLAW_CHANNELS_QQ_MAX_MESSAGE_LENGTH"`
	MaxBase64FileSizeMiB int64               `json:"max_base64_file_size_mib" env:"PICOCLAW_CHANNELS_QQ_MAX_BASE64_FILE_SIZE_MIB"`
	SendMarkdown         bool                `json:"send_markdown"            env:"PICOCLAW_CHANNELS_QQ_SEND_MARKDOWN"`
	ReasoningChannelID   string              `json:"reasoning_channel_id"     env:"PICOCLAW_CHANNELS_QQ_REASONING_CHANNEL_ID"`
}

func (v *qqConfigV0) ToQQConfig() QQConfig {
	return QQConfig{
		Enabled:              v.Enabled,
		AppID:                v.AppID,
		AllowFrom:            v.AllowFrom,
		GroupTrigger:         v.GroupTrigger,
		MaxMessageLength:     v.MaxMessageLength,
		MaxBase64FileSizeMiB: v.MaxBase64FileSizeMiB,
		SendMarkdown:         v.SendMarkdown,
		ReasoningChannelID:   v.ReasoningChannelID,
		AppSecret:            *NewSecureString(v.AppSecret),
	}
}

type telegramConfigV0 struct {
	Enabled            bool                `json:"enabled"                 env:"PICOCLAW_CHANNELS_TELEGRAM_ENABLED"`
	Token              string              `json:"token"                   env:"PICOCLAW_CHANNELS_TELEGRAM_TOKEN"`
	BaseURL            string              `json:"base_url"                env:"PICOCLAW_CHANNELS_TELEGRAM_BASE_URL"`
	Proxy              string              `json:"proxy"                   env:"PICOCLAW_CHANNELS_TELEGRAM_PROXY"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_TELEGRAM_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_TELEGRAM_REASONING_CHANNEL_ID"`
	UseMarkdownV2      bool                `json:"use_markdown_v2"         env:"PICOCLAW_CHANNELS_TELEGRAM_USE_MARKDOWN_V2"`
}

func (v *telegramConfigV0) ToTelegramConfig() TelegramConfig {
	cfg := TelegramConfig{
		Enabled:            v.Enabled,
		BaseURL:            v.BaseURL,
		Proxy:              v.Proxy,
		AllowFrom:          v.AllowFrom,
		GroupTrigger:       v.GroupTrigger,
		Typing:             v.Typing,
		Placeholder:        v.Placeholder,
		ReasoningChannelID: v.ReasoningChannelID,
		UseMarkdownV2:      v.UseMarkdownV2,
	}
	if v.Token != "" {
		cfg.Token = *NewSecureString(v.Token)
	}
	return cfg
}

type feishuConfigV0 struct {
	Enabled             bool                `json:"enabled"                 env:"PICOCLAW_CHANNELS_FEISHU_ENABLED"`
	AppID               string              `json:"app_id"                  env:"PICOCLAW_CHANNELS_FEISHU_APP_ID"`
	AppSecret           string              `json:"app_secret"              env:"PICOCLAW_CHANNELS_FEISHU_APP_SECRET"`
	EncryptKey          string              `json:"encrypt_key"             env:"PICOCLAW_CHANNELS_FEISHU_ENCRYPT_KEY"`
	VerificationToken   string              `json:"verification_token"      env:"PICOCLAW_CHANNELS_FEISHU_VERIFICATION_TOKEN"`
	AllowFrom           FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_FEISHU_ALLOW_FROM"`
	GroupTrigger        GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Placeholder         PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID  string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_FEISHU_REASONING_CHANNEL_ID"`
	RandomReactionEmoji FlexibleStringSlice `json:"random_reaction_emoji"   env:"PICOCLAW_CHANNELS_FEISHU_RANDOM_REACTION_EMOJI"`
	IsLark              bool                `json:"is_lark"                 env:"PICOCLAW_CHANNELS_FEISHU_IS_LARK"`
}

func (v *feishuConfigV0) ToFeishuConfig() FeishuConfig {
	cfg := FeishuConfig{
		Enabled:            v.Enabled,
		AppID:              v.AppID,
		AllowFrom:          v.AllowFrom,
		GroupTrigger:       v.GroupTrigger,
		Placeholder:        v.Placeholder,
		ReasoningChannelID: v.ReasoningChannelID,
	}
	if v.AppSecret != "" {
		cfg.AppSecret = *NewSecureString(v.AppSecret)
	}
	if v.EncryptKey != "" {
		cfg.EncryptKey = *NewSecureString(v.EncryptKey)
	}
	if v.VerificationToken != "" {
		cfg.VerificationToken = *NewSecureString(v.VerificationToken)
	}
	return cfg
}

type discordConfigV0 struct {
	Enabled            bool                `json:"enabled"                 env:"PICOCLAW_CHANNELS_DISCORD_ENABLED"`
	Token              string              `json:"token"                   env:"PICOCLAW_CHANNELS_DISCORD_TOKEN"`
	Proxy              string              `json:"proxy"                   env:"PICOCLAW_CHANNELS_DISCORD_PROXY"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_DISCORD_ALLOW_FROM"`
	MentionOnly        bool                `json:"mention_only"            env:"PICOCLAW_CHANNELS_DISCORD_MENTION_ONLY"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_DISCORD_REASONING_CHANNEL_ID"`
}

func (v *discordConfigV0) ToDiscordConfig() DiscordConfig {
	cfg := DiscordConfig{
		Enabled:            v.Enabled,
		Proxy:              v.Proxy,
		AllowFrom:          v.AllowFrom,
		MentionOnly:        v.MentionOnly,
		GroupTrigger:       v.GroupTrigger,
		Typing:             v.Typing,
		Placeholder:        v.Placeholder,
		ReasoningChannelID: v.ReasoningChannelID,
	}
	if v.Token != "" {
		cfg.Token = *NewSecureString(v.Token)
	}
	return cfg
}

type maixcamConfigV0 struct {
	Enabled            bool                `json:"enabled"              env:"PICOCLAW_CHANNELS_MAIXCAM_ENABLED"`
	Host               string              `json:"host"                 env:"PICOCLAW_CHANNELS_MAIXCAM_HOST"`
	Port               int                 `json:"port"                 env:"PICOCLAW_CHANNELS_MAIXCAM_PORT"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"           env:"PICOCLAW_CHANNELS_MAIXCAM_ALLOW_FROM"`
	ReasoningChannelID string              `json:"reasoning_channel_id" env:"PICOCLAW_CHANNELS_MAIXCAM_REASONING_CHANNEL_ID"`
}

func (v *maixcamConfigV0) ToMaixCamConfig() MaixCamConfig {
	return MaixCamConfig{
		Enabled:            v.Enabled,
		Host:               v.Host,
		Port:               v.Port,
		AllowFrom:          v.AllowFrom,
		ReasoningChannelID: v.ReasoningChannelID,
	}
}

type dingtalkConfigV0 struct {
	Enabled            bool                `json:"enabled"                 env:"PICOCLAW_CHANNELS_DINGTALK_ENABLED"`
	ClientID           string              `json:"client_id"               env:"PICOCLAW_CHANNELS_DINGTALK_CLIENT_ID"`
	ClientSecret       string              `json:"client_secret"           env:"PICOCLAW_CHANNELS_DINGTALK_CLIENT_SECRET"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_DINGTALK_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_DINGTALK_REASONING_CHANNEL_ID"`
}

func (v *dingtalkConfigV0) ToDingTalkConfig() DingTalkConfig {
	cfg := DingTalkConfig{
		Enabled:            v.Enabled,
		ClientID:           v.ClientID,
		AllowFrom:          v.AllowFrom,
		GroupTrigger:       v.GroupTrigger,
		ReasoningChannelID: v.ReasoningChannelID,
	}
	if v.ClientSecret != "" {
		cfg.ClientSecret = *NewSecureString(v.ClientSecret)
	}
	return cfg
}

type slackConfigV0 struct {
	Enabled            bool                `json:"enabled"                 env:"PICOCLAW_CHANNELS_SLACK_ENABLED"`
	BotToken           string              `json:"bot_token"               env:"PICOCLAW_CHANNELS_SLACK_BOT_TOKEN"`
	AppToken           string              `json:"app_token"               env:"PICOCLAW_CHANNELS_SLACK_APP_TOKEN"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_SLACK_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_SLACK_REASONING_CHANNEL_ID"`
}

func (v *slackConfigV0) ToSlackConfig() SlackConfig {
	cfg := SlackConfig{
		Enabled:            v.Enabled,
		AllowFrom:          v.AllowFrom,
		GroupTrigger:       v.GroupTrigger,
		Typing:             v.Typing,
		Placeholder:        v.Placeholder,
		ReasoningChannelID: v.ReasoningChannelID,
	}
	if v.BotToken != "" {
		cfg.BotToken = *NewSecureString(v.BotToken)
	}
	if v.AppToken != "" {
		cfg.AppToken = *NewSecureString(v.AppToken)
	}
	return cfg
}

type matrixConfigV0 struct {
	Enabled            bool                `json:"enabled"                  env:"PICOCLAW_CHANNELS_MATRIX_ENABLED"`
	Homeserver         string              `json:"homeserver"               env:"PICOCLAW_CHANNELS_MATRIX_HOMESERVER"`
	UserID             string              `json:"user_id"                  env:"PICOCLAW_CHANNELS_MATRIX_USER_ID"`
	AccessToken        string              `json:"access_token"             env:"PICOCLAW_CHANNELS_MATRIX_ACCESS_TOKEN"`
	DeviceID           string              `json:"device_id,omitempty"      env:"PICOCLAW_CHANNELS_MATRIX_DEVICE_ID"`
	JoinOnInvite       bool                `json:"join_on_invite"           env:"PICOCLAW_CHANNELS_MATRIX_JOIN_ON_INVITE"`
	MessageFormat      string              `json:"message_format,omitempty" env:"PICOCLAW_CHANNELS_MATRIX_MESSAGE_FORMAT"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"               env:"PICOCLAW_CHANNELS_MATRIX_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"     env:"PICOCLAW_CHANNELS_MATRIX_REASONING_CHANNEL_ID"`
}

func (v *matrixConfigV0) ToMatrixConfig() MatrixConfig {
	cfg := MatrixConfig{
		Enabled:            v.Enabled,
		Homeserver:         v.Homeserver,
		UserID:             v.UserID,
		DeviceID:           v.DeviceID,
		JoinOnInvite:       v.JoinOnInvite,
		MessageFormat:      v.MessageFormat,
		AllowFrom:          v.AllowFrom,
		GroupTrigger:       v.GroupTrigger,
		Placeholder:        v.Placeholder,
		ReasoningChannelID: v.ReasoningChannelID,
	}
	if v.AccessToken != "" {
		cfg.AccessToken = *NewSecureString(v.AccessToken)
	}
	return cfg
}

type lineConfigV0 struct {
	Enabled            bool                `json:"enabled"                 env:"PICOCLAW_CHANNELS_LINE_ENABLED"`
	ChannelSecret      string              `json:"channel_secret"          env:"PICOCLAW_CHANNELS_LINE_CHANNEL_SECRET"`
	ChannelAccessToken string              `json:"channel_access_token"    env:"PICOCLAW_CHANNELS_LINE_CHANNEL_ACCESS_TOKEN"`
	WebhookHost        string              `json:"webhook_host"            env:"PICOCLAW_CHANNELS_LINE_WEBHOOK_HOST"`
	WebhookPort        int                 `json:"webhook_port"            env:"PICOCLAW_CHANNELS_LINE_WEBHOOK_PORT"`
	WebhookPath        string              `json:"webhook_path"            env:"PICOCLAW_CHANNELS_LINE_WEBHOOK_PATH"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_LINE_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_LINE_REASONING_CHANNEL_ID"`
}

func (v *lineConfigV0) ToLINEConfig() LINEConfig {
	cfg := LINEConfig{
		Enabled:            v.Enabled,
		WebhookHost:        v.WebhookHost,
		WebhookPort:        v.WebhookPort,
		WebhookPath:        v.WebhookPath,
		AllowFrom:          v.AllowFrom,
		GroupTrigger:       v.GroupTrigger,
		Typing:             v.Typing,
		Placeholder:        v.Placeholder,
		ReasoningChannelID: v.ReasoningChannelID,
	}
	if v.ChannelSecret != "" {
		cfg.ChannelSecret = *NewSecureString(v.ChannelSecret)
	}
	if v.ChannelAccessToken != "" {
		cfg.ChannelAccessToken = *NewSecureString(v.ChannelAccessToken)
	}
	return cfg
}

type onebotConfigV0 struct {
	Enabled            bool                `json:"enabled"                 env:"PICOCLAW_CHANNELS_ONEBOT_ENABLED"`
	WSUrl              string              `json:"ws_url"                  env:"PICOCLAW_CHANNELS_ONEBOT_WS_URL"`
	AccessToken        string              `json:"access_token"            env:"PICOCLAW_CHANNELS_ONEBOT_ACCESS_TOKEN"`
	ReconnectInterval  int                 `json:"reconnect_interval"      env:"PICOCLAW_CHANNELS_ONEBOT_RECONNECT_INTERVAL"`
	GroupTriggerPrefix []string            `json:"group_trigger_prefix"    env:"PICOCLAW_CHANNELS_ONEBOT_GROUP_TRIGGER_PREFIX"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_ONEBOT_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	Placeholder        PlaceholderConfig   `json:"placeholder,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_ONEBOT_REASONING_CHANNEL_ID"`
}

func (v *onebotConfigV0) ToOneBotConfig() OneBotConfig {
	cfg := OneBotConfig{
		Enabled:            v.Enabled,
		WSUrl:              v.WSUrl,
		ReconnectInterval:  v.ReconnectInterval,
		GroupTriggerPrefix: v.GroupTriggerPrefix,
		AllowFrom:          v.AllowFrom,
		GroupTrigger:       v.GroupTrigger,
		Typing:             v.Typing,
		Placeholder:        v.Placeholder,
		ReasoningChannelID: v.ReasoningChannelID,
	}
	if v.AccessToken != "" {
		cfg.AccessToken = *NewSecureString(v.AccessToken)
	}
	return cfg
}

type wecomConfigV0 struct {
	Enabled             bool                        `json:"enabled"                    env:"ENABLED"`
	BotID               string                      `json:"bot_id"                     env:"BOT_ID"`
	Secret              string                      `json:"secret"                     env:"SECRET"`
	WebSocketURL        string                      `json:"websocket_url,omitempty"    env:"WEBSOCKET_URL"`
	SendThinkingMessage bool                        `json:"send_thinking_message"      env:"SEND_THINKING_MESSAGE"`
	DMPolicy            string                      `json:"dm_policy,omitempty"        env:"DM_POLICY"`
	AllowFrom           FlexibleStringSlice         `json:"allow_from"                 env:"ALLOW_FROM"`
	GroupPolicy         string                      `json:"group_policy,omitempty"     env:"GROUP_POLICY"`
	GroupAllowFrom      FlexibleStringSlice         `json:"group_allow_from,omitempty" env:"GROUP_ALLOW_FROM"`
	Groups              map[string]WeComGroupConfig `json:"groups,omitempty"`
	ReasoningChannelID  string                      `json:"reasoning_channel_id"       env:"REASONING_CHANNEL_ID"`
}

func (v *wecomConfigV0) ToWeComConfig() WeComConfig {
	cfg := WeComConfig{
		Enabled:             v.Enabled,
		BotID:               v.BotID,
		WebSocketURL:        v.WebSocketURL,
		SendThinkingMessage: v.SendThinkingMessage,
		AllowFrom:           v.AllowFrom,
		ReasoningChannelID:  v.ReasoningChannelID,
	}
	if v.Secret != "" {
		cfg.Secret = *NewSecureString(v.Secret)
	}
	return cfg
}

type weixinConfigV0 struct {
	Enabled            bool                `json:"enabled"              env:"PICOCLAW_CHANNELS_WEIXIN_ENABLED"`
	Token              string              `json:"token"                env:"PICOCLAW_CHANNELS_WEIXIN_TOKEN"`
	BaseURL            string              `json:"base_url"             env:"PICOCLAW_CHANNELS_WEIXIN_BASE_URL"`
	CDNBaseURL         string              `json:"cdn_base_url"         env:"PICOCLAW_CHANNELS_WEIXIN_CDN_BASE_URL"`
	Proxy              string              `json:"proxy"                env:"PICOCLAW_CHANNELS_WEIXIN_PROXY"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"           env:"PICOCLAW_CHANNELS_WEIXIN_ALLOW_FROM"`
	ReasoningChannelID string              `json:"reasoning_channel_id" env:"PICOCLAW_CHANNELS_WEIXIN_REASONING_CHANNEL_ID"`
}

func (v *weixinConfigV0) ToWeiXinConfig() WeixinConfig {
	cfg := WeixinConfig{
		Enabled:            v.Enabled,
		BaseURL:            v.BaseURL,
		CDNBaseURL:         v.CDNBaseURL,
		Proxy:              v.Proxy,
		AllowFrom:          v.AllowFrom,
		ReasoningChannelID: v.ReasoningChannelID,
	}
	if v.Token != "" {
		cfg.Token = *NewSecureString(v.Token)
	}
	return cfg
}

type picoConfigV0 struct {
	Enabled         bool                `json:"enabled"                     env:"PICOCLAW_CHANNELS_PICO_ENABLED"`
	Token           string              `json:"token"                       env:"PICOCLAW_CHANNELS_PICO_TOKEN"`
	AllowTokenQuery bool                `json:"allow_token_query,omitempty"`
	AllowOrigins    []string            `json:"allow_origins,omitempty"`
	PingInterval    int                 `json:"ping_interval,omitempty"`
	ReadTimeout     int                 `json:"read_timeout,omitempty"`
	WriteTimeout    int                 `json:"write_timeout,omitempty"`
	MaxConnections  int                 `json:"max_connections,omitempty"`
	AllowFrom       FlexibleStringSlice `json:"allow_from"                  env:"PICOCLAW_CHANNELS_PICO_ALLOW_FROM"`
	Placeholder     PlaceholderConfig   `json:"placeholder,omitempty"`
}

func (v *picoConfigV0) ToPicoConfig() PicoConfig {
	cfg := PicoConfig{
		Enabled:         v.Enabled,
		AllowTokenQuery: v.AllowTokenQuery,
		AllowOrigins:    v.AllowOrigins,
		PingInterval:    v.PingInterval,
		ReadTimeout:     v.ReadTimeout,
		WriteTimeout:    v.WriteTimeout,
		MaxConnections:  v.MaxConnections,
		AllowFrom:       v.AllowFrom,
		Placeholder:     v.Placeholder,
	}
	if v.Token != "" {
		cfg.Token = *NewSecureString(v.Token)
	}
	return cfg
}

type ircConfigV0 struct {
	Enabled            bool                `json:"enabled"                 env:"PICOCLAW_CHANNELS_IRC_ENABLED"`
	Server             string              `json:"server"                  env:"PICOCLAW_CHANNELS_IRC_SERVER"`
	TLS                bool                `json:"tls"                     env:"PICOCLAW_CHANNELS_IRC_TLS"`
	Nick               string              `json:"nick"                    env:"PICOCLAW_CHANNELS_IRC_NICK"`
	User               string              `json:"user,omitempty"          env:"PICOCLAW_CHANNELS_IRC_USER"`
	RealName           string              `json:"real_name,omitempty"     env:"PICOCLAW_CHANNELS_IRC_REAL_NAME"`
	Password           string              `json:"password"                env:"PICOCLAW_CHANNELS_IRC_PASSWORD"`
	NickServPassword   string              `json:"nickserv_password"       env:"PICOCLAW_CHANNELS_IRC_NICKSERV_PASSWORD"`
	SASLUser           string              `json:"sasl_user"               env:"PICOCLAW_CHANNELS_IRC_SASL_USER"`
	SASLPassword       string              `json:"sasl_password"           env:"PICOCLAW_CHANNELS_IRC_SASL_PASSWORD"`
	Channels           FlexibleStringSlice `json:"channels"                env:"PICOCLAW_CHANNELS_IRC_CHANNELS"`
	RequestCaps        FlexibleStringSlice `json:"request_caps,omitempty"  env:"PICOCLAW_CHANNELS_IRC_REQUEST_CAPS"`
	AllowFrom          FlexibleStringSlice `json:"allow_from"              env:"PICOCLAW_CHANNELS_IRC_ALLOW_FROM"`
	GroupTrigger       GroupTriggerConfig  `json:"group_trigger,omitempty"`
	Typing             TypingConfig        `json:"typing,omitempty"`
	ReasoningChannelID string              `json:"reasoning_channel_id"    env:"PICOCLAW_CHANNELS_IRC_REASONING_CHANNEL_ID"`
}

func (v *ircConfigV0) ToIRCConfig() IRCConfig {
	cfg := IRCConfig{
		Enabled:            v.Enabled,
		Server:             v.Server,
		TLS:                v.TLS,
		Nick:               v.Nick,
		User:               v.User,
		RealName:           v.RealName,
		SASLUser:           v.SASLUser,
		Channels:           v.Channels,
		RequestCaps:        v.RequestCaps,
		AllowFrom:          v.AllowFrom,
		GroupTrigger:       v.GroupTrigger,
		Typing:             v.Typing,
		ReasoningChannelID: v.ReasoningChannelID,
	}
	if v.Password != "" {
		cfg.Password = *NewSecureString(v.Password)
	}
	if v.NickServPassword != "" {
		cfg.NickServPassword = *NewSecureString(v.NickServPassword)
	}
	if v.SASLPassword != "" {
		cfg.SASLPassword = *NewSecureString(v.SASLPassword)
	}
	return cfg
}

type providersConfigV0 struct {
	Anthropic     providerConfigV0       `json:"anthropic"`
	OpenAI        openAIProviderConfigV0 `json:"openai"`
	LiteLLM       providerConfigV0       `json:"litellm"`
	OpenRouter    providerConfigV0       `json:"openrouter"`
	Groq          providerConfigV0       `json:"groq"`
	Zhipu         providerConfigV0       `json:"zhipu"`
	VLLM          providerConfigV0       `json:"vllm"`
	Gemini        providerConfigV0       `json:"gemini"`
	Nvidia        providerConfigV0       `json:"nvidia"`
	Ollama        providerConfigV0       `json:"ollama"`
	Moonshot      providerConfigV0       `json:"moonshot"`
	ShengSuanYun  providerConfigV0       `json:"shengsuanyun"`
	DeepSeek      providerConfigV0       `json:"deepseek"`
	Cerebras      providerConfigV0       `json:"cerebras"`
	Vivgrid       providerConfigV0       `json:"vivgrid"`
	VolcEngine    providerConfigV0       `json:"volcengine"`
	GitHubCopilot providerConfigV0       `json:"github_copilot"`
	Antigravity   providerConfigV0       `json:"antigravity"`
	Qwen          providerConfigV0       `json:"qwen"`
	Mistral       providerConfigV0       `json:"mistral"`
	Avian         providerConfigV0       `json:"avian"`
	Minimax       providerConfigV0       `json:"minimax"`
	LongCat       providerConfigV0       `json:"longcat"`
	ModelScope    providerConfigV0       `json:"modelscope"`
	Novita        providerConfigV0       `json:"novita"`
}

// IsEmpty checks if all provider configs are empty (no API keys or API bases set)
// Note: WebSearch is an optimization option and doesn't count as "non-empty"
func (p providersConfigV0) IsEmpty() bool {
	return p.Anthropic.APIKey == "" && p.Anthropic.APIBase == "" &&
		p.OpenAI.APIKey == "" && p.OpenAI.APIBase == "" &&
		p.LiteLLM.APIKey == "" && p.LiteLLM.APIBase == "" &&
		p.OpenRouter.APIKey == "" && p.OpenRouter.APIBase == "" &&
		p.Groq.APIKey == "" && p.Groq.APIBase == "" &&
		p.Zhipu.APIKey == "" && p.Zhipu.APIBase == "" &&
		p.VLLM.APIKey == "" && p.VLLM.APIBase == "" &&
		p.Gemini.APIKey == "" && p.Gemini.APIBase == "" &&
		p.Nvidia.APIKey == "" && p.Nvidia.APIBase == "" &&
		p.Ollama.APIKey == "" && p.Ollama.APIBase == "" &&
		p.Moonshot.APIKey == "" && p.Moonshot.APIBase == "" &&
		p.ShengSuanYun.APIKey == "" && p.ShengSuanYun.APIBase == "" &&
		p.DeepSeek.APIKey == "" && p.DeepSeek.APIBase == "" &&
		p.Cerebras.APIKey == "" && p.Cerebras.APIBase == "" &&
		p.Vivgrid.APIKey == "" && p.Vivgrid.APIBase == "" &&
		p.VolcEngine.APIKey == "" && p.VolcEngine.APIBase == "" &&
		p.GitHubCopilot.APIKey == "" && p.GitHubCopilot.APIBase == "" &&
		p.Antigravity.APIKey == "" && p.Antigravity.APIBase == "" &&
		p.Qwen.APIKey == "" && p.Qwen.APIBase == "" &&
		p.Mistral.APIKey == "" && p.Mistral.APIBase == "" &&
		p.Avian.APIKey == "" && p.Avian.APIBase == "" &&
		p.Minimax.APIKey == "" && p.Minimax.APIBase == "" &&
		p.LongCat.APIKey == "" && p.LongCat.APIBase == "" &&
		p.ModelScope.APIKey == "" && p.ModelScope.APIBase == "" &&
		p.Novita.APIKey == "" && p.Novita.APIBase == ""
}

type providerConfigV0 struct {
	APIKey         string `json:"api_key"                   env:"PICOCLAW_PROVIDERS_{{.Name}}_API_KEY"`
	APIBase        string `json:"api_base"                  env:"PICOCLAW_PROVIDERS_{{.Name}}_API_BASE"`
	Proxy          string `json:"proxy,omitempty"           env:"PICOCLAW_PROVIDERS_{{.Name}}_PROXY"`
	RequestTimeout int    `json:"request_timeout,omitempty" env:"PICOCLAW_PROVIDERS_{{.Name}}_REQUEST_TIMEOUT"`
	AuthMethod     string `json:"auth_method,omitempty"     env:"PICOCLAW_PROVIDERS_{{.Name}}_AUTH_METHOD"`
	ConnectMode    string `json:"connect_mode,omitempty"    env:"PICOCLAW_PROVIDERS_{{.Name}}_CONNECT_MODE"` // only for Github Copilot, `stdio` or `grpc`
}

// MarshalJSON implements custom JSON marshaling for providersConfig
// to omit the entire section when empty
func (p providersConfigV0) MarshalJSON() ([]byte, error) {
	if p.IsEmpty() {
		return []byte("null"), nil
	}
	type Alias providersConfigV0
	return json.Marshal((*Alias)(&p))
}

type openAIProviderConfigV0 struct {
	providerConfigV0
	WebSearch bool `json:"web_search" env:"PICOCLAW_PROVIDERS_OPENAI_WEB_SEARCH"`
}

type modelConfigV0 struct {
	// Required fields
	ModelName string `json:"model_name"` // User-facing alias for the model
	Model     string `json:"model"`      // Protocol/model-identifier (e.g., "openai/gpt-4o", "anthropic/claude-sonnet-4.6")

	// HTTP-based providers
	APIBase   string   `json:"api_base,omitempty"`  // API endpoint URL
	APIKey    string   `json:"api_key"`             // API authentication key (single key)
	APIKeys   []string `json:"api_keys,omitempty"`  // API authentication keys (multiple keys for failover)
	Proxy     string   `json:"proxy,omitempty"`     // HTTP proxy URL
	Fallbacks []string `json:"fallbacks,omitempty"` // Fallback model names for failover

	// Special providers (CLI-based, OAuth, etc.)
	AuthMethod  string `json:"auth_method,omitempty"`  // Authentication method: oauth, token
	ConnectMode string `json:"connect_mode,omitempty"` // Connection mode: stdio, grpc
	Workspace   string `json:"workspace,omitempty"`    // Workspace path for CLI-based providers

	// Optional optimizations
	RPM            int    `json:"rpm,omitempty"`              // Requests per minute limit
	MaxTokensField string `json:"max_tokens_field,omitempty"` // Field name for max tokens (e.g., "max_completion_tokens")
	RequestTimeout int    `json:"request_timeout,omitempty"`
	ThinkingLevel  string `json:"thinking_level,omitempty"` // Extended thinking: off|low|medium|high|xhigh|adaptive
}

func (c *configV0) migrateChannelConfigs() {
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

func (c *configV0) Migrate() (*Config, error) {
	// Migrate legacy channel config fields to new unified structures
	cfg := DefaultConfig()

	// Always copy user's Agents config to preserve settings like Provider, Model, MaxTokens
	cfg.Agents.List = c.Agents.List
	cfg.Agents.Defaults.Workspace = c.Agents.Defaults.Workspace
	cfg.Agents.Defaults.RestrictToWorkspace = c.Agents.Defaults.RestrictToWorkspace
	cfg.Agents.Defaults.AllowReadOutsideWorkspace = c.Agents.Defaults.AllowReadOutsideWorkspace
	cfg.Agents.Defaults.Provider = c.Agents.Defaults.Provider
	cfg.Agents.Defaults.ModelName = c.Agents.Defaults.GetModelName()
	cfg.Agents.Defaults.ModelFallbacks = c.Agents.Defaults.ModelFallbacks
	cfg.Agents.Defaults.ImageModel = c.Agents.Defaults.ImageModel
	cfg.Agents.Defaults.ImageModelFallbacks = c.Agents.Defaults.ImageModelFallbacks
	cfg.Agents.Defaults.MaxTokens = c.Agents.Defaults.MaxTokens
	cfg.Agents.Defaults.Temperature = c.Agents.Defaults.Temperature
	cfg.Agents.Defaults.MaxToolIterations = c.Agents.Defaults.MaxToolIterations
	cfg.Agents.Defaults.SummarizeMessageThreshold = c.Agents.Defaults.SummarizeMessageThreshold
	cfg.Agents.Defaults.SummarizeTokenPercent = c.Agents.Defaults.SummarizeTokenPercent
	cfg.Agents.Defaults.MaxMediaSize = c.Agents.Defaults.MaxMediaSize
	cfg.Agents.Defaults.Routing = c.Agents.Defaults.Routing

	// Copy other top-level fields
	cfg.Bindings = c.Bindings
	cfg.Session = c.Session
	cfg.Channels = c.Channels.ToChannelsConfig()
	cfg.Gateway = c.Gateway
	cfg.Tools.Web = c.Tools.Web.ToWebToolsConfig()
	cfg.Tools.Cron = c.Tools.Cron
	cfg.Tools.Exec = c.Tools.Exec
	cfg.Tools.Skills = c.Tools.Skills.ToSkillsToolsConfig()
	cfg.Tools.MediaCleanup = c.Tools.MediaCleanup
	cfg.Tools.MCP = c.Tools.MCP
	cfg.Tools.AppendFile = c.Tools.AppendFile
	cfg.Tools.EditFile = c.Tools.EditFile
	cfg.Tools.FindSkills = c.Tools.FindSkills
	cfg.Tools.I2C = c.Tools.I2C
	cfg.Tools.InstallSkill = c.Tools.InstallSkill
	cfg.Tools.ListDir = c.Tools.ListDir
	cfg.Tools.Message = c.Tools.Message
	cfg.Tools.ReadFile = c.Tools.ReadFile
	cfg.Tools.SendFile = c.Tools.SendFile
	cfg.Tools.Spawn = c.Tools.Spawn
	cfg.Tools.SpawnStatus = c.Tools.SpawnStatus
	cfg.Tools.SPI = c.Tools.SPI
	cfg.Tools.Subagent = c.Tools.Subagent
	cfg.Tools.WebFetch = c.Tools.WebFetch
	cfg.Tools.AllowReadPaths = c.Tools.AllowReadPaths
	cfg.Tools.AllowWritePaths = c.Tools.AllowWritePaths
	cfg.Heartbeat = c.Heartbeat
	cfg.Devices = c.Devices

	if len(c.ModelList) > 0 {
		// Convert []modelConfigV0 to []ModelConfig
		cfg.ModelList = make([]*ModelConfig, len(c.ModelList))
		for i, m := range c.ModelList {
			mergedKeys := toSecureStrings(mergeAPIKeys(m.APIKey, m.APIKeys))
			mc := &ModelConfig{
				ModelName:      m.ModelName,
				Model:          m.Model,
				APIBase:        m.APIBase,
				Proxy:          m.Proxy,
				Fallbacks:      m.Fallbacks,
				AuthMethod:     m.AuthMethod,
				ConnectMode:    m.ConnectMode,
				Workspace:      m.Workspace,
				RPM:            m.RPM,
				MaxTokensField: m.MaxTokensField,
				RequestTimeout: m.RequestTimeout,
				ThinkingLevel:  m.ThinkingLevel,
				APIKeys:        mergedKeys,
			}
			// Infer Enabled during V0→V1 migration
			if len(mergedKeys) > 0 || m.ModelName == "local-model" {
				mc.Enabled = true
			}
			cfg.ModelList[i] = mc
		}
	}

	cfg.Version = CurrentVersion
	return cfg, nil
}

type configV1 struct {
	Config
}

// Migrate applies V1→Current Version migrations to an already-loaded Config.
//
// It must be called AFTER loadSecurityConfig so that API keys (which live in
// the security file) are available for the Enabled inference.
func (c *configV1) Migrate() (*Config, error) {
	c.migrateModelEnabled()
	c.migrateChannelConfigs()
	return &c.Config, nil
}

// migrateModelEnabled infers the Enabled field for models loaded from V1 configs
// that predate the field (JSON where "enabled" is absent).
//
// Rules (only applied when Enabled has not been explicitly set by the user):
//   - Models with API keys are considered enabled.
//   - The reserved "local-model" entry is considered enabled.
func (cfg *configV1) migrateModelEnabled() {
	for _, m := range cfg.ModelList {
		if m.Enabled {
			continue
		}
		if len(m.APIKeys) > 0 || m.ModelName == "local-model" {
			m.Enabled = true
		}
	}
}

// migrateChannelConfigs migrates legacy channel config fields in a V1 Config
// to the new unified structures.
func (cfg *configV1) migrateChannelConfigs() {
	// Discord: mention_only -> group_trigger.mention_only
	if cfg.Channels.Discord.MentionOnly && !cfg.Channels.Discord.GroupTrigger.MentionOnly {
		cfg.Channels.Discord.GroupTrigger.MentionOnly = true
	}

	// OneBot: group_trigger_prefix -> group_trigger.prefixes
	if len(cfg.Channels.OneBot.GroupTriggerPrefix) > 0 &&
		len(cfg.Channels.OneBot.GroupTrigger.Prefixes) == 0 {
		cfg.Channels.OneBot.GroupTrigger.Prefixes = cfg.Channels.OneBot.GroupTriggerPrefix
	}
}

type webToolsConfigV0 struct {
	ToolConfig           `                    envPrefix:"PICOCLAW_TOOLS_WEB_"`
	Brave                braveConfigV0       `                                json:"brave"`
	Tavily               tavilyConfigV0      `                                json:"tavily"`
	DuckDuckGo           DuckDuckGoConfig    `                                json:"duckduckgo"`
	Perplexity           perplexityConfigV0  `                                json:"perplexity"`
	SearXNG              SearXNGConfig       `                                json:"searxng"`
	GLMSearch            glmSearchConfigV0   `                                json:"glm_search"`
	BaiduSearch          baiduSearchConfigV0 `                                json:"baidu_search"`
	PreferNative         bool                `                                json:"prefer_native"                    env:"PICOCLAW_TOOLS_WEB_PREFER_NATIVE"`
	Proxy                string              `                                json:"proxy,omitempty"                  env:"PICOCLAW_TOOLS_WEB_PROXY"`
	FetchLimitBytes      int64               `                                json:"fetch_limit_bytes,omitempty"      env:"PICOCLAW_TOOLS_WEB_FETCH_LIMIT_BYTES"`
	Format               string              `                                json:"format,omitempty"                 env:"PICOCLAW_TOOLS_WEB_FORMAT"`
	PrivateHostWhitelist FlexibleStringSlice `                                json:"private_host_whitelist,omitempty" env:"PICOCLAW_TOOLS_WEB_PRIVATE_HOST_WHITELIST"`
}

type braveConfigV0 struct {
	Enabled    bool     `json:"enabled"     env:"PICOCLAW_TOOLS_WEB_BRAVE_ENABLED"`
	APIKey     string   `json:"api_key"     env:"PICOCLAW_TOOLS_WEB_BRAVE_API_KEY"`
	APIKeys    []string `json:"api_keys"    env:"PICOCLAW_TOOLS_WEB_BRAVE_API_KEYS"`
	MaxResults int      `json:"max_results" env:"PICOCLAW_TOOLS_WEB_BRAVE_MAX_RESULTS"`
}

func toSecureStrings(keys []string) SecureStrings {
	apikeys := make(SecureStrings, len(keys))
	for i, key := range keys {
		apikeys[i] = NewSecureString(key)
	}
	return apikeys
}

func (v *braveConfigV0) ToBraveConfig() BraveConfig {
	return BraveConfig{
		Enabled:    v.Enabled,
		MaxResults: v.MaxResults,
		APIKeys:    toSecureStrings(mergeAPIKeys(v.APIKey, v.APIKeys)),
	}
}

type tavilyConfigV0 struct {
	Enabled    bool     `json:"enabled"     env:"PICOCLAW_TOOLS_WEB_TAVILY_ENABLED"`
	APIKey     string   `json:"api_key"     env:"PICOCLAW_TOOLS_WEB_TAVILY_API_KEY"`
	APIKeys    []string `json:"api_keys"    env:"PICOCLAW_TOOLS_WEB_TAVILY_API_KEYS"`
	BaseURL    string   `json:"base_url"    env:"PICOCLAW_TOOLS_WEB_TAVILY_BASE_URL"`
	MaxResults int      `json:"max_results" env:"PICOCLAW_TOOLS_WEB_TAVILY_MAX_RESULTS"`
}

func (v *tavilyConfigV0) ToTavilyConfig() TavilyConfig {
	return TavilyConfig{
		Enabled:    v.Enabled,
		BaseURL:    v.BaseURL,
		MaxResults: v.MaxResults,
		APIKeys:    toSecureStrings(mergeAPIKeys(v.APIKey, v.APIKeys)),
	}
}

type perplexityConfigV0 struct {
	Enabled    bool     `json:"enabled"     env:"PICOCLAW_TOOLS_WEB_PERPLEXITY_ENABLED"`
	APIKey     string   `json:"api_key"     env:"PICOCLAW_TOOLS_WEB_PERPLEXITY_API_KEY"`
	APIKeys    []string `json:"api_keys"    env:"PICOCLAW_TOOLS_WEB_PERPLEXITY_API_KEYS"`
	MaxResults int      `json:"max_results" env:"PICOCLAW_TOOLS_WEB_PERPLEXITY_MAX_RESULTS"`
}

func (v *perplexityConfigV0) ToPerplexityConfig() PerplexityConfig {
	return PerplexityConfig{
		Enabled:    v.Enabled,
		MaxResults: v.MaxResults,
		APIKeys:    toSecureStrings(mergeAPIKeys(v.APIKey, v.APIKeys)),
	}
}

type glmSearchConfigV0 struct {
	Enabled      bool   `json:"enabled"       env:"PICOCLAW_TOOLS_WEB_GLM_ENABLED"`
	APIKey       string `json:"api_key"       env:"PICOCLAW_TOOLS_WEB_GLM_API_KEY"`
	BaseURL      string `json:"base_url"      env:"PICOCLAW_TOOLS_WEB_GLM_BASE_URL"`
	SearchEngine string `json:"search_engine" env:"PICOCLAW_TOOLS_WEB_GLM_SEARCH_ENGINE"`
}

func (v *glmSearchConfigV0) ToGLMSearchConfig() GLMSearchConfig {
	return GLMSearchConfig{
		Enabled:      v.Enabled,
		APIKey:       *NewSecureString(v.APIKey),
		BaseURL:      v.BaseURL,
		SearchEngine: v.SearchEngine,
	}
}

type baiduSearchConfigV0 struct {
	Enabled    bool   `json:"enabled"     env:"PICOCLAW_TOOLS_WEB_BAIDU_ENABLED"`
	APIKey     string `json:"api_key"     env:"PICOCLAW_TOOLS_WEB_BAIDU_API_KEY"`
	BaseURL    string `json:"base_url"    env:"PICOCLAW_TOOLS_WEB_BAIDU_BASE_URL"`
	MaxResults int    `json:"max_results" env:"PICOCLAW_TOOLS_WEB_BAIDU_MAX_RESULTS"`
}

func (v *baiduSearchConfigV0) ToBaiduSearchConfig() BaiduSearchConfig {
	return BaiduSearchConfig{
		Enabled:    v.Enabled,
		APIKey:     *NewSecureString(v.APIKey),
		BaseURL:    v.BaseURL,
		MaxResults: v.MaxResults,
	}
}

func (v *webToolsConfigV0) ToWebToolsConfig() WebToolsConfig {
	brave := v.Brave.ToBraveConfig()
	tavily := v.Tavily.ToTavilyConfig()
	perplexity := v.Perplexity.ToPerplexityConfig()
	glmSearch := v.GLMSearch.ToGLMSearchConfig()
	baiduSearch := v.BaiduSearch.ToBaiduSearchConfig()

	return WebToolsConfig{
		ToolConfig:           v.ToolConfig,
		Brave:                brave,
		Tavily:               tavily,
		DuckDuckGo:           v.DuckDuckGo,
		Perplexity:           perplexity,
		SearXNG:              v.SearXNG,
		GLMSearch:            glmSearch,
		PreferNative:         v.PreferNative,
		Proxy:                v.Proxy,
		FetchLimitBytes:      v.FetchLimitBytes,
		Format:               v.Format,
		PrivateHostWhitelist: v.PrivateHostWhitelist,
		BaiduSearch:          baiduSearch,
	}
}

type skillsToolsConfigV0 struct {
	ToolConfig            `                         envPrefix:"PICOCLAW_TOOLS_SKILLS_"`
	Registries            skillsRegistriesConfigV0 `                                   json:"registries"`
	Github                skillsGithubConfigV0     `                                   json:"github"`
	MaxConcurrentSearches int                      `                                   json:"max_concurrent_searches" env:"PICOCLAW_TOOLS_SKILLS_MAX_CONCURRENT_SEARCHES"`
	SearchCache           SearchCacheConfig        `                                   json:"search_cache"`
}

type skillsRegistriesConfigV0 struct {
	ClawHub clawHubRegistryConfigV0 `json:"clawhub"`
}

type clawHubRegistryConfigV0 struct {
	Enabled    bool   `json:"enabled"     env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_ENABLED"`
	BaseURL    string `json:"base_url"    env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_BASE_URL"`
	AuthToken  string `json:"auth_token"  env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_AUTH_TOKEN"`
	SearchPath string `json:"search_path" env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_SEARCH_PATH"`
	SkillsPath string `json:"skills_path" env:"PICOCLAW_SKILLS_REGISTRIES_CLAWHUB_SKILLS_PATH"`
}

func (v *clawHubRegistryConfigV0) ToClawHubRegistryConfig() ClawHubRegistryConfig {
	cfg := ClawHubRegistryConfig{
		Enabled:    v.Enabled,
		BaseURL:    v.BaseURL,
		SearchPath: v.SearchPath,
		SkillsPath: v.SkillsPath,
	}
	if v.AuthToken != "" {
		cfg.AuthToken = *NewSecureString(v.AuthToken)
	}
	return cfg
}

type skillsGithubConfigV0 struct {
	Token string `json:"token"           env:"PICOCLAW_TOOLS_SKILLS_GITHUB_TOKEN"`
	Proxy string `json:"proxy,omitempty" env:"PICOCLAW_TOOLS_SKILLS_GITHUB_PROXY"`
}

func (v *skillsGithubConfigV0) ToSkillsGithubConfig() SkillsGithubConfig {
	return SkillsGithubConfig{
		Token: *NewSecureString(v.Token),
		Proxy: v.Proxy,
	}
}

func (v *skillsRegistriesConfigV0) ToSkillsRegistriesConfig() SkillsRegistriesConfig {
	clawHub := v.ClawHub.ToClawHubRegistryConfig()

	return SkillsRegistriesConfig{
		ClawHub: clawHub,
	}
}

func (v *skillsToolsConfigV0) ToSkillsToolsConfig() SkillsToolsConfig {
	registries := v.Registries.ToSkillsRegistriesConfig()
	github := v.Github.ToSkillsGithubConfig()
	return SkillsToolsConfig{
		ToolConfig:            v.ToolConfig,
		Registries:            registries,
		Github:                github,
		MaxConcurrentSearches: v.MaxConcurrentSearches,
		SearchCache:           v.SearchCache,
	}
}
