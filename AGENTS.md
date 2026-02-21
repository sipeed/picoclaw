# AGENTS.md — Picoclaw Coding Agent Guide

## Build & Test Commands

```bash
# Build (runs go generate first, requires -tags stdjson)
make build                              # builds to build/picoclaw-<os>-<arch>
go build -tags stdjson ./cmd/picoclaw/  # quick build without ldflags

# Test
go test ./...                           # all tests
go test ./pkg/doctor/                   # single package
go test ./pkg/doctor/ -run TestCheckSessionMessages_OrphanToolCall  # single test
go test ./pkg/doctor/ -v                # verbose

# Lint & format
go fmt ./...
go vet ./...
# CI uses golangci-lint v2.10.1 — config in .golangci.yaml

# Full check (deps + fmt + vet + test)
make check
```

**Build note**: `cmd/picoclaw/cmd_onboard.go` uses `//go:embed workspace` which requires
a `workspace/` directory inside `cmd/picoclaw/`. The Makefile's `generate` target handles
this. For quick manual builds: `cp -r workspace cmd/picoclaw/workspace && go build -tags stdjson -o /tmp/picoclaw ./cmd/picoclaw/ && rm -rf cmd/picoclaw/workspace`

## Project Structure

```
cmd/picoclaw/       CLI entrypoint — one cmd_*.go file per subcommand, dispatch via switch in main.go
pkg/
  agent/            Core agent loop, context builder, memory, registry
  auth/             Credential store (~/.picoclaw/auth.json), OAuth flows, PKCE
  bus/              In-process message bus (buffered channels)
  channels/         Chat platform adapters (telegram, discord, slack, etc.)
  config/           JSON config + env var overlay (~/.picoclaw/config.json)
  doctor/           Diagnostic checks (picoclaw doctor)
  logger/           Custom structured logger (no third-party lib)
  providers/        LLM provider abstraction
    anthropic/      Anthropic SDK wrapper
    openai_compat/  OpenAI-compatible HTTP provider
  session/          Session persistence (~/.picoclaw/workspace/sessions/*.json)
  tools/            Tool system — one file per tool (exec, web, edit, filesystem, etc.)
  tui/              Bubbletea terminal UI (chat viewport, markdown, tool indicators)
```

## Code Style

### Imports
Two groups separated by a blank line: (1) stdlib, (2) everything else (third-party and internal mixed).
Both groups sorted alphabetically. Use aliases only when needed for disambiguation.

```go
import (
    "context"
    "fmt"
    "strings"

    "github.com/sipeed/picoclaw/pkg/config"
    "github.com/sipeed/picoclaw/pkg/providers"
)
```

### Naming
- **Exported types**: `PascalCase` — `AgentLoop`, `SessionManager`, `AuthCredential`
- **Unexported types**: `camelCase` — `processOptions`, `providerSelection`
- **Constructors**: `NewXxx()` or `NewXxxWithYyy()` for variants
- **Receivers**: 1–2 letter abbreviation of type — `(al *AgentLoop)`, `(sm *SessionManager)`, `(p *HTTPProvider)`
- **Files**: `snake_case.go`, tests in same package (not `_test` external package)
- **Constants**: exported `PascalCase`, unexported `camelCase` prefixed with context (`defaultAnthropicAPIBase`)
- **Errors**: lowercase messages without trailing punctuation — `"loading auth credentials: %w"`

### Error Handling
- Wrap at module boundaries with `fmt.Errorf("context: %w", err)`
- Pass through raw `return nil, err` inside a function when context is obvious
- Return `nil, nil` for "not found" (no sentinel errors)
- Log-then-return at important boundaries: log with `logger.ErrorCF`, return wrapped error
- No custom error types except `FallbackExhaustedError`, `FailoverError` in providers

```go
store, err := auth.LoadStore()
if err != nil {
    return fmt.Errorf("loading auth store: %w", err)
}
```

### Types & Interfaces
- Interfaces are small (1–5 methods), defined where consumed
- Optional interfaces via type assertion: `if ct, ok := tool.(ContextualTool); ok { ... }`
- All config/data structs use `json:"snake_case"` tags with `omitempty` where appropriate
- Thread-safe structs embed `sync.RWMutex` as `mu` field
- Delegate pattern for provider wrappers: `ClaudeProvider.delegate *anthropicprovider.Provider`

### Logging
Custom logger in `pkg/logger/` with levels `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`.
Primary API: `logger.InfoCF(component, message, fields)` where fields is `map[string]interface{}`.

