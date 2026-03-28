# WhatsApp Channel Configuration Guide

WhatsApp support in PicoClaw uses [whatsmeow](https://github.com/tulir/whatsmeow) (native mode) or an external WebSocket bridge (bridge mode). Native mode is recommended for most deployments.

> **Note:** WhatsApp native support requires a custom build. The default prebuilt binaries do not include it. See [Build Requirements](#build-requirements) below.

---

## 1. Build Requirements

The standard release binary does not include WhatsApp support. You must build from source with the `whatsapp_native` build tag.

**Prerequisites:**
- Go 1.21 or later ([install guide](https://go.dev/doc/install))
- Git

**Build steps:**

```bash
# Clone the repository
git clone https://github.com/sipeed/picoclaw
cd picoclaw

# Generate embedded assets (required before building)
go generate ./...

# Build with WhatsApp native support (Linux x86_64)
go build -tags whatsapp_native -ldflags "-s -w" -o picoclaw-whatsapp ./cmd/picoclaw

# Verify the binary
./picoclaw-whatsapp version
```

For other architectures:

```bash
# ARM64 (Raspberry Pi 4, etc.)
GOOS=linux GOARCH=arm64 go build -tags whatsapp_native -ldflags "-s -w" -o picoclaw-whatsapp-arm64 ./cmd/picoclaw

# ARMv7 (Raspberry Pi 3, etc.)
GOOS=linux GOARCH=arm GOARM=7 go build -tags whatsapp_native -ldflags "-s -w" -o picoclaw-whatsapp-armv7 ./cmd/picoclaw
```

> **Binary size:** The WhatsApp-enabled binary is approximately 60 MB (vs 20 MB for the standard binary) due to the whatsmeow library and SQLite dependency.

---

## 2. Example Configuration

> **Important:** Use absolute paths for `session_store_path` and `workspace`. Tilde (`~`) expansion is not supported in these fields — using `~` will cause PicoClaw to create a literal `~/` directory in your current working directory instead of your home directory.

Add the following to `~/.picoclaw/config.json`:

### Native Mode (recommended)

```json
{
  "channels": {
    "whatsapp": {
      "enabled": true,
      "use_native": true,
      "session_store_path": "/home/YOUR_USERNAME/.picoclaw/whatsapp-session/",
      "allow_from": []
    }
  }
}
```

### Bridge Mode

Bridge mode connects to an external WhatsApp bridge over WebSocket (for example [whatsapp-bridge](https://github.com/example/whatsapp-bridge)):

```json
{
  "channels": {
    "whatsapp": {
      "enabled": true,
      "use_native": false,
      "bridge_url": "ws://localhost:3001",
      "allow_from": []
    }
  }
}
```

### Full Config Example (native mode with all options)

```json
{
  "agents": {
    "defaults": {
      "workspace": "/home/YOUR_USERNAME/.picoclaw/workspace",
      "model_name": "my-model",
      "max_tokens": 2048,
      "temperature": 0.7,
      "max_tool_iterations": 50
    }
  },
  "model_list": [
    {
      "model_name": "my-model",
      "model": "anthropic/claude-haiku-4-5-20251001",
      "api_key": "YOUR_ANTHROPIC_API_KEY"
    }
  ],
  "channels": {
    "whatsapp": {
      "enabled": true,
      "use_native": true,
      "session_store_path": "/home/YOUR_USERNAME/.picoclaw/whatsapp-session/",
      "allow_from": ["+1234567890"],
      "reasoning_channel_id": ""
    }
  }
}
```

---

## 3. Field Reference

| Field | Type | Required | Description |
|---|---|---|---|
| `enabled` | bool | Yes | Enable or disable the WhatsApp channel |
| `use_native` | bool | Yes | `true` for native mode (whatsmeow); `false` for bridge mode |
| `session_store_path` | string | No (native) | Absolute path to store the WhatsApp session database. Session persists across restarts — no re-scan needed. Must be an absolute path (see note above). |
| `bridge_url` | string | Yes (bridge) | WebSocket URL of the external WhatsApp bridge (for example `ws://localhost:3001`). Only used when `use_native` is `false`. |
| `allow_from` | []string | No | Phone number whitelist in E.164 format (for example `["+919876543210"]`). Leave empty `[]` to accept messages from all numbers. |
| `reasoning_channel_id` | string | No | Route reasoning/thinking output to a separate channel ID. Leave empty to disable. |

---

## 4. First Run — QR Code Pairing

On first run, PicoClaw displays a QR code in the terminal:

```bash
picoclaw-whatsapp gateway
```

Expected output:

```
✓ Channels enabled: [whatsapp_native]
✓ Gateway started on 127.0.0.1:18790

Scan this QR code with WhatsApp (Linked Devices):
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄...
█ ▄▄▄▄▄ █...
...
```

**To pair your phone:**
1. Open WhatsApp on your phone
2. Tap **⋮ (three dots)** → **Linked Devices**
3. Tap **Link a Device**
4. Scan the QR code displayed in the terminal

After scanning, you will see:
```
WhatsApp login event event=success
WhatsApp native channel connected
```

The session is saved to `session_store_path`. Subsequent restarts connect automatically without re-scanning.

> **Note:** If the QR code expires before you scan it (roughly 60 seconds), restart the gateway to generate a new one.

---

## 5. Using a Local LLM (Ollama)

PicoClaw works with any OpenAI-compatible API, including a local [Ollama](https://ollama.com) server. This keeps all inference on-device — nothing leaves the machine.

### Install Ollama

```bash
# Download and extract (requires zstd)
curl -L -o /tmp/ollama-linux-amd64.tar.zst \
  https://github.com/ollama/ollama/releases/latest/download/ollama-linux-amd64.tar.zst
zstd -d /tmp/ollama-linux-amd64.tar.zst -o /tmp/ollama-linux-amd64.tar
tar -xf /tmp/ollama-linux-amd64.tar -C ~/.local   # extracts bin/ollama AND lib/ollama/
```

> **Important:** You must extract the full tar, not just the binary. The `lib/ollama/` directory contains the CUDA runtime libraries (`libcublas`, `libcudart`, `libggml-cuda.so`) required for GPU inference. Extracting only the `bin/ollama` binary causes Ollama to silently fall back to CPU-only mode, resulting in 2-minute timeouts on every request.

Pull a model and start the server:

```bash
export PATH="$HOME/.local/bin:$PATH"
ollama pull llama3.2:3b          # ~2 GB
ollama serve &
```

### Config for Ollama

```json
{
  "model_list": [
    {
      "model_name": "llama-local",
      "model": "openai/llama3.2:3b",
      "api_base": "http://localhost:11434/v1",
      "api_key": "ollama",
      "request_timeout": 120
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "llama-local"
    }
  }
}
```

> **`request_timeout` note:** The first inference request after starting Ollama loads the model into VRAM (30–60 seconds on a laptop GPU). Set `request_timeout` to at least `120` to avoid a premature timeout on cold start. Subsequent requests are fast once the model is resident in memory.

---

## 6. Running as a Background Service

### Using systemd (recommended for servers)

Create `/etc/systemd/system/picoclaw.service`:

```ini
[Unit]
Description=PicoClaw WhatsApp Gateway
After=network.target

[Service]
Type=simple
User=YOUR_USERNAME
ExecStart=/usr/local/bin/picoclaw-whatsapp gateway
Restart=on-failure
RestartSec=5
Environment=PICOCLAW_CONFIG=/home/YOUR_USERNAME/.picoclaw/config.json

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable picoclaw
sudo systemctl start picoclaw
sudo systemctl status picoclaw
```

### Using nohup (quick testing)

```bash
nohup picoclaw-whatsapp gateway > /home/YOUR_USERNAME/.picoclaw/gateway.log 2>&1 &
```

---

## 7. HEARTBEAT.md Warning

PicoClaw runs a heartbeat every 30 minutes and executes tasks listed in `HEARTBEAT.md` in your workspace. The default file ships with example tasks:

```
- Check for unread messages
- Review upcoming calendar events
- Check device status (e.g., MaixCam)
```

> **Warning:** The agent will attempt to execute every task in this file. If the model has no actual tools to fulfil a task (for example calendar access or device APIs), it will **hallucinate results** — inventing fake meeting schedules, device readings, and API responses — and send them to your WhatsApp.

**Before going to production, either:**
- Remove all example tasks and add only tasks the agent can genuinely fulfil with its available tools, or
- Delete the section below the `---` divider to disable heartbeat tasks entirely

Safe example:
```
## Heartbeat Tasks
- Report current time and confirm gateway is running
```

---

## 8. Currently Supported

- Text message send and receive
- QR code pairing via terminal
- Persistent session (no re-scan after restart)
- Per-number allow-list (`allow_from`)
- Individual (DM) message handling
- Reasoning output routing (`reasoning_channel_id`)
- Bridge mode via external WebSocket

---

## 9. Troubleshooting

| Problem | Cause | Fix |
|---|---|---|
| Gateway starts but shows no QR code | Previous session exists | Delete `session_store_path` directory and restart |
| `failed to read frame header: EOF` after restart | WhatsApp session was logged out from phone | Delete session directory and restart to re-scan QR |
| Session created in wrong directory (literal `~/` folder) | Tilde not expanded in `session_store_path` | Use absolute path: `/home/username/.picoclaw/whatsapp-session/` |
| `build tag whatsapp_native not found` | Building without required tag | Use `go build -tags whatsapp_native` |
| QR code not rendering correctly | Terminal does not support Unicode block characters | Switch to a terminal that supports Unicode (for example GNOME Terminal, iTerm2) |
| `can't link new devices at this time` | WhatsApp rate limiting | Wait 10 minutes and restart gateway |
| Bot not responding to messages | `allow_from` set incorrectly | Check number format is E.164 (for example `+919876543210`), or set `allow_from: []` to allow all |
| Agent sends hallucinated calendar/device data | HEARTBEAT.md contains tasks with no real tools | Remove example tasks from HEARTBEAT.md (see [HEARTBEAT.md Warning](#heartbeatmd-warning)) |
| `go generate` step fails | Missing dependencies | Run `go mod download` first, then retry `go generate ./...` |
| Ollama times out on every request (CPU only, 2-minute hang) | Ollama tar.zst was partially extracted — only the binary, not `lib/ollama/` | Re-extract the full `ollama-linux-amd64.tar.zst`; it must unpack both `bin/ollama` **and** `lib/ollama/cuda_v12/` (cuBLAS, CUDA runtime, GGML CUDA kernel). Without these the runner silently falls back to CPU and hits the `request_timeout`. |

**To reconnect after logging out from your phone:**
```bash
rm -rf /home/YOUR_USERNAME/.picoclaw/whatsapp-session/
# then restart the gateway — a new QR code will appear
```

---

## 10. TODO

- Group message support
- Media message send/receive (images, audio, documents)
- Typing indicator
- Read receipts
- Placeholder message while agent is processing
- Status message support
