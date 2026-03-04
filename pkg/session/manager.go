package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// AgentMessage represents a message with extended metadata
// This is a local copy to avoid import cycle with pkg/agent
type AgentMessage struct {
	// Core Fields (compatible with providers.Message)
	Role             string               `json:"role"`
	Content          string               `json:"content"`
	ReasoningContent string               `json:"reasoning_content,omitempty"`
	ToolCalls        []providers.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string               `json:"tool_call_id,omitempty"`

	// Extended Fields
	Type      string         `json:"type"`                 // Semantic message type
	Metadata  map[string]any `json:"metadata,omitempty"`   // Arbitrary metadata
	Timestamp time.Time      `json:"timestamp"`            // Message creation time
	SessionID string         `json:"session_id,omitempty"` // Associated session

	// Type-specific fields (artifact, attachment, event, subagent, progress, etc.)
	ArtifactID         string         `json:"artifact_id,omitempty"`
	ArtifactType       string         `json:"artifact_type,omitempty"`
	ArtifactMIME       string         `json:"artifact_mime,omitempty"`
	ArtifactSize       int64          `json:"artifact_size,omitempty"`
	AttachmentURL      string         `json:"attachment_url,omitempty"`
	AttachmentSize     int64          `json:"attachment_size,omitempty"`
	AttachmentFilename string         `json:"attachment_filename,omitempty"`
	EventType          string         `json:"event_type,omitempty"`
	EventData          map[string]any `json:"event_data,omitempty"`
	SubagentID         string         `json:"subagent_id,omitempty"`
	SubagentLabel      string         `json:"subagent_label,omitempty"`
	SubagentStatus     string         `json:"subagent_status,omitempty"`
	Iterations         int            `json:"iterations,omitempty"`
	Progress           float64        `json:"progress,omitempty"`
	ProgressText       string         `json:"progress_text,omitempty"`
	OriginChannel      string         `json:"origin_channel,omitempty"`
	OriginChatID       string         `json:"origin_chat_id,omitempty"`
	AgentID            string         `json:"agent_id,omitempty"`
}

// Session stores conversation history using AgentMessage for rich metadata support
type Session struct {
	Key      string          `json:"key"`
	Messages []*AgentMessage `json:"messages"` // Phase 2: Changed to AgentMessage
	Summary  string          `json:"summary,omitempty"`
	Created  time.Time       `json:"created"`
	Updated  time.Time       `json:"updated"`
	Version  int             `json:"version"` // Schema version for migration tracking
}

type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	storage  string
}

func NewSessionManager(storage string) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
		storage:  storage,
	}

	if storage != "" {
		os.MkdirAll(storage, 0o755)
		sm.loadSessions()
	}

	return sm
}

func (sm *SessionManager) GetOrCreate(key string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if ok {
		return session
	}

	session = &Session{
		Key:      key,
		Messages: []*AgentMessage{}, // Phase 2: Use AgentMessage
		Created:  time.Now(),
		Updated:  time.Now(),
		Version:  1, // Current schema version
	}
	sm.sessions[key] = session

	return session
}

// ============================================================================
// Conversion Helpers (to avoid import cycle with pkg/agent)
// ============================================================================

// fromLLMMessage converts a providers.Message to AgentMessage
func fromLLMMessage(msg providers.Message) *AgentMessage {
	agentMsg := &AgentMessage{
		Role:             msg.Role,
		Content:          msg.Content,
		ReasoningContent: msg.ReasoningContent,
		ToolCalls:        msg.ToolCalls,
		ToolCallID:       msg.ToolCallID,
		Timestamp:        time.Now(),
		Metadata:         make(map[string]any),
	}

	// Infer type from role
	switch msg.Role {
	case "user":
		agentMsg.Type = "user"
	case "assistant":
		agentMsg.Type = "assistant"
	case "tool":
		agentMsg.Type = "tool"
	case "system":
		agentMsg.Type = "system"
	default:
		agentMsg.Type = "user"
	}

	return agentMsg
}

// fromLLMMessages converts a slice of providers.Message to AgentMessage slice
func fromLLMMessages(messages []providers.Message) []*AgentMessage {
	result := make([]*AgentMessage, len(messages))
	for i, msg := range messages {
		result[i] = fromLLMMessage(msg)
	}
	return result
}

