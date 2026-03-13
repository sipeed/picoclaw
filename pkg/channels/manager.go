// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package channels

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
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
	defaultChannelQueueSize = 16
	defaultRateLimit        = 10 // default 10 msg/s
	maxRetries              = 3
	rateLimitDelay          = 1 * time.Second
	baseBackoff             = 500 * time.Millisecond
	maxBackoff              = 8 * time.Second

	janitorInterval = 10 * time.Second
	typingStopTTL   = 5 * time.Minute
	placeholderTTL  = 10 * time.Minute
	statusMsgTTL    = 5 * time.Minute
	taskMsgTTL      = 30 * time.Minute

	// statusEditInterval is the minimum interval between EditMessage calls
	// for the same status/task bubble. EditMessage APIs are more rate-sensitive
	// than SendMessageDraft, so we throttle edits to avoid "(edited)" flicker
	// and API rate limit errors. Draft-based channels bypass this throttle.
	statusEditInterval = 500 * time.Millisecond
)

// typingEntry wraps a typing stop function with a creation timestamp for TTL eviction.
type typingEntry struct {
	stop      func()
	createdAt time.Time
}

// reactionEntry wraps a reaction undo function with a creation timestamp for TTL eviction.
type reactionEntry struct {
	undo      func()
	createdAt time.Time
}

// placeholderEntry wraps a placeholder ID with a creation timestamp for TTL eviction.
type placeholderEntry struct {
	id        string
	createdAt time.Time
}

// statusMsgEntry tracks a status or task message ID for later editing.
type statusMsgEntry struct {
	messageID string
	draftID   int // non-zero when using draft-based streaming
	createdAt time.Time
}

// channelRateConfig maps channel name to per-second rate limit.
var channelRateConfig = map[string]float64{
	"telegram": 20,
	"discord":  1,
	"slack":    1,
	"matrix":   2,
	"line":     10,
	"qq":       5,
	"irc":      2,
}

type channelWorker struct {
	ch         Channel
	queue      chan bus.OutboundMessage
	mediaQueue chan bus.OutboundMediaMessage
	done       chan struct{}
	mediaDone  chan struct{}
	limiter    *rate.Limiter
}

type Manager struct {
	channels        map[string]Channel
	workers         map[string]*channelWorker
	bus             *bus.MessageBus
	config          *config.Config
	mediaStore      media.MediaStore
	dispatchTask    *asyncTask
	mu              sync.RWMutex
	placeholders    sync.Map // "channel:chatID" → placeholderEntry
	typingStops     sync.Map // "channel:chatID" → typingEntry
	reactionUndos   sync.Map // "channel:chatID" → reactionEntry
	statusMsgIDs    sync.Map // "channel:chatID" → statusMsgEntry (streaming preview)
	taskMsgIDs      sync.Map // "channel:chatID:taskID" → statusMsgEntry (background task status)
	statusEditTimes sync.Map // key → time.Time — last EditMessage time for throttling
}

type asyncTask struct {
	cancel context.CancelFunc
}

// RecordPlaceholder registers a placeholder message for later editing.
// Implements PlaceholderRecorder.
func (m *Manager) RecordPlaceholder(channel, chatID, placeholderID string) {
	key := channel + ":" + chatID
	m.placeholders.Store(key, placeholderEntry{id: placeholderID, createdAt: time.Now()})
}

// SendPlaceholder sends a "Thinking..." placeholder for the given channel/chatID
// and records it for later editing. Returns true if a placeholder was sent.
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

// RecordTypingStop registers a typing stop function for later invocation.
// Implements PlaceholderRecorder.
//
// If a previous typing indicator exists for the same chat, it is stopped
// immediately before the new one is recorded.  This prevents the next
// preSend (which finalizes the *previous* processing cycle) from
// consuming the *new* message's typing entry.
func (m *Manager) RecordTypingStop(channel, chatID string, stop func()) {
	key := channel + ":" + chatID
	entry := typingEntry{stop: stop, createdAt: time.Now()}
	if previous, loaded := m.typingStops.Swap(key, entry); loaded {
		if oldEntry, ok := previous.(typingEntry); ok && oldEntry.stop != nil {
			oldEntry.stop()
		}
	}
}

