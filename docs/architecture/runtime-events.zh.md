# Runtime Events 与事件日志

PicoClaw 的 runtime event 是运行时观察面，用来描述 agent、channel、gateway、message bus、MCP 等组件发生了什么。事件发布和日志打印是两件事：

- 事件发布：组件把 `pkg/events.Event` 发布到 runtime event bus，供 hook、测试、调试工具或后续 UI 消费。
- 事件日志：内置 runtime event logger 订阅同一个 bus，并按配置把匹配的事件打印到日志。

这样可以让业务流程继续只负责发布事件，日志策略统一收口到一个地方。

## 默认行为

默认配置只打印 `agent.*` 事件：

```json
{
  "events": {
    "logging": {
      "enabled": true,
      "include": ["agent.*"],
      "min_severity": "info",
      "include_payload": false
    }
  }
}
```

这个默认值保持了旧行为：agent turn、LLM、tool、steering、subturn、error 等事件会出现在日志中；channel、gateway、bus、MCP 事件仍会发布到 runtime event bus，但默认不打印，避免网关启动和消息投递日志过于嘈杂。

## 配置项

配置位于 `config.json` 的 `events.logging`：

| 字段 | 类型 | 默认值 | 说明 |
| ---- | ---- | ------ | ---- |
| `enabled` | bool | `true` | 是否启用内置事件日志订阅器 |
| `include` | string[] | `["agent.*"]` | 允许打印的事件 kind，支持精确匹配、`*`、`agent.*` 这类 glob/prefix |
| `exclude` | string[] | `[]` | 在 include 命中后排除的事件 kind，匹配规则同 include |
| `min_severity` | string | `info` | 最低打印级别：`debug`、`info`、`warn`、`error` |
| `include_payload` | bool | `false` | 是否把原始 payload 放进日志字段 |

`include_payload` 默认关闭。agent 事件日志会输出安全摘要字段，例如 `user_len`、`args_count`、`content_len`，不会默认输出完整用户消息或工具参数。只有在排查问题、并且确认日志存储环境可信时，才建议临时打开 `include_payload`。

## 匹配规则

`include` 和 `exclude` 都匹配 `Event.Kind` 字符串：

```json
{
  "events": {
    "logging": {
      "include": ["gateway.*", "channel.lifecycle.*", "agent.error"],
      "exclude": ["gateway.ready"],
      "min_severity": "info"
    }
  }
}
```

常用写法：

- `["agent.*"]`：只打印 agent 事件。
- `["*"]`：打印所有 runtime events。
- `["gateway.*", "channel.*"]`：只打印 gateway 和 channel 事件。
- `exclude: ["agent.llm.delta"]`：排除高频流式 delta 事件。
- `min_severity: "warn"`：只打印 warn/error 事件。

## 环境变量

同一组配置也可以通过环境变量覆盖，适合临时调试：

```bash
PICOCLAW_EVENTS_LOGGING_ENABLED=true
PICOCLAW_EVENTS_LOGGING_INCLUDE="gateway.*,channel.lifecycle.*"
PICOCLAW_EVENTS_LOGGING_EXCLUDE="gateway.ready"
PICOCLAW_EVENTS_LOGGING_MIN_SEVERITY=info
PICOCLAW_EVENTS_LOGGING_INCLUDE_PAYLOAD=false
```

`include` 和 `exclude` 的环境变量使用逗号分隔。

## 事件名称与触发时机

下面列出当前 runtime event kind、触发时机和主要事件详情。`Source`、`Scope`、`Correlation` 是所有事件都可能携带的 envelope 字段；表里的“主要详情”指 payload 或日志摘要中最有用的字段。

### Agent

