> Back to [README](../../../../README.md)

# WeCom AI Bot

The WeCom AI Bot is an official AI conversation integration provided by WeCom. It supports both private and group chats, has a built-in streaming response protocol, and supports proactively pushing the final reply via `response_url` after a timeout.

## Comparison with Other WeCom Channels

| Feature | WeCom Bot | WeCom App | **WeCom AI Bot** |
|---------|-----------|-----------|-----------------|
| Private Chat | ✅ | ✅ | ✅ |
| Group Chat | ✅ | ❌ | ✅ |
| Streaming Output | ❌ | ❌ | ✅ |
| Proactive Push on Timeout | ❌ | ✅ | ✅ |
| Configuration Complexity | Low | High | Medium |

## Configuration

```json
{
  "channels": {
    "wecom_aibot": {
      "enabled": true,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_43_CHAR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-aibot",
      "allow_from": [],
      "welcome_message": "你好！有什么可以帮助你的吗？",
      "max_steps": 10
    }
  }
}
```

| Field | Type | Required | Description |
| ---------------- | ------ | -------- | -------------------------------------------------- |
| token | string | Yes | Callback verification token, configured on the AI Bot management page |
| encoding_aes_key | string | Yes | 43-character AES key, randomly generated on the AI Bot management page |
| webhook_path | string | No | Webhook path (default: /webhook/wecom-aibot) |
| allow_from | array | No | User ID allowlist; empty array allows all users |
| welcome_message | string | No | Welcome message sent when a user opens the chat; leave empty to disable |
| reply_timeout | int | No | Reply timeout in seconds (default: 5) |
| max_steps | int | No | Maximum agent execution steps (default: 10) |

## Setup

1. Log in to the [WeCom Admin Console](https://work.weixin.qq.com/wework_admin)
2. Go to "App Management" → "AI Bot", then create or select an AI Bot
3. On the AI Bot configuration page, fill in the "Message Reception" details:
   - **URL**: `http://<your-server-ip>:18790/webhook/wecom-aibot`
   - **Token**: Randomly generated or custom
   - **EncodingAESKey**: Click "Random Generate" to get a 43-character key
4. Enter the Token and EncodingAESKey into the PicoClaw config file, start the service, then return to the admin console to save (WeCom will send a verification request)

> [!TIP]
> The server must be accessible by WeCom's servers. If you are on an intranet or developing locally, use [ngrok](https://ngrok.com) or frp for tunneling.

## Streaming Response Protocol

WeCom AI Bot uses a "streaming pull" protocol, which differs from the one-shot reply of a standard webhook:

```
User sends a message
  │
  ▼
PicoClaw immediately returns {finish: false} (Agent starts processing)
  │
  ▼
WeCom pulls approximately every 1 second with {msgtype: "stream", stream: {id: "..."}}
  │
  ├─ Agent not done → returns {finish: false} (keep waiting)
  │
  └─ Agent done → returns {finish: true, content: "reply content"}
```

**Timeout Handling** (task exceeds 30 seconds):

If the Agent takes longer than approximately 30 seconds (WeCom's maximum polling window is 6 minutes), PicoClaw will:

1. Immediately close the stream and show the user: "⏳ 正在处理中，请稍候，结果将稍后发送。"
2. The Agent continues running in the background
3. Once the Agent finishes, the final reply is proactively pushed to the user via the `response_url` included in the message

> `response_url` is issued by WeCom, valid for 1 hour, can only be used once, requires no encryption — just POST the markdown message body directly.

## Welcome Message

When `welcome_message` is configured, PicoClaw will automatically reply with it when a user opens the chat window with the AI Bot (`enter_chat` event). Leave it empty to silently ignore the event.

```json
"welcome_message": "你好！我是 PicoClaw AI 助手，有什么可以帮你？"
```

## FAQ

### Callback URL Verification Failed

- Confirm the server firewall has the relevant port open (default 18790)
- Confirm `token` and `encoding_aes_key` are entered correctly
- Check PicoClaw logs to see if a GET request from WeCom was received

### Messages Not Getting a Reply

- Check whether `allow_from` is accidentally restricting the sender
- Look for `context canceled` or Agent errors in the logs
- Confirm the Agent configuration (e.g., `model_name`) is correct

### No Final Push Received for Long-Running Tasks

- Confirm the message callback includes `response_url` (only supported by the newer WeCom AI Bot)
- Confirm the server can make outbound requests (needs to POST to `response_url`)
- Check logs for keywords `response_url mode` and `Sending reply via response_url`

## Reference

- [WeCom AI Bot Integration Docs](https://developer.work.weixin.qq.com/document/path/100719)
- [Streaming Response Protocol](https://developer.work.weixin.qq.com/document/path/100719)
- [Proactive Reply via response_url](https://developer.work.weixin.qq.com/document/path/101138)
