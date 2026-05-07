# 路由使用指南

> 返回 [README](../project/README.zh.md)

PicoClaw 里用户能直接感知到的“路由”主要有两部分：

- **agent 路由**：决定哪一个 agent 处理一条消息
- **模型路由**：决定这一轮是走主模型，还是走轻量模型

这份文档面向真实部署中的配置使用场景。

## 快速开始

### 把一个 Telegram 群路由给 support agent

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
          "name": "telegram support group",
          "agent": "support",
          "when": {
            "channel": "telegram",
            "chat": "group:-1001234567890"
          }
        }
      ]
    }
  }
}
```

### 只处理某个 Slack workspace 里的 @提及

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
          "name": "slack mentions",
          "agent": "support",
          "when": {
            "channel": "slack",
            "space": "workspace:t001",
            "mentioned": true
          }
        }
      ]
    }
  }
}
```

### 给简单请求启用轻量模型

```json
{
  "model_list": [
    {
      "model_name": "gpt-main",
      "provider": "openai",
      "model": "gpt-5.4",
      "api_keys": ["sk-main"]
    },
    {
      "model_name": "flash-light",
      "provider": "gemini",
      "model": "gemini-2.0-flash-exp",
      "api_keys": ["sk-light"]
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "gpt-main",
      "routing": {
        "enabled": true,
        "light_model": "flash-light",
        "threshold": 0.35
      }
    }
  }
}
```

## Agent 路由

Agent 路由通过下面这个配置项定义：

```text
agents.dispatch.rules
```

规则从上到下依次检查。
**第一条匹配的规则直接生效**。
如果没有规则命中，PicoClaw 会回退到默认 agent。

## 支持的匹配字段

| 字段 | 含义 | 示例 |
| --- | --- | --- |
| `channel` | Channel 名称 | `telegram`、`slack`、`discord` |
| `account` | 归一化后的 account ID | `default`、`bot2` |
| `space` | workspace、guild 等上层容器 | `workspace:t001`、`guild:123456` |
| `chat` | 私聊、群或频道 | `direct:user123`、`group:-100123`、`channel:c123` |
| `topic` | 线程或话题 | `topic:42` |
| `sender` | 归一化后的发送者身份 | `12345`、`john` |
| `mentioned` | 是否显式 @ 了 bot | `true` |

注意，配置里要写的是运行时归一化后的值，不是原始 webhook / SDK payload。

## 规则顺序

把更具体的规则放前面，把更宽泛的规则放后面。

正确顺序：

1. 某个群里的 VIP 用户
2. 这个群的全部消息
3. 某个 channel 的更宽泛兜底

错误顺序：

1. 这个群的全部消息
2. 同一个群里的 VIP 用户

在错误顺序下，宽泛规则会先命中，VIP 规则永远不会生效。

## 和 Session 的关系

路由和 Session 是相关但不同的两件事：

- 路由决定由哪个 agent 处理
- Session 决定这些消息是否共享同一段记忆

如果你想让某条命中的路由使用不同的会话策略，可以用 `session_dimensions` 覆盖全局 `session.dimensions`。

示例：

```json
{
  "agents": {
    "list": [
      { "id": "main", "default": true },
      { "id": "support" },
      { "id": "sales" }
    ],
    "dispatch": {
      "rules": [
        {
          "name": "vip in support group",
          "agent": "sales",
          "when": {
            "channel": "telegram",
            "chat": "group:-1001234567890",
            "sender": "12345"
          },
          "session_dimensions": ["chat", "sender"]
        },
        {
          "name": "support group",
          "agent": "support",
          "when": {
            "channel": "telegram",
            "chat": "group:-1001234567890"
          },
          "session_dimensions": ["chat"]
        }
      ]
    }
  },
  "session": {
    "dimensions": ["chat"]
  }
}
```

在这个配置里：

- VIP 用户会被路由到 `sales`
- 其他群成员会进入 `support`
- VIP 路由还会额外按 `chat + sender` 做每用户隔离

## Identity Links

当你用 `sender` 做匹配时，`session.identity_links` 也会影响路由结果。
适合这种场景：同一个真实用户可能出现为多个原始 sender ID。

示例：

```json
{
  "session": {
    "identity_links": {
      "john": ["slack:u123", "legacy-user-42"]
    }
  },
  "agents": {
    "dispatch": {
      "rules": [
        {
          "name": "john goes to sales",
          "agent": "sales",
          "when": {
            "sender": "john"
          }
        }
      ]
    }
  }
}
```

## 模型路由

模型路由配置在：

```text
agents.defaults.routing
```

当前支持字段：

| 字段 | 含义 |
| --- | --- |
| `enabled` | 开启或关闭模型路由 |
| `light_model` | `model_list` 中用于简单请求的 `model_name` |
| `threshold` | `[0, 1]` 范围内的复杂度阈值 |

关键行为：

- `light_model` 必须存在于 `model_list`
- PicoClaw 会在启动时解析轻量模型；如果模型无效，路由会被禁用
- 同一轮 turn 只会使用同一档模型，不会中途切档

## 什么会影响复杂度分数

当前模型路由会看一些结构化信号，例如：

- 消息长度
- fenced code block
- 同一 session 最近是否频繁调用工具
- 会话深度
- 是否带有媒体或附件

因此，看起来“很简单”的消息，在以下情况下仍可能走主模型：

- 带代码
- 带图片或音频
- prompt 很长
- 当前是一个工具调用很多的工作流

## 阈值怎么选

推荐起点：

```json
{
  "agents": {
    "defaults": {
      "routing": {
        "enabled": true,
        "light_model": "flash-light",
        "threshold": 0.35
      }
    }
  }
}
```

通用规律：

- 阈值越低，越容易回到主模型
- 阈值越高，越积极地使用轻量模型

实用建议：

- `0.25`：更保守，更少轻量模型 turn
- `0.35`：默认推荐起点
- `0.50+`：只有当你的轻量模型已经能覆盖大多数聊天任务时再考虑

## 常见问题

### 某条规则没有命中

优先检查：

- 规则顺序
- 值的形状是否写成了归一化格式，例如 `group:-100123` 而不是裸 `-100123`
- 当前 channel 是否真的提供了 `space`、`topic` 或 `mentioned`

### 消息被错误的 agent 处理了

最常见原因还是顺序。
记住：第一条匹配的规则直接生效。

### 轻量模型从来没有被用到

检查：

- `agents.defaults.routing.enabled` 是否为 `true`
- `light_model` 是否存在于 `model_list`
- 轻量模型能否成功初始化
- 阈值是不是设得太低

### 明明是短消息，还是走了主模型

这通常是因为当前 turn 同时满足了其他“复杂”信号，例如：

- 带代码块
- 带媒体或附件
- 最近的 session 历史里工具调用很多

### 路由没问题，但上下文还是共享得太多

去调整 `session.dimensions` 或某条 route 上的 `session_dimensions`。
路由只决定“谁来处理”，session 才决定“记忆怎么共享”。

## 相关文档

- [Session 使用指南](session-guide.zh.md)
- [配置指南](configuration.zh.md)
- [Provider 与模型配置](providers.zh.md)
