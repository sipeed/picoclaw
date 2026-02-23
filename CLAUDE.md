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

## Security TODOs

- **Log Fields masking**: `LogEntry.Fields` (`map[string]any`) is exposed via WebSocket (`/miniapp/api/logs/ws`) and snapshots (`/miniapp/api/logs/snapshot`). If any code logs sensitive values (tokens, API keys, passwords) in Fields, they will be visible to Mini App users. Add a sanitizer in `RecentLogs()` and `wsLogs()` that masks values for keys matching patterns like `token`, `key`, `secret`, `password`, `authorization`. Track in: `pkg/logger/logger.go` (RecentLogs), `pkg/miniapp/miniapp.go` (wsLogs stream).
