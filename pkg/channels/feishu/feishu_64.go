//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// errCodeTenantTokenInvalid is the Feishu API error code for an expired/revoked
// tenant_access_token. The Lark SDK's built-in retry does not clear its cache
// on this error, so we do it ourselves.
const errCodeTenantTokenInvalid = 99991663

type FeishuChannel struct {
	*channels.BaseChannel
	config     config.FeishuConfig
	client     *lark.Client
	wsClient   *larkws.Client
	tokenCache *tokenCache // custom cache that supports invalidation

	botOpenID atomic.Value // stores string; populated lazily for @mention detection

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewFeishuChannel(cfg config.FeishuConfig, bus *bus.MessageBus) (*FeishuChannel, error) {
	base := channels.NewBaseChannel("feishu", cfg, bus, cfg.AllowFrom,
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	tc := newTokenCache()
	opts := []lark.ClientOptionFunc{lark.WithTokenCache(tc)}
	if cfg.IsLark {
		opts = append(opts, lark.WithOpenBaseUrl(lark.LarkBaseUrl))
	}
	ch := &FeishuChannel{
		BaseChannel: base,
		config:      cfg,
		tokenCache:  tc,
		client:      lark.NewClient(cfg.AppID, cfg.AppSecret.String(), opts...),
	}
	ch.SetOwner(ch)
	return ch, nil
}

func (c *FeishuChannel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret.String() == "" {
		return fmt.Errorf("feishu app_id or app_secret is empty")
	}

	// Fetch bot open_id via API for reliable @mention detection.
	if err := c.fetchBotOpenID(ctx); err != nil {
		logger.ErrorCF("feishu", "Failed to fetch bot open_id, @mention detection may not work", map[string]any{
			"error": err.Error(),
		})
	}

	dispatcher := larkdispatcher.NewEventDispatcher(c.config.VerificationToken.String(), c.config.EncryptKey.String()).
		OnP2MessageReceiveV1(c.handleMessageReceive)

	runCtx, cancel := context.WithCancel(ctx)

	c.mu.Lock()
	c.cancel = cancel
	domain := lark.FeishuBaseUrl
	if c.config.IsLark {
		domain = lark.LarkBaseUrl
	}
	c.wsClient = larkws.NewClient(
		c.config.AppID,
		c.config.AppSecret.String(),
		larkws.WithEventHandler(dispatcher),
		larkws.WithDomain(domain),
	)
	wsClient := c.wsClient
	c.mu.Unlock()

	c.SetRunning(true)
	logger.InfoC("feishu", "Feishu channel started (websocket mode)")

	go func() {
		if err := wsClient.Start(runCtx); err != nil {
			logger.ErrorCF("feishu", "Feishu websocket stopped with error", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

func (c *FeishuChannel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.wsClient = nil
	c.mu.Unlock()

	c.SetRunning(false)
	logger.InfoC("feishu", "Feishu channel stopped")
	return nil
}

// Send sends a message using Interactive Card format for markdown rendering.
// Falls back to plain text message if card sending fails (e.g., table limit exceeded).
func (c *FeishuChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	if msg.ChatID == "" {
		return nil, fmt.Errorf("chat ID is empty: %w", channels.ErrSendFailed)
	}

	// Build interactive card with markdown content
	cardContent, err := buildMarkdownCard(msg.Content)
	if err != nil {
		// If card build fails, fall back to plain text
		return nil, c.sendText(ctx, msg.ChatID, msg.Content)
	}

	// First attempt: try sending as interactive card
	err = c.sendCard(ctx, msg.ChatID, cardContent)
	if err == nil {
		return nil, nil
	}

	// Check if error is due to card table limit (error code 11310)
	// See: https://open.feishu.cn/document/server-docs/im-api/message-content-description/create_json
	errMsg := err.Error()
	isCardLimitError := strings.Contains(errMsg, "11310")

	if isCardLimitError {
		logger.WarnCF("feishu", "Card send failed (table limit), falling back to text message", map[string]any{
			"chat_id": msg.ChatID,
			"error":   errMsg,
		})

		// Second attempt: fall back to plain text message
		textErr := c.sendText(ctx, msg.ChatID, msg.Content)
		if textErr == nil {
			return nil, nil
		}
		// If text also fails, return the text error
		return nil, textErr
	}

	// For other errors, return the original card error
	return nil, err
}

// EditMessage implements channels.MessageEditor.
// Uses Message.Patch to update an interactive card message.
func (c *FeishuChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	cardContent, err := buildMarkdownCard(content)
	if err != nil {
		return fmt.Errorf("feishu edit: card build failed: %w", err)
	}

	req := larkim.NewPatchMessageReqBuilder().
		MessageId(messageID).
		Body(larkim.NewPatchMessageReqBodyBuilder().Content(cardContent).Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Patch(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu edit: %w", err)
	}
	if !resp.Success() {
		c.invalidateTokenOnAuthError(resp.Code)
		return fmt.Errorf("feishu edit api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}
	return nil
}

// SendPlaceholder implements channels.PlaceholderCapable.
// Sends an interactive card with placeholder text and returns its message ID.
func (c *FeishuChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	if !c.config.Placeholder.Enabled {
		logger.DebugCF("feishu", "Placeholder disabled, skipping", map[string]any{
			"chat_id": chatID,
		})
		return "", nil
	}

	text := c.config.Placeholder.GetRandomText()

	cardContent, err := buildMarkdownCard(text)
	if err != nil {
		return "", fmt.Errorf("feishu placeholder: card build failed: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeInteractive).
			Content(cardContent).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("feishu placeholder send: %w", err)
	}
	if !resp.Success() {
		c.invalidateTokenOnAuthError(resp.Code)
		return "", fmt.Errorf("feishu placeholder api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}

	if resp.Data != nil && resp.Data.MessageId != nil {
		return *resp.Data.MessageId, nil
	}
	return "", nil
}

// ReactToMessage implements channels.ReactionCapable.
// Adds a reaction (randomly chosen from config) and returns an undo function to remove it.
func (c *FeishuChannel) ReactToMessage(ctx context.Context, chatID, messageID string) (func(), error) {
	// Get emoji list from config (Feishu emoji_type keys, e.g. Pin, THUMBSUP).
	// Ignore empty entries so a list like ["", "Pin"] does not randomly pick "" (API 231001).
	var candidates []string
	for _, e := range c.config.RandomReactionEmoji {
		e = strings.TrimSpace(e)
		if e != "" {
			candidates = append(candidates, e)
		}
	}
	chosenEmoji := "Pin"
	if len(candidates) > 0 {
		chosenEmoji = candidates[rand.Intn(len(candidates))]
	}

	req := larkim.NewCreateMessageReactionReqBuilder().
		MessageId(messageID).
		Body(larkim.NewCreateMessageReactionReqBodyBuilder().
			ReactionType(larkim.NewEmojiBuilder().EmojiType(chosenEmoji).Build()).
			Build()).
		Build()

	resp, err := c.client.Im.V1.MessageReaction.Create(ctx, req)
	if err != nil {
		logger.ErrorCF("feishu", "Failed to add reaction", map[string]any{
			"emoji":      chosenEmoji,
			"message_id": messageID,
			"error":      err.Error(),
		})
		return func() {}, fmt.Errorf("feishu react: %w", err)
	}
	if !resp.Success() {
		c.invalidateTokenOnAuthError(resp.Code)
		logger.ErrorCF("feishu", "Reaction API error", map[string]any{
			"emoji":      chosenEmoji,
			"message_id": messageID,
			"code":       resp.Code,
			"msg":        resp.Msg,
		})
		return func() {}, fmt.Errorf("feishu react api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}

	var reactionID string
	if resp.Data != nil && resp.Data.ReactionId != nil {
		reactionID = *resp.Data.ReactionId
	}
	if reactionID == "" {
		return func() {}, nil
	}

	var undone atomic.Bool
	undo := func() {
		if !undone.CompareAndSwap(false, true) {
			return
		}
		delReq := larkim.NewDeleteMessageReactionReqBuilder().
			MessageId(messageID).
			ReactionId(reactionID).
			Build()
		_, _ = c.client.Im.V1.MessageReaction.Delete(context.Background(), delReq)
	}
	return undo, nil
}

// SendMedia implements channels.MediaSender.
// Uploads images/files via Feishu API then sends as messages.
func (c *FeishuChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	if msg.ChatID == "" {
		return nil, fmt.Errorf("chat ID is empty: %w", channels.ErrSendFailed)
	}

	store := c.GetMediaStore()
	if store == nil {
		return nil, fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	for _, part := range msg.Parts {
		if err := c.sendMediaPart(ctx, msg.ChatID, part, store); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// sendMediaPart resolves and sends a single media part.
func (c *FeishuChannel) sendMediaPart(
	ctx context.Context,
	chatID string,
	part bus.MediaPart,
	store media.MediaStore,
) error {
	localPath, err := store.Resolve(part.Ref)
	if err != nil {
		logger.ErrorCF("feishu", "Failed to resolve media ref", map[string]any{
			"ref":   part.Ref,
			"error": err.Error(),
		})
		return nil // skip this part
	}

	file, err := os.Open(localPath)
	if err != nil {
		logger.ErrorCF("feishu", "Failed to open media file", map[string]any{
			"path":  localPath,
			"error": err.Error(),
		})
		return nil // skip this part
	}
	defer file.Close()

	switch part.Type {
	case "image":
		err = c.sendImage(ctx, chatID, file)
	default:
		filename := part.Filename
		if filename == "" {
			filename = "file"
		}
		err = c.sendFile(ctx, chatID, file, filename, part.Type)
	}

	if err != nil {
		logger.ErrorCF("feishu", "Failed to send media", map[string]any{
			"type":  part.Type,
			"error": err.Error(),
		})
		return fmt.Errorf("feishu send media: %w", channels.ErrTemporary)
	}
	return nil
}

// --- Inbound message handling ---

func (c *FeishuChannel) handleMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}

	message := event.Event.Message
	sender := event.Event.Sender

	chatID := stringValue(message.ChatId)
	if chatID == "" {
		return nil
	}

	senderID := extractFeishuSenderID(sender)
	if senderID == "" {
		senderID = "unknown"
	}

	messageType := stringValue(message.MessageType)
	messageID := stringValue(message.MessageId)
	rawContent := stringValue(message.Content)

	// Check allowlist early to avoid downloading media for rejected senders.
	// BaseChannel.HandleMessage will check again, but this avoids wasted network I/O.
	senderInfo := bus.SenderInfo{
		Platform:    "feishu",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("feishu", senderID),
	}
	if !c.IsAllowedSender(senderInfo) {
		return nil
	}

	// Extract content based on message type
	content := extractContent(messageType, rawContent)

	// Handle media messages (download and store)
	var mediaRefs []string
	if store := c.GetMediaStore(); store != nil && messageID != "" {
		mediaRefs = c.downloadInboundMedia(ctx, chatID, messageID, messageType, rawContent, store)
	}

	// For interactive cards, pass external image URLs via media refs.
	// Keep content as valid raw JSON for downstream parsing.
	if messageType == larkim.MsgTypeInteractive {
		_, externalURLs := extractCardImageKeys(rawContent)
		if len(externalURLs) > 0 {
			mediaRefs = append(mediaRefs, externalURLs...)
		}
	}

	// Append media tags to content (like Telegram does)
	content = appendMediaTags(content, messageType, mediaRefs)

	if content == "" {
		content = "[empty message]"
	}

	metadata := map[string]string{}
	if messageID != "" {
		metadata["message_id"] = messageID
	}
	if messageType != "" {
		metadata["message_type"] = messageType
	}
	chatType := stringValue(message.ChatType)
	if chatType != "" {
		metadata["chat_type"] = chatType
	}
	if sender != nil && sender.TenantKey != nil {
		metadata["tenant_key"] = *sender.TenantKey
	}

	var peer bus.Peer
	if chatType == "p2p" {
		peer = bus.Peer{Kind: "direct", ID: senderID}
	} else {
		peer = bus.Peer{Kind: "group", ID: chatID}

		// Check if bot was mentioned
		isMentioned := c.isBotMentioned(message)

		// Strip mention placeholders from content before group trigger check
		if len(message.Mentions) > 0 {
			content = stripMentionPlaceholders(content, message.Mentions)
		}

		// In group chats, apply unified group trigger filtering
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			return nil
		}
		content = cleaned
	}

	logger.InfoCF("feishu", "Feishu message received", map[string]any{
		"sender_id":  senderID,
		"chat_id":    chatID,
		"message_id": messageID,
		"preview":    utils.Truncate(content, 80),
	})

	c.HandleMessage(ctx, peer, messageID, senderID, chatID, content, mediaRefs, metadata, senderInfo)
	return nil
}

// --- Internal helpers ---

// fetchBotOpenID calls the Feishu bot info API to retrieve and store the bot's open_id.
func (c *FeishuChannel) fetchBotOpenID(ctx context.Context) error {
	resp, err := c.client.Do(ctx, &larkcore.ApiReq{
		HttpMethod:                http.MethodGet,
		ApiPath:                   "/open-apis/bot/v3/info",
		SupportedAccessTokenTypes: []larkcore.AccessTokenType{larkcore.AccessTokenTypeTenant},
	})
	if err != nil {
		return fmt.Errorf("bot info request: %w", err)
	}

	var result struct {
		Code int `json:"code"`
		Bot  struct {
			OpenID string `json:"open_id"`
		} `json:"bot"`
	}
	if err := json.Unmarshal(resp.RawBody, &result); err != nil {
		return fmt.Errorf("bot info parse: %w", err)
	}
	if result.Code != 0 {
		c.invalidateTokenOnAuthError(result.Code)
		return fmt.Errorf("bot info api error (code=%d)", result.Code)
	}
	if result.Bot.OpenID == "" {
		return fmt.Errorf("bot info: empty open_id")
	}

	c.botOpenID.Store(result.Bot.OpenID)
	logger.InfoCF("feishu", "Fetched bot open_id from API", map[string]any{
		"open_id": result.Bot.OpenID,
	})
	return nil
}

// isBotMentioned checks if the bot was @mentioned in the message.
func (c *FeishuChannel) isBotMentioned(message *larkim.EventMessage) bool {
	if message.Mentions == nil {
		return false
	}

	knownID, _ := c.botOpenID.Load().(string)
	if knownID == "" {
		logger.DebugCF("feishu", "Bot open_id unknown, cannot detect @mention", nil)
		return false
	}

	for _, m := range message.Mentions {
		if m.Id == nil {
			continue
		}
		if m.Id.OpenId != nil && *m.Id.OpenId == knownID {
			return true
		}
	}
	return false
}

// extractContent extracts text content from different message types.
func extractContent(messageType, rawContent string) string {
	if rawContent == "" {
		return ""
	}

	switch messageType {
	case larkim.MsgTypeText:
		var textPayload struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(rawContent), &textPayload); err == nil {
			return textPayload.Text
		}
		return rawContent

	case larkim.MsgTypePost:
		// Pass raw JSON to LLM — structured rich text is more informative than flattened plain text
		return rawContent

	case larkim.MsgTypeInteractive:
		// Pass raw JSON to LLM — structured card is more informative than flattened text
		return rawContent

	case larkim.MsgTypeImage:
		// Image messages don't have text content
		return ""

	case larkim.MsgTypeFile, larkim.MsgTypeAudio, larkim.MsgTypeMedia:
		// File/audio/video messages may have a filename
		name := extractFileName(rawContent)
		if name != "" {
			return name
		}
		return ""

	default:
		return rawContent
	}
}

// downloadInboundMedia downloads media from inbound messages and stores in MediaStore.
func (c *FeishuChannel) downloadInboundMedia(
	ctx context.Context,
	chatID, messageID, messageType, rawContent string,
	store media.MediaStore,
) []string {
	var refs []string
	scope := channels.BuildMediaScope("feishu", chatID, messageID)

	switch messageType {
	case larkim.MsgTypeImage:
		imageKey := extractImageKey(rawContent)
		if imageKey == "" {
			return nil
		}
		ref := c.downloadResource(ctx, messageID, imageKey, "image", ".jpg", store, scope)
		if ref != "" {
			refs = append(refs, ref)
		}

	case larkim.MsgTypeInteractive:
		// Extract and download images embedded in interactive cards
		feishuKeys, _ := extractCardImageKeys(rawContent)
		// Download Feishu-hosted images via API
		for _, imageKey := range feishuKeys {
			ref := c.downloadResource(ctx, messageID, imageKey, "image", ".jpg", store, scope)
			if ref != "" {
				refs = append(refs, ref)
			}
		}
		// External URLs are passed directly to LLM, not downloaded

	case larkim.MsgTypeFile, larkim.MsgTypeAudio, larkim.MsgTypeMedia:
		fileKey := extractFileKey(rawContent)
		if fileKey == "" {
			return nil
		}
		// Derive a fallback extension from the message type.
		var ext string
		switch messageType {
		case larkim.MsgTypeAudio:
			ext = ".ogg"
		case larkim.MsgTypeMedia:
			ext = ".mp4"
		default:
			ext = "" // generic file — rely on resp.FileName
		}
		ref := c.downloadResource(ctx, messageID, fileKey, "file", ext, store, scope)
		if ref != "" {
			refs = append(refs, ref)
		}
	}

	return refs
}

// downloadResource downloads a message resource (image/file) from Feishu,
// writes it to the project media directory, and stores the reference in MediaStore.
// fallbackExt (e.g. ".jpg") is appended when the resolved filename has no extension.
func (c *FeishuChannel) downloadResource(
	ctx context.Context,
	messageID, fileKey, resourceType, fallbackExt string,
	store media.MediaStore,
	scope string,
) string {
	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(messageID).
		FileKey(fileKey).
		Type(resourceType).
		Build()

	resp, err := c.client.Im.V1.MessageResource.Get(ctx, req)
	if err != nil {
		logger.ErrorCF("feishu", "Failed to download resource", map[string]any{
			"message_id": messageID,
			"file_key":   fileKey,
			"error":      err.Error(),
		})
		return ""
	}
	if !resp.Success() {
		c.invalidateTokenOnAuthError(resp.Code)
		logger.ErrorCF("feishu", "Resource download api error", map[string]any{
			"code": resp.Code,
			"msg":  resp.Msg,
		})
		return ""
	}

	if resp.File == nil {
		return ""
	}
	// Safely close the underlying reader if it implements io.Closer (e.g. HTTP response body).
	if closer, ok := resp.File.(io.Closer); ok {
		defer closer.Close()
	}

	filename := resp.FileName
	if filename == "" {
		filename = fileKey
	}
	// If filename still has no extension, append the fallback (like Telegram's ext parameter).
	if filepath.Ext(filename) == "" && fallbackExt != "" {
		filename += fallbackExt
	}

	// Write to the shared picoclaw_media directory using a unique name to avoid collisions.
	mediaDir := media.TempDir()
	if mkdirErr := os.MkdirAll(mediaDir, 0o700); mkdirErr != nil {
		logger.ErrorCF("feishu", "Failed to create media directory", map[string]any{
			"error": mkdirErr.Error(),
		})
		return ""
	}
	ext := filepath.Ext(filename)
	localPath := filepath.Join(mediaDir, utils.SanitizeFilename(messageID+"-"+fileKey+ext))

	out, err := os.Create(localPath)
	if err != nil {
		logger.ErrorCF("feishu", "Failed to create local file for resource", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	if _, copyErr := io.Copy(out, resp.File); copyErr != nil {
		out.Close()
		os.Remove(localPath)
		logger.ErrorCF("feishu", "Failed to write resource to file", map[string]any{
			"error": copyErr.Error(),
		})
		return ""
	}
	out.Close()

	ref, err := store.Store(localPath, media.MediaMeta{
		Filename:      filename,
		Source:        "feishu",
		CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
	}, scope)
	if err != nil {
		logger.ErrorCF("feishu", "Failed to store downloaded resource", map[string]any{
			"file_key": fileKey,
			"error":    err.Error(),
		})
		os.Remove(localPath)
		return ""
	}

	return ref
}

// appendMediaTags appends media type tags to content (like Telegram's "[image: photo]").
// For interactive cards, media tags are not appended because content is raw JSON
// and appending would produce invalid JSON format.
func appendMediaTags(content, messageType string, mediaRefs []string) string {
	if len(mediaRefs) == 0 {
		return content
	}

	// Don't append tags to JSON content (interactive cards) - would produce invalid JSON
	if messageType == larkim.MsgTypeInteractive {
		return content
	}

	var tag string
	switch messageType {
	case larkim.MsgTypeImage:
		tag = "[image: photo]"
	case larkim.MsgTypeAudio:
		tag = "[audio]"
	case larkim.MsgTypeMedia:
		tag = "[video]"
	case larkim.MsgTypeFile:
		tag = "[file]"
	default:
		tag = "[attachment]"
	}

	if content == "" {
		return tag
	}
	return content + " " + tag
}

// sendCard sends an interactive card message to a chat.
func (c *FeishuChannel) sendCard(ctx context.Context, chatID, cardContent string) error {
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeInteractive).
			Content(cardContent).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu send card: %w", channels.ErrTemporary)
	}

	if !resp.Success() {
		c.invalidateTokenOnAuthError(resp.Code)
		return fmt.Errorf("feishu api error (code=%d msg=%s): %w", resp.Code, resp.Msg, channels.ErrTemporary)
	}

	logger.DebugCF("feishu", "Feishu card message sent", map[string]any{
		"chat_id": chatID,
	})

	return nil
}

// sendText sends a plain text message to a chat (fallback when card fails).
func (c *FeishuChannel) sendText(ctx context.Context, chatID, text string) error {
	content, _ := json.Marshal(map[string]string{"text": text})

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeText).
			Content(string(content)).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu send text: %w", channels.ErrTemporary)
	}

	if !resp.Success() {
		return fmt.Errorf("feishu text api error (code=%d msg=%s): %w", resp.Code, resp.Msg, channels.ErrTemporary)
	}

	logger.DebugCF("feishu", "Feishu text message sent (fallback)", map[string]any{
		"chat_id": chatID,
	})

	return nil
}

// sendImage uploads an image and sends it as a message.
func (c *FeishuChannel) sendImage(ctx context.Context, chatID string, file *os.File) error {
	// Upload image to get image_key
	uploadReq := larkim.NewCreateImageReqBuilder().
		Body(larkim.NewCreateImageReqBodyBuilder().
			ImageType("message").
			Image(file).
			Build()).
		Build()

	uploadResp, err := c.client.Im.V1.Image.Create(ctx, uploadReq)
	if err != nil {
		return fmt.Errorf("feishu image upload: %w", err)
	}
	if !uploadResp.Success() {
		c.invalidateTokenOnAuthError(uploadResp.Code)
		return fmt.Errorf("feishu image upload api error (code=%d msg=%s)", uploadResp.Code, uploadResp.Msg)
	}
	if uploadResp.Data == nil || uploadResp.Data.ImageKey == nil {
		return fmt.Errorf("feishu image upload: no image_key returned")
	}

	imageKey := *uploadResp.Data.ImageKey

	// Send image message
	content, _ := json.Marshal(map[string]string{"image_key": imageKey})
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeImage).
			Content(string(content)).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu image send: %w", err)
	}
	if !resp.Success() {
		c.invalidateTokenOnAuthError(resp.Code)
		return fmt.Errorf("feishu image send api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}
	return nil
}

// sendFile uploads a file and sends it as a message.
func (c *FeishuChannel) sendFile(ctx context.Context, chatID string, file *os.File, filename, fileType string) error {
	// Map part type to Feishu file type
	feishuFileType := "stream"
	switch fileType {
	case "audio":
		feishuFileType = "opus"
	case "video":
		feishuFileType = "mp4"
	}

	// Upload file to get file_key
	uploadReq := larkim.NewCreateFileReqBuilder().
		Body(larkim.NewCreateFileReqBodyBuilder().
			FileType(feishuFileType).
			FileName(filename).
			File(file).
			Build()).
		Build()

	uploadResp, err := c.client.Im.V1.File.Create(ctx, uploadReq)
	if err != nil {
		return fmt.Errorf("feishu file upload: %w", err)
	}
	if !uploadResp.Success() {
		c.invalidateTokenOnAuthError(uploadResp.Code)
		return fmt.Errorf("feishu file upload api error (code=%d msg=%s)", uploadResp.Code, uploadResp.Msg)
	}
	if uploadResp.Data == nil || uploadResp.Data.FileKey == nil {
		return fmt.Errorf("feishu file upload: no file_key returned")
	}

	fileKey := *uploadResp.Data.FileKey

	// Send file message
	content, _ := json.Marshal(map[string]string{"file_key": fileKey})
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeFile).
			Content(string(content)).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu file send: %w", err)
	}
	if !resp.Success() {
		c.invalidateTokenOnAuthError(resp.Code)
		return fmt.Errorf("feishu file send api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}
	return nil
}

func extractFeishuSenderID(sender *larkim.EventSender) string {
	if sender == nil || sender.SenderId == nil {
		return ""
	}

	if sender.SenderId.UserId != nil && *sender.SenderId.UserId != "" {
		return *sender.SenderId.UserId
	}
	if sender.SenderId.OpenId != nil && *sender.SenderId.OpenId != "" {
		return *sender.SenderId.OpenId
	}
	if sender.SenderId.UnionId != nil && *sender.SenderId.UnionId != "" {
		return *sender.SenderId.UnionId
	}

	return ""
}

// invalidateTokenOnAuthError clears the cached tenant_access_token when the
// Feishu API reports it as invalid (99991663), so the next request fetches a
// fresh one. The Lark SDK's built-in retry does not clear the cache, causing
// all API calls to fail until the token naturally expires (~2 hours).
func (c *FeishuChannel) invalidateTokenOnAuthError(code int) {
	if code == errCodeTenantTokenInvalid {
		c.tokenCache.InvalidateAll()
		logger.WarnCF("feishu", "Invalidated cached token due to auth error", nil)
	}
}
