package wecom

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	// WebSocket 默认配置
	defaultWSURL                = "wss://openws.work.weixin.qq.com"
	defaultReconnectInterval    = 5 * time.Second
	defaultHeartbeatInterval    = 30 * time.Second
	defaultReplyTimeout         = 30 * time.Second
	defaultMaxReconnectAttempts = 100

	// 消息处理超时
	messageProcessTimeout = 5 * time.Minute

	// 消息状态清理配置
	messageStateCleanupInterval = 60 * time.Second
	messageStateMaxSize         = 500
	messageStateTTL             = 10 * time.Minute

	// 文本分块大小
	textChunkLimit = 4000

	// 媒体下载配置
	defaultMediaMaxMB        = 5
	imageDownloadTimeoutMs   = 30000
	fileDownloadTimeoutMs    = 60000
	replySendTimeoutMs       = 15000
	thinkingMessage          = "<think></think>"
	mediaImagePlaceholder    = "<media:image>"
	mediaDocumentPlaceholder = "<media:document>"
)

// WeComCommand 企业微信 WebSocket 命令类型
type WeComCommand string

const (
	// 认证订阅
	CmdSubscribe WeComCommand = "aibot_subscribe"
	// 心跳
	CmdPing WeComCommand = "ping"
	// 企业微信推送消息
	CmdAIBotCallback WeComCommand = "aibot_callback"
	// 企业微信事件回调
	CmdAIBotEventCallback WeComCommand = "aibot_event_callback"
	// 企业微信消息回调
	CmdAIBotMsgCallback WeComCommand = "aibot_msg_callback"
	// picoclaw 响应消息（官方命令名：aibot_respond_msg）
	CmdAIBotResponse WeComCommand = "aibot_respond_msg"
)

// WeComWSMessage WebSocket 消息基础结构
type WeComWSMessage struct {
	Cmd     string          `json:"cmd"`
	Headers MessageHeaders  `json:"headers"`
	Body    json.RawMessage `json:"body"`
}

// MessageHeaders 消息头
type MessageHeaders struct {
	ReqID string `json:"req_id"`
}

// SubscribeRequest 订阅认证请求
type SubscribeRequest struct {
	Secret string `json:"secret"`
	BotID  string `json:"bot_id"`
}

// SubscribeBody 订阅请求体
type SubscribeBody struct {
	Secret string `json:"secret"`
	BotID  string `json:"bot_id"`
}

// CallbackMessage 企业微信推送消息
type CallbackMessage struct {
	MsgID       string  `json:"msgid"`
	AIBotID     string  `json:"aibotid"`
	ChatID      string  `json:"chatid"`
	ChatType    string  `json:"chattype"`
	From        From    `json:"from"`
	ResponseURL string  `json:"response_url"`
	MsgType     string  `json:"msgtype"`
	Text        *Text   `json:"text,omitempty"`
	Image       *Image  `json:"image,omitempty"`
	Voice       *Voice  `json:"voice,omitempty"`
	Video       *Video  `json:"video,omitempty"`
	File        *File   `json:"file,omitempty"`
	Mixed       *Mixed  `json:"mixed,omitempty"`
	Quote       *Quote  `json:"quote,omitempty"`
	Stream      *Stream `json:"stream,omitempty"`
}

// Stream 流式消息
type Stream struct {
	ID string `json:"id"`
}

// From 发送者信息
type From struct {
	UserID string `json:"userid"`
}

// Text 文本消息
type Text struct {
	Content string `json:"content"`
}

// Image 图片消息
type Image struct {
	URL    string `json:"url,omitempty"`
	Base64 string `json:"base64,omitempty"`
	MD5    string `json:"md5,omitempty"`
	AESKey string `json:"aeskey,omitempty"`
}

// Voice 语音消息
type Voice struct {
	Content string `json:"content,omitempty"` // 语音转文字后的内容
	URL     string `json:"url,omitempty"`
	AESKey  string `json:"aeskey,omitempty"`
}

// Video 视频消息
type Video struct {
	URL    string `json:"url,omitempty"`
	AESKey string `json:"aeskey,omitempty"`
}

// File 文件消息
type File struct {
	URL      string `json:"url,omitempty"`
	Filename string `json:"filename,omitempty"`
	AESKey   string `json:"aeskey,omitempty"`
}

// Quote 引用消息
type Quote struct {
	MsgType string `json:"msgtype"`
	Text    *Text  `json:"text,omitempty"`
	Voice   *Voice `json:"voice,omitempty"`
	Image   *Image `json:"image,omitempty"`
	File    *File  `json:"file,omitempty"`
}

// Mixed 图文混排消息
type Mixed struct {
	MsgItem []MixedItem `json:"msg_item"`
}

// MixedItem 混排消息项
type MixedItem struct {
	MsgType string `json:"msgtype"`
	Text    *Text  `json:"text,omitempty"`
	Image   *Image `json:"image,omitempty"`
}

// ResponseMessage 响应消息
type ResponseMessage struct {
	MsgType  string         `json:"msgtype"`
	Stream   *StreamContent `json:"stream,omitempty"`
	Text     *Text          `json:"text,omitempty"`
	Markdown *Markdown      `json:"markdown,omitempty"`
	Image    *ResponseImage `json:"image,omitempty"`
	File     *ResponseFile  `json:"file,omitempty"`
}

// StreamContent 流式内容
type StreamContent struct {
	ID       string          `json:"id"`
	Finish   bool            `json:"finish"`
	Content  string          `json:"content"`
	MsgItem  []StreamMsgItem `json:"msg_item,omitempty"`
	Feedback *StreamFeedback `json:"feedback,omitempty"`
}

// StreamMsgItem 流式消息中的媒体项
type StreamMsgItem struct {
	MsgType string         `json:"msgtype"`
	Image   *ResponseImage `json:"image,omitempty"`
	File    *ResponseFile  `json:"file,omitempty"`
}

// StreamFeedback 流式反馈
type StreamFeedback struct {
	ID string `json:"id"`
}

// ResponseImage 响应图片
type ResponseImage struct {
	Base64 string `json:"base64"`
	MD5    string `json:"md5"`
}

// ResponseFile 响应文件
type ResponseFile struct {
	Base64   string `json:"base64"`
	Filename string `json:"filename"`
}

// Markdown Markdown 消息
type Markdown struct {
	Content string `json:"content"`
}

// MessageState 消息状态（用于流式回复）
type MessageState struct {
	AccumulatedText string
	StreamID        string
	ReqID           string // 透传收到的 req_id
	CreatedAt       time.Time
}