// RecordReactionUndo registers a reaction undo function for later invocation.
// Implements PlaceholderRecorder.
//
// If a previous reaction exists for the same chat, it is undone immediately
// before the new one is recorded.  Same rationale as RecordTypingStop: the
// old entry belongs to the previous processing cycle and must not leak into
// the next preSend call.
func (m *Manager) RecordReactionUndo(channel, chatID string, undo func()) {
	key := channel + ":" + chatID
	if v, loaded := m.reactionUndos.Load(key); loaded {
		if entry, ok := v.(reactionEntry); ok {
			entry.undo() // idempotent
		}
	}
	m.reactionUndos.Store(key, reactionEntry{undo: undo, createdAt: time.Now()})
}

// preSend handles typing stop, reaction undo, and placeholder/status editing before sending a message.
// Returns true if the message was edited into an existing message (skip Send).
func (m *Manager) preSend(ctx context.Context, name string, msg bus.OutboundMessage, ch Channel) bool {
	key := name + ":" + msg.ChatID

	// 1. Stop typing
	if v, loaded := m.typingStops.LoadAndDelete(key); loaded {
		if entry, ok := v.(typingEntry); ok {
			entry.stop() // idempotent, safe
		}
	}

	// 2. Undo reaction
	if v, loaded := m.reactionUndos.LoadAndDelete(key); loaded {
		if entry, ok := v.(reactionEntry); ok {
			entry.undo() // idempotent, safe
		}
	}

	// 3. Try editing a tracked status message (from streaming preview).
	// For draft-based entries, explicitly dismiss the draft bubble before
	// sending the permanent message.  Without this, a user message sent
	// between the last draft update and sendMessage may prevent the
	// platform from auto-replacing the draft, leaving a ghost bubble.
	if v, loaded := m.statusMsgIDs.LoadAndDelete(key); loaded {
		if entry, ok := v.(statusMsgEntry); ok {
			if entry.draftID != 0 {
				if drafter, ok := ch.(DraftSender); ok {
					_ = drafter.SendDraft(ctx, msg.ChatID, entry.draftID, "")
				}
				m.statusEditTimes.Delete(key)
			} else if entry.messageID != "" {
				if editor, ok := ch.(MessageEditor); ok {
					if err := editor.EditMessage(ctx, msg.ChatID, entry.messageID, msg.Content); err == nil {
						return true // edited successfully, skip Send
					}
				}
			}
		}
	}

	// 4. Try editing placeholder
	if v, loaded := m.placeholders.LoadAndDelete(key); loaded {
		if entry, ok := v.(placeholderEntry); ok && entry.id != "" {
			if editor, ok := ch.(MessageEditor); ok {
				if err := editor.EditMessage(ctx, msg.ChatID, entry.id, msg.Content); err == nil {
					return true // edited successfully, skip Send
				}
				// edit failed → fall through to normal Send
			}
		}
	}

	return false
}

func NewManager(cfg *config.Config, messageBus *bus.MessageBus, store media.MediaStore) (*Manager, error) {
	m := &Manager{
		channels:   make(map[string]Channel),
		workers:    make(map[string]*channelWorker),
		bus:        messageBus,
		config:     cfg,
		mediaStore: store,
	}

	if err := m.initChannels(); err != nil {
		return nil, err
	}

	return m, nil
}

// initChannel is a helper that looks up a factory by name and creates the channel.
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
		// Inject MediaStore if channel supports it
		if m.mediaStore != nil {
			if setter, ok := ch.(interface{ SetMediaStore(s media.MediaStore) }); ok {
				setter.SetMediaStore(m.mediaStore)
			}
		}
		// Inject PlaceholderRecorder if channel supports it
		if setter, ok := ch.(interface{ SetPlaceholderRecorder(r PlaceholderRecorder) }); ok {
			setter.SetPlaceholderRecorder(m)
		}
		// Inject owner reference so BaseChannel.HandleMessage can auto-trigger typing/reaction
		if setter, ok := ch.(interface{ SetOwner(ch Channel) }); ok {
			setter.SetOwner(ch)
		}
		m.channels[name] = ch
		logger.InfoCF("channels", "Channel enabled successfully", map[string]any{
			"channel": displayName,
		})
	}
}

