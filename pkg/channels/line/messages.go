package line

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"jane/pkg/bus"
	"jane/pkg/channels"
	"jane/pkg/logger"
)

// Send sends a message to LINE. It first tries the Reply API (free)
// using a cached reply token, then falls back to the Push API.
func (c *LINEChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	// Load and consume quote token for this chat
	var quoteToken string
	if qt, ok := c.quoteTokens.LoadAndDelete(msg.ChatID); ok {
		quoteToken = qt.(string)
	}

	// Try reply token first (free, valid for ~25 seconds)
	if entry, ok := c.replyTokens.LoadAndDelete(msg.ChatID); ok {
		tokenEntry := entry.(replyTokenEntry)
		if time.Since(tokenEntry.timestamp) < lineReplyTokenMaxAge {
			if err := c.sendReply(ctx, tokenEntry.token, msg.Content, quoteToken); err == nil {
				logger.DebugCF("line", "Message sent via Reply API", map[string]any{
					"chat_id": msg.ChatID,
					"quoted":  quoteToken != "",
				})
				return nil
			}
			logger.DebugC("line", "Reply API failed, falling back to Push API")
		}
	}

	// Fall back to Push API
	return c.sendPush(ctx, msg.ChatID, msg.Content, quoteToken)
}

// buildTextMessage creates a text message object, optionally with quoteToken.
func buildTextMessage(content, quoteToken string) map[string]string {
	msg := map[string]string{
		"type": "text",
		"text": content,
	}
	if quoteToken != "" {
		msg["quoteToken"] = quoteToken
	}
	return msg
}

// sendReply sends a message using the LINE Reply API.
func (c *LINEChannel) sendReply(ctx context.Context, replyToken, content, quoteToken string) error {
	payload := map[string]any{
		"replyToken": replyToken,
		"messages":   []map[string]string{buildTextMessage(content, quoteToken)},
	}

	return c.callAPI(ctx, lineReplyEndpoint, payload)
}

// sendPush sends a message using the LINE Push API.
func (c *LINEChannel) sendPush(ctx context.Context, to, content, quoteToken string) error {
	payload := map[string]any{
		"to":       to,
		"messages": []map[string]string{buildTextMessage(content, quoteToken)},
	}

	return c.callAPI(ctx, linePushEndpoint, payload)
}

// callAPI makes an authenticated POST request to the LINE API.
func (c *LINEChannel) callAPI(ctx context.Context, endpoint string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.ChannelAccessToken)

	resp, err := c.apiClient.Do(req)
	if err != nil {
		return channels.ClassifyNetError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return channels.ClassifySendError(resp.StatusCode, fmt.Errorf("reading LINE API error response: %w", err))
		}
		return channels.ClassifySendError(resp.StatusCode, fmt.Errorf("LINE API error: %s", string(respBody)))
	}

	return nil
}
