# DingTalk Setup

DingTalk (钉钉) is a popular enterprise messaging and collaboration platform in China.

## Prerequisites

- A DingTalk account
- Admin access to a DingTalk organization (or ability to create one)

## Step 1: Create a DingTalk Application

1. Go to [DingTalk Developer Console](https://open-dev.dingtalk.com/)
2. Log in with your DingTalk account
3. Click **"Create Application"** (创建应用)
4. Select **"Enterprise Internal Application"** (企业内部应用)
5. Fill in the application information:
   - Application name (e.g., "PicoClaw Bot")
   - Application description
   - Application logo
6. Click **"Confirm"**

## Step 2: Get Client Credentials

1. In your application dashboard, go to **"Application Information"** (应用信息)
2. Copy the **"Client ID"** (ClientID)
3. Copy the **"Client Secret"** (ClientSecret)

## Step 3: Configure Bot Capabilities

1. Go to **"Robot & Message Push"** (机器人与消息推送)
2. Enable the robot capability
3. Configure message receiving:
   - Enable **"Stream Mode"** (Stream Mode / 流式模式)
   - This allows real-time message receiving via WebSocket

## Step 4: Set Permissions

1. Go to **"Permission Management"** (权限管理)
2. Request these permissions:
   - **"Chat with users"** (与企业内人员聊天)
   - **"Send messages"** (发送消息)
3. Wait for approval if required

## Step 5: Configure PicoClaw

Edit `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

| Option | Required | Description |
|--------|----------|-------------|
| `enabled` | Yes | Set to `true` to enable |
| `client_id` | Yes | Client ID from DingTalk Developer Console |
| `client_secret` | Yes | Client Secret from DingTalk Developer Console |
| `allow_from` | No | Array of allowed user IDs |

## Step 6: Start the Gateway

```bash
picoclaw gateway
```

You should see:
```
Starting DingTalk channel (Stream Mode)...
DingTalk channel started (Stream Mode)
```

## Step 7: Test

### Single Chat

1. Open DingTalk
2. Find your bot in the organization
3. Send a message to the bot
4. The bot should respond!

### Group Chat

1. Add the bot to a DingTalk group
2. Mention the bot or send a message
3. The bot should respond!

## Features

### Stream Mode

PicoClaw uses DingTalk's Stream Mode for real-time message receiving via WebSocket. This provides:

- Instant message delivery
- No need for a public webhook URL
- Automatic reconnection

### Markdown Support

The bot sends responses in Markdown format with a title header.

### Group Chat Support

The bot works in both:
- One-on-one conversations
- Group conversations

## Troubleshooting

### Bot not responding

1. Verify client_id and client_secret are correct
2. Check that Stream Mode is enabled
3. Ensure permissions are approved
4. Check the gateway is running

### "Authentication failed" error

1. Verify client_id and client_secret are correct
2. Check if the application is published/active
3. Ensure no extra whitespace in credentials

### Stream connection issues

1. Check your network allows WebSocket connections
2. Verify firewall allows outgoing connections to DingTalk servers
3. Check logs for specific error messages

### Permission denied errors

1. Go to DingTalk Developer Console
2. Verify all required permissions are granted
3. Request additional permissions if needed

## Advanced Configuration

### Allow Specific Users

```json
{
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": ["user123", "user456"]
    }
  }
}
```

### Multi-Tenant Setup

For multiple organizations, create separate applications and use separate configurations.

## Important Notes

### Enterprise Application

DingTalk bots require an enterprise application. Personal bots are not supported.

### Approval Process

Some permissions require admin approval within your DingTalk organization.

### Rate Limits

DingTalk has rate limits on API calls. For high-volume usage, consider:
- Implementing message batching
- Contacting DingTalk for higher limits

## See Also

- [Channels Overview](README.md)
- [Gateway Command](../cli/gateway.md)
- [DingTalk Developer Documentation](https://open.dingtalk.com/document/)
- [Troubleshooting](../../operations/troubleshooting.md)
