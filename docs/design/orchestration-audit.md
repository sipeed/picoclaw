# Orchestration and Audit Runtime Design

This document describes how PicoClaw's subagent orchestration and periodic audit pipeline work after the `orchestration` and `audit` extensions.

## Goals

- Align subagent orchestration semantics with OpenClaw-style multi-level delegation.
- Keep existing behavior backward compatible when features are disabled.
- Provide periodic checks for missed tasks, low-quality outputs, and execution inconsistencies.

## Config Keys and Runtime Effects

### `orchestration`

- `enabled`
  - Feature gate for orchestration controls. Existing subagent tools still work; limits are always read from this section.
- `max_spawn_depth`
  - Enforced in `SubagentManager.SpawnTask`.
  - Nested spawns are detected through `sender_id=subagent:<task-id>`.
  - When depth is exceeded, spawn is rejected with `max spawn depth reached`.
- `max_parallel_workers`
  - Enforced as max concurrent running tasks per manager.
- `tool_calls_parallel_enabled`
  - Enables parallel execution for eligible tool calls in one LLM turn.
- `max_tool_call_concurrency`
  - Bounded worker count for one tool-call batch (`<=0` means no explicit cap).
- `parallel_tools_mode`
  - `read_only_only` (default): only tools marked `parallel_read_only` are parallelized.
  - `all`: all tool calls are eligible only when the tool instance is concurrent-safe.
- `tool_parallel_overrides`
  - Optional per-tool override map.
  - Values:
    - `parallel_read_only`: force this tool to be parallel-eligible.
    - `serial_only`: force this tool to execute serially.
  - Overrides take precedence over built-in tool policy and mode defaults.
  - Overrides do not bypass instance safety checks.
- `max_tasks_per_agent`
  - Enforced as max active (non-terminal) tasks per manager.
- `default_task_timeout_seconds`
  - Used as default deadline metadata in task ledger entries.
- `retry_limit_per_task`
  - Used by audit logic to detect failed tasks that still have retry budget.
  - Also caps how many automatic remediation retries can be spawned for one task.

### `audit`

- `enabled`
  - Starts a background audit loop in `AgentLoop.Run`.
- `interval_minutes`
  - Periodic audit cadence.
- `lookback_minutes`
  - Task window scanned in each cycle.
- `min_confidence`
  - Threshold for supervisor model score.
- `inconsistency_policy`
  - `strict` mode flags completed tasks with no tool evidence.
- `auto_remediation`
  - `safe_only` records low-risk remediation actions in ledger (no retries).
  - `retry_missed` automatically spawns subagent retries for `missed` findings.
  - `retry_all` automatically spawns retries for `missed`, `quality`, and `inconsistency` findings.
  - `retry` is accepted as an alias for `retry_missed`.
- `max_auto_remediations_per_cycle`
  - Caps how many remediation tasks can be spawned in one audit cycle (prevents runaway loops).
- `remediation_cooldown_minutes`
  - Suppresses repeated remediation attempts for the same task within the cooldown window.
- `remediation_agent_id`
  - Optional agent id used to execute remediation retries (requires subagent allowlist when targeting a different agent).
- `notify_channel`
  - Destination for audit report:
    - `last_active`: last recorded user channel/chat.
    - `channel:chat_id`: explicit destination.
    - `channel`: uses last active chat id with an overridden channel.

### `audit.supervisor`

- `enabled`
  - Enables model-based review in addition to deterministic rule checks.
- `model.primary` / `model.fallbacks`
  - Model alias resolved through `model_list`.
- `temperature`, `max_tokens`
  - Passed into supervisor model calls.

## Task Ledger

`TaskLedger` persists orchestration records under:

- `<workspace>/tasks/ledger.json`

Each entry tracks:

- task identity and lineage (`id`, `parent_task_id`, `agent_id`)
- routing context (`origin_channel`, `origin_chat_id`)
- execution state and result (`status`, `result`, `error`)
- timing (`created_at_ms`, `updated_at_ms`, `deadline_at_ms`)
- evidence and remediation arrays

## Subagent Lifecycle

1. `spawn`/`sessions_spawn` creates a task entry (`created` event).
2. Manager resolves execution profile:
   - default provider/model/tools, or
   - target agent profile when `agent_id` is provided.
3. Tool loop runs and records per-tool traces.
4. Task emits final event (`completed`, `failed`, or `cancelled`).
5. System inbound message is published for main-loop post-processing.

## Parent/Child and Cascade Semantics

- Nested spawns capture `parent_task_id`.
- If a parent task fails or is cancelled, manager cancels descendants (task tree) by context cancellation.
- Descendants transition to cancelled when they observe context cancellation.

## Audit Rules

Deterministic checks:

- `missed`
  - planned task overdue
  - running task timeout
  - failed task with retry budget
- `quality`
  - completed task with empty result
- `inconsistency`
  - completed task with zero evidence in `strict` mode

Optional model checks:

- Supervisor model receives task JSON and returns structured score/issues.
- Findings are merged into deterministic report.

## Auto Remediation (Retry / Make-up)

When `audit.auto_remediation` is set to a retry mode (`retry_missed` / `retry_all`), the audit loop can automatically **spawn a subagent** to retry missed or low-quality tasks.

Behavior:

- The audit loop scans the task ledger for findings.
- For eligible findings, it spawns a new subagent task with stricter acceptance criteria.
- It records a remediation entry (`action=retry`) and increments `retry_count` on the original task.
- It respects:
  - `audit.max_auto_remediations_per_cycle` (per-cycle cap),
  - `audit.remediation_cooldown_minutes` (per-task cooldown),
  - `orchestration.retry_limit_per_task` (per-task retry budget).

Delivery:

- By default, remediation retries run on the default agent using the same provider/model as normal subagent tasks.
- If `audit.remediation_agent_id` is set, the retry is delegated to that agent id (requires subagent allowlist when targeting a different agent).
- The retry result is delivered back to the original task's `origin_channel/origin_chat_id` when available; otherwise it falls back to `audit.notify_channel`.

### Example Configuration

```json
{
  "orchestration": {
    "retry_limit_per_task": 10
  },
  "audit": {
    "enabled": true,
    "interval_minutes": 5,
    "lookback_minutes": 180,
    "auto_remediation": "retry_all",
    "max_auto_remediations_per_cycle": 3,
    "remediation_cooldown_minutes": 10,
    "notify_channel": "last_active"
  }
}
```

## Backward Compatibility

- Existing tools and loop behavior remain unchanged when `audit.enabled=false`.
- New fields are additive and optional.
- `spawn` and `subagent` retain previous parameter contract; `agent_id` is additive for `subagent`.
