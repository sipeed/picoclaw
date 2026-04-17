package channels

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
)

var (
	uniqueIDCounter uint64
	uniqueIDPrefix  string
)

func init() {
	// One-time read from crypto/rand for a unique prefix (single syscall).
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// fallback to time-based prefix
		binary.BigEndian.PutUint64(b[:], uint64(time.Now().UnixNano()))
	}
	uniqueIDPrefix = hex.EncodeToString(b[:])
}

// audioAnnotationRe matches audio/voice annotations injected by channels (e.g. [voice], [audio: file.ogg]).
var audioAnnotationRe = regexp.MustCompile(`\[(voice|audio)(?::[^\]]*)?\]`)

// uniqueID generates a process-unique ID using a random prefix and an atomic counter.
// This ID is intended for internal correlation (e.g. media scope keys) and is NOT
// cryptographically secure — it must not be used in contexts where unpredictability matters.
func uniqueID() string {
	n := atomic.AddUint64(&uniqueIDCounter, 1)
	return uniqueIDPrefix + strconv.FormatUint(n, 16)
}

type Channel interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error)
	IsRunning() bool
	IsAllowed(senderID string) bool
	IsAllowedSender(sender bus.SenderInfo) bool
	ReasoningChannelID() string
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

// WithReasoningChannelID sets the reasoning channel ID where thoughts should be sent.
func WithReasoningChannelID(id string) BaseChannelOption {
	return func(c *BaseChannel) { c.reasoningChannelID = id }
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
	owner               Channel // the concrete channel that embeds this BaseChannel
	reasoningChannelID  string
}

