# OneBot

OneBot 是一个面向 QQ 机器人的开放协议标准，为多种 QQ 机器人实现（例如 go-cqhttp、Mirai）提供了统一的接口。它使用 WebSocket 进行通信。

## 配置

```json
{
  "channels": {
    "onebot": {
      "enabled": true,
      "ws_url": "ws://localhost:8080",
      "access_token": "",
      "allow_from": []
    }
  }
}
```

| 字段               | 类型   | 必填 | 默认值 | 描述                                                                 |
| ------------------ | ------ | ---- | ------ | -------------------------------------------------------------------- |
| enabled            | bool   | 是   | -      | 是否启用 OneBot 频道                                                 |
| ws_url             | string | 是   | -      | OneBot 服务器的 WebSocket URL                                        |
| access_token       | string | 否   | -      | 连接 OneBot 服务器的访问令牌                                         |
| mentioned_only     | bool   | 否   | true   | 为 true 时仅处理 @ 了机器人的群消息；为 false 时处理所有群消息（适用于官方机器人等仅推送已 @ 消息的实现） |
| group_trigger_prefix | array  | 否   | []     | 群消息触发前缀，消息以此列表中某一项开头也会触发                     |
| allow_from         | array  | 否   | []     | 用户ID白名单，空表示允许所有用户                                     |

## 设置流程

1. 部署一个 OneBot 兼容的实现(例如napcat)
2. 配置 OneBot 实现以启用 WebSocket 服务并设置访问令牌(如果需要)
3. 将 WebSocket URL 和访问令牌填入配置文件中
