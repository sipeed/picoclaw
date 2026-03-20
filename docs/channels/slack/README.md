> Back to [README](../../../README.md)

# Slack

Slack is a leading enterprise instant messaging platform. PicoClaw uses Slack's Socket Mode for real-time bidirectional communication, with no need to configure a public webhook endpoint.

## Configuration

```json
{
  "channels": {
    "slack": {
      "enabled": true,
      "bot_token": "xoxb-...",
      "app_token": "xapp-...",
      "allow_from": []
    }
  }
}
```

| Field      | Type   | Required | Description                                                              |
| ---------- | ------ | -------- | ------------------------------------------------------------------------ |
| enabled    | bool   | Yes      | Whether to enable the Slack channel                                      |
| bot_token  | string | Yes      | Bot User OAuth Token for the Slack bot (starts with xoxb-)               |
| app_token  | string | Yes      | Socket Mode App Level Token for the Slack app (starts with xapp-)        |
| allow_from | array  | No       | User ID whitelist; empty means all users are allowed                     |

## Setup

1. Go to [Slack API](https://api.slack.com/) and create a new Slack app
2. Enable Socket Mode and obtain the App Level Token
3. Add Bot Token Scopes (e.g. `chat:write`, `im:history`, etc.)
4. Install the app to your workspace and obtain the Bot User OAuth Token
5. Fill in the Bot Token and App Token in the configuration file
