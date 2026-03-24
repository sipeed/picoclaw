package wecom

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// Long-connection WebSocket endpoint.
// Ref: https://developer.work.weixin.qq.com/document/path/101463
const (
	wsEndpoint          = "wss://openws.work.weixin.qq.com"
	wsHeartbeatInterval = 30 * time.Second
	wsConnectTimeout    = 15 * time.Second
	wsSubscribeTimeout  = 10 * time.Second
	wsSendMsgTimeout    = 10 * time.Second
	wsRespondMsgTimeout = 10 * time.Second
	wsWelcomeMsgTimeout = 5 * time.Second // WeCom requires welcome reply within 5 seconds
	wsMaxReconnectWait  = 60 * time.Second
	wsInitialReconnect  = time.Second

	// WeCom requires finish=true within 6 minutes of the first stream frame.
	// wsStreamTickInterval controls how often we send an in-progress hint.
	// wsStreamMaxDuration is a safety margin below the 6-minute hard limit.
	wsStreamTickInterval = 30 * time.Second
	wsStreamMaxDuration  = 5*time.Minute + 30*time.Second

	// wsImageDownloadTimeout caps the time we spend downloading an inbound image.
	wsImageDownloadTimeout = 30 * time.Second

	// Keep req_id -> chat route for late fallback pushes after stream window closes.
	wsLateReplyRouteTTL = 30 * time.Minute

	// wsStreamMaxContentBytes is the maximum UTF-8 byte length for the content field
	// of a single WeCom AI Bot stream / text / markdown frame.
	// Ref: https://developer.work.weixin.qq.com/document/path/101463
	wsStreamMaxContentBytes = 20480
)

// wsImageHTTPClient is a shared HTTP client for downloading inbound images.
// Reusing it enables connection pooling across multiple image downloads.
var wsImageHTTPClient = &http.Client{Timeout: wsImageDownloadTimeout}

// WeComAIBotWSChannel implements channels.Channel for WeCom AI Bot using the
// WebSocket long-connection API.
// Unlike the webhook counterpart it does NOT implement WebhookHandler, so the
// HTTP manager will not register any callback URL for it.
type WeComAIBotWSChannel struct {
	*channels.BaseChannel
	config config.WeComAIBotConfig
	ctx    context.Context
	cancel context.CancelFunc

	// conn is the active WebSocket connection; nil when disconnected.
	// All writes are serialized through connMu.
	conn   *websocket.Conn
	connMu sync.Mutex

	// dedupe prevents duplicate message processing (WeCom may re-deliver).
	dedupe *MessageDeduplicator

	// reqStates holds per-req_id runtime state.
	// It unifies active task state and late-reply fallback routing.
	reqStates   map[string]*wsReqState
	reqStatesMu sync.Mutex

	// reqPending correlates command req_ids with response channels.
	// Used only for subscribe/ping command-response pairs.
	reqPending   map[string]chan wsEnvelope
	reqPendingMu sync.Mutex
}

// wsTask tracks one in-progress agent reply for a single chat turn.
type wsTask struct {
	ReqID    string // req_id echoed in all replies for this turn
	ChatID   string
	ChatType uint32
	StreamID string      // our generated stream.id
	answerCh chan string // agent delivers its reply here via Send()
	ctx      context.Context
	cancel   context.CancelFunc
}

type wsReqState struct {
	Task  *wsTask
	Route wsLateReplyRoute
}

type wsLateReplyRoute struct {
	ChatID    string
	ChatType  uint32
	ReadyAt   time.Time
	ExpiresAt time.Time
}

// ---- WebSocket protocol types ----

// wsEnvelope is the generic JSON envelope for all WebSocket messages.
type wsEnvelope struct {
	Cmd     string          `json:"cmd,omitempty"`
	Headers wsHeaders       `json:"headers"`
	Body    json.RawMessage `json:"body,omitempty"`
	ErrCode int             `json:"errcode,omitempty"`
	ErrMsg  string          `json:"errmsg,omitempty"`
}

type wsHeaders struct {
	ReqID string `json:"req_id"`
}

// wsCommand is an outgoing request sent over the WebSocket.
type wsCommand struct {
	Cmd     string    `json:"cmd"`
	Headers wsHeaders `json:"headers"`
	Body    any       `json:"body,omitempty"`
}

type wsSendMsgBody struct {
	ChatID   string             `json:"chatid"`
	ChatType uint32             `json:"chat_type,omitempty"`
	MsgType  string             `json:"msgtype"`
	Markdown *wsMarkdownContent `json:"markdown,omitempty"`
}

// wsRespondMsgBody is the body for aibot_respond_msg / aibot_respond_welcome_msg.
type wsRespondMsgBody struct {
	MsgType  string             `json:"msgtype"`
	Stream   *wsStreamContent   `json:"stream,omitempty"`
	Text     *wsTextContent     `json:"text,omitempty"`
	Markdown *wsMarkdownContent `json:"markdown,omitempty"`
	Image    *wsImageContent    `json:"image,omitempty"`
}

type wsStreamContent struct {
	ID      string `json:"id"`
	Finish  bool   `json:"finish"`
	Content string `json:"content,omitempty"`
}

// wsImageContent carries a base64-encoded image payload for outbound messages.
type wsImageContent struct {
	Base64 string `json:"base64"`
	MD5    string `json:"md5"`
}

type wsTextContent struct {
	Content string `json:"content"`
}

type wsMarkdownContent struct {
	Content string `json:"content"`
}

