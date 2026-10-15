package qqc2c

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// Channel is the QQ single-chat (C2C) channel.
//
// It implements channels.Channel plus the optional capability interfaces:
// TypingCapable, StreamingCapable, MessageLengthProvider, CommandRegistrarCapable.
type Channel struct {
	*channels.BaseChannel

	bc     *config.Channel
	config *config.QQC2CSettings

	tm  *tokenManager
	api *apiClient
	ws  *wsClient

	ctx    context.Context
	cancel context.CancelFunc

	msgSeq       sync.Map // openid -> *atomic.Int64
	lastMsgID    sync.Map // openid -> string
	streams      sync.Map // openid -> *streamSession
	interactions sync.Map // interaction id -> chan string
	seen         sync.Map // message id -> struct{} (dedup)
}

var _ channels.Channel = (*Channel)(nil)

// NewChannel builds the channel from decoded settings.
func NewChannel(bc *config.Channel, cfg *config.QQC2CSettings, messageBus *bus.MessageBus) (*Channel, error) {
	if cfg == nil {
		return nil, fmt.Errorf("qq_c2c: settings missing")
	}
	opts := []channels.BaseChannelOption{
		channels.WithMaxMessageLength(cfg.MaxMessageLength),
		channels.WithReasoningChannelID(bc.ReasoningChannelID),
	}
	c := &Channel{bc: bc, config: cfg}
	c.BaseChannel = channels.NewBaseChannel(bc.Name(), cfg, messageBus, bc.AllowFrom, opts...)
	c.SetOwner(c)
	return c, nil
}

// ---------------------------------------------------------------------------
// lifecycle
// ---------------------------------------------------------------------------