// ParsedMessageContent 解析后的消息内容
type ParsedMessageContent struct {
	TextParts    []string
	ImageURLs    []string
	ImageAESKeys map[string]string // URL -> AES key
	FileURLs     []string
	FileAESKeys  map[string]string // URL -> AES key
	QuoteContent string
	MediaList    []MediaInfo
}

// MediaInfo 媒体信息
type MediaInfo struct {
	URL         string
	Type        string // "image" or "file"
	ContentType string
	Filename    string
	Path        string // 本地缓存路径
	AESKey      string
}

// WeComWSChannel 企业微信 WebSocket Channel 实现
type WeComWSChannel struct {
	*channels.BaseChannel
	config     config.WeComWSConfig
	ctx        context.Context
	cancel     context.CancelFunc
	wsConn     *websocket.Conn
	connMu     sync.RWMutex
	reconnects int

	// 消息状态管理
	messageStates map[string]*MessageState
	statesMu      sync.RWMutex

	// 发送队列
	sendCh chan *WeComWSMessage

	// 连接状态
	connected bool

	// 群组管理器
	groupManager *GroupManager

	// req_id 存储（持久化去重）
	reqIDStore *ReqIDStore

	// 事件管理器
	eventManager *EventManager
}

// NewWeComWSChannel 创建新的 WeCom WebSocket Channel
func NewWeComWSChannel(cfg config.WeComWSConfig, messageBus *bus.MessageBus) (*WeComWSChannel, error) {
	if cfg.BotID == "" || cfg.Secret == "" {
		return nil, fmt.Errorf("wecom_ws bot_id and secret are required")
	}

	// 设置默认值
	if cfg.WSURL == "" {
		cfg.WSURL = defaultWSURL
	}
	if cfg.ReconnectInterval <= 0 {
		cfg.ReconnectInterval = int(defaultReconnectInterval.Seconds())
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = int(defaultHeartbeatInterval.Seconds())
	}
	if cfg.ReplyTimeout <= 0 {
		cfg.ReplyTimeout = int(defaultReplyTimeout.Seconds())
	}
	if cfg.MaxReconnectAttempts <= 0 {
		cfg.MaxReconnectAttempts = defaultMaxReconnectAttempts
	}

	base := channels.NewBaseChannel("wecom_ws", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(textChunkLimit),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	ctx, cancel := context.WithCancel(context.Background())

	ch := &WeComWSChannel{
		BaseChannel:   base,
		config:        cfg,
		ctx:           ctx,
		cancel:        cancel,
		messageStates: make(map[string]*MessageState),
		sendCh:        make(chan *WeComWSMessage, 100),
		groupManager:  NewGroupManager(cfg.GroupPolicies),
		reqIDStore:    NewReqIDStore(cfg.PersistencePath),
		eventManager:  NewEventManager(),
	}

	ch.SetOwner(ch)
	return ch, nil
}

// Name 返回 Channel 名称
func (c *WeComWSChannel) Name() string {
	return "wecom_ws"
}

// Start 启动 WebSocket 连接
func (c *WeComWSChannel) Start(ctx context.Context) error {
	logger.InfoC("wecom_ws", "Starting WeCom WebSocket channel...")

	// 取消旧的 context（如果存在）
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(ctx)

	// 启动 req_id 自动保存
	if c.reqIDStore != nil {
		c.reqIDStore.StartAutoSave(5 * time.Minute)
	}

	// 启动连接管理
	go c.connectionManager()

	// 启动消息状态清理
	go c.cleanupLoop()

	c.SetRunning(true)
	logger.InfoC("wecom_ws", "WeCom WebSocket channel started")
	return nil
}

// Stop 停止 WebSocket 连接
func (c *WeComWSChannel) Stop(ctx context.Context) error {
	logger.InfoC("wecom_ws", "Stopping WeCom WebSocket channel...")

	if c.cancel != nil {
		c.cancel()
	}

	c.closeConnection()

	// 保存 req_id 存储
	if c.reqIDStore != nil {
		if err := c.reqIDStore.Stop(); err != nil {
			logger.ErrorCF("wecom_ws", "Failed to save req_id store", map[string]any{
				"error": err.Error(),
			})
		}
	}

	c.SetRunning(false)
	logger.InfoC("wecom_ws", "WeCom WebSocket channel stopped")
	return nil
}

// Send 发送消息到企业微信
func (c *WeComWSChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	c.connMu.RLock()
	connected := c.connected
	c.connMu.RUnlock()

	if !connected {
		return fmt.Errorf("websocket not connected: %w", channels.ErrTemporary)
	}

	logger.DebugCF("wecom_ws", "Sending message", map[string]any{
		"chat_id": msg.ChatID,
		"preview": utils.Truncate(msg.Content, 100),
	})

	// 获取或创建消息状态
	state := c.getOrCreateMessageState(msg.ChatID)

	// 检查是否为 Markdown 内容
	isMarkdown := c.isMarkdownContent(msg.Content)

	// 发送文本/Markdown 内容
	if msg.Content != "" {
		// 分块发送长消息
		chunks := c.splitMessage(msg.Content)
		for i, chunk := range chunks {
			isLast := i == len(chunks)-1

			var response *ResponseMessage
			if isMarkdown {
				// Markdown 格式响应
				response = &ResponseMessage{
					MsgType: "markdown",
					Markdown: &Markdown{
						Content: chunk,
					},
				}
			} else {
				// 流式文本响应
				response = &ResponseMessage{
					MsgType: "stream",
					Stream: &StreamContent{
						ID:      state.StreamID,
						Finish:  isLast,
						Content: chunk,
					},
				}
			}

			body, err := json.Marshal(response)
			if err != nil {
				return fmt.Errorf("failed to marshal response: %w", err)
			}

			// 企业微信 WS 规范：回复时需要透传收到消息的 req_id
			reqID := state.ReqID
			if reqID == "" {
				reqID = generateReqID()
			}
			wsMsg := &WeComWSMessage{
				Cmd: string(CmdAIBotResponse),
				Headers: MessageHeaders{
					ReqID: reqID,
				},
				Body: body,
			}

			if err := c.sendMessage(wsMsg); err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}

			// 非最后一条消息时添加小延迟，避免消息顺序混乱
			if !isLast {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	// 清理消息状态
	c.deleteMessageState(msg.ChatID)

	// 触发发送消息事件
	msgType := "stream"
	if isMarkdown {
		msgType = "markdown"
	}
	c.eventManager.Emit(EventMessageSent, EventPayloadMessageSent{
		MsgType: msgType,
		ChatID:  msg.ChatID,
	})

	return nil
}

// connectionManager 管理 WebSocket 连接生命周期
func (c *WeComWSChannel) connectionManager() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		if err := c.connect(); err != nil {
			logger.ErrorCF("wecom_ws", "Connection failed", map[string]any{
				"error": err.Error(),
			})

			c.reconnects++
			if c.config.MaxReconnectAttempts > 0 && c.reconnects > c.config.MaxReconnectAttempts {
				logger.ErrorC("wecom_ws", "Max reconnection attempts reached, but continuing to loop to ensure service alive")
				// 触发错误事件
				c.eventManager.Emit(EventError, EventPayloadError{
					Error: fmt.Errorf("max reconnection attempts reached"),
				})
				// 不退出，只保留最大延迟
				c.reconnects = c.config.MaxReconnectAttempts
			}

			// 指数退避重连
			backoff := time.Duration(c.config.ReconnectInterval) * time.Second
			if c.reconnects > 5 {
				backoff = time.Duration(c.config.ReconnectInterval*c.reconnects/5) * time.Second
			}
			if backoff > 60*time.Second {
				backoff = 60 * time.Second
			}

			// 触发重连事件
			c.eventManager.Emit(EventReconnecting, EventPayloadReconnecting{
				URL:       c.config.WSURL,
				Attempt:   c.reconnects,
				BackoffMs: int(backoff.Milliseconds()),
			})

			logger.InfoCF("wecom_ws", "Reconnecting...", map[string]any{
				"attempt": c.reconnects,
				"backoff": backoff.Seconds(),
			})
			time.Sleep(backoff)
			continue
		}

		// 连接成功，重置重连计数
		c.reconnects = 0

		// 使用上下文控制这部分的读写协程生命周期
		connCtx, connCancel := context.WithCancel(c.ctx)

		// 启动读写协程
		var wg sync.WaitGroup
		wg.Add(1)

		// 清空旧的发送队列，防止将上一次遗留的数据发给新建立的连接
	drainLoop:
		for {
			select {
			case <-c.sendCh:
			default:
				break drainLoop
			}
		}

		// 先启动 writeLoop，确保订阅消息能被发送
		go func() {
			defer wg.Done()
			defer connCancel()
			c.writeLoop(connCtx)
		}()

		// 等待一小段时间确保 writeLoop 已启动
		time.Sleep(100 * time.Millisecond)

		// 发送认证订阅消息
		if err := c.subscribe(); err != nil {
			logger.ErrorCF("wecom_ws", "Subscription failed", map[string]any{
				"error": err.Error(),
			})
			c.closeConnection()
			connCancel()
			wg.Wait()
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer connCancel()
			c.readLoop(connCtx)
		}()

		// 等待读写协程结束
		wg.Wait()

		c.closeConnection()
		logger.InfoC("wecom_ws", "Connection closed, will reconnect...")
	}
}

// connect 建立 WebSocket 连接
func (c *WeComWSChannel) connect() error {
	logger.InfoCF("wecom_ws", "Connecting to WebSocket", map[string]any{
		"url":    c.config.WSURL,
		"bot_id": c.config.BotID,
	})

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		// 使用默认 TLS 配置
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	// 添加自定义请求头
	headers := http.Header{}

	logger.DebugC("wecom_ws", "Dialing WebSocket...")
	conn, resp, err := dialer.DialContext(c.ctx, c.config.WSURL, headers)
	if err != nil {
		logger.ErrorCF("wecom_ws", "WebSocket dial error", map[string]any{
			"error": err.Error(),
			"url":   c.config.WSURL,
		})
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	if resp != nil {
		logger.DebugCF("wecom_ws", "WebSocket handshake response", map[string]any{
			"status": resp.Status,
			"code":   resp.StatusCode,
		})
		resp.Body.Close()
	}

	c.connMu.Lock()
	c.wsConn = conn
	c.connected = true
	c.connMu.Unlock()

	// 设置心跳超时和标准 ping/pong 处理器 (处理底层心跳)
	deadline := time.Duration(c.config.HeartbeatInterval) * 3 * time.Second
	conn.SetReadDeadline(time.Now().Add(deadline))
	
	originalPingHandler := conn.PingHandler()
	conn.SetPingHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(deadline))
		if originalPingHandler != nil {
			return originalPingHandler(appData)
		}
		return nil
	})
	
	originalPongHandler := conn.PongHandler()
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(deadline))
		if originalPongHandler != nil {
			return originalPongHandler(appData)
		}
		return nil
	})

	logger.InfoC("wecom_ws", "WebSocket connected")

	// 触发连接成功事件
	c.eventManager.Emit(EventConnected, EventPayloadConnected{
		URL: c.config.WSURL,
	})

	// 订阅消息将在 connectionManager 中发送，确保 writeLoop 已启动
	return nil
}

