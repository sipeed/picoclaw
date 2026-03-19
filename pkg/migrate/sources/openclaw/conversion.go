package openclaw

import (
	"fmt"

	"jane/pkg/config"
)

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

	if c.Channels.Matrix != nil && supportedChannels["matrix"] {
		enabled := c.Channels.Matrix.Enabled == nil || *c.Channels.Matrix.Enabled
		channels.Matrix = MatrixConfig{
			Enabled:   enabled,
			AllowFrom: c.Channels.Matrix.AllowFrom,
		}
		if c.Channels.Matrix.Homeserver != nil {
			channels.Matrix.Homeserver = *c.Channels.Matrix.Homeserver
		}
		if c.Channels.Matrix.UserID != nil {
			channels.Matrix.UserID = *c.Channels.Matrix.UserID
		}
		if c.Channels.Matrix.AccessToken != nil {
			channels.Matrix.AccessToken = *c.Channels.Matrix.AccessToken
		}
	}

	if c.Channels.Signal != nil {
		*warnings = append(*warnings, "Channel 'signal': No PicoClaw adapter available")
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
		Matrix: config.MatrixConfig{
			Enabled:      c.Matrix.Enabled,
			Homeserver:   c.Matrix.Homeserver,
			UserID:       c.Matrix.UserID,
			AccessToken:  c.Matrix.AccessToken,
			AllowFrom:    c.Matrix.AllowFrom,
			JoinOnInvite: true,
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
				APIKeys:    c.Web.Brave.APIKeys,
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
			AllowRemote:        config.DefaultConfig().Tools.Exec.AllowRemote,
		},
	}
}
