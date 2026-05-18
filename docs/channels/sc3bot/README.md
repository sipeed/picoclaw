> Back to [README](../../../README.md)

# Server酱³ Bot

The Server酱³ Bot channel integrates with [Server酱³](https://sc3.ft07.com/) Bot API, allowing PicoClaw to send and receive messages through the Server酱³ messaging platform. It supports both polling mode (getUpdates) and webhook mode for receiving messages.

## Configuration

```json
{
  "channel_list": {
    "sc3bot": {
      "enabled": true,
      "type": "sc3bot",
      "settings": {
        "token": "your_bot_token_here"
      }
    }
  }
}
```

| Field   | Type   | Required | Description                                  |
| ------- | ------ | -------- | -------------------------------------------- |
| enabled | bool   | Yes      | Whether to enable the Server酱³ Bot channel  |
| token   | string | Yes      | Server酱³ Bot Token                          |
| proxy   | string | No       | HTTP proxy URL (e.g., http://127.0.0.1:7890) |
| secret  | string | No       | Webhook secret for request verification      |

## Setup

1. Visit [Server酱³](https://sc3.ft07.com/) and create an account
2. Navigate to the Bot management page and create a new Bot
3. Obtain the Bot Token
4. Fill in the Token in the configuration file

## Webhook Mode (Optional)

By default, the channel uses polling mode to receive messages. To use webhook mode:

1. Configure a public webhook URL in the Server酱³ client
2. The channel automatically handles webhook requests at `/webhook/sc3bot`
3. (Optional) Set `secret` for webhook request verification

## API Reference

The channel implements the following Server酱³ Bot API methods:

- `getMe` - Get bot information (called on startup)
- `sendMessage` - Send text messages
- `sendChatAction` - Send typing indicators
- `getUpdates` - Poll for new messages (polling mode)

For more details, see the [Server酱³ Bot API documentation](https://sc3.ft07.com/).
