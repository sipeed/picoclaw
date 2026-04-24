# Agent Self-Evolution Skill Lifecycle Design

## Summary

This design adds a lightweight self-evolution loop for PicoClaw that can learn reusable workflows from completed tasks without slowing down the user's main interaction path.

The system separates short-lived learning evidence from durable skills:

- `Learning Note`: a tiny structured note written after a task.
- `Learning Topic`: a grouped pattern of related notes.
- `Skill Draft`: a candidate skill change proposed from a mature topic.
- `Skill Profile`: lifecycle metadata for a formal workspace skill.

The design is workspace-scoped in v1, uses graded governance, keeps heavy learning work off the hot path, and supports candidate review, merge/replace decisions, rollback, and eventual cold/archive/delete handling for stale skills.

## Goals

- Let the agent learn reusable workflows from real tasks.
- Allow the system to learn "shortcut" strategies from repeated trial-and-error traces.
- Avoid increasing normal turn latency or prompt token cost in the common path.
- Keep learned behavior governable through candidates, validation, rollback, and lifecycle state.
- Prefer evolving existing skills over endlessly creating new ones.

## Non-Goals

- Global cross-workspace skill evolution in v1.
- Fully autonomous destructive edits to high-impact skills without governance.
- LLM-dependent logic on the synchronous user reply path.
- Replacing the existing `memory` or `skills` subsystems.

## Context and Existing PicoClaw Structure

PicoClaw already has:

- workspace/global/builtin skill loading in `pkg/skills`
- prompt-time skill summary injection in `pkg/agent/context.go`
- lightweight markdown-based memory in `pkg/agent/memory.go`

What is missing is a lifecycle layer between "a useful thing happened" and "a skill should exist or evolve." This design adds that missing middle layer without changing the role of existing memory or skill loading code:

- `memory` remains the place for durable facts and notes.
- `skills` remains the place for formal reusable procedures (`SKILL.md`).
- the new evolution subsystem decides when experience should become a candidate skill and how that skill should live or die over time.

## Design Principles

- Hot path must stay tiny: normal turns only emit small learning evidence.
- Heavy work must be cold path: clustering, proposal generation, comparison, and cleanup happen asynchronously.
- Skills and memory stay separate: procedural learning must not be mixed into generic memory.
- Candidate-first governance: new learned procedures enter as candidates before formal adoption.
- Prefer preserving good skill content: `append` is safer than `replace`; `replace` is safer than `merge`.
- Keep rollback cheap and deterministic.

## Comparison with OpenClaw

This design borrows two proven ideas from OpenClaw while extending them for PicoClaw's lightweight runtime goals.

### What to borrow

- From `memory-core` dreaming:
  - short-term evidence before durable promotion
  - thresholded promotion
  - background maintenance instead of constant prompt inflation
- From `skill-workshop`:
  - proposal-oriented skill evolution
  - `create` / `append` / `replace` style changes
  - quarantine and scanning before application

### What to change

- PicoClaw should avoid running LLM reviewers on the synchronous `agent_end` path in v1.
- PicoClaw needs stronger version backup and rollback than OpenClaw's current proposal application model.
- PicoClaw should add a true lifecycle for formal skills: `active -> cold -> archived -> deleted`.

## User-Facing Mental Model

The system should use intuitive names in docs, logs, and future UI:

- `Learning Note`: one tiny learning card from a finished task
- `Learning Topic`: a grouped pattern that may be worth turning into a skill
- `Skill Draft`: a candidate skill update that has not become official yet
- `Skill Profile`: the lifecycle and version card for an official skill

Only `skills/<name>/SKILL.md` is a real formal skill. The other objects are runtime/internal management structures.

## High-Level Architecture

The evolution subsystem has four conceptual layers:

1. Evidence layer
   - stores `Learning Note`
   - low-cost, append-only, hot-path safe
2. Pattern layer
   - groups notes into `Learning Topic`
   - computes maturity and promotion-worthiness
3. Candidate layer
   - materializes `Skill Draft`
   - performs matching, validation, quarantine, and candidate storage
