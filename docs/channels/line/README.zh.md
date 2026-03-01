# Line

PicoClaw 通过 LINE Messaging API 配合 Webhook 回调功能实现对 LINE 的支持。

## 配置

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

| 字段                 | 类型   | 必填 | 描述                                       |
| -------------------- | ------ | ---- | ------------------------------------------ |
| enabled              | bool   | 是   | 是否启用 LINE Channel                      |
| channel_secret       | string | 是   | LINE Messaging API 的 Channel Secret       |
| channel_access_token | string | 是   | LINE Messaging API 的 Channel Access Token |
| webhook_path         | string | 是   | Webhook 的路径 (默认为 /webhook/line)      |
| allow_from           | array  | 否   | 用户ID白名单，空表示允许所有用户           |

## 设置流程

1. 前往 [LINE Developers Console](https://developers.line.biz/console/) 创建一个服务提供商和一个 Messaging API Channel
2. 获取 Channel Secret 和 Channel Access Token
3. 配置Webhook:
   - LINE 要求 Webhook 必须使用 HTTPS 协议，因此需要部署一个支持 HTTPS 的服务器，或者使用反向代理工具如 ngrok 将本地服务器暴露到公网
   - PicoClaw 现在使用共享的 Gateway HTTP 服务器来接收所有渠道的 webhook 回调，默认监听地址为 127.0.0.1:18790
   - 将 Webhook URL 设置为 `https://your-domain.com/webhook/line`，然后将外部域名反向代理到本机的 Gateway（默认端口 18790）
   - 启用 Webhook 并验证 URL
4. 将 Channel Secret 和 Channel Access Token 填入配置文件中
