# CLAUDE.md

## Project: picoclaw

Go-based AI agent with multi-channel messaging (Telegram, Discord, Slack, etc.) and a Telegram Mini App UI.

## Build & Test

```bash
go build ./...
go test ./...
go vet ./...
```

Lint: `golangci-lint run`

## Plan Mode

- **Interview tool filtering**: `interviewAllowedTools` in `pkg/agent/loop.go` is the single source of truth for tools available during interview/review phases. Both `filterInterviewTools` (strips definitions before LLM call) and `isToolAllowedDuringInterview` (argument-level gating) reference this map.
- **History clear**: `/plan start clear` wipes session history and summary on transition to executing. The Mini App review UI offers two sliders: standard approve and approve-with-clear.

## Security TODOs

- ~~**Log Fields masking**~~: Done. `SanitizeFields()` in `pkg/logger/logger.go` masks keys matching `token`, `key`, `secret`, `password`, `authorization`, `credential`. Applied in `RecentLogs()` and `wsLogs()` stream.

## Known Gaps

- **Mini App log viewer has no frontend tests**: `renderLogs()` in `pkg/miniapp/static/index.html` is inline vanilla JS with no unit/E2E test coverage. Backend (Go) tests cover `RecentLogs`, `SanitizeFields`, and JSON serialization, but nothing verifies the JS rendering. This allowed the Fields display bug (fields sent but not rendered) to ship undetected.
- **No human intervention for heartbeat worktrees**: Heartbeat sessions create git worktrees (`.picoclaw/worktrees/heartbeat-YYYYMMDD/`) but there is no CLI or Mini App command to list, inspect, or manually dispose them. Need a `/plan worktrees` command (or similar) that shows active worktrees with branch/commit info and allows manual merge/dispose. `PruneOrphaned` on startup only removes directories without auto-committing first, so uncommitted changes in orphaned worktrees are silently lost.
