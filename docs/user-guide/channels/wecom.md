# WeCom (Enterprise WeChat)

WeCom (企业微信) is a enterprise messaging platform by Tencent. PicoClaw supports two types of WeCom integrations.

## Integration Types

| Type | Description | Group Chat | Active Messaging |
|------|-------------|------------|------------------|
| **WeCom Bot** | Group bot (智能机器人) | ✅ | ❌ |
| **WeCom App** | Custom app (自建应用) | ❌ | ✅ |

### WeCom Bot (Group Bot)

Simple setup for group chat notifications. The bot can receive and respond to messages in group chats.

**Quick Setup:**

1. WeCom Admin Console → Group Chat → Add Group Bot
2. Copy the webhook URL (format: `https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx`)

**Configuration:**

```json
{
  "channels": {
    "wecom": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY",
      "webhook_host": "0.0.0.0",
      "webhook_port": 18793,
      "webhook_path": "/webhook/wecom",
      "allow_from": []
    }
  }
}
```

### WeCom App (Custom App)

Full-featured integration with active messaging capabilities. Can send proactive messages to users.

**Features:**
- ✅ Receive messages
- ✅ Send messages proactively
- ✅ Private chat support
- ❌ Group chat not supported

**Detailed Configuration:**

- **[English Guide](wecom-app-en.md)**
- **[中文指南](wecom-app-cn.md)**

## Configuration

### Basic Settings

| Option | Description |
|--------|-------------|
| `enabled` | Enable/disable the channel |
| `token` | Token for signature verification |
| `encoding_aes_key` | Encryption key (43 characters) |
| `webhook_host` | Host for webhook server |
| `webhook_port` | Port for webhook (default: 18792 for app, 18793 for bot) |
| `webhook_path` | URL path for webhook |
| `allow_from` | User IDs allowed to interact (empty = all) |

## Troubleshooting

### Callback URL Verification Failed

1. Check firewall has webhook port open
2. Verify `token` and `encoding_aes_key` match WeCom console settings
3. Check PicoClaw logs for incoming requests

### Chinese Message Decryption Failed

Ensure you're using the latest PicoClaw version - WeCom uses non-standard PKCS7 padding (32-byte blocks).

## References

- [WeCom Official Documentation](https://developer.work.weixin.qq.com/)
