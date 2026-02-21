# QQ Setup

QQ is a popular instant messaging platform in China. PicoClaw connects via the QQ Official Bot API.

## Prerequisites

- A QQ account
- Access to [QQ Open Platform](https://bot.q.qq.com/)

## Step 1: Create a QQ Bot

1. Go to [QQ Bot Platform](https://bot.q.qq.com/)
2. Log in with your QQ account
3. Click **"Create Bot"** (创建机器人)
4. Fill in the bot information:
   - Bot name
   - Bot description
   - Bot avatar
5. Submit for review (approval may take 1-3 days)

## Step 2: Get App Credentials

After approval:

1. Go to your bot's dashboard
2. Find **"App ID"** (AppID)
3. Find **"App Secret"** (AppSecret)
4. Copy both values

## Step 3: Configure Bot Settings

1. In the bot dashboard, go to **"Intents"** (事件订阅)
2. Enable these intents:
   - **Private messages** (C2C消息)
   - **Group @ messages** (群@消息)
3. Save settings

## Step 4: Configure PicoClaw

Edit `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "qq": {
      "enabled": true,
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

| Option | Required | Description |
|--------|----------|-------------|
| `enabled` | Yes | Set to `true` to enable |
| `app_id` | Yes | App ID from QQ Bot Platform |
| `app_secret` | Yes | App Secret from QQ Bot Platform |
| `allow_from` | No | Array of allowed user IDs |

## Step 5: Start the Gateway

```bash
picoclaw gateway
```

You should see:
```
Starting QQ bot (WebSocket mode)
QQ bot started successfully
```

## Step 6: Test

### Private Messages

1. Open QQ
2. Search for your bot by name
3. Send a friend request if needed
4. Send a message to the bot

### Group Messages

1. Add the bot to a QQ group
2. Mention the bot: `@YourBotName hello`
3. The bot should respond!

## Features

### Private Messages (C2C)

Direct messages to the bot are processed automatically.

### Group @ Mentions

In group chats, the bot responds only when:
- The bot is mentioned with `@BotName`
- The message is a reply to the bot

### Automatic Token Refresh

The bot automatically refreshes access tokens as needed.

## Troubleshooting

### Bot not responding

1. Verify app_id and app_secret are correct
2. Check that intents are enabled in the QQ Bot Platform
3. Ensure the gateway is running
4. Check firewall allows WebSocket connections

### "Authentication failed" error

1. Verify app_id and app_secret are correct
2. Check if the bot is approved and active
3. Ensure no extra whitespace in credentials

### Group messages ignored

1. Ensure you're using `@BotName` to mention the bot
2. Check that "Group @ messages" intent is enabled
3. Verify the bot has proper permissions in the group

### WebSocket connection issues

1. Check your network allows WebSocket connections to QQ servers
2. Verify no firewall is blocking outgoing connections
3. Check logs for specific error messages

## Advanced Configuration

### Allow Specific Users

```json
{
  "channels": {
    "qq": {
      "enabled": true,
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": ["123456789", "987654321"]
    }
  }
}
```

### Multiple Guilds/Servers

The bot can be added to multiple QQ guilds. Use `allow_from` to control access across all of them.

## Important Notes

### Bot Approval

QQ bots require approval before they can be used publicly. During development, you can test with your own account.

### Rate Limits

QQ has rate limits on API calls. If you experience issues with high-volume usage, consider:
- Reducing message frequency
- Implementing message batching
- Contacting QQ for higher limits

### Regional Restrictions

QQ Bot Platform may have restrictions based on:
- Your account region
- Bot content type
- Target audience

## See Also

- [Channels Overview](README.md)
- [Gateway Command](../cli/gateway.md)
- [QQ Bot Documentation](https://bot.q.qq.com/wiki/)
- [Troubleshooting](../../operations/troubleshooting.md)
