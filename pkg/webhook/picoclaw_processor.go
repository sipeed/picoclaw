package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// PicoClawProcessor creates a processor that uses PicoClaw's AI agent
func PicoClawProcessor(wsURL, token string) ProcessorFunc {
	return func(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
		// Extract prompt from payload
		prompt, ok := payload["prompt"]
		if !ok {
			// If no prompt field, use the entire payload as a string
			promptBytes, err := json.Marshal(payload)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal payload: %w", err)
			}
			prompt = string(promptBytes)
		}

		promptStr, ok := prompt.(string)
		if !ok {
			return nil, fmt.Errorf("prompt must be a string")
		}

		// Call PicoClaw AI via WebSocket
		response, err := callPicoClawAI(ctx, wsURL, token, promptStr)
		if err != nil {
			return nil, fmt.Errorf("AI processing failed: %w", err)
		}

		// Return result in expected format
		return map[string]interface{}{
			"data":  response,
			"error": nil,
		}, nil
	}
}

// callPicoClawAI sends a message to PicoClaw via WebSocket and waits for response
func callPicoClawAI(ctx context.Context, wsURL, token, prompt string) (string, error) {
	// Set up WebSocket connection with timeout
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// Add token to Authorization header (Bearer authentication)
	headers := map[string][]string{
		"Authorization": {"Bearer " + token},
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		return "", fmt.Errorf("failed to connect to PicoClaw: %w", err)
	}
	defer conn.Close()

	// Set read deadline
	deadline := time.Now().Add(2 * time.Minute)
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	}
	conn.SetReadDeadline(deadline)

	// Send message using Pico Protocol format
	message := map[string]interface{}{
		"type":      "message.send",
		"timestamp": time.Now().UnixMilli(),
		"payload": map[string]interface{}{
			"content": prompt,
		},
	}

	if err := conn.WriteJSON(message); err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	logger.DebugC("webhook", fmt.Sprintf("Sent prompt to PicoClaw: %s", prompt))

	// Read responses until we get a complete answer
	// The Pico protocol sends message.create for responses, and may stream them
	var fullResponse string
	responseTimeout := 3 * time.Second // Wait up to 3 seconds after last message
	lastMessageTime := time.Now()

	for {
		// Set a read deadline to detect when no more messages are coming
		conn.SetReadDeadline(time.Now().Add(responseTimeout))

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)

		if err != nil {
			// Check if this is a timeout (means response is complete)
			if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
				// Timeout means no more messages coming
				if fullResponse != "" {
					logger.DebugC("webhook", fmt.Sprintf("Response complete (timeout), length: %d", len(fullResponse)))
					return fullResponse, nil
				}
				// Still waiting for first response
				if time.Since(lastMessageTime) > 30*time.Second {
					return "", fmt.Errorf("no response received within timeout")
				}
				continue
			}

			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				break
			}
			return "", fmt.Errorf("failed to read response: %w", err)
		}

		lastMessageTime = time.Now()
		msgType, _ := msg["type"].(string)
		logger.DebugC("webhook", fmt.Sprintf("Received message type: %s", msgType))

		switch msgType {
		case "message.create":
			// Extract content from payload
			if payload, ok := msg["payload"].(map[string]interface{}); ok {
				// Check if this is a thought message (skip it)
				if thought, ok := payload["thought"].(bool); ok && thought {
					logger.DebugC("webhook", "Skipping thought message")
					continue
				}

				if content, ok := payload["content"].(string); ok {
					fullResponse += content
					logger.DebugC("webhook", fmt.Sprintf("Accumulated response length: %d", len(fullResponse)))
				}
			}
		case "typing.start", "typing.stop":
			// Skip typing indicators
			continue
		case "error":
			// Extract error from payload
			errorMsg := "unknown error"
			if payload, ok := msg["payload"].(map[string]interface{}); ok {
				if message, ok := payload["message"].(string); ok {
					errorMsg = message
				} else if code, ok := payload["code"].(string); ok {
					errorMsg = code
				}
			}
			logger.ErrorC("webhook", fmt.Sprintf("AI returned error: %s, full message: %+v", errorMsg, msg))
			return "", fmt.Errorf("AI error: %s", errorMsg)
		case "pong":
			// Skip pong messages
			continue
		}
	}

	if fullResponse == "" {
		return "No response received", nil
	}

	return fullResponse, nil
}

// CreatePicoClawProcessor creates a processor that uses PicoClaw's AI
// wsURL should be like "ws://localhost:18790/pico/ws" (gateway's Pico channel endpoint)
// token is the composed token (pico-<pid_token><pico_token>)
func CreatePicoClawProcessor(wsURL, token string) *Processor {
	return NewProcessor(PicoClawProcessor(wsURL, token))
}
