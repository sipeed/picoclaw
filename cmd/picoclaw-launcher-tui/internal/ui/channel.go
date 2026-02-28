package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	picoclawconfig "github.com/sipeed/picoclaw/pkg/config"
)

func (s *appState) channelMenu() tview.Primitive {
	items := []MenuItem{
		{Label: "Back", Description: "Return to main menu", Action: func() { s.pop() }},
		channelItem(
			"Telegram",
			"Telegram bot settings",
			s.config.Channels.Telegram.Enabled,
			func() { s.push("channel-telegram", s.telegramForm()) },
		),
		channelItem(
			"Discord",
			"Discord bot settings",
			s.config.Channels.Discord.Enabled,
			func() { s.push("channel-discord", s.discordForm()) },
		),
		channelItem(
			"QQ",
			"QQ bot settings",
			s.config.Channels.QQ.Enabled,
			func() { s.push("channel-qq", s.qqForm()) },
		),
		channelItem(
			"MaixCam",
			"MaixCam gateway",
			s.config.Channels.MaixCam.Enabled,
			func() { s.push("channel-maixcam", s.maixcamForm()) },
		),
		channelItem(
			"WhatsApp",
			"WhatsApp bridge",
			s.config.Channels.WhatsApp.Enabled,
			func() { s.push("channel-whatsapp", s.whatsappForm()) },
		),
		channelItem(
			"Feishu",
			"Feishu bot settings",
			s.config.Channels.Feishu.Enabled,
			func() { s.push("channel-feishu", s.feishuForm()) },
		),
		channelItem(
			"DingTalk",
			"DingTalk bot settings",
			s.config.Channels.DingTalk.Enabled,
			func() { s.push("channel-dingtalk", s.dingtalkForm()) },
		),
		channelItem(
			"Slack",
			"Slack bot settings",
			s.config.Channels.Slack.Enabled,
			func() { s.push("channel-slack", s.slackForm()) },
		),
		channelItem(
			"LINE",
			"LINE bot settings",
			s.config.Channels.LINE.Enabled,
			func() { s.push("channel-line", s.lineForm()) },
		),
		channelItem(
			"OneBot",
			"OneBot settings",
			s.config.Channels.OneBot.Enabled,
			func() { s.push("channel-onebot", s.onebotForm()) },
		),
		channelItem(
			"WeCom",
			"WeCom bot settings",
			s.config.Channels.WeCom.Enabled,
			func() { s.push("channel-wecom", s.wecomForm()) },
		),
		channelItem(
			"WeCom App",
			"WeCom App settings",
			s.config.Channels.WeComApp.Enabled,
			func() { s.push("channel-wecomapp", s.wecomAppForm()) },
		),
	}

	menu := NewMenu("Channels", items)
	menu.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			s.pop()
			return nil
		}
		if event.Rune() == 'q' {
			s.pop()
			return nil
		}
		return event
	})
	return menu
}

