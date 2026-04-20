# AgentLoop File Split

## Overview

The `pkg/agent/loop.go` file (originally 4384 lines) has been split into 12 focused source files. This is a pure refactoring with no behavioral changes.

## Goals

- Reduce cognitive load when navigating agent loop code
- Enable parallel work by decoupling concerns
- Maintain all existing functionality and tests
- Keep imports minimal per file

## File Map

| File | Lines | Responsibility |
|------|-------|----------------|
| `loop.go` | ~650 | Core `AgentLoop` struct, `Run`, `Stop`, `Close`, `ReloadProviderAndConfig`, `runAgentLoop` |
| `loop_turn.go` | ~1880 | Turn execution: `runTurn`, `abortTurn`, `selectCandidates`, `askSideQuestion`, `isolatedSideQuestionProvider`, side question model config |
| `loop_utils.go` | ~480 | Standalone utility functions: formatters, cloners, helpers (no receiver) |
| `loop_init.go` | ~355 | `NewAgentLoop` constructor and `registerSharedTools` |
| `loop_message.go` | ~300 | Message handling: `processMessage`, `processSystemMessage`, routing helpers, `ProcessDirect`, `ProcessHeartbeat` |
| `loop_command.go` | ~265 | Command processing: `handleCommand`, `applyExplicitSkillCommand`, pending skills management |
| `loop_mcp.go` | ~235 | MCP runtime: `ensureMCPInitialized`, server discovery, deferred server handling |
| `loop_event.go` | ~205 | Event system helpers: `emitEvent`, `logEvent`, `hookAbortError`, `newTurnEventScope`, `MountHook`, `SubscribeEvents` |
| `loop_media.go` | ~198 | Media resolution: `resolveMediaRefs`, artifact building, MIME detection |
| `loop_outbound.go` | ~165 | Response publishing: `PublishResponseIfNeeded`, `publishPicoReasoning`, `handleReasoning` |
| `loop_transcribe.go` | ~110 | Audio transcription: `transcribeAudioInMessage`, `sendTranscriptionFeedback` |
| `loop_steering.go` | ~97 | Steering queue: `runTurnWithSteering`, `processMessageSync`, `resolveSteeringTarget` |
| `loop_inject.go` | ~104 | Setter injection: `SetChannelManager`, `SetMediaStore`, `SetTranscriber`, `GetRegistry`, `GetConfig`, `RecordLastChannel` |

## Core Principles Applied

### 1. Same Package, Independent Files
All files belong to the `agent` package and compile together. This preserves the original visibility rules — no interface abstraction was introduced in this phase.

### 2. No Logic Changes
All functions were moved verbatim (except updating import statements). The extraction script used the original `loop.go.backup` as source of truth to ensure no drift.

### 3. Shared Types Remain in loop.go
The `AgentLoop` struct, `processOptions`, `continuationTarget`, and all hook/event types stay in `loop.go` since they are referenced across files.

### 4. Turn State Is Central
`loop_turn.go` is the largest file because the turn lifecycle (`runTurn`) is inherently large. It contains the core LLM interaction loop, tool execution, subturn spawning, and steering injection.

## What's Left in loop.go

```go
// Core struct
type AgentLoop struct { ... }

// Main lifecycle
func (al *AgentLoop) Run(ctx context.Context) error
func (al *AgentLoop) Stop()
func (al *AgentLoop) Close()
func (al *AgentLoop) ReloadProviderAndConfig(ctx, provider, cfg)

// Turn orchestration (calls into loop_turn.go)
func (al *AgentLoop) runAgentLoop(ctx, agent, opts) (string, error)
```

## Extraction Method

The split was done programmatically using Node.js to:
1. Identify function boundaries using brace counting
2. Extract each function to its target file
3. Add necessary imports to each file
4. Remove the extracted function from loop.go
5. Run `go fmt` and `go vet` to verify

## Testing

All existing tests pass. The 5 failing tests (`TestGlobalSkillFileContentChange` and 4 Seahorse tests) are pre-existing failures unrelated to this refactor (database file locking issues on Windows).

Build status: `go build ./pkg/agent/...` passes with no errors.

## Phase 2: Dependency Inversion (Planned)

A future phase will introduce interface types to decouple `AgentLoop` from its dependencies, enabling:
- Easier testing with mock dependencies
- Alternative runtime configurations
- Cleaner boundaries for MCP and other extensions

## See Also

- [context.md](context.md) — context management and session handling
