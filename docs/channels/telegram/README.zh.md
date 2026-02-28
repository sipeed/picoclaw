# Telegram

Telegram Channel 通过 Telegram 机器人 API 使用长轮询实现基于机器人的通信。它支持文本消息、媒体附件（照片、语音、音频、文档）、通过 Groq Whisper 进行语音转录以及内置命令处理器。

## 配置

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
      "allow_from": ["123456789"],
      "proxy": ""
    }
  }
}
```

| 字段       | 类型   | 必填 | 描述                                                      |
| ---------- | ------ | ---- | --------------------------------------------------------- |
| enabled    | bool   | 是   | 是否启用 Telegram 频道                                    |
| token      | string | 是   | Telegram 机器人 API Token                                 |
| allow_from | array  | 否   | 用户ID白名单，空表示允许所有用户                          |
| proxy      | string | 否   | 连接 Telegram API 的代理 URL (例如 http://127.0.0.1:7890) |

## 设置流程

1. 在 Telegram 中搜索 `@BotFather`
2. 发送 `/newbot` 命令并按照提示创建新机器人
3. 获取 HTTP API Token
4. 将 Token 填入配置文件中
5. (可选) 配置 `allow_from` 以限制允许互动的用户 ID (可通过 `@userinfobot` 获取 ID)