func refreshChannelMenuFromState(menu *Menu, s *appState) {
	items := []MenuItem{
		{Label: "Back", Description: "Return to main menu", Action: func() { s.pop() }},
		channelItem(
			"Telegram",
			"Telegram bot settings",
			s.config.Channels.Telegram.Enabled,
			func() { s.push("channel-telegram", s.telegramForm()) },
		),
		channelItem(
			"Discord",
			"Discord bot settings",
			s.config.Channels.Discord.Enabled,
			func() { s.push("channel-discord", s.discordForm()) },
		),
		channelItem(
			"QQ",
			"QQ bot settings",
			s.config.Channels.QQ.Enabled,
			func() { s.push("channel-qq", s.qqForm()) },
		),
		channelItem(
			"MaixCam",
			"MaixCam gateway",
			s.config.Channels.MaixCam.Enabled,
			func() { s.push("channel-maixcam", s.maixcamForm()) },
		),
		channelItem(
			"WhatsApp",
			"WhatsApp bridge",
			s.config.Channels.WhatsApp.Enabled,
			func() { s.push("channel-whatsapp", s.whatsappForm()) },
		),
		channelItem(
			"Feishu",
			"Feishu bot settings",
			s.config.Channels.Feishu.Enabled,
			func() { s.push("channel-feishu", s.feishuForm()) },
		),
		channelItem(
			"DingTalk",
			"DingTalk bot settings",
			s.config.Channels.DingTalk.Enabled,
			func() { s.push("channel-dingtalk", s.dingtalkForm()) },
		),
		channelItem(
			"Slack",
			"Slack bot settings",
			s.config.Channels.Slack.Enabled,
			func() { s.push("channel-slack", s.slackForm()) },
		),
		channelItem(
			"LINE",
			"LINE bot settings",
			s.config.Channels.LINE.Enabled,
			func() { s.push("channel-line", s.lineForm()) },
		),
		channelItem(
			"OneBot",
			"OneBot settings",
			s.config.Channels.OneBot.Enabled,
			func() { s.push("channel-onebot", s.onebotForm()) },
		),
		channelItem(
			"WeCom",
			"WeCom bot settings",
			s.config.Channels.WeCom.Enabled,
			func() { s.push("channel-wecom", s.wecomForm()) },
		),
		channelItem(
			"WeCom App",
			"WeCom App settings",
			s.config.Channels.WeComApp.Enabled,
			func() { s.push("channel-wecomapp", s.wecomAppForm()) },
		),
	}
	menu.applyItems(items)
}

func (s *appState) telegramForm() tview.Primitive {
	cfg := &s.config.Channels.Telegram
	form := baseChannelForm("Telegram", cfg.Enabled, func(v bool) {
		cfg.Enabled = v
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["channel"]; ok {
			refreshChannelMenuFromState(menu, s)
		}
	})
	form.AddInputField("Token", cfg.Token, 128, nil, func(text string) {
		cfg.Token = strings.TrimSpace(text)
	})
	form.AddInputField("Proxy", cfg.Proxy, 128, nil, func(text string) {
		cfg.Proxy = strings.TrimSpace(text)
	})
	form.AddInputField("Allow From", strings.Join(cfg.AllowFrom, ","), 128, nil, func(text string) {
		cfg.AllowFrom = splitCSV(text)
	})
	return wrapWithBack(form, s)
}

func (s *appState) discordForm() tview.Primitive {
	cfg := &s.config.Channels.Discord
	form := baseChannelForm("Discord", cfg.Enabled, func(v bool) {
		cfg.Enabled = v
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["channel"]; ok {
			refreshChannelMenuFromState(menu, s)
		}
	})
	form.AddInputField("Token", cfg.Token, 128, nil, func(text string) {
		cfg.Token = strings.TrimSpace(text)
	})
	form.AddCheckbox("Mention Only", cfg.MentionOnly, func(checked bool) {
		cfg.MentionOnly = checked
	})
	form.AddInputField("Allow From", strings.Join(cfg.AllowFrom, ","), 128, nil, func(text string) {
		cfg.AllowFrom = splitCSV(text)
	})
	return wrapWithBack(form, s)
}

func (s *appState) qqForm() tview.Primitive {
	cfg := &s.config.Channels.QQ
	form := baseChannelForm("QQ", cfg.Enabled, func(v bool) {
		cfg.Enabled = v
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["channel"]; ok {
			refreshChannelMenuFromState(menu, s)
		}
	})
	form.AddInputField("App ID", cfg.AppID, 64, nil, func(text string) {
		cfg.AppID = strings.TrimSpace(text)
	})
	form.AddInputField("App Secret", cfg.AppSecret, 128, nil, func(text string) {
		cfg.AppSecret = strings.TrimSpace(text)
	})
	form.AddInputField("Allow From", strings.Join(cfg.AllowFrom, ","), 128, nil, func(text string) {
		cfg.AllowFrom = splitCSV(text)
	})
	return wrapWithBack(form, s)
}

