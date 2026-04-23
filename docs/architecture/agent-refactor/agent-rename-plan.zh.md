# Agent 文件重命名计划

## 目标

统一 `pkg/agent/` 包的文件命名，解决 `loop_*` 前缀命名混乱、职责边界不清晰的问题。

## 变更概览

### 文件重命名（12 个）

| 原文件 | 新文件 | 说明 |
|--------|--------|------|
| `loop.go` | `agent.go` | AgentLoop 主体 + 生命周期方法 |
| `loop_message.go` | `agent_message.go` | 消息处理和路由 |
| `loop_outbound.go` | `agent_outbound.go` | 响应发布 |
| `loop_event.go` | `agent_event.go` | 事件系统 |
| `loop_command.go` | `agent_command.go` | 命令处理 |
| `loop_steering.go` | `agent_steering.go` | Steering 消息处理 |
| `loop_transcribe.go` | `agent_transcribe.go` | 音频转录 |
| `loop_media.go` | `agent_media.go` | 媒体处理 |
| `loop_mcp.go` | `agent_mcp.go` | MCP 初始化 |
| `loop_utils.go` | `agent_utils.go` | 工具函数 |
| `loop_inject.go` | `agent_inject.go` | 依赖注入 |
| `loop_turn.go` | `turn_coord.go` | Turn 协调器 |

### 文件合并（2 → 1）

| 原文件 | 新文件 | 说明 |
|--------|--------|------|
| `turn.go` + `turn_exec.go` | `turn_state.go` | Turn 相关类型定义 |

## 最终文件结构

```
pkg/agent/
├── agent.go              # AgentLoop + Run/Stop/Close 生命周期
├── agent_message.go     # 消息处理
├── agent_outbound.go    # 响应发布
├── agent_event.go       # 事件系统
├── agent_command.go     # 命令处理
├── agent_steering.go    # Steering
├── agent_transcribe.go  # 转录
├── agent_media.go       # 媒体处理
├── agent_mcp.go         # MCP
├── agent_utils.go       # 工具函数
├── agent_inject.go      # 依赖注入
├── turn_coord.go       # runTurn + 协调器
├── turn_state.go       # turnState + turnExecution + Control + ToolControl + LLMPhase
├── pipeline.go         # Pipeline struct + NewPipeline
├── pipeline_setup.go
├── pipeline_llm.go
├── pipeline_execute.go
└── pipeline_finalize.go
```

## 命名约定

| 前缀 | 内容 | 示例 |
|------|------|------|
| `agent_*` | AgentLoop 的方法文件 | `agent_message.go`, `agent_event.go` |
| `turn_*` | Turn 生命周期相关 | `turn_coord.go`, `turn_state.go` |
| `pipeline_*` | Pipeline 方法 | `pipeline_setup.go`, `pipeline_llm.go` |
| `context_*` | 上下文管理 | `context_manager.go`, `context_legacy.go` |
| `hook_*` | Hook 系统 | `hook_process.go`, `hook_mount.go` |

## 架构层次

```
┌─────────────────────────────────────────────────────────┐
│                    AgentLoop (agent.go)                │
│  - 消息循环 Run/Stop/Close                              │
│  - 依赖注入 (agent_inject.go)                           │
│  - 消息路由 (agent_message.go)                          │
│  - 响应发布 (agent_outbound.go)                         │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│              Turn Coordinator (turn_coord.go)           │
│  - runTurn(): 主协调器                                  │
│  - abortTurn(): 中止                                  │
│  - askSideQuestion(): 侧问                             │
│  - selectCandidates(): 模型选择                        │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                 Pipeline (pipeline_*.go)               │
│  - SetupTurn(): 初始化                                 │
│  - CallLLM(): LLM 调用                                │
│  - ExecuteTools(): 工具执行                            │
│  - Finalize(): 终结                                   │
└─────────────────────────────────────────────────────────┘
```

## 验证结果

- ✅ `go build ./pkg/agent/...` - 通过
- ✅ `go vet ./pkg/agent/...` - 无警告
- ✅ `go test ./pkg/agent/... -skip "TestSeahorse|TestGlobalSkillFileContentChange"` - 通过
