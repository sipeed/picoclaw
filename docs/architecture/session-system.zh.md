# Session 系统

> 返回 [README](../README.md)

本文说明 PicoClaw 运行时的 Session 系统如何完成以下事情：

- 把入站消息映射到稳定的会话作用域
- 持久化消息历史与摘要
- 在运行时使用不透明 canonical key 的同时，继续兼容旧的 `agent:...` session key

本文覆盖 `pkg/session`、`pkg/memory` 和 `pkg/agent` 中的核心运行时链路。
它不讨论 `web/backend/middleware` 中 launcher 登录 Cookie 或 dashboard 鉴权 session。

## 职责

Session 系统承担四件事：

1. 决定哪些消息应该共享同一段上下文。
2. 让这段上下文能跨 turn、跨进程重启持久存在。
3. 向 agent loop 暴露一个足够小的 `SessionStore` 抽象。
4. 在存储层和路由层迁移期间继续兼容旧 session key。

## 主要组件

| 层次 | 文件 | 作用 |
| --- | --- | --- |
| Session 抽象 | `pkg/session/session_store.go` | 定义 agent loop 依赖的 `SessionStore` 接口。 |
| 旧后端 | `pkg/session/manager.go` | 每个 session 一个 JSON 文件的旧实现，仍作为回退方案保留。 |
| Session 适配层 | `pkg/session/jsonl_backend.go` | 把 `pkg/memory.Store` 适配成 `SessionStore`，并支持 alias 与 scope metadata。 |
| 持久化存储 | `pkg/memory/jsonl.go` | Append-only JSONL 存储与 `.meta.json` 元数据侧文件。 |
| Scope / Key 构建 | `pkg/session/scope.go`、`pkg/session/key.go`、`pkg/session/allocator.go` | 从路由结果生成结构化 scope、不透明 canonical key 和 legacy alias。 |
| 运行时集成 | `pkg/agent/instance.go`、`pkg/agent/loop.go`、`pkg/agent/loop_message.go` | 初始化存储、分配 session scope，并在 turn 执行前落 metadata。 |

## Session 数据模型

结构化的会话身份由 `session.SessionScope` 表示：

| 字段 | 含义 |
| --- | --- |
| `Version` | Scope 模式版本，当前为 `ScopeVersionV1`。 |
| `AgentID` | 处理该 turn 的路由 agent。 |
| `Channel` | 归一化后的入站 channel 名称。 |
| `Account` | 归一化后的 bot / account 标识。 |
| `Dimensions` | 当前启用的隔离维度顺序，例如 `chat` 或 `sender`。 |
| `Values` | 每个维度对应的具体归一化值。 |

Allocator 当前只识别四个维度：

- `space`
- `chat`
- `topic`
- `sender`

默认配置是：

```json
{
  "session": {
    "dimensions": ["chat"]
  }
}
```

也就是默认按 chat 共享上下文；如果 dispatch rule 覆盖了维度，则以 rule 为准。

## Canonical Key 与 Legacy Alias

运行时现在优先使用不透明 canonical key：

```text
sk_v1_<sha256>
```

它由 `pkg/session/key.go` 中的 scope signature 计算得到。
这样可以让存储 key 稳定，同时不再把持久化格式和某一种旧文本 key 绑定死。

为了兼容旧数据，allocator 还会生成 legacy alias，例如：

```text
agent:main:direct:user123
agent:main:slack:channel:c001
agent:main:pico:direct:pico:session-123
```

这些 alias 很重要，因为旧 session、部分测试以及某些工具仍然会引用这种格式。
JSONL backend 会在读写前先把 alias 解析回 canonical key。

此外，如果调用方已经显式传入了受支持的 session key，agent loop 会保留它，不强行改成新分配的 routed key。
这条逻辑在 `pkg/agent/loop_utils.go:resolveScopeKey` 中：

- 不透明 canonical key
- legacy `agent:...` key

都属于“显式 key”。

## 分配流程

普通入站消息的完整链路如下：

```text
InboundMessage
  -> RouteResolver.ResolveRoute(...)
  -> session.AllocateRouteSession(...)
  -> resolveScopeKey(...)
  -> ensureSessionMetadata(...)
  -> AgentLoop turn 执行
  -> SessionStore 读写
```

具体来说：

1. `pkg/agent/loop_message.go` 先用归一化后的 inbound context 解析 agent route。
2. `session.AllocateRouteSession` 把 route 的 `SessionPolicy` 和 inbound context 组合成结构化 `SessionScope`。
3. Allocator 会生成：
   - `SessionKey`：当前路由会话的 canonical key
   - `SessionAliases`：该路由会话的兼容 alias
   - `MainSessionKey`：agent 级主会话 key
   - `MainAliases`：主会话对应的 legacy alias
4. `runAgentLoop` 通过 `ensureSessionMetadata` 持久化 scope metadata 和 alias。
5. 后续读写时，`JSONLBackend.ResolveSessionKey` 会先把 alias 映射回 canonical key。

`MainSessionKey` 和普通聊天会话是分开的。
它主要服务于 agent 级、系统级的上下文场景，比如 `processSystemMessage`。

## Scope 构建规则