// subscribe 发送认证订阅
func (c *WeComWSChannel) subscribe() error {
	subscribeBody := SubscribeBody{
		Secret: c.config.Secret,
		BotID:  c.config.BotID,
	}

	body, err := json.Marshal(subscribeBody)
	if err != nil {
		return fmt.Errorf("failed to marshal subscribe body: %w", err)
	}

	msg := &WeComWSMessage{
		Cmd: string(CmdSubscribe),
		Headers: MessageHeaders{
			ReqID: generateReqID(),
		},
		Body: body,
	}

	if err := c.sendMessage(msg); err != nil {
		return err
	}

	logger.InfoC("wecom_ws", "Subscription sent")

	// 触发认证成功事件
	c.eventManager.Emit(EventAuthenticated, EventPayloadAuthenticated{
		BotID: c.config.BotID,
	})

	return nil
}

// closeConnection 关闭 WebSocket 连接
func (c *WeComWSChannel) closeConnection() {
	c.connMu.Lock()
	wasConnected := c.connected
	if c.wsConn != nil {
		c.wsConn.Close()
		c.wsConn = nil
	}
	c.connected = false
	c.connMu.Unlock()

	// 触发断开连接事件
	if wasConnected {
		c.eventManager.Emit(EventDisconnected, EventPayloadDisconnected{
			URL: c.config.WSURL,
		})
	}
}

