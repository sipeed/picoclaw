---
name: self-debug
description: Tools for picoclaw to inspect its own health, logs, and configuration.
---
# self-debug

Tools for picoclaw to inspect its own health, logs, and configuration.

## Actions via `debug.sh`

This skill uses a helper script `scripts/debug.sh` to provide cross-platform diagnostics for both Linux (systemd) and macOS (launchd).

**Usage:** `exec "skills/self-debug/scripts/debug.sh [action] [service_name] [log_lines]"`

| Action | Description | Default Service | Default Lines |
|---|---|---|---|
| `logs` | Fetches recent logs for the agent service. | `$PICOCLAW_SERVICE_NAME` | 50 |
| `logs-errors`| Fetches only error logs for the agent service. | `$PICOCLAW_SERVICE_NAME` | 50 |
| `service-status`| Checks the status of the agent service. | `$PICOCLAW_SERVICE_NAME` | N/A |
| `config-status`| Shows the agent's configuration status (`picoclaw status`).| N/A | N/A |
| `config-safe` | Displays the config file with sensitive keys redacted. | N/A | N/A |

### Examples

- **Get latest logs:**
  `exec "skills/self-debug/scripts/debug.sh logs"`

- **Get 100 lines of logs for a specific service 'pico-prod':**
  `exec "skills/self-debug/scripts/debug.sh logs pico-prod 100"`

- **Check service status:**
  `exec "skills/self-debug/scripts/debug.sh service-status"`

- **Show the sanitized configuration:**
  `exec "skills/self-debug/scripts/debug.sh config-safe"`

## Installation - Linux

The agent can be installed as a systemd service using:

```bash
./scripts/install-service-systemd.sh [service_name] [default|multi]
```

To persist the service accross reboots suggest the user runs `sudo loginctl enable-linger $(whoami)`

### Advanced use-cases

Although picoclaw has built in support for multiple agents, this scheme provides the flexibility for
parallel deployments with entirely different configurations.

- Fixer - a second stable instance whose role is to be available to debug/monitor/fix the first.
- Blue/Green Stable/Canary pairings.
- Picoclaw farm

# Installation - MacOS

The agent can be installed as a launchd service using:

```bash
./scripts/install-service-launchd.sh [service_name]
```

Logs are sent to /tmp/$service_name

## Troubleshooting

- **Logs not showing?** Ensure the user is in the `systemd-journal` group: `sudo usermod -a -G systemd-journal $(whoami)`
- **Service inactive?** Use `systemctl --user start picoclaw`.
- **Workspace issues?** Use `picoclaw status` to verify the current workspace path.
