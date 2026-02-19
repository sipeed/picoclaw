# Session Manager API Reference

This document provides detailed reference for the Session Manager API.

## Overview

The Session Manager handles conversation history storage and retrieval. It provides:

- In-memory caching for fast access
- Disk persistence for durability
- Automatic summarization support
- Thread-safe operations

## Types

### Session

Represents a conversation session.

```go
type Session struct {
    Key      string              `json:"key"`
    Messages []providers.Message `json:"messages"`
    Summary  string              `json:"summary,omitempty"`
    Created  time.Time           `json:"created"`
    Updated  time.Time           `json:"updated"`
}
```

**Fields:**

| Field | Type | Description |
|-------|------|-------------|
| Key | string | Unique session identifier |
| Messages | []providers.Message | Conversation history |
| Summary | string | Summary of older messages |
| Created | time.Time | Session creation time |
| Updated | time.Time | Last update time |

### SessionManager

Manages sessions with caching and persistence.

```go
type SessionManager struct {
    sessions map[string]*Session
    mu       sync.RWMutex
    storage  string
}
```

## Functions

### NewSessionManager

Creates a new session manager.

```go
func NewSessionManager(storage string) *SessionManager
```

**Parameters:**
- `storage`: Directory path for session files (empty for memory-only)

**Returns:** New SessionManager instance

```go
// With persistence
sm := session.NewSessionManager("/path/to/sessions")

// Memory-only
sm := session.NewSessionManager("")
```

## Methods

### GetOrCreate

Gets an existing session or creates a new one.

```go
func (sm *SessionManager) GetOrCreate(key string) *Session
```

**Parameters:**
- `key`: Session identifier

**Returns:** Session instance (existing or new)

```go
session := sm.GetOrCreate("telegram:123456")
```

### AddMessage

Adds a simple message to the session.

```go
func (sm *SessionManager) AddMessage(sessionKey, role, content string)
```

**Parameters:**
- `sessionKey`: Session identifier
- `role`: Message role ("user", "assistant", "system")
- `content`: Message content

```go
sm.AddMessage("telegram:123456", "user", "Hello!")
sm.AddMessage("telegram:123456", "assistant", "Hi there!")
```

### AddFullMessage

Adds a complete message with all fields.

```go
func (sm *SessionManager) AddFullMessage(sessionKey string, msg providers.Message)
```

**Parameters:**
- `sessionKey`: Session identifier
- `msg`: Complete message with tool calls, etc.

```go
sm.AddFullMessage("telegram:123456", providers.Message{
    Role:    "assistant",
    Content: "Let me check that.",
    ToolCalls: []providers.ToolCall{
        {ID: "call_123", Name: "files_read", Arguments: map[string]interface{}{"path": "/tmp/file.txt"}},
    },
})
```

### GetHistory

Gets the conversation history for a session.

```go
func (sm *SessionManager) GetHistory(key string) []providers.Message
```

**Parameters:**
- `key`: Session identifier

**Returns:** Copy of message history

```go
history := sm.GetHistory("telegram:123456")
for _, msg := range history {
    fmt.Printf("%s: %s\n", msg.Role, msg.Content)
}
```

### SetHistory

Replaces the entire message history.

```go
func (sm *SessionManager) SetHistory(key string, history []providers.Message)
```

**Parameters:**
- `key`: Session identifier
- `history`: New message history

```go
newHistory := []providers.Message{
    {Role: "system", Content: "You are helpful."},
    {Role: "user", Content: "Hello"},
}
sm.SetHistory("telegram:123456", newHistory)
```

### GetSummary

Gets the session summary.

```go
func (sm *SessionManager) GetSummary(key string) string
```

**Parameters:**
- `key`: Session identifier

**Returns:** Summary string (empty if none)

```go
summary := sm.GetSummary("telegram:123456")
if summary != "" {
    fmt.Println("Previous context:", summary)
}
```

### SetSummary

Sets the session summary.

```go
func (sm *SessionManager) SetSummary(key string, summary string)
```

**Parameters:**
- `key`: Session identifier
- `summary`: Summary text

```go
sm.SetSummary("telegram:123456", "User asked about weather. Discussed forecast for the week.")
```

### TruncateHistory

Truncates history to keep only the last N messages.

```go
func (sm *SessionManager) TruncateHistory(key string, keepLast int)
```

**Parameters:**
- `key`: Session identifier
- `keepLast`: Number of recent messages to keep (0 to clear all)

```go
// Keep only last 4 messages
sm.TruncateHistory("telegram:123456", 4)

// Clear all messages
sm.TruncateHistory("telegram:123456", 0)
```

### Save

Saves a session to disk.

```go
func (sm *SessionManager) Save(key string) error
```

**Parameters:**
- `key`: Session identifier

**Returns:** Error if save failed

**Note:** Only saves if storage path was configured.

```go
if err := sm.Save("telegram:123456"); err != nil {
    log.Printf("Failed to save session: %v", err)
}
```

## Session Keys

Session keys uniquely identify conversations. Common formats:

| Format | Example | Description |
|--------|---------|-------------|
| channel:chatID | `telegram:123456` | Basic session |
| agent:agentID:channel:chatID | `agent:main:telegram:123456` | Agent-specific |

### Key Sanitization

Keys are sanitized for filesystem safety:

