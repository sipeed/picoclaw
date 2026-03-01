# Plugin System Roadmap

This document defines how PicoClaw evolves from hook-based extension points to a fuller plugin system in low-risk phases.

## Current Status (Phase 0: Foundation)

Implemented in current hooks PR:

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

## Phase 1: Static Plugin Contract (Compile-time) — Implemented

Goal: define a minimal public plugin contract for Go modules.

Implemented:

- Add `pkg/plugin` with a small interface:
  - `Name() string`
  - `APIVersion() string`
  - `Register(*hooks.HookRegistry) error`
- Register plugins at startup in code.
- Add compatibility metadata (`plugin.APIVersion`) and registration-time checks.

Exit criteria (met):

- Example plugin module builds against the contract.
- Startup validation logs loaded plugins and registration errors clearly.

## Phase 2: Config-driven Enable/Disable — Implemented

Goal: operational control without code changes.

Implemented:

- Add typed plugin selection config in `config.json`:
  - `plugins.default_enabled`
  - `plugins.enabled`
  - `plugins.disabled`
- Add deterministic plugin resolution and conflict handling in the plugin manager.
- Wire resolved plugins into startup for both `agent` and `gateway` entrypoints.

Exit criteria (met):

- Users can toggle built-in plugins without rebuilding.
- Invalid plugin selection in config is surfaced during startup/lint flow.

## Phase 3: Metadata Introspection + CLI — Implemented

Goal: make plugin state inspectable and config validation straightforward.

Implemented:

- Add plugin metadata introspection in the plugin manager (internal API surface).
- Add CLI inspection commands:
  - `picoclaw plugin list`
  - `picoclaw plugin list --format json`
- Add CLI lint command:
  - `picoclaw plugin lint --config <path>`
- Add startup plugin resolution summary diagnostics:
  - `plugins_enabled`
  - `plugins_disabled`
  - `plugins_unknown_enabled`
  - `plugins_unknown_disabled`
  - `plugins_warnings`

Exit criteria (met):

- Operators can inspect plugin status in text/JSON outputs (`name`, `status`).
- Plugin metadata introspection is available via plugin manager APIs.
- Operators can validate plugin config before startup.

## Future DX Work (Post-Phase 3)

- Provide `examples/plugins/*` reference implementations.
- Publish plugin authoring guide (lifecycle map, best practices, safety constraints).
- Add plugin-focused test harness patterns for hook behavior verification.

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

The current hooks PR should be reviewed as Phase 0+1 only. It intentionally establishes extension points while avoiding high-risk runtime plugin mechanics.