| 事件名 | 触发时机 | 主要详情 |
| ------ | -------- | -------- |
| `agent.turn.start` | agent 开始处理一次用户输入或系统输入，turn scope 已创建时 | `user_len`, `media_count`; scope 通常包含 `agent_id`, `session_key`, `turn_id`, `channel`, `chat_id`, `message_id` |
| `agent.turn.end` | 一次 turn 退出时，无论完成、报错还是 hard abort | `status` (`completed`/`error`/`aborted`), `iterations_total`, `duration_ms`, `final_len` |
| `agent.llm.request` | 每次调用 LLM provider 前 | `model`, `messages`, `tools`, `max_tokens` |
| `agent.llm.delta` | 预留给流式 LLM delta；当前实现已定义但没有自然发送点 | `content_delta_len`, `reasoning_delta_len` |
| `agent.llm.response` | LLM provider 返回完整响应后 | `content_len`, `tool_calls`, `has_reasoning` |
| `agent.llm.retry` | LLM 请求因上下文、限流、临时错误等原因准备重试前 | `attempt`, `max_retries`, `reason`, `error`, `backoff_ms` |
| `agent.context.compress` | 上下文历史被压缩时，例如主动预算检查或 LLM retry 处理 | `reason`, `dropped_messages`, `remaining_messages` |
| `agent.session.summarize` | 会话历史异步摘要完成时 | `summarized_messages`, `kept_messages`, `summary_len`, `omitted_oversized` |
| `agent.tool.exec_start` | agent 准备执行一个工具调用前 | `tool`, `args_count`; 默认不打印完整参数 |
| `agent.tool.exec_end` | 工具调用完成后，包括成功、工具错误和 async 结果 | `tool`, `duration_ms`, `for_llm_len`, `for_user_len`, `is_error`, `async` |
| `agent.tool.exec_skipped` | 工具调用被跳过时，例如工具不可用、参数无效或 turn 控制逻辑要求跳过 | `tool`, `reason` |
| `agent.steering.injected` | queued steering message 被注入下一轮 LLM 上下文时 | `count`, `total_content_len` |
| `agent.follow_up.queued` | async 工具结果被重新排入 inbound/follow-up 流程时 | `source_tool`, `content_len` |
| `agent.interrupt.received` | turn 接受 steering、graceful interrupt 或 hard abort 指令时 | `interrupt_kind`, `role`, `content_len`, `queue_depth`, `hint_len` |
| `agent.subturn.spawn` | 父 turn 创建子 turn/subagent 时 | `child_agent_id`, `label`, `parent_turn_id` |
| `agent.subturn.end` | 子 turn 结束时 | `child_agent_id`, `status` |
| `agent.subturn.result_delivered` | 子 turn 结果成功投递到目标 channel/chat 时 | `target_channel`, `target_chat_id`, `content_len` |
| `agent.subturn.orphan` | 子 turn 结果无法投递或无法关联回父 turn 时 | `parent_turn_id`, `child_turn_id`, `reason` |
| `agent.error` | agent 执行流程报告错误时 | `stage`, `error` |

### Channel

| 事件名 | 触发时机 | 主要详情 |
| ------ | -------- | -------- |
| `channel.lifecycle.initialized` | channel manager 根据配置创建并注册 channel 实例后 | `type`; scope 包含 `channel` |
| `channel.lifecycle.started` | channel `Start()` 成功，worker 已启动时；热重载新增 channel 也会触发 | `type` |
| `channel.lifecycle.start_failed` | channel `Start()` 失败时 | `type`, `error`; severity 为 `error` |
| `channel.lifecycle.stopped` | channel `Stop()` 成功后 | `type` |
| `channel.webhook.registered` | channel 的 webhook handler 被注册到共享 HTTP mux 时 | `type`; scope 包含 `channel` |
| `channel.webhook.unregistered` | channel 的 webhook handler 从共享 HTTP mux 移除时 | `type`; scope 包含 `channel` |
| `channel.message.outbound_queued` | outbound 文本或媒体消息被放入对应 channel worker 队列时 | `media`, `content_len`, `reply_to_message_id`; scope 来自原 inbound context |
| `channel.message.outbound_sent` | outbound 文本或媒体消息成功发送，或 placeholder edit 已处理响应时 | `media`, `content_len`, `message_ids`, `reply_to_message_id` |
| `channel.message.outbound_failed` | outbound 文本或媒体消息重试耗尽或遇到永久失败时 | `media`, `content_len`, `retries`, `error`, `reply_to_message_id`; severity 为 `error` |
| `channel.rate_limited` | channel worker 等待 rate limiter token 时被 context 取消，导致本次发送被限流/中断 | `media`, `content_len`, `error`, `reply_to_message_id`; severity 为 `warn` |

### Message Bus

| 事件名 | 触发时机 | 主要详情 |
| ------ | -------- | -------- |
| `bus.publish.failed` | inbound、outbound、media、audio 或 voice control 发布失败，或缺少必要 context 时 | `stream`, `error`; scope 尽量来自消息 context |
| `bus.close.started` | message bus 开始关闭时 | `drained` 通常为 `0` |
| `bus.close.drained` | close 期间等待队列 drain，并且 drain 到至少一条 buffered message 时 | `drained` |
| `bus.close.completed` | message bus 完成关闭时 | `drained` |

### Gateway

