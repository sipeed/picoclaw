package wecom

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	wecomConnectTimeout    = 15 * time.Second
	wecomCommandTimeout    = 10 * time.Second
	wecomUploadTimeout     = 30 * time.Second
	wecomHeartbeatInterval = 30 * time.Second
	wecomStreamMaxDuration = 5*time.Minute + 30*time.Second
	wecomStreamMinInterval = 500 * time.Millisecond
	wecomRouteTTL          = 30 * time.Minute
	wecomMediaTimeout      = 30 * time.Second
	wecomRecentMessageMax  = 1000
)

type WeComChannel struct {
	*channels.BaseChannel
	config *config.WeComSettings

	ctx    context.Context
	cancel context.CancelFunc

	conn   *websocket.Conn
	connMu sync.Mutex

	pendingMu sync.Mutex
	pending   map[string]chan wecomEnvelope

	turnsMu sync.Mutex
	turns   map[string][]wecomTurn

	recent      *recentMessageSet
	routes      *reqIDStore
	mediaClient *http.Client
	commandSend func(wecomCommand, time.Duration) (wecomEnvelope, error)
}

type wecomTurn struct {
	ReqID     string
	ChatID    string
	ChatType  uint32
	StreamID  string
	CreatedAt time.Time
}

type wecomStreamer struct {
	channel *WeComChannel
	chatID  string
	turn    wecomTurn

	mu         sync.Mutex
	closed     bool
	lastSentAt time.Time
	content    string
}

type recentMessageSet struct {
	mu   sync.Mutex
	seen map[string]struct{}
	ring []string
	idx  int
}

func newRecentMessageSet(capacity int) *recentMessageSet {
	if capacity <= 0 {
		capacity = wecomRecentMessageMax
	}
	return &recentMessageSet{
		seen: make(map[string]struct{}, capacity),
		ring: make([]string, capacity),
	}
}

