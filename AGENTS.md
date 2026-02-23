# AGENTS.md (Project-Level)

## Scope
This file applies to the `picoclaw` project root.

## Git Upstream and Periodic Sync

Current remote setup:
- `origin`: `https://github.com/xwysyy/picoclaw.git` (your fork)
- `upstream`: `https://github.com/sipeed/picoclaw.git` (original project)

If `upstream` is missing, add it:
- `git remote add upstream https://github.com/sipeed/picoclaw.git`

### Periodic sync routine (recommended)
Run these commands regularly to sync latest upstream changes into your fork branch:
1. `source ~/.zshrc && proxy_on && git fetch upstream`
2. `git checkout main`
3. `git rebase upstream/main`
4. `source ~/.zshrc && proxy_on && git push origin main`

If your workflow prefers merge over rebase, use:
- `git merge upstream/main`

## Known API Error: `No tool output found for function call`

### Symptom
OpenAI Responses API returns HTTP 400 with message:
`No tool output found for function call <call_id>`.

### Meaning
The request includes a `function_call` item, but the next round input is missing the matching
`function_call_output` for the same `call_id`.

### Project-specific root cause we fixed
In this repo, the failure was caused by history sanitization dropping valid tool outputs when
one assistant turn emitted multiple tool calls.

- Fixed logic: `pkg/agent/context.go` (`sanitizeHistoryForProvider`)
- Behavior now:
  - Tracks pending tool call IDs for an assistant tool-call turn.
  - Preserves multiple consecutive tool outputs from the same turn.
  - Drops truly orphaned tool outputs (unknown or empty `tool_call_id` when IDs are required).

### Regression tests
- `pkg/agent/context_test.go`
- Run:
  - `go test ./pkg/agent -run TestSanitizeHistoryForProvider -count=1`

If toolchain/dependency download needs proxy in this environment, run:
- `source ~/.zshrc && proxy_on && go test ./pkg/agent -run TestSanitizeHistoryForProvider -count=1`

## Docker Redeploy (Important: profiles are required)

`docker-compose.yml` defines services under profiles (`agent`, `gateway`).
Running `docker compose up -d --build` without profile may fail with:
`no service selected`.

### Gateway redeploy
1. `docker compose down`
2. `docker compose --profile gateway up -d --build`
3. `docker compose ps`
4. `curl -sS http://127.0.0.1:18790/health`

Expected healthy state:
- container: `picoclaw-gateway`
- compose status: `Up ... (healthy)`
- health response contains `"status":"ok"`

### Useful checks
- `docker logs --tail 100 picoclaw-gateway`
- `docker ps --filter name=picoclaw-gateway`

## Proxy note for non-interactive shells

`proxy_on` is a shell function defined in `~/.zshrc` (not a standalone binary).
In non-interactive command contexts, use:

- `source ~/.zshrc && proxy_on && <your-command>`
