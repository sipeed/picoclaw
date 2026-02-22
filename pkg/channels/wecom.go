// PicoClaw - Ultra-lightweight personal AI agent
// WeCom Bot (企业微信智能机器人) channel implementation
// Uses webhook callback mode for receiving messages and webhook API for sending replies

package channels

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// WeComBotChannel implements the Channel interface for WeCom Bot (企业微信智能机器人)
// Uses webhook callback mode - simpler than WeCom App but only supports passive replies
type WeComBotChannel struct {
	*BaseChannel
	config        config.WeComConfig
	server        *http.Server
	ctx           context.Context
	cancel        context.CancelFunc
	processedMsgs map[string]bool // Message deduplication: msg_id -> processed
	msgMu         sync.RWMutex
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

	base := NewBaseChannel("wecom", cfg, messageBus, cfg.AllowFrom)

	return &WeComBotChannel{
		BaseChannel:   base,
		config:        cfg,
		processedMsgs: make(map[string]bool),
	}, nil
}

// Name returns the channel name
func (c *WeComBotChannel) Name() string {
	return "wecom"
}

// Start initializes the WeCom Bot channel with HTTP webhook server
func (c *WeComBotChannel) Start(ctx context.Context) error {
	logger.InfoC("wecom", "Starting WeCom Bot channel...")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Setup HTTP server for webhook
	mux := http.NewServeMux()
	webhookPath := c.config.WebhookPath
	if webhookPath == "" {
		webhookPath = "/webhook/wecom"
	}
	mux.HandleFunc(webhookPath, c.handleWebhook)

	// Health check endpoint
	mux.HandleFunc("/health/wecom", c.handleHealth)

	addr := fmt.Sprintf("%s:%d", c.config.WebhookHost, c.config.WebhookPort)
	c.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	c.setRunning(true)
	logger.InfoCF("wecom", "WeCom Bot channel started", map[string]any{
		"address": addr,
		"path":    webhookPath,
	})

	// Start server in goroutine
	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("wecom", "HTTP server error", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

// Stop gracefully stops the WeCom Bot channel
func (c *WeComBotChannel) Stop(ctx context.Context) error {
	logger.InfoC("wecom", "Stopping WeCom Bot channel...")

	if c.cancel != nil {
		c.cancel()
	}

	if c.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		c.server.Shutdown(shutdownCtx)
	}

	c.setRunning(false)
	logger.InfoC("wecom", "WeCom Bot channel stopped")
	return nil
}

// Send sends a message to WeCom user via webhook API
// Note: WeCom Bot can only reply within the configured timeout (default 5 seconds) of receiving a message
// For delayed responses, we use the webhook URL
func (c *WeComBotChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("wecom channel not running")
	}

	logger.DebugCF("wecom", "Sending message via webhook", map[string]any{
		"chat_id": msg.ChatID,
		"preview": utils.Truncate(msg.Content, 100),
	})

	return c.sendWebhookReply(ctx, msg.ChatID, msg.Content)
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
	if !WeComVerifySignature(c.config.Token, msgSignature, timestamp, nonce, echostr) {
		logger.WarnC("wecom", "Signature verification failed")
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	// Decrypt echostr
	// For AIBOT (智能机器人), receiveid should be empty string ""
	// Reference: https://developer.work.weixin.qq.com/document/path/101033
	decryptedEchoStr, err := WeComDecryptMessageWithVerify(echostr, c.config.EncodingAESKey, "")
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
	if !WeComVerifySignature(c.config.Token, msgSignature, timestamp, nonce, encryptedMsg.Encrypt) {
		logger.WarnC("wecom", "Message signature verification failed")
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	// Decrypt message
	// For AIBOT (智能机器人), receiveid should be empty string ""
	// Reference: https://developer.work.weixin.qq.com/document/path/101033
	decryptedMsg, err := WeComDecryptMessageWithVerify(encryptedMsg.Encrypt, c.config.EncodingAESKey, "")
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

	// Process the message asynchronously with context
	go c.processMessage(ctx, msg)

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
	c.msgMu.Lock()
	if c.processedMsgs[msgID] {
		c.msgMu.Unlock()
		logger.DebugCF("wecom", "Skipping duplicate message", map[string]any{
			"msg_id": msgID,
		})
		return
	}
	c.processedMsgs[msgID] = true
	c.msgMu.Unlock()

	// Clean up old messages periodically (keep last 1000)
	if len(c.processedMsgs) > 1000 {
		c.msgMu.Lock()
		c.processedMsgs = make(map[string]bool)
		c.msgMu.Unlock()
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
	switch msg.MsgType {
	case "text":
		content = msg.Text.Content
	case "voice":
		content = msg.Voice.Content // Voice to text content
	case "mixed":
		// For mixed messages, concatenate text items
		for _, item := range msg.Mixed.MsgItem {
			if item.MsgType == "text" {
				content += item.Text.Content
			}
		}
	case "image", "file":
		// For image and file, we don't have text content
		content = ""
	}

	// Build metadata
	metadata := map[string]string{
		"msg_type":     msg.MsgType,
		"msg_id":       msg.MsgID,
		"platform":     "wecom",
		"peer_kind":    peerKind,
		"peer_id":      peerID,
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

	// Handle the message through the base channel
	c.HandleMessage(senderID, chatID, content, nil, metadata)
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

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook reply: %w", err)
	}
	defer resp.Body.Close()

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

// WeCom common utilities for both WeCom Bot and WeCom App
// The following functions were moved from wecom_common.go

// WeComVerifySignature verifies the message signature for WeCom
// This is a common function used by both WeCom Bot and WeCom App
func WeComVerifySignature(token, msgSignature, timestamp, nonce, msgEncrypt string) bool {
	if token == "" {
		return true // Skip verification if token is not set
	}

	// Sort parameters
	params := []string{token, timestamp, nonce, msgEncrypt}
	sort.Strings(params)

	// Concatenate
	str := strings.Join(params, "")

	// SHA1 hash
	hash := sha1.Sum([]byte(str))
	expectedSignature := fmt.Sprintf("%x", hash)

	return expectedSignature == msgSignature
}

// WeComDecryptMessage decrypts the encrypted message using AES
// This is a common function used by both WeCom Bot and WeCom App
// For AIBOT, receiveid should be the aibotid; for other apps, it should be corp_id
func WeComDecryptMessage(encryptedMsg, encodingAESKey string) (string, error) {
	return WeComDecryptMessageWithVerify(encryptedMsg, encodingAESKey, "")
}

// WeComDecryptMessageWithVerify decrypts the encrypted message and optionally verifies receiveid
// receiveid: for AIBOT use aibotid, for WeCom App use corp_id. If empty, skip verification.
func WeComDecryptMessageWithVerify(encryptedMsg, encodingAESKey, receiveid string) (string, error) {
	if encodingAESKey == "" {
		// No encryption, return as is (base64 decode)
		decoded, err := base64.StdEncoding.DecodeString(encryptedMsg)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}

	// Decode AES key (base64)
	aesKey, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return "", fmt.Errorf("failed to decode AES key: %w", err)
	}

	// Decode encrypted message
	cipherText, err := base64.StdEncoding.DecodeString(encryptedMsg)
	if err != nil {
		return "", fmt.Errorf("failed to decode message: %w", err)
	}

	// AES decrypt
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(cipherText) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// IV is the first 16 bytes of AESKey
	iv := aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)
	plainText := make([]byte, len(cipherText))
	mode.CryptBlocks(plainText, cipherText)

	// Remove PKCS7 padding
	plainText, err = pkcs7UnpadWeCom(plainText)
	if err != nil {
		return "", fmt.Errorf("failed to unpad: %w", err)
	}

	// Parse message structure
	// Format: random(16) + msg_len(4) + msg + receiveid
	if len(plainText) < 20 {
		return "", fmt.Errorf("decrypted message too short")
	}

	msgLen := binary.BigEndian.Uint32(plainText[16:20])
	if int(msgLen) > len(plainText)-20 {
		return "", fmt.Errorf("invalid message length")
	}

	msg := plainText[20 : 20+msgLen]

	// Verify receiveid if provided
	if receiveid != "" && len(plainText) > 20+int(msgLen) {
		actualReceiveID := string(plainText[20+msgLen:])
		if actualReceiveID != receiveid {
			return "", fmt.Errorf("receiveid mismatch: expected %s, got %s", receiveid, actualReceiveID)
		}
	}

	return string(msg), nil
}

// pkcs7UnpadWeCom removes PKCS7 padding with validation
// WeCom uses block size of 32 (not standard AES block size of 16)
const wecomBlockSize = 32

func pkcs7UnpadWeCom(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	padding := int(data[len(data)-1])
	// WeCom uses 32-byte block size for PKCS7 padding
	if padding == 0 || padding > wecomBlockSize {
		return nil, fmt.Errorf("invalid padding size: %d", padding)
	}
	if padding > len(data) {
		return nil, fmt.Errorf("padding size larger than data")
	}
	// Verify all padding bytes
	for i := 0; i < padding; i++ {
		if data[len(data)-1-i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding byte at position %d", i)
		}
	}
	return data[:len(data)-padding], nil
}
