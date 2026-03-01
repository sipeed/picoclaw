# PR #473 Phase 2/3 Rethink

## What Changed

The previous draft mixed control-plane work with schema-heavy plugin config too early.
This rethink narrows scope so implementation matches the current codebase:

- Phase 2 is only plugin selection and runtime wiring.
- Phase 3 is introspection, linting, and optional plugin-specific settings.
- `plugin.Plugin` stays unchanged in both phases.

## Hard Scope

- In-process built-in plugins only.
- No runtime loader, no dynamic module loading, no hot reload.
- Runtime loading remains Phase 4 and must reuse Phase 2/3 abstractions.

## Baseline

- Baseline is PR #473 phase-0/phase-1 behavior (`pkg/plugin`, `pkg/hooks`, `pkg/agent/loop.go`).
- Existing deployments without `plugins` config must keep the same effective behavior.

## Implemented Snapshot

Implemented in current Phase 2/3 scope:

- Phase 2:
- typed plugin config schema with `plugins.default_enabled`, `plugins.enabled`, `plugins.disabled`.
- deterministic plugin resolver.
- startup wiring in both `agent` and `gateway` paths.
- Phase 3:
- plugin metadata introspection in manager APIs for built-ins.
- CLI support for listing and linting plugin config (`list` output is `name` + `status`).

Command examples:

```bash
picoclaw plugin list
picoclaw plugin list --format json
picoclaw plugin lint --config ~/.picoclaw/config.json
```

Precision note:

- No dynamic runtime plugin loading/hot reload.
- Plugin-specific `settings` remain optional future work and are not required by the implemented Phase 2 selection plane.

## Phase 2: Selection Plane (Minimal, Deterministic)

### Goal

Make plugin enable/disable operational from config with deterministic behavior and fail-fast handling.

### Config (Phase 2 only)

Project-facing examples should follow the repo default config format (JSON).

```json
{
  "plugins": {
    "default_enabled": true,
    "enabled": ["policy-demo"],
    "disabled": ["legacy_policy"]
  }
}
```

No plugin-specific `settings` in Phase 2.

### Resolution Rules (authoritative)

For each built-in plugin name in sorted order:

1. normalize names (`trim`, `lowercase`) for matching.
2. if in `disabled`, mark disabled.
3. else if `enabled` list is non-empty, enable only if listed.
4. else enable only if `default_enabled=true`.

### Error Policy (Phase 2)

- unknown name in `enabled`: startup error.
- unknown name in `disabled`: warning only.
- duplicates after normalization: dedupe and warn.
- overlap between `enabled` and `disabled`: disabled wins.

### Required Code Changes

- `pkg/config/config.go`, `pkg/config/defaults.go`
  - add typed `plugins` block (`default_enabled`, `enabled`, `disabled`).
- `pkg/plugin/manager.go`
  - add built-in registry map and deterministic resolver.
  - expose resolution result buckets (`enabled`, `disabled`, `unknown`).
- `cmd/picoclaw/internal/agent/helpers.go`
  - resolve plugins from config and wire into `loop.EnablePlugins(...)`.
- `cmd/picoclaw/internal/gateway/helpers.go`
  - same as agent path.
- `pkg/agent/loop.go`
  - keep existing plugin interface and lifecycle.
  - add startup diagnostics with final resolved plugin names.

### PR Slicing (Review-Friendly)

Keep Phase 2 and Phase 3 as separate PR series.

1. Phase 2 PR-A: config schema + resolver only.
2. Phase 2 PR-B: agent/gateway startup wiring + startup diagnostics.
3. Phase 2 PR-C: tests + docs updates.
4. Phase 3 PR-A: manager introspection + metadata side interface.
5. Phase 3 PR-B: `plugin list` and `plugin lint` commands.
6. Phase 3 PR-C: observability polishing + integration tests + docs.

### Phase 2 Acceptance Gate

- No `plugins` block: behavior matches baseline.
- Unknown name in `enabled` fails startup with actionable message.
- Resolution order and result are deterministic.
- Entry points actually wire resolved plugins (no silent no-op).
- Startup logs show enabled/disabled plugin sets.

### Phase 2 Test Matrix

