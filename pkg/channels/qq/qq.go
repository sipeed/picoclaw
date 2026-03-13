package qq

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
	"golang.org/x/oauth2"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	dedupTTL      = 5 * time.Minute
	dedupInterval = 60 * time.Second
	dedupMaxSize  = 10000 // hard cap on dedup map entries
	typingResend  = 8 * time.Second
	typingSeconds = 10
)

type QQChannel struct {
	*channels.BaseChannel
	config         config.QQConfig
	api            openapi.OpenAPI
	tokenSource    oauth2.TokenSource
	ctx            context.Context
	cancel         context.CancelFunc
	sessionManager botgo.SessionManager

	// Chat routing: track whether a chatID is group or direct.
	chatType sync.Map // chatID → "group" | "direct"

	// Passive reply: store last inbound message ID per chat.
	lastMsgID sync.Map // chatID → string

	// msg_seq: per-chat atomic counter for multi-part replies.
	msgSeqCounters sync.Map // chatID → *atomic.Uint64

	// Time-based dedup replacing the unbounded map.
	dedup   map[string]time.Time
	muDedup sync.Mutex

	// done is closed on Stop to shut down the dedup janitor.
	done     chan struct{}
	stopOnce sync.Once
}

func NewQQChannel(cfg config.QQConfig, messageBus *bus.MessageBus) (*QQChannel, error) {
	base := channels.NewBaseChannel("qq", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(cfg.MaxMessageLength),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &QQChannel{
		BaseChannel: base,
		config:      cfg,
		dedup:       make(map[string]time.Time),
		done:        make(chan struct{}),
	}, nil
}

func (c *QQChannel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		return fmt.Errorf("QQ app_id and app_secret not configured")
	}

	botgo.SetLogger(logger.NewLogger("botgo"))
	logger.InfoC("qq", "Starting QQ bot (WebSocket mode)")

	// Reinitialize shutdown signal for clean restart.
	c.done = make(chan struct{})
	c.stopOnce = sync.Once{}

	// create token source
	credentials := &token.QQBotCredentials{
		AppID:     c.config.AppID,
		AppSecret: c.config.AppSecret,
	}
	c.tokenSource = token.NewQQBotTokenSource(credentials)

	// create child context
	c.ctx, c.cancel = context.WithCancel(ctx)

	// start auto-refresh token goroutine
	if err := token.StartRefreshAccessToken(c.ctx, c.tokenSource); err != nil {
		return fmt.Errorf("failed to start token refresh: %w", err)
	}

	// initialize OpenAPI client
	c.api = botgo.NewOpenAPI(c.config.AppID, c.tokenSource).WithTimeout(5 * time.Second)

	// register event handlers
	intent := event.RegisterHandlers(
		c.handleC2CMessage(),
		c.handleGroupATMessage(),
	)

	// get WebSocket endpoint
	wsInfo, err := c.api.WS(c.ctx, nil, "")
	if err != nil {
		return fmt.Errorf("failed to get websocket info: %w", err)
	}

	logger.InfoCF("qq", "Got WebSocket info", map[string]any{
		"shards": wsInfo.Shards,
	})

	// create and save sessionManager
	c.sessionManager = botgo.NewSessionManager()

	// start WebSocket connection in goroutine to avoid blocking
	go func() {
		if err := c.sessionManager.Start(wsInfo, c.tokenSource, &intent); err != nil {
			logger.ErrorCF("qq", "WebSocket session error", map[string]any{
				"error": err.Error(),
			})
			c.SetRunning(false)
		}
	}()

	// start dedup janitor goroutine
	go c.dedupJanitor()

	// Pre-register reasoning_channel_id as group chat if configured,
	// so outbound-only destinations are routed correctly.
	if c.config.ReasoningChannelID != "" {
		c.chatType.Store(c.config.ReasoningChannelID, "group")
	}

	c.SetRunning(true)
	logger.InfoC("qq", "QQ bot started successfully")

	return nil
}

func (c *QQChannel) Stop(ctx context.Context) error {
	logger.InfoC("qq", "Stopping QQ bot")
	c.SetRunning(false)

	// Signal the dedup janitor to stop (idempotent).
	c.stopOnce.Do(func() { close(c.done) })

	if c.cancel != nil {
		c.cancel()
	}

	return nil
}

