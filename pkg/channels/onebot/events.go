package onebot

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"jane/pkg/bus"
	"jane/pkg/channels"
	"jane/pkg/identity"
	"jane/pkg/logger"
)

func (c *OneBotChannel) listen() {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		logger.WarnC("onebot", "WebSocket connection is nil, listener exiting")
		return
	}

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				logger.ErrorCF("onebot", "WebSocket read error", map[string]any{
					"error": err.Error(),
				})
				c.mu.Lock()
				if c.conn == conn {
					c.conn.Close()
					c.conn = nil
				}
				c.mu.Unlock()
				return
			}

			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			var raw oneBotRawEvent
			if err := json.Unmarshal(message, &raw); err != nil {
				logger.WarnCF("onebot", "Failed to unmarshal raw event", map[string]any{
					"error":   err.Error(),
					"payload": string(message),
				})
				continue
			}

			logger.DebugCF("onebot", "WebSocket event", map[string]any{
				"length":    len(message),
				"post_type": raw.PostType,
				"sub_type":  raw.SubType,
			})

			if raw.Echo != "" {
				c.pendingMu.Lock()
				ch, ok := c.pending[raw.Echo]
				c.pendingMu.Unlock()

				if ok {
					select {
					case ch <- message:
					default:
					}
				} else {
					logger.DebugCF("onebot", "Received API response (no waiter)", map[string]any{
						"echo":   raw.Echo,
						"status": string(raw.Status),
					})
				}
				continue
			}

			if isAPIResponse(raw.Status) {
				logger.DebugCF("onebot", "Received API response without echo, skipping", map[string]any{
					"status": string(raw.Status),
				})
				continue
			}

			c.handleRawEvent(&raw)
		}
	}
}

func (c *OneBotChannel) handleRawEvent(raw *oneBotRawEvent) {
	switch raw.PostType {
	case "message":
		if userID, err := parseJSONInt64(raw.UserID); err == nil && userID > 0 {
			// Build minimal sender for allowlist check
			sender := bus.SenderInfo{
				Platform:    "onebot",
				PlatformID:  strconv.FormatInt(userID, 10),
				CanonicalID: identity.BuildCanonicalID("onebot", strconv.FormatInt(userID, 10)),
			}
			if !c.IsAllowedSender(sender) {
				logger.DebugCF("onebot", "Message rejected by allowlist", map[string]any{
					"user_id": userID,
				})
				return
			}
		}
		c.handleMessage(raw)

	case "message_sent":
		logger.DebugCF("onebot", "Bot sent message event", map[string]any{
			"message_type": raw.MessageType,
			"message_id":   parseJSONString(raw.MessageID),
		})

	case "meta_event":
		c.handleMetaEvent(raw)

	case "notice":
		c.handleNoticeEvent(raw)

	case "request":
		logger.DebugCF("onebot", "Request event received", map[string]any{
			"sub_type": raw.SubType,
		})

	case "":
		logger.DebugCF("onebot", "Event with empty post_type (possibly API response)", map[string]any{
			"echo":   raw.Echo,
			"status": raw.Status,
		})

	default:
		logger.DebugCF("onebot", "Unknown post_type", map[string]any{
			"post_type": raw.PostType,
		})
	}
}

func (c *OneBotChannel) handleMetaEvent(raw *oneBotRawEvent) {
	if raw.MetaEventType == "lifecycle" {
		logger.InfoCF("onebot", "Lifecycle event", map[string]any{"sub_type": raw.SubType})
	} else if raw.MetaEventType != "heartbeat" {
		logger.DebugCF("onebot", "Meta event: "+raw.MetaEventType, nil)
	}
}

func (c *OneBotChannel) handleNoticeEvent(raw *oneBotRawEvent) {
	fields := map[string]any{
		"notice_type": raw.NoticeType,
		"sub_type":    raw.SubType,
		"group_id":    parseJSONString(raw.GroupID),
		"user_id":     parseJSONString(raw.UserID),
		"message_id":  parseJSONString(raw.MessageID),
	}
	switch raw.NoticeType {
	case "group_recall", "group_increase", "group_decrease",
		"friend_add", "group_admin", "group_ban":
		logger.InfoCF("onebot", "Notice: "+raw.NoticeType, fields)
	default:
		logger.DebugCF("onebot", "Notice: "+raw.NoticeType, fields)
	}
}

