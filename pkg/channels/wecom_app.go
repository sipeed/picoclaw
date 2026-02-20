// PicoClaw - Ultra-lightweight personal AI agent
// WeCom App (企业微信自建应用) channel implementation
// Supports receiving messages via webhook callback and sending messages proactively

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
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	wecomAPIBase = "https://qyapi.weixin.qq.com"
)

// WeComAppChannel implements the Channel interface for WeCom App (企业微信自建应用)
type WeComAppChannel struct {
	*BaseChannel
	config        config.WeComAppConfig
	server        *http.Server
	accessToken   string
	tokenExpiry   time.Time
	tokenMu       sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	processedMsgs map[string]bool // Message deduplication: msg_id -> processed
	msgMu         sync.RWMutex
}

// WeComXMLMessage represents the XML message structure from WeCom
type WeComXMLMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgId        int64    `xml:"MsgId"`
	AgentID      int64    `xml:"AgentID"`
	PicUrl       string   `xml:"PicUrl"`
	MediaId      string   `xml:"MediaId"`
	Format       string   `xml:"Format"`
	ThumbMediaId string   `xml:"ThumbMediaId"`
	LocationX    float64  `xml:"Location_X"`
	LocationY    float64  `xml:"Location_Y"`
	Scale        int      `xml:"Scale"`
	Label        string   `xml:"Label"`
	Title        string   `xml:"Title"`
	Description  string   `xml:"Description"`
	Url          string   `xml:"Url"`
	Event        string   `xml:"Event"`
	EventKey     string   `xml:"EventKey"`
}