// readLoop 读取 WebSocket 消息
func (c *WeComWSChannel) readLoop(ctx context.Context) {
	defer c.closeConnection()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		c.connMu.RLock()
		conn := c.wsConn
		c.connMu.RUnlock()

		if conn == nil {
			return
		}

		// 更新读超时，心跳间隔的 3 倍，如果没有收到任何消息则主动断开
		conn.SetReadDeadline(time.Now().Add(time.Duration(c.config.HeartbeatInterval) * 3 * time.Second))

		// 先读取原始消息
		messageType, rawMessage, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.ErrorCF("wecom_ws", "WebSocket read error", map[string]any{
					"error": err.Error(),
				})
			}
			return
		}

		logger.DebugCF("wecom_ws", "Raw message received", map[string]any{
			"message_type": messageType,
			"raw_data":     string(rawMessage),
		})

		// 解析 JSON 消息
		var msg WeComWSMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			logger.WarnCF("wecom_ws", "Failed to unmarshal message", map[string]any{
				"error":    err.Error(),
				"raw_data": string(rawMessage),
				"msg_type": messageType,
			})
			continue
		}

		c.handleMessage(&msg)
	}
}

// writeLoop 写入 WebSocket 消息
func (c *WeComWSChannel) writeLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(c.config.HeartbeatInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-c.sendCh:
			c.connMu.RLock()
			conn := c.wsConn
			c.connMu.RUnlock()

			if conn == nil {
				logger.DebugC("wecom_ws", "Connection is nil, exiting writeLoop")
				return
			}

			// 更新写超时
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteJSON(msg); err != nil {
				logger.ErrorCF("wecom_ws", "WebSocket write error", map[string]any{
					"error": err.Error(),
				})
				c.closeConnection()
				return
			}

		case <-ticker.C:
			c.connMu.RLock()
			conn := c.wsConn
			c.connMu.RUnlock()
			
			if conn == nil {
				logger.DebugC("wecom_ws", "Connection is nil, exiting writeLoop on ticker")
				return
			}

			// 发送心跳
			if err := c.sendPing(); err != nil {
				logger.ErrorCF("wecom_ws", "Ping failed", map[string]any{
					"error": err.Error(),
				})
				return
			}
		}
	}
}

// sendMessage 发送消息到发送队列
func (c *WeComWSChannel) sendMessage(msg *WeComWSMessage) error {
	select {
	case c.sendCh <- msg:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("send timeout")
	}
}

// sendPing 发送心跳
func (c *WeComWSChannel) sendPing() error {
	msg := &WeComWSMessage{
		Cmd: string(CmdPing),
		Headers: MessageHeaders{
			ReqID: generateReqID(),
		},
		Body: json.RawMessage("{}"),
	}
	return c.sendMessage(msg)
}

// handleMessage 处理收到的消息
func (c *WeComWSChannel) handleMessage(msg *WeComWSMessage) {
	logger.DebugCF("wecom_ws", "Received message", map[string]any{
		"cmd":      msg.Cmd,
		"req_id":   msg.Headers.ReqID,
		"body_len": len(msg.Body),
	})

	// 处理空命令（可能是心跳或空消息）
	if msg.Cmd == "" {
		logger.DebugC("wecom_ws", "Empty command received, ignoring")
		return
	}

	switch WeComCommand(msg.Cmd) {
	case CmdPing:
		// 心跳响应，无需处理
		logger.DebugC("wecom_ws", "Ping received")

	case CmdAIBotCallback, CmdAIBotMsgCallback:
		// 企业微信推送消息
		go c.handleCallback(msg)

	case CmdAIBotEventCallback:
		// 企业微信事件回调
		go c.handleEventCallback(msg)

	default:
		logger.WarnCF("wecom_ws", "Unknown command", map[string]any{
			"cmd": msg.Cmd,
		})
	}
}

// handleEventCallback 处理企业微信事件回调
func (c *WeComWSChannel) handleEventCallback(msg *WeComWSMessage) {
	logger.DebugCF("wecom_ws", "Event callback received", map[string]any{
		"body": string(msg.Body),
	})
	// 事件回调通常不需要处理，只需记录日志
}

// handleCallback 处理企业微信推送的消息
func (c *WeComWSChannel) handleCallback(msg *WeComWSMessage) {
	var callback CallbackMessage
	if err := json.Unmarshal(msg.Body, &callback); err != nil {
		logger.ErrorCF("wecom_ws", "Failed to unmarshal callback", map[string]any{
			"error": err.Error(),
		})
		return
	}

	// 检查消息是否重复（使用 msgid 去重）
	if c.reqIDStore != nil && c.reqIDStore.Exists(callback.MsgID) {
		logger.DebugCF("wecom_ws", "Duplicate message ignored", map[string]any{
			"msg_id": callback.MsgID,
		})
		return
	}

	// 记录 req_id
	if c.reqIDStore != nil {
		c.reqIDStore.Add(callback.MsgID)
	}

	logger.DebugCF("wecom_ws", "Handling callback", map[string]any{
		"msg_type": callback.MsgType,
		"chat_id":  callback.ChatID,
		"from":     callback.From.UserID,
	})

	// 触发收到消息事件
	c.eventManager.Emit(EventMessageReceived, EventPayloadMessageReceived{
		MsgType: callback.MsgType,
		ChatID:  callback.ChatID,
		From:    callback.From.UserID,
	})

	// 保存 req_id 以便回复时透传（企业微信 WS 规范要求）
	// chatID: 群聊用 chatid，单聊用 userid
	chatIDForState := callback.ChatID
	if chatIDForState == "" {
		chatIDForState = callback.From.UserID
	}
	if chatIDForState != "" && msg.Headers.ReqID != "" {
		state := c.getOrCreateMessageState(chatIDForState)
		state.ReqID = msg.Headers.ReqID
	}

	// 使用超时控制包装消息处理
	err := withTimeout(c.ctx, messageProcessTimeout, func() error {
		// 解析消息内容
		parsedContent := c.parseMessageContent(&callback)

		// 添加详细的消息处理日志
		logger.DebugCF("wecom_ws", "Processing message", map[string]any{
			"chat_type":   callback.ChatType,
			"chat_id":     callback.ChatID,
			"user_id":     callback.From.UserID,
			"msg_id":      callback.MsgID,
			"text_parts":  len(parsedContent.TextParts),
			"media_count": len(parsedContent.MediaList),
			"has_quote":   parsedContent.QuoteContent != "",
		})

		// 检查是否为空消息（没有文本、媒体、引用）
		if len(parsedContent.TextParts) == 0 &&
			len(parsedContent.MediaList) == 0 &&
			parsedContent.QuoteContent == "" {
			logger.DebugC("wecom_ws", "Skipping empty message (no text, image, file or quote)")
			return nil
		}

		// 下载媒体文件
		if len(parsedContent.MediaList) > 0 {
			parsedContent.MediaList = c.downloadAndSaveMediaList(parsedContent.MediaList)
		}

		// 根据消息类型处理
		switch callback.MsgType {
		case "text":
			c.handleTextMessage(&callback, parsedContent)
		case "image":
			c.handleImageMessage(&callback, parsedContent)
		case "voice":
			c.handleVoiceMessage(&callback, parsedContent)
		case "video":
			c.handleVideoMessage(&callback, parsedContent)
		case "file":
			c.handleFileMessage(&callback, parsedContent)
		case "mixed":
			c.handleMixedMessage(&callback, parsedContent)
		case "quote":
			c.handleQuoteMessage(&callback, parsedContent)
		case "stream":
			// 流式消息轮询，更新消息状态
			c.handleStreamMessage(&callback)
		default:
			logger.WarnCF("wecom_ws", "Unsupported message type", map[string]any{
				"msg_type": callback.MsgType,
			})
		}
		return nil
	})

	if err != nil {
		logger.ErrorCF("wecom_ws", "Message processing timed out or failed", map[string]any{
			"msg_id": callback.MsgID,
			"error":  err.Error(),
		})
		// 触发错误事件
		c.eventManager.Emit(EventError, EventPayloadError{
			Error: fmt.Errorf("message processing failed: %w", err),
		})
	}
}