func (s *appState) maixcamForm() tview.Primitive {
	cfg := &s.config.Channels.MaixCam
	form := baseChannelForm("MaixCam", cfg.Enabled, func(v bool) {
		cfg.Enabled = v
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["channel"]; ok {
			refreshChannelMenuFromState(menu, s)
		}
	})
	form.AddInputField("Host", cfg.Host, 64, nil, func(text string) {
		cfg.Host = strings.TrimSpace(text)
	})
	addIntField(form, "Port", cfg.Port, func(value int) { cfg.Port = value })
	form.AddInputField("Allow From", strings.Join(cfg.AllowFrom, ","), 128, nil, func(text string) {
		cfg.AllowFrom = splitCSV(text)
	})
	return wrapWithBack(form, s)
}

func (s *appState) whatsappForm() tview.Primitive {
	cfg := &s.config.Channels.WhatsApp
	form := baseChannelForm("WhatsApp", cfg.Enabled, func(v bool) {
		cfg.Enabled = v
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["channel"]; ok {
			refreshChannelMenuFromState(menu, s)
		}
	})
	form.AddInputField("Bridge URL", cfg.BridgeURL, 128, nil, func(text string) {
		cfg.BridgeURL = strings.TrimSpace(text)
	})
	form.AddInputField("Allow From", strings.Join(cfg.AllowFrom, ","), 128, nil, func(text string) {
		cfg.AllowFrom = splitCSV(text)
	})
	return wrapWithBack(form, s)
}

func (s *appState) feishuForm() tview.Primitive {
	cfg := &s.config.Channels.Feishu
	form := baseChannelForm("Feishu", cfg.Enabled, func(v bool) {
		cfg.Enabled = v
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["channel"]; ok {
			refreshChannelMenuFromState(menu, s)
		}
	})
	form.AddInputField("App ID", cfg.AppID, 64, nil, func(text string) {
		cfg.AppID = strings.TrimSpace(text)
	})
	form.AddInputField("App Secret", cfg.AppSecret, 128, nil, func(text string) {
		cfg.AppSecret = strings.TrimSpace(text)
	})
	form.AddInputField("Encrypt Key", cfg.EncryptKey, 128, nil, func(text string) {
		cfg.EncryptKey = strings.TrimSpace(text)
	})
	form.AddInputField("Verification Token", cfg.VerificationToken, 128, nil, func(text string) {
		cfg.VerificationToken = strings.TrimSpace(text)
	})
	form.AddInputField("Allow From", strings.Join(cfg.AllowFrom, ","), 128, nil, func(text string) {
		cfg.AllowFrom = splitCSV(text)
	})
	return wrapWithBack(form, s)
}

func (s *appState) dingtalkForm() tview.Primitive {
	cfg := &s.config.Channels.DingTalk
	form := baseChannelForm("DingTalk", cfg.Enabled, func(v bool) {
		cfg.Enabled = v
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["channel"]; ok {
			refreshChannelMenuFromState(menu, s)
		}
	})
	form.AddInputField("Client ID", cfg.ClientID, 64, nil, func(text string) {
		cfg.ClientID = strings.TrimSpace(text)
	})
	form.AddInputField("Client Secret", cfg.ClientSecret, 128, nil, func(text string) {
		cfg.ClientSecret = strings.TrimSpace(text)
	})
	form.AddInputField("Allow From", strings.Join(cfg.AllowFrom, ","), 128, nil, func(text string) {
		cfg.AllowFrom = splitCSV(text)
	})
	return wrapWithBack(form, s)
}

func (s *appState) slackForm() tview.Primitive {
	cfg := &s.config.Channels.Slack
	form := baseChannelForm("Slack", cfg.Enabled, func(v bool) {
		cfg.Enabled = v
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["channel"]; ok {
			refreshChannelMenuFromState(menu, s)
		}
	})
	form.AddInputField("Bot Token", cfg.BotToken, 128, nil, func(text string) {
		cfg.BotToken = strings.TrimSpace(text)
	})
	form.AddInputField("App Token", cfg.AppToken, 128, nil, func(text string) {
		cfg.AppToken = strings.TrimSpace(text)
	})
	form.AddInputField("Allow From", strings.Join(cfg.AllowFrom, ","), 128, nil, func(text string) {
		cfg.AllowFrom = splitCSV(text)
	})
	return wrapWithBack(form, s)
}