// WeComAIBotWSMessage is the decoded body of aibot_msg_callback /
// aibot_event_callback in WebSocket long-connection mode.
// The structure mirrors WeComAIBotMessage but includes extra fields
// that only appear in long-connection callbacks (Voice, AESKey on Image/File).
type WeComAIBotWSMessage struct {
	MsgID      string `json:"msgid"`
	CreateTime int64  `json:"create_time,omitempty"`
	AIBotID    string `json:"aibotid"`
	ChatID     string `json:"chatid,omitempty"`
	ChatType   string `json:"chattype,omitempty"` // "single" | "group"
	From       struct {
		UserID string `json:"userid"`
	} `json:"from"`
	MsgType string `json:"msgtype"`
	Text    *struct {
		Content string `json:"content"`
	} `json:"text,omitempty"`
	Image *struct {
		URL    string `json:"url"`
		AESKey string `json:"aeskey,omitempty"` // long-connection: per-resource decrypt key
	} `json:"image,omitempty"`
	Voice *struct {
		Content string `json:"content"` // WeCom transcribes voice to text in callbacks
	} `json:"voice,omitempty"`
	Mixed *struct {
		MsgItem []struct {
			MsgType string `json:"msgtype"`
			Text    *struct {
				Content string `json:"content"`
			} `json:"text,omitempty"`
			Image *struct {
				URL    string `json:"url"`
				AESKey string `json:"aeskey,omitempty"`
			} `json:"image,omitempty"`
		} `json:"msg_item"`
	} `json:"mixed,omitempty"`
	Event *struct {
		EventType string `json:"eventtype"`
	} `json:"event,omitempty"`
	File *struct {
		URL    string `json:"url"`
		AESKey string `json:"aeskey,omitempty"`
	} `json:"file,omitempty"`
	Video *struct {
		URL    string `json:"url"`
		AESKey string `json:"aeskey,omitempty"`
	} `json:"video,omitempty"`
}

// ---- Constructor ----