func (m *Manager) initChannels() error {
	logger.InfoC("channels", "Initializing channel manager")

	if m.config.Channels.Telegram.Enabled && m.config.Channels.Telegram.Token != "" {
		m.initChannel("telegram", "Telegram")
	}

	if m.config.Channels.WhatsApp.Enabled {
		waCfg := m.config.Channels.WhatsApp
		if waCfg.UseNative {
			m.initChannel("whatsapp_native", "WhatsApp Native")
		} else if waCfg.BridgeURL != "" {
			m.initChannel("whatsapp", "WhatsApp")
		}
	}

	if m.config.Channels.Feishu.Enabled {
		m.initChannel("feishu", "Feishu")
	}

	if m.config.Channels.Discord.Enabled && m.config.Channels.Discord.Token != "" {
		m.initChannel("discord", "Discord")
	}

	if m.config.Channels.MaixCam.Enabled {
		m.initChannel("maixcam", "MaixCam")
	}

	if m.config.Channels.QQ.Enabled {
		m.initChannel("qq", "QQ")
	}

	if m.config.Channels.DingTalk.Enabled && m.config.Channels.DingTalk.ClientID != "" {
		m.initChannel("dingtalk", "DingTalk")
	}

	if m.config.Channels.Slack.Enabled && m.config.Channels.Slack.BotToken != "" {
		m.initChannel("slack", "Slack")
	}

	if m.config.Channels.Matrix.Enabled &&
		m.config.Channels.Matrix.Homeserver != "" &&
		m.config.Channels.Matrix.UserID != "" &&
		m.config.Channels.Matrix.AccessToken != "" {
		m.initChannel("matrix", "Matrix")
	}

	if m.config.Channels.LINE.Enabled && m.config.Channels.LINE.ChannelAccessToken != "" {
		m.initChannel("line", "LINE")
	}

	if m.config.Channels.OneBot.Enabled && m.config.Channels.OneBot.WSUrl != "" {
		m.initChannel("onebot", "OneBot")
	}

	if m.config.Channels.WeCom.Enabled && m.config.Channels.WeCom.Token != "" {
		m.initChannel("wecom", "WeCom")
	}

	if m.config.Channels.WeComAIBot.Enabled && m.config.Channels.WeComAIBot.Token != "" {
		m.initChannel("wecom_aibot", "WeCom AI Bot")
	}

	if m.config.Channels.WeComApp.Enabled && m.config.Channels.WeComApp.CorpID != "" {
		m.initChannel("wecom_app", "WeCom App")
	}

	if m.config.Channels.Pico.Enabled && m.config.Channels.Pico.Token != "" {
		m.initChannel("pico", "Pico")
	}

	if m.config.Channels.IRC.Enabled && m.config.Channels.IRC.Server != "" {
		m.initChannel("irc", "IRC")
	}

	logger.InfoCF("channels", "Channel initialization completed", map[string]any{
		"enabled_channels": len(m.channels),
	})

	return nil
}

// SetupHTTPServer registers channel webhook handlers and health checkers
// onto the provided health server's mux. The health server owns the HTTP
// listener; the channel manager no longer creates its own http.Server.
func (m *Manager) SetupHTTPServer(_ string, healthServer *health.Server) {
	if healthServer == nil {
		return
	}
	mux := healthServer.Mux()

	// Discover and register webhook handlers and health checkers
	for name, ch := range m.channels {
		if wh, ok := ch.(WebhookHandler); ok {
			mux.Handle(wh.WebhookPath(), wh)
			logger.InfoCF("channels", "Webhook handler registered", map[string]any{
				"channel": name,
				"path":    wh.WebhookPath(),
			})
		}
		if hc, ok := ch.(HealthChecker); ok {
			mux.HandleFunc(hc.HealthPath(), hc.HealthHandler)
			logger.InfoCF("channels", "Health endpoint registered", map[string]any{
				"channel": name,
				"path":    hc.HealthPath(),
			})
		}
	}
}

func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.channels) == 0 {
		logger.WarnC("channels", "No channels enabled")
		return errors.New("no channels enabled")
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
		// Lazily create worker only after channel starts successfully
		w := newChannelWorker(name, channel)
		m.workers[name] = w
		go m.runWorker(dispatchCtx, name, w)
		go m.runMediaWorker(dispatchCtx, name, w)
	}

	// Start the dispatcher that reads from the bus and routes to workers
	go m.dispatchOutbound(dispatchCtx)
	go m.dispatchOutboundMedia(dispatchCtx)

	// Start the TTL janitor that cleans up stale typing/placeholder entries
	go m.runTTLJanitor(dispatchCtx)

	logger.InfoC("channels", "All channels started")
	return nil
}

