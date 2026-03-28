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
	if c.config.AppID == "" || c.config.AppSecret.String() == "" {
		return fmt.Errorf("QQ app_id and app_secret not configured")
	}

	botgo.SetLogger(newBotGoLogger("botgo"))
	logger.InfoC("qq", "Starting QQ bot (WebSocket mode)")

	// Reinitialize shutdown signal for clean restart.
	c.done = make(chan struct{})
	c.stopOnce = sync.Once{}

	// create token source
	credentials := &token.QQBotCredentials{
		AppID:     c.config.AppID,
		AppSecret: c.config.AppSecret.String(),
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
	FileName   string `json:"file_name,omitempty"`
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
	payload := &qqMediaUpload{}

	mediaRef := part.Ref
	if isHTTPURL(mediaRef) {
		payload.FileType = qqFileType(c.outboundMediaType(part, ""))
		payload.URL = mediaRef
		payload.FileName = qqUploadFilename(part, mediaRef, payload.FileType)
		return payload, nil
	}

	store := c.GetMediaStore()
	if store == nil {
		return nil, fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	resolved, meta, err := store.ResolveWithMeta(part.Ref)
	if err != nil {
		return nil, fmt.Errorf("qq resolve media ref %q: %v: %w", part.Ref, err, channels.ErrSendFailed)
	}
	if part.Filename == "" {
		part.Filename = meta.Filename
	}
	if part.ContentType == "" {
		part.ContentType = meta.ContentType
	}

	if isHTTPURL(resolved) {
		payload.FileType = qqFileType(c.outboundMediaType(part, ""))
		payload.URL = resolved
		payload.FileName = qqUploadFilename(part, resolved, payload.FileType)
		return payload, nil
	}
	payload.FileType = qqFileType(c.outboundMediaType(part, resolved))
	payload.FileName = qqUploadFilename(part, resolved, payload.FileType)

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

func qqUploadFilename(part bus.MediaPart, resolved string, fileType uint64) string {
	if fileType != qqFileType("file") {
		return ""
	}
	if part.Filename != "" {
		return part.Filename
	}
	if isHTTPURL(resolved) {
		if parsed, err := url.Parse(resolved); err == nil {
			if base := path.Base(parsed.Path); base != "" && base != "." && base != "/" {
				return base
			}
		}
		return ""
	}

	if base := filepath.Base(resolved); base != "" && base != "." {
		return base
	}
	return ""
}

func (c *QQChannel) outboundMediaType(part bus.MediaPart, localPath string) string {
	if part.Type != "audio" {
		return part.Type
	}

	if localPath == "" {
		logger.InfoCF("qq", "Sending audio as file because duration is unavailable", map[string]any{
			"ref":      part.Ref,
			"filename": part.Filename,
		})
		return "file"
	}

	duration, ok, err := qqAudioDuration(localPath, part.Filename, part.ContentType)
	if err != nil {
		logger.WarnCF("qq", "Failed to detect audio duration, sending as file", map[string]any{
			"ref":      part.Ref,
			"filename": part.Filename,
			"error":    err.Error(),
		})
		return "file"
	}
	if !ok {
		logger.InfoCF("qq", "Sending audio as file because duration is unavailable", map[string]any{
			"ref":      part.Ref,
			"filename": part.Filename,
		})
		return "file"
	}
	if duration > qqVoiceMaxDuration {
		logger.InfoCF("qq", "Sending audio as file because it exceeds QQ voice limit", map[string]any{
			"ref":              part.Ref,
			"filename":         part.Filename,
			"duration_seconds": duration.Seconds(),
			"limit_seconds":    qqVoiceMaxDuration.Seconds(),
		})
		return "file"
	}

	return "audio"
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
				Filename:      qqAttachmentFilename(attachment),
				ContentType:   attachment.ContentType,
				Source:        "qq",
				CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
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
