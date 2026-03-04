package agent

import (
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// AgentMessageType defines the semantic type of a message beyond standard LLM roles
type AgentMessageType string

const (
	// Standard LLM message types
	MessageTypeUser      AgentMessageType = "user"
	MessageTypeAssistant AgentMessageType = "assistant"
	MessageTypeTool      AgentMessageType = "tool"
	MessageTypeSystem    AgentMessageType = "system"

	// Extended types for business semantics
	MessageTypeArtifact       AgentMessageType = "artifact"        // LLM-generated artifacts (code, images, documents)
	MessageTypeAttachment     AgentMessageType = "attachment"      // User-provided attachments
	MessageTypeEvent          AgentMessageType = "event"           // System events (task started, interrupted, etc.)
	MessageTypeSubagentResult AgentMessageType = "subagent_result" // Results from subagent execution
	MessageTypeToolProgress   AgentMessageType = "tool_progress"   // Progress updates from long-running tools
)

// ArtifactType categorizes the type of artifact
type ArtifactType string

const (
	ArtifactTypeCode     ArtifactType = "code"
	ArtifactTypeImage    ArtifactType = "image"
	ArtifactTypeDocument ArtifactType = "document"
	ArtifactTypeData     ArtifactType = "data"
	ArtifactTypeOther    ArtifactType = "other"
)

// AgentMessage extends the standard providers.Message with business semantics
// and metadata that are useful for the agent runtime but not necessarily for the LLM.
type AgentMessage struct {
	// ===== Core Fields (compatible with providers.Message) =====
	Role             string               `json:"role"`
	Content          string               `json:"content"`
	ReasoningContent string               `json:"reasoning_content,omitempty"`
	ToolCalls        []providers.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string               `json:"tool_call_id,omitempty"`

	// ===== Extended Fields =====
	Type      AgentMessageType `json:"type"`                 // Semantic message type
	Metadata  map[string]any   `json:"metadata,omitempty"`   // Arbitrary metadata
	Timestamp time.Time        `json:"timestamp"`            // Message creation time
	SessionID string           `json:"session_id,omitempty"` // Associated session

	// ===== Artifact-specific Fields =====
	ArtifactID   string       `json:"artifact_id,omitempty"`   // Unique identifier for artifact
	ArtifactType ArtifactType `json:"artifact_type,omitempty"` // Type of artifact
	ArtifactMIME string       `json:"artifact_mime,omitempty"` // MIME type if applicable
	ArtifactSize int64        `json:"artifact_size,omitempty"` // Size in bytes

	// ===== Attachment-specific Fields =====
	AttachmentURL      string `json:"attachment_url,omitempty"`      // URL or file path
	AttachmentSize     int64  `json:"attachment_size,omitempty"`     // Size in bytes
	AttachmentFilename string `json:"attachment_filename,omitempty"` // Original filename

	// ===== Event-specific Fields =====
	EventType string         `json:"event_type,omitempty"` // Event category (task_started, interrupted, etc.)
	EventData map[string]any `json:"event_data,omitempty"` // Event-specific data

	// ===== Subagent-specific Fields (Enhanced) =====
	SubagentID     string `json:"subagent_id,omitempty"`     // Subagent task ID
	SubagentLabel  string `json:"subagent_label,omitempty"`  // Subagent task label
	SubagentStatus string `json:"subagent_status,omitempty"` // Status: running, completed, failed, canceled
	Iterations     int    `json:"iterations,omitempty"`      // Tool loop iterations executed

	// ===== Progress-specific Fields =====
	Progress     float64 `json:"progress,omitempty"`      // Progress percentage (0-100)
	ProgressText string  `json:"progress_text,omitempty"` // Human-readable progress message

	// ===== Context and Routing Fields =====
	OriginChannel string `json:"origin_channel,omitempty"` // Source channel for routing
	OriginChatID  string `json:"origin_chat_id,omitempty"` // Source chat ID for routing
	AgentID       string `json:"agent_id,omitempty"`       // Agent that created this message
}

// ToLLMMessage converts an AgentMessage to a standard providers.Message
// that can be sent to the LLM. Extended fields are dropped or converted to content.
func (am *AgentMessage) ToLLMMessage() providers.Message {
	return providers.Message{
		Role:             am.Role,
		Content:          am.Content,
		ReasoningContent: am.ReasoningContent,
		ToolCalls:        am.ToolCalls,
		ToolCallID:       am.ToolCallID,
	}
}

// ToLLMMessageWithContext converts an AgentMessage to a providers.Message,
// but includes contextual information about artifacts/attachments/subagents in the content.
func (am *AgentMessage) ToLLMMessageWithContext() providers.Message {
	msg := am.ToLLMMessage()

	// Add artifact reference to content if present
	if am.Type == MessageTypeArtifact && am.ArtifactID != "" {
		if msg.Content == "" {
			msg.Content = am.formatArtifactReference()
		} else {
			msg.Content = am.formatArtifactReference() + "\n\n" + msg.Content
		}
	}

	// Add attachment reference to content if present
	if am.Type == MessageTypeAttachment && am.AttachmentURL != "" {
		if msg.Content == "" {
			msg.Content = am.formatAttachmentReference()
		} else {
			msg.Content = am.formatAttachmentReference() + "\n\n" + msg.Content
		}
	}

	// Add subagent result context if present
	if am.Type == MessageTypeSubagentResult && am.SubagentID != "" {
		if msg.Content == "" {
			msg.Content = am.formatSubagentReference()
		} else {
			msg.Content = am.formatSubagentReference() + "\n\n" + msg.Content
		}
	}

	// Add progress context if present
	if am.Type == MessageTypeToolProgress && am.ProgressText != "" {
		if msg.Content == "" {
			msg.Content = am.formatProgressReference()
		} else {
			msg.Content = am.formatProgressReference() + "\n\n" + msg.Content
		}
	}

	return msg
}

// formatArtifactReference creates a human-readable reference to an artifact
func (am *AgentMessage) formatArtifactReference() string {
	ref := "[Artifact"
	if am.ArtifactID != "" {
		ref += " ID: " + am.ArtifactID
	}
	if am.ArtifactType != "" {
		ref += " Type: " + string(am.ArtifactType)
	}
	if am.ArtifactSize > 0 {
		ref += " Size: " + formatBytes(am.ArtifactSize)
	}
	ref += "]"
	return ref
}

// formatAttachmentReference creates a human-readable reference to an attachment
func (am *AgentMessage) formatAttachmentReference() string {
	ref := "[Attachment"
	if am.AttachmentFilename != "" {
		ref += " File: " + am.AttachmentFilename
	}
	if am.AttachmentSize > 0 {
		ref += " Size: " + formatBytes(am.AttachmentSize)
	}
	ref += "]"
	return ref
}

// formatBytes formats byte size in human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return string(rune(bytes)) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return string(rune(bytes/div)) + " " + "KMGTPE"[exp:exp+1] + "B"
}