// newWeComAIBotWSChannel creates a WeComAIBotWSChannel for WebSocket mode.
func newWeComAIBotWSChannel(
	cfg config.WeComAIBotConfig,
	messageBus *bus.MessageBus,
) (*WeComAIBotWSChannel, error) {
	if cfg.BotID == "" || cfg.Secret() == "" {
		return nil, fmt.Errorf("bot_id and secret are required for WeCom AI Bot WebSocket mode")
	}

	base := channels.NewBaseChannel("wecom_aibot", cfg, messageBus, cfg.AllowFrom,
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &WeComAIBotWSChannel{
		BaseChannel: base,
		config:      cfg,
		dedupe:      NewMessageDeduplicator(wecomMaxProcessedMessages),
		reqStates:   make(map[string]*wsReqState),
		reqPending:  make(map[string]chan wsEnvelope),
	}, nil
}

// ---- Channel interface ----

// Name implements channels.Channel.
func (c *WeComAIBotWSChannel) Name() string { return "wecom_aibot" }

// Start connects to the WeCom WebSocket endpoint and begins message processing.
func (c *WeComAIBotWSChannel) Start(ctx context.Context) error {
	logger.InfoC("wecom_aibot", "Starting WeCom AI Bot channel (WebSocket long-connection mode)...")
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.SetRunning(true)
	go c.connectLoop()
	logger.InfoC("wecom_aibot", "WeCom AI Bot channel started (WebSocket mode)")
	return nil
}

// Stop shuts down the channel and closes the WebSocket connection.
func (c *WeComAIBotWSChannel) Stop(_ context.Context) error {
	logger.InfoC("wecom_aibot", "Stopping WeCom AI Bot channel (WebSocket mode)...")
	if c.cancel != nil {
		c.cancel()
	}
	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()
	c.SetRunning(false)
	logger.InfoC("wecom_aibot", "WeCom AI Bot channel stopped")
	return nil
}

// Send delivers the agent reply for msg.ChatID.
// The waiting task goroutine picks it up and writes the final stream response.
func (c *WeComAIBotWSChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	// msg.ChatID carries the inbound req_id (set by dispatchWSAgentTask).
	// For cron-triggered messages, msg.ChatID is the real WeCom chat/user ID
	// and there will be no matching entry in reqStates; fall through to proactive push.
	task, route, ok := c.getReqState(msg.ChatID)
	if !ok {
		// No req_id record found — this is a cron/scheduler-originated message.
		// Send it as a proactive markdown push using the chat ID directly.
		logger.InfoCF("wecom_aibot", "Send: no req_id state, delivering via proactive push (cron/scheduler)",
			map[string]any{"chat_id": msg.ChatID})
		if err := c.wsSendActivePush(msg.ChatID, 0, msg.Content); err != nil {
			logger.WarnCF("wecom_aibot", "Proactive push failed",
				map[string]any{"chat_id": msg.ChatID, "error": err.Error()})
			return fmt.Errorf("websocket delivery failed: %w", channels.ErrSendFailed)
		}
		return nil
	}

	if task == nil {
		if time.Now().Before(route.ReadyAt) {
			// Keep using aibot_respond_msg within stream window; do not proactively
			// push unless wsStreamMaxDuration has elapsed.
			logger.WarnCF("wecom_aibot", "Send: stream window still open, skip proactive push",
				map[string]any{"req_id": msg.ChatID, "ready_at": route.ReadyAt.Format(time.RFC3339)})
			return nil
		}

		if err := c.wsSendActivePush(route.ChatID, route.ChatType, msg.Content); err != nil {
			logger.WarnCF("wecom_aibot", "Late reply proactive push failed",
				map[string]any{"req_id": msg.ChatID, "chat_id": route.ChatID, "error": err.Error()})
			return fmt.Errorf("websocket delivery failed: %w", channels.ErrSendFailed)
		}
		logger.InfoCF("wecom_aibot", "Late reply delivered via proactive push",
			map[string]any{"req_id": msg.ChatID, "chat_id": route.ChatID, "chat_type": route.ChatType})
		c.deleteReqState(msg.ChatID)
		return nil
	}

	// Non-blocking fast path: when answerCh has space, deliver without racing
	// against task.ctx.Done() (which fires when the task is canceled by a new
	// incoming message, but the response must still be sent).
	select {
	case task.answerCh <- msg.Content:
		return nil
	default:
	}
	// answerCh was full; block with cancellation guards.
	select {
	case task.answerCh <- msg.Content:
	case <-task.ctx.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// ---- Connection management ----

// wsBackoffResetDuration is the minimum duration a WebSocket connection must
// stay up before we reset the reconnect backoff to its initial value. This
// prevents a short burst of failures from causing long waits after later,
// stable connection periods.
const wsBackoffResetDuration = time.Minute

// connectLoop maintains the WebSocket connection, reconnecting on failure with
// exponential backoff.
func (c *WeComAIBotWSChannel) connectLoop() {
	backoff := wsInitialReconnect
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		logger.InfoC("wecom_aibot", "Connecting to WeCom WebSocket endpoint...")
		start := time.Now()
		if err := c.runConnection(); err != nil {
			elapsed := time.Since(start)
			// If the connection was stable for long enough, reset backoff so that
			// a previous burst of failures does not keep us at the maximum delay.
			if elapsed >= wsBackoffResetDuration {
				backoff = wsInitialReconnect
			}
			select {
			case <-c.ctx.Done():
				return
			default:
				logger.WarnCF("wecom_aibot", "WebSocket connection lost, reconnecting",
					map[string]any{"error": err.Error(), "backoff": backoff.String()})
				select {
				case <-time.After(backoff):
				case <-c.ctx.Done():
					return
				}
				if backoff < wsMaxReconnectWait {
					backoff *= 2
					if backoff > wsMaxReconnectWait {
						backoff = wsMaxReconnectWait
					}
				}
			}
		} else {
			// Clean exit (context canceled); stop reconnecting.
			return
		}
	}
}

// runConnection dials, subscribes, and runs the read/heartbeat loops until the
// connection closes or the channel context is canceled.
func (c *WeComAIBotWSChannel) runConnection() error {
	dialCtx, dialCancel := context.WithTimeout(c.ctx, wsConnectTimeout)
	conn, httpResp, err := websocket.DefaultDialer.DialContext(dialCtx, wsEndpoint, nil)
	dialCancel()
	if httpResp != nil {
		httpResp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	defer func() {
		c.connMu.Lock()
		if c.conn == conn {
			c.conn = nil
		}
		c.connMu.Unlock()
		// Cancel any tasks that were started over this connection so their
		// agent goroutines do not keep running after the connection is gone.
		c.cancelAllTasks()
	}()

	// ---- Read loop (must start BEFORE subscribing) ----
	// sendAndWait blocks waiting for the subscribe response on reqPending;
	// readLoop is the only goroutine that delivers messages to reqPending.
	// Starting readLoop first avoids a deadlock where sendAndWait times out
	// because no one reads the server's reply.
	readErrCh := make(chan error, 1)
	go func() { readErrCh <- c.readLoop(conn) }()

	// ---- Subscribe ----
	reqID := wsGenerateID()
	resp, err := c.sendAndWait(conn, reqID, wsCommand{
		Cmd:     "aibot_subscribe",
		Headers: wsHeaders{ReqID: reqID},
		Body: map[string]string{
			"bot_id": c.config.BotID,
			"secret": c.config.Secret(),
		},
	}, wsSubscribeTimeout)
	if err != nil {
		conn.Close() // stop readLoop
		<-readErrCh
		return fmt.Errorf("subscribe failed: %w", err)
	}
	if resp.ErrCode != 0 {
		conn.Close()
		<-readErrCh
		return fmt.Errorf("subscribe rejected (errcode=%d): %s", resp.ErrCode, resp.ErrMsg)
	}

	logger.InfoC("wecom_aibot", "WebSocket subscription successful")

	// ---- Heartbeat goroutine ----
	hbDone := make(chan struct{})
	go func() {
		defer close(hbDone)
		c.heartbeatLoop(conn)
	}()

	// Wait for the read loop to exit, then tear down the heartbeat.
	readErr := <-readErrCh
	conn.Close() // signal heartbeat to stop (idempotent)
	<-hbDone
	return readErr
}

// sendAndWait registers a pending-response slot, sends cmd, and blocks until
// the matching response arrives or the timeout/context fires.
func (c *WeComAIBotWSChannel) sendAndWait(
	conn *websocket.Conn,
	reqID string,
	cmd wsCommand,
	timeout time.Duration,
) (wsEnvelope, error) {
	ch := make(chan wsEnvelope, 1)
	c.reqPendingMu.Lock()
	c.reqPending[reqID] = ch
	c.reqPendingMu.Unlock()

	cleanup := func() {
		c.reqPendingMu.Lock()
		delete(c.reqPending, reqID)
		c.reqPendingMu.Unlock()
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		cleanup()
		return wsEnvelope{}, fmt.Errorf("marshal command: %w", err)
	}
	c.connMu.Lock()
	err = conn.WriteMessage(websocket.TextMessage, data)
	c.connMu.Unlock()
	if err != nil {
		cleanup()
		return wsEnvelope{}, fmt.Errorf("write command: %w", err)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case env := <-ch:
		return env, nil
	case <-timer.C:
		cleanup()
		return wsEnvelope{}, fmt.Errorf("timeout waiting for response (req_id=%s)", reqID)
	case <-c.ctx.Done():
		cleanup()
		return wsEnvelope{}, c.ctx.Err()
	}
}

// heartbeatLoop sends a ping every wsHeartbeatInterval until conn is closed.
// It validates the server's pong response via sendAndWait; a failed pong
// triggers a reconnection by closing the connection.
func (c *WeComAIBotWSChannel) heartbeatLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(wsHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			reqID := wsGenerateID()
			resp, err := c.sendAndWait(conn, reqID, wsCommand{
				Cmd:     "ping",
				Headers: wsHeaders{ReqID: reqID},
			}, wsHeartbeatInterval)
			if err != nil {
				logger.WarnCF("wecom_aibot", "Heartbeat failed, closing connection",
					map[string]any{"error": err.Error()})
				conn.Close()
				return
			}
			if resp.ErrCode != 0 {
				logger.WarnCF("wecom_aibot", "Heartbeat rejected",
					map[string]any{"errcode": resp.ErrCode, "errmsg": resp.ErrMsg})
				conn.Close()
				return
			}
			logger.DebugCF("wecom_aibot", "Heartbeat pong received", map[string]any{"req_id": reqID})
		case <-c.ctx.Done():
			return
		}
	}
}

// readLoop reads WebSocket messages and dispatches them until the connection
// closes or the channel is stopped.
func (c *WeComAIBotWSChannel) readLoop(conn *websocket.Conn) error {
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-c.ctx.Done():
				return nil // clean shutdown
			default:
				return fmt.Errorf("read error: %w", err)
			}
		}

		var env wsEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			logger.WarnCF("wecom_aibot", "Failed to parse WebSocket message",
				map[string]any{"error": err.Error(), "raw": string(raw)})
			continue
		}

		// Command responses have an empty Cmd field; forward to any waiting
		// sendAndWait() call, or silently drop if no one is waiting (e.g.
		// late responses after timeout).
		if env.Cmd == "" && env.Headers.ReqID != "" {
			c.reqPendingMu.Lock()
			ch, ok := c.reqPending[env.Headers.ReqID]
			if ok {
				delete(c.reqPending, env.Headers.ReqID)
			}
			c.reqPendingMu.Unlock()
			if ok {
				ch <- env
			}
			continue
		}

		// Dispatch to appropriate handler in a separate goroutine so the
		// read loop is never blocked by a slow agent.
		go c.handleEnvelope(env)
	}
}

