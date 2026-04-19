# AgentLoop File Split

> **Note:** This document describes the file split that was completed in a previous phase. The `loop_*` naming has since been renamed to `agent_*` and `turn_*`. See [agent-rename-plan.md](./agent-rename-plan.md) for the current file structure.

## Overview

The `pkg/agent/loop.go` file (originally 4384 lines) has been split into 12 focused source files. This is a pure refactoring with no behavioral changes.

## Goals

- Reduce cognitive load when navigating agent loop code
- Enable parallel work by decoupling concerns
- Maintain all existing functionality and tests
- Keep imports minimal per file

## Original File Map (Renamed in Phase 2)

| Old File | New File | Responsibility |
|----------|----------|----------------|
| `loop.go` | `agent.go` | Core `AgentLoop` struct, `Run`, `Stop`, `Close` |
| `loop_turn.go` | `turn_coord.go` + `pipeline_*.go` | Turn execution: coordinator + Pipeline methods |
| `loop_utils.go` | `agent_utils.go` | Standalone utility functions |
| `loop_init.go` | `agent_init.go` | `NewAgentLoop` constructor and tool registration |
| `loop_message.go` | `agent_message.go` | Message handling and routing |
| `loop_command.go` | `agent_command.go` | Command processing |
| `loop_mcp.go` | `agent_mcp.go` | MCP runtime |
| `loop_event.go` | `agent_event.go` | Event system helpers |
| `loop_media.go` | `agent_media.go` | Media resolution |
| `loop_outbound.go` | `agent_outbound.go` | Response publishing |
| `loop_transcribe.go` | `agent_transcribe.go` | Audio transcription |
| `loop_steering.go` | `agent_steering.go` | Steering queue |
| `loop_inject.go` | `agent_inject.go` | Setter injection |

## Current File Structure

See [agent-rename-plan.md](./agent-rename-plan.md) for the complete current file structure.

## Phase 2: Rename and Pipeline Restructuring

Phase 2 completed the following:

1. **File renaming**: All `loop_*` files renamed to `agent_*` or `turn_*`
2. **Turn state merging**: `turn.go` + `turn_exec.go` → `turn_state.go`
3. **Pipeline extraction**: Split large `runTurn` into Pipeline methods

### Pipeline Architecture

The Pipeline methods provide structured turn execution:

| Method | File | Responsibility |
|--------|------|----------------|
| `SetupTurn()` | `pipeline_setup.go` | History assembly, message building, candidate selection |
| `CallLLM()` | `pipeline_llm.go` | PreLLM hooks, fallback, retry, AfterLLM hooks |
| `ExecuteTools()` | `pipeline_execute.go` | Tool execution with hooks |
| `Finalize()` | `pipeline_finalize.go` | Session persistence, compression |

## Core Principles Applied

### 1. Same Package, Independent Files
All files belong to the `agent` package and compile together. This preserves the original visibility rules.

### 2. No Logic Changes
All functions were moved verbatim. The extraction preserved behavioral equivalence.

### 3. Shared Types in turn_state.go
The `turnState`, `turnExecution`, `Control`, `ToolControl`, and `LLMPhase` types are centralized in `turn_state.go`.

## Testing

All existing tests pass. The 5 failing tests (`TestGlobalSkillFileContentChange` and 4 Seahorse tests) are pre-existing failures unrelated to this refactor.

Build status: `go build ./pkg/agent/...` passes with no errors.

## See Also

- [agent-rename-plan.md](./agent-rename-plan.md) — Current file naming convention
- [context.md](context.md) — context management and session handling