| 事件名 | 触发时机 | 主要详情 |
| ------ | -------- | -------- |
| `gateway.start` | gateway 完成 agent/runtime event bus/bootstrap 绑定后 | `duration_ms` |
| `gateway.ready` | gateway 服务、channel manager、HTTP 等关键服务启动完成后 | `duration_ms` |
| `gateway.shutdown` | gateway 开始关闭流程时 | 无固定 payload，可能只有 envelope 字段 |
| `gateway.reload.started` | 热重载开始执行时 | `duration_ms` |
| `gateway.reload.completed` | 热重载成功完成时 | `duration_ms` |
| `gateway.reload.failed` | 热重载失败时 | `duration_ms`, `error`; severity 为 `error` |

### MCP

| 事件名 | 触发时机 | 主要详情 |
| ------ | -------- | -------- |
| `mcp.server.connecting` | MCP manager 准备连接某个 server 前 | `server`, `type`, `url`, `command` |
| `mcp.server.connected` | MCP server 连接成功并完成工具列表初始化后 | `server`, `type`, `url`, `command`, `tool_count` |
| `mcp.server.failed` | MCP server 连接失败，或 manager 已关闭导致无法连接时 | `server`, `type`, `url`, `command`, `error`; severity 为 `error` |
| `mcp.tool.discovered` | MCP server 的某个工具被发现并注册时 | `server`, `type`, `url`, `command`, `tool` |
| `mcp.tool.call.start` | MCP tool wrapper 开始执行一次远端工具调用前 | `server`, `tool`; 如果在 agent turn 内触发，scope 会带上对应 turn/chat 信息 |
| `mcp.tool.call.end` | MCP tool wrapper 完成一次远端工具调用后，包括失败结果 | `server`, `tool`, `duration_ms`, `is_error`, `error` |

## 日志字段

所有事件日志都会尽量包含稳定 envelope 字段：

- `event_id`
- `event_kind`
- `severity`
- `event_time`
- `source_component`
- `source_name`
- `agent_id`
- `session_key`
- `turn_id`
- `channel`
- `account`
- `chat_id`
- `topic_id`
- `space_id`
- `space_type`
- `chat_type`
- `sender_id`
- `message_id`
- `trace_id`
- `parent_turn_id`
- `request_id`
- `reply_to_id`

agent 事件还会追加 payload 摘要字段：

| 事件 | 摘要字段 |
| ---- | -------- |
| `agent.turn.start` | `user_len`, `media_count` |
| `agent.turn.end` | `status`, `iterations_total`, `duration_ms`, `final_len` |
| `agent.llm.request` | `model`, `messages`, `tools`, `max_tokens` |
| `agent.llm.delta` | `content_delta_len`, `reasoning_delta_len` |
| `agent.llm.response` | `content_len`, `tool_calls`, `has_reasoning` |
| `agent.llm.retry` | `attempt`, `max_retries`, `reason`, `error`, `backoff_ms` |
| `agent.context.compress` | `reason`, `dropped_messages`, `remaining_messages` |
| `agent.session.summarize` | `summarized_messages`, `kept_messages`, `summary_len`, `omitted_oversized` |
| `agent.tool.exec_start` | `tool`, `args_count` |
| `agent.tool.exec_end` | `tool`, `duration_ms`, `for_llm_len`, `for_user_len`, `is_error`, `async` |
| `agent.tool.exec_skipped` | `tool`, `reason` |
| `agent.steering.injected` | `count`, `total_content_len` |
| `agent.follow_up.queued` | `source_tool`, `content_len` |
| `agent.interrupt.received` | `interrupt_kind`, `role`, `content_len`, `queue_depth`, `hint_len` |
| `agent.subturn.spawn` | `child_agent_id`, `label` |
| `agent.subturn.end` | `child_agent_id`, `status` |
| `agent.subturn.result_delivered` | `target_channel`, `target_chat_id`, `content_len` |
| `agent.subturn.orphan` | `parent_turn_id`, `child_turn_id`, `reason` |
| `agent.error` | `stage`, `error` |

## 可打印的事件域

当前 runtime event kind 定义在 `pkg/events/kind.go`。事件日志配置可以选择这些域：

- `agent.*`：agent turn、LLM、tool、context、steering、interrupt、subturn、error。
- `channel.*`：channel lifecycle、webhook 注册、outbound queued/sent/failed、rate limited。
- `bus.*`：publish failed、close started/drained/completed。
- `gateway.*`：start、ready、shutdown、reload started/completed/failed。
- `mcp.*`：server connecting/connected/failed、tool discovered、tool call start/end。

默认事件日志示例见 [`../../config/config.example.json`](../../config/config.example.json)。
