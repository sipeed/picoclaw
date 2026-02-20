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

// WeComBotXMLMessage represents the XML message structure from WeCom Bot
type WeComBotXMLMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgId        int64    `xml:"MsgId"`
	PicUrl       string   `xml:"PicUrl"`
	MediaId      string   `xml:"MediaId"`
	Format       string   `xml:"Format"`
	Recognition  string   `xml:"Recognition"` // Voice recognition result
}

// WeComBotReplyMessage represents the reply message structure
type WeComBotReplyMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
}

// WeComBotWebhookReply represents the webhook API reply
type WeComBotWebhookReply struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text,omitempty"`
	Markdown struct {
		Content string `json:"content"`
	} `json:"markdown,omitempty"`
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
	logger.InfoCF("wecom", "WeCom Bot channel started", map[string]interface{}{
		"address": addr,
		"path":    webhookPath,
	})

	// Start server in goroutine
	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("wecom", "HTTP server error", map[string]interface{}{
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

	logger.DebugCF("wecom", "Sending message via webhook", map[string]interface{}{
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
	if !c.verifySignature(msgSignature, timestamp, nonce, echostr) {
		logger.WarnC("wecom", "Signature verification failed")
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	// Decrypt echostr
	decryptedEchoStr, err := c.decryptMessage(echostr)
	if err != nil {
		logger.ErrorCF("wecom", "Failed to decrypt echostr", map[string]interface{}{
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

	if err := xml.Unmarshal(body, &encryptedMsg); err != nil {
		logger.ErrorCF("wecom", "Failed to parse XML", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid XML", http.StatusBadRequest)
		return
	}

	// Verify signature
	if !c.verifySignature(msgSignature, timestamp, nonce, encryptedMsg.Encrypt) {
		logger.WarnC("wecom", "Message signature verification failed")
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	// Decrypt message
	decryptedMsg, err := c.decryptMessage(encryptedMsg.Encrypt)
	if err != nil {
		logger.ErrorCF("wecom", "Failed to decrypt message", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Decryption failed", http.StatusInternalServerError)
		return
	}

	// Parse decrypted XML message
	var msg WeComBotXMLMessage
	if err := xml.Unmarshal([]byte(decryptedMsg), &msg); err != nil {
		logger.ErrorCF("wecom", "Failed to parse decrypted message", map[string]interface{}{
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
func (c *WeComBotChannel) processMessage(ctx context.Context, msg WeComBotXMLMessage) {
	// Skip non-text messages for now (can be extended)
	if msg.MsgType != "text" && msg.MsgType != "image" && msg.MsgType != "voice" {
		logger.DebugCF("wecom", "Skipping non-supported message type", map[string]interface{}{
			"msg_type": msg.MsgType,
		})
		return
	}

	// Message deduplication: Use msg_id to prevent duplicate processing
	// As per WeCom documentation, use msg_id for deduplication
	msgID := fmt.Sprintf("%d", msg.MsgId)
	c.msgMu.Lock()
	if c.processedMsgs[msgID] {
		c.msgMu.Unlock()
		logger.DebugCF("wecom", "Skipping duplicate message", map[string]interface{}{
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

	senderID := msg.FromUserName
	chatID := senderID // WeCom Bot uses user ID as chat ID

	// Use voice recognition result if available
	content := msg.Content
	if msg.MsgType == "voice" && msg.Recognition != "" {
		content = msg.Recognition
	}

	// Build metadata
	// WeCom Bot only supports direct messages (private chat)
	metadata := map[string]string{
		"msg_type":    msg.MsgType,
		"msg_id":      fmt.Sprintf("%d", msg.MsgId),
		"platform":    "wecom",
		"media_id":    msg.MediaId,
		"create_time": fmt.Sprintf("%d", msg.CreateTime),
		"peer_kind":   "direct",
		"peer_id":     senderID,
	}

	logger.DebugCF("wecom", "Received message", map[string]interface{}{
		"sender_id": senderID,
		"msg_type":  msg.MsgType,
		"preview":   utils.Truncate(content, 50),
	})

	// Handle the message through the base channel
	c.HandleMessage(senderID, chatID, content, nil, metadata)
}

// verifySignature verifies the message signature
func (c *WeComBotChannel) verifySignature(msgSignature, timestamp, nonce, msgEncrypt string) bool {
	if c.config.Token == "" {
		return true // Skip verification if token is not set
	}

	// Sort parameters
	params := []string{c.config.Token, timestamp, nonce, msgEncrypt}
	sort.Strings(params)

	// Concatenate
	str := strings.Join(params, "")

	// SHA1 hash
	hash := sha1.Sum([]byte(str))
	expectedSignature := fmt.Sprintf("%x", hash)

	return expectedSignature == msgSignature
}

// decryptMessage decrypts the encrypted message using AES
func (c *WeComBotChannel) decryptMessage(encryptedMsg string) (string, error) {
	if c.config.EncodingAESKey == "" {
		// No encryption, return as is (base64 decode)
		decoded, err := base64.StdEncoding.DecodeString(encryptedMsg)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}

	// Decode AES key (base64)
	aesKey, err := base64.StdEncoding.DecodeString(c.config.EncodingAESKey + "=")
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

	mode := cipher.NewCBCDecrypter(block, aesKey[:aes.BlockSize])
	plainText := make([]byte, len(cipherText))
	mode.CryptBlocks(plainText, cipherText)

	// Remove PKCS7 padding
	plainText, err = pkcs7UnpadWeCom(plainText)
	if err != nil {
		return "", fmt.Errorf("failed to unpad: %w", err)
	}

	// Parse message structure
	// Format: random(16) + msg_len(4) + msg + corp_id
	if len(plainText) < 20 {
		return "", fmt.Errorf("decrypted message too short")
	}

	msgLen := binary.BigEndian.Uint32(plainText[16:20])
	if int(msgLen) > len(plainText)-20 {
		return "", fmt.Errorf("invalid message length")
	}

	msg := plainText[20 : 20+msgLen]
	// corpID := plainText[20+msgLen:] // Could be used for verification

	return string(msg), nil
}

// pkcs7UnpadWeCom removes PKCS7 padding with validation
func pkcs7UnpadWeCom(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > aes.BlockSize {
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

// sendWebhookReply sends a reply using the webhook URL
func (c *WeComBotChannel) sendWebhookReply(ctx context.Context, userID, content string) error {
	reply := WeComBotWebhookReply{
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
	status := map[string]interface{}{
		"status":  "ok",
		"running": c.IsRunning(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