func (s *appState) lineForm() tview.Primitive {
	cfg := &s.config.Channels.LINE
	form := baseChannelForm("LINE", cfg.Enabled, func(v bool) {
		cfg.Enabled = v
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["channel"]; ok {
			refreshChannelMenuFromState(menu, s)
		}
	})
	form.AddInputField("Channel Secret", cfg.ChannelSecret, 128, nil, func(text string) {
		cfg.ChannelSecret = strings.TrimSpace(text)
	})
	form.AddInputField("Channel Access Token", cfg.ChannelAccessToken, 128, nil, func(text string) {
		cfg.ChannelAccessToken = strings.TrimSpace(text)
	})
	form.AddInputField("Webhook Host", cfg.WebhookHost, 64, nil, func(text string) {
		cfg.WebhookHost = strings.TrimSpace(text)
	})
	addIntField(form, "Webhook Port", cfg.WebhookPort, func(value int) { cfg.WebhookPort = value })
	form.AddInputField("Webhook Path", cfg.WebhookPath, 64, nil, func(text string) {
		cfg.WebhookPath = strings.TrimSpace(text)
	})
	form.AddInputField("Allow From", strings.Join(cfg.AllowFrom, ","), 128, nil, func(text string) {
		cfg.AllowFrom = splitCSV(text)
	})
	return wrapWithBack(form, s)
}

func (s *appState) onebotForm() tview.Primitive {
	cfg := &s.config.Channels.OneBot
	form := baseChannelForm("OneBot", cfg.Enabled, func(v bool) {
		cfg.Enabled = v
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["channel"]; ok {
			refreshChannelMenuFromState(menu, s)
		}
	})
	form.AddInputField("WS URL", cfg.WSUrl, 128, nil, func(text string) {
		cfg.WSUrl = strings.TrimSpace(text)
	})
	form.AddInputField("Access Token", cfg.AccessToken, 128, nil, func(text string) {
		cfg.AccessToken = strings.TrimSpace(text)
	})
	addIntField(
		form,
		"Reconnect Interval",
		cfg.ReconnectInterval,
		func(value int) { cfg.ReconnectInterval = value },
	)
	form.AddInputField(
		"Group Trigger Prefix",
		strings.Join(cfg.GroupTriggerPrefix, ","),
		128,
		nil,
		func(text string) {
			cfg.GroupTriggerPrefix = splitCSV(text)
		},
	)
	form.AddInputField("Allow From", strings.Join(cfg.AllowFrom, ","), 128, nil, func(text string) {
		cfg.AllowFrom = splitCSV(text)
	})
	return wrapWithBack(form, s)
}

func (s *appState) wecomForm() tview.Primitive {
	cfg := &s.config.Channels.WeCom
	form := baseChannelForm("WeCom", cfg.Enabled, func(v bool) {
		cfg.Enabled = v
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["channel"]; ok {
			refreshChannelMenuFromState(menu, s)
		}
	})
	form.AddInputField("Token", cfg.Token, 128, nil, func(text string) {
		cfg.Token = strings.TrimSpace(text)
	})
	form.AddInputField("Encoding AES Key", cfg.EncodingAESKey, 128, nil, func(text string) {
		cfg.EncodingAESKey = strings.TrimSpace(text)
	})
	form.AddInputField("Webhook URL", cfg.WebhookURL, 128, nil, func(text string) {
		cfg.WebhookURL = strings.TrimSpace(text)
	})
	form.AddInputField("Webhook Host", cfg.WebhookHost, 64, nil, func(text string) {
		cfg.WebhookHost = strings.TrimSpace(text)
	})
	addIntField(form, "Webhook Port", cfg.WebhookPort, func(value int) { cfg.WebhookPort = value })
	form.AddInputField("Webhook Path", cfg.WebhookPath, 64, nil, func(text string) {
		cfg.WebhookPath = strings.TrimSpace(text)
	})
	form.AddInputField("Allow From", strings.Join(cfg.AllowFrom, ","), 128, nil, func(text string) {
		cfg.AllowFrom = splitCSV(text)
	})
	addIntField(
		form,
		"Reply Timeout",
		cfg.ReplyTimeout,
		func(value int) { cfg.ReplyTimeout = value },
	)
	return wrapWithBack(form, s)
}

