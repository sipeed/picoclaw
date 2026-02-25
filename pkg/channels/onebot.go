package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
	"github.com/sipeed/picoclaw/pkg/voice"
)

type OneBotChannel struct {
	*BaseChannel
	config          config.OneBotConfig
	conn            *websocket.Conn
	ctx             context.Context
	cancel          context.CancelFunc
	dedup           map[string]struct{}
	dedupRing       []string
	dedupIdx        int
	mu              sync.Mutex
	writeMu         sync.Mutex
	echoCounter     int64
	selfID          int64
	pending         map[string]chan json.RawMessage
	pendingMu       sync.Mutex
	transcriber     *voice.GroqTranscriber
	lastMessageID   sync.Map
	pendingEmojiMsg sync.Map
}

type oneBotRawEvent struct {
	PostType      string          `json:"post_type"`
	MessageType   string          `json:"message_type"`
	SubType       string          `json:"sub_type"`
	MessageID     json.RawMessage `json:"message_id"`
	UserID        json.RawMessage `json:"user_id"`
	GroupID       json.RawMessage `json:"group_id"`
	RawMessage    string          `json:"raw_message"`
	Message       json.RawMessage `json:"message"`
	Sender        json.RawMessage `json:"sender"`
	SelfID        json.RawMessage `json:"self_id"`
	Time          json.RawMessage `json:"time"`
	MetaEventType string          `json:"meta_event_type"`
	NoticeType    string          `json:"notice_type"`
	Echo          string          `json:"echo"`
	RetCode       json.RawMessage `json:"retcode"`
	Status        json.RawMessage `json:"status"`
	Data          json.RawMessage `json:"data"`
}

type BotStatus struct {
	Online bool `json:"online"`
	Good   bool `json:"good"`
}

func isAPIResponse(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s == "ok" || s == "failed"
	}
	var bs BotStatus
	if json.Unmarshal(raw, &bs) == nil {
		return bs.Online || bs.Good
	}
	return false
}

type oneBotSender struct {
	UserID   json.RawMessage `json:"user_id"`
	Nickname string          `json:"nickname"`
	Card     string          `json:"card"`
}

type oneBotAPIRequest struct {
	Action string `json:"action"`
	Params any    `json:"params"`
	Echo   string `json:"echo,omitempty"`
}

type oneBotMessageSegment struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

func NewOneBotChannel(cfg config.OneBotConfig, messageBus *bus.MessageBus) (*OneBotChannel, error) {
	base := NewBaseChannel("onebot", cfg, messageBus, cfg.AllowFrom)

	const dedupSize = 1024
	return &OneBotChannel{
		BaseChannel: base,
		config:      cfg,
		dedup:       make(map[string]struct{}, dedupSize),
		dedupRing:   make([]string, dedupSize),
		dedupIdx:    0,
		pending:     make(map[string]chan json.RawMessage),
	}, nil
}

func (c *OneBotChannel) SetTranscriber(transcriber *voice.GroqTranscriber) {
	c.transcriber = transcriber
}

func (c *OneBotChannel) setMsgEmojiLike(messageID string, emojiID int, set bool) {
	go func() {
		_, err := c.sendAPIRequest("set_msg_emoji_like", map[string]any{
			"message_id": messageID,
			"emoji_id":   emojiID,
			"set":        set,
		}, 5*time.Second)
		if err != nil {
			logger.DebugCF("onebot", "Failed to set emoji like", map[string]any{
				"message_id": messageID,
				"error":      err.Error(),
			})
		}
	}()
}

func (c *OneBotChannel) Start(ctx context.Context) error {
	if c.config.WSUrl == "" {
		return fmt.Errorf("OneBot ws_url not configured")
	}

	logger.InfoCF("onebot", "Starting OneBot channel", map[string]any{
		"ws_url": c.config.WSUrl,
	})

	c.ctx, c.cancel = context.WithCancel(ctx)

	if err := c.connect(); err != nil {
		logger.WarnCF("onebot", "Initial connection failed, will retry in background", map[string]any{
			"error": err.Error(),
		})
	} else {
		go c.listen()
		c.fetchSelfID()
	}

	if c.config.ReconnectInterval > 0 {
		go c.reconnectLoop()
	} else {
		if c.conn == nil {
			return fmt.Errorf("failed to connect to OneBot and reconnect is disabled")
		}
	}

	c.setRunning(true)
	logger.InfoC("onebot", "OneBot channel started successfully")

	return nil
}