func (s *recentMessageSet) Mark(id string) bool {
	if id == "" {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.seen[id]; ok {
		return false
	}
	if old := s.ring[s.idx]; old != "" {
		delete(s.seen, old)
	}
	s.ring[s.idx] = id
	s.idx = (s.idx + 1) % len(s.ring)
	s.seen[id] = struct{}{}
	return true
}

func NewChannel(bc *config.Channel, cfg *config.WeComSettings, messageBus *bus.MessageBus) (*WeComChannel, error) {
	if cfg.BotID == "" || cfg.Secret.String() == "" {
		return nil, fmt.Errorf("wecom bot_id and secret are required")
	}
	if cfg.WebSocketURL == "" {
		cfg.WebSocketURL = wecomDefaultWebSocketURL
	}

	base := channels.NewBaseChannel(
		"wecom",
		cfg,
		messageBus,
		bc.AllowFrom,
		channels.WithReasoningChannelID(bc.ReasoningChannelID),
	)

	ch := &WeComChannel{
		BaseChannel: base,
		config:      cfg,
		pending:     make(map[string]chan wecomEnvelope),
		turns:       make(map[string][]wecomTurn),
		recent:      newRecentMessageSet(wecomRecentMessageMax),
		routes:      newReqIDStore(""),
		mediaClient: &http.Client{Timeout: wecomMediaTimeout},
	}
	ch.SetOwner(ch)
	return ch, nil
}

func (c *WeComChannel) Name() string { return "wecom" }

func (c *WeComChannel) Start(ctx context.Context) error {
	logger.InfoC("wecom", "Starting WeCom channel...")
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.SetRunning(true)
	go c.connectLoop()
	return nil
}

func (c *WeComChannel) Stop(_ context.Context) error {
	logger.InfoC("wecom", "Stopping WeCom channel...")
	if c.cancel != nil {
		c.cancel()
	}
	c.connMu.Lock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()
	c.clearTurns()
	c.SetRunning(false)
	return nil
}

func (c *WeComChannel) BeginStream(_ context.Context, chatID string) (channels.Streamer, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	turn, ok := c.getTurn(chatID)
	if !ok {
		return nil, fmt.Errorf("wecom streaming unavailable: no active turn")
	}
	if time.Since(turn.CreatedAt) > wecomStreamMaxDuration {
		c.consumeTurn(chatID, turn)
		return nil, fmt.Errorf("wecom streaming unavailable: turn expired")
	}

	return &wecomStreamer{
		channel: c,
		chatID:  chatID,
		turn:    turn,
	}, nil
}

func (c *WeComChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return nil, nil
	}

	if turn, ok := c.getTurn(msg.ChatID); ok {
		if time.Since(turn.CreatedAt) <= wecomStreamMaxDuration {
			if err := c.sendStreamReply(turn, content); err == nil {
				c.consumeTurn(msg.ChatID, turn)
				return nil, nil
			}
		}
		c.consumeTurn(msg.ChatID, turn)
	}

	if route, ok := c.routes.Get(msg.ChatID); ok {
		if err := c.sendActivePush(route.ChatID, route.ChatType, content); err != nil {
			return nil, err
		}
		return nil, nil
	}

	if err := c.sendActivePush(msg.ChatID, 0, content); err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *WeComChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	route, chatType, hasTurn := c.resolveMediaRoute(msg.ChatID)
	chatID := route.ChatID
	if chatID == "" {
		chatID = msg.ChatID
	}

	for _, part := range msg.Parts {
		if strings.TrimSpace(part.Ref) == "" {
			if caption := strings.TrimSpace(part.Caption); caption != "" {
				if err := c.sendActivePush(chatID, chatType, caption); err != nil {
					return nil, err
				}
			}
			continue
		}

		localPath, filename, contentType, cleanup, err := c.resolveOutboundPart(ctx, part)
		if err != nil {
			return nil, fmt.Errorf("wecom resolve media %q: %v: %w", part.Ref, err, channels.ErrSendFailed)
		}

		func() {
			if cleanup != nil {
				defer cleanup()
			}

			uploaded, uploadErr := c.uploadOutboundMedia(ctx, localPath, filename, contentType, part)
			if uploadErr != nil {
				logger.WarnCF("wecom", "Falling back to placeholder after media upload failure", map[string]any{
					"chat_id":      chatID,
					"ref":          part.Ref,
					"filename":     filename,
					"content_type": contentType,
					"error":        uploadErr.Error(),
				})
				if hasTurn {
					if finishErr := c.sendStreamChunk(route, true, ""); finishErr != nil {
						err = finishErr
						return
					}
					c.deleteTurn(msg.ChatID)
					hasTurn = false
				}
				err = c.sendActivePush(chatID, chatType, fallbackWeComMediaText(part, "", filename))
				return
			}

			if hasTurn {
				err = c.sendTurnMedia(route, uploaded)
				c.deleteTurn(msg.ChatID)
				hasTurn = false
			} else {
				err = c.sendActiveMedia(chatID, chatType, uploaded)
			}
			if err != nil {
				return
			}
			if caption := strings.TrimSpace(part.Caption); caption != "" {
				err = c.sendActivePush(chatID, chatType, caption)
			}
		}()
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (c *WeComChannel) connectLoop() {
	backoff := time.Second
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		if err := c.runConnection(); err != nil {
			logger.WarnCF("wecom", "WeCom connection lost", map[string]any{
				"error":   err.Error(),
				"backoff": backoff.String(),
			})
			select {
			case <-time.After(backoff):
			case <-c.ctx.Done():
				return
			}
			if backoff < time.Minute {
				backoff *= 2
				if backoff > time.Minute {
					backoff = time.Minute
				}
			}
			continue
		}
		return
	}
}

func (c *WeComChannel) runConnection() error {
	dialCtx, cancel := context.WithTimeout(c.ctx, wecomConnectTimeout)
	defer cancel()

	conn, resp, err := websocket.DefaultDialer.DialContext(dialCtx, c.config.WebSocketURL, nil)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("%w: %v", channels.ErrTemporary, err)
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
		_ = conn.Close()
		c.clearTurns()
	}()

	readErrCh := make(chan error, 1)
	go func() {
		readErrCh <- c.readLoop(conn)
	}()

	if writeErr := c.writeAndWait(conn, wecomCommand{
		Cmd:     wecomCmdSubscribe,
		Headers: wecomHeaders{ReqID: randomID(10)},
		Body: map[string]string{
			"bot_id": c.config.BotID,
			"secret": c.config.Secret.String(),
		},
	}, wecomCommandTimeout); writeErr != nil {
		return writeErr
	}

	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		c.heartbeatLoop(conn)
	}()

	err = <-readErrCh
	_ = conn.Close()
	<-heartbeatDone
	return err
}

