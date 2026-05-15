// PicoClaw - Ultra-lightweight personal AI agent

package adapters

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/agent/interfaces"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
)

// channelManagerAdapter wraps *channels.Manager to implement interfaces.ChannelManager.
type channelManagerAdapter struct {
	inner *channels.Manager
}

// NewChannelManager creates an adapter for *channels.Manager.
func NewChannelManager(inner *channels.Manager) interfaces.ChannelManager {
	return &channelManagerAdapter{inner: inner}
}

func (a *channelManagerAdapter) GetChannel(name string) (channels.Channel, bool) {
	return a.inner.GetChannel(name)
}

func (a *channelManagerAdapter) GetEnabledChannels() []string {
	return a.inner.GetEnabledChannels()
}

func (a *channelManagerAdapter) InvokeTypingStop(channel, chatID string) {
	a.inner.InvokeTypingStop(channel, chatID)
}

func (a *channelManagerAdapter) SendMessage(ctx context.Context, msg bus.OutboundMessage) error {
	return a.inner.SendMessage(ctx, msg)
}

func (a *channelManagerAdapter) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	return a.inner.SendMedia(ctx, msg)
}

func (a *channelManagerAdapter) SendPlaceholder(ctx context.Context, channel, chatID string) bool {
	return a.inner.SendPlaceholder(ctx, channel, chatID)
}

func (a *channelManagerAdapter) DismissToolFeedback(
	ctx context.Context, channel, chatID string, outboundCtx *bus.InboundContext,
) {
	a.inner.DismissToolFeedback(ctx, channel, chatID, outboundCtx)
}

func (a *channelManagerAdapter) DismissToolFeedbackForSession(
	ctx context.Context,
	channel, chatID string,
	outboundCtx *bus.InboundContext,
	sessionKey string,
) {
	a.inner.DismissToolFeedbackForSession(ctx, channel, chatID, outboundCtx, sessionKey)
}
