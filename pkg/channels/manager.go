// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package channels

import (
	"context"
	"fmt"
	"sync"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type Manager struct {
	channels     map[string]Channel
	bus          *bus.MessageBus
	config       *config.Config
	dispatchTask *asyncTask
	mu           sync.RWMutex
}

type asyncTask struct {
	cancel context.CancelFunc
}

func NewManager(cfg *config.Config, messageBus *bus.MessageBus) (*Manager, error) {
	m := &Manager{
		channels: make(map[string]Channel),
		bus:      messageBus,
		config:   cfg,
	}

	if err := m.initChannels(); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Manager) initChannels() error {
	logger.InfoC("channels", "Initializing channel manager")

	if m.config.Channels.Telegram.Enabled && m.config.Channels.Telegram.Token != "" {
		logger.DebugC("channels", "Attempting to initialize Telegram channel")
		telegram, err := NewTelegramChannel(m.config.Channels.Telegram, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize Telegram channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["telegram"] = telegram
			logger.InfoC("channels", "Telegram channel enabled successfully")
		}
	}

	if m.config.Channels.WhatsApp.Enabled && m.config.Channels.WhatsApp.BridgeURL != "" {
		logger.DebugC("channels", "Attempting to initialize WhatsApp channel")
		whatsapp, err := NewWhatsAppChannel(m.config.Channels.WhatsApp, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize WhatsApp channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["whatsapp"] = whatsapp
			logger.InfoC("channels", "WhatsApp channel enabled successfully")
		}
	}

	if m.config.Channels.Feishu.Enabled {
		logger.DebugC("channels", "Attempting to initialize Feishu channel")
		feishu, err := NewFeishuChannel(m.config.Channels.Feishu, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize Feishu channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["feishu"] = feishu
			logger.InfoC("channels", "Feishu channel enabled successfully")
		}
	}

	if m.config.Channels.Discord.Enabled && m.config.Channels.Discord.Token != "" {
		logger.DebugC("channels", "Attempting to initialize Discord channel")
		discord, err := NewDiscordChannel(m.config.Channels.Discord, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize Discord channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["discord"] = discord
			logger.InfoC("channels", "Discord channel enabled successfully")
		}
	}

	if m.config.Channels.MaixCam.Enabled {
		logger.DebugC("channels", "Attempting to initialize MaixCam channel")
		maixcam, err := NewMaixCamChannel(m.config.Channels.MaixCam, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize MaixCam channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["maixcam"] = maixcam
			logger.InfoC("channels", "MaixCam channel enabled successfully")
		}
	}

	if m.config.Channels.QQ.Enabled {
		logger.DebugC("channels", "Attempting to initialize QQ channel")
		qq, err := NewQQChannel(m.config.Channels.QQ, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize QQ channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["qq"] = qq
			logger.InfoC("channels", "QQ channel enabled successfully")
		}
	}

	if m.config.Channels.DingTalk.Enabled && m.config.Channels.DingTalk.ClientID != "" {
		logger.DebugC("channels", "Attempting to initialize DingTalk channel")
		dingtalk, err := NewDingTalkChannel(m.config.Channels.DingTalk, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize DingTalk channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["dingtalk"] = dingtalk
			logger.InfoC("channels", "DingTalk channel enabled successfully")
		}
	}

	if m.config.Channels.Slack.Enabled && m.config.Channels.Slack.BotToken != "" {
		logger.DebugC("channels", "Attempting to initialize Slack channel")
		slackCh, err := NewSlackChannel(m.config.Channels.Slack, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize Slack channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["slack"] = slackCh
			logger.InfoC("channels", "Slack channel enabled successfully")
		}
	}

	if m.config.Channels.LINE.Enabled && m.config.Channels.LINE.ChannelAccessToken != "" {
		logger.DebugC("channels", "Attempting to initialize LINE channel")
		line, err := NewLINEChannel(m.config.Channels.LINE, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize LINE channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["line"] = line
			logger.InfoC("channels", "LINE channel enabled successfully")
		}
	}

	if m.config.Channels.OneBot.Enabled && m.config.Channels.OneBot.WSUrl != "" {
		logger.DebugC("channels", "Attempting to initialize OneBot channel")
		onebot, err := NewOneBotChannel(m.config.Channels.OneBot, m.bus)
		if err != nil {
			logger.ErrorCF("channels", "Failed to initialize OneBot channel", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			m.channels["onebot"] = onebot
			logger.InfoC("channels", "OneBot channel enabled successfully")
		}
	}

	logger.InfoCF("channels", "Channel initialization completed", map[string]interface{}{
		"enabled_channels": len(m.channels),
	})

	return nil
}

func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.channels) == 0 {
		logger.WarnC("channels", "No channels enabled")
		return nil
	}

	logger.InfoC("channels", "Starting all channels")

	dispatchCtx, cancel := context.WithCancel(ctx)
	m.dispatchTask = &asyncTask{cancel: cancel}

	go m.dispatchOutbound(dispatchCtx)

	for name, channel := range m.channels {
		logger.InfoCF("channels", "Starting channel", map[string]interface{}{
			"channel": name,
		})
		if err := channel.Start(ctx); err != nil {
			logger.ErrorCF("channels", "Failed to start channel", map[string]interface{}{
				"channel": name,
				"error":   err.Error(),
			})
		}
	}

	logger.InfoC("channels", "All channels started")
	return nil
}

