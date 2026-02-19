# Channels Overview

PicoClaw supports multiple chat platforms, allowing you to interact with your AI assistant through your preferred messaging app.

## Supported Channels

| Channel | Setup Difficulty | Features |
|---------|------------------|----------|
| [Telegram](telegram.md) | Easy | Full support, voice transcription |
| [Discord](discord.md) | Easy | Full support, voice transcription |
| [Slack](slack.md) | Medium | Full support, voice transcription |
| [LINE](line.md) | Medium | Full support, webhooks |
| [QQ](qq.md) | Easy | Basic support |
| [DingTalk](dingtalk.md) | Medium | Basic support |
| [WhatsApp](whatsapp.md) | Medium | Requires bridge |
| [Feishu/Lark](feishu-lark.md) | Medium | Enterprise features |
| [OneBot](onebot.md) | Medium | Universal protocol |
| [MaixCam](maixcam.md) | Easy | Hardware device |

## Quick Setup

### 1. Choose a Channel

Select the platform you want to use.

### 2. Get Credentials

Each channel requires different credentials:

- **Telegram**: Bot token from @BotFather
- **Discord**: Bot token from Developer Portal
- **Slack**: Bot token and App token
- **LINE**: Channel secret and access token

### 3. Configure

Add credentials to `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    }
  }
}
```

### 4. Start Gateway

```bash
picoclaw gateway
```

## Common Options

All channels support:

| Option | Description |
|--------|-------------|
| `enabled` | Enable/disable the channel |
| `allow_from` | Whitelist of allowed user IDs (empty = all users) |

## Access Control

Use `allow_from` to restrict who can interact with your bot:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "allow_from": ["123456789", "987654321"]
    }
  }
}
```

An empty `allow_from` allows all users.

## Voice Transcription

With Groq configured, voice messages are automatically transcribed on:
- Telegram
- Discord
- Slack
- OneBot

```json
{
  "providers": {
    "groq": {
      "api_key": "gsk_xxx"
    }
  }
}
```

## Multi-Channel Setup

You can enable multiple channels simultaneously:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "telegram-token"
    },
    "discord": {
      "enabled": true,
      "token": "discord-token"
    },
    "slack": {
      "enabled": true,
      "bot_token": "xoxb-token",
      "app_token": "xapp-token"
    }
  }
}
```

## Channel-Specific Guides

- [Telegram Setup](telegram.md)
- [Discord Setup](discord.md)
- [Slack Setup](slack.md)
- [LINE Setup](line.md)
- [QQ Setup](qq.md)
- [DingTalk Setup](dingtalk.md)
- [WhatsApp Setup](whatsapp.md)
- [Feishu/Lark Setup](feishu-lark.md)
- [OneBot Setup](onebot.md)
- [MaixCam Setup](maixcam.md)
