# LINE Setup

LINE is a popular messaging app in Japan, Taiwan, Thailand, and other Asian countries.

## Prerequisites

- A LINE account
- A LINE Official Account (free tier available)
- A server accessible from the internet (for webhooks)

## Step 1: Create a LINE Official Account

1. Go to [LINE Developers Console](https://developers.line.biz/)
2. Log in with your LINE account
3. Create a new provider (or select existing)
4. Create a new channel:
   - Select **"Messaging API"**
   - Fill in the required information
   - Click **"Create"**

## Step 2: Get Channel Credentials

1. In your channel settings, go to **"Messaging API"** tab
2. Copy the **"Channel secret"**
3. Click **"Issue"** next to **"Channel access token"**
4. Copy the **"Channel access token"**

## Step 3: Configure Webhook

LINE requires a webhook URL to receive messages. You have two options:

### Option A: Use a Tunnel (Development)

Use a service like ngrok or cloudflared:

```bash
# Using ngrok
ngrok http 18791

# Using cloudflared
cloudflared tunnel --url http://localhost:18791
```

Note the HTTPS URL (e.g., `https://abc123.ngrok.io`).

### Option B: Use a Public Server

Deploy PicoClaw on a server with a public IP and configure your domain.

## Step 4: Configure PicoClaw

Edit `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_host": "0.0.0.0",
      "webhook_port": 18791,
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

| Option | Required | Description |
|--------|----------|-------------|
| `enabled` | Yes | Set to `true` to enable |
| `channel_secret` | Yes | Channel secret from LINE Developers Console |
| `channel_access_token` | Yes | Channel access token from LINE Developers Console |
| `webhook_host` | No | Host to bind webhook server (default: `0.0.0.0`) |
| `webhook_port` | No | Port for webhook server (default: `18791`) |
| `webhook_path` | No | Path for webhook endpoint (default: `/webhook/line`) |
| `allow_from` | No | Array of allowed user IDs |

## Step 5: Start the Gateway

```bash
picoclaw gateway
```

You should see:
```
LINE webhook server listening addr=0.0.0.0:18791 path=/webhook/line
LINE channel started (Webhook Mode)
```

## Step 6: Register Webhook URL

1. Go back to [LINE Developers Console](https://developers.line.biz/)
2. In your channel settings, go to **"Messaging API"** tab
3. Set the **"Webhook URL"**:
   - For ngrok: `https://abc123.ngrok.io/webhook/line`
4. Click **"Verify"** to test the connection
5. Enable **"Use webhook"**

## Step 7: Test

1. Open LINE on your phone
2. Add your Official Account as a friend (scan QR code or search by ID)
3. Send a message to the bot
4. The bot should respond!

## Features

### Reply vs Push API

LINE has two message sending methods:

- **Reply API**: Free, valid for ~25 seconds after receiving a message
- **Push API**: Requires paid plan, sends messages proactively

PicoClaw automatically uses Reply API when possible and falls back to Push API.

### Group Chats

In group chats, the bot only responds when:
- Mentioned with `@BotName`
- Or @mentioned via LINE's mention feature

### Loading Indicator

PicoClaw sends a loading indicator while processing messages.

### Media Support

The bot can receive images, audio, and video files.

## Troubleshooting

### Webhook verification fails

1. Ensure the gateway is running
2. Verify the webhook URL is correct
3. Check firewall allows incoming connections
4. Verify `channel_secret` is correct

### Bot not responding

1. Check webhook is enabled in LINE Developers Console
2. Verify `channel_access_token` is valid
3. Check gateway logs for errors

### "Invalid signature" error

1. Verify `channel_secret` is correct
2. Ensure no extra whitespace in the secret

### Messages sent but no response

1. Check if Reply API timed out (> 25 seconds)
2. Consider upgrading to a paid plan for Push API

## Advanced Configuration

### Custom Webhook Path

```json
{
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/my-custom-webhook"
    }
  }
}
```

### Allow Specific Users

```json
{
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "allow_from": ["U1234567890abcdef", "U0987654321fedcba"]
    }
  }
}
```

## See Also

- [Channels Overview](README.md)
- [Gateway Command](../cli/gateway.md)
- [LINE Messaging API Documentation](https://developers.line.biz/en/docs/messaging-api/)
- [Troubleshooting](../../operations/troubleshooting.md)
