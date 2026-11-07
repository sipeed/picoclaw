# 🔄 Spawn & Async Tasks

> Back to [README](../README.md)

PicoClaw supports **asynchronous task execution** via the `spawn` tool. This is primarily used by the **Heartbeat** system to run long-running tasks without blocking the main agent loop.

## Heartbeat

The heartbeat system periodically checks `workspace/HEARTBEAT.md` for scheduled tasks. On first run, a default template is auto-generated. You can customize it to define quick tasks (handled inline) and long tasks (delegated via `spawn`).

**Example `HEARTBEAT.md`:**

```markdown
## Quick Tasks (respond directly)

- Report current time

## Long Tasks (use spawn for async)

- Search the web for AI news and summarize
- Check email and report important messages
```

**Key behaviors:**

| Feature                 | Description                                               |
| ----------------------- | --------------------------------------------------------- |
| **spawn**               | Creates async subagent, doesn't block heartbeat           |
| **Independent context** | Subagent has its own context, no session history          |
| **message tool**        | Subagent communicates with user directly via message tool |
| **Non-blocking**        | After spawning, heartbeat continues to next task          |

### `allow_direct_message` parameter

The `spawn` tool accepts an optional `allow_direct_message` boolean parameter (default: `true`) to control whether the spawned subagent can use the `message` tool to send messages directly to the user during execution:

| Value | Behavior |
|-------|----------|
| `true` (default) | Subagent can directly `message` the user during execution (existing behavior) |
| `false` | The `message` tool is removed from the subagent's toolset; it cannot directly message users. Results must be returned to the main agent for presentation. |

**Example:**

```json
{
  "task": "Search for AI news and summarize",
  "allow_direct_message": false
}
```

Use `false` when you want to keep conversation flow centralized and have the main agent present all results to the user.

#### How Subagent Communication Works

```
Heartbeat triggers
    ↓
Agent reads HEARTBEAT.md
    ↓
For long task: spawn subagent
    ↓                           ↓
Continue to next task      Subagent works independently
    ↓                           ↓
All tasks done            Subagent uses "message" tool
    ↓                           ↓
Respond HEARTBEAT_OK      User receives result directly
```

The subagent has access to tools (message, web_search, etc.) and can communicate with the user independently without going through the main agent.

**Configuration:**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| Option     | Default | Description                        |
| ---------- | ------- | ---------------------------------- |
| `enabled`  | `true`  | Enable/disable heartbeat           |
| `interval` | `30`    | Check interval in minutes (min: 5) |

**Environment variables:**

* `PICOCLAW_HEARTBEAT_ENABLED=false` to disable
* `PICOCLAW_HEARTBEAT_INTERVAL=60` to change interval
