# Matrix Channel Configuration Guide

## 1. Example Configuration

Add this to `config.json`:

```json
{
  "channels": {
    "matrix": {
      "enabled": true,
      "homeserver": "https://matrix.org",
      "user_id": "@your-bot:matrix.org",
      "access_token": "YOUR_MATRIX_ACCESS_TOKEN",
      "device_id": "",
      "join_on_invite": true,
      "allow_from": [],
      "group_trigger": {
        "mention_only": true
      },
      "placeholder": {
        "enabled": true,
        "text": "Thinking..."
      },
      "reasoning_channel_id": "",
      "message_format": "richtext"
    }
  }
}
```

## 2. Field Reference

| Field                | Type     | Required | Description |
|----------------------|----------|----------|-------------|
| enabled              | bool     | Yes      | Enable or disable the Matrix channel |
| homeserver           | string   | Yes      | Matrix homeserver URL (for example `https://matrix.org`) |
| user_id              | string   | Yes      | Bot Matrix user ID (for example `@bot:matrix.org`) |
| access_token         | string   | Yes      | Bot access token |
| device_id            | string   | No       | Optional Matrix device ID |
| join_on_invite       | bool     | No       | Auto-join invited rooms |
| allow_from           | []string | No       | User whitelist (Matrix user IDs) |
| group_trigger        | object   | No       | Group trigger strategy (`mention_only` / `prefixes`) |
| placeholder          | object   | No       | Placeholder message config |
| reasoning_channel_id | string   | No       | Target channel for reasoning output |
| message_format       | string   | No       | Output format: `"richtext"` (default) renders markdown as HTML; `"plain"` sends plain text only |

## 3. Currently Supported

- Text message send/receive with markdown rendering (bold, italic, headers, code blocks, etc.)
- Configurable message format (`richtext` / `plain`)
- Incoming image/audio/video/file download (MediaStore first, local path fallback)
- Incoming audio normalization into existing transcription flow (`[audio: ...]`)
- Outgoing image/audio/video/file upload and send
- Group trigger rules (including mention-only mode)
- Typing state (`m.typing`)
- Placeholder message + final reply replacement
- Auto-join invited rooms (can be disabled)

## 4. TODO

- Rich media metadata improvements (for example image/video size and thumbnails)
