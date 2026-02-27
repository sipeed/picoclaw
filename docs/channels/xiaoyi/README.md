# XiaoYi (小艺)

XiaoYi Channel enables AI agent communication via Huawei's A2A (Agent-to-Agent) protocol, supporting text messages, file attachments, streaming responses, and status updates.

## Configuration

```json
{
  "channels": {
    "xiaoyi": {
      "enabled": true,
      "ak": "your-access-key",
      "sk": "your-secret-key",
      "agent_id": "your-agent-id",
      "ws_url1": "",
      "ws_url2": "",
      "allow_from": []
    }
  }
}
```

| Field      | Type   | Required | Description                                    |
| ---------- | ------ | -------- | ---------------------------------------------- |
| enabled    | bool   | Yes      | Enable XiaoYi channel                          |
| ak         | string | Yes      | Access Key                                     |
| sk         | string | Yes      | Secret Key                                     |
| agent_id   | string | Yes      | Agent identifier                               |
| ws_url1    | string | No       | Server 1 URL, defaults to official server      |
| ws_url2    | string | No       | Server 2 URL, defaults to backup server        |
| allow_from | array  | No       | User ID whitelist, empty allows all users      |

## Setup

For detailed setup instructions, please refer to the official Huawei documentation: [OpenClaw Integration Guide](https://developer.huawei.com/consumer/cn/doc/service/openclaw-0000002518410344)

1. Register and create an OpenClaw type agent on Huawei Developer Platform
2. Obtain AK (Access Key) and SK (Secret Key)
3. Get Agent ID
4. Add configuration to config file
5. Start PicoClaw, XiaoYi Channel will automatically connect to XiaoYi servers

## Features

- **WebSocket long connection**: Dual server hot-standby support
- **Auto-reconnect**: Exponential backoff strategy, max 50 retries
- **Heartbeat**: Protocol-level + application-level dual heartbeat
- **Streaming response**: Support progressive result return
- **Status update**: Send "processing" status immediately upon receiving message
