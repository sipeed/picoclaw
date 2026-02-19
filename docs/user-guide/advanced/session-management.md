# Session Management

PicoClaw maintains conversation history through a session management system. Sessions persist across conversations and can be scoped differently based on configuration.

## Overview

The session system provides:

- **Conversation persistence** - Messages are saved and loaded automatically
- **Session scoping** - Control how sessions are shared across agents/channels
- **Identity linking** - Share sessions across different platforms
- **History management** - Automatic summarization for long conversations

## Session Storage

Sessions are stored as JSON files in the workspace:

```
~/.picoclaw/workspace/
└── sessions/
    ├── main.json                    # Main session
    ├── telegram_123456789.json      # Telegram user session
    ├── discord_987654321.json       # Discord user session
    └── agent_assistant_telegram_123456789.json  # Agent-specific session
```

## Session Structure

Each session file contains:

```json
{
  "key": "telegram:123456789",
  "messages": [
    {
      "role": "user",
      "content": "Hello!"
    },
    {
      "role": "assistant",
      "content": "Hi! How can I help you?"
    }
  ],
  "summary": "Previous conversation about project setup...",
  "created": "2024-01-15T10:30:00Z",
  "updated": "2024-01-15T11:45:00Z"
}
```

### Session Fields

| Field | Type | Description |
|-------|------|-------------|
| `key` | string | Unique session identifier |
| `messages` | array | Conversation history |
| `summary` | string | Auto-generated summary of older messages |
| `created` | timestamp | Session creation time |
| `updated` | timestamp | Last update time |

## DM Scope Configuration

Control how direct message sessions are scoped with `session.dm_scope`:

```json
{
  "session": {
    "dm_scope": "per-agent"
  }
}
```

### Scope Options

| Value | Description | Session Key Format |
|-------|-------------|-------------------|
| `main` | Single shared session (default) | `main` |
| `per-agent` | Separate session per agent | `agent:{agent_id}:{channel}:{user_id}` |
| `per-channel` | Separate session per channel | `{channel}:{user_id}` |

### Main Scope

All conversations use a single session:

```json
{
  "session": {
    "dm_scope": "main"
  }
}
```

Best for: Simple single-agent setups

### Per-Agent Scope

Each agent maintains its own conversation history:

```json
{
  "session": {
    "dm_scope": "per-agent"
  }
}
```

Best for: Multi-agent setups where each agent should have independent context

### Per-Channel Scope

Separate sessions for each messaging platform:

```json
{
  "session": {
    "dm_scope": "per-channel"
  }
}
```

Best for: Users who want platform-specific conversations

## Identity Links

Share sessions across different platforms by linking user identities:

```json
{
  "session": {
    "dm_scope": "per-agent",
    "identity_links": {
      "telegram:123456789": ["discord:987654321", "slack:U12345"]
    }
  }
}
```

### How Identity Links Work

1. User sends message from Telegram (ID: 123456789)
2. PicoClaw checks `identity_links` for linked accounts
3. Session key uses canonical identity (first in array)
4. Same user on Discord uses the same session

### Example Configuration

```json
{
  "session": {
    "dm_scope": "per-agent",
    "identity_links": {
      "telegram:123456789": ["discord:987654321"],
      "telegram:987654321": ["slack:U67890", "discord:111222333"]
    }
  }
}
```

## Session Key Construction

Session keys uniquely identify conversations. The format depends on context:

### DM Sessions

```
{scope_prefix}{channel}:{user_id}
```

Examples:
- `main` (main scope)
- `telegram:123456789` (per-channel)
- `agent:assistant:telegram:123456789` (per-agent)

### Group Sessions

```
{channel}:group:{group_id}
```

Example:
- `telegram:group:-1001234567890`

### Channel Sessions

```
{channel}:channel:{channel_id}
```

Example:
- `telegram:channel:@mychannel`

## History Management

### Automatic Summarization

When conversation history grows too long, PicoClaw automatically:

1. Keeps recent messages (last N messages)
2. Generates a summary of older messages
3. Stores the summary in the session

### History Truncation

The session manager provides methods for history management:

- `TruncateHistory(key, keepLast)` - Keep only the last N messages
- `SetSummary(key, summary)` - Store a conversation summary

### Manual History Control

You can control history programmatically or through tools:

```bash
# Clear session history
rm ~/.picoclaw/workspace/sessions/telegram_123456789.json
```

## Session Manager API

The session manager provides these operations:

| Method | Description |
|--------|-------------|
| `GetOrCreate(key)` | Get existing session or create new |
| `AddMessage(key, role, content)` | Add a message to session |
| `AddFullMessage(key, message)` | Add message with tool calls |
| `GetHistory(key)` | Get all messages |
| `SetHistory(key, messages)` | Replace all messages |
| `GetSummary(key)` | Get session summary |
| `SetSummary(key, summary)` | Set session summary |
| `TruncateHistory(key, keepLast)` | Trim message history |
| `Save(key)` | Persist session to disk |

## Session Persistence

### Automatic Saving

Sessions are automatically saved:

- After each message exchange
- When summaries are generated
- On graceful shutdown

### File Format

Sessions use JSON format with indentation for readability:

```json
{
  "key": "telegram:123456789",
  "messages": [...],
  "summary": "...",
  "created": "2024-01-15T10:30:00Z",
  "updated": "2024-01-15T11:45:00Z"
}
```

### Safe File Writing

Sessions use atomic write operations:

1. Write to temporary file
2. Sync to disk
3. Rename to final location

This prevents corruption from crashes or power failures.

## Use Cases

### Cross-Platform Identity

User wants the same conversation on Telegram and Discord:

```json
{
  "session": {
    "dm_scope": "per-agent",
    "identity_links": {
      "telegram:123456789": ["discord:987654321"]
    }
  }
}
```

### Isolated Work Conversations

Keep work Slack separate from personal Telegram:

```json
{
  "session": {
    "dm_scope": "per-channel"
  }
}
```

### Agent-Specific Context

Different agents have different conversation history:

```json
{
  "session": {
    "dm_scope": "per-agent"
  },
  "agents": {
    "list": [
      { "id": "personal", "default": true },
      { "id": "work" }
    ]
  }
}
```

## Troubleshooting

### Session Not Persisting

Check:
1. Workspace directory exists and is writable
2. `sessions/` subdirectory exists
3. No file permission issues

### Wrong Session Used

1. Check `dm_scope` configuration
2. Verify `identity_links` mapping
3. Enable debug mode to see session key construction

### History Too Long

1. Sessions auto-summarize when long
2. Manually clear session file if needed
3. Consider per-agent scope to isolate contexts

## Related Topics

- [Message Routing](routing.md) - How sessions are selected
- [Multi-Agent System](multi-agent.md) - Agent-specific sessions
- [Workspace Management](../workspace/README.md) - Workspace structure