// handleTextMessage 处理文本消息
func (c *WeComWSChannel) handleTextMessage(msg *CallbackMessage, parsedContent *ParsedMessageContent) {
	if msg.Text == nil && len(parsedContent.TextParts) == 0 {
		return
	}

	content := msg.Text.Content
	if len(parsedContent.TextParts) > 0 {
		// 使用解析后的文本（可能包含语音转文字内容）
		content = ""
		for _, part := range parsedContent.TextParts {
			content += part + " "
		}
		content = content[:len(content)-1] // 去掉最后的空格
	}

	senderID := msg.From.UserID
	chatID := msg.ChatID
	if chatID == "" {
		chatID = senderID
	}

	// 如果文本为空但存在引用消息，使用引用消息内容
	if content == "" && parsedContent.QuoteContent != "" {
		content = parsedContent.QuoteContent
		logger.DebugC("wecom_ws", "Using quote content as message body (user only mentioned bot)")
	}

	// 判断是否为群聊
	isGroupChat := msg.ChatType == "group"

	// 群聊触发检查 - 使用群组管理器
	if isGroupChat {
		// 移除 @提及标记（如 @机器人）
		content = regexp.MustCompile(`@\S+`).ReplaceAllString(content, "")
		content = strings.TrimSpace(content)

		// 检查用户是否在群组白名单中
		if !c.groupManager.IsAllowedInGroup(chatID, senderID) {
			logger.DebugCF("wecom_ws", "User not allowed in group", map[string]any{
				"group_id":  chatID,
				"sender_id": senderID,
			})
			return
		}

		// 检查是否应该响应
		respond, cleaned := c.groupManager.ShouldRespondInGroup(chatID, false, content)
		if !respond {
			return
		}
		content = cleaned
	}

	// 初始化消息状态
	state := c.getOrCreateMessageState(chatID)
	state.StreamID = generateStreamID()

	// 发送"思考中"消息（如果启用）
	if c.config.SendThinkingMessage {
		c.sendThinkingMessage(chatID, state.StreamID)
	}

	// 构建 metadata
	metadata := map[string]string{
		"msg_type":     "text",
		"msg_id":       msg.MsgID,
		"platform":     "wecom_ws",
		"response_url": msg.ResponseURL,
		"stream_id":    state.StreamID,
	}
	if isGroupChat {
		metadata["chat_id"] = msg.ChatID
		metadata["sender_id"] = senderID
	}

	// 添加媒体信息到 metadata
	if len(parsedContent.MediaList) > 0 {
		var mediaPaths []string
		var mediaTypes []string

		for _, media := range parsedContent.MediaList {
			if media.Path != "" {
				mediaPaths = append(mediaPaths, media.Path)
				mediaTypes = append(mediaTypes, media.ContentType)
			}
		}

		// 向后兼容：单个媒体
		if len(mediaPaths) > 0 {
			metadata["media_path"] = mediaPaths[0]
			metadata["media_type"] = mediaTypes[0]
		}

		// 新功能：多个媒体数组
		if len(mediaPaths) > 0 {
			metadata["media_paths"] = strings.Join(mediaPaths, ",")
			metadata["media_types"] = strings.Join(mediaTypes, ",")
		}
	}

	// 添加引用消息内容
	if parsedContent.QuoteContent != "" {
		metadata["reply_to_body"] = parsedContent.QuoteContent
	}

	// 构建 sender
	sender := bus.SenderInfo{
		Platform:    "wecom_ws",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("wecom_ws", senderID),
	}

	if !c.IsAllowedSender(sender) {
		return
	}

	// 确定 peer
	peerKind := "direct"
	if isGroupChat {
		peerKind = "group"
	}
	peer := bus.Peer{Kind: peerKind, ID: chatID}

	logger.DebugCF("wecom_ws", "Publishing message", map[string]any{
		"sender_id": senderID,
		"chat_id":   chatID,
		"content":   utils.Truncate(content, 50),
	})

	// 发布消息到 bus
	c.HandleMessage(c.ctx, peer, msg.MsgID, senderID, chatID, content, nil, metadata, sender)
}

// handleImageMessage 处理图片消息
func (c *WeComWSChannel) handleImageMessage(msg *CallbackMessage, parsedContent *ParsedMessageContent) {
	logger.DebugCF("wecom_ws", "Handling image message", map[string]any{
		"chat_id":     msg.ChatID,
		"image_count": len(parsedContent.ImageURLs),
	})

	// 图片已经在 parseMessageContent 和 downloadAndSaveMediaList 中处理
	// 这里只需要记录日志，实际的媒体路径已经在 parsedContent.MediaList 中
	for _, media := range parsedContent.MediaList {
		if media.Type == "image" && media.Path != "" {
			logger.DebugCF("wecom_ws", "Image saved", map[string]any{
				"url":  media.URL,
				"path": media.Path,
			})
		}
	}
}

// handleVoiceMessage 处理语音消息
func (c *WeComWSChannel) handleVoiceMessage(msg *CallbackMessage, parsedContent *ParsedMessageContent) {
	// 语音消息的内容已经在 parseMessageContent 中提取为文本
	// 直接复用文本消息处理
	if len(parsedContent.TextParts) > 0 {
		c.handleTextMessage(msg, parsedContent)
	}
}