```go
logger.ErrorCF("agent", "LLM call failed", map[string]interface{}{
    "agent_id":  agent.ID,
    "iteration": iteration,
    "error":     err.Error(),
})
```

Components are short strings: `"agent"`, `"tool"`, `"telegram"`, `"discord"`.

### Testing
- Standard `testing` package — no testify assertions
- Table-driven tests with `t.Run()` are the dominant pattern
- Test variable: `tt` for table entries, `tests` for the slice
- Mocks: in-file structs in `_test.go` or dedicated `mock_*_test.go` files
- Testable package-level vars with `t.Cleanup()` restore: `originalFn := someFn; t.Cleanup(func() { someFn = originalFn })`
- Temp dirs: `os.MkdirTemp("", "prefix-*")` with `defer os.RemoveAll(tmpDir)`
- Assertions: direct `if` checks with `t.Fatalf` (test-stopping) or `t.Errorf` (continue)

```go
tests := []struct {
    name string
    input string
    want  int
}{
    {name: "empty", input: "", want: 0},
    {name: "single", input: "a", want: 1},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := Len(tt.input)
        if got != tt.want {
            t.Errorf("Len(%q) = %d, want %d", tt.input, got, tt.want)
        }
    })
}
```

### CLI Commands
Commands are registered in `cmd/picoclaw/main.go` via a `switch` on `os.Args[1]`.
Each command lives in its own `cmd_<name>.go` file with a `<name>Cmd()` function.
Flag parsing is manual (`for i := 0; i < len(args); i++ { switch args[i] { ... } }`).

### File I/O
Atomic writes use temp-file + rename pattern (see `session/manager.go`, `state/state.go`).
Auth files should be `0600`. Config files should be `0644`.

### Concurrency
- `context.Context` is always the first parameter in I/O methods
- Message passing via buffered channels (`pkg/bus/`)
- Mutex naming: `mu sync.RWMutex` field, lock/unlock at method start

### Comments
- Go doc comments on exported symbols (`// TypeName does X`)
- No requirement for trailing periods (godot linter is disabled)
- Numbered inline steps for complex functions: `// 1. Build messages`, `// 2. Call LLM`
- File headers (optional): `// PicoClaw - Ultra-lightweight personal AI agent` + MIT license

### JSON & Config
- `json:"snake_case"` struct tags, `omitempty` for optional fields
- Environment overlay via `env:"PICOCLAW_UPPER_SNAKE"` tags
- Custom JSON marshal/unmarshal for flexible deserialization (see `FlexibleStringSlice`)
- Config path: `~/.picoclaw/config.json`, auth: `~/.picoclaw/auth.json`

## Git Workflow

### Remotes
- `origin` — `https://github.com/rahul-hudle/picoclaw` (our fork, push here)
- `upstream` — `https://github.com/sipeed/picoclaw.git` (original repo, fetch only)

### Atomic Commits
Make small, focused commits — one logical change per commit. Do NOT bundle unrelated
changes. Commit message format: `type: concise description` where type is one of
`feat`, `fix`, `refactor`, `test`, `docs`, `chore`.

```bash
# Good — each concern is a separate commit
git add pkg/doctor/doctor.go pkg/doctor/doctor_test.go
git commit -m "feat: add picoclaw doctor command with session integrity checks"

git add cmd/picoclaw/cmd_doctor.go cmd/picoclaw/main.go
git commit -m "feat: wire doctor command into CLI"

# Bad — mixing unrelated changes
git add -A && git commit -m "various changes"
```

After committing, push to origin:
```bash
git push origin main
```

### Syncing with Upstream
Periodically fetch and merge from the upstream repo to stay current:

```bash
git fetch upstream
git merge upstream/main          # or: git rebase upstream/main
# Resolve any conflicts, then:
git push origin main
```

Do this before starting new work and whenever upstream has significant changes.

### Updating the CLI
When adding or modifying commands, always update **all three** of these:
1. **`cmd/picoclaw/cmd_<name>.go`** — the command implementation
2. **`cmd/picoclaw/main.go`** — add the `case` in the `switch` block AND add to `printHelp()`
3. **`AGENTS.md`** — update the Project Structure section if a new package was added

## CI Pipeline (`.github/workflows/pr.yml`)

PRs run in parallel: `gofmt -l` diff check, `go vet ./...`, `golangci-lint run`, `go test ./...`.
All must pass. The lint config (`.golangci.yaml`) starts from `default: all` and disables
specific linters — check the file before adding new code patterns.