// ---- Message / event handlers ----

// handleEnvelope routes a WebSocket envelope to the right handler.
func (c *WeComAIBotWSChannel) handleEnvelope(env wsEnvelope) {
	switch env.Cmd {
	case "aibot_msg_callback":
		c.handleMsgCallback(env)
	case "aibot_event_callback":
		c.handleEventCallback(env)
	default:
		logger.DebugCF("wecom_aibot", "Unhandled WebSocket command",
			map[string]any{"cmd": env.Cmd})
	}
}

// handleMsgCallback processes aibot_msg_callback.
func (c *WeComAIBotWSChannel) handleMsgCallback(env wsEnvelope) {
	var msg WeComAIBotWSMessage
	if err := json.Unmarshal(env.Body, &msg); err != nil {
		logger.WarnCF("wecom_aibot", "Failed to parse msg callback body",
			map[string]any{"error": err.Error()})
		return
	}

	// Deduplicate by msgid (WeCom may re-deliver on network issues).
	if msg.MsgID != "" && !c.dedupe.MarkMessageProcessed(msg.MsgID) {
		logger.DebugCF("wecom_aibot", "Duplicate message ignored",
			map[string]any{"msgid": msg.MsgID})
		return
	}

	reqID := env.Headers.ReqID
	switch msg.MsgType {
	case "text":
		c.handleWSTextMessage(reqID, msg)
	case "image":
		c.handleWSImageMessage(reqID, msg)
	case "voice":
		c.handleWSVoiceMessage(reqID, msg)
	case "mixed":
		c.handleWSMixedMessage(reqID, msg)
	case "file":
		c.handleWSFileMessage(reqID, msg)
	case "video":
		c.handleWSVideoMessage(reqID, msg)
	default:
		logger.WarnCF("wecom_aibot", "Unsupported message type",
			map[string]any{"msgtype": msg.MsgType})
		c.wsSendStreamFinish(reqID, wsGenerateID(),
			"Unsupported message type: "+msg.MsgType)
	}
}

// handleEventCallback processes aibot_event_callback.
func (c *WeComAIBotWSChannel) handleEventCallback(env wsEnvelope) {
	var msg WeComAIBotWSMessage
	if err := json.Unmarshal(env.Body, &msg); err != nil {
		logger.WarnCF("wecom_aibot", "Failed to parse event callback body",
			map[string]any{"error": err.Error()})
		return
	}

	// Deduplicate by msgid.
	if msg.MsgID != "" && !c.dedupe.MarkMessageProcessed(msg.MsgID) {
		logger.DebugCF("wecom_aibot", "Duplicate event ignored",
			map[string]any{"msgid": msg.MsgID})
		return
	}

	var eventType string
	if msg.Event != nil {
		eventType = msg.Event.EventType
	}
	logger.DebugCF("wecom_aibot", "Received event callback",
		map[string]any{"event_type": eventType})

	switch eventType {
	case "enter_chat":
		if c.config.WelcomeMessage != "" {
			c.wsSendWelcomeMsg(env.Headers.ReqID, c.config.WelcomeMessage)
		}
	case "disconnected_event":
		// The server will close this connection after sending this event.
		// connectLoop will detect the closure and reconnect automatically.
		logger.WarnC("wecom_aibot",
			"Received disconnected_event: this connection is being replaced by a newer one")
	default:
		logger.DebugCF("wecom_aibot", "Unhandled event type",
			map[string]any{"event_type": eventType})
	}
}

// handleWSTextMessage dispatches a plain-text message to the agent and streams
// the reply back over the WebSocket connection.
func (c *WeComAIBotWSChannel) handleWSTextMessage(reqID string, msg WeComAIBotWSMessage) {
	if msg.Text == nil {
		logger.ErrorC("wecom_aibot", "text message missing text field")
		return
	}
	c.dispatchWSAgentTask(reqID, msg, msg.Text.Content, nil)
}

