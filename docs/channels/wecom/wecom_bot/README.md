> Back to [README](../../../../README.md)

# WeCom Bot

WeCom Bot is a quick integration method provided by WeCom that can receive messages via a Webhook URL.

## Configuration

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
      "webhook_path": "/webhook/wecom",
      "allow_from": [],
      "reply_timeout": 5
    }
  }
}
```

| Field | Type | Required | Description |
| ---------------- | ------ | -------- | -------------------------------------------- |
| token | string | Yes | Signature verification token |
| encoding_aes_key | string | Yes | 43-character AES key used for decryption |
| webhook_url | string | Yes | WeCom group bot webhook URL used to send replies |
| webhook_path | string | No | Webhook endpoint path (default: /webhook/wecom) |
| allow_from | array | No | User ID allowlist (empty = allow all users) |
| reply_timeout | int | No | Reply timeout in seconds (default: 5) |

## Setup

1. Add a bot to a WeCom group
2. Obtain the Webhook URL
3. (To receive messages) Configure the message receiving API address (callback URL), Token, and EncodingAESKey on the bot configuration page
4. Enter the relevant information into the config file

   Note: PicoClaw now uses a shared Gateway HTTP server to receive webhook callbacks for all channels. The default listening address is 127.0.0.1:18790. To receive callbacks from the public internet, reverse-proxy your external domain to the Gateway (default port 18790).
