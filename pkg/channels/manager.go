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
	"math"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
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
	channels      map[string]Channel
	workers       map[string]*channelWorker
	bus           *bus.MessageBus
	runtimeEvents runtimeevents.Bus
	config        *config.Config
	mediaStore    media.MediaStore
	dispatchTask  *asyncTask
	mux           *dynamicServeMux
	httpServer    *http.Server
	httpListeners []net.Listener
	mu            sync.RWMutex
	placeholders  sync.Map          // "channel:chatID" → placeholderID (string)
	typingStops   sync.Map          // "channel:chatID" → func()
	reactionUndos sync.Map          // "channel:chatID" → reactionEntry
	streamActive  sync.Map          // "channel:chatID" → true (set when streamer.Finalize sent the message)
	channelHashes map[string]string // channel name → config hash
}

// ManagerOption configures a channel Manager.
type ManagerOption func(*Manager)

// WithRuntimeEvents injects the runtime event bus used for channel observations.
func WithRuntimeEvents(eventBus runtimeevents.Bus) ManagerOption {
	return func(m *Manager) {
		m.runtimeEvents = eventBus
	}
}

// ChannelLifecyclePayload describes channel lifecycle runtime events.
type ChannelLifecyclePayload struct {
	Type  string `json:"type,omitempty"`
	Error string `json:"error,omitempty"`
}

// ChannelOutboundPayload describes channel outbound message runtime events.
type ChannelOutboundPayload struct {
	Media            bool     `json:"media,omitempty"`
	ContentLen       int      `json:"content_len,omitempty"`
	MessageIDs       []string `json:"message_ids,omitempty"`
	ReplyToMessageID string   `json:"reply_to_message_id,omitempty"`
	Error            string   `json:"error,omitempty"`
	Retries          int      `json:"retries,omitempty"`
}

type toolFeedbackMessageTracker interface {
	RecordToolFeedbackMessage(chatID, messageID, content string)
	ClearToolFeedbackMessage(chatID string)
}

type toolFeedbackMessageCleaner interface {
	DismissToolFeedbackMessage(ctx context.Context, chatID string)
}

type toolFeedbackMessageTargetResolver interface {
	ToolFeedbackMessageChatID(chatID string, outboundCtx *bus.InboundContext) string
}

type toolFeedbackMessageContentPreparer interface {
	PrepareToolFeedbackMessageContent(content string) string
}

type asyncTask struct {
	cancel context.CancelFunc
}

func outboundMessageChannel(msg bus.OutboundMessage) string {
	return msg.Context.Channel
}

func outboundMessageChatID(msg bus.OutboundMessage) string {
	return msg.ChatID
}

func outboundMessageIsToolFeedback(msg bus.OutboundMessage) bool {
	if len(msg.Context.Raw) == 0 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(msg.Context.Raw["message_kind"]), "tool_feedback")
}

func outboundMessageBypassesPlaceholderEdit(msg bus.OutboundMessage) bool {
	if len(msg.Context.Raw) == 0 {
		return false
	}
	kind := strings.TrimSpace(msg.Context.Raw["message_kind"])
	return strings.EqualFold(kind, "thought") ||
		strings.EqualFold(kind, "tool_calls") ||
		strings.EqualFold(kind, "final_reply")
}

func outboundMediaChannel(msg bus.OutboundMediaMessage) string {
	return msg.Context.Channel
}

func outboundMediaChatID(msg bus.OutboundMediaMessage) string {
	return msg.ChatID
}

func trackedToolFeedbackMessageChatID(ch Channel, chatID string, outboundCtx *bus.InboundContext) string {
	if resolver, ok := ch.(toolFeedbackMessageTargetResolver); ok {
		if resolved := strings.TrimSpace(resolver.ToolFeedbackMessageChatID(chatID, outboundCtx)); resolved != "" {
			return resolved
		}
	}
	return strings.TrimSpace(chatID)
}

func dismissTrackedToolFeedbackMessage(
	ctx context.Context,
	ch Channel,
	chatID string,
	outboundCtx *bus.InboundContext,
) {
	trackedChatID := trackedToolFeedbackMessageChatID(ch, chatID, outboundCtx)
	if trackedChatID == "" {
		return
	}
	if cleaner, ok := ch.(toolFeedbackMessageCleaner); ok {
		cleaner.DismissToolFeedbackMessage(ctx, trackedChatID)
		return
	}
	if tracker, ok := ch.(toolFeedbackMessageTracker); ok {
		tracker.ClearToolFeedbackMessage(trackedChatID)
	}
}

func clearTrackedToolFeedbackMessage(
	ch Channel,
	chatID string,
	outboundCtx *bus.InboundContext,
) {
	trackedChatID := trackedToolFeedbackMessageChatID(ch, chatID, outboundCtx)
	if trackedChatID == "" {
		return
	}
	if tracker, ok := ch.(toolFeedbackMessageTracker); ok {
		tracker.ClearToolFeedbackMessage(trackedChatID)
	}
}