func (c *WeComChannel) heartbeatLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(wecomHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := c.writeAndWait(conn, wecomCommand{
				Cmd:     wecomCmdPing,
				Headers: wecomHeaders{ReqID: randomID(10)},
			}, wecomCommandTimeout); err != nil {
				logger.WarnCF("wecom", "Heartbeat failed", map[string]any{"error": err.Error()})
				_ = conn.Close()
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *WeComChannel) readLoop(conn *websocket.Conn) error {
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-c.ctx.Done():
				return nil
			default:
				return fmt.Errorf("%w: %v", channels.ErrTemporary, err)
			}
		}

		var env wecomEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			logger.WarnCF("wecom", "Failed to parse WebSocket message", map[string]any{"error": err.Error()})
			continue
		}

		if env.Cmd == "" && env.Headers.ReqID != "" {
			c.pendingMu.Lock()
			ch, ok := c.pending[env.Headers.ReqID]
			if ok {
				delete(c.pending, env.Headers.ReqID)
			}
			c.pendingMu.Unlock()
			if ok {
				ch <- env
			}
			continue
		}

		go c.handleEnvelope(env)
	}
}

func (c *WeComChannel) handleEnvelope(env wecomEnvelope) {
	switch env.Cmd {
	case wecomCmdMsgCallback:
		c.handleMessageCallback(env)
	case wecomCmdEventCallback:
		c.handleEventCallback(env)
	default:
		logger.DebugCF("wecom", "Ignoring unsupported WeCom command", map[string]any{"cmd": env.Cmd})
	}
}

func (c *WeComChannel) handleEventCallback(env wecomEnvelope) {
	var msg wecomIncomingMessage
	if err := json.Unmarshal(env.Body, &msg); err != nil {
		logger.WarnCF("wecom", "Failed to parse WeCom event callback", map[string]any{"error": err.Error()})
	}
}

func (c *WeComChannel) handleMessageCallback(env wecomEnvelope) {
	var msg wecomIncomingMessage
	if err := json.Unmarshal(env.Body, &msg); err != nil {
		logger.WarnCF("wecom", "Failed to parse WeCom message callback", map[string]any{"error": err.Error()})
		return
	}
	if !c.recent.Mark(msg.MsgID) {
		return
	}

	reqID := env.Headers.ReqID
	if reqID == "" {
		logger.WarnC("wecom", "WeCom message callback missing req_id")
		return
	}
	if msg.Event != nil && msg.Event.EventType != "" {
		return
	}

	if err := c.dispatchIncoming(reqID, msg); err != nil {
		logger.WarnCF("wecom", "Failed to dispatch WeCom message", map[string]any{
			"req_id": reqID,
			"error":  err.Error(),
		})
		_ = c.respondImmediate(reqID, "The WeCom message could not be processed.")
	}
}

