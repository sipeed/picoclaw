package wecom

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// responseURLHTTPClient is a shared HTTP client for posting to WeCom response_url.
// Reusing it enables connection pooling across replies.
var responseURLHTTPClient = &http.Client{Timeout: 15 * time.Second}

// WeComAIBotChannel implements the Channel interface for WeCom AI Bot (企业微信智能机器人)
type WeComAIBotChannel struct {
	*channels.BaseChannel
	config      config.WeComAIBotConfig
	ctx         context.Context
	cancel      context.CancelFunc
	streamTasks map[string]*streamTask   // streamID -> task (for poll lookups)
	chatTasks   map[string][]*streamTask // chatID   -> in-flight tasks queue (FIFO)
	taskMu      sync.RWMutex
}

// streamTask represents a streaming task for AI Bot.
//
// Mutable fields (Finished, StreamClosed, StreamClosedAt) must be read/written
// while holding WeComAIBotChannel.taskMu. Immutable fields (StreamID, ChatID,
// ResponseURL, Question, CreatedTime, Deadline, answerCh, ctx, cancel) are set
// once at creation and never modified, so they are safe to read without a lock.
type streamTask struct {
	// immutable after creation
	StreamID    string
	ChatID      string // used by Send() to find this task
	ResponseURL string // temporary URL for proactive reply (valid 1 hour, use once)
	Question    string
	CreatedTime time.Time
	Deadline    time.Time          // ~30s, we close the stream here and switch to response_url
	answerCh    chan string        // receives agent reply from Send()
	ctx         context.Context    // canceled when task is removed; used to interrupt the agent goroutine
	cancel      context.CancelFunc // call on task removal to cancel ctx

	// mutable — guarded by WeComAIBotChannel.taskMu
	StreamClosed   bool      // stream returned finish:true; waiting for agent to reply via response_url
	StreamClosedAt time.Time // set when StreamClosed becomes true; used for accelerated cleanup
	Finished       bool      // fully done
}

// WeComAIBotMessage represents the decrypted JSON message from WeCom AI Bot
// Ref: https://developer.work.weixin.qq.com/document/path/100719
type WeComAIBotMessage struct {
	MsgID    string `json:"msgid"`
	AIBotID  string `json:"aibotid"`
	ChatID   string `json:"chatid"`   // only for group chat
	ChatType string `json:"chattype"` // "single" or "group"
	From     struct {
		UserID string `json:"userid"`
	} `json:"from"`
	ResponseURL string `json:"response_url"` // temporary URL for proactive reply
	MsgType     string `json:"msgtype"`
	// text message
	Text *struct {
		Content string `json:"content"`
	} `json:"text,omitempty"`
	// stream polling refresh
	Stream *struct {
		ID string `json:"id"`
	} `json:"stream,omitempty"`
	// image message
	Image *struct {
		URL     string `json:"url"`
		MediaID string `json:"media_id"`
		AESKey  string `json:"aeskey,omitempty"`
	} `json:"image,omitempty"`
	// mixed message (text + image)
	Mixed *struct {
		MsgItem []struct {
			MsgType string `json:"msgtype"`
			Text    *struct {
				Content string `json:"content"`
			} `json:"text,omitempty"`
			Image *struct {
				URL     string `json:"url"`
				MediaID string `json:"media_id"`
				AESKey  string `json:"aeskey,omitempty"`
			} `json:"image,omitempty"`
		} `json:"msg_item"`
	} `json:"mixed,omitempty"`
	// event field
	Event *struct {
		EventType string `json:"eventtype"`
	} `json:"event,omitempty"`
}

// WeComAIBotMsgItemImage holds the image payload inside a stream message item.
type WeComAIBotMsgItemImage struct {
	Base64 string `json:"base64"`
	MD5    string `json:"md5"`
}

// WeComAIBotMsgItem is a single item inside a stream's msg_item list.
type WeComAIBotMsgItem struct {
	MsgType string                  `json:"msgtype"`
	Image   *WeComAIBotMsgItemImage `json:"image,omitempty"`
}

// WeComAIBotStreamInfo represents the detailed stream content in streaming responses.
type WeComAIBotStreamInfo struct {
	ID      string              `json:"id"`
	Finish  bool                `json:"finish"`
	Content string              `json:"content,omitempty"`
	MsgItem []WeComAIBotMsgItem `json:"msg_item,omitempty"`
}

// WeComAIBotStreamResponse represents the streaming response format
type WeComAIBotStreamResponse struct {
	MsgType string               `json:"msgtype"`
	Stream  WeComAIBotStreamInfo `json:"stream"`
}

// WeComAIBotEncryptedResponse represents the encrypted response wrapper
// Fields match WXBizJsonMsgCrypt.generate() in Python SDK
type WeComAIBotEncryptedResponse struct {
	Encrypt      string `json:"encrypt"`
	MsgSignature string `json:"msgsignature"`
	Timestamp    string `json:"timestamp"`
	Nonce        string `json:"nonce"`
}

