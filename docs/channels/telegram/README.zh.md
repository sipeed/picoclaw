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
      "proxy": "",
      "groups": {
        "-1001234567890": {
          "topics": {
            "1": { "agent_id": "main" },
            "42": { "agent_id": "coder" }
          }
        }
      }
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
| groups     | object | 否   | 群组级配置；可用于 Telegram forum topics 的每 topic agent 路由 |

## 设置流程

1. 在 Telegram 中搜索 `@BotFather`
2. 发送 `/newbot` 命令并按照提示创建新机器人
3. 获取 HTTP API Token
4. 将 Token 填入配置文件中
5. (可选) 配置 `allow_from` 以限制允许互动的用户 ID (可通过 `@userinfobot` 获取 ID)

## Forum Topics / 话题线程

PicoClaw 现在支持 Telegram forum supergroup 的 topics（话题线程）：

- 每个 topic 会使用独立会话，上下文不会再和同群其他 topic 混在一起
- 回复、占位消息、typing 指示器都会回到原 topic
- topic 的内部目标格式是 `-1001234567890:topic:42`
- `General` 话题的 thread id 固定为 `1`
- 普通群聊中的 reply thread 不会被当成独立 session

### 按 topic 指定 agent

配置路径：

`channels.telegram.groups.<chat_id>.topics.<thread_id>.agent_id`

示例：

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
      "groups": {
        "-1001234567890": {
          "topics": {
            "1": { "agent_id": "main" },
            "42": { "agent_id": "coder" },
            "77": { "agent_id": "support" }
          }
        }
      }
    }
  }
}
```

上面配置表示：

- General 话题走 `main`
- thread `42` 走 `coder`
- thread `77` 走 `support`

### 进阶：使用 bindings 绑定指定 topic

如果你更想走通用 routing 机制，也可以直接把 topic 当作 group peer：

```json
{
  "bindings": [
    {
      "agent_id": "coder",
      "match": {
        "channel": "telegram",
        "peer": {
          "kind": "group",
          "id": "-1001234567890:topic:42"
        }
      }
    }
  ]
}
```
