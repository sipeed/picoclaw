package wecom

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

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

// streamTask represents a streaming task for AI Bot
type streamTask struct {
	StreamID       string
	ChatID         string // used by Send() to find this task
	ResponseURL    string // temporary URL for proactive reply (valid 1 hour, use once)
	Question       string
	CreatedTime    time.Time
	Deadline       time.Time // ~30s, we close the stream here and switch to response_url
	StreamClosed   bool      // stream returned finish:true; waiting for agent to reply via response_url
	StreamClosedAt time.Time // set when StreamClosed becomes true; used for accelerated cleanup
	Finished       bool      // fully done
	mu             sync.Mutex
	answerCh       chan string        // receives agent reply from Send()
	ctx            context.Context    // canceled when task is removed; used to interrupt the agent goroutine
	cancel         context.CancelFunc // call on task removal to cancel ctx
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
		URL string `json:"url"`
	} `json:"image,omitempty"`
	// mixed message (text + image)
	Mixed *struct {
		MsgItem []struct {
			MsgType string `json:"msgtype"`
			Text    *struct {
				Content string `json:"content"`
			} `json:"text,omitempty"`
			Image *struct {
				URL string `json:"url"`
			} `json:"image,omitempty"`
		} `json:"msg_item"`
	} `json:"mixed,omitempty"`
	// event field
	Event *struct {
		EventType string `json:"eventtype"`
	} `json:"event,omitempty"`
}