`pkg/session/allocator.go` 会从归一化后的 inbound context 生成 scope 值。
关键规则如下：

- `space` 变成 `<space_type>:<space_id>`
- `chat` 变成 `<chat_type>:<chat_id>`
- `topic` 变成 `topic:<topic_id>`
- `sender` 会先经过 `session.identity_links` 归一化再写入

其中有两个需要单独记住的特殊规则。

### Telegram forum 隔离

Telegram forum topic 必须默认保持隔离，即使配置只写了 `chat` 维度。
为此，如果消息来自 Telegram forum 且策略里没有显式包含 `topic`，allocator 会把 `/<topic_id>` 拼到 `chat` 值后面。

例如：

```text
group:-1001234567890/42
group:-1001234567890/99
```

这两者会得到不同的 session key。

### Identity links

`session.identity_links` 可以把多个 sender 标识折叠为一个 canonical identity。
dispatch 匹配和 session 分配都会使用这套映射，因此同一个人即使跨 channel 或 account 使用不同原始 sender ID，也可以继续落到同一段上下文里。

## 存储格式

默认运行时后端是 `pkg/memory.JSONLStore`，外面包了一层 `session.JSONLBackend`。

每个 session 使用两类文件：

```text
{sanitized_key}.jsonl
{sanitized_key}.meta.json
```

各自保存：

- `.jsonl`：一行一个 `providers.Message`，append-only
- `.meta.json`：摘要、时间戳、行数、逻辑截断偏移、scope、aliases

`SessionMeta` 当前包含：

- `Key`
- `Summary`
- `Skip`
- `Count`
- `CreatedAt`
- `UpdatedAt`
- `Scope`
- `Aliases`

## 写入与崩溃语义

JSONL store 的设计核心是“追加优先、宁可暂时读到旧数据也不要丢数据”：

- `AddMessage` / `AddFullMessage` 先追加一行 JSON，再 `fsync`，最后更新 metadata。
- `TruncateHistory` 先做逻辑截断，本质上只是推进 `meta.Skip`。
- `Compact` 才会真正重写 JSONL 文件，把被跳过的旧行物理移除。
- `SetHistory` 和 `Compact` 都会先写 metadata 再改写 JSONL；如果中途崩溃，最多短时间暴露旧数据，不应丢数据。
- 读取 JSONL 时如果碰到损坏行，会跳过该行，而不是让整个 session 读取失败。

`JSONLBackend.Save` 对应到底层的 `store.Compact(...)`。
也就是说，`Save` 在新实现里不再是“把内存脏数据刷盘”，而是“在逻辑截断后回收无效行占用的磁盘空间”。

## 并发模型

`pkg/memory.JSONLStore` 使用固定 64 分片 mutex，按 session key 的 hash 做串行化。
这样既能做到“按 session 串行”，又不会因为 session 数量增长而把 mutex map 做成无界结构。

旧的 `SessionManager` 则是一个内存 map 加 RW mutex。

这两个实现都满足同一个 `SessionStore` 接口，所以 agent loop 不需要写任何存储后端特化逻辑。

## 兼容与迁移

`pkg/agent/instance.go:initSessionStore` 会优先初始化 JSONL 后端。

启动过程如下：

1. 创建 `memory.NewJSONLStore(dir)`。
2. 执行 `memory.MigrateFromJSON(...)`，把旧 `.json` session 迁入新格式。
3. 用 `session.NewJSONLBackend(store)` 包装。
4. 如果 JSONL 初始化或迁移失败，则回退到 `session.NewSessionManager(dir)`。

这个回退是刻意设计的：做一半的迁移，比整轮继续使用旧后端更危险。

### Alias 提升

第一次为 canonical key 建 metadata 时，`EnsureSessionMetadata` 会尝试把某个非空 legacy alias 的历史提升到 canonical session。
但这件事只会在 canonical session 仍然为空时发生，因此不会覆盖已经存在的 canonical 历史。

这保证了系统在迁移到 opaque key 的同时，仍能保留旧历史，例如：

- 旧的 direct-message key
- 旧的 Pico direct-session key

## 其他 SessionStore 实现

`pkg/agent/subturn.go` 里定义了 `ephemeralSessionStore`。
它同样实现 `SessionStore`，但只存在于内存里，在 sub-turn 结束时销毁。

这样 SubTurn 就能复用相同的 session 接口，而不会把子任务历史写进父会话的持久存储。

## 运行时消费者

Session 系统不只被 agent loop 使用：

- `web/backend/api/session.go` 会读取 JSONL metadata 和旧 JSON session，并把历史暴露给 launcher UI。
- `pkg/agent/steering.go` 可以在 steering 场景下恢复 scope metadata。
- 因为 alias 解析发生在 agent loop 之下，测试和工具仍然可以继续使用 legacy alias。

## 相关文件

- `pkg/session/session_store.go`
- `pkg/session/manager.go`
- `pkg/session/jsonl_backend.go`
- `pkg/session/scope.go`
- `pkg/session/key.go`
- `pkg/session/allocator.go`
- `pkg/memory/jsonl.go`
- `pkg/agent/instance.go`
- `pkg/agent/loop.go`
- `pkg/agent/loop_message.go`
