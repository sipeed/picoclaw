# Agent File Rename Plan

## Goal

Unify `pkg/agent/` package file naming to resolve the `loop_*` prefix naming confusion and unclear responsibility boundaries.

## Change Overview

### File Renames (12 files)

| Original | New | Description |
|----------|-----|-------------|
| `loop.go` | `agent.go` | AgentLoop main body + lifecycle methods |
| `loop_message.go` | `agent_message.go` | Message handling and routing |
| `loop_outbound.go` | `agent_outbound.go` | Response publishing |
| `loop_event.go` | `agent_event.go` | Event system |
| `loop_command.go` | `agent_command.go` | Command processing |
| `loop_steering.go` | `agent_steering.go` | Steering message handling |
| `loop_transcribe.go` | `agent_transcribe.go` | Audio transcription |
| `loop_media.go` | `agent_media.go` | Media processing |
| `loop_mcp.go` | `agent_mcp.go` | MCP initialization |
| `loop_utils.go` | `agent_utils.go` | Utility functions |
| `loop_inject.go` | `agent_inject.go` | Dependency injection |
| `loop_turn.go` | `turn_coord.go` | Turn coordinator |

### File Merges (2 → 1)

| Original | New | Description |
|----------|-----|-------------|
| `turn.go` + `turn_exec.go` | `turn_state.go` | Turn-related type definitions |

## Final File Structure

```
pkg/agent/
├── agent.go              # AgentLoop + Run/Stop/Close lifecycle
├── agent_message.go     # Message processing
├── agent_outbound.go    # Response publishing
├── agent_event.go       # Event system
├── agent_command.go     # Command processing
├── agent_steering.go    # Steering
├── agent_transcribe.go  # Transcription
├── agent_media.go       # Media processing
├── agent_mcp.go         # MCP
├── agent_utils.go       # Utility functions
├── agent_inject.go      # Dependency injection
├── turn_coord.go       # runTurn + coordinator
├── turn_state.go       # turnState + turnExecution + Control + ToolControl + LLMPhase
├── pipeline.go         # Pipeline struct + NewPipeline
├── pipeline_setup.go
├── pipeline_llm.go
├── pipeline_execute.go
└── pipeline_finalize.go
```

## Naming Convention

| Prefix | Content | Example |
|--------|---------|---------|
| `agent_*` | AgentLoop method files | `agent_message.go`, `agent_event.go` |
| `turn_*` | Turn lifecycle related | `turn_coord.go`, `turn_state.go` |
| `pipeline_*` | Pipeline methods | `pipeline_setup.go`, `pipeline_llm.go` |
| `context_*` | Context management | `context_manager.go`, `context_legacy.go` |
| `hook_*` | Hook system | `hook_process.go`, `hook_mount.go` |

## Architecture Layers

```
┌─────────────────────────────────────────────────────────┐
│                    AgentLoop (agent.go)                │
│  - Message loop Run/Stop/Close                        │
│  - Dependency injection (agent_inject.go)             │
│  - Message routing (agent_message.go)                 │
│  - Response publishing (agent_outbound.go)            │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│              Turn Coordinator (turn_coord.go)          │
│  - runTurn(): main coordinator                         │
│  - abortTurn(): abort                                 │
│  - askSideQuestion(): side question                   │
│  - selectCandidates(): model selection                │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                 Pipeline (pipeline_*.go)               │
│  - SetupTurn(): initialization                        │
│  - CallLLM(): LLM call                               │
│  - ExecuteTools(): tool execution                     │
│  - Finalize(): finalization                          │
└─────────────────────────────────────────────────────────┘
```

## Verification Results

- ✅ `go build ./pkg/agent/...` - Pass
- ✅ `go vet ./pkg/agent/...` - No warnings
- ✅ `go test ./pkg/agent/... -skip "TestSeahorse|TestGlobalSkillFileContentChange"` - Pass