func (c *Channel) Start(ctx context.Context) error {
	if strings.TrimSpace(c.config.AppID) == "" || strings.TrimSpace(c.config.AppSecret.String()) == "" {
		return fmt.Errorf("qq_c2c: app_id and app_secret are required")
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.tm = newTokenManager(c.config.AppID, c.config.AppSecret.String())
	c.api = newAPIClient(c.config.APIBase, c.tm)
	c.ws = newWSClient(c.api, c.config.Intents, func(t string, d json.RawMessage) {
		c.handleEvent(t, d)
	})

	// Verify credentials up-front so Start fails loudly instead of silently.
	if _, err := c.tm.Access(c.ctx); err != nil {
		return fmt.Errorf("qq_c2c: access token: %w", err)
	}

	go func() {
		if err := c.ws.Run(c.ctx); err != nil && c.ctx.Err() == nil {
			logger.ErrorCF("qq_c2c", "websocket run ended", map[string]any{"error": err.Error()})
			c.SetRunning(false)
		}
	}()

	c.SetRunning(true)
	logger.InfoCF("qq_c2c", "QQ C2C channel started", map[string]any{
		"app_id":    c.config.AppID,
		"streaming": c.config.Streaming,
		"markdown":  c.config.Markdown,
	})
	return nil
}

func (c *Channel) Stop(_ context.Context) error {
	if c.cancel != nil {
		c.cancel()
	}
	if c.ws != nil {
		c.ws.SendClose()
	}
	c.SetRunning(false)
	logger.InfoC("qq_c2c", "QQ C2C channel stopped")
	return nil
}

func (c *Channel) MaxMessageLength() int { return c.config.MaxMessageLength }

// ---------------------------------------------------------------------------
// events
// ---------------------------------------------------------------------------

func (c *Channel) handleEvent(t string, d json.RawMessage) {
	switch t {
	case "C2C_MESSAGE_CREATE":
		var data c2cMessageData
		if err := json.Unmarshal(d, &data); err != nil {
			logger.ErrorCF("qq_c2c", "decode C2C_MESSAGE_CREATE failed", map[string]any{"error": err.Error()})
			return
		}
		if err := c.onC2CMessage(&data); err != nil {
			logger.ErrorCF("qq_c2c", "handle C2C message failed", map[string]any{"error": err.Error()})
		}
	case "INTERACTION_CREATE":
		var data interactionData
		if err := json.Unmarshal(d, &data); err != nil {
			return
		}
		c.onInteraction(&data)
	case "FRIEND_ADD", "C2C_MSG_RECEIVE":
		// Informational; the user becoming reachable is enough.
	default:
		logger.DebugCF("qq_c2c", "unhandled event", map[string]any{"event": t})
	}
}

func (c *Channel) onC2CMessage(data *c2cMessageData) error {
	openID := data.Author.UserOpenID
	if openID == "" {
		return fmt.Errorf("event without user_openid")
	}
	if data.ID != "" {
		if _, loaded := c.seen.LoadOrStore(data.ID, struct{}{}); loaded {
			return nil // duplicate delivery
		}
	}

	content := strings.TrimSpace(data.Content)
	if content == "" {
		return nil
	}

	// Remember the message id: passive replies must reference it.
	if data.ID != "" {
		c.lastMsgID.Store(openID, data.ID)
	}
	seq := new(atomic.Int64)
	seq.Store(1)
	c.msgSeq.Store(openID, seq)

	sender := bus.SenderInfo{
		Platform:    "qq_c2c",
		PlatformID:  openID,
		CanonicalID: openID,
	}
	if !c.IsAllowedSender(sender) {
		logger.WarnCF("qq_c2c", "sender rejected by allow_from", map[string]any{"sender": openID})
		return nil
	}

	inboundCtx := bus.InboundContext{
		Channel:   c.Name(),
		ChatID:    openID,
		SenderID:  openID,
		MessageID: data.ID,
	}
	logger.InfoCF("qq_c2c", "received C2C message", map[string]any{
		"sender": openID,
		"length": len(content),
	})
	return c.HandleInboundContext(c.ctx, openID, content, nil, inboundCtx, sender)
}

func (c *Channel) onInteraction(data *interactionData) {
	// Always acknowledge within 3 seconds, even if nobody is waiting.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = c.api.PutInteraction(ctx, data.ID, 0)

	if ch, ok := c.interactions.LoadAndDelete(data.ID); ok {
		if waiter, ok := ch.(chan string); ok {
			select {
			case waiter <- data.Data.Resolved.ButtonData:
			default:
			}
		}
	}
	logger.InfoCF("qq_c2c", "interaction received", map[string]any{
		"type": data.Type,
		"data": data.Data.Resolved.ButtonData,
	})
}

// ---------------------------------------------------------------------------
// outbound
// ---------------------------------------------------------------------------

// Send performs a *passive reply*: QQ requires the inbound msg_id, and each
// msg_id may be answered at most 4 times within 60 minutes.
func (c *Channel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}
	openID := msg.ChatID
	if openID == "" {
		return nil, fmt.Errorf("qq_c2c: empty chat id")
	}

	msgID := ""
	if v, ok := c.lastMsgID.Load(openID); ok {
		msgID, _ = v.(string)
	}
	// Also accept an explicit reply target from the agent.
	if msg.ReplyToMessageID != "" {
		msgID = msg.ReplyToMessageID
	}
	if msgID == "" {
		return nil, fmt.Errorf("qq_c2c: no inbound msg_id for %s (QQ only allows passive replies; the user must talk to the bot first)", openID)
	}

	seq := c.nextSeq(openID)

	body := &MessageToCreate{MsgID: msgID, MsgSeq: seq}
	if c.config.Markdown {
		body.MsgType = 2
		body.Markdown = &Markdown{Content: msg.Content}
	} else {
		body.MsgType = 0
		body.Content = msg.Content
	}

	sent, err := c.api.PostC2C(ctx, openID, body)
	if err != nil && isMsgSeqErr(err) {
		body.MsgSeq = c.nextSeq(openID)
		sent, err = c.api.PostC2C(ctx, openID, body)
	}
	if err != nil {
		logger.ErrorCF("qq_c2c", "send failed", map[string]any{
			"chat_id": openID,
			"msg_seq": body.MsgSeq,
			"error":   err.Error(),
		})
		if isNotFoundErr(err) {
			return nil, fmt.Errorf("qq_c2c: %w", channels.ErrSendFailed)
		}
		return nil, fmt.Errorf("qq_c2c send: %w", channels.ErrTemporary)
	}
	if sent == nil {
		return nil, nil
	}
	logger.InfoCF("qq_c2c", "message sent", map[string]any{
		"chat_id": openID,
		"msg_id":  sent.ID,
		"msg_seq": body.MsgSeq,
	})
	return []string{sent.ID}, nil
}

