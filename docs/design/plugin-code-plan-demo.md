# Plugin System Demo Code Plan

## Goal
Build a minimal, executable demo to prove plugin value with measurable behavior (not conceptual discussion).

## Why this demo
This demo validates runtime capabilities that skill text cannot reliably enforce:
- deterministic tool-call blocking at runtime
- deterministic outbound content rewrite at runtime
- reversible behavior (no plugin config => no effect)

## Scope (1-day demo)
In scope:
- add one compile-time plugin: `policy-demo`
- add tests for:
  - global tool block
  - channel-specific tool allowlist
  - outbound redaction
  - outbound deny-pattern guard
  - tool argument normalization (timeout clamp)
  - hook-based audit counters
  - no-config no-effect path
- provide a reviewer-friendly verification command

Out of scope:
- dynamic plugin loading
- plugin marketplace/distribution
- UI/config schema work

## Code Changes
1. `pkg/plugin/demoplugin/policy_demo.go`
- add `PolicyDemoPlugin`
- register `before_tool_call` hook to block configured tools
- support channel-specific tool allowlist
- normalize timeout-like tool args (`timeout`, `timeout_seconds`)
- register `message_sending` hook for redaction and deny-pattern guard
- register `session_start`, `session_end`, `after_tool_call` audit hooks
- expose `Snapshot()` for deterministic verification

2. `pkg/plugin/demoplugin/policy_demo_test.go`
- `TestPolicyDemoPluginBlocksConfiguredTool`
- `TestPolicyDemoPluginRedactsOutboundContent`
- `TestPolicyDemoPluginChannelAllowlist`
- `TestPolicyDemoPluginOutboundGuard`
- `TestPolicyDemoPluginNormalizesTimeoutArg`
- `TestPolicyDemoPluginAuditHooks`
- `TestPolicyDemoPluginNoConfigNoEffect`

## Verification
Run:

```bash
go test ./pkg/plugin/... ./pkg/agent/...
```

Expected:
- all tests pass
- plugin test suite demonstrates deterministic runtime interception and rewrite

## Acceptance Criteria
- plugin can enforce a hard runtime policy (`before_tool_call` cancel)
- plugin can enforce channel-level runtime policy without core code changes
- plugin can enforce outbound transformation (`message_sending` rewrite)
- plugin can enforce outbound hard-block (`message_sending` cancel)
- plugin can mutate tool args before execution in a deterministic way
- plugin can collect lifecycle audit counters
- empty plugin config causes zero behavior change
- behavior is covered by automated tests

## Reviewer Notes
If this demo is accepted, next step is wiring a config-driven enable/disable path (Roadmap Phase 2), while keeping current compile-time contract (`pkg/plugin`) unchanged.
