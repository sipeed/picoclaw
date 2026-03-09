# PicoClaw Launcher

> [!WARNING]
> This project is a temporary solution and will be refactored in the future to provide a complete web service. Therefore, the APIs in this directory are not stable.

A standalone launcher for PicoClaw, providing visual JSON editing, OAuth provider authentication management, and gateway process control.

## Features

- 📝 **Config Editor** — Sidebar-based settings UI with model management, channel configuration forms, and a raw JSON editor
- 🤖 **Model Management** — Model card grid with availability status (grayed out without API key), primary model selection, add/edit/delete with required/optional field separation
- 📡 **Channel Configuration** — Form-based settings for 14+ channel types (Telegram, Discord, Slack, Matrix, WeCom, DingTalk, Feishu, LINE, WhatsApp, QQ, OneBot, MaixCAM, MagicForm, IRC, etc.) with documentation links
- 🔐 **Provider Auth** — Login to OpenAI (Device Code), Anthropic (API Token), Google Antigravity (Browser OAuth with PKCE)
- 🚀 **Gateway Process Control** — Start, stop, and monitor the `picoclaw gateway` process with live log streaming
- 🌐 **Embedded Frontend** — Compiles to a single binary with no external dependencies
- 🌍 **i18n** — Chinese/English language switching with browser auto-detection
- 🎨 **Theme** — Light / Dark / System theme toggle with localStorage persistence
- 🔒 **Security Headers** — `X-Content-Type-Options`, `X-Frame-Options`, and `Content-Security-Policy` on all responses

## Quick Start

```bash
# Build
go build -o picoclaw-launcher ./cmd/picoclaw-launcher/

# Run with default config path (~/.picoclaw/config.json)
./picoclaw-launcher

# Specify a config file
./picoclaw-launcher ./config.json

# Allow LAN access
./picoclaw-launcher -public
```

The launcher automatically opens `http://localhost:18800` in your default browser on startup.

## CLI Options

```
Usage: picoclaw-launcher [options] [config.json]

Arguments:
  config.json    Path to the configuration file (default: ~/.picoclaw/config.json)

Options:
  -public        Listen on all interfaces (0.0.0.0), allowing access from other devices
```

When `-public` is set, the startup banner also prints the local network IP address for LAN access.

## API Reference

Base URL: `http://localhost:18800`

Default port: `18800`

---

### Static Files

#### GET /

Serves the embedded frontend (`index.html`).

---

### Config API

#### GET /api/config

Reads the current configuration file.

**Response** `200 OK`

```json
{
  "config": { ... },
  "path": "/home/user/.picoclaw/config.json"
}
```

---

#### PUT /api/config

Saves the configuration. The request body must be a complete Config JSON object (max 1 MB).

**Request Body** — `application/json`

```json
{
  "agents": { "defaults": { "model_name": "gpt-5.2" } },
  "model_list": [
    {
      "model_name": "gpt-5.2",
      "model": "openai/gpt-5.2",
      "auth_method": "oauth"
    }
  ]
}
```

**Response** `200 OK`

```json
{ "status": "ok" }
```

**Error** `400 Bad Request` — Invalid JSON

---

### Auth API

#### GET /api/auth/status

Returns the authentication status of all providers and any in-progress device code login.

**Response** `200 OK`

```json
{
  "providers": [
    {
      "provider": "openai",
      "auth_method": "oauth",
      "status": "active",
      "account_id": "user-xxx",
      "expires_at": "2026-03-01T00:00:00Z"
    },
    {
      "provider": "google-antigravity",
      "auth_method": "oauth",
      "status": "active",
      "email": "user@example.com",
      "project_id": "projects/123/locations/global/codeAssistModels/default"
    }
  ],
  "pending_device": {
    "provider": "openai",
    "status": "pending",
    "device_url": "https://auth.openai.com/activate",
    "user_code": "ABCD-1234"
  }
}
```

`status` values: `active` | `expired` | `needs_refresh`

`pending_device` is only present when a device code login is in progress. Once completed, it shows `status: "success"` and is cleared on the next poll.

---

#### POST /api/auth/login

Initiates a provider login.

