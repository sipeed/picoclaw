# QQ

PicoClaw supports QQ via the official bot API of the QQ Open Platform.

## Configuration

```json
{
  "channels": {
    "qq": {
      "enabled": true,
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

| Field      | Type   | Required | Description                              |
| ---------- | ------ | -------- | ---------------------------------------- |
| enabled    | bool   | Yes      | Whether to enable the QQ channel         |
| app_id     | string | Yes      | App ID of the QQ bot application         |
| app_secret | string | Yes      | App Secret of the QQ bot application     |
| allow_from | array  | No       | User ID allowlist, empty means allow all |

## Setup Process

1. Go to the [QQ Open Platform](https://q.qq.com/) and create a bot.
2. Obtain the App ID and App Secret from the dashboard.
3. Enable bot sandbox mode and add users and groups to the sandbox.
4. Fill in the App ID and App Secret in the configuration file.
