// PicoClaw - Ultra-lightweight personal AI agent
// DingTalk channel implementation using Stream Mode

package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
	"github.com/sipeed/picoclaw/pkg/voice"
)

// DingTalkChannel implements the Channel interface for DingTalk (钉钉)
// It uses WebSocket for receiving messages via stream mode and API for sending
type DingTalkChannel struct {
	*channels.BaseChannel
	config       config.DingTalkConfig
	clientID     string
	clientSecret string
	streamClient *client.StreamClient
	ctx          context.Context
	cancel       context.CancelFunc
	transcriber  *voice.GroqTranscriber
	// Map to store session webhooks for each chat
	sessionWebhooks sync.Map // chatID -> sessionWebhook
}

// NewDingTalkChannel creates a new DingTalk channel instance
func NewDingTalkChannel(cfg config.DingTalkConfig, messageBus *bus.MessageBus) (*DingTalkChannel, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("dingtalk client_id and client_secret are required")
	}

	base := channels.NewBaseChannel("dingtalk", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(20000),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &DingTalkChannel{
		BaseChannel:  base,
		config:       cfg,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
	}, nil
}

// SetTranscriber sets the voice transcriber for voice message processing
func (c *DingTalkChannel) SetTranscriber(transcriber *voice.GroqTranscriber) {
	c.transcriber = transcriber
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
func (c *DingTalkChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	// Get session webhook from storage
	sessionWebhookRaw, ok := c.sessionWebhooks.Load(msg.ChatID)
	if !ok {
		return fmt.Errorf("no session_webhook found for chat %s, cannot send message", msg.ChatID)
	}

	sessionWebhook, ok := sessionWebhookRaw.(string)
	if !ok {
		return fmt.Errorf("invalid session_webhook type for chat %s", msg.ChatID)
	}

	logger.DebugCF("dingtalk", "Sending message", map[string]any{
		"chat_id": msg.ChatID,
		"preview": utils.Truncate(msg.Content, 100),
	})

	// Use the session webhook to send the reply
	return c.SendDirectReply(ctx, sessionWebhook, msg.Content)
}

// onChatBotMessageReceived implements the IChatBotMessageHandler function signature
// This is called by the Stream SDK when a new message arrives
// IChatBotMessageHandler is: func(c context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error)
func (c *DingTalkChannel) onChatBotMessageReceived(
	ctx context.Context,
	data *chatbot.BotCallbackDataModel,
) ([]byte, error) {
	// Extract message content
	content := data.Text.Content
	var localFiles []string

	// If content is empty, try to extract from Content interface{}
	if content == "" {
		if contentMap, ok := data.Content.(map[string]any); ok {
			// Check for different message types in the content map
			if textContent, ok := contentMap["content"].(string); ok && textContent != "" {
				content = textContent
			}
			// Handle image
			if imgURL, ok := contentMap["imageUrl"].(string); ok && imgURL != "" {
				imagePath := c.downloadImage(imgURL)
				if imagePath != "" {
					localFiles = append(localFiles, imagePath)
					if content != "" {
						content += "\n"
					}
					content += "[image]"
				}
			}
			// Handle file
			if fileURL, ok := contentMap["fileUrl"].(string); ok && fileURL != "" {
				filePath := c.downloadFile(fileURL)
				if filePath != "" {
					localFiles = append(localFiles, filePath)
					if content != "" {
						content += "\n"
					}
					content += "[file]"
				}
			}
			// Handle voice
			if voiceURL, ok := contentMap["mediaUrl"].(string); ok && voiceURL != "" {
				voicePath := c.downloadVoice(voiceURL)
				if voicePath != "" {
					localFiles = append(localFiles, voicePath)
					// Try to transcribe
					if c.transcriber != nil && c.transcriber.IsAvailable() {
						transcribeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
						defer cancel()
						result, err := c.transcriber.Transcribe(transcribeCtx, voicePath)
						if err != nil {
							content += "[voice transcription failed]"
						} else {
							content += fmt.Sprintf("[voice: %s]", result.Text)
						}
					} else {
						content += "[voice]"
					}
				}
			}
		}
	}

	if content == "" {
		return nil, nil // Ignore empty messages
	}

	senderID := data.SenderStaffId
	senderNick := data.SenderNick
	chatID := senderID
	if data.ConversationType != "1" {
		// For group chats
		chatID = data.ConversationId
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

	var peer bus.Peer
	if data.ConversationType == "1" {
		peer = bus.Peer{Kind: "direct", ID: senderID}
	} else {
		peer = bus.Peer{Kind: "group", ID: data.ConversationId}
		// In group chats, apply unified group trigger filtering
		respond, cleaned := c.ShouldRespondInGroup(false, content)
		if !respond {
			return nil, nil
		}
		content = cleaned
	}

	logger.DebugCF("dingtalk", "Received message", map[string]any{
		"sender_nick": senderNick,
		"sender_id":   senderID,
		"preview":     utils.Truncate(content, 50),
		"files":       len(localFiles),
	})

	// Build sender info
	sender := bus.SenderInfo{
		Platform:    "dingtalk",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("dingtalk", senderID),
		DisplayName: senderNick,
	}

	if !c.IsAllowedSender(sender) {
		return nil, nil
	}

	// Handle the message through the base channel
	c.HandleMessage(ctx, peer, "", senderID, chatID, content, localFiles, metadata, sender)

	// Return nil to indicate we've handled the message asynchronously
	// The response will be sent through the message bus
	return nil, nil
}

// downloadImage downloads an image from DingTalk
func (c *DingTalkChannel) downloadImage(url string) string {
	if url == "" {
		return ""
	}

	// Get access token first
	token, err := c.getAccessToken()
	if err != nil {
		logger.ErrorCF("dingtalk", "Failed to get access token", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	// Construct download URL
	downloadURL := fmt.Sprintf("https://oapi.dingtalk.com/media/download?access_token=%s&media_id=%s", token, url)

	filename := fmt.Sprintf("dingtalk_image_%d.jpg", time.Now().Unix())
	return utils.DownloadFile(downloadURL, filename, utils.DownloadOptions{
		LoggerPrefix: "dingtalk",
	})
}

// downloadFile downloads a file from DingTalk
func (c *DingTalkChannel) downloadFile(url string) string {
	if url == "" {
		return ""
	}

	// Get access token first
	token, err := c.getAccessToken()
	if err != nil {
		logger.ErrorCF("dingtalk", "Failed to get access token", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	// Construct download URL
	downloadURL := fmt.Sprintf("https://oapi.dingtalk.com/media/download?access_token=%s&media_id=%s", token, url)

	filename := fmt.Sprintf("dingtalk_file_%d", time.Now().Unix())
	return utils.DownloadFile(downloadURL, filename, utils.DownloadOptions{
		LoggerPrefix: "dingtalk",
	})
}

// downloadVoice downloads a voice file from DingTalk
func (c *DingTalkChannel) downloadVoice(url string) string {
	if url == "" {
		return ""
	}

	// Get access token first
	token, err := c.getAccessToken()
	if err != nil {
		logger.ErrorCF("dingtalk", "Failed to get access token", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	// Construct download URL
	downloadURL := fmt.Sprintf("https://oapi.dingtalk.com/media/download?access_token=%s&media_id=%s", token, url)

	filename := fmt.Sprintf("dingtalk_voice_%d.amr", time.Now().Unix())
	return utils.DownloadFile(downloadURL, filename, utils.DownloadOptions{
		LoggerPrefix: "dingtalk",
	})
}

// getAccessToken gets the DingTalk access token
func (c *DingTalkChannel) getAccessToken() (string, error) {
	url := fmt.Sprintf("https://oapi.dingtalk.com/gettoken?appkey=%s&appsecret=%s", c.clientID, c.clientSecret)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("dingtalk API error: %s", result.ErrMsg)
	}

	return result.AccessToken, nil
}

// SendMedia implements channels.MediaSender for sending media messages
func (c *DingTalkChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	store := c.GetMediaStore()
	if store == nil {
		return fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	// Get session webhook from storage
	sessionWebhookRaw, ok := c.sessionWebhooks.Load(msg.ChatID)
	if !ok {
		return fmt.Errorf("no session_webhook found for chat %s, cannot send media", msg.ChatID)
	}

	sessionWebhook, ok := sessionWebhookRaw.(string)
	if !ok {
		return fmt.Errorf("invalid session_webhook type for chat %s", msg.ChatID)
	}

	for _, part := range msg.Parts {
		localPath, err := store.Resolve(part.Ref)
		if err != nil {
			logger.ErrorCF("dingtalk", "Failed to resolve media ref", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			continue
		}

		// Upload media and send
		mediaID, err := c.uploadMedia(ctx, localPath, part.Type)
		if err != nil {
			logger.ErrorCF("dingtalk", "Failed to upload media", map[string]any{
				"type":  part.Type,
				"error": err.Error(),
			})
			continue
		}

		// Send media info via session webhook using direct HTTP call
		caption := part.Caption
		if caption == "" {
			caption = part.Filename
		}

		var content string
		switch part.Type {
		case "image":
			// Send image URL as markdown
			content = fmt.Sprintf("![image](%s)", mediaID)
			if caption != "" {
				content = caption + "\n" + content
			}
		default:
			content = fmt.Sprintf("[%s: %s](%s)", part.Type, part.Filename, mediaID)
			if caption != "" {
				content = caption + "\n" + content
			}
		}

		if err := c.sendTextToWebhook(ctx, sessionWebhook, content); err != nil {
			logger.ErrorCF("dingtalk", "Failed to send media message", map[string]any{
				"error": err.Error(),
			})
		}
	}

	return nil
}

// sendTextToWebhook sends text to a session webhook
func (c *DingTalkChannel) sendTextToWebhook(ctx context.Context, webhook, content string) error {
	payload := map[string]any{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook status: %d", resp.StatusCode)
	}
	return nil
}

// uploadMedia uploads media to DingTalk and returns the media ID
func (c *DingTalkChannel) uploadMedia(ctx context.Context, filePath, mediaType string) (string, error) {
	token, err := c.getAccessToken()
	if err != nil {
		return "", fmt.Errorf("get access token: %w", err)
	}

	// Determine media type for DingTalk API
	dingtalkMediaType := "file"
	switch mediaType {
	case "image":
		dingtalkMediaType = "image"
	case "voice":
		dingtalkMediaType = "voice"
	case "video":
		dingtalkMediaType = "video"
	}

	// Upload to DingTalk
	uploadURL := fmt.Sprintf("https://oapi.dingtalk.com/media/upload?access_token=%s&type=%s", token, dingtalkMediaType)

	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Create multipart form request
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("copy file: %w", err)
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload status: %d", resp.StatusCode)
	}

	var uploadResp struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		MediaID string `json:"media_id"`
		Type    string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if uploadResp.ErrCode != 0 {
		return "", fmt.Errorf("upload error: %s", uploadResp.ErrMsg)
	}

	return uploadResp.MediaID, nil
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
