# Coding Conventions

**Analysis Date:** 2026-04-10

## Naming Patterns

**Packages:**
- Lowercase, single-word names (idiomatic Go). Examples: `logger`, `config`, `session`, `providers`, `seahorse`, `tokenizer`, `memory`, `isolation`, `credential`.
- Located under `pkg/` for shared libraries and `cmd/picoclaw/internal/` for CLI-internal packages.
- Internal packages live in `cmd/picoclaw/internal/<feature>/`, each feature gets its own sub-package: `auth`, `agent`, `gateway`, `cron`, `skills`, `onboard`, `migrate`, `model`, `status`, `version`.

**Functions:**
- Public: `PascalCase` -- `NewPicoclawCommand`, `SetLevelFromString`, `GetOrCreateConversation`, `RegisterLauncherAuthRoutes`.
- Private: `camelCase` -- `newBenchStore`, `logMessage`, `appendFields`, `getCallerSkip`, `openTestDB`.
- Constructor prefix: `New` for public (`NewAntigravityProvider`, `NewSubagentManager`), `new` for private (`newBenchStore`, `newTestCompactionEngineWithStore`).
- Cobra command constructors: `NewXxxCommand()` in `cmd/picoclaw/internal/<pkg>/` (e.g., `NewAuthCommand()`, `NewAgentCommand()`). Tests call `newXxxCommand()` (lowercase) when the constructor is internal.

**Variables:**
- `camelCase` for locals and package-level variables: `currentLevel`, `logFile`, `rrCounter`, `consoleWriter`.
- Constants use `PascalCase` or `UPPER_SNAKE_CASE`: `CurrentVersion`, `Component`, `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`.
- Environment variable constants: `EnvHome`, `EnvConfig` in `pkg/config/`. Actual env var names: `PICOCLAW_HOME`, `PICOCLAW_CONFIG`, `PICOCLAW_LOG_FILE`, `TZ`, `ZONEINFO`.

**Types/Interfaces:**
- `PascalCase` structs: `Config`, `Store`, `SessionManager`, `AntigravityProvider`, `JSONLBackend`.
- Interface names: `LLMProvider`, `SessionStore`, `CompleteFn` (function type alias), `Tool` (implicit).
- Type aliases for external types: `type LogLevel = zerolog.Level` in `pkg/logger/logger.go`.
- Config sub-types use `XxxConfig` suffix: `IsolationConfig`, `AgentsConfig`, `SessionConfig`, `ChannelsConfig`.

## Code Formatting

**Tooling:** `golangci-lint` v2 with formatters configured in `.golangci.yaml`.

**Formatters enabled:**
- `gci` -- import grouping: `standard` -> `default` -> `localmodule` with custom order.
- `gofmt` -- with rewrite rules: `interface{}` -> `any`, `a[b:len(a)]` -> `a[b:]`.
- `gofumpt` -- strict formatting.
- `goimports` -- auto import management.
- `golines` -- max line length 120.

**Commands:**
```bash
make fmt    # runs golangci-lint fmt
make lint   # runs golangci-lint run --build-tags=goolm,stdjson
make fix    # runs golangci-lint run --fix --build-tags=goolm,stdjson
```

**Lint settings:**
- `default: all` with 35+ linters disabled (see `.golangci.yaml` lines 5-63).
- Line length: 120 chars (`.golangci.yaml` line 85).
- `funlen`: 120 lines / 40 statements.
- `gocognit`: min-complexity 25.
- `gocyclo`: min-complexity 20.
- `lll` exclusions for `//go:generate` lines.
- Test files excluded from `funlen`, `maintidx`, `gocognit`, `gocyclo` (`.golangci.yaml` lines 147-148).
- `testpackage` is disabled -- tests can use the same package name (not `_test`).

## Import Organization

**Order (gci config):**
1. Standard library
2. Third-party packages
3. Local module (`github.com/sipeed/picoclaw/...`)

**Example from `cmd/picoclaw/main.go`:**
```go
import (
    "fmt"
    "os"
    "time"

    "github.com/spf13/cobra"

    "github.com/sipeed/picoclaw/cmd/picoclaw/internal"
    "github.com/sipeed/picoclaw/cmd/picoclaw/internal/agent"
    // ... more internal imports
    "github.com/sipeed/picoclaw/pkg/config"
    "github.com/sipeed/picoclaw/pkg/updater"
)
```

**Path aliases:** None used. Full import paths everywhere: `github.com/sipeed/picoclaw/pkg/...`.

## Error Handling

**Patterns:**
- `fmt.Errorf("context: %w", err)` for error wrapping with `fmt` and `%w`. Example from `pkg/seahorse/schema.go`:
  ```go
  return fmt.Errorf("FTS5 check: %w", err)
  ```
- Direct `return nil, fmt.Errorf("antigravity auth: %w", err)` in provider code (`pkg/providers/antigravity_provider.go`).
- Config errors use descriptive messages with `fmt.Errorf("failed to create log directory: %w", err)` (`pkg/logger/logger.go`).
- Tests use `t.Fatalf("operation: %v", err)` for fatal errors and `t.Errorf("description: %v", err)` for non-fatal.
- Some code logs errors rather than returning them (fire-and-forget pattern in `pkg/session/jsonl_backend.go`):
  ```go
  if err := b.store.AddMessage(...); err != nil {
      log.Printf("session: add message: %v", err)
  }
  ```
- Sentinel errors not widely used. Most errors are constructed inline with `fmt.Errorf`.

## Logging

**Framework:** `github.com/rs/zerolog` wrapped by a custom logger at `pkg/logger/logger.go`.

