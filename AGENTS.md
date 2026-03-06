# AGENTS.md

This guide is for AI agents and contributors working in this repository.
Its goal is to make code understanding and safe development faster.

Style note: this is a principle-first guide, not a strict checklist. Prefer good engineering judgment over mechanical rule-following.

## 1) Project Snapshot

- Project: `PicoClaw` (`github.com/sipeed/picoclaw`)
- Language: Go (use `go.mod` as source of truth; currently `go 1.25.7`)
- Main binary: `picoclaw`
- Core traits: lightweight AI assistant, multi-channel gateway, tool-enabled agent loop

## 2) Where To Start Reading

Use this order when learning the codebase:

1. CLI entrypoint: `cmd/picoclaw/main.go`
2. CLI command wiring: `cmd/picoclaw/internal/*/command.go`
3. Agent runtime: `pkg/agent/`
4. Tool system and execution safety: `pkg/tools/`
5. Channel orchestration: `pkg/channels/manager.go` and `pkg/channels/*`
6. Message bus contracts: `pkg/bus/`
7. Config model and defaults: `pkg/config/`
8. Provider abstraction and implementations: `pkg/providers/`

## 3) Architecture Mental Model

- `cmd/picoclaw`: Cobra CLI entry, exposes subcommands like `agent`, `gateway`, `onboard`, `skills`, `cron`.
- `pkg/agent`: main loop, tool registration, fallback provider chain, message processing.
- `pkg/channels`: channel manager + platform adapters (Telegram, Discord, Slack, LINE, QQ, WeCom, etc.).
- `pkg/bus`: inbound/outbound message transport between channels and agent.
- `pkg/providers`: LLM provider adapters and fallback logic.
- `pkg/tools`: tool registry and tool implementations (`exec`, web search/fetch, filesystem, mcp, skills, hardware, etc.).
- `pkg/config`: config schema, defaults, migration logic.

Think in data flow:
`Channel -> bus.Inbound -> AgentLoop (+Tools/Provider) -> bus.Outbound -> ChannelManager -> Channel`.

## 4) Common Dev Commands

Prefer `make` targets (they encode project conventions):

```bash
make deps       # download + verify Go modules
make generate   # run go generate
make build      # generate + build current platform binary
make test       # generate + go test ./...
make vet        # generate + go vet ./...
make fmt        # golangci-lint fmt
make lint       # golangci-lint run
make check      # deps + fmt + vet + test
```

Useful focused runs:

```bash
go test -run TestName -v ./pkg/session/
make run ARGS='agent -m "hello"'
```

## 5) Development Rules For Agents

- Keep changes small and scoped to one concern.
- Preserve existing public behavior unless explicitly changing it.
- Update/add tests for behavior changes.
- Prefer existing abstractions over adding new layers.
- Follow current package boundaries; do not couple channel implementations directly to unrelated modules.
- Reuse `Makefile` workflow; run at least relevant tests before finishing.

When trade-offs are unclear, prioritize: correctness > security > minimal diff > speed.

If multiple instruction files exist (for example root `AGENTS.md` and `workspace/AGENTS.md`), prefer keeping them aligned to avoid behavior drift across execution contexts.

## 6) Safety-Critical Areas

Apply extra care in these paths:

- `pkg/tools/shell.go` and command execution paths (deny/allow rules, workspace restriction).
- Channel input handling in `pkg/channels/*` (untrusted external events).
- Config and secrets handling in `pkg/config`.
- Provider fallback/retry paths in `pkg/providers`.

For these areas:

- Validate inputs and path handling.
- Avoid expanding command execution capability without explicit checks.
- Keep default behavior secure.
- If unexpected workspace changes are discovered during a task, pause and confirm with the user before proceeding.

## 7) Testing Expectations

Before proposing a final change:

1. Run targeted package tests for touched code.
2. Run broader checks when feasible (`make test`, ideally `make check`).
3. If full checks are skipped (time/env limits), clearly state what was not run.

Lightweight test mapping (guidance, not mandatory):

- `pkg/agent/*` changes: run `go test ./pkg/agent/...`
- `pkg/tools/*` changes: run `go test ./pkg/tools/...`
- `pkg/channels/*` changes: run at least the touched channel package and `go test ./pkg/channels/...` when feasible

## 8) Docs To Keep In Sync

When behavior changes, check whether these should be updated:

- `README.md` (user-facing usage)
- `README.zh.md` and other localized READMEs when user-visible behavior/text changes
- `CONTRIBUTING.md` (developer workflow)
- `pkg/channels/README.md` (channel architecture specifics)
- `docs/tools_configuration.md` (tooling/config behavior)
- `docs/design/*` (design decisions/refactors)

## 9) Practical First Steps For Any New Task

1. Locate entrypoint and call chain with `rg` and targeted file reads.
2. Confirm current behavior with tests or command output.
3. Implement minimal fix/feature.
4. Validate with focused tests.
5. Summarize changed files, behavior impact, and test evidence.
