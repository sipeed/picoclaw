# Telegram Bot Tutorial

This tutorial guides you through creating a Telegram chatbot with PicoClaw.

## Prerequisites

- 15 minutes
- PicoClaw installed and configured
- A Telegram account
- A configured LLM provider (see [Basic Assistant](basic-assistant.md))

## Step 1: Create a Telegram Bot

### Talk to BotFather

1. Open Telegram
2. Search for `@BotFather`
3. Start a conversation

### Create New Bot

Send `/newbot` to BotFather:

```
You: /newbot
BotFather: Alright, a new bot. How are we going to call it?
           Please choose a name for your bot.

You: My PicoClaw Assistant
BotFather: Good. Now let's choose a username for your bot.
           It must end in `bot`. Like this, for example: TetrisBot or tetris_bot.

You: my_picoclaw_assistant_bot
BotFather: Done! Congratulations on your new bot...
           Use this token to access the HTTP API:
           7123456789:AAHxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
           Keep your token secure...
```

### Save the Token

Copy and save the bot token (the long string starting with numbers). You'll need it for configuration.

## Step 2: Get Your User ID

### Use userinfobot

1. Search for `@userinfobot`
2. Start a conversation
3. It will reply with your user ID

```
You: /start
UserInfoBot: Your user ID: 123456789
```

Save this ID for the allow list.

## Step 3: Configure PicoClaw

### Edit Configuration

```bash
nano ~/.picoclaw/config.json
```

Add Telegram channel configuration:

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "model": "anthropic/claude-opus-4-5"
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "7123456789:AAHxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "allow_from": [123456789]
    }
  }
}
```

Replace:
- `7123456789:AAH...` with your bot token
- `123456789` with your user ID

### Configuration Options

| Option | Required | Description |
|--------|----------|-------------|
| `enabled` | Yes | Enable the channel |
| `token` | Yes | Bot token from BotFather |
| `allow_from` | Recommended | List of allowed user IDs |

## Step 4: Start the Gateway

The gateway connects PicoClaw to Telegram:

```bash
picoclaw gateway
```

Output:

```
2024-01-15T10:30:00Z INFO main Starting PicoClaw gateway...
2024-01-15T10:30:00Z INFO channel Connecting to Telegram...
2024-01-15T10:30:01Z INFO channel Telegram connected as @my_picoclaw_assistant_bot
2024-01-15T10:30:01Z INFO main Gateway started on port 18790
```

## Step 5: Test Your Bot

### Send a Message

1. Open Telegram
2. Find your bot
3. Send `/start`
4. Send any message

```
You: Hello!
Bot: Hello! I'm your PicoClaw assistant. How can I help you today?

You: What time is it?
Bot: [Uses exec tool]
     The current time is 10:35 AM.

You: Search for the weather
Bot: [Uses web_search tool]
     Based on my search, the weather today is...
```

## Step 6: Debug Mode

If something isn't working, run with debug output:

```bash
picoclaw gateway --debug
```

This shows:
- Incoming messages
- LLM requests
- Tool calls
- Outgoing responses

## Step 7: Customize Bot Behavior

### Edit AGENT.md

Customize how your bot responds:

```bash
nano ~/.picoclaw/workspace/AGENT.md
```

```markdown
# Telegram Bot Behavior

You are a helpful Telegram assistant.

## Response Style
- Keep messages concise (Telegram users prefer brief responses)
- Use emoji sparingly
- Format long responses in paragraphs

## Capabilities
- Answer questions
- Search the web
- Manage files

## Commands
Respond helpfully to these commands:
- /start - Greet the user
- /help - Show available features
- /status - Show system status
```

### Restart Gateway

After editing:

```bash
# Stop the gateway (Ctrl+C)
# Start again
picoclaw gateway
```

## Step 8: Add More Users

### Get Their User IDs

Have other users message `@userinfobot` to get their IDs.

### Update Allow List

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "your-bot-token",
      "allow_from": [123456789, 987654321, 555555555]
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
      "token": "your-bot-token",
      "allow_from": []
    }
  }
}
```

Warning: An empty `allow_from` allows anyone to use your bot and consume your API quota.

## Step 9: Group Chats

To use in Telegram groups:

### Get Group ID

1. Add `@RawDataBot` to your group
2. Send any message
3. It will reply with the group ID (negative number)

### Configure

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "your-bot-token",
      "allow_from": [123456789, -1001234567890]
    }
  }
}
```

### Privacy Mode

By default, bots only see:
- Commands (messages starting with `/`)
- Messages where the bot is mentioned
- Replies to the bot's messages

To receive all messages, disable privacy mode:

1. Talk to @BotFather
2. Send `/setprivacy`
3. Select your bot
4. Choose "Disable"

## Step 10: Set Bot Commands

Define commands in BotFather:

1. Send `/setcommands` to @BotFather
2. Select your bot
3. Paste:

```
start - Start the bot
help - Show help
status - Check system status
clear - Clear conversation history
```

Now users will see command suggestions when typing `/`.

## Step 11: Run as Background Service

### Using Systemd (Linux)

Create a service file:

```bash
sudo nano /etc/systemd/system/picoclaw.service
```

```ini
[Unit]
Description=PicoClaw Gateway
After=network.target

[Service]
Type=simple
User=your-username
WorkingDirectory=/home/your-username
ExecStart=/usr/local/bin/picoclaw gateway
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable picoclaw
sudo systemctl start picoclaw
```

Check status:

```bash
sudo systemctl status picoclaw
```

### Using Docker

```bash
docker run -d \
  --name picoclaw \
  -v ~/.picoclaw:/root/.picoclaw \
  picoclaw:latest gateway
```

## Troubleshooting

### Bot Not Responding

1. Check gateway is running:
   ```bash
   picoclaw gateway --debug
   ```

2. Verify token is correct:
   ```bash
   curl https://api.telegram.org/botYOUR_TOKEN/getMe
   ```

3. Check user ID is in allow list

### Conflict Error

```
Conflict: terminated by other getUpdates request
```

Only one instance can use the bot token:
1. Stop other instances
2. Wait a few seconds
3. Restart gateway

### Rate Limits

Telegram has rate limits:
- 30 messages/second to different users
- 1 message/second to same user

PicoClaw handles rate limiting automatically.

## Advanced Features

### Voice Messages

Enable voice transcription with Groq:

```json
{
  "providers": {
    "groq": {
      "api_key": "your-groq-api-key"
    }
  },
  "agents": {
    "defaults": {
      "voice_transcription": {
        "enabled": true,
        "provider": "groq"
      }
    }
  }
}
```

### Webhook Mode (Production)

For production deployments, use webhooks instead of polling:

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "your-token",
      "webhook": {
        "enabled": true,
        "url": "https://your-domain.com/telegram/webhook",
        "port": 8443
      }
    }
  }
}
```

## Next Steps

Now that you have a working Telegram bot:

- [Scheduled Tasks Tutorial](scheduled-tasks.md) - Automate periodic tasks
- [Multi-Agent Tutorial](multi-agent-setup.md) - Specialized agents
- [Telegram Channel Docs](../user-guide/channels/telegram.md) - Full configuration

## Summary

You learned:
- How to create a Telegram bot with BotFather
- How to configure the Telegram channel
- How to run the gateway
- How to customize bot behavior
- How to deploy as a service

Your PicoClaw assistant is now available on Telegram!