func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logger.InfoC("channels", "Stopping all channels")

	// Cancel dispatcher
	if m.dispatchTask != nil {
		m.dispatchTask.cancel()
		m.dispatchTask = nil
	}

	// Close all worker queues and wait for them to drain
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
	// Close all media worker queues and wait for them to drain
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

	// Stop all channels
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

// newChannelWorker creates a channelWorker with a rate limiter configured
// for the given channel name.
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

// runWorker processes outbound messages for a single channel, splitting
// messages that exceed the channel's maximum message length.
func (m *Manager) runWorker(ctx context.Context, name string, w *channelWorker) {
	defer close(w.done)
	for {
		select {
		case msg, ok := <-w.queue:
			if !ok {
				return
			}

			// Route status/task messages to dedicated handlers
			if msg.IsStatus {
				m.handleStatusSend(ctx, name, w, msg)
				continue
			}
			if msg.IsTaskStatus {
				m.handleTaskStatusSend(ctx, name, w, msg)
				continue
			}

			maxLen := 0
			if mlp, ok := w.ch.(MessageLengthProvider); ok {
				maxLen = mlp.MaxMessageLength()
			}
			if maxLen > 0 && len([]rune(msg.Content)) > maxLen {
				chunks := SplitMessage(msg.Content, maxLen)
				for _, chunk := range chunks {
					chunkMsg := msg
					chunkMsg.Content = chunk
					m.sendWithRetry(ctx, name, w, chunkMsg)
				}
			} else {
				m.sendWithRetry(ctx, name, w, msg)
			}
		case <-ctx.Done():
			return
		}
	}
}

// handleStatusSend processes IsStatus messages (streaming previews).
// It reuses an existing placeholder or tracked status message, or sends a new
// one via SendWithID so subsequent status updates edit the same bubble.
// For channels implementing DraftSender (e.g. Telegram private chats),
// sendMessageDraft is preferred as it avoids the "(edited)" indicator.
// If the channel doesn't support editing, the message is silently dropped.
func (m *Manager) handleStatusSend(ctx context.Context, name string, w *channelWorker, msg bus.OutboundMessage) {
	if err := w.limiter.Wait(ctx); err != nil {
		return
	}

	key := name + ":" + msg.ChatID

	// 0. Draft-based streaming (preferred for supported channels)
	if drafter, ok := w.ch.(DraftSender); ok {
		var did int
		if v, loaded := m.statusMsgIDs.Load(key); loaded {
			if entry, ok := v.(statusMsgEntry); ok && entry.draftID != 0 {
				did = entry.draftID
			}
		}
		if did == 0 {
			did = generateDraftID(key)
		}
		if err := drafter.SendDraft(ctx, msg.ChatID, did, msg.Content); err == nil {
			// Track draft only after successful send. If draft fails (e.g. group
			// main thread), keep existing messageID entry so fallback edits can
			// reuse the same status bubble instead of creating duplicates.
			m.statusMsgIDs.Store(key, statusMsgEntry{
				draftID:   did,
				createdAt: time.Now(),
			})
			return
		}
		// Draft failed — fall through to edit-based approach
	}

	// Edit-based path: throttle to statusEditInterval per key to avoid
	// API rate limit errors and "(edited)" flicker.
	if v, loaded := m.statusEditTimes.Load(key); loaded {
		if t, ok := v.(time.Time); ok && time.Since(t) < statusEditInterval {
			return // too recent, skip this update
		}
	}

	// 1. Try editing an existing placeholder
	if v, loaded := m.placeholders.Load(key); loaded {
		if entry, ok := v.(placeholderEntry); ok && entry.id != "" {
			if editor, ok := w.ch.(MessageEditor); ok {
				if err := editor.EditMessage(ctx, msg.ChatID, entry.id, msg.Content); err == nil {
					m.statusEditTimes.Store(key, time.Now())
					return
				}
			}
		}
	}

	// 2. Try editing a previously tracked status message
	if v, loaded := m.statusMsgIDs.Load(key); loaded {
		if entry, ok := v.(statusMsgEntry); ok && entry.messageID != "" {
			if editor, ok := w.ch.(MessageEditor); ok {
				if err := editor.EditMessage(ctx, msg.ChatID, entry.messageID, msg.Content); err == nil {
					m.statusEditTimes.Store(key, time.Now())
					return
				}
			}
		}
	}

	// 3. Send new message via SendWithID and track it
	if sender, ok := w.ch.(MessageSenderWithID); ok {
		if msgID, err := sender.SendWithID(ctx, msg.ChatID, msg.Content); err == nil && msgID != "" {
			m.statusMsgIDs.Store(key, statusMsgEntry{
				messageID: msgID,
				createdAt: time.Now(),
			})
			return
		}
	}

	// 4. Channel doesn't support SendWithID or editing — drop silently
}

