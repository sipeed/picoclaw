package line

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	lineContentEndpoint  = "https://api-data.line.me/v2/bot/message/%s/content"
	lineReplyTokenMaxAge = 25 * time.Second
)

type replyTokenEntry struct {
	token     string
	timestamp time.Time
}

// LINEChannel implements the Channel interface for LINE Official Account
// using the LINE Messaging API with HTTP webhook for receiving messages
// and REST API for sending messages.
type LINEChannel struct {
	*channels.BaseChannel
	config         config.LINEConfig
	client         *messaging_api.MessagingApiAPI
	botUserID      string   // Bot's user ID
	botBasicID     string   // Bot's basic ID (e.g. @216ru...)
	botDisplayName string   // Bot's display name for text-based mention detection
	replyTokens    sync.Map // chatID -> replyTokenEntry
	quoteTokens    sync.Map // chatID -> quoteToken (string)
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewLINEChannel creates a new LINE channel instance.
func NewLINEChannel(cfg config.LINEConfig, messageBus *bus.MessageBus) (*LINEChannel, error) {
	if cfg.ChannelSecret == "" || cfg.ChannelAccessToken == "" {
		return nil, fmt.Errorf("line channel_secret and channel_access_token are required")
	}

	client, err := messaging_api.NewMessagingApiAPI(cfg.ChannelAccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create LINE messaging client: %w", err)
	}

	base := channels.NewBaseChannel("line", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(5000),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &LINEChannel{
		BaseChannel: base,
		config:      cfg,
		client:      client,
	}, nil
}

// Start initializes the LINE channel.
func (c *LINEChannel) Start(ctx context.Context) error {
	logger.InfoC("line", "Starting LINE channel (Webhook Mode)")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Fetch bot profile to get bot's userId for mention detection
	info, err := c.client.GetBotInfo()
	if err != nil {
		logger.WarnCF("line", "Failed to fetch bot info (mention detection disabled)", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		c.botUserID = info.UserId
		c.botBasicID = info.BasicId
		c.botDisplayName = info.DisplayName
		logger.InfoCF("line", "Bot info fetched", map[string]interface{}{
			"bot_user_id":  c.botUserID,
			"basic_id":     c.botBasicID,
			"display_name": c.botDisplayName,
		})
	}

	c.SetRunning(true)
	logger.InfoC("line", "LINE channel started (Webhook Mode)")
	return nil
}

// Stop gracefully stops the LINE channel.
func (c *LINEChannel) Stop(ctx context.Context) error {
	logger.InfoC("line", "Stopping LINE channel")

	if c.cancel != nil {
		c.cancel()
	}

	c.SetRunning(false)
	logger.InfoC("line", "LINE channel stopped")
	return nil
}

// WebhookPath returns the path for registering on the shared HTTP server.
func (c *LINEChannel) WebhookPath() string {
	if c.config.WebhookPath != "" {
		return c.config.WebhookPath
	}
	return "/webhook/line"
}

// ServeHTTP implements http.Handler for the shared HTTP server.
func (c *LINEChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.webhookHandler(w, r)
}

// webhookHandler handles incoming LINE webhook requests.
func (c *LINEChannel) webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cb, err := webhook.ParseRequest(c.config.ChannelSecret, r)
	if err != nil {
		if err == webhook.ErrInvalidSignature {
			logger.WarnC("line", "Invalid webhook signature")
			http.Error(w, "Forbidden", http.StatusForbidden)
		} else {
			logger.ErrorCF("line", "Failed to parse webhook request", map[string]interface{}{
				"error": err.Error(),
			})
			http.Error(w, "Bad request", http.StatusBadRequest)
		}
		return
	}

	// Return 200 immediately, process events asynchronously
	w.WriteHeader(http.StatusOK)

	for _, event := range cb.Events {
		go c.processEvent(event)
	}
}

func (c *LINEChannel) processEvent(event webhook.EventInterface) {
	msgEvent, ok := event.(webhook.MessageEvent)
	if !ok {
		logger.DebugCF("line", "Ignoring non-message event", map[string]interface{}{
			"type": event.GetType(),
		})
		return
	}

	senderID, chatID, sourceType := c.resolveSource(msgEvent.Source)
	isGroup := sourceType == "group" || sourceType == "room"

	// Store reply token for later use
	if msgEvent.ReplyToken != "" {
		c.replyTokens.Store(chatID, replyTokenEntry{
			token:     msgEvent.ReplyToken,
			timestamp: time.Now(),
		})
	}

	var content string
	var mediaPaths []string
	var messageID string
	var isMentioned bool
	if f := reflect.Indirect(reflect.ValueOf(msgEvent.Message)).FieldByName("Id"); f.IsValid() {
		messageID = f.String()
	}

	// Helper to register a local file with the media store
	storeMedia := func(localPath, filename, scope string) string {
		if store := c.GetMediaStore(); store != nil {
			ref, err := store.Store(localPath, media.MediaMeta{
				Filename: filename,
				Source:   "line",
			}, scope)
			if err == nil {
				return ref
			}
		}
		return localPath // fallback
	}

	switch msg := msgEvent.Message.(type) {
	case webhook.TextMessageContent:
		content = msg.Text
		isMentioned = c.isBotMentioned(msg)
		if msg.QuoteToken != "" {
			c.quoteTokens.Store(chatID, msg.QuoteToken)
		}
		if isGroup {
			content = c.stripBotMention(content, msg)
		}
	case webhook.ImageMessageContent:
		localPath := c.downloadContent(messageID, "image.jpg")
		if localPath != "" {
			scope := channels.BuildMediaScope("line", chatID, msg.Id)
			mediaPaths = append(mediaPaths, storeMedia(localPath, "image.jpg", scope))
			content = "[image]"
		}
	case webhook.AudioMessageContent:
		localPath := c.downloadContent(messageID, "audio.m4a")
		if localPath != "" {
			scope := channels.BuildMediaScope("line", chatID, msg.Id)
			mediaPaths = append(mediaPaths, storeMedia(localPath, "audio.m4a", scope))
			content = "[audio]"
		}
	case webhook.VideoMessageContent:
		localPath := c.downloadContent(messageID, "video.mp4")
		if localPath != "" {
			scope := channels.BuildMediaScope("line", chatID, msg.Id)
			mediaPaths = append(mediaPaths, storeMedia(localPath, "video.mp4", scope))
			content = "[video]"
		}
	case webhook.FileMessageContent:
		content = "[file]"
	case webhook.StickerMessageContent:
		content = "[sticker]"
	default:
		logger.WarnCF("line", "Ignoring unsupported message type", map[string]interface{}{
			"type": msgEvent.Message.GetType(),
		})
		return
	}

	if strings.TrimSpace(content) == "" {
		return
	}

	// In group chats, apply unified group trigger filtering
	if isGroup {
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			logger.DebugCF("line", "Ignoring group message by group trigger", map[string]any{
				"chat_id": chatID,
			})
			return
		}
		content = cleaned
	}

	metadata := map[string]string{
		"platform":    "line",
		"source_type": sourceType,
		"message_id":  messageID,
	}

	var peer bus.Peer
	if isGroup {
		peer = bus.Peer{Kind: "group", ID: chatID}
	} else {
		peer = bus.Peer{Kind: "direct", ID: senderID}
	}

	logger.DebugCF("line", "Received message", map[string]any{
		"sender_id":    senderID,
		"chat_id":      chatID,
		"message_type": msgEvent.Message.GetType(),
		"is_group":     isGroup,
		"preview":      utils.Truncate(content, 50),
	})

	sender := bus.SenderInfo{
		Platform:    "line",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("line", senderID),
	}

	if !c.IsAllowedSender(sender) {
		return
	}

	c.HandleMessage(c.ctx, peer, messageID, senderID, chatID, content, mediaPaths, metadata, sender)
}

// isBotMentioned checks if the bot is mentioned in the message.
// It first checks the mention metadata (userId match), then falls back
// to text-based detection using the bot's display name, since LINE may
// not include userId in mentionees for Official Accounts.
func (c *LINEChannel) isBotMentioned(msg webhook.TextMessageContent) bool {
	if msg.Mention != nil {
		for _, m := range msg.Mention.Mentionees {
			switch mentionee := m.(type) {
			case webhook.AllMentionee:
				return true
			case webhook.UserMentionee:
				if c.botUserID != "" && mentionee.UserId == c.botUserID {
					return true
				}
				// Check if mentionee text overlaps with bot display name
				if c.botDisplayName != "" && mentionee.Index >= 0 && mentionee.Length > 0 {
					runes := []rune(msg.Text)
					end := int(mentionee.Index) + int(mentionee.Length)
					if end <= len(runes) {
						mentionText := string(runes[mentionee.Index:end])
						if strings.Contains(mentionText, c.botDisplayName) {
							return true
						}
					}
				}
			}
		}
	}

	// Fallback: text-based detection with display name
	if c.botDisplayName != "" && strings.Contains(msg.Text, "@"+c.botDisplayName) {
		return true
	}

	return false
}

// stripBotMention removes the @BotName mention text from the message.
func (c *LINEChannel) stripBotMention(text string, msg webhook.TextMessageContent) string {
	stripped := false

	if msg.Mention != nil {
		runes := []rune(text)
		for i := len(msg.Mention.Mentionees) - 1; i >= 0; i-- {
			m := msg.Mention.Mentionees[i]
			shouldStrip := false
			var index, length int32

			switch mentionee := m.(type) {
			case webhook.UserMentionee:
				index = mentionee.Index
				length = mentionee.Length
				if c.botUserID != "" && mentionee.UserId == c.botUserID {
					shouldStrip = true
				} else if c.botDisplayName != "" && index >= 0 && length > 0 {
					end := int(index) + int(length)
					if end <= len(runes) {
						mentionText := string(runes[index:end])
						if strings.Contains(mentionText, c.botDisplayName) {
							shouldStrip = true
						}
					}
				}
			case webhook.AllMentionee:
				// Don't strip @All mentions
				continue
			default:
				continue
			}

			if shouldStrip {
				start := int(index)
				end := int(index) + int(length)
				if start >= 0 && end <= len(runes) {
					runes = append(runes[:start], runes[end:]...)
					stripped = true
				}
			}
		}
		if stripped {
			return strings.TrimSpace(string(runes))
		}
	}

	// Fallback: strip @DisplayName from text
	if c.botDisplayName != "" {
		text = strings.ReplaceAll(text, "@"+c.botDisplayName, "")
	}

	return strings.TrimSpace(text)
}

// resolveSource extracts senderID, chatID, and source type from the event source.
func (c *LINEChannel) resolveSource(source webhook.SourceInterface) (senderID, chatID, sourceType string) {
	switch src := source.(type) {
	case webhook.GroupSource:
		return src.UserId, src.GroupId, "group"
	case webhook.RoomSource:
		return src.UserId, src.RoomId, "room"
	case webhook.UserSource:
		return src.UserId, src.UserId, "user"
	default:
		return "", "", "unknown"
	}
}

// Send sends a message to LINE. It first tries the Reply API (free)
// using a cached reply token, then falls back to the Push API.
func (c *LINEChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	// Load and consume quote token for this chat
	var quoteToken string
	if qt, ok := c.quoteTokens.LoadAndDelete(msg.ChatID); ok {
		quoteToken = qt.(string)
	}

	textMsg := messaging_api.TextMessage{
		Text:       msg.Content,
		QuoteToken: quoteToken,
	}

	// Try reply token first (free, valid for ~25 seconds)
	if entry, ok := c.replyTokens.LoadAndDelete(msg.ChatID); ok {
		tokenEntry := entry.(replyTokenEntry)
		if time.Since(tokenEntry.timestamp) < lineReplyTokenMaxAge {
			_, err := c.client.ReplyMessage(&messaging_api.ReplyMessageRequest{
				ReplyToken: tokenEntry.token,
				Messages:   []messaging_api.MessageInterface{&textMsg},
			})
			if err == nil {
				logger.DebugCF("line", "Message sent via Reply API", map[string]interface{}{
					"chat_id": msg.ChatID,
					"quoted":  quoteToken != "",
				})
				return nil
			}
			logger.DebugC("line", "Reply API failed, falling back to Push API")
		}
	}

	// Fall back to Push API
	_, err := c.client.PushMessage(&messaging_api.PushMessageRequest{
		To:       msg.ChatID,
		Messages: []messaging_api.MessageInterface{&textMsg},
	}, "")
	return err
}

// SendMedia implements the channels.MediaSender interface.
// LINE requires media to be accessible via public URL; since we only have local files,
// we fall back to sending a text message with the filename/caption.
// For full support, an external file hosting service would be needed.
func (c *LINEChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	store := c.GetMediaStore()
	if store == nil {
		return fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	// LINE Messaging API requires publicly accessible URLs for media messages.
	// Since we only have local file paths, send caption text as fallback.
	for _, part := range msg.Parts {
		caption := part.Caption
		if caption == "" {
			caption = fmt.Sprintf("[%s: %s]", part.Type, part.Filename)
		}

		textMsg := messaging_api.TextMessage{Text: caption}
		if _, err := c.client.PushMessage(&messaging_api.PushMessageRequest{
			To:       msg.ChatID,
			Messages: []messaging_api.MessageInterface{&textMsg},
		}, ""); err != nil {
			return err
		}
	}

	return nil
}

// StartTyping implements channels.TypingCapable using LINE's loading animation.
//
// NOTE: The LINE loading animation API only works for 1:1 chats.
// Group/room chat IDs (starting with "C" or "R") are detected automatically;
// for these, a no-op stop function is returned without calling the API.
func (c *LINEChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	if chatID == "" {
		return func() {}, nil
	}

	// Group/room chats: LINE loading animation is 1:1 only.
	if strings.HasPrefix(chatID, "C") || strings.HasPrefix(chatID, "R") {
		return func() {}, nil
	}

	typingCtx, cancel := context.WithCancel(ctx)
	var once sync.Once
	stop := func() { once.Do(cancel) }

	// Send immediately, then refresh periodically for long-running tasks.
	if err := c.sendLoading(typingCtx, chatID); err != nil {
		stop()
		return stop, err
	}

	ticker := time.NewTicker(50 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				if err := c.sendLoading(typingCtx, chatID); err != nil {
					logger.DebugCF("line", "Failed to refresh loading indicator", map[string]any{
						"error": err.Error(),
					})
				}
			}
		}
	}()

	return stop, nil
}

// sendLoading sends a loading animation indicator to the chat.
func (c *LINEChannel) sendLoading(ctx context.Context, chatID string) error {
	_, err := c.client.ShowLoadingAnimation(&messaging_api.ShowLoadingAnimationRequest{
		ChatId:         chatID,
		LoadingSeconds: 60,
	})
	return err
}

// downloadContent downloads media content from the LINE content API.
func (c *LINEChannel) downloadContent(messageID, filename string) string {
	url := fmt.Sprintf(lineContentEndpoint, messageID)
	return utils.DownloadFile(url, filename, utils.DownloadOptions{
		LoggerPrefix: "line",
		ExtraHeaders: map[string]string{
			"Authorization": "Bearer " + c.config.ChannelAccessToken,
		},
	})
}
