> Back to [README](../../../README.md)

# Line

PicoClaw supports LINE through the LINE Messaging API with webhook callbacks.

## Configuration

```json
{
  "channel_list": {
    "line": {
      "enabled": true,
      "type": "line",
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

| Field                | Type   | Required | Description                                                        |
| -------------------- | ------ | -------- | ------------------------------------------------------------------ |
| enabled              | bool   | Yes      | Whether to enable the LINE channel                                 |
| channel_secret       | string | Yes      | Channel Secret for the LINE Messaging API                          |
| channel_access_token | string | Yes      | Channel Access Token for the LINE Messaging API                    |
| webhook_path         | string | No       | Webhook path (default: /webhook/line)                              |
| allow_from           | array  | No       | User ID whitelist; empty means all users are allowed               |

## Setup

1. Go to the [LINE Developers Console](https://developers.line.biz/console/) and create a provider and a Messaging API channel
2. Obtain the Channel Secret and Channel Access Token
3. Configure the webhook:
   - LINE requires webhooks to use HTTPS, so you need to deploy a server with HTTPS support, or use a reverse proxy tool like ngrok to expose your local server to the internet
   - PicoClaw uses a shared Gateway HTTP server to receive webhook callbacks for all channels, listening on 127.0.0.1:18790 by default
   - Set the Webhook URL to `https://your-domain.com/webhook/line`, then reverse-proxy your external domain to the local Gateway (default port 18790)
   - Enable the webhook and verify the URL
4. Fill in the Channel Secret and Channel Access Token in the configuration file
