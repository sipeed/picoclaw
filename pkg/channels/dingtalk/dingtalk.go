// PicoClaw - Ultra-lightweight personal AI agent
// DingTalk channel implementation using Stream Mode

package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	dinglog "github.com/open-dingtalk/dingtalk-stream-sdk-go/logger"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// DingTalkChannel implements the Channel interface for DingTalk (钉钉)
// It uses WebSocket for receiving messages via stream mode and API for sending
type DingTalkChannel struct {
	*channels.BaseChannel
	config       *config.DingTalkSettings
	clientID     string
	clientSecret string
	streamClient *client.StreamClient
	ctx          context.Context
	cancel       context.CancelFunc
	tokenMu      sync.Mutex
	accessToken  string
	tokenExpires time.Time
	// Map to store session webhooks for each chat
	sessionWebhooks sync.Map // chatID -> sessionWebhook
}

// NewDingTalkChannel creates a new DingTalk channel instance
func NewDingTalkChannel(
	bc *config.Channel,
	cfg *config.DingTalkSettings,
	messageBus *bus.MessageBus,
) (*DingTalkChannel, error) {
	if cfg.ClientID == "" || cfg.ClientSecret.String() == "" {
		return nil, fmt.Errorf("dingtalk client_id and client_secret are required")
	}

	// Set the logger for the Stream SDK
	dinglog.SetLogger(logger.NewLogger("dingtalk"))

	base := channels.NewBaseChannel("dingtalk", cfg, messageBus, bc.AllowFrom,
		channels.WithMaxMessageLength(20000),
		channels.WithGroupTrigger(bc.GroupTrigger),
		channels.WithReasoningChannelID(bc.ReasoningChannelID),
	)

	return &DingTalkChannel{
		BaseChannel:  base,
		config:       cfg,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret.String(),
	}, nil
}

// Start initializes the DingTalk channel with Stream Mode
func (c *DingTalkChannel) Start(ctx context.Context) error {
	logger.InfoC("dingtalk", "Starting DingTalk channel (Stream Mode)...")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Create credential config
	cred := client.NewAppCredentialConfig(c.clientID, c.clientSecret)

	// Create the stream client with options
	c.streamClient = client.NewStreamClient(
		client.WithAppCredential(cred),
		client.WithAutoReconnect(true),
	)

	// Register chatbot callback handler (IChatBotMessageHandler is a function type)
	c.streamClient.RegisterChatBotCallbackRouter(c.onChatBotMessageReceived)

	// Start the stream client
	if err := c.streamClient.Start(c.ctx); err != nil {
		return fmt.Errorf("failed to start stream client: %w", err)
	}

	c.SetRunning(true)
	logger.InfoC("dingtalk", "DingTalk channel started (Stream Mode)")
	return nil
}

// Stop gracefully stops the DingTalk channel
func (c *DingTalkChannel) Stop(ctx context.Context) error {
	logger.InfoC("dingtalk", "Stopping DingTalk channel...")

	if c.cancel != nil {
		c.cancel()
	}

	if c.streamClient != nil {
		c.streamClient.Close()
	}

	c.SetRunning(false)
	logger.InfoC("dingtalk", "DingTalk channel stopped")
	return nil
}

// Send sends a message to DingTalk via the chatbot reply API
func (c *DingTalkChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	// Get session webhook from storage
	sessionWebhookRaw, ok := c.sessionWebhooks.Load(msg.ChatID)
	if !ok {
		return nil, fmt.Errorf("no session_webhook found for chat %s, cannot send message", msg.ChatID)
	}

	sessionWebhook, ok := sessionWebhookRaw.(string)
	if !ok {
		return nil, fmt.Errorf("invalid session_webhook type for chat %s", msg.ChatID)
	}

	logger.DebugCF("dingtalk", "Sending message", map[string]any{
		"chat_id": msg.ChatID,
		"preview": utils.Truncate(msg.Content, 100),
	})

	// Use the session webhook to send the reply
	return nil, c.SendDirectReply(ctx, sessionWebhook, msg.Content)
}