func taskStatusKey(channel, chatID, taskID string) string {
	if taskID == "" {
		return ""
	}
	if channel == "" || chatID == "" {
		return taskID
	}
	return channel + ":" + chatID + ":" + taskID
}

// handleTaskStatusSend processes IsTaskStatus messages (background task status).
// It reuses a previously tracked task message, or sends a new one via SendWithID.
// For channels implementing DraftSender, sendMessageDraft is used to avoid "(edited)".
// If the channel doesn't support editing, falls back to regular Send.
func (m *Manager) handleTaskStatusSend(ctx context.Context, name string, w *channelWorker, msg bus.OutboundMessage) {
	if err := w.limiter.Wait(ctx); err != nil {
		return
	}

	taskKey := taskStatusKey(name, msg.ChatID, msg.TaskID)

	// Final message: reuse the existing bubble when possible to avoid
	// duplicate messages. If a permanent message (messageID) is tracked,
	// edit it in-place. If a draft (draftID) is tracked, update it with
	// the completion content (the draft persists in Telegram and serves
	// as the visible message; sending a separate permanent message would
	// create a duplicate).
	if msg.Final {
		v, loaded := m.taskMsgIDs.LoadAndDelete(taskKey)
		m.statusEditTimes.Delete(taskKey)

		if loaded {
			if entry, ok := v.(statusMsgEntry); ok {
				// Path A: a permanent message exists — edit it in-place.
				if entry.messageID != "" {
					if editor, ok := w.ch.(MessageEditor); ok {
						if err := editor.EditMessage(ctx, msg.ChatID, entry.messageID, msg.Content); err == nil {
							return
						}
					}
					// Edit failed — fall through to send a new message.
				}

				// Path B: a draft exists — update it with the final
				// content. Drafts persist visibly in Telegram, so do NOT
				// send a separate permanent message (that causes duplicates).
				if entry.draftID != 0 {
					if drafter, ok := w.ch.(DraftSender); ok {
						if err := drafter.SendDraft(ctx, msg.ChatID, entry.draftID, msg.Content); err == nil {
							return
						}
					}
					// Draft update failed — fall through to send permanent.
				}
			}
		}

		// No existing bubble to reuse — send a new permanent message.
		if sender, ok := w.ch.(MessageSenderWithID); ok {
			if msgID, err := sender.SendWithID(ctx, msg.ChatID, msg.Content); err == nil && msgID != "" {
				return
			}
		}
		_ = w.ch.Send(ctx, msg)
		return
	}

	// 0. Draft-based streaming (preferred for supported channels)
	if drafter, ok := w.ch.(DraftSender); ok && taskKey != "" {
		var did int
		if v, loaded := m.taskMsgIDs.Load(taskKey); loaded {
			if entry, ok := v.(statusMsgEntry); ok && entry.draftID != 0 {
				did = entry.draftID
			}
		}
		if did == 0 {
			did = generateDraftID(taskKey)
		}
		if err := drafter.SendDraft(ctx, msg.ChatID, did, msg.Content); err == nil {
			// Track draft only after successful send to avoid clobbering an
			// existing messageID entry when drafts are unsupported.
			m.taskMsgIDs.Store(taskKey, statusMsgEntry{
				draftID:   did,
				createdAt: time.Now(),
			})
			return
		}
		// Draft failed — fall through to edit-based approach
	}

	// Edit-based path: throttle to statusEditInterval per task key.
	if taskKey != "" {
		if v, loaded := m.statusEditTimes.Load(taskKey); loaded {
			if t, ok := v.(time.Time); ok && time.Since(t) < statusEditInterval {
				return
			}
		}
	}

	// 1. Try editing an existing task message
	if taskKey != "" {
		if v, loaded := m.taskMsgIDs.Load(taskKey); loaded {
			if entry, ok := v.(statusMsgEntry); ok && entry.messageID != "" {
				if editor, ok := w.ch.(MessageEditor); ok {
					if err := editor.EditMessage(ctx, msg.ChatID, entry.messageID, msg.Content); err == nil {
						m.statusEditTimes.Store(taskKey, time.Now())
						return
					}
				}
			}
		}
	}

	// 2. Send new message via SendWithID and track it
	if sender, ok := w.ch.(MessageSenderWithID); ok {
		if msgID, err := sender.SendWithID(ctx, msg.ChatID, msg.Content); err == nil && msgID != "" {
			if taskKey != "" {
				m.taskMsgIDs.Store(taskKey, statusMsgEntry{
					messageID: msgID,
					createdAt: time.Now(),
				})
			}
			return
		}
	}

	// 3. Fallback: regular Send (for channels without SendWithID)
	_ = w.ch.Send(ctx, msg)
}

