# Agent Self-Evolution Implementation Status and PR Summary

## 1. Goal

This implementation connects the self-evolving skill design into PicoClaw and focuses on a minimal, safe, and modular loop:

1. the agent records learnable signals after a task finishes
2. the system organizes those signals in the background or on demand
3. the system generates candidate skill drafts
4. a human reviews and decides whether to apply them
5. applied skills keep version history, support rollback, and can cool down over time

The goal of this batch is to complete the data flow, lifecycle, and operational entry points, not to jump straight to a fully automatic self-updating system.

## 2. Scope Implemented in This Batch

The current code covers the main objects and execution paths defined by the design:

1. `Learning Record`
2. `Skill Draft`
3. `Skill Profile`
4. hot-path `task` record writing
5. cold-path `pattern` aggregation
6. draft generation
7. draft review and quarantine
8. skill apply, backup, and rollback
9. learned skill lifecycle maintenance
10. CLI operations

Against the design document's implementation scope, this batch completes the core self-evolution loop. Remaining items are mostly automation or UX improvements.

## 3. How the System Connects to PicoClaw

### 3.1 Integration Points

The feature is integrated as a separate module with three layers:

1. an event bridge in `pkg/agent`
2. the self-evolution core in `pkg/evolution`
3. CLI management commands in `cmd/picoclaw/internal/evolution`

This keeps the integration modular:

1. runtime switches are centralized in `config.EvolutionConfig`
2. evolution state is stored separately and does not directly pollute the regular skill layout
3. drafts, rollback, and lifecycle actions can be managed independently through the CLI
4. the whole feature can be disabled with `enabled: false`

### 3.2 Runtime Flow

The current runtime flow is:

1. the agent finishes a task turn
2. `pkg/agent/evolution_bridge.go` listens for the `TurnEnd` event
3. the hot path writes a `Learning Record(kind=task)`
4. if `auto_run_cold_path` is enabled, it schedules one cold-path run
5. the cold path aggregates multiple `task` records into a `pattern`
6. the draft generator creates a `Skill Draft`
7. draft review performs structural validation and sensitive-content scanning
8. valid drafts move to `candidate`, invalid drafts move to `quarantined`
9. a human uses CLI `review` and `apply` to decide whether to publish the skill change
10. after apply, the system updates the `Skill Profile`, keeps backups, and supports later `rollback` or `prune`

### 3.3 Disable Path

The feature now uses a single `enabled` switch. There is no separate `off` mode.

When `evolution.enabled = false`:

1. the agent does not write evolution records
2. the cold path does not run
3. normal skill loading and normal task execution remain unaffected

## 4. Core Capabilities Already Completed

### 4.1 Hot-Path Recording

Completed:

1. a `Learning Record(kind=task)` is written after each task turn
2. it records workspace, turn, session, tool kinds, active skills, attempted skills, and related context
3. it records skill attempt trajectory signals:
   `AttemptedSkills`
   `FinalSuccessfulPath`
   `SkillContextSnapshots`
4. it can capture the pattern where many skills were tried and the final successful path came from a later-added skill
5. the hot path only writes records and does not modify formal skills

This gives the system the basic ability to remember which skill path actually solved the task.

### 4.2 Cold-Path Organization

Completed:

1. multiple `task` records can be aggregated into `pattern` records
2. the system uses a minimum sample count and success-rate threshold before learning
3. it prefers the final successful path when extracting a stable winning path
4. it captures late-added skill hints and the final snapshot trigger

The cold path is responsible for turning task-level observations into reusable patterns without slowing down the user's live interaction.

### 4.3 Draft Generation

Completed:

1. LLM-based draft generation
2. provider-backed generation is used first when a provider is available
3. local fallback generation is used automatically when the provider is unavailable, errors, or returns invalid content
4. support for `create`, `append`, `replace`, and `merge`
5. fallback generation now prefers `append` when extending an existing skill

`merge` is used to merge newly learned stable knowledge into an existing skill instead of replacing it too aggressively.

### 4.4 Draft Review and Quarantine

Completed:

1. draft states:
   `candidate`
   `quarantined`
   `accepted`
2. structurally invalid drafts are quarantined
3. secret-like or sensitive content is quarantined
4. invalid skill names are quarantined
5. quarantined drafts never go directly into formal skills

