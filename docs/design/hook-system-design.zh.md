# PicoClaw Hook 系统设计（基于 `refactor/agent`）

> 当前状态：本文是 hook 系统的早期设计记录。事件系统升级后，观察型 hook 的主路径已经切到
> `pkg/events.Event`、`RuntimeEventObserver` 和进程 hook 的 `hook.runtime_event`。
> 旧 `agent.Event`、`EventKind`、`hook.event` 兼容层已经删除。

## 背景

本设计围绕两个议题展开：

- `#1316`：把 agent loop 重构为事件驱动、可中断、可追加、可观测
- `#1796`：在 runtime event bus 稳定后，把 hooks 设计为事件 consumer，而不是重新发明一套事件模型

当前分支已经完成了第一步里的“事件系统基础”，但还没有真正的 hook 挂载层。因此这里的目标不是重新设计 event，而是在已有实现上补出一层可扩展、可拦截、可外挂的 HookManager。

## 外部项目对比

### OpenClaw

OpenClaw 的扩展能力分成三层：

- Internal hooks：目录发现，运行在 Gateway 进程内
- Plugin hooks：插件在运行时注册 hook，也在进程内
- Webhooks：外部系统通过 HTTP 触发 Gateway 动作，属于进程外

值得借鉴的点：

- 有“项目内挂载”和“项目外挂载”两种路径
- hook 是配置驱动，可启停
- 外部入口有明确的安全边界和映射层

不建议直接照搬的点：

- OpenClaw 的 hooks / plugin hooks / webhooks 是三套路由，PicoClaw 当前体量下会偏重
- HTTP webhook 更适合“事件进入系统”，不适合作为“可同步拦截 agent loop”的基础机制

### pi-mono

pi-mono 的核心思路更接近当前分支：

- 扩展统一为 extension API
- 事件分为观察型和可变更型
- 某些阶段允许 `transform` / `block` / `replace`
- 扩展代码主要是进程内执行
- RPC mode 把 UI 交互桥接到进程外客户端

值得借鉴的点：

- 不把“观察”和“拦截”混成一个接口
- 允许返回结构化动作，而不是只有回调
- 进程外通信只暴露必要协议，不把整个内部对象图泄露出去

## 当前分支现状

### 已有能力

当前分支已经具备 hook 系统的地基：

- `pkg/events` 定义 runtime event envelope、kind、scope、source、severity 和 fan-out bus
- `pkg/agent/event_payloads.go` 保留 agent domain payload
- agent domain payload 保留在 `pkg/agent/event_payloads.go`
- `pkg/agent/loop.go` 中的 `runTurn()` 已在 turn、llm、tool、interrupt、follow-up、summary 等节点发射事件
- `pkg/agent/steering.go` 已支持 steering、graceful interrupt、hard abort
- `pkg/agent/turn.go` 已维护 turn phase、恢复点、active turn、abort 状态

### 现有缺口

早期设计时的缺口包括 HookManager、Before/After LLM、Before/After Tool、审批型 hook
以及 sub-turn 接入。当前实现已经覆盖主 turn 的 HookManager、LLM/Tool 拦截和审批；
sub-turn 事件已接入 runtime event bus。

### 一个关键现实

`#1316` 文案里提到“只读并行、写入串行”的工具执行策略，但当前 `runTurn()` 实现已经先收敛成“顺序执行 + 每个工具后检查 steering / interrupt”。因此 hook 设计不应依赖未来的并行模型，而应该先兼容当前顺序执行，再为以后增加 `ReadOnlyIndicator` 留口子。

## 设计原则

- Hook 必须建立在 `pkg/events` runtime event bus 和 turn 上下文之上
- runtime event bus 负责广播，HookManager 负责拦截，两者职责分离
- 项目内挂载要简单，项目外挂载必须走 IPC
- 观察型 hook 不能阻塞 loop；拦截型 hook 必须有超时
- 先覆盖主 turn，不把 sub-turn 一次做满
- 不新增第二套用户事件命名系统，新观察点统一使用 `pkg/events.Kind`

## 总体架构

分成三层：

1. `pkg/events` runtime event bus
   负责广播只读事件，覆盖 agent、channel、gateway、bus、MCP 等运行时组件

2. `HookManager`
   负责管理 hook、排序、超时、错误隔离，并在 `runTurn()` 的明确检查点执行同步拦截

3. `HookMount`
   负责两种挂载方式：
   - 进程内 Go hook
   - 进程外 IPC hook

换句话说：

- runtime event bus 是“发生了什么”
- HookManager 是“谁能介入”
- HookMount 是“这些 hook 从哪里来”

