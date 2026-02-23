# Plugin System Roadmap

This document defines how PicoClaw evolves from hook-based extension points to a fuller plugin system in low-risk phases.

## Current Status (Phase 0: Foundation)

Implemented in current hooks MR:

- Typed lifecycle hooks (`pkg/hooks`)
- Priority-based handler ordering
- Cancellation support for modifying hooks
- Panic recovery and error isolation
- Agent-loop integration via `agentLoop.SetHooks(...)`

Compatibility:

- If no hooks are registered, runtime behavior is unchanged.
- No config migration is required.

## Non-Goals in Phase 0

- No dynamic runtime plugin loading
- No remote plugin marketplace/distribution
- No plugin sandboxing model
- No stable external plugin ABI yet
- No Go `.so` plugin loading as default direction

## Phase Plan

## Phase 1: Static Plugin Contract (Compile-time)

Goal: define a minimal public plugin contract for Go modules.

Proposed:

- Add `pkg/plugin` with a small interface:
  - `Name() string`
  - `Register(*hooks.HookRegistry) error`
- Register plugins at startup in code.
- Add compatibility metadata (`PluginAPIVersion`) for forward checks.

Exit criteria:

- Example plugin module builds against the contract.
- Startup validation logs loaded plugins and registration errors clearly.

## Phase 2: Config-driven Enable/Disable

Goal: operational control without code changes.

Proposed:

- Add plugin list/config in `config.json`:
  - enabled/disabled flags
  - optional plugin-specific settings
- Deterministic load order and conflict resolution rules.

Exit criteria:

- Users can toggle plugins without rebuilding.
- Clear startup diagnostics for invalid plugin config.

## Phase 3: Developer Experience

Goal: make third-party plugin development straightforward.

Proposed:

- Provide `examples/plugins/*` reference implementations.
- Publish plugin authoring guide (lifecycle map, best practices, safety constraints).
- Add plugin-focused test harness pattern for hook behavior verification.

Exit criteria:

- New plugin can be built from template with minimal boilerplate.
- CI examples demonstrate expected behavior and regression checks.

## Phase 4: Optional Dynamic Loading (Separate RFC)

Goal: support runtime-loaded plugins only if security and operability are acceptable.

Preferred direction:

- Runtime plugins run as subprocesses.
- Host and plugin communicate via RPC/gRPC.
- Host manages lifecycle (spawn/health/timeout/restart), not in-process dynamic loading.

Why this direction:

- Go native `.so` plugin loading has strict toolchain/ABI coupling with host binary.
- Subprocess RPC model reduces coupling and improves fault isolation.
- Process boundary provides a cleaner place for permissions and sandbox controls.

Preconditions:

- Threat model approved
- Signature/trust model defined
- Sandboxing and permission boundaries defined
- Rollback and safe-disable behavior validated
- Versioned RPC handshake and capability negotiation defined
- Process supervision policy defined (timeouts, retries, crash loop backoff)

Until then, compile-time registration remains the recommended model.

## Maintainer Review Notes

The current hooks MR should be reviewed as Phase 0 only. It intentionally establishes extension points while avoiding high-risk runtime plugin mechanics.