// handleVideoMessage 处理视频消息
func (c *WeComWSChannel) handleVideoMessage(msg *CallbackMessage, parsedContent *ParsedMessageContent) {
	logger.DebugCF("wecom_ws", "Handling video message", map[string]any{
		"chat_id": msg.ChatID,
		"videos":  len(parsedContent.MediaList),
	})

	// 视频已经在 downloadAndSaveMediaList 中处理
	for _, media := range parsedContent.MediaList {
		if media.Type == "video" && media.Path != "" {
			logger.DebugCF("wecom_ws", "Video saved", map[string]any{
				"url":  media.URL,
				"path": media.Path,
			})
		}
	}
}

// handleFileMessage 处理文件消息
func (c *WeComWSChannel) handleFileMessage(msg *CallbackMessage, parsedContent *ParsedMessageContent) {
	logger.DebugCF("wecom_ws", "Handling file message", map[string]any{
		"chat_id":    msg.ChatID,
		"file_count": len(parsedContent.FileURLs),
	})

	// 文件已经在 downloadAndSaveMediaList 中处理
	for _, media := range parsedContent.MediaList {
		if media.Type == "file" && media.Path != "" {
			logger.DebugCF("wecom_ws", "File saved", map[string]any{
				"url":      media.URL,
				"path":     media.Path,
				"filename": media.Filename,
			})
		}
	}
}

// handleMixedMessage 处理图文混排消息
func (c *WeComWSChannel) handleMixedMessage(msg *CallbackMessage, parsedContent *ParsedMessageContent) {
	// 图文混排消息已经在 parseMessageContent 中解析
	// 直接复用文本消息处理，媒体信息已经在 parsedContent 中
	c.handleTextMessage(msg, parsedContent)
}

// handleQuoteMessage 处理引用消息
func (c *WeComWSChannel) handleQuoteMessage(msg *CallbackMessage, parsedContent *ParsedMessageContent) {
	logger.DebugCF("wecom_ws", "Handling quote message", map[string]any{
		"chat_id":       msg.ChatID,
		"quote_content": utils.Truncate(parsedContent.QuoteContent, 50),
	})

	// 引用消息的内容已经在 parseMessageContent 中提取
	// 直接复用文本消息处理
	if len(parsedContent.TextParts) > 0 || parsedContent.QuoteContent != "" {
		c.handleTextMessage(msg, parsedContent)
	}
}

// handleStreamMessage 处理流式消息轮询
func (c *WeComWSChannel) handleStreamMessage(msg *CallbackMessage) {
	// 流式轮询消息，更新消息状态以继续响应
	logger.DebugCF("wecom_ws", "Stream poll received", map[string]any{
		"chat_id": msg.ChatID,
	})
}

// sendThinkingMessage 发送"思考中"消息
func (c *WeComWSChannel) sendThinkingMessage(chatID, streamID string) {
	response := &ResponseMessage{
		MsgType: "stream",
		Stream: &StreamContent{
			ID:      streamID,
			Finish:  false,
			Content: "",
		},
	}

	body, err := json.Marshal(response)
	if err != nil {
		logger.ErrorCF("wecom_ws", "Failed to marshal thinking message", map[string]any{
			"error": err.Error(),
		})
		return
	}

	wsMsg := &WeComWSMessage{
		Cmd: string(CmdAIBotResponse),
		Headers: MessageHeaders{
			ReqID: generateReqID(),
		},
		Body: body,
	}

	if err := c.sendMessage(wsMsg); err != nil {
		logger.ErrorCF("wecom_ws", "Failed to send thinking message", map[string]any{
			"error": err.Error(),
		})
	}
}

// getOrCreateMessageState 获取或创建消息状态
func (c *WeComWSChannel) getOrCreateMessageState(chatID string) *MessageState {
	c.statesMu.Lock()
	defer c.statesMu.Unlock()

	if state, exists := c.messageStates[chatID]; exists {
		return state
	}

	state := &MessageState{
		StreamID:  generateStreamID(),
		CreatedAt: time.Now(),
	}
	c.messageStates[chatID] = state
	return state
}

// deleteMessageState 删除消息状态
func (c *WeComWSChannel) deleteMessageState(chatID string) {
	c.statesMu.Lock()
	defer c.statesMu.Unlock()
	delete(c.messageStates, chatID)
}

// cleanupLoop 清理过期的消息状态
func (c *WeComWSChannel) cleanupLoop() {
	ticker := time.NewTicker(messageStateCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupMessageStates()
		case <-c.ctx.Done():
			return
		}
	}
}

// cleanupMessageStates 清理过期消息状态
func (c *WeComWSChannel) cleanupMessageStates() {
	c.statesMu.Lock()
	defer c.statesMu.Unlock()

	now := time.Now()
	for chatID, state := range c.messageStates {
		if now.Sub(state.CreatedAt) > messageStateTTL {
			delete(c.messageStates, chatID)
		}
	}

	// 如果超过最大数量，清理最旧的
	if len(c.messageStates) > messageStateMaxSize {
		// 简单的清理策略：删除一半
		count := 0
		for chatID := range c.messageStates {
			if count >= messageStateMaxSize/2 {
				break
			}
			delete(c.messageStates, chatID)
			count++
		}
	}
}

// splitMessage 分割长消息
func (c *WeComWSChannel) splitMessage(content string) []string {
	if len(content) <= textChunkLimit {
		return []string{content}
	}

	var chunks []string
	runes := []rune(content)
	for len(runes) > 0 {
		end := textChunkLimit
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[:end]))
		runes = runes[end:]
	}
	return chunks
}

// generateReqID 生成请求 ID
func generateReqID() string {
	return generateRandomString(16)
}

// generateStreamID 生成流 ID
func generateStreamID() string {
	return generateRandomString(10)
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[n.Int64()]
	}
	return string(b)
}