// WeComTextMessage represents text message for sending
type WeComTextMessage struct {
	ToUser  string `json:"touser"`
	MsgType string `json:"msgtype"`
	AgentID int64  `json:"agentid"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
	Safe int `json:"safe,omitempty"`
}

// WeComMarkdownMessage represents markdown message for sending
type WeComMarkdownMessage struct {
	ToUser   string `json:"touser"`
	MsgType  string `json:"msgtype"`
	AgentID  int64  `json:"agentid"`
	Markdown struct {
		Content string `json:"content"`
	} `json:"markdown"`
}

// WeComImageMessage represents image message for sending
type WeComImageMessage struct {
	ToUser  string `json:"touser"`
	MsgType string `json:"msgtype"`
	AgentID int64  `json:"agentid"`
	Image   struct {
		MediaID string `json:"media_id"`
	} `json:"image"`
}

// WeComAccessTokenResponse represents the access token API response
type WeComAccessTokenResponse struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// WeComSendMessageResponse represents the send message API response
type WeComSendMessageResponse struct {
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
	InvalidUser  string `json:"invaliduser"`
	InvalidParty string `json:"invalidparty"`
	InvalidTag   string `json:"invalidtag"`
}

// PKCS7Padding adds PKCS7 padding
type PKCS7Padding struct{}

// NewWeComAppChannel creates a new WeCom App channel instance
func NewWeComAppChannel(cfg config.WeComAppConfig, messageBus *bus.MessageBus) (*WeComAppChannel, error) {
	if cfg.CorpID == "" || cfg.CorpSecret == "" || cfg.AgentID == 0 {
		return nil, fmt.Errorf("wecom_app corp_id, corp_secret and agent_id are required")
	}

	base := NewBaseChannel("wecom_app", cfg, messageBus, cfg.AllowFrom)

	return &WeComAppChannel{
		BaseChannel:   base,
		config:        cfg,
		processedMsgs: make(map[string]bool),
	}, nil
}

// Name returns the channel name
func (c *WeComAppChannel) Name() string {
	return "wecom_app"
}

// Start initializes the WeCom App channel with HTTP webhook server
func (c *WeComAppChannel) Start(ctx context.Context) error {
	logger.InfoC("wecom_app", "Starting WeCom App channel...")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Get initial access token
	if err := c.refreshAccessToken(); err != nil {
		logger.WarnCF("wecom_app", "Failed to get initial access token", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Start token refresh goroutine
	go c.tokenRefreshLoop()

	// Setup HTTP server for webhook
	mux := http.NewServeMux()
	webhookPath := c.config.WebhookPath
	if webhookPath == "" {
		webhookPath = "/webhook/wecom-app"
	}
	mux.HandleFunc(webhookPath, c.handleWebhook)

	// Health check endpoint
	mux.HandleFunc("/health/wecom-app", c.handleHealth)

	addr := fmt.Sprintf("%s:%d", c.config.WebhookHost, c.config.WebhookPort)
	c.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	c.setRunning(true)
	logger.InfoCF("wecom_app", "WeCom App channel started", map[string]interface{}{
		"address": addr,
		"path":    webhookPath,
	})

	// Start server in goroutine
	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("wecom_app", "HTTP server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

// Stop gracefully stops the WeCom App channel
func (c *WeComAppChannel) Stop(ctx context.Context) error {
	logger.InfoC("wecom_app", "Stopping WeCom App channel...")

	if c.cancel != nil {
		c.cancel()
	}

	if c.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		c.server.Shutdown(shutdownCtx)
	}

	c.setRunning(false)
	logger.InfoC("wecom_app", "WeCom App channel stopped")
	return nil
}

// Send sends a message to WeCom user proactively using access token
func (c *WeComAppChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("wecom_app channel not running")
	}

	accessToken := c.getAccessToken()
	if accessToken == "" {
		return fmt.Errorf("no valid access token available")
	}

	logger.DebugCF("wecom_app", "Sending message", map[string]interface{}{
		"chat_id": msg.ChatID,
		"preview": utils.Truncate(msg.Content, 100),
	})

	return c.sendTextMessage(ctx, accessToken, msg.ChatID, msg.Content)
}

// handleWebhook handles incoming webhook requests from WeCom
func (c *WeComAppChannel) handleWebhook(w http.ResponseWriter, r *http.Request) {
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
func (c *WeComAppChannel) handleVerification(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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
		logger.WarnC("wecom_app", "Signature verification failed")
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	// Decrypt echostr
	decryptedEchoStr, err := c.decryptMessage(echostr)
	if err != nil {
		logger.ErrorCF("wecom_app", "Failed to decrypt echostr", map[string]interface{}{
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
func (c *WeComAppChannel) handleMessageCallback(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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
		logger.ErrorCF("wecom_app", "Failed to parse XML", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid XML", http.StatusBadRequest)
		return
	}

	// Verify signature
	if !c.verifySignature(msgSignature, timestamp, nonce, encryptedMsg.Encrypt) {
		logger.WarnC("wecom_app", "Message signature verification failed")
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	// Decrypt message
	decryptedMsg, err := c.decryptMessage(encryptedMsg.Encrypt)
	if err != nil {
		logger.ErrorCF("wecom_app", "Failed to decrypt message", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Decryption failed", http.StatusInternalServerError)
		return
	}

	// Parse decrypted XML message
	var msg WeComXMLMessage
	if err := xml.Unmarshal([]byte(decryptedMsg), &msg); err != nil {
		logger.ErrorCF("wecom_app", "Failed to parse decrypted message", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid message format", http.StatusBadRequest)
		return
	}

	// Process the message with context
	go c.processMessage(ctx, msg)

	// Return success response immediately
	// WeCom App requires response within configured timeout (default 5 seconds)
	w.Write([]byte("success"))
}

// processMessage processes the received message
func (c *WeComAppChannel) processMessage(ctx context.Context, msg WeComXMLMessage) {
	// Skip non-text messages for now (can be extended)
	if msg.MsgType != "text" && msg.MsgType != "image" && msg.MsgType != "voice" {
		logger.DebugCF("wecom_app", "Skipping non-supported message type", map[string]interface{}{
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
		logger.DebugCF("wecom_app", "Skipping duplicate message", map[string]interface{}{
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
	chatID := senderID // WeCom App uses user ID as chat ID for direct messages

	// Build metadata
	// WeCom App only supports direct messages (private chat)
	metadata := map[string]string{
		"msg_type":    msg.MsgType,
		"msg_id":      fmt.Sprintf("%d", msg.MsgId),
		"agent_id":    fmt.Sprintf("%d", msg.AgentID),
		"platform":    "wecom_app",
		"media_id":    msg.MediaId,
		"create_time": fmt.Sprintf("%d", msg.CreateTime),
		"peer_kind":   "direct",
		"peer_id":     senderID,
	}

	content := msg.Content

	logger.DebugCF("wecom_app", "Received message", map[string]interface{}{
		"sender_id": senderID,
		"msg_type":  msg.MsgType,
		"preview":   utils.Truncate(content, 50),
	})

	// Handle the message through the base channel
	c.HandleMessage(senderID, chatID, content, nil, metadata)
}

// verifySignature verifies the message signature
func (c *WeComAppChannel) verifySignature(msgSignature, timestamp, nonce, msgEncrypt string) bool {
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
func (c *WeComAppChannel) decryptMessage(encryptedMsg string) (string, error) {
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
	plainText, err = pkcs7Unpad(plainText)
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
	// corpID := plainText[20+msgLen:] // Can be used for verification

	return string(msg), nil
}

// pkcs7Unpad removes PKCS7 padding with validation
func pkcs7Unpad(data []byte) ([]byte, error) {
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

// tokenRefreshLoop periodically refreshes the access token
func (c *WeComAppChannel) tokenRefreshLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.refreshAccessToken(); err != nil {
				logger.ErrorCF("wecom_app", "Failed to refresh access token", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	}
}

// refreshAccessToken gets a new access token from WeCom API
func (c *WeComAppChannel) refreshAccessToken() error {
	apiURL := fmt.Sprintf("%s/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		wecomAPIBase, url.QueryEscape(c.config.CorpID), url.QueryEscape(c.config.CorpSecret))

	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to request access token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp WeComAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.ErrCode != 0 {
		return fmt.Errorf("API error: %s (code: %d)", tokenResp.ErrMsg, tokenResp.ErrCode)
	}

	c.tokenMu.Lock()
	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-300) * time.Second) // Refresh 5 minutes early
	c.tokenMu.Unlock()

	logger.DebugC("wecom_app", "Access token refreshed successfully")
	return nil
}

// getAccessToken returns the current valid access token
func (c *WeComAppChannel) getAccessToken() string {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()

	if time.Now().After(c.tokenExpiry) {
		return ""
	}

	return c.accessToken
}

// sendTextMessage sends a text message to a user
func (c *WeComAppChannel) sendTextMessage(ctx context.Context, accessToken, userID, content string) error {
	apiURL := fmt.Sprintf("%s/cgi-bin/message/send?access_token=%s", wecomAPIBase, accessToken)

	msg := WeComTextMessage{
		ToUser:  userID,
		MsgType: "text",
		AgentID: c.config.AgentID,
	}
	msg.Text.Content = content

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Use configurable timeout (default 5 seconds)
	timeout := c.config.ReplyTimeout
	if timeout <= 0 {
		timeout = 5
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var sendResp WeComSendMessageResponse
	if err := json.Unmarshal(body, &sendResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if sendResp.ErrCode != 0 {
		return fmt.Errorf("API error: %s (code: %d)", sendResp.ErrMsg, sendResp.ErrCode)
	}

	return nil
}

// sendMarkdownMessage sends a markdown message to a user
func (c *WeComAppChannel) sendMarkdownMessage(ctx context.Context, accessToken, userID, content string) error {
	apiURL := fmt.Sprintf("%s/cgi-bin/message/send?access_token=%s", wecomAPIBase, accessToken)

	msg := WeComMarkdownMessage{
		ToUser:  userID,
		MsgType: "markdown",
		AgentID: c.config.AgentID,
	}
	msg.Markdown.Content = content

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Use configurable timeout (default 5 seconds)
	timeout := c.config.ReplyTimeout
	if timeout <= 0 {
		timeout = 5
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var sendResp WeComSendMessageResponse
	if err := json.Unmarshal(body, &sendResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if sendResp.ErrCode != 0 {
		return fmt.Errorf("API error: %s (code: %d)", sendResp.ErrMsg, sendResp.ErrCode)
	}

	return nil
}

// handleHealth handles health check requests
func (c *WeComAppChannel) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "ok",
		"running":   c.IsRunning(),
		"has_token": c.getAccessToken() != "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
