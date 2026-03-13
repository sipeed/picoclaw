# Telegram

The Telegram Channel uses the Telegram Bot API with long polling to enable bot-based communication. It supports text messages, media attachments (photos, voice, audio, documents), voice transcription via Groq Whisper, and a built-in command processor.

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

| Field      | Type   | Required | Description                                               |
| ---------- | ------ | -------- | --------------------------------------------------------- |
| enabled    | bool   | Yes      | Whether to enable the Telegram channel                    |
| token      | string | Yes      | Telegram Bot API Token                                    |
| allow_from | array  | No       | User ID allowlist, empty means allow all                  |
| proxy      | string | No       | Proxy URL for connecting to Telegram API (e.g., http://127.0.0.1:7890)|

## Setup Process

1. Search for `@BotFather` on Telegram.
2. Send the `/newbot` command and follow the prompts to create a new bot.
3. Obtain the HTTP API Token.
4. Fill in the Token in the configuration file.
5. (Optional) Configure `allow_from` to restrict the user IDs allowed to interact (obtain IDs via `@userinfobot`).