1. No `plugins` block.
- Expected: same enabled plugin set as baseline.
- Expected startup summary keys: `plugins_enabled`, `plugins_disabled`, `plugins_unknown_enabled`, `plugins_unknown_disabled`, `plugins_warnings`.
2. `enabled=["policy-demo"]`, empty `disabled`.
- Expected: only `policy-demo` loaded.
- Expected startup summary key examples: `plugins_enabled=["policy-demo"]`, `plugins_disabled=[]`.
3. `disabled=["policy-demo"]`, empty `enabled`.
- Expected: `policy-demo` not loaded.
- Expected startup summary key examples: `plugins_enabled=[]`, `plugins_disabled=["policy-demo"]`.
4. Overlap: `enabled=["policy-demo"]`, `disabled=["policy-demo"]`.
- Expected: plugin disabled (disabled wins).
- Expected startup summary key examples: `plugins_enabled=[]`, `plugins_disabled=["policy-demo"]`.
5. Unknown in `enabled`: `enabled=["not_exists"]`.
- Expected: startup fails.
- Expected error text includes: `unknown plugin in enabled`.
6. Unknown in `disabled`: `disabled=["not_exists"]`.
- Expected: startup continues with warning.
- Expected warning text includes: `unknown plugin in disabled`.
7. Duplicates/case variants: `enabled=["Policy_Demo","policy_demo"]`.
- Expected: deduped after normalization and warning emitted.
- Expected warning text includes: `duplicate plugin name after normalization`.

## Phase 3: Introspection Plane (DX + Validation)

### Goal

Make plugin state inspectable and config validation review-friendly.

### Capabilities

- Manager introspection:
  - `DescribeAll() []PluginInfo`
  - `DescribeEnabled() []PluginInfo`
- Optional metadata side interface for plugins:

```go
type PluginDescriptor interface {
	Info() PluginInfo
}
```

- CLI:
  - `picoclaw plugin list` (text/json).
  - `picoclaw plugin lint --config <path>`.
- Diagnostics (implemented now):
  - startup summary fields emitted by entrypoints:
    - `plugins_enabled`
    - `plugins_disabled`
    - `plugins_unknown_enabled`
    - `plugins_unknown_disabled`
    - `plugins_warnings`
- Diagnostics (deferred):
  - per-hook invocation outcome fields (`plugin`, `hook`, `result`, `duration_ms`).

### Optional Phase 3 Extension

If needed after list/lint lands, introduce plugin-specific `settings` with strict schema validation.
This is explicitly Phase 3, not Phase 2.

### Phase 3 Acceptance Gate

- `plugin list` output is stable in text and JSON with fields: `name`, `status`.
- `plugin lint` returns non-zero on invalid plugin names/config.
- Startup diagnostics include plugin resolution summary fields listed above.
- Tests cover one disabled path and one lint failure path (per current command contracts).

### Phase 3 Test Matrix

1. `picoclaw plugin list` text output.
- Expected: deterministic ordering and fields: `name`, `status`.
2. `picoclaw plugin list --format json`.
- Expected: stable JSON schema and deterministic ordering (`name`, `status` only).
3. `picoclaw plugin lint --config <path>` valid config.
- Expected: exit code `0`.
4. `picoclaw plugin lint --config <path>` invalid plugin name.
- Expected: non-zero exit.
- Expected error text includes: `unknown plugin`.

## Rollback Runbook

1. Revert to baseline behavior.
- Action: remove `plugins` block from config and restart process.
- Success signal: startup summaries return to expected enabled/disabled sets.
2. Recover from bad selection config.
- Action: clear `plugins.enabled` and `plugins.disabled`, restart.
- Success signal: startup summaries show valid resolution and no startup error.
3. Recover from Phase 3 command regressions.
- Action: disable plugin command surface with feature flag and restart.
- Success signal: `plugin` command group hidden/disabled in CLI help.
4. Incident confirmation checks.
- Verify startup logs include:
- `plugins_enabled`
- `plugins_disabled`
- `plugins_unknown_enabled`
- `plugins_unknown_disabled`
- `plugins_warnings`

## Why This Is More Sound

- Matches current interfaces (`EnablePlugins` with concrete plugin instances).
- Avoids premature schema coupling before metadata/lint tooling exists.
- Eliminates silent rollout risk by making entrypoint wiring a Phase 2 gate.
- Keeps a clean migration path to Phase 4 runtime sources.

## External Alignment (Informational)

This phase split follows a common Go OSS progression:
- compile-time plugin selection first
- discovery/validation CLI second
- runtime loader and trust model last

These references are context only (not proof of direct feature parity with PicoClaw implementation):

Reference patterns:
- Go `plugin` package caveats: `https://pkg.go.dev/plugin`
- HashiCorp `go-plugin` runtime model: `https://pkg.go.dev/github.com/hashicorp/go-plugin`
- module listing/validation command patterns (Caddy/Terraform style):
  - `https://caddyserver.com/docs/command-line`
  - `https://developer.hashicorp.com/terraform/cli/commands/providers/schema`