// handleWSImageMessage downloads and stores the inbound image, then dispatches
// it to the agent as a media-tagged message.
func (c *WeComAIBotWSChannel) handleWSImageMessage(reqID string, msg WeComAIBotWSMessage) {
	if msg.Image == nil {
		logger.WarnC("wecom_aibot", "Image message missing image field")
		c.wsSendStreamFinish(reqID, wsGenerateID(), "Image message could not be processed.")
		return
	}
	c.wsHandleMediaMessage(reqID, msg, msg.Image.URL, msg.Image.AESKey, "image")
}

// wsHandleMediaMessage is a shared helper for image, file and video messages.
// It downloads the resource, stores it in MediaStore, and dispatches to the agent.
func (c *WeComAIBotWSChannel) wsHandleMediaMessage(
	reqID string, msg WeComAIBotWSMessage,
	resourceURL, aesKey, label string,
) {
	chatID := wsChatID(msg)

	ctx, cancel := context.WithTimeout(c.ctx, wsImageDownloadTimeout)
	defer cancel()

	ref, err := c.storeWSMedia(ctx, chatID, msg.MsgID, resourceURL, aesKey, wsLabelToDefaultExt(label))
	if err != nil {
		logger.WarnCF("wecom_aibot", "Failed to download/store WS "+label,
			map[string]any{"error": err.Error(), "url": resourceURL})
		c.wsSendStreamFinish(reqID, wsGenerateID(),
			strings.ToUpper(label[:1])+label[1:]+" message could not be processed.")
		return
	}

	c.dispatchWSAgentTask(reqID, msg, "["+label+"]", []string{ref})
}

// handleWSMixedMessage handles mixed text+image messages.
// All text parts are collected into the content string; all image parts are
// downloaded and stored in MediaStore before dispatching to the agent.
func (c *WeComAIBotWSChannel) handleWSMixedMessage(reqID string, msg WeComAIBotWSMessage) {
	if msg.Mixed == nil {
		logger.WarnC("wecom_aibot", "Mixed message has no content")
		c.wsSendStreamFinish(reqID, wsGenerateID(), "Mixed message type is not yet fully supported.")
		return
	}

	chatID := wsChatID(msg)

	ctx, cancel := context.WithTimeout(c.ctx, wsImageDownloadTimeout)
	defer cancel()

	var textParts []string
	var mediaRefs []string
	for _, item := range msg.Mixed.MsgItem {
		switch item.MsgType {
		case "text":
			if item.Text != nil && item.Text.Content != "" {
				textParts = append(textParts, item.Text.Content)
			}
		case "image":
			if item.Image != nil {
				ref, err := c.storeWSMedia(ctx, chatID,
					msg.MsgID+"-"+wsGenerateID(), item.Image.URL, item.Image.AESKey, ".jpg")
				if err != nil {
					logger.WarnCF("wecom_aibot", "Failed to download/store mixed image",
						map[string]any{"error": err.Error()})
				} else {
					mediaRefs = append(mediaRefs, ref)
				}
			}
		default:
			logger.WarnCF("wecom_aibot", "Unsupported item type in mixed message",
				map[string]any{"msgtype": item.MsgType})
		}
	}

	if len(textParts) == 0 && len(mediaRefs) == 0 {
		logger.WarnC("wecom_aibot", "Mixed message has no usable content")
		c.wsSendStreamFinish(reqID, wsGenerateID(), "Mixed message type is not yet fully supported.")
		return
	}

	content := strings.Join(textParts, "\n")
	if content == "" {
		content = "[images]"
	}
	c.dispatchWSAgentTask(reqID, msg, content, mediaRefs)
}

