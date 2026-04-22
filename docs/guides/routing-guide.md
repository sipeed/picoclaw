# Routing Guide

> Back to [README](../README.md)

In PicoClaw, routing has two user-facing parts:

- **agent routing**: choose which agent should handle a message
- **model routing**: choose whether a turn should use the primary model or the configured light model

This guide explains how to configure both for real deployments.

## Quick Start

### Route one Telegram group to a support agent

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
          "name": "telegram support group",
          "agent": "support",
          "when": {
            "channel": "telegram",
            "chat": "group:-1001234567890"
          }
        }
      ]
    }
  }
}
```

### Route only Slack mentions in one workspace

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
          "name": "slack mentions",
          "agent": "support",
          "when": {
            "channel": "slack",
            "space": "workspace:t001",
            "mentioned": true
          }
        }
      ]
    }
  }
}
```

### Use a light model for simple turns

```json
{
  "model_list": [
    {
      "model_name": "gpt-main",
      "provider": "openai",
      "model": "gpt-5.4",
      "api_keys": ["sk-main"]
    },
    {
      "model_name": "flash-light",
      "provider": "gemini",
      "model": "gemini-2.0-flash-exp",
      "api_keys": ["sk-light"]
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "gpt-main",
      "routing": {
        "enabled": true,
        "light_model": "flash-light",
        "threshold": 0.35
      }
    }
  }
}
```

## Agent Routing

Agent routing is configured with:

```text
agents.dispatch.rules
```

Rules are evaluated from top to bottom.
The **first matching rule wins**.
If no rule matches, PicoClaw falls back to the default agent.

## Supported Match Fields

| Field | Meaning | Example |
| --- | --- | --- |
| `channel` | Channel name | `telegram`, `slack`, `discord` |
| `account` | Normalized account ID | `default`, `bot2` |
| `space` | Workspace, guild, or similar container | `workspace:t001`, `guild:123456` |
| `chat` | Direct chat, group, or channel | `direct:user123`, `group:-100123`, `channel:c123` |
| `topic` | Thread or topic | `topic:42` |
| `sender` | Normalized sender identity | `12345`, `john` |
| `mentioned` | Whether the bot was explicitly mentioned | `true` |

Values must match the normalized runtime shape, not the raw incoming payload.

## Rule Ordering

Put more specific rules before broader rules.

Good:

1. VIP sender inside one group
2. all traffic for that group
3. channel-wide fallback

Bad:

1. all traffic for that group
2. VIP sender inside the same group

In the bad ordering, the broad rule wins first and the VIP rule never runs.

## Session Interaction

Routing and sessions are related but different.

- routing decides which agent handles the message
- session settings decide which messages share memory

You can override the global `session.dimensions` value for one matched rule with `session_dimensions`.

Example:

```json
{
  "agents": {
    "list": [
      { "id": "main", "default": true },
      { "id": "support" },
      { "id": "sales" }
    ],
    "dispatch": {
      "rules": [
        {
          "name": "vip in support group",
          "agent": "sales",
          "when": {
            "channel": "telegram",
            "chat": "group:-1001234567890",
            "sender": "12345"
          },
          "session_dimensions": ["chat", "sender"]
        },
        {
          "name": "support group",
          "agent": "support",
          "when": {
            "channel": "telegram",
            "chat": "group:-1001234567890"
          },
          "session_dimensions": ["chat"]
        }
      ]
    }
  },
  "session": {
    "dimensions": ["chat"]
  }
}
```

In this configuration:

- the VIP gets routed to `sales`
- everyone else in the group goes to `support`
- the VIP route also gets per-user session isolation

## Identity Links

`session.identity_links` also affects routing when you match on `sender`.
Use it when the same real user may appear under multiple raw sender IDs.

Example:

```json
{
  "session": {
    "identity_links": {
      "john": ["slack:u123", "legacy-user-42"]
    }
  },
  "agents": {
    "dispatch": {
      "rules": [
        {
          "name": "john goes to sales",
          "agent": "sales",
          "when": {
            "sender": "john"
          }
        }
      ]
    }
  }
}
```

## Model Routing

Model routing is configured under:

```text
agents.defaults.routing
```

Current fields:

| Field | Meaning |
| --- | --- |
| `enabled` | Turn model routing on or off |
| `light_model` | `model_name` from `model_list` used for simple turns |
| `threshold` | Complexity cutoff in `[0, 1]` |

Important behavior:

- the light model must exist in `model_list`
- PicoClaw resolves the light model at startup; if it is invalid, routing is disabled
- one turn stays on one model tier, even if it later calls tools

## What Affects The Complexity Score

The current model router looks at structural signals such as:

- message length
- fenced code blocks
- recent tool calls in the same session
- conversation depth
- media or attachments

This means a "simple" turn may still go to the primary model if it includes:

- code
- images or audio
- a very long prompt
- a tool-heavy ongoing workflow

## Choosing A Threshold

Recommended starting point:

```json
{
  "agents": {
    "defaults": {
      "routing": {
        "enabled": true,
        "light_model": "flash-light",
        "threshold": 0.35
      }
    }
  }
}
```

General rule:

- lower threshold: use the primary model more often
- higher threshold: use the light model more aggressively

Practical suggestions:

- `0.25` if you want safer routing with fewer light-model turns
- `0.35` as the default starting point
- `0.50+` only if your light model is already strong enough for most chat traffic

## Troubleshooting

### A rule is not matching

Check:

- rule order
- normalized value shape such as `group:-100123` instead of just `-100123`
- whether the channel actually provides `space`, `topic`, or `mentioned`

### The wrong agent handles a message

The most common cause is ordering.
Remember: first match wins.

### The light model is never used

Check:

- `agents.defaults.routing.enabled` is `true`
- `light_model` exists in `model_list`
- the light model can actually initialize
- your threshold is not too low

### The primary model is still chosen for short messages

That can still happen when the turn includes:

- a code block
- media or attachments
- recent tool-heavy history

### Routing works, but the conversation memory is still too shared

Adjust `session.dimensions` globally or `session_dimensions` on the specific route.
Routing chooses the agent, but sessions decide context sharing.

## Related Guides

- [Session Guide](session-guide.md)
- [Configuration Guide](configuration.md)
- [Providers & Model Configuration](providers.md)
