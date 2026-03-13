// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"encoding/json"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// parseTurnBoundaries returns the starting index of each Turn in the history.
// A Turn is a complete "user input → LLM iterations → final response" cycle
// (as defined in #1316). Each Turn begins at a user message and extends
// through all subsequent assistant/tool messages until the next user message.
//
// Cutting at a Turn boundary guarantees that no tool-call sequence
// (assistant+ToolCalls → tool results) is split across the cut.
func parseTurnBoundaries(history []providers.Message) []int {
	var starts []int
	for i, msg := range history {
		if msg.Role == "user" {
			starts = append(starts, i)
		}
	}
	return starts
}

// isSafeBoundary reports whether index is a valid Turn boundary — i.e.,
// a position where the kept portion (history[index:]) begins at a user
// message, so no tool-call sequence is torn apart.
func isSafeBoundary(history []providers.Message, index int) bool {
	if index <= 0 || index >= len(history) {
		return true
	}
	return history[index].Role == "user"
}

// findSafeBoundary locates the nearest Turn boundary to targetIndex.
// It prefers the boundary at or before targetIndex (preserving more recent
// context). Falls back to the nearest boundary after targetIndex, and
// returns targetIndex unchanged only when no Turn boundary exists at all.
func findSafeBoundary(history []providers.Message, targetIndex int) int {
	if len(history) == 0 {
		return 0
	}
	if targetIndex <= 0 {
		return 0
	}
	if targetIndex >= len(history) {
		return len(history)
	}

	turns := parseTurnBoundaries(history)
	if len(turns) == 0 {
		return targetIndex
	}

	// Find the last Turn boundary at or before targetIndex.
	// Prefer backward: keeps more recent messages.
	backward := -1
	for _, t := range turns {
		if t <= targetIndex {
			backward = t
		}
	}
	if backward > 0 {
		return backward
	}

	// No valid Turn boundary before target (or only at index 0 which
	// would keep everything). Use the first Turn after targetIndex.
	for _, t := range turns {
		if t > targetIndex {
			return t
		}
	}

	return targetIndex
}

// estimateMessageTokens estimates the token count for a single message,
// including Content, ReasoningContent, ToolCalls arguments, ToolCallID
// metadata, and Media items. Uses a heuristic of 2.5 characters per token.
func estimateMessageTokens(msg providers.Message) int {
	chars := utf8.RuneCountInString(msg.Content)

	// ReasoningContent (extended thinking / chain-of-thought) can be
	// substantial and is stored in session history via AddFullMessage.
	if msg.ReasoningContent != "" {
		chars += utf8.RuneCountInString(msg.ReasoningContent)
	}

	for _, tc := range msg.ToolCalls {
		// Count tool call metadata: ID, type, function name
		chars += len(tc.ID) + len(tc.Type) + len(tc.Name)
		if tc.Function != nil {
			chars += len(tc.Function.Name) + len(tc.Function.Arguments)
		}
	}

	if msg.ToolCallID != "" {
		chars += len(msg.ToolCallID)
	}

	// Media items (images, files) are serialized by provider adapters into
	// multipart or image_url payloads. Use a fixed per-item estimate since
	// actual token cost depends on resolution and provider tokenization.
	const mediaTokensPerItem = 256
	chars += len(msg.Media) * mediaTokensPerItem

	// Per-message overhead for role label, JSON structure, separators.
	const messageOverhead = 12
	chars += messageOverhead

	return chars * 2 / 5
}

// estimateToolDefsTokens estimates the total token cost of tool definitions
// as they appear in the LLM request. Each tool's name, description, and
// JSON schema parameters contribute to the context window budget.
func estimateToolDefsTokens(defs []providers.ToolDefinition) int {
	if len(defs) == 0 {
		return 0
	}

	totalChars := 0
	for _, d := range defs {
		totalChars += len(d.Function.Name) + len(d.Function.Description)

		if d.Function.Parameters != nil {
			if paramJSON, err := json.Marshal(d.Function.Parameters); err == nil {
				totalChars += len(paramJSON)
			}
		}

		// Per-tool overhead: type field, JSON structure, separators.
		totalChars += 20
	}

	return totalChars * 2 / 5
}

// isOverContextBudget checks whether the assembled messages plus tool definitions
// and output reserve would exceed the model's context window. This enables
// proactive compression before calling the LLM, rather than reacting to 400 errors.
func isOverContextBudget(
	contextWindow int,
	messages []providers.Message,
	toolDefs []providers.ToolDefinition,
	maxTokens int,
) bool {
	msgTokens := 0
	for _, m := range messages {
		msgTokens += estimateMessageTokens(m)
	}

	toolTokens := estimateToolDefsTokens(toolDefs)
	total := msgTokens + toolTokens + maxTokens

	return total > contextWindow
}