// NewWeComAIBotChannel creates a WeCom AI Bot channel instance.
// If cfg.BotID and cfg.Secret are both set, it returns a WeComAIBotWSChannel
// using the WebSocket long-connection API.
// Otherwise it returns the webhook-mode WeComAIBotChannel (requires Token +
// EncodingAESKey).
func NewWeComAIBotChannel(
	cfg config.WeComAIBotConfig,
	messageBus *bus.MessageBus,
) (channels.Channel, error) {
	// WebSocket long-connection mode takes priority when BotID + Secret are set.
	if cfg.BotID != "" && cfg.Secret != "" {
		logger.InfoC("wecom_aibot", "BotID and Secret provided, using WebSocket mode")
		return newWeComAIBotWSChannel(cfg, messageBus)
	}
	// Webhook (short-connection) mode.
	if cfg.Token == "" || cfg.EncodingAESKey == "" {
		return nil, fmt.Errorf(
			"WeCom AI Bot requires either (bot_id + secret) for WebSocket mode " +
				"or (token + encoding_aes_key) for webhook mode")
	}
	if cfg.ProcessingMessage == "" {
		cfg.ProcessingMessage = config.DefaultWeComAIBotProcessingMessage
	}

	base := channels.NewBaseChannel("wecom_aibot", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(2048),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &WeComAIBotChannel{
		BaseChannel: base,
		config:      cfg,
		streamTasks: make(map[string]*streamTask),
		chatTasks:   make(map[string][]*streamTask),
	}, nil
}

// Name returns the channel name
func (c *WeComAIBotChannel) Name() string {
	return "wecom_aibot"
}

// Start initializes the WeCom AI Bot channel
func (c *WeComAIBotChannel) Start(ctx context.Context) error {
	logger.InfoC("wecom_aibot", "Starting WeCom AI Bot channel...")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Start cleanup goroutine for old tasks
	go c.cleanupLoop()

	c.SetRunning(true)
	logger.InfoC("wecom_aibot", "WeCom AI Bot channel started")

	return nil
}

// Stop gracefully stops the WeCom AI Bot channel
func (c *WeComAIBotChannel) Stop(ctx context.Context) error {
	logger.InfoC("wecom_aibot", "Stopping WeCom AI Bot channel...")

	if c.cancel != nil {
		c.cancel()
	}

	c.SetRunning(false)
	logger.InfoC("wecom_aibot", "WeCom AI Bot channel stopped")
	return nil
}

// Send delivers the agent reply into the active streamTask for msg.ChatID.
// It writes into the earliest unfinished task in the queue (FIFO per chatID).
// If the stream has already closed (deadline passed), it posts directly to response_url.
func (c *WeComAIBotChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	c.taskMu.Lock()
	queue := c.chatTasks[msg.ChatID]
	// Only compact Finished tasks at the head of the queue.
	// Tasks that are Finished in the middle are NOT removed here: doing a full
	// scan on every Send() call would be O(n) and is unnecessary given that
	// removeTask() always splices the task out of the queue immediately.
	// Any Finished task left stranded in the middle (e.g. due to an unexpected
	// code path) will be collected by cleanupOldTasks.
	for len(queue) > 0 && queue[0].Finished {
		queue = queue[1:]
	}
	c.chatTasks[msg.ChatID] = queue
	var task *streamTask
	var streamClosed bool
	var responseURL string
	if len(queue) > 0 {
		task = queue[0]
		// Read mutable fields while holding c.taskMu to avoid data races.
		streamClosed = task.StreamClosed
		responseURL = task.ResponseURL
	}
	c.taskMu.Unlock()

	if task == nil {
		logger.DebugCF(
			"wecom_aibot",
			"Send: no active task for chat (may have timed out)",
			map[string]any{
				"chat_id": msg.ChatID,
			},
		)
		return nil
	}

	if streamClosed {
		// Stream already ended with a "please wait" notice; send the real reply via response_url.
		// Note: task.StreamID and task.ChatID are immutable, safe to read without a lock.
		logger.InfoCF("wecom_aibot", "Sending reply via response_url", map[string]any{
			"stream_id": task.StreamID,
			"chat_id":   msg.ChatID,
		})
		if responseURL != "" {
			if err := c.sendViaResponseURL(responseURL, msg.Content); err != nil {
				logger.ErrorCF("wecom_aibot", "Failed to send via response_url", map[string]any{
					"error":     err,
					"stream_id": task.StreamID,
				})
				c.removeTask(task)
				return fmt.Errorf("response_url delivery failed: %w", channels.ErrSendFailed)
			}
		} else {
			logger.WarnCF("wecom_aibot", "Stream closed but no response_url available", map[string]any{
				"stream_id": task.StreamID,
			})
		}
		c.removeTask(task)
		return nil
	}

	// Stream still open: deliver via answerCh for the next poll response.
	select {
	case task.answerCh <- msg.Content:
	case <-task.ctx.Done():
		// Task was canceled (cleanup removed it); silently drop the reply.
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// WebhookPath returns the path for registering on the shared HTTP server
func (c *WeComAIBotChannel) WebhookPath() string {
	if c.config.WebhookPath == "" {
		return "/webhook/wecom-aibot"
	}
	return c.config.WebhookPath
}

// ServeHTTP implements http.Handler for the shared HTTP server
func (c *WeComAIBotChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.handleWebhook(w, r)
}

// HealthPath returns the health check endpoint path
func (c *WeComAIBotChannel) HealthPath() string {
	return c.WebhookPath() + "/health"
}

// HealthHandler handles health check requests
func (c *WeComAIBotChannel) HealthHandler(w http.ResponseWriter, r *http.Request) {
	c.handleHealth(w, r)
}

// handleWebhook handles incoming webhook requests from WeCom AI Bot
func (c *WeComAIBotChannel) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Log all incoming requests for debugging
	logger.DebugCF("wecom_aibot", "Received webhook request", map[string]any{
		"method": r.Method,
		"path":   r.URL.Path,
		"query":  r.URL.RawQuery,
	})

	switch r.Method {
	case http.MethodGet:
		// URL verification
		c.handleVerification(ctx, w, r)
	case http.MethodPost:
		// Message callback
		c.handleMessageCallback(ctx, w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleVerification handles the URL verification request from WeCom
func (c *WeComAIBotChannel) handleVerification(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	msgSignature := r.URL.Query().Get("msg_signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")
	echostr := r.URL.Query().Get("echostr")

	logger.DebugCF("wecom_aibot", "URL verification request", map[string]any{
		"msg_signature": msgSignature,
		"timestamp":     timestamp,
		"nonce":         nonce,
	})

	// Verify signature
	if !verifySignature(c.config.Token, msgSignature, timestamp, nonce, echostr) {
		logger.ErrorC("wecom_aibot", "Signature verification failed")
		http.Error(w, "Signature verification failed", http.StatusUnauthorized)
		return
	}

	// Decrypt echostr
	// For WeCom AI Bot (智能机器人), receiveid should be empty string
	decrypted, err := decryptMessageWithVerify(echostr, c.config.EncodingAESKey, "")
	if err != nil {
		logger.ErrorCF("wecom_aibot", "Failed to decrypt echostr", map[string]any{
			"error": err,
		})
		http.Error(w, "Decryption failed", http.StatusInternalServerError)
		return
	}

	// Remove BOM and whitespace as per WeCom documentation
	decrypted = strings.TrimPrefix(decrypted, "\ufeff")
	decrypted = strings.TrimSpace(decrypted)

	logger.InfoC("wecom_aibot", "URL verification successful")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(decrypted))
}

// handleMessageCallback handles incoming messages from WeCom AI Bot
func (c *WeComAIBotChannel) handleMessageCallback(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	msgSignature := r.URL.Query().Get("msg_signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")

	// Read request body (limit to 4 MB to prevent memory exhaustion)
	const maxBodySize = 4 << 20 // 4 MB
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize+1))
	if err != nil {
		logger.ErrorCF("wecom_aibot", "Failed to read request body", map[string]any{
			"error": err,
		})
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	if len(body) > maxBodySize {
		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Parse JSON body to get encrypted message
	// Format: {"encrypt": "base64_encrypted_string"}
	var encryptedMsg struct {
		Encrypt string `json:"encrypt"`
	}
	if unmarshalErr := json.Unmarshal(body, &encryptedMsg); unmarshalErr != nil {
		logger.ErrorCF("wecom_aibot", "Failed to parse JSON body", map[string]any{
			"error": unmarshalErr,
			"body":  string(body),
		})
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	// Verify signature
	if !verifySignature(c.config.Token, msgSignature, timestamp, nonce, encryptedMsg.Encrypt) {
		logger.ErrorC("wecom_aibot", "Signature verification failed")
		http.Error(w, "Signature verification failed", http.StatusUnauthorized)
		return
	}

	// Decrypt message
	// For WeCom AI Bot (智能机器人), receiveid is empty string
	decrypted, err := decryptMessageWithVerify(encryptedMsg.Encrypt, c.config.EncodingAESKey, "")
	if err != nil {
		logger.ErrorCF("wecom_aibot", "Failed to decrypt message", map[string]any{
			"error": err,
		})
		http.Error(w, "Decryption failed", http.StatusInternalServerError)
		return
	}

	// Parse decrypted JSON message
	var msg WeComAIBotMessage
	if unmarshalErr := json.Unmarshal([]byte(decrypted), &msg); unmarshalErr != nil {
		logger.ErrorCF("wecom_aibot", "Failed to parse decrypted JSON", map[string]any{
			"error":     unmarshalErr,
			"decrypted": decrypted,
		})
		http.Error(w, "Failed to parse message", http.StatusInternalServerError)
		return
	}

	logger.DebugCF("wecom_aibot", "Decrypted message", map[string]any{
		"msgtype": msg.MsgType,
	})

	// Process the message and get streaming response
	response := c.processMessage(ctx, msg, timestamp, nonce)

	// Check if response is empty (e.g. due to unsupported message type)
	if response == "" {
		response = c.encryptEmptyResponse(timestamp, nonce)
	}

	// Return encrypted JSON response
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

// processMessage processes the received message and returns encrypted response
func (c *WeComAIBotChannel) processMessage(
	ctx context.Context,
	msg WeComAIBotMessage,
	timestamp, nonce string,
) string {
	logger.DebugCF("wecom_aibot", "Processing message", map[string]any{
		"msgtype":   msg.MsgType,
		"msgid":     msg.MsgID,
		"from":      msg.From.UserID,
		"chatid":    msg.ChatID,
		"chattype":  msg.ChatType,
		"has_image": msg.Image != nil,
		"has_mixed": msg.Mixed != nil,
		"image_url": func() string {
			if msg.Image != nil {
				return msg.Image.URL
			}
			return ""
		}(),
		"text": func() string {
			if msg.Text != nil {
				return utils.Truncate(msg.Text.Content, 100)
			}
			return ""
		}(),
	})

	switch msg.MsgType {
	case "text":
		return c.handleTextMessage(ctx, msg, timestamp, nonce)
	case "stream":
		return c.handleStreamMessage(ctx, msg, timestamp, nonce)
	case "image":
		return c.handleImageMessage(ctx, msg, timestamp, nonce)
	case "mixed":
		return c.handleMixedMessage(ctx, msg, timestamp, nonce)
	case "event":
		return c.handleEventMessage(ctx, msg, timestamp, nonce)
	default:
		logger.WarnCF("wecom_aibot", "Unsupported message type", map[string]any{
			"msgtype": msg.MsgType,
		})
		return c.encryptResponse("", timestamp, nonce, WeComAIBotStreamResponse{
			MsgType: "stream",
			Stream: WeComAIBotStreamInfo{
				ID:      c.generateStreamID(),
				Finish:  true,
				Content: "Unsupported message type: " + msg.MsgType,
			},
		})
	}
}

// handleTextMessage handles text messages by starting a new streaming task
func (c *WeComAIBotChannel) handleTextMessage(
	ctx context.Context,
	msg WeComAIBotMessage,
	timestamp, nonce string,
) string {
	if msg.Text == nil {
		logger.ErrorC("wecom_aibot", "text message missing text field")
		return c.encryptEmptyResponse(timestamp, nonce)
	}

	content := msg.Text.Content
	userID := msg.From.UserID
	if userID == "" {
		userID = "unknown"
	}

	// chatID: group chat uses chatid, single chat uses userid
	chatID := msg.ChatID
	if chatID == "" {
		chatID = userID
	}

	streamID := c.generateStreamID()

	// WeCom stops sending stream-refresh callbacks after 6 minutes.
	// Set a slightly shorter deadline so we can send a timeout notice before it gives up.
	deadline := time.Now().Add(30 * time.Second)

	// Each task gets its own context derived from the channel lifetime context.
	// Canceling taskCancel interrupts the agent goroutine when the task is removed.
	taskCtx, taskCancel := context.WithCancel(c.ctx)

	task := &streamTask{
		StreamID:    streamID,
		ChatID:      chatID,
		ResponseURL: msg.ResponseURL,
		Question:    content,
		CreatedTime: time.Now(),
		Deadline:    deadline,
		Finished:    false,
		answerCh:    make(chan string, 1),
		ctx:         taskCtx,
		cancel:      taskCancel,
	}

	c.taskMu.Lock()
	c.streamTasks[streamID] = task
	c.chatTasks[chatID] = append(c.chatTasks[chatID], task)
	c.taskMu.Unlock()

	// Publish to agent asynchronously; agent will call Send() with reply.
	// Use task.ctx (not c.ctx) so the agent goroutine is canceled when the task is removed.
	go func() {
		sender := bus.SenderInfo{
			Platform:    "wecom_aibot",
			PlatformID:  userID,
			CanonicalID: identity.BuildCanonicalID("wecom_aibot", userID),
			DisplayName: userID,
		}
		peerKind := "direct"
		if msg.ChatType == "group" {
			peerKind = "group"
		}
		peer := bus.Peer{Kind: peerKind, ID: chatID}
		metadata := map[string]string{
			"channel":      "wecom_aibot",
			"chat_type":    msg.ChatType,
			"msg_type":     "text",
			"msgid":        msg.MsgID,
			"aibotid":      msg.AIBotID,
			"stream_id":    streamID,
			"response_url": msg.ResponseURL,
		}
		c.HandleMessage(task.ctx, peer, msg.MsgID, userID, chatID,
			content, nil, metadata, sender)
	}()

	// Return first streaming response immediately (finish=false, content empty)
	return c.getStreamResponse(task, timestamp, nonce)
}

// handleStreamMessage handles stream polling requests
func (c *WeComAIBotChannel) handleStreamMessage(
	ctx context.Context,
	msg WeComAIBotMessage,
	timestamp, nonce string,
) string {
	if msg.Stream == nil {
		logger.ErrorC("wecom_aibot", "Stream message missing stream field")
		return c.encryptEmptyResponse(timestamp, nonce)
	}

	streamID := msg.Stream.ID

	c.taskMu.RLock()
	task, exists := c.streamTasks[streamID]
	c.taskMu.RUnlock()

	if !exists {
		logger.DebugCF(
			"wecom_aibot",
			"Stream task not found (may be from previous session)",
			map[string]any{
				"stream_id": streamID,
			},
		)
		return c.encryptResponse(streamID, timestamp, nonce, WeComAIBotStreamResponse{
			MsgType: "stream",
			Stream: WeComAIBotStreamInfo{
				ID:      streamID,
				Finish:  true,
				Content: "Task not found or already finished. Please resend your message to start a new session.",
			},
		})
	}

	// Get next response
	return c.getStreamResponse(task, timestamp, nonce)
}

// handleImageMessage handles image messages
func (c *WeComAIBotChannel) handleImageMessage(
	ctx context.Context,
	msg WeComAIBotMessage,
	timestamp, nonce string,
) string {
	logger.InfoCF("wecom_aibot", "Handling image message", map[string]any{
		"msgid":     msg.MsgID,
		"user_id":   msg.From.UserID,
		"chat_id":   msg.ChatID,
		"imageURL":  msg.Image.URL,
		"media_id":  msg.Image.MediaID,
	})

	if msg.Image == nil {
		logger.ErrorC("wecom_aibot", "Image message missing image field")
		return c.encryptEmptyResponse(timestamp, nonce)
	}

	imageURL := msg.Image.URL
	mediaID := msg.Image.MediaID
	aesKey := msg.Image.AESKey // Use the per-message AESKey for decryption

	// If no per-message AESKey, fall back to config EncodingAESKey
	if aesKey == "" && c.config.EncodingAESKey != "" {
		aesKey = c.config.EncodingAESKey
		logger.InfoCF("wecom_aibot", "Using config EncodingAESKey as fallback", map[string]any{
			"msg_id": msg.MsgID,
		})
	}

	userID := msg.From.UserID
	if userID == "" {
		userID = "unknown"
	}

	// chatID: group chat uses chatid, single chat uses userid
	chatID := msg.ChatID
	if chatID == "" {
		chatID = userID
	}

	logger.InfoCF("wecom_aibot", "Image message with AESKey", map[string]any{
		"chat_id":  chatID,
		"imageURL": imageURL,
		"aes_key":  aesKey,
		"msg_id":   msg.MsgID,
		"user_id":  userID,
	})

	// Download image and store in media store
	var mediaRefs []string
	store := c.GetMediaStore()
	if store == nil {
		logger.WarnCF("wecom_aibot", "MediaStore not available, image will not be processed", map[string]any{
			"user_id": userID,
			"chat_id": chatID,
			"msg_id":  msg.MsgID,
		})
	} else {
		var ref string
		scope := channels.BuildMediaScope("wecom_aibot", chatID, msg.MsgID)

		// Try to download via media_id first (WeCom AI Bot callback directly provides media_id)
		if mediaID != "" {
			logger.InfoCF("wecom_aibot", "Trying to download image via media_id", map[string]any{
				"media_id": mediaID,
				"msg_id":   msg.MsgID,
			})
			ref = c.downloadMediaFromMediaIDNoAuth(ctx, mediaID, "image", store, scope, msg.MsgID)
		}

		// Fallback to URL if media_id download failed or not available
		if ref == "" && imageURL != "" {
			logger.InfoCF("wecom_aibot", "Falling back to URL download", map[string]any{
				"url":     imageURL,
				"msg_id":  msg.MsgID,
				"aes_key": aesKey,
			})
			ref = c.downloadMediaFromURL(ctx, imageURL, "image.jpg", aesKey, store, scope, msg.MsgID)
		}

		if ref != "" {
			mediaRefs = append(mediaRefs, ref)
			logger.DebugCF("wecom_aibot", "Image downloaded successfully", map[string]any{
				"user_id":      userID,
				"chat_id":     chatID,
				"msg_id":      msg.MsgID,
				"media_count": len(mediaRefs),
			})
		} else {
			logger.WarnCF("wecom_aibot", "Image download failed", map[string]any{
				"user_id":  userID,
				"chat_id":  chatID,
				"msg_id":   msg.MsgID,
				"url":      imageURL,
				"media_id": mediaID,
			})
		}
	}

	// Build content with image tag
	content := "[image]"

	streamID := c.generateStreamID()
	deadline := time.Now().Add(30 * time.Second)

	taskCtx, taskCancel := context.WithCancel(c.ctx)

	task := &streamTask{
		StreamID:    streamID,
		ChatID:      chatID,
		ResponseURL: msg.ResponseURL,
		Question:    content,
		CreatedTime: time.Now(),
		Deadline:    deadline,
		Finished:    false,
		answerCh:    make(chan string, 1),
		ctx:         taskCtx,
		cancel:      taskCancel,
	}

	c.taskMu.Lock()
	c.streamTasks[streamID] = task
	c.chatTasks[chatID] = append(c.chatTasks[chatID], task)
	c.taskMu.Unlock()

	// Publish to agent asynchronously with media refs
	go func() {
		sender := bus.SenderInfo{
			Platform:    "wecom_aibot",
			PlatformID:  userID,
			CanonicalID: identity.BuildCanonicalID("wecom_aibot", userID),
			DisplayName: userID,
		}
		peerKind := "direct"
		if msg.ChatType == "group" {
			peerKind = "group"
		}
		peer := bus.Peer{Kind: peerKind, ID: chatID}
		metadata := map[string]string{
			"channel":      "wecom_aibot",
			"chat_type":    msg.ChatType,
			"msg_type":     "image",
			"msgid":        msg.MsgID,
			"aibotid":      msg.AIBotID,
			"stream_id":    streamID,
			"response_url": msg.ResponseURL,
		}
		logger.InfoCF("wecom_aibot", "Calling HandleMessage with media", map[string]any{
			"msgid":     msg.MsgID,
			"content":   content,
			"mediaRefs": mediaRefs,
		})
		c.HandleMessage(task.ctx, peer, msg.MsgID, userID, chatID,
			content, mediaRefs, metadata, sender)
	}()

	// Return first streaming response immediately
	return c.getStreamResponse(task, timestamp, nonce)
}

// handleMixedMessage handles mixed (text + image) messages
func (c *WeComAIBotChannel) handleMixedMessage(
	ctx context.Context,
	msg WeComAIBotMessage,
	timestamp, nonce string,
) string {
	if msg.Mixed == nil || len(msg.Mixed.MsgItem) == 0 {
		logger.ErrorC("wecom_aibot", "Mixed message missing msg_item field")
		return c.encryptEmptyResponse(timestamp, nonce)
	}

	userID := msg.From.UserID
	if userID == "" {
		userID = "unknown"
	}

	// chatID: group chat uses chatid, single chat uses userid
	chatID := msg.ChatID
	if chatID == "" {
		chatID = userID
	}

	// Process text and image items
	var content string
	var mediaRefs []string
	scope := channels.BuildMediaScope("wecom_aibot", chatID, msg.MsgID)

	store := c.GetMediaStore()
	if store == nil {
		logger.WarnCF("wecom_aibot", "MediaStore not available, mixed message images will not be processed", map[string]any{
			"user_id": userID,
			"chat_id": chatID,
			"msg_id":  msg.MsgID,
		})
		// Still process text content even without media store
		for _, item := range msg.Mixed.MsgItem {
			if item.MsgType == "text" && item.Text != nil && item.Text.Content != "" {
				content += item.Text.Content + "\n"
			}
		}
	} else {
		for _, item := range msg.Mixed.MsgItem {
			switch item.MsgType {
			case "text":
				if item.Text != nil && item.Text.Content != "" {
					content += item.Text.Content + "\n"
				}
			case "image":
				if item.Image != nil && item.Image.URL != "" {
					aesKey := item.Image.AESKey
					// If no per-message AESKey, fall back to config EncodingAESKey
					if aesKey == "" && c.config.EncodingAESKey != "" {
						aesKey = c.config.EncodingAESKey
					}
					ref := c.downloadMediaFromURL(ctx, item.Image.URL, "image.jpg", aesKey, store, scope, msg.MsgID)
					if ref != "" {
						mediaRefs = append(mediaRefs, ref)
						logger.DebugCF("wecom_aibot", "Mixed message image downloaded", map[string]any{
							"user_id":  userID,
							"chat_id":  chatID,
							"msg_id":   msg.MsgID,
							"media_ref": ref,
						})
					} else {
						logger.WarnCF("wecom_aibot", "Mixed message image download failed", map[string]any{
							"user_id": userID,
							"chat_id": chatID,
							"msg_id":  msg.MsgID,
							"url":     item.Image.URL,
						})
					}
				}
			}
		}
	}

	// If no content from text, use image placeholder
	if content == "" {
		if len(mediaRefs) > 0 {
			content = "[image]"
		} else {
			content = "[mixed message]"
		}
	}

	if len(mediaRefs) > 0 {
		logger.DebugCF("wecom_aibot", "Mixed message processed", map[string]any{
			"user_id":     userID,
			"chat_id":     chatID,
			"msg_id":      msg.MsgID,
			"media_count": len(mediaRefs),
		})
	}

	streamID := c.generateStreamID()
	deadline := time.Now().Add(30 * time.Second)

	taskCtx, taskCancel := context.WithCancel(c.ctx)

	task := &streamTask{
		StreamID:    streamID,
		ChatID:      chatID,
		ResponseURL: msg.ResponseURL,
		Question:    content,
		CreatedTime: time.Now(),
		Deadline:    deadline,
		Finished:    false,
		answerCh:    make(chan string, 1),
		ctx:         taskCtx,
		cancel:      taskCancel,
	}

	c.taskMu.Lock()
	c.streamTasks[streamID] = task
	c.chatTasks[chatID] = append(c.chatTasks[chatID], task)
	c.taskMu.Unlock()

	// Publish to agent asynchronously with media refs
	go func() {
		sender := bus.SenderInfo{
			Platform:    "wecom_aibot",
			PlatformID:  userID,
			CanonicalID: identity.BuildCanonicalID("wecom_aibot", userID),
			DisplayName: userID,
		}
		peerKind := "direct"
		if msg.ChatType == "group" {
			peerKind = "group"
		}
		peer := bus.Peer{Kind: peerKind, ID: chatID}
		metadata := map[string]string{
			"channel":      "wecom_aibot",
			"chat_type":    msg.ChatType,
			"msg_type":     "mixed",
			"msgid":        msg.MsgID,
			"aibotid":      msg.AIBotID,
			"stream_id":    streamID,
			"response_url": msg.ResponseURL,
		}
		c.HandleMessage(task.ctx, peer, msg.MsgID, userID, chatID,
			content, mediaRefs, metadata, sender)
	}()

	// Return first streaming response immediately
	return c.getStreamResponse(task, timestamp, nonce)
}

// handleEventMessage handles event messages
func (c *WeComAIBotChannel) handleEventMessage(
	ctx context.Context,
	msg WeComAIBotMessage,
	timestamp, nonce string,
) string {
	eventType := ""
	if msg.Event != nil {
		eventType = msg.Event.EventType
	}
	logger.DebugCF("wecom_aibot", "Received event", map[string]any{
		"event_type": eventType,
	})

	// Send welcome message when user opens the chat window
	if eventType == "enter_chat" && c.config.WelcomeMessage != "" {
		streamID := c.generateStreamID()
		return c.encryptResponse(streamID, timestamp, nonce, WeComAIBotStreamResponse{
			MsgType: "stream",
			Stream: WeComAIBotStreamInfo{
				ID:      streamID,
				Finish:  true,
				Content: c.config.WelcomeMessage,
			},
		})
	}

	return c.encryptEmptyResponse(timestamp, nonce)
}

// getStreamResponse gets the next streaming response for a task.
// - If agent replied: return finish=true with the real answer.
// - If deadline passed: return finish=true with a "please wait" notice, keep task alive for response_url.
// - Otherwise: return finish=false (empty), client will poll again.
func (c *WeComAIBotChannel) getStreamResponse(task *streamTask, timestamp, nonce string) string {
	var content string
	var finish bool
	var closeStreamOnly bool // close stream but do NOT remove task (response_url still pending)

	select {
	case answer := <-task.answerCh:
		// Agent replied before deadline — normal finish.
		content = answer
		finish = true
	default:
		if time.Now().After(task.Deadline) {
			// Deadline reached: close the stream with a notice, then wait for agent via response_url.
			content = c.config.ProcessingMessage
			finish = true
			closeStreamOnly = true
			logger.InfoCF(
				"wecom_aibot",
				"Stream deadline reached, switching to response_url mode",
				map[string]any{
					"stream_id":    task.StreamID,
					"chat_id":      task.ChatID,
					"response_url": task.ResponseURL != "",
				},
			)
		}
		// else: still waiting, return finish=false
	}

	if finish && !closeStreamOnly {
		// Normal finish: remove from all maps.
		c.removeTask(task)
	} else if closeStreamOnly {
		// Mark stream as closed and remove from streamTasks under a single lock
		// to keep StreamClosed/StreamClosedAt consistent with map membership.
		c.taskMu.Lock()
		task.StreamClosed = true
		task.StreamClosedAt = time.Now()
		delete(c.streamTasks, task.StreamID)
		c.taskMu.Unlock()
	}

	response := WeComAIBotStreamResponse{
		MsgType: "stream",
		Stream: WeComAIBotStreamInfo{
			ID:      task.StreamID,
			Finish:  finish,
			Content: content,
		},
	}

	return c.encryptResponse(task.StreamID, timestamp, nonce, response)
}

// removeTask removes a task from both streamTasks and chatTasks, marks it finished,
// and cancels its context to interrupt the associated agent goroutine.
func (c *WeComAIBotChannel) removeTask(task *streamTask) {
	// Cancel first so the agent goroutine stops as soon as possible,
	// before we acquire the write lock.
	task.cancel()

	c.taskMu.Lock()
	task.Finished = true // written under c.taskMu, consistent with all readers
	delete(c.streamTasks, task.StreamID)
	queue := c.chatTasks[task.ChatID]
	for i, t := range queue {
		if t == task {
			c.chatTasks[task.ChatID] = append(queue[:i], queue[i+1:]...)
			break
		}
	}
	if len(c.chatTasks[task.ChatID]) == 0 {
		delete(c.chatTasks, task.ChatID)
	}
	c.taskMu.Unlock()
}

// sendViaResponseURL posts a markdown reply to the WeCom response_url.
// response_url is valid for 1 hour and can only be used once per callback.
// Returned errors are wrapped with channels.ErrRateLimit, channels.ErrTemporary,
// or channels.ErrSendFailed so the manager can apply the right retry policy.
func (c *WeComAIBotChannel) sendViaResponseURL(responseURL, content string) error {
	payload := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, responseURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := responseURLHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("post to response_url failed: %w: %w", channels.ErrTemporary, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	const maxErrBody = 64 << 10 // 64 KB is more than enough for any error response
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxErrBody))
	if err != nil {
		return fmt.Errorf("reading response_url body: %w: %w", channels.ErrTemporary, err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return fmt.Errorf("response_url rate limited (%d): %s: %w",
			resp.StatusCode, respBody, channels.ErrRateLimit)
	case resp.StatusCode >= 500:
		return fmt.Errorf("response_url server error (%d): %s: %w",
			resp.StatusCode, respBody, channels.ErrTemporary)
	default:
		return fmt.Errorf("response_url returned %d: %s: %w",
			resp.StatusCode, respBody, channels.ErrSendFailed)
	}
}

// encryptResponse encrypts a streaming response
func (c *WeComAIBotChannel) encryptResponse(
	streamID, timestamp, nonce string,
	response WeComAIBotStreamResponse,
) string {
	// Marshal response to JSON
	plaintext, err := json.Marshal(response)
	if err != nil {
		logger.ErrorCF("wecom_aibot", "Failed to marshal response", map[string]any{
			"error": err,
		})
		return ""
	}

	logger.DebugCF("wecom_aibot", "Encrypting response", map[string]any{
		"stream_id": streamID,
		"finish":    response.Stream.Finish,
		"preview":   utils.Truncate(response.Stream.Content, 100),
	})

	// Encrypt message
	encrypted, err := c.encryptMessage(string(plaintext), "")
	if err != nil {
		logger.ErrorCF("wecom_aibot", "Failed to encrypt message", map[string]any{
			"error": err,
		})
		return ""
	}

	// Generate signature
	signature := computeSignature(c.config.Token, timestamp, nonce, encrypted)

	// Build encrypted response
	encryptedResp := WeComAIBotEncryptedResponse{
		Encrypt:      encrypted,
		MsgSignature: signature,
		Timestamp:    timestamp,
		Nonce:        nonce,
	}

	respJSON, err := json.Marshal(encryptedResp)
	if err != nil {
		logger.ErrorCF("wecom_aibot", "Failed to marshal encrypted response", map[string]any{
			"error": err,
		})
		return ""
	}

	logger.DebugCF("wecom_aibot", "Response encrypted", map[string]any{
		"stream_id": streamID,
	})

	return string(respJSON)
}

// encryptEmptyResponse returns a minimal valid encrypted response
func (c *WeComAIBotChannel) encryptEmptyResponse(timestamp, nonce string) string {
	// Construct a zero-value stream response and encrypt it so that
	// WeCom always receives a syntactically valid encrypted JSON object.
	emptyResp := WeComAIBotStreamResponse{}
	return c.encryptResponse("", timestamp, nonce, emptyResp)
}

// encryptMessage encrypts a plain text message for WeCom AI Bot
func (c *WeComAIBotChannel) encryptMessage(plaintext, receiveid string) (string, error) {
	aesKey, err := decodeWeComAESKey(c.config.EncodingAESKey)
	if err != nil {
		return "", err
	}

	frame, err := packWeComFrame(plaintext, receiveid)
	if err != nil {
		return "", err
	}

	// PKCS7 padding then AES-CBC encrypt
	paddedFrame := pkcs7Pad(frame, blockSize)
	ciphertext, err := encryptAESCBC(aesKey, paddedFrame)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// func (c *WeComAIBotChannel) downloadAndDecryptImage(
// 	ctx context.Context,
// 	imageURL string,
// ) ([]byte, error) {
// 	// Download image
// 	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create request: %w", err)
// 	}

// 	client := &http.Client{
// 		Timeout: 15 * time.Second,
// 	}

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to download image: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
// 	}

// 	// Limit image download to 20 MB to prevent memory exhaustion
// 	const maxImageSize = 20 << 20 // 20 MB
// 	encryptedData, err := io.ReadAll(io.LimitReader(resp.Body, maxImageSize+1))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read image data: %w", err)
// 	}
// 	if len(encryptedData) > maxImageSize {
// 		return nil, fmt.Errorf("image too large (exceeds %d MB)", maxImageSize>>20)
// 	}

// 	logger.DebugCF("wecom_aibot", "Image downloaded", map[string]any{
// 		"size": len(encryptedData),
// 	})

// 	// Decode AES key
// 	aesKey, err := decodeWeComAESKey(c.config.EncodingAESKey)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Decrypt image (AES-CBC with IV = first 16 bytes of key, PKCS7 padding stripped)
// 	decryptedData, err := decryptAESCBC(aesKey, encryptedData)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decrypt image: %w", err)
// 	}

// 	logger.DebugCF("wecom_aibot", "Image decrypted", map[string]any{
// 		"size": len(decryptedData),
// 	})

// 	return decryptedData, nil
// }

// generateRandomID generates a cryptographically random alphanumeric ID of
// length n.  Used for stream IDs and WebSocket request IDs.
func generateRandomID(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[num.Int64()]
	}
	return string(b)
}

// generateStreamID generates a random 10-character stream ID (webhook mode).
func (c *WeComAIBotChannel) generateStreamID() string {
	return generateRandomID(10)
}

// cleanupLoop periodically cleans up old streaming tasks
func (c *WeComAIBotChannel) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupOldTasks()
		case <-c.ctx.Done():
			return
		}
	}
}

// cleanupOldTasks removes tasks that have exceeded their expected lifetime:
//   - Active tasks (in streamTasks): cleaned up after 1 hour (response_url validity window).
//   - StreamClosed tasks (in chatTasks only): cleaned up after streamClosedGracePeriod.
//     These tasks are waiting for the agent to call Send() via response_url. If the agent
//     crashes or times out without calling Send(), we must not let them accumulate indefinitely.
//     The grace period is generous enough to cover typical LLM latency but far shorter than 1 hour,
//     preventing chatTasks from filling up when many requests time out in quick succession.
const (
	streamClosedGracePeriod = 10 * time.Minute // max wait for agent after stream closes
	taskMaxLifetime         = 1 * time.Hour    // absolute max (≈ response_url validity)
)

func (c *WeComAIBotChannel) cleanupOldTasks() {
	c.taskMu.Lock()
	defer c.taskMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-taskMaxLifetime)
	for id, task := range c.streamTasks {
		if task.CreatedTime.Before(cutoff) {
			delete(c.streamTasks, id)
			task.cancel() // interrupt agent goroutine still waiting for LLM
			queue := c.chatTasks[task.ChatID]
			for i, t := range queue {
				if t == task {
					c.chatTasks[task.ChatID] = append(queue[:i], queue[i+1:]...)
					break
				}
			}
			if len(c.chatTasks[task.ChatID]) == 0 {
				delete(c.chatTasks, task.ChatID)
			}
			logger.DebugCF("wecom_aibot", "Cleaned up expired task", map[string]any{
				"stream_id": id,
			})
		}
	}
	// Clean up StreamClosed tasks from chatTasks.
	// Two expiry conditions are checked:
	//  1. Absolute expiry: task was created more than taskMaxLifetime ago.
	//  2. Grace expiry: stream closed more than streamClosedGracePeriod ago
	//     (agent had enough time to reply; it is not coming back).
	for chatID, queue := range c.chatTasks {
		filtered := queue[:0]
		for i, t := range queue {
			absoluteExpired := t.CreatedTime.Before(cutoff)
			graceExpired := t.StreamClosed &&
				!t.StreamClosedAt.IsZero() &&
				t.StreamClosedAt.Before(now.Add(-streamClosedGracePeriod))
			if t.Finished {
				// Finished tasks should have been removed by removeTask().
				// Finding one here (especially not at position 0) means an
				// unexpected code path left it stranded, causing the queue to
				// grow silently. Log a warning so it is visible, then drop it.
				if i > 0 {
					logger.WarnCF("wecom_aibot",
						"Found stranded Finished task in the middle of chatTasks queue; "+
							"this should not happen — removeTask() should have spliced it out",
						map[string]any{
							"chat_id":   chatID,
							"stream_id": t.StreamID,
							"position":  i,
						})
				}
				// The task is already finished; its context was already canceled
				// by removeTask(), so no further action is required.
				continue
			} else if !absoluteExpired && !graceExpired {
				filtered = append(filtered, t)
			} else {
				t.cancel() // cancel any lingering agent goroutine
			}
		}
		if len(filtered) == 0 {
			delete(c.chatTasks, chatID)
		} else {
			c.chatTasks[chatID] = filtered
		}
	}
}

// handleHealth handles health check requests
func (c *WeComAIBotChannel) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	if !c.IsRunning() {
		status = "not running"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": status,
	})
}