func (c *WeComChannel) dispatchIncoming(reqID string, msg wecomIncomingMessage) error {
	senderID := msg.From.UserID
	if senderID == "" {
		senderID = "unknown"
	}
	actualChatID := incomingChatID(msg)
	chatType := incomingChatTypeCode(msg.ChatType)
	peerKind := "direct"
	if msg.ChatType == "group" {
		peerKind = "group"
	}

	sender := bus.SenderInfo{
		Platform:    "wecom",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("wecom", senderID),
		DisplayName: senderID,
	}

	var (
		content   string
		quoteText string
		mediaRefs []string
		err       error
	)
	scope := channels.BuildMediaScope("wecom", actualChatID, msg.MsgID)
	switch msg.MsgType {
	case "text":
		if msg.Text != nil {
			content = strings.TrimSpace(msg.Text.Content)
		}
	case "voice":
		if msg.Voice != nil {
			content = strings.TrimSpace(msg.Voice.Content)
		}
	case "image":
		content = "[image]"
		mediaRefs, err = c.collectSingleMedia(c.ctx, scope, msg.MsgID, &mediaPayload{
			url:    msg.Image.URL,
			aesKey: msg.Image.AESKey,
		}, "image", ".jpg")
	case "file":
		content = "[file]"
		mediaRefs, err = c.collectSingleMedia(c.ctx, scope, msg.MsgID, &mediaPayload{
			url:    msg.File.URL,
			aesKey: msg.File.AESKey,
		}, "file", ".bin")
	case "video":
		content = "[video]"
		mediaRefs, err = c.collectSingleMedia(c.ctx, scope, msg.MsgID, &mediaPayload{
			url:    msg.Video.URL,
			aesKey: msg.Video.AESKey,
		}, "video", ".mp4")
	case "mixed":
		content, mediaRefs, err = c.collectMixedMedia(c.ctx, scope, msg)
	default:
		return c.respondImmediate(reqID, "Unsupported WeCom message type: "+msg.MsgType)
	}
	if err != nil {
		return err
	}
	if msg.Quote != nil && msg.Quote.Text != nil {
		quoteText = strings.TrimSpace(msg.Quote.Text.Content)
		if content == "" {
			content = quoteText
		}
	}
	if content == "" && len(mediaRefs) == 0 {
		return c.respondImmediate(reqID, "The WeCom message did not contain usable content.")
	}

	turn := wecomTurn{
		ReqID:     reqID,
		ChatID:    actualChatID,
		ChatType:  chatType,
		StreamID:  randomID(10),
		CreatedAt: time.Now(),
	}
	c.queueTurn(actualChatID, turn)
	if err := c.routes.Put(actualChatID, reqID, chatType, wecomRouteTTL); err != nil {
		logger.WarnCF("wecom", "Failed to persist req_id route", map[string]any{
			"chat_id": actualChatID,
			"req_id":  reqID,
			"error":   err.Error(),
		})
	}

	opening := ""
	if c.config.SendThinkingMessage {
		opening = "Processing..."
	}
	if err := c.sendStreamChunk(turn, false, opening); err != nil {
		return err
	}

	peer := bus.Peer{Kind: peerKind, ID: actualChatID}
	metadata := map[string]string{
		"channel":   "wecom",
		"req_id":    reqID,
		"chat_id":   actualChatID,
		"chat_type": msg.ChatType,
		"msg_id":    msg.MsgID,
		"msg_type":  msg.MsgType,
	}
	if quoteText != "" {
		metadata["quote_text"] = quoteText
	}

	c.HandleMessage(c.ctx, peer, msg.MsgID, senderID, actualChatID, content, mediaRefs, metadata, sender)
	return nil
}

func (c *WeComChannel) collectSingleMedia(
	ctx context.Context,
	scope, msgID string,
	payload interface {
		GetURL() string
		GetAESKey() string
	},
	label, fallbackExt string,
) ([]string, error) {
	if payload == nil || payload.GetURL() == "" {
		return nil, fmt.Errorf("%s payload is empty", label)
	}
	ref, err := c.storeRemoteMedia(ctx, scope, msgID, payload.GetURL(), payload.GetAESKey(), fallbackExt)
	if err != nil {
		return nil, err
	}
	return []string{ref}, nil
}

type mediaPayload struct {
	url    string
	aesKey string
}

func (p *mediaPayload) GetURL() string    { return p.url }
func (p *mediaPayload) GetAESKey() string { return p.aesKey }

func (c *WeComChannel) collectMixedMedia(
	ctx context.Context,
	scope string,
	msg wecomIncomingMessage,
) (string, []string, error) {
	if msg.Mixed == nil {
		return "", nil, fmt.Errorf("mixed message is empty")
	}

	var textParts []string
	var refs []string
	for idx, item := range msg.Mixed.MsgItem {
		switch item.MsgType {
		case "text":
			if item.Text != nil && strings.TrimSpace(item.Text.Content) != "" {
				textParts = append(textParts, strings.TrimSpace(item.Text.Content))
			}
		case "image":
			if item.Image != nil && item.Image.URL != "" {
				ref, err := c.storeRemoteMedia(
					ctx,
					scope,
					fmt.Sprintf("%s-%d", msg.MsgID, idx),
					item.Image.URL,
					item.Image.AESKey,
					".jpg",
				)
				if err != nil {
					return "", nil, err
				}
				refs = append(refs, ref)
			}
		case "file":
			if item.File != nil && item.File.URL != "" {
				ref, err := c.storeRemoteMedia(
					ctx,
					scope,
					fmt.Sprintf("%s-%d", msg.MsgID, idx),
					item.File.URL,
					item.File.AESKey,
					".bin",
				)
				if err != nil {
					return "", nil, err
				}
				refs = append(refs, ref)
			}
		}
	}

	content := strings.Join(textParts, "\n")
	if content == "" && len(refs) > 0 {
		content = "[media]"
	}
	return content, refs, nil
}

