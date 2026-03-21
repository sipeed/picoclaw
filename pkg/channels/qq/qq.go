package qq

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/constant"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/openapi/options"
	"github.com/tencent-connect/botgo/token"
	"golang.org/x/oauth2"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	dedupTTL      = 5 * time.Minute
	dedupInterval = 60 * time.Second
	dedupMaxSize  = 10000 // hard cap on dedup map entries
	typingResend  = 8 * time.Second
	typingSeconds = 10
	bytesPerMiB   = 1024 * 1024

	// Reconnection constants (used as defaults when config values are 0)
	reconnectInitialDefault = 5 * time.Second
	reconnectMaxDefault     = 5 * time.Minute
	reconnectMultiplier     = 2.0

	// Retry constants (used as defaults when config values are 0)
	maxRetriesDefault        = 3
	retryInitialDelayDefault = 500 * time.Millisecond
	retryMaxDelayDefault     = 10 * time.Second

	// Rate limit defaults
	rateLimitGroupDefault  = 500 * time.Millisecond
	rateLimitDirectDefault = 200 * time.Millisecond
)

type qqAPI interface {
	WS(ctx context.Context, params map[string]string, body string) (*dto.WebsocketAP, error)
	PostGroupMessage(
		ctx context.Context, groupID string, msg dto.APIMessage, opt ...options.Option,
	) (*dto.Message, error)
	PostC2CMessage(
		ctx context.Context, userID string, msg dto.APIMessage, opt ...options.Option,
	) (*dto.Message, error)
	Transport(ctx context.Context, method, url string, body any) ([]byte, error)
}

type QQChannel struct {
	*channels.BaseChannel
	config         config.QQConfig
	api            qqAPI
	tokenSource    oauth2.TokenSource
	ctx            context.Context
	cancel         context.CancelFunc
	sessionManager botgo.SessionManager
	downloadFn     func(urlStr, filename string) string

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

	// Reconnection state
	reconnectMu   sync.Mutex
	reconnecting  bool
	stopReconnect chan struct{}

	// Rate limiting
	groupRateLimiter  *rateLimiter
	directRateLimiter *rateLimiter
}

func NewQQChannel(cfg config.QQConfig, messageBus *bus.MessageBus) (*QQChannel, error) {
	base := channels.NewBaseChannel("qq", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(cfg.MaxMessageLength),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	// Use config values with defaults for rate limiters
	groupRateMs := cfg.RateLimitGroupMs
	if groupRateMs <= 0 {
		groupRateMs = int(rateLimitGroupDefault / time.Millisecond)
	}
	directRateMs := cfg.RateLimitDirectMs
	if directRateMs <= 0 {
		directRateMs = int(rateLimitDirectDefault / time.Millisecond)
	}

	return &QQChannel{
		BaseChannel:       base,
		config:            cfg,
		dedup:             make(map[string]time.Time),
		done:              make(chan struct{}),
		groupRateLimiter:  newRateLimiter(time.Duration(groupRateMs) * time.Millisecond),
		directRateLimiter: newRateLimiter(time.Duration(directRateMs) * time.Millisecond),
	}, nil
}

func (c *QQChannel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		return fmt.Errorf("QQ app_id and app_secret not configured")
	}

	botgo.SetLogger(newBotGoLogger("botgo"))
	logger.InfoC("qq", "Starting QQ bot (WebSocket mode)")

	// Reinitialize shutdown signals for clean restart.
	c.done = make(chan struct{})
	c.stopOnce = sync.Once{}
	c.stopReconnect = make(chan struct{})
	c.reconnectMu.Lock()
	c.reconnecting = false
	c.reconnectMu.Unlock()

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

	// start session with reconnection support
	go c.startSession()

	// start dedup janitor goroutine
	go c.dedupJanitor()

	c.SetRunning(true)
	logger.InfoC("qq", "QQ bot started successfully")

	return nil
}

