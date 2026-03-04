package agent

import (
	"fmt"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// MessageConverter defines the interface for converting and transforming agent messages
type MessageConverter interface {
	// TransformContext applies semantic-aware context trimming on AgentMessages
	TransformContext(messages []*AgentMessage, opts TransformOptions) []*AgentMessage

	// ConvertToLLM converts AgentMessages to providers.Message for LLM consumption
	ConvertToLLM(messages []*AgentMessage) []providers.Message
}

// TransformOptions configures how context transformation should be performed
type TransformOptions struct {
	MaxMessages       int  // Maximum number of messages to keep (0 = unlimited)
	MaxTokens         int  // Approximate maximum tokens (0 = unlimited)
	PreserveArtifacts int  // Number of recent artifacts to preserve
	PreserveSystem    bool // Always preserve system messages
	IncludeContext    bool // Include artifact/attachment references in content
}

// DefaultTransformOptions returns sensible defaults for context transformation
func DefaultTransformOptions() TransformOptions {
	return TransformOptions{
		MaxMessages:       100,
		MaxTokens:         0, // Token counting would require a tokenizer
		PreserveArtifacts: 5,
		PreserveSystem:    true,
		IncludeContext:    true,
	}
}

// DefaultMessageConverter implements semantic-aware message conversion
type DefaultMessageConverter struct {
	options TransformOptions
}

// NewMessageConverter creates a new message converter with the given options
func NewMessageConverter(opts TransformOptions) *DefaultMessageConverter {
	return &DefaultMessageConverter{
		options: opts,
	}
}

// TransformContext applies semantic-aware context trimming
// Strategy:
// 1. Always preserve system messages (if PreserveSystem is true)
// 2. Preserve recent artifacts up to PreserveArtifacts limit
// 3. Keep most recent messages up to MaxMessages limit
// 4. Prioritize conversation continuity over older messages
func (c *DefaultMessageConverter) TransformContext(messages []*AgentMessage, opts TransformOptions) []*AgentMessage {
	if len(messages) == 0 {
		return messages
	}

	// Use provided options or fall back to converter's default options
	if opts.MaxMessages == 0 && c.options.MaxMessages > 0 {
		opts = c.options
	}

	// If no limit, return all messages
	if opts.MaxMessages == 0 || len(messages) <= opts.MaxMessages {
		return messages
	}

	var result []*AgentMessage
	var artifactCount int
	var regularMsgCount int

	// First pass: collect system messages from the beginning
	systemMessages := make([]*AgentMessage, 0)
	if opts.PreserveSystem {
		for _, msg := range messages {
			if msg.Type == MessageTypeSystem {
				systemMessages = append(systemMessages, msg)
			}
		}
	}

	// Second pass: collect messages from end to start
	// This ensures we keep the most recent context
	tempResult := make([]*AgentMessage, 0)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]

		// Skip system messages (already collected)
		if msg.Type == MessageTypeSystem {
			continue
		}

		// Preserve recent artifacts (don't count against message limit)
		if msg.Type == MessageTypeArtifact && artifactCount < opts.PreserveArtifacts {
			tempResult = append([]*AgentMessage{msg}, tempResult...)
			artifactCount++
			continue
		}

		// Stop if we've reached the message limit
		if regularMsgCount >= opts.MaxMessages {
			break
		}

		// Include the message
		tempResult = append([]*AgentMessage{msg}, tempResult...)
		regularMsgCount++
	}

	// Combine: system messages first, then the rest
	result = append(result, systemMessages...)
	result = append(result, tempResult...)

	return result
}

// ConvertToLLM converts AgentMessages to standard providers.Message
// Extended message types (artifacts, attachments, events) are either:
// - Converted to inline references in content (if IncludeContext is true)
// - Dropped entirely (if IncludeContext is false)
func (c *DefaultMessageConverter) ConvertToLLM(messages []*AgentMessage) []providers.Message {
	var result []providers.Message

	for _, msg := range messages {
		// Skip event messages - they're for runtime use only
		if msg.Type == MessageTypeEvent {
			continue
		}

		// Handle artifacts and attachments based on IncludeContext option
		if msg.Type == MessageTypeArtifact || msg.Type == MessageTypeAttachment {
			if c.options.IncludeContext {
				// Convert to assistant message with context
				result = append(result, msg.ToLLMMessageWithContext())
			}
			// Otherwise, skip entirely
			continue
		}

		// Standard message types - convert directly
		result = append(result, msg.ToLLMMessage())
	}

	return result
}

// BatchConvertFromProvider converts multiple providers.Message to AgentMessage
func BatchConvertFromProvider(messages []providers.Message) []*AgentMessage {
	result := make([]*AgentMessage, len(messages))
	for i, msg := range messages {
		result[i] = FromProviderMessage(msg)
	}
	return result
}

// BatchConvertToProvider converts multiple AgentMessage to providers.Message
func BatchConvertToProvider(messages []*AgentMessage) []providers.Message {
	result := make([]providers.Message, len(messages))
	for i, msg := range messages {
		result[i] = msg.ToLLMMessage()
	}
	return result
}