// toLLMMessage converts an AgentMessage to providers.Message
func toLLMMessage(msg *AgentMessage) providers.Message {
	llmMsg := providers.Message{
		Role:             msg.Role,
		Content:          msg.Content,
		ReasoningContent: msg.ReasoningContent,
		ToolCalls:        msg.ToolCalls,
		ToolCallID:       msg.ToolCallID,
	}

	// Add context for special message types
	if msg.Type != "" && msg.Type != msg.Role {
		// Add metadata context to content if it's a special type
		switch msg.Type {
		case "subagent_result":
			if msg.SubagentLabel != "" || msg.SubagentStatus != "" {
				prefix := fmt.Sprintf("[Subagent Result: %s, Status: %s]\n", msg.SubagentLabel, msg.SubagentStatus)
				llmMsg.Content = prefix + llmMsg.Content
			}
		case "artifact":
			if msg.ArtifactType != "" {
				prefix := fmt.Sprintf("[Artifact: %s]\n", msg.ArtifactType)
				llmMsg.Content = prefix + llmMsg.Content
			}
		case "attachment":
			if msg.AttachmentFilename != "" {
				prefix := fmt.Sprintf("[Attachment: %s]\n", msg.AttachmentFilename)
				llmMsg.Content = prefix + llmMsg.Content
			}
		}
	}

	return llmMsg
}

// toLLMMessages converts a slice of AgentMessage to providers.Message slice
func toLLMMessages(messages []*AgentMessage) []providers.Message {
	result := make([]providers.Message, len(messages))
	for i, msg := range messages {
		result[i] = toLLMMessage(msg)
	}
	return result
}

// AddMessage creates and adds an AgentMessage from role and content
// This is the primary method for adding simple messages to sessions
func (sm *SessionManager) AddMessage(sessionKey, role, content string) {
	// Create AgentMessage with proper type inference
	msg := &AgentMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}

	// Infer type from role
	switch role {
	case "user":
		msg.Type = "user"
	case "assistant":
		msg.Type = "assistant"
	case "tool":
		msg.Type = "tool"
	case "system":
		msg.Type = "system"
	default:
		msg.Type = "user"
	}

	sm.AddAgentMessage(sessionKey, msg)
}

// AddFullMessage adds a complete providers.Message to the session
// Converts it to AgentMessage for storage (backward compatibility)
func (sm *SessionManager) AddFullMessage(sessionKey string, msg providers.Message) {
	agentMsg := fromLLMMessage(msg)
	sm.AddAgentMessage(sessionKey, agentMsg)
}

// AddAgentMessage adds an AgentMessage directly to the session
// This is the core method that all other add methods delegate to
func (sm *SessionManager) AddAgentMessage(sessionKey string, msg *AgentMessage) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionKey]
	if !ok {
		session = &Session{
			Key:      sessionKey,
			Messages: []*AgentMessage{},
			Created:  time.Now(),
			Version:  1,
		}
		sm.sessions[sessionKey] = session
	}

	session.Messages = append(session.Messages, msg)
	session.Updated = time.Now()
}

// GetHistory returns the session history as providers.Message slice
// Converts AgentMessage to providers.Message for backward compatibility with LLM calls
func (sm *SessionManager) GetHistory(key string) []providers.Message {
	agentHistory := sm.GetAgentHistory(key)
	return toLLMMessages(agentHistory)
}

// GetAgentHistory returns the raw AgentMessage slice for a session
// This provides access to full metadata and extended fields
func (sm *SessionManager) GetAgentHistory(key string) []*AgentMessage {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[key]
	if !ok {
		return []*AgentMessage{}
	}

	// Return a copy to prevent external modification
	history := make([]*AgentMessage, len(session.Messages))
	copy(history, session.Messages)
	return history
}

func (sm *SessionManager) GetSummary(key string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, ok := sm.sessions[key]
	if !ok {
		return ""
	}
	return session.Summary
}

func (sm *SessionManager) SetSummary(key string, summary string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if ok {
		session.Summary = summary
		session.Updated = time.Now()
	}
}

// ClearHistory clears all messages from a session, resetting it to an empty state
// It also deletes the session from memory and removes the session file from disk
func (sm *SessionManager) ClearHistory(key string) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if !ok {
		return 0
	}

	clearedCount := len(session.Messages)

	// Remove session from memory
	delete(sm.sessions, key)

	// Delete session file from disk if storage is configured
	if sm.storage != "" {
		filename := sanitizeFilename(key)
		sessionPath := filepath.Join(sm.storage, filename+".json")
		_ = os.Remove(sessionPath) // Ignore error if file doesn't exist
	}

	return clearedCount
}

func (sm *SessionManager) TruncateHistory(key string, keepLast int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if !ok {
		return
	}

	if keepLast <= 0 {
		session.Messages = []*AgentMessage{}
		session.Updated = time.Now()
		return
	}

	if len(session.Messages) <= keepLast {
		return
	}

	session.Messages = session.Messages[len(session.Messages)-keepLast:]
	session.Updated = time.Now()
}