## Hook 分类

不建议把所有 hook 都设计成 `OnEvent(evt)`。

建议拆成两类。

### 1. 观察型

只消费事件，不修改流程：

```go
type EventObserver interface {
    OnRuntimeEvent(ctx context.Context, evt events.Event) error
}
```

这类 hook 直接订阅 runtime event bus 即可。

适用场景：

- 审计日志
- 指标上报
- 调试 trace
- 将事件转发给外部 UI / TUI / Web 面板

### 2. 拦截型

只在少数明确节点触发，允许返回动作：

```go
type LLMInterceptor interface {
    BeforeLLM(ctx context.Context, req *LLMRequest) HookDecision[*LLMRequest]
    AfterLLM(ctx context.Context, resp *LLMResponse) HookDecision[*LLMResponse]
}

type ToolInterceptor interface {
    BeforeTool(ctx context.Context, call *ToolCall) HookDecision[*ToolCall]
    AfterTool(ctx context.Context, result *ToolResultView) HookDecision[*ToolResultView]
}

type ToolApprover interface {
    ApproveTool(ctx context.Context, req *ToolApprovalRequest) ApprovalDecision
}
```

这里的 `HookDecision` 统一支持：

- `continue`
- `modify`
- `deny_tool`
- `abort_turn`
- `hard_abort`

## 对外暴露的最小 hook 面

V1 不需要把所有 runtime event kind 都变成可拦截点。

建议只开放这些同步 hook：

- `before_llm`
- `after_llm`
- `before_tool`
- `after_tool`
- `approve_tool`

其余节点继续作为只读事件暴露：

- `agent.turn.start`
- `agent.turn.end`
- `agent.llm.request`
- `agent.llm.response`
- `agent.tool.exec_start`
- `agent.tool.exec_end`
- `agent.tool.exec_skipped`
- `agent.steering.injected`
- `agent.follow_up.queued`
- `agent.interrupt.received`
- `agent.context.compress`
- `agent.session.summarize`
- `agent.error`

`subturn_*` 在 V1 中保留名字，但不承诺一定触发，直到子 turn 迁移完成。

## 项目内挂载

内部挂载必须尽量低摩擦。

建议提供两种等价方式，底层都走 HookManager。

### 方式 A：代码显式挂载

```go
al.MountHook(hooks.Named("audit", &AuditHook{}))
```

适用于：

- 仓内内建 hook
- 单元测试
- feature flag 控制

### 方式 B：内建 registry

```go
func init() {
    hooks.RegisterBuiltin("audit", func() hooks.Hook {
        return &AuditHook{}
    })
}
```

启动时根据配置启用：

```json
{
  "hooks": {
    "builtins": {
      "audit": { "enabled": true }
    }
  }
}
```

这比 OpenClaw 的目录扫描更轻，也更贴合 Go 项目。

## 项目外挂载

这是本设计的硬要求。

建议 V1 采用：

- `JSON-RPC over stdio`

原因：

- 跨平台最简单
- 不依赖额外端口
- 非常适合“由 PicoClaw 启动一个外部 hook 进程”
- 比 HTTP webhook 更适合同步拦截

### 外部 hook 进程模型

PicoClaw 启动外部进程，并在其 stdin/stdout 上跑协议。

配置示例：

```json
{
  "hooks": {
    "processes": {
      "review-gate": {
        "enabled": true,
        "transport": "stdio",
        "command": ["uvx", "picoclaw-hook-reviewer"],
        "observe": ["turn_start", "turn_end", "tool_exec_end"],
        "intercept": ["before_tool", "approve_tool"],
        "timeout_ms": 5000
      }
    }
  }
}
```

### 协议边界

不要把内部 Go 结构体直接暴露给 IPC。

建议定义稳定的协议对象：

- `HookHandshake`
- `HookEventNotification`
- `BeforeLLMRequest`
- `AfterLLMRequest`
- `BeforeToolRequest`
- `AfterToolRequest`
- `ApproveToolRequest`
- `HookDecision`

其中：

- 观察型事件用 notification，fire-and-forget
- 拦截型事件用 request/response，同步等待

### 为什么是 stdio，而不是直接用 HTTP webhook

因为两者用途不同：

- HTTP webhook 更适合“外部系统向 PicoClaw 投递事件”
- stdio/RPC 更适合“PicoClaw 在 turn 内同步询问外部 hook 是否改写 / 放行 / 拒绝”

如果未来需要 OpenClaw 式 webhook，可以作为独立入口层，再把外部事件转成 inbound message 或 steering，而不是直接替代 hook IPC。

## Hook 执行顺序

