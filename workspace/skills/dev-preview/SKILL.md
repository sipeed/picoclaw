---
name: dev-preview
description: Start a dev server in the background and preview it through the Mini App reverse proxy.
metadata: {"nanobot":{"emoji":"🌐"}}
---

# dev-preview Skill

Launch a local dev server as a background process, wait for it to become ready, and connect it to the Mini App dev preview proxy.

## Quickstart

```
1. exec(command="npm run dev", background=true)
   → bg-1 started

2. bg_monitor(action="watch", bg_id="bg-1", pattern="ready|listening|localhost")
   → Match: "Server ready on http://localhost:3000"

3. dev_preview(action="start", target="http://localhost:3000", name="frontend")
   → Dev preview started
```

## Tools Overview

### exec (background mode)

Start a long-running process without blocking.

| Call | Purpose |
|------|---------|
| `exec(command="npm run dev", background=true)` | Start dev server |
| `exec(bg_action="output", bg_id="bg-1")` | Get latest output |
| `exec(bg_action="kill", bg_id="bg-1")` | Stop process |

- Background processes auto-terminate after **45 minutes**.
- Initial output (first 3 seconds) is included in the start response.
- Output is kept in a **32 KB ring buffer** (most recent bytes).
- Maximum **10** concurrent background processes.

### bg_monitor

Inspect and wait on background processes.

| Call | Purpose |
|------|---------|
| `bg_monitor(action="list")` | List all bg processes |
| `bg_monitor(action="watch", bg_id="bg-1", pattern="ready")` | Wait for pattern (default 30s timeout) |
| `bg_monitor(action="tail", bg_id="bg-1", lines=30)` | Get last N lines |

- `watch` polls every 100ms and returns the matching line.
- Set `watch_timeout` (seconds) to override the default 30s.
- If the process exits before a match, returns an error with the final output.

### dev_preview

Control the Mini App dev reverse proxy.

| Call | Purpose |
|------|---------|
| `dev_preview(action="start", target="http://localhost:3000")` | Register + activate |
| `dev_preview(action="stop")` | Deactivate proxy |
| `dev_preview(action="status")` | Show all targets |
| `dev_preview(action="unregister", id="...")` | Remove a target |

- Only **localhost** targets are allowed (localhost, 127.0.0.1, ::1).
- `name` is optional; auto-generated from host:port if omitted.

## System Prompt Integration

Active background processes are automatically injected into the system prompt:

```
## Background Processes

  [bg-1] pid=1234 running  (uptime: 5m, max: 45m) npm run dev
  [bg-2] pid=5678 exited=0 (ran: 2m)               go build .
```

This means the agent always knows which processes are running, even across conversation turns and heartbeats.

## Common Patterns

### Python HTTP server

```
exec(command="python -m http.server 8080", background=true)
bg_monitor(action="watch", bg_id="bg-1", pattern="Serving")
dev_preview(action="start", target="http://localhost:8080")
```

### Vite / Next.js

```
exec(command="npm run dev", background=true)
bg_monitor(action="watch", bg_id="bg-1", pattern="ready|localhost|Local:")
dev_preview(action="start", target="http://localhost:5173", name="vite-app")
```

### Debugging

```
bg_monitor(action="tail", bg_id="bg-1", lines=50)
exec(bg_action="output", bg_id="bg-1")
```

### Cleanup

```
exec(bg_action="kill", bg_id="bg-1")
dev_preview(action="stop")
```

## Important Notes

- Always use `bg_monitor(action="watch")` between starting a server and calling `dev_preview(action="start")`. Without it, the server may not be ready yet.
- If `watch` times out, check the output with `bg_monitor(action="tail")` to diagnose startup errors.
- Background processes persist across tool calls but are cleaned up on app shutdown.
- Exited processes remain visible (for output/exit code inspection) until explicitly killed.
