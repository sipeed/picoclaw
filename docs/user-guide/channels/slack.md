# Slack Setup

Slack is ideal for team collaboration and workplace automation.

## Prerequisites

- A Slack workspace with admin access (or ability to create apps)
- A Slack account

## Step 1: Create a Slack App

1. Go to [Slack API: Applications](https://api.slack.com/apps)
2. Click **"Create New App"**
3. Choose **"From scratch"**
4. Give it a name (e.g., "PicoClaw Bot")
5. Select your workspace
6. Click **"Create App"**

## Step 2: Get Bot Token

1. In your app, go to **"OAuth & Permissions"** in the left sidebar
2. Under **"Bot Token Scopes"**, add these scopes:
   - `app_mentions:read` - Read messages that mention the bot
   - `channels:history` - Read messages in public channels
   - `chat:write` - Send messages
   - `files:read` - Read files shared with the bot
   `groups:history` - Read messages in private channels
   - `im:history` - Read messages in direct messages
   - `mpim:history` - Read messages in multi-party DMs
   - `reactions:read` - Read message reactions
   - `reactions:write` - Add reactions to messages
3. Click **"Install to Workspace"**
4. Copy the **"Bot User OAuth Token"** (starts with `xoxb-`)

## Step 3: Enable Socket Mode

1. Go to **"Socket Mode"** in the left sidebar
2. Enable Socket Mode
3. Click **"Generate Token"**
4. Copy the **"App-Level Token"** (starts with `xapp-`)

## Step 4: Subscribe to Events

1. Go to **"Event Subscriptions"** in the left sidebar
2. Enable events
3. Under **"Subscribe to bot events"**, add:
   - `message.channels` - Messages in public channels
   - `message.groups` - Messages in private channels
   - `message.im` - Direct messages
   - `message.mpim` - Multi-party direct messages
   - `app_mention` - Bot mentions
4. Save changes

## Step 5: Get Your User ID

1. Open Slack
2. Click your profile picture (top right)
3. Click **"Profile"**
4. Click the **"..."** menu
5. Select **"Copy member ID"**

## Step 6: Configure PicoClaw

Edit `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "slack": {
      "enabled": true,
      "bot_token": "xoxb-...",
      "app_token": "xapp-...",
      "allow_from": ["U1234567890"]
    }
  }
}
```

| Option | Required | Description |
|--------|----------|-------------|
| `enabled` | Yes | Set to `true` to enable |
| `bot_token` | Yes | Bot User OAuth Token (starts with `xoxb-`) |
| `app_token` | Yes | App-Level Token for Socket Mode (starts with `xapp-`) |
| `allow_from` | No | Array of allowed user IDs |

## Step 7: Start the Gateway

```bash
picoclaw gateway
```

You should see:
```
Channels enabled: slack
Slack bot connected
Slack channel started (Socket Mode)
```

## Step 8: Test

1. Open Slack
2. Invite the bot to a channel: `/invite @PicoClaw Bot`
3. Send a message mentioning the bot: `@PicoClaw Bot hello`
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

### Direct Messages

Users can DM the bot directly for private conversations.

### Thread Replies

The bot maintains thread context - replies in threads stay in threads.

### Reactions

The bot adds a "eyes" reaction when processing a message and a "white_check_mark" when done.

## Troubleshooting

### Bot not responding

1. Verify Socket Mode is enabled
2. Check both bot_token and app_token are correct
3. Ensure events are subscribed
4. Check `allow_from` includes your user ID

### "not_authed" or "invalid_auth" error

1. Re-install the app to your workspace
2. Verify tokens are correct and not expired

### Bot can't read messages

1. Ensure the bot is invited to the channel
2. Check that required OAuth scopes are added
3. Re-install the app after adding scopes

### Socket Mode connection issues

1. Verify your network allows WebSocket connections
2. Check the app_token is an App-Level Token (starts with `xapp-`)

## Advanced Configuration

### Multiple Users

```json
{
  "channels": {
    "slack": {
      "enabled": true,
      "bot_token": "xoxb-...",
      "app_token": "xapp-...",
      "allow_from": ["U1234567890", "U0987654321"]
    }
  }
}
```

### Slash Commands

To add slash commands:

1. Go to **"Slash Commands"** in your app
2. Create a new command (e.g., `/picoclaw`)
3. Set the request URL (not needed for Socket Mode)
4. Install the app to update

## See Also

- [Channels Overview](README.md)
- [Gateway Command](../cli/gateway.md)
- [Troubleshooting](../../operations/troubleshooting.md)
