# 路由系统

> 返回 [README](../README.md)

在 PicoClaw 里，“路由系统”不是单一判断。
它实际上是组合起来的一条运行时决策链，负责决定：

1. 哪个 agent 来处理一条入站消息
2. 这条消息应该落在哪种 session 隔离维度下
3. 这一轮该使用 agent 的主模型，还是配置中的轻量模型

本文覆盖 `pkg/routing` 及其在 `pkg/agent` 中的集成方式。
它不讨论 `web/` 目录下 launcher 的 HTTP `ServeMux` 路由，也不讨论前端 TanStack Router 文件路由。

## 路由分层

| 层次 | 文件 | 作用 |
| --- | --- | --- |
| Agent 分发 | `pkg/routing/route.go`、`pkg/routing/agent_id.go` | 为入站消息选择目标 agent。 |
| Session 策略选择 | `pkg/routing/route.go` | 决定该 turn 的会话隔离维度。 |
| 模型路由 | `pkg/routing/router.go`、`pkg/routing/features.go`、`pkg/routing/classifier.go` | 根据消息复杂度在主模型和轻量模型之间做选择。 |
| 运行时集成 | `pkg/agent/registry.go`、`pkg/agent/loop_message.go`、`pkg/agent/loop_turn.go` | 应用 route 结果、分配 session scope，并在真正调用 provider 前选出模型候选集。 |

## 端到端流程

普通用户消息的路径如下：

```text
InboundMessage
  -> NormalizeInboundContext
  -> RouteResolver.ResolveRoute(...)
  -> session.AllocateRouteSession(...)
  -> ensureSessionMetadata(...)
  -> Router.SelectModel(...)
  -> provider execution
```

前半段回答的是“谁来处理，以及属于哪段会话”。
后半段回答的是“这个 agent 这一轮该走哪一档模型”。

## Agent 分发

`routing.RouteResolver` 会把归一化后的 `bus.InboundContext` 转成 `ResolvedRoute`：

```go
type ResolvedRoute struct {
    AgentID       string
    Channel       string
    AccountID     string
    SessionPolicy SessionPolicy
    MatchedBy     string
}
```

`MatchedBy` 主要用于日志和调试，常见值包括：

- `default`
- `dispatch.rule`
- `dispatch.rule:<rule-name>`

## Dispatch 输入视图

真正做规则匹配前，resolver 会先构造一个归一化后的 `dispatchView`。
每个字段都会变成规则匹配所期待的固定形状。

| Selector 字段 | 运行时形状 |
| --- | --- |
| `channel` | 小写 channel 名称 |
| `account` | 归一化后的 account ID |
| `space` | `<space_type>:<space_id>` |
| `chat` | `<chat_type>:<chat_id>` |
| `topic` | `topic:<topic_id>` |
| `sender` | 小写 canonical sender ID |
| `mentioned` | 直接来自 inbound context 的布尔值 |

这意味着 dispatch rule 必须写成归一化后的形状，例如：

