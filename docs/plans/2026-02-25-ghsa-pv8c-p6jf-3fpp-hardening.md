# GHSA-pv8c-p6jf-3fpp Hardening Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Break the unauthenticated-to-RCE chain by enforcing fail-closed ingress auth and fail-closed dangerous tool policies in network-exposed deployments.

**Architecture:** Apply trust-boundary hardening in layers: channel ingress verification, strict tool execution policy, SSRF connect-time controls, and persistence/permissions hardening. Implement with TDD and small commits so each security claim is backed by one regression test.

**Tech Stack:** Go, `testing`, `httptest`, PicoClaw `pkg/channels`, `pkg/tools`, `pkg/config`, `pkg/state`.

---

### Task 0: Channel Ingress Auth Matrix And Fail-Closed Validation

**Files:**
- Modify: `pkg/channels/manager.go`
- Create: `pkg/channels/security_matrix_test.go`
- Modify: `docs/tools_configuration.md`
- Modify: `.docs/GHSA-pv8c-p6jf-3fpp-context-analysis.md`

**Step 1: Write the failing test**

Add `pkg/channels/security_matrix_test.go`:

```go
func TestChannelSecurityMatrix_FailClosedRequirements(t *testing.T) {
	cfg := config.DefaultConfig()
	msgBus := bus.NewMessageBus()

	cfg.Channels.WeComApp.Enabled = true
	cfg.Channels.WeComApp.CorpID = "corp"
	cfg.Channels.WeComApp.CorpSecret = "secret"
	cfg.Channels.WeComApp.AgentID = 1000002
	cfg.Channels.WeComApp.Token = "" // must fail closed

	m, err := NewManager(cfg, msgBus)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}

	if _, ok := m.channels["wecom_app"]; ok {
		t.Fatal("wecom_app must not be enabled without token")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/channels -run TestChannelSecurityMatrix_FailClosedRequirements -v`
Expected: FAIL if manager still enables channels with missing critical auth fields.

**Step 3: Write minimal implementation**

In `pkg/channels/manager.go`, enforce explicit fail-closed checks before channel creation:

- `wecom`: require `Enabled && Token != ""`
- `wecom_app`: require `Enabled && CorpID != "" && CorpSecret != "" && AgentID != 0 && Token != ""`
- add similar explicit checks for any webhook-based channel that has a verification secret field.

Add inline comment near each check:
- `"Fail closed: do not expose webhook channel without verification secret."`

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/channels -run TestChannelSecurityMatrix_FailClosedRequirements -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/channels/manager.go pkg/channels/security_matrix_test.go docs/tools_configuration.md .docs/GHSA-pv8c-p6jf-3fpp-context-analysis.md
git commit -m "fix(security): enforce fail-closed channel auth initialization"
```

---

### Task 1: Add Exec Remote Policy Config

**Files:**
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/defaults.go`
- Modify: `pkg/config/config_test.go`
- Modify: `docs/tools_configuration.md`

**Step 1: Write the failing test**

Add tests in `pkg/config/config_test.go`:

```go
func TestDefaultConfig_ExecAllowRemoteDisabled(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Tools.Exec.AllowRemote {
		t.Fatal("Tools.Exec.AllowRemote should default to false")
	}
}

func TestLoadConfig_ExecAllowRemoteFromEnv(t *testing.T) {
	t.Setenv("PICOCLAW_TOOLS_EXEC_ALLOW_REMOTE", "true")
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if !cfg.Tools.Exec.AllowRemote {
		t.Fatal("expected Tools.Exec.AllowRemote=true from env")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/config -run 'TestDefaultConfig_ExecAllowRemoteDisabled|TestLoadConfig_ExecAllowRemoteFromEnv' -v`
Expected: FAIL because `ExecConfig.AllowRemote` does not exist.

**Step 3: Write minimal implementation**

Update `pkg/config/config.go`:

```go
type ExecConfig struct {
	EnableDenyPatterns bool     `json:"enable_deny_patterns" env:"PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS"`
	CustomDenyPatterns []string `json:"custom_deny_patterns" env:"PICOCLAW_TOOLS_EXEC_CUSTOM_DENY_PATTERNS"`
	AllowRemote        bool     `json:"allow_remote" env:"PICOCLAW_TOOLS_EXEC_ALLOW_REMOTE"`
}
```

