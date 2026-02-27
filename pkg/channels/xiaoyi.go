package channels

import (
	"context"
	"fmt"
	"strings"
	"sync"

	xiaoyi "github.com/ystyle/xiaoyi-agent-sdk/pkg/client"
	"github.com/ystyle/xiaoyi-agent-sdk/pkg/types"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type ProcessFunc func(ctx context.Context, content, sessionKey, channel, chatID string) (string, error)

type XiaoYiChannel struct {
	*BaseChannel
	config      config.XiaoYiConfig
	client      xiaoyi.Client
	processFunc ProcessFunc
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
}

func NewXiaoYiChannel(cfg config.XiaoYiConfig, messageBus *bus.MessageBus) (*XiaoYiChannel, error) {
	base := NewBaseChannel("xiaoyi", cfg, messageBus, cfg.AllowFrom)

	return &XiaoYiChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

func (c *XiaoYiChannel) SetProcessFunc(fn ProcessFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.processFunc = fn
}

func (c *XiaoYiChannel) Start(ctx context.Context) error {
	if c.config.AK == "" || c.config.SK == "" || c.config.AgentID == "" {
		return fmt.Errorf("xiaoyi ak, sk and agent_id are required")
	}

	logger.InfoC("xiaoyi", "Starting XiaoYi channel")

	cfg := &types.Config{
		AK:           c.config.AK,
		SK:           c.config.SK,
		AgentID:      c.config.AgentID,
		WSUrl1:       c.config.WSUrl1,
		WSUrl2:       c.config.WSUrl2,
		SingleServer: true,
	}

	c.client = xiaoyi.New(cfg)

	c.client.OnMessage(func(ctx context.Context, msg types.Message) error {
		sessionID := msg.SessionID()
		taskID := msg.TaskID()
		text := strings.TrimSpace(msg.Text())

		logger.InfoCF("xiaoyi", "Received message", map[string]any{
			"session": sessionID,
			"task":    taskID,
			"text":    text,
		})

		chatID := fmt.Sprintf("%s:%s", sessionID, taskID)

		metadata := map[string]string{
			"session_id": sessionID,
			"task_id":    taskID,
		}

		c.HandleMessage(sessionID, chatID, text, []string{}, metadata)

		return c.client.SendStatus(ctx, taskID, sessionID, "处理中...")
	})

	c.client.OnClear(func(sessionID string) {
		logger.InfoCF("xiaoyi", "Session cleared", map[string]any{
			"session": sessionID,
		})
	})

	c.client.OnCancel(func(sessionID, taskID string) {
		logger.InfoCF("xiaoyi", "Task cancelled", map[string]any{
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

	c.setRunning(true)
	logger.InfoC("xiaoyi", "XiaoYi channel started successfully")

	return nil
}

func (c *XiaoYiChannel) Stop(ctx context.Context) error {
	logger.InfoC("xiaoyi", "Stopping XiaoYi channel")
	c.setRunning(false)

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
		return fmt.Errorf("xiaoyi channel not running")
	}

	parts := strings.SplitN(msg.ChatID, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid chat_id format, expected sessionID:taskID")
	}

	sessionID := parts[0]
	taskID := parts[1]

	logger.InfoCF("xiaoyi", "Sending message", map[string]any{
		"session": sessionID,
		"task":    taskID,
		"length":  len(msg.Content),
	})

	return c.client.Reply(ctx, taskID, sessionID, msg.Content)
}
