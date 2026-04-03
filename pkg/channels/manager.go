// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// Package channels 负责管理所有消息渠道（Telegram/Discord/Slack/微信等）。
//
// # 核心架构
//
// Manager 是渠道管理的核心结构，负责渠道生命周期管理和消息分发。
//
// # 消息处理流水线
//
// 消息从生成到发送经过以下环节：
//
//	bus.OutboundChan → dispatchOutbound → per-channel worker queue → runWorker → preSend → Send
//
// # 核心接口
//
//   - Channel: 基础发送接口
//   - MessageEditor: 编辑已发送的消息
//   - PlaceholderCapable: 发送"思考中..."占位消息
//   - StreamingCapable: 流式推送消息内容
//   - WebhookHandler: 处理 Webhook 回调
//
// # 速率限制
//
// 每个渠道拥有独立的 rate.Limiter，按渠道类型配置不同的速率限制（如 telegram 20条/秒、discord 1条/秒）。
//
// # TTL 清理
//
// janitor 定时器定期清理过期的 typing/placeholder/reaction 条目，
// 防止在出站路径未能触发 preSend（如 LLM 错误）时产生内存泄漏。
package channels

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
)

const (
	defaultChannelQueueSize = 16               // 每个渠道 worker 的消息队列大小
	defaultRateLimit        = 10               // 默认速率限制（10条/秒）
	maxRetries              = 3                // 发送失败最大重试次数
	rateLimitDelay          = 1 * time.Second  // 速率限制错误的固定延迟
	baseBackoff             = 500 * time.Millisecond // 指数退避的基础延迟
	maxBackoff              = 8 * time.Second  // 指数退避的最大延迟

	janitorInterval = 10 * time.Second  // TTL 清理定时器的执行间隔
	typingStopTTL   = 5 * time.Minute   // typing 停止条目的过期时间
	placeholderTTL  = 10 * time.Minute  // 占位消息条目的过期时间
)

// typingEntry 封装 typing 停止函数及其创建时间，用于 TTL 过期清理。
type typingEntry struct {
	stop      func()
	createdAt time.Time
}

// reactionEntry 封装 reaction 撤销函数及其创建时间，用于 TTL 过期清理。
type reactionEntry struct {
	undo      func()
	createdAt time.Time
}

// placeholderEntry 封装占位消息 ID 及其创建时间，用于 TTL 过期清理。
type placeholderEntry struct {
	id        string
	createdAt time.Time
}

// channelRateConfig 各渠道的速率限制配置（每秒允许发送的消息数）。
// 未在此映射中配置的渠道使用 defaultRateLimit。
var channelRateConfig = map[string]float64{
	"telegram": 20, // Telegram: 20条/秒
	"discord":  1,  // Discord: 1条/秒
	"slack":    1,  // Slack: 1条/秒
	"matrix":   2,  // Matrix: 2条/秒
	"line":     10, // LINE: 10条/秒
	"qq":       5,  // QQ: 5条/秒
	"irc":      2,  // IRC: 2条/秒
}

// channelWorker 每个渠道对应一个 worker，负责从消息队列中取出消息并发送。
// 包含独立的消息队列、媒体队列和速率限制器。
type channelWorker struct {
	ch         Channel                        // 渠道实例
	queue      chan bus.OutboundMessage        // 文本消息队列
	mediaQueue chan bus.OutboundMediaMessage   // 媒体消息队列
	done       chan struct{}                   // 文本 worker 退出信号
	mediaDone  chan struct{}                   // 媒体 worker 退出信号
	limiter    *rate.Limiter                   // 速率限制器
}

// Manager 是渠道管理的核心结构，管理所有渠道的生命周期、消息分发和状态跟踪。
type Manager struct {
	channels      map[string]Channel              // 渠道名称 → Channel 实例
	workers       map[string]*channelWorker       // 渠道名称 → worker 实例
	bus           *bus.MessageBus                 // 消息总线，用于接收出站消息
	config        *config.Config                  // 全局配置
	mediaStore    media.MediaStore                // 媒体存储
	dispatchTask  *asyncTask                      // 分发任务（持有 cancel 函数）
	mux           *dynamicServeMux                // 动态 HTTP 路由器
	httpServer    *http.Server                    // 共享 HTTP 服务器
	mu            sync.RWMutex                    // 保护 channels/workers 等字段的读写锁
	placeholders  sync.Map                        // "channel:chatID" → placeholderEntry（占位消息 ID）
	typingStops   sync.Map                        // "channel:chatID" → typingEntry（typing 停止函数）
	reactionUndos sync.Map                        // "channel:chatID" → reactionEntry（reaction 撤销函数）
	streamActive  sync.Map                        // "channel:chatID" → true（流式推送完成标记）
	channelHashes map[string]string               // 渠道名称 → 配置哈希（用于热重载时对比变更）
}

type asyncTask struct {
	cancel context.CancelFunc
}

// RecordPlaceholder 记录占位消息 ID，供后续 preSend 编辑或删除。
// 实现 PlaceholderRecorder 接口。
func (m *Manager) RecordPlaceholder(channel, chatID, placeholderID string) {
	key := channel + ":" + chatID
	m.placeholders.Store(key, placeholderEntry{id: placeholderID, createdAt: time.Now()})
}