func (c *OneBotChannel) connect() error {
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	header := make(map[string][]string)
	if c.config.AccessToken != "" {
		header["Authorization"] = []string{"Bearer " + c.config.AccessToken}
	}

	conn, resp, err := dialer.Dial(c.config.WSUrl, header)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return err
	}

	conn.SetPongHandler(func(appData string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	go c.pinger(conn)

	logger.InfoC("onebot", "WebSocket connected")
	return nil
}

func (c *OneBotChannel) pinger(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.writeMu.Lock()
			err := conn.WriteMessage(websocket.PingMessage, nil)
			c.writeMu.Unlock()
			if err != nil {
				logger.DebugCF("onebot", "Ping write failed, stopping pinger", map[string]any{
					"error": err.Error(),
				})
				return
			}
		}
	}
}

func (c *OneBotChannel) fetchSelfID() {
	resp, err := c.sendAPIRequest("get_login_info", nil, 5*time.Second)
	if err != nil {
		logger.WarnCF("onebot", "Failed to get_login_info", map[string]any{
			"error": err.Error(),
		})
		return
	}

	type loginInfo struct {
		UserID   json.RawMessage `json:"user_id"`
		Nickname string          `json:"nickname"`
	}
	for _, extract := range []func() (*loginInfo, error){
		func() (*loginInfo, error) {
			var w struct {
				Data loginInfo `json:"data"`
			}
			err := json.Unmarshal(resp, &w)
			return &w.Data, err
		},
		func() (*loginInfo, error) {
			var f loginInfo
			err := json.Unmarshal(resp, &f)
			return &f, err
		},
	} {
		info, err := extract()
		if err != nil || len(info.UserID) == 0 {
			continue
		}
		if uid, err := parseJSONInt64(info.UserID); err == nil && uid > 0 {
			atomic.StoreInt64(&c.selfID, uid)
			logger.InfoCF("onebot", "Bot self ID retrieved", map[string]any{
				"self_id":  uid,
				"nickname": info.Nickname,
			})
			return
		}
	}

	logger.WarnCF("onebot", "Could not parse self ID from get_login_info response", map[string]any{
		"response": string(resp),
	})
}

func (c *OneBotChannel) sendAPIRequest(action string, params any, timeout time.Duration) (json.RawMessage, error) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return nil, fmt.Errorf("WebSocket not connected")
	}

	echo := fmt.Sprintf("api_%d_%d", time.Now().UnixNano(), atomic.AddInt64(&c.echoCounter, 1))

	ch := make(chan json.RawMessage, 1)
	c.pendingMu.Lock()
	c.pending[echo] = ch
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, echo)
		c.pendingMu.Unlock()
	}()

	req := oneBotAPIRequest{
		Action: action,
		Params: params,
		Echo:   echo,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal API request: %w", err)
	}

	c.writeMu.Lock()
	err = conn.WriteMessage(websocket.TextMessage, data)
	c.writeMu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to write API request: %w", err)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("API request %s timed out after %v", action, timeout)
	case <-c.ctx.Done():
		return nil, fmt.Errorf("context cancelled")
	}
}

func (c *OneBotChannel) reconnectLoop() {
	interval := time.Duration(c.config.ReconnectInterval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(interval):
			c.mu.Lock()
			conn := c.conn
			c.mu.Unlock()

			if conn == nil {
				logger.InfoC("onebot", "Attempting to reconnect...")
				if err := c.connect(); err != nil {
					logger.ErrorCF("onebot", "Reconnect failed", map[string]any{
						"error": err.Error(),
					})
				} else {
					go c.listen()
					c.fetchSelfID()
				}
			}
		}
	}
}

