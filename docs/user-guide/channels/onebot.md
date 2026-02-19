# OneBot Setup

OneBot is a universal chatbot protocol that provides a standardized API for various messaging platforms.

## Prerequisites

- A OneBot-compatible implementation running
- Understanding of which platform you're connecting to (QQ, Telegram, etc.)

## Overview

OneBot (formerly CQHTTP) is a protocol specification that allows bots to communicate with various messaging platforms through a unified API. PicoClaw connects to a OneBot implementation via WebSocket.

## Supported OneBot Implementations

### For QQ

- [go-cqhttp](https://github.com/Mrs4s/go-cqhttp) - Popular Go implementation
- [Lagrange](https://github.com/LagrangeDev/Lagrange.Core) - Modern .NET implementation
- [NapCat](https://github.com/NapNeko/NapCatQQ) - Modern implementation

### For Other Platforms

- Various community implementations exist for WeChat, Telegram, etc.

## Step 1: Install a OneBot Implementation

### go-cqhttp Example

1. Download go-cqhttp from [GitHub Releases](https://github.com/Mrs4s/go-cqhttp/releases)
2. Run it once to generate config files
3. Edit `config.yml`:

```yaml
# config.yml
account:
  uin: YOUR_QQ_NUMBER
  password: ''

servers:
  - ws:
      host: 127.0.0.1
      port: 3001

message:
  post-format: string
```

4. Run go-cqhttp and scan the QR code to log in

## Step 2: Configure PicoClaw

Edit `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "onebot": {
      "enabled": true,
      "ws_url": "ws://127.0.0.1:3001",
      "access_token": "",
      "reconnect_interval": 5,
      "group_trigger_prefix": [],
      "allow_from": []
    }
  }
}
```

| Option | Required | Description |
|--------|----------|-------------|
| `enabled` | Yes | Set to `true` to enable |
| `ws_url` | Yes | WebSocket URL of OneBot implementation |
| `access_token` | No | Access token for authentication |
| `reconnect_interval` | No | Reconnection interval in seconds (default: 5) |
| `group_trigger_prefix` | No | Prefixes to trigger bot in groups (e.g., `["!", "/"]`) |
| `allow_from` | No | Array of allowed user IDs |

## Step 3: Start the Gateway

```bash
picoclaw gateway
```

You should see:
```
Starting OneBot channel ws_url=ws://127.0.0.1:3001
WebSocket connected
OneBot channel started successfully
```

## Step 4: Test

### Private Messages

1. Open your messaging app (e.g., QQ)
2. Send a message to the bot account
3. The bot should respond!

### Group Messages

In group chats, the bot responds when:
- Mentioned with `@BotName`
- Message starts with a configured trigger prefix

## Features

### Voice Transcription

With Groq configured, voice messages are automatically transcribed:

```json
{
  "providers": {
    "groq": {
      "api_key": "gsk_xxx"
    }
  }
}
```

### Group Trigger Prefixes

Configure prefixes to trigger the bot in groups without mentioning:

```json
{
  "channels": {
    "onebot": {
      "enabled": true,
      "ws_url": "ws://127.0.0.1:3001",
      "group_trigger_prefix": ["!", "/bot"]
    }
  }
}
```

With this configuration:
- `!hello` triggers the bot
- `/bot hello` triggers the bot
- `hello` does not trigger the bot (unless mentioned)

### Automatic Reconnection

The bot automatically reconnects if the WebSocket connection is lost.

### Message Replies

The bot replies to messages using the original message's context.

### Emoji Reactions

In group chats, the bot adds emoji reactions to indicate processing status.

## Chat ID Format

PicoClaw uses specific chat ID formats for OneBot:

| Chat Type | Format | Example |
|-----------|--------|---------|
| Private | `private:USER_ID` | `private:123456789` |
| Group | `group:GROUP_ID` | `group:987654321` |

## Troubleshooting

### Connection refused

1. Verify your OneBot implementation is running
2. Check the ws_url is correct
3. Ensure the WebSocket server is listening

### Bot not responding

1. Check OneBot logs for errors
2. Verify the messaging account is logged in
3. Check `allow_from` includes your user ID

### "WebSocket not connected" error

1. Verify OneBot WebSocket server is running
2. Check network connectivity
3. Increase `reconnect_interval` if connection is unstable

### Group messages ignored

1. Use `@BotName` or configure `group_trigger_prefix`
2. Verify the bot has proper permissions in the group

### Voice transcription fails

1. Verify Groq API key is configured
2. Check network allows connections to Groq API
3. Check logs for specific error messages

## Advanced Configuration

### With Access Token

```json
{
  "channels": {
    "onebot": {
      "enabled": true,
      "ws_url": "ws://127.0.0.1:3001",
      "access_token": "your_secure_token",
      "reconnect_interval": 10
    }
  }
}
```

### Allow Specific Users

```json
{
  "channels": {
    "onebot": {
      "enabled": true,
      "ws_url": "ws://127.0.0.1:3001",
      "allow_from": ["123456789", "987654321"]
    }
  }
}
```

## OneBot Configuration Examples

### go-cqhttp

```yaml
# config.yml
account:
  uin: 123456789
  password: ''

servers:
  - ws:
      host: 127.0.0.1
      port: 3001
      middle:
        access-token: your_secure_token

message:
  post-format: array

database:
  leveldb:
    enable: true
```

### Lagrange

```json
{
  "Implementations": [
    {
      "Type": "ForwardWebSocket",
      "Host": "127.0.0.1",
      "Port": 3001,
      "AccessToken": "your_secure_token"
    }
  ]
}
```

## Important Notes

### Account Security

- Use a dedicated account for the bot
- Enable two-factor authentication
- Keep access tokens secure

### Rate Limits

Messaging platforms have rate limits. OneBot implementations typically handle this, but be aware of:
- Message sending frequency
- API call limits
- Group operation limits

### Platform Terms of Service

Using unofficial clients may violate platform terms of service. Use at your own risk.

## See Also

- [Channels Overview](README.md)
- [Gateway Command](../cli/gateway.md)
- [OneBot Specification](https://github.com/botuniverse/onebot)
- [go-cqhttp](https://github.com/Mrs4s/go-cqhttp)
- [Troubleshooting](../../operations/troubleshooting.md)