// dispatchWSAgentTask registers a new agent task, sends the opening stream frame,
// and starts a goroutine that runs the agent and streams the reply back.
// content is the text forwarded to the agent; mediaRefs are optional media
// store references attached to the inbound message.
func (c *WeComAIBotWSChannel) dispatchWSAgentTask(
	reqID string,
	msg WeComAIBotWSMessage,
	content string,
	mediaRefs []string,
) {
	userID := msg.From.UserID
	if userID == "" {
		userID = "unknown"
	}
	// actualChatID is the real WeCom chat/user ID used for peer identification.
	// reqID is used as the routing chatID so each turn is independently addressable.
	actualChatID := wsChatID(msg)

	streamID := wsGenerateID()
	chatType := wsChatTypeValue(msg.ChatType)
	taskCtx, taskCancel := context.WithCancel(c.ctx)

	task := &wsTask{
		ReqID:    reqID,
		ChatID:   actualChatID,
		ChatType: chatType,
		StreamID: streamID,
		answerCh: make(chan string, 1),
		ctx:      taskCtx,
		cancel:   taskCancel,
	}
	// Each req_id is unique per WeCom turn; tasks run concurrently, no cancellation.
	c.setReqState(reqID, &wsReqState{
		Task: task,
		Route: wsLateReplyRoute{
			ChatID:    actualChatID,
			ChatType:  chatType,
			ReadyAt:   time.Now().Add(wsStreamMaxDuration),
			ExpiresAt: time.Now().Add(wsLateReplyRouteTTL),
		},
	})

	logger.DebugCF("wecom_aibot", "Registered new agent task",
		map[string]any{"chat_id": actualChatID, "req_id": reqID, "stream_id": streamID})

	// Send an empty stream opening frame (finish=false) immediately.
	c.wsSendStreamChunk(reqID, streamID, false, "")

	go func() {
		defer func() {
			taskCancel()
			c.clearReqTask(reqID, task)
		}()

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
		peer := bus.Peer{Kind: peerKind, ID: actualChatID}
		metadata := map[string]string{
			"channel":   "wecom_aibot",
			"chat_id":   actualChatID,
			"chat_type": msg.ChatType,
			"msg_type":  msg.MsgType,
			"msgid":     msg.MsgID,
			"aibotid":   msg.AIBotID,
			"stream_id": streamID,
		}
		// Pass reqID as chatID: OutboundMessage.ChatID = reqID → Send() finds tasks[reqID].
		c.HandleMessage(taskCtx, peer, reqID, userID, reqID,
			content, mediaRefs, metadata, sender)

		// Wait for the agent reply. While waiting, send periodic finish=false
		// hints so the user knows processing is still in progress.
		// WeCom requires finish=true within 6 minutes of the first stream frame;
		// wsStreamMaxDuration enforces that limit with a safety margin.
		waitHints := []string{
			"⏳ Processing, please wait...",
			"⏳ Still processing, please wait...",
			"⏳ Almost there, please wait...",
		}
		ticker := time.NewTicker(wsStreamTickInterval)
		defer ticker.Stop()
		deadlineTimer := time.NewTimer(wsStreamMaxDuration)
		defer deadlineTimer.Stop()
		tickCount := 0
		for {
			select {
			case answer := <-task.answerCh:
				// Split the answer into byte-bounded chunks and send as stream frames.
				// All but the last carry finish=false; the final frame closes the stream.
				chunks := splitWSContent(answer, wsStreamMaxContentBytes)
				for i, chunk := range chunks {
					c.wsSendStreamChunk(reqID, streamID, i == len(chunks)-1, chunk)
				}
				c.deleteReqState(reqID)
				return
			case <-ticker.C:
				hint := waitHints[tickCount%len(waitHints)]
				tickCount++
				logger.DebugCF("wecom_aibot", "Sending stream progress hint",
					map[string]any{"chat_id": actualChatID, "tick": tickCount})
				c.wsSendStreamChunk(reqID, streamID, false, hint)
			case <-deadlineTimer.C:
				logger.WarnCF("wecom_aibot",
					"Stream response deadline reached, closing stream; late reply will be pushed",
					map[string]any{"chat_id": actualChatID})
				c.wsSendStreamFinish(reqID, streamID,
					"⏳ Processing is taking longer than expected, the response will be sent as a follow-up message.")
				return
			case <-taskCtx.Done():
				// Give a short grace period so that a response queued in the bus
				// just before cancellation can still be delivered.  This closes a
				// race where a rapid second message cancels this task after the
				// agent already published but before Send() wrote to answerCh.
				//
				// The connection is gone at this point, so we cannot use
				// wsSendStreamFinish.  Try wsSendActivePush on the (possibly
				// already-restored) connection; if that also fails, leave the
				// route intact so Send() can push the reply once reconnected.
				select {
				case answer := <-task.answerCh:
					if err := c.wsSendActivePush(task.ChatID, task.ChatType, answer); err != nil {
						logger.WarnCF("wecom_aibot",
							"Grace-period push failed after task cancellation; reply may be lost",
							map[string]any{"req_id": reqID, "chat_id": task.ChatID, "error": err.Error()})
					} else {
						c.deleteReqState(reqID)
					}
				case <-time.After(100 * time.Millisecond):
				}
				return
			}
		}
	}()
}

// handleWSVoiceMessage handles voice messages.
// WeCom transcribes voice to text in the callback; if the transcription is
// present it is dispatched as plain text to the agent.
func (c *WeComAIBotWSChannel) handleWSVoiceMessage(reqID string, msg WeComAIBotWSMessage) {
	if msg.Voice != nil && msg.Voice.Content != "" {
		c.dispatchWSAgentTask(reqID, msg, msg.Voice.Content, nil)
		return
	}
	c.wsSendStreamFinish(reqID, wsGenerateID(), "Voice messages are not yet supported.")
}

// handleWSFileMessage handles file messages.
func (c *WeComAIBotWSChannel) handleWSFileMessage(reqID string, msg WeComAIBotWSMessage) {
	if msg.File == nil {
		logger.WarnC("wecom_aibot", "File message missing file field")
		c.wsSendStreamFinish(reqID, wsGenerateID(), "File message could not be processed.")
		return
	}
	c.wsHandleMediaMessage(reqID, msg, msg.File.URL, msg.File.AESKey, "file")
}

// handleWSVideoMessage handles video messages.
func (c *WeComAIBotWSChannel) handleWSVideoMessage(reqID string, msg WeComAIBotWSMessage) {
	if msg.Video == nil {
		logger.WarnC("wecom_aibot", "Video message missing video field")
		c.wsSendStreamFinish(reqID, wsGenerateID(), "Video message could not be processed.")
		return
	}
	c.wsHandleMediaMessage(reqID, msg, msg.Video.URL, msg.Video.AESKey, "video")
}

// ---- WebSocket write helpers ----

// wsSendStreamChunk sends an aibot_respond_msg stream frame.
func (c *WeComAIBotWSChannel) wsSendStreamChunk(reqID, streamID string, finish bool, content string) {
	logger.DebugCF("wecom_aibot", "Sending stream chunk", map[string]any{
		"stream_id": streamID,
		"finish":    finish,
		"preview":   utils.Truncate(content, 100),
	})
	cmd := wsCommand{
		Cmd:     "aibot_respond_msg",
		Headers: wsHeaders{ReqID: reqID},
		Body: wsRespondMsgBody{
			MsgType: "stream",
			Stream: &wsStreamContent{
				ID:      streamID,
				Finish:  finish,
				Content: content,
			},
		},
	}
	if err := c.writeWSAndWait(cmd, wsRespondMsgTimeout); err != nil {
		logger.WarnCF("wecom_aibot", "Stream chunk ack failed", map[string]any{
			"req_id":    reqID,
			"stream_id": streamID,
			"finish":    finish,
			"error":     err,
		})
	}
}

