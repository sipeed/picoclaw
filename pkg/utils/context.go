// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package utils

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// CalculateDefaultMaxContextRunes computes a default context limit based on the model's context window.
// Strategy: Use 75% of the context window and convert to rune estimate.
//
// Token-to-rune conversion ratios (conservative estimates):
//   - English: ~4 chars per token
//   - Chinese: ~1.5-2 chars per token
//   - Mixed: ~3 chars per token (used here for safety)
func CalculateDefaultMaxContextRunes(contextWindow int) int {
	if contextWindow <= 0 {
		// Conservative fallback when context window is unknown
		return 8000 // ~2000 tokens
	}

	// Use 75% of context window to leave headroom
	targetTokens := int(float64(contextWindow) * 0.75)

	// Convert tokens to runes using conservative ratio
	const avgCharsPerToken = 3
	return targetTokens * avgCharsPerToken
}

// ResolveMaxContextRunes determines the final MaxContextRunes value to use.
// Priority: explicit config > auto-calculate > conservative default
func ResolveMaxContextRunes(configValue, contextWindow int) int {
	switch {
	case configValue > 0:
		// Explicitly configured, use as-is
		return configValue
	case configValue == -1:
		// Explicitly disabled
		return -1
	default:
		// 0 or unset: auto-calculate
		return CalculateDefaultMaxContextRunes(contextWindow)
	}
}

// MeasureContextRunes calculates the total rune count of a message list.
// Includes content, reasoning content, and estimates for tool calls.
func MeasureContextRunes(messages []providers.Message) int {
	totalRunes := 0
	for _, msg := range messages {
		totalRunes += utf8.RuneCountInString(msg.Content)
		totalRunes += utf8.RuneCountInString(msg.ReasoningContent)

		// Tool calls: serialize to JSON and count
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				totalRunes += utf8.RuneCountInString(tc.Name)
				// Arguments: serialize and count
				if argsJSON, err := json.Marshal(tc.Arguments); err == nil {
					totalRunes += utf8.RuneCount(argsJSON)
				} else {
					// Fallback estimate if serialization fails
					totalRunes += 100
				}
			}
		}

		// ToolCallID
		totalRunes += utf8.RuneCountInString(msg.ToolCallID)
	}
	return totalRunes
}

// TruncateContextSmart intelligently truncates message history to fit within maxRunes.
//
// Strategy:
//  1. Always preserve system messages (they define the agent's behavior)
//  2. Keep the most recent messages (they contain current context)
//  3. Drop older middle messages when necessary
//  4. Insert a truncation notice to inform the LLM
//
// Returns the truncated message list.
func TruncateContextSmart(messages []providers.Message, maxRunes int) []providers.Message {
	if len(messages) == 0 {
		return messages
	}

	// Separate system messages from others
	var systemMsgs []providers.Message
	var otherMsgs []providers.Message

	for _, msg := range messages {
		if msg.Role == "system" {
			systemMsgs = append(systemMsgs, msg)
		} else {
			otherMsgs = append(otherMsgs, msg)
		}
	}

	// Calculate system message size
	systemRunes := 0
	for _, msg := range systemMsgs {
		systemRunes += utf8.RuneCountInString(msg.Content)
		systemRunes += utf8.RuneCountInString(msg.ReasoningContent)
	}

	// Reserve space for truncation notice (estimate ~80 runes)
	const truncationNoticeEstimate = 80

	// Allocate remaining space for other messages
	remainingRunes := maxRunes - systemRunes - truncationNoticeEstimate
	if remainingRunes <= 0 {
		// System messages already exceed limit - return only system messages
		return systemMsgs
	}

	// Collect recent messages in reverse order until we hit the limit
	var keptMsgs []providers.Message
	currentRunes := 0

	for i := len(otherMsgs) - 1; i >= 0; i-- {
		msg := otherMsgs[i]
		msgRunes := utf8.RuneCountInString(msg.Content) +
			utf8.RuneCountInString(msg.ReasoningContent)

		// Estimate tool call size
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				msgRunes += utf8.RuneCountInString(tc.Name)
				if argsJSON, err := json.Marshal(tc.Arguments); err == nil {
					msgRunes += utf8.RuneCount(argsJSON)
				} else {
					msgRunes += 100
				}
			}
		}
		msgRunes += utf8.RuneCountInString(msg.ToolCallID)

		if currentRunes+msgRunes > remainingRunes {
			// Would exceed limit, stop collecting
			break
		}

		// Prepend to maintain chronological order
		keptMsgs = append([]providers.Message{msg}, keptMsgs...)
		currentRunes += msgRunes
	}

	// If we dropped messages, add a truncation notice
	result := systemMsgs
	if len(keptMsgs) < len(otherMsgs) {
		droppedCount := len(otherMsgs) - len(keptMsgs)
		truncationNotice := providers.Message{
			Role: "system",
			Content: fmt.Sprintf(
				"[Context truncated: %d earlier messages omitted to stay within context limits]",
				droppedCount,
			),
		}
		result = append(result, truncationNotice)
	}

	result = append(result, keptMsgs...)
	return result
}
