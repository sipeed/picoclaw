<div align="center">
  <img src="assets/logo.webp" alt="PicoClaw" width="512">

  <h1>PicoClaw: Ultra-Efficient AI Assistant in Go</h1>

  <h3>$10 Hardware · <10MB RAM · <1s Boot · 皮皮虾，我们走！</h3>
  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20MIPS%2C%20RISC--V%2C%20LoongArch-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://picoclaw.io"><img src="https://img.shields.io/badge/Website-picoclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://docs.picoclaw.io/"><img src="https://img.shields.io/badge/Docs-Official-007acc?style=flat&logo=read-the-docs&logoColor=white" alt="Docs"></a>
    <a href="https://deepwiki.com/sipeed/picoclaw"><img src="https://img.shields.io/badge/Wiki-DeepWiki-FFA500?style=flat&logo=wikipedia&logoColor=white" alt="Wiki"></a>
    <br>
    <a href="https://x.com/SipeedIO"><img src="https://img.shields.io/badge/X_(Twitter)-SipeedIO-black?style=flat&logo=x&logoColor=white" alt="Twitter"></a>
    <a href="./assets/wechat.png"><img src="https://img.shields.io/badge/WeChat-Group-41d56b?style=flat&logo=wechat&logoColor=white"></a>
    <a href="https://discord.gg/V4sAZ9XWpN"><img src="https://img.shields.io/badge/Discord-Community-4c60eb?style=flat&logo=discord&logoColor=white" alt="Discord"></a>
  </p>

[中文](README.zh.md) | [日本語](README.ja.md) | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | [Français](README.fr.md) | **English**

</div>

---

> **PicoClaw** is an independent open-source project initiated by [Sipeed](https://sipeed.com). It is written entirely in **Go** — not a fork of OpenClaw, NanoBot, or any other project.

🦐 PicoClaw is an ultra-lightweight personal AI Assistant inspired by [NanoBot](https://github.com/HKUDS/nanobot), refactored from the ground up in Go through a self-bootstrapping process, where the AI agent itself drove the entire architectural migration and code optimization.

⚡️ Runs on $10 hardware with <10MB RAM: That's 99% less memory than OpenClaw and 98% cheaper than a Mac mini!

<table align="center">
  <tr align="center">
    <td align="center" valign="top">
      <p align="center">
        <img src="assets/picoclaw_mem.gif" width="360" height="240">
      </p>
    </td>
    <td align="center" valign="top">
      <p align="center">
        <img src="assets/licheervnano.png" width="400" height="240">
      </p>
    </td>
  </tr>
</table>

> [!CAUTION]
> **🚨 SECURITY & OFFICIAL CHANNELS / 安全声明**
>
> * **NO CRYPTO:** PicoClaw has **NO** official token/coin. All claims on `pump.fun` or other trading platforms are **SCAMS**.
>
> * **OFFICIAL DOMAIN:** The **ONLY** official website is **[picoclaw.io](https://picoclaw.io)**, and company website is **[sipeed.com](https://sipeed.com)**
> * **Warning:** Many `.ai/.org/.com/.net/...` domains are registered by third parties.
> * **Warning:** picoclaw is in early development now and may have unresolved network security issues. Do not deploy to production environments before the v1.0 release.
> * **Note:** picoclaw has recently merged a lot of PRs, which may result in a larger memory footprint (10–20MB) in the latest versions. We plan to prioritize resource optimization as soon as the current feature set reaches a stable state.

## 📢 News

2026-03-17 🚀 **v0.2.3 Released!** System tray UI (Windows & Linux), sub-agent status tracking (`spawn_status`), experimental gateway hot-reload, cron security gates, and 2 security fixes. PicoClaw now at **25K ⭐**!

2026-03-09 🎉 **v0.2.1 — Biggest update yet!** MCP protocol support, 4 new channels (Matrix/IRC/WeCom/Discord Proxy), 3 new providers (Kimi/Minimax/Avian), vision pipeline, JSONL memory store, and model routing.

2026-02-28 📦 **v0.2.0** released with Docker Compose support and Web UI launcher.

2026-02-26 🎉 PicoClaw hit **20K stars** in just 17 days! Channel auto-orchestration and capability interfaces landed.

<details>
<summary>Older news...</summary>

2026-02-16 🎉 PicoClaw hit 12K stars in one week! Community maintainer roles and [roadmap](ROADMAP.md) officially posted.

2026-02-13 🎉 PicoClaw hit 5000 stars in 4 days! Project Roadmap and Developer Group setup underway.

2026-02-09 🎉 **PicoClaw Launched!** Built in 1 day to bring AI Agents to $10 hardware with <10MB RAM. 🦐 PicoClaw，Let's Go！

</details>

## ✨ Features

🪶 **Ultra-Lightweight**: <10MB Memory footprint — 99% smaller than OpenClaw core functionality.*

💰 **Minimal Cost**: Efficient enough to run on $10 Hardware — 98% cheaper than a Mac mini.

⚡️ **Lightning Fast**: 400X Faster startup time, boot in <1 second even on 0.6GHz single core.

🌍 **True Portability**: Single self-contained binary across RISC-V, ARM, MIPS, and x86, One-click to Go!

🤖 **AI-Bootstrapped**: Autonomous Go-native implementation — 95% Agent-generated core with human-in-the-loop refinement.

🔌 **MCP Support**: Native [Model Context Protocol](https://modelcontextprotocol.io/) integration — connect any MCP server to extend agent capabilities.

👁️ **Vision Pipeline**: Send images and files directly to the agent — automatic base64 encoding for multimodal LLMs.

🧠 **Smart Routing**: Rule-based model routing — simple queries go to lightweight models, saving API costs.

_*Recent versions may use 10–20MB due to rapid feature merges. Resource optimization is planned. Startup comparison based on 0.8GHz single-core benchmarks (see table below)._

|                               | OpenClaw      | NanoBot                  | **PicoClaw**                              |
| ----------------------------- | ------------- | ------------------------ | ----------------------------------------- |
| **Language**                  | TypeScript    | Python                   | **Go**                                    |
| **RAM**                       | >1GB          | >100MB                   | **< 10MB***                               |
| **Startup**</br>(0.8GHz core) | >500s         | >30s                     | **<1s**                                   |
| **Cost**                      | Mac Mini $599 | Most Linux SBC </br>~$50 | **Any Linux Board**</br>**As low as $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

## 🦾 Demonstration

### 🛠️ Standard Assistant Workflows

<table align="center">
  <tr align="center">
    <th><p align="center">🧩 Full-Stack Engineer</p></th>
    <th><p align="center">🗂️ Logging & Planning Management</p></th>
    <th><p align="center">🔎 Web Search & Learning</p></th>
  </tr>
  <tr>
    <td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
    <td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
  </tr>
  <tr>
    <td align="center">Develop • Deploy • Scale</td>
    <td align="center">Schedule • Automate • Memory</td>
    <td align="center">Discovery • Insights • Trends</td>
  </tr>
</table>

### 📱 Run on old Android Phones

Give your decade-old phone a second life! Turn it into a smart AI Assistant with PicoClaw. Quick Start:

1. **Install [Termux](https://github.com/termux/termux-app)** (Download from [GitHub Releases](https://github.com/termux/termux-app/releases), or search in F-Droid / Google Play).
2. **Execute cmds**

```bash
# Download the latest release from https://github.com/sipeed/picoclaw/releases
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard
```

And then follow the instructions in the "Quick Start" section to complete the configuration!

<img src="assets/termux.jpg" alt="PicoClaw" width="512">

### 🐜 Innovative Low-Footprint Deploy

PicoClaw can be deployed on almost any Linux device!

- $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) E(Ethernet) or W(WiFi6) version, for Minimal Home Assistant
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html), or $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html) for Automated Server Maintenance
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) or $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera) for Smart Monitoring

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 More Deployment Cases Await！

