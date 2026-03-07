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
	done chan struct{}
}

func NewQQChannel(cfg config.QQConfig, messageBus *bus.MessageBus) (*QQChannel, error) {
	maxLen := cfg.MaxMessageLength
	if maxLen == 0 {
		maxLen = 2000
	}

	base := channels.NewBaseChannel("qq", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(maxLen),
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

	logger.InfoC("qq", "Starting QQ bot (WebSocket mode)")

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

	c.SetRunning(true)
	logger.InfoC("qq", "QQ bot started successfully")

	return nil
}

func (c *QQChannel) Stop(ctx context.Context) error {
	logger.InfoC("qq", "Stopping QQ bot")
	c.SetRunning(false)

	// Signal the dedup janitor to stop.
	close(c.done)

	if c.cancel != nil {
		c.cancel()
	}

	return nil
}

func (c *QQChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	// Determine chat type (fallback to "direct" if not tracked).
	chatKind := "direct"
	if v, ok := c.chatType.Load(msg.ChatID); ok {
		if k, ok := v.(string); ok {
			chatKind = k
		}
	}

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

	chatKind := "direct"
	if kv, ok := c.chatType.Load(chatID); ok {
		if k, ok := kv.(string); ok {
			chatKind = k
		}
	}

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
	sendTyping(ctx)

	typingCtx, cancel := context.WithCancel(ctx)
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
// It sends rich media (images, videos, audio, files) via QQ RichMediaMessage.
// Note: RichMediaMessage requires an HTTP/HTTPS URL. Local-only files are skipped
// with a warning since QQ API does not accept local file paths.
func (c *QQChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	chatKind := "direct"
	if v, ok := c.chatType.Load(msg.ChatID); ok {
		if k, ok := v.(string); ok {
			chatKind = k
		}
	}

	store := c.GetMediaStore()
	if store == nil {
		return fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	for _, part := range msg.Parts {
		localPath, err := store.Resolve(part.Ref)
		if err != nil {
			logger.ErrorCF("qq", "Failed to resolve media ref", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			continue
		}

		// QQ RichMediaMessage requires an HTTP/HTTPS URL.
		// If the resolved path is a local file, skip with a warning.
		if !isHTTPURL(localPath) {
			logger.WarnCF("qq", "QQ media requires HTTP/HTTPS URL, skipping local file", map[string]any{
				"ref":  part.Ref,
				"path": localPath,
			})
			continue
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
			URL:        localPath,
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
		var counter atomic.Uint64
		c.msgSeqCounters.Store(senderID, &counter)

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
		var counter atomic.Uint64
		c.msgSeqCounters.Store(data.GroupID, &counter)

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
func (c *QQChannel) isDuplicate(messageID string) bool {
	c.muDedup.Lock()
	defer c.muDedup.Unlock()

	if ts, exists := c.dedup[messageID]; exists && time.Since(ts) < dedupTTL {
		return true
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
			c.muDedup.Lock()
			now := time.Now()
			for id, ts := range c.dedup {
				if now.Sub(ts) >= dedupTTL {
					delete(c.dedup, id)
				}
			}
			c.muDedup.Unlock()
		}
	}
}

// isHTTPURL returns true if s starts with http:// or https://.
func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// urlPattern matches URLs like http(s)://domain.tld/path and bare domain.tld/path patterns.
var urlPattern = regexp.MustCompile(
	`(?i)` +
		`(?:https?://)?` + // optional scheme
		`(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+` + // domain parts
		`[a-zA-Z]{2,}` + // TLD
		`(?:[/?#]\S*)?`, // optional path/query/fragment
)

// sanitizeURLs replaces dots in URL domains with "。" (fullwidth period)
// to prevent QQ's URL blacklist from rejecting the message.
func sanitizeURLs(text string) string {
	return urlPattern.ReplaceAllStringFunc(text, func(match string) string {
		// Split into scheme + rest.
		var scheme, rest string
		if idx := strings.Index(match, "://"); idx != -1 {
			scheme = match[:idx+3]
			rest = match[idx+3:]
		} else {
			rest = match
		}

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
