# OneBot

OneBot is an open protocol standard for QQ bots, providing a unified interface for various QQ bot implementations (e.g., go-cqhttp, Mirai). It uses WebSockets for communication.

## Configuration

```json
{
  "channels": {
    "onebot": {
      "enabled": true,
      "ws_url": "ws://localhost:8080",
      "access_token": "",
      "allow_from": []
    }
  }
}
```

| Field        | Type   | Required | Description                               |
| ------------ | ------ | -------- | ----------------------------------------- |
| enabled      | bool   | Yes      | Whether to enable the OneBot channel      |
| ws_url       | string | Yes      | WebSocket URL of the OneBot server        |
| access_token | string | No       | Access token for the OneBot server        |
| allow_from   | array  | No       | User ID allowlist, empty means allow all  |

## Setup Process

1. Deploy a OneBot-compatible implementation (e.g., NapCat).
2. Configure the OneBot implementation to enable WebSocket services and set an access token (if required).
3. Fill in the WebSocket URL and access token in the configuration file.