**API pattern:** Suffix convention for logging variants:
- `Debug(message)` -- plain message, auto-detected component
- `DebugC(component, message)` -- explicit component
- `Debugf(message, args...)` -- sprintf-style
- `DebugF(message, fields)` -- structured fields as `map[string]any`
- `DebugCF(component, message, fields)` -- component + fields

**Same pattern for all levels:** `Info/InfoC/Infof/InfoF/InfoCF`, `Warn/WarnC/Warnf/WarnF/WarnCF`, `Error/ErrorC/Errorf/ErrorF/ErrorCF`, `Fatal/FatalC/Fatalf/FatalF/FatalCF`.

**Log format (TTY):**
```
15:04:05 WARN  component  caller  message
```
Component shown in yellow (`\x1b[33m`). Time-only timestamps.

**Log format (non-TTY):** JSON (zerolog default).

**Configuration from env:** `PICOCLAW_LOG_FILE` -- if set, enables file logging and disables console. Supports `~/` expansion.

**Global logger:** Package-level singleton with `sync.RWMutex` for thread safety. Not passed as dependency -- imported directly.

## Configuration Patterns

**Main config:** `pkg/config/config.go` -- JSON-based with `json` struct tags.

**Loading:**
- Config file path via `PICOCLAW_CONFIG` env var or `PICOCLAW_HOME` env var (defaults to `~/.picoclaw`).
- Environment variable overrides via `github.com/caarlos0/env/v11`.
- Schema versioning (`CurrentVersion = 2`) with migration support.
- Config struct uses `json:"-"` on most fields (only `channels`, `model_list`, and build info serialize).

**Version injection via ldflags:**
```
-X github.com/sipeed/picoclaw/pkg/config.Version=...
-X github.com/sipeed/picoclaw/pkg/config.GitCommit=...
-X github.com/sipeed/picoclaw/pkg/config.BuildTime=...
-X github.com/sipeed/picoclaw/pkg/config.GoVersion=...
```
Accessed via `config.GetVersion()`, `config.GetGitCommit()`, etc.

## Interface Design

**Style:** Small, focused interfaces. Examples:
- `LLMProvider` -- `Chat(ctx, messages, tools, model, options) (*LLMResponse, error)` + `GetDefaultModel()`, `SupportsTools()`, `GetContextWindow()`.
- `SessionStore` -- `AddMessage`, `AddFullMessage`, `GetHistory`, `GetSummary`, `SetSummary`, `SetHistory`, `TruncateHistory`, `Save`, `Close`.
- `memory.Store` -- file-based session persistence.

**Function type aliases** used for callbacks: `type CompleteFn func(ctx, prompt, opts) (string, error)` in seahorse compaction.

**Interface implementations** are created via constructor functions: `NewAntigravityProvider()`, `NewJSONLBackend(store)`, `NewSubagentManager(provider, model, workspace)`.

## Struct Organization

**Pattern:**
- Exported structs with public fields for configuration (JSON tags).
- Unexported fields for internal state: `sensitiveCache *SensitiveDataCache`, `store memory.Store`.
- Methods on pointer receivers: `func (c *Config) FilterSensitiveData(...)`.
- Small config sub-structs composed into main `Config`.

**Example from `pkg/config/config.go`:**
```go
type Config struct {
    Version   int             `json:"version"`
    Isolation IsolationConfig `json:"isolation,omitempty"`
    Agents    AgentsConfig    `json:"agents"`
    // ... many more sub-configs
    sensitiveCache *SensitiveDataCache  // unexported, computed cache
}
```

## Git Commit Message Style

**Format:** Conventional Commits. Examples from recent commits:
```
fix(chat): keep tool-call summary and assistant output in sync (#2449)
fix(seahorse): sanitize user input for FTS5 MATCH queries (#2436)
fix(launcher): align react and react-dom versions (#2467)
build(deps): bump github.com/modelcontextprotocol/go-sdk (#2455)
feat(launcher): standard HTTP login/setup/logout flow for dashboard...
style(lint): satisfy gci and golines for review fixes
fix(agent): gate pico interim publish for internal turns
```

**Rules (from CONTRIBUTING.md):**
- English language, imperative mood: "Add retry logic" not "Added retry logic".
- Reference issues: `Fix session leak (#123)`.
- One logical change per commit.
- Squash minor cleanups/typos into a single commit.
- Follow https://www.conventionalcommits.org/zh-hans/v1.0.0/
- Squash merge is the default strategy.

## Branch Naming

**Pattern:** `type/description` -- examples from CONTRIBUTING.md:
- `fix/telegram-timeout`
- `feat/ollama-provider`
- `docs/contributing-guide`

**Long-lived branches:** `main` (active development), `release/x.y` (stable releases).

## Documentation Style

**Package comments:** Minimal. Only the main entry point (`cmd/picoclaw/main.go`) has a header comment block with project description and license.

**Function comments:** Godoc style when present, mostly on public API:
```go
// ParseLevel converts a case-insensitive level name to a LogLevel.
// Returns the level and true if valid, or (INFO, false) if unrecognized.
func ParseLevel(s string) (LogLevel, bool) { ... }

// NewAntigravityProvider creates a new Antigravity provider using stored auth credentials.
func NewAntigravityProvider() *AntigravityProvider { ... }
```

**Internal code:** Comments explain non-obvious logic, especially in `pkg/seahorse/` (schema, compaction) and complex SQL. Bug-fix tests include detailed BUG comments explaining the issue (see `pkg/seahorse/store_test.go` lines 497-619).

**nolint directives:** Used sparingly with justification:
```go
//nolint:zerologlint
func getEvent(logger zerolog.Logger, level LogLevel) *zerolog.Event { ... }
```

---

*Convention analysis: 2026-04-10*
