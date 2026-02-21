# Heartbeat Tasks

HEARTBEAT.md defines periodic tasks that the agent executes automatically.

## Location

```
~/.picoclaw/workspace/HEARTBEAT.md
```

## Configuration

Configure heartbeat in `~/.picoclaw/config.json`:

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `true` | Enable/disable heartbeat |
| `interval` | `30` | Check interval in minutes (min: 5) |

## How It Works

1. Agent reads HEARTBEAT.md every N minutes
2. Executes the tasks listed
3. Can respond directly or spawn subagents

## Example Content

```markdown
# Periodic Tasks

## Quick Checks (respond directly)

- Check current time and date
- Verify system status

## Research Tasks (use spawn for async)

- Search for AI news and summarize key developments
- Check weather forecast for tomorrow

## Reminders

- Remind me to take a break every 30 minutes
- Check if I have any upcoming meetings
```

## Task Types

### Direct Tasks

Simple tasks the agent handles immediately:

```markdown
- Report the current time
- Summarize your capabilities
```

### Async Tasks (spawn)

Long-running tasks using subagents:

```markdown
- Use spawn to search for tech news and summarize
- Use spawn to check email for important messages
```

Using `spawn` prevents blocking the heartbeat cycle.

## Communication

### Responding to User

For direct responses:

```markdown
- Send a message with current stock prices for AAPL, GOOGL
- Remind me about my 3 PM meeting
```

### Silent Tasks

Tasks that don't need user notification:

```markdown
- Clean up temporary files in workspace (silent)
- Log system status (silent)
```

## Environment Variables

```bash
export PICOCLAW_HEARTBEAT_ENABLED=true
export PICOCLAW_HEARTBEAT_INTERVAL=30
```

## Example Scenarios

### Daily Summary

```markdown
# Periodic Tasks

Every 4 hours, provide a summary of:
- Current time and date
- Weather forecast
- Any scheduled reminders
```

### News Monitor

```markdown
# Periodic Tasks

Every 30 minutes:
- Use spawn to search for "AI news" and summarize 3 top stories
- If important news found, send me a message
```

### Health Check

```markdown
# Periodic Tasks

Every 5 minutes:
- Check system resources (memory, CPU)
- Alert if usage > 80%
```

## Disabling Heartbeat

```json
{
  "heartbeat": {
    "enabled": false
  }
}
```

Or via environment:

```bash
export PICOCLAW_HEARTBEAT_ENABLED=false
```

## See Also

- [Cron Tool](../tools/cron.md)
- [Spawn Tool](../tools/spawn.md)
- [Workspace Structure](structure.md)