func (c *OneBotChannel) Stop(ctx context.Context) error {
	logger.InfoC("onebot", "Stopping OneBot channel")
	c.setRunning(false)

	if c.cancel != nil {
		c.cancel()
	}

	c.pendingMu.Lock()
	for echo, ch := range c.pending {
		close(ch)
		delete(c.pending, echo)
	}
	c.pendingMu.Unlock()

	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()

	return nil
}

func (c *OneBotChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("OneBot channel not running")
	}

	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("OneBot WebSocket not connected")
	}

	action, params, err := c.buildSendRequest(msg)
	if err != nil {
		return err
	}

	echo := fmt.Sprintf("send_%d", atomic.AddInt64(&c.echoCounter, 1))

	req := oneBotAPIRequest{
		Action: action,
		Params: params,
		Echo:   echo,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal OneBot request: %w", err)
	}

	c.writeMu.Lock()
	err = conn.WriteMessage(websocket.TextMessage, data)
	c.writeMu.Unlock()

	if err != nil {
		logger.ErrorCF("onebot", "Failed to send message", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	if msgID, ok := c.pendingEmojiMsg.LoadAndDelete(msg.ChatID); ok {
		if mid, ok := msgID.(string); ok && mid != "" {
			c.setMsgEmojiLike(mid, 289, false)
		}
	}

	return nil
}

func (c *OneBotChannel) buildMessageSegments(chatID, content string) []oneBotMessageSegment {
	var segments []oneBotMessageSegment

	if lastMsgID, ok := c.lastMessageID.Load(chatID); ok {
		if msgID, ok := lastMsgID.(string); ok && msgID != "" {
			segments = append(segments, oneBotMessageSegment{
				Type: "reply",
				Data: map[string]any{"id": msgID},
			})
		}
	}

	segments = append(segments, oneBotMessageSegment{
		Type: "text",
		Data: map[string]any{"text": content},
	})

	return segments
}

func (c *OneBotChannel) buildSendRequest(msg bus.OutboundMessage) (string, any, error) {
	chatID := msg.ChatID
	segments := c.buildMessageSegments(chatID, msg.Content)

	var action, idKey string
	var rawID string
	if rest, ok := strings.CutPrefix(chatID, "group:"); ok {
		action, idKey, rawID = "send_group_msg", "group_id", rest
	} else if rest, ok := strings.CutPrefix(chatID, "private:"); ok {
		action, idKey, rawID = "send_private_msg", "user_id", rest
	} else {
		action, idKey, rawID = "send_private_msg", "user_id", chatID
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return "", nil, fmt.Errorf("invalid %s in chatID: %s", idKey, chatID)
	}
	return action, map[string]any{idKey: id, "message": segments}, nil
}

func (c *OneBotChannel) listen() {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		logger.WarnC("onebot", "WebSocket connection is nil, listener exiting")
		return
	}

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				logger.ErrorCF("onebot", "WebSocket read error", map[string]any{
					"error": err.Error(),
				})
				c.mu.Lock()
				if c.conn == conn {
					c.conn.Close()
					c.conn = nil
				}
				c.mu.Unlock()
				return
			}

			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			var raw oneBotRawEvent
			if err := json.Unmarshal(message, &raw); err != nil {
				logger.WarnCF("onebot", "Failed to unmarshal raw event", map[string]any{
					"error":   err.Error(),
					"payload": string(message),
				})
				continue
			}

			logger.DebugCF("onebot", "WebSocket event", map[string]any{
				"length":    len(message),
				"post_type": raw.PostType,
				"sub_type":  raw.SubType,
			})

			if raw.Echo != "" {
				c.pendingMu.Lock()
				ch, ok := c.pending[raw.Echo]
				c.pendingMu.Unlock()

				if ok {
					select {
					case ch <- message:
					default:
					}
				} else {
					logger.DebugCF("onebot", "Received API response (no waiter)", map[string]any{
						"echo":   raw.Echo,
						"status": string(raw.Status),
					})
				}
				continue
			}

			if isAPIResponse(raw.Status) {
				logger.DebugCF("onebot", "Received API response without echo, skipping", map[string]any{
					"status": string(raw.Status),
				})
				continue
			}

			c.handleRawEvent(&raw)
		}
	}
}