// getChatKind returns the chat type for a given chatID ("group" or "direct").
// Unknown chatIDs default to "group" and log a warning, since QQ group IDs are
// more common as outbound-only destinations (e.g. reasoning_channel_id).
func (c *QQChannel) getChatKind(chatID string) string {
	if v, ok := c.chatType.Load(chatID); ok {
		if k, ok := v.(string); ok {
			return k
		}
	}
	logger.DebugCF("qq", "Unknown chat type for chatID, defaulting to group", map[string]any{
		"chat_id": chatID,
	})
	return "group"
}

func (c *QQChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	chatKind := c.getChatKind(msg.ChatID)

	// Build message with content.
	msgToCreate := &dto.MessageToCreate{
		Content: msg.Content,
		MsgType: dto.TextMsg,
	}

	// Use Markdown message type if enabled in config.
	if c.config.SendMarkdown {
		msgToCreate.MsgType = dto.MarkdownMsg
		msgToCreate.Markdown = &dto.Markdown{
			Content: msg.Content,
		}
		// Clear plain content to avoid sending duplicate text.
		msgToCreate.Content = ""
	}

	// Attach passive reply msg_id and msg_seq if available.
	if v, ok := c.lastMsgID.Load(msg.ChatID); ok {
		if msgID, ok := v.(string); ok && msgID != "" {
			msgToCreate.MsgID = msgID

			// Increment msg_seq atomically for multi-part replies.
			if counterVal, ok := c.msgSeqCounters.Load(msg.ChatID); ok {
				if counter, ok := counterVal.(*atomic.Uint64); ok {
					seq := counter.Add(1)
					msgToCreate.MsgSeq = uint32(seq)
				}
			}
		}
	}

	// Sanitize URLs in group messages to avoid QQ's URL blacklist rejection.
	if chatKind == "group" {
		if msgToCreate.Content != "" {
			msgToCreate.Content = sanitizeURLs(msgToCreate.Content)
		}
		if msgToCreate.Markdown != nil && msgToCreate.Markdown.Content != "" {
			msgToCreate.Markdown.Content = sanitizeURLs(msgToCreate.Markdown.Content)
		}
	}

	// Route to group or C2C.
	var err error
	if chatKind == "group" {
		_, err = c.api.PostGroupMessage(ctx, msg.ChatID, msgToCreate)
	} else {
		_, err = c.api.PostC2CMessage(ctx, msg.ChatID, msgToCreate)
	}

	if err != nil {
		logger.ErrorCF("qq", "Failed to send message", map[string]any{
			"chat_id":   msg.ChatID,
			"chat_kind": chatKind,
			"error":     err.Error(),
		})
		return fmt.Errorf("qq send: %w", channels.ErrTemporary)
	}

	return nil
}

// StartTyping implements channels.TypingCapable.
// It sends an InputNotify (msg_type=6) immediately and re-sends every 8 seconds.
// The returned stop function is idempotent and cancels the goroutine.
func (c *QQChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	// We need a stored msg_id for passive InputNotify; skip if none available.
	v, ok := c.lastMsgID.Load(chatID)
	if !ok {
		return func() {}, nil
	}
	msgID, ok := v.(string)
	if !ok || msgID == "" {
		return func() {}, nil
	}

	chatKind := c.getChatKind(chatID)

	sendTyping := func(sendCtx context.Context) {
		typingMsg := &dto.MessageToCreate{
			MsgType: dto.InputNotifyMsg,
			MsgID:   msgID,
			InputNotify: &dto.InputNotify{
				InputType:   1,
				InputSecond: typingSeconds,
			},
		}

		var err error
		if chatKind == "group" {
			_, err = c.api.PostGroupMessage(sendCtx, chatID, typingMsg)
		} else {
			_, err = c.api.PostC2CMessage(sendCtx, chatID, typingMsg)
		}
		if err != nil {
			logger.DebugCF("qq", "Failed to send typing indicator", map[string]any{
				"chat_id": chatID,
				"error":   err.Error(),
			})
		}
	}

	// Send immediately.
	sendTyping(c.ctx)

	typingCtx, cancel := context.WithCancel(c.ctx)
	go func() {
		ticker := time.NewTicker(typingResend)
		defer ticker.Stop()
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				sendTyping(typingCtx)
			}
		}
	}()

	return cancel, nil
}

