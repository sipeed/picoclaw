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
      "webhook_host": "0.0.0.0",
      "webhook_port": 18791,
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
| webhook_host         | string | 是   | Webhook 监听的主机地址 (通常为 0.0.0.0)    |
| webhook_port         | int    | 是   | Webhook 监听的端口 (默认为 18791)          |
| webhook_path         | string | 是   | Webhook 的路径 (默认为 /webhook/line)      |
| allow_from           | array  | 否   | 用户ID白名单，空表示允许所有用户           |

## 设置流程

1. 前往 [LINE Developers Console](https://developers.line.biz/console/) 创建一个服务提供商和一个 Messaging API Channel
2. 获取 Channel Secret 和 Channel Access Token
3. 配置Webhook:
   - Line要求Webhook必须使用HTTPS协议，因此需要部署一个支持HTTPS的服务器，或者使用反向代理工具如ngrok将本地服务器暴露到公网
   - 将 Webhook URL 设置为 `https://your-domain.com/webhook/line`
   - 启用 Webhook 并验证 URL
4. 将 Channel Secret 和 Channel Access Token 填入配置文件中
