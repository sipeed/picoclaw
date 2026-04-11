# VK (VKontakte)

The VK channel uses Bots Long Poll API for bot-based communication with VK social network. It supports text messages, media attachments (photos, videos, audio, documents, stickers), and group chat interactions.

## Configuration

```json
{
  "channel_list": {
    "vk": {
      "enabled": true,
      "type": "vk",
      "token": "NOT_HERE",
      "group_id": 123456789,
      "allow_from": ["123456789"],
      "group_trigger": {
        "mention_only": false,
        "prefixes": ["/bot", "!bot"]
      }
    }
  }
}
```

| Field            | Type   | Required | Description                                                        |
| ---------------- | ------ | -------- | ------------------------------------------------------------------ |
| enabled          | bool   | Yes      | Whether to enable the VK channel                                   |
| token            | string | Yes      | Set to `NOT_HERE` - token is stored securely (see Token Storage)   |
| group_id         | int    | Yes      | VK Community ID (Group ID)                                         |
| allow_from       | array  | No       | Allowlist of user IDs; empty means all users are allowed           |
| group_trigger    | object | No       | Configuration for group chat triggers                              |

### Token Storage

For security reasons, the VK access token should not be stored directly in the configuration file. Instead:

1. Set `token` to `"NOT_HERE"` in the configuration
2. Store the actual token using one of these methods:
   - **Environment variable**: Set `PICOCLAW_CHANNELS_VK_TOKEN` environment variable
   - **Secure storage**: Use PicoClaw's secure token storage mechanism

Example using environment variable:
```bash
export PICOCLAW_CHANNELS_VK_TOKEN="vk1.a.abc123..."
```

### Group Trigger Configuration

| Field        | Type     | Description                                                        |
| ------------ | -------- | ------------------------------------------------------------------ |
| mention_only | bool     | Only respond when bot is mentioned in group chats                  |
| prefixes     | []string | List of prefixes that trigger bot response in group chats          |

## Setup

### 1. Create a VK Community

1. Go to [VK](https://vk.com) and log in
2. Create a new community or use an existing one
3. Note your Community ID (found in the community URL, e.g., `public123456789`)

### 2. Enable Messages

1. Go to your community page
2. Click "Manage" → "Messages" → "Community Messages"
3. Enable community messages

### 3. Create Access Token

1. Go to "Manage" → "API usage" → "Access tokens"
2. Click "Create token"
3. Select the following permissions:
   - `messages` - Access to messages
   - `photos` - Access to photos (optional)
   - `docs` - Access to documents (optional)
4. Copy the generated access token
5. Store the token securely (see Token Storage section below)

### 4. Configure PicoClaw

1. Add the token to your PicoClaw configuration
2. Set the `group_id` to your community ID (numeric value)
3. (Optional) Configure `allow_from` to restrict which user IDs can interact

## Features

### Supported Message Types

- **Text messages**: Full support for text messages
- **Photos**: Photos are displayed as `[photo]` placeholder
- **Videos**: Videos are displayed as `[video]` placeholder
- **Audio**: Audio files are displayed as `[audio]` placeholder
- **Voice messages**: Voice messages are displayed as `[voice]` placeholder and support transcription
- **Documents**: Documents are displayed as `[document: filename]`
- **Stickers**: Stickers are displayed as `[sticker]` placeholder

### Voice Support

The VK channel supports both voice message reception and text-to-speech capabilities:

- **ASR (Automatic Speech Recognition)**: Voice messages can be transcribed to text using configured voice models
- **TTS (Text-to-Speech)**: Text responses can be converted to voice messages

To enable voice transcription, configure a voice model in your providers setup. See [Voice Transcription](../../providers.md#voice-transcription) for details.

### Group Chat Support

The VK channel supports group chats with configurable triggers:

- **Mention-only mode**: Bot only responds when mentioned
- **Prefix mode**: Bot responds to messages starting with specified prefixes
- **Permissive mode**: Bot responds to all messages (default)

### Message Length

VK has a maximum message length of 4000 characters. PicoClaw automatically splits longer messages into multiple parts.

## Example Configuration

### Basic Configuration

```json
{
  "channel_list": {
    "vk": {
      "enabled": true,
      "type": "vk",
      "token": "NOT_HERE",
      "group_id": 123456789
    }
  }
}
```

### With User Whitelist

```json
{
  "channel_list": {
    "vk": {
      "enabled": true,
      "type": "vk",
      "token": "NOT_HERE",
      "group_id": 123456789,
      "allow_from": ["123456789", "987654321"]
    }
  }
}
```

### With Group Chat Triggers

```json
{
  "channel_list": {
    "vk": {
      "enabled": true,
      "type": "vk",
      "token": "NOT_HERE",
      "group_id": 123456789,
      "group_trigger": {
        "prefixes": ["/bot", "!bot"]
      }
    }
  }
}
```

## Troubleshooting

### Bot Not Responding

1. Check that the access token is valid
2. Verify that the `group_id` is correct
3. Ensure the user ID is in `allow_from` if configured
4. Check PicoClaw logs for error messages

### Permission Errors

Make sure the access token has the necessary permissions:
- `messages` - Required for sending and receiving messages
- `photos` - Optional, for handling photo attachments
- `docs` - Optional, for handling document attachments

### Group Chat Issues

If the bot doesn't respond in group chats:
1. Check `group_trigger` configuration
2. Try using a prefix to trigger the bot
3. Check if the bot has permission to read group messages

## API Reference

The VK channel uses the [VK SDK for Go](https://github.com/SevereCloud/vksdk) library, which supports VK API version 5.199.

For more information about VK API, see:
- [VK API Documentation](https://dev.vk.com/en)
- [VK Bots Long Poll API](https://dev.vk.com/en/api/bots-long-poll/getting-started)