// wsSendStreamFinish sends the final aibot_respond_msg frame (finish=true, no images).
func (c *WeComAIBotWSChannel) wsSendStreamFinish(reqID, streamID, content string) {
	c.wsSendStreamChunk(reqID, streamID, true, content)
}

// wsSendWelcomeMsg sends a text welcome message via aibot_respond_welcome_msg.
func (c *WeComAIBotWSChannel) wsSendWelcomeMsg(reqID, content string) {
	logger.DebugCF("wecom_aibot", "Sending welcome message", map[string]any{"req_id": reqID})
	cmd := wsCommand{
		Cmd:     "aibot_respond_welcome_msg",
		Headers: wsHeaders{ReqID: reqID},
		Body: wsRespondMsgBody{
			MsgType: "text",
			Text:    &wsTextContent{Content: content},
		},
	}
	if err := c.writeWSAndWait(cmd, wsWelcomeMsgTimeout); err != nil {
		logger.WarnCF("wecom_aibot", "Welcome message ack failed",
			map[string]any{"req_id": reqID, "error": err.Error()})
	}
}

// wsSendActivePush sends a proactive markdown message using aibot_send_msg.
// Long content is automatically split into byte-bounded chunks (≤ wsStreamMaxContentBytes
// each) and delivered as consecutive messages.
// It is used as a fallback for late replies after stream response window expires.
func (c *WeComAIBotWSChannel) wsSendActivePush(chatID string, chatType uint32, content string) error {
	if chatID == "" {
		return fmt.Errorf("chatid is empty")
	}
	for _, chunk := range splitWSContent(content, wsStreamMaxContentBytes) {
		reqID := wsGenerateID()
		if err := c.writeWSAndWait(wsCommand{
			Cmd:     "aibot_send_msg",
			Headers: wsHeaders{ReqID: reqID},
			Body: wsSendMsgBody{
				ChatID:   chatID,
				ChatType: chatType,
				MsgType:  "markdown",
				Markdown: &wsMarkdownContent{Content: chunk},
			},
		}, wsSendMsgTimeout); err != nil {
			return err
		}
	}
	return nil
}

// writeWSAndWait writes cmd to the active connection and validates the command response.
func (c *WeComAIBotWSChannel) writeWSAndWait(cmd wsCommand, timeout time.Duration) error {
	if cmd.Headers.ReqID == "" {
		return fmt.Errorf("req_id is empty")
	}

	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()
	if conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	resp, err := c.sendAndWait(conn, cmd.Headers.ReqID, cmd, timeout)
	if err != nil {
		return err
	}
	if resp.ErrCode != 0 {
		return fmt.Errorf("%s rejected (errcode=%d): %s", cmd.Cmd, resp.ErrCode, resp.ErrMsg)
	}
	return nil
}

// cancelAllTasks cancels every pending agent task; called when the connection drops.
// It also expires each task's stream window (ReadyAt = now) so that when the agent
// eventually delivers its reply via Send(), the message is forwarded via
// wsSendActivePush on the restored connection instead of being silently discarded.
func (c *WeComAIBotWSChannel) cancelAllTasks() {
	c.reqStatesMu.Lock()
	defer c.reqStatesMu.Unlock()
	now := time.Now()
	for _, state := range c.reqStates {
		if state != nil && state.Task != nil {
			state.Task.cancel()
			state.Task = nil
			// Expire the stream window immediately so Send() uses wsSendActivePush.
			state.Route.ReadyAt = now
		}
	}
}

func (c *WeComAIBotWSChannel) setReqState(reqID string, state *wsReqState) {
	c.reqStatesMu.Lock()
	defer c.reqStatesMu.Unlock()
	now := time.Now()
	for k, v := range c.reqStates {
		if v == nil || now.After(v.Route.ExpiresAt) {
			delete(c.reqStates, k)
		}
	}
	c.reqStates[reqID] = state
}

func (c *WeComAIBotWSChannel) getReqState(reqID string) (*wsTask, wsLateReplyRoute, bool) {
	c.reqStatesMu.Lock()
	defer c.reqStatesMu.Unlock()
	state, ok := c.reqStates[reqID]
	if !ok || state == nil {
		return nil, wsLateReplyRoute{}, false
	}
	if time.Now().After(state.Route.ExpiresAt) {
		delete(c.reqStates, reqID)
		return nil, wsLateReplyRoute{}, false
	}
	return state.Task, state.Route, true
}

func (c *WeComAIBotWSChannel) deleteReqState(reqID string) {
	c.reqStatesMu.Lock()
	delete(c.reqStates, reqID)
	c.reqStatesMu.Unlock()
}

func (c *WeComAIBotWSChannel) clearReqTask(reqID string, task *wsTask) {
	c.reqStatesMu.Lock()
	defer c.reqStatesMu.Unlock()
	state, ok := c.reqStates[reqID]
	if !ok || state == nil {
		return
	}
	if state.Task == task {
		state.Task = nil
	}
}

func wsChatTypeValue(chatType string) uint32 {
	if chatType == "group" {
		return 2
	}
	return 1
}

// wsChatID returns the effective chat ID from a WS message.
// For group messages it is msg.ChatID; for single chats it falls back to the sender's UserID.
func wsChatID(msg WeComAIBotWSMessage) string {
	if msg.ChatID != "" {
		return msg.ChatID
	}
	return msg.From.UserID
}

// wsGenerateID generates a random 10-character alphanumeric ID.
// It is package-level (not a method) so it can be shared by both channel modes.
func wsGenerateID() string {
	return generateRandomID(10)
}

