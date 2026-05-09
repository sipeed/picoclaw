> Back to [README](../../../README.md)

# Telegram

The Telegram channel uses long polling via the Telegram Bot API for bot-based communication. It supports text messages, media attachments (photos, voice, audio, documents), voice transcription ([setup](../../guides/providers.md#voice-transcription)), and built-in command handling.

## Configuration

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
      "allow_from": ["123456789"],
      "proxy": "",
      "use_markdown_v2": false
    }
  }
}
```

| Field            | Type   | Required | Description                                                        |
| ---------------- | ------ | -------- | ------------------------------------------------------------------ |
| enabled          | bool   | Yes      | Whether to enable the Telegram channel                             |
| token            | string | Yes      | Telegram Bot API Token                                             |
| allow_from       | array  | No       | Allowlist of user IDs; empty means all users are allowed           |
| proxy            | string | No       | Proxy URL for connecting to the Telegram API (e.g. http://127.0.0.1:7890) |
| use_markdown_v2 | bool   | No       | Enable Telegram MarkdownV2 formatting                              |
| group_trigger    | object | No       | Group trigger strategy (`mention_only`, `prefixes`, and Telegram forum topic overrides) |

## Setup

1. Search for `@BotFather` in Telegram
2. Send the `/newbot` command and follow the prompts to create a new bot
3. Obtain the HTTP API Token
4. Fill in the Token in the configuration file
5. (Optional) Configure `allow_from` to restrict which user IDs can interact (you can get IDs via `@userinfobot`)

## Group Trigger

By default, the bot responds to every message in allowed group chats. Use
`group_trigger.mention_only` to make it respond only when mentioned:

```json
{
  "channel_list": {
    "telegram": {
      "group_trigger": { "mention_only": true }
    }
  }
}
```

For Telegram supergroups with forum topics, `group_trigger.topics` can override
the group trigger for a specific topic ID. Topic entries replace the channel-wide
trigger for that topic.

This is useful when the bot should stay mention-only in most of a group, but be
active by default in a dedicated topic:

```json
{
  "channel_list": {
    "telegram": {
      "group_trigger": {
        "mention_only": true,
        "topics": {
          "1771": { "mention_only": false }
        }
      }
    }
  }
}
```

You can find a topic ID in Telegram update logs or by inspecting
`message_thread_id` from the Telegram Bot API update payload.

## Built-in Commands

Telegram auto-registers PicoClaw's top-level bot commands at startup, including `/start`, `/help`, `/show`, `/list`, and `/use`.

Skill-related commands:

- `/list skills` lists the installed skills visible to the current agent.
- `/list mcp` lists configured MCP servers and whether they are deferred/connected.
- `/show mcp <server>` lists the active tools for a connected MCP server.
- `/use <skill> <message>` forces a skill for a single request.
- `/use <skill>` arms the skill for your next message in the same chat.
- `/use clear` clears a pending skill override.

Examples:

```text
/list skills
/list mcp
/show mcp github
/use git explain how to squash the last 3 commits
/use git
explain how to squash the last 3 commits
```

## Advanced Formatting

You can set `use_markdown_v2: true` to enable enhanced formatting options. This allows the bot to utilize the full range of Telegram MarkdownV2 features, including nested styles, spoilers, and custom fixed-width blocks.

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "use_markdown_v2": true
    }
  }
}
```