```go
// Colon (:) is replaced with underscore (_)
// "telegram:123456" -> file "telegram_123456.json"
```

## File Format

Sessions are stored as JSON files:

```json
{
  "key": "telegram:123456",
  "messages": [
    {
      "role": "user",
      "content": "Hello!"
    },
    {
      "role": "assistant",
      "content": "Hi there!"
    }
  ],
  "summary": "",
  "created": "2024-01-15T10:30:00Z",
  "updated": "2024-01-15T10:31:00Z"
}
```

## Usage Patterns

### Basic Conversation

```go
sm := session.NewSessionManager(storagePath)
sessionKey := "telegram:123456"

// Get or create session
sess := sm.GetOrCreate(sessionKey)

// Add user message
sm.AddMessage(sessionKey, "user", "What's the weather?")

// ... LLM call ...

// Add assistant response
sm.AddMessage(sessionKey, "assistant", "It's sunny today.")

// Save
sm.Save(sessionKey)
```

### With Tool Calls

```go
// Add user message
sm.AddMessage(sessionKey, "user", "Read the config file")

// Add assistant message with tool call
sm.AddFullMessage(sessionKey, providers.Message{
    Role:    "assistant",
    Content: "",
    ToolCalls: []providers.ToolCall{{
        ID:   "call_1",
        Name: "files_read",
        Arguments: map[string]interface{}{"path": "config.json"},
    }},
})

// Add tool result
sm.AddFullMessage(sessionKey, providers.Message{
    Role:       "tool",
    Content:    `{"setting": "value"}`,
    ToolCallID: "call_1",
})

// Add final assistant response
sm.AddMessage(sessionKey, "assistant", "The config contains...")
```

### Summarization

```go
// Check if summarization needed
history := sm.GetHistory(sessionKey)
if len(history) > 20 {
    // Generate summary
    summary := generateSummary(history[:len(history)-4])

    // Set summary
    sm.SetSummary(sessionKey, summary)

    // Keep only recent messages
    sm.TruncateHistory(sessionKey, 4)

    // Save
    sm.Save(sessionKey)
}
```

### Building LLM Context

```go
func buildContext(sm *session.SessionManager, sessionKey, userMessage string) []providers.Message {
    var messages []providers.Message

    // System prompt
    messages = append(messages, providers.Message{
        Role:    "system",
        Content: "You are a helpful assistant.",
    })

    // Add summary if exists
    if summary := sm.GetSummary(sessionKey); summary != "" {
        messages = append(messages, providers.Message{
            Role:    "system",
            Content: "Previous conversation summary: " + summary,
        })
    }

    // Add history
    messages = append(messages, sm.GetHistory(sessionKey)...)

    // Add current message
    messages = append(messages, providers.Message{
        Role:    "user",
        Content: userMessage,
    })

    return messages
}
```

## Thread Safety

The SessionManager is thread-safe:

- All operations are protected by RWMutex
- Multiple goroutines can read simultaneously
- Write operations are exclusive

```go
// Safe to use from multiple goroutines
go func() {
    sm.AddMessage("session1", "user", "Message 1")
}()

go func() {
    sm.AddMessage("session2", "user", "Message 2")
}()
```

## Persistence

### Atomic Writes

Sessions are saved atomically:

1. Write to temporary file
2. Sync to disk
3. Rename to final location

This prevents corruption from crashes.

### Loading

Sessions are loaded lazily on `GetOrCreate`:

1. Check in-memory cache
2. If not found, load from disk
3. Cache in memory

### Storage Location

Typical storage locations:

```
~/.picoclaw/workspace/sessions/
├── telegram_123456.json
├── telegram_-1001234567890.json  # Group chat
├── discord_123456789012345678.json
└── ...
```

## Error Handling

### Save Errors

```go
if err := sm.Save(sessionKey); err != nil {
    if errors.Is(err, os.ErrInvalid) {
        // Invalid session key
    } else {
        // Filesystem error
    }
}
```

### Missing Sessions

```go
// GetHistory returns empty slice for missing sessions
history := sm.GetHistory("nonexistent")
// history == []providers.Message{}

// GetSummary returns empty string for missing sessions
summary := sm.GetSummary("nonexistent")
// summary == ""
```

## Best Practices

1. **Save After Changes**: Save after important operations
2. **Use Meaningful Keys**: Use consistent key formats
3. **Handle Errors**: Check for save errors
4. **Don't Block**: Save in goroutine if needed
5. **Summarize**: Keep history manageable with summaries

### Async Saving

```go
// Save without blocking
go func() {
    if err := sm.Save(sessionKey); err != nil {
        log.Printf("Failed to save session: %v", err)
    }
}()
```

### Periodic Saving

```go
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        for _, key := range getActiveSessions() {
            sm.Save(key)
        }
    }
}()
```

## Performance Characteristics

| Metric | Value |
|--------|-------|
| Memory | O(n) where n = total messages across sessions |
| Read | O(1) for cache hit, O(n) for file load |
| Write | O(1) for memory, O(n) for disk save |
| Thread Safety | Yes (RWMutex) |

## See Also

- [Data Flow](../data-flow.md)
- [Session Manager Implementation](https://github.com/sipeed/picoclaw/tree/main/pkg/session)
- [Provider Types](https://github.com/sipeed/picoclaw/tree/main/pkg/providers/protocoltypes)