// SendPlaceholder 向指定渠道/聊天发送"思考中..."占位消息，并记录供后续编辑。
// 如果渠道不支持 PlaceholderCapable 或发送失败，返回 false。
func (m *Manager) SendPlaceholder(ctx context.Context, channel, chatID string) bool {
	m.mu.RLock()
	ch, ok := m.channels[channel]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	pc, ok := ch.(PlaceholderCapable)
	if !ok {
		return false
	}
	phID, err := pc.SendPlaceholder(ctx, chatID)
	if err != nil || phID == "" {
		return false
	}
	m.RecordPlaceholder(channel, chatID, phID)
	return true
}

// RecordTypingStop 记录 typing 停止函数，供后续 preSend 调用。
// 如果已有旧的停止函数，会先调用它（确保前一个 typing 指示器被停止）。
// 实现 PlaceholderRecorder 接口。
func (m *Manager) RecordTypingStop(channel, chatID string, stop func()) {
	key := channel + ":" + chatID
	entry := typingEntry{stop: stop, createdAt: time.Now()}
	if previous, loaded := m.typingStops.Swap(key, entry); loaded {
		if oldEntry, ok := previous.(typingEntry); ok && oldEntry.stop != nil {
			oldEntry.stop()
		}
	}
}

// InvokeTypingStop 调用已注册的 typing 停止函数。
// 即使没有活跃的 typing 指示器也可安全调用（无操作）。
// 由 agent 循环在处理完成时调用（无论成功、错误或 panic），
// 确保无论是否发布出站消息都能停止 typing。
func (m *Manager) InvokeTypingStop(channel, chatID string) {
	key := channel + ":" + chatID
	if v, loaded := m.typingStops.LoadAndDelete(key); loaded {
		if entry, ok := v.(typingEntry); ok {
			entry.stop()
		}
	}
}

// RecordReactionUndo 记录 reaction 撤销函数，供后续 preSend 调用。
// 实现 PlaceholderRecorder 接口。
func (m *Manager) RecordReactionUndo(channel, chatID string, undo func()) {
	key := channel + ":" + chatID
	m.reactionUndos.Store(key, reactionEntry{undo: undo, createdAt: time.Now()})
}

// preSend 在发送消息前执行预处理：
//  1. 停止 typing 指示器
//  2. 撤销 reaction
//  3. 检查流式推送是否已完成（streamActive），若已完成则删除占位消息并跳过 Send
//  4. 尝试编辑占位消息（将"思考中..."替换为实际内容），编辑成功则跳过 Send
//
// 返回 true 表示消息已通过编辑/流式方式投递，应跳过后续的 Send 调用。
func (m *Manager) preSend(ctx context.Context, name string, msg bus.OutboundMessage, ch Channel) bool {
	key := name + ":" + msg.ChatID

	// 1. 停止 typing 指示器
	if v, loaded := m.typingStops.LoadAndDelete(key); loaded {
		if entry, ok := v.(typingEntry); ok {
			entry.stop() // 幂等操作，安全
		}
	}

	// 2. 撤销 reaction
	if v, loaded := m.reactionUndos.LoadAndDelete(key); loaded {
		if entry, ok := v.(reactionEntry); ok {
			entry.undo() // 幂等操作，安全
		}
	}

	// 3. 如果流式推送已完成，删除占位消息并跳过 Send
	if _, loaded := m.streamActive.LoadAndDelete(key); loaded {
		if v, loaded := m.placeholders.LoadAndDelete(key); loaded {
			if entry, ok := v.(placeholderEntry); ok && entry.id != "" {
				// 优先删除占位消息（比编辑为相同内容更干净的用户体验）
				if deleter, ok := ch.(MessageDeleter); ok {
					deleter.DeleteMessage(ctx, msg.ChatID, entry.id) // 尽力而为
				} else if editor, ok := ch.(MessageEditor); ok {
					editor.EditMessage(ctx, msg.ChatID, entry.id, msg.Content) // 回退方案
				}
			}
		}
		return true
	}

	// 4. 尝试编辑占位消息（将"思考中..."替换为实际内容）
	if v, loaded := m.placeholders.LoadAndDelete(key); loaded {
		if entry, ok := v.(placeholderEntry); ok && entry.id != "" {
			if editor, ok := ch.(MessageEditor); ok {
				if err := editor.EditMessage(ctx, msg.ChatID, entry.id, msg.Content); err == nil {
					return true // 编辑成功，跳过 Send
				}
				// 编辑失败 → 继续走正常 Send 流程
			}
		}
	}

	return false
}