func (c *Channel) nextSeq(openID string) int {
	v, ok := c.msgSeq.Load(openID)
	if !ok {
		s := new(atomic.Int64)
		v, _ = c.msgSeq.LoadOrStore(openID, s)
	}
	s, _ := v.(*atomic.Int64)
	if s == nil {
		return 1
	}
	return int(s.Add(1))
}

// ---------------------------------------------------------------------------
// capabilities
// ---------------------------------------------------------------------------

// StartTyping implements channels.TypingCapable using msg_type=6 input_notify.
func (c *Channel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	if !c.config.Typing {
		return func() {}, nil
	}
	msgID := ""
	if v, ok := c.lastMsgID.Load(chatID); ok {
		msgID, _ = v.(string)
	}
	if msgID == "" {
		return func() {}, nil
	}
	body := &MessageToCreate{
		MsgType:     6,
		InputNotify: &InputNotify{InputType: 1, InputSecond: 60},
		MsgID:       msgID,
		MsgSeq:      c.nextSeq(chatID),
	}
	if _, err := c.api.PostC2C(ctx, chatID, body); err != nil {
		logger.DebugCF("qq_c2c", "typing notify failed", map[string]any{"error": err.Error()})
	}
	return func() {}, nil
}

// BeginStream implements channels.StreamingCapable.
func (c *Channel) BeginStream(ctx context.Context, chatID string) (channels.Streamer, error) {
	if !c.config.Streaming {
		return nil, fmt.Errorf("qq_c2c: streaming disabled in settings")
	}
	msgID := ""
	if v, ok := c.lastMsgID.Load(chatID); ok {
		msgID, _ = v.(string)
	}
	if msgID == "" {
		return nil, fmt.Errorf("qq_c2c: no inbound msg_id for streaming")
	}
	s := &streamSession{
		api:      c.api,
		openID:   chatID,
		msgID:    msgID,
		msgSeq:   c.nextSeq(chatID),
		interval: time.Duration(max(c.config.StreamIntervalMS, 100)) * time.Millisecond,
		lastSent: time.Now().Add(-time.Hour),
	}
	c.streams.Store(chatID, s)
	return s, nil
}

// AskWithButtons sends a question with a keyboard and waits for the chosen value.
func (c *Channel) AskWithButtons(ctx context.Context, chatID, question string, options []Button, timeout time.Duration) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("qq_c2c: no options")
	}
	msgID := ""
	if v, ok := c.lastMsgID.Load(chatID); ok {
		msgID, _ = v.(string)
	}
	body := &MessageToCreate{
		MsgType: 2,
		Markdown: &Markdown{Content: question},
		Keyboard: BuildKeyboard([][]Button{options}),
		MsgID:    msgID,
		MsgSeq:   c.nextSeq(chatID),
	}
	if _, err := c.api.PostC2C(ctx, chatID, body); err != nil {
		return "", err
	}
	// Note: interactions are matched by id, so we register a broadcast waiter.
	ch := make(chan string, 1)
	c.interactions.Store("pending:"+chatID, ch)
	defer c.interactions.Delete("pending:" + chatID)
	select {
	case v := <-ch:
		return v, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("qq_c2c: interaction timeout")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
