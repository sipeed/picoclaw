# Feishu/Lark Setup

Feishu (飞书, known as Lark internationally) is an enterprise collaboration platform by ByteDance.

## Prerequisites

- A Feishu/Lark account
- Admin access to a Feishu/Lark organization (or ability to create one)

## Step 1: Create a Feishu App

1. Go to [Feishu Open Platform](https://open.feishu.cn/) (China) or [Lark Developer](https://open.larksuite.com/) (International)
2. Log in with your Feishu/Lark account
3. Click **"Create Custom App"** (创建企业自建应用)
4. Fill in the application information:
   - App name (e.g., "PicoClaw Bot")
   - App description
   - App icon
5. Click **"Create"**

## Step 2: Get App Credentials

1. In your app dashboard, go to **"Credentials & Basic Info"** (凭证与基础信息)
2. Copy the **"App ID"**
3. Copy the **"App Secret"**

## Step 3: Configure Event Subscription

1. Go to **"Event Subscriptions"** (事件订阅)
2. Enable event subscription
3. Configure encryption (optional but recommended):
   - Copy the **"Verification Token"** (验证令牌)
   - Copy the **"Encrypt Key"** (加密密钥)
4. Add events to subscribe:
   - **"Receive message"** (接收消息) - `im.message.receive_v1`

## Step 4: Set Permissions

1. Go to **"Permission Management"** (权限管理)
2. Request these permissions:
   - `im:message` - Get and send messages
   - `im:message:send_as_bot` - Send messages as bot
   - `im:chat` - Get chat information
3. Wait for approval if required

## Step 5: Publish App

1. Go to **"Version Management & Release"** (版本管理与发布)
2. Create a version
3. Submit for review
4. After approval, publish to your organization

## Step 6: Configure PicoClaw

Edit `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxxxxxxxxxxx",
      "app_secret": "xxxxxxxxxxxxxxxxxxxx",
      "verification_token": "xxxxxxxx",
      "encrypt_key": "xxxxxxxx",
      "allow_from": []
    }
  }
}
```

| Option | Required | Description |
|--------|----------|-------------|
| `enabled` | Yes | Set to `true` to enable |
| `app_id` | Yes | App ID from Feishu Open Platform |
| `app_secret` | Yes | App Secret from Feishu Open Platform |
| `verification_token` | No | Verification token for event validation |
| `encrypt_key` | No | Encrypt key for event decryption |
| `allow_from` | No | Array of allowed user IDs |

## Step 7: Start the Gateway

```bash
picoclaw gateway
```

You should see:
```
Feishu channel started (websocket mode)
```

## Step 8: Test

1. Open Feishu/Lark
2. Find your bot in the organization
3. Send a message to the bot
4. The bot should respond!

## Features

### WebSocket Mode

PicoClaw uses Feishu's WebSocket connection for real-time event delivery. Benefits include:

- No need for a public webhook URL
- Instant message delivery
- Automatic reconnection

### Text Messages

The bot processes text messages and responds accordingly.

### Chat Types

The bot works in various chat types:
- One-on-one conversations (P2P)
- Group chats
- Topic-based discussions

## Troubleshooting

### Bot not responding

1. Verify app_id and app_secret are correct
2. Check that events are subscribed
3. Ensure the app is published
4. Check the gateway is running

### "Authentication failed" error

1. Verify app_id and app_secret are correct
2. Check if the app is approved and active
3. Ensure no extra whitespace in credentials

### WebSocket connection issues

1. Check your network allows WebSocket connections
2. Verify firewall allows connections to Feishu servers
3. Check logs for specific error messages

### Events not received

1. Verify event subscription is enabled
2. Check that the correct events are subscribed
3. Ensure the app has necessary permissions

### Permission errors

1. Go to Feishu Open Platform
2. Verify all required permissions are granted
3. Request additional permissions if needed

## Advanced Configuration

### Allow Specific Users

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxxxxxxxxxxx",
      "app_secret": "xxxxxxxxxxxxxxxxxxxx",
      "verification_token": "xxxxxxxx",
      "encrypt_key": "xxxxxxxx",
      "allow_from": ["ou_xxxxx", "ou_yyyyy"]
    }
  }
}
```

### Multi-Tenant Setup

For multiple organizations, create separate apps and use separate configurations.

## Regional Differences

### Feishu (China)

- Platform: https://open.feishu.cn/
- Use for users in mainland China
- May require ICP filing for some features

### Lark (International)

- Platform: https://open.larksuite.com/
- Use for international users
- Different data centers and API endpoints

## Important Notes

### App Review

Internal enterprise apps typically don't require review, but publishing to the app store does.

### Rate Limits

Feishu has rate limits on API calls. For high-volume usage:
- Implement message batching
- Consider caching frequent queries
- Contact Feishu for higher limits

### Security

- Keep app_secret secure
- Use encrypt_key for sensitive data
- Implement proper user authorization

## See Also

- [Channels Overview](README.md)
- [Gateway Command](../cli/gateway.md)
- [Feishu Open Platform](https://open.feishu.cn/)
- [Lark Developer Documentation](https://open.larksuite.com/document/)
- [Troubleshooting](../../operations/troubleshooting.md)