// DismissToolFeedback clears any tracked tool feedback animation for the
// given channel/chat. This is called when a turn ends without a final
// response (e.g., ResponseHandled tools) to stop orphaned animation goroutines.
// outboundCtx carries topic/thread info for channels that use scoped tracker
// keys (e.g., Telegram forum topics); may be nil for non-topic channels.
func (m *Manager) DismissToolFeedback(
	ctx context.Context, channelName, chatID string, outboundCtx *bus.InboundContext,
) {
	ch, ok := m.GetChannel(channelName)
	if !ok {
		return
	}
	dismissTrackedToolFeedbackMessage(ctx, ch, chatID, outboundCtx)
}

func prepareToolFeedbackMessageContent(ch Channel, content string) string {
	prepared := strings.TrimSpace(content)
	if prepared == "" {
		return ""
	}
	if preparer, ok := ch.(toolFeedbackMessageContentPreparer); ok {
		if candidate := strings.TrimSpace(preparer.PrepareToolFeedbackMessageContent(prepared)); candidate != "" {
			return candidate
		}
	}
	return prepared
}

func (m *Manager) toolFeedbackSeparateMessagesEnabled() bool {
	if m == nil || m.config == nil {
		return false
	}
	return m.config.Agents.Defaults.IsToolFeedbackSeparateMessagesEnabled()
}

// RecordPlaceholder registers a placeholder message for later editing.
// Implements PlaceholderRecorder.
func (m *Manager) RecordPlaceholder(channel, chatID, placeholderID string) {
	key := channel + ":" + chatID
	m.placeholders.Store(key, placeholderEntry{id: placeholderID, createdAt: time.Now()})
}

// SendPlaceholder sends a "Thinking…" placeholder for the given channel/chatID
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
func (m *Manager) RecordTypingStop(channel, chatID string, stop func()) {
	key := channel + ":" + chatID
	entry := typingEntry{stop: stop, createdAt: time.Now()}
	if previous, loaded := m.typingStops.Swap(key, entry); loaded {
		if oldEntry, ok := previous.(typingEntry); ok && oldEntry.stop != nil {
			oldEntry.stop()
		}
	}
}

// InvokeTypingStop invokes the registered typing stop function for the given channel and chatID.
// It is safe to call even when no typing indicator is active (no-op).
// Used by the agent loop to stop typing when processing completes (success, error, or panic),
// regardless of whether an outbound message is published.
func (m *Manager) InvokeTypingStop(channel, chatID string) {
	key := channel + ":" + chatID
	if v, loaded := m.typingStops.LoadAndDelete(key); loaded {
		if entry, ok := v.(typingEntry); ok {
			entry.stop()
		}
	}
}

// RecordReactionUndo registers a reaction undo function for later invocation.
// Implements PlaceholderRecorder.
func (m *Manager) RecordReactionUndo(channel, chatID string, undo func()) {
	key := channel + ":" + chatID
	m.reactionUndos.Store(key, reactionEntry{undo: undo, createdAt: time.Now()})
}

