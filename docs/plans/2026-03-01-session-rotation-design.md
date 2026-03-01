# Session Rotation and Scope-Aware Session Control Design

## Background

PicoClaw currently derives a single `SessionKey` from routing and uses it directly for message history and summary. There is no first-class session rotation state, and no user commands for listing/resuming historical sessions in a routing scope.

Current limitations:

- No `/new` (or `/reset`) command to start a fresh conversation context.
- No `/session list` or `/session resume <n>` command.
- No backlog cap/cleanup strategy for old sessions.
- `SessionManager` stores session content only; it lacks active-session index semantics.

## Goals (Phase 1)

- Add `/new` command (alias `/reset`) to start a new session in current routing scope.
- Add `/session list` and `/session resume <n>` for scope-local session navigation.
- Enforce configurable backlog limit per scope and remove oldest sessions when exceeded.
- Keep behavior aligned with existing routing/session boundaries (`dm_scope` etc.), with no override that breaks routing semantics.

## Non-Goals (Phase 1)

- No automatic extraction of archived sessions into `memory/MEMORY.md`.
- No cross-scope or global session browsing/resume.
- No changes to routing rules or `dm_scope` semantics.

## Domain Boundaries

### Existing boundaries (preserve)

- `routing` computes canonical session boundary from channel/account/peer + `dm_scope`.
- `session` stores conversation history and summary.
- `state` stores runtime notification state (`last_channel`, `last_chat_id`) for heartbeat/devices.

### Design decision

Use **SessionManager-owned session index semantics** (B+ approach), not a new `state`-domain session index.

Reasoning:

- Session lifecycle belongs to the session domain, not notification state.
- Avoid cross-domain write consistency problems (`state` index + `sessions` content).
- Better long-term extensibility for phase-2 archival queue integration.

## Terminology

- `scopeKey`: routing-produced key representing session scope anchor (currently `route.SessionKey`).
- `sessionKey`: concrete key used to read/write messages in current turn.

Relation:

- `scopeKey` is not a new entity type; it is naming for `route.SessionKey` usage in rotation logic.
- `sessionKey = ResolveActive(scopeKey)` where active may be `scopeKey` itself or rotated variants.

## Data Model

### Session index storage

Persist under:

- `<workspace>/sessions/index.json`

Model sketch:

```json
{
  "version": 1,
  "scopes": {
    "agent:main:telegram:direct:user123": {
      "active_session_key": "agent:main:telegram:direct:user123#3",
      "ordered_sessions": [
        "agent:main:telegram:direct:user123#3",
        "agent:main:telegram:direct:user123#2",
        "agent:main:telegram:direct:user123"
      ],
      "updated_at": "2026-03-01T11:00:00+08:00"
    }
  }
}
```

### Key generation

- First session for a scope remains exactly `scopeKey`.
- Rotated sessions use monotonic suffix `#<n>` under the same scope (e.g. `scopeKey#2`, `scopeKey#3`).

## API Design (SessionManager Extension)

- `ResolveActive(scopeKey string) (sessionKey string, err error)`
- `StartNew(scopeKey string) (newSessionKey string, err error)`
- `List(scopeKey string) ([]SessionMeta, err error)`
- `Resume(scopeKey string, index int) (sessionKey string, err error)`
- `Prune(scopeKey string, limit int) (pruned []string, err error)`
- `DeleteSession(sessionKey string) error` (content + file + index adjustment)

`SessionMeta` includes:

- ordinal index (for user-facing stable numbering)
- session key
- updated timestamp
- optional message count
- active flag

## Runtime Flow

1. `AgentLoop.processMessage` routes inbound message and gets `scopeKey` (`route.SessionKey`).
2. Before `runAgentLoop`, call `sessionKey := Sessions.ResolveActive(scopeKey)`.
3. `runAgentLoop` uses resolved `sessionKey` for history/summary read/write.
4. Commands:
   - `/new` / `/reset`: `StartNew(scopeKey)` + `Prune(scopeKey, backlogLimit)`
   - `/session list`: `List(scopeKey)`
   - `/session resume <n>`: `Resume(scopeKey, n)`

## Command Scope Semantics

- Commands operate within current routing scope only.
- This naturally follows `dm_scope` behavior:
  - `main`: one shared scope
  - `per-peer` / `per-channel-peer` / `per-account-channel-peer`: corresponding isolated scopes

## Consistency & Recovery

### Concurrency

- SessionManager guards session map and index model with one lock domain.
- No external package writes index structures directly.

### Persistence order

- Write/ensure session content, then atomically write `index.json`.
- On startup:
  - Remove index entries referencing missing session files.
  - Optionally recover orphan session files into scope lists (recommended).

### Failure behavior

- User command failures are non-fatal and user-readable.
- File-delete failure during prune logs warning and retries/repair on next startup.

## Configuration

Add `session.backlog_limit`:

- default: `20`
- min: `1`
- apply per `scopeKey`

Invalid values fallback to default with warning log.

## Testing Strategy

### Unit

- ResolveActive bootstrap behavior.
- StartNew monotonic key generation and active pointer update.
- List stable ordering/index mapping.
- Resume bounds checking and activation.
- Prune oldest deletion + active safety.

### Integration

- `/new` creates isolated subsequent conversation context.
- `/session list` and `/session resume` affect only current scope.
- Scope semantics respect `dm_scope`.

### Recovery

- Corrupted or stale index references are self-healed.
- Backlog cleanup removes both in-memory and on-disk artifacts.

## Phase 2 Extension Hook (Planned)

When phase 2 starts, add archival queue semantics in session domain:

- Enqueue old sessions on rotate/prune.
- Async worker summarizes/extracts value into memory files.
- Keep this decoupled from phase 1 command behavior.

