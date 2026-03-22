package weixin

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// WeixinChannel is the Weixin channel implementation over Tencent iLink REST API.
type WeixinChannel struct {
	*channels.BaseChannel
	api    *ApiClient
	config config.WeixinConfig
	ctx    context.Context
	cancel context.CancelFunc
	bus    *bus.MessageBus
	// contextTokens stores the last context_token per user (from_user_id → context_token).
	// This is required by the iLink API to associate replies with the right chat session.
	contextTokens sync.Map
}

func init() {
	channels.RegisterFactory("weixin", func(cfg *config.Config, bus *bus.MessageBus) (channels.Channel, error) {
		return NewWeixinChannel(cfg.Channels.Weixin, bus)
	})
}

// NewWeixinChannel creates a new WeixinChannel from config.
func NewWeixinChannel(cfg config.WeixinConfig, messageBus *bus.MessageBus) (*WeixinChannel, error) {
	api, err := NewApiClient(cfg.BaseURL, cfg.Token, cfg.Proxy)
	if err != nil {
		return nil, fmt.Errorf("weixin: failed to create API client: %w", err)
	}

	base := channels.NewBaseChannel(
		"weixin",
		cfg,
		messageBus,
		cfg.AllowFrom,
		channels.WithMaxMessageLength(4000),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &WeixinChannel{
		BaseChannel: base,
		api:         api,
		config:      cfg,
		bus:         messageBus,
	}, nil
}

func (c *WeixinChannel) Start(ctx context.Context) error {
	logger.InfoC("weixin", "Starting Weixin channel")
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.SetRunning(true)
	go c.pollLoop(c.ctx)
	logger.InfoC("weixin", "Weixin channel started")
	return nil
}

func (c *WeixinChannel) Stop(ctx context.Context) error {
	logger.InfoC("weixin", "Stopping Weixin channel")
	c.SetRunning(false)
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

// pollLoop is the long-poll receive loop. It runs until ctx is canceled.
func (c *WeixinChannel) pollLoop(ctx context.Context) {
	const (
		defaultPollTimeoutMs = 35_000
		retryDelay           = 2 * time.Second
		backoffDelay         = 30 * time.Second
		maxConsecutiveFails  = 3
	)

	consecutiveFails := 0
	getUpdatesBuf := ""
	nextTimeoutMs := defaultPollTimeoutMs

	for {
		select {
		case <-ctx.Done():
			logger.InfoC("weixin", "Weixin poll loop stopped")
			return
		default:
		}

		// Build a context with timeout slightly longer than the long-poll
		pollCtx, pollCancel := context.WithTimeout(ctx, time.Duration(nextTimeoutMs+5000)*time.Millisecond)

		resp, err := c.api.GetUpdates(pollCtx, GetUpdatesReq{
			GetUpdatesBuf: getUpdatesBuf,
		})
		pollCancel()

		if err != nil {
			// Check if we're shutting down
			if ctx.Err() != nil {
				return
			}

			consecutiveFails++
			logger.WarnCF("weixin", "getUpdates failed", map[string]any{
				"error":   err.Error(),
				"attempt": consecutiveFails,
			})

			if consecutiveFails >= maxConsecutiveFails {
				logger.ErrorCF("weixin", "Too many consecutive failures, backing off", map[string]any{
					"duration": backoffDelay,
				})
				consecutiveFails = 0
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoffDelay):
				}
			} else {
				select {
				case <-ctx.Done():
					return
				case <-time.After(retryDelay):
				}
			}
			continue
		}

		// Check for API-level error codes (-14 = session expired)
		const sessionExpiredErrcode = -14
		if resp.Errcode != 0 || (resp.Ret != 0 && resp.Ret != sessionExpiredErrcode) {
			consecutiveFails++
			logger.ErrorCF("weixin", "getUpdates API error", map[string]any{
				"ret":     resp.Ret,
				"errcode": resp.Errcode,
				"errmsg":  resp.Errmsg,
			})
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryDelay):
			}
			continue
		}

		if resp.Errcode == sessionExpiredErrcode || resp.Ret == sessionExpiredErrcode {
			logger.ErrorC("weixin", "Session expired — please re-run login")
			// Pause for a long time to avoid hammering with a bad token
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Minute):
			}
			continue
		}

		consecutiveFails = 0

		// Update the long-poll timeout from server hint
		if resp.LongpollingTimeoutMs > 0 {
			nextTimeoutMs = resp.LongpollingTimeoutMs
		}

		// Advance cursor
		if resp.GetUpdatesBuf != "" {
			getUpdatesBuf = resp.GetUpdatesBuf
		}

		// Dispatch messages
		for _, msg := range resp.Msgs {
			c.handleInboundMessage(ctx, msg)
		}
	}
}

