# Discord Setup

Discord is great for community servers and personal use.

## Prerequisites

- A Discord account
- Admin access to a Discord server (or ability to create one)

## Step 1: Create a Discord Application

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Click **"New Application"**
3. Give it a name (e.g., "PicoClaw Bot")
4. Click **"Create"**

## Step 2: Create a Bot

1. In your application, go to **"Bot"** in the left sidebar
2. Click **"Add Bot"**
3. Confirm by clicking **"Yes, do it!"**

## Step 3: Get Bot Token

1. In the Bot settings, find **"Token"**
2. Click **"Reset Token"** if needed
3. Copy the token (starts with `MTk4Nj...`)

## Step 4: Enable Intents

In the Bot settings, enable:

- **MESSAGE CONTENT INTENT** (required)
- **SERVER MEMBERS INTENT** (optional, for member-based features)

Click **"Save Changes"**.

## Step 5: Get Your User ID

1. Open Discord
2. Go to **Settings** → **Advanced**
3. Enable **"Developer Mode"**
4. Right-click your avatar in any chat
5. Select **"Copy User ID"**

## Step 6: Configure PicoClaw

Edit `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token": "MTk4NjIyNDgzNDc...",
      "allow_from": ["123456789012345678"]
    }
  }
}
```

| Option | Required | Description |
|--------|----------|-------------|
| `enabled` | Yes | Set to `true` to enable |
| `token` | Yes | Bot token |
| `allow_from` | No | Array of allowed user IDs |

## Step 7: Invite the Bot

1. In the Developer Portal, go to **"OAuth2"** → **"URL Generator"**
2. Under **Scopes**, check `bot`
3. Under **Bot Permissions**, check:
   - `Send Messages`
   - `Read Message History`
   - (Optional) `Attach Files`
4. Copy the generated URL
5. Open the URL in your browser
6. Select your server and authorize the bot

## Step 8: Start the Gateway

```bash
picoclaw gateway
```

You should see:
```
✓ Channels enabled: discord
✓ Gateway started on 0.0.0.0:18790
```

## Step 9: Test

1. Go to your Discord server
2. Find the channel where the bot is present
3. Send a message
4. The bot should respond!

## Features

### Voice Messages

With Groq configured, voice messages are automatically transcribed.

### Server Channels

The bot works in any channel it has access to:
- Public channels
- Private channels (if bot has permission)

### Direct Messages

Users can DM the bot directly for private conversations.

## Troubleshooting

### Bot not responding

1. Check intents are enabled (especially MESSAGE CONTENT INTENT)
2. Verify the bot token is correct
3. Check `allow_from` includes your user ID

### "Missing Access" error

1. Ensure the bot has proper permissions in the channel
2. Re-invite the bot with correct permissions

### Bot appears offline

1. Make sure the gateway is running
2. Check for errors in the console output

## Advanced Configuration

### Multiple Servers

The bot can be invited to multiple servers. Use `allow_from` to control access.

### Role-Based Access

Use `allow_from` with user IDs to restrict who can use the bot.

## See Also

- [Channels Overview](README.md)
- [Gateway Command](../cli/gateway.md)
- [Troubleshooting](../../operations/troubleshooting.md)