// sanitizeFilename converts a session key into a cross-platform safe filename.
// Session keys use "channel:chatID" (e.g. "telegram:123456") but ':' is the
// volume separator on Windows, so filepath.Base would misinterpret the key.
// We replace it with '_'. The original key is preserved inside the JSON file,
// so loadSessions still maps back to the right in-memory key.
func sanitizeFilename(key string) string {
	return strings.ReplaceAll(key, ":", "_")
}

func (sm *SessionManager) Save(key string) error {
	if sm.storage == "" {
		return nil
	}

	filename := sanitizeFilename(key)

	// filepath.IsLocal rejects empty names, "..", absolute paths, and
	// OS-reserved device names (NUL, COM1 … on Windows).
	// The extra checks reject "." and any directory separators so that
	// the session file is always written directly inside sm.storage.
	if filename == "." || !filepath.IsLocal(filename) || strings.ContainsAny(filename, `/\`) {
		return os.ErrInvalid
	}

	// Snapshot under read lock, then perform slow file I/O after unlock.
	sm.mu.RLock()
	stored, ok := sm.sessions[key]
	if !ok {
		sm.mu.RUnlock()
		return nil
	}

	snapshot := Session{
		Key:     stored.Key,
		Summary: stored.Summary,
		Created: stored.Created,
		Updated: stored.Updated,
		Version: stored.Version,
	}
	if len(stored.Messages) > 0 {
		snapshot.Messages = make([]*AgentMessage, len(stored.Messages))
		copy(snapshot.Messages, stored.Messages)
	} else {
		snapshot.Messages = []*AgentMessage{}
	}
	sm.mu.RUnlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	sessionPath := filepath.Join(sm.storage, filename+".json")
	tmpFile, err := os.CreateTemp(sm.storage, "session-*.tmp")
	if err != nil {
		return err
	}

	tmpPath := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Chmod(0o644); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, sessionPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

// loadSessions loads session files with automatic migration from legacy format
func (sm *SessionManager) loadSessions() error {
	files, err := os.ReadDir(sm.storage)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		sessionPath := filepath.Join(sm.storage, file.Name())
		data, err := os.ReadFile(sessionPath)
		if err != nil {
			logger.WarnF(fmt.Sprintf("Failed to read session file: %s", file.Name()), map[string]any{"error": err})
			continue
		}

		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			// Try loading as legacy format ([]providers.Message)
			var legacySession struct {
				Key      string              `json:"key"`
				Messages []providers.Message `json:"messages"`
				Summary  string              `json:"summary,omitempty"`
				Created  time.Time           `json:"created"`
				Updated  time.Time           `json:"updated"`
			}

			err2 := json.Unmarshal(data, &legacySession)
			if err2 == nil {
				// Successfully loaded legacy format - migrate to new format
				session = Session{
					Key:      legacySession.Key,
					Messages: fromLLMMessages(legacySession.Messages),
					Summary:  legacySession.Summary,
					Created:  legacySession.Created,
					Updated:  legacySession.Updated,
					Version:  1,
				}

				logger.Info(fmt.Sprintf("Migrated legacy session to new format: %s", session.Key))

				// Save migrated session immediately
				sm.sessions[session.Key] = &session
				if saveErr := sm.Save(session.Key); saveErr != nil {
					logger.WarnF(fmt.Sprintf("Failed to save migrated session: %s", session.Key), map[string]any{"error": saveErr})
				}
				continue
			}

			// Both attempts failed
			logger.WarnF(fmt.Sprintf("Failed to load session file: %s", file.Name()), map[string]any{"new_format_error": err, "legacy_format_error": err2})
			continue
		}

		// Successfully loaded new format
		// Ensure version is set for sessions that might not have it
		if session.Version == 0 {
			session.Version = 1
		}

		sm.sessions[session.Key] = &session
	}

	return nil
}

// SetHistory updates the messages of a session from providers.Message slice
// Converts to AgentMessage for storage (backward compatibility)
func (sm *SessionManager) SetHistory(key string, history []providers.Message) {
	agentHistory := fromLLMMessages(history)
	sm.SetAgentHistory(key, agentHistory)
}

// SetAgentHistory updates the messages of a session with AgentMessage slice
// This is the core method for bulk history updates
func (sm *SessionManager) SetAgentHistory(key string, history []*AgentMessage) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[key]
	if ok {
		// Create a deep copy to strictly isolate internal state
		msgs := make([]*AgentMessage, len(history))
		copy(msgs, history)
		session.Messages = msgs
		session.Updated = time.Now()
	}
}