// handleInboundMessage converts a WeixinMessage to a bus.InboundMessage.
func (c *WeixinChannel) handleInboundMessage(ctx context.Context, msg WeixinMessage) {
	fromUserID := msg.FromUserID
	if fromUserID == "" {
		return
	}

	// Build text content from item_list
	var parts []string
	for _, item := range msg.ItemList {
		switch item.Type {
		case MessageItemTypeText:
			if item.TextItem != nil && item.TextItem.Text != "" {
				parts = append(parts, item.TextItem.Text)
			}
		case MessageItemTypeVoice:
			if item.VoiceItem != nil && item.VoiceItem.Text != "" {
				// Use voice → text transcription from server
				parts = append(parts, item.VoiceItem.Text)
			} else {
				parts = append(parts, "[voice message]")
			}
		case MessageItemTypeImage:
			parts = append(parts, "[image]")
		case MessageItemTypeFile:
			if item.FileItem != nil && item.FileItem.FileName != "" {
				parts = append(parts, fmt.Sprintf("[file: %s]", item.FileItem.FileName))
			} else {
				parts = append(parts, "[file]")
			}
		case MessageItemTypeVideo:
			parts = append(parts, "[video]")
		}
	}

	content := strings.Join(parts, "\n")
	if content == "" {
		return
	}

	sender := bus.SenderInfo{
		Platform:    "weixin",
		PlatformID:  fromUserID,
		CanonicalID: identity.BuildCanonicalID("weixin", fromUserID),
		Username:    fromUserID,
		DisplayName: fromUserID,
	}

	if !c.IsAllowedSender(sender) {
		logger.DebugCF("weixin", "Message rejected by allowlist", map[string]any{
			"from_user_id": fromUserID,
		})
		return
	}

	messageID := msg.ClientID
	if messageID == "" {
		messageID = uuid.New().String()
	}

	peer := bus.Peer{Kind: "direct", ID: fromUserID}

	metadata := map[string]string{
		"from_user_id":  fromUserID,
		"context_token": msg.ContextToken,
		"session_id":    msg.SessionID,
	}

	logger.DebugCF("weixin", "Received message", map[string]any{
		"from_user_id": fromUserID,
		"content_len":  len(content),
	})

	// Store context_token for outbound reply association
	if msg.ContextToken != "" {
		c.contextTokens.Store(fromUserID, msg.ContextToken)
	}

	c.HandleMessage(ctx, peer, messageID, fromUserID, fromUserID, content, nil, metadata, sender)
}

// Send implements channels.Channel by sending a text message to the WeChat user.
func (c *WeixinChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	if msg.Content == "" {
		return nil
	}

	// We need a context_token to send a reply. It should be stored in the conversation metadata.
	// The chat_id is the weixin user_id (from_user_id).
	toUserID := msg.ChatID

	// Retrieve context_token from our per-user map (stored on last inbound)
	contextToken := ""
	if ct, ok := c.contextTokens.Load(toUserID); ok {
		contextToken, _ = ct.(string)
	}

	// If we don't have a context token for this user, we cannot send a valid reply.
	// Treat this as a non-temporary error so the manager doesn't keep retrying.
	if contextToken == "" {
		logger.ErrorCF("weixin", "Missing context token, cannot send message", map[string]any{
			"to_user_id": toUserID,
		})
		return fmt.Errorf("weixin send: %w: missing context token for chat %s", channels.ErrSendFailed, toUserID)
	}
	clientID := "picoclaw-" + uuid.New().String()

	req := SendMessageReq{
		Msg: WeixinMessage{
			FromUserID:   "",
			ToUserID:     toUserID,
			ClientID:     clientID,
			MessageType:  MessageTypeBot,
			MessageState: MessageStateFinish,
			ItemList: []MessageItem{
				{
					Type: MessageItemTypeText,
					TextItem: &TextItem{
						Text: msg.Content,
					},
				},
			},
			ContextToken: contextToken,
		},
	}

	if err := c.api.SendMessage(ctx, req); err != nil {
		logger.ErrorCF("weixin", "Failed to send message", map[string]any{
			"to_user_id": toUserID,
			"error":      err.Error(),
		})
		return fmt.Errorf("weixin send: %w", channels.ErrTemporary)
	}

	return nil
}
