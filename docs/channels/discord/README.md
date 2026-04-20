> Back to [README](../../../README.md)

# Discord

Discord is a free voice, video, and text chat application designed for communities. PicoClaw connects to Discord servers via the Discord Bot API, supporting both receiving and sending messages.

## Configuration

```json
{
  "channel_list": {
    "discord": {
      "enabled": true,
      "type": "discord",
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "group_trigger": {
        "mention_only": false
      }
    }
  }
}
```

| Field         | Type   | Required | Description                                                                 |
| ------------- | ------ | -------- | --------------------------------------------------------------------------- |
| enabled       | bool   | Yes      | Whether to enable the Discord channel                                       |
| token         | string | Yes      | Discord Bot Token                                                           |
| allow_from    | array  | No       | Allowlist of user IDs; empty means all users are allowed                    |
| group_trigger | object | No       | Group trigger settings (example: { "mention_only": false })                 |

## Setup

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications) and create a new application
2. Enable Intents:
   - Message Content Intent
   - Server Members Intent
3. Obtain the Bot Token
4. Fill in the Bot Token in the configuration file
5. Invite the bot to your server and grant the necessary permissions (e.g. Send Messages, Read Message History)
