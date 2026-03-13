// PicoClaw - Ultra-lightweight personal AI agent
// DingTalk robot API for proactive messaging

package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// sendGroupMessage sends a message to a group chat via robot API
func (c *DingTalkChannel) sendGroupMessage(ctx context.Context, accessToken, openConversationId, content string) error {
	url := fmt.Sprintf("%s/v1.0/robot/groupMessages/send", dingtalkAPIBase)

	// Build msgParam for markdown message
	msgParam, _ := json.Marshal(map[string]string{
		"title": "PicoClaw",
		"text":  content,
	})

	body := map[string]any{
		"openConversationId": openConversationId,
		"robotCode":          c.clientID,
		"msgKey":             "sampleMarkdown",
		"msgParam":           string(msgParam),
	}

	return c.sendRobotAPIRequest(ctx, accessToken, url, body, "group")
}

// sendOToMessage sends a message to a user via robot API (one-to-one)
func (c *DingTalkChannel) sendOToMessage(ctx context.Context, accessToken, userID, content string) error {
	url := fmt.Sprintf("%s/v1.0/robot/oToMessages/batchSend", dingtalkAPIBase)

	// Build msgParam for markdown message
	msgParam, _ := json.Marshal(map[string]string{
		"title": "PicoClaw",
		"text":  content,
	})

	body := map[string]any{
		"robotCode": c.clientID,
		"userIds":   []string{userID},
		"msgKey":    "sampleMarkdown",
		"msgParam":  string(msgParam),
	}

	return c.sendRobotAPIRequest(ctx, accessToken, url, body, "direct")
}

// sendRobotAPIRequest sends a request to DingTalk robot API
func (c *DingTalkChannel) sendRobotAPIRequest(
	ctx context.Context,
	accessToken, url string,
	body any,
	msgType string,
) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Acs-Dingtalk-Access-Token", accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return channels.ClassifyNetError(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Parse error response for better error message
		var errResp struct {
			Code string `json:"code"`
			Msg  string `json:"message"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Msg != "" {
			return fmt.Errorf("dingtalk robot API error: %s (%s)", errResp.Msg, errResp.Code)
		}
		return channels.ClassifySendError(resp.StatusCode,
			fmt.Errorf("dingtalk robot API error (status %d): %s", resp.StatusCode, string(respBody)))
	}

	logger.InfoCF("dingtalk", "Robot API message sent successfully", map[string]any{
		"type":   msgType,
		"status": resp.StatusCode,
	})

	return nil
}

// sendViaRobotAPI sends a message via the robot API
// For proactive messaging: chatID is used directly as staffId (direct) or openConversationId (group)
// Group IDs start with "cid", everything else is treated as direct message staffId
func (c *DingTalkChannel) sendViaRobotAPI(ctx context.Context, chatID, content string) error {
	accessToken, err := c.ensureValidToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// First, check if we have session info (from prior conversation)
	session := c.getSession(chatID)
	if session != nil {
		if session.ConversationType == "1" {
			// Direct message with session info
			if session.SenderStaffId != "" {
				return c.sendOToMessage(ctx, accessToken, session.SenderStaffId, content)
			}
		} else {
			// Group message with session info
			if session.OpenConversationId != "" {
				return c.sendGroupMessage(ctx, accessToken, session.OpenConversationId, content)
			}
		}
	}

	// No session info - detect type based on chatID format
	// Group IDs start with "cid" (openConversationId format)
	isGroup := len(chatID) > 3 && chatID[:3] == "cid"

	if isGroup {
		logger.DebugCF("dingtalk", "Sending proactive group message", map[string]any{
			"open_conversation_id": chatID,
		})
		return c.sendGroupMessage(ctx, accessToken, chatID, content)
	}

	// Direct message - chatID is the staffId
	logger.DebugCF("dingtalk", "Sending proactive direct message", map[string]any{
		"staff_id": chatID,
	})
	return c.sendOToMessage(ctx, accessToken, chatID, content)
}