// downloadMediaFromURL downloads media from a URL and stores it in MediaStore.
func (c *WeComAIBotChannel) downloadMediaFromURL(
	ctx context.Context,
	mediaURL, filename, aesKey string,
	store media.MediaStore,
	scope string,
	msgID string,
) string {
	// Download the media file from the URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		logger.ErrorCF("wecom_aibot", "Failed to create media download request", map[string]any{
			"error": err.Error(),
			"url":    mediaURL,
		})
		return ""
	}

	// Minimal headers - WeCom COS signature may be sensitive
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Use default client - no custom redirects
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.ErrorCF("wecom_aibot", "Failed to download media", map[string]any{
			"error": err.Error(),
			"url":  mediaURL,
		})
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF("wecom_aibot", "Media download failed with status", map[string]any{
			"status": resp.StatusCode,
			"url":    mediaURL,
		})
		return ""
	}

	// Read the media body first
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorCF("wecom_aibot", "Failed to read media body", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	// Try to decrypt the encrypted file content using the per-message AESKey
	// WeCom AI Bot image URLs return encrypted data
	var decryptedBody []byte
	if aesKey != "" {
		decryptedBody, err = decryptMediaContentWithKey(body, aesKey)
		if err != nil {
			logger.WarnCF("wecom_aibot", "Decryption failed, using raw data", map[string]any{
				"error":        err.Error(),
				"data_len":     len(body),
				"first_bytes": fmt.Sprintf("%x", body[:min(32, len(body))]),
			})
			// Fallback to raw data if decryption fails
			decryptedBody = body
		}
	} else {
		logger.InfoCF("wecom_aibot", "No AESKey provided, using raw data", map[string]any{})
		decryptedBody = body
	}

	// Debug: Log first few bytes to diagnose issues
	firstBytesLen := 20
	if len(decryptedBody) < firstBytesLen {
		firstBytesLen = len(decryptedBody)
	}

	logger.InfoCF("wecom_aibot", "Media download response", map[string]any{
		"status":       resp.StatusCode,
		"content_type": resp.Header.Get("Content-Type"),
		"content_len":  len(decryptedBody),
		"first_bytes":  fmt.Sprintf("%x", decryptedBody[:firstBytesLen]),
	})

	// Determine actual file type from content (magic bytes)
	contentType := resp.Header.Get("Content-Type")
	if kind, err := filetype.Match(decryptedBody); err == nil && kind != filetype.Unknown {
		// Use the detected type from content
		contentType = kind.MIME.Value
		filename = "image." + kind.Extension
		logger.DebugCF("wecom_aibot", "Filetype detected from content", map[string]any{
			"detected_type": kind.MIME.Value,
			"extension":     kind.Extension,
			"size":          len(decryptedBody),
		})
	} else {
		// Filetype detection failed, try to determine from header or extension
		logger.WarnCF("wecom_aibot", "Filetype detection failed, using extension inference", map[string]any{
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
	mediaDir := filepath.Join(tempDir, "picoclaw_media", "wecom_aibot")
	if mkdirErr := os.MkdirAll(mediaDir, 0o755); mkdirErr != nil {
		logger.ErrorCF("wecom_aibot", "Failed to create media directory", map[string]any{
			"error": mkdirErr.Error(),
		})
		return ""
	}

	// Create a unique filename to avoid collisions
	uniqueFilename := fmt.Sprintf("%s-%d-%s", msgID, time.Now().Unix(), filename)
	localPath := filepath.Join(mediaDir, uniqueFilename)

	// Write the file
	err = os.WriteFile(localPath, decryptedBody, 0o644)
	if err != nil {
		logger.ErrorCF("wecom_aibot", "Failed to write media file", map[string]any{
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
			logger.ErrorCF("wecom_aibot", "Downloaded file is not a valid image", map[string]any{
				"path":   localPath,
				"error":  decodeErr.Error(),
				"size":   len(decryptedBody),
				"header": fmt.Sprintf("%x", decryptedBody[:min(20, len(decryptedBody))]),
			})
			// Remove invalid file
			os.Remove(localPath)
			return ""
		}
	} else {
		logger.WarnCF("wecom_aibot", "Failed to open file for verification", map[string]any{
			"error": err.Error(),
			"path":  localPath,
		})
	}

	// Store in media store
	ref, err := store.Store(localPath, media.MediaMeta{
		Filename:    filename,
		ContentType: contentType,
		Source:      "wecom_aibot",
	}, scope)
	if err != nil {
		logger.ErrorCF("wecom_aibot", "Failed to store media", map[string]any{
			"error": err.Error(),
			"path":  localPath,
		})
		os.Remove(localPath)
		return ""
	}

	logger.DebugCF("wecom_aibot", "Media downloaded successfully", map[string]any{
		"url":          mediaURL,
		"path":         localPath,
		"size":         len(decryptedBody),
		"content_type": contentType,
		"fileref":      ref,
	})

	return ref
}

// decryptMediaContent decrypts the encrypted media content using EncodingAESKey
// WeCom encryption format: 16-byte random pad + 4-byte length + plaintext + appid
func (c *WeComAIBotChannel) decryptMediaContent(encryptedData []byte) ([]byte, error) {
	if c.config.EncodingAESKey == "" {
		return nil, fmt.Errorf("encoding_aes_key not configured")
	}

	logger.DebugCF("wecom_aibot", "Starting decryption", map[string]any{
		"encrypted_len": len(encryptedData),
		"first_bytes":   fmt.Sprintf("%x", encryptedData[:min(32, len(encryptedData))]),
	})

	// EncodingAESKey is base64 encoded with padding
	// Add padding if needed (43 chars + "=" = 44 chars)
	keyStr := c.config.EncodingAESKey
	if len(keyStr)%4 != 0 {
		keyStr += strings.Repeat("=", 4-len(keyStr)%4)
	}

	// Decode AES key
	keyBytes, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encoding_aes_key: %w", err)
	}
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("invalid encoding_aes_key length after decoding: %d, expected 32", len(keyBytes))
	}

	logger.DebugCF("wecom_aibot", "AES key decoded", map[string]any{
		"key_len": len(keyBytes),
		"iv":      fmt.Sprintf("%x", keyBytes[:16]),
	})

	// AES-256-CBC decryption
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	iv := keyBytes[:16] // IV is first 16 bytes of key

	if len(encryptedData) < aes.BlockSize {
		return nil, fmt.Errorf("encrypted data too short")
	}

	// CBC decrypt
	decrypted := make([]byte, len(encryptedData))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decrypted, encryptedData)

	logger.DebugCF("wecom_aibot", "After CBC decryption", map[string]any{
		"decrypted_len":   len(decrypted),
		"decrypted_first": fmt.Sprintf("%x", decrypted[:min(32, len(decrypted))]),
	})

	// Remove PKCS7 padding using common function
	decrypted, err = pkcs7Unpad(decrypted)
	if err != nil {
		logger.WarnCF("wecom_aibot", "PKCS7 unpad failed, returning decrypted data anyway", map[string]any{
			"error":          err.Error(),
			"last_byte":     fmt.Sprintf("%x", decrypted[len(decrypted)-1]),
			"decrypted_len": len(decrypted),
		})
		// Return decrypted data even without unpadding - might be valid image data
		return decrypted, nil
	}

	logger.DebugCF("wecom_aibot", "After PKCS7 unpad", map[string]any{
		"unpadded_len":   len(decrypted),
		"unpadded_first": fmt.Sprintf("%x", decrypted[:min(32, len(decrypted))]),
	})

	// WeCom decrypted format: 16-byte random pad + 4-byte length + plaintext + appid
	// Extract the actual plaintext
	if len(decrypted) < 20 {
		return nil, fmt.Errorf("decrypted data invalid, too short: %d", len(decrypted))
	}

	// Extract plaintext length (big-endian 4 bytes)
	plainLen := int(decrypted[16])<<24 | int(decrypted[17])<<16 | int(decrypted[18])<<8 | int(decrypted[19])
	logger.DebugCF("wecom_aibot", "Extracted plaintext info", map[string]any{
		"plain_len":       plainLen,
		"total_len":       len(decrypted),
		"remaining_after": len(decrypted) - 20,
	})

	if plainLen <= 0 || plainLen > len(decrypted)-20 {
		// If plaintext length is invalid, return the whole data as-is
		logger.WarnCF("wecom_aibot", "Plaintext length invalid, returning raw decrypted data", map[string]any{
			"plain_len":  plainLen,
			"total_len":  len(decrypted),
		})
		return decrypted, nil
	}

	// Extract plaintext (skip 16-byte pad + 4-byte length, remove appid at the end)
	plainText := decrypted[20 : 20+plainLen]

	logger.DebugCF("wecom_aibot", "Decryption successful", map[string]any{
		"plaintext_len": len(plainText),
		"plaintext_head": fmt.Sprintf("%x", plainText[:min(32, len(plainText))]),
	})

	return plainText, nil
}

// decryptMediaContentWithKey decrypts media content using the provided AESKey
// This is used for WeCom AI Bot per-message AESKey (not the config EncodingAESKey)
// Reference: wecom-aibot-go decryptFile function
func decryptMediaContentWithKey(encryptedData []byte, aesKey string) ([]byte, error) {
	if len(encryptedData) == 0 {
		return nil, fmt.Errorf("encrypted buffer is empty")
	}
	if aesKey == "" {
		return nil, fmt.Errorf("aesKey cannot be empty")
	}

	// Decode AES key from base64 (support both standard and URL-safe base64)
	// Try URL-safe first, then standard
	key, err := base64.URLEncoding.DecodeString(aesKey)
	if err != nil {
		// Try standard base64 with padding
		keyStr := aesKey
		if len(keyStr)%4 != 0 {
			keyStr += strings.Repeat("=", 4-len(keyStr)%4)
		}
		key, err = base64.StdEncoding.DecodeString(keyStr)
		if err != nil {
			return nil, fmt.Errorf("aesKey base64 decode failed: %w", err)
		}
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("aesKey length error, expected 32 bytes, got %d", len(key))
	}
	if len(encryptedData)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("encrypted buffer length not block aligned")
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cipher init failed: %w", err)
	}

	// IV is first 16 bytes of key
	iv := key[:aes.BlockSize]
	decrypted := make([]byte, len(encryptedData))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decrypted, encryptedData)

	// Remove PKCS#7 padding (32 byte block size)
	trimmed, err := trimPKCS7Padding32(decrypted)
	if err != nil {
		return nil, fmt.Errorf("trim padding failed: %w", err)
	}
	return trimmed, nil
}

