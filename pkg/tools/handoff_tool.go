// PicoClaw - Ultra-lightweight personal AI agent
// Swarm mode support for multi-agent coordination
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package tools

import (
	"context"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/swarm"
)

// HandoffTool implements the handoff tool for swarm mode.
type HandoffTool struct {
	coordinator *swarm.HandoffCoordinator
	channel     string
	chatID      string
}

// NewHandoffTool creates a new handoff tool.
func NewHandoffTool(coordinator *swarm.HandoffCoordinator) *HandoffTool {
	return &HandoffTool{
		coordinator: coordinator,
		channel:     "cli",
		chatID:      "direct",
	}
}

// Name returns the tool name.
func (t *HandoffTool) Name() string {
	return "handoff"
}

// Description returns the tool description.
func (t *HandoffTool) Description() string {
	return "Delegate this task to another agent in the swarm. Use when you cannot handle the task due to capability constraints or system overload."
}

// Parameters returns the tool parameters schema.
func (t *HandoffTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"reason": map[string]any{
				"type":        "string",
				"enum":        []string{"no_capability", "overloaded", "user_request"},
				"description": "The reason for handing off this task",
			},
			"required_capability": map[string]any{
				"type":        "string",
				"description": "The specific capability required to handle this task",
			},
			"context": map[string]any{
				"type":        "string",
				"description": "Additional context about why this handoff is needed",
			},
		},
	}
}

// SetContext sets the channel and chat ID for the tool.
func (t *HandoffTool) SetContext(channel, chatID string) {
	t.channel = channel
	t.chatID = chatID
}

// Execute executes the handoff tool.
func (t *HandoffTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t.coordinator == nil {
		return ErrorResult("Swarm mode is not enabled or handoff coordinator not configured").WithError(
			fmt.Errorf("handoff coordinator is nil"))
	}

	// Parse reason
	reasonStr, _ := args["reason"].(string)
	var reason swarm.HandoffReason
	switch reasonStr {
	case "no_capability":
		reason = swarm.ReasonNoCapability
	case "overloaded":
		reason = swarm.ReasonOverloaded
	case "user_request":
		reason = swarm.ReasonUserRequest
	default:
		reason = swarm.ReasonNoCapability
	}

	// Parse required capability
	requiredCap, _ := args["required_capability"].(string)

	// Parse context
	contextMsg, _ := args["context"].(string)

	// Build handoff request
	req := &swarm.HandoffRequest{
		Reason:      reason,
		RequiredCap: requiredCap,
		Metadata:    make(map[string]string),
	}

	if contextMsg != "" {
		req.Metadata["context"] = contextMsg
	}

	// Execute handoff
	resp, err := t.coordinator.InitiateHandoff(ctx, req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Handoff failed: %v", err)).WithError(err)
	}

	if !resp.Accepted {
		return ErrorResult(fmt.Sprintf("Handoff rejected by all nodes: %s", resp.Reason)).WithError(
			fmt.Errorf("handoff rejected: %s", resp.Reason))
	}

	// Build result message
	resultMsg := fmt.Sprintf("Task handed off to node %s\n", resp.NodeID)
	if resp.Reason != "" {
		resultMsg += fmt.Sprintf("Note: %s\n", resp.Reason)
	}

	return &ToolResult{
		ForLLM:  resultMsg + "The target node will process this task and respond to the user.",
		ForUser: "Your task has been delegated to another agent in the swarm. They will respond shortly.",
		Silent:  false,
		IsError: false,
		Async:   true, // Handoff is async - target node will respond directly
	}
}

// CanHandle reports whether the local node can handle the given capability.
func (t *HandoffTool) CanHandle(requiredCap string) bool {
	if t.coordinator == nil {
		return true // If swarm is disabled, we can "handle" everything
	}
	return t.coordinator.CanHandle(requiredCap)
}
