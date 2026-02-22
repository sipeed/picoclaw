# Slack

Slack 是全球领先的企业级即时通讯平台。PicoClaw 采用 Slack 的 Socket Mode 实现实时双向通信，无需配置公开的 Webhook 端点。

## 配置

```json
{
  "channels": {
    "slack": {
      "enabled": true,
      "bot_token": "xoxb-...",
      "app_token": "xapp-...",
      "allow_from": []
    }
  }
}
```

| 字段       | 类型   | 必填 | 描述                                                     |
| ---------- | ------ | ---- | -------------------------------------------------------- |
| enabled    | bool   | 是   | 是否启用 Slack 频道                                      |
| bot_token  | string | 是   | Slack 机器人的 Bot User OAuth Token (以 xoxb- 开头)      |
| app_token  | string | 是   | Slack 应用的 Socket Mode App Level Token (以 xapp- 开头) |
| allow_from | array  | 否   | 用户ID白名单，空表示允许所有用户                         |

## 设置流程

1. 前往 [Slack API](https://api.slack.com/) 创建一个新的 Slack 应用
2. 启用 Socket Mode 并获取 App Level Token
3. 添加 Bot Token Scopes(例如`chat:write`、`im:history`等)
4. 安装应用到工作区并获取 Bot User OAuth Token
5. 将 Bot Token 和 App Token 填入配置文件中
