// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import "strings"

// CalcTurnScore computes a value score for a completed turn.
//
// Scoring rules (range roughly -2 to 15):
//
//	+3  has tool calls
//	+2  has write/edit/append tool call (modifying tools)
//	+2  tool count > 3
//	+3  intent = task / code / debug
//	+1  intent = question
//	+0  intent = chat (or empty)
//	+2  reply length > 500 chars
//	-2  user + reply total < 80 chars
//	+3  user message contains "记住" or "重要" (remember / important)
//
// alwaysKeepThreshold (≥ 7) marks a Turn as always_keep in instant memory.
func CalcTurnScore(input RuntimeInput) int {
	score := 0

	// --- Tool activity ---
	if len(input.ToolCalls) > 0 {
		score += 3
	}
	for _, tc := range input.ToolCalls {
		n := strings.ToLower(tc.Name)
		if n == "write_file" || n == "edit_file" || n == "append_file" ||
			n == "write" || n == "edit" || n == "append" {
			score += 2
			break // count once
		}
	}
	if len(input.ToolCalls) > 3 {
		score += 2
	}

	// --- Intent weight ---
	switch strings.ToLower(input.Intent) {
	case "task", "code", "debug":
		score += 3
	case "question":
		score += 1
	// "chat" or empty: 0
	}

	// --- Content density ---
	if len(input.AssistantReply) > 500 {
		score += 2
	}
	if len(input.UserMessage)+len(input.AssistantReply) < 80 {
		score -= 2
	}

	// --- Explicit importance markers ---
	if strings.Contains(input.UserMessage, "记住") ||
		strings.Contains(input.UserMessage, "重要") ||
		strings.Contains(strings.ToLower(input.UserMessage), "remember") ||
		strings.Contains(strings.ToLower(input.UserMessage), "important") {
		score += 3
	}

	return score
}

// alwaysKeepThreshold is the minimum score for a Turn to be unconditionally
// included in instant memory (regardless of tag matching).
const alwaysKeepThreshold = 7