建议统一排序规则：

- 先内建 in-process hook
- 再外部 IPC hook
- 同组内按 `priority` 从小到大执行

原因：

- 内建 hook 延迟更低，适合做基础规范化
- 外部 hook 更适合做审批、审计、组织级策略

## 超时与错误策略

### 观察型

- 默认超时：`500ms`
- 超时或报错：记录日志，继续主流程

### 拦截型

- `before_llm` / `after_llm` / `before_tool` / `after_tool`：默认 `5s`
- `approve_tool`：默认 `60s`

超时行为：

- 普通拦截：`continue`
- 审批：`deny`

这点应直接沿用 `#1316` 的安全倾向。

## 与当前分支的对接点

### 直接复用

- 事件定义：`pkg/agent/events.go`
- 事件广播：`pkg/agent/eventbus.go`
- 活跃 turn / interrupt / rollback：`pkg/agent/turn.go`
- 事件发射点：`pkg/agent/loop.go`

### 需要新增

- `pkg/agent/hooks.go`
  - Hook 接口
  - HookDecision / ApprovalDecision
  - HookManager

- `pkg/agent/hook_mount.go`
  - 内建 hook 注册
  - 外部进程 hook 注册

- `pkg/agent/hook_ipc.go`
  - stdio JSON-RPC bridge

- `pkg/agent/hook_types.go`
  - IPC 稳定载荷

### 需要改造

- `pkg/agent/loop.go`
  - 在 LLM 和 tool 关键路径前后插入 HookManager 调用

- `pkg/tools/base.go`
  - 可选新增 `ReadOnlyIndicator`

- `pkg/tools/spawn.go`
- `pkg/tools/subagent.go`
  - 先保留现状
  - 等 sub-turn 迁移后再接入 `subturn_*` hook

## 一个更贴合当前分支的数据流

### 观察链路

```text
runTurn() -> emitEvent() -> runtime event bus -> observers
```

### 拦截链路

```text
runTurn()
  -> HookManager.BeforeLLM()
  -> Provider.Chat()
  -> HookManager.AfterLLM()
  -> HookManager.BeforeTool()
  -> HookManager.ApproveTool()
  -> tool.Execute()
  -> HookManager.AfterTool()
```

也就是说：

- observer 不改变现有 `emitEvent()`
- interceptor 直接插在 `runTurn()` 热路径

## 用户可见配置

建议新增：

```json
{
  "hooks": {
    "enabled": true,
    "builtins": {},
    "processes": {},
    "defaults": {
      "observer_timeout_ms": 500,
      "interceptor_timeout_ms": 5000,
      "approval_timeout_ms": 60000
    }
  }
}
```

V1 不做复杂自动发现。

原因：

- 当前分支重点是把地基打稳
- 目录扫描、安装器、脚手架可以后置
- 先让仓内和仓外都能挂上去，比“管理体验完整”更重要

## 推荐的 V1 范围

### 必做

- HookManager
- in-process 挂载
- stdio IPC 挂载
- observer hooks
- `before_tool` / `after_tool` / `approve_tool`
- `before_llm` / `after_llm`

### 可后置

- hook CLI 管理命令
- hook 自动发现
- Unix socket / named pipe transport
- sub-turn hook 生命周期
- read-only 并行分组
- webhook 到 inbound message 的映射入口

## 分阶段落地

### Phase 1

- 引入 HookManager
- 支持 in-process observer + interceptor
- 先只接主 turn

### Phase 2

- 引入 `stdio` 外部 hook 进程桥
- 支持组织级审批 / 审计 / 参数改写

### Phase 3

- 把 `SubagentManager` 迁移到 `runTurn/sub-turn`
- 接通 `agent.subturn.spawn` / `agent.subturn.end` / `agent.subturn.result_delivered`

### Phase 4

- 视需求补 `ReadOnlyIndicator`
- 在主 turn 和 sub-turn 上统一只读并行策略

## 最终结论

最适合 PicoClaw 当前分支的方案，不是直接复制 OpenClaw 的 hooks，也不是完整照搬 pi-mono 的 extension system，而是：

- 以 `pkg/events` runtime event bus 为只读观察面
- 以新增 `HookManager` 为同步拦截面
- 项目内通过 Go 对象直接挂载
- 项目外通过 `stdio JSON-RPC` 进程通信挂载

这样做有三个好处：

- 和 `#1796` 一致，hooks 只是 runtime event bus 之上的消费层
- 和当前 `refactor/agent` 实现一致，不需要推翻已有事件系统
- 同时满足“仓内简单挂载”和“仓外进程通信挂载”两个硬需求