func (c *WeComChannel) respondImmediate(reqID, content string) error {
	turn := wecomTurn{
		ReqID:     reqID,
		StreamID:  randomID(10),
		CreatedAt: time.Now(),
	}
	return c.sendStreamChunk(turn, true, content)
}

func (c *WeComChannel) sendStreamReply(turn wecomTurn, content string) error {
	return c.sendStreamChunk(turn, true, content)
}

func (c *WeComChannel) sendStreamChunk(turn wecomTurn, finish bool, content string) error {
	return c.sendCommand(wecomCommand{
		Cmd:     wecomCmdRespondMsg,
		Headers: wecomHeaders{ReqID: turn.ReqID},
		Body: wecomRespondMsgBody{
			MsgType: "stream",
			Stream: &wecomStreamContent{
				ID:      turn.StreamID,
				Finish:  finish,
				Content: content,
			},
		},
	}, wecomCommandTimeout)
}

func (c *WeComChannel) sendTurnMedia(turn wecomTurn, uploaded *wecomOutboundMedia) error {
	if uploaded == nil {
		return fmt.Errorf("wecom outbound media is nil: %w", channels.ErrSendFailed)
	}
	if err := c.sendCommand(wecomCommand{
		Cmd:     wecomCmdRespondMsg,
		Headers: wecomHeaders{ReqID: turn.ReqID},
		Body:    uploaded.respondBody(),
	}, wecomCommandTimeout); err != nil {
		return err
	}
	return c.sendStreamChunk(turn, true, "")
}

func (c *WeComChannel) sendActivePush(chatID string, chatType uint32, content string) error {
	if strings.TrimSpace(chatID) == "" {
		return fmt.Errorf("empty chat ID: %w", channels.ErrSendFailed)
	}
	return c.sendCommand(wecomCommand{
		Cmd:     wecomCmdSendMsg,
		Headers: wecomHeaders{ReqID: randomID(10)},
		Body: wecomSendMsgBody{
			ChatID:   chatID,
			ChatType: chatType,
			MsgType:  "markdown",
			Markdown: &wecomMarkdownContent{Content: content},
		},
	}, wecomCommandTimeout)
}

func (c *WeComChannel) sendActiveMedia(chatID string, chatType uint32, uploaded *wecomOutboundMedia) error {
	if strings.TrimSpace(chatID) == "" {
		return fmt.Errorf("empty chat ID: %w", channels.ErrSendFailed)
	}
	if uploaded == nil {
		return fmt.Errorf("wecom outbound media is nil: %w", channels.ErrSendFailed)
	}
	return c.sendCommand(wecomCommand{
		Cmd:     wecomCmdSendMsg,
		Headers: wecomHeaders{ReqID: randomID(10)},
		Body:    uploaded.sendBody(chatID, chatType),
	}, wecomCommandTimeout)
}

func (c *WeComChannel) sendCommand(cmd wecomCommand, timeout time.Duration) error {
	_, err := c.sendCommandAck(cmd, timeout)
	return err
}

func (c *WeComChannel) sendCommandAck(cmd wecomCommand, timeout time.Duration) (wecomEnvelope, error) {
	if c.commandSend != nil {
		return c.commandSend(cmd, timeout)
	}
	return c.writeCurrentAck(cmd, timeout)
}

func (c *WeComChannel) writeCurrentAck(cmd wecomCommand, timeout time.Duration) (wecomEnvelope, error) {
	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()
	if conn == nil {
		return wecomEnvelope{}, fmt.Errorf("wecom websocket not connected: %w", channels.ErrTemporary)
	}
	return c.writeAndWaitAck(conn, cmd, timeout)
}

func (c *WeComChannel) writeAndWait(conn *websocket.Conn, cmd wecomCommand, timeout time.Duration) error {
	_, err := c.writeAndWaitAck(conn, cmd, timeout)
	return err
}

