package qq

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
	"github.com/tidwall/gjson"

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
	typingSeconds = 20
)

var emojiRegexp = regexp.MustCompile(`<[^<]*?ext="([^"]+)"[^<]*?faceType=(\d+)[^<]*?>|<[^<]*?faceType=(\d+)[^<]*?ext="([^"]+)"[^<]*?>`)

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

	// 消息回复时API时的SEQ，用于回复时去重
	// 被动回复： 同一个chatID+replyMsgID+seq 去重。 主动回复：不限制。
	replySeq atomic.Uint32
	seqLock  sync.Mutex

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
	c.api = botgo.NewOpenAPI(c.config.AppID, c.tokenSource).WithTimeout(20 * time.Second)

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

// Send sends a message to the specified chatID.
// First attempt to send a Markdown message, fallback to plain text if failed.
func (c *QQChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	chatKind := c.getChatKind(msg.ChatID)
	textMsg, mdMsg := c.genReplyMsg(ctx, msg, chatKind)
	var err error
	for _, replyMsg := range []dto.MessageToCreate{mdMsg, textMsg} {
		var replyMsgID *dto.Message
		if chatKind == "group" {
			replyMsgID, err = c.api.PostGroupMessage(ctx, msg.ChatID, replyMsg)
		} else {
			replyMsgID, err = c.api.PostC2CMessage(ctx, msg.ChatID, replyMsg)
		}
		if err == nil {
			logger.InfoCF("qq", "Sent message", map[string]any{"postrsp ": replyMsgID})
			return nil
		}
		if err != nil {
			logger.ErrorCF("qq", "Failed to send message", map[string]any{
				"chat_id":   msg.ChatID,
				"chat_kind": chatKind,
				"error":     err.Error(),
			})
		}
	}
	return err
}

func (c *QQChannel) genReplyMsg(ctx context.Context, msg bus.OutboundMessage, chatKind string) (dto.MessageToCreate,
	dto.MessageToCreate) {
	textMsg := dto.MessageToCreate{
		Content: sanitizeURLs(msg.Content),
		MsgType: dto.TextMsg,
	}

	mdMsg := dto.MessageToCreate{
		MsgType: dto.MarkdownMsg,
		Markdown: &dto.Markdown{
			Content: msg.Content,
		},
	}

	return textMsg, mdMsg
}