// FromProviderMessage creates an AgentMessage from a standard providers.Message
func FromProviderMessage(pm providers.Message) *AgentMessage {
	msgType := AgentMessageType(pm.Role)
	if pm.Role == "system" {
		msgType = MessageTypeSystem
	}

	return &AgentMessage{
		Role:             pm.Role,
		Content:          pm.Content,
		ReasoningContent: pm.ReasoningContent,
		ToolCalls:        pm.ToolCalls,
		ToolCallID:       pm.ToolCallID,
		Type:             msgType,
		Timestamp:        time.Now(),
	}
}

// NewUserMessage creates a new user message
func NewUserMessage(content string) *AgentMessage {
	return &AgentMessage{
		Role:      "user",
		Content:   content,
		Type:      MessageTypeUser,
		Timestamp: time.Now(),
	}
}

// NewAssistantMessage creates a new assistant message
func NewAssistantMessage(content string) *AgentMessage {
	return &AgentMessage{
		Role:      "assistant",
		Content:   content,
		Type:      MessageTypeAssistant,
		Timestamp: time.Now(),
	}
}

// NewToolMessage creates a new tool result message
func NewToolMessage(toolCallID, content string) *AgentMessage {
	return &AgentMessage{
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
		Type:       MessageTypeTool,
		Timestamp:  time.Now(),
	}
}

// NewArtifactMessage creates a new artifact message
func NewArtifactMessage(artifactID string, artifactType ArtifactType, content string) *AgentMessage {
	return &AgentMessage{
		Role:         "assistant",
		Content:      content,
		Type:         MessageTypeArtifact,
		ArtifactID:   artifactID,
		ArtifactType: artifactType,
		ArtifactSize: int64(len(content)),
		Timestamp:    time.Now(),
	}
}