func parseJSONInt64(raw json.RawMessage) (int64, error) {
	if len(raw) == 0 {
		return 0, nil
	}

	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strconv.ParseInt(s, 10, 64)
	}
	return 0, fmt.Errorf("cannot parse as int64: %s", string(raw))
}

func parseJSONString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	return string(raw)
}

type parseMessageResult struct {
	Text           string
	IsBotMentioned bool
	Media          []string
	LocalFiles     []string
	ReplyTo        string
}

func (c *OneBotChannel) parseMessageSegments(raw json.RawMessage, selfID int64) parseMessageResult {
	if len(raw) == 0 {
		return parseMessageResult{}
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		mentioned := false
		if selfID > 0 {
			cqAt := fmt.Sprintf("[CQ:at,qq=%d]", selfID)
			if strings.Contains(s, cqAt) {
				mentioned = true
				s = strings.ReplaceAll(s, cqAt, "")
				s = strings.TrimSpace(s)
			}
		}
		return parseMessageResult{Text: s, IsBotMentioned: mentioned}
	}

	var segments []map[string]any
	if err := json.Unmarshal(raw, &segments); err != nil {
		return parseMessageResult{}
	}

	var textParts []string
	mentioned := false
	selfIDStr := strconv.FormatInt(selfID, 10)
	var media []string
	var localFiles []string
	var replyTo string

	for _, seg := range segments {
		segType, _ := seg["type"].(string)
		data, _ := seg["data"].(map[string]any)

		switch segType {
		case "text":
			if data != nil {
				if t, ok := data["text"].(string); ok {
					textParts = append(textParts, t)
				}
			}

		case "at":
			if data != nil && selfID > 0 {
				qqVal := fmt.Sprintf("%v", data["qq"])
				if qqVal == selfIDStr || qqVal == "all" {
					mentioned = true
				}
			}

		case "image", "video", "file":
			if data != nil {
				url, _ := data["url"].(string)
				if url != "" {
					defaults := map[string]string{"image": "image.jpg", "video": "video.mp4", "file": "file"}
					filename := defaults[segType]
					if f, ok := data["file"].(string); ok && f != "" {
						filename = f
					} else if n, ok := data["name"].(string); ok && n != "" {
						filename = n
					}
					localPath := utils.DownloadFile(url, filename, utils.DownloadOptions{
						LoggerPrefix: "onebot",
					})
					if localPath != "" {
						media = append(media, localPath)
						localFiles = append(localFiles, localPath)
						textParts = append(textParts, fmt.Sprintf("[%s]", segType))
					}
				}
			}

		case "record":
			if data != nil {
				url, _ := data["url"].(string)
				if url != "" {
					localPath := utils.DownloadFile(url, "voice.amr", utils.DownloadOptions{
						LoggerPrefix: "onebot",
					})
					if localPath != "" {
						localFiles = append(localFiles, localPath)
						if c.transcriber != nil && c.transcriber.IsAvailable() {
							tctx, tcancel := context.WithTimeout(c.ctx, 30*time.Second)
							result, err := c.transcriber.Transcribe(tctx, localPath)
							tcancel()
							if err != nil {
								logger.WarnCF("onebot", "Voice transcription failed", map[string]any{
									"error": err.Error(),
								})
								textParts = append(textParts, "[voice (transcription failed)]")
								media = append(media, localPath)
							} else {
								textParts = append(textParts, fmt.Sprintf("[voice transcription: %s]", result.Text))
							}
						} else {
							textParts = append(textParts, "[voice]")
							media = append(media, localPath)
						}
					}
				}
			}

		case "reply":
			if data != nil {
				if id, ok := data["id"]; ok {
					replyTo = fmt.Sprintf("%v", id)
				}
			}

		case "face":
			if data != nil {
				faceID, _ := data["id"]
				textParts = append(textParts, fmt.Sprintf("[face:%v]", faceID))
			}

		case "forward":
			textParts = append(textParts, "[forward message]")

		default:

		}
	}

	return parseMessageResult{
		Text:           strings.TrimSpace(strings.Join(textParts, "")),
		IsBotMentioned: mentioned,
		Media:          media,
		LocalFiles:     localFiles,
		ReplyTo:        replyTo,
	}
}

