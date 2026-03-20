# 🔄 异步任务与 Spawn

> 返回 [README](../../README.zh.md)

### 使用 Spawn 的异步任务

对于耗时较长的任务（网络搜索、API 调用），使用 `spawn` 工具创建一个 **子 Agent (subagent)**：

```markdown
# Periodic Tasks

## Quick Tasks (respond directly)

- Report current time

## Long Tasks (use spawn for async)

- Search the web for AI news and summarize
- Check email and report important messages
```

**关键行为：**

| 特性             | 描述                                     |
| ---------------- | ---------------------------------------- |
| **spawn**        | 创建异步子 Agent，不阻塞主心跳进程       |
| **独立上下文**   | 子 Agent 拥有独立上下文，无会话历史      |
| **message tool** | 子 Agent 通过 message 工具直接与用户通信 |
| **非阻塞**       | spawn 后，心跳继续处理下一个任务         |

#### 子 Agent 通信原理

```
心跳触发 (Heartbeat triggers)
    ↓
Agent 读取 HEARTBEAT.md
    ↓
对于长任务: spawn 子 Agent
    ↓                           ↓
继续下一个任务               子 Agent 独立工作
    ↓                           ↓
所有任务完成                 子 Agent 使用 "message" 工具
    ↓                           ↓
响应 HEARTBEAT_OK            用户直接收到结果
```

子 Agent 可以访问工具（message, web_search 等），并且无需通过主 Agent 即可独立与用户通信。

**配置：**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| 选项       | 默认值 | 描述                         |
| ---------- | ------ | ---------------------------- |
| `enabled`  | `true` | 启用/禁用心跳                |
| `interval` | `30`   | 检查间隔，单位分钟 (最小: 5) |

**环境变量:**

- `PICOCLAW_HEARTBEAT_ENABLED=false` 禁用
- `PICOCLAW_HEARTBEAT_INTERVAL=60` 更改间隔
