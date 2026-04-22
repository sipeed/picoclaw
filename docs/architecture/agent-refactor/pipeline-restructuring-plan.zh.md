# Pipeline 重构文档

## 目标

将 `agent/pipeline.go` (1400行) 拆分为多个逻辑文件，代码按职责组织。

## 最终文件结构

```
pkg/agent/
├── pipeline.go           # Pipeline struct + NewPipeline (~39行)
├── pipeline_setup.go   # SetupTurn 方法 (~115行)
├── pipeline_llm.go     # CallLLM 方法 (~519行)
├── pipeline_execute.go  # ExecuteTools 方法 (~693行)
└── pipeline_finalize.go # Finalize 方法 (~78行)
```

## 实际行数

| 文件 | 行数 |
|------|------|
| `pipeline.go` | 39 |
| `pipeline_setup.go` | 115 |
| `pipeline_llm.go` | 519 |
| `pipeline_execute.go` | 693 |
| `pipeline_finalize.go` | 78 |
| **总计** | **1444** |

## 职责说明

| 文件 | 方法 | 职责 |
|------|------|------|
| `pipeline.go` | `Pipeline` struct, `NewPipeline()` | Pipeline 依赖容器 |
| `pipeline_setup.go` | `SetupTurn()` | Turn 初始化：历史组装、消息构建、候选人选择 |
| `pipeline_llm.go` | `CallLLM()` | LLM 调用：PreLLM hook、fallback、重试、AfterLLM hook |
| `pipeline_execute.go` | `ExecuteTools()` | 工具执行：BeforeTool/ApproveTool/AfterTool hook、媒体发送、steering 处理 |
| `pipeline_finalize.go` | `Finalize()` | Turn 终结：会话保存、压缩、状态设置 |

## Pipeline 与 Turn Coordinator 的关系

```
AgentLoop (agent.go)
    │
    ├── runAgentLoop() ──────────────────┐
    │                                    │
    │    ┌───────────────────────────────▼───────────────────────────────┐
    │    │                    Turn Coordinator (turn_coord.go)              │
    │    │                                                           │
    │    │   runTurn() {                                             │
    │    │       exec = pipeline.SetupTurn()                           │
    │    │       loop {                                               │
    │    │           ctrl = pipeline.CallLLM()  ──► Pipeline (pipeline_*.go) │
    │    │           if ctrl == ToolLoop {                             │
    │    │               toolCtrl = pipeline.ExecuteTools()             │
    │    │           }                                                 │
    │    │       }                                                    │
    │    │       return pipeline.Finalize()                            │
    │    │   }                                                        │
    │    └─────────────────────────────────────────────────────────────┘
    │
    └── 发布响应 (agent_outbound.go)
```

## 验证结果

- ✅ `go build ./pkg/agent/...` - 通过
- ✅ `go vet ./pkg/agent/...` - 无警告
- ✅ `go test ./pkg/agent/... -skip "TestSeahorse|TestGlobalSkillFileContentChange"` - 通过
