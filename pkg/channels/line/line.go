package line

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	lineAPIBase          = "https://api.line.me/v2/bot"
	lineDataAPIBase      = "https://api-data.line.me/v2/bot"
	lineReplyEndpoint    = lineAPIBase + "/message/reply"
	linePushEndpoint     = lineAPIBase + "/message/push"
	lineContentEndpoint  = lineDataAPIBase + "/message/%s/content"
	lineBotInfoEndpoint  = lineAPIBase + "/info"
	lineLoadingEndpoint  = lineAPIBase + "/chat/loading/start"
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

	base := channels.NewBaseChannel("line", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(5000),
		channels.WithGroupTrigger(cfg.GroupTrigger),
	)

	return &LINEChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

// Start initializes the LINE channel.
func (c *LINEChannel) Start(ctx context.Context) error {
	logger.InfoC("line", "Starting LINE channel (Webhook Mode)")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Fetch bot profile to get bot's userId for mention detection
	if err := c.fetchBotInfo(); err != nil {
		logger.WarnCF("line", "Failed to fetch bot info (mention detection disabled)", map[string]any{
			"error": err.Error(),
		})
	} else {
		logger.InfoCF("line", "Bot info fetched", map[string]any{
			"bot_user_id":  c.botUserID,
			"basic_id":     c.botBasicID,
			"display_name": c.botDisplayName,
		})
	}

	c.SetRunning(true)
	logger.InfoC("line", "LINE channel started (Webhook Mode)")
	return nil
}