4. Lifecycle layer
   - manages formal `Skill Profile`
   - handles activation, cooling, archiving, deletion, and rollback metadata

## Hot Path vs Cold Path

### Hot path

Allowed on the normal user turn:

- write a `Learning Note`
- attach lightweight rule-based signals such as:
  - repeated pattern hint
  - user correction observed
  - final winning path observed
  - likely skill gap observed

Not allowed on the normal user turn:

- LLM proposal generation
- full skill similarity comparison
- draft-to-skill application
- lifecycle cleanup
- multi-skill review scans over large corpora

### Cold path

Triggered by heartbeat, cron, maintenance runs, or explicit operator actions:

- group notes into topics
- score topic maturity
- retrieve similar existing skills
- generate `Skill Draft`
- run structural validation and safety scan
- move drafts into candidate/quarantine/accepted states
- update `Skill Profile` usage and lifecycle state
- perform cold/archive/delete cleanup

## LLM Dependency Map

### Steps that do not require an LLM

- writing `Learning Note`
- attaching rule-based learning signals
- topic grouping
- maturity threshold scoring
- non-LLM similar-skill retrieval
- structural validation
- safety scanning
- version backup and rollback
- usage counting and lifecycle transitions

### Steps that do require an LLM

- generating the text of a `Skill Draft`
- optionally rewriting a mature `Learning Topic` into:
  - a new workflow skill
  - an appended section
  - a replacement patch
  - a shortcut-oriented "start here" section

### Performance hotspot to avoid

The main UX risk is doing LLM-backed review or proposal generation at synchronous task completion time. In v1, all LLM-backed evolution must stay out of the user's reply path.

## Core Objects

### Learning Note

Purpose:

- capture a small structured trace of what may be worth learning from a finished task

Suggested fields:

- `id`
- `created_at`
- `workspace_id`
- `session_key`
- `task_hash`
- `task_summary`
- `success`
- `tool_calls_count`
- `tool_kinds`
- `had_user_correction`
- `active_skill_names`
- `signals`
- `artifact_refs`
- `attempt_trail`

Implementation form:

- integrated runtime program data
- stored under state, not in `skills/`
- not injected into normal prompt flow

LLM dependency:

- none

### Learning Topic

Purpose:

- represent a repeatable procedural theme discovered from multiple notes

Suggested fields:

- `id`
- `created_at`
- `updated_at`
- `workspace_id`
- `fingerprint`
- `title_hint`
- `tool_signature`
- `event_ids`
- `event_count`
- `success_rate`
- `correction_rate`
- `diversity_score`
- `recency_score`
- `promotion_score`
- `matched_skill_candidates`
- `winning_path`
- `status`

Implementation form:

- integrated runtime program data
- built by background maintenance logic

LLM dependency:

- none in v1

### Skill Draft

Purpose:

- represent a candidate skill update before it becomes official

Suggested fields:

- `id`
- `created_at`
- `updated_at`
- `workspace_id`
- `source_topic_id`
- `source_note_ids`
- `target_skill_name`
- `draft_type`
- `change_kind`
- `reason`
- `description`
- `body_or_patch`
- `similar_skill_refs`
- `llm_generation_meta`
- `scan_findings`
- `status`

`draft_type`:

- `workflow`
- `shortcut`

`change_kind`:

- `create`
- `append`
- `replace`
- `merge`

Implementation form:

- candidate asset stored in state
- not a formal skill until accepted and applied

LLM dependency:

- required for draft content generation

### Skill Profile

Purpose:

- manage the lifecycle and version metadata of a formal skill

Suggested fields:

- `skill_name`
- `workspace_id`
- `current_version`
- `status`
- `origin`
- `last_used_at`
- `use_count`
- `success_count`
- `shortcut_win_count`
- `superseded_count`
- `specificity_score`
- `retention_score`
- `cooldown_reason`
- `archive_reason`
- `last_matched_topic_id`
- `version_history`

Implementation form:

- sidecar metadata around formal `SKILL.md` files
- internal program data, not itself a skill