// trimPKCS7Padding32 removes PKCS#7 padding with 32-byte block size
func trimPKCS7Padding32(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	padLen := int(data[len(data)-1])
	if padLen < 1 || padLen > 32 || padLen > len(data) {
		return nil, fmt.Errorf("invalid padding value: %d", padLen)
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if int(data[i]) != padLen {
			return nil, fmt.Errorf("padding bytes not consistent")
		}
	}
	return data[:len(data)-padLen], nil
}

// downloadMediaFromMediaIDNoAuth downloads media using media_id without access_token
// This is used for WeCom AI Bot which provides media_id directly in the callback
func (c *WeComAIBotChannel) downloadMediaFromMediaIDNoAuth(
	ctx context.Context,
	mediaID, filename string,
	store media.MediaStore,
	scope string,
	msgID string,
) string {
	// Try different possible download URLs
	// WeCom AI Bot media can be accessed via different endpoints
	urls := []string{
		// Option 1: Direct media download API
		fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/media/get?media_id=%s", url.QueryEscape(mediaID)),
		// Option 2: Alternative endpoint
		fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/media/get?media_id=%s", url.QueryEscape(mediaID)),
	}

	var lastErr error
	for _, apiURL := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			logger.ErrorCF("wecom_aibot", "Failed to create media download request", map[string]any{
				"error":    err.Error(),
				"media_id": mediaID,
				"url":      apiURL,
			})
			lastErr = err
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.ErrorCF("wecom_aibot", "Failed to download media", map[string]any{
				"error":    err.Error(),
				"media_id": mediaID,
				"url":      apiURL,
			})
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logger.WarnCF("wecom_aibot", "Media download failed with status", map[string]any{
				"status":   resp.StatusCode,
				"media_id": mediaID,
				"url":      apiURL,
			})
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}

		// Read the media body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.ErrorCF("wecom_aibot", "Failed to read media body", map[string]any{
				"error":    err.Error(),
				"media_id": mediaID,
			})
			lastErr = err
			continue
		}

		// Check if response is an error JSON (not the actual media)
		if strings.HasPrefix(string(body), "{\"errcode\"") {
			var errResp struct {
				ErrCode int    `json:"errcode"`
				ErrMsg  string `json:"errmsg"`
			}
			if json.Unmarshal(body, &errResp) == nil {
				logger.WarnCF("wecom_aibot", "Media download API error", map[string]any{
					"errcode":  errResp.ErrCode,
					"errmsg":   errResp.ErrMsg,
					"media_id": mediaID,
					"url":      apiURL,
				})
				lastErr = fmt.Errorf("API error: %s", errResp.ErrMsg)
				continue
			}
		}

		// Determine content type from response header
		contentType := resp.Header.Get("Content-Type")
		if contentType == "" || contentType == "application/octet-stream" {
			// Try to detect from file content
			if kind, err := filetype.Match(body); err == nil && kind != filetype.Unknown {
				contentType = kind.MIME.Value
			}
		}

		// Generate temp file path
		tempDir := os.TempDir()
		mediaDir := filepath.Join(tempDir, "picoclaw_media", "wecom_aibot")
		if mkdirErr := os.MkdirAll(mediaDir, 0o755); mkdirErr != nil {
			logger.ErrorCF("wecom_aibot", "Failed to create media directory", map[string]any{
				"error": mkdirErr.Error(),
			})
			lastErr = fmt.Errorf("mkdir failed: %w", mkdirErr)
			continue
		}

		// Determine file extension from content type
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

		// Create a unique filename
		uniqueFilename := fmt.Sprintf("%s-%d-%s%s", msgID, time.Now().Unix(), filename, ext)
		localPath := filepath.Join(mediaDir, uniqueFilename)

		// Write the file
		err = os.WriteFile(localPath, body, 0o644)
		if err != nil {
			logger.ErrorCF("wecom_aibot", "Failed to write media file", map[string]any{
				"error": err.Error(),
				"path":  localPath,
			})
			lastErr = err
			continue
		}

		// Verify the image is valid by decoding it
		f, err := os.Open(localPath)
		if err == nil {
			_, _, decodeErr := image.Decode(f)
			f.Close()
			if decodeErr != nil {
				logger.ErrorCF("wecom_aibot", "Downloaded file is not a valid image", map[string]any{
					"path":   localPath,
					"error":  decodeErr.Error(),
					"size":   len(body),
					"header": fmt.Sprintf("%x", body[:min(20, len(body))]),
				})
				os.Remove(localPath)
				lastErr = fmt.Errorf("invalid image: %w", decodeErr)
				continue
			}
		} else {
			logger.WarnCF("wecom_aibot", "Failed to open file for verification", map[string]any{
				"error": err.Error(),
				"path":  localPath,
			})
		}

		// Determine final content type for storage
		if contentType == "" || contentType == "application/octet-stream" {
			contentType = "image/jpeg"
		}

		// Store in media store
		ref, err := store.Store(localPath, media.MediaMeta{
			Filename:    filename + ext,
			ContentType: contentType,
			Source:      "wecom_aibot",
		}, scope)
		if err != nil {
			logger.ErrorCF("wecom_aibot", "Failed to store media", map[string]any{
				"error": err.Error(),
				"path":  localPath,
			})
			os.Remove(localPath)
			return ""
		}

		logger.DebugCF("wecom_aibot", "Media downloaded via media_id successfully", map[string]any{
			"media_id":     mediaID,
			"path":         localPath,
			"size":         len(body),
			"content_type": contentType,
			"fileref":      ref,
			"url_tried":    apiURL,
		})

		return ref
	}

	logger.ErrorCF("wecom_aibot", "All media download attempts failed", map[string]any{
		"media_id": mediaID,
		"last_err": lastErr.Error(),
	})

	return ""
}
