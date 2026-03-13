# LINE

PicoClaw supports LINE through the LINE Messaging API and Webhook callbacks.

## Configuration

```json
{
  "channels": {
    "line": {
      "enabled": true,
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

| Field                | Type   | Required | Description                                |
| -------------------- | ------ | -------- | ------------------------------------------ |
| enabled              | bool   | Yes      | Whether to enable the LINE channel         |
| channel_secret       | string | Yes      | Channel Secret for LINE Messaging API      |
| channel_access_token | string | Yes      | Channel Access Token for LINE Messaging API|
| webhook_path         | string | No       | Webhook path (default: /webhook/line)      |
| allow_from           | array  | No       | User ID allowlist, empty means allow all   |

## Setup Process

1. Go to the [LINE Developers Console](https://developers.line.biz/console/) and create a provider and a Messaging API Channel.
2. Obtain the Channel Secret and Channel Access Token.
3. Configure Webhook:
   - LINE requires Webhooks to use the HTTPS protocol. You will need to deploy an HTTPS-enabled server or use a reverse proxy tool like ngrok to expose your local server to the public internet.
   - PicoClaw now uses a shared Gateway HTTP server to receive webhook callbacks for all channels, listening by default at 127.0.0.1:18790.
   - Set the Webhook URL to `https://your-domain.com/webhook/line`, and then reverse proxy your external domain to the local Gateway (default port 18790).
   - Enable the Webhook and verify the URL.
4. Fill in the Channel Secret and Channel Access Token in the configuration file.