// SendMedia implements the channels.MediaSender interface.
// QQ RichMediaMessage requires an HTTP/HTTPS URL — local file paths are not supported.
// If part.Ref is already an http(s) URL it is used directly; otherwise we try
// the media store, and skip with a warning if the resolved path is not an HTTP URL.
func (c *QQChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	chatKind := c.getChatKind(msg.ChatID)

	for _, part := range msg.Parts {
		// If the ref is already an HTTP(S) URL, use it directly.
		mediaURL := part.Ref
		if !isHTTPURL(mediaURL) {
			// Try resolving through media store.
			store := c.GetMediaStore()
			if store == nil {
				logger.WarnCF("qq", "QQ media requires HTTP/HTTPS URL, no media store available", map[string]any{
					"ref": part.Ref,
				})
				continue
			}

			resolved, err := store.Resolve(part.Ref)
			if err != nil {
				logger.ErrorCF("qq", "Failed to resolve media ref", map[string]any{
					"ref":   part.Ref,
					"error": err.Error(),
				})
				continue
			}

			if !isHTTPURL(resolved) {
				logger.WarnCF("qq", "QQ media requires HTTP/HTTPS URL, local files not supported", map[string]any{
					"ref":      part.Ref,
					"resolved": resolved,
				})
				continue
			}

			mediaURL = resolved
		}

		// Map part type to QQ file type: 1=image, 2=video, 3=audio, 4=file.
		var fileType uint64
		switch part.Type {
		case "image":
			fileType = 1
		case "video":
			fileType = 2
		case "audio":
			fileType = 3
		default:
			fileType = 4 // file
		}

		richMedia := &dto.RichMediaMessage{
			FileType:   fileType,
			URL:        mediaURL,
			SrvSendMsg: true,
		}

		var sendErr error
		if chatKind == "group" {
			_, sendErr = c.api.PostGroupMessage(ctx, msg.ChatID, richMedia)
		} else {
			_, sendErr = c.api.PostC2CMessage(ctx, msg.ChatID, richMedia)
		}

		if sendErr != nil {
			logger.ErrorCF("qq", "Failed to send media", map[string]any{
				"type":    part.Type,
				"chat_id": msg.ChatID,
				"error":   sendErr.Error(),
			})
			return fmt.Errorf("qq send media: %w", channels.ErrTemporary)
		}
	}

	return nil
}

// handleC2CMessage handles QQ private messages.
func (c *QQChannel) handleC2CMessage() event.C2CMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSC2CMessageData) error {
		// deduplication check
		if c.isDuplicate(data.ID) {
			return nil
		}

		// extract user info
		var senderID string
		if data.Author != nil && data.Author.ID != "" {
			senderID = data.Author.ID
		} else {
			logger.WarnC("qq", "Received message with no sender ID")
			return nil
		}

		// extract message content
		content := data.Content
		if content == "" {
			logger.DebugC("qq", "Received empty message, ignoring")
			return nil
		}

		logger.InfoCF("qq", "Received C2C message", map[string]any{
			"sender": senderID,
			"length": len(content),
		})

		// Store chat routing context.
		c.chatType.Store(senderID, "direct")
		c.lastMsgID.Store(senderID, data.ID)

		// Reset msg_seq counter for new inbound message.
		c.msgSeqCounters.Store(senderID, new(atomic.Uint64))

		metadata := map[string]string{}

		sender := bus.SenderInfo{
			Platform:    "qq",
			PlatformID:  data.Author.ID,
			CanonicalID: identity.BuildCanonicalID("qq", data.Author.ID),
		}

		if !c.IsAllowedSender(sender) {
			return nil
		}

		c.HandleMessage(c.ctx,
			bus.Peer{Kind: "direct", ID: senderID},
			data.ID,
			senderID,
			senderID,
			content,
			[]string{},
			metadata,
			sender,
		)

		return nil
	}
}