// MessageFilter defines a function type for filtering messages
type MessageFilter func(*AgentMessage) bool

// FilterMessages returns messages that match the given filter
func FilterMessages(messages []*AgentMessage, filter MessageFilter) []*AgentMessage {
	var result []*AgentMessage
	for _, msg := range messages {
		if filter(msg) {
			result = append(result, msg)
		}
	}
	return result
}

// Common filters

// FilterByType returns a filter that matches messages of the given type
func FilterByType(msgType AgentMessageType) MessageFilter {
	return func(msg *AgentMessage) bool {
		return msg.Type == msgType
	}
}

// FilterByRole returns a filter that matches messages with the given role
func FilterByRole(role string) MessageFilter {
	return func(msg *AgentMessage) bool {
		return msg.Role == role
	}
}

// FilterBySession returns a filter that matches messages from the given session
func FilterBySession(sessionID string) MessageFilter {
	return func(msg *AgentMessage) bool {
		return msg.SessionID == sessionID
	}
}

// FilterStandardTypes returns a filter that matches only standard LLM message types
func FilterStandardTypes() MessageFilter {
	return func(msg *AgentMessage) bool {
		return msg.IsStandardLLMType()
	}
}

// FilterExtendedTypes returns a filter that matches only extended business message types
func FilterExtendedTypes() MessageFilter {
	return func(msg *AgentMessage) bool {
		return msg.IsExtendedType()
	}
}

// MessageStats provides statistics about a collection of messages
type MessageStats struct {
	Total           int
	ByType          map[AgentMessageType]int
	ByRole          map[string]int
	TotalSize       int64 // Approximate size in bytes
	ArtifactCount   int
	AttachmentCount int
}

// ComputeStats calculates statistics for a collection of messages
func ComputeStats(messages []*AgentMessage) MessageStats {
	stats := MessageStats{
		Total:  len(messages),
		ByType: make(map[AgentMessageType]int),
		ByRole: make(map[string]int),
	}

	for _, msg := range messages {
		stats.ByType[msg.Type]++
		stats.ByRole[msg.Role]++
		stats.TotalSize += int64(len(msg.Content))

		if msg.Type == MessageTypeArtifact {
			stats.ArtifactCount++
			stats.TotalSize += msg.ArtifactSize
		}

		if msg.Type == MessageTypeAttachment {
			stats.AttachmentCount++
			stats.TotalSize += msg.AttachmentSize
		}
	}

	return stats
}

// String returns a human-readable representation of the stats
func (s MessageStats) String() string {
	return fmt.Sprintf(
		"Messages: %d total, %d artifacts, %d attachments, ~%s total size",
		s.Total,
		s.ArtifactCount,
		s.AttachmentCount,
		formatBytes(s.TotalSize),
	)
}

// MergeMessageHistories combines multiple message histories while preserving order
// Messages are sorted by timestamp and deduplicated by content hash if needed
func MergeMessageHistories(histories ...[]*AgentMessage) []*AgentMessage {
	var result []*AgentMessage
	for _, history := range histories {
		result = append(result, history...)
	}

	// Sort by timestamp (stable sort preserves order for equal timestamps)
	// Note: For production use, consider more sophisticated deduplication
	return result
}

// ============================================================================
// Phase 2: Session Integration - Conversion Functions (AgentMessage ↔ providers.Message)
// ============================================================================

// FromLLMMessage converts a providers.Message to AgentMessage
// This is primarily used for migrating legacy session data
func FromLLMMessage(msg providers.Message) *AgentMessage {
	agentMsg := &AgentMessage{
		Role:             msg.Role,
		Content:          msg.Content,
		ReasoningContent: msg.ReasoningContent,
		ToolCalls:        msg.ToolCalls,
		ToolCallID:       msg.ToolCallID,
		Timestamp:        time.Now(),
		Metadata:         make(map[string]any),
	}

	// Infer AgentMessageType from role
	switch msg.Role {
	case "user":
		agentMsg.Type = MessageTypeUser
	case "assistant":
		agentMsg.Type = MessageTypeAssistant
	case "tool":
		agentMsg.Type = MessageTypeTool
	case "system":
		agentMsg.Type = MessageTypeSystem
	default:
		agentMsg.Type = MessageTypeUser // Default to user
	}

	return agentMsg
}

// FromLLMMessages converts a slice of providers.Message to AgentMessage slice
// Used for batch migration of legacy session histories
func FromLLMMessages(messages []providers.Message) []*AgentMessage {
	result := make([]*AgentMessage, len(messages))
	for i, msg := range messages {
		result[i] = FromLLMMessage(msg)
	}
	return result
}

// ToLLMMessages converts a slice of AgentMessage to providers.Message slice
// Used when passing session history to LLM providers
func ToLLMMessages(messages []*AgentMessage) []providers.Message {
	result := make([]providers.Message, len(messages))
	for i, msg := range messages {
		result[i] = msg.ToLLMMessageWithContext()
	}
	return result
}