// WeComAIBotStreamInfo represents the detailed stream content in streaming responses
type WeComAIBotStreamInfo struct {
	ID      string `json:"id"`
	Finish  bool   `json:"finish"`
	Content string `json:"content,omitempty"`
	MsgItem []struct {
		MsgType string `json:"msgtype"`
		Image   *struct {
			Base64 string `json:"base64"`
			MD5    string `json:"md5"`
		} `json:"image,omitempty"`
	} `json:"msg_item,omitempty"`
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

// NewWeComAIBotChannel creates a new WeCom AI Bot channel instance
func NewWeComAIBotChannel(
	cfg config.WeComAIBotConfig,
	messageBus *bus.MessageBus,
) (*WeComAIBotChannel, error) {
	if cfg.Token == "" || cfg.EncodingAESKey == "" {
		return nil, fmt.Errorf("token and encoding_aes_key are required for WeCom AI Bot")
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
	for len(queue) > 0 && queue[0].Finished {
		queue = queue[1:]
	}
	c.chatTasks[msg.ChatID] = queue
	var task *streamTask
	if len(queue) > 0 {
		task = queue[0]
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

	task.mu.Lock()
	streamClosed := task.StreamClosed
	responseURL := task.ResponseURL
	task.mu.Unlock()

	if streamClosed {
		// Stream already ended with a "please wait" notice; send the real reply via response_url.
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

	if r.Method == http.MethodGet {
		// URL verification
		c.handleVerification(ctx, w, r)
	} else if r.Method == http.MethodPost {
		// Message callback
		c.handleMessageCallback(ctx, w, r)
	} else {
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
		"msgtype": msg.MsgType,
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
			Stream: struct {
				ID      string `json:"id"`
				Finish  bool   `json:"finish"`
				Content string `json:"content,omitempty"`
				MsgItem []struct {
					MsgType string `json:"msgtype"`
					Image   *struct {
						Base64 string `json:"base64"`
						MD5    string `json:"md5"`
					} `json:"image,omitempty"`
				} `json:"msg_item,omitempty"`
			}{
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
			CanonicalID: "wecom_aibot:" + userID,
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
			Stream: struct {
				ID      string `json:"id"`
				Finish  bool   `json:"finish"`
				Content string `json:"content,omitempty"`
				MsgItem []struct {
					MsgType string `json:"msgtype"`
					Image   *struct {
						Base64 string `json:"base64"`
						MD5    string `json:"md5"`
					} `json:"image,omitempty"`
				} `json:"msg_item,omitempty"`
			}{
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
	logger.WarnC("wecom_aibot", "Image message type not yet fully implemented")
	if msg.Image == nil {
		logger.ErrorC("wecom_aibot", "Image message missing image field")
		return c.encryptEmptyResponse(timestamp, nonce)
	}

	imageURL := msg.Image.URL

	// Download and decrypt image
	_, err := c.downloadAndDecryptImage(ctx, imageURL)
	if err != nil {
		logger.ErrorCF("wecom_aibot", "Failed to process image", map[string]any{
			"error": err,
			"url":   imageURL,
		})
		return c.encryptResponse("", timestamp, nonce, WeComAIBotStreamResponse{
			MsgType: "stream",
			Stream: struct {
				ID      string `json:"id"`
				Finish  bool   `json:"finish"`
				Content string `json:"content,omitempty"`
				MsgItem []struct {
					MsgType string `json:"msgtype"`
					Image   *struct {
						Base64 string `json:"base64"`
						MD5    string `json:"md5"`
					} `json:"image,omitempty"`
				} `json:"msg_item,omitempty"`
			}{
				ID:     c.generateStreamID(),
				Finish: true,
				Content: fmt.Sprintf(
					"Image received (URL: %s), but image messages are not yet supported",
					imageURL,
				),
			},
		})
	}

	// Echo back the image (simple demo behavior)
	// streamID := c.generateStreamID()
	// return c.encryptImageResponse(streamID, timestamp, nonce, imageData)

	// For now, just acknowledge receipt without echoing the image
	return c.encryptResponse("", timestamp, nonce, WeComAIBotStreamResponse{
		MsgType: "stream",
		Stream: struct {
			ID      string `json:"id"`
			Finish  bool   `json:"finish"`
			Content string `json:"content,omitempty"`
			MsgItem []struct {
				MsgType string `json:"msgtype"`
				Image   *struct {
					Base64 string `json:"base64"`
					MD5    string `json:"md5"`
				} `json:"image,omitempty"`
			} `json:"msg_item,omitempty"`
		}{
			ID:     c.generateStreamID(),
			Finish: true,
			Content: fmt.Sprintf(
				"Image received (URL: %s), but image messages are not yet supported",
				imageURL,
			),
		},
	})
}

// handleMixedMessage handles mixed (text + image) messages
func (c *WeComAIBotChannel) handleMixedMessage(
	ctx context.Context,
	msg WeComAIBotMessage,
	timestamp, nonce string,
) string {
	logger.WarnC("wecom_aibot", "Mixed message type not yet fully implemented")
	return c.encryptResponse("", timestamp, nonce, WeComAIBotStreamResponse{
		MsgType: "stream",
		Stream: struct {
			ID      string `json:"id"`
			Finish  bool   `json:"finish"`
			Content string `json:"content,omitempty"`
			MsgItem []struct {
				MsgType string `json:"msgtype"`
				Image   *struct {
					Base64 string `json:"base64"`
					MD5    string `json:"md5"`
				} `json:"image,omitempty"`
			} `json:"msg_item,omitempty"`
		}{
			ID:      c.generateStreamID(),
			Finish:  true,
			Content: "Mixed message type is not yet supported",
		},
	})
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
			Stream: struct {
				ID      string `json:"id"`
				Finish  bool   `json:"finish"`
				Content string `json:"content,omitempty"`
				MsgItem []struct {
					MsgType string `json:"msgtype"`
					Image   *struct {
						Base64 string `json:"base64"`
						MD5    string `json:"md5"`
					} `json:"image,omitempty"`
				} `json:"msg_item,omitempty"`
			}{
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
			content = "⏳ Processing, please wait. The results will be sent shortly."
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
		// Only mark stream as closed; keep in chatTasks for Send() to find.
		task.mu.Lock()
		task.StreamClosed = true
		task.StreamClosedAt = time.Now()
		task.mu.Unlock()
		// Remove from streamTasks (no more stream polls expected).
		c.taskMu.Lock()
		delete(c.streamTasks, task.StreamID)
		c.taskMu.Unlock()
	}

	response := WeComAIBotStreamResponse{
		MsgType: "stream",
		Stream: struct {
			ID      string `json:"id"`
			Finish  bool   `json:"finish"`
			Content string `json:"content,omitempty"`
			MsgItem []struct {
				MsgType string `json:"msgtype"`
				Image   *struct {
					Base64 string `json:"base64"`
					MD5    string `json:"md5"`
				} `json:"image,omitempty"`
			} `json:"msg_item,omitempty"`
		}{
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
	task.mu.Lock()
	task.Finished = true
	task.mu.Unlock()
	task.cancel() // interrupt agent goroutine bound to this task

	c.taskMu.Lock()
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

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post to response_url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("response_url returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
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
	signature := c.generateSignature(timestamp, nonce, encrypted)

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

// encryptEmptyResponse returns empty encrypted response
func (c *WeComAIBotChannel) encryptEmptyResponse(timestamp, nonce string) string {
	return ""
}

// encryptMessage encrypts a plain text message for WeCom AI Bot
func (c *WeComAIBotChannel) encryptMessage(plaintext, receiveid string) (string, error) {
	// Decode AES key
	aesKey, err := base64.StdEncoding.DecodeString(c.config.EncodingAESKey + "=")
	if err != nil {
		return "", fmt.Errorf("failed to decode AES key: %w", err)
	}

	if len(aesKey) != 32 {
		return "", fmt.Errorf("invalid AES key length: %d", len(aesKey))
	}

	// Generate 16-byte random string
	randomBytes := make([]byte, 16)
	for i := range 16 {
		n, randErr := rand.Int(rand.Reader, big.NewInt(10))
		if randErr != nil {
			return "", fmt.Errorf("failed to generate random: %w", randErr)
		}
		randomBytes[i] = byte('0' + n.Int64())
	}

	// Build message: random(16) + msg_len(4) + msg + receiveid
	plaintextBytes := []byte(plaintext)
	receiveidBytes := []byte(receiveid)

	msgLen := uint32(len(plaintextBytes))
	msgLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLenBytes, msgLen)

	// Concatenate
	var buffer bytes.Buffer
	buffer.Write(randomBytes)
	buffer.Write(msgLenBytes)
	buffer.Write(plaintextBytes)
	buffer.Write(receiveidBytes)

	// PKCS7 padding
	plainData := buffer.Bytes()
	plainData = pkcs7Pad(plainData, blockSize)

	// AES-CBC encrypt
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	ciphertext := make([]byte, len(plainData))
	iv := aesKey[:aes.BlockSize]
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plainData)

	// Base64 encode
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	return encoded, nil
}

// pkcs7Pad adds PKCS7 padding
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	if padding == 0 {
		padding = blockSize
	}
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// generateSignature generates message signature using common function
func (c *WeComAIBotChannel) generateSignature(timestamp, nonce, encrypt string) string {
	// Sort parameters
	params := []string{c.config.Token, timestamp, nonce, encrypt}
	sort.Strings(params)

	// Concatenate
	str := strings.Join(params, "")

	// SHA1 hash
	hash := sha1.Sum([]byte(str))
	return fmt.Sprintf("%x", hash)
}

// generateStreamID generates a random stream ID
func (c *WeComAIBotChannel) generateStreamID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 10)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[n.Int64()]
	}
	return string(b)
}

// downloadAndDecryptImage downloads and decrypts an encrypted image
func (c *WeComAIBotChannel) downloadAndDecryptImage(
	ctx context.Context,
	imageURL string,
) ([]byte, error) {
	// Download image
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Limit image download to 20 MB to prevent memory exhaustion
	const maxImageSize = 20 << 20 // 20 MB
	encryptedData, err := io.ReadAll(io.LimitReader(resp.Body, maxImageSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}
	if len(encryptedData) > maxImageSize {
		return nil, fmt.Errorf("image too large (exceeds %d MB)", maxImageSize>>20)
	}

	logger.DebugCF("wecom_aibot", "Image downloaded", map[string]any{
		"size": len(encryptedData),
	})

	// Decode AES key
	aesKey, err := base64.StdEncoding.DecodeString(c.config.EncodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("failed to decode AES key: %w", err)
	}

	if len(aesKey) != 32 {
		return nil, fmt.Errorf("invalid AES key length: %d", len(aesKey))
	}

	// Decrypt image (AES-CBC)
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(encryptedData)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("encrypted data size not multiple of block size")
	}

	iv := aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)

	decryptedData := make([]byte, len(encryptedData))
	mode.CryptBlocks(decryptedData, encryptedData)

	// Remove PKCS7 padding
	decryptedData, err = pkcs7Unpad(decryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to unpad: %w", err)
	}

	logger.DebugCF("wecom_aibot", "Image decrypted", map[string]any{
		"size": len(decryptedData),
	})

	return decryptedData, nil
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
		for _, t := range queue {
			absoluteExpired := t.CreatedTime.Before(cutoff)
			graceExpired := t.StreamClosed &&
				!t.StreamClosedAt.IsZero() &&
				t.StreamClosedAt.Before(now.Add(-streamClosedGracePeriod))
			if !t.Finished && !absoluteExpired && !graceExpired {
				filtered = append(filtered, t)
			} else if !t.Finished {
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
