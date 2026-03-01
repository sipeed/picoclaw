# Cross-Channel Command Centralization Design

## Background

This centralization is now implemented for generic slash commands.

- `pkg/commands` owns command definitions, channel support metadata, parser behavior, and handlers.
- `pkg/agent/loop.go` invokes `commands.Executor` for generic slash command execution before LLM flow.
- Channel adapters forward inbound text to the bus/agent path and do not consume generic commands locally.
- Telegram command menu registration still exists and is sourced from command definitions.

This document captures the resulting runtime policy and architecture.

## Goals

- Make generic command execution policy consistent across channels while keeping per-command channel support explicit in command definitions.
- Centralize command definition and execution in one domain (`pkg/commands`).
- Keep architecture extensible and easy to reason about: channel adapters do transport, agent does orchestration, commands do command policy + execution.

## Non-Goals

- No permission/role system redesign in this phase.
- No platform-specific slash command UX redesign (autocomplete, rich command metadata beyond current fields).
- No routing/session semantic change outside command execution flow.

## Confirmed Runtime Policies

- Unknown slash command (e.g. `/foo`) must pass through to LLM as normal user input.
- Registered command that is unsupported on current channel must return explicit user-facing error and stop further processing.

## Historical Root Cause Summary (Before Centralization)

The mismatch comes from mixed responsibilities:

1. `commands.Dispatcher` can identify commands by registry, but commands without handlers may be passed down.
2. Channel adapters use dispatcher results to decide local interception vs fallback.
3. `AgentLoop.handleCommand` contains an independent switch that executes command logic without using channel constraints from registry.

Result: command registry is not authoritative for behavior.

## Design Decision

Adopted **Agent-central execution with Commands-domain authority**:

- `pkg/commands` is the only source of command metadata, support scope, parser behavior, and handler execution.
- `AgentLoop` becomes an orchestrator that invokes `commands.Executor` before LLM execution.
- Channel packages stop owning business command execution; they only normalize inbound/outbound platform messages.
- Telegram command menu registration continues, driven by `pkg/commands` metadata.

## Architecture

### 1) Commands Domain (`pkg/commands`)

Introduce/normalize these concepts:

- `Definition`: command metadata + aliases + supported channels + handler.
- `Executor`: parses inbound text and returns a tri-state decision.
- `Runtime`: minimal capability interface injected by `AgentLoop` (session ops, route scope, config reads, reply sink).

Tri-state result contract:

- `handled`: command executed and response produced.
- `rejected`: command recognized but unsupported on this channel (explicit error).
- `passthrough`: unknown command or non-command; continue to normal LLM flow.

### 2) Agent Domain (`pkg/agent`)

`processMessage` flow becomes:

1. Resolve route + agent.
2. Build command runtime context (scopeKey/session services/config/channel/reply).
3. Call `commands.Executor.Execute(...)`.
4. Branch by result:
   - `handled` -> return response.
   - `rejected` -> return explicit error.
   - `passthrough` -> continue existing session+LLM path.

`AgentLoop` no longer owns business command switch logic.

### 3) Channel Domain (`pkg/channels/*`)

- Inbound text is published to bus as-is.
- No local business execution for generic commands.
- Telegram keeps asynchronous `RegisterCommands(...)`, sourcing command list from registry filtered for channel and visibility.

## Data Flow

1. Channel receives message.
2. Channel publishes `bus.InboundMessage`.
3. `AgentLoop.processMessage` resolves route/scope.
4. `commands.Executor` decides `handled/rejected/passthrough`.
5. Only `passthrough` continues to LLM.

This ensures one command behavior path for Telegram, WhatsApp, CLI, and Cron.

## API Shape (Sketch)

```go
type Outcome int

const (
	OutcomePassthrough Outcome = iota
	OutcomeHandled
	OutcomeRejected
)

type ExecuteResult struct {
	Outcome Outcome
	Command string
	Reply   string
	Err     error
}

type Runtime interface {
	Channel() string
	ScopeKey() string
	SessionOps() SessionOps
	Config() *config.Config
}
```

## Implemented Migration Strategy

1. Add executor tri-state contract in `pkg/commands` with tests.
2. Move `/new` and `/session` logic from `AgentLoop` into command handlers using runtime session ops.
3. Move `/show` and `/list` logic into command handlers.
4. Replace `AgentLoop` command switch with `commands.Executor` call.
5. Remove channel-side business execution fallback paths for generic commands.
6. Keep Telegram command registration driven by same registry.

## Testing Strategy

### Unit (commands)

- Parse normal slash and mention syntax.
- Registered+supported -> handled.
- Registered+unsupported -> rejected.
- Unknown slash -> passthrough.

### Agent integration

- Command handled path skips LLM.
- Rejected path returns explicit error and skips LLM.
- Passthrough path still hits LLM.
- `/new` and `/session` work consistently for non-Telegram channels.

### Channel regression

- Telegram startup command registration unaffected.
- WhatsApp/WhatsApp native no longer diverge due to local command execution.

## Acceptance Criteria (Implemented)

- Command behavior is determined only by `pkg/commands` definitions + executor.
- Generic command behavior (`/start`, `/help`, `/new`, `/session`, `/show`, `/list`) is executed via `commands.Executor` rather than channel-local business paths.
- Unknown slash commands continue to LLM.
- Unsupported registered commands return explicit errors.
- README command behavior statements align with runtime behavior.