func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logger.InfoC("channels", "Stopping all channels")

	if m.dispatchTask != nil {
		m.dispatchTask.cancel()
		m.dispatchTask = nil
	}

	for name, channel := range m.channels {
		logger.InfoCF("channels", "Stopping channel", map[string]interface{}{
			"channel": name,
		})
		if err := channel.Stop(ctx); err != nil {
			logger.ErrorCF("channels", "Error stopping channel", map[string]interface{}{
				"channel": name,
				"error":   err.Error(),
			})
		}
	}

	logger.InfoC("channels", "All channels stopped")
	return nil
}

func (m *Manager) dispatchOutbound(ctx context.Context) {
	logger.InfoC("channels", "Outbound dispatcher started")

	for {
		select {
		case <-ctx.Done():
			logger.InfoC("channels", "Outbound dispatcher stopped")
			return
		default:
			msg, ok := m.bus.SubscribeOutbound(ctx)
			if !ok {
				continue
			}

			// Silently skip internal channels
			if constants.IsInternalChannel(msg.Channel) {
				continue
			}

			m.mu.RLock()
			channel, exists := m.channels[msg.Channel]
			m.mu.RUnlock()

			if !exists {
				logger.WarnCF("channels", "Unknown channel for outbound message", map[string]interface{}{
					"channel": msg.Channel,
				})
				continue
			}

			if err := channel.Send(ctx, msg); err != nil {
				logger.ErrorCF("channels", "Error sending message to channel", map[string]interface{}{
					"channel": msg.Channel,
					"error":   err.Error(),
				})
			}
		}
	}
}

func (m *Manager) GetChannel(name string) (Channel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	channel, ok := m.channels[name]
	return channel, ok
}

func (m *Manager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]interface{})
	for name, channel := range m.channels {
		status[name] = map[string]interface{}{
			"enabled": true,
			"running": channel.IsRunning(),
		}
	}
	return status
}

func (m *Manager) GetEnabledChannels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}

func (m *Manager) RegisterChannel(name string, channel Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[name] = channel
}

func (m *Manager) UnregisterChannel(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.channels, name)
}

func (m *Manager) SendToChannel(ctx context.Context, channelName, chatID, content string) error {
	m.mu.RLock()
	channel, exists := m.channels[channelName]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}

	msg := bus.OutboundMessage{
		Channel: channelName,
		ChatID:  chatID,
		Content: content,
	}

	return channel.Send(ctx, msg)
}