func (s *appState) wecomAppForm() tview.Primitive {
	cfg := &s.config.Channels.WeComApp
	form := baseChannelForm("WeCom App", cfg.Enabled, func(v bool) {
		cfg.Enabled = v
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["channel"]; ok {
			refreshChannelMenuFromState(menu, s)
		}
	})
	form.AddInputField("Corp ID", cfg.CorpID, 64, nil, func(text string) {
		cfg.CorpID = strings.TrimSpace(text)
	})
	form.AddInputField("Corp Secret", cfg.CorpSecret, 128, nil, func(text string) {
		cfg.CorpSecret = strings.TrimSpace(text)
	})
	addInt64Field(form, "Agent ID", cfg.AgentID, func(value int64) { cfg.AgentID = value })
	form.AddInputField("Token", cfg.Token, 128, nil, func(text string) {
		cfg.Token = strings.TrimSpace(text)
	})
	form.AddInputField("Encoding AES Key", cfg.EncodingAESKey, 128, nil, func(text string) {
		cfg.EncodingAESKey = strings.TrimSpace(text)
	})
	form.AddInputField("Webhook Host", cfg.WebhookHost, 64, nil, func(text string) {
		cfg.WebhookHost = strings.TrimSpace(text)
	})
	addIntField(form, "Webhook Port", cfg.WebhookPort, func(value int) { cfg.WebhookPort = value })
	form.AddInputField("Webhook Path", cfg.WebhookPath, 64, nil, func(text string) {
		cfg.WebhookPath = strings.TrimSpace(text)
	})
	form.AddInputField("Allow From", strings.Join(cfg.AllowFrom, ","), 128, nil, func(text string) {
		cfg.AllowFrom = splitCSV(text)
	})
	addIntField(
		form,
		"Reply Timeout",
		cfg.ReplyTimeout,
		func(value int) { cfg.ReplyTimeout = value },
	)
	return wrapWithBack(form, s)
}

func baseChannelForm(title string, enabled bool, onEnabled func(bool)) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(fmt.Sprintf("Channel: %s", title))
	form.SetButtonBackgroundColor(tcell.NewRGBColor(80, 250, 123))
	form.SetButtonTextColor(tcell.NewRGBColor(12, 13, 22))
	form.AddCheckbox("Enabled", enabled, func(checked bool) {
		onEnabled(checked)
	})
	return form
}

func wrapWithBack(form *tview.Form, s *appState) tview.Primitive {
	form.AddButton("Back", func() {
		s.pop()
	})
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			s.pop()
			return nil
		}
		return event
	})
	return form
}

func splitCSV(input string) picoclawconfig.FlexibleStringSlice {
	parts := strings.Split(strings.TrimSpace(input), ",")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func addIntField(form *tview.Form, label string, value int, onChange func(int)) {
	form.AddInputField(label, fmt.Sprintf("%d", value), 16, nil, func(text string) {
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(text), "%d", &parsed); err == nil {
			onChange(parsed)
		}
	})
}

func addInt64Field(form *tview.Form, label string, value int64, onChange func(int64)) {
	form.AddInputField(label, fmt.Sprintf("%d", value), 16, nil, func(text string) {
		var parsed int64
		if _, err := fmt.Sscanf(strings.TrimSpace(text), "%d", &parsed); err == nil {
			onChange(parsed)
		}
	})
}

func channelItem(label, description string, enabled bool, action MenuAction) MenuItem {
	item := MenuItem{
		Label:       label,
		Description: description,
		Action:      action,
	}
	if !enabled {
		color := tcell.ColorGray
		item.MainColor = &color
	}
	return item
}