func (c *OneBotChannel) handleRawEvent(raw *oneBotRawEvent) {
	switch raw.PostType {
	case "message":
		if userID, err := parseJSONInt64(raw.UserID); err == nil && userID > 0 {
			if !c.IsAllowed(strconv.FormatInt(userID, 10)) {
				logger.DebugCF("onebot", "Message rejected by allowlist", map[string]any{
					"user_id": userID,
				})
				return
			}
		}
		c.handleMessage(raw)

	case "message_sent":
		logger.DebugCF("onebot", "Bot sent message event", map[string]any{
			"message_type": raw.MessageType,
			"message_id":   parseJSONString(raw.MessageID),
		})

	case "meta_event":
		c.handleMetaEvent(raw)

	case "notice":
		c.handleNoticeEvent(raw)

	case "request":
		logger.DebugCF("onebot", "Request event received", map[string]any{
			"sub_type": raw.SubType,
		})

	case "":
		logger.DebugCF("onebot", "Event with empty post_type (possibly API response)", map[string]any{
			"echo":   raw.Echo,
			"status": raw.Status,
		})

	default:
		logger.DebugCF("onebot", "Unknown post_type", map[string]any{
			"post_type": raw.PostType,
		})
	}
}

func (c *OneBotChannel) handleMetaEvent(raw *oneBotRawEvent) {
	if raw.MetaEventType == "lifecycle" {
		logger.InfoCF("onebot", "Lifecycle event", map[string]any{"sub_type": raw.SubType})
	} else if raw.MetaEventType != "heartbeat" {
		logger.DebugCF("onebot", "Meta event: "+raw.MetaEventType, nil)
	}
}

func (c *OneBotChannel) handleNoticeEvent(raw *oneBotRawEvent) {
	fields := map[string]any{
		"notice_type": raw.NoticeType,
		"sub_type":    raw.SubType,
		"group_id":    parseJSONString(raw.GroupID),
		"user_id":     parseJSONString(raw.UserID),
		"message_id":  parseJSONString(raw.MessageID),
	}
	switch raw.NoticeType {
	case "group_recall", "group_increase", "group_decrease",
		"friend_add", "group_admin", "group_ban":
		logger.InfoCF("onebot", "Notice: "+raw.NoticeType, fields)
	default:
		logger.DebugCF("onebot", "Notice: "+raw.NoticeType, fields)
	}
}