// ---- Inbound media download helpers ----

// storeWSMedia downloads the resource at resourceURL (with optional AES-CBC
// decryption) and stores it in the MediaStore. The file extension is inferred
// from the HTTP Content-Type response header; defaultExt is used as a fallback
// when the content type is absent or unrecognized.
func (c *WeComAIBotWSChannel) storeWSMedia(
	ctx context.Context,
	chatID, msgID, resourceURL, aesKey, defaultExt string,
) (string, error) {
	store := c.GetMediaStore()
	if store == nil {
		return "", fmt.Errorf("no media store available")
	}

	const maxSize = 20 << 20 // 20 MB

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resourceURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := wsImageHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download HTTP %d", resp.StatusCode)
	}

	// Infer file extension from the Content-Type response header.
	ext := wsMediaExtFromContentType(resp.Header.Get("Content-Type"))
	if ext == "" {
		ext = defaultExt
	}

	// Buffer the media in memory, bounded to maxSize.
	data, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxSize)+1))
	if err != nil {
		return "", fmt.Errorf("read media: %w", err)
	}
	if len(data) > maxSize {
		return "", fmt.Errorf("media too large (> %d MB)", maxSize>>20)
	}

	// AES-CBC decryption if a key is present.
	if aesKey != "" {
		key, decErr := base64.StdEncoding.DecodeString(aesKey)
		if decErr != nil || len(key) != 32 {
			key, decErr = decodeWeComAESKey(aesKey)
			if decErr != nil {
				return "", fmt.Errorf("decode media AES key: %w", decErr)
			}
		}
		data, err = decryptAESCBC(key, data)
		if err != nil {
			return "", fmt.Errorf("decrypt media: %w", err)
		}
	}

	// Write to a temp file. The file is owned by the MediaStore and deleted by
	// store.ReleaseAll — no caller-side cleanup needed.
	mediaDir := filepath.Join(os.TempDir(), "picoclaw_media")
	if err = os.MkdirAll(mediaDir, 0o700); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	tmpFile, err := os.CreateTemp(mediaDir, msgID+"-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_, writeErr := tmpFile.Write(data)
	closeErr := tmpFile.Close()
	if writeErr != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("write media: %w", writeErr)
	}
	if closeErr != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("close media: %w", closeErr)
	}

	scope := channels.BuildMediaScope("wecom_aibot", chatID, msgID)
	ref, err := store.Store(tmpPath, media.MediaMeta{
		Filename:      msgID + ext,
		Source:        "wecom_aibot",
		CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
	}, scope)
	if err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("store: %w", err)
	}
	return ref, nil
}

// wsMediaExtFromContentType returns the lowercase file extension (with leading
// dot) for the given Content-Type value, or "" when the type is unrecognized.
func wsMediaExtFromContentType(contentType string) string {
	if contentType == "" {
		return ""
	}
	// Strip parameters (e.g. "image/jpeg; charset=utf-8" → "image/jpeg").
	mt := strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	switch mt {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/mpeg", "video/x-mpeg":
		return ".mpeg"
	case "video/quicktime":
		return ".mov"
	case "video/webm":
		return ".webm"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	case "audio/wav":
		return ".wav"
	case "application/pdf":
		return ".pdf"
	case "application/zip":
		return ".zip"
	case "application/x-rar-compressed", "application/vnd.rar":
		return ".rar"
	case "text/plain":
		return ".txt"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "application/vnd.ms-excel":
		return ".xls"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx"
	case "application/vnd.ms-powerpoint":
		return ".ppt"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return ".pptx"
	}
	return ""
}

// wsLabelToDefaultExt returns the default file extension for the given media label
// used in wsHandleMediaMessage. It is the fallback when Content-Type detection fails.
func wsLabelToDefaultExt(label string) string {
	switch label {
	case "image":
		return ".jpg"
	case "video":
		return ".mp4"
	default: // "file" and any future labels
		return ".bin"
	}
}

// ---- Content length helpers ----

// splitWSContent splits content into chunks each fitting within maxBytes UTF-8
// bytes, preserving code block integrity via channels.SplitMessage.
// When SplitMessage still produces an oversized chunk (e.g. dense CJK content),
// splitAtByteBoundary is applied as a last-resort byte-level fallback.
func splitWSContent(content string, maxBytes int) []string {
	if len(content) <= maxBytes {
		return []string{content}
	}
	// SplitMessage works in runes. Use maxBytes as the rune limit: for pure ASCII
	// this is exact; for multibyte content the byte verification below catches
	// any chunk that still overflows.
	chunks := channels.SplitMessage(content, maxBytes)
	var result []string
	for _, chunk := range chunks {
		if len(chunk) <= maxBytes {
			result = append(result, chunk)
		} else {
			// Still too large in bytes (e.g. dense CJK); force-split at UTF-8 boundaries.
			result = append(result, splitAtByteBoundary(chunk, maxBytes)...)
		}
	}
	return result
}

// splitAtByteBoundary splits s into parts each ≤ maxBytes bytes by walking back
// from the hard byte limit to find a valid UTF-8 rune start boundary.
// This is a last-resort fallback; it does not try to preserve code blocks.
func splitAtByteBoundary(s string, maxBytes int) []string {
	var parts []string
	for len(s) > maxBytes {
		end := maxBytes
		// Walk back past any UTF-8 continuation bytes (high two bits == 10).
		for end > 0 && s[end]>>6 == 0b10 {
			end--
		}
		if end == 0 {
			end = maxBytes // shouldn't happen with valid UTF-8
		}
		parts = append(parts, s[:end])
		s = strings.TrimLeft(s[end:], " \t\n\r")
	}
	if s != "" {
		parts = append(parts, s)
	}
	return parts
}
