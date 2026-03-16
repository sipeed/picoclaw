package onebot

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"jane/pkg/bus"
	"jane/pkg/channels"
	"jane/pkg/logger"
)

func (c *OneBotChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	// Check ctx before entering write path
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("OneBot WebSocket not connected")
	}

	action, params, err := c.buildSendRequest(msg)
	if err != nil {
		return err
	}

	echo := fmt.Sprintf("send_%d", atomic.AddInt64(&c.echoCounter, 1))

	req := oneBotAPIRequest{
		Action: action,
		Params: params,
		Echo:   echo,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal OneBot request: %w", err)
	}

	c.writeMu.Lock()
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err = conn.WriteMessage(websocket.TextMessage, data)
	_ = conn.SetWriteDeadline(time.Time{})
	c.writeMu.Unlock()

	if err != nil {
		logger.ErrorCF("onebot", "Failed to send message", map[string]any{
			"error": err.Error(),
		})
		return fmt.Errorf("onebot send: %w", channels.ErrTemporary)
	}

	return nil
}

// SendMedia implements the channels.MediaSender interface.
func (c *OneBotChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("OneBot WebSocket not connected")
	}

	store := c.GetMediaStore()
	if store == nil {
		return fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	// Build media segments
	var segments []oneBotMessageSegment
	for _, part := range msg.Parts {
		localPath, err := store.Resolve(part.Ref)
		if err != nil {
			logger.ErrorCF("onebot", "Failed to resolve media ref", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			continue
		}

		var segType string
		switch part.Type {
		case "image":
			segType = "image"
		case "video":
			segType = "video"
		case "audio":
			segType = "record"
		default:
			segType = "file"
		}

		segments = append(segments, oneBotMessageSegment{
			Type: segType,
			Data: map[string]any{"file": "file://" + localPath},
		})

		if part.Caption != "" {
			segments = append(segments, oneBotMessageSegment{
				Type: "text",
				Data: map[string]any{"text": part.Caption},
			})
		}
	}

	if len(segments) == 0 {
		return nil
	}

	chatID := msg.ChatID
	var action, idKey string
	var rawID string
	if rest, ok := strings.CutPrefix(chatID, "group:"); ok {
		action, idKey, rawID = "send_group_msg", "group_id", rest
	} else if rest, ok := strings.CutPrefix(chatID, "private:"); ok {
		action, idKey, rawID = "send_private_msg", "user_id", rest
	} else {
		action, idKey, rawID = "send_private_msg", "user_id", chatID
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid %s in chatID: %s: %w", idKey, chatID, channels.ErrSendFailed)
	}

	echo := fmt.Sprintf("send_%d", atomic.AddInt64(&c.echoCounter, 1))

	req := oneBotAPIRequest{
		Action: action,
		Params: map[string]any{idKey: id, "message": segments},
		Echo:   echo,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal OneBot request: %w", err)
	}

	c.writeMu.Lock()
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err = conn.WriteMessage(websocket.TextMessage, data)
	_ = conn.SetWriteDeadline(time.Time{})
	c.writeMu.Unlock()

	if err != nil {
		logger.ErrorCF("onebot", "Failed to send media message", map[string]any{
			"error": err.Error(),
		})
		return fmt.Errorf("onebot send media: %w", channels.ErrTemporary)
	}

	return nil
}

func (c *OneBotChannel) buildMessageSegments(chatID, content string) []oneBotMessageSegment {
	var segments []oneBotMessageSegment

	if lastMsgID, ok := c.lastMessageID.Load(chatID); ok {
		if msgID, ok := lastMsgID.(string); ok && msgID != "" {
			segments = append(segments, oneBotMessageSegment{
				Type: "reply",
				Data: map[string]any{"id": msgID},
			})
		}
	}

	segments = append(segments, oneBotMessageSegment{
		Type: "text",
		Data: map[string]any{"text": content},
	})

	return segments
}

func (c *OneBotChannel) buildSendRequest(msg bus.OutboundMessage) (string, any, error) {
	chatID := msg.ChatID
	segments := c.buildMessageSegments(chatID, msg.Content)

	var action, idKey string
	var rawID string
	if rest, ok := strings.CutPrefix(chatID, "group:"); ok {
		action, idKey, rawID = "send_group_msg", "group_id", rest
	} else if rest, ok := strings.CutPrefix(chatID, "private:"); ok {
		action, idKey, rawID = "send_private_msg", "user_id", rest
	} else {
		action, idKey, rawID = "send_private_msg", "user_id", chatID
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return "", nil, fmt.Errorf("invalid %s in chatID: %s", idKey, chatID)
	}
	return action, map[string]any{idKey: id, "message": segments}, nil
}
