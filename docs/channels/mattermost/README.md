# Mattermost

Mattermost is a popular team collaboration platform. PicoClaw integrates with Mattermost via WebSocket + REST API to provide real-time messaging, typing indicators, placeholder messages, and attachment handling.

## Configuration

```json
{
  "channels": {
    "mattermost": {
      "enabled": true,
      "url": "https://your-mattermost.example.com",
      "bot_token": "YOUR_MATTERMOST_BOT_TOKEN",
      "allow_from": [],
      "group_trigger": {
        "mention_only": true
      },
      "typing": {
        "enabled": true
      },
      "placeholder": {
        "enabled": false,
        "text": "Thinking..."
      },
      "reasoning_channel_id": ""
    }
  }
}
```

| Field                | Type   | Required | Description |
| -------------------- | ------ | -------- | ----------- |
| enabled              | bool   | Yes      | Whether to enable the Mattermost channel |
| url                  | string | Yes      | Mattermost server URL (e.g. `https://chat.example.com`) |
| bot_token            | string | Yes      | Bot Access Token |
| allow_from           | array  | No       | User allowlist (Mattermost user IDs); empty means all users are allowed |
| group_trigger        | object | No       | Group trigger strategy (`mention_only` / `prefixes`) |
| typing               | object | No       | Typing indicator configuration |
| placeholder          | object | No       | Placeholder message configuration (sends a placeholder first, then edits it with the final reply) |
| reasoning_channel_id | string | No       | Target channel ID for reasoning/thinking output |

## Setup

1. Enable Bot Accounts in the Mattermost System Console (if not already enabled)
2. Create a Bot account and copy the access token
3. Add the Bot to the channels/groups where it should respond
4. Set `url` and `bot_token` in `config.json`
5. Start `picoclaw gateway`

## Behavior

- Direct messages (DMs) are responded to by default
- Group/channel messages are responded to by default; set `group_trigger.mention_only=true` to only trigger on @mentions
- Automatically reconnects on connection failure and resumes messaging after reconnection

## FAQ

1. `no channels enabled` error on startup

- Verify that `channels.mattermost.enabled=true`
- Verify that `url` and `bot_token` are not empty
- Verify that the correct config file is being loaded (check if `PICOCLAW_CONFIG` / `PICOCLAW_HOME` overrides are in effect)

2. Bot does not respond in a channel

- Check if `allow_from` is restricting the user
- If `mention_only` is enabled, make sure the message contains an @mention of the bot
- Verify that the bot has been added to the channel and has permission to post
