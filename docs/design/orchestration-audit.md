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
- `max_tasks_per_agent`
  - Enforced as max active (non-terminal) tasks per manager.
- `default_task_timeout_seconds`
  - Used as default deadline metadata in task ledger entries.
- `retry_limit_per_task`
  - Used by audit logic to detect failed tasks that still have retry budget.

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
  - `safe_only` records low-risk remediation actions in ledger.
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

## Backward Compatibility

- Existing tools and loop behavior remain unchanged when `audit.enabled=false`.
- New fields are additive and optional.
- `spawn` and `subagent` retain previous parameter contract; `agent_id` is additive for `subagent`.

