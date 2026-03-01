# Audit Logging Package

The `audit` package provides comprehensive audit logging for PicoClaw, capturing a complete trail of bot activity for debugging, compliance, security analysis, and operational monitoring.

## Features

- **Structured JSON Logging**: Machine-parseable format for analysis
- **Event Filtering**: Configurable per-event-type filtering
- **Async Write**: Non-blocking with buffered channel (1000 entries)
- **Log Rotation**: Size-based and daily rotation with compression
- **Context Propagation**: Request tracing via context
- **Secure**: File permissions 0600 (owner read/write only)

## Quick Start

```go
import "github.com/sipeed/picoclaw/pkg/audit"

// Initialize
err := audit.InitGlobal(cfg.Audit, workspace)
if err != nil {
    log.Fatal(err)
}
defer audit.CloseGlobal()

// Log events
ctx := audit.WithRequestID(context.Background(), "req-123")
audit.LogSystem(ctx, "operation", map[string]interface{}{"key": "value"})
```

## Configuration

```json
{
  "audit": {
    "enabled": true,
    "location": "workspace/logs",
    "format": "json",
    "rotation": {
      "max_size_mb": 100,
      "max_age_days": 30,
      "max_backups": 10,
      "compress": true
    },
    "events": {
      "tool_calls": true,
      "messages": true,
      "errors": true,
      "system": false
    }
  }
}
```

## Event Types

### Tool Calls
Logged when tools are executed:
```json
{
  "timestamp": "2025-03-01T12:00:00Z",
  "level": "INFO",
  "component": "tool",
  "event_type": "tool_call",
  "request_id": "req-123",
  "tool_call": {
    "tool_id": "read_file",
    "name": "read_file",
    "arguments": {"path": "/tmp/test.txt"},
    "is_error": false,
    "is_async": false
  },
  "duration_ms": 150
}
```

### Messages
Logged for inbound/outbound messages:
```json
{
  "timestamp": "2025-03-01T12:00:00Z",
  "level": "INFO",
  "component": "channel",
  "event_type": "message",
  "request_id": "req-123",
  "channel": "telegram",
  "chat_id": "123456",
  "message": {
    "direction": "inbound",
    "content_type": "text",
    "content": "Hello bot"
  }
}
```

### Errors
Logged for failures:
```json
{
  "timestamp": "2025-03-01T12:00:00Z",
  "level": "ERROR",
  "component": "system",
  "event_type": "error",
  "error": {
    "error_type": "send_failed",
    "message": "connection timeout",
    "recoverable": true
  }
}
```

## Log Rotation

Files are named: `audit-DDMMYYYY.log[.N][.gz]`

Rotation triggers:
- **Daily**: New file at midnight
- **Size**: When file exceeds `max_size_mb`
- **Cleanup**: Files deleted after `max_age_days` or exceeding `max_backups`
- **Compression**: Old files gzip-compressed if `compress: true`

## Request Tracing

Use context to correlate events:

```go
// At request entry
ctx := audit.WithRequestID(context.Background(), generateID())
ctx = audit.WithSessionID(ctx, sessionKey)
ctx = audit.WithAgentID(ctx, agentID)

// Pass ctx through call chain
// All logged events will include these IDs
```

## Nil Safety

All logger methods are safe to call on nil:

```go
var logger *audit.Logger // nil
logger.Log(entry) // No panic, no-op
```

## Performance

- Async write (background worker)
- 1000-entry buffer (drops if full)
- Batch processing
- Minimal allocation

## Security

- Log files created with 0600 permissions
- Arguments masked for sensitive tools
- No passwords/tokens logged
- Automatic cleanup prevents disk exhaustion
