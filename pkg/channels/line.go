package channels

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
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
	*BaseChannel
	config         config.LINEConfig
	client         *messaging_api.MessagingApiAPI
	httpServer     *http.Server
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

	base := NewBaseChannel("line", cfg, messageBus, cfg.AllowFrom)

	return &LINEChannel{
		BaseChannel: base,
		config:      cfg,
		client:      client,
	}, nil
}

// Start launches the HTTP webhook server.
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

	mux := http.NewServeMux()
	path := c.config.WebhookPath
	if path == "" {
		path = "/webhook/line"
	}
	mux.HandleFunc(path, c.webhookHandler)

	addr := fmt.Sprintf("%s:%d", c.config.WebhookHost, c.config.WebhookPort)
	c.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		logger.InfoCF("line", "LINE webhook server listening", map[string]interface{}{
			"addr": addr,
			"path": path,
		})
		if err := c.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("line", "Webhook server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	c.setRunning(true)
	logger.InfoC("line", "LINE channel started (Webhook Mode)")
	return nil
}

// Stop gracefully shuts down the HTTP server.
func (c *LINEChannel) Stop(ctx context.Context) error {
	logger.InfoC("line", "Stopping LINE channel")

	if c.cancel != nil {
		c.cancel()
	}

	if c.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := c.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.ErrorCF("line", "Webhook server shutdown error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	c.setRunning(false)
	logger.InfoC("line", "LINE channel stopped")
	return nil
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

	// In group chats, only respond when the bot is mentioned
	textMsg, isText := msgEvent.Message.(webhook.TextMessageContent)
	if isGroup {
		if !isText || !c.isBotMentioned(textMsg) {
			logger.DebugCF("line", "Ignoring group message without mention", map[string]interface{}{
				"chat_id": chatID,
			})
			return
		}
	}

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
	if f := reflect.Indirect(reflect.ValueOf(msgEvent.Message)).FieldByName("Id"); f.IsValid() {
		messageID = f.String()
	}
	localFiles := []string{}

	defer func() {
		for _, file := range localFiles {
			if err := os.Remove(file); err != nil {
				logger.DebugCF("line", "Failed to cleanup temp file", map[string]interface{}{
					"file":  file,
					"error": err.Error(),
				})
			}
		}
	}()

	switch msg := msgEvent.Message.(type) {
	case webhook.TextMessageContent:
		content = msg.Text
		if msg.QuoteToken != "" {
			c.quoteTokens.Store(chatID, msg.QuoteToken)
		}
		if isGroup {
			content = c.stripBotMention(content, msg)
		}
	case webhook.ImageMessageContent:
		localPath := c.downloadContent(messageID, "image.jpg")
		if localPath != "" {
			localFiles = append(localFiles, localPath)
			mediaPaths = append(mediaPaths, localPath)
			content = "[image]"
		}
	case webhook.AudioMessageContent:
		localPath := c.downloadContent(messageID, "audio.m4a")
		if localPath != "" {
			localFiles = append(localFiles, localPath)
			mediaPaths = append(mediaPaths, localPath)
			content = "[audio]"
		}
	case webhook.VideoMessageContent:
		localPath := c.downloadContent(messageID, "video.mp4")
		if localPath != "" {
			localFiles = append(localFiles, localPath)
			mediaPaths = append(mediaPaths, localPath)
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

	metadata := map[string]string{
		"platform":    "line",
		"source_type": sourceType,
		"message_id":  messageID,
	}

	if isGroup {
		metadata["peer_kind"] = "group"
		metadata["peer_id"] = chatID
	} else {
		metadata["peer_kind"] = "direct"
		metadata["peer_id"] = senderID
	}

	logger.DebugCF("line", "Received message", map[string]interface{}{
		"sender_id":    senderID,
		"chat_id":      chatID,
		"message_type": msgEvent.Message.GetType(),
		"is_group":     isGroup,
		"preview":      utils.Truncate(content, 50),
	})

	// Show typing/loading indicator (requires user ID, not group ID)
	c.sendLoading(senderID)

	c.HandleMessage(senderID, chatID, content, mediaPaths, metadata)
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
		return fmt.Errorf("line channel not running")
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

// sendLoading sends a loading animation indicator to the chat.
func (c *LINEChannel) sendLoading(chatID string) {
	_, err := c.client.ShowLoadingAnimation(&messaging_api.ShowLoadingAnimationRequest{
		ChatId:         chatID,
		LoadingSeconds: 60,
	})
	if err != nil {
		logger.DebugCF("line", "Failed to send loading indicator", map[string]interface{}{
			"error": err.Error(),
		})
	}
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
