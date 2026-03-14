package wecom

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/h2non/filetype"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// WeComBotChannel implements the Channel interface for WeCom Bot (企业微信智能机器人)
// Uses webhook callback mode - simpler than WeCom App but only supports passive replies
type WeComBotChannel struct {
	*channels.BaseChannel
	config        config.WeComConfig
	client        *http.Client
	ctx           context.Context
	cancel        context.CancelFunc
	processedMsgs *MessageDeduplicator
}

// WeComBotMessage represents the JSON message structure from WeCom Bot (AIBOT)
type WeComBotMessage struct {
	MsgID    string `json:"msgid"`
	AIBotID  string `json:"aibotid"`
	ChatID   string `json:"chatid"`   // Session ID, only present for group chats
	ChatType string `json:"chattype"` // "single" for DM, "group" for group chat
	From     struct {
		UserID string `json:"userid"`
	} `json:"from"`
	ResponseURL string `json:"response_url"`
	MsgType     string `json:"msgtype"` // text, image, voice, file, mixed
	Text        struct {
		Content string `json:"content"`
	} `json:"text"`
	Image struct {
		URL string `json:"url"`
	} `json:"image"`
	Voice struct {
		Content string `json:"content"` // Voice to text content
	} `json:"voice"`
	File struct {
		URL string `json:"url"`
	} `json:"file"`
	Mixed struct {
		MsgItem []struct {
			MsgType string `json:"msgtype"`
			Text    struct {
				Content string `json:"content"`
			} `json:"text"`
			Image struct {
				URL string `json:"url"`
			} `json:"image"`
		} `json:"msg_item"`
	} `json:"mixed"`
	Quote struct {
		MsgType string `json:"msgtype"`
		Text    struct {
			Content string `json:"content"`
		} `json:"text"`
	} `json:"quote"`
}

// WeComBotReplyMessage represents the reply message structure
type WeComBotReplyMessage struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text,omitempty"`
}

