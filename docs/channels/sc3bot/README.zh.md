> 返回 [README](../../../README.md)

# Server酱³ Bot

Server酱³ Bot 频道集成了 [Server酱³](https://sc3.ft07.com/) Bot API，允许 PicoClaw 通过 Server酱³ 消息平台发送和接收消息。支持轮询模式（getUpdates）和 Webhook 模式接收消息。

## 配置

```json
{
  "channel_list": {
    "sc3bot": {
      "enabled": true,
      "type": "sc3bot",
      "settings": {
        "token": "your_bot_token_here"
      }
    }
  }
}
```

| 字段    | 类型   | 必需 | 说明                                         |
| ------- | ------ | ---- | -------------------------------------------- |
| enabled | bool   | 是   | 是否启用 Server酱³ Bot 频道                  |
| token   | string | 是   | Server酱³ Bot Token                          |
| proxy   | string | 否   | HTTP 代理 URL（例如：http://127.0.0.1:7890） |
| secret  | string | 否   | Webhook 密钥，用于请求验证                   |

## 设置

1. 访问 [Server酱³](https://sc3.ft07.com/) 并创建账户
2. 进入 Bot 管理页面，创建一个新的 Bot
3. 获取 Bot Token
4. 将 Token 填入配置文件

## Webhook 模式（可选）

默认情况下，频道使用轮询模式接收消息。如需使用 Webhook 模式：

1. 在 Server酱³ 客户端中配置公网 Webhook URL
2. 频道会自动处理 `/webhook/sc3bot` 路径的 Webhook 请求
3. （可选）设置 `secret` 进行 Webhook 请求验证

## API 参考

该频道实现了以下 Server酱³ Bot API 方法：

- `getMe` - 获取 Bot 信息（启动时调用）
- `sendMessage` - 发送文本消息
- `sendChatAction` - 发送输入状态指示器
- `getUpdates` - 轮询新消息（轮询模式）

更多详情，请参阅 [Server酱³ Bot API 文档](https://sc3.ft07.com/)。
