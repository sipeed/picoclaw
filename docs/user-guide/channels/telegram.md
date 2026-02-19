# Telegram Setup

Telegram is one of the easiest platforms to set up and is recommended for personal use.

## Prerequisites

- A Telegram account
- Access to @BotFather on Telegram

## Step 1: Create a Bot

1. Open Telegram and search for `@BotFather`
2. Send `/newbot`
3. Follow the prompts:
   - Choose a name for your bot (e.g., "My PicoClaw Bot")
   - Choose a username (must end in `bot`, e.g., `mypicoclaw_bot`)
4. Copy the **bot token** (looks like `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`)

## Step 2: Get Your User ID

1. Search for `@userinfobot` on Telegram
2. Send any message
3. Copy your **User ID** (a number like `123456789`)

## Step 3: Configure PicoClaw

Edit `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
      "allow_from": ["123456789"]
    }
  }
}
```

| Option | Required | Description |
|--------|----------|-------------|
| `enabled` | Yes | Set to `true` to enable |
| `token` | Yes | Bot token from @BotFather |
| `proxy` | No | Proxy URL if needed |
| `allow_from` | No | Array of allowed user IDs |

## Step 4: Start the Gateway

```bash
picoclaw gateway
```

You should see:
```
✓ Channels enabled: telegram
✓ Gateway started on 0.0.0.0:18790
```

## Step 5: Test

1. Open Telegram
2. Find your bot (search for the username)
3. Send a message
4. The bot should respond!

## Features

### Voice Messages

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

### Group Chats

The bot works in group chats:
- Mention the bot with `@your_bot_name message`
- Or reply to the bot's messages

### Commands

Telegram supports bot commands:

| Command | Description |
|---------|-------------|
| `/start` | Start interacting with the bot |
| `/help` | Show help message |
| `/reset` | Reset conversation session |

## Troubleshooting

### Bot not responding

1. Check the gateway is running: `picoclaw gateway`
2. Verify your user ID is in `allow_from`
3. Check the bot token is correct

### "Conflict: terminated by other getUpdates"

This means another instance is using the bot token:
1. Stop other running instances
2. Make sure only one `picoclaw gateway` is running

### Proxy issues

If you need a proxy:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "proxy": "http://127.0.0.1:7890"
    }
  }
}
```

## Advanced Configuration

### Allow Multiple Users

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "allow_from": ["123456789", "987654321", "111222333"]
    }
  }
}
```

### Allow All Users (Not Recommended)

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "allow_from": []
    }
  }
}
```

## See Also

- [Channels Overview](README.md)
- [Gateway Command](../cli/gateway.md)
- [Troubleshooting](../../operations/troubleshooting.md)