// preSendMedia 在发送媒体附件前执行预处理（停止 typing、撤销 reaction、清理占位消息）。
// 与文本消息的 preSend 不同，媒体发送不会编辑占位消息（因为没有文本内容可替换），
// 仅在渠道支持 MessageDeleter 时尝试删除占位消息。
func (m *Manager) preSendMedia(ctx context.Context, name string, msg bus.OutboundMediaMessage, ch Channel) {
	key := name + ":" + msg.ChatID

	// 1. 停止 typing 指示器
	if v, loaded := m.typingStops.LoadAndDelete(key); loaded {
		if entry, ok := v.(typingEntry); ok {
			entry.stop() // 幂等操作，安全
		}
	}

	// 2. 撤销 reaction
	if v, loaded := m.reactionUndos.LoadAndDelete(key); loaded {
		if entry, ok := v.(reactionEntry); ok {
			entry.undo() // 幂等操作，安全
		}
	}

	// 3. 清除此聊天的流式推送完成标记
	m.streamActive.LoadAndDelete(key)

	// 4. 如果存在占位消息则删除
	if v, loaded := m.placeholders.LoadAndDelete(key); loaded {
		if entry, ok := v.(placeholderEntry); ok && entry.id != "" {
			if deleter, ok := ch.(MessageDeleter); ok {
				deleter.DeleteMessage(ctx, msg.ChatID, entry.id) // 尽力而为
			}
		}
	}
}

// NewManager 创建 Manager 实例，根据配置初始化所有渠道，并将自身注册为流式推送代理。
func NewManager(cfg *config.Config, messageBus *bus.MessageBus, store media.MediaStore) (*Manager, error) {
	m := &Manager{
		channels:      make(map[string]Channel),
		workers:       make(map[string]*channelWorker),
		bus:           messageBus,
		config:        cfg,
		mediaStore:    store,
		channelHashes: make(map[string]string),
	}

	// 注册为流式推送代理，使 agent 循环可以获取流式推送器
	messageBus.SetStreamDelegate(m)

	if err := m.initChannels(&cfg.Channels); err != nil {
		return nil, err
	}

	// 保存所有渠道的初始配置哈希（用于热重载时对比变更）
	m.channelHashes = toChannelHashes(cfg)

	return m, nil
}

// GetStreamer 实现 bus.StreamDelegate 接口。
// 检查指定渠道是否支持流式推送，若支持则返回一个 Streamer。
// 返回的 Streamer 在 Finalize 时会标记 streamActive，使 preSend 知道应清理占位消息。
func (m *Manager) GetStreamer(ctx context.Context, channelName, chatID string) (bus.Streamer, bool) {
	m.mu.RLock()
	ch, exists := m.channels[channelName]
	m.mu.RUnlock()

	if !exists {
		return nil, false
	}

	sc, ok := ch.(StreamingCapable)
	if !ok {
		return nil, false
	}

	streamer, err := sc.BeginStream(ctx, chatID)
	if err != nil {
		logger.DebugCF("channels", "Streaming unavailable, falling back to placeholder", map[string]any{
			"channel": channelName,
			"error":   err.Error(),
		})
		return nil, false
	}

	// 在 Finalize 时标记 streamActive，使 preSend 知道应清理占位消息
	key := channelName + ":" + chatID
	return &finalizeHookStreamer{
		Streamer:   streamer,
		onFinalize: func() { m.streamActive.Store(key, true) },
	}, true
}

// finalizeHookStreamer 包装 Streamer，在 Finalize 时执行钩子函数（标记 streamActive）。
type finalizeHookStreamer struct {
	Streamer
	onFinalize func()
}

func (s *finalizeHookStreamer) Finalize(ctx context.Context, content string) error {
	if err := s.Streamer.Finalize(ctx, content); err != nil {
		return err
	}
	s.onFinalize()
	return nil
}

// initChannel 根据渠道名称查找工厂函数并创建渠道实例。
// 创建成功后会注入 MediaStore、PlaceholderRecorder 和 Owner 引用。
func (m *Manager) initChannel(name, displayName string) {
	f, ok := getFactory(name)
	if !ok {
		logger.WarnCF("channels", "Factory not registered", map[string]any{
			"channel": displayName,
		})
		return
	}
	logger.DebugCF("channels", "Attempting to initialize channel", map[string]any{
		"channel": displayName,
	})
	ch, err := f(m.config, m.bus)
	if err != nil {
		logger.ErrorCF("channels", "Failed to initialize channel", map[string]any{
			"channel": displayName,
			"error":   err.Error(),
		})
	} else {
		// 注入 MediaStore（如果渠道支持）
		if m.mediaStore != nil {
			if setter, ok := ch.(interface{ SetMediaStore(s media.MediaStore) }); ok {
				setter.SetMediaStore(m.mediaStore)
			}
		}
		// 注入 PlaceholderRecorder（如果渠道支持）
		if setter, ok := ch.(interface{ SetPlaceholderRecorder(r PlaceholderRecorder) }); ok {
			setter.SetPlaceholderRecorder(m)
		}
		// 注入 Owner 引用，使 BaseChannel.HandleMessage 可以自动触发 typing/reaction
		if setter, ok := ch.(interface{ SetOwner(ch Channel) }); ok {
			setter.SetOwner(ch)
		}
		m.channels[name] = ch
		logger.InfoCF("channels", "Channel enabled successfully", map[string]any{
			"channel": displayName,
		})
	}
}

