> Back to [README](../../../README.md)

# Telegram

The Telegram channel uses long polling via the Telegram Bot API for bot-based communication. It supports text messages, media attachments (photos, voice, audio, documents), voice transcription ([setup](../../guides/providers.md#voice-transcription)), built-in command handling, and optional Telegram Business chats.

## Configuration

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
      "allow_from": ["123456789"],
      "settings": {
        "proxy": "",
        "use_markdown_v2": false,
        "business_mode": false,
        "business_owner": "123456789",
        "business_commands_enable": false
      }
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

## Setup

1. Search for `@BotFather` in Telegram
2. Send the `/newbot` command and follow the prompts to create a new bot
3. Obtain the HTTP API Token
4. Fill in the Token in the configuration file
5. (Optional) Configure `allow_from` to restrict which user IDs can interact (you can get IDs via `@userinfobot`)

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

## Telegram Business Mode

Set `settings.business_mode: true` to receive and reply to Telegram Business messages from connected business accounts. Business replies are sent with the incoming `business_connection_id`, and incoming business messages are marked as read when the bot has the `can_read_messages` business right. If marking a message as read fails, PicoClaw still processes the message.

Use `settings.business_owner` to store the Telegram user ID of the business account owner. Business messages from that user are skipped, which prevents the bot from responding to messages you send manually from the connected business account.

By default, bot commands in business chats are ignored. Set `settings.business_commands_enable: true` if you want commands such as `/new`, `/help`, `/show`, `/list`, and `/use` to be handled in Telegram Business chats.

Example:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "allow_from": ["123456789"],
      "settings": {
        "token": "YOUR_BOT_TOKEN",
        "business_mode": true,
        "business_owner": "123456789",
        "business_commands_enable": true
      }
    }
  }
}
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