## 📦 Install

### Install with precompiled binary

Download the binary for your platform from the [Releases](https://github.com/sipeed/picoclaw/releases) page.

### Install from source (latest features, recommended for development)

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# Build, no need to install
make build

# Build for multiple platforms
make build-all

# Build for Raspberry Pi Zero 2 W (32-bit: make build-linux-arm; 64-bit: make build-linux-arm64)
make build-pi-zero

# Build And Install
make install
```

**Raspberry Pi Zero 2 W:** Use the binary that matches your OS: 32-bit Raspberry Pi OS → `make build-linux-arm`; 64-bit → `make build-linux-arm64`. Or run `make build-pi-zero` to build both.

## 📚 Documentation

For detailed guides, see the docs below. The README covers quick start only.

| Topic | Description |
|-------|-------------|
| 🐳 [Docker & Quick Start](docs/docker.md) | Docker Compose setup, Launcher/Agent modes, Quick Start configuration |
| 💬 [Chat Apps](docs/chat-apps.md) | Telegram, Discord, WhatsApp, Matrix, QQ, Slack, IRC, DingTalk, LINE, Feishu, WeCom, and more |
| ⚙️ [Configuration](docs/configuration.md) | Environment variables, workspace layout, skill sources, security sandbox, heartbeat |
| 🔌 [Providers & Models](docs/providers.md) | 20+ LLM providers, model routing, model_list configuration, provider architecture |
| 🔄 [Spawn & Async Tasks](docs/spawn-tasks.md) | Quick tasks, long tasks with spawn, async sub-agent orchestration |
| 🐛 [Troubleshooting](docs/troubleshooting.md) | Common issues and solutions |
| 🔧 [Tools Configuration](docs/tools_configuration.md) | Per-tool enable/disable, exec policies |

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Join the Agent Social Network

Connect Picoclaw to the Agent Social Network simply by sending a single message via the CLI or any integrated Chat App.

**Read `https://clawdchat.ai/skill.md` and follow the instructions to join [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ CLI Reference

| Command                   | Description                   |
| ------------------------- | ----------------------------- |
| `picoclaw onboard`        | Initialize config & workspace |
| `picoclaw agent -m "..."` | Chat with the agent           |
| `picoclaw agent`          | Interactive chat mode         |
| `picoclaw gateway`        | Start the gateway             |
| `picoclaw status`         | Show status                   |
| `picoclaw version`        | Show version info             |
| `picoclaw cron list`      | List all scheduled jobs       |
| `picoclaw cron add ...`   | Add a scheduled job           |
| `picoclaw cron disable`   | Disable a scheduled job       |
| `picoclaw cron remove`    | Remove a scheduled job        |
| `picoclaw skills list`    | List installed skills         |
| `picoclaw skills install` | Install a skill               |
| `picoclaw migrate`        | Migrate data from older versions |
| `picoclaw auth login`     | Authenticate with providers   |

### Scheduled Tasks / Reminders

PicoClaw supports scheduled reminders and recurring tasks through the `cron` tool:

* **One-time reminders**: "Remind me in 10 minutes" → triggers once after 10min
* **Recurring tasks**: "Remind me every 2 hours" → triggers every 2 hours
* **Cron expressions**: "Remind me at 9am daily" → uses cron expression

## 🤝 Contribute & Roadmap

PRs welcome! The codebase is intentionally small and readable. 🤗

See our full [Community Roadmap](https://github.com/sipeed/picoclaw/blob/main/ROADMAP.md).

Developer group building, join after your first merged PR!

User Groups:

discord: <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">
