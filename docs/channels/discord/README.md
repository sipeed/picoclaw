> Back to [README](../../../README.md)

# Discord

Discord is a free voice, video, and text chat application designed for communities. PicoClaw connects to Discord servers via the Discord Bot API, supporting both receiving and sending messages.

## Configuration

```json
{
  "agents": {
    "defaults": {
      "tool_feedback": {
        "enabled": true,
        "max_args_length": 300
      }
    }
  },
  "channel_list": {
    "discord": {
      "enabled": true,
      "type": "discord",
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "placeholder": {
        "enabled": true,
        "text": ["Thinking... 💭"]
      },
      "group_trigger": {
        "mention_only": false
      },
      "reasoning_channel_id": ""
    }
  }
}
```

| Field                | Type   | Required | Description                                                                 |
| -------------------- | ------ | -------- | --------------------------------------------------------------------------- |
| enabled              | bool   | Yes      | Whether to enable the Discord channel                                       |
| token                | string | Yes      | Discord Bot Token                                                           |
| allow_from           | array  | No       | Allowlist of user IDs; empty means all users are allowed                    |
| placeholder          | object | No       | Placeholder message config shown while the agent is working                 |
| group_trigger        | object | No       | Group trigger settings (example: { "mention_only": false })                 |
| reasoning_channel_id | string | No       | Optional target channel ID for reasoning/thinking output                    |

## Visible Execution Feedback

Discord can show three different kinds of "working" feedback:

1. Typing indicator: automatic, no extra config needed.
2. Placeholder message: enable `channel_list.discord.placeholder.enabled` to send a visible `Thinking...` message that is later edited into the final reply.
3. Tool execution feedback: enable `agents.defaults.tool_feedback.enabled` to send a short message before each tool call, for example:

```text
🔧 `web_search`
Checking the latest PicoClaw release notes before I answer.
```

If you only see `Bot is typing`, check that `placeholder.enabled` or `tool_feedback.enabled` is actually set in your runtime config.

## Setup

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications) and create a new application
2. Enable Intents:
   - Message Content Intent
   - Server Members Intent
3. Obtain the Bot Token
4. Fill in the Bot Token in the configuration file
5. Invite the bot to your server and grant the necessary permissions (e.g. Send Messages, Read Message History)
