# MagicForm Integration Spec

MagicForm delegates agentic tasks to PicoClaw via webhooks. PicoClaw processes the request asynchronously and POSTs the result back to a callback URL.

> For general PicoClaw installation, CLI usage, and config reference, see [cli.md](cli.md).

---

## Pre-requisites

1. **Install PicoClaw** — see [cli.md § Install](cli.md#install)
2. **Global config** — see [cli.md § Global Config](cli.md#global-config)

### Directory layout

MagicForm pre-provisions directories on disk before calling PicoClaw (see [cli.md § Directory Layout](cli.md#directory-layout) for the general structure):

```
{workspace_root}/
  {stackId}/
    config/                    # configDir -- shared per-stack
      config.json              # API key, model, agent settings for this stack
      AGENTS.md                # Agent instructions (optional)
      IDENTITY.md              # Agent identity (optional)
      SOUL.md                  # Agent personality (optional)
      USER.md                  # User context (optional)
    {conversationId}/          # workspace -- per-conversation
      sessions/                # Conversation history (managed by PicoClaw)
      memory/                  # Persistent agent memory (managed by PicoClaw)
      skills/                  # Workspace-local skills (optional)
```

### Workspace config

Per-stack config overlays are placed in the config directory. See [cli.md § Workspace Config](cli.md#workspace-config) for merge rules and example.

---

## Gateway Mode

### Channel config

Add to `~/.picoclaw/config.json`:

```jsonc
{
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790
  },
  "channels": {
    "magicform": {
      "enabled": true,
      "token": "your-shared-secret",
      "backend_url": "https://api.magicform.example.com",
      "webhook_path": "/hooks/magicform",
      "workspace_root": "/data/workspaces",
      "allow_from": []
    }
  }
}
```

| Field | Description | Default |
|-------|-------------|---------|
| `token` | Bearer token for webhook auth. Empty = allow all (dev only). | `""` |
| `backend_url` | Fallback callback URL base (used when payload omits `callbackUrl`). | `""` |
| `webhook_path` | HTTP path for the webhook endpoint. | `/hooks/magicform` |
| `workspace_root` | Root directory for workspace/configDir path validation. Required for path security. | `""` |
| `allow_from` | Sender ID allowlist. Empty = allow all. Accepts strings and numbers (e.g. `["user1", 12345]`). | `[]` |

All fields can be set via environment variables:

```bash
PICOCLAW_CHANNELS_MAGICFORM_ENABLED=true
PICOCLAW_CHANNELS_MAGICFORM_TOKEN=your-shared-secret
PICOCLAW_CHANNELS_MAGICFORM_BACKEND_URL=https://api.magicform.example.com
PICOCLAW_CHANNELS_MAGICFORM_WEBHOOK_PATH=/hooks/magicform
PICOCLAW_CHANNELS_MAGICFORM_WORKSPACE_ROOT=/data/workspaces
PICOCLAW_CHANNELS_MAGICFORM_ALLOW_FROM=sender1,sender2
```

### Start the gateway

```bash
picoclaw gateway
# or with debug logging:
picoclaw gateway -d
```

Listens on `{host}:{port}` (default `127.0.0.1:18790`).

### Health check

```
GET /health/magicform
```

Response:

```json
{"status": "ok", "channel": "magicform"}
```

### Webhook: send a message

```
POST /hooks/magicform
Authorization: Bearer your-shared-secret
Content-Type: application/json
```

#### Request body

```json
{
  "stackId": "s1",
  "conversationId": "c1",
  "userId": "user-123",
  "message": "Summarize the latest sales report",
  "workspace": "s1/c1",
  "configDir": "s1/config",
  "callbackUrl": "https://api.magicform.example.com/claw-agent/callback",
  "allowedTools": ["read_file", "web_fetch"],
  "allowedSkills": ["summarize"]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `stackId` | string | Yes | Tenant/stack identifier. |
| `conversationId` | string | Yes | Conversation identifier. |
| `userId` | string | No | Sender identifier (defaults to `"anonymous"`). |
| `message` | string | Yes | The user's message. |
| `workspace` | string | No | Agent working directory, relative to `workspace_root`. |
| `configDir` | string | No | Config directory, relative to `workspace_root`. Contains `config.json` and bootstrap files. |
| `callbackUrl` | string | No | Where to POST the response. Falls back to `backend_url + "/claw-agent/callback"`. |
| `allowedTools` | string[] | No | Tool allowlist. Empty = all tools enabled. See [cli.md § Tool Names](cli.md#tool-names-reference). |
| `allowedSkills` | string[] | No | Skill filter. Empty = all skills loaded. |

**Path security**: `workspace` and `configDir` must be relative paths that resolve under `workspace_root`. Traversal attempts (e.g. `../../etc`) are rejected with `400 Bad Request`.

**Request size limit**: Webhook payloads are limited to 1 MB. Larger requests receive `413 Request Entity Too Large`.

**Method**: Only `POST` is accepted. Other methods receive `405 Method Not Allowed`.

#### Response

Returns `200 OK` immediately. Processing happens asynchronously.

### Callback: receive the result

PicoClaw POSTs the result to the callback URL:

```
POST {callbackUrl}
Authorization: Bearer your-shared-secret
Content-Type: application/json
```

```json
{
  "stackId": "s1",
  "conversationId": "c1",
  "response": "Here is the summary of the latest sales report...",
  "type": "final"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `stackId` | string | Echoed from request. |
| `conversationId` | string | Echoed from request. |
| `response` | string | The agent's response text. |
| `type` | string | Always `"final"`. |

### Session isolation

Each request gets a unique session key: `agent:main:magicform:{stackId}:{conversationId}`.

Sessions are stored at `{workspace}/sessions/`. Different conversations within the same stack share the config directory (API keys, bootstrap files) but have separate workspace directories, sessions, and memory.

### Processing flow

```
MagicForm                          PicoClaw Gateway
    |                                     |
    |-- POST /hooks/magicform ----------->|
    |<------------ 200 OK ---------------|
    |                                     |
    |                              resolveWorkspace(workspace)
    |                              resolveWorkspace(configDir)
    |                              publish InboundMessage to bus
    |                                     |
    |                              Agent loop:
    |                                create temp sessions + context
    |                                copyBootstrapFiles(configDir -> workspace)
    |                                loadWorkspaceConfig(configDir)
    |                                mergeWorkspaceConfig into cloned global cfg
    |                                createProvider (per-request)
    |                                apply tool/skill filters
    |                                run LLM iterations
    |                                     |
    |<-- POST callbackUrl (result) -------|
```

---

## Testing with CLI

You can test the same workspace/config setup without the gateway using `picoclaw agent`. See [cli.md § picoclaw agent](cli.md#picoclaw-agent) for full flag reference.

```bash
# One-shot with tenant isolation (same paths MagicForm would use)
picoclaw agent -m "Summarize the report" \
  -s s1:c1 \
  --workspace /data/workspaces/s1/c1 \
  --config-dir /data/workspaces/s1/config

# Restricted tools, matching a webhook allowedTools filter
picoclaw agent -m "Search the web for recent news" \
  --tools web,web_fetch

# Debug mode to see session key, model, and iteration details
picoclaw agent -d -m "Hello" -s test
```

---

## Troubleshooting

**Webhook returns 405 Method Not Allowed**
- The endpoint only accepts `POST` requests. Ensure you are not sending a `GET` or other method.

**Webhook returns 401 Unauthorized**
- Check that the `Authorization: Bearer {token}` header matches the `token` in the MagicForm channel config. Token comparison uses constant-time comparison.

**Webhook returns 400 "workspace path escapes workspace_root"**
- The `workspace` or `configDir` path in the payload resolves outside `workspace_root`. Ensure paths are relative (e.g. `s1/c1`, not `/data/workspaces/s1/c1`).

**Webhook returns 413 Request Entity Too Large**
- The request payload exceeds the 1 MB limit. Reduce the message size.

**Webhook returns 400 "workspace_root not configured"**
- Set `workspace_root` in the MagicForm channel config.

**Callback not received**
- Check that `callbackUrl` in the payload or `backend_url` in config is reachable from PicoClaw.
- Check PicoClaw logs for callback errors.
- Request contexts expire after 10 minutes.

**Session not persisting across requests**
- Ensure the same `workspace` path is sent for the same conversation.
- Sessions are stored at `{workspace}/sessions/`. Different workspace paths = different sessions.
