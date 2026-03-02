# 企业微信机器人

企业微信机器人是企业微信提供的一种快速接入方式，可以通过 Webhook URL 接收消息。

## 配置

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

| 字段             | 类型   | 必填 | 描述                                         |
| ---------------- | ------ | ---- | -------------------------------------------- |
| token            | string | 是   | 签名验证代币                                 |
| encoding_aes_key | string | 是   | 用于解密的 43 字符 AES 密钥                  |
| webhook_url      | string | 是   | 用于发送回复的企业微信群聊机器人 Webhook URL |
| webhook_path     | string | 否   | Webhook 端点路径（默认：/webhook/wecom）     |
| allow_from       | array  | 否   | 用户 ID 白名单（空值 = 允许所有用户）        |
| reply_timeout    | int    | 否   | 回复超时时间（单位：秒，默认值：5）          |

## 设置流程

1. 在企业微信群中添加机器人
2. 获取 Webhook URL
3. (如需接收消息) 在机器人配置页面设置接收消息的 API 地址（回调地址）以及 Token 和 EncodingAESKey
4. 将相关信息填入配置文件

   注意: PicoClaw 现在使用共享的 Gateway HTTP 服务器来接收所有渠道的 webhook 回调，默认监听地址为 127.0.0.1:18790。如需从公网接收回调，请把外部域名反向代理到 Gateway（默认端口 18790）。