// onChatBotMessageReceived implements the IChatBotMessageHandler function signature
// This is called by the Stream SDK when a new message arrives
// IChatBotMessageHandler is: func(c context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error)
func (c *DingTalkChannel) onChatBotMessageReceived(
	ctx context.Context,
	data *chatbot.BotCallbackDataModel,
) ([]byte, error) {
	if data == nil {
		return nil, nil
	}

	// Extract message content from Text field
	content := strings.TrimSpace(data.Text.Content)
	if content == "" {
		// Try to extract from Content interface{} if Text is empty
		if contentMap, ok := data.Content.(map[string]any); ok {
			if textContent, ok := contentMap["content"].(string); ok {
				content = strings.TrimSpace(textContent)
			}
		}
	}

	// Check for image/picture messages before returning on empty content
	var mediaRefs []string
	if data.Content != nil {
		if contentMap, ok := data.Content.(map[string]any); ok {
			msgtype := stringValue(contentMap["msgtype"])
			if msgtype == "picture" || msgtype == "image" {
				refs, err := c.downloadInboundPicture(ctx, data)
				if err != nil {
					logger.ErrorCF("dingtalk", "Failed to download inbound picture", map[string]any{
						"error": err.Error(),
					})
				}
				if refs != nil {
					mediaRefs = refs
					content += " [image: photo]"
				}
			}
		}
	}

	if content == "" {
		return nil, nil // Ignore messages with neither text nor media
	}

	senderID := strings.TrimSpace(data.SenderStaffId)
	if senderID == "" {
		senderID = strings.TrimSpace(data.SenderId)
	}
	senderNick := strings.TrimSpace(data.SenderNick)

	chatID := strings.TrimSpace(data.ConversationId)
	if chatID == "" && data.ConversationType == "1" {
		// Fallback for direct chats when conversation_id is absent.
		chatID = senderID
	}
	if chatID == "" {
		return nil, nil
	}

	// Store the session webhook for this chat so we can reply later
	c.sessionWebhooks.Store(chatID, data.SessionWebhook)

	metadata := map[string]string{
		"sender_name":       senderNick,
		"conversation_id":   data.ConversationId,
		"conversation_type": data.ConversationType,
		"platform":          "dingtalk",
		"session_webhook":   data.SessionWebhook,
	}

	var (
		chatType    string
		isMentioned bool
	)
	if data.ConversationType == "1" {
		chatType = "direct"
	} else {
		chatType = "group"
		isMentioned = data.IsInAtList
		if isMentioned {
			content = stripLeadingAtMentions(content)
		}
		// In group chats, apply unified group trigger filtering
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			return nil, nil
		}
		content = cleaned
	}

	logger.DebugCF("dingtalk", "Received message", map[string]any{
		"sender_nick": senderNick,
		"sender_id":   senderID,
		"preview":     utils.Truncate(content, 50),
	})

	// Build sender info
	platformID := senderID
	if platformID == "" {
		platformID = chatID
	}
	resolvedSenderID := senderID
	if resolvedSenderID == "" {
		resolvedSenderID = platformID
	}
	sender := bus.SenderInfo{
		Platform:    "dingtalk",
		PlatformID:  platformID,
		CanonicalID: identity.BuildCanonicalID("dingtalk", platformID),
		DisplayName: senderNick,
	}

	if !c.IsAllowedSender(sender) {
		return nil, nil
	}

	inboundCtx := bus.InboundContext{
		Channel:   "dingtalk",
		ChatID:    chatID,
		ChatType:  chatType,
		SenderID:  resolvedSenderID,
		Mentioned: isMentioned,
		Raw:       metadata,
	}
	if data.SessionWebhook != "" {
		inboundCtx.ReplyHandles = map[string]string{
			"session_webhook": data.SessionWebhook,
		}
	}

	c.HandleInboundContext(ctx, chatID, content, mediaRefs, inboundCtx, sender)

	// Return nil to indicate we've handled the message asynchronously
	// The response will be sent through the message bus
	return nil, nil
}

// getAccessToken obtains and caches a DingTalk OpenAPI access token.
// Tokens are cached for their lifetime minus a 2-minute safety buffer.
func (c *DingTalkChannel) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Return cached token if still valid (with 5-minute buffer)
	if c.accessToken != "" && time.Now().Before(c.tokenExpires) {
		return c.accessToken, nil
	}

	// Request new token from DingTalk OpenAPI
	body := map[string]string{
		"appKey":    c.clientID,
		"appSecret": c.clientSecret,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.dingtalk.com/v1.0/oauth2/accessToken",
		bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken string `json:"accessToken"`
		ExpireIn    int64  `json:"expireIn"` // seconds
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}

	// Cache with 120s safety buffer
	expireDuration := time.Duration(result.ExpireIn) * time.Second
	c.accessToken = result.AccessToken
	c.tokenExpires = time.Now().Add(expireDuration - 120*time.Second)

	logger.DebugC("dingtalk", "Obtained new access token")
	return c.accessToken, nil
}