// NewAttachmentMessage creates a new attachment message
func NewAttachmentMessage(url, filename string, size int64) *AgentMessage {
	return &AgentMessage{
		Role:               "user",
		Content:            "Attachment: " + filename,
		Type:               MessageTypeAttachment,
		AttachmentURL:      url,
		AttachmentFilename: filename,
		AttachmentSize:     size,
		Timestamp:          time.Now(),
	}
}

// NewEventMessage creates a new event message
func NewEventMessage(eventType string, eventData map[string]any) *AgentMessage {
	return &AgentMessage{
		Role:      "system",
		Type:      MessageTypeEvent,
		EventType: eventType,
		EventData: eventData,
		Timestamp: time.Now(),
	}
}

// IsStandardLLMType returns true if the message type is a standard LLM type
func (am *AgentMessage) IsStandardLLMType() bool {
	return am.Type == MessageTypeUser ||
		am.Type == MessageTypeAssistant ||
		am.Type == MessageTypeTool ||
		am.Type == MessageTypeSystem
}

// IsExtendedType returns true if the message type is an extended business type
func (am *AgentMessage) IsExtendedType() bool {
	return !am.IsStandardLLMType()
}

// Clone creates a deep copy of the message
func (am *AgentMessage) Clone() *AgentMessage {
	clone := *am

	// Deep copy slices and maps
	if am.ToolCalls != nil {
		clone.ToolCalls = make([]providers.ToolCall, len(am.ToolCalls))
		copy(clone.ToolCalls, am.ToolCalls)
	}

	if am.Metadata != nil {
		clone.Metadata = make(map[string]any, len(am.Metadata))
		for k, v := range am.Metadata {
			clone.Metadata[k] = v
		}
	}

	if am.EventData != nil {
		clone.EventData = make(map[string]any, len(am.EventData))
		for k, v := range am.EventData {
			clone.EventData[k] = v
		}
	}

	return &clone
}

// WithMetadata adds metadata to the message (builder pattern)
func (am *AgentMessage) WithMetadata(key string, value any) *AgentMessage {
	if am.Metadata == nil {
		am.Metadata = make(map[string]any)
	}
	am.Metadata[key] = value
	return am
}

// WithSessionID sets the session ID (builder pattern)
func (am *AgentMessage) WithSessionID(sessionID string) *AgentMessage {
	am.SessionID = sessionID
	return am
}

// NewSubagentResultMessage creates a new subagent result message
func NewSubagentResultMessage(subagentID, label, status, content string, iterations int) *AgentMessage {
	return &AgentMessage{
		Role:           "tool",
		Content:        content,
		Type:           MessageTypeSubagentResult,
		SubagentID:     subagentID,
		SubagentLabel:  label,
		SubagentStatus: status,
		Iterations:     iterations,
		Timestamp:      time.Now(),
		Metadata: map[string]any{
			"source":     "subagent",
			"task_id":    subagentID,
			"task_label": label,
			"status":     status,
			"iterations": iterations,
		},
	}
}

// NewProgressMessage creates a new progress update message
func NewProgressMessage(progressText string, progress float64) *AgentMessage {
	return &AgentMessage{
		Role:         "assistant",
		Content:      progressText,
		Type:         MessageTypeToolProgress,
		Progress:     progress,
		ProgressText: progressText,
		Timestamp:    time.Now(),
	}
}

// WithOrigin sets the origin channel and chat ID (builder pattern)
func (am *AgentMessage) WithOrigin(channel, chatID string) *AgentMessage {
	am.OriginChannel = channel
	am.OriginChatID = chatID
	return am
}

// WithAgentID sets the agent ID (builder pattern)
func (am *AgentMessage) WithAgentID(agentID string) *AgentMessage {
	am.AgentID = agentID
	return am
}

// formatSubagentReference creates a human-readable reference to subagent execution
func (am *AgentMessage) formatSubagentReference() string {
	ref := "[Subagent"
	if am.SubagentLabel != "" {
		ref += " '" + am.SubagentLabel + "'"
	}
	if am.SubagentStatus != "" {
		ref += " " + am.SubagentStatus
	}
	if am.Iterations > 0 {
		ref += " after " + string(rune(am.Iterations)) + " iterations"
	}
	ref += "]"
	return ref
}

// formatProgressReference creates a human-readable progress indicator
func (am *AgentMessage) formatProgressReference() string {
	if am.Progress > 0 {
		return "[Progress: " + string(rune(int(am.Progress))) + "%] " + am.ProgressText
	}
	return am.ProgressText
}