This keeps generation separate from publication and prevents direct promotion of bad drafts.

### 4.5 Skill Apply and Rollback

Completed:

1. backups are saved before formal apply
2. failed applies can roll back automatically
3. `Skill Profile.version_history` records version changes
4. `rollback <skill-name>` restores the latest backup
5. if apply fails or profile persistence fails, the skill file is restored and audit information is recorded

Current rollback is structure-level and file-level rollback, with the primary goal of preventing broken skill files from being left behind.

### 4.6 Lifecycle Maintenance

Completed:

1. lifecycle states:
   `active`
   `cold`
   `archived`
   `deleted`
2. usage count, last-used time, and retention score tracking
3. `prune` recomputes lifecycle states
4. recently reused cold skills can become active again
5. profiles and statistics remain workspace-scoped even when using a shared `state_dir`

### 4.7 Human Review CLI

Completed CLI commands:

1. `picoclaw evolution drafts`
2. `picoclaw evolution review <draft-id>`
3. `picoclaw evolution apply <draft-id>`
4. `picoclaw evolution rollback <skill-name>`
5. `picoclaw evolution status`
6. `picoclaw evolution run-once`
7. `picoclaw evolution prune`

`review` now shows:

1. draft metadata
2. target skill profile summary
3. recent version history
4. impact preview
5. current body
6. rendered body
7. `diff_preview`

`diff_preview` uses a unified diff style so human review is easier.

## 5. Current Configuration

The feature is configured through `EvolutionConfig` in `pkg/config/config.go`:

```json
{
  "evolution": {
    "enabled": false,
    "mode": "observe",
    "state_dir": "",
    "min_case_count": 3,
    "min_success_rate": 0.7,
    "auto_run_cold_path": false,
    "auto_apply": false
  }
}
```

Field meanings:

1. `enabled`
   The master switch. When false, the whole self-evolution system is off.
2. `mode`
   Currently supports `observe`, `review`, and `apply`.
   `observe` focuses on recording and candidate generation.
   `review` keeps the flow in the human review path.
   `apply` allows the cold path to enter apply logic, while actual automatic apply is still controlled by `auto_apply`.
3. `state_dir`
   Stores self-evolution state separately from the skill working tree.
4. `min_case_count`
   Minimum task sample count before a reusable pattern is considered mature enough.
5. `min_success_rate`
   Minimum success rate required before a pattern is considered stable enough to learn from.
6. `auto_run_cold_path`
   Whether to trigger cold-path processing automatically after each task turn.
7. `auto_apply`
   Whether qualified drafts should be written into formal skills automatically when apply mode is active.

## 6. How “Human Confirmation Before Apply” Works Today

There is currently no interactive approval popup inside the agent runtime.

Human confirmation is implemented through the CLI flow:

1. the cold path first generates a `candidate` draft
2. a human runs `picoclaw evolution review <draft-id>` to inspect it
3. a human runs `picoclaw evolution apply <draft-id>` to publish it

In other words, manual confirmation exists today as an explicit CLI review flow, not as a live turn-time prompt.

## 7. What Is Done vs. What Is Still Deferred

### 7.1 Design Goals Already Covered

The current implementation already covers:

1. the full `Learning Record -> Skill Draft -> Skill Profile -> formal Skill` chain
2. hot path writes only `task`
3. cold path aggregates `pattern`
4. cold-path LLM draft generation
5. shortcut and winning-path learning
6. structural validation
7. sensitive-content scanning
8. backup
9. rollback
10. candidate and quarantined states
11. lifecycle state maintenance

### 7.2 Items Not Included in This PR

The following are still future improvements and are not blockers for this PR:

1. cross-workspace self-evolution
2. fully automatic merge into formal skills without human review
3. behavior-level rollback
4. full UI
5. hot-path LLM rerank

Clarifications:

1. “fully automatic merge”
   means the system would directly merge a qualified draft into the formal skill without a human running `review` and `apply`.
   The current implementation still prioritizes human review.
2. “behavior-level rollback”
   means rollback would be triggered by later evidence of degraded real task behavior, reduced hit rate, or regression, not only by structural apply failures.
   The current implementation mainly handles structure-level rollback and apply-failure rollback.
