# PicoClaw Trace Viewer

A debug UI that shows real-time LLM traces from the running PicoClaw gateway — per-turn LLM calls, system prompts, messages, available tools, and tool executions.

## How it works

The gateway writes structured logs to `~/.picoclaw/logs/gateway.log`. The tracer reads that file and serves a web UI showing every LLM call made during each conversation turn.

## Prerequisites

- Go 1.21+
- Node.js 18+ and npm

## Running

**Step 1 — Start the gateway with debug logging:**
```bash
picoclaw gateway --debug --no-truncate
```

**Step 2 — Build the frontend (first time only):**
```bash
cd tracer/frontend
npm install && npm run build
```

**Step 3 — Start the tracer:**
```bash
go run ./cmd/tracer
```

Open `http://localhost:7331`.

## Options

```
--port          Port to listen on (default: 7331)
--log           Path to gateway.log (default: ~/.picoclaw/logs/gateway.log)
--frontend-dir  Path to frontend/dist (dev only)
```

## Building a single binary

```bash
cd tracer/frontend && npm install && npm run build && cd ../..
go build -tags embed -o tracer-bin ./cmd/tracer
./tracer-bin
```
