// PicoClaw - Ultra-lightweight personal AI agent

package interfaces

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
)

// MessageBus publishes inbound and outbound messages.
// It is the primary communication channel for the agent loop.
type MessageBus interface {
	// PublishInbound sends an inbound message to be processed.
	PublishInbound(ctx context.Context, msg bus.InboundMessage) error

	// PublishOutbound sends an outbound message to the appropriate channel.
	PublishOutbound(ctx context.Context, msg bus.OutboundMessage) error

	// PublishOutboundMedia sends an outbound media message.
	PublishOutboundMedia(ctx context.Context, msg bus.OutboundMediaMessage) error

	// InboundChan returns the channel for receiving inbound messages.
	InboundChan() <-chan bus.InboundMessage
}

// ChannelManager manages channel lifecycle and provides channel access.
type ChannelManager interface {
	// GetChannel returns the channel with the given name.
	GetChannel(name string) (channels.Channel, bool)

	// GetEnabledChannels returns the list of enabled channel names.
	GetEnabledChannels() []string

	// InvokeTypingStop signals that typing has stopped.
	InvokeTypingStop(channel, chatID string)

	// SendMessage sends a text message to the specified channel and chat.
	SendMessage(ctx context.Context, msg bus.OutboundMessage) error

	// SendMedia sends a media message to the specified channel and chat.
	SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error

	// SendPlaceholder sends a placeholder message (e.g., for audio transcription).
	SendPlaceholder(ctx context.Context, channel, chatID string) bool
}