3. “hot-path LLM rerank”
   means using an LLM before execution to rerank recalled skills and choose the most likely successful one first.
   This is intentionally not part of the current hot path, so it does not add that extra inference cost yet.

## 8. Best PR Boundary for the Current State

This PR is best framed as the first complete batch of:

1. self-evolution infrastructure
2. hot-path task learning loop
3. cold-path draft generation
4. lifecycle and operations CLI

That boundary works well because:

1. the feature is runnable, testable, and rollback-safe
2. the master switch is clear and easy to disable
3. evolution state and formal skill ownership stay clearly separated
4. the human review path is already complete
5. future automation improvements can land in later PRs independently

## 9. Verification

The following test commands passed:

```bash
/usr/local/go/bin/go test ./cmd/picoclaw/internal/evolution -count=1
/usr/local/go/bin/go test ./pkg/evolution -count=1
/usr/local/go/bin/go test ./pkg/agent -count=1
```

This batch also includes test coverage for:

1. hot-path record writing
2. attempted-skill and final-success-path extraction
3. cold-path aggregation and draft generation
4. LLM generator vs. fallback generator selection
5. draft quarantine
6. apply and rollback
7. lifecycle and prune
8. workspace isolation
9. CLI output and audit details

## 10. Main Change Areas

### 10.1 Core Implementation

1. `pkg/evolution/`
   Runtime, storage, aggregation, drafts, review, rollback, lifecycle, and preview logic.

### 10.2 Agent Integration

1. `pkg/agent/evolution_bridge.go`
   Connects agent turn-end events to the self-evolution runtime.
2. `pkg/agent/events.go`
3. `pkg/agent/turn_state.go`
4. `pkg/agent/turn_coord.go`

### 10.3 CLI

1. `cmd/picoclaw/internal/evolution/`
   Operational commands for self-evolution.

### 10.4 Config and Supporting Integration

1. `pkg/config/config.go`
2. `pkg/config/defaults.go`
3. `pkg/config/config_test.go`
4. `cmd/picoclaw/main.go`
5. `pkg/skills/loader.go`
6. `pkg/gateway/gateway.go`

## 11. Suggested PR Title

Recommended title:

`feat: add modular agent self-evolution foundation, drafts, lifecycle, and CLI`

Alternative if you want a shorter “first loop” framing:

`feat: add first self-evolution loop for learned skills`

## 12. Suggested PR Description

The following text can be used directly as the PR body:

~~~md
## Summary

This PR adds the first modular self-evolution loop for PicoClaw skills.

It introduces:

1. hot-path task learning records
2. cold-path pattern aggregation
3. LLM-backed and fallback draft generation
4. draft review states (`candidate`, `quarantined`, `accepted`)
5. skill apply / backup / rollback
6. skill lifecycle maintenance (`active`, `cold`, `archived`, `deleted`)
7. workspace-scoped evolution state isolation
8. CLI commands for review and operations

## What is included

1. `Learning Record`, `Skill Draft`, and `Skill Profile`
2. `task -> pattern -> draft -> skill` main pipeline
3. learning from attempted skills and final successful paths
4. diff-based review output for human inspection
5. modular config gate via `evolution.enabled`
6. cold-path auto-run and optional auto-apply controls

## CLI

1. `picoclaw evolution drafts`
2. `picoclaw evolution review <draft-id>`
3. `picoclaw evolution apply <draft-id>`
4. `picoclaw evolution rollback <skill-name>`
5. `picoclaw evolution status`
6. `picoclaw evolution run-once`
7. `picoclaw evolution prune`

## Safety

1. no skill file is modified on the hot path
2. drafts are validated before acceptance
3. suspicious or invalid drafts are quarantined
4. apply keeps backups and supports rollback
5. evolution can be fully disabled with `evolution.enabled = false`

## Verification

```bash
/usr/local/go/bin/go test ./cmd/picoclaw/internal/evolution -count=1
/usr/local/go/bin/go test ./pkg/evolution -count=1
/usr/local/go/bin/go test ./pkg/agent -count=1
```

## Not included in this PR

1. cross-workspace evolution
2. behavior-level rollback
3. fully automatic merge without human review
4. full UI
5. hot-path LLM rerank
~~~