// handleGroupATMessage handles QQ group @ messages.
func (c *QQChannel) handleGroupATMessage() event.GroupATMessageEventHandler {
	return func(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
		// deduplication check
		if c.isDuplicate(data.ID) {
			return nil
		}

		// extract user info
		var senderID string
		if data.Author != nil && data.Author.ID != "" {
			senderID = data.Author.ID
		} else {
			logger.WarnC("qq", "Received group message with no sender ID")
			return nil
		}

		// extract message content (remove @ bot part)
		content := data.Content
		if content == "" {
			logger.DebugC("qq", "Received empty group message, ignoring")
			return nil
		}

		// GroupAT event means bot is always mentioned; apply group trigger filtering
		respond, cleaned := c.ShouldRespondInGroup(true, content)
		if !respond {
			return nil
		}
		content = cleaned

		logger.InfoCF("qq", "Received group AT message", map[string]any{
			"sender": senderID,
			"group":  data.GroupID,
			"length": len(content),
		})

		// Store chat routing context using GroupID as chatID.
		c.chatType.Store(data.GroupID, "group")
		c.lastMsgID.Store(data.GroupID, data.ID)

		// Reset msg_seq counter for new inbound message.
		c.msgSeqCounters.Store(data.GroupID, new(atomic.Uint64))

		metadata := map[string]string{
			"group_id": data.GroupID,
		}

		sender := bus.SenderInfo{
			Platform:    "qq",
			PlatformID:  data.Author.ID,
			CanonicalID: identity.BuildCanonicalID("qq", data.Author.ID),
		}

		if !c.IsAllowedSender(sender) {
			return nil
		}

		c.HandleMessage(c.ctx,
			bus.Peer{Kind: "group", ID: data.GroupID},
			data.ID,
			senderID,
			data.GroupID,
			content,
			[]string{},
			metadata,
			sender,
		)

		return nil
	}
}

// isDuplicate checks whether a message has been seen within the TTL window.
// It also enforces a hard cap on map size by evicting oldest entries.
func (c *QQChannel) isDuplicate(messageID string) bool {
	c.muDedup.Lock()
	defer c.muDedup.Unlock()

	if ts, exists := c.dedup[messageID]; exists && time.Since(ts) < dedupTTL {
		return true
	}

	// Enforce hard cap: evict oldest entries when at capacity.
	if len(c.dedup) >= dedupMaxSize {
		var oldestID string
		var oldestTS time.Time
		for id, ts := range c.dedup {
			if oldestID == "" || ts.Before(oldestTS) {
				oldestID = id
				oldestTS = ts
			}
		}
		if oldestID != "" {
			delete(c.dedup, oldestID)
		}
	}

	c.dedup[messageID] = time.Now()
	return false
}

// dedupJanitor periodically evicts expired entries from the dedup map.
func (c *QQChannel) dedupJanitor() {
	ticker := time.NewTicker(dedupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			// Collect expired keys under read-like scan.
			c.muDedup.Lock()
			now := time.Now()
			var expired []string
			for id, ts := range c.dedup {
				if now.Sub(ts) >= dedupTTL {
					expired = append(expired, id)
				}
			}
			for _, id := range expired {
				delete(c.dedup, id)
			}
			c.muDedup.Unlock()
		}
	}
}

// isHTTPURL returns true if s starts with http:// or https://.
func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// urlPattern matches URLs with explicit http(s):// scheme.
// Only scheme-prefixed URLs are matched to avoid false positives on bare text
// like version numbers (e.g., "1.2.3") or domain-like fragments.
var urlPattern = regexp.MustCompile(
	`(?i)` +
		`https?://` + // required scheme
		`(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+` + // domain parts
		`[a-zA-Z]{2,}` + // TLD
		`(?:[/?#]\S*)?`, // optional path/query/fragment
)

// sanitizeURLs replaces dots in URL domains with "。" (fullwidth period)
// to prevent QQ's URL blacklist from rejecting the message.
func sanitizeURLs(text string) string {
	return urlPattern.ReplaceAllStringFunc(text, func(match string) string {
		// Split into scheme + rest (scheme is always present).
		idx := strings.Index(match, "://")
		scheme := match[:idx+3]
		rest := match[idx+3:]

		// Find where the domain ends (first / ? or #).
		domainEnd := len(rest)
		for i, ch := range rest {
			if ch == '/' || ch == '?' || ch == '#' {
				domainEnd = i
				break
			}
		}

		domain := rest[:domainEnd]
		path := rest[domainEnd:]

		// Replace dots in domain only.
		domain = strings.ReplaceAll(domain, ".", "。")

		return scheme + domain + path
	})
}