func (c *OneBotChannel) handleMessage(raw *oneBotRawEvent) {
	// Parse fields from raw event
	userID, err := parseJSONInt64(raw.UserID)
	if err != nil {
		logger.WarnCF("onebot", "Failed to parse user_id", map[string]any{
			"error": err.Error(),
			"raw":   string(raw.UserID),
		})
		return
	}

	groupID, _ := parseJSONInt64(raw.GroupID)
	selfID, _ := parseJSONInt64(raw.SelfID)
	messageID := parseJSONString(raw.MessageID)

	if selfID == 0 {
		selfID = atomic.LoadInt64(&c.selfID)
	}

	// Compute scope for media store before parsing (parsing may download files)
	var chatIDForScope string
	switch raw.MessageType {
	case "group":
		chatIDForScope = "group:" + strconv.FormatInt(groupID, 10)
	default:
		chatIDForScope = "private:" + strconv.FormatInt(userID, 10)
	}
	scope := channels.BuildMediaScope("onebot", chatIDForScope, messageID)

	parsed := c.parseMessageSegments(raw.Message, selfID, c.GetMediaStore(), scope)
	isBotMentioned := parsed.IsBotMentioned

	content := raw.RawMessage
	if content == "" {
		content = parsed.Text
	} else if selfID > 0 {
		cqAt := fmt.Sprintf("[CQ:at,qq=%d]", selfID)
		if strings.Contains(content, cqAt) {
			isBotMentioned = true
			content = strings.ReplaceAll(content, cqAt, "")
			content = strings.TrimSpace(content)
		}
	}

	if parsed.Text != "" && content != parsed.Text && (len(parsed.Media) > 0 || parsed.ReplyTo != "") {
		content = parsed.Text
	}

	var sender oneBotSender
	if len(raw.Sender) > 0 {
		if err := json.Unmarshal(raw.Sender, &sender); err != nil {
			logger.WarnCF("onebot", "Failed to parse sender", map[string]any{
				"error":  err.Error(),
				"sender": string(raw.Sender),
			})
		}
	}

	if c.isDuplicate(messageID) {
		logger.DebugCF("onebot", "Duplicate message, skipping", map[string]any{
			"message_id": messageID,
		})
		return
	}

	if content == "" {
		logger.DebugCF("onebot", "Received empty message, ignoring", map[string]any{
			"message_id": messageID,
		})
		return
	}

	senderID := strconv.FormatInt(userID, 10)
	var chatID string

	var peer bus.Peer

	metadata := map[string]string{}

	if parsed.ReplyTo != "" {
		metadata["reply_to_message_id"] = parsed.ReplyTo
	}

	switch raw.MessageType {
	case "private":
		chatID = "private:" + senderID
		peer = bus.Peer{Kind: "direct", ID: senderID}

	case "group":
		groupIDStr := strconv.FormatInt(groupID, 10)
		chatID = "group:" + groupIDStr
		peer = bus.Peer{Kind: "group", ID: groupIDStr}
		metadata["group_id"] = groupIDStr

		senderUserID, _ := parseJSONInt64(sender.UserID)
		if senderUserID > 0 {
			metadata["sender_user_id"] = strconv.FormatInt(senderUserID, 10)
		}

		if sender.Card != "" {
			metadata["sender_name"] = sender.Card
		} else if sender.Nickname != "" {
			metadata["sender_name"] = sender.Nickname
		}

		respond, strippedContent := c.ShouldRespondInGroup(isBotMentioned, content)
		if !respond {
			logger.DebugCF("onebot", "Group message ignored (no trigger)", map[string]any{
				"sender":       senderID,
				"group":        groupIDStr,
				"is_mentioned": isBotMentioned,
				"content":      truncate(content, 100),
			})
			return
		}
		content = strippedContent

	default:
		logger.WarnCF("onebot", "Unknown message type, cannot route", map[string]any{
			"type":       raw.MessageType,
			"message_id": messageID,
			"user_id":    userID,
		})
		return
	}

	logger.InfoCF("onebot", "Received "+raw.MessageType+" message", map[string]any{
		"sender":      senderID,
		"chat_id":     chatID,
		"message_id":  messageID,
		"length":      len(content),
		"content":     truncate(content, 100),
		"media_count": len(parsed.Media),
	})

	if sender.Nickname != "" {
		metadata["nickname"] = sender.Nickname
	}

	c.lastMessageID.Store(chatID, messageID)

	senderInfo := bus.SenderInfo{
		Platform:    "onebot",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("onebot", senderID),
		DisplayName: sender.Nickname,
	}

	if !c.IsAllowedSender(senderInfo) {
		logger.DebugCF("onebot", "Message rejected by allowlist (senderInfo)", map[string]any{
			"sender": senderID,
		})
		return
	}

	c.HandleMessage(c.ctx, peer, messageID, senderID, chatID, content, parsed.Media, metadata, senderInfo)
}
