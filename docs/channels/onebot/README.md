> Back to [README](../../../README.md)

# OneBot

OneBot is an open protocol standard for QQ bots, providing a unified interface for various QQ bot implementations (e.g. go-cqhttp, Mirai). It uses WebSocket for communication.

## Configuration

```json
{
  "channel_list": {
    "onebot": {
      "enabled": true,
      "type": "onebot",
      "ws_url": "ws://localhost:8080",
      "access_token": "",
      "allow_from": []
    }
  }
}
```

| Field        | Type   | Required | Description                                                      |
| ------------ | ------ | -------- | ---------------------------------------------------------------- |
| enabled      | bool   | Yes      | Whether to enable the OneBot channel                             |
| ws_url       | string | Yes      | WebSocket URL of the OneBot server                               |
| access_token | string | No       | Access token for connecting to the OneBot server                 |
| allow_from   | array  | No       | User ID whitelist; empty means all users are allowed             |

## Setup

1. Deploy a OneBot-compatible implementation (e.g. napcat)
2. Configure the OneBot implementation to enable the WebSocket service and set an access token (if needed)
3. Fill in the WebSocket URL and access token in the configuration file
