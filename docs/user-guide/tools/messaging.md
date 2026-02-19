# Messaging Tool

Send messages to users through configured channels.

## Tool

### message

Send a message to a specific channel and recipient.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `channel` | string | Yes | Channel name (telegram, discord, etc.) |
| `chat_id` | string | Yes | Chat/user ID |
| `content` | string | Yes | Message content |

**Example:**

```json
{
  "channel": "telegram",
  "chat_id": "123456789",
  "content": "Task completed successfully!"
}
```

## Use Cases

### Notifications

Send notifications from scheduled tasks:

```markdown
# HEARTBEAT.md

- Every hour, check server status
- If issues found, send message via telegram
```

### Subagent Responses

Subagents use the message tool to communicate results:

```
Subagent: [Uses message tool]
{
  "channel": "telegram",
  "chat_id": "123456789",
  "content": "Research complete! Here's the summary..."
}
```

### Alerts

Send alerts based on conditions:

```
Agent: "I found an important update. Sending notification..."
[Uses message tool]
```

## Channel Support

| Channel | Supports message tool |
|---------|----------------------|
| Telegram | Yes |
| Discord | Yes |
| Slack | Yes |
| LINE | Yes |
| QQ | Yes |
| DingTalk | Yes |
| OneBot | Yes |

## Finding Chat IDs

### Telegram

1. Send a message to your bot
2. Visit: `https://api.telegram.org/bot<TOKEN>/getUpdates`
3. Find `chat.id` in the response

Or use @userinfobot to get your user ID.

### Discord

1. Enable Developer Mode in Discord settings
2. Right-click a channel or user
3. Select "Copy ID"

### Slack

User IDs start with `U`, channel IDs start with `C`.

## Examples

```
User (via Telegram): "Let me know when the build finishes"

Agent spawns subagent for async task...

Subagent uses message tool:
{
  "channel": "telegram",
  "chat_id": "123456789",
  "content": "Build finished successfully! All tests passed."
}
```

## Security

The message tool can only send to:
- Channels that are enabled
- Chat IDs that have interacted with the bot

## See Also

- [Spawn Tool](spawn.md)
- [Channels Overview](../channels/README.md)