Update `pkg/config/defaults.go`:

```go
Exec: ExecConfig{
	EnableDenyPatterns: true,
	AllowRemote:        false,
},
```

Document in `docs/tools_configuration.md`:
- `tools.exec.allow_remote` / `PICOCLAW_TOOLS_EXEC_ALLOW_REMOTE`
- default `false`
- security note: enabling allows remote channels to run shell commands.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/config -run 'TestDefaultConfig_ExecAllowRemoteDisabled|TestLoadConfig_ExecAllowRemoteFromEnv' -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/config/config.go pkg/config/defaults.go pkg/config/config_test.go docs/tools_configuration.md
git commit -m "feat(security): add exec remote policy config"
```

---

### Task 2: Enforce Fail-Closed Exec Channel Guard

**Files:**
- Modify: `pkg/tools/shell.go`
- Modify: `pkg/tools/shell_test.go`
- Modify: `pkg/agent/loop.go`
- Modify: `pkg/agent/loop_test.go`
- Modify: `pkg/constants/channels.go`

**Step 1: Write the failing tests**

Add tests in `pkg/tools/shell_test.go`:

```go
func TestShellTool_NoContextSetBlocked(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewExecToolWithConfig("", false, cfg)
	result := tool.Execute(context.Background(), map[string]any{"command": "echo hi"})
	if !result.IsError || !strings.Contains(result.ForLLM, "disabled for remote channels") {
		t.Fatalf("expected fail-closed block, got: %#v", result)
	}
}

func TestShellTool_RemoteChannelBlockedByDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewExecToolWithConfig("", false, cfg)
	tool.SetContext("telegram", "chat-1")
	result := tool.Execute(context.Background(), map[string]any{"command": "echo hi"})
	if !result.IsError {
		t.Fatal("expected remote-channel exec to be blocked")
	}
}

func TestShellTool_InternalChannelAllowed(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewExecToolWithConfig("", false, cfg)
	tool.SetContext("cli", "direct")
	result := tool.Execute(context.Background(), map[string]any{"command": "echo hi"})
	if result.IsError {
		t.Fatalf("expected internal channel allow, got: %s", result.ForLLM)
	}
}
```

Add tests in `pkg/agent/loop_test.go` to verify `updateToolContexts` sets context for `exec` and `cron`.

Add tests in `pkg/constants/channels.go` companion test to verify strict allowlist (`cli`, `system`, `subagent`) and remote channels return false.

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/tools -run 'TestShellTool_NoContextSetBlocked|TestShellTool_RemoteChannelBlockedByDefault|TestShellTool_InternalChannelAllowed' -v`
Expected: FAIL.

Run: `go test ./pkg/agent -run TestUpdateToolContexts_ExecAndCron -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

In `pkg/tools/shell.go`:
- Add fields: `allowRemote`, `channel`, `chatID`
- Implement `SetContext(channel, chatID string)`
- Add compile-time assertion:

```go
var _ ContextualTool = (*ExecTool)(nil)
```

- Fail-closed guard in `Execute`:

```go
if !t.allowRemote && !constants.IsInternalChannel(t.channel) {
	return ErrorResult("command execution is disabled for remote channels; set tools.exec.allow_remote=true to override")
}
```

Do not add `t.channel != ""` check.

Wire `allowRemote` from config in constructor.

In `pkg/agent/loop.go` update `updateToolContexts` to call `SetContext` for `exec` and `cron`.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/tools -run 'TestShellTool_NoContextSetBlocked|TestShellTool_RemoteChannelBlockedByDefault|TestShellTool_InternalChannelAllowed' -v`
Expected: PASS.

Run: `go test ./pkg/agent -run TestUpdateToolContexts_ExecAndCron -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/tools/shell.go pkg/tools/shell_test.go pkg/agent/loop.go pkg/agent/loop_test.go pkg/constants/channels.go
git commit -m "fix(security): enforce fail-closed exec channel policy"
```

---