// preSend handles typing stop, reaction undo, and placeholder editing before sending a message.
// Returns the delivered message IDs and true when delivery completed before a normal Send.
func (m *Manager) preSend(ctx context.Context, name string, msg bus.OutboundMessage, ch Channel) ([]string, bool) {
	chatID := outboundMessageChatID(msg)
	key := name + ":" + chatID

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

	isToolFeedback := outboundMessageIsToolFeedback(msg)
	separateToolFeedbackMessages := m.toolFeedbackSeparateMessagesEnabled()

	// 3. If a stream already finalized this chat, stale tool feedback must be
	// dropped without consuming the final-response marker. Streaming finalization
	// bypasses the worker queue, so older queued feedback can arrive before the
	// normal final outbound message that cleans up the marker and placeholder.
	if isToolFeedback {
		if _, loaded := m.streamActive.Load(key); loaded {
			return nil, true
		}
	}

	// 4. If a stream already finalized this message, delete the placeholder and skip send
	if _, loaded := m.streamActive.LoadAndDelete(key); loaded {
		if v, loaded := m.placeholders.LoadAndDelete(key); loaded {
			if entry, ok := v.(placeholderEntry); ok && entry.id != "" {
				// Prefer deleting the placeholder (cleaner UX than editing to same content)
				if deleter, ok := ch.(MessageDeleter); ok {
					deleter.DeleteMessage(ctx, chatID, entry.id) // best effort
				} else if editor, ok := ch.(MessageEditor); ok {
					editor.EditMessage(ctx, chatID, entry.id, msg.Content) // fallback
				}
			}
		}
		if !isToolFeedback {
			if separateToolFeedbackMessages {
				clearTrackedToolFeedbackMessage(ch, chatID, &msg.Context)
			} else {
				dismissTrackedToolFeedbackMessage(ctx, ch, chatID, &msg.Context)
			}
		}
		return nil, true
	}

	if separateToolFeedbackMessages {
		clearTrackedToolFeedbackMessage(ch, chatID, &msg.Context)
	}

	// 5. Try editing placeholder
	if v, loaded := m.placeholders.LoadAndDelete(key); loaded {
		if entry, ok := v.(placeholderEntry); ok && entry.id != "" {
			if isToolFeedback && separateToolFeedbackMessages {
				if deleter, ok := ch.(MessageDeleter); ok {
					deleter.DeleteMessage(ctx, chatID, entry.id) // best effort
				}
				return nil, false
			}
			if outboundMessageBypassesPlaceholderEdit(msg) {
				if deleter, ok := ch.(MessageDeleter); ok {
					deleter.DeleteMessage(ctx, chatID, entry.id) // best effort
				}
				if strings.EqualFold(strings.TrimSpace(msg.Context.Raw["message_kind"]), "final_reply") {
					dismissTrackedToolFeedbackMessage(ctx, ch, chatID, &msg.Context)
				}
				return nil, false
			}
			if editor, ok := ch.(MessageEditor); ok {
				content := msg.Content
				trackedContent := msg.Content
				if isToolFeedback {
					trackedContent = prepareToolFeedbackMessageContent(ch, msg.Content)
					content = InitialAnimatedToolFeedbackContent(trackedContent)
				}
				if err := editor.EditMessage(ctx, chatID, entry.id, content); err == nil {
					trackedChatID := trackedToolFeedbackMessageChatID(ch, chatID, &msg.Context)
					if tracker, ok := ch.(toolFeedbackMessageTracker); ok && isToolFeedback {
						tracker.RecordToolFeedbackMessage(trackedChatID, entry.id, trackedContent)
					} else if !isToolFeedback {
						dismissTrackedToolFeedbackMessage(ctx, ch, chatID, &msg.Context)
					}
					return []string{entry.id}, true
				}
				// edit failed → fall through to normal Send
			}
		}
	}

	return nil, false
}

// preSendMedia handles typing stop, reaction undo, and placeholder cleanup
// before sending media attachments. Unlike preSend for text messages, media
// delivery never edits the placeholder because there is no text payload to
// replace it with; it only attempts to delete the placeholder when possible.
func (m *Manager) preSendMedia(ctx context.Context, name string, msg bus.OutboundMediaMessage, ch Channel) {
	chatID := outboundMediaChatID(msg)
	key := name + ":" + chatID

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

	// 3. Clear any finalized stream marker for this chat before media delivery.
	m.streamActive.LoadAndDelete(key)

	if m.toolFeedbackSeparateMessagesEnabled() {
		clearTrackedToolFeedbackMessage(ch, chatID, &msg.Context)
	}

	// 4. Delete placeholder if present.
	if v, loaded := m.placeholders.LoadAndDelete(key); loaded {
		if entry, ok := v.(placeholderEntry); ok && entry.id != "" {
			if deleter, ok := ch.(MessageDeleter); ok {
				deleter.DeleteMessage(ctx, chatID, entry.id) // best effort
			}
		}
	}
}

func NewManager(
	cfg *config.Config,
	messageBus *bus.MessageBus,
	store media.MediaStore,
	opts ...ManagerOption,
) (*Manager, error) {
	m := &Manager{
		channels:      make(map[string]Channel),
		workers:       make(map[string]*channelWorker),
		bus:           messageBus,
		config:        cfg,
		mediaStore:    store,
		channelHashes: make(map[string]string),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}

	// Register as streaming delegate so the agent loop can obtain streamers
	messageBus.SetStreamDelegate(m)

	if err := m.initChannels(&cfg.Channels); err != nil {
		return nil, err
	}

	// Store initial config hashes for all channels
	m.channelHashes = toChannelHashes(cfg)

	return m, nil
}

// GetStreamer implements bus.StreamDelegate.
// It checks if the named channel supports streaming and returns a Streamer.
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

	// Mark streamActive on Finalize so preSend knows to clean up the placeholder
	key := channelName + ":" + chatID
	return &finalizeHookStreamer{
		Streamer: streamer,
		onFinalize: func(finalizeCtx context.Context) {
			if m.toolFeedbackSeparateMessagesEnabled() {
				clearTrackedToolFeedbackMessage(
					ch,
					chatID,
					&bus.InboundContext{
						Channel: channelName,
						ChatID:  chatID,
					},
				)
			} else {
				dismissTrackedToolFeedbackMessage(
					finalizeCtx,
					ch,
					chatID,
					&bus.InboundContext{
						Channel: channelName,
						ChatID:  chatID,
					},
				)
			}
			m.streamActive.Store(key, true)
		},
	}, true
}