func NewBaseChannel(
	name string,
	config any,
	bus *bus.MessageBus,
	allowList []string,
	opts ...BaseChannelOption,
) *BaseChannel {
	isEmpty := true
	for _, s := range allowList {
		if s != "" {
			isEmpty = false
			break
		}
	}
	if isEmpty {
		allowList = []string{}
	}
	bc := &BaseChannel{
		config:    config,
		bus:       bus,
		name:      name,
		allowList: allowList,
	}
	for _, opt := range opts {
		opt(bc)
	}

	// Security Audit: Check for open-by-default (unsecured) channels.
	// PicoClaw aims to be secure-by-default. If allow_from is empty, the bot
	// currently defaults to accepting messages from ANYONE. To explicitly
	// acknowledge and permit this (e.g. for a public bot), use ["*"].
	if len(bc.allowList) == 0 {
		logger.WarnCF("channels", "SECURITY: Channel allows EVERYONE (allow_from is empty)", map[string]any{
			"channel": bc.name,
			"hint":    "Set allow_from to your ID, or use '*' to explicitly acknowledge open access.",
		})
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

// SetName updates the channel name. Used by the manager after channel creation
// to ensure the name matches the config key (which may differ from the type).
func (c *BaseChannel) SetName(name string) {
	c.name = name
}

func (c *BaseChannel) ReasoningChannelID() string {
	return c.reasoningChannelID
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
		if allowed == "*" {
			return true
		}
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

// IsAllowedSender checks whether a structured SenderInfo is permitted by the allow-list.
// It delegates to identity.MatchAllowed for each entry, providing unified matching
// across all legacy formats and the new canonical "platform:id" format.
func (c *BaseChannel) IsAllowedSender(sender bus.SenderInfo) bool {
	if len(c.allowList) == 0 {
		return true
	}

	for _, allowed := range c.allowList {
		if allowed == "*" || identity.MatchAllowed(sender, allowed) {
			return true
		}
	}

	return false
}

func (c *BaseChannel) HandleMessageWithContext(
	ctx context.Context,
	deliveryChatID, content string,
	media []string,
	inboundCtx bus.InboundContext,
	senderOpts ...bus.SenderInfo,
) {
	// Use SenderInfo-based allow check when available, else fall back to string
	var sender bus.SenderInfo
	if len(senderOpts) > 0 {
		sender = senderOpts[0]
	}
	senderID := strings.TrimSpace(inboundCtx.SenderID)
	if sender.CanonicalID != "" || sender.PlatformID != "" {
		if !c.IsAllowedSender(sender) {
			return
		}
	} else {
		if !c.IsAllowed(senderID) {
			return
		}
	}

	// Set SenderID to canonical if available, otherwise keep the raw senderID
	resolvedSenderID := senderID
	if sender.CanonicalID != "" {
		resolvedSenderID = sender.CanonicalID
	}

	if resolvedSenderID == "" {
		resolvedSenderID = senderID
	}

	inboundCtx.Channel = c.name
	if inboundCtx.ChatID == "" {
		inboundCtx.ChatID = deliveryChatID
	}
	if inboundCtx.SenderID == "" {
		inboundCtx.SenderID = resolvedSenderID
	}

	scope := BuildMediaScope(c.name, deliveryChatID, inboundCtx.MessageID)

	msg := bus.InboundMessage{
		Context:    inboundCtx,
		Sender:     sender,
		Content:    content,
		Media:      media,
		MediaScope: scope,
	}
	msg = bus.NormalizeInboundMessage(msg)

	// Auto-trigger typing indicator, message reaction, and placeholder before publishing.
	// Each capability is independent — all three may fire for the same message.
	// Note: even when streaming is available, we still show typing + placeholder on inbound.
	// If streaming actually activates, preSend will skip the placeholder edit (streamActive map)
	// and the typing stop will still be called. This avoids the problem of compile-time interface
	// checks incorrectly skipping indicators when streaming may not work at runtime.
	if c.owner != nil && c.placeholderRecorder != nil {
		// Typing
		if tc, ok := c.owner.(TypingCapable); ok {
			if stop, err := tc.StartTyping(ctx, deliveryChatID); err == nil {
				c.placeholderRecorder.RecordTypingStop(c.name, deliveryChatID, stop)
			}
		}
		// Reaction
		if rc, ok := c.owner.(ReactionCapable); ok && msg.MessageID != "" {
			if undo, err := rc.ReactToMessage(ctx, deliveryChatID, msg.MessageID); err == nil {
				c.placeholderRecorder.RecordReactionUndo(c.name, deliveryChatID, undo)
			}
		}
		// Placeholder — independent pipeline.
		// Skip when the message contains audio: the agent will send the
		// placeholder after transcription completes, so the user sees
		// "Thinking…" only once the voice has been processed.
		if !audioAnnotationRe.MatchString(content) {
			if pc, ok := c.owner.(PlaceholderCapable); ok {
				if phID, err := pc.SendPlaceholder(ctx, deliveryChatID); err == nil && phID != "" {
					c.placeholderRecorder.RecordPlaceholder(c.name, deliveryChatID, phID)
				}
			}
		}
	}

	if err := c.bus.PublishInbound(ctx, msg); err != nil {
		logger.ErrorCF("channels", "Failed to publish inbound message", map[string]any{
			"channel": c.name,
			"chat_id": deliveryChatID,
			"error":   err.Error(),
		})
	}
}

// HandleInboundContext publishes a normalized inbound message using only the
// structured context.
func (c *BaseChannel) HandleInboundContext(
	ctx context.Context,
	deliveryChatID, content string,
	media []string,
	inboundCtx bus.InboundContext,
	senderOpts ...bus.SenderInfo,
) {
	c.HandleMessageWithContext(ctx, deliveryChatID, content, media, inboundCtx, senderOpts...)
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

// SetOwner injects the concrete channel that embeds this BaseChannel.
// This allows HandleMessage to auto-trigger TypingCapable / ReactionCapable / PlaceholderCapable.
func (c *BaseChannel) SetOwner(ch Channel) {
	c.owner = ch
}

// BuildMediaScope constructs a scope key for media lifecycle tracking.
func BuildMediaScope(channel, chatID, messageID string) string {
	id := messageID
	if id == "" {
		id = uniqueID()
	}
	return channel + ":" + chatID + ":" + id
}
