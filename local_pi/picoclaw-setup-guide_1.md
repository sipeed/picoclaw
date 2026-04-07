# PicoClaw — Complete Setup Guide
> Raspberry Pi Zero 2W · Go Agent Runtime · Multi-LLM Routing  
> Hostname: `picoclaw` · User: `tim` · SD Card + USB Stick

---

> ⚠️ **Prerequisites — Complete Argus First**  
> Argus (your networking Pi) must already be running Pi-hole + Unbound so that hostname-based DNS resolution works on your LAN. PicoClaw depends on it to resolve `vulcan` for local Ollama routing.

---

## Table of Contents

1. [Architecture Overview](#0-architecture-overview)
2. [Accounts & API Keys](#1-accounts--api-keys--do-this-first)
3. [Hardware Setup & SD Card Flash](#2-hardware-setup--sd-card-flash)
4. [System Configuration](#3-system-configuration-on-the-pi)
5. [PicoClaw Installation](#4-picoclaw-installation)
6. [SOUL.md — System Identity File](#5-soulmd--the-system-identity-file)
7. [Main Configuration — config.yaml](#6-main-configuration--configyaml)
8. [Ollama vs OpenRouter](#7-ollama-vs-openrouter--when-each-fires)
9. [The First Three Agents](#8-the-first-three-agents)
10. [Migration: Moving to USB](#10-migration-moving-to-usb)
11. [Running PicoClaw as a Service](#9-running-picoclaw-as-a-service)
12. [Telegram Bot Setup](#10-telegram-bot-setup--full-walkthrough)
13. [Vulcan Prerequisites](#11-vulcan-windows-desktop--prerequisites)
14. [Onboarding Checklist](#12-onboarding-checklist)
15. [Credentials Reference](#13-credentials-reference)
16. [Troubleshooting](#14-troubleshooting)
17. [Next Steps](#15-next-steps)

---

## 0. Architecture Overview

PicoClaw is a Go-based, single-binary AI agent framework. It uses under 10 MB of RAM and runs on ARM64. It does **not** run LLMs — it orchestrates them, routing prompts to cloud APIs (OpenRouter) or to your local Ollama instance on Vulcan over LAN.

```
You (Telegram/Discord)
        ↓
PicoClaw on Pi Zero 2W
  → reads SOUL.md + agent config
  → scores message complexity
        ↓
┌───────────────────────────────────────┐
│           Complexity Router           │
│  score < 40  →  Ollama (local/free)   │
│  score >= 40 →  OpenRouter (cloud)    │
└───────────────────────────────────────┘
        ↓                    ↓
Ollama on Vulcan     OpenRouter API
mistral:7b etc.      Gemini Flash (default)
                     Claude Sonnet (complex)
        ↓
Response back to Telegram/Discord
```

> 💡 **Cost philosophy:** Start with one agent, learn cost patterns, then expand. Gemini Flash is ~$0.075/M tokens — almost free. Claude Sonnet only fires for genuinely complex tasks. Local Ollama costs nothing.

---

## 1. Core Architecture: "One Bot, Multiple Souls"
To optimize the limited RAM (512MB) of the Pi Zero 2W, we implemented a single-instance architecture:
- **One Telegram Bot / One Harness:** The `config.json` stores the primary Telegram token. Sub-agents (`agent.yaml`) omit their tokens so no extra network connections are opened.
- **Hot-Swapping:** Use the `/agent [AgentName]` command in Telegram to seamlessly cycle between your agents (Alpha, Pulse, Forge) while keeping the same chat history.

## 2. Accounts & API Keys — Do This First

Complete all account setups before touching the Pi. You need these credentials on hand.

### 1.1 OpenRouter (Primary Cloud LLM Gateway)

OpenRouter gives you access to Gemini Flash, Claude Sonnet, and many others from a single API key — no separate Anthropic/Google accounts needed.

1. Go to: https://openrouter.ai → Sign up
2. Settings → API Keys → **Create Key**
   - **Name:** `picoclaw` (or `picoclaw_pi`)
   - **Credit Limit:** $5.00 (as a safety "circuit breaker")
   - **Reset Frequency:** Monthly (optional, e.g., $2.00/mo)
   - **Expiration:** No expiration
3. Add credit: Settings → Credits → top up $5–10 (Gemini Flash is extremely cheap)
4. Save your key: It starts with `sk-or-v1-...` — store it in a password manager.

> 💡 **Note:** The key name is just a label for you; you can change it later in the dashboard without breaking your bot. What matters is the `sk-or-v1...` string itself.
> 📌 **Save this:** Key format: `sk-or-v1-xxxxxxxxxxxxxxxx` | Dashboard shows per-model spend. Budget alerts available.

---

### 1.2 Telegram Bot (Primary Interface)

You will create one Telegram bot per agent. Each bot has its own token and acts as the chat interface for that agent.

#### Creating your three bots with BotFather

You will create one Telegram bot for each agent. Each bot acts as a separate interface. Repeat these steps for all three:

1. Open Telegram → Search for `@BotFather`
2. Send: `/newbot`
3. Follow the naming convention below:

| Agent | Display Name | Bot Username (Example) |
| :--- | :--- | :--- |
| **Alpha** | `Picoclaw Alpha` | `picoclaw_alpha_bot` |
| **Pulse** | `Picoclaw Pulse` | `picoclaw_pulse_bot` |
| **Forge** | `Picoclaw Forge` | `picoclaw_forge_bot` |

4. BotFather replies with your **API Token** — save each one separately.
   - **Important:** Copy the **entire string**, including the numbers and the colon (e.g., `123456789:ABCDefgh...`).
5. **Disable Privacy Mode** (Crucial): `/setprivacy` → select bot → **Disable**.
6. **Get your User ID**: Search for `@userinfobot` → `/start` → save your ID. This number is what you use to whitelist yourself in the PicoClaw config.

> ⚠️ **One token per agent.** Alpha gets Token A, Pulse gets Token B. You will enter these into their respective `agent.yaml` files later. You will use it to whitelist yourself in PicoClaw config.

---

### 1.3 Google Gemini API (Optional — for direct access)

**Why do this?** OpenRouter already gives you access to Gemini. You only need this separate key if you want a "free fallback" or to bypass OpenRouter later and talk to Gemini directly.

1. Go to: https://aistudio.google.com → Sign in with your Google account
2. Click "Get API key" → **Create API key**
3. **Choose an imported project:** Leave it as **"Default Gemini Project"** — this is perfectly fine.
4. **Funding/Cost:** This key uses a **free tier** (15 requests/min). You do **not** need to add any money here. Your primary funding goes to **OpenRouter**, which then pays for the models you use through their bridge.

---

### 1.4 Anthropic API (Optional — direct Claude access)

Only needed if you want to bypass OpenRouter for Claude Sonnet calls. OpenRouter is recommended — simpler billing.

1. Go to: https://console.anthropic.com → Sign up / log in
2. Settings → API Keys → Create Key → name it `picoclaw-claude`
3. Add credits in Billing section
4. Current Sonnet model string: `claude-sonnet-4-20250514`

---

### 1.5 Vulcan — Ollama Setup (Local LLM)

Ollama runs on Vulcan (Windows desktop, Ryzen 3700X, GTX 1660 Super 6GB). No account required.

1. Download from: https://ollama.com → Install the Windows `.exe`
2. Open PowerShell and pull your models:

```powershell
ollama pull mistral:7b-instruct-q4_K_M
ollama pull qwen2.5:7b-q4_K_M
ollama pull qwen2.5-coder:7b-q4_K_M
ollama pull nomic-embed-text
```

3. By default, Ollama listens on `localhost` only. To allow the Pi to reach it over LAN:
   - Press **Windows Key** → type "env" → select **Edit the system environment variables**.
   - Click **Environment Variables** (bottom right).
   - Under **System variables** (bottom section) → click **New...**.
   - **Variable name:** `OLLAMA_HOST` | **Variable value:** `0.0.0.0:11434`
   - Click OK (on all windows).
   - **Restart Ollama:** Right-click the Ollama icon in your tray → Quit → then re-launch it.
5. From PicoClaw Pi it will be reachable at: `http://vulcan:11434` (once Argus DNS resolves it)

> 🔒 **Security note:** `OLLAMA_HOST=0.0.0.0` exposes Ollama to your whole LAN. This is fine for initial setup.
> 
> **To harden it later:**
> - **Option 1 (Tailscale):** Set `OLLAMA_HOST` to your Vulcan's **Tailscale IP** instead. This keeps it entirely off your local WiFi and only on your private encrypted mesh.
> - **Option 2 (Firewall):** Create a Windows Firewall rule for Port 11434 that only allows inbound traffic from your specific Pi IPs (`.108`, `.109`).

---

## 2. Hardware Setup & SD Card Flash

### 2.1 Hardware checklist

| Item | Notes |
|---|---|
| Raspberry Pi Zero 2W | ARM Cortex-A53 quad-core, 512MB RAM |
| microSD card 32GB+ | SanDisk MAX Endurance or Samsung Pro Endurance |
| USB flash drive (USB-A) | Used via OTG adapter for `/mnt/usb` — holds agent workspace |
| Micro-USB OTG adapter | USB-A female to Micro-USB male |
| USB-C power supply 3A | For the power port (separate from OTG data port) |
| Vulcan (your desktop) | Used to flash the SD card and SSH in |

> 🔌 **Port layout on Pi Zero 2W:** Two Micro-USB ports side by side. The one labelled `PWR IN` is power-only. The one labelled `USB` is the OTG data port. Your USB flash drive connects to the OTG port via the OTG adapter. Power goes to `PWR IN` separately.

---

### 2.2 Flash the SD card (on Vulcan, Windows)

1. Download Raspberry Pi Imager: https://www.raspberrypi.com/software/
2. Insert SD card into Vulcan's card reader
3. Open Imager → Choose Device: `Raspberry Pi Zero 2 W`
4. Choose OS: `Raspberry Pi OS Lite (64-bit)` — no desktop, minimal
5. Choose Storage: your SD card
6. Click the gear/settings icon ⚙️ **before** writing:
   - Hostname: `picoclaw`
   - Username: `tim` | Password: choose a strong password, save it
   - Configure WiFi: enter your router's **main SSID** (not the `_EXT` extender SSID)
   - WiFi country: `DE` (or your country code)
   - Enable SSH: check "Use password authentication"
7. Click Save → Write → Yes to confirm
8. Wait for write + verify to complete (~3–5 min)
9. Eject the SD card safely

---

### 2.3 First boot

1. Insert SD card into the Pi Zero 2W
2. Connect the USB flash drive to the OTG port via OTG adapter
3. Connect USB-C power to the `PWR IN` port
4. Wait **3 minutes** — first boot does filesystem expansion
5. Find the Pi's IP from your router admin page, or try:

```bash
ping picoclaw.local
```

6. SSH in from Vulcan (PowerShell):

```bash
ssh tim@picoclaw.local
```

If `.local` resolution fails, use the IP directly: `ssh tim@192.168.x.x`

---

### 2.4 DHCP reservation (on your TP-Link VX800v router)

1. Log into router admin: `http://192.168.1.1`
2. Go to Advanced → DHCP Server → Address Reservation
3. Find `picoclaw` in the DHCP client list (it shows after first boot)
4. Reserve a static IP — e.g. **`192.168.1.118`** (Adapter) and **`192.168.1.117`** (WiFi)
5. Add to Pi-hole's local DNS (on Argus): Admin → Local DNS → DNS Records
   - Domain: `picoclaw` → IP: `192.168.1.118`
   - Domain: `picoclaw` → IP: `192.168.1.117`
6. From now on, SSH via: `ssh tim@picoclaw`

> 💡 **Important: Network Identity & IP Persistence**
> - **IPs follow MACs:** If you use a USB-to-Ethernet adapter and move it from Argus to Picoclaw, Picoclaw will take over Argus's "reserved" IP. The router identifies the device by the adapter's MAC address, not the Pi itself.
> - **WiFi vs Ethernet:** Each has its own MAC. If you switch Argus from Ethernet to WiFi, it will get a different IP address because its WiFi MAC is different from your Ethernet adapter.
> - **Verification:** Always check `ping argus.local` or `ping picoclaw.local` to confirm where your devices ended up after a hardware swap.

> 💡 **How to Apply Network Changes:**
> After saving your reservation in the router, you must reboot the Pi to "pick up" the new IP. 
> 1. Use the current IP: `ssh tim@192.168.1.x`
> 2. Run: `sudo reboot`
> 3. After 90 seconds, reconnect via the new IP: `ssh tim@192.168.1.107`

---

## 3. System Configuration (on the Pi)

SSH into `picoclaw` and run these steps in order.

### 3.1 System update & essentials

```bash
sudo apt update && sudo apt full-upgrade -y
sudo apt install -y git curl wget python3 python3-pip python3-venv unzip htop btop vim tmux fail2ban unattended-upgrades

# Configure btop (redirect config to RAM to save SD card)
mkdir -p /dev/shm/btop
ln -s /dev/shm/btop ~/.config/btop || true
sudo reboot
```

---

### 3.2 Mount the USB flash drive

1. Find the drive's device name:

```bash
lsblk
# Look for sda or sda1 — your flash drive
```

2. Format if new (skip if already has files):

```bash
sudo mkfs.ext4 /dev/sda1
```

3. Create mount point and mount:

```bash
sudo mkdir -p /mnt/usb
sudo mount /dev/sda1 /mnt/usb
```

4. Make it mount on boot — get the UUID:

```bash
sudo blkid /dev/sda1
# Copy the UUID value, e.g: UUID="a1b2c3d4-..."

sudo nano /etc/fstab
# Add this line at the bottom:
UUID=YOUR_UUID_HERE /mnt/usb ext4 defaults,nofail 0 2
```

5. Test the fstab entry:

```bash
sudo mount -a && df -h | grep usb
```

6. Set ownership:

```bash
sudo chown -R tim:tim /mnt/usb
```

---

### 3.3 Hostname resolution check

```bash
ping vulcan                   # should resolve via Pi-hole on Argus
curl http://vulcan:11434      # should return Ollama version JSON
```

> ⚠️ **If `vulcan` does not resolve:** Check that Argus Pi-hole has a DNS record for `vulcan → 192.168.1.x`. Also verify your Pi is using Argus as its DNS server (`cat /etc/resolv.conf`). If Argus is down, add `vulcan` to `/etc/hosts` as a temporary fallback.

---

### 3.4 Fail2Ban (SSH protection)

```bash
sudo systemctl enable fail2ban
sudo systemctl start fail2ban
sudo fail2ban-client status sshd    # verify jail is active
```

---

### 3.5 Tailscale (remote access — optional but recommended)

```bash
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up
# Follow the auth URL — log into your Tailscale account: https://login.tailscale.com/a/1d75a1d001aafb
# Once connected: ssh tim@picoclaw.your-tailnet.ts.net from anywhere
```

---

### 3.6 Automated backups to Vulcan

1. On Vulcan (PowerShell), enable OpenSSH server if not already running:

```powershell
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0
Start-Service sshd
Set-Service -Name sshd -StartupType Automatic
```

2. On the Pi — create SSH key and copy to Vulcan:

### Step 1: Initialize Backup Path & SSH on Vulcan
1. **On Vulcan (PowerShell - run as Admin):**
   ```powershell
   # 1. Create the backup destination on Google Drive
   $path = "G:\Meine Ablage\PicoclawBackups"
   New-Item -ItemType Directory -Force -Path $path

   # 2. Add Picoclaw's identity to your desktop (optional if already one-click)
   $env:USERPROFILE\.ssh\authorized_keys
   ```

### Step 2: Set up SSH Key on Picoclaw
1. **On Picoclaw (Pi Terminal):**
   ```bash
   # 1. Generate the outbound key
   ssh-keygen -t ed25519 -f ~/.ssh/backup_key -N ""

   # 2. Get the key text to copy
   cat ~/.ssh/backup_key.pub
   ```
   *Copy the long line starting with `ssh-ed25519...argus-backup`*

3. **Back on Vulcan (PowerShell):**
   ```powershell
   # Add the key to your desktop (Paste the key where it says 'PASTE_HERE')
   Add-Content -Path "$env:USERPROFILE\.ssh\authorized_keys" -Value "PASTE_HERE"
   ```

3. Create backup script:

```bash
nano ~/backup.sh
```

Content of `backup.sh`:

```bash
#!/bin/bash
# Backup PicoClaw config and USB workspace to Vulcan
# --- VULCAN FAILOVER LOGIC ---
VULCAN_IPS=("192.168.1.115" "192.168.1.114") # Ethernet Base, then WiFi Buddy
TARGET_IP=""

for ip in "${VULCAN_IPS[@]}"; do
    if ping -c 1 -W 1 "$ip" >/dev/null 2>&1; then
        TARGET_IP="$ip"
        break
    fi
done

if [ -z "$TARGET_IP" ]; then
    echo "Error: Could not reach Vulcan on any expected IP (.115 or .110)."
    exit 1
fi
# -----------------------------

DEST="tim@$TARGET_IP:/c/Users/TimZickenrott/Backups/picoclaw"
KEY="$HOME/.ssh/id_picoclaw_backup"

# Backup USB workspace (SOUL.md, agents, memory)
rsync -av --delete -e "ssh -i $KEY" /mnt/usb/picoclaw/ "$DEST/usb/"

# Backup home config
rsync -av --delete -e "ssh -i $KEY" ~/picoclaw/ "$DEST/config/"

echo "Backup complete via $TARGET_IP: $(date)"
```

Now make it executable and add a cron job to run it daily at 9 PM:
```bash
chmod +x ~/backup.sh

# Add crontab entry — run daily at 9 PM
crontab -e
# Add:
0 21 * * * /home/tim/backup.sh >> /home/tim/backup.log 2>&1
```

---

## 4. PicoClaw Installation

### 4.1 Download & Deploy Binary (32-bit / armv7l)
Since you are on 32-bit Pi OS, use the `armv7` asset:

```bash
mkdir -p ~/picoclaw && cd ~/picoclaw

# 1. Clean up old/wrong files first
rm -f picoclaw picoclaw.tar.gz

# 1. Download the 32-bit archive
curl -L "https://github.com/sipeed/picoclaw/releases/download/v0.2.5/picoclaw_Linux_armv7.tar.gz" -o picoclaw.tar.gz

# 2. Extract it
tar -xvf picoclaw.tar.gz

# 3. Test the binary
chmod +x picoclaw
./picoclaw --version
```

---

### 4.2 Directory structure

The USB stick at `/mnt/usb` holds all PicoClaw runtime state. 

> 💡 **No USB stick yet?** No problem. Use `/home/tim/picoclaw-data/` instead. When the stick arrives, you can simply move the folder to the USB and update the paths in `config.yaml`.

```
/mnt/usb/picoclaw/
  config/
    config.yaml               ← main config (API keys, routing, platforms)
  SOUL.md                     ← system identity file
  agents/
    agent-01-alpha/           ← Finance, investing, portfolio
      agent.yaml
      memory.md
    agent-02-pulse/           ← Biohacking, health, fitness
      agent.yaml
      memory.md
    agent-03-forge/           ← Coding, dev, systems
      agent.yaml
      memory.md
  workspace/                  ← agent working files, outputs, logs
    ibkr/                     ← IBKR Flex Query exports
    tr/                       ← Trade Republic CSV exports
    watchlist.csv
    picoclaw.log
```

```bash
mkdir -p /mnt/usb/picoclaw/{config,agents,workspace}
mkdir -p /mnt/usb/picoclaw/agents/{agent-01-alpha,agent-02-pulse,agent-03-forge}
mkdir -p /mnt/usb/picoclaw/workspace/{ibkr,tr}
touch /mnt/usb/picoclaw/SOUL.md
touch /mnt/usb/picoclaw/agents/agent-01-alpha/memory.md
touch /mnt/usb/picoclaw/agents/agent-02-pulse/memory.md
touch /mnt/usb/picoclaw/agents/agent-03-forge/memory.md


```bash
# 💡 NO USB? RUN THIS INSTEAD (Uses your Home directory)
mkdir -p ~/picoclaw-data/{config,agents,workspace}
mkdir -p ~/picoclaw-data/agents/{agent-01-alpha,agent-02-pulse,agent-03-forge}
mkdir -p ~/picoclaw-data/workspace/{ibkr,tr}
touch ~/picoclaw-data/SOUL.md
touch ~/picoclaw-data/agents/agent-01-alpha/memory.md
touch ~/picoclaw-data/agents/agent-02-pulse/memory.md
touch ~/picoclaw-data/agents/agent-03-forge/memory.md
```

---

## 5. SOUL.md — The System Identity File

SOUL.md is injected into every agent's system prompt. It defines who the AI is — its personality, values, tone, boundaries, and operating context. Agents can override or extend it via their own `agent.yaml` system prompt.

### 5.1 Open and fill in SOUL.md

```bash
# USB stick
nano /mnt/usb/picoclaw/SOUL.md
# SD card
nano ~/picoclaw-data/SOUL.md
```

### 5.2 Template — customize to your taste

```markdown
# SOUL — System Identity

## Who you are
You are an AI assistant running locally on a Raspberry Pi Zero 2W
as part of Tim's personal homelab. You are helpful, direct, and honest.

## Operating context
- You run on constrained hardware — be efficient with token usage
- You route to local Ollama models when possible (free)
- You use cloud models (Gemini, Claude) for complex tasks only
- Your primary user is Tim. Only respond to whitelisted user IDs.

## Tone & style
- Direct and clear. No unnecessary preamble.
- German or English depending on what Tim writes in.
- Concise responses unless asked to elaborate.
- Honest about limitations — if unsure, say so.

## Privacy & boundaries
- Do not store sensitive data in memory unless explicitly asked.
- Do not share information with unauthorized users.
- Refuse requests that conflict with Tim's interests.

## Memory
- Each agent has its own memory.md file.
- You can reference past context from memory when relevant.
```

> 💡 **SOUL.md evolves.** Start simple. After a few weeks you will discover what personality traits and instructions actually matter. Iterate on it like a living document.

---

## 6. Main Configuration — config.json

### 6.1 Create the config file

```bash
nano /mnt/usb/picoclaw/config/config.yaml
```

### 6.2 config.yaml template

```yaml
# PicoClaw main configuration

# ── User whitelist ──────────────────────────────────────
authorized_users:
  - YOUR_TELEGRAM_USER_ID     # get this from @userinfobot

# ── Platform connections ────────────────────────────────
telegram:
  enabled: true
  # Agent-specific tokens are set in each agent.yaml

# ── LLM routing ────────────────────────────────────────
routing:
  # Complexity scorer: 0-100 score per message
  # Low score (< 40)  → local Ollama
  # High score (>= 40) → cloud via OpenRouter
  complexity_threshold: 40

  # Default cloud model (cheap, fast)
  cloud_default: "google/gemini-flash-1.5"

  # Complex/reasoning tasks
  cloud_complex: "anthropic/claude-sonnet-4-5"

  # Local Ollama endpoint (resolves via Pi-hole DNS)
  local_endpoint: "http://vulcan:11434"
  local_default: "mistral:7b-instruct-q4_K_M"

# ── API keys ────────────────────────────────────────────
api_keys:
  openrouter: "sk-or-v1-YOUR_KEY_HERE"
  # Optional direct keys (uncomment if needed):
  # anthropic: "sk-ant-YOUR_KEY"
  # gemini: "YOUR_GEMINI_KEY"

# ── Agent workspace ─────────────────────────────────────
workspace: "/mnt/usb/picoclaw"
soul_file: "/mnt/usb/picoclaw/SOUL.md"

# ── Logging ─────────────────────────────────────────────
log_level: "info"     # debug | info | warn | error
log_file: "/mnt/usb/picoclaw/workspace/picoclaw.log"
```

# Option A: SD Card (Start here)
workspace: "/home/tim/picoclaw-data"
soul_file: "/home/tim/picoclaw-data/SOUL.md"
log_file: "/home/tim/picoclaw-data/workspace/picoclaw.log"

# Option B: USB Stick (uncomment after migration)
# workspace: "/mnt/usb/picoclaw"
# soul_file: "/mnt/usb/picoclaw/SOUL.md"
# log_file: "/mnt/usb/picoclaw/workspace/picoclaw.log"

```bash
# Lock down permissions — only tim can read API keys
chmod 600 ~/picoclaw-data/config/config.yaml
# USB
# chmod 600 /mnt/usb/picoclaw/config/config.yaml
```

> 🔒 `config.yaml` contains your API keys. The `chmod 600` above ensures only `tim` can read it.

---

## 7. Ollama vs OpenRouter — When Each Fires

| | Ollama (Local, Vulcan) | OpenRouter (Cloud) |
|---|---|---|
| **Cost** | Free (your GPU) | Pay-per-token — very cheap |
| **Privacy** | Fully local, nothing leaves LAN | Tokens sent to provider |
| **Speed** | ~5–15 tok/s on 1660 Super | ~50–200 tok/s |
| **Quality** | Good for simple tasks | Better for complex reasoning |
| **Availability** | Vulcan must be on | Always available |
| **Best for** | Quick Q&A, formatting, summaries | Analysis, long-form, code review |
| **Default model** | `mistral:7b-instruct-q4_K_M` | `google/gemini-flash-1.5` |
| **Triggered by** | Complexity score < 40 | Complexity score >= 40 |

### Complexity scorer heuristics

- **Message length** — longer messages score higher
- **Question type** — `why/how/analyze` = high; `what/who/when` = low
- **Keywords** — `compare`, `explain`, `debug`, `plan` → score bumped up
- **Agent-level override** — finance/research agents can force cloud model

> 💡 **Force an override:** Prefix your message with `!cloud` or `!local` to bypass the router. Useful when testing.

---

## 8. The First Three Agents & How to Use Your Agent 🤖

### 8.1 Interaction
You don't need strict commands! You can talk to Picoclaw like a human.
- **Natural Language:** "What's the status of my Pi?" or "Summarize my network logs."
- **Commands:** 
  - `/start` - Wake up the agent and see its "Soul".
  - `/switch` - Change between your agents (Alpha, Forge, Pulse).
  - `/status` - Check system health and VRAM usage.

### 8.2 Real-Time Response
Picoclaw works in **Real-Time**. Every message you send triggers an immediate response from the agent on your Pi Zero. It does not batch messages.

### 8.3 Complexity Routing (Adaptive Brain) 🧠

Picoclaw uses an LLM-based "Scout" to score every incoming prompt from **0 to 100**.
- **The Threshold:** Usually set to **40**.
- **Simple Tasks (Score < 40):** Handled by **Gemini 3.1 Flash Lite** (High speed, 500 free requests/day).
- **Complex Tasks (Score > 40):** Automatically escalated to **Claude 3.5 Sonnet** (Deep reasoning).
- **How it works:** A small local model or a cheap cloud model "reads" your intent first and decides if a "heavyweight" brain is needed.

### 8.4 Multi-Agent Architecture (One Bot vs. Three)
Since you are on a **Pi Zero 2W** (limited RAM), the "One Bot to Rule Them All" approach is best. Think of it like this:

- **🏠 The House (Instance):** You only have **one** Picoclaw process running on your Pi. It has your Telegram token and your `config.json`.
- **📜 The Global Rules (`SOUL.md`):** This is the **Family Code**. It’s the set of rules that every person in the house must follow (e.g., "Always start with a model header"). Every agent "inherits" these.
- **🎭 The Three People (Agents):** Each lives in its own folder (`agent-01-alpha`, etc.) with its own **Unique Instructions** (`agent.yaml`) and **Unique Memory** (`memory.md`).

**How to Use it:** When you type `/agent Alpha`, the "House" (Picoclaw) essentially says: *"I'm now putting on the Alpha costume. I'll use his instructions and his specific memory for this chat."*

**The result:** You get 3 totally distinct personalities with separate brains, but you only pay the "RAM price" (memory cost) for one of them at a time! 🧠🦾✨🤖

---

## 9. Monitoring & Auditing 🔍

To see exactly what the agent is "thinking" or why it chose a certain model, run these on the Pi:

```bash
# ── Live Brain Feed ──
journalctl -u picoclaw -f  # This is the PRIMARY log — see thoughts & routing here.

# ── Agent Search ──
# If you need to find where a specific agent is logging:
find /home/tim/picoclaw-data/workspace -name "*.log"

Agents are stateless folder definitions on disk — zero idle overhead. Each lives in its own folder on the USB stick with an `agent.yaml` and `memory.md`. **Start with Alpha only.** Add Pulse and Forge once you understand cost patterns and routing behavior.

The three agents and their scope:

| Name | Domain | Telegram bot name |
|---|---|---|
| **Alpha** | Finance, investing, portfolio (IBKR, Trade Republic) | `picoclaw_alpha_bot` |
| **Pulse** | Biohacking, health, fitness, recovery, supplements | `picoclaw_pulse_bot` |
| **Forge** | Coding, dev, systems, homelab, architecture | `picoclaw_forge_bot` |

Create all three folders now, activate Alpha first:

```bash
# USB stick
mkdir -p /mnt/usb/picoclaw/agents/{agent-01-alpha,agent-02-pulse,agent-03-forge}
touch /mnt/usb/picoclaw/agents/agent-01-alpha/memory.md
touch /mnt/usb/picoclaw/agents/agent-02-pulse/memory.md
touch /mnt/usb/picoclaw/agents/agent-03-forge/memory.md
# SD card
mkdir -p ~/picoclaw-data/agents/{agent-01-alpha,agent-02-pulse,agent-03-forge}
touch ~/picoclaw-data/agents/agent-01-alpha/memory.md
touch ~/picoclaw-data/agents/agent-02-pulse/memory.md
touch ~/picoclaw-data/agents/agent-03-forge/memory.md
```

---

### Agent 1: Alpha — Finance, Investing & Portfolio

Alpha is your personal investment intelligence agent. It knows your IBKR and Trade Republic positions, tracks your watchlist, helps analyze earnings and semiconductor sector news, and answers portfolio questions. Start here — this is the most immediately useful and the one that justifies the whole setup.

```bash
nano /mnt/usb/picoclaw/agents/agent-01-alpha/agent.yaml
```bash
# Edit the agent configuration (e.g., Alpha)
nano ~/picoclaw-data/agents/agent-01-alpha/agent.yaml
```

```yaml
# Agent 01 — Alpha (Finance, Investing & Portfolio)
name: "Alpha"
description: "Finance, investing, portfolio monitoring — IBKR + Trade Republic"
version: "1.0"

telegram_token: "YOUR_ALPHA_BOT_TOKEN_HERE"

# Finance analysis needs strong reasoning — push to cloud by default
complexity_override: 55      # most queries go cloud
force_complex_model: true    # use Claude Sonnet for deep analysis

system_prompt: |
  You are Alpha, Tim's personal finance and investing intelligence agent.
  You are sharp, data-driven, and direct — no filler, no hedging unless
  genuinely uncertain.

  ## Your domain
  - Investment portfolio: IBKR (primary) + Trade Republic
  - Sectors of deep interest: semiconductors, AI infrastructure, European tech
  - Investing philosophy: long-term, fundamentals-first, occasional tactical trades
  - Base currency: EUR

  ## What you do
  1. Summarize portfolio performance and positions on request
  2. Monitor the ticker watchlist for notable moves (>5% daily)
  3. Pull and summarize recent news for holdings or watched tickers
  4. Analyze earnings transcripts and investor presentations when pasted in
  5. Help think through investment theses — pros, cons, risks, catalysts
  6. Generate tax-relevant summaries from IBKR Flex Query data
  7. Flag positions with >10% unrealized loss prominently (❌)
  8. Flag strong performers with ✅

  ## Data sources
  # USB mounted data
  - IBKR Flex Query exports → /mnt/usb/picoclaw/workspace/ibkr/
  - Trade Republic CSV → /mnt/usb/picoclaw/workspace/tr/
  - Ticker watchlist → /mnt/usb/picoclaw/workspace/watchlist.csv

  # SD card data
  - IBKR Flex Query exports → ~/picoclaw-data/workspace/ibkr/
  - Trade Republic CSV → ~/picoclaw-data/workspace/tr/
  - Ticker watchlist → ~/picoclaw-data/workspace/watchlist.csv

  ## Response style
  - Always state when data was last updated
  - Lead with the headline number / key takeaway
  - Use tables for position summaries
  - Flag sector concentration risks if relevant
  - If Tim pastes an earnings transcript, extract: revenue, guidance,
    margin trend, management tone, and 3 key risks

memory_file: "agents/agent-01-alpha/memory.md"

commands:
  - name: "portfolio"
    description: "Full portfolio summary with P&L"
  - name: "watchlist"
    description: "Watchlist tickers with last known prices"
  - name: "news [ticker]"
    description: "Recent news for a specific ticker"
  - name: "pnl"
    description: "P&L breakdown by position"
  - name: "thesis [ticker]"
    description: "Investment thesis summary for a holding"
  - name: "memory"
    description: "Show what Alpha remembers about your portfolio"
  - name: "status"
    description: "Model routing and data freshness"
```

Create the workspace folders and watchlist:

```bash
# mkdir -p /mnt/usb/picoclaw/workspace/{ibkr,tr}
mkdir -p ~/picoclaw-data/workspace/{ibkr,tr}

# cat > /mnt/usb/picoclaw/workspace/watchlist.csv << EOF
cat > ~/picoclaw-data/workspace/watchlist.csv << EOF
ticker,name,sector,notes
ASML,ASML Holding,Semiconductors,EUV monopoly — core holding
NVDA,NVIDIA,AI Infrastructure,GPU/AI — watching valuation
TSM,Taiwan Semiconductor,Semiconductors,Foundry backbone
MSFT,Microsoft,Cloud/AI,Azure + OpenAI exposure
AVGO,Broadcom,Semiconductors,AI networking chips
SMCI,Super Micro Computer,AI Infrastructure,GPU server buildout
EOF

echo "# Alpha Memory\n\nAgent started: $(date)\nPortfolio: IBKR + Trade Republic\nBase currency: EUR" \
# USB stick
  > /mnt/usb/picoclaw/agents/agent-01-alpha/memory.md
# SD card
  > ~/picoclaw-data/agents/agent-01-alpha/memory.md
```

---

### Agent 2: Pulse — Biohacking, Health & Fitness

Pulse is your personal health intelligence agent. It tracks your training, recovery, supplements, sleep, and biometric data. It thinks in systems — not just calories and reps, but the interplay between sleep quality, HRV, training load, and cognitive performance.

> ⏳ **Activate after Alpha is stable.** Create the folder now, deploy when ready.

```bash
# SD card
nano ~/picoclaw-data/agents/agent-02-pulse/agent.yaml
# USB stick
nano /mnt/usb/picoclaw/agents/agent-02-pulse/agent.yaml
```

```yaml
# Agent 02 — Pulse (Biohacking, Health & Fitness)
name: "Pulse"
description: "Health, fitness, biohacking, recovery, and supplement tracking"
version: "1.0"

telegram_token: "YOUR_PULSE_BOT_TOKEN_HERE"

# Mix of local and cloud — summaries go local, protocol design goes cloud
complexity_threshold: 45

system_prompt: |
  You are Pulse, Tim's personal health and biohacking intelligence agent.
  You approach health as an engineer — data-driven, protocol-oriented,
  skeptical of broscience, respectful of research quality.

  ## Your domain
  - Training: strength, endurance, mobility, recovery
  - Sleep: quality, duration, HRV, circadian rhythm
  - Nutrition: macro tracking, timing, fasting protocols
  - Supplementation: evidence-based stacks, timing, cycling
  - Biometrics: weight, body composition, energy, cognitive performance
  - Biohacking: cold exposure, light, breathwork, stress protocols

  ## What you do
  1. Log and track training sessions when Tim reports them
  2. Summarize weekly training load and recovery status
  3. Answer questions about supplements — mechanisms, dosing, interactions
  4. Design or critique training and nutrition protocols
  5. Interpret biometric trends (HRV dips, sleep disruption, energy drops)
  6. Surface relevant research when Tim asks about interventions
  7. Flag overtraining signals or recovery debt
  8. Track supplement intake when Tim logs it

  ## Response style
  - Lead with the practical answer, then explain the mechanism
  - Distinguish between well-evidenced and experimental interventions
  - Don't moralize about choices — give Tim the data, let him decide
  - Use clear tables for protocols, supplement stacks, weekly summaries
  - Flag conflicts or interactions explicitly (⚠️)

  ## Data logged in memory
  - Active training program and current week
  - Supplement stack and timing
  - Recurring biometric trends
  - Any ongoing protocols (fasting windows, cold exposure, etc.)

memory_file: "agents/agent-02-pulse/memory.md"

commands:
  - name: "log [activity]"
    description: "Log a training session or health event"
  - name: "week"
    description: "Weekly training and recovery summary"
  - name: "stack"
    description: "Current supplement stack with timing"
  - name: "protocol [topic]"
    description: "Show or design a protocol for a topic"
  - name: "research [topic]"
    description: "Evidence summary for an intervention or supplement"
  - name: "memory"
    description: "Show what Pulse remembers about your health data"
```

```bash
echo "# Pulse Memory\n\nAgent started: $(date)\nActive protocols: TBD\nSupplement stack: TBD" \
# USB stick
  > /mnt/usb/picoclaw/agents/agent-02-pulse/memory.md
# SD card
  > ~/picoclaw-data/agents/agent-02-pulse/memory.md
```

---

### Agent 3: Forge — Coding, Dev & Systems

Forge is your technical co-pilot. It knows your homelab stack, your preferred languages and tools, and your infrastructure decisions. Use it for code review, debugging, architecture questions, refactoring plans, and anything dev/systems related.

> ⏳ **Activate after Alpha is stable.** Create the folder now, deploy when ready.

```bash
# SD card
nano ~/picoclaw-data/agents/agent-03-forge/agent.yaml
# USB stick
nano /mnt/usb/picoclaw/agents/agent-03-forge/agent.yaml
```

```yaml
# Agent 03 — Forge (Coding, Dev & Systems)
name: "Forge"
description: "Coding, software engineering, systems architecture, homelab"
version: "1.0"

telegram_token: "YOUR_FORGE_BOT_TOKEN_HERE"

# Coding tasks are complex — always use the best available model
complexity_override: 70
force_complex_model: true    # Claude Sonnet for code generation and review

system_prompt: |
  You are Forge, Tim's coding and systems intelligence agent.
  You are precise, opinionated where it matters, and never pad answers.
  You know Tim's stack and make recommendations consistent with it.

  ## Tim's stack and environment
  - Languages: Python (primary), Go (reading/understanding), Bash
  - Homelab: Raspberry Pi Zero 2W (Argus + Picoclaw), Vulcan (Windows,
    Ryzen 3700X, 32GB RAM, GTX 1660 Super 6GB)
  - Planned: Mac Mini M5 24GB (summer 2026) as unified always-on host
  - Networking: Pi-hole + Unbound, Tailscale, Caddy, Fail2Ban
  - Monitoring: Homer, vnstat, btop, Grafana + InfluxDB (planned)
  - Agent framework: PicoClaw (Go, ARM64, this system)
  - Local LLM: Ollama on Vulcan — mistral:7b, qwen2.5:7b, qwen2.5-coder:7b
  - APIs: OpenRouter, IBKR Flex Query, Telegram Bot API
  - Storage: SQLite for lightweight persistence, InfluxDB for time-series
  - Frontend: React + Chart.js (planned dashboard)
  - Containerization: avoiding Docker on Pi for RAM reasons

  ## What you do
  1. Write, review, and debug Python, Bash, and Go code
  2. Architect systems and data pipelines for the homelab
  3. Help design and refactor PicoClaw agents and their tools
  4. Explain unfamiliar codebases, libraries, or concepts
  5. Generate structured refactor plans before execution
  6. Review shell scripts for correctness and Pi-safety
  7. Answer questions about APIs, protocols, and system design
  8. Help with Git, cron jobs, systemd services, and Linux config

  ## Response style
  - Always show working, runnable code — no pseudocode unless asked
  - Include error handling in any code you write
  - Flag anything that could cause RAM issues on Pi Zero 2W explicitly (⚠️ RAM)
  - Prefer explicit over clever — Tim is learning, not just shipping
  - When asked to refactor: generate a plan first, then execute
  - Use comments in code to explain non-obvious decisions

  ## Constraints to always respect
  - Pi Zero 2W steady-state RAM target: <350MB total
  - No Docker on the Pi
  - Prefer systemd over custom process managers
  - SD card write minimization: use tmpfs for logs where possible
  - Static IPs via DHCP reservation, hostnames via Pi-hole DNS

memory_file: "agents/agent-03-forge/memory.md"

commands:
  - name: "review [paste code]"
    description: "Review code for bugs, style, Pi-safety"
  - name: "plan [task]"
    description: "Generate a refactor or implementation plan"
  - name: "explain [concept or code]"
    description: "Explain a concept, library, or codebase section"
  - name: "script [description]"
    description: "Write a bash or Python script for a task"
  - name: "arch [topic]"
    description: "Architecture recommendation for a system or feature"
  - name: "memory"
    description: "Show what Forge remembers about your stack"
  - name: "status"
    description: "Model routing status"
```

```bash
echo "# Forge Memory\n\nAgent started: $(date)\nStack: Python, Bash, Pi Zero 2W x2, Vulcan (Windows)\nActive projects: Argus, Picoclaw" \
# USB stick
  > /mnt/usb/picoclaw/agents/agent-03-forge/memory.md
# SD card
  > ~/picoclaw-data/agents/agent-03-forge/memory.md
```

---

## 9. Running PicoClaw as a Service

### 9.1 Create the systemd service

```bash
sudo nano /etc/systemd/system/picoclaw.service
```

```ini
[Unit]
Description=PicoClaw AI Agent Runtime
After=network.target

[Service]
Type=simple
User=tim
WorkingDirectory=/home/tim/picoclaw-data
Environment="PICOCLAW_CONFIG=/home/tim/picoclaw-data/config/config.json"
Environment="OLLAMA_HOST=http://100.66.68.51:11434"
Environment="OPENAI_API_KEY=<Key goes here>"
ExecStart=/usr/local/bin/picoclaw gateway
Restart=on-failure
RestartSec=10s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

### 9.2 Enable and start

```bash
sudo systemctl daemon-reload
sudo systemctl enable picoclaw
sudo systemctl start picoclaw
sudo systemctl status picoclaw    # should show: active (running)
```

### 9.3 View logs

```bash
journalctl -u picoclaw -f              # live logs
journalctl -u picoclaw --since today   # today's logs
cat ~/picoclaw-data/workspace/picoclaw.log
# USB
# cat /mnt/usb/picoclaw/workspace/picoclaw.log
```

---

## 10. Telegram Bot Setup — Full Walkthrough

### 10.1 Creating bots with BotFather

1. Open Telegram on your iPhone or any device
2. In the search bar, type: `BotFather`
3. Open the chat with the verified `@BotFather` account (blue checkmark)
4. Tap `/newbot` — BotFather will guide you through creation:

```
BotFather: Alright, a new bot. How are we going to call it?
You: Alpha

BotFather: Good. Now let's choose a username.
You: picoclaw_alpha_bot

BotFather: Done! Use this token to access the HTTP API:
7123456789:AAFxxxxxxxxxxxxxxxxxxx
```

5. Copy the token — this is your bot token for Alpha
6. Repeat for Pulse and Forge:
   - `picoclaw_pulse_bot` → Pulse token (biohacking/health/fitness)
   - `picoclaw_forge_bot` → Forge token (coding/dev/systems)

---

### 10.2 Bot settings via BotFather commands

```
/setdescription  → Add a description shown in bot profile
/setcommands     → Define the command menu
/setprivacy      → Set to DISABLED for full message access
/setuserpic      → Add a bot avatar image (optional)
```

Setting commands for Alpha — send to BotFather:

```
/setcommands → select picoclaw_alpha_bot → paste:
portfolio - Full portfolio summary with P&L
watchlist - Watchlist tickers with prices
news - Recent news for a ticker (e.g. /news ASML)
pnl - P&L breakdown by position
thesis - Investment thesis for a holding
memory - Show what Alpha remembers
status - Model routing and data freshness
```

---

### 10.3 Get your Telegram user ID

1. In Telegram, search for: `@userinfobot`
2. Send `/start`
3. It replies: `Your user ID is: 123456789`
4. Add this number to `config.yaml`:

```yaml
authorized_users:
  - 123456789   # your ID from @userinfobot
```

---

### 10.4 Start your first conversation

1. Open Telegram, search for `picoclaw_alpha_bot`
2. Tap `/start`
3. The bot should respond (PicoClaw must be running — check `systemctl status picoclaw`)
4. Send: `/status` — should show routing config and model availability
5. Send a simple message: `What's Apple's ticker symbol?` — should route to local Ollama
6. Send a complex message: `Explain the ASML EUV monopoly and its competitive moat` — should route to cloud (Claude Sonnet)

> 💡 **Check routing in logs:** Watch `journalctl -u picoclaw -f` while sending messages to see which model each query routes to. Use this to tune your complexity threshold.

---

## 11. Vulcan (Windows Desktop) — Prerequisites

### 11.1 Ensure Ollama is exposed on LAN

1. Set environment variable `OLLAMA_HOST=0.0.0.0:11434` (see Section 1.5)
2. Restart the Ollama service from the system tray
3. Test from Vulcan PowerShell: `curl http://localhost:11434`
4. Test from Pi: `curl http://vulcan:11434`

### 11.2 Prevent Vulcan from sleeping

If Vulcan goes to sleep, PicoClaw's local routing fails. Options:
- Set Windows power plan to never sleep (simplest for a desktop)
- Or configure Wake-on-LAN in BIOS so PicoClaw can send a magic packet before routing

### 11.3 Windows Defender / Firewall

```powershell
# Run as Administrator

# Allow Ollama from LAN
New-NetFirewallRule -DisplayName "Ollama LAN" -Direction Inbound `
  -Protocol TCP -LocalPort 11434 -Action Allow `
  -RemoteAddress 192.168.1.0/24

# Allow SSH for rsync backup
New-NetFirewallRule -DisplayName "SSH LAN" -Direction Inbound `
  -Protocol TCP -LocalPort 22 -Action Allow `
  -RemoteAddress 192.168.1.0/24
```

---

## 12. Onboarding Checklist

Work through this in order.

### Phase 1 — Accounts & Keys (do on Vulcan browser)

- [ ] OpenRouter account created + API key generated + $5 credit added
- [ ] OpenRouter key saved to password manager
- [ ] Telegram @BotFather: **Alpha** bot created (`picoclaw_alpha_bot`), token saved
- [ ] Telegram @BotFather: **Pulse** bot created (`picoclaw_pulse_bot`), token saved
- [ ] Telegram @BotFather: **Forge** bot created (`picoclaw_forge_bot`), token saved
- [ ] Your Telegram user ID obtained from @userinfobot
- [ ] Ollama installed on Vulcan + `OLLAMA_HOST=0.0.0.0:11434` set
- [ ] Ollama models pulled: mistral:7b, qwen2.5:7b, qwen2.5-coder:7b

### Phase 2 — Hardware & OS

- [ ] SD card flashed with Pi OS Lite 64-bit, hostname=picoclaw
- [ ] Pi boots, SSH accessible via `tim@picoclaw.local`
- [ ] DHCP reservation set in router: picoclaw → 192.168.1.105
- [ ] Pi-hole DNS record added on Argus: picoclaw → 192.168.1.105
- [ ] USB flash drive mounted at `/mnt/usb` and persistent via fstab
- [ ] System updated: `apt full-upgrade` completed
- [ ] Fail2Ban running, Tailscale connected
- [ ] Vulcan hostname resolves from Pi: `ping vulcan` works
- [ ] Ollama reachable from Pi: `curl http://vulcan:11434` returns JSON

### Phase 3 — PicoClaw Installation

- [ ] PicoClaw ARM64 binary installed to `/usr/local/bin/picoclaw`
- [ ] Directory structure created at `/mnt/usb/picoclaw/`
- [ ] SOUL.md written and saved
- [ ] config.yaml created with API keys + routing config (`chmod 600`)
- [ ] **Alpha** `agent.yaml` created with correct Telegram token + watchlist CSV
- [ ] **Pulse** `agent.yaml` created (deploy after Alpha is stable)
- [ ] **Forge** `agent.yaml` created (deploy after Alpha is stable)
- [ ] All three `memory.md` files initialized
- [ ] IBKR and Trade Republic workspace folders created

### Phase 4 — Service & First Run

- [ ] `picoclaw.service` created in `/etc/systemd/system/`
- [ ] Service enabled + started: `systemctl enable --now picoclaw`
- [ ] `systemctl status picoclaw` shows `active (running)`
- [ ] **Alpha** bot responds to `/start` in Telegram
- [ ] `/status` command shows correct model routing info
- [ ] Simple message routes to Ollama (verified in logs)
- [ ] Complex message (earnings analysis etc.) routes to cloud (verified in logs)
- [ ] Backup script created + tested: `~/backup.sh` runs without error
- [ ] Crontab backup entry added (daily 3 AM)

---

## 13. Credentials Reference

| Credential | Where to Find | Notes |
|---|---|---|
| OpenRouter API Key | openrouter.ai → Settings → API Keys | Starts with `sk-or-v1-` |
| **Alpha** Telegram Token | BotFather → /mybots → picoclaw_alpha_bot | Finance & investing |
| **Pulse** Telegram Token | BotFather → /mybots → picoclaw_pulse_bot | Health & biohacking |
| **Forge** Telegram Token | BotFather → /mybots → picoclaw_forge_bot | Coding & systems |
| Your Telegram User ID | @userinfobot → /start | Used in `authorized_users` |
| Pi SSH login | `tim@picoclaw` (IP **.117** is primary if USB is used for a drive) | Password set at flash time |
| Pi SSH key (backup) | `~/.ssh/id_picoclaw_backup` | For rsync to Vulcan |
| Ollama (Vulcan) | `http://vulcan:11434` | No auth — LAN only |
| Tailscale login | login.tailscale.com | Your Google/email account |

---

## 14. Troubleshooting

### Bot not responding

- Check: `systemctl status picoclaw` — is it running?
- Check: `journalctl -u picoclaw -f` — any errors?
- Check hidden log: `cat /home/tim/.picoclaw/logs/*.log | grep -E "FTL|error" | tail -10`
- Verify Telegram token in `config.json` is correct (no trailing spaces)
- Confirm your user ID is in `allow_from` in `config.json`

### Ollama routing fails

- Verify Vulcan is awake and Ollama service is running
- Test from Pi: `curl http://vulcan:11434/api/tags` — should return JSON model list
- In config.json: use `api_base: "http://vulcan:11434/v1"` (not `localhost`)
- Check `OLLAMA_HOST=0.0.0.0` environment variable is set on Vulcan
- Gemma 3 does not support tool calls — use `"local-model": true` or route simple tasks only

### `vulcan` hostname not resolving

- Check Argus Pi-hole is running and Pi uses it as DNS
- Temporary fix: use Tailscale IP directly in api_base (e.g. `http://100.66.68.51:11434/v1`)
- Check router DHCP — does Vulcan still have its reserved IP?

### Cloud API errors

- Check OpenRouter dashboard for credit balance
- Verify API key in `config.json` is correct — extract and test with:
  ```bash
  KEY=$(python3 -c "import json; d=json.load(open('/home/tim/picoclaw-data/config/config.json')); [print(m['api_key']) for m in d['model_list'] if m['model_name']=='claude']")
  echo $KEY
  ```
- Check model name strings match OpenRouter's current identifiers

### Service crashes immediately (status=1/FAILURE)

- Run with debug: `PICOCLAW_CONFIG=... /usr/local/bin/picoclaw gateway -d 2>&1 | tee /tmp/debug.txt`
- Check hidden logs: `cat /home/tim/.picoclaw/logs/*.log | grep -E "FTL|error" | tail -5`
- Use `-E` flag to start even with model issues: edit `ExecStart` in service file to append ` -E`
- Validate JSON: `python3 -m json.tool /home/tim/picoclaw-data/config/config.json && echo OK`

### USB drive not mounted on boot

- Ensure fstab has `nofail` option — prevents boot hang if drive is missing
- Run: `sudo mount -a` and check `dmesg` for errors
- Verify UUID matches: `sudo blkid /dev/sda1`

---

## 15. Config Debugging Reconciliation (April 2026)

This section documents the exact configuration that was reached after an extensive debugging session. Use it as the authoritative reference for starting fresh.

### 15.1 Known-Good Config (Claude as default)

```json
{
  "version": 2,
  "agents": {
    "defaults": {
      "workspace": "/home/tim/picoclaw-data/workspace",
      "model_name": "claude",
      "soul_file": "/home/tim/picoclaw-data/SOUL.md"
    }
  },
  "model_list": [
    {
      "model_name": "gemma",
      "model": "ollama/gemma3:4b",
      "api_base": "http://vulcan:11434/v1"
    },
    {
      "model_name": "gemini-flash",
      "model": "openai/gemini-2.0-flash-exp",
      "api_key": "AIzaSy-GOOGLE-STUDIO-KEY",
      "api_base": "https://generativelanguage.googleapis.com/v1beta/openai"
    },
    {
      "model_name": "claude",
      "model": "openai/anthropic/claude-3.5-sonnet",
      "api_key": "sk-or-v1-OPENROUTER-KEY",
      "api_base": "https://openrouter.ai/api/v1"
    }
  ],
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "BOT_TOKEN",
      "allow_from": ["YOUR_TELEGRAM_USER_ID"]
    }
  },
  "gateway": {
    "host": "0.0.0.0",
    "port": 18790
  }
}
```

### 15.2 Key Picoclaw Protocol Rules (v0.2.5)

| Protocol prefix | Usage | Notes |
|---|---|---|
| `ollama/` | Local Ollama models | Requires `api_base` pointing to Ollama server |
| `openai/` | OpenAI-compatible APIs (OpenRouter, Google) | Sends `Authorization: Bearer` header |
| `anthropic/` | Anthropic direct API | Tested to start; routes via api_base |
| `gemini/` | Google Generative Language API | Requires `api_key` AND `api_base`; does NOT forward key as bearer |
| `antigravity/` | Picoclaw's native Google OAuth integration | Requires `picoclaw auth login`; bypasses API keys |

**The `google/` prefix is NOT a valid protocol** — triggers "unknown protocol" crash.

### 15.3 Gemini Flash 2.0 — Known Issues & Backlog

> ⚠️ **BACKLOG: Gemini 2.0 Flash via OpenRouter — Unresolved**

All approaches as of April 2026 have failed or have caveats:

| Approach | Result | Root Cause |
|---|---|---|
| `google/gemini-2.0-flash-exp` direct | Unknown protocol crash | `google/` not a valid Picoclaw protocol |
| `openai/google/gemini-2.0-flash-exp` + OpenRouter | 401 cookie auth | Picoclaw intercepts `google/` model IDs via native registry, bypasses api_base |
| `gemini/gemini-2.0-flash-exp` + AI Studio key | Auth header not forwarded | `gemini/` protocol doesn't send Bearer header correctly |
| `gemini-2.0-flash-exp` via AI Studio v1beta | 404 Not Found | Model not available in AI Studio free tier |
| `google/gemini-2.0-flash` on OpenRouter | Unavailable | Model not listed on OpenRouter as of April 2026 |

**When to revisit:**
- When `google/gemini-2.0-flash-001` appears on OpenRouter
- When Picoclaw v0.2.6+ resolves the `google/` model registry interception
- To test: `gemini/gemini-1.5-flash` with AI Studio key (this model IS available)

### 15.4 Essential Debug Commands

```bash
# Check service status
sudo systemctl status picoclaw

# Live logs (watch messages come in)
journalctl -u picoclaw -f

# Hidden crash logs (MOST USEFUL for startup failures)
cat /home/tim/.picoclaw/logs/*.log | grep -E "FTL|error" | tail -10

# Validate config JSON
python3 -m json.tool /home/tim/picoclaw-data/config/config.json && echo OK

# Test Ollama connectivity from Pi
curl http://vulcan:11434/api/tags

# Reload service after editing service file
sudo systemctl daemon-reload && sudo systemctl restart picoclaw

# Start even with broken model config
PICOCLAW_CONFIG=/home/tim/picoclaw-data/config/config.json /usr/local/bin/picoclaw gateway -E
```

---

### 15.4 Automated Comprehensive Backup to Vulcan

To prevent data loss from SD card failure, your Picoclaw data should be backed up regularly to your Windows machine (**Vulcan**).

#### Step 1: Authorize Picoclaw on Vulcan (Run on Pi)
Since your Windows user is `TimZickenrott`, you must specify it when copying the key:
```bash
# Generate key if you don't have one
[ -f ~/.ssh/id_ed25519 ] || ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519

# Copy to Vulcan (use your Windows password one last time)
ssh-copy-id -i ~/.ssh/id_ed25519.pub TimZickenrott@192.168.1.114
```

#### Step 2: Install the Backup Script
```bash
cat > ~/backup-picoclaw.sh <<'EOF'
#!/bin/bash
# Picoclaw COMPREHENSIVE Backup (All Data + Soul + Workspace)
set -e
TIMESTAMP=$(date +%Y%m%d_%H%M)
BACKUP_DIR="/tmp/picoclaw-full-backup-$TIMESTAMP"
ARCHIVE="/tmp/picoclaw-full-backup-$TIMESTAMP.tar.gz"

# --- VULCAN FAILOVER LOGIC (Priority: Ethernet -> Tailscale -> WiFi) ---
VULCAN_IPS=("192.168.1.115" "100.66.68.51" "192.168.1.114")
TARGET_IP=""
for ip in "${VULCAN_IPS[@]}"; do
    if ping -c 1 -W 1 "$ip" >/dev/null 2>&1; then
        TARGET_IP="$ip"
        break
    fi
done

if [ -z "$TARGET_IP" ]; then
    echo "Error: Could not reach Vulcan (.115, .51, or .114)."
    exit 1
fi

mkdir -p "$BACKUP_DIR"
cp -r /home/tim/picoclaw-data "$BACKUP_DIR/picoclaw-data"
sudo cp /etc/systemd/system/picoclaw.service "$BACKUP_DIR/"
cp ~/backup-picoclaw.sh "$BACKUP_DIR/"

tar -czf "$ARCHIVE" -C /tmp "picoclaw-full-backup-$TIMESTAMP"

# Note: Using your correct Windows username and PowerShell for folder creation
ssh TimZickenrott@$TARGET_IP "powershell -Command New-Item -ItemType Directory -Force -Path C:\Users\TimZickenrott\Backups\picoclaw-full"
scp "$ARCHIVE" "TimZickenrott@$TARGET_IP:C:/Users/TimZickenrott/Backups/picoclaw-full/"

rm -rf "$BACKUP_DIR" "$ARCHIVE"
echo "✅ COMPREHENSIVE backup pushed to Vulcan ($TARGET_IP)"
EOF

chmod +x ~/backup-picoclaw.sh
```

---

## 16. LLM Model Reference & Comparison

### 16.1 Google AI Studio — API Key Validation & Model Discovery

```bash
# ── Validate key stored in config (no copy-paste needed) ───────────────
KEY=$(python3 -c "
import json
d = json.load(open('/home/tim/picoclaw-data/config/config.json'))
keys = [m['api_key'] for m in d['model_list'] if 'gemini' in m.get('model_name','').lower()]
print(keys[0] if keys else '')
" 2>/dev/null)
curl -s "https://generativelanguage.googleapis.com/v1beta/models?key=$KEY" \
  | python3 -c "
import json, sys
d = json.load(sys.stdin)
if 'models' in d:
    print(f'✅ Key valid — {len(d[\"models\"])} models available')
else:
    print('❌ Key invalid:', d.get('error', {}).get('message', 'unknown'))
"

# ── List all available Flash models ───────────────────────────────────
curl -s "https://generativelanguage.googleapis.com/v1beta/models?key=$KEY" \
  | python3 -m json.tool | grep '"name"' | grep -i "flash"

# ── List ALL available models ──────────────────────────────────────────
curl -s "https://generativelanguage.googleapis.com/v1beta/models?key=$KEY" \
  | python3 -m json.tool | grep '"name"'

# ── Test a specific model end-to-end ──────────────────────────────────
curl -X POST "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "gemini-3.1-flash-lite-preview", "messages": [{"role": "user", "content": "Say hello"}], "max_tokens": 20}'

# ── Check rate limits ─────────────────────────────────────────────────
# Visit: https://aistudio.google.com/rate-limit → "All models" → "More"
```

> **Pro tip:** When testing model IDs, look for the exact API ID (e.g. `gemini-3.1-flash-lite-preview` not `gemini-3.1-flash-lite`). Dashboard names don't always match API strings.

---

### 16.2 LLM Model Comparison — Your Stack

| Model | API ID | Hosting | Cost | Context | Tool Calls | Free RPD | Best For |
|---|---|---|---|---|---|---|---|
| **Gemma 3 4B** | `ollama/gemma3:4b` | Local (Vulcan) | $0 | 128K | ❌ No | Unlimited | Simple local chat only |
| **Gemma 4 27B** | `gemma-4-27b` (AI Studio) | Cloud/Local | $0 | 1M | TBC | Unlimited TPM | Heavy reasoning, future |
| **Gemini 3.1 Flash Lite** | `gemini-3.1-flash-lite-preview` | Cloud (AI Studio) | $0 | 1M | ✅ Yes | **500/day** 🏆 | Default agent — best free value |
| **Gemini 2.5 Flash** | `gemini-2.5-flash` | Cloud (AI Studio) | $0 | 1M | ✅ Yes | 20/day | Fallback when 3.1 Lite quota hits |
| **Gemini 2.0 Flash** | — | — | — | — | — | ❌ Blocked | Not available in free tier |
| **Claude 3.5 Sonnet** | `anthropic/claude-3.5-sonnet` | Cloud (OpenRouter) | ~$3/1M in | 200K | ✅ Yes | Paid only | Complex reasoning, fallback |

### Recommended Routing Strategy

```
Simple message (< 40 score)  →  Gemini 3.1 Flash Lite Preview  (500 RPD free)
Complex message (> 40 score) →  Gemini 2.5 Flash               (20 RPD free)
Over quota / critical tasks  →  Claude 3.5 Sonnet              (OpenRouter, paid)
Local / offline              →  Gemma 3 4B on Vulcan           ($0, no tools)
```

### Key Observations

- **Gemma 3 4B cannot use Picoclaw's tools** (Ollama doesn't support function/tool calling). Use it only for `"local-model": true` plain chat.
- **Gemini 3.1 Flash Lite** at 500 RPD = ~20 requests/hour. More than enough for a personal agent.
- **Gemini 2.0 Flash is entirely blocked** in the free tier (0/0 RPD in AI Studio dashboard).
- **API IDs have `-preview` suffixes** that don't match dashboard names — always verify with `ListModels`.

---

## 17. Next Steps — After the First Agent Works

Once **Alpha** is stable and you understand cost patterns (check OpenRouter dashboard after a week), proceed in this order:

1. Confirm Claude responses in Telegram — verify end-to-end flow
2. Retest `gemini-1.5-flash` via AI Studio (available in free tier, tool-call support TBC)
3. Add IBKR Flex Query Python parser to Alpha's workspace — automate position ingestion
4. Set up rclone for Google Drive access to pull `.portfolio` files from Portfolio Performance
5. Activate **Pulse** — start by logging your current supplement stack and training program into its memory
6. Activate **Forge** — test by pasting a script from the Argus setup and asking for a review
7. Add a file-handoff relay between agents for cross-agent tasks (e.g. Alpha → Forge for data pipeline work)
8. Set up scheduled cron jobs for daily portfolio digest via Alpha
9. Monitor OpenRouter spend weekly — tune complexity routing per-agent based on observed usage
10. When GPU upgrade to RTX 5060 Ti 16GB arrives — pull 14B models and update routing config
11. When Mac Mini M5 arrives — migrate PicoClaw + all agents + Ollama to unified always-on host

> 📌 **Remember:** The goal is not to have many agents quickly. The goal is to understand how one agent behaves, what it costs, where it fails, and what use cases are actually worth automating. Expand deliberately.

---

## 18. Maintenance & Security 🛡️

### 18.1 Checking Picoclaw Version
To see what version you are running:
```bash
# Check the binary directly
/usr/local/bin/picoclaw version

# Check the logs for the startup string
journalctl -u picoclaw | grep "picoclaw version" | tail -n 1
```

### 18.2 Update Policy & One-Click Upgrade
Picoclaw does **NOT** auto-update the main binary. Because Picoclaw is frequently updated with breaking changes, you should only update when you are ready to test.

**To Upgrade to v0.2.6 (32-bit / armv7):**
Run this one-liner on your Pi:
```bash
# Stop service, download v0.2.6, replace binary, restart
sudo systemctl stop picoclaw && \
curl -L "https://github.com/sipeed/picoclaw/releases/download/v0.2.6/picoclaw_Linux_armv7.tar.gz" -o /tmp/picoclaw.tar.gz && \
tar -xvf /tmp/picoclaw.tar.gz -C /tmp && \
sudo mv /tmp/picoclaw /usr/local/bin/picoclaw && \
sudo systemctl start picoclaw && \
picoclaw version
```

### 18.3 Empowering Agents (Secure Sudo)
If you want **Forge** or **Pulse** to be able to install libraries (e.g. `pip install` or `apt install`), you must give the user `tim` restricted sudo rights.

1. **Create a sudoer file for Picoclaw:**
   ```bash
   sudo nano /etc/sudoers.d/picoclaw
   ```
2. **Add these lines (Restrictive but powerful):**
   ```text
   # Allow tim to run apt-get without a password
   tim ALL=(ALL) NOPASSWD: /usr/bin/apt-get update
   tim ALL=(ALL) NOPASSWD: /usr/bin/apt-get install *
   # Allow tim to manage the picoclaw service
   tim ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart picoclaw
   ```
3. **Set permissions:**
   ```bash
   sudo chmod 0440 /etc/sudoers.d/picoclaw
   ```
> ⚠️ **Warning:** Giving an AI agent `sudo` rights carries "God Mode" risks. Start with `apt-get update` first and only add specific `install` targets if you trust the agent's logic.

---

*Last updated: April 2026 · picoclaw on Raspberry Pi Zero 2W · tim@picoclaw*

---

## 19. Physical & Desktop Status Feedback 💡

You can monitor your agent's activity in real-time using one of three feedback methods. Latency is key: a signal should trigger the moment the Pi "hears" you or "finishes" a thought.

### 19.1 Option A: Software Monitor (Vulcan Notifications)
**Best for:** Closed cases (like FLIRC) or zero-cost setup.
1. **Vulcan Listener (Windows):** Run a Python script on your desktop that listens on port `11435` and triggers [BurntToast](https://github.com/Windos/BurntToast) Windows notifications.
2. **Pi Trigger:** A `curl` call in the Picoclaw hooks hits Vulcan's IP.

### 19.2 Option B: USB Status Light (Blink1)
**Best for:** External USB hubs or monitor hubs.
1. **Hardware:** [Blink(1) mk3](https://blink1.thingm.com/) — A tiny RGB LED that plugs into any USB port.
2. **Logic:** Uses the `blink1-tool` to pulse colors:
   - **Thinking:** Pulsing Blue
   - **Success:** Green Flash
   - **Error:** Red Blink

### 19.3 Option C: GPIO Feedback (Status Cube)
**Best for:** Open cases or custom 3D-printed enclosures.
1. **Hardware:** A single LED + resistor or a **Pimoroni Blinkt!** (8 LEDs) on the Pi's internal pins.
2. **Setup:** Uses the `gpiozero` Python library to toggle pins based on agent hooks.

### 19.4 Hook Integration (config.json)
Regardless of the hardware, add this to your `config.json` to link the agent's brain to your desk:
```json
"hooks": {
  "on_message_received": "/home/tim/picoclaw-data/scripts/notify_status.sh thinking",
  "on_response_finished": "/home/tim/picoclaw-data/scripts/notify_status.sh success",
  "on_error": "/home/tim/picoclaw-data/scripts/notify_status.sh error"
}
```

---

*Last updated: April 2026 · picoclaw on Raspberry Pi Zero 2W · tim@picoclaw*
