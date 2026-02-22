package wecom

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	wecomAPIBase = "https://qyapi.weixin.qq.com"
)

// WeComAppChannel implements the Channel interface for WeCom App (企业微信自建应用)
type WeComAppChannel struct {
	*channels.BaseChannel
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

	base := channels.NewBaseChannel("wecom_app", cfg, messageBus, cfg.AllowFrom, channels.WithMaxMessageLength(2048))

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
		logger.WarnCF("wecom_app", "Failed to get initial access token", map[string]any{
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

	c.SetRunning(true)
	logger.InfoCF("wecom_app", "WeCom App channel started", map[string]any{
		"address": addr,
		"path":    webhookPath,
	})

	// Start server in goroutine
	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("wecom_app", "HTTP server error", map[string]any{
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

	c.SetRunning(false)
	logger.InfoC("wecom_app", "WeCom App channel stopped")
	return nil
}

// Send sends a message to WeCom user proactively using access token
func (c *WeComAppChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	accessToken := c.getAccessToken()
	if accessToken == "" {
		return fmt.Errorf("no valid access token available")
	}

	logger.DebugCF("wecom_app", "Sending message", map[string]any{
		"chat_id": msg.ChatID,
		"preview": utils.Truncate(msg.Content, 100),
	})

	return c.sendTextMessage(ctx, accessToken, msg.ChatID, msg.Content)
}

// handleWebhook handles incoming webhook requests from WeCom
func (c *WeComAppChannel) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Log all incoming requests for debugging
	logger.DebugCF("wecom_app", "Received webhook request", map[string]any{
		"method": r.Method,
		"url":    r.URL.String(),
		"path":   r.URL.Path,
		"query":  r.URL.RawQuery,
	})

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

	logger.WarnCF("wecom_app", "Method not allowed", map[string]any{
		"method": r.Method,
	})
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleVerification handles the URL verification request from WeCom
func (c *WeComAppChannel) handleVerification(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	msgSignature := query.Get("msg_signature")
	timestamp := query.Get("timestamp")
	nonce := query.Get("nonce")
	echostr := query.Get("echostr")

	logger.DebugCF("wecom_app", "Handling verification request", map[string]any{
		"msg_signature": msgSignature,
		"timestamp":     timestamp,
		"nonce":         nonce,
		"echostr":       echostr,
		"corp_id":       c.config.CorpID,
	})

	if msgSignature == "" || timestamp == "" || nonce == "" || echostr == "" {
		logger.ErrorC("wecom_app", "Missing parameters in verification request")
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}

	// Verify signature
	if !verifySignature(c.config.Token, msgSignature, timestamp, nonce, echostr) {
		logger.WarnCF("wecom_app", "Signature verification failed", map[string]any{
			"token":         c.config.Token,
			"msg_signature": msgSignature,
			"timestamp":     timestamp,
			"nonce":         nonce,
		})
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	logger.DebugC("wecom_app", "Signature verification passed")

	// Decrypt echostr with CorpID verification
	// For WeCom App (自建应用), receiveid should be corp_id
	logger.DebugCF("wecom_app", "Attempting to decrypt echostr", map[string]any{
		"encoding_aes_key": c.config.EncodingAESKey,
		"corp_id":          c.config.CorpID,
	})
	decryptedEchoStr, err := decryptMessageWithVerify(echostr, c.config.EncodingAESKey, c.config.CorpID)
	if err != nil {
		logger.ErrorCF("wecom_app", "Failed to decrypt echostr", map[string]any{
			"error":            err.Error(),
			"encoding_aes_key": c.config.EncodingAESKey,
			"corp_id":          c.config.CorpID,
		})
		http.Error(w, "Decryption failed", http.StatusInternalServerError)
		return
	}

	logger.DebugCF("wecom_app", "Successfully decrypted echostr", map[string]any{
		"decrypted": decryptedEchoStr,
	})

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

	if err = xml.Unmarshal(body, &encryptedMsg); err != nil {
		logger.ErrorCF("wecom_app", "Failed to parse XML", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Invalid XML", http.StatusBadRequest)
		return
	}

	// Verify signature
	if !verifySignature(c.config.Token, msgSignature, timestamp, nonce, encryptedMsg.Encrypt) {
		logger.WarnC("wecom_app", "Message signature verification failed")
		http.Error(w, "Invalid signature", http.StatusForbidden)
		return
	}

	// Decrypt message with CorpID verification
	// For WeCom App (自建应用), receiveid should be corp_id
	decryptedMsg, err := decryptMessageWithVerify(encryptedMsg.Encrypt, c.config.EncodingAESKey, c.config.CorpID)
	if err != nil {
		logger.ErrorCF("wecom_app", "Failed to decrypt message", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Decryption failed", http.StatusInternalServerError)
		return
	}

	// Parse decrypted XML message
	var msg WeComXMLMessage
	if err := xml.Unmarshal([]byte(decryptedMsg), &msg); err != nil {
		logger.ErrorCF("wecom_app", "Failed to parse decrypted message", map[string]any{
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
		logger.DebugCF("wecom_app", "Skipping non-supported message type", map[string]any{
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
		logger.DebugCF("wecom_app", "Skipping duplicate message", map[string]any{
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
	peer := bus.Peer{Kind: "direct", ID: senderID}
	messageID := fmt.Sprintf("%d", msg.MsgId)

	metadata := map[string]string{
		"msg_type":    msg.MsgType,
		"msg_id":      fmt.Sprintf("%d", msg.MsgId),
		"agent_id":    fmt.Sprintf("%d", msg.AgentID),
		"platform":    "wecom_app",
		"media_id":    msg.MediaId,
		"create_time": fmt.Sprintf("%d", msg.CreateTime),
	}

	content := msg.Content

	logger.DebugCF("wecom_app", "Received message", map[string]any{
		"sender_id": senderID,
		"msg_type":  msg.MsgType,
		"preview":   utils.Truncate(content, 50),
	})

	// Handle the message through the base channel
	c.HandleMessage(peer, messageID, senderID, chatID, content, nil, metadata)
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
				logger.ErrorCF("wecom_app", "Failed to refresh access token", map[string]any{
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
		return channels.ClassifyNetError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return channels.ClassifySendError(resp.StatusCode, fmt.Errorf("wecom_app API error: %s", string(body)))
	}

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
		return channels.ClassifyNetError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return channels.ClassifySendError(resp.StatusCode, fmt.Errorf("wecom_app API error: %s", string(body)))
	}

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
	status := map[string]any{
		"status":    "ok",
		"running":   c.IsRunning(),
		"has_token": c.getAccessToken() != "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