// initChannels 根据渠道配置逐个初始化所有已启用的渠道。
func (m *Manager) initChannels(channels *config.ChannelsConfig) error {
	logger.InfoC("channels", "Initializing channel manager")

	if channels.Telegram.Enabled && channels.Telegram.Token.String() != "" {
		m.initChannel("telegram", "Telegram")
	}

	if channels.WhatsApp.Enabled {
		waCfg := channels.WhatsApp
		if waCfg.UseNative {
			m.initChannel("whatsapp_native", "WhatsApp Native")
		} else if waCfg.BridgeURL != "" {
			m.initChannel("whatsapp", "WhatsApp")
		}
	}

	if channels.Feishu.Enabled {
		m.initChannel("feishu", "Feishu")
	}

	if channels.Discord.Enabled && channels.Discord.Token.String() != "" {
		m.initChannel("discord", "Discord")
	}

	if channels.MaixCam.Enabled {
		m.initChannel("maixcam", "MaixCam")
	}

	if channels.QQ.Enabled {
		m.initChannel("qq", "QQ")
	}

	if channels.DingTalk.Enabled && channels.DingTalk.ClientID != "" {
		m.initChannel("dingtalk", "DingTalk")
	}

	if channels.Slack.Enabled && channels.Slack.BotToken.String() != "" {
		m.initChannel("slack", "Slack")
	}

	if channels.Matrix.Enabled &&
		m.config.Channels.Matrix.Homeserver != "" &&
		m.config.Channels.Matrix.UserID != "" &&
		m.config.Channels.Matrix.AccessToken.String() != "" {
		m.initChannel("matrix", "Matrix")
	}

	if channels.LINE.Enabled && channels.LINE.ChannelAccessToken.String() != "" {
		m.initChannel("line", "LINE")
	}

	if channels.OneBot.Enabled && channels.OneBot.WSUrl != "" {
		m.initChannel("onebot", "OneBot")
	}

	if channels.WeCom.Enabled && channels.WeCom.BotID != "" && channels.WeCom.Secret.String() != "" {
		m.initChannel("wecom", "WeCom")
	}

	if channels.Weixin.Enabled && channels.Weixin.Token.String() != "" {
		m.initChannel("weixin", "Weixin")
	}

	if channels.Pico.Enabled && channels.Pico.Token.String() != "" {
		m.initChannel("pico", "Pico")
	}

	if channels.PicoClient.Enabled && channels.PicoClient.URL != "" {
		m.initChannel("pico_client", "Pico Client")
	}

	if channels.IRC.Enabled && channels.IRC.Server != "" {
		m.initChannel("irc", "IRC")
	}

	logger.InfoCF("channels", "Channel initialization completed", map[string]any{
		"enabled_channels": len(m.channels),
	})

	return nil
}

// SetupHTTPServer 创建共享 HTTP 服务器。
// 注册健康检查端点，并自动发现实现了 WebhookHandler 和/或 HealthChecker 的渠道，
// 为它们注册对应的 HTTP 处理器。
func (m *Manager) SetupHTTPServer(addr string, healthServer *health.Server) {
	m.mux = newDynamicServeMux()

	// 注册健康检查端点
	if healthServer != nil {
		healthServer.RegisterOnMux(m.mux)
	}

	// 发现并注册 Webhook 处理器和健康检查处理器
	m.registerHTTPHandlersLocked()

	m.httpServer = &http.Server{
		Addr:         addr,
		Handler:      m.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

// registerHTTPHandlersLocked 为 m.channels 中所有渠道注册 Webhook 和健康检查处理器。
// 调用者必须持有 m.mu（或确保独占访问）。
func (m *Manager) registerHTTPHandlersLocked() {
	for name, ch := range m.channels {
		m.registerChannelHTTPHandler(name, ch)
	}
}

// registerChannelHTTPHandler 为单个渠道注册 Webhook 和健康检查处理器到 m.mux。
func (m *Manager) registerChannelHTTPHandler(name string, ch Channel) {
	if wh, ok := ch.(WebhookHandler); ok {
		m.mux.Handle(wh.WebhookPath(), wh)
		logger.InfoCF("channels", "Webhook handler registered", map[string]any{
			"channel": name,
			"path":    wh.WebhookPath(),
		})
	}
	if hc, ok := ch.(HealthChecker); ok {
		m.mux.HandleFunc(hc.HealthPath(), hc.HealthHandler)
		logger.InfoCF("channels", "Health endpoint registered", map[string]any{
			"channel": name,
			"path":    hc.HealthPath(),
		})
	}
}

// unregisterChannelHTTPHandler 从 m.mux 中移除单个渠道的 Webhook 和健康检查处理器。
func (m *Manager) unregisterChannelHTTPHandler(name string, ch Channel) {
	if wh, ok := ch.(WebhookHandler); ok {
		m.mux.Unhandle(wh.WebhookPath())
		logger.InfoCF("channels", "Webhook handler unregistered", map[string]any{
			"channel": name,
			"path":    wh.WebhookPath(),
		})
	}
	if hc, ok := ch.(HealthChecker); ok {
		m.mux.Unhandle(hc.HealthPath())
		logger.InfoCF("channels", "Health endpoint unregistered", map[string]any{
			"channel": name,
			"path":    hc.HealthPath(),
		})
	}
}

// StartAll 启动所有已初始化的渠道。
// 为每个渠道创建 worker 并启动分发循环、TTL 清理定时器和共享 HTTP 服务器。
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.channels) == 0 {
		logger.WarnC("channels", "No channels enabled")
	}

	logger.InfoC("channels", "Starting all channels")

	dispatchCtx, cancel := context.WithCancel(ctx)
	m.dispatchTask = &asyncTask{cancel: cancel}

	for name, channel := range m.channels {
		logger.InfoCF("channels", "Starting channel", map[string]any{
			"channel": name,
		})
		if err := channel.Start(ctx); err != nil {
			logger.ErrorCF("channels", "Failed to start channel", map[string]any{
				"channel": name,
				"error":   err.Error(),
			})
			continue
		}
		// 渠道成功启动后才创建 worker（延迟创建）
		w := newChannelWorker(name, channel)
		m.workers[name] = w
		go m.runWorker(dispatchCtx, name, w)
		go m.runMediaWorker(dispatchCtx, name, w)
	}

	// 启动分发器，从消息总线读取消息并路由到各渠道 worker
	go m.dispatchOutbound(dispatchCtx)
	go m.dispatchOutboundMedia(dispatchCtx)

	// 启动 TTL 清理定时器，定期清理过期的 typing/placeholder/reaction 条目
	go m.runTTLJanitor(dispatchCtx)

	// 启动共享 HTTP 服务器（如果已配置）
	if m.httpServer != nil {
		go func() {
			logger.InfoCF("channels", "Shared HTTP server listening", map[string]any{
				"addr": m.httpServer.Addr,
			})
			if err := m.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.FatalCF("channels", "Shared HTTP server error", map[string]any{
					"error": err.Error(),
				})
			}
		}()
	}

	logger.InfoC("channels", "All channels started")
	return nil
}