// NewWeComBotChannel creates a new WeCom Bot channel instance
func NewWeComBotChannel(cfg config.WeComConfig, messageBus *bus.MessageBus) (*WeComBotChannel, error) {
	if cfg.Token == "" || cfg.WebhookURL == "" {
		return nil, fmt.Errorf("wecom token and webhook_url are required")
	}

	base := channels.NewBaseChannel("wecom", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(2048),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	// Client timeout must be >= the configured ReplyTimeout so the
	// per-request context deadline is always the effective limit.
	clientTimeout := 30 * time.Second
	if d := time.Duration(cfg.ReplyTimeout) * time.Second; d > clientTimeout {
		clientTimeout = d
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &WeComBotChannel{
		BaseChannel:   base,
		config:        cfg,
		client:        &http.Client{Timeout: clientTimeout},
		ctx:           ctx,
		cancel:        cancel,
		processedMsgs: NewMessageDeduplicator(wecomMaxProcessedMessages),
	}, nil
}

// Name returns the channel name
func (c *WeComBotChannel) Name() string {
	return "wecom"
}

// Start initializes the WeCom Bot channel
func (c *WeComBotChannel) Start(ctx context.Context) error {
	logger.InfoC("wecom", "Starting WeCom Bot channel...")

	// Cancel the context created in the constructor to avoid a resource leak.
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(ctx)

	c.SetRunning(true)
	logger.InfoC("wecom", "WeCom Bot channel started")

	return nil
}

// Stop gracefully stops the WeCom Bot channel
func (c *WeComBotChannel) Stop(ctx context.Context) error {
	logger.InfoC("wecom", "Stopping WeCom Bot channel...")

	if c.cancel != nil {
		c.cancel()
	}

	c.SetRunning(false)
	logger.InfoC("wecom", "WeCom Bot channel stopped")
	return nil
}

// Send sends a message to WeCom user via webhook API
// Note: WeCom Bot can only reply within the configured timeout (default 5 seconds) of receiving a message
// For delayed responses, we use the webhook URL
func (c *WeComBotChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	logger.DebugCF("wecom", "Sending message via webhook", map[string]any{
		"chat_id": msg.ChatID,
		"preview": utils.Truncate(msg.Content, 100),
	})

	return c.sendWebhookReply(ctx, msg.ChatID, msg.Content)
}

// WebhookPath returns the path for registering on the shared HTTP server.
func (c *WeComBotChannel) WebhookPath() string {
	if c.config.WebhookPath != "" {
		return c.config.WebhookPath
	}
	return "/webhook/wecom"
}

// ServeHTTP implements http.Handler for the shared HTTP server.
func (c *WeComBotChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.handleWebhook(w, r)
}

// HealthPath returns the health check endpoint path.
func (c *WeComBotChannel) HealthPath() string {
	return "/health/wecom"
}

// HealthHandler handles health check requests.
func (c *WeComBotChannel) HealthHandler(w http.ResponseWriter, r *http.Request) {
	c.handleHealth(w, r)
}

// handleWebhook handles incoming webhook requests from WeCom
func (c *WeComBotChannel) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method == http.MethodGet {
		// Handle verification request
		c.handleVerification(ctx, w, r)
		return
	}

	if r.Method == http.MethodPost {
		// Handle message callback
		c.handleMessageCallback(ctx, w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleVerification handles the URL verification request from WeCom
func (c *WeComBotChannel) handleVerification(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	msgSignature := query.Get("msg_signature")
	timestamp := query.Get("timestamp")
	nonce := query.Get("nonce")
	echostr := query.Get("echostr")

	if msgSignature == "" || timestamp == "" || nonce == "" || echostr == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}

	// Verify signature
	if !verifySignature(c.config.Token, msgSignature, timestamp, nonce, echostr) {
		logger.WarnC("wecom", "Signature verification failed")
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	// Decrypt echostr
	// For AIBOT (智能机器人), receiveid should be empty string ""
	// Reference: https://developer.work.weixin.qq.com/document/path/101033
	decryptedEchoStr, err := decryptMessageWithVerify(echostr, c.config.EncodingAESKey, "")
	if err != nil {
		logger.ErrorCF("wecom", "Failed to decrypt echostr", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Decryption failed", http.StatusInternalServerError)
		return
	}

	// Remove BOM and whitespace as per WeCom documentation
	// The response must be plain text without quotes, BOM, or newlines
	decryptedEchoStr = strings.TrimSpace(decryptedEchoStr)
	decryptedEchoStr = strings.TrimPrefix(decryptedEchoStr, "\xef\xbb\xbf") // Remove UTF-8 BOM
	w.Write([]byte(decryptedEchoStr))
}

// handleMessageCallback handles incoming messages from WeCom
func (c *WeComBotChannel) handleMessageCallback(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	msgSignature := query.Get("msg_signature")
	timestamp := query.Get("timestamp")
	nonce := query.Get("nonce")

	if msgSignature == "" || timestamp == "" || nonce == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse XML to get encrypted message
	var encryptedMsg struct {
		XMLName    xml.Name `xml:"xml"`
		ToUserName string   `xml:"ToUserName"`
		Encrypt    string   `xml:"Encrypt"`
		AgentID    string   `xml:"AgentID"`
	}

	if err = xml.Unmarshal(body, &encryptedMsg); err != nil {
		logger.ErrorCF("wecom", "Failed to parse XML", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Invalid XML", http.StatusBadRequest)
		return
	}

	// Verify signature
	if !verifySignature(c.config.Token, msgSignature, timestamp, nonce, encryptedMsg.Encrypt) {
		logger.WarnC("wecom", "Message signature verification failed")
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	// Decrypt message
	// For AIBOT (智能机器人), receiveid should be empty string ""
	// Reference: https://developer.work.weixin.qq.com/document/path/101033
	decryptedMsg, err := decryptMessageWithVerify(encryptedMsg.Encrypt, c.config.EncodingAESKey, "")
	if err != nil {
		logger.ErrorCF("wecom", "Failed to decrypt message", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Decryption failed", http.StatusInternalServerError)
		return
	}

	// Parse decrypted JSON message (AIBOT uses JSON format)
	var msg WeComBotMessage
	if err := json.Unmarshal([]byte(decryptedMsg), &msg); err != nil {
		logger.ErrorCF("wecom", "Failed to parse decrypted message", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Invalid message format", http.StatusBadRequest)
		return
	}

	// Process the message with the channel's long-lived context (not the HTTP
	// request context, which is canceled as soon as we return the response).
	go c.processMessage(c.ctx, msg)

	// Return success response immediately
	// WeCom Bot requires response within configured timeout (default 5 seconds)
	w.Write([]byte("success"))
}

// processMessage processes the received message
func (c *WeComBotChannel) processMessage(ctx context.Context, msg WeComBotMessage) {
	// Skip unsupported message types
	if msg.MsgType != "text" && msg.MsgType != "image" && msg.MsgType != "voice" && msg.MsgType != "file" &&
		msg.MsgType != "mixed" {
		logger.DebugCF("wecom", "Skipping non-supported message type", map[string]any{
			"msg_type": msg.MsgType,
		})
		return
	}

	// Message deduplication: Use msg_id to prevent duplicate processing
	msgID := msg.MsgID
	if !c.processedMsgs.MarkMessageProcessed(msgID) {
		logger.DebugCF("wecom", "Skipping duplicate message", map[string]any{
			"msg_id": msgID,
		})
		return
	}

	senderID := msg.From.UserID

	// Determine if this is a group chat or direct message
	// ChatType: "single" for DM, "group" for group chat
	isGroupChat := msg.ChatType == "group"

	var chatID, peerKind, peerID string
	if isGroupChat {
		// Group chat: use ChatID as chatID and peer_id
		chatID = msg.ChatID
		peerKind = "group"
		peerID = msg.ChatID
	} else {
		// Direct message: use senderID as chatID and peer_id
		chatID = senderID
		peerKind = "direct"
		peerID = senderID
	}

	// Extract content based on message type
	var content string
	var mediaRefs []string

	scope := channels.BuildMediaScope("wecom", chatID, msg.MsgID)

	switch msg.MsgType {
	case "text":
		content = msg.Text.Content
	case "voice":
		content = msg.Voice.Content // Voice to text content
	case "mixed":
		// For mixed messages, process text and image items
		for _, item := range msg.Mixed.MsgItem {
			if item.MsgType == "text" {
				content += item.Text.Content
			}
		}
		// Download images from mixed messages
		if store := c.GetMediaStore(); store != nil {
			for _, item := range msg.Mixed.MsgItem {
				if item.MsgType == "image" && item.Image.URL != "" {
					ref := c.downloadMediaFromURL(ctx, item.Image.URL, "image.jpg", store, scope, msg.MsgID)
					if ref != "" {
						mediaRefs = append(mediaRefs, ref)
					}
				}
			}
		}
	case "image":
		// Download image from URL
		if msg.Image.URL != "" {
			if store := c.GetMediaStore(); store != nil {
				ref := c.downloadMediaFromURL(ctx, msg.Image.URL, "image.jpg", store, scope, msg.MsgID)
				if ref != "" {
					mediaRefs = append(mediaRefs, ref)
				}
			}
		}
		content = "[image]"
	case "file":
		// Download file from URL
		if msg.File.URL != "" {
			if store := c.GetMediaStore(); store != nil {
				ref := c.downloadMediaFromURL(ctx, msg.File.URL, "file", store, scope, msg.MsgID)
				if ref != "" {
					mediaRefs = append(mediaRefs, ref)
				}
			}
		}
		content = "[file]"
	}

	// Append media tags to content if needed
	if len(mediaRefs) > 0 && content == "" {
		switch msg.MsgType {
		case "image":
			content = "[image]"
		case "file":
			content = "[file]"
		}
	}

	// Build metadata
	peer := bus.Peer{Kind: peerKind, ID: peerID}

	// In group chats, apply unified group trigger filtering
	if isGroupChat {
		respond, cleaned := c.ShouldRespondInGroup(false, content)
		if !respond {
			return
		}
		content = cleaned
	}

	metadata := map[string]string{
		"msg_type":     msg.MsgType,
		"msg_id":       msg.MsgID,
		"platform":     "wecom",
		"response_url": msg.ResponseURL,
	}
	if isGroupChat {
		metadata["chat_id"] = msg.ChatID
		metadata["sender_id"] = senderID
	}

	logger.DebugCF("wecom", "Received message", map[string]any{
		"sender_id":     senderID,
		"msg_type":      msg.MsgType,
		"peer_kind":     peerKind,
		"is_group_chat": isGroupChat,
		"preview":       utils.Truncate(content, 50),
	})

	// Build sender info
	sender := bus.SenderInfo{
		Platform:    "wecom",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("wecom", senderID),
	}

	if !c.IsAllowedSender(sender) {
		return
	}

	// Handle the message through the base channel
	c.HandleMessage(ctx, peer, msg.MsgID, senderID, chatID, content, mediaRefs, metadata, sender)
}

// sendWebhookReply sends a reply using the webhook URL
func (c *WeComBotChannel) sendWebhookReply(ctx context.Context, userID, content string) error {
	reply := WeComBotReplyMessage{
		MsgType: "text",
	}
	reply.Text.Content = content

	jsonData, err := json.Marshal(reply)
	if err != nil {
		return fmt.Errorf("failed to marshal reply: %w", err)
	}

	// Use configurable timeout (default 5 seconds)
	timeout := c.config.ReplyTimeout
	if timeout <= 0 {
		timeout = 5
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.config.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return channels.ClassifyNetError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return channels.ClassifySendError(
				resp.StatusCode,
				fmt.Errorf("reading webhook error response: %w", readErr),
			)
		}
		return channels.ClassifySendError(
			resp.StatusCode,
			fmt.Errorf("webhook API error: %s", string(body)),
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check response
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("webhook API error: %s (code: %d)", result.ErrMsg, result.ErrCode)
	}

	return nil
}

// handleHealth handles health check requests
func (c *WeComBotChannel) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]any{
		"status":  "ok",
		"running": c.IsRunning(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// downloadMediaFromURL downloads media from a URL and stores it in MediaStore.
func (c *WeComBotChannel) downloadMediaFromURL(
	ctx context.Context,
	mediaURL, filename string,
	store media.MediaStore,
	scope string,
	msgID string,
) string {
	// Download the media file from the URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		logger.ErrorCF("wecom", "Failed to create media download request", map[string]any{
			"error": err.Error(),
			"url":    mediaURL,
		})
		return ""
	}

	// Add User-Agent header for WeCom/COS to properly serve the image
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")

	resp, err := c.client.Do(req)
	if err != nil {
		logger.ErrorCF("wecom", "Failed to download media", map[string]any{
			"error": err.Error(),
			"url":  mediaURL,
		})
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF("wecom", "Media download failed with status", map[string]any{
			"status": resp.StatusCode,
			"url":    mediaURL,
		})
		return ""
	}

	// Read the media body first
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorCF("wecom", "Failed to read media body", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	// Determine actual file type from content (magic bytes)
	contentType := resp.Header.Get("Content-Type")
	if kind, err := filetype.Match(body); err == nil && kind != filetype.Unknown {
		// Use the detected type from content
		contentType = kind.MIME.Value
		filename = "image." + kind.Extension
		logger.DebugCF("wecom", "Filetype detected from content", map[string]any{
			"detected_type": kind.MIME.Value,
			"extension":     kind.Extension,
			"size":          len(body),
		})
	} else {
		// Filetype detection failed, try to determine from header or extension
		logger.WarnCF("wecom", "Filetype detection failed, using extension inference", map[string]any{
			"header_content_type": contentType,
			"error":              err,
		})
		// Try to determine from file extension
		ext := filepath.Ext(filename)
		if ext == "" {
			// No extension, try header
			switch {
			case strings.Contains(contentType, "image/jpeg"):
				filename += ".jpg"
				contentType = "image/jpeg"
			case strings.Contains(contentType, "image/png"):
				filename += ".png"
				contentType = "image/png"
			case strings.Contains(contentType, "image/gif"):
				filename += ".gif"
				contentType = "image/gif"
			case strings.Contains(contentType, "image/webp"):
				filename += ".webp"
				contentType = "image/webp"
			case strings.Contains(contentType, "audio/amr"):
				filename += ".amr"
				contentType = "audio/amr"
			case strings.Contains(contentType, "audio/mp3") || strings.Contains(contentType, "audio/mpeg"):
				filename += ".mp3"
				contentType = "audio/mpeg"
			case strings.Contains(contentType, "video/mp4"):
				filename += ".mp4"
				contentType = "video/mp4"
			case strings.Contains(contentType, "application/pdf"):
				filename += ".pdf"
				contentType = "application/pdf"
			default:
				filename += ".bin"
			}
		} else {
			// Has extension, set correct MIME type based on extension
			switch ext {
			case ".jpg", ".jpeg":
				contentType = "image/jpeg"
			case ".png":
				contentType = "image/png"
			case ".gif":
				contentType = "image/gif"
			case ".webp":
				contentType = "image/webp"
			case ".amr":
				contentType = "audio/amr"
			case ".mp3":
				contentType = "audio/mpeg"
			case ".mp4":
				contentType = "video/mp4"
			case ".pdf":
				contentType = "application/pdf"
			}
		}
	}

	// Generate temp file path
	tempDir := os.TempDir()
	mediaDir := filepath.Join(tempDir, "picoclaw_media", "wecom")
	if mkdirErr := os.MkdirAll(mediaDir, 0o755); mkdirErr != nil {
		logger.ErrorCF("wecom", "Failed to create media directory", map[string]any{
			"error": mkdirErr.Error(),
		})
		return ""
	}

	// Create a unique filename to avoid collisions
	uniqueFilename := fmt.Sprintf("%s-%d-%s", msgID, time.Now().Unix(), filename)
	localPath := filepath.Join(mediaDir, uniqueFilename)

	// Write the file
	err = os.WriteFile(localPath, body, 0o644)
	if err != nil {
		logger.ErrorCF("wecom", "Failed to write media file", map[string]any{
			"error": err.Error(),
			"path":  localPath,
		})
		return ""
	}

	// Verify the image is valid by decoding it
	f, err := os.Open(localPath)
	if err == nil {
		_, _, decodeErr := image.Decode(f)
		f.Close()
		if decodeErr != nil {
			logger.ErrorCF("wecom", "Downloaded file is not a valid image", map[string]any{
				"path":   localPath,
				"error":  decodeErr.Error(),
				"size":   len(body),
				"header": fmt.Sprintf("%x", body[:min(20, len(body))]),
			})
			// Remove invalid file
			os.Remove(localPath)
			return ""
		}
	} else {
		logger.WarnCF("wecom", "Failed to open file for verification", map[string]any{
			"error": err.Error(),
			"path":  localPath,
		})
	}

	// Store in media store
	ref, err := store.Store(localPath, media.MediaMeta{
		Filename:    filename,
		ContentType: contentType,
		Source:      "wecom",
	}, scope)
	if err != nil {
		logger.ErrorCF("wecom", "Failed to store media", map[string]any{
			"error": err.Error(),
			"path":  localPath,
		})
		os.Remove(localPath)
		return ""
	}

	logger.DebugCF("wecom", "Media downloaded successfully", map[string]any{
		"url":  mediaURL,
		"path": localPath,
		"size": len(body),
	})

	return ref
}
