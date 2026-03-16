package matrix

import (
	"context"
	"fmt"
	"strings"
	"time"

	"maunium.net/go/mautrix/event"

	"jane/pkg/bus"
	"jane/pkg/channels"
	"jane/pkg/identity"
	"jane/pkg/logger"
)

func (c *MatrixChannel) handleMemberEvent(ctx context.Context, evt *event.Event) {
	if !c.config.JoinOnInvite {
		return
	}
	if evt == nil {
		return
	}

	member := evt.Content.AsMember()
	if member.Membership != event.MembershipInvite {
		return
	}
	if evt.GetStateKey() != c.client.UserID.String() {
		return
	}

	_, err := c.client.JoinRoomByID(c.baseContext(), evt.RoomID)
	if err != nil {
		logger.WarnCF("matrix", "Failed to auto-join invited room", map[string]any{
			"room_id": evt.RoomID.String(),
			"error":   err.Error(),
		})
		return
	}

	logger.InfoCF("matrix", "Joined room after invite", map[string]any{
		"room_id": evt.RoomID.String(),
	})
}

func (c *MatrixChannel) handleMessageEvent(ctx context.Context, evt *event.Event) {
	if evt == nil {
		return
	}

	// Ignore our own messages.
	if evt.Sender == c.client.UserID {
		return
	}

	// Ignore historical events on first sync.
	if time.UnixMilli(evt.Timestamp).Before(c.startTime) {
		return
	}

	msgEvt := evt.Content.AsMessage()
	if msgEvt == nil {
		return
	}

	// Ignore edits.
	if msgEvt.RelatesTo != nil && msgEvt.RelatesTo.GetReplaceID() != "" {
		return
	}

	roomID := evt.RoomID.String()
	scope := channels.BuildMediaScope("matrix", roomID, evt.ID.String())

	content, mediaPaths, ok := c.extractInboundContent(ctx, msgEvt, scope)
	if !ok {
		return
	}
	content = strings.TrimSpace(content)
	if content == "" && len(mediaPaths) == 0 {
		return
	}

	senderID := evt.Sender.String()
	sender := bus.SenderInfo{
		Platform:    "matrix",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("matrix", senderID),
		Username:    senderID,
		DisplayName: senderID,
	}

	if !c.IsAllowedSender(sender) {
		logger.DebugCF("matrix", "Message rejected by allowlist", map[string]any{
			"sender_id": senderID,
		})
		return
	}

	isGroup := c.isGroupRoom(ctx, evt.RoomID)
	if isGroup {
		isMentioned := c.isBotMentioned(msgEvt)
		if isMentioned {
			content = c.stripSelfMention(content)
		}
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			logger.DebugCF("matrix", "Ignoring group message by trigger rules", map[string]any{
				"room_id":      roomID,
				"is_mentioned": isMentioned,
				"mention_only": c.config.GroupTrigger.MentionOnly,
				"prefixes":     c.config.GroupTrigger.Prefixes,
			})
			return
		}
		content = cleaned
	} else {
		content = c.stripSelfMention(content)
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return
	}

	peerKind := "direct"
	peerID := senderID
	if isGroup {
		peerKind = "group"
		peerID = roomID
	}

	metadata := map[string]string{
		"room_id":    roomID,
		"timestamp":  fmt.Sprintf("%d", evt.Timestamp),
		"is_group":   fmt.Sprintf("%t", isGroup),
		"sender_raw": senderID,
	}

	logger.DebugCF("matrix", "Received message", map[string]any{
		"sender_id": senderID,
		"room_id":   roomID,
		"is_group":  isGroup,
	})

	if replyTo := msgEvt.GetRelatesTo().GetReplyTo(); replyTo != "" {
		metadata["reply_to_msg_id"] = replyTo.String()
	}

	c.HandleMessage(
		c.baseContext(),
		bus.Peer{Kind: peerKind, ID: peerID},
		evt.ID.String(),
		senderID,
		roomID,
		content,
		mediaPaths,
		metadata,
		sender,
	)
}

func (c *MatrixChannel) extractInboundContent(
	ctx context.Context,
	msgEvt *event.MessageEventContent,
	scope string,
) (string, []string, bool) {
	switch msgEvt.MsgType {
	case event.MsgText, event.MsgNotice:
		return msgEvt.Body, nil, true
	case event.MsgImage, event.MsgAudio, event.MsgVideo, event.MsgFile:
		return c.extractInboundMedia(ctx, msgEvt, scope)
	default:
		logger.DebugCF("matrix", "Ignoring unsupported matrix msgtype", map[string]any{
			"msgtype": msgEvt.MsgType,
		})
		return "", nil, false
	}
}