// parseMessageContent 解析消息内容
func (c *WeComWSChannel) parseMessageContent(msg *CallbackMessage) *ParsedMessageContent {
	result := &ParsedMessageContent{
		TextParts:    []string{},
		ImageURLs:    []string{},
		ImageAESKeys: make(map[string]string),
		FileURLs:     []string{},
		FileAESKeys:  make(map[string]string),
		MediaList:    []MediaInfo{},
	}

	switch msg.MsgType {
	case "text":
		if msg.Text != nil && msg.Text.Content != "" {
			result.TextParts = append(result.TextParts, msg.Text.Content)
		}

	case "image":
		if msg.Image != nil && msg.Image.URL != "" {
			result.ImageURLs = append(result.ImageURLs, msg.Image.URL)
			if msg.Image.AESKey != "" {
				result.ImageAESKeys[msg.Image.URL] = msg.Image.AESKey
			}
			result.MediaList = append(result.MediaList, MediaInfo{
				URL:    msg.Image.URL,
				Type:   "image",
				AESKey: msg.Image.AESKey,
			})
		}

	case "voice":
		if msg.Voice != nil && msg.Voice.Content != "" {
			// 语音转文字后的内容作为文本
			result.TextParts = append(result.TextParts, msg.Voice.Content)
		}

	case "video":
		if msg.Video != nil && msg.Video.URL != "" {
			result.MediaList = append(result.MediaList, MediaInfo{
				URL:    msg.Video.URL,
				Type:   "video",
				AESKey: msg.Video.AESKey,
			})
		}

	case "file":
		if msg.File != nil && msg.File.URL != "" {
			result.FileURLs = append(result.FileURLs, msg.File.URL)
			if msg.File.AESKey != "" {
				result.FileAESKeys[msg.File.URL] = msg.File.AESKey
			}
			result.MediaList = append(result.MediaList, MediaInfo{
				URL:      msg.File.URL,
				Type:     "file",
				Filename: msg.File.Filename,
				AESKey:   msg.File.AESKey,
			})
		}

	case "mixed":
		if msg.Mixed != nil {
			for _, item := range msg.Mixed.MsgItem {
				switch item.MsgType {
				case "text":
					if item.Text != nil && item.Text.Content != "" {
						result.TextParts = append(result.TextParts, item.Text.Content)
					}
				case "image":
					if item.Image != nil && item.Image.URL != "" {
						result.ImageURLs = append(result.ImageURLs, item.Image.URL)
						if item.Image.AESKey != "" {
							result.ImageAESKeys[item.Image.URL] = item.Image.AESKey
						}
						result.MediaList = append(result.MediaList, MediaInfo{
							URL:    item.Image.URL,
							Type:   "image",
							AESKey: item.Image.AESKey,
						})
					}
				}
			}
		}

	case "quote":
		if msg.Quote != nil {
			// 提取引用消息的内容
			switch msg.Quote.MsgType {
			case "text":
				if msg.Quote.Text != nil && msg.Quote.Text.Content != "" {
					result.QuoteContent = msg.Quote.Text.Content
				}
			case "voice":
				if msg.Quote.Voice != nil && msg.Quote.Voice.Content != "" {
					result.QuoteContent = msg.Quote.Voice.Content
				}
			}
		}
	}

	return result
}

// withTimeout 带超时的函数执行
func withTimeout(ctx context.Context, timeout time.Duration, fn func() error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("operation timed out after %v", timeout)
	}
}