// fetchBotInfo retrieves the bot's userId, basicId, and displayName from the LINE API.
func (c *LINEChannel) fetchBotInfo() error {
	req, err := http.NewRequest(http.MethodGet, lineBotInfoEndpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.ChannelAccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bot info API returned status %d", resp.StatusCode)
	}

	var info struct {
		UserID      string `json:"userId"`
		BasicID     string `json:"basicId"`
		DisplayName string `json:"displayName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return err
	}

	c.botUserID = info.UserID
	c.botBasicID = info.BasicID
	c.botDisplayName = info.DisplayName
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.ErrorCF("line", "Failed to read request body", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("X-Line-Signature")
	if !c.verifySignature(body, signature) {
		logger.WarnC("line", "Invalid webhook signature")
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var payload struct {
		Events []lineEvent `json:"events"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		logger.ErrorCF("line", "Failed to parse webhook payload", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Return 200 immediately, process events asynchronously
	w.WriteHeader(http.StatusOK)

	for _, event := range payload.Events {
		go c.processEvent(event)
	}
}

// verifySignature validates the X-Line-Signature using HMAC-SHA256.
func (c *LINEChannel) verifySignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(c.config.ChannelSecret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

// LINE webhook event types
type lineEvent struct {
	Type       string          `json:"type"`
	ReplyToken string          `json:"replyToken"`
	Source     lineSource      `json:"source"`
	Message    json.RawMessage `json:"message"`
	Timestamp  int64           `json:"timestamp"`
}

type lineSource struct {
	Type    string `json:"type"` // "user", "group", "room"
	UserID  string `json:"userId"`
	GroupID string `json:"groupId"`
	RoomID  string `json:"roomId"`
}

type lineMessage struct {
	ID         string `json:"id"`
	Type       string `json:"type"` // "text", "image", "video", "audio", "file", "sticker"
	Text       string `json:"text"`
	QuoteToken string `json:"quoteToken"`
	Mention    *struct {
		Mentionees []lineMentionee `json:"mentionees"`
	} `json:"mention"`
	ContentProvider struct {
		Type string `json:"type"`
	} `json:"contentProvider"`
}

type lineMentionee struct {
	Index  int    `json:"index"`
	Length int    `json:"length"`
	Type   string `json:"type"` // "user", "all"
	UserID string `json:"userId"`
}

func (c *LINEChannel) processEvent(event lineEvent) {
	if event.Type != "message" {
		logger.DebugCF("line", "Ignoring non-message event", map[string]any{
			"type": event.Type,
		})
		return
	}

	senderID := event.Source.UserID
	chatID := c.resolveChatID(event.Source)
	isGroup := event.Source.Type == "group" || event.Source.Type == "room"

	var msg lineMessage
	if err := json.Unmarshal(event.Message, &msg); err != nil {
		logger.ErrorCF("line", "Failed to parse message", map[string]any{
			"error": err.Error(),
		})
		return
	}

	// Store reply token for later use
	if event.ReplyToken != "" {
		c.replyTokens.Store(chatID, replyTokenEntry{
			token:     event.ReplyToken,
			timestamp: time.Now(),
		})
	}

	// Store quote token for quoting the original message in reply
	if msg.QuoteToken != "" {
		c.quoteTokens.Store(chatID, msg.QuoteToken)
	}

	var content string
	var mediaPaths []string

	scope := channels.BuildMediaScope("line", chatID, msg.ID)

	// Helper to register a local file with the media store
	storeMedia := func(localPath, filename string) string {
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

	switch msg.Type {
	case "text":
		content = msg.Text
		// Strip bot mention from text in group chats
		if isGroup {
			content = c.stripBotMention(content, msg)
		}
	case "image":
		localPath := c.downloadContent(msg.ID, "image.jpg")
		if localPath != "" {
			mediaPaths = append(mediaPaths, storeMedia(localPath, "image.jpg"))
			content = "[image]"
		}
	case "audio":
		localPath := c.downloadContent(msg.ID, "audio.m4a")
		if localPath != "" {
			mediaPaths = append(mediaPaths, storeMedia(localPath, "audio.m4a"))
			content = "[audio]"
		}
	case "video":
		localPath := c.downloadContent(msg.ID, "video.mp4")
		if localPath != "" {
			mediaPaths = append(mediaPaths, storeMedia(localPath, "video.mp4"))
			content = "[video]"
		}
	case "file":
		content = "[file]"
	case "sticker":
		content = "[sticker]"
	default:
		content = fmt.Sprintf("[%s]", msg.Type)
	}

	if strings.TrimSpace(content) == "" {
		return
	}

	// In group chats, apply unified group trigger filtering
	if isGroup {
		isMentioned := c.isBotMentioned(msg)
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
		"source_type": event.Source.Type,
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
		"message_type": msg.Type,
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

	c.HandleMessage(c.ctx, peer, msg.ID, senderID, chatID, content, mediaPaths, metadata, sender)
}

// isBotMentioned checks if the bot is mentioned in the message.
// It first checks the mention metadata (userId match), then falls back
// to text-based detection using the bot's display name, since LINE may
// not include userId in mentionees for Official Accounts.
func (c *LINEChannel) isBotMentioned(msg lineMessage) bool {
	// Check mention metadata
	if msg.Mention != nil {
		for _, m := range msg.Mention.Mentionees {
			if m.Type == "all" {
				return true
			}
			if c.botUserID != "" && m.UserID == c.botUserID {
				return true
			}
		}
		// Mention metadata exists with mentionees but bot not matched by userId.
		// The bot IS likely mentioned (LINE includes mention struct when bot is @-ed),
		// so check if any mentionee overlaps with bot display name in text.
		if c.botDisplayName != "" {
			for _, m := range msg.Mention.Mentionees {
				if m.Index >= 0 && m.Length > 0 {
					runes := []rune(msg.Text)
					end := m.Index + m.Length
					if end <= len(runes) {
						mentionText := string(runes[m.Index:end])
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
func (c *LINEChannel) stripBotMention(text string, msg lineMessage) string {
	stripped := false

	// Try to strip using mention metadata indices
	if msg.Mention != nil {
		runes := []rune(text)
		for i := len(msg.Mention.Mentionees) - 1; i >= 0; i-- {
			m := msg.Mention.Mentionees[i]
			// Strip if userId matches OR if the mention text contains the bot display name
			shouldStrip := false
			if c.botUserID != "" && m.UserID == c.botUserID {
				shouldStrip = true
			} else if c.botDisplayName != "" && m.Index >= 0 && m.Length > 0 {
				end := m.Index + m.Length
				if end <= len(runes) {
					mentionText := string(runes[m.Index:end])
					if strings.Contains(mentionText, c.botDisplayName) {
						shouldStrip = true
					}
				}
			}
			if shouldStrip {
				start := m.Index
				end := m.Index + m.Length
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

// resolveChatID determines the chat ID from the event source.
// For group/room messages, use the group/room ID; for 1:1, use the user ID.
func (c *LINEChannel) resolveChatID(source lineSource) string {
	switch source.Type {
	case "group":
		return source.GroupID
	case "room":
		return source.RoomID
	default:
		return source.UserID
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

	// Try reply token first (free, valid for ~25 seconds)
	if entry, ok := c.replyTokens.LoadAndDelete(msg.ChatID); ok {
		tokenEntry := entry.(replyTokenEntry)
		if time.Since(tokenEntry.timestamp) < lineReplyTokenMaxAge {
			if err := c.sendReply(ctx, tokenEntry.token, msg.Content, quoteToken); err == nil {
				logger.DebugCF("line", "Message sent via Reply API", map[string]any{
					"chat_id": msg.ChatID,
					"quoted":  quoteToken != "",
				})
				return nil
			}
			logger.DebugC("line", "Reply API failed, falling back to Push API")
		}
	}

	// Fall back to Push API
	return c.sendPush(ctx, msg.ChatID, msg.Content, quoteToken)
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

		if err := c.sendPush(ctx, msg.ChatID, caption, ""); err != nil {
			return err
		}
	}

	return nil
}

// buildTextMessage creates a text message object, optionally with quoteToken.
func buildTextMessage(content, quoteToken string) map[string]string {
	msg := map[string]string{
		"type": "text",
		"text": content,
	}
	if quoteToken != "" {
		msg["quoteToken"] = quoteToken
	}
	return msg
}

// sendReply sends a message using the LINE Reply API.
func (c *LINEChannel) sendReply(ctx context.Context, replyToken, content, quoteToken string) error {
	payload := map[string]any{
		"replyToken": replyToken,
		"messages":   []map[string]string{buildTextMessage(content, quoteToken)},
	}

	return c.callAPI(ctx, lineReplyEndpoint, payload)
}

// sendPush sends a message using the LINE Push API.
func (c *LINEChannel) sendPush(ctx context.Context, to, content, quoteToken string) error {
	payload := map[string]any{
		"to":       to,
		"messages": []map[string]string{buildTextMessage(content, quoteToken)},
	}

	return c.callAPI(ctx, linePushEndpoint, payload)
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
	payload := map[string]any{
		"chatId":         chatID,
		"loadingSeconds": 60,
	}
	return c.callAPI(ctx, lineLoadingEndpoint, payload)
}

// callAPI makes an authenticated POST request to the LINE API.
func (c *LINEChannel) callAPI(ctx context.Context, endpoint string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.ChannelAccessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return channels.ClassifyNetError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return channels.ClassifySendError(resp.StatusCode, fmt.Errorf("LINE API error: %s", string(respBody)))
	}

	return nil
}

// downloadContent downloads media content from the LINE API.
func (c *LINEChannel) downloadContent(messageID, filename string) string {
	url := fmt.Sprintf(lineContentEndpoint, messageID)
	return utils.DownloadFile(url, filename, utils.DownloadOptions{
		LoggerPrefix: "line",
		ExtraHeaders: map[string]string{
			"Authorization": "Bearer " + c.config.ChannelAccessToken,
		},
	})
}
