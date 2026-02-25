# PicoClaw Service (Background Gateway)

The `picoclaw service` command manages the PicoClaw gateway as a **background service** using the system’s native service manager. This keeps the gateway running after you close the terminal and across reboots (when enabled).

## When to Use Service vs Gateway

| Mode | Command | Use case |
|------|---------|----------|
| **Foreground** | `picoclaw gateway` | Quick run in a terminal; stops when you press Ctrl+C. |
| **Background service** | `picoclaw service install` then `picoclaw service start` | Long‑running gateway; survives logout and can start on boot. |

If you run `picoclaw gateway` while the service is already running, the CLI will warn you and exit to avoid two gateways (and port conflicts).

---

## Supported Platforms

| Platform | Backend | Notes |
|----------|---------|--------|
| **macOS** | launchd | Per-user agent: `~/Library/LaunchAgents/io.picoclaw.gateway.plist`. |
| **Linux** | systemd user | Per-user unit: `~/.config/systemd/user/picoclaw-gateway.service`. Requires `systemctl --user` (user session). |
| **WSL** | systemd user or unsupported | If systemd is enabled in WSL, service works. Otherwise use `picoclaw gateway` in a terminal. |
| **Windows / other** | Unsupported | Use `picoclaw gateway` in a terminal or another process manager. |

---

## Commands Overview

```text
picoclaw service install      # Register the gateway with launchd (macOS) or systemd (Linux)
picoclaw service uninstall    # Remove the service registration
picoclaw service start        # Start the background gateway
picoclaw service stop         # Stop the background gateway
picoclaw service restart      # Stop then start (e.g. after config change)
picoclaw service status       # Show whether the service is installed and running
picoclaw service logs         # Show recent gateway logs (options: -n, -f)
picoclaw service refresh      # Reinstall unit/plist and restart (e.g. after upgrading the binary)
```

---

## Subcommands in Detail

### install

Registers the PicoClaw gateway as a background service. The executable path is resolved from how you invoked `picoclaw` (e.g. `/opt/homebrew/bin/picoclaw` or `~/go/bin/picoclaw`).

- **macOS**: Creates a launchd plist and runs `launchctl bootstrap`.
- **Linux**: Creates a systemd user unit and runs `systemctl --user daemon-reload` and `systemctl --user enable`.

The service is configured to run `picoclaw gateway` with a PATH that includes common system paths and, if present, `~/.picoclaw/workspace/.venv/bin`. **`config.json` is not modified** by install.

**Example**

```bash
picoclaw service install
# ✓ Service installed
#   Start with: picoclaw service start
```

---

### uninstall

Removes the service registration and stops the gateway if it is running.

- **macOS**: `launchctl bootout` and deletes the plist.
- **Linux**: `systemctl --user disable --now` and deletes the unit file.

**Example**

```bash
picoclaw service uninstall
# ✓ Service uninstalled
```

---

### start

Starts the background gateway. Fails with a clear message if the service is not installed (run `picoclaw service install` first).

**Example**

```bash
picoclaw service start
# ✓ Service started
```

---

### stop

Stops the background gateway. No error if it was already stopped.

**Example**

```bash
picoclaw service stop
# ✓ Service stopped
```

---

### restart

Stops then starts the gateway. Useful after editing `config.json` or upgrading the binary (or use `picoclaw service refresh` to also reinstall the unit/plist).

**Example**

```bash
picoclaw service restart
# ✓ Service restarted
```

---

### status

Shows whether the service is installed, running, and (on Linux) enabled. Also shows the backend (e.g. `launchd` or `systemd-user`) and optional detail (e.g. “installed but not loaded”).

**Example**

```bash
picoclaw service status
```

**Example output**

```text
Gateway service status:
  Backend:   launchd
  Installed: yes
  Running:   yes
  Enabled:   yes
```

---

### logs

Prints recent gateway logs. On macOS these are the launchd stdout/stderr files; on Linux they come from `journalctl --user -u picoclaw-gateway.service`.

**Options**

| Option | Description |
|--------|-------------|
| `-n`, `--lines <N>` | Number of lines to show (default: 100). |
| `-f`, `--follow` | Follow log output (like `tail -f`). Press Ctrl+C to stop. |

**Examples**

```bash
picoclaw service logs
picoclaw service logs -n 200
picoclaw service logs --lines 50
picoclaw service logs -f
picoclaw service logs -f -n 20
```

---

### refresh

Reinstalls the service definition (plist or unit) and restarts the gateway. Use after upgrading the `picoclaw` binary so the service runs the new version and keeps PATH/env in sync.

**Example**

```bash
picoclaw service refresh
# ✓ Service refreshed
#   Reinstalled and restarted (run: picoclaw service status)
```

---

## File Locations

### macOS (launchd)

| Item | Path |
|------|------|
| Plist | `~/Library/LaunchAgents/io.picoclaw.gateway.plist` |
| Stdout log | `~/.picoclaw/gateway.log` |
| Stderr log | `~/.picoclaw/gateway.err.log` |

### Linux (systemd user)

| Item | Path |
|------|------|
| Unit file | `~/.config/systemd/user/picoclaw-gateway.service` |
| Logs | `journalctl --user -u picoclaw-gateway.service` (see `picoclaw service logs`) |

---

## Environment and PATH

- **install** uses the current process `PATH` (and, if present, `~/.picoclaw/workspace/.venv/bin`) when generating the plist or unit so the background process has a sane PATH.
- **config.json** is **not** updated by `service install`; the gateway still reads `~/.picoclaw/config.json` at runtime.

---

## Troubleshooting

### "Gateway is already running via launchd/service"

You started `picoclaw gateway` while the background service is already running. Either:

- Use the service: `picoclaw service stop` / `picoclaw service restart` / `picoclaw service logs`, or  
- Stop the service first: `picoclaw service stop`, then run `picoclaw gateway` if you want foreground only.

### "service is not installed; run \`picoclaw service install\`"

You ran `picoclaw service start` (or `restart`) before installing. Run:

```bash
picoclaw service install
picoclaw service start
```

### "WSL detected but systemd user manager is not active"

On WSL, enable systemd (e.g. in `/etc/wsl.conf`) or run the gateway in the foreground:

```bash
picoclaw gateway
```

### "directory does not exist" or "no such file or directory" in logs

Those messages come from the gateway (e.g. tools or config). They are not caused by the service command itself. Check `config.json` and paths (e.g. workspace) used by the agent.

### Logs are empty or "no launchd logs found"

- **macOS**: Ensure the service has been started at least once (`picoclaw service start`). Logs are written to `~/.picoclaw/gateway.log` and `~/.picoclaw/gateway.err.log`.
- **Linux**: Use `picoclaw service logs` (or `journalctl --user -u picoclaw-gateway.service`) after the service has run.

---

## Quick Reference

```bash
# First-time: install and start
picoclaw service install && picoclaw service start

# Daily use
picoclaw service status    # check it's running
picoclaw service logs -f   # follow logs
picoclaw service restart   # after editing config

# Upgrade binary then refresh service
picoclaw service refresh

# Remove service
picoclaw service stop
picoclaw service uninstall
```