func (c *QQChannel) Stop(ctx context.Context) error {
	logger.InfoC("qq", "Stopping QQ bot")
	c.SetRunning(false)

	// Signal reconnection loop to stop
	c.reconnectMu.Lock()
	c.reconnecting = false
	c.reconnectMu.Unlock()

	// Close stop channel to terminate reconnect goroutine
	if c.stopReconnect != nil {
		close(c.stopReconnect)
	}

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

	// Apply rate limiting before sending
	if chatKind == "group" {
		if err := c.groupRateLimiter.waitWithContext(msg.ChatID, ctx); err != nil {
			return fmt.Errorf("qq send: %w", err)
		}
	} else {
		if err := c.directRateLimiter.waitWithContext(msg.ChatID, ctx); err != nil {
			return fmt.Errorf("qq send: %w", err)
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

	c.applyPassiveReplyMetadata(msg.ChatID, msgToCreate)

	// Sanitize URLs in group messages to avoid QQ's URL blacklist rejection.
	if chatKind == "group" {
		if msgToCreate.Content != "" {
			msgToCreate.Content = sanitizeURLs(msgToCreate.Content)
		}
		if msgToCreate.Markdown != nil && msgToCreate.Markdown.Content != "" {
			msgToCreate.Markdown.Content = sanitizeURLs(msgToCreate.Markdown.Content)
		}
	}

	// Route to group or C2C with retry.
	maxRetries := c.maxRetries()
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := c.calculateBackoff(attempt - 1)
			logger.InfoCF("qq", "Retrying send", map[string]any{
				"attempt": attempt,
				"chat_id": msg.ChatID,
				"backoff": backoff.String(),
			})

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		var err error
		if chatKind == "group" {
			_, err = c.api.PostGroupMessage(ctx, msg.ChatID, msgToCreate)
		} else {
			_, err = c.api.PostC2CMessage(ctx, msg.ChatID, msgToCreate)
		}

		if err == nil {
			return nil
		}

		lastErr = err
		logger.WarnCF("qq", "Send attempt failed", map[string]any{
			"attempt":   attempt + 1,
			"chat_id":   msg.ChatID,
			"chat_kind": chatKind,
			"error":     err.Error(),
		})

		if !isRetryableError(err) {
			break
		}
	}

	if lastErr != nil {
		logger.ErrorCF("qq", "Failed to send message after retries", map[string]any{
			"chat_id":   msg.ChatID,
			"chat_kind": chatKind,
			"error":     lastErr.Error(),
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
// QQ group/C2C media sending is a two-step flow:
// 1. Upload media to /files using a remote URL or base64-encoded local bytes.
// 2. Send a msg_type=7 message using the returned file_info.
func (c *QQChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	chatKind := c.getChatKind(msg.ChatID)

	for _, part := range msg.Parts {
		fileInfo, err := c.uploadMedia(ctx, chatKind, msg.ChatID, part)
		if err != nil {
			logger.ErrorCF("qq", "Failed to upload media", map[string]any{
				"type":    part.Type,
				"chat_id": msg.ChatID,
				"error":   err.Error(),
			})
			if errors.Is(err, channels.ErrSendFailed) {
				return err
			}
			return fmt.Errorf("qq send media: %w", channels.ErrTemporary)
		}

		if err := c.sendUploadedMedia(ctx, chatKind, msg.ChatID, part, fileInfo); err != nil {
			logger.ErrorCF("qq", "Failed to send media", map[string]any{
				"type":    part.Type,
				"chat_id": msg.ChatID,
				"error":   err.Error(),
			})
			return fmt.Errorf("qq send media: %w", channels.ErrTemporary)
		}
	}

	return nil
}

type qqMediaUpload struct {
	FileType   uint64 `json:"file_type"`
	URL        string `json:"url,omitempty"`
	FileData   string `json:"file_data,omitempty"`
	SrvSendMsg bool   `json:"srv_send_msg,omitempty"`
}

func (c *QQChannel) uploadMedia(
	ctx context.Context,
	chatKind, chatID string,
	part bus.MediaPart,
) ([]byte, error) {
	payload, err := c.buildMediaUpload(part)
	if err != nil {
		return nil, err
	}

	body, err := c.api.Transport(ctx, http.MethodPost, c.mediaUploadURL(chatKind, chatID), payload)
	if err != nil {
		return nil, err
	}

	var uploaded dto.Message
	if err := json.Unmarshal(body, &uploaded); err != nil {
		return nil, fmt.Errorf("qq decode media upload response: %w", err)
	}
	if len(uploaded.FileInfo) == 0 {
		return nil, fmt.Errorf("qq upload media: missing file_info")
	}

	return uploaded.FileInfo, nil
}

func (c *QQChannel) buildMediaUpload(part bus.MediaPart) (*qqMediaUpload, error) {
	payload := &qqMediaUpload{
		FileType: qqFileType(part.Type),
	}

	mediaRef := part.Ref
	if isHTTPURL(mediaRef) {
		payload.URL = mediaRef
		return payload, nil
	}

	store := c.GetMediaStore()
	if store == nil {
		return nil, fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	resolved, err := store.Resolve(part.Ref)
	if err != nil {
		return nil, fmt.Errorf("qq resolve media ref %q: %v: %w", part.Ref, err, channels.ErrSendFailed)
	}

	if isHTTPURL(resolved) {
		payload.URL = resolved
		return payload, nil
	}

	if limitBytes := c.maxBase64FileSizeBytes(); limitBytes > 0 {
		info, statErr := os.Stat(resolved)
		if statErr != nil {
			return nil, fmt.Errorf("qq stat local media %q: %v: %w", resolved, statErr, channels.ErrSendFailed)
		}
		if info.Size() > limitBytes {
			return nil, fmt.Errorf(
				"qq local media %q exceeds max_base64_file_size_mib (%d > %d bytes): %w",
				resolved,
				info.Size(),
				limitBytes,
				channels.ErrSendFailed,
			)
		}
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("qq read local media %q: %v: %w", resolved, err, channels.ErrSendFailed)
	}

	payload.FileData = base64.StdEncoding.EncodeToString(data)
	return payload, nil
}

func (c *QQChannel) sendUploadedMedia(
	ctx context.Context,
	chatKind, chatID string,
	part bus.MediaPart,
	fileInfo []byte,
) error {
	msg := &dto.MessageToCreate{
		Content: part.Caption,
		MsgType: dto.RichMediaMsg,
		Media: &dto.MediaInfo{
			FileInfo: fileInfo,
		},
	}
	c.applyPassiveReplyMetadata(chatID, msg)

	if chatKind == "group" && msg.Content != "" {
		msg.Content = sanitizeURLs(msg.Content)
	}

	if chatKind == "group" {
		_, err := c.api.PostGroupMessage(ctx, chatID, msg)
		return err
	}
	_, err := c.api.PostC2CMessage(ctx, chatID, msg)
	return err
}

func (c *QQChannel) applyPassiveReplyMetadata(chatID string, msg *dto.MessageToCreate) {
	if v, ok := c.lastMsgID.Load(chatID); ok {
		if msgID, ok := v.(string); ok && msgID != "" {
			msg.MsgID = msgID

			// Increment msg_seq atomically for multi-part replies.
			if counterVal, ok := c.msgSeqCounters.Load(chatID); ok {
				if counter, ok := counterVal.(*atomic.Uint64); ok {
					seq := counter.Add(1)
					msg.MsgSeq = uint32(seq)
				}
			}
		}
	}
}

func (c *QQChannel) mediaUploadURL(chatKind, chatID string) string {
	base := constant.APIDomain
	if chatKind == "group" {
		return fmt.Sprintf("%s/v2/groups/%s/files", base, chatID)
	}
	return fmt.Sprintf("%s/v2/users/%s/files", base, chatID)
}

func qqFileType(partType string) uint64 {
	switch partType {
	case "image":
		return 1
	case "video":
		return 2
	case "audio":
		return 3
	default:
		return 4
	}
}

func (c *QQChannel) maxBase64FileSizeBytes() int64 {
	if c.config.MaxBase64FileSizeMiB <= 0 {
		return 0
	}
	return c.config.MaxBase64FileSizeMiB * bytesPerMiB
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

		sender := bus.SenderInfo{
			Platform:    "qq",
			PlatformID:  data.Author.ID,
			CanonicalID: identity.BuildCanonicalID("qq", data.Author.ID),
		}

		if !c.IsAllowedSender(sender) {
			return nil
		}

		content := strings.TrimSpace(data.Content)
		mediaPaths, attachmentNotes := c.extractInboundAttachments(senderID, data.ID, data.Attachments)
		for _, note := range attachmentNotes {
			content = appendContent(content, note)
		}
		if content == "" && len(mediaPaths) == 0 {
			logger.DebugC("qq", "Received empty C2C message with no attachments, ignoring")
			return nil
		}

		logger.InfoCF("qq", "Received C2C message", map[string]any{
			"sender":      senderID,
			"length":      len(content),
			"media_count": len(mediaPaths),
		})

		// Store chat routing context.
		c.chatType.Store(senderID, "direct")
		c.lastMsgID.Store(senderID, data.ID)

		// Reset msg_seq counter for new inbound message.
		c.msgSeqCounters.Store(senderID, new(atomic.Uint64))

		metadata := map[string]string{
			"account_id": senderID,
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

		sender := bus.SenderInfo{
			Platform:    "qq",
			PlatformID:  data.Author.ID,
			CanonicalID: identity.BuildCanonicalID("qq", data.Author.ID),
		}

		if !c.IsAllowedSender(sender) {
			return nil
		}

		content := strings.TrimSpace(data.Content)
		mediaPaths, attachmentNotes := c.extractInboundAttachments(data.GroupID, data.ID, data.Attachments)
		for _, note := range attachmentNotes {
			content = appendContent(content, note)
		}

		// GroupAT event means bot is always mentioned; apply group trigger filtering.
		respond, cleaned := c.ShouldRespondInGroup(true, content)
		if !respond {
			return nil
		}
		content = cleaned
		if content == "" && len(mediaPaths) == 0 {
			logger.DebugC("qq", "Received empty group message with no attachments, ignoring")
			return nil
		}

		logger.InfoCF("qq", "Received group AT message", map[string]any{
			"sender":      senderID,
			"group":       data.GroupID,
			"length":      len(content),
			"media_count": len(mediaPaths),
		})

		// Store chat routing context using GroupID as chatID.
		c.chatType.Store(data.GroupID, "group")
		c.lastMsgID.Store(data.GroupID, data.ID)

		// Reset msg_seq counter for new inbound message.
		c.msgSeqCounters.Store(data.GroupID, new(atomic.Uint64))

		metadata := map[string]string{
			"account_id": senderID,
			"group_id":   data.GroupID,
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

func (c *QQChannel) extractInboundAttachments(
	chatID, messageID string,
	attachments []*dto.MessageAttachment,
) ([]string, []string) {
	if len(attachments) == 0 {
		return nil, nil
	}

	scope := channels.BuildMediaScope("qq", chatID, messageID)
	mediaPaths := make([]string, 0, len(attachments))
	notes := make([]string, 0, len(attachments))

	storeMedia := func(localPath string, attachment *dto.MessageAttachment) string {
		if store := c.GetMediaStore(); store != nil {
			ref, err := store.Store(localPath, media.MediaMeta{
				Filename:    qqAttachmentFilename(attachment),
				ContentType: attachment.ContentType,
				Source:      "qq",
			}, scope)
			if err == nil {
				return ref
			}
		}
		return localPath
	}

	for _, attachment := range attachments {
		if attachment == nil {
			continue
		}

		filename := qqAttachmentFilename(attachment)
		if localPath := c.downloadAttachment(attachment.URL, filename); localPath != "" {
			mediaPaths = append(mediaPaths, storeMedia(localPath, attachment))
		} else if attachment.URL != "" {
			mediaPaths = append(mediaPaths, attachment.URL)
		}

		notes = append(notes, qqAttachmentNote(attachment))
	}

	return mediaPaths, notes
}

func (c *QQChannel) downloadAttachment(urlStr, filename string) string {
	if urlStr == "" {
		return ""
	}
	if c.downloadFn != nil {
		return c.downloadFn(urlStr, filename)
	}

	return utils.DownloadFile(urlStr, filename, utils.DownloadOptions{
		LoggerPrefix: "qq",
		ExtraHeaders: c.downloadHeaders(),
	})
}

func (c *QQChannel) downloadHeaders() map[string]string {
	headers := map[string]string{}

	if c.config.AppID != "" {
		headers["X-Union-Appid"] = c.config.AppID
	}

	if c.tokenSource != nil {
		if tk, err := c.tokenSource.Token(); err == nil && tk.AccessToken != "" {
			auth := strings.TrimSpace(tk.TokenType + " " + tk.AccessToken)
			if auth != "" {
				headers["Authorization"] = auth
			}
		}
	}

	if len(headers) == 0 {
		return nil
	}
	return headers
}

func qqAttachmentFilename(attachment *dto.MessageAttachment) string {
	if attachment == nil {
		return "attachment"
	}
	if attachment.FileName != "" {
		return attachment.FileName
	}
	if attachment.URL != "" {
		if parsed, err := url.Parse(attachment.URL); err == nil {
			if base := path.Base(parsed.Path); base != "" && base != "." && base != "/" {
				return base
			}
		}
	}

	switch qqAttachmentKind(attachment) {
	case "image":
		return "image"
	case "audio":
		return "audio"
	case "video":
		return "video"
	default:
		return "attachment"
	}
}

func qqAttachmentKind(attachment *dto.MessageAttachment) string {
	if attachment == nil {
		return "file"
	}

	contentType := strings.ToLower(attachment.ContentType)
	filename := strings.ToLower(attachment.FileName)

	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case strings.HasPrefix(contentType, "video/"):
		return "video"
	case strings.HasPrefix(contentType, "audio/"), contentType == "application/ogg", contentType == "application/x-ogg":
		return "audio"
	}

	switch filepath.Ext(filename) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return "image"
	case ".mp4", ".avi", ".mov", ".webm", ".mkv":
		return "video"
	case ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".aac", ".wma", ".opus", ".silk":
		return "audio"
	default:
		return "file"
	}
}

func qqAttachmentNote(attachment *dto.MessageAttachment) string {
	filename := qqAttachmentFilename(attachment)

	switch qqAttachmentKind(attachment) {
	case "image":
		return fmt.Sprintf("[image: %s]", filename)
	case "audio":
		return fmt.Sprintf("[audio: %s]", filename)
	case "video":
		return fmt.Sprintf("[video: %s]", filename)
	default:
		return fmt.Sprintf("[file: %s]", filename)
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

func appendContent(content, suffix string) string {
	if suffix == "" {
		return content
	}
	if content == "" {
		return suffix
	}
	return content + "\n" + suffix
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

// startSession starts the WebSocket session with reconnection support.
func (c *QQChannel) startSession() {
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-c.stopReconnect:
				return
			default:
				c.runSession()
				// Session ended unexpectedly, attempt reconnect
				if c.IsRunning() {
					c.reconnect()
				}
			}
		}
	}()
}

// runSession runs a single WebSocket session.
func (c *QQChannel) runSession() {
	// Get WebSocket endpoint
	wsInfo, err := c.api.WS(c.ctx, nil, "")
	if err != nil {
		logger.ErrorCF("qq", "Failed to get websocket info", map[string]any{
			"error": err.Error(),
		})
		return
	}

	logger.InfoCF("qq", "Got WebSocket info", map[string]any{
		"shards": wsInfo.Shards,
	})

	// Create and start session
	c.sessionManager = botgo.NewSessionManager()
	intent := event.RegisterHandlers(
		c.handleC2CMessage(),
		c.handleGroupATMessage(),
	)

	if err := c.sessionManager.Start(wsInfo, c.tokenSource, &intent); err != nil {
		logger.ErrorCF("qq", "WebSocket session error", map[string]any{
			"error": err.Error(),
		})
		c.SetRunning(false)
	}
}

// reconnect attempts to reconnect with exponential backoff.
func (c *QQChannel) reconnect() {
	c.reconnectMu.Lock()
	if c.reconnecting {
		c.reconnectMu.Unlock()
		return
	}
	c.reconnecting = true
	c.reconnectMu.Unlock()

	defer func() {
		c.reconnectMu.Lock()
		c.reconnecting = false
		c.reconnectMu.Unlock()
	}()

	backoff := c.reconnectInitial()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.stopReconnect:
			return
		default:
		}

		logger.InfoCF("qq", "Attempting to reconnect", map[string]any{
			"backoff": backoff.String(),
		})

		// Reset internal state for new session
		c.resetSessionState()

		c.runSession()

		if c.IsRunning() {
			logger.InfoC("qq", "Reconnected successfully")
			return
		}

		logger.WarnCF("qq", "Reconnect failed, retrying", map[string]any{
			"backoff": backoff.String(),
		})

		select {
		case <-c.ctx.Done():
			return
		case <-c.stopReconnect:
			return
		case <-time.After(backoff):
			reconnectMax := c.reconnectMax()
			if backoff < reconnectMax {
				backoff = time.Duration(float64(backoff) * reconnectMultiplier)
				if backoff > reconnectMax {
					backoff = reconnectMax
				}
			}
		}
	}
}

// resetSessionState clears transient state that may be invalid after reconnection.
func (c *QQChannel) resetSessionState() {
	// Clear transient state that may be invalid after reconnection
	c.chatType.Range(func(key, value any) bool {
		c.chatType.Delete(key)
		return true
	})
	c.lastMsgID.Range(func(key, value any) bool {
		c.lastMsgID.Delete(key)
		return true
	})
	c.msgSeqCounters.Range(func(key, value any) bool {
		c.msgSeqCounters.Delete(key)
		return true
	})

	// Clear rate limiters to allow immediate sending after reconnection
	if c.groupRateLimiter != nil {
		c.groupRateLimiter.clearAll()
	}
	if c.directRateLimiter != nil {
		c.directRateLimiter.clearAll()
	}

	// Re-register reasoning channel if configured
	if c.config.ReasoningChannelID != "" {
		c.chatType.Store(c.config.ReasoningChannelID, "group")
	}
}

// isRetryableError returns true if the error should be retried.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Common transient error patterns
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "context deadline") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "temporary failure") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503")
}

// calculateBackoff returns the next backoff duration with exponential increase.
func (c *QQChannel) calculateBackoff(attempt int) time.Duration {
	initialDelay := c.retryInitialDelay()
	maxDelay := c.retryMaxDelay()
	backoff := initialDelay * time.Duration(1<<uint(attempt))
	if backoff > maxDelay {
		backoff = maxDelay
	}
	return backoff
}

// reconnectInitial returns the configured or default initial reconnect interval.
func (c *QQChannel) reconnectInitial() time.Duration {
	if c.config.ReconnectInitialMs > 0 {
		return time.Duration(c.config.ReconnectInitialMs) * time.Millisecond
	}
	return reconnectInitialDefault
}

// reconnectMax returns the configured or default max reconnect interval.
func (c *QQChannel) reconnectMax() time.Duration {
	if c.config.ReconnectMaxMs > 0 {
		return time.Duration(c.config.ReconnectMaxMs) * time.Millisecond
	}
	return reconnectMaxDefault
}

// maxRetries returns the configured or default max retry count.
func (c *QQChannel) maxRetries() int {
	if c.config.MaxRetries > 0 {
		return c.config.MaxRetries
	}
	return maxRetriesDefault
}

// retryInitialDelay returns the configured or default initial retry delay.
func (c *QQChannel) retryInitialDelay() time.Duration {
	if c.config.RetryInitialDelayMs > 0 {
		return time.Duration(c.config.RetryInitialDelayMs) * time.Millisecond
	}
	return retryInitialDelayDefault
}

// retryMaxDelay returns the configured or default max retry delay.
func (c *QQChannel) retryMaxDelay() time.Duration {
	if c.config.RetryMaxDelayMs > 0 {
		return time.Duration(c.config.RetryMaxDelayMs) * time.Millisecond
	}
	return retryMaxDelayDefault
}
