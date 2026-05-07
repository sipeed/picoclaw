# MagicForm Customizations

This file is the index of customizations the `magicform` fork carries on top
of `sipeed/picoclaw` upstream. Read this first when:
- Doing an upstream sync (start by replaying each customization against the
  new layout if upstream has changed the surrounding code)
- Reviewing a PR that touches any of the listed files
- Onboarding a new engineer to the fork

The single source of truth for *what changed* is the git log
(`git log upstream/main..main`). This file tells you *why* and *where*.

---

## Goals of the fork

1. **Multi-tenant isolation.** A single `picoclaw` process serves many
   tenants. Each tenant has its own filesystem workspace, config, sessions,
   provider credentials, and tool/skill allowlist. Inbound messages carry
   tenant hints in `bus.InboundContext.Raw`; the agent loop validates them
   against a security boundary and applies them per-turn.
2. **Defense-in-depth on workspace boundary.** Every path-manipulating tool
   (`fs`, `exec`, skill installer) honours `agents.defaults.workspace_root`
   as a containment root. Tenants cannot read or write outside it.
3. **MagicForm webhook channel.** A webhook-driven channel
   (`pkg/channels/magicform`) accepts inbound messages from MagicForm and
   posts agent responses back via callback URL. Lives alongside upstream's
   stock channels.
4. **Bounded resource use.** Search APIs and write tools have explicit byte
   limits to keep a hostile or buggy upstream from exhausting memory/disk.

---

## Subsystem map (read top-down when syncing)

Each entry: subsystem → files touched → most recent commit on `main`.
When upstream restructures a subsystem, only the matching entry needs to be
forward-ported.

### 1. MagicForm webhook channel
- **Owns:** `pkg/channels/magicform/{magicform.go,init.go}`
- **Registers in:** `pkg/gateway/gateway.go` (blank import)
- **Config plumbing:** `MagicFormSettings` in `pkg/config/config.go`,
  `ChannelMagicForm` constant + `channelSettingsFactory` entry in
  `pkg/config/config_channel.go`, validation case in
  `pkg/channels/manager.go::getChannelConfigAndEnabled`.
- **Protocol:** HTTP POST to `/hooks/magicform` with bearer token; outbound
  is HTTP POST callback to a per-request URL with JSON payload.
- **Tenancy hints:** the webhook handler stuffs `workspace_override`,
  `config_dir`, `allowed_tools`, `allowed_skills` (and `callback_url`,
  `stack_id`, `conversation_id`) into `bus.InboundContext.Raw`; the agent
  loop reads them in `agent_tenant.go`.

### 2. Multi-tenancy in the agent loop
- **Owns:** `pkg/agent/agent_tenant.go` (entirely fork-owned, isolated for
  sync friendliness) and a small wire-up block in
  `pkg/agent/agent_message.go::processMessage`.
- **Surface on processOptions:** `WorkspaceOverride`, `ConfigDir`,
  `AllowedTools`, `AllowedSkills` (defined in `pkg/agent/agent.go`).
- **Status: Phase 1.** Hints are validated and plumbed onto
  `processOptions`. Phase 2 (effSessions, effContextBuilder, effProvider,
  effModel threaded through `pipeline_llm.go`, `turn_state.go`,
  `context_manager.go`) is a separate PR.
- **Security boundary:** validation uses
  `pathutil.ResolveWorkspacePath(agents.defaults.workspace_root, hint)`;
  fails closed when `workspace_root` is unset.

### 3. Workspace path security utility
- **Owns:** `pkg/pathutil/{resolve.go,resolve_test.go}` (fork-owned).
- Used by: `agent_tenant.go`, `pkg/config/config.go::mergeAgentDefaults`,
  `cmd/picoclaw/internal/agent/helpers.go::validateWorkspacePaths`,
  channels that accept tenant paths.

### 4. CLI overrides and workspace config overlay
- **Owns:** `cmd/picoclaw/internal/agent/{helpers.go,helpers_test.go}` —
  validates `--workspace` / `--config-dir` flags, loads
  `<config-dir>/config.json` and merges over the base config via
  `Config.MergeWorkspaceConfig`.
- **Owns:** `pkg/config/config.go::MergeWorkspaceConfig` and
  `mergeAgentDefaults` (fork additions; not in upstream).

### 5. Tool hardening (filesystem)
- **Owns:** customizations in `pkg/tools/fs/filesystem.go::sandboxFs.WriteFile`:
  - `MaxWriteFileSize` cap (20 MB) before opening any file.
  - `crypto/rand` temp suffixes instead of `time.Now().UnixNano()`.
- Last forward-ported: commit `cd1720f4` (after upstream `4c133dc2`
  reorganized `pkg/tools/`).

### 6. Tool hardening (web search)
- **Owns:** customizations in `pkg/tools/integration/web.go`:
  - `searchMaxResponseSize` constant (2 MB) used by every search provider's
    `io.ReadAll(io.LimitReader(...))`.
- Last forward-ported: commit `94d28c1b`.

### 7. Output-channel plumbing for tenancy callbacks
- **Owns:** `pkg/bus/types.go` additions on `OutboundMessage`: `Type`,
  `Metrics`, `Progress`, `Escalation`. Plus types `ResponseMetrics`,
  `OutboundProgress`, `OutboundEscalation`, `TokenUsage`.
- The MagicForm channel `Send` reads these to compose its callback payload.
  Other channels ignore them.

### 8. Exec tool: filterEnv
- **Owns:** `filterEnv` field in `ExecTool` and `ExecConfig`. Strips
  non-`PICOCLAW_*`-prefixed env vars before child processes.
- Files: `pkg/tools/shell.go`, `pkg/config/config.go::ExecConfig`.

### 9. Channel base hook
- **Owns:** `BaseChannel.Bus()` accessor in `pkg/channels/base.go`. Used
  by the magicform channel to publish directly with a non-default SessionKey.

---

## Sync playbook

When pulling a new upstream:

1. `git fetch upstream && git fetch origin`
2. Branch: `git checkout -b sync/upstream-YYYY-MM-DD origin/main`
3. `git merge upstream/main` and resolve conflicts.
4. For each subsystem above, replay any commits whose files now no longer
   exist (modify/delete conflicts) onto upstream's new locations.
5. `go build ./... && go test ./...`
6. Open a PR against `main`. Each forward-port should be its own commit
   prefixed `forward-port:` so the merge commit and customization commits
   are visually distinct in `git log`.
7. Update this index if the file map shifts.

---

## What is *not* customized any more

- **Launcher (`cmd/picoclaw-launcher/`)**: dropped during the
  2026-05-07 sync. Half its dependencies were deleted upstream and
  MagicForm doesn't use the launcher's HTTP API. Upstream's
  `web/backend/` is the replacement if a web admin is ever needed.
- **Deprecated `AgentDefaults.Model`**: dropped during the 2026-05-07
  sync. Use `model_name`. Workspace overlays that still set the old
  `"model"` JSON key will be silently ignored — migrate them.
