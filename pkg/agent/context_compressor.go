package agent

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// ContextCompressor implements a 6-phase context compression algorithm
// inspired by Hermes Agent's context_compressor.py.
//
// Phases:
//  1. Prune — replace old tool results with placeholders (no LLM)
//  2. Protect head — keep first N messages (system prompt + setup)
//  3. Protect tail — keep last messages by token budget
//  4. Summarize — generate structured summary (done externally by caller)
//  5. Assemble — combine head + summary + tail
//  6. Sanitize — fix orphaned tool_call/result pairs
type ContextCompressor struct {
	mu sync.Mutex

	contextLength   int // total context window in tokens
	thresholdTokens int // trigger compression at this token count

	// Protection boundaries
	protectFirstN int // head messages to never compress (default: 3)
	protectLastN  int // fallback tail protection (default: 20)

	// Compression state
	compressionCount int
	previousSummary  string // iterative summary from last compression

	// Token tracking
	lastPromptTokens int
}

const (
	defaultThresholdPercent = 50  // compress at 50% of context
	defaultProtectFirstN    = 3
	defaultProtectLastN     = 20
	charsPerToken           = 4 // rough estimate
	maxPrunedContentLen     = 200
)

// CompressorOption configures the compressor.
type CompressorOption func(*ContextCompressor)

// WithThresholdPercent sets when compression triggers (default: 50%).
func WithThresholdPercent(pct int) CompressorOption {
	return func(cc *ContextCompressor) {
		cc.thresholdTokens = cc.contextLength * pct / 100
	}
}

// WithProtectFirstN sets how many head messages to protect.
func WithProtectFirstN(n int) CompressorOption {
	return func(cc *ContextCompressor) { cc.protectFirstN = n }
}

// WithProtectLastN sets the fallback tail protection count.
func WithProtectLastN(n int) CompressorOption {
	return func(cc *ContextCompressor) { cc.protectLastN = n }
}

// NewContextCompressor creates a compressor for the given context window.
func NewContextCompressor(contextLength int, opts ...CompressorOption) *ContextCompressor {
	cc := &ContextCompressor{
		contextLength:   contextLength,
		thresholdTokens: contextLength * defaultThresholdPercent / 100,
		protectFirstN:   defaultProtectFirstN,
		protectLastN:    defaultProtectLastN,
	}
	for _, opt := range opts {
		opt(cc)
	}
	return cc
}

// ShouldCompress returns true if the current token count exceeds threshold.
func (cc *ContextCompressor) ShouldCompress(promptTokens int) bool {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return promptTokens >= cc.thresholdTokens
}

// UpdateFromResponse tracks token usage from the last LLM response.
func (cc *ContextCompressor) UpdateFromResponse(usage *providers.UsageInfo) {
	if usage == nil {
		return
	}
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.lastPromptTokens = usage.PromptTokens
}

// GetStatus returns compression statistics.
func (cc *ContextCompressor) GetStatus() map[string]any {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return map[string]any{
		"context_length":    cc.contextLength,
		"threshold_tokens":  cc.thresholdTokens,
		"compression_count": cc.compressionCount,
		"has_summary":       cc.previousSummary != "",
		"last_prompt_tokens": cc.lastPromptTokens,
	}
}

// Compress runs the 6-phase algorithm and returns compressed messages
// plus a structured summary string suitable for the session summary.
//
// The summary is generated as a template — the caller should pass it to
// an LLM for actual summarization. This keeps the compressor LLM-agnostic.
func (cc *ContextCompressor) Compress(messages []providers.Message) (compressed []providers.Message, summaryInput string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if len(messages) <= cc.protectFirstN+cc.protectLastN {
		return messages, ""
	}

	// Phase 1: Prune old tool results (cheap, no LLM).
	pruned, prunedCount := cc.pruneOldToolResults(messages, cc.protectLastN)

	// Phase 2+3: Determine boundaries.
	headEnd := cc.protectFirstN
	if headEnd > len(pruned) {
		headEnd = len(pruned)
	}
	tailStart := cc.findTailCut(pruned, headEnd)

	head := pruned[:headEnd]
	middle := pruned[headEnd:tailStart]
	tail := pruned[tailStart:]

	if len(middle) == 0 {
		return messages, ""
	}

	// Phase 4: Serialize middle for summarization.
	summaryInput = cc.serializeForSummary(middle)

	// Build structured summary prompt.
	var sb strings.Builder
	if cc.previousSummary != "" {
		sb.WriteString("UPDATE the previous summary with NEW TURNS below.\n")
		sb.WriteString("PRESERVE all existing information that is still relevant.\n\n")
		sb.WriteString("PREVIOUS SUMMARY:\n")
		sb.WriteString(cc.previousSummary)
		sb.WriteString("\n\nNEW TURNS:\n")
	} else {
		sb.WriteString("Create a structured handoff summary of this conversation:\n\n")
	}
	sb.WriteString(summaryInput)
	sb.WriteString("\n\nUse this structure:\n")
	sb.WriteString("## Goal\n## Progress\n### Done\n### In Progress\n")
	sb.WriteString("## Key Decisions\n## Relevant Files\n## Next Steps\n## Critical Context\n")

	summaryPrompt := sb.String()

	// Phase 5: Assemble — head + placeholder for summary + tail.
	// The actual summary will be injected by the caller after LLM generates it.
	compressed = make([]providers.Message, 0, len(head)+1+len(tail))
	compressed = append(compressed, head...)

	// Add compression notice.
	notice := fmt.Sprintf("[Context compressed: %d messages summarized, %d tool results pruned. Compression #%d]",
		len(middle), prunedCount, cc.compressionCount+1)
	compressed = append(compressed, providers.Message{
		Role:    "system",
		Content: notice,
	})

	compressed = append(compressed, tail...)

	// Phase 6: Sanitize tool pairs.
	compressed = cc.sanitizeToolPairs(compressed)

	cc.compressionCount++

	logger.DebugCF("compressor", "compressed context", map[string]any{
		"original":      len(messages),
		"compressed":    len(compressed),
		"middle_dropped": len(middle),
		"pruned_results": prunedCount,
		"compression_n":  cc.compressionCount,
	})

	return compressed, summaryPrompt
}

