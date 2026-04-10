# Codebase Concerns

**Analysis Date:** 2026-04-10

---

## Security

### 1. Hardcoded OAuth Client Credentials (Obfuscated but Reversible)

**Files:** `pkg/auth/oauth.go:47-50`

The Google Antigravity OAuth client ID and secret are base64-encoded inline strings, not environment variables. These are public OAuth client credentials (used by OpenCode/pi-ai) rather than secrets, so the risk is low. However, the pattern of embedding credentials in source code is a concern if extended to actual secrets.

### 2. Self-Update Endpoint with No Binary Signature Verification

**Files:** `pkg/updater/updater.go:33-80`, `pkg/updater/updater.go:611-646`

The self-update mechanism downloads release archives from GitHub and extracts them. SHA256 checksum verification exists for release downloads, but there is no code signing verification of the binary itself. The `minio/selfupdate` library handles binary replacement.

- **Mitigation present:** SHA256 checksum verification on release downloads
- **Remaining risk:** No cryptographic signature verification before applying the update
- **Fix approach:** Add signature verification (minisign is already an indirect dependency)

### 3. WebSocket Proxy Token Validation via Custom Header

**Files:** `web/backend/api/pico.go:57-100`

The Pico WebSocket proxy validates tokens via a custom header. The token is compared against a cached config value. If the token changes in config while the gateway is running, there is a brief window where the cached token may be stale, allowing old tokens to work or rejecting valid ones.

### 4. Login Rate Limiting is In-Memory Only

**Files:** `web/backend/api/auth_login_limiter.go:17-40`

The dashboard login rate limiter uses in-memory maps keyed by IP address. Rate limits reset on process restart, and in distributed deployments each instance has independent limits.

### 5. No CSRF Protection on API Endpoints

**Files:** `web/backend/api/router.go:52-95`

The API routes use `http.ServeMux` directly with no CSRF middleware. If the server is run with `-public` flag, any website could make cross-origin requests unless CORS is properly configured.

---

## Performance

### 1. AgentLoop is a God File (3685 lines)

**File:** `pkg/agent/loop.go` (3685 lines)

Largest non-test file. Contains message routing (lines ~444-580), turn execution with tool loop (lines ~1800-2700), provider hot-reloading with goroutine isolation (lines ~982-1077), tool registration for all agents (lines ~165-442), and media resolution.

**Fix approach:** Extract sub-components into separate files within `pkg/agent/`.

### 2. Seahorse Store is Large (1542 lines)

**File:** `pkg/seahorse/store.go` (1542 lines)

All database operations in a single file with 44+ `ExecContext`/`QueryContext` calls. Uses parameterized queries (safe from injection), but size makes auditing difficult.

### 3. JSONL Store maxLineSize = 10 MB

**File:** `pkg/memory/jsonl.go:32`

A single tool result can be up to 10 MB. Messages are never physically deleted from JSONL files -- only logically skipped via metadata offset.

### 4. Media Cleanup Disabled

**File:** `pkg/agent/loop.go:487-498`

```go
// TODO: Re-enable media cleanup after inbound media is properly consumed by the agent.
// Currently disabled because files are deleted before the LLM can access their content.
```

Media files are never cleaned up, consuming disk space over time.

---

## Complexity

### 1. Complex Goroutine Lifecycle Management

Key unbounded goroutine spawns:
- `pkg/agent/loop.go:1001` -- Registry creation in goroutine with recover
- `pkg/seahorse/short_compaction.go:80` -- Async condensed compaction per conversation
- `web/backend/api/gateway.go:758-782` -- Multiple goroutines for gateway process management
- `pkg/agent/loop.go:477` -- `drainBusToSteering` goroutine per message

**Risk:** Goroutine leaks during error paths or rapid config reloads.

### 2. Global Mutable State in Gateway Package

**File:** `web/backend/api/gateway.go:29-43`

Package-level mutable singleton holding process state, config signatures, and cached tokens. Protected by `sync.Mutex` but creates tight coupling between gateway lifecycle and API handler.

### 3. Multiple Mutexes in Channel Implementations

