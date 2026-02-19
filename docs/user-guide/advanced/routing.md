# Message Routing

Message routing determines which agent handles incoming messages from different channels. PicoClaw uses a flexible binding system with a priority cascade for precise control.

## Overview

The routing system:

1. Receives messages from channels (Telegram, Discord, Slack, etc.)
2. Evaluates bindings to find a matching agent
3. Routes the message to the appropriate agent
4. Constructs the session key for conversation persistence

## Binding Configuration

Bindings are defined in the `bindings` array:

```json
{
  "bindings": [
    {
      "agent_id": "personal",
      "match": {
        "channel": "telegram",
        "peer": { "kind": "user", "id": "123456789" }
      }
    },
    {
      "agent_id": "work",
      "match": {
        "channel": "slack",
        "team_id": "T12345"
      }
    }
  ]
}
```

## Binding Properties

| Property | Type | Description |
|----------|------|-------------|
| `agent_id` | string | Target agent ID |
| `match.channel` | string | Channel name (telegram, discord, slack, etc.) |
| `match.account_id` | string | Account filter (use `*` for wildcard) |
| `match.peer.kind` | string | Peer type: `user`, `group`, `channel` |
| `match.peer.id` | string | Peer/user ID |
| `match.guild_id` | string | Discord guild ID |
| `match.team_id` | string | Slack team ID |

## Priority Cascade

Bindings are evaluated in this priority order (highest to lowest):

1. **Peer binding** - Direct user/group match
2. **Parent peer binding** - Parent group for forwarded messages
3. **Guild binding** - Discord server match
4. **Team binding** - Slack workspace match
5. **Account binding** - Specific account match
6. **Channel wildcard** - Any message from channel (`account_id: "*"`)
7. **Default agent** - Fallback when no binding matches

## Match Examples

### Telegram User

Route messages from a specific Telegram user:

```json
{
  "agent_id": "personal",
  "match": {
    "channel": "telegram",
    "peer": { "kind": "user", "id": "123456789" }
  }
}
```

### Telegram Group

Route all messages from a Telegram group:

```json
{
  "agent_id": "team-bot",
  "match": {
    "channel": "telegram",
    "peer": { "kind": "group", "id": "-1001234567890" }
  }
}
```

### Discord Guild

Route all messages from a Discord server:

```json
{
  "agent_id": "community-bot",
  "match": {
    "channel": "discord",
    "guild_id": "123456789012345678"
  }
}
```

### Slack Team

Route all messages from a Slack workspace:

```json
{
  "agent_id": "work",
  "match": {
    "channel": "slack",
    "team_id": "T12345"
  }
}
```

### Channel Wildcard

Route all Telegram messages to an agent:

```json
{
  "agent_id": "telegram-bot",
  "match": {
    "channel": "telegram",
    "account_id": "*"
  }
}
```

### Multi-Account

Route from a specific bot account:

```json
{
  "agent_id": "bot-alpha",
  "match": {
    "channel": "telegram",
    "account_id": "bot_123456"
  }
}
```

## Session Key Construction

The routing system constructs session keys for conversation persistence. The session key format depends on the DM scope configuration:

### Default (Main Scope)

All conversations share one session:

```
main
```

### Per-Agent Scope

Separate sessions per agent:

```
agent:assistant:telegram:123456789
```

### Per-Channel Scope

Separate sessions per channel:

```
telegram:123456789
```

See [Session Management](session-management.md) for details.

## Routing Flow

```
Incoming Message
       │
       ▼
┌─────────────────┐
│ Filter by       │
│ Channel         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Check Peer      │──► Match? ──► Use Agent
│ Binding         │
└────────┬────────┘
         │ No
         ▼
┌─────────────────┐
│ Check Parent    │──► Match? ──► Use Agent
│ Peer Binding    │
└────────┬────────┘
         │ No
         ▼
┌─────────────────┐
│ Check Guild/    │──► Match? ──► Use Agent
│ Team Binding    │
└────────┬────────┘
         │ No
         ▼
┌─────────────────┐
│ Check Account   │──► Match? ──► Use Agent
│ Binding         │
└────────┬────────┘
         │ No
         ▼
┌─────────────────┐
│ Check Channel   │──► Match? ──► Use Agent
│ Wildcard        │
└────────┬────────┘
         │ No
         ▼
┌─────────────────┐
│ Use Default     │
│ Agent           │
└─────────────────┘
```

## Use Cases

### Personal Assistant with Work Separation

```json
{
  "bindings": [
    {
      "agent_id": "personal",
      "match": {
        "channel": "telegram",
        "peer": { "kind": "user", "id": "123456789" }
      }
    },
    {
      "agent_id": "work",
      "match": {
        "channel": "slack",
        "team_id": "T12345"
      }
    },
    {
      "agent_id": "work",
      "match": {
        "channel": "discord",
        "guild_id": "987654321"
      }
    }
  ]
}
```

### Team Bot for Group Chats

```json
{
  "bindings": [
    {
      "agent_id": "team-assistant",
      "match": {
        "channel": "telegram",
        "peer": { "kind": "group", "id": "-1001234567890" }
      }
    },
    {
      "agent_id": "team-assistant",
      "match": {
        "channel": "discord",
        "guild_id": "123456789012345678"
      }
    }
  ]
}
```

### Multiple Discord Servers

```json
{
  "bindings": [
    {
      "agent_id": "community-bot",
      "match": {
        "channel": "discord",
        "guild_id": "111111111111111111"
      }
    },
    {
      "agent_id": "support-bot",
      "match": {
        "channel": "discord",
        "guild_id": "222222222222222222"
      }
    }
  ]
}
```

### Channel-Specific Agents

```json
{
  "bindings": [
    {
      "agent_id": "telegram-bot",
      "match": {
        "channel": "telegram",
        "account_id": "*"
      }
    },
    {
      "agent_id": "discord-bot",
      "match": {
        "channel": "discord",
        "account_id": "*"
      }
    }
  ]
}
```

## Debugging Routing

Enable debug mode to see routing decisions:

```bash
picoclaw gateway --debug
```

Debug output includes:

- Which bindings were evaluated
- Which binding matched (if any)
- Final agent selection
- Session key construction

## Related Topics

- [Multi-Agent System](multi-agent.md) - Define multiple agents
- [Session Management](session-management.md) - Understand session persistence
- [Channels](../channels/README.md) - Configure chat platform integrations