LLM dependency:

- none for lifecycle management

## Shortcut Learning from Trial-and-Error

The system must explicitly support learning from a task where the agent tries multiple skills or methods and the final attempt succeeds.

This is not just "the last skill was good." It is a routing lesson:

- earlier paths were wasteful or unstable
- the final path is the preferred entry path for similar tasks

### Required capture

`Learning Note.attempt_trail` should record ordered attempts such as:

- tried skill/method
- outcome class: failed, partial, superseded, success
- why it failed when that can be detected structurally

### Required promotion behavior

Repeated winning-path evidence can create a `shortcut` type `Skill Draft`.

These drafts typically become:

- a `## Start Here` section
- a `## Preferred Path` section
- a `## Avoid Common Dead Ends` section

This allows future similar tasks to start at the successful path instead of replaying old trial-and-error.

## Draft Generation and Matching Flow

When a `Learning Topic` reaches maturity:

1. retrieve the nearest existing official skills and relevant candidate drafts
2. classify whether the change is best represented as:
   - `create`
   - `append`
   - `replace`
   - `merge`
3. invoke the LLM to generate a draft body or patch
4. run structural validation
5. run safety scanning
6. place the result into:
   - `candidate`
   - `quarantined`
   - `accepted` only after promotion conditions are met

## Change Decision Rules

### Create

Use when:

- no sufficiently similar formal skill exists
- the learned process is meaningfully new in this workspace

### Append

Use when:

- the target skill is fundamentally correct
- the new learning only adds a section, an exception, a check, or a shortcut

This is the preferred default when safe.

### Replace

Use when:

- an existing section is stale, wrong, or misleading
- the new content is meant to supersede old guidance rather than supplement it

This requires backup before application.

### Merge

Use when:

- two overlapping skills should become one clearer skill
- or a new draft materially subsumes multiple narrow skills

This is the riskiest operation and should default to candidate review, not silent acceptance.

## Draft State Machine

Suggested states:

- `draft`
- `candidate`
- `quarantined`
- `accepted`
- `rejected`
- `superseded`
- `rolled_back`

Transition outline:

1. topic reaches maturity
2. LLM creates `draft`
3. validation/scan:
   - clean -> `candidate`
   - blocked -> `quarantined`
4. later promotion:
   - accepted automatically for low-risk cases
   - accepted manually for high-impact cases
5. applied update creates or updates a formal skill and `Skill Profile`
6. failed application or invalid resulting skill -> `rolled_back`

## Candidate Promotion Rules

### Low-risk automatic promotion

Allowed only for low-impact changes such as:

- narrow `append`
- shortcut guidance that does not delete or rewrite core workflow
- small additions to an existing workspace skill

Conditions:

- structure validation passes
- safety scan passes
- later similar tasks confirm usefulness
- no high-risk replacement or merge is involved

### Manual or explicit confirmation

Required for:

- `replace`
- `merge`
- large `create`
- drafts that significantly change default behavior

This matches the graded governance requirement:

- low-risk changes may self-progress
- high-impact changes require confirmation

## Validation, Safety, and Rollback

### Validation and safety

Every draft must pass:

- structural validation of the resulting skill layout
- safety scanning for prompt injection, permission bypass, and dangerous shell patterns

If these checks fail, the draft becomes `quarantined` and never touches official skills.

### Backup rules

Before applying any modification to an existing skill, create a backup for:

- `append`
- `replace`
- `merge`

Backup must include:

- original `SKILL.md`
- supported companion files if modified
- proposal id
- timestamp

### Rollback triggers

In v1 rollback is deterministic and structure-driven, not behavior-driven.

Rollback should occur when:

- the new skill fails structural validation after materialization
- required files are missing
- frontmatter is invalid
- patch application did not resolve correctly
- the resulting skill is empty or oversized
- the loader can no longer recognize the resulting skill

### Rollback recording

`Skill Profile.version_history` should record:

- which draft was applied
- which backup was used
- why rollback occurred
- what version became current again

## Formal Skill Lifecycle

Official skills use these states:

- `active`
- `cold`
- `archived`
- `deleted`

### Active

- participates in normal skill use and recommendation

### Cold

- retained but deprioritized
- suitable for skills that were useful but are no longer frequently relevant

### Archived

- preserved for later recovery
- excluded from normal recommendation and prompt-time preference

### Deleted

- formal removal after extended inactivity and low retained value
- leave behind a minimal tombstone record so the system does not immediately relearn a known-low-value skill

## Retention and Anti-Deletion Safeguards

Skill deletion must not rely on age alone. Compute a retention score using:

- recency
- usage frequency
- success reliability
- whether the skill often becomes the final winning path
- specificity to the workspace

### Required safeguards

- manually authored skills should not auto-delete in v1
- low-frequency but high-value skills should not auto-delete
- recently replaced skills must keep rollback-capable history for a protection window
- newly accepted skills should not immediately enter cooling

### Lifecycle transition rules

- `active -> cold`
  - when usage falls and the skill is increasingly superseded
- `cold -> active`
  - when it becomes useful again
- `cold -> archived`
  - after extended inactivity with limited retained value
- `archived -> active`
  - when re-matched successfully
- `archived -> deleted`
  - only after prolonged irrelevance and when safeguards do not block deletion

## Example End-to-End Flow

Example: Chinese-city weather tasks repeatedly end with a final native-name lookup path succeeding after several poorer attempts.

1. Each completed task writes a `Learning Note`.
2. Those notes group into a `Learning Topic` such as `weather-cn-routing`.
3. The topic records a repeated winning path:
   - generic geocode path is wasteful
   - native-name resolution wins
4. A cold-path maintenance run asks the LLM to create a `shortcut` `Skill Draft`.
5. The matching step decides `append` is best for the existing `weather` skill.
6. The generated draft adds:

```md
## Start Here

For Chinese-city weather requests, try native-name resolution first.
Do not start with generic geocoding unless the native query is ambiguous.
```

7. The draft passes structural validation and safety scan and enters `candidate`.
8. After similar tasks confirm the shortcut again, it is accepted and applied.
9. The updated `weather` skill gets a new `Skill Profile` version entry.
10. If the applied change breaks skill structure, PicoClaw restores the previous version immediately.

## Implementation Shape in PicoClaw

This should be implemented as an integrated runtime subsystem, not as a normal workspace skill and not as a user-facing tool in v1.

Recommended rough placement:

- agent/runtime hooks near `pkg/agent` for learning-note emission
- evolution state store under workspace or state-dir owned data
- formal skills still live under `workspace/skills`
- lifecycle metadata and backups stored beside state data or as controlled sidecars

Potential future operator surfaces may exist, but the core mechanism is not itself "a skill." It is integrated program logic that learns when skills should exist.

## Success Criteria

The design is successful if:

- common user turns remain fast and do not incur new LLM learning latency
- learned procedures first appear as candidates rather than silently mutating official skills
- repeated trial-and-error can become reusable shortcut guidance
- high-impact skill changes can be rolled back deterministically
- stale learned skills can cool down and eventually leave the active set without deleting rare but valuable knowledge

## Risks and Tradeoffs

- Candidate governance adds complexity but prevents prompt and skill pollution.
- Non-LLM similarity matching may be imperfect; that is acceptable in v1 if it avoids hot-path cost.
- Merge logic is powerful but risky; keep it conservative in early versions.
- Lifecycle cleanup can accidentally over-prune if retention scoring is too naive; this is why manual, rare, and rollback-relevant skills get stronger protection.

## Recommended V1 Scope

Include:

- `Learning Note`
- `Learning Topic`
- `Skill Draft`
- `Skill Profile`
- cold-path LLM draft generation
- `create` / `append` / `replace`
- `shortcut` drafts
- validation, scan, candidate state, backup, rollback
- `active` / `cold` / `archived` / `deleted`

Defer:

- cross-workspace evolution
- behavior-based rollback
- fully automatic `merge`
- full UI for all lifecycle objects
- LLM reranking on the hot path
