# Matrix Integration Setup

This guide shows you how to connect PicoClaw to a Matrix homeserver.

## Prerequisites

1. A Matrix account (e.g., @bot:matrix.org or @bot:matrix.example.com)
2. An access token for your Matrix bot account

## Getting a Matrix Access Token

### Method 1: Using Element Web Client

1. Log in to Element (https://app.element.io or your homeserver's web client)
2. Go to **Settings** → **Help & About**
3. Scroll down to **Advanced** section
4. Click on `<click to reveal>` next to **Access Token**
5. Copy the token (it starts with `syt_` or `MDAxOG...`)

### Method 2: Using curl

```bash
curl -X POST https://matrix.org/_matrix/client/r0/login \
  -H "Content-Type: application/json" \
  -d '{
    "type": "m.login.password",
    "user": "your_username",
    "password": "your_password"
  }'
```

The response will include an `access_token` field.

## Configuration

Edit your `~/.picoclaw/config.json`:

```json
{
  "channels": {
    "matrix": {
      "enabled": true,
      "homeserver": "https://matrix.org",
      "user_id": "@bot:matrix.org",
      "access_token": "syt_YOUR_ACCESS_TOKEN_HERE",
      "device_id": "",
      "allow_from": [],
      "join_on_invite": true,
      "require_mention_in_group": true
    }
  }
}
```

### Configuration Options

- **`enabled`**: Set to `true` to enable Matrix integration
- **`homeserver`**: Your Matrix homeserver URL (e.g., `https://matrix.org`, `https://matrix.example.com`)
- **`user_id`**: Full Matrix user ID including homeserver (e.g., `@bot:matrix.org`)
- **`access_token`**: The access token obtained from your Matrix account
- **`device_id`**: (Optional) Specific device ID, leave empty to auto-generate
- **`allow_from`**: (Optional) List of Matrix user IDs allowed to interact with the bot. Empty array = allow all
- **`join_on_invite`**: Set to `true` to auto-join rooms when invited
- **`require_mention_in_group`**: (Default: `true`) Only respond in group chats (3+ members) when the bot is mentioned. Set to `false` to respond to all messages in groups

### Access Control Example

To restrict bot access to specific users:

```json
"allow_from": [
  "@admin:matrix.org",
  "@user1:example.com"
]
```

## Running PicoClaw with Matrix

```bash
picoclaw gateway
```

The bot will:
- Connect to the Matrix homeserver
- Auto-join any rooms it's invited to (if `join_on_invite: true`)
- Listen for messages and respond using the configured AI provider

## Testing

1. Invite your bot to a Matrix room or direct message
2. Send a message like "Hello!"
3. The bot should respond using your configured AI model

## Logs

Matrix-specific logs appear with the `[matrix]` component tag:

```
[INFO] matrix: Starting Matrix client...
[INFO] matrix: Auto-joining room after invite {room_id=!abc123:matrix.org}
[INFO] matrix: Successfully joined room {room_id=!abc123:matrix.org}
[INFO] matrix: Received message {sender=@user:matrix.org, room=Room Name, content=Hello!}
```

## Troubleshooting

### "Failed to create matrix client: M_UNKNOWN_TOKEN"
- Your access token is invalid or expired
- Regenerate the token and update config.json

### "Failed to join room: M_FORBIDDEN"
- The bot doesn't have permission to join
- Check room settings or reinvite the bot

### Bot doesn't respond
- Check `allow_from` configuration - empty array allows everyone
- Verify the AI provider is configured correctly in `agents.defaults.provider`
- Check logs for errors: `picoclaw gateway` will show detailed logs

## Security Notes

- **Never commit your access token to git!**
- Store `config.json` securely with restricted permissions (`chmod 600 ~/.picoclaw/config.json`)
- Consider using environment variables or secrets management for production deployments
- Matrix access tokens grant full account access - treat them like passwords

## Advanced: Using with Docker

Mount your config as a volume:

```bash
docker run -v ~/.picoclaw/config.json:/app/config.json picoclaw gateway
```

Or use environment variables:

```bash
docker run \
  -e MATRIX_HOMESERVER=https://matrix.org \
  -e MATRIX_USER_ID=@bot:matrix.org \
  -e MATRIX_ACCESS_TOKEN=syt_... \
  picoclaw gateway
```

---

**Last Updated:** February 16, 2026  
**PicoClaw Version:** v0.1.1+
