# Spawn Tool

Create subagents to handle background tasks asynchronously.

## Tool

### spawn

Spawn a subagent to execute a task independently.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `prompt` | string | Yes | Task for the subagent |
| `agent_id` | string | No | Specific agent to use |

**Example:**

```json
{
  "prompt": "Search for the latest AI news and summarize the top 3 stories"
}
```

## How Spawn Works

1. Main agent spawns a subagent with a task
2. Main agent continues immediately (non-blocking)
3. Subagent works independently with its own context
4. Subagent uses `message` tool to report results
5. User receives the result directly

## Use Cases

### Long-Running Tasks

Don't block the main conversation:

```
User: "Search for information about quantum computing and write a summary"

Agent spawns subagent:
{
  "prompt": "Research quantum computing thoroughly and send a comprehensive summary to the user via message tool"
}

Agent: "I've started researching quantum computing. I'll send you the summary when complete."
```

### Heartbeat Tasks

For periodic tasks that might take time:

```markdown
# HEARTBEAT.md

## Use spawn for long tasks

- Spawn to search AI news and send summary
- Spawn to check weather and alert if rain expected
```

### Parallel Tasks

Multiple tasks at once:

```
User: "Check stock prices for AAPL, GOOGL, and MSFT"

Agent spawns multiple subagents for parallel processing.
```

## Subagent Capabilities

Subagents have access to:
- All tools (file, web, exec, etc.)
- Message tool to communicate with user
- Same workspace restrictions

Subagents do NOT have:
- Access to main conversation history
- Ability to spawn more subagents (by default)

## Configuration

### Allowed Subagents

Restrict which agents can be spawned:

```json
{
  "agents": {
    "list": [
      {
        "id": "main",
        "subagents": {
          "allow_agents": ["researcher", "coder"]
        }
      }
    ]
  }
}
```

### Subagent Model

Use a different model for subagents:

```json
{
  "agents": {
    "list": [
      {
        "id": "main",
        "subagents": {
          "model": "gpt-4o-mini"
        }
      }
    ]
  }
}
```

## Example Flow

```
1. User: "Research AI trends and report back"

2. Main agent spawns subagent with prompt

3. Main agent responds: "Research started. I'll update you when done."

4. User continues chatting with main agent

5. Subagent completes research

6. Subagent uses message tool to send results

7. User receives: "AI Trends Report: [comprehensive summary]"
```

## See Also

- [Messaging Tool](messaging.md)
- [Heartbeat Tasks](../workspace/heartbeat-tasks.md)
- [Multi-Agent System](../advanced/multi-agent.md)