// downloadInboundPicture downloads a picture message from DingTalk,
// saves it to the media temp directory, registers it in the MediaStore,
// and returns the media ref string. Failures degrade gracefully.
func (c *DingTalkChannel) downloadInboundPicture(
	ctx context.Context,
	data *chatbot.BotCallbackDataModel,
) ([]string, error) {
	contentMap, ok := data.Content.(map[string]any)
	if !ok {
		return nil, nil
	}

	downloadCode := stringValue(contentMap["downloadCode"])
	robotCode := stringValue(contentMap["robotCode"])
	if downloadCode == "" || robotCode == "" {
		logger.WarnCF("dingtalk", "Inbound picture missing downloadCode or robotCode", map[string]any{
			"keys": keysOf(contentMap),
		})
		return nil, nil
	}

	// Get access token
	token, err := c.getAccessToken(ctx)
	if err != nil {
		logger.ErrorCF("dingtalk", "Failed to get access token for picture download", map[string]any{
			"error": err.Error(),
		})
		return nil, nil
	}

	// Call DingTalk API to get download URL
	reqBody := map[string]string{
		"downloadCode": downloadCode,
		"robotCode":    robotCode,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		logger.ErrorCF("dingtalk", "Failed to marshal download request", map[string]any{
			"error": err.Error(),
		})
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.dingtalk.com/v1.0/robot/messageFiles/download",
		bytes.NewReader(reqBytes))
	if err != nil {
		logger.ErrorCF("dingtalk", "Failed to create download request", map[string]any{
			"error": err.Error(),
		})
		return nil, nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-acs-dingtalk-access-token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.ErrorCF("dingtalk", "Download URL request failed", map[string]any{
			"error": err.Error(),
		})
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.ErrorCF("dingtalk", "Download URL request returned non-OK", map[string]any{
			"status": resp.StatusCode,
			"body":   string(respBody),
		})
		return nil, nil
	}

	var downloadResult struct {
		DownloadURL string `json:"downloadUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&downloadResult); err != nil {
		logger.ErrorCF("dingtalk", "Failed to decode download URL response", map[string]any{
			"error": err.Error(),
		})
		return nil, nil
	}

	if downloadResult.DownloadURL == "" {
		logger.WarnC("dingtalk", "Empty download URL in response")
		return nil, nil
	}

	// Download the actual file content
	fileResp, err := http.Get(downloadResult.DownloadURL)
	if err != nil {
		logger.ErrorCF("dingtalk", "Failed to download picture file", map[string]any{
			"error": err.Error(),
		})
		return nil, nil
	}
	defer fileResp.Body.Close()

	fileBytes, err := io.ReadAll(fileResp.Body)
	if err != nil {
		logger.ErrorCF("dingtalk", "Failed to read picture file", map[string]any{
			"error": err.Error(),
		})
		return nil, nil
	}

	// Determine file extension from Content-Type
	contentType := fileResp.Header.Get("Content-Type")
	ext := ".jpg"
	if strings.Contains(contentType, "png") {
		ext = ".png"
	} else if strings.Contains(contentType, "gif") {
		ext = ".gif"
	} else if strings.Contains(contentType, "webp") {
		ext = ".webp"
	} else if strings.Contains(contentType, "bmp") {
		ext = ".bmp"
	}

	// Write to temp file
	mediaDir := media.TempDir()
	if err := os.MkdirAll(mediaDir, 0o700); err != nil {
		logger.ErrorCF("dingtalk", "Failed to create media directory", map[string]any{
			"error": err.Error(),
		})
		return nil, nil
	}

	filename := fmt.Sprintf("dingtalk_%s%s", downloadCode, ext)
	localPath := filepath.Join(mediaDir, utils.SanitizeFilename(filename))
	if err := os.WriteFile(localPath, fileBytes, 0o644); err != nil {
		logger.ErrorCF("dingtalk", "Failed to write picture file", map[string]any{
			"error": err.Error(),
		})
		return nil, nil
	}

	// Register in MediaStore
	store := c.GetMediaStore()
	if store == nil {
		logger.WarnC("dingtalk", "No MediaStore available, skipping media registration")
		return nil, nil
	}

	scope := channels.BuildMediaScope("dingtalk", "", downloadCode)
	ref, err := store.Store(localPath, media.MediaMeta{
		Filename:      filename,
		ContentType:   contentType,
		Source:        "dingtalk",
		CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
	}, scope)
	if err != nil {
		logger.ErrorCF("dingtalk", "Failed to store picture in MediaStore", map[string]any{
			"error": err.Error(),
		})
		os.Remove(localPath)
		return nil, nil
	}

	logger.DebugCF("dingtalk", "Downloaded and stored inbound picture", map[string]any{
		"ref": ref,
	})
	return []string{ref}, nil
}

// SendDirectReply sends a direct reply using the session webhook
func (c *DingTalkChannel) SendDirectReply(ctx context.Context, sessionWebhook, content string) error {
	replier := chatbot.NewChatbotReplier()

	// Convert string content to []byte for the API
	contentBytes := []byte(content)
	titleBytes := []byte("PicoClaw")

	// Send markdown formatted reply
	err := replier.SimpleReplyMarkdown(
		ctx,
		sessionWebhook,
		titleBytes,
		contentBytes,
	)
	if err != nil {
		return fmt.Errorf("dingtalk send: %w", channels.ErrTemporary)
	}

	return nil
}

// stripLeadingAtMentions removes leading @mentions from the message content
func stripLeadingAtMentions(content string) string {
	fields := strings.Fields(content)
	if len(fields) == 0 {
		return ""
	}

	i := 0
	for i < len(fields) && strings.HasPrefix(fields[i], "@") {
		i++
	}
	if i == 0 {
		return strings.TrimSpace(content)
	}
	return strings.Join(fields[i:], " ")
}

// keysOf returns all keys from a map for debugging/logging purposes.
func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// stringValue safely extracts a string from an interface{} value.
func stringValue(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
