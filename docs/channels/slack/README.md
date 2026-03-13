# Slack

Slack is a leading enterprise-grade instant messaging platform. PicoClaw uses Slack's Socket Mode for real-time bidirectional communication, eliminating the need to configure public Webhook endpoints.

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

| Field      | Type   | Required | Description                                               |
| ---------- | ------ | -------- | --------------------------------------------------------- |
| enabled    | bool   | Yes      | Whether to enable the Slack channel                       |
| bot_token  | string | Yes      | Bot User OAuth Token for the Slack bot (starts with xoxb-)|
| app_token  | string | Yes      | Socket Mode App Level Token for the Slack app (starts with xapp-)|
| allow_from | array  | No       | User ID allowlist, empty means allow all                  |

## Setup Process

1. Go to the [Slack API](https://api.slack.com/) and create a new Slack application.
2. Enable Socket Mode and obtain an App Level Token.
3. Add Bot Token Scopes (e.g., `chat:write`, `im:history`, etc.).
4. Install the application to your workspace and obtain the Bot User OAuth Token.
5. Fill in the Bot Token and App Token in the configuration file.