// StopAll 优雅停止所有渠道。关闭顺序：
//  1. 停止共享 HTTP 服务器
//  2. 取消分发任务
//  3. 关闭所有 worker 的消息队列并等待排空
//  4. 关闭所有媒体 worker 的队列并等待排空
//  5. 停止所有渠道
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logger.InfoC("channels", "Stopping all channels")

	// 1. 先关闭共享 HTTP 服务器
	if m.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := m.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("channels", "Shared HTTP server shutdown error", map[string]any{
				"error": err.Error(),
			})
		}
		m.httpServer = nil
	}

	// 2. 取消分发任务
	if m.dispatchTask != nil {
		m.dispatchTask.cancel()
		m.dispatchTask = nil
	}

	// 3. 关闭所有文本 worker 的消息队列并等待排空
	for _, w := range m.workers {
		if w != nil {
			close(w.queue)
		}
	}
	for _, w := range m.workers {
		if w != nil {
			<-w.done
		}
	}
	// 4. 关闭所有媒体 worker 的队列并等待排空
	for _, w := range m.workers {
		if w != nil {
			close(w.mediaQueue)
		}
	}
	for _, w := range m.workers {
		if w != nil {
			<-w.mediaDone
		}
	}

	// 5. 停止所有渠道
	for name, channel := range m.channels {
		logger.InfoCF("channels", "Stopping channel", map[string]any{
			"channel": name,
		})
		if err := channel.Stop(ctx); err != nil {
			logger.ErrorCF("channels", "Error stopping channel", map[string]any{
				"channel": name,
				"error":   err.Error(),
			})
		}
	}

	logger.InfoC("channels", "All channels stopped")
	return nil
}

// newChannelWorker 创建一个带速率限制的渠道 worker。
// 根据渠道名称从 channelRateConfig 查找速率限制配置，未找到则使用默认值。
func newChannelWorker(name string, ch Channel) *channelWorker {
	rateVal := float64(defaultRateLimit)
	if r, ok := channelRateConfig[name]; ok {
		rateVal = r
	}
	burst := int(math.Max(1, math.Ceil(rateVal/2)))

	return &channelWorker{
		ch:         ch,
		queue:      make(chan bus.OutboundMessage, defaultChannelQueueSize),
		mediaQueue: make(chan bus.OutboundMediaMessage, defaultChannelQueueSize),
		done:       make(chan struct{}),
		mediaDone:  make(chan struct{}),
		limiter:    rate.NewLimiter(rate.Limit(rateVal), burst),
	}
}

// runWorker 处理单个渠道的出站消息。
// 消息处理流程：
//  1. SplitByMarker（如果配置启用）—— 基于标记的语义分割
//  2. splitByLength —— 基于渠道最大消息长度的分割（MaxMessageLength）
//  3. 对每个分片调用 sendWithRetry 发送
func (m *Manager) runWorker(ctx context.Context, name string, w *channelWorker) {
	defer close(w.done)
	for {
		select {
		case msg, ok := <-w.queue:
			if !ok {
				return
			}
			maxLen := 0
			if mlp, ok := w.ch.(MessageLengthProvider); ok {
				maxLen = mlp.MaxMessageLength()
			}

			// 收集所有待发送的消息分片
			var chunks []string

			// 步骤 1：尝试基于标记的语义分割（如果配置启用）
			if m.config != nil && m.config.Agents.Defaults.SplitOnMarker {
				if markerChunks := SplitByMarker(msg.Content); len(markerChunks) > 1 {
					for _, chunk := range markerChunks {
						chunks = append(chunks, splitByLength(chunk, maxLen)...)
					}
				}
			}

			// 步骤 2：如果标记分割未产生分片，回退到长度分割
			if len(chunks) == 0 {
				chunks = splitByLength(msg.Content, maxLen)
			}

			// 步骤 3：逐个发送所有分片
			for _, chunk := range chunks {
				chunkMsg := msg
				chunkMsg.Content = chunk
				m.sendWithRetry(ctx, name, w, chunkMsg)
			}
		case <-ctx.Done():
			return
		}
	}
}