```json
{
  "agents": {
    "dispatch": {
      "rules": [
        {
          "name": "support-group",
          "agent": "support",
          "when": {
            "channel": "telegram",
            "chat": "group:-100123"
          }
        },
        {
          "name": "slack-mentions",
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

## Dispatch 算法

`ResolveRoute(...)` 的流程是：

1. 归一化 `channel` 和 `account`。
2. 从配置复制 `session.identity_links`。
3. 构建归一化后的 dispatch view。
4. 按顺序扫描 `agents.dispatch.rules`。
5. 没有任何约束条件的 rule 会被跳过。
6. 第一个所有 selector 字段都精确匹配的 rule 胜出。
7. 如果没有 rule 匹配，则回退到默认 agent。

这带来几个重要结论：

- 第一条命中的规则优先，没有额外 priority 字段
- rule 顺序本身就是优先级
- 指向无效 agent 的 rule 最终会回退到默认 agent
- sender 匹配看到的是经过 `identity_links` 归一化后的身份

## 默认 Agent 解析

如果没有 dispatch rule 命中，或者 rule 指向了不存在的 agent，resolver 会按以下顺序选择默认 agent：

1. `default: true` 的 agent
2. 否则取 `agents.list` 的第一项
3. 如果配置里没有 agent，则使用隐式 `main`

Agent ID 和 Account ID 都会经过 `pkg/routing/agent_id.go` 中的归一化逻辑。

## Session 策略交接

Agent 分发本身不会直接生成 session key。
它只会产出一个 `SessionPolicy`：

```go
type SessionPolicy struct {
    Dimensions    []string
    IdentityLinks map[string][]string
}
```

维度来源有两种：

- 全局 `session.dimensions`
- 如果命中的 dispatch rule 指定了 `session_dimensions`，则用 rule 覆盖

最终只有这些维度名会被保留下来：

- `space`
- `chat`
- `topic`
- `sender`

非法项或重复项会被静默丢弃。

随后 `pkg/session/AllocateRouteSession(...)` 再把这份策略转成：

- 结构化 `SessionScope`
- canonical routed session key
- legacy 兼容 alias

所以可以把职责边界理解为：

- `pkg/routing` 决定“这段对话应该按什么维度隔离”
- `pkg/session` 决定“这些维度如何变成 key 和持久化状态”

## Identity Links

`session.identity_links` 会同时被 dispatch 和 session allocation 使用。
这是刻意保持一致的设计：如果某个 sender 在路由阶段已经被规范化，那么 session 阶段也应该落到同一个身份上。

否则就会出现“消息路由到了同一个 agent，但上下文仍被拆成多个 session”的问题。

## 模型路由

第二阶段路由决定这一轮能否使用更便宜或更快的轻量模型。

配置形状如下：

```json
{
  "routing": {
    "enabled": true,
    "light_model": "gemini-2.0-flash",
    "threshold": 0.35
  }
}
```

`pkg/routing.Router` 会根据当前 turn 的结构特征，返回：

- 选中的模型名
- 是否使用了 light model
- 复杂度分数

当分数低于阈值时，走轻量模型；否则仍使用 agent 的主模型。
但在运行时，只有当 agent 实际配置了 light-model candidates 时，这个判断才会产生效果；否则仍会停留在主模型候选集上。

## 复杂度特征

`ExtractFeatures(...)` 会计算一个与自然语言内容无关、偏结构化的特征向量：

| 特征 | 含义 |
| --- | --- |
| `TokenEstimate` | 估算 token 数；对 CJK 文本比简单 rune 平分更准确。 |
| `CodeBlockCount` | 当前消息中 fenced code block 的数量。 |
| `RecentToolCalls` | 最近 6 条历史消息中的 tool call 总数。 |
| `ConversationDepth` | 整体历史长度。 |
| `HasAttachments` | 是否检测到嵌入媒体或常见媒体 URL / 文件扩展名。 |

这样做的目的，是让模型路由不依赖关键词，从而在不同语言下都保持一致行为。

## RuleClassifier 评分

当前分类器是 `RuleClassifier`，使用加权求和并把结果截断到 `[0, 1]`。

| 信号 | 分值 |
| --- | --- |
| 存在附件 | `1.00` |
| token 估计 `> 200` | `0.35` |
| token 估计 `> 50` | `0.15` |
| 存在代码块 | `0.40` |
| 最近 tool calls `> 3` | `0.25` |
| 最近 tool calls `1..3` | `0.10` |
| 会话深度 `> 10` | `0.10` |

默认阈值是 `0.35`。
这意味着以下行为是刻意设计出来的：

- 很轻的闲聊仍走轻量模型
- 编码类请求通常会立刻切到重模型
- 带附件的请求一定走重模型
- 很长的纯文本请求在默认阈值下也会跨过重模型边界

## 运行时集成

Agent 分发和模型路由发生在不同位置：

- `pkg/agent/registry.go` 持有 `RouteResolver`
- `pkg/agent/loop_message.go` 负责 resolve route 并分配 session scope
- `pkg/agent/loop_turn.go:selectCandidates` 调用 `agent.Router.SelectModel(...)`

当 light model 被选中时，agent loop 会切换到 `agent.LightCandidates`。
如果没有被选中，则继续使用 agent 的主 provider 候选集。

## 显式 Session Key

还有一个不在 `pkg/routing` 内部、但对整体“路由语义”很重要的细节。

在 route 分配完成后，`pkg/agent/loop_utils.go:resolveScopeKey` 会优先保留调用方显式传入的 session key，只要它属于以下格式之一：

- 不透明 canonical key
- legacy `agent:...` key

这样一来，手工系统流、测试和兼容路径即使在正常路由 scope 会生成不同 key 的情况下，仍然能保持确定性。

## 本文不覆盖的内容

仓库里还存在两套和这里无关的“route”系统：

- `web/backend/api/router.go` 注册的后端 HTTP 路由
- `web/frontend/src/routes/` 下的前端文件路由

它们属于 launcher 的实现细节，和本文描述的运行时路由系统是两回事。

## 相关文件

- `pkg/routing/route.go`
- `pkg/routing/router.go`
- `pkg/routing/classifier.go`
- `pkg/routing/features.go`
- `pkg/routing/agent_id.go`
- `pkg/session/allocator.go`
- `pkg/agent/registry.go`
- `pkg/agent/loop_message.go`
- `pkg/agent/loop_turn.go`
