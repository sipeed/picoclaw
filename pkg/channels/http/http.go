package http

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func init() {
	channels.RegisterFactory("http", NewHTTPChannel)
}

type HTTPChannel struct {
	*channels.BaseChannel
}

func NewHTTPChannel(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
	bc := channels.NewBaseChannel("http", nil, b, nil)
	return &HTTPChannel{
		BaseChannel: bc,
	}, nil
}

func (c *HTTPChannel) Start(ctx context.Context) error {
	c.SetRunning(true)
	return nil
}

func (c *HTTPChannel) Stop(ctx context.Context) error {
	c.SetRunning(false)
	return nil
}

func (c *HTTPChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	logger.InfoCF("channels", "HTTP channel received outbound message", map[string]any{
		"chat_id": msg.ChatID,
		"content": msg.Content,
	})
	// For synchronous HTTP, the response is usually handled by the caller of ProcessDirectWithChannel.
	// Asynchronous messages (e.g. from subagents) will just be logged here for now.
	return nil, nil
}