// splitByLength 按最大长度分割消息内容。如果内容未超出限制则返回单个分片。
func splitByLength(content string, maxLen int) []string {
	if maxLen > 0 && len([]rune(content)) > maxLen {
		return SplitMessage(content, maxLen)
	}
	return []string{content}
}

// sendWithRetry 带速率限制和重试逻辑的消息发送。
// 错误分类与重试策略：
//   - ErrNotRunning / ErrSendFailed: 永久性错误，不重试
//   - ErrRateLimit: 固定延迟重试
//   - ErrTemporary / 未知错误: 指数退避重试
func (m *Manager) sendWithRetry(ctx context.Context, name string, w *channelWorker, msg bus.OutboundMessage) {
	// 速率限制：等待令牌
	if err := w.limiter.Wait(ctx); err != nil {
		// ctx 已取消，正在关闭
		return
	}

	// 发送前处理：停止 typing，尝试编辑占位消息
	if m.preSend(ctx, name, msg, w.ch) {
		return // 占位消息已编辑成功，跳过 Send
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		lastErr = w.ch.Send(ctx, msg)
		if lastErr == nil {
			return
		}

		// 永久性错误 —— 不重试
		if errors.Is(lastErr, ErrNotRunning) || errors.Is(lastErr, ErrSendFailed) {
			break
		}

		// 已用尽最后一次重试 —— 不再等待
		if attempt == maxRetries {
			break
		}

		// 速率限制错误 —— 固定延迟重试
		if errors.Is(lastErr, ErrRateLimit) {
			select {
			case <-time.After(rateLimitDelay):
				continue
			case <-ctx.Done():
				return
			}
		}

		// ErrTemporary 或未知错误 —— 指数退避重试
		backoff := min(time.Duration(float64(baseBackoff)*math.Pow(2, float64(attempt))), maxBackoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
	}

	// 所有重试用尽或永久性错误
	logger.ErrorCF("channels", "Send failed", map[string]any{
		"channel": name,
		"chat_id": msg.ChatID,
		"error":   lastErr.Error(),
		"retries": maxRetries,
	})
}

// dispatchLoop 泛型消息分发循环。
// 从输入通道读取消息，根据 getChannel 函数获取目标渠道名称，
// 将消息入队到对应渠道的 worker。跳过内部渠道的消息。
func dispatchLoop[M any](
	ctx context.Context,
	m *Manager,
	ch <-chan M,
	getChannel func(M) string,
	enqueue func(context.Context, *channelWorker, M) bool,
	startMsg, stopMsg, unknownMsg, noWorkerMsg string,
) {
	logger.InfoC("channels", startMsg)

	for {
		select {
		case <-ctx.Done():
			logger.InfoC("channels", stopMsg)
			return

		case msg, ok := <-ch:
			if !ok {
				logger.InfoC("channels", stopMsg)
				return
			}

			channel := getChannel(msg)

			// 静默跳过内部渠道
			if constants.IsInternalChannel(channel) {
				continue
			}

			m.mu.RLock()
			_, exists := m.channels[channel]
			w, wExists := m.workers[channel]
			m.mu.RUnlock()

			if !exists {
				logger.WarnCF("channels", unknownMsg, map[string]any{"channel": channel})
				continue
			}

			if wExists && w != nil {
				if !enqueue(ctx, w, msg) {
					return
				}
			} else if exists {
				logger.WarnCF("channels", noWorkerMsg, map[string]any{"channel": channel})
			}
		}
	}
}

// dispatchOutbound 出站文本消息分发循环。
// 从 bus.OutboundChan 读取消息并路由到对应渠道的 worker 队列。
func (m *Manager) dispatchOutbound(ctx context.Context) {
	dispatchLoop(
		ctx, m,
		m.bus.OutboundChan(),
		func(msg bus.OutboundMessage) string { return msg.Channel },
		func(ctx context.Context, w *channelWorker, msg bus.OutboundMessage) bool {
			select {
			case w.queue <- msg:
				return true
			case <-ctx.Done():
				return false
			}
		},
		"Outbound dispatcher started",
		"Outbound dispatcher stopped",
		"Unknown channel for outbound message",
		"Channel has no active worker, skipping message",
	)
}

// dispatchOutboundMedia 出站媒体消息分发循环。
// 从 bus.OutboundMediaChan 读取媒体消息并路由到对应渠道的 worker 媒体队列。
func (m *Manager) dispatchOutboundMedia(ctx context.Context) {
	dispatchLoop(
		ctx, m,
		m.bus.OutboundMediaChan(),
		func(msg bus.OutboundMediaMessage) string { return msg.Channel },
		func(ctx context.Context, w *channelWorker, msg bus.OutboundMediaMessage) bool {
			select {
			case w.mediaQueue <- msg:
				return true
			case <-ctx.Done():
				return false
			}
		},
		"Outbound media dispatcher started",
		"Outbound media dispatcher stopped",
		"Unknown channel for outbound media message",
		"Channel has no active worker, skipping media message",
	)
}

// runMediaWorker 处理单个渠道的出站媒体消息。
func (m *Manager) runMediaWorker(ctx context.Context, name string, w *channelWorker) {
	defer close(w.mediaDone)
	for {
		select {
		case msg, ok := <-w.mediaQueue:
			if !ok {
				return
			}
			_ = m.sendMediaWithRetry(ctx, name, w, msg)
		case <-ctx.Done():
			return
		}
	}
}

// sendMediaWithRetry 带速率限制和重试逻辑的媒体消息发送。
// 成功返回 nil，重试用尽后返回最后的错误（包括渠道不支持 MediaSender 的情况）。
func (m *Manager) sendMediaWithRetry(
	ctx context.Context,
	name string,
	w *channelWorker,
	msg bus.OutboundMediaMessage,
) error {
	ms, ok := w.ch.(MediaSender)
	if !ok {
		err := fmt.Errorf("channel %q does not support media sending", name)
		logger.WarnCF("channels", "Channel does not support MediaSender", map[string]any{
			"channel": name,
			"error":   err.Error(),
		})
		return err
	}

	// 速率限制：等待令牌
	if err := w.limiter.Wait(ctx); err != nil {
		return err
	}

	// 发送前处理：停止 typing 并清理占位消息
	m.preSendMedia(ctx, name, msg, w.ch)

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		lastErr = ms.SendMedia(ctx, msg)
		if lastErr == nil {
			return nil
		}

		// 永久性错误 —— 不重试
		if errors.Is(lastErr, ErrNotRunning) || errors.Is(lastErr, ErrSendFailed) {
			break
		}

		// 已用尽最后一次重试 —— 不再等待
		if attempt == maxRetries {
			break
		}

		// 速率限制错误 —— 固定延迟重试
		if errors.Is(lastErr, ErrRateLimit) {
			select {
			case <-time.After(rateLimitDelay):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// ErrTemporary 或未知错误 —— 指数退避重试
		backoff := min(time.Duration(float64(baseBackoff)*math.Pow(2, float64(attempt))), maxBackoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// 所有重试用尽或永久性错误
	logger.ErrorCF("channels", "SendMedia failed", map[string]any{
		"channel": name,
		"chat_id": msg.ChatID,
		"error":   lastErr.Error(),
		"retries": maxRetries,
	})
	return lastErr
}

// runTTLJanitor TTL 清理定时器。定期扫描 typingStops、reactionUndos 和 placeholders，
// 清除超过 TTL 的条目。防止出站路径未能触发 preSend（如 LLM 错误）时产生内存泄漏。
func (m *Manager) runTTLJanitor(ctx context.Context) {
	ticker := time.NewTicker(janitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			m.typingStops.Range(func(key, value any) bool {
				if entry, ok := value.(typingEntry); ok {
					if now.Sub(entry.createdAt) > typingStopTTL {
						if _, loaded := m.typingStops.LoadAndDelete(key); loaded {
							entry.stop() // 幂等操作，安全
						}
					}
				}
				return true
			})
			m.reactionUndos.Range(func(key, value any) bool {
				if entry, ok := value.(reactionEntry); ok {
					if now.Sub(entry.createdAt) > typingStopTTL {
						if _, loaded := m.reactionUndos.LoadAndDelete(key); loaded {
							entry.undo() // 幂等操作，安全
						}
					}
				}
				return true
			})
			m.placeholders.Range(func(key, value any) bool {
				if entry, ok := value.(placeholderEntry); ok {
					if now.Sub(entry.createdAt) > placeholderTTL {
						m.placeholders.Delete(key)
					}
				}
				return true
			})
		}
	}
}

// GetChannel 按名称获取渠道实例。
func (m *Manager) GetChannel(name string) (Channel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	channel, ok := m.channels[name]
	return channel, ok
}

// GetStatus 返回所有渠道的状态信息（是否启用、是否运行中）。
func (m *Manager) GetStatus() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]any)
	for name, channel := range m.channels {
		status[name] = map[string]any{
			"enabled": true,
			"running": channel.IsRunning(),
		}
	}
	return status
}

