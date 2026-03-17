package onebot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"jane/pkg/logger"
)

func (c *OneBotChannel) setMsgEmojiLike(messageID string, emojiID int, set bool) {
	go func() {
		_, err := c.sendAPIRequest("set_msg_emoji_like", map[string]any{
			"message_id": messageID,
			"emoji_id":   emojiID,
			"set":        set,
		}, 5*time.Second)
		if err != nil {
			logger.DebugCF("onebot", "Failed to set emoji like", map[string]any{
				"message_id": messageID,
				"error":      err.Error(),
			})
		}
	}()
}

// ReactToMessage implements channels.ReactionCapable.
// It adds an emoji reaction (ID 289) to group messages and returns an undo function.
// Private messages return a no-op since reactions are only meaningful in groups.
func (c *OneBotChannel) ReactToMessage(ctx context.Context, chatID, messageID string) (func(), error) {
	// Only react in group chats
	if !strings.HasPrefix(chatID, "group:") {
		return func() {}, nil
	}

	c.setMsgEmojiLike(messageID, 289, true)

	return func() {
		c.setMsgEmojiLike(messageID, 289, false)
	}, nil
}

func (c *OneBotChannel) fetchSelfID() {
	resp, err := c.sendAPIRequest("get_login_info", nil, 5*time.Second)
	if err != nil {
		logger.WarnCF("onebot", "Failed to get_login_info", map[string]any{
			"error": err.Error(),
		})
		return
	}

	type loginInfo struct {
		UserID   json.RawMessage `json:"user_id"`
		Nickname string          `json:"nickname"`
	}
	for _, extract := range []func() (*loginInfo, error){
		func() (*loginInfo, error) {
			var w struct {
				Data loginInfo `json:"data"`
			}
			err := json.Unmarshal(resp, &w)
			return &w.Data, err
		},
		func() (*loginInfo, error) {
			var f loginInfo
			err := json.Unmarshal(resp, &f)
			return &f, err
		},
	} {
		info, err := extract()
		if err != nil || len(info.UserID) == 0 {
			continue
		}
		if uid, err := parseJSONInt64(info.UserID); err == nil && uid > 0 {
			atomic.StoreInt64(&c.selfID, uid)
			logger.InfoCF("onebot", "Bot self ID retrieved", map[string]any{
				"self_id":  uid,
				"nickname": info.Nickname,
			})
			return
		}
	}

	logger.WarnCF("onebot", "Could not parse self ID from get_login_info response", map[string]any{
		"response": string(resp),
	})
}

func (c *OneBotChannel) sendAPIRequest(action string, params any, timeout time.Duration) (json.RawMessage, error) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return nil, fmt.Errorf("WebSocket not connected")
	}

	echo := fmt.Sprintf("api_%d_%d", time.Now().UnixNano(), atomic.AddInt64(&c.echoCounter, 1))

	ch := make(chan json.RawMessage, 1)
	c.pendingMu.Lock()
	c.pending[echo] = ch
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, echo)
		c.pendingMu.Unlock()
	}()

	req := oneBotAPIRequest{
		Action: action,
		Params: params,
		Echo:   echo,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal API request: %w", err)
	}

	c.writeMu.Lock()
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err = conn.WriteMessage(websocket.TextMessage, data)
	_ = conn.SetWriteDeadline(time.Time{})
	c.writeMu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to write API request: %w", err)
	}

	select {
	case resp := <-ch:
		if resp == nil {
			return nil, fmt.Errorf("API request %s: channel stopped", action)
		}
		return resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("API request %s timed out after %v", action, timeout)
	case <-c.ctx.Done():
		return nil, fmt.Errorf("context canceled")
	}
}
