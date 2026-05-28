# Session Guide

> Back to [README](../README.md)

PicoClaw sessions decide which messages share the same conversation history.
If your bot "remembers too much" or "forgets too much", the first thing to check is the session configuration.

This guide is for users configuring session behavior in `config.json`.
For implementation details, see the architecture docs instead.

## What Sessions Control

A session controls:

- which previous messages are visible to the agent
- when summarization starts for that conversation
- whether two users in the same group share context
- whether different chats, threads, or spaces stay isolated

Session data is stored under your workspace, typically:

```text
~/.picoclaw/workspace/sessions/
```

## Quick Start

### Default: one context per chat

This is the default and is the right choice for most bots.

```json
{
  "session": {
    "dimensions": ["chat"]
  }
}
```

Use this when:

- each group/channel should have its own shared memory
- each direct message should have its own separate memory

### Separate each user inside a group

If users in the same group should not share memory, add `sender`:

```json
{
  "session": {
    "dimensions": ["chat", "sender"]
  }
}
```

Use this when:

- one shared assistant sits in a busy group
- each user should keep a private thread of context even inside the same room

### Share one context across multiple rooms in the same workspace or guild

If your channel exposes a `space` value, you can route by workspace or guild instead of by room:

```json
{
  "session": {
    "dimensions": ["space"]
  }
}
```

Use this when:

- a Slack workspace assistant should share context across channels
- a Discord guild assistant should share context across channels

### Split by thread or forum topic

If your channel exposes `topic`, you can isolate per thread:

```json
{
  "session": {
    "dimensions": ["chat", "topic"]
  }
}
```

Use this when:

- each forum topic should keep its own history
- each threaded discussion should stay separate

## Available Dimensions

| Dimension | What it means | Good for |
| --- | --- | --- |
| `space` | Workspace, guild, or similar top-level container | One shared assistant across many rooms |
| `chat` | Direct chat, group, or channel | Default per-room isolation |
| `topic` | Thread, topic, or forum sub-channel | Keep threaded discussions separate |
| `sender` | The message sender after normalization | Per-user context inside shared rooms |

Not every channel provides every field.
If a channel does not supply `space` or `topic`, those dimensions simply have no effect for that message.

## Important Behavior

### Sessions are always separated by agent

Even if two agents receive messages from the same chat, they do not share one session.

### Sessions are still separated by channel and account

`session.dimensions` adds finer-grained isolation, but PicoClaw still keeps a baseline separation by:

- agent
- channel
- account

That means an empty or very small `dimensions` list does **not** create one global memory across every platform.

### Telegram forum topics already stay isolated in the default `chat` mode

Telegram forum messages keep topic isolation by default even when `dimensions` only contains `chat`.
You usually do not need a special workaround for Telegram forums.

### Summaries happen per session

`summarize_message_threshold` and `summarize_token_percent` apply inside each session independently.
If you create smaller sessions, summarization also happens on smaller per-session histories.

## Common Recipes

### One shared assistant per group or direct chat

```json
{
  "session": {
    "dimensions": ["chat"]
  }
}
```

### One context per user inside each chat

```json
{
  "session": {
    "dimensions": ["chat", "sender"]
  }
}
```

### One context per sender across one workspace or guild

```json
{
  "session": {
    "dimensions": ["space", "sender"]
  }
}
```

This is useful for workspace-wide assistants where each user should keep their own memory while moving across rooms in the same workspace.

### Use a different session policy for one routed agent only

You can keep the global default and override it for one dispatch rule:

```json
{
  "agents": {
    "list": [
      { "id": "main", "default": true },
      { "id": "support" }
    ],
    "dispatch": {
      "rules": [
        {
          "name": "support group",
          "agent": "support",
          "when": {
            "channel": "telegram",
            "chat": "group:-1001234567890"
          },
          "session_dimensions": ["chat", "sender"]
        }
      ]
    }
  },
  "session": {
    "dimensions": ["chat"]
  }
}
```

In this example:

- most traffic uses one shared context per chat
- the support group uses one context per user inside that chat

## Identity Links

`session.identity_links` helps when the same user may appear under multiple raw sender IDs and you want PicoClaw to treat them as one sender identity.

Example:

```json
{
  "session": {
    "dimensions": ["chat", "sender"],
    "identity_links": {
      "john": ["slack:u123", "u123", "legacy-user-42"]
    }
  }
}
```

This is mainly useful for:

- migrated sender IDs
- platform-specific ID aliases
- cleanup after changing channel adapters or account naming

Current limitation:

- `identity_links` does not make one user share memory across different channels automatically
- channel and account remain part of the baseline session scope

## Troubleshooting

### Users in one group are sharing memory

Your current session is probably keyed only by `chat`.
Switch to:

```json
{
  "session": {
    "dimensions": ["chat", "sender"]
  }
}
```

### The same user does not share memory across Slack and Telegram

That is expected.
PicoClaw still separates sessions by channel even if you use `sender`.

### Threads are mixing together

Add `topic` when the channel provides one:

```json
{
  "session": {
    "dimensions": ["chat", "topic"]
  }
}
```

### Old sessions seem to use legacy keys

That is normal during migration.
PicoClaw keeps compatibility with older `agent:...` session keys while moving runtime storage to opaque canonical keys.

## Related Guides

- [Configuration Guide](configuration.md)
- [Routing Guide](routing-guide.md)
- [Providers & Model Configuration](providers.md)
