> Back to [README](../../../README.md)

# Telegram

The Telegram channel uses long polling via the Telegram Bot API for bot-based communication. It supports text messages, media attachments (photos, voice, audio, documents), voice transcription ([setup](../../providers.md#voice-transcription)), and built-in command handling.

## Configuration

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
      "allow_from": ["123456789"],
      "proxy": ""
    }
  }
}
```

| Field      | Type   | Required | Description                                                        |
| ---------- | ------ | -------- | ------------------------------------------------------------------ |
| enabled    | bool   | Yes      | Whether to enable the Telegram channel                             |
| token      | string | Yes      | Telegram Bot API Token                                             |
| allow_from | array  | No       | Allowlist of user IDs; empty means all users are allowed           |
| proxy      | string | No       | Proxy URL for connecting to the Telegram API (e.g. http://127.0.0.1:7890) |

## Setup

1. Search for `@BotFather` in Telegram
2. Send the `/newbot` command and follow the prompts to create a new bot
3. Obtain the HTTP API Token
4. Fill in the Token in the configuration file
5. (Optional) Configure `allow_from` to restrict which user IDs can interact (you can get IDs via `@userinfobot`)