// GetEnabledChannels 返回所有已启用渠道的名称列表。
func (m *Manager) GetEnabledChannels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}

// Reload 热重载渠道配置。通过对比配置哈希确定新增和移除的渠道，
// 停止旧渠道、初始化并启动新渠道。如果配置未变更则仅更新配置引用。
// 出错时回滚到旧配置。
func (m *Manager) Reload(ctx context.Context, cfg *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 保存旧配置以便出错时回滚
	oldConfig := m.config

	// 提前更新配置：initChannel 通过 factory(m.config, m.bus) 使用 m.config
	m.config = cfg

	list := toChannelHashes(cfg)
	added, removed := compareChannels(m.channelHashes, list)

	deferFuncs := make([]func(), 0, len(removed)+len(added))
	for _, name := range removed {
		// 停止所有需要移除的渠道
		channel := m.channels[name]
		logger.InfoCF("channels", "Stopping channel", map[string]any{
			"channel": name,
		})
		if err := channel.Stop(ctx); err != nil {
			logger.ErrorCF("channels", "Error stopping channel", map[string]any{
				"channel": name,
				"error":   err.Error(),
			})
		}
		deferFuncs = append(deferFuncs, func() {
			m.UnregisterChannel(name)
		})
	}
	dispatchCtx, cancel := context.WithCancel(ctx)
	m.dispatchTask = &asyncTask{cancel: cancel}
	cc, err := toChannelConfig(cfg, added)
	if err != nil {
		logger.ErrorC("channels", fmt.Sprintf("toChannelConfig error: %v", err))
		m.config = oldConfig
		cancel()
		return err
	}
	err = m.initChannels(cc)
	if err != nil {
		logger.ErrorC("channels", fmt.Sprintf("initChannels error: %v", err))
		m.config = oldConfig
		cancel()
		return err
	}
	for _, name := range added {
		channel := m.channels[name]
		logger.InfoCF("channels", "Starting channel", map[string]any{
			"channel": name,
		})
		if err := channel.Start(ctx); err != nil {
			logger.ErrorCF("channels", "Failed to start channel", map[string]any{
				"channel": name,
				"error":   err.Error(),
			})
			continue
		}
		// 渠道成功启动后才创建 worker（延迟创建）
		w := newChannelWorker(name, channel)
		m.workers[name] = w
		go m.runWorker(dispatchCtx, name, w)
		go m.runMediaWorker(dispatchCtx, name, w)
		deferFuncs = append(deferFuncs, func() {
			m.RegisterChannel(name, channel)
		})
	}

	// 仅在全部成功时提交配置哈希
	m.channelHashes = list
	go func() {
		for _, f := range deferFuncs {
			f()
		}
	}()
	return nil
}