// finalizeHookStreamer wraps a Streamer to run a hook on Finalize.
type finalizeHookStreamer struct {
	Streamer
	onFinalize func(context.Context)
}

func (s *finalizeHookStreamer) Finalize(ctx context.Context, content string) error {
	if err := s.Streamer.Finalize(ctx, content); err != nil {
		return err
	}
	if s.onFinalize != nil {
		s.onFinalize(ctx)
	}
	return nil
}

// initChannel is a helper that looks up a factory by type name and creates the channel.
// typeName is the channel type used for factory lookup (e.g., "telegram").
// channelName is the config map key used as the channel's runtime name (e.g., "my_telegram").
func (m *Manager) initChannel(typeName, channelName string) {
	f, ok := getFactory(typeName)
	if !ok {
		logger.WarnCF("channels", "Factory not registered", map[string]any{
			"channel": channelName,
			"type":    typeName,
		})
		return
	}
	logger.DebugCF("channels", "Attempting to initialize channel", map[string]any{
		"channel": channelName,
		"type":    typeName,
	})
	ch, err := f(channelName, typeName, m.config, m.bus)
	if err != nil {
		logger.ErrorCF("channels", "Failed to initialize channel", map[string]any{
			"channel": channelName,
			"type":    typeName,
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
		m.channels[channelName] = ch
		m.publishChannelEvent(
			runtimeevents.KindChannelLifecycleInitialized,
			channelName,
			runtimeevents.Scope{Channel: channelName},
			runtimeevents.SeverityInfo,
			ChannelLifecyclePayload{Type: typeName},
		)
		logger.InfoCF("channels", "Channel enabled successfully", map[string]any{
			"channel": channelName,
			"type":    typeName,
		})
	}
}

func (m *Manager) getChannelConfigAndEnabled(channelName string) (*config.Channel, bool) {
	bc, ok := m.config.Channels[channelName]
	if !ok || bc == nil {
		return nil, false
	}
	if !bc.Enabled {
		return bc, false
	}

	// Use Type to determine the config struct for validation.
	// The map key (channelName) is the config key, which may differ from the type.
	channelType := bc.Type
	if channelType == "" {
		channelType = channelName
	}

	// Settings have already been decoded by InitChannelList, so we just need to
	// type-assert and check the relevant fields.
	decoded, err := bc.GetDecoded()
	if err != nil {
		return bc, false
	}
	//nolint:revive
	switch settings := decoded.(type) {
	case *config.WhatsAppSettings:
		if channelType == config.ChannelWhatsApp {
			return bc, settings.BridgeURL != ""
		}
		return bc, channelType == config.ChannelWhatsAppNative && settings.UseNative
	case *config.MatrixSettings:
		return bc, settings.Homeserver != "" && settings.UserID != "" && settings.AccessToken.String() != ""
	case *config.WeComSettings:
		return bc, settings.BotID != "" && settings.Secret.String() != ""
	case *config.PicoClientSettings:
		return bc, settings.URL != ""
	case *config.DingTalkSettings:
		return bc, settings.ClientID != ""
	case *config.SlackSettings:
		return bc, settings.BotToken.String() != ""
	case *config.WeixinSettings:
		return bc, settings.Token.String() != ""
	case *config.PicoSettings:
		return bc, settings.Token.String() != ""
	case *config.IRCSettings:
		return bc, settings.Server != ""
	case *config.LINESettings:
		return bc, settings.ChannelAccessToken.String() != ""
	case *config.OneBotSettings:
		return bc, settings.WSUrl != ""
	case *config.QQSettings:
		return bc, settings.AppSecret.String() != ""
	case *config.TelegramSettings:
		return bc, settings.Token.String() != ""
	case *config.FeishuSettings:
		return bc, settings.AppSecret.String() != ""
	case *config.MaixCamSettings:
		return bc, true
	case *config.TeamsWebhookSettings:
		return bc, true
	case *config.DiscordSettings:
		return bc, settings.Token.String() != ""
	case *config.VKSettings:
		return bc, settings.GroupID != 0 && settings.Token.String() != ""
	case *config.MQTTSettings:
		return bc, settings.Broker != "" && settings.AgentID != ""
	}

	return bc, bc.Enabled
}

// initChannels initializes all enabled channels based on the configuration.
// It iterates config entries and uses bc.Type to look up the appropriate factory.
func (m *Manager) initChannels(channels *config.ChannelsConfig) error {
	logger.InfoC("channels", "Initializing channel manager")

	for name, bc := range *channels {
		if !bc.Enabled {
			continue
		}
		_, ready := m.getChannelConfigAndEnabled(name)
		if !ready {
			continue
		}
		typeName := bc.Type
		if typeName == "" {
			typeName = name
		}
		m.initChannel(typeName, name)
	}

	logger.InfoCF("channels", "Channel initialization completed", map[string]any{
		"enabled_channels": len(m.channels),
	})

	return nil
}

// SetupHTTPServer creates a shared HTTP server with the given listen address.
// It registers health endpoints from the health server and discovers channels
// that implement WebhookHandler and/or HealthChecker to register their handlers.
func (m *Manager) SetupHTTPServer(addr string, healthServer *health.Server) {
	m.SetupHTTPServerListeners(nil, addr, healthServer)
}

// SetupHTTPServerListeners creates a shared HTTP server on pre-opened listeners.
// When listeners is empty it falls back to Addr-based ListenAndServe behavior.
func (m *Manager) SetupHTTPServerListeners(listeners []net.Listener, addr string, healthServer *health.Server) {
	m.mux = newDynamicServeMux()

	// Register health endpoints
	if healthServer != nil {
		healthServer.RegisterOnMux(m.mux)
	}

	// Discover and register webhook handlers and health checkers
	m.registerHTTPHandlersLocked()

	m.httpServer = &http.Server{
		Addr:         addr,
		Handler:      m.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	m.httpListeners = append([]net.Listener(nil), listeners...)
}

// registerHTTPHandlersLocked registers webhook and health-check handlers for
// all channels currently in m.channels. Caller must hold m.mu (or ensure
// exclusive access).
func (m *Manager) registerHTTPHandlersLocked() {
	for name, ch := range m.channels {
		m.registerChannelHTTPHandler(name, ch)
	}
}

// registerChannelHTTPHandler registers the webhook/health handlers for a
// single channel onto m.mux.
func (m *Manager) registerChannelHTTPHandler(name string, ch Channel) {
	if wh, ok := ch.(WebhookHandler); ok {
		m.mux.Handle(wh.WebhookPath(), wh)
		m.publishChannelEvent(
			runtimeevents.KindChannelWebhookRegistered,
			name,
			runtimeevents.Scope{Channel: name},
			runtimeevents.SeverityInfo,
			ChannelLifecyclePayload{Type: channelTypeForEvent(m, name)},
		)
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

// unregisterChannelHTTPHandler removes the webhook/health handlers for a
// single channel from m.mux.
func (m *Manager) unregisterChannelHTTPHandler(name string, ch Channel) {
	if wh, ok := ch.(WebhookHandler); ok {
		m.mux.Unhandle(wh.WebhookPath())
		m.publishChannelEvent(
			runtimeevents.KindChannelWebhookUnregistered,
			name,
			runtimeevents.Scope{Channel: name},
			runtimeevents.SeverityInfo,
			ChannelLifecyclePayload{Type: channelTypeForEvent(m, name)},
		)
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

func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.channels) == 0 {
		logger.WarnC("channels", "No channels enabled")
	}

	logger.InfoC("channels", "Starting all channels")

	dispatchCtx, cancel := context.WithCancel(ctx)
	m.dispatchTask = &asyncTask{cancel: cancel}
	failedStarts := make([]error, 0, len(m.channels))
	failedNames := make([]string, 0, len(m.channels))

	for name, channel := range m.channels {
		logger.InfoCF("channels", "Starting channel", map[string]any{
			"channel": name,
		})
		if err := channel.Start(ctx); err != nil {
			logger.ErrorCF("channels", "Failed to start channel", map[string]any{
				"channel": name,
				"error":   err.Error(),
			})
			m.publishChannelEvent(
				runtimeevents.KindChannelLifecycleStartFailed,
				name,
				runtimeevents.Scope{Channel: name},
				runtimeevents.SeverityError,
				ChannelLifecyclePayload{Type: channelTypeForEvent(m, name), Error: err.Error()},
			)
			failedStarts = append(failedStarts, fmt.Errorf("channel %s: %w", name, err))
			failedNames = append(failedNames, name)
			continue
		}
		// Lazily create worker only after channel starts successfully
		channelType := name
		if m.config != nil {
			if bc := m.config.Channels.Get(name); bc != nil && bc.Type != "" {
				channelType = bc.Type
			}
		}
		w := newChannelWorker(name, channel, channelType)
		m.workers[name] = w
		go m.runWorker(dispatchCtx, name, w)
		go m.runMediaWorker(dispatchCtx, name, w)
		m.publishChannelEvent(
			runtimeevents.KindChannelLifecycleStarted,
			name,
			runtimeevents.Scope{Channel: name},
			runtimeevents.SeverityInfo,
			ChannelLifecyclePayload{Type: channelType},
		)
	}

	if len(m.channels) > 0 && len(m.workers) == 0 {
		if m.dispatchTask != nil {
			m.dispatchTask.cancel()
			m.dispatchTask = nil
		}

		sort.Strings(failedNames)
		if len(failedStarts) == 0 {
			return fmt.Errorf("failed to start any enabled channels")
		}

		logger.ErrorCF("channels", "All enabled channels failed to start", map[string]any{
			"failed":          len(failedNames),
			"total":           len(m.channels),
			"failed_channels": failedNames,
		})

		return fmt.Errorf("failed to start any enabled channels: %w", errors.Join(failedStarts...))
	}

	if len(failedNames) > 0 {
		sort.Strings(failedNames)
		logger.WarnCF("channels", "Some channels failed to start", map[string]any{
			"failed":          len(failedNames),
			"started":         len(m.workers),
			"total":           len(m.channels),
			"failed_channels": failedNames,
		})
	}

	// Start the dispatcher that reads from the bus and routes to workers
	go m.dispatchOutbound(dispatchCtx)
	go m.dispatchOutboundMedia(dispatchCtx)

	// Start the TTL janitor that cleans up stale typing/placeholder entries
	go m.runTTLJanitor(dispatchCtx)

	// Start shared HTTP server if configured
	if m.httpServer != nil {
		if len(m.httpListeners) > 0 {
			for _, listener := range m.httpListeners {
				ln := listener
				go func() {
					logger.InfoCF("channels", "Shared HTTP server listening", map[string]any{
						"addr": ln.Addr().String(),
					})
					if err := m.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
						logger.FatalCF("channels", "Shared HTTP server error", map[string]any{
							"addr":  ln.Addr().String(),
							"error": err.Error(),
						})
					}
				}()
			}
		} else {
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
	}

	logger.InfoCF("channels", "Channel startup completed", map[string]any{
		"started": len(m.workers),
		"failed":  len(failedNames),
		"total":   len(m.channels),
	})
	return nil
}

func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	logger.InfoC("channels", "Stopping all channels")

	// Shutdown shared HTTP server first
	if m.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := m.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("channels", "Shared HTTP server shutdown error", map[string]any{
				"error": err.Error(),
			})
		}
		m.httpServer = nil
		m.httpListeners = nil
	}

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
			continue
		}
		m.publishChannelEvent(
			runtimeevents.KindChannelLifecycleStopped,
			name,
			runtimeevents.Scope{Channel: name},
			runtimeevents.SeverityInfo,
			ChannelLifecyclePayload{Type: channelTypeForEvent(m, name)},
		)
	}

	logger.InfoC("channels", "All channels stopped")
	return nil
}

