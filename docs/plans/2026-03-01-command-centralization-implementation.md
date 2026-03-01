# Cross-Channel Command Centralization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Centralize generic command parsing/policy/execution in `pkg/commands`, remove duplicated command business logic from `AgentLoop` and channel adapters, and enforce consistent cross-channel behavior.

**Architecture:** Add a tri-state command executor (`handled` / `rejected` / `passthrough`) in `pkg/commands` and make `AgentLoop.processMessage` call it before LLM flow. Command handlers receive minimal runtime capabilities from agent (session operations, scope key, config, channel). Channels stop local business command execution and only forward inbound messages + platform command registration metadata.

**Tech Stack:** Go, existing `pkg/commands`, `pkg/agent`, `pkg/channels/*`, table-driven tests with `go test`.

---

**Required skills during execution:** @test-driven-development, @verification-before-completion, @requesting-code-review

### Task 1: Add Executor Tri-State Contract in `pkg/commands`

**Files:**
- Create: `pkg/commands/executor.go`
- Test: `pkg/commands/executor_test.go`
- Modify: `pkg/commands/definition.go`

**Step 1: Write the failing test**

```go
func TestExecutor_RegisteredButUnsupported_ReturnsRejected(t *testing.T) {
	defs := []Definition{{Name: "show", Channels: []string{"telegram"}}}
	ex := NewExecutor(NewRegistry(defs))

	res := ex.Execute(context.Background(), Request{Channel: "whatsapp", Text: "/show"}, nil)
	if res.Outcome != OutcomeRejected {
		t.Fatalf("outcome=%v", res.Outcome)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/commands -run TestExecutor_RegisteredButUnsupported_ReturnsRejected -v`  
Expected: FAIL with undefined `NewExecutor/OutcomeRejected`.

**Step 3: Write minimal implementation**

```go
type Outcome int

const (
	OutcomePassthrough Outcome = iota
	OutcomeHandled
	OutcomeRejected
)

type ExecuteResult struct {
	Outcome Outcome
	Command string
	Reply   string
	Err     error
}

type Executor struct { reg *Registry }
```

