# `pkg/isolation`

`pkg/isolation` provides process-level isolation for child processes started by `picoclaw`.

It does not sandbox the main `picoclaw` process itself.

## Scope

The current scope is the child-process startup path:

- `exec` tool
- CLI providers such as `claude-cli` and `codex-cli`
- process hooks
- MCP `stdio` servers

## One-Sentence Model

- The `picoclaw` main process still runs in the host environment.
- Every child process should enter the shared `pkg/isolation` startup path first.
- The startup path applies platform-specific isolation according to config.

## Architecture

The implementation has four layers:

1. Configuration layer: reads `config.Config.Isolation` and injects it through `isolation.Configure(cfg)`.
2. Instance layout layer: resolves `config.GetHome()`, prepares instance directories, and builds the runtime user environment.
3. Platform backend layer: Linux uses `bwrap`; Windows uses a restricted token, low integrity, and a `Job Object`; other platforms are not implemented.
4. Unified startup layer: `PrepareCommand(cmd)`, `Start(cmd)`, and `Run(cmd)`.

All integrations that spawn subprocesses should reuse these helpers instead of calling `cmd.Start` or `cmd.Run` directly.

## Configuration

Isolation lives under:

```json
{
  "isolation": {
    "enabled": false,
    "expose_paths": []
  }
}
```

Field meanings:

- `enabled`: enables or disables subprocess isolation. Default: `false`.
- `expose_paths`: explicitly exposes host paths inside the isolated environment. It only matters when `enabled=true`. This is currently supported on Linux only.

Example:

```json
{
  "isolation": {
    "enabled": true,
    "expose_paths": [
      {
        "source": "/opt/toolchains/go",
        "target": "/opt/toolchains/go",
        "mode": "ro"
      },
      {
        "source": "/data/shared-assets",
        "target": "/opt/picoclaw-instance-a/workspace/assets",
        "mode": "rw"
      }
    ]
  }
}
```

Rules for `expose_paths`:

- `source` is a host path.
- `target` is the path inside the isolated environment.
- `mode` must be `ro` or `rw`.
- When `target` is empty, it defaults to `source`.
- Only one final rule may exist for the same `target`.
- Later-loaded config overrides earlier rules for the same `target`.

Platform note:

- Linux uses a real `source -> target` mount view.
- Windows does not currently support `expose_paths`.

## Instance Root And Directories

The instance root follows `config.GetHome()`:

- If `PICOCLAW_HOME` is set, use it.
- Otherwise use the default `.picoclaw` directory under the user home.

If `config.GetHome()` falls back to `.` while isolation is enabled, startup should fail.

Default instance directories include:

- instance root
- `skills`
- `logs`
- `cache`
- `state`
- `runtime-user-env`

`workspace` is derived from `cfg.WorkspacePath()` when configured, otherwise from the default workspace rule.

Windows also prepares:

- `runtime-user-env/AppData/Roaming`
- `runtime-user-env/AppData/Local`

## User Environment Redirect

When isolation is enabled, child processes receive a redirected per-instance user environment.

Linux variables:

- `HOME`
- `TMPDIR`
- `XDG_CONFIG_HOME`
- `XDG_CACHE_HOME`
- `XDG_STATE_HOME`

Windows variables:

- `USERPROFILE`
- `HOME`
- `TEMP`
- `TMP`
- `APPDATA`
- `LOCALAPPDATA`

These paths point into `runtime-user-env` under the instance root.

## Platform Behavior

### Linux

The Linux backend currently depends on `bwrap` (`bubblewrap`).

Capabilities:

- minimal filesystem view
- `ipc` namespace isolation
- redirected child-process user environment
- `source -> target` read-only or read-write mounts

Default mounts include the instance root plus the minimum runtime system paths such as `/usr`, `/bin`, `/lib`, `/lib64`, and `/etc/resolv.conf`.

At runtime, PicoClaw also adds the executable path, its directory, the effective working directory, and absolute path arguments when needed.

There is no automatic fallback when `bwrap` is missing.

Install examples:

- `apt install bubblewrap`
- `dnf install bubblewrap`
- `yum install bubblewrap`
- `pacman -S bubblewrap`
- `apk add bubblewrap`

If isolation must be disabled temporarily:

```json
{
  "isolation": {
    "enabled": false
  }
}
```

Disabling isolation increases the risk that child processes can access or modify more host files.

### Windows

Windows isolation currently supports process-level restrictions such as restricted tokens, low integrity, job objects, and redirected user-environment directories.

`expose_paths` is not currently supported on Windows. If it is configured, startup should fail instead of pretending the paths were exposed.

The Windows backend currently uses:

- a restricted primary token
- low integrity level
- a `Job Object`
- redirected child-process user environment

It does not currently implement true `source -> target` filesystem remapping.

### macOS And Other Platforms

They are not implemented yet.

When isolation is explicitly enabled on an unsupported platform, the higher-level runtime should surface that as an unsupported configuration instead of pretending isolation succeeded.

## Logging And Debugging

When isolation is enabled, PicoClaw logs the generated isolation plan.

Linux log name:

- `linux isolation mount plan`

Windows log name:

- `windows isolation access rules`

If you suspect isolation is ineffective, check whether unexpected host paths appear in those logs.

## Relationship To `restrict_to_workspace`

- `restrict_to_workspace` limits the paths an agent is normally allowed to access.
- `pkg/isolation` limits what a child process can see and where its user environment points.

They complement each other and do not replace each other.

## Current Limits

- Linux isolation is implemented with `bwrap`, not a custom in-process isolation runtime.
- Linux does not currently enable a dedicated `pid` namespace by default.
- Windows does not yet implement full host ACL enforcement for every allowed or denied path.
- macOS is not implemented.
- The current design isolates child processes, not the main `picoclaw` process.

## Suggested Reading Order

If you are new to this code, read it in this order:

1. `pkg/config/config.go`
2. `pkg/isolation/runtime.go`
3. `pkg/isolation/platform_linux.go`
4. `pkg/isolation/platform_windows.go`
5. Call sites:
6. `pkg/tools/shell.go`
7. `pkg/providers/*.go`
8. `pkg/agent/hook_process.go`
9. `pkg/mcp/manager.go`

That path gives the fastest overview of the configuration model, runtime flow, and platform-specific limits.