// newChannelWorker creates a channelWorker with a rate limiter configured
// for the given channel type. channelType is used for rate limit lookup.
func newChannelWorker(name string, ch Channel, channelType string) *channelWorker {
	rateVal := float64(defaultRateLimit)
	if r, ok := channelRateConfig[channelType]; ok {
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

// runWorker processes outbound messages for a single channel.
// Message processing follows this order:
//  1. SplitByMarker (if enabled in config) - LLM semantic marker-based splitting
//  2. SplitMessage - channel-specific length-based splitting (MaxMessageLength)
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

			// Collect all message chunks to send
			var chunks []string

			// Step 1: Try marker-based splitting if enabled.
			// Tool feedback must stay a single message, so it skips marker splitting.
			if m.config != nil && m.config.Agents.Defaults.SplitOnMarker && !outboundMessageIsToolFeedback(msg) {
				if markerChunks := SplitByMarker(msg.Content); len(markerChunks) > 1 {
					for _, chunk := range markerChunks {
						chunkMsg := msg
						chunkMsg.Content = chunk
						chunks = append(chunks, splitOutboundMessageContent(chunkMsg, maxLen)...)
					}
				}
			}

			// Step 2: Fallback to length-based splitting if no chunks from marker
			if len(chunks) == 0 {
				chunks = splitOutboundMessageContent(msg, maxLen)
			}

			// Step 3: Send all chunks
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

// splitOutboundMessageContent splits regular outbound content by maxLen, but
// keeps tool feedback in a single message by truncating the explanation body.
func splitOutboundMessageContent(msg bus.OutboundMessage, maxLen int) []string {
	if maxLen > 0 {
		if outboundMessageIsToolFeedback(msg) {
			animationSafeLen := maxLen - MaxToolFeedbackAnimationFrameLength()
			if animationSafeLen <= 0 {
				animationSafeLen = maxLen
			}
			if len([]rune(msg.Content)) > animationSafeLen {
				return []string{utils.FitToolFeedbackMessage(msg.Content, animationSafeLen)}
			}
			return []string{msg.Content}
		}
		if len([]rune(msg.Content)) > maxLen {
			return SplitMessage(msg.Content, maxLen)
		}
	}
	return []string{msg.Content}
}

// sendWithRetry sends a message through the channel with rate limiting and
// retry logic. It classifies errors to determine the retry strategy:
//   - ErrNotRunning / ErrSendFailed: permanent, no retry
//   - ErrRateLimit: fixed delay retry
//   - ErrTemporary / unknown: exponential backoff retry
func (m *Manager) sendWithRetry(
	ctx context.Context,
	name string,
	w *channelWorker,
	msg bus.OutboundMessage,
) ([]string, bool) {
	// Rate limit: wait for token
	if err := w.limiter.Wait(ctx); err != nil {
		// ctx canceled, shutting down
		m.publishChannelEvent(
			runtimeevents.KindChannelRateLimited,
			name,
			scopeFromOutboundContext(msg.Context),
			runtimeevents.SeverityWarn,
			ChannelOutboundPayload{
				ContentLen:       len([]rune(msg.Content)),
				ReplyToMessageID: msg.ReplyToMessageID,
				Error:            err.Error(),
			},
		)
		return nil, false
	}

	// Pre-send: stop typing and try to edit placeholder
	if msgIDs, handled := m.preSend(ctx, name, msg, w.ch); handled {
		m.publishOutboundSent(name, msg, msgIDs)
		return msgIDs, true
	}

	var lastErr error
	var msgIDs []string
	for attempt := 0; attempt <= maxRetries; attempt++ {
		msgIDs, lastErr = w.ch.Send(ctx, msg)
		if lastErr == nil {
			m.publishOutboundSent(name, msg, msgIDs)
			return msgIDs, true
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
				return nil, false
			}
		}

		// ErrTemporary or unknown error — exponential backoff
		backoff := min(time.Duration(float64(baseBackoff)*math.Pow(2, float64(attempt))), maxBackoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return nil, false
		}
	}

	// All retries exhausted or permanent failure
	logger.ErrorCF("channels", "Send failed", map[string]any{
		"channel": name,
		"chat_id": outboundMessageChatID(msg),
		"error":   lastErr.Error(),
		"retries": maxRetries,
	})
	m.publishOutboundFailed(name, msg, lastErr, false)

	return nil, false
}

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
}