### Task 3: Harden Cron Command Scheduling Policy

**Files:**
- Modify: `pkg/tools/cron.go`
- Create: `pkg/tools/cron_test.go`
- Modify: `docs/tools_configuration.md`

**Step 1: Write the failing tests**

Add tests in `pkg/tools/cron_test.go`:

```go
func TestCronTool_AddRemoteCommandBlocked(t *testing.T) {
	tool := newCronToolForTest(t, "telegram", "chat1")
	result := tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "run command",
		"at_seconds": float64(60),
		"command":    "id",
	})
	if !result.IsError || !strings.Contains(result.ForLLM, "not allowed from remote channels") {
		t.Fatalf("expected remote command block, got: %#v", result)
	}
}

func TestCronTool_AddCommandRequiresConfirm(t *testing.T) {
	tool := newCronToolForTest(t, "cli", "direct")
	result := tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "run command",
		"at_seconds": float64(60),
		"command":    "id",
	})
	if !result.IsError || !strings.Contains(result.ForLLM, "command_confirm=true") {
		t.Fatalf("expected command_confirm validation error, got: %#v", result)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/tools -run 'TestCronTool_AddRemoteCommandBlocked|TestCronTool_AddCommandRequiresConfirm' -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

In `pkg/tools/cron.go`:

1. Enforce remote-channel block for `command` scheduling:

```go
if command != "" && !constants.IsInternalChannel(channel) {
	return ErrorResult("cron command scheduling is not allowed from remote channels")
}
```

2. Require `command_confirm=true` only for internal channels as friction:

```go
if command != "" {
	commandConfirm, _ := args["command_confirm"].(bool)
	if !commandConfirm {
		return ErrorResult("command_confirm=true is required when scheduling shell commands")
	}
}
```

3. Add comment in code/docs:
- `command_confirm` is defense-in-depth only, not an authentication boundary.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/tools -run 'TestCronTool_AddRemoteCommandBlocked|TestCronTool_AddCommandRequiresConfirm' -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/tools/cron.go pkg/tools/cron_test.go docs/tools_configuration.md
git commit -m "fix(security): restrict cron command scheduling to internal channels"
```

---

### Task 4: Block Suspicious Skill Installs By Default With Explicit Criteria

**Files:**
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/defaults.go`
- Modify: `pkg/tools/skills_install.go`
- Modify: `pkg/tools/skills_install_test.go`
- Modify: `pkg/agent/loop.go`

**Step 1: Write the failing tests**

In `pkg/tools/skills_install_test.go`, add:

```go
func TestInstallSkillTool_BlocksSuspiciousByDefault(t *testing.T) { ... }
func TestInstallSkillTool_AllowsSuspiciousWithConfig(t *testing.T) { ... }
```

Add one concrete criteria-driven test:
- if registry returns `IsSuspicious=true`, install must fail when config default is false.

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/tools -run 'TestInstallSkillTool_BlocksSuspiciousByDefault|TestInstallSkillTool_AllowsSuspiciousWithConfig' -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

Add config:

```go
AllowSuspiciousInstall bool `json:"allow_suspicious_install" env:"PICOCLAW_SKILLS_ALLOW_SUSPICIOUS_INSTALL"`
```

Default:

```go
AllowSuspiciousInstall: false,
```

In `pkg/tools/skills_install.go`:
- add constructor param `allowSuspicious bool`
- if `result.IsSuspicious && !allowSuspicious`: remove target dir and return error.

In `pkg/agent/loop.go`, wire constructor with config.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/tools -run 'TestInstallSkillTool_BlocksSuspiciousByDefault|TestInstallSkillTool_AllowsSuspiciousWithConfig' -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/config/config.go pkg/config/defaults.go pkg/tools/skills_install.go pkg/tools/skills_install_test.go pkg/agent/loop.go
git commit -m "fix(security): block suspicious skill installs by default"
```

---

### Task 5: State File Permission Hardening (Final Path + Directory)

**Files:**
- Modify: `pkg/state/state.go`
- Modify: `pkg/state/state_test.go`

**Step 1: Write the failing test**

Add tests in `pkg/state/state_test.go`:

```go
func TestStateFilePermissions0600(t *testing.T) { ... }
func TestStateDirPermissions0700(t *testing.T) { ... }
```

Assertions:
- `state/state.json` mode is `0600`.
- `state/` mode is `0700` (non-Windows).

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/state -run 'TestStateFilePermissions0600|TestStateDirPermissions0700' -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

In `pkg/state/state.go`:

1. Create state dir with `0700`:

```go
os.MkdirAll(stateDir, 0o700)
```

2. Write temp file with `0600`:

```go
os.WriteFile(tempFile, data, 0o600)
```

3. After rename, enforce final mode:

```go
if err := os.Chmod(sm.stateFile, 0o600); err != nil { ... }
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/state -run 'TestStateFilePermissions0600|TestStateDirPermissions0700' -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/state/state.go pkg/state/state_test.go
git commit -m "fix(security): enforce secure state file and directory permissions"
```

---

### Task 6: Strengthen SSRF Validation And Tests (Rebinding/IPv6/Metadata)

**Files:**
- Modify: `pkg/tools/web.go`
- Modify: `pkg/tools/web_test.go`

**Step 1: Write the failing tests**

Add tests in `pkg/tools/web_test.go`:

```go
func TestWebFetch_BlocksIPv6MappedLoopback(t *testing.T) { ... }      // ::ffff:127.0.0.1
func TestWebFetch_BlocksMetadataIP(t *testing.T) { ... }              // 169.254.169.254
func TestWebFetch_RedirectToPrivateBlocked(t *testing.T) { ... }      // existing + strict assertion
func TestWebFetch_DNSRebindingMitigationConnectTime(t *testing.T) { ... } // custom dialer check behavior
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/tools -run 'TestWebFetch_(BlocksIPv6MappedLoopback|BlocksMetadataIP|RedirectToPrivateBlocked|DNSRebindingMitigationConnectTime)' -v`
Expected: FAIL for missing checks/coverage.

**Step 3: Write minimal implementation**

In `pkg/tools/web.go`:
- Ensure host classification includes IPv4/IPv6 loopback/link-local/private ranges and metadata IP.
- Ensure enforcement at connect-time in HTTP client transport/dial path, not only preflight hostname checks.
- Keep redirect checks.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/tools -run 'TestWebFetch_(BlocksIPv6MappedLoopback|BlocksMetadataIP|RedirectToPrivateBlocked|DNSRebindingMitigationConnectTime)' -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/tools/web.go pkg/tools/web_test.go
git commit -m "fix(security): harden web_fetch against advanced SSRF vectors"
```

---

### Task 7: Regression Sweep And Upgrade Notes

**Files:**
- Modify: `pkg/channels/wecom_test.go`
- Modify: `pkg/channels/wecom_app_test.go`
- Modify: `docs/tools_configuration.md`
- Modify: `.docs/GHSA-pv8c-p6jf-3fpp-context-analysis.md`
- Modify: `README.md`

**Step 1: Write remaining regression tests/document checks**

Ensure explicit regressions exist for:
- empty token verification failures in WeCom paths
- remote exec blocked by default
- remote cron command scheduling blocked
- suspicious skill install blocked by default

**Step 2: Run verification commands**

Run:

```bash
go test ./pkg/config ./pkg/tools ./pkg/channels ./pkg/state -v
go vet ./...
staticcheck ./...
```

Expected: PASS.

**Step 3: Write upgrade notes**

Document new keys and defaults:
- `tools.exec.allow_remote=false`
- `tools.skills.allow_suspicious_install=false`
- cron command restrictions and `command_confirm`

Add rollback instructions:
- temporary override env vars for emergency compatibility.

**Step 4: Commit**

```bash
git add pkg/channels/wecom_test.go pkg/channels/wecom_app_test.go docs/tools_configuration.md .docs/GHSA-pv8c-p6jf-3fpp-context-analysis.md README.md
git commit -m "docs(security): add regressions and upgrade notes for GHSA hardening"
```

**Step 5: Prepare PR**

Include:
- broken attack-chain summary (before/after)
- tests proving each boundary
- migration notes
- residual risks + follow-up tasks (human approval flow for dangerous actions)

