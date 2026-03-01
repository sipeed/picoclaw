# Plugin System Phase 2/3 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement Phase 2 (config-driven plugin selection + runtime wiring) and Phase 3 (plugin introspection + CLI list/lint + diagnostics) with small, reviewable PRs.

**Architecture:** Keep in-process compile-time plugins as the only runtime model. Build a deterministic selection plane first (`config` + `plugin` resolver + bootstrap wiring), then add a non-breaking introspection plane (`PluginInfo`, list/lint commands) without changing `plugin.Plugin` contract. Use TDD for each slice and keep changes split into small commits.

**Tech Stack:** Go, Cobra CLI, Testify, existing PicoClaw `pkg/config`, `pkg/plugin`, `pkg/agent`, `cmd/picoclaw/internal/*`.

---

## Preflight Notes

- Execute this plan inside the dedicated worktree created during design/brainstorming.
- Use `@test-driven-development` for every task (`red -> green -> refactor`).
- Before final handoff, use `@verification-before-completion`.
- Before opening PRs, use `@requesting-code-review`.

### Task 1: Add Plugin Config Schema and Defaults (Phase 2)

**Files:**
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/defaults.go`
- Test: `pkg/config/config_test.go`

**Step 1: Write the failing test**

```go
func TestDefaultConfig_PluginsDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Plugins.DefaultEnabled {
		t.Fatal("plugins.default_enabled should default to true")
	}
	if len(cfg.Plugins.Enabled) != 0 || len(cfg.Plugins.Disabled) != 0 {
		t.Fatal("plugins enabled/disabled should default empty")
	}
}