func (c *QQChannel) getReplyExtInfo(ctx context.Context, chatID string) (replyID string, seq uint32) {
	// Attach passive reply msg_id and msg_seq if available.
	if v, ok := c.lastMsgID.Load(chatID); ok {
		if msgID, ok := v.(string); ok && msgID != "" {
			replyID = msgID
		}
	}

	// Attach msg_seq for active reply.
	c.seqLock.Lock()
	defer c.seqLock.Unlock()
	seq = c.replySeq.Add(1)
	if seq > math.MaxInt32 {
		c.replySeq.Store(0)
		seq = c.replySeq.Add(1)
	}

	return replyID, seq
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
func (c *QQChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	for _, part := range msg.Parts {
		if err := c.sendOneMedia(ctx, msg.ChatID, part); err != nil {
			logger.ErrorCF("qq", "Failed to send media", map[string]any{
				"part":  part,
				"error": err.Error(),
			})
			continue
		}
	}
	return nil
}

// Upload file and then send it via API
// QQ groups do not support file sending
// When sending local files via QQ, the file size cannot exceed 10M
func (c *QQChannel) sendOneMedia(ctx context.Context, chatID string, part bus.MediaPart) error {
	chatKind := c.getChatKind(chatID)

	mediaPath := part.Ref
	var meta media.MediaMeta
	if !isHTTPURL(mediaPath) {
		store := c.GetMediaStore()
		if store == nil {
			logger.WarnCF("qq", "QQ media requires HTTP/HTTPS URL, no media store available", map[string]any{
				"ref": part.Ref,
			})
			return fmt.Errorf("store not available")
		}
		var resolved string
		var err error
		resolved, meta, err = store.ResolveWithMeta(part.Ref)
		if err != nil {
			logger.ErrorCF("qq", "Failed to resolve media ref", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			return fmt.Errorf("store resolve failed")
		}
		mediaPath = resolved
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

	richMedia := &RichMediaMessage{FileType: fileType}
	if isHTTPURL(mediaPath) {
		richMedia.URL = mediaPath
	} else {
		fdata, err := os.ReadFile(mediaPath)
		if err != nil {
			logger.ErrorCF("qq", "Failed to read media file[%v]", map[string]any{"path": mediaPath,
				"error": err.Error()})
			return fmt.Errorf("read file failed")
		}
		richMedia.FileData = fdata
		richMedia.FileName = meta.Filename
	}

	if (chatKind == "group" && fileType == 4) || len(richMedia.FileData) > 10*1024*1024 {
		logger.WarnCF("qq", "File size exceeds 10M, skipping send", map[string]any{
			"filename": richMedia.FileName, "size": len(richMedia.FileData)})
		return nil
	}

	var sendErr error
	var result *dto.Message
	if chatKind == "group" {
		result, sendErr = c.api.PostGroupMessage(ctx, chatID, richMedia)
	} else {
		result, sendErr = c.api.PostC2CMessage(ctx, chatID, richMedia)
	}

	if sendErr != nil {
		logger.ErrorCF("qq", "Failed to send media", map[string]any{
			"type":    part.Type,
			"chat_id": chatID,
			"error":   sendErr.Error(),
		})
		return fmt.Errorf("qq send media: %w err:%v", channels.ErrTemporary, sendErr)
	}

	msg := dto.MessageToCreate{
		MsgType: dto.RichMediaMsg,
		Media:   &dto.MediaInfo{FileInfo: result.FileInfo},
	}
	msg.MsgID, msg.MsgSeq = c.getReplyExtInfo(ctx, chatID)

	if chatKind == "group" {
		result, sendErr = c.api.PostGroupMessage(ctx, chatID, msg)
	} else {
		result, sendErr = c.api.PostC2CMessage(ctx, chatID, msg)
	}
	if sendErr != nil {
		logger.ErrorCF("qq", "Failed to send media", map[string]any{
			"type":    part.Type,
			"chat_id": chatID,
			"error":   sendErr.Error(),
		})
		return fmt.Errorf("qq send media: %w err:%v", channels.ErrTemporary, sendErr)
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

		scope := channels.BuildMediaScope("qq", senderID, data.ID)

		content, mediaPaths := c.decodeMessage(context.Background(), event, (*dto.Message)(data), scope)
		if content == "" {
			logger.DebugC("qq", "Received empty C2C message, ignoring")
			return nil
		}

		logger.InfoCF("qq", "Received C2C message", map[string]any{
			"sender": senderID,
			"length": len(content),
		})

		// Store chat routing context.
		c.chatType.Store(senderID, "direct")
		c.lastMsgID.Store(senderID, data.ID)

		metadata := map[string]string{
			"account_id": senderID,
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
			bus.Peer{Kind: "direct", ID: senderID},
			data.ID,
			senderID,
			senderID,
			content,
			mediaPaths,
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
		scope := channels.BuildMediaScope("qq", data.GroupID, data.ID)

		content, mediaPaths := c.decodeMessage(context.Background(), event, (*dto.Message)(data), scope)
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
			mediaPaths,
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

func (c *QQChannel) decodeMessage(ctx context.Context, event *dto.WSPayload, data *dto.Message,
	scope string) (content string, mediaPaths []string) {

	content = parseEmojiText(data.Content)
	wavURL, asrReferText := getVoiceInfo(event)
	if data.Attachments != nil && len(data.Attachments) > 0 {
		var attachments []MessageAttachment
		for _, att := range data.Attachments {
			if att.ContentType == "voice" && wavURL != "" {
				attachments = append(attachments, MessageAttachment{
					ContentType:  "voice",
					URL:          wavURL,
					FileName:     filepath.Base(wavURL),
					AsrReferText: asrReferText,
				})
				continue
			} else {
				attachments = append(attachments, MessageAttachment{
					ContentType: att.ContentType,
					URL:         att.URL,
					FileName:    att.FileName,
				})
			}
		}
		processedPaths, attachmentContent := c.processAttachments(ctx, attachments, scope)
		if asrReferText != "" {
			attachmentContent = fmt.Sprintf("[audio: %v]", asrReferText)
		}
		mediaPaths = processedPaths
		if content != "" {
			content += "\n"
		}
		content += attachmentContent
	}
	return content, mediaPaths
}

// processAttachments processes all attachments in a message
func (c *QQChannel) processAttachments(ctx context.Context, attachments []MessageAttachment,
	scope string) (mediaPaths []string, content string) {

	// Helper to register a local file with the media store
	storeMedia := func(localPath, filename string) string {
		store := c.GetMediaStore()
		if store == nil {
			logger.ErrorCF("qq", "media store is nil", map[string]any{
				"scope": scope,
			})
			return ""
		}
		ref, err := store.Store(localPath, media.MediaMeta{Filename: filename, Source: "qq"}, scope)
		if err == nil {
			logger.InfoCF("qq", "Stored media", map[string]any{
				"scope":     scope,
				"localPath": localPath,
				"filename":  filename,
			})
			return ref
		}
		logger.ErrorCF("qq", "Stored media err ", map[string]any{
			"scope":     scope,
			"localPath": localPath,
			"err":       err.Error(),
		})
		return localPath
	}

	for _, attachment := range attachments {
		attachmentType := c.getAttachmentType(attachment)
		localPath := c.downloadAttachment(ctx, attachment)
		if localPath == "" {
			mediaPaths = append(mediaPaths, attachment.URL)
			content += appendContent(content, fmt.Sprintf("[%v: %s]", attachment.ContentType, attachment.URL))
			continue
		}
		ref := storeMedia(localPath, attachment.FileName)
		mediaPaths = append(mediaPaths, ref)
		if attachmentType == "audio" && attachment.AsrReferText != "" {
			content += appendContent(content, fmt.Sprintf("[audio: %s]", attachment.AsrReferText))
			continue
		}
		content += appendContent(content, fmt.Sprintf("[%v: %s]", attachment.ContentType, ref))
	}

	return mediaPaths, content
}

// downloadAttachment downloads an attachment from QQ server
func (c *QQChannel) downloadAttachment(ctx context.Context, attachment MessageAttachment) string {
	logger.InfoCF("qq", "Downloading attachment", map[string]any{
		"attachment": attachment,
	})
	return utils.DownloadFile(attachment.URL, attachment.FileName, utils.DownloadOptions{
		LoggerPrefix: "qq",
	})
}

// getAttachmentType determines the type of attachment (image, audio, video, file)
func (c *QQChannel) getAttachmentType(attachment MessageAttachment) string {

	if strings.HasPrefix(attachment.ContentType, "image") {
		return "image"
	} else if strings.HasPrefix(attachment.ContentType, "video") {
		return "video"
	} else if strings.HasPrefix(attachment.ContentType, "voice") {
		return "audio"
	}
	return "file"
}

// appendContent safely appends content to existing text
func appendContent(content, suffix string) string {
	if content == "" {
		return suffix
	}
	return content + "\n" + suffix
}

func getVoiceInfo(event *dto.WSPayload) (string, string) {
	_raw, err := json.Marshal(event.Data)
	if err != nil {
		logger.ErrorCF("qq", "Failed to marshal event data", map[string]any{
			"error": err.Error(),
		})
		return "", ""
	}
	// 使用gjson提取voice_wav_url字段
	rawJSON := string(_raw)

	// 首先尝试从attachments数组的第一个元素中提取voice_wav_url
	voiceWavURL := gjson.Get(rawJSON, "attachments.0.voice_wav_url").String()
	asrReferText := gjson.Get(rawJSON, "attachments.0.asr_refer_text").String()
	logger.DebugCF("qq", "Found voice_wav_url in attachments", map[string]any{
		"url": voiceWavURL, "asr_refer_text": asrReferText,
	})
	return voiceWavURL, asrReferText
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

// parseEmojiText decodes emoji text
func parseEmojiText(content string) string {
	content = strings.ReplaceAll(content, `\\`, `\`)
	content = strings.ReplaceAll(content, "\\u003c", "<")
	content = strings.ReplaceAll(content, "\\u003e", ">")
	content = strings.ReplaceAll(content, `\"`, `"`)

	contentParts := emojiRegexp.Split(content, -1)
	matches := emojiRegexp.FindAllString(content, -1)

	var result strings.Builder
	for i, part := range contentParts {
		if strings.TrimSpace(part) != "" {
			result.WriteString(part)
		}
		if i < len(matches) {
			match := matches[i]
			if strings.Contains(match, "faceType=") {
				result.WriteString(processEmoji(match))
			}
		}
	}

	return result.String()
}

func processEmoji(match string) string {
	extRegexp := regexp.MustCompile(`ext="([^"]+)"`)
	extMatch := extRegexp.FindStringSubmatch(match)

	if len(extMatch) > 1 {
		ext, err := base64.StdEncoding.DecodeString(extMatch[1])
		if err == nil {
			var faceDesc map[string]string
			json.Unmarshal(ext, &faceDesc)
			return fmt.Sprintf("[表情 %v]", faceDesc["text"])
		}
	}
	return ""
}