Implement `Execute(...)` to:
- parse slash command token,
- detect `registered+unsupported => rejected`,
- detect `unknown => passthrough`,
- keep `handled` path for supported commands with handlers.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/commands -run TestExecutor_RegisteredButUnsupported_ReturnsRejected -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/commands/executor.go pkg/commands/executor_test.go pkg/commands/definition.go
git commit -m "feat(commands): add tri-state command executor contract"
```

### Task 2: Define Command Runtime Interfaces for Agent-Backed Handlers

**Files:**
- Create: `pkg/commands/runtime.go`
- Test: `pkg/commands/runtime_test.go`

**Step 1: Write the failing test**

```go
func TestRuntimeContracts_MinimalSessionOps(t *testing.T) {
	var _ SessionOps = (*fakeSessionOps)(nil)
	var _ Runtime = (*fakeRuntime)(nil)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/commands -run TestRuntimeContracts_MinimalSessionOps -v`  
Expected: FAIL with undefined interfaces.

**Step 3: Write minimal implementation**

```go
type SessionOps interface {
	ResolveActive(scopeKey string) (string, error)
	StartNew(scopeKey string) (string, error)
	List(scopeKey string) ([]session.SessionMeta, error)
	Resume(scopeKey string, index int) (string, error)
	Prune(scopeKey string, limit int) ([]string, error)
}

type Runtime interface {
	Channel() string
	ScopeKey() string
	SessionOps() SessionOps
	Config() *config.Config
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/commands -run TestRuntimeContracts_MinimalSessionOps -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/commands/runtime.go pkg/commands/runtime_test.go
git commit -m "refactor(commands): add runtime interfaces for agent-backed handlers"
```

### Task 3: Move `/new` and `/session` Business Logic into `pkg/commands`

**Files:**
- Modify: `pkg/commands/builtin.go`
- Test: `pkg/commands/builtin_test.go`
- Create: `pkg/commands/session_handlers_test.go`

**Step 1: Write the failing test**

```go
func TestSessionHandlers_NewAndResume_UseRuntimeSessionOps(t *testing.T) {
	// build fake runtime + fake request channel=whatsapp
	// execute /new then /session resume 1 through Executor
	// assert handler path is OutcomeHandled and fake session ops were called
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/commands -run TestSessionHandlers_NewAndResume_UseRuntimeSessionOps -v`  
Expected: FAIL because `/new` and `/session` definitions are metadata-only.

**Step 3: Write minimal implementation**

Add handlers for:
- `/new` and alias `/reset` using `runtime.SessionOps().StartNew(...)` + `Prune(...)`
- `/session list`
- `/session resume <index>`

Keep existing output format stable where possible.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/commands -run TestSessionHandlers_NewAndResume_UseRuntimeSessionOps -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/commands/builtin.go pkg/commands/builtin_test.go pkg/commands/session_handlers_test.go
git commit -m "feat(commands): implement session command handlers via runtime"
```

### Task 4: Move `/show` and `/list` Agent Logic into Command Handlers

**Files:**
- Modify: `pkg/commands/builtin.go`
- Test: `pkg/commands/builtin_test.go`
- Create: `pkg/commands/show_list_handlers_test.go`

**Step 1: Write the failing test**

```go
func TestShowListHandlers_ChannelPolicy(t *testing.T) {
	// /show on telegram => handled
	// /show on whatsapp => rejected (registered but unsupported)
	// /foo on whatsapp => passthrough
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/commands -run TestShowListHandlers_ChannelPolicy -v`  
Expected: FAIL with missing rejected/passthrough policy for these paths.

**Step 3: Write minimal implementation**

- Keep `/show` and `/list` handlers in `pkg/commands` only.
- Ensure unsupported registered command returns `OutcomeRejected` with explicit user text.
- Ensure unknown slash command returns `OutcomePassthrough`.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/commands -run TestShowListHandlers_ChannelPolicy -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/commands/builtin.go pkg/commands/builtin_test.go pkg/commands/show_list_handlers_test.go
git commit -m "fix(commands): enforce reject-vs-passthrough command policy"
```

### Task 5: Wire `AgentLoop` to Executor and Remove Hardcoded Command Switch

**Files:**
- Modify: `pkg/agent/loop.go`
- Test: `pkg/agent/loop_test.go`

**Step 1: Write the failing test**

```go
func TestProcessMessage_CommandOutcomes(t *testing.T) {
	// Case A: /show on whatsapp => explicit unsupported message, no LLM call
	// Case B: /foo on whatsapp => passes through and triggers LLM call
	// Case C: /new on whatsapp => handled via commands runtime
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/agent -run TestProcessMessage_CommandOutcomes -v`  
Expected: FAIL while `AgentLoop.handleCommand` switch remains authoritative.

**Step 3: Write minimal implementation**

- Add command executor field initialization in `AgentLoop` construction.
- Build runtime adapter from `route+agent+config`.
- In `processMessage`, call executor and branch by tri-state.
- Remove command business switch for `/new`, `/session`, `/show`, `/list`.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/agent -run TestProcessMessage_CommandOutcomes -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/agent/loop.go pkg/agent/loop_test.go
git commit -m "refactor(agent): delegate generic command execution to commands executor"
```

### Task 6: Stop Channel-Side Generic Command Business Execution

**Files:**
- Modify: `pkg/channels/telegram/telegram.go`
- Modify: `pkg/channels/telegram/telegram_dispatch.go`
- Modify: `pkg/channels/whatsapp/whatsapp.go`
- Modify: `pkg/channels/whatsapp_native/whatsapp_native.go`
- Test: `pkg/channels/telegram/telegram_dispatch_test.go`
- Test: `pkg/channels/whatsapp/whatsapp_command_test.go`
- Test: `pkg/channels/whatsapp_native/whatsapp_command_test.go`

**Step 1: Write the failing test**

```go
func TestChannelInbound_DoesNotConsumeGenericCommandsLocally(t *testing.T) {
	// verify inbound /help,/new are still forwarded to bus/agent path,
	// while Telegram registration path remains intact
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/channels/telegram ./pkg/channels/whatsapp -run DoesNotConsumeGenericCommandsLocally -v`  
Expected: FAIL with existing local interception path.

**Step 3: Write minimal implementation**

- Remove/disable local generic command dispatch short-circuit in inbound handlers.
- Keep Telegram `RegisterCommands(...)` startup behavior unchanged.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/channels/telegram ./pkg/channels/whatsapp -run DoesNotConsumeGenericCommandsLocally -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/channels/telegram/telegram.go pkg/channels/telegram/telegram_dispatch.go \
  pkg/channels/whatsapp/whatsapp.go pkg/channels/whatsapp_native/whatsapp_native.go \
  pkg/channels/telegram/telegram_dispatch_test.go pkg/channels/whatsapp/whatsapp_command_test.go \
  pkg/channels/whatsapp_native/whatsapp_command_test.go
git commit -m "refactor(channels): forward generic commands to agent-centric executor path"
```

### Task 7: Documentation + Full Verification + Review Gate

**Files:**
- Modify: `README.md`
- Modify: `README.zh.md`
- Modify: `docs/plans/2026-03-01-command-centralization-design.md` (if needed for implementation notes)

**Step 1: Add/extend failing behavior test (optional smoke)**

```go
// Optional smoke in pkg/agent/loop_test.go:
// /show on whatsapp returns unsupported, /foo passes to LLM
```

**Step 2: Run targeted verification**

Run: `go test ./pkg/commands ./pkg/agent ./pkg/channels/... -count=1`  
Expected: PASS.

**Step 3: Update docs**

- Document unified command execution path.
- Document policy:
  - unknown slash => LLM passthrough,
  - registered but unsupported => explicit error.

**Step 4: Run full verification**

Run: `go test ./... -count=1`  
Expected: PASS.

Run: `git status --short`  
Expected: clean working tree.

**Step 5: Request code review before merge**

Use `@requesting-code-review` skill with `BASE_SHA` and `HEAD_SHA`, fix Critical/Important findings before merge.

**Step 6: Commit docs updates**

```bash
git add README.md README.zh.md docs/plans/2026-03-01-command-centralization-design.md
git commit -m "docs(commands): document centralized command execution policy"
```