- `pkg/channels/wecom/wecom.go` -- 6 separate mutexes
- `pkg/channels/onebot/onebot.go` -- 3 mutexes
- `pkg/channels/manager.go` -- 1 RWMutex plus per-channel operations

Increases deadlock risk if lock ordering is not consistent.

---

## Technical Debt

### 1. config_old.go -- Legacy Config Migration Code

**File:** `pkg/config/config_old.go` (1001 lines)

Contains V0 config structs for backward compatibility migration. Will grow as config schema evolves.

### 2. Media Cleanup Commented Out (TODO)

**File:** `pkg/agent/loop.go:487-498`

Known feature regression -- media cleanup disabled due to timing issue.

### 3. Logger TimeFormat Not Configurable (TODO)

**File:** `pkg/logger/logger.go:55`

### 4. MCP Tool Artifact Lifecycle Not Managed (TODO)

**File:** `pkg/tools/mcp_tool.go:365`

### 5. GitHub Copilot Provider Incomplete (TODO)

**File:** `pkg/providers/github_copilot_provider.go:29`

Only supports HTTP mode, not stdio.

### 6. ASR Model Restriction Incomplete (TODO)

**File:** `pkg/audio/asr/asr.go:36`

---

## Error Handling

### 1. Swallowed Errors in Config Reload

**File:** `web/backend/api/gateway.go:55-60`

```go
func refreshPicoTokensLocked(configPath string) {
    cfg, err := config.LoadConfig(h.configPath)
    if err != nil {
        return  // Error silently swallowed
    }
```

If config reload fails, the token is not updated and no error is logged.

### 2. Ignored `LastInsertId` Error

**File:** `pkg/seahorse/store.go:57`

### 3. Panics in Package Initialization

- `pkg/agent/context_seahorse.go:267`
- `pkg/logger/panic_unix.go:16-19`
- `web/backend/main.go:126-140`

---

## Scalability

### 1. Single-Process Architecture

Web backend, agent loop, channel connections, MCP servers, and cron jobs all share one process.

### 2. In-Memory Rate Limiting

**File:** `pkg/providers/ratelimiter.go:13`

All rate limiting is in-memory; state lost on restart.

### 3. JSONL File-per-Session Storage

**File:** `pkg/memory/jsonl.go:46-55`

Each session creates two files. Lock sharding (`numLockShards = 64`) mitigates contention but not file count growth.

---

## Dependencies

### 1. Large Dependency Surface for Channel Integrations

15+ messaging platform SDKs compiled in regardless of usage.

**Fix approach:** Consider build tags to compile only needed channels.

### 2. WebRTC Dependency for Discord Voice

**Files:** `pkg/channels/discord/voice.go`

Pion WebRTC stack (~10 transitive deps) used only for Discord voice.

---

## Maintainability

### 1. Large Test Files

- `pkg/agent/loop_test.go` -- 3367 lines
- `pkg/agent/subturn_test.go` -- 2067 lines
- `pkg/agent/steering_test.go` -- 1591 lines
- `pkg/config/config_test.go` -- 1976 lines

### 2. Duplicate Test Patterns

Agent test files contain repeated mock structures (`mockProvider`, `mockChannel`, `turnState`) defined inline rather than shared.

---

## Data Integrity

### 1. Race Condition Window in Conversation Creation

**File:** `pkg/seahorse/store.go:35-62`

Classic TOCTOU pattern handled via unique violation detection and retry. Correct for SQLite but fragile.

### 2. No WAL Mode Configuration for SQLite

Under concurrent load, this could cause "database is locked" errors. Compaction goroutines (`runCondensedLoop`) can write concurrently with ingestion.

---

## Missing Critical Features

### 1. No Audit Logging

No structured audit log for security-sensitive operations.

### 2. No Health/Metrics Endpoint

`pkg/health/server.go` provides basic health checking but no Prometheus-compatible metrics.

### 3. No Graceful Shutdown for All Components

`AgentLoop.Run()` returns on context cancellation, but sub-goroutines (compaction, media, drains) may outlive the main loop.

---

*Concerns analysis: 2026-04-10*