func (m *Manager) dispatchOutbound(ctx context.Context) {
	dispatchLoop(
		ctx, m,
		m.bus.OutboundChan(),
		func(msg bus.OutboundMessage) string { return outboundMessageChannel(msg) },
		func(ctx context.Context, w *channelWorker, msg bus.OutboundMessage) bool {
			select {
			case w.queue <- msg:
				m.publishOutboundQueued(outboundMessageChannel(msg), msg)
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
		m.bus.OutboundMediaChan(),
		func(msg bus.OutboundMediaMessage) string { return outboundMediaChannel(msg) },
		func(ctx context.Context, w *channelWorker, msg bus.OutboundMediaMessage) bool {
			select {
			case w.mediaQueue <- msg:
				m.publishOutboundMediaQueued(outboundMediaChannel(msg), msg)
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
			_, _ = m.sendMediaWithRetry(ctx, name, w, msg)
		case <-ctx.Done():
			return
		}
	}
}

// sendMediaWithRetry sends a media message through the channel with rate limiting and
// retry logic. It returns the message IDs and nil on success, or nil and the last error
// after retries, including when the channel does not support MediaSender.
func (m *Manager) sendMediaWithRetry(
	ctx context.Context,
	name string,
	w *channelWorker,
	msg bus.OutboundMediaMessage,
) ([]string, error) {
	ms, ok := w.ch.(MediaSender)
	if !ok {
		err := fmt.Errorf("channel %q does not support media sending", name)
		logger.WarnCF("channels", "Channel does not support MediaSender", map[string]any{
			"channel": name,
			"error":   err.Error(),
		})
		return nil, err
	}

	// Rate limit: wait for token
	if err := w.limiter.Wait(ctx); err != nil {
		m.publishChannelEvent(
			runtimeevents.KindChannelRateLimited,
			name,
			scopeFromOutboundContext(msg.Context),
			runtimeevents.SeverityWarn,
			ChannelOutboundPayload{
				Media: true,
				Error: err.Error(),
			},
		)
		return nil, err
	}

	// Pre-send: stop typing and clean up any placeholder before sending media.
	m.preSendMedia(ctx, name, msg, w.ch)

	var lastErr error
	var msgIDs []string
	for attempt := 0; attempt <= maxRetries; attempt++ {
		msgIDs, lastErr = ms.SendMedia(ctx, msg)
		if lastErr == nil {
			m.publishOutboundMediaSent(name, msg, msgIDs)
			return msgIDs, nil
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
				return nil, ctx.Err()
			}
		}

		// ErrTemporary or unknown error — exponential backoff
		backoff := min(time.Duration(float64(baseBackoff)*math.Pow(2, float64(attempt))), maxBackoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// All retries exhausted or permanent failure
	logger.ErrorCF("channels", "SendMedia failed", map[string]any{
		"channel": name,
		"chat_id": outboundMediaChatID(msg),
		"error":   lastErr.Error(),
		"retries": maxRetries,
	})
	m.publishOutboundMediaFailed(name, msg, lastErr)
	return nil, lastErr
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
		}
	}
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

// Reload updates the config reference without restarting channels.
// This is used when channel config hasn't changed but other parts of the config have.
func (m *Manager) Reload(ctx context.Context, cfg *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Save old config so we can revert on error.
	oldConfig := m.config

	// Update config early: initChannel uses m.config via factory(m.config, m.bus).
	m.config = cfg

	list := toChannelHashes(cfg)
	added, removed := compareChannels(m.channelHashes, list)

	deferFuncs := make([]func(), 0, len(removed)+len(added))
	for _, name := range removed {
		// Stop all channels
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
			m.publishChannelEvent(
				runtimeevents.KindChannelLifecycleStartFailed,
				name,
				runtimeevents.Scope{Channel: name},
				runtimeevents.SeverityError,
				ChannelLifecyclePayload{Type: channelTypeForEvent(m, name), Error: err.Error()},
			)
			continue
		}
		// Lazily create worker only after channel starts successfully
		channelType := name
		if m.config != nil {
			if bc := m.config.Channels.Get(name); bc != nil && bc.Type != "" {
				channelType = bc.Type
			}
		}
		w := newChannelWorker(name, channel, channelType)
		m.workers[name] = w
		go m.runWorker(dispatchCtx, name, w)
		go m.runMediaWorker(dispatchCtx, name, w)
		m.publishChannelEvent(
			runtimeevents.KindChannelLifecycleStarted,
			name,
			runtimeevents.Scope{Channel: name},
			runtimeevents.SeverityInfo,
			ChannelLifecyclePayload{Type: channelType},
		)
		deferFuncs = append(deferFuncs, func() {
			m.RegisterChannel(name, channel)
		})
	}

	// Commit hashes only on full success.
	m.channelHashes = list
	go func() {
		for _, f := range deferFuncs {
			f()
		}
	}()
	return nil
}

func (m *Manager) RegisterChannel(name string, channel Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[name] = channel
	if m.mux != nil {
		m.registerChannelHTTPHandler(name, channel)
	}
}

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

// SendMessage sends an outbound message synchronously through the channel
// worker's rate limiter and retry logic. It blocks until the message is
// delivered (or all retries are exhausted), which preserves ordering when
// a subsequent operation depends on the message having been sent.
func (m *Manager) SendMessage(ctx context.Context, msg bus.OutboundMessage) error {
	msg = bus.NormalizeOutboundMessage(msg)
	channelName := outboundMessageChannel(msg)

	m.mu.RLock()
	_, exists := m.channels[channelName]
	w, wExists := m.workers[channelName]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}
	if !wExists || w == nil {
		return fmt.Errorf("channel %s has no active worker", channelName)
	}

	maxLen := 0
	if mlp, ok := w.ch.(MessageLengthProvider); ok {
		maxLen = mlp.MaxMessageLength()
	}
	if chunks := splitOutboundMessageContent(msg, maxLen); len(chunks) > 1 {
		for _, chunk := range chunks {
			chunkMsg := msg
			chunkMsg.Content = chunk
			m.sendWithRetry(ctx, channelName, w, chunkMsg)
		}
	} else {
		if len(chunks) == 1 {
			msg.Content = chunks[0]
		}
		m.sendWithRetry(ctx, channelName, w, msg)
	}
	return nil
}

// SendMedia sends outbound media synchronously through the channel worker's
// rate limiter and retry logic. It blocks until the media is delivered (or all
// retries are exhausted), which preserves ordering when later agent behavior
// depends on actual media delivery.
func (m *Manager) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	msg = bus.NormalizeOutboundMediaMessage(msg)
	channelName := outboundMediaChannel(msg)

	m.mu.RLock()
	_, exists := m.channels[channelName]
	w, wExists := m.workers[channelName]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}
	if !wExists || w == nil {
		return fmt.Errorf("channel %s has no active worker", channelName)
	}

	_, err := m.sendMediaWithRetry(ctx, channelName, w, msg)
	return err
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
		Context: bus.NewOutboundContext(channelName, chatID, ""),
		Content: content,
	}
	msg = bus.NormalizeOutboundMessage(msg)

	if wExists && w != nil {
		select {
		case w.queue <- msg:
			m.publishOutboundQueued(channelName, msg)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Fallback: direct send (should not happen)
	channel, _ := m.channels[channelName]
	_, err := channel.Send(ctx, msg)
	return err
}