// generateDraftID produces a stable non-zero int from a key string.
// The same key always maps to the same draft ID so successive calls
// animate the same Telegram draft bubble.
func generateDraftID(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	v := int(h.Sum32())
	if v == 0 {
		v = 1 // draftID must be non-zero
	}
	if v < 0 {
		v = -v
	}
	return v
}

// sendWithRetry sends a message through the channel with rate limiting and
// retry logic. It classifies errors to determine the retry strategy:
//   - ErrNotRunning / ErrSendFailed: permanent, no retry
//   - ErrRateLimit: fixed delay retry
//   - ErrTemporary / unknown: exponential backoff retry
func (m *Manager) sendWithRetry(ctx context.Context, name string, w *channelWorker, msg bus.OutboundMessage) {
	// Rate limit: wait for token
	if err := w.limiter.Wait(ctx); err != nil {
		// ctx canceled, shutting down
		return
	}

	// Pre-send: stop typing and try to edit placeholder
	if m.preSend(ctx, name, msg, w.ch) {
		return // placeholder was edited successfully, skip Send
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		lastErr = w.ch.Send(ctx, msg)
		if lastErr == nil {
			return
		}

		// Permanent failures — don't retry
		if errors.Is(lastErr, ErrNotRunning) || errors.Is(lastErr, ErrSendFailed) {
			break
		}

		// Last attempt exhausted — don't sleep
		if attempt == maxRetries {
			break
		}

		// Rate limit error — fixed delay
		if errors.Is(lastErr, ErrRateLimit) {
			select {
			case <-time.After(rateLimitDelay):
				continue
			case <-ctx.Done():
				return
			}
		}

		// ErrTemporary or unknown error — exponential backoff
		backoff := min(time.Duration(float64(baseBackoff)*math.Pow(2, float64(attempt))), maxBackoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
	}

	// All retries exhausted or permanent failure
	logger.ErrorCF("channels", "Send failed", map[string]any{
		"channel": name,
		"chat_id": msg.ChatID,
		"error":   lastErr.Error(),
		"retries": maxRetries,
	})
}

func dispatchLoop[M any](
	ctx context.Context,
	m *Manager,
	subscribe func(context.Context) (M, bool),
	getChannel func(M) string,
	enqueue func(context.Context, *channelWorker, M) bool,
	startMsg, stopMsg, unknownMsg, noWorkerMsg string,
) {
	logger.InfoC("channels", startMsg)

	for {
		msg, ok := subscribe(ctx)
		if !ok {
			logger.InfoC("channels", stopMsg)
			return
		}

		channel := getChannel(msg)

		// Silently skip internal channels
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

func (m *Manager) dispatchOutbound(ctx context.Context) {
	dispatchLoop(
		ctx, m,
		m.bus.SubscribeOutbound,
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

func (m *Manager) dispatchOutboundMedia(ctx context.Context) {
	dispatchLoop(
		ctx, m,
		m.bus.SubscribeOutboundMedia,
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

// runMediaWorker processes outbound media messages for a single channel.
func (m *Manager) runMediaWorker(ctx context.Context, name string, w *channelWorker) {
	defer close(w.mediaDone)
	for {
		select {
		case msg, ok := <-w.mediaQueue:
			if !ok {
				return
			}
			m.sendMediaWithRetry(ctx, name, w, msg)
		case <-ctx.Done():
			return
		}
	}
}

// sendMediaWithRetry sends a media message through the channel with rate limiting and
// retry logic. If the channel does not implement MediaSender, it silently skips.
func (m *Manager) sendMediaWithRetry(ctx context.Context, name string, w *channelWorker, msg bus.OutboundMediaMessage) {
	ms, ok := w.ch.(MediaSender)
	if !ok {
		logger.DebugCF("channels", "Channel does not support MediaSender, skipping media", map[string]any{
			"channel": name,
		})
		return
	}

	// Rate limit: wait for token
	if err := w.limiter.Wait(ctx); err != nil {
		return
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		lastErr = ms.SendMedia(ctx, msg)
		if lastErr == nil {
			return
		}

		// Permanent failures — don't retry
		if errors.Is(lastErr, ErrNotRunning) || errors.Is(lastErr, ErrSendFailed) {
			break
		}

		// Last attempt exhausted — don't sleep
		if attempt == maxRetries {
			break
		}

		// Rate limit error — fixed delay
		if errors.Is(lastErr, ErrRateLimit) {
			select {
			case <-time.After(rateLimitDelay):
				continue
			case <-ctx.Done():
				return
			}
		}

		// ErrTemporary or unknown error — exponential backoff
		backoff := min(time.Duration(float64(baseBackoff)*math.Pow(2, float64(attempt))), maxBackoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
	}

	// All retries exhausted or permanent failure
	logger.ErrorCF("channels", "SendMedia failed", map[string]any{
		"channel": name,
		"chat_id": msg.ChatID,
		"error":   lastErr.Error(),
		"retries": maxRetries,
	})
}

// runTTLJanitor periodically scans the typingStops and placeholders maps
// and evicts entries that have exceeded their TTL. This prevents memory
// accumulation when outbound paths fail to trigger preSend (e.g. LLM errors).
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
							entry.stop() // idempotent, safe
						}
					}
				}
				return true
			})
			m.reactionUndos.Range(func(key, value any) bool {
				if entry, ok := value.(reactionEntry); ok {
					if now.Sub(entry.createdAt) > typingStopTTL {
						if _, loaded := m.reactionUndos.LoadAndDelete(key); loaded {
							entry.undo() // idempotent, safe
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
			m.statusMsgIDs.Range(func(key, value any) bool {
				if entry, ok := value.(statusMsgEntry); ok {
					if now.Sub(entry.createdAt) > statusMsgTTL {
						m.statusMsgIDs.Delete(key)
					}
				}
				return true
			})
			m.taskMsgIDs.Range(func(key, value any) bool {
				if entry, ok := value.(statusMsgEntry); ok {
					if now.Sub(entry.createdAt) > taskMsgTTL {
						m.taskMsgIDs.Delete(key)
					}
				}
				return true
			})
			// Clean up stale edit-time entries (only needed for a few seconds,
			// but janitor runs infrequently so use a generous TTL).
			m.statusEditTimes.Range(func(key, value any) bool {
				if t, ok := value.(time.Time); ok {
					if now.Sub(t) > statusMsgTTL {
						m.statusEditTimes.Delete(key)
					}
				}
				return true
			})
		}
	}
}

// PromoteStatusToTask moves the tracked streaming status message for the given
// channel:chatID key into the task message map under channel:chatID:taskID. This allows the
// next IsTaskStatus publish to edit the streaming bubble instead of creating a
// new message. Returns true if a status message was found and promoted.
func (m *Manager) PromoteStatusToTask(statusKey, taskID string) bool {
	v, loaded := m.statusMsgIDs.LoadAndDelete(statusKey)
	if !loaded {
		return false
	}

	parts := strings.SplitN(statusKey, ":", 2)
	if len(parts) == 2 {
		m.taskMsgIDs.Store(taskStatusKey(parts[0], parts[1], taskID), v)
		return true
	}

	m.taskMsgIDs.Store(taskID, v)
	return true
}

func (m *Manager) GetChannel(name string) (Channel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	channel, ok := m.channels[name]
	return channel, ok
}

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
	if w, ok := m.workers[name]; ok && w != nil {
		close(w.queue)
		<-w.done
		close(w.mediaQueue)
		<-w.mediaDone
	}
	delete(m.workers, name)
	delete(m.channels, name)
}

// SendMessage sends an outbound message synchronously through the channel
// worker's rate limiter and retry logic. It blocks until the message is
// delivered (or all retries are exhausted), which preserves ordering when
// a subsequent operation depends on the message having been sent.
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

	// Fallback: direct send (should not happen)
	channel, _ := m.channels[channelName]
	return channel.Send(ctx, msg)
}
