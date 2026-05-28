# Pipeline Restructuring Plan

## Goal

Split `agent/pipeline.go` (~1400 lines) into multiple logical files, organizing code by responsibility.

## Final File Structure

```
pkg/agent/
├── pipeline.go           # Pipeline struct + NewPipeline (~39 lines)
├── pipeline_setup.go   # SetupTurn method (~115 lines)
├── pipeline_llm.go     # CallLLM method (~519 lines)
├── pipeline_execute.go  # ExecuteTools method (~693 lines)
└── pipeline_finalize.go # Finalize method (~78 lines)
```

## Actual Line Counts

| File | Lines |
|------|-------|
| `pipeline.go` | 39 |
| `pipeline_setup.go` | 115 |
| `pipeline_llm.go` | 519 |
| `pipeline_execute.go` | 693 |
| `pipeline_finalize.go` | 78 |
| **Total** | **1444** |

## Responsibility Matrix

| File | Method | Responsibility |
|------|--------|----------------|
| `pipeline.go` | `Pipeline` struct, `NewPipeline()` | Pipeline dependency container |
| `pipeline_setup.go` | `SetupTurn()` | Turn initialization: history assembly, message building, candidate selection |
| `pipeline_llm.go` | `CallLLM()` | LLM call: PreLLM hooks, fallback, retry, AfterLLM hooks |
| `pipeline_execute.go` | `ExecuteTools()` | Tool execution: BeforeTool/ApproveTool/AfterTool hooks, media sending, steering handling |
| `pipeline_finalize.go` | `Finalize()` | Turn finalization: session save, compression, status setting |

## Relationship Between Pipeline and Turn Coordinator

```
AgentLoop (agent.go)
    │
    ├── runAgentLoop() ──────────────────┐
    │                                    │
    │    ┌───────────────────────────────▼───────────────────────────────┐
    │    │                    Turn Coordinator (turn_coord.go)           │
    │    │                                                           │
    │    │   runTurn() {                                             │
    │    │       exec = pipeline.SetupTurn()                          │
    │    │       loop {                                              │
    │    │           ctrl = pipeline.CallLLM()  ──► Pipeline (pipeline_*.go) │
    │    │           if ctrl == ToolLoop {                            │
    │    │               toolCtrl = pipeline.ExecuteTools()             │
    │    │           }                                                │
    │    │       }                                                    │
    │    │       return pipeline.Finalize()                            │
    │    │   }                                                         │
    │    └─────────────────────────────────────────────────────────────┘
    │
    └── Publish response (agent_outbound.go)
```

## Verification Results

- ✅ `go build ./pkg/agent/...` - Pass
- ✅ `go vet ./pkg/agent/...` - No warnings
- ✅ `go test ./pkg/agent/... -skip "TestSeahorse|TestGlobalSkillFileContentChange"` - Pass