// SetPreviousSummary stores the summary from the last compression
// for iterative updates.
func (cc *ContextCompressor) SetPreviousSummary(summary string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.previousSummary = summary
}

// --- Internal phases ---

// pruneOldToolResults replaces long tool results outside the protected
// tail with short placeholders. This is a cheap pre-pass (no LLM).
func (cc *ContextCompressor) pruneOldToolResults(messages []providers.Message, protectTailCount int) ([]providers.Message, int) {
	pruned := make([]providers.Message, len(messages))
	copy(pruned, messages)

	protectFrom := len(messages) - protectTailCount
	if protectFrom < 0 {
		protectFrom = 0
	}

	count := 0
	for i := 0; i < protectFrom; i++ {
		if pruned[i].Role == "tool" && utf8.RuneCountInString(pruned[i].Content) > maxPrunedContentLen {
			pruned[i] = providers.Message{
				Role:       "tool",
				Content:    fmt.Sprintf("[Tool result truncated — originally %d chars]", utf8.RuneCountInString(messages[i].Content)),
				ToolCallID: messages[i].ToolCallID,
			}
			count++
		}
	}
	return pruned, count
}

// findTailCut determines where the protected tail begins.
// Uses token budget (20% of context) walking backwards.
func (cc *ContextCompressor) findTailCut(messages []providers.Message, headEnd int) int {
	budget := cc.contextLength * 20 / 100 // 20% for tail
	tokens := 0

	for i := len(messages) - 1; i >= headEnd; i-- {
		msgTokens := estimateTokens(messages[i])
		if tokens+msgTokens > budget {
			// Don't break tool_call/result pairs.
			cut := i + 1
			cut = alignToolBoundary(messages, cut)
			if cut <= headEnd {
				cut = headEnd + 1
			}
			return cut
		}
		tokens += msgTokens
	}

	// Everything fits in tail budget — protect at least protectLastN.
	cut := len(messages) - cc.protectLastN
	if cut < headEnd {
		cut = headEnd
	}
	return cut
}

// serializeForSummary converts messages to a text format suitable for
// LLM summarization.
func (cc *ContextCompressor) serializeForSummary(turns []providers.Message) string {
	var sb strings.Builder
	for _, msg := range turns {
		content := msg.Content
		if utf8.RuneCountInString(content) > 3000 {
			runes := []rune(content)
			content = string(runes[:1500]) + "\n...[truncated]...\n" + string(runes[len(runes)-1500:])
		}

		role := strings.ToUpper(msg.Role)
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", role, content))

		// Include tool call names for context.
		for _, tc := range msg.ToolCalls {
			sb.WriteString(fmt.Sprintf("  → tool: %s\n", tc.Name))
		}
	}
	return sb.String()
}

// sanitizeToolPairs fixes orphaned tool_call/result pairs after compression.
// - Tool result without matching assistant call → remove
// - Assistant call without result → add stub
func (cc *ContextCompressor) sanitizeToolPairs(messages []providers.Message) []providers.Message {
	// Collect surviving call IDs from assistant messages.
	callIDs := make(map[string]bool)
	for _, msg := range messages {
		if msg.Role == "assistant" {
			for _, tc := range msg.ToolCalls {
				callIDs[tc.ID] = true
			}
		}
	}

	// Collect result IDs.
	resultIDs := make(map[string]bool)
	for _, msg := range messages {
		if msg.Role == "tool" && msg.ToolCallID != "" {
			resultIDs[msg.ToolCallID] = true
		}
	}

	var sanitized []providers.Message

	for _, msg := range messages {
		if msg.Role == "tool" && msg.ToolCallID != "" {
			// Orphan result: call was compressed away.
			if !callIDs[msg.ToolCallID] {
				continue // skip
			}
		}
		sanitized = append(sanitized, msg)
	}

	// Add stubs for calls without results.
	for id := range callIDs {
		if !resultIDs[id] {
			sanitized = append(sanitized, providers.Message{
				Role:       "tool",
				Content:    "[Result from earlier conversation — see context summary]",
				ToolCallID: id,
			})
		}
	}

	return sanitized
}

// --- Helpers ---

// estimateTokens gives a rough token estimate for a message.
func estimateTokens(msg providers.Message) int {
	chars := utf8.RuneCountInString(msg.Content)
	chars += utf8.RuneCountInString(msg.ReasoningContent)
	for _, tc := range msg.ToolCalls {
		chars += utf8.RuneCountInString(tc.Name) + 50 // args overhead
	}
	return chars / charsPerToken
}

// alignToolBoundary moves a cut point forward to avoid splitting
// a tool_call from its result.
func alignToolBoundary(messages []providers.Message, cut int) int {
	if cut >= len(messages) {
		return cut
	}
	// If cut lands on a tool result, include the preceding assistant message.
	if messages[cut].Role == "tool" {
		for i := cut - 1; i >= 0; i-- {
			if messages[i].Role == "assistant" && len(messages[i].ToolCalls) > 0 {
				return i
			}
		}
	}
	return cut
}
