# DingTalk

DingTalk is an enterprise communication platform by Alibaba, widely popular in Chinese workplaces. It uses a streaming SDK to maintain persistent connections.

## Configuration

```json
{
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

| Field         | Type   | Required | Description                               |
| ------------- | ------ | -------- | ----------------------------------------- |
| enabled       | bool   | Yes      | Whether to enable the DingTalk channel    |
| client_id     | string | Yes      | Client ID of the DingTalk application     |
| client_secret | string | Yes      | Client Secret of the DingTalk application |
| allow_from    | array  | No       | User ID allowlist, empty means allow all  |

## Setup Process

1. Go to the [DingTalk Open Platform](https://open.dingtalk.com/)
2. Create an internal enterprise application
3. Obtain the Client ID and Client Secret from the application settings
4. Configure OAuth and event subscriptions (if needed)
5. Fill in the Client ID and Client Secret in the configuration file