func (c *WeComChannel) writeAndWaitAck(
	conn *websocket.Conn,
	cmd wecomCommand,
	timeout time.Duration,
) (wecomEnvelope, error) {
	if cmd.Headers.ReqID == "" {
		cmd.Headers.ReqID = randomID(10)
	}
	waitCh := make(chan wecomEnvelope, 1)
	c.pendingMu.Lock()
	c.pending[cmd.Headers.ReqID] = waitCh
	c.pendingMu.Unlock()
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, cmd.Headers.ReqID)
		c.pendingMu.Unlock()
	}()

	data, err := json.Marshal(cmd)
	if err != nil {
		return wecomEnvelope{}, fmt.Errorf("%w: %v", channels.ErrSendFailed, err)
	}
	c.connMu.Lock()
	err = conn.WriteMessage(websocket.TextMessage, data)
	c.connMu.Unlock()
	if err != nil {
		return wecomEnvelope{}, fmt.Errorf("%w: %v", channels.ErrTemporary, err)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case env := <-waitCh:
		if env.ErrCode != 0 {
			return wecomEnvelope{}, fmt.Errorf(
				"%w: wecom errcode=%d errmsg=%s",
				channels.ErrTemporary,
				env.ErrCode,
				env.ErrMsg,
			)
		}
		return env, nil
	case <-timer.C:
		return wecomEnvelope{}, fmt.Errorf("%w: timeout waiting for WeCom ack", channels.ErrTemporary)
	case <-c.ctx.Done():
		return wecomEnvelope{}, c.ctx.Err()
	}
}

func (c *WeComChannel) getTurn(chatID string) (wecomTurn, bool) {
	c.turnsMu.Lock()
	defer c.turnsMu.Unlock()
	queue := c.turns[chatID]
	if len(queue) == 0 {
		return wecomTurn{}, false
	}
	return queue[0], true
}

func (c *WeComChannel) deleteTurn(chatID string) {
	c.turnsMu.Lock()
	defer c.turnsMu.Unlock()
	queue := c.turns[chatID]
	if len(queue) <= 1 {
		delete(c.turns, chatID)
		return
	}
	c.turns[chatID] = queue[1:]
}

func (c *WeComChannel) queueTurn(chatID string, turn wecomTurn) {
	c.turnsMu.Lock()
	defer c.turnsMu.Unlock()
	c.turns[chatID] = append(c.turns[chatID], turn)
}

func (c *WeComChannel) consumeTurn(chatID string, turn wecomTurn) bool {
	c.turnsMu.Lock()
	defer c.turnsMu.Unlock()

	queue := c.turns[chatID]
	if len(queue) == 0 {
		return false
	}
	current := queue[0]
	if current.ReqID != turn.ReqID || current.StreamID != turn.StreamID {
		return false
	}
	if len(queue) == 1 {
		delete(c.turns, chatID)
		return true
	}
	c.turns[chatID] = queue[1:]
	return true
}

func (c *WeComChannel) clearTurns() {
	c.turnsMu.Lock()
	c.turns = make(map[string][]wecomTurn)
	c.turnsMu.Unlock()
}

func randomID(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if n <= 0 {
		n = 10
	}
	buf := make([]byte, n)
	for i := range buf {
		v, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		buf[i] = alphabet[v.Int64()]
	}
	return string(buf)
}

func (s *wecomStreamer) Update(ctx context.Context, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	if err := s.validateActiveTurn(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if !s.lastSentAt.IsZero() {
		wait := time.Until(s.lastSentAt.Add(wecomStreamMinInterval))
		if wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
	}

	if err := s.channel.sendStreamChunk(s.turn, false, content); err != nil {
		return err
	}
	s.content = content
	s.lastSentAt = time.Now()
	return nil
}

func (s *wecomStreamer) Finalize(ctx context.Context, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	if err := s.validateActiveTurn(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.channel.sendStreamChunk(s.turn, true, content); err != nil {
		return err
	}

	s.content = content
	s.closed = true
	s.channel.consumeTurn(s.chatID, s.turn)
	return nil
}

func (s *wecomStreamer) Cancel(_ context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	if s.validateActiveTurn() == nil {
		_ = s.channel.sendStreamChunk(s.turn, true, s.content)
		s.channel.consumeTurn(s.chatID, s.turn)
	}
	s.closed = true
}

func (s *wecomStreamer) validateActiveTurn() error {
	if time.Since(s.turn.CreatedAt) > wecomStreamMaxDuration {
		s.channel.consumeTurn(s.chatID, s.turn)
		return fmt.Errorf("wecom streaming unavailable: turn expired")
	}
	current, ok := s.channel.getTurn(s.chatID)
	if !ok || current.ReqID != s.turn.ReqID || current.StreamID != s.turn.StreamID {
		return fmt.Errorf("wecom streaming unavailable: turn no longer active")
	}
	return nil
}