// RegisterChannel 动态注册一个渠道，将其添加到 channels 映射并注册 HTTP 处理器。
func (m *Manager) RegisterChannel(name string, channel Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[name] = channel
	if m.mux != nil {
		m.registerChannelHTTPHandler(name, channel)
	}
}

// UnregisterChannel 动态注销一个渠道。移除 HTTP 处理器、等待 worker 排空、从映射中删除。
func (m *Manager) UnregisterChannel(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ch, ok := m.channels[name]; ok && m.mux != nil {
		m.unregisterChannelHTTPHandler(name, ch)
	}
	if w, ok := m.workers[name]; ok && w != nil {
		close(w.queue)
		<-w.done
		close(w.mediaQueue)
		<-w.mediaDone
	}
	delete(m.workers, name)
	delete(m.channels, name)
}

// SendMessage 同步发送消息，阻塞直到发送完成（或重试用尽）。
// 会按渠道最大消息长度自动分割消息。保证消息顺序，适用于后续操作依赖消息已发送的场景。
func (m *Manager) SendMessage(ctx context.Context, msg bus.OutboundMessage) error {
	m.mu.RLock()
	_, exists := m.channels[msg.Channel]
	w, wExists := m.workers[msg.Channel]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", msg.Channel)
	}
	if !wExists || w == nil {
		return fmt.Errorf("channel %s has no active worker", msg.Channel)
	}

	maxLen := 0
	if mlp, ok := w.ch.(MessageLengthProvider); ok {
		maxLen = mlp.MaxMessageLength()
	}
	if maxLen > 0 && len([]rune(msg.Content)) > maxLen {
		for _, chunk := range SplitMessage(msg.Content, maxLen) {
			chunkMsg := msg
			chunkMsg.Content = chunk
			m.sendWithRetry(ctx, msg.Channel, w, chunkMsg)
		}
	} else {
		m.sendWithRetry(ctx, msg.Channel, w, msg)
	}
	return nil
}

// SendMedia 同步发送媒体消息，阻塞直到发送完成（或重试用尽）。
// 保证媒体发送顺序，适用于后续 agent 行为依赖媒体已发送的场景。
func (m *Manager) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	m.mu.RLock()
	_, exists := m.channels[msg.Channel]
	w, wExists := m.workers[msg.Channel]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", msg.Channel)
	}
	if !wExists || w == nil {
		return fmt.Errorf("channel %s has no active worker", msg.Channel)
	}

	return m.sendMediaWithRetry(ctx, msg.Channel, w, msg)
}

// SendToChannel 异步发送消息到指定渠道。将消息入队到 worker 的消息队列，
// 如果没有活跃的 worker 则直接调用渠道的 Send 方法（回退方案）。
func (m *Manager) SendToChannel(ctx context.Context, channelName, chatID, content string) error {
	m.mu.RLock()
	_, exists := m.channels[channelName]
	w, wExists := m.workers[channelName]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}

	msg := bus.OutboundMessage{
		Channel: channelName,
		ChatID:  chatID,
		Content: content,
	}

	if wExists && w != nil {
		select {
		case w.queue <- msg:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// 回退方案：直接发送（正常情况下不应发生）
	channel, _ := m.channels[channelName]
	return channel.Send(ctx, msg)
}