**Request Body** — `application/json`

```json
{ "provider": "openai" }
```

Supported `provider` values: `openai` | `anthropic` | `google-antigravity` (alias: `antigravity`)

##### OpenAI (Device Code Flow)

Returns device code info. The server polls for completion in the background (15-minute timeout).

```json
{
  "status": "pending",
  "device_url": "https://auth.openai.com/activate",
  "user_code": "ABCD-1234",
  "message": "Open the URL and enter the code to authenticate."
}
```

The user opens `device_url` in a browser and enters `user_code`. Once authenticated, `GET /api/auth/status` will show `pending_device.status` as `success`. If a device code flow is already in progress, the existing session is returned.

##### Anthropic (API Token)

Requires a `token` field in the request:

```json
{ "provider": "anthropic", "token": "sk-ant-xxx" }
```

**Response:**

```json
{ "status": "success", "message": "Anthropic token saved" }
```

The token is saved to the auth credential store and the config is updated to set `auth_method: "token"` on any Anthropic model entry.

##### Google Antigravity (Browser OAuth with PKCE)

Returns an authorization URL for the frontend to open in a new tab:

```json
{
  "status": "redirect",
  "auth_url": "https://accounts.google.com/o/oauth2/auth?...",
  "message": "Open the URL to authenticate with Google."
}
```

After authentication, Google redirects to `GET /auth/callback`, which exchanges the authorization code for tokens using PKCE, fetches the user's email and Cloud Code Assist project ID, saves the credentials, and redirects back to the launcher UI at `/#auth`. OAuth sessions expire after 10 minutes if not completed.

---

#### POST /api/auth/logout

Logs out from a provider.

**Request Body** — `application/json`

```json
{ "provider": "openai" }
```

Omit or leave `provider` empty to log out from all providers. Clears both the auth credential store and `auth_method` fields in the config file.

**Response** `200 OK`

```json
{ "status": "ok" }
```

---

#### GET /auth/callback

OAuth browser callback endpoint (used by Google Antigravity). Called by the OAuth provider's redirect — **not invoked directly by the frontend**.

**Query Parameters:**
- `state` — OAuth state for CSRF validation
- `code` — Authorization code

On success, redirects to `/#auth`. On failure, displays an error page.

---

### Process API

#### GET /api/process/status

Gets the running status of the `picoclaw gateway` process by probing its health endpoint.

The gateway address is read from the config file (`gateway.host` and `gateway.port`, default `127.0.0.1:18790`).

**Query Parameters** (optional, for incremental log streaming):
- `log_offset` — Last received log line index (0-based)
- `log_run_id` — Run ID from previous response (detects gateway restarts)

**Response** `200 OK` (Running)

```json
{
  "process_status": "running",
  "status": "ok",
  "uptime": "1.010814s",
  "logs": ["[INFO] Gateway started on :18790", "..."],
  "log_total": 42,
  "log_run_id": 1,
  "log_source": "launcher"
}
```

**Response** `200 OK` (Stopped)

```json
{
  "process_status": "stopped",
  "error": "Get \"http://localhost:18790/health\": dial tcp [::1]:18790: connect: connection refused",
  "logs": [],
  "log_total": 0,
  "log_run_id": 0,
  "log_source": "none"
}
```

`log_source` values: `launcher` (logs captured from a process started by the launcher) | `none` (no log source available, e.g. gateway started externally or never launched)

---

#### POST /api/process/start

Starts the `picoclaw gateway` process in the background. The launcher looks for the `picoclaw` binary first in the same directory as itself, then falls back to `$PATH`.

Stdout and stderr from the gateway process are captured into a ring buffer (200 lines) and can be streamed via `GET /api/process/status`.

**Response** `200 OK`

```json
{
  "status": "ok",
  "pid": 12345
}
```

---

#### POST /api/process/stop

Stops the running `picoclaw gateway` process.

On Linux/macOS, uses `pkill -f "picoclaw gateway"`. On Windows, uses PowerShell to find and stop matching processes.

**Response** `200 OK`

```json
{
  "status": "ok"
}
```

---

## Testing

```bash
go test -v ./cmd/picoclaw-launcher/...
```
