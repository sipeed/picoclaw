package channels

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
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

// WithGroupTrigger sets the group trigger configuration for a channel.
func WithGroupTrigger(gt config.GroupTriggerConfig) BaseChannelOption {
	return func(c *BaseChannel) { c.groupTrigger = gt }
}

// MessageLengthProvider is an opt-in interface that channels implement
// to advertise their maximum message length. The Manager uses this via
// type assertion to decide whether to split outbound messages.
type MessageLengthProvider interface {
	MaxMessageLength() int
}

type BaseChannel struct {
	config              any
	bus                 *bus.MessageBus
	running             atomic.Bool
	name                string
	allowList           []string
	maxMessageLength    int
	groupTrigger        config.GroupTriggerConfig
	mediaStore          media.MediaStore
	placeholderRecorder PlaceholderRecorder
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

// ShouldRespondInGroup determines whether the bot should respond in a group chat.
// Each channel is responsible for:
//  1. Detecting isMentioned (platform-specific)
//  2. Stripping bot mention from content (platform-specific)
//  3. Calling this method to get the group response decision
//
// Logic:
//   - If isMentioned → always respond
//   - If mention_only configured and not mentioned → ignore
//   - If prefixes configured → respond if content starts with any prefix (strip it)
//   - If prefixes configured but no match and not mentioned → ignore
//   - Otherwise (no group_trigger configured) → respond to all (permissive default)
func (c *BaseChannel) ShouldRespondInGroup(isMentioned bool, content string) (bool, string) {
	gt := c.groupTrigger

	// Mentioned → always respond
	if isMentioned {
		return true, strings.TrimSpace(content)
	}

	// mention_only → require mention
	if gt.MentionOnly {
		return false, content
	}

	// Prefix matching
	if len(gt.Prefixes) > 0 {
		for _, prefix := range gt.Prefixes {
			if prefix != "" && strings.HasPrefix(content, prefix) {
				return true, strings.TrimSpace(strings.TrimPrefix(content, prefix))
			}
		}
		// Prefixes configured but none matched and not mentioned → ignore
		return false, content
	}

	// No group_trigger configured → permissive (respond to all)
	return true, strings.TrimSpace(content)
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

	scope := BuildMediaScope(c.name, chatID, messageID)

	msg := bus.InboundMessage{
		Channel:    c.name,
		SenderID:   senderID,
		ChatID:     chatID,
		Content:    content,
		Media:      media,
		Peer:       peer,
		MessageID:  messageID,
		MediaScope: scope,
		Metadata:   metadata,
	}

	c.bus.PublishInbound(context.TODO(), msg)
}

func (c *BaseChannel) SetRunning(running bool) {
	c.running.Store(running)
}

// SetMediaStore injects a MediaStore into the channel.
func (c *BaseChannel) SetMediaStore(s media.MediaStore) { c.mediaStore = s }

// GetMediaStore returns the injected MediaStore (may be nil).
func (c *BaseChannel) GetMediaStore() media.MediaStore { return c.mediaStore }

// SetPlaceholderRecorder injects a PlaceholderRecorder into the channel.
func (c *BaseChannel) SetPlaceholderRecorder(r PlaceholderRecorder) {
	c.placeholderRecorder = r
}

// GetPlaceholderRecorder returns the injected PlaceholderRecorder (may be nil).
func (c *BaseChannel) GetPlaceholderRecorder() PlaceholderRecorder {
	return c.placeholderRecorder
}

// BuildMediaScope constructs a scope key for media lifecycle tracking.
func BuildMediaScope(channel, chatID, messageID string) string {
	id := messageID
	if id == "" {
		id = uuid.New().String()
	}
	return channel + ":" + chatID + ":" + id
}
