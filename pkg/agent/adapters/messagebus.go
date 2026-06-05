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

func (a *messageBusAdapter) AckInbound(ctx context.Context, msg bus.InboundMessage) error {
	return a.inner.AckInbound(ctx, msg)
}

func (a *messageBusAdapter) ReleaseInbound(ctx context.Context, msg bus.InboundMessage, cause error) error {
	return a.inner.ReleaseInbound(ctx, msg, cause)
}

func (a *messageBusAdapter) PendingInboundSpool(ctx context.Context) ([]bus.InboundMessage, error) {
	return a.inner.PendingInboundSpool(ctx)
}

func (a *messageBusAdapter) PublishObserved(ctx context.Context, msg bus.ObservedMessage) error {
	return a.inner.PublishObserved(ctx, msg)
}

func (a *messageBusAdapter) PublishOutbound(ctx context.Context, msg bus.OutboundMessage) error {
	return a.inner.PublishOutbound(ctx, msg)
}

func (a *messageBusAdapter) PublishOutboundMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	return a.inner.PublishOutboundMedia(ctx, msg)
}

func (a *messageBusAdapter) GetStreamer(ctx context.Context, channel, chatID, sessionKey string) (bus.Streamer, bool) {
	return a.inner.GetStreamer(ctx, channel, chatID, sessionKey)
}

func (a *messageBusAdapter) InboundChan() <-chan bus.InboundMessage {
	return a.inner.InboundChan()
}

func (a *messageBusAdapter) ObservedChan() <-chan bus.ObservedMessage {
	return a.inner.ObservedChan()
}
