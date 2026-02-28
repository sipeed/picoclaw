// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package pico

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/pico/protocol"
)

// PicoNodeClient sends inter-node messages by connecting to a target node's
// Pico WebSocket endpoint. It delegates the low-level WebSocket lifecycle to
// Client and focuses on swarm-specific payload construction.
type PicoNodeClient struct {
	sourceNodeID string
	client       *Client
}

// NewPicoNodeClient creates a new Pico-based inter-node client.
func NewPicoNodeClient(sourceNodeID, token string) *PicoNodeClient {
	return &PicoNodeClient{
		sourceNodeID: sourceNodeID,
		client:       NewClient(token),
	}
}

// SendMessage connects to the target node's Pico WebSocket, sends a node.request
// with action "message", and blocks until a node.reply is received (or timeout).
func (c *PicoNodeClient) SendMessage(
	ctx context.Context,
	targetAddr, targetNodeID, content, channel, chatID, senderID string,
) (string, error) {
	payload := NewMessagePayload(c.sourceNodeID, content, channel, chatID, senderID)
	return c.sendRequest(ctx, targetAddr, targetNodeID, payload)
}

// SendNodeAction sends an action-based request to a target node via Pico.
// The payload must contain an "action" key. Returns the raw reply payload.
func (c *PicoNodeClient) SendNodeAction(
	ctx context.Context,
	targetAddr string,
	payload NodePayload,
) (NodePayload, error) {
	requestID := uuid.New().String()
	payload[PayloadKeySourceNodeID] = c.sourceNodeID
	payload[PayloadKeyRequestID] = requestID

	logger.InfoCF("pico", "Sending node action via Pico", map[string]any{
		"action":      payload.Action(),
		"target_addr": targetAddr,
		"request_id":  requestID,
	})

	return c.doSend(ctx, targetAddr, requestID, payload)
}

// sendRequest is the internal method for sending a request and returning the "response" string.
func (c *PicoNodeClient) sendRequest(
	ctx context.Context,
	targetAddr, targetNodeID string,
	payload NodePayload,
) (string, error) {
	requestID := uuid.New().String()
	payload[PayloadKeyRequestID] = requestID

	logger.InfoCF("pico", "Sending node request via Pico", map[string]any{
		"action":         payload.Action(),
		"target_node_id": targetNodeID,
		"target_addr":    targetAddr,
		"request_id":     requestID,
	})

	replyPayload, err := c.doSend(ctx, targetAddr, requestID, payload)
	if err != nil {
		return "", err
	}

	if errStr := replyPayload.ErrorMsg(); errStr != "" {
		return "", fmt.Errorf("node error: %s", errStr)
	}

	return replyPayload.Response(), nil
}

// doSend builds a protocol.Message, delegates to Client.SendRequest,
// and validates the reply envelope before returning the payload.
func (c *PicoNodeClient) doSend(
	ctx context.Context,
	targetAddr, requestID string,
	payload NodePayload,
) (NodePayload, error) {
	reqMsg := protocol.Message{
		Type:      protocol.TypeNodeRequest,
		ID:        requestID,
		SessionID: c.sourceNodeID,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}

	reply, err := c.client.SendRequest(ctx, targetAddr, c.sourceNodeID, reqMsg)
	if err != nil {
		return nil, err
	}

	if reply.Type != protocol.TypeNodeReply {
		return nil, fmt.Errorf("unexpected reply type: %s (expected %s)", reply.Type, protocol.TypeNodeReply)
	}

	replyPayload := NodePayload(reply.Payload)
	if replyPayload.RequestID() != requestID {
		return nil, fmt.Errorf("request ID mismatch: got %s, want %s", replyPayload.RequestID(), requestID)
	}

	return replyPayload, nil
}
