package vk

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/SevereCloud/vksdk/v3/api"
	"github.com/SevereCloud/vksdk/v3/api/params"
	"github.com/SevereCloud/vksdk/v3/events"
	"github.com/SevereCloud/vksdk/v3/longpoll-bot"
	"github.com/SevereCloud/vksdk/v3/object"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type VKChannel struct {
	*channels.BaseChannel
	vk          *api.VK
	lp          *longpoll.LongPoll
	channelName string
	bc          *config.Channel
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewVKChannel(channelName string, bc *config.Channel, bus *bus.MessageBus) (*VKChannel, error) {
	var vkCfg config.VKSettings
	if err := bc.Decode(&vkCfg); err != nil {
		return nil, err
	}

	vk := api.NewVK(vkCfg.Token.String())

	base := channels.NewBaseChannel(
		channelName,
		&vkCfg,
		bus,
		bc.AllowFrom,
		channels.WithMaxMessageLength(4000),
		channels.WithGroupTrigger(bc.GroupTrigger),
		channels.WithReasoningChannelID(bc.ReasoningChannelID),
	)

	return &VKChannel{
		BaseChannel: base,
		vk:          vk,
		channelName: channelName,
		bc:          bc,
	}, nil
}

func (c *VKChannel) getVKCfg() *config.VKSettings {
	var v config.VKSettings
	if err := c.bc.Decode(&v); err != nil {
		return nil
	}
	return &v
}

func (c *VKChannel) Start(ctx context.Context) error {
	logger.InfoC("vk", "Starting VK bot (Long Poll mode)...")

	c.ctx, c.cancel = context.WithCancel(ctx)

	groupID := c.getVKCfg().GroupID
	if groupID == 0 {
		c.cancel()
		return fmt.Errorf("group_id is required for VK bot")
	}

	lp, err := longpoll.NewLongPoll(c.vk, groupID)
	if err != nil {
		c.cancel()
		return fmt.Errorf("failed to create long poll: %w", err)
	}
	c.lp = lp

	lp.MessageNew(func(_ context.Context, obj events.MessageNewObject) {
		c.handleMessage(obj.Message)
	})

	c.SetRunning(true)

	logger.InfoCF("vk", "VK bot connected", map[string]any{
		"group_id": groupID,
	})

	go func() {
		if err := lp.Run(); err != nil {
			logger.ErrorCF("vk", "Long poll failed", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

func (c *VKChannel) Stop(ctx context.Context) error {
	logger.InfoC("vk", "Stopping VK bot...")
	c.SetRunning(false)

	if c.lp != nil {
		c.lp.Shutdown()
	}

	if c.cancel != nil {
		c.cancel()
	}

	return nil
}

func (c *VKChannel) handleMessage(msg object.MessagesMessage) {
	if msg.Action.Type != "" {
		return
	}

	if bool(msg.Out) {
		return
	}

	peerID := msg.PeerID
	chatID := strconv.Itoa(peerID)

	fromID := msg.FromID
	userID := strconv.Itoa(fromID)

	platformID := userID
	sender := bus.SenderInfo{
		Platform:    "vk",
		PlatformID:  platformID,
		CanonicalID: identity.BuildCanonicalID("vk", platformID),
		DisplayName: c.getUserName(fromID),
	}

	if !c.IsAllowedSender(sender) {
		logger.DebugCF("vk", "Message from unauthorized user", map[string]any{
			"peer_id": peerID,
		})
		return
	}

	text := msg.Text
	if text == "" && len(msg.Attachments) > 0 {
		text = c.processAttachments(msg.Attachments)
	}

	if text == "" {
		return
	}

	groupTrigger := c.bc.GroupTrigger
	isGroupChat := peerID != fromID

	if isGroupChat {
		isMentioned := c.isMentioned(msg)
		if isMentioned {
			text = c.stripBotMention(text)
		}
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, text)
		if !respond {
			return
		}
		text = cleaned
		_ = groupTrigger
	}

	peerKind := "direct"
	peerIDStr := userID
	if isGroupChat {
		peerKind = "group"
		peerIDStr = chatID
	}

	peer := bus.Peer{Kind: peerKind, ID: peerIDStr}
	messageID := strconv.Itoa(msg.ConversationMessageID)

	metadata := map[string]string{
		"user_id":  userID,
		"is_group": fmt.Sprintf("%t", isGroupChat),
	}

	c.HandleMessage(c.ctx,
		peer,
		messageID,
		userID,
		chatID,
		text,
		nil,
		metadata,
		sender,
	)
}

func (c *VKChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	peerID, err := strconv.Atoi(msg.ChatID)
	if err != nil {
		return nil, fmt.Errorf("invalid chat ID %s: %w", msg.ChatID, channels.ErrSendFailed)
	}

	if msg.Content == "" {
		return nil, nil
	}

	var messageIDs []string
	chunks := channels.SplitMessage(msg.Content, 4000)

	for _, chunk := range chunks {
		if chunk == "" {
			continue
		}

		b := params.NewMessagesSendBuilder()
		b.Message(chunk)
		b.RandomID(0)
		b.PeerID(peerID)

		if msg.ReplyToMessageID != "" {
			if replyID, err := strconv.Atoi(msg.ReplyToMessageID); err == nil {
				b.ReplyTo(replyID)
			}
		}

		resp, err := c.vk.MessagesSend(b.Params)
		if err != nil {
			logger.ErrorCF("vk", "Failed to send message", map[string]any{
				"error":   err.Error(),
				"peer_id": peerID,
			})
			return messageIDs, fmt.Errorf("failed to send message: %w", err)
		}

		messageIDs = append(messageIDs, strconv.Itoa(resp))
	}

	return messageIDs, nil
}

func (c *VKChannel) isMentioned(msg object.MessagesMessage) bool {
	return false
}

func (c *VKChannel) stripBotMention(text string) string {
	return strings.TrimSpace(text)
}

func (c *VKChannel) getUserName(userID int) string {
	users, err := c.vk.UsersGet(api.Params{
		"user_ids": userID,
	})
	if err != nil || len(users) == 0 {
		return strconv.Itoa(userID)
	}

	user := users[0]
	return fmt.Sprintf("%s %s", user.FirstName, user.LastName)
}

func (c *VKChannel) processAttachments(attachments []object.MessagesMessageAttachment) string {
	var parts []string

	for _, att := range attachments {
		switch att.Type {
		case "photo":
			parts = append(parts, "[photo]")
		case "video":
			parts = append(parts, "[video]")
		case "audio":
			parts = append(parts, "[audio]")
		case "doc":
			if att.Doc.Title != "" {
				parts = append(parts, fmt.Sprintf("[document: %s]", att.Doc.Title))
			} else {
				parts = append(parts, "[document]")
			}
		case "audio_message":
			parts = append(parts, "[voice]")
		case "sticker":
			parts = append(parts, "[sticker]")
		}
	}

	return strings.Join(parts, " ")
}

func (c *VKChannel) VoiceCapabilities() channels.VoiceCapabilities {
	return channels.VoiceCapabilities{ASR: true, TTS: true}
}