func (c *OneBotChannel) handleMessage(raw *oneBotRawEvent) {
	// Parse fields from raw event
	userID, err := parseJSONInt64(raw.UserID)
	if err != nil {
		logger.WarnCF("onebot", "Failed to parse user_id", map[string]any{
			"error": err.Error(),
			"raw":   string(raw.UserID),
		})
		return
	}

	groupID, _ := parseJSONInt64(raw.GroupID)
	selfID, _ := parseJSONInt64(raw.SelfID)
	messageID := parseJSONString(raw.MessageID)

	if selfID == 0 {
		selfID = atomic.LoadInt64(&c.selfID)
	}

	parsed := c.parseMessageSegments(raw.Message, selfID)
	isBotMentioned := parsed.IsBotMentioned

	content := raw.RawMessage
	if content == "" {
		content = parsed.Text
	} else if selfID > 0 {
		cqAt := fmt.Sprintf("[CQ:at,qq=%d]", selfID)
		if strings.Contains(content, cqAt) {
			isBotMentioned = true
			content = strings.ReplaceAll(content, cqAt, "")
			content = strings.TrimSpace(content)
		}
	}

	if parsed.Text != "" && content != parsed.Text && (len(parsed.Media) > 0 || parsed.ReplyTo != "") {
		content = parsed.Text
	}

	var sender oneBotSender
	if len(raw.Sender) > 0 {
		if err := json.Unmarshal(raw.Sender, &sender); err != nil {
			logger.WarnCF("onebot", "Failed to parse sender", map[string]any{
				"error":  err.Error(),
				"sender": string(raw.Sender),
			})
		}
	}

	// Clean up temp files when done
	if len(parsed.LocalFiles) > 0 {
		defer func() {
			for _, f := range parsed.LocalFiles {
				if err := os.Remove(f); err != nil {
					logger.DebugCF("onebot", "Failed to remove temp file", map[string]any{
						"path":  f,
						"error": err.Error(),
					})
				}
			}
		}()
	}

	if c.isDuplicate(messageID) {
		logger.DebugCF("onebot", "Duplicate message, skipping", map[string]any{
			"message_id": messageID,
		})
		return
	}

	if content == "" {
		logger.DebugCF("onebot", "Received empty message, ignoring", map[string]any{
			"message_id": messageID,
		})
		return
	}

	senderID := strconv.FormatInt(userID, 10)
	var chatID string

	metadata := map[string]string{
		"message_id": messageID,
	}

	if parsed.ReplyTo != "" {
		metadata["reply_to_message_id"] = parsed.ReplyTo
	}

	switch raw.MessageType {
	case "private":
		chatID = "private:" + senderID
		metadata["peer_kind"] = "direct"
		metadata["peer_id"] = senderID

	case "group":
		groupIDStr := strconv.FormatInt(groupID, 10)
		chatID = "group:" + groupIDStr
		metadata["peer_kind"] = "group"
		metadata["peer_id"] = groupIDStr
		metadata["group_id"] = groupIDStr

		senderUserID, _ := parseJSONInt64(sender.UserID)
		if senderUserID > 0 {
			metadata["sender_user_id"] = strconv.FormatInt(senderUserID, 10)
		}

		if sender.Card != "" {
			metadata["sender_name"] = sender.Card
		} else if sender.Nickname != "" {
			metadata["sender_name"] = sender.Nickname
		}

		triggered, strippedContent := c.checkGroupTrigger(content, isBotMentioned)
		if !triggered {
			logger.DebugCF("onebot", "Group message ignored (no trigger)", map[string]any{
				"sender":       senderID,
				"group":        groupIDStr,
				"is_mentioned": isBotMentioned,
				"content":      truncate(content, 100),
			})
			return
		}
		content = strippedContent

	default:
		logger.WarnCF("onebot", "Unknown message type, cannot route", map[string]any{
			"type":       raw.MessageType,
			"message_id": messageID,
			"user_id":    userID,
		})
		return
	}

	logger.InfoCF("onebot", "Received "+raw.MessageType+" message", map[string]any{
		"sender":      senderID,
		"chat_id":     chatID,
		"message_id":  messageID,
		"length":      len(content),
		"content":     truncate(content, 100),
		"media_count": len(parsed.Media),
	})

	if sender.Nickname != "" {
		metadata["nickname"] = sender.Nickname
	}

	c.lastMessageID.Store(chatID, messageID)

	if raw.MessageType == "group" && messageID != "" && messageID != "0" {
		c.setMsgEmojiLike(messageID, 289, true)
		c.pendingEmojiMsg.Store(chatID, messageID)
	}

	c.HandleMessage(senderID, chatID, content, parsed.Media, metadata)
}

func (c *OneBotChannel) isDuplicate(messageID string) bool {
	if messageID == "" || messageID == "0" {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.dedup[messageID]; exists {
		return true
	}

	if old := c.dedupRing[c.dedupIdx]; old != "" {
		delete(c.dedup, old)
	}
	c.dedupRing[c.dedupIdx] = messageID
	c.dedup[messageID] = struct{}{}
	c.dedupIdx = (c.dedupIdx + 1) % len(c.dedupRing)

	return false
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

func (c *OneBotChannel) checkGroupTrigger(
	content string,
	isBotMentioned bool,
) (triggered bool, strippedContent string) {
	if isBotMentioned {
		return true, strings.TrimSpace(content)
	}

	for _, prefix := range c.config.GroupTriggerPrefix {
		if prefix == "" {
			continue
		}
		if strings.HasPrefix(content, prefix) {
			return true, strings.TrimSpace(strings.TrimPrefix(content, prefix))
		}
	}

	return false, content
}
