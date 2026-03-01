# PicoClaw Launcher

> [!WARNING]
> This project is a temporary solution and will be refactored in the future to provide a complete web service. Therefore, the APIs in this directory are not stable.

A standalone launcher for PicoClaw, providing visual JSON editing and OAuth provider authentication management.

## Features

- 📝 **Config Editor** — Sidebar-based settings UI with model management, channel configuration forms, and a raw JSON editor
- 🤖 **Model Management** — Model card grid with availability status (grayed out without API key), primary model selection, add/edit/delete with required/optional field separation
- 📡 **Channel Configuration** — Form-based settings for 12 channel types (Telegram, Discord, Slack, WeCom, DingTalk, Feishu, LINE, WhatsApp, QQ, OneBot, MaixCAM, etc.) with documentation links
- 🔐 **Provider Auth** — Login to OpenAI (Device Code), Anthropic (API Token), Google Antigravity (Browser OAuth)
- 🌐 **Embedded Frontend** — Compiles to a single binary with no external dependencies
- 🌍 **i18n** — Chinese/English language switching with browser auto-detection
- 🎨 **Theme** — Light / Dark / System theme toggle with localStorage persistence

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

Open `http://localhost:18800` in your browser.

## CLI Options

```
Usage: picoclaw-config [options] [config.json]

Arguments:
  config.json    Path to the configuration file (default: ~/.picoclaw/config.json)

Options:
  -public        Listen on all interfaces (0.0.0.0), allowing access from other devices
```

## API Reference

Base URL: `http://localhost:18800`

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
  "path": "/Users/xiao/.picoclaw/config.json"
}
```

---

#### PUT /api/config

Saves the configuration. The request body must be a complete Config JSON object.

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

`pending_device` is only present when a device code login is in progress.

---

#### POST /api/auth/login

Initiates a provider login.

**Request Body** — `application/json`

```json
{ "provider": "openai" }
```

Supported `provider` values: `openai` | `anthropic` | `google-antigravity`

##### OpenAI (Device Code Flow)

Returns device code info. The server polls for completion in the background.

```json
{
  "status": "pending",
  "device_url": "https://auth.openai.com/activate",
  "user_code": "ABCD-1234",
  "message": "Open the URL and enter the code to authenticate."
}
```

The user opens `device_url` in a browser and enters `user_code`. Once authenticated, `GET /api/auth/status` will show `pending_device.status` as `success`.

##### Anthropic (API Token)

Requires a `token` field in the request:

```json
{ "provider": "anthropic", "token": "sk-ant-xxx" }
```

**Response:**

```json
{ "status": "success", "message": "Anthropic token saved" }
```

##### Google Antigravity (Browser OAuth)

Returns an authorization URL for the frontend to open in a new tab:

```json
{
  "status": "redirect",
  "auth_url": "https://accounts.google.com/o/oauth2/auth?...",
  "message": "Open the URL to authenticate with Google."
}
```

After authentication, Google redirects to `GET /auth/callback`, which saves the credentials and redirects back to the picoclaw-config UI.

---

#### POST /api/auth/logout

Logs out from a provider.

**Request Body** — `application/json`

```json
{ "provider": "openai" }
```

Omit or leave `provider` empty to log out from all providers.

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

On success, redirects to `/#auth`.


### Process API

#### GET /api/process/status

Gets the running status of the `picoclaw gateway` process.

**Response** `200 OK` (Running)

```json
{
  "process_status": "running",
  "status": "ok",
  "uptime": "1.010814s"
}
```

**Response** `200 OK` (Stopped)

```json
{
  "process_status": "stopped",
  "error": "Get \"http://localhost:18790/health\": dial tcp [::1]:18790: connect: connection refused"
}
```

---

#### POST /api/process/start

Starts the `picoclaw gateway` process in the background.

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

**Response** `200 OK`

```json
{
  "status": "ok"
}
```

---

## Testing

```bash
go test -v ./cmd/picoclaw-launcher/
```
