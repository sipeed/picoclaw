# Session 使用指南

> 返回 [README](../project/README.zh.md)

PicoClaw 的 Session 决定了哪些消息会共享同一段对话历史。
如果你的 bot 表现为“记得太多”或“忘得太快”，首先就该检查 session 配置。

这份文档面向编辑 `config.json` 的普通用户。
如果你想看内部实现细节，请看 architecture 文档，而不是这里。

## Session 控制什么

一个 session 会影响：

- Agent 能看到哪些历史消息
- 这段对话何时开始触发摘要
- 同一个群里的不同用户是否共享上下文
- 不同聊天、不同线程、不同空间是否保持隔离

Session 数据保存在工作区目录下，通常是：

```text
~/.picoclaw/workspace/sessions/
```

## 快速开始

### 默认：每个 chat 一段上下文

这是默认值，也是大多数 bot 的正确起点。

```json
{
  "session": {
    "dimensions": ["chat"]
  }
}
```

适用场景：

- 每个群 / 频道都有自己的共享记忆
- 每个私聊都有各自独立的记忆

### 在同一个群里按用户分开

如果同一个群里的不同用户不应该共享上下文，增加 `sender`：

```json
{
  "session": {
    "dimensions": ["chat", "sender"]
  }
}
```

适用场景：

- 一个群里挂着一个共享 assistant，但不希望用户之间串上下文
- 希望每个用户在同一个房间里保留自己的独立记忆

### 在同一个 workspace / guild 下跨多个房间共享上下文

如果你的 channel 会提供 `space`，可以按 workspace 或 guild 共享，而不是按单个房间共享：

```json
{
  "session": {
    "dimensions": ["space"]
  }
}
```

适用场景：

- Slack workspace 里的 assistant 想跨多个 channel 共享上下文
- Discord guild 里的 assistant 想跨多个 channel 共享上下文

### 按线程或论坛 topic 隔离

如果 channel 会提供 `topic`，可以显式按线程隔离：

```json
{
  "session": {
    "dimensions": ["chat", "topic"]
  }
}
```

适用场景：

- 每个论坛 topic 都要保留独立历史
- 每个 threaded discussion 都不能串上下文

## 可用维度

| 维度 | 含义 | 适合什么场景 |
| --- | --- | --- |
| `space` | workspace、guild 或类似的上层容器 | 一个 assistant 跨多个房间共享上下文 |
| `chat` | 私聊、群聊或频道 | 默认按房间隔离 |
| `topic` | 线程、topic 或 forum 子通道 | 让 threaded discussion 保持隔离 |
| `sender` | 归一化后的消息发送者 | 在共享房间内按用户隔离 |

并不是每个 channel 都会提供全部字段。
如果某个 channel 没有 `space` 或 `topic`，对应维度对那条消息就不会生效。

## 关键行为

### Session 总是按 agent 分开

即使两个 agent 处理同一个 chat，它们也不会共享同一段 session。

### Session 仍然会按 channel 和 account 分开

`session.dimensions` 只是添加更细的隔离维度，PicoClaw 仍然保留一层基础隔离：

- agent
- channel
- account

这意味着即使 `dimensions` 为空，系统也**不会**把所有平台的消息都混成一个全局记忆。

### Telegram forum topic 在默认 `chat` 模式下也会保持隔离

Telegram forum 消息在默认 `chat` 模式下就会保留 topic 隔离。
通常不需要额外为 Telegram forum 单独写 workaround。

### 摘要是按 session 触发的

`summarize_message_threshold` 和 `summarize_token_percent` 都是针对单个 session 生效。
如果你把 session 切得更小，摘要也会按更小的历史范围触发。

## 常见配置方案

### 每个群 / 私聊共享一段上下文

```json
{
  "session": {
    "dimensions": ["chat"]
  }
}
```

### 每个 chat 内再按用户拆分

```json
{
  "session": {
    "dimensions": ["chat", "sender"]
  }
}
```

### 在同一个 workspace / guild 内按用户保留上下文

```json
{
  "session": {
    "dimensions": ["space", "sender"]
  }
}
```

这适合做 workspace 级 assistant：用户在同一个 workspace 里跨多个房间移动，但仍保留自己的上下文。

### 只给某个路由出来的 agent 覆盖 session 策略

你可以保留全局默认值，再在某条 dispatch rule 上单独覆盖：

```json
{
  "agents": {
    "list": [
      { "id": "main", "default": true },
      { "id": "support" }
    ],
    "dispatch": {
      "rules": [
        {
          "name": "support group",
          "agent": "support",
          "when": {
            "channel": "telegram",
            "chat": "group:-1001234567890"
          },
          "session_dimensions": ["chat", "sender"]
        }
      ]
    }
  },
  "session": {
    "dimensions": ["chat"]
  }
}
```

在这个例子里：

- 大部分流量仍然按 `chat` 共享上下文
- 只有 support 群按 `chat + sender` 拆成每人一段上下文

## Identity Links

`session.identity_links` 适合处理这种场景：同一个人可能会以多个原始 sender ID 出现，但你希望 PicoClaw 把它们视为同一个发送者身份。

示例：

```json
{
  "session": {
    "dimensions": ["chat", "sender"],
    "identity_links": {
      "john": ["slack:u123", "u123", "legacy-user-42"]
    }
  }
}
```

这主要适用于：

- sender ID 迁移
- 同一平台下的多个 ID 别名
- 调整 channel adapter 或 account 命名后的兼容清理

当前限制：

- `identity_links` 不会自动让同一个用户跨不同 channel 共享记忆
- channel 和 account 仍然属于基础 session scope 的一部分

## 常见问题

### 同一个群里的用户在共享记忆

大概率是当前 session 只按 `chat` 建。
改成：

```json
{
  "session": {
    "dimensions": ["chat", "sender"]
  }
}
```

### 同一个用户在 Slack 和 Telegram 之间没有共享记忆

这是当前实现下的预期行为。
即使使用了 `sender`，PicoClaw 仍然会按 channel 做基础隔离。

### 不同线程混在一起了

如果这个 channel 提供 `topic`，加上它：

```json
{
  "session": {
    "dimensions": ["chat", "topic"]
  }
}
```

### 升级后看到旧的 session key

这属于正常兼容行为。
PicoClaw 在迁移到新的 opaque canonical key 时，仍会兼容旧的 `agent:...` session key。

## 相关文档

- [配置指南](configuration.zh.md)
- [路由指南](routing-guide.zh.md)
- [Provider 与模型配置](providers.zh.md)
