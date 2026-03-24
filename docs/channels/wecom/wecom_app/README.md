> Back to [README](../../../../README.md)

# WeCom Internal App

A WeCom Internal App is an application created by an enterprise within WeCom, primarily intended for internal use. Through WeCom Internal Apps, enterprises can achieve efficient communication and collaboration with employees, improving productivity.

## Configuration

```json
{
  "channels": {
    "wecom_app": {
      "enabled": true,
      "corp_id": "wwxxxxxxxxxxxxxxxx",
      "corp_secret": "YOUR_CORP_SECRET",
      "agent_id": 1000002,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-app",
      "allow_from": [],
      "reply_timeout": 5
    }
  }
}
```

| Field | Type | Required | Description |
| ---------------- | ------ | -------- | ---------------------------------------- |
| corp_id | string | Yes | Enterprise ID |
| corp_secret | string | Yes | Application secret |
| agent_id | int | Yes | Application agent ID |
| token | string | Yes | Callback verification token |
| encoding_aes_key | string | Yes | 43-character AES key |
| webhook_path | string | No | Webhook path (default: /webhook/wecom-app) |
| allow_from | array | No | User ID allowlist |
| reply_timeout | int | No | Reply timeout in seconds |

## Setup

1. Log in to the [WeCom Admin Console](https://work.weixin.qq.com/)
2. Go to "App Management" -> "Create App"
3. Obtain the Enterprise ID (CorpID) and App Secret
4. Configure "Receive Messages" in the app settings to get the Token and EncodingAESKey
5. Set the callback URL to `http://<your-server-ip>:<port>/webhook/wecom-app`
6. Enter the CorpID, Secret, AgentID, and other details into the config file

   Note: PicoClaw now uses a shared Gateway HTTP server to receive webhook callbacks for all channels. The default listening address is 127.0.0.1:18790. To receive callbacks from the public internet, reverse-proxy your external domain to the Gateway (default port 18790).
