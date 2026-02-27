package xiaoyi

import (
	"context"
	"fmt"
	"strings"

	xiaoyi "github.com/ystyle/xiaoyi-agent-sdk/pkg/client"
	"github.com/ystyle/xiaoyi-agent-sdk/pkg/types"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type XiaoYiChannel struct {
	*channels.BaseChannel
	config *config.Config
	client xiaoyi.Client
	ctx    context.Context
	cancel context.CancelFunc
}

func NewXiaoYiChannel(cfg *config.Config, messageBus *bus.MessageBus) (*XiaoYiChannel, error) {
	xiaoyiCfg := cfg.Channels.XiaoYi

	base := channels.NewBaseChannel(
		"xiaoyi",
		xiaoyiCfg,
		messageBus,
		xiaoyiCfg.AllowFrom,
	)

	return &XiaoYiChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

func (c *XiaoYiChannel) Start(ctx context.Context) error {
	xiaoyiCfg := c.config.Channels.XiaoYi

	if xiaoyiCfg.AK == "" || xiaoyiCfg.SK == "" || xiaoyiCfg.AgentID == "" {
		return fmt.Errorf("xiaoyi ak, sk and agent_id are required")
	}

	logger.InfoC("xiaoyi", "Starting XiaoYi channel")

	cfg := &types.Config{
		AK:           xiaoyiCfg.AK,
		SK:           xiaoyiCfg.SK,
		AgentID:      xiaoyiCfg.AgentID,
		WSUrl1:       xiaoyiCfg.WSUrl1,
		WSUrl2:       xiaoyiCfg.WSUrl2,
		SingleServer: true,
	}

	c.client = xiaoyi.New(cfg)

	c.client.OnMessage(func(ctx context.Context, msg types.Message) error {
		return c.handleMessage(ctx, msg)
	})

	c.client.OnClear(func(sessionID string) {
		logger.InfoCF("xiaoyi", "Session cleared", map[string]any{
			"session": sessionID,
		})
	})

	c.client.OnCancel(func(sessionID, taskID string) {
		logger.InfoCF("xiaoyi", "Task canceled", map[string]any{
			"session": sessionID,
			"task":    taskID,
		})
	})

	c.client.OnError(func(serverID string, err error) {
		logger.ErrorCF("xiaoyi", "Server error", map[string]any{
			"server": serverID,
			"error":  err.Error(),
		})
	})

	c.ctx, c.cancel = context.WithCancel(ctx)

	if err := c.client.Connect(c.ctx); err != nil {
		return fmt.Errorf("failed to connect xiaoyi: %w", err)
	}

	c.SetRunning(true)
	logger.InfoC("xiaoyi", "XiaoYi channel started successfully")

	return nil
}

func (c *XiaoYiChannel) Stop(ctx context.Context) error {
	logger.InfoC("xiaoyi", "Stopping XiaoYi channel")
	c.SetRunning(false)

	if c.cancel != nil {
		c.cancel()
	}

	if c.client != nil {
		c.client.Close()
	}

	logger.InfoC("xiaoyi", "XiaoYi channel stopped")
	return nil
}

func (c *XiaoYiChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	parts := strings.SplitN(msg.ChatID, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid chat_id format: %w", channels.ErrSendFailed)
	}

	sessionID := parts[0]
	taskID := parts[1]

	logger.DebugCF("xiaoyi", "Sending message", map[string]any{
		"session": sessionID,
		"task":    taskID,
		"length":  len(msg.Content),
	})

	if err := c.client.ReplyStream(ctx, taskID, sessionID, msg.Content, false, false); err != nil {
		return fmt.Errorf("xiaoyi reply: %w", channels.ErrTemporary)
	}

	if err := c.client.SendStatus(ctx, taskID, sessionID, "Completed", "completed"); err != nil {
		logger.WarnCF("xiaoyi", "Failed to send completed status", map[string]any{"error": err.Error()})
	}

	return nil
}

func (c *XiaoYiChannel) handleMessage(ctx context.Context, msg types.Message) error {
	sessionID := msg.SessionID()
	taskID := msg.TaskID()
	text := strings.TrimSpace(msg.Text())

	logger.InfoCF("xiaoyi", "Received message", map[string]any{
		"session": sessionID,
		"task":    taskID,
		"text":    text,
	})

	sender := bus.SenderInfo{
		Platform:    "xiaoyi",
		PlatformID:  sessionID,
		CanonicalID: identity.BuildCanonicalID("xiaoyi", sessionID),
	}

	if !c.IsAllowedSender(sender) {
		logger.DebugCF("xiaoyi", "Message rejected by allowlist", map[string]any{
			"session": sessionID,
		})
		return nil
	}

	chatID := fmt.Sprintf("%s:%s", sessionID, taskID)
	peer := bus.Peer{Kind: "direct", ID: sessionID}

	if err := c.client.SendStatus(ctx, taskID, sessionID, "Processing...", "working"); err != nil {
		logger.WarnCF("xiaoyi", "Failed to send status", map[string]any{"error": err.Error()})
	}

	metadata := map[string]string{
		"task_id": taskID,
	}

	c.HandleMessage(c.ctx,
		peer,
		"",
		sessionID,
		chatID,
		text,
		nil,
		metadata,
		sender,
	)

	return nil
}