// downloadMediaWithTimeout 带超时的媒体下载
func (c *WeComWSChannel) downloadMediaWithTimeout(url string, timeout time.Duration, maxSizeMB int) ([]byte, error) {
	var result []byte
	var downloadErr error

	err := withTimeout(c.ctx, timeout, func() error {
		result, downloadErr = c.downloadMedia(url, timeout, maxSizeMB)
		return downloadErr
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// downloadMedia 下载媒体文件
func (c *WeComWSChannel) downloadMedia(url string, timeout time.Duration, maxSizeMB int) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("empty URL")
	}

	logger.DebugCF("wecom_ws", "Downloading media", map[string]any{
		"url":     url,
		"timeout": timeout.Seconds(),
	})

	// 创建带超时的 HTTP 客户端
	client := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequestWithContext(c.ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 检查文件大小
	maxBytes := int64(maxSizeMB) * 1024 * 1024
	if resp.ContentLength > maxBytes {
		return nil, fmt.Errorf("file too large: %d bytes (max %d MB)", resp.ContentLength, maxSizeMB)
	}

	// 读取文件内容
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	logger.DebugCF("wecom_ws", "Media downloaded", map[string]any{
		"url":  url,
		"size": len(data),
	})

	return data, nil
}

// saveMediaToCache 保存媒体文件到缓存
func (c *WeComWSChannel) saveMediaToCache(data []byte, filename string) (string, error) {
	// 获取缓存目录
	cacheDir := c.getMediaCacheDir()

	// 创建目录
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// 生成文件名
	if filename == "" {
		filename = generateRandomString(16)
	}

	filepath := filepath.Join(cacheDir, filename)

	// 写入文件
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	logger.DebugCF("wecom_ws", "Media saved to cache", map[string]any{
		"path": filepath,
		"size": len(data),
	})

	return filepath, nil
}

// getMediaCacheDir 获取媒体缓存目录
func (c *WeComWSChannel) getMediaCacheDir() string {
	// 优先使用配置的缓存路径
	if c.config.MediaCachePath != "" {
		return c.config.MediaCachePath
	}

	// 使用系统临时目录
	return filepath.Join(os.TempDir(), "picoclaw", "wecom_ws", "media")
}

// downloadAndSaveMediaList 下载并保存媒体列表（并发下载）
func (c *WeComWSChannel) downloadAndSaveMediaList(mediaList []MediaInfo) []MediaInfo {
	if !c.config.EnableMediaDownload || len(mediaList) == 0 {
		return mediaList
	}

	maxSize := c.config.MediaMaxSize
	if maxSize == 0 {
		maxSize = defaultMediaMaxMB
	}

	// 使用 WaitGroup 并发下载
	var wg sync.WaitGroup
	results := make([]MediaInfo, len(mediaList))
	var mu sync.Mutex

	for i, media := range mediaList {
		wg.Add(1)
		go func(index int, m MediaInfo) {
			defer wg.Done()

			var timeout time.Duration
			if m.Type == "image" {
				timeout = imageDownloadTimeoutMs * time.Millisecond
			} else {
				timeout = fileDownloadTimeoutMs * time.Millisecond
			}

			// 下载媒体
			data, err := c.downloadMediaWithTimeout(m.URL, timeout, maxSize)
			if err != nil {
				logger.WarnCF("wecom_ws", "Failed to download media", map[string]any{
					"url":   m.URL,
					"error": err.Error(),
				})
				mu.Lock()
				results[index] = m
				mu.Unlock()
				return
			}

			// 解密（如果有 AES key）
			if m.AESKey != "" {
				decrypted, err := c.decryptMediaData(data, m.AESKey)
				if err != nil {
					logger.WarnCF("wecom_ws", "Failed to decrypt media", map[string]any{
						"url":   m.URL,
						"error": err.Error(),
					})
					// 解密失败，使用原始数据
				} else {
					data = decrypted
				}
			}

			// 检测文件类型
			fileType := DetectFileType(data)
			m.ContentType = string(fileType.Type)

			// 生成文件名
			filename := m.Filename
			if filename == "" {
				filename = generateRandomString(16) + fileType.Ext
			}

			// 保存到缓存
			cachePath, err := c.saveMediaToCache(data, filename)
			if err != nil {
				logger.WarnCF("wecom_ws", "Failed to save media to cache", map[string]any{
					"url":   m.URL,
					"error": err.Error(),
				})
				mu.Lock()
				results[index] = m
				mu.Unlock()
				return
			}

			m.Path = cachePath
			mu.Lock()
			results[index] = m
			mu.Unlock()

			logger.DebugCF("wecom_ws", "Media downloaded and saved", map[string]any{
				"url":  m.URL,
				"path": cachePath,
				"type": m.Type,
			})
		}(i, media)
	}

	wg.Wait()
	return results
}

// isMarkdownContent 检查内容是否为 Markdown 格式
func (c *WeComWSChannel) isMarkdownContent(content string) bool {
	// 简单的 Markdown 检测规则
	markdownPatterns := []string{
		"# ",   // 标题
		"## ",  // 二级标题
		"### ", // 三级标题
		"**",   // 粗体
		"*",    // 斜体
		"`",    // 代码
		"```",  // 代码块
		"[",    // 链接
		"!",    // 图片
		"- ",   // 列表
		"1. ",  // 有序列表
		"> ",   // 引用
		"|",    // 表格
		"---",  // 分隔线
	}

	for _, pattern := range markdownPatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

// sendMediaResponse 发送媒体响应（图片或文件）
func (c *WeComWSChannel) sendMediaResponse(streamID, mediaPath, mediaType string) error {
	if mediaPath == "" {
		return nil
	}

	// 读取媒体文件
	data, err := os.ReadFile(mediaPath)
	if err != nil {
		return fmt.Errorf("failed to read media file: %w", err)
	}

	// 根据文件类型发送
	fileTypeInfo := DetectFileType(data)
	if IsImage(fileTypeInfo.Type) {
		return c.sendImageResponse(streamID, data, true)
	}

	// 文件响应
	filename := filepath.Base(mediaPath)
	return c.sendFileResponse(streamID, data, filename, true)
}

// cleanupMediaCache 清理媒体缓存
func (c *WeComWSChannel) cleanupMediaCache(maxAge time.Duration) {
	cacheDir := c.getMediaCacheDir()

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		logger.DebugCF("wecom_ws", "Failed to read cache directory", map[string]any{
			"error": err.Error(),
		})
		return
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			path := filepath.Join(cacheDir, entry.Name())
			if err := os.Remove(path); err != nil {
				logger.DebugCF("wecom_ws", "Failed to remove old cache file", map[string]any{
					"path":  path,
					"error": err.Error(),
				})
			}
		}
	}
}

// decryptMediaData 解密媒体文件数据
// 企业微信媒体文件使用 AES-256-CBC 加密
func (c *WeComWSChannel) decryptMediaData(encryptedData []byte, aesKey string) ([]byte, error) {
	if aesKey == "" {
		// 没有加密密钥，直接返回
		return encryptedData, nil
	}

	logger.DebugC("wecom_ws", "Decrypting media data")

	// 解码 AES key
	key, err := decodeWeComAESKey(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode AES key: %w", err)
	}

	// 解密数据
	decryptedData, err := decryptAESCBC(key, encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt media: %w", err)
	}

	logger.DebugCF("wecom_ws", "Media decrypted", map[string]any{
		"original_size":  len(encryptedData),
		"decrypted_size": len(decryptedData),
	})

	return decryptedData, nil
}

// downloadAndProcessMedia 下载并处理媒体文件（包括解密）
func (c *WeComWSChannel) downloadAndProcessMedia(url, aesKey, filename string, timeout time.Duration, maxSizeMB int) (string, error) {
	// 下载媒体文件
	data, err := c.downloadMedia(url, timeout, maxSizeMB)
	if err != nil {
		return "", fmt.Errorf("failed to download media: %w", err)
	}

	// 如果有 AES key，解密数据
	if aesKey != "" && c.config.EnableMediaDownload {
		data, err = c.decryptMediaData(data, aesKey)
		if err != nil {
			logger.WarnCF("wecom_ws", "Failed to decrypt media, using encrypted data", map[string]any{
				"error": err.Error(),
			})
			// 解密失败，继续使用加密数据
		}
	}

	// 保存到缓存
	cachePath, err := c.saveMediaToCache(data, filename)
	if err != nil {
		return "", fmt.Errorf("failed to save media to cache: %w", err)
	}

	return cachePath, nil
}

// sendImageResponse 发送图片响应
func (c *WeComWSChannel) sendImageResponse(streamID string, imageData []byte, finish bool) error {
	// 计算 MD5
	md5Hash := fmt.Sprintf("%x", md5.Sum(imageData))

	// 转换为 base64
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	response := &ResponseMessage{
		MsgType: "stream",
		Stream: &StreamContent{
			ID:     streamID,
			Finish: finish,
			MsgItem: []StreamMsgItem{
				{
					MsgType: "image",
					Image: &ResponseImage{
						Base64: base64Data,
						MD5:    md5Hash,
					},
				},
			},
		},
	}

	body, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal image response: %w", err)
	}

	wsMsg := &WeComWSMessage{
		Cmd: string(CmdAIBotResponse),
		Headers: MessageHeaders{
			ReqID: generateReqID(),
		},
		Body: body,
	}

	return c.sendMessage(wsMsg)
}

// sendFileResponse 发送文件响应
func (c *WeComWSChannel) sendFileResponse(streamID string, fileData []byte, filename string, finish bool) error {
	// 转换为 base64
	base64Data := base64.StdEncoding.EncodeToString(fileData)

	response := &ResponseMessage{
		MsgType: "stream",
		Stream: &StreamContent{
			ID:     streamID,
			Finish: finish,
			MsgItem: []StreamMsgItem{
				{
					MsgType: "file",
					File: &ResponseFile{
						Base64:   base64Data,
						Filename: filename,
					},
				},
			},
		},
	}

	body, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal file response: %w", err)
	}

	wsMsg := &WeComWSMessage{
		Cmd: string(CmdAIBotResponse),
		Headers: MessageHeaders{
			ReqID: generateReqID(),
		},
		Body: body,
	}

	return c.sendMessage(wsMsg)
}