func TestConfig_PluginsJSONUnmarshal(t *testing.T) {
	cfg := DefaultConfig()
	err := json.Unmarshal([]byte(`{"plugins":{"default_enabled":false,"enabled":["policy-demo"],"disabled":["x"]}}`), cfg)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.Plugins.DefaultEnabled {
		t.Fatal("expected default_enabled=false from JSON")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/config -run 'TestDefaultConfig_PluginsDefaults|TestConfig_PluginsJSONUnmarshal' -v`  
Expected: FAIL with `cfg.Plugins undefined`.

**Step 3: Write minimal implementation**

```go
type PluginsConfig struct {
	DefaultEnabled bool     `json:"default_enabled"`
	Enabled        []string `json:"enabled,omitempty"`
	Disabled       []string `json:"disabled,omitempty"`
}

type Config struct {
	// ...existing fields...
	Plugins PluginsConfig `json:"plugins,omitempty"`
}
```

```go
Plugins: PluginsConfig{
	DefaultEnabled: true,
	Enabled:        []string{},
	Disabled:       []string{},
},
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/config -run 'TestDefaultConfig_PluginsDefaults|TestConfig_PluginsJSONUnmarshal' -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/config/config.go pkg/config/defaults.go pkg/config/config_test.go
git commit -m "feat(config): add plugins selection config schema"
```

### Task 2: Build Deterministic Selection Resolver (Phase 2)

**Files:**
- Modify: `pkg/plugin/manager.go`
- Test: `pkg/plugin/manager_test.go`

**Step 1: Write the failing test**

```go
func TestResolveSelection_DefaultEnabled(t *testing.T) {}
func TestResolveSelection_EnabledListOnly(t *testing.T) {}
func TestResolveSelection_DisabledWinsOverlap(t *testing.T) {}
func TestResolveSelection_UnknownEnabledFails(t *testing.T) {}
func TestResolveSelection_UnknownDisabledWarns(t *testing.T) {}
func TestResolveSelection_NormalizeAndDedupe(t *testing.T) {}
```

In each test, assert deterministic sorted resolution and expected error/warning behavior.

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/plugin -run 'TestResolveSelection_' -v`  
Expected: FAIL with undefined resolver types/functions.

**Step 3: Write minimal implementation**

```go
type SelectionInput struct {
	DefaultEnabled bool
	Enabled        []string
	Disabled       []string
}

type SelectionResult struct {
	EnabledNames    []string
	DisabledNames   []string
	UnknownEnabled  []string
	UnknownDisabled []string
	Warnings        []string
}

func NormalizePluginName(s string) string { /* strings.TrimSpace + strings.ToLower */ }
func ResolveSelection(available []string, in SelectionInput) (SelectionResult, error) { /* deterministic rules */ }
```

Rules implemented exactly:
- unknown in `enabled` => error
- unknown in `disabled` => warning bucket
- overlap => disabled wins
- sorted deterministic output

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/plugin -run 'TestResolveSelection_' -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/plugin/manager.go pkg/plugin/manager_test.go
git commit -m "feat(plugin): add deterministic plugin selection resolver"
```

### Task 3: Add Built-in Plugin Catalog Without Import Cycles (Phase 2)

**Files:**
- Create: `pkg/plugin/builtin/catalog.go`
- Test: `pkg/plugin/builtin/catalog_test.go`

**Step 1: Write the failing test**

```go
func TestCatalogContainsPolicyDemo(t *testing.T) {
	c := Catalog()
	fn, ok := c["policy-demo"]
	if !ok {
		t.Fatal("expected policy-demo in builtin catalog")
	}
	if fn() == nil {
		t.Fatal("expected non-nil plugin instance")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/plugin/builtin -run TestCatalogContainsPolicyDemo -v`  
Expected: FAIL with package/file missing.

**Step 3: Write minimal implementation**

```go
package builtin

import (
	"github.com/sipeed/picoclaw/pkg/plugin"
	"github.com/sipeed/picoclaw/pkg/plugin/demoplugin"
)

type Factory func() plugin.Plugin

func Catalog() map[string]Factory {
	return map[string]Factory{
		"policy-demo": func() plugin.Plugin {
			return demoplugin.NewPolicyDemoPlugin(demoplugin.PolicyDemoConfig{})
		},
	}
}

func Names() []string { /* return sorted keys */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/plugin/builtin -run TestCatalogContainsPolicyDemo -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/plugin/builtin/catalog.go pkg/plugin/builtin/catalog_test.go
git commit -m "feat(plugin): add builtin plugin catalog package"
```

### Task 4: Add Bootstrap Resolver Module for Agent/Gateway (Phase 2)

**Files:**
- Create: `cmd/picoclaw/internal/pluginruntime/bootstrap.go`
- Test: `cmd/picoclaw/internal/pluginruntime/bootstrap_test.go`

**Step 1: Write the failing test**

```go
func TestResolveConfiguredPlugins_UnknownEnabledReturnsError(t *testing.T) {}
func TestResolveConfiguredPlugins_ReturnsDeterministicInstances(t *testing.T) {}
func TestResolveConfiguredPlugins_UnknownDisabledWarns(t *testing.T) {}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/picoclaw/internal/pluginruntime -run 'TestResolveConfiguredPlugins_' -v`  
Expected: FAIL with package/file missing.

**Step 3: Write minimal implementation**

```go
type Summary struct {
	Enabled         []string
	Disabled        []string
	UnknownEnabled  []string
	UnknownDisabled []string
	Warnings        []string
}

func ResolveConfiguredPlugins(cfg *config.Config) ([]plugin.Plugin, Summary, error) {
	// 1) get catalog names from builtin.Names()
	// 2) call plugin.ResolveSelection(...)
	// 3) instantiate enabled plugins via builtin.Catalog factories
	// 4) return instances + summary
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/picoclaw/internal/pluginruntime -run 'TestResolveConfiguredPlugins_' -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add cmd/picoclaw/internal/pluginruntime/bootstrap.go cmd/picoclaw/internal/pluginruntime/bootstrap_test.go
git commit -m "feat(cli): add plugin runtime bootstrap resolver"
```

### Task 5: Wire Phase 2 Plugin Bootstrap into `agent` and `gateway`

**Files:**
- Modify: `cmd/picoclaw/internal/agent/helpers.go`
- Modify: `cmd/picoclaw/internal/gateway/helpers.go`
- Test: `cmd/picoclaw/internal/agent/command_test.go`
- Test: `cmd/picoclaw/internal/gateway/command_test.go`

**Step 1: Write the failing test**

Add focused assertions that command constructors remain stable after importing plugin bootstrap package and wiring helper calls (no regressions in command metadata).  
If needed, add table-driven compile/runtime smoke tests in a new `_test.go` under each package.

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/picoclaw/internal/agent ./cmd/picoclaw/internal/gateway -run 'TestNew.*Command|Test.*Plugin.*' -v`  
Expected: FAIL once bootstrap calls are referenced but not integrated correctly.

**Step 3: Write minimal implementation**

```go
pluginsToEnable, summary, err := pluginruntime.ResolveConfiguredPlugins(cfg)
if err != nil {
	return fmt.Errorf("resolve plugins: %w", err)
}
if len(pluginsToEnable) > 0 {
	if err := agentLoop.EnablePlugins(pluginsToEnable...); err != nil {
		return fmt.Errorf("enable plugins: %w", err)
	}
}
logger.InfoCF("plugin", "Plugin selection resolved", map[string]any{
	"enabled": summary.Enabled, "disabled": summary.Disabled,
	"unknown_disabled": summary.UnknownDisabled,
})
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/picoclaw/internal/agent ./cmd/picoclaw/internal/gateway -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add cmd/picoclaw/internal/agent/helpers.go cmd/picoclaw/internal/gateway/helpers.go cmd/picoclaw/internal/agent/command_test.go cmd/picoclaw/internal/gateway/command_test.go
git commit -m "feat(cli): wire plugin selection into agent and gateway startup"
```

### Task 6: Expose Plugin Resolution in Startup Diagnostics (Phase 2)

**Files:**
- Modify: `pkg/agent/loop.go`
- Test: `pkg/agent/loop_test.go`
- Test: `pkg/agent/plugin_test.go`

**Step 1: Write the failing test**

```go
func TestGetStartupInfo_IncludesPluginSummary(t *testing.T) {
	// create AgentLoop, enable a test plugin, call GetStartupInfo
	// assert "plugins" key exists with enabled list
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/agent -run 'TestGetStartupInfo_IncludesPluginSummary' -v`  
Expected: FAIL because `plugins` section is absent.

**Step 3: Write minimal implementation**

```go
if al.pluginManager != nil {
	info["plugins"] = map[string]any{
		"enabled": al.pluginManager.Names(),
		"count":   len(al.pluginManager.Names()),
	}
} else {
	info["plugins"] = map[string]any{
		"enabled": []string{},
		"count":   0,
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/agent -run 'TestGetStartupInfo_IncludesPluginSummary|TestSetPluginManagerInstallsHookRegistry' -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/agent/loop.go pkg/agent/loop_test.go pkg/agent/plugin_test.go
git commit -m "feat(agent): include plugin summary in startup diagnostics"
```

### Task 7: Add Non-Breaking Plugin Metadata Introspection (Phase 3)

**Files:**
- Modify: `pkg/plugin/manager.go`
- Test: `pkg/plugin/manager_test.go`

**Step 1: Write the failing test**

```go
func TestDescribeAll_UsesDescriptorWhenImplemented(t *testing.T) {}
func TestDescribeAll_FallsBackForPlainPlugin(t *testing.T) {}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/plugin -run 'TestDescribeAll_' -v`  
Expected: FAIL with undefined `PluginInfo` / `DescribeAll`.

**Step 3: Write minimal implementation**

```go
type PluginInfo struct {
	Name       string `json:"name"`
	APIVersion string `json:"api_version"`
	Status     string `json:"status"`
}

type PluginDescriptor interface {
	Info() PluginInfo
}

func (m *Manager) DescribeAll() []PluginInfo { /* include fallback info */ }
func (m *Manager) DescribeEnabled() []PluginInfo { /* status=enabled */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/plugin -run 'TestDescribeAll_|TestRegisterPluginAndTriggerHook' -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add pkg/plugin/manager.go pkg/plugin/manager_test.go
git commit -m "feat(plugin): add non-breaking plugin metadata introspection"
```

### Task 8: Add `picoclaw plugin list` Command (Phase 3)

**Files:**
- Create: `cmd/picoclaw/internal/plugin/command.go`
- Create: `cmd/picoclaw/internal/plugin/list.go`
- Test: `cmd/picoclaw/internal/plugin/command_test.go`
- Test: `cmd/picoclaw/internal/plugin/list_test.go`
- Modify: `cmd/picoclaw/main.go`
- Modify: `cmd/picoclaw/main_test.go`

**Step 1: Write the failing test**

```go
func TestNewPluginCommand(t *testing.T) {}
func TestNewListSubcommand(t *testing.T) {}
func TestNewPicoclawCommand_IncludesPluginCommand(t *testing.T) {}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/picoclaw/internal/plugin ./cmd/picoclaw -run 'TestNewPluginCommand|TestNewListSubcommand|TestNewPicoclawCommand' -v`  
Expected: FAIL with missing package/command registration.

**Step 3: Write minimal implementation**

```go
func NewPluginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Inspect and validate plugins",
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(newListCommand())
	return cmd
}
```

`newListCommand()` should load config, resolve selection, and print text or JSON list.

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/picoclaw/internal/plugin ./cmd/picoclaw -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add cmd/picoclaw/internal/plugin/command.go cmd/picoclaw/internal/plugin/list.go cmd/picoclaw/internal/plugin/command_test.go cmd/picoclaw/internal/plugin/list_test.go cmd/picoclaw/main.go cmd/picoclaw/main_test.go
git commit -m "feat(cli): add plugin list command"
```

### Task 9: Add `picoclaw plugin lint` Command (Phase 3)

**Files:**
- Create: `cmd/picoclaw/internal/plugin/lint.go`
- Test: `cmd/picoclaw/internal/plugin/lint_test.go`
- Modify: `cmd/picoclaw/internal/plugin/command.go`
- Modify: `cmd/picoclaw/internal/pluginruntime/bootstrap.go`
- Modify: `cmd/picoclaw/internal/pluginruntime/bootstrap_test.go`

**Step 1: Write the failing test**

```go
func TestPluginLint_ValidConfigExitZero(t *testing.T) {}
func TestPluginLint_UnknownEnabledExitNonZero(t *testing.T) {}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/picoclaw/internal/plugin ./cmd/picoclaw/internal/pluginruntime -run 'TestPluginLint_|TestResolveConfiguredPlugins_' -v`  
Expected: FAIL with missing lint command/validation path.

**Step 3: Write minimal implementation**

```go
func newLintCommand() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Validate plugin configuration",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.LoadConfig(configPath)
			if err != nil { return err }
			_, _, err = pluginruntime.ResolveConfiguredPlugins(cfg)
			return err
		},
	}
	cmd.Flags().StringVar(&configPath, "config", internal.GetConfigPath(), "Path to config.json")
	return cmd
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/picoclaw/internal/plugin ./cmd/picoclaw/internal/pluginruntime -v`  
Expected: PASS.

**Step 5: Commit**

```bash
git add cmd/picoclaw/internal/plugin/lint.go cmd/picoclaw/internal/plugin/lint_test.go cmd/picoclaw/internal/plugin/command.go cmd/picoclaw/internal/pluginruntime/bootstrap.go cmd/picoclaw/internal/pluginruntime/bootstrap_test.go
git commit -m "feat(cli): add plugin lint command"
```

### Task 10: Documentation and Final Verification

**Files:**
- Modify: `docs/plugin-system-roadmap.md`
- Modify: `docs/plans/2026-02-28-plugin-system-phase2-phase3-design.md`
- Optional Modify: `README.md` (if command docs are surfaced there)

**Step 1: Write docs-oriented failing checks**

Add/update checklist assertions in docs PR description:
- Phase 2 gates explicitly checked.
- Phase 3 list/lint behavior and exit semantics documented.

**Step 2: Run verification commands**

Run:

```bash
go test ./pkg/config ./pkg/plugin ./pkg/plugin/builtin ./pkg/agent ./cmd/picoclaw ./cmd/picoclaw/internal/plugin ./cmd/picoclaw/internal/pluginruntime -v
```

Expected: PASS.

**Step 3: Minimal doc implementation**

Document:
- JSON `plugins` config examples
- deterministic precedence rules
- `plugin list` usage (`--format json`)
- `plugin lint --config` usage and non-zero behavior

**Step 4: Re-run verification**

Run:

```bash
go test ./pkg/config ./pkg/plugin ./pkg/plugin/builtin ./pkg/agent ./cmd/picoclaw ./cmd/picoclaw/internal/plugin ./cmd/picoclaw/internal/pluginruntime -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add docs/plugin-system-roadmap.md docs/plans/2026-02-28-plugin-system-phase2-phase3-design.md README.md
git commit -m "docs(plugin): document phase2/phase3 behavior and cli usage"
```

---

## PR Plan (maintainer-friendly)

1. PR-1: Tasks 1-2 (`config` + resolver).
2. PR-2: Tasks 3-6 (catalog + bootstrap + startup diagnostics).
3. PR-3: Tasks 7-9 (metadata + plugin list/lint CLI).
4. PR-4: Task 10 docs-only cleanup if needed.

Each PR should include:
- complete PR template fields
- AI disclosure
- test environment and command evidence
- unresolved comments = 0 before merge

