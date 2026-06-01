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

| Feature                 | Description                                                                 |
| ----------------------- | --------------------------------------------------------------------------- |
| **spawn**               | Creates an async task/subagent and records it in the durable task registry  |
| **Independent context** | Subagent has its own context, no session history                            |
| **Delivery mode**       | Completion can be delivered to the user, the parent agent, or both          |
| **Non-blocking**        | After spawning, heartbeat continues to next task                            |
| **Status**              | Use `task_status` for durable status; `spawn_status` is a spawn-only view   |

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
All tasks done            Subagent completes task record
    ↓                           ↓
Respond HEARTBEAT_OK      Delivery coordinator routes result
```

The subagent has access to its configured tools, but completion delivery is owned by the async task delivery path. A terminal background task usually uses user delivery. A compositional task can route the completion back to the parent so the parent can synthesize the final user-facing answer. See [Async Task Delivery](../architecture/async-task-delivery.md) for the registry and delivery model.

## Multi-Step Task Boards

For workflows that call multiple child agents or run several dependent child
steps, use `task_board` to create one durable board, then use the task-board
fields on `spawn` and `delegate`:

- Create one stable `board_id` for the whole workflow.
- Add planned steps with `task_board {"action":"add_step", ...}` when the
  workflow has known steps.
- Pass the same `board_id` to every related `spawn` or `delegate` call.
- Use stable `step_id` values such as `download-media` or `translate-recipe`.
- Use readable `step_title` values for status output.
- Use `depends_on` when one step requires another step to finish first.
- For synchronous `delegate` steps that can block the workflow, set
  `timeout_seconds` explicitly.
- Inspect the workflow with `task_board {"action":"list","board_id":"..."}` or
  `task_status {"board_id":"..."}`.
- Read completed step outputs with `task_board {"action":"results","board_id":"..."}`.

Example:

```json
{
  "action": "create",
  "board_id": "instagram-recipe-20260527",
  "title": "Instagram recipe workflow"
}
```

```json
{
  "action": "add_step",
  "board_id": "instagram-recipe-20260527",
  "step_id": "download-media",
  "step_title": "Download media and caption",
  "owner": "media"
}
```

```json
{
  "agent_id": "media",
  "task": "Download the Instagram Reel and return the local media artifact plus caption.",
  "board_id": "instagram-recipe-20260527",
  "step_id": "download-media",
  "step_title": "Download media and caption"
}
```

```json
{
  "agent_id": "orchestrator",
  "task": "Translate the extracted recipe and prepare the final answer.",
  "board_id": "instagram-recipe-20260527",
  "step_id": "translate-recipe",
  "step_title": "Translate recipe",
  "depends_on": ["download-media"]
}
```

Single-step work does not need explicit board metadata; the registry defaults
`board_id` and `step_id` to the task id.

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
