# Agent Self-Evolution

Agent self-evolution lets PicoClaw learn from completed turns and turn repeated successful behavior into skill improvements. The runtime is controlled by the top-level `evolution` config block.

## Flow

The hot path runs at the end of an agent turn. When `evolution.enabled` is true, it records a learning record with the turn summary, success state, used skills, tool executions, and session/workspace metadata. Heartbeat turns are skipped.

The cold path groups related task records, checks the configured success threshold, and prepares skill drafts for patterns that have enough evidence. Drafts can target new skills or append/replace/merge existing workspace skills.

The apply path validates generated `SKILL.md` content before writing. Invalid drafts are rejected before a skill directory or file is created.

## Modes

| Mode | Behavior |
|------|----------|
| `observe` | Record learning data only. No cold-path draft generation runs automatically. |
| `draft` | Record learning data and generate candidate skill drafts when the cold path runs. |
| `apply` | Generate drafts and allow accepted drafts to update workspace skills. |

When `evolution.enabled` is false, `mode` is treated as disabled at runtime.

## Cold Path Trigger

`cold_path_trigger` only matters in `draft` and `apply` modes.

| Trigger | Behavior |
|---------|----------|
| `after_turn` | Run the cold path after eligible turns. |
| `scheduled` | Run the cold path at configured `cold_path_times`. |
| `manual` | Do not run automatically. There is no user-facing Web/API/CLI trigger yet; code can still invoke `Runtime.RunColdPathOnce`. |

`cold_path_times` uses `HH:MM` strings and is ignored unless the trigger is `scheduled`.

## State

By default, evolution state is stored under the workspace. `state_dir` can redirect that state to another directory. The state includes learning records, clustered pattern records, drafts, and skill profiles.

For user-facing configuration fields, see the [Configuration Guide](../guides/configuration.md#agent-self-evolution).
