// PicoClaw - Ultra-lightweight personal AI agent

package adapters

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/agent/interfaces"
	"github.com/sipeed/picoclaw/pkg/bus"
)

// messageBusAdapter wraps *bus.MessageBus to implement interfaces.MessageBus.
type messageBusAdapter struct {
	inner *bus.MessageBus
}

// NewMessageBus creates an adapter for *bus.MessageBus.
func NewMessageBus(inner *bus.MessageBus) interfaces.MessageBus {
	return &messageBusAdapter{inner: inner}
}

func (a *messageBusAdapter) PublishInbound(ctx context.Context, msg bus.InboundMessage) error {
	return a.inner.PublishInbound(ctx, msg)
}

func (a *messageBusAdapter) PublishOutbound(ctx context.Context, msg bus.OutboundMessage) error {
	return a.inner.PublishOutbound(ctx, msg)
}

func (a *messageBusAdapter) PublishOutboundMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	return a.inner.PublishOutboundMedia(ctx, msg)
}

func (a *messageBusAdapter) InboundChan() <-chan bus.InboundMessage {
	return a.inner.InboundChan()
}
