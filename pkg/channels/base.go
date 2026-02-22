package channels

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/sipeed/picoclaw/pkg/bus"
)

type Channel interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Send(ctx context.Context, msg bus.OutboundMessage) error
	IsRunning() bool
	IsAllowed(senderID string) bool
}

// BaseChannelOption is a functional option for configuring a BaseChannel.
type BaseChannelOption func(*BaseChannel)

// WithMaxMessageLength sets the maximum message length (in runes) for a channel.
// Messages exceeding this limit will be automatically split by the Manager.
// A value of 0 means no limit.
func WithMaxMessageLength(n int) BaseChannelOption {
	return func(c *BaseChannel) { c.maxMessageLength = n }
}

// MessageLengthProvider is an opt-in interface that channels implement
// to advertise their maximum message length. The Manager uses this via
// type assertion to decide whether to split outbound messages.
type MessageLengthProvider interface {
	MaxMessageLength() int
}

type BaseChannel struct {
	config           any
	bus              *bus.MessageBus
	running          atomic.Bool
	name             string
	allowList        []string
	maxMessageLength int
}

func NewBaseChannel(
	name string,
	config any,
	bus *bus.MessageBus,
	allowList []string,
	opts ...BaseChannelOption,
) *BaseChannel {
	bc := &BaseChannel{
		config:    config,
		bus:       bus,
		name:      name,
		allowList: allowList,
	}
	for _, opt := range opts {
		opt(bc)
	}
	return bc
}

// MaxMessageLength returns the maximum message length (in runes) for this channel.
// A value of 0 means no limit.
func (c *BaseChannel) MaxMessageLength() int {
	return c.maxMessageLength
}

func (c *BaseChannel) Name() string {
	return c.name
}

func (c *BaseChannel) IsRunning() bool {
	return c.running.Load()
}

func (c *BaseChannel) IsAllowed(senderID string) bool {
	if len(c.allowList) == 0 {
		return true
	}

	// Extract parts from compound senderID like "123456|username"
	idPart := senderID
	userPart := ""
	if idx := strings.Index(senderID, "|"); idx > 0 {
		idPart = senderID[:idx]
		userPart = senderID[idx+1:]
	}

	for _, allowed := range c.allowList {
		// Strip leading "@" from allowed value for username matching
		trimmed := strings.TrimPrefix(allowed, "@")
		allowedID := trimmed
		allowedUser := ""
		if idx := strings.Index(trimmed, "|"); idx > 0 {
			allowedID = trimmed[:idx]
			allowedUser = trimmed[idx+1:]
		}

		// Support either side using "id|username" compound form.
		// This keeps backward compatibility with legacy Telegram allowlist entries.
		if senderID == allowed ||
			idPart == allowed ||
			senderID == trimmed ||
			idPart == trimmed ||
			idPart == allowedID ||
			(allowedUser != "" && senderID == allowedUser) ||
			(userPart != "" && (userPart == allowed || userPart == trimmed || userPart == allowedUser)) {
			return true
		}
	}

	return false
}

func (c *BaseChannel) HandleMessage(
	peer bus.Peer,
	messageID, senderID, chatID, content string,
	media []string,
	metadata map[string]string,
) {
	if !c.IsAllowed(senderID) {
		return
	}

	msg := bus.InboundMessage{
		Channel:   c.name,
		SenderID:  senderID,
		ChatID:    chatID,
		Content:   content,
		Media:     media,
		Peer:      peer,
		MessageID: messageID,
		Metadata:  metadata,
	}

	c.bus.PublishInbound(msg)
}

func (c *BaseChannel) SetRunning(running bool) {
	c.running.Store(running)
}
