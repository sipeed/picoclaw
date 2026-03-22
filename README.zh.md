<div align="center">
<img src="assets/logo.webp" alt="PicoClaw" width="512">

<h1>PicoClaw: 基于Go语言的超高效 AI 助手</h1>

<h3>$10 硬件 · <10MB 内存 · <1s 启动 · 皮皮虾，我们走！</h3>
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

**中文** | [日本語](README.ja.md) | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | [Français](README.fr.md) | [Italiano](README.it.md) | [Bahasa Indonesia](README.id.md) | [English](README.md)

</div>

---

> **PicoClaw** 是由 [矽速科技 (Sipeed)](https://sipeed.com) 发起的独立开源项目，完全使用 **Go 语言**从零编写——不是 OpenClaw、NanoBot 或其他项目的分支。

🦐 **PicoClaw** 是一个受 [NanoBot](https://github.com/HKUDS/nanobot) 启发的超轻量级个人 AI 助手。它采用 **Go 语言** 从零重构，经历了一个"自举"过程——即由 AI Agent 自身驱动了整个架构迁移和代码优化。

⚡️ **极致轻量**：可在 **10 美元** 的硬件上运行，内存占用 **<10MB**。这意味着比 OpenClaw 节省 99% 的内存，比 Mac mini 便宜 98%！

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
> **🚨 安全声明**
>
> - **无加密货币 (NO CRYPTO):** PicoClaw **没有** 发行任何官方代币、Token 或虚拟货币。所有在 `pump.fun` 或其他交易平台上的相关声称均为 **诈骗**。
> - **官方域名:** 唯一的官方网站是 **[picoclaw.io](https://picoclaw.io)**，公司官网是 **[sipeed.com](https://sipeed.com)**。
> - **警惕:** 许多 `.ai/.org/.com/.net/...` 后缀的域名被第三方抢注，请勿轻信。
> - **注意:** PicoClaw 正在初期的快速功能开发阶段，可能有尚未修复的网络安全问题，在 1.0 正式版发布前，请不要将其部署到生产环境中。
> - **注意:** PicoClaw 最近合并了大量 PR，近期版本可能内存占用较大 (10~20MB)，我们将在功能较为收敛后进行资源占用优化。

## 📢 新闻

2026-03-17 🚀 **v0.2.3 发布！** 系统托盘 UI（Windows & Linux）、子 Agent 状态查询 (`spawn_status`)、实验性 Gateway 热重载、Cron 安全门控，以及 2 项安全修复。PicoClaw 已达 **25K ⭐**！

2026-03-09 🎉 **v0.2.1 — 史上最大更新！** MCP 协议支持、4 个新频道 (Matrix/IRC/WeCom/Discord Proxy)、3 个新 Provider (Kimi/Minimax/Avian)、视觉管线、JSONL 记忆存储、模型路由。

2026-02-28 📦 **v0.2.0** 发布，支持 Docker Compose 和 Web UI 启动器。

2026-02-26 🎉 PicoClaw 仅 17 天突破 **20K Stars**！频道自动编排和能力接口上线。

<details>
<summary>更早的新闻...</summary>

2026-02-16 🎉 PicoClaw 一周内突破 12K Stars！社区维护者角色和 [路线图](ROADMAP.md) 正式发布。

2026-02-13 🎉 PicoClaw 4 天内突破 5000 Stars！项目路线图和开发者群组筹建中。

2026-02-09 🎉 **PicoClaw 正式发布！** 仅用 1 天构建，将 AI Agent 带入 $10 硬件与 <10MB 内存的世界。🦐 皮皮虾，我们走！

</details>

## ✨ 特性

🪶 **超轻量级**: 核心功能内存占用 <10MB — 比 OpenClaw 小 99%。*

💰 **极低成本**: 高效到足以在 $10 的硬件上运行 — 比 Mac mini 便宜 98%。

⚡️ **闪电启动**: 启动速度快 400 倍，即使在 0.6GHz 单核处理器上也能在 1 秒内启动。

🌍 **真正可移植**: 跨 RISC-V、ARM、MIPS 和 x86 架构的单二进制文件，一键运行！

🤖 **AI 自举**: 纯 Go 语言原生实现 — 95% 的核心代码由 Agent 生成，并经由"人机回环"微调。

🔌 **MCP 支持**: 原生 [Model Context Protocol](https://modelcontextprotocol.io/) 集成 — 连接任意 MCP 服务器扩展 Agent 能力。

👁️ **视觉管线**: 直接向 Agent 发送图片和文件 — 自动 base64 编码对接多模态 LLM。

🧠 **智能路由**: 基于规则的模型路由 — 简单查询走轻量模型，节省 API 成本。

_*近期版本因快速合并 PR 可能占用 10–20MB，资源优化已列入计划。启动速度对比基于 0.8GHz 单核实测（见下方对比表）。_

|                                | OpenClaw      | NanoBot                  | **PicoClaw**                           |
| ------------------------------ | ------------- | ------------------------ | -------------------------------------- |
| **语言**                       | TypeScript    | Python                   | **Go**                                 |
| **RAM**                        | >1GB          | >100MB                   | **< 10MB***                            |
| **启动时间**</br>(0.8GHz core) | >500s         | >30s                     | **<1s**                                |
| **成本**                       | Mac Mini $599 | 大多数 Linux 开发板 ~$50 | **任意 Linux 开发板**</br>**低至 $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

> 📋 **[硬件兼容列表](docs/hardware-compatibility.md)** — 查看所有已测试的板卡，从 $5 RISC-V 到树莓派到安卓手机。你的板卡没在列表中？欢迎提交 PR！

## 🦾 演示

### 🛠️ 标准助手工作流

<table align="center">
<tr align="center">
<th><p align="center">🧩 全栈工程师模式</p></th>
<th><p align="center">🗂️ 日志与规划管理</p></th>
<th><p align="center">🔎 网络搜索与学习</p></th>
</tr>
<tr>
<td align="center"><p align="center"><img src="assets/picoclaw_code.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_memory.gif" width="240" height="180"></p></td>
<td align="center"><p align="center"><img src="assets/picoclaw_search.gif" width="240" height="180"></p></td>
</tr>
<tr>
<td align="center">开发 • 部署 • 扩展</td>
<td align="center">日程 • 自动化 • 记忆</td>
<td align="center">发现 • 洞察 • 趋势</td>
</tr>
</table>

### 📱 在手机上轻松运行

PicoClaw 可以将你 10 年前的老旧手机废物利用，变身成为你的 AI 助理！快速指南：

1. 安装 [Termux](https://github.com/termux/termux-app)（可从 [GitHub Releases](https://github.com/termux/termux-app/releases) 下载，或在 F-Droid 等应用商店搜索）
2. 打开后执行指令

```bash
# 从 Release 页面下载最新版本
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard   # chroot 提供标准 Linux 文件系统布局
```

然后跟随下面的"快速开始"章节继续配置 PicoClaw 即可使用！

<img src="assets/termux.jpg" alt="PicoClaw" width="512">

### 🐜 创新的低占用部署

PicoClaw 几乎可以部署在任何 Linux 设备上！

- $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) E(网口) 或 W(WiFi6) 版本，用于极简家庭助手
- $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html)，或 $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html)，用于自动化服务器运维
- $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) 或 $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera)，用于智能监控

<https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4>

🌟 更多部署案例敬请期待！

## 📦 安装

### 从 picoclaw.io 下载（推荐）

访问 **[picoclaw.io](https://picoclaw.io)** — 官网自动检测你的平台，提供一键下载，无需手动选择架构。

### 下载预编译二进制文件

也可以从 [GitHub Releases](https://github.com/sipeed/picoclaw/releases) 页面手动下载对应平台的二进制文件。

### 从源码构建（开发用）

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# 构建（无需安装）
make build

# 为多平台构建
make build-all

# 为 Raspberry Pi Zero 2 W 构建（32位: make build-linux-arm; 64位: make build-linux-arm64）
make build-pi-zero

# 构建并安装
make install
```

**Raspberry Pi Zero 2 W:** 请使用与系统匹配的二进制文件：32 位 Raspberry Pi OS → `make build-linux-arm`；64 位 → `make build-linux-arm64`。或运行 `make build-pi-zero` 同时构建两者。

## 📚 文档

详细指南请参阅以下文档，README 仅涵盖快速入门。

| 主题 | 说明 |
|------|------|
| 🐳 [Docker 与快速开始](docs/zh/docker.md) | Docker Compose 配置、Launcher/Agent 模式、快速开始 |
| 💬 [聊天应用配置](docs/zh/chat-apps.md) | Telegram、Discord、WhatsApp、Matrix、QQ、Slack、IRC、钉钉、LINE、飞书、企业微信等 |
| ⚙️ [配置指南](docs/zh/configuration.md) | 环境变量、工作区布局、技能来源、安全沙箱、心跳任务 |
| 🔌 [提供商与模型配置](docs/zh/providers.md) | 20+ LLM 提供商、模型路由、model_list 配置、Provider 架构 |
| 🔄 [异步任务与 Spawn](docs/zh/spawn-tasks.md) | 快速任务、长任务与 Spawn、异步子 Agent 编排 |
| 🐛 [疑难解答](docs/zh/troubleshooting.md) | 常见问题与解决方案 |
| 🔧 [工具配置](docs/zh/tools_configuration.md) | 工具启用/禁用、执行策略 |
| 📋 [硬件兼容列表](docs/hardware-compatibility.md) | 已测试板卡、最低要求、如何添加你的板卡 |

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> 加入 Agent 社交网络

通过 CLI 或任何已集成的聊天应用发送一条消息，即可将 PicoClaw 连接到 Agent 社交网络。

**阅读 `https://clawdchat.ai/skill.md` 并按照说明加入 [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ CLI 命令行参考

| 命令                       | 说明                   |
| ------------------------- | ---------------------- |
| `picoclaw onboard`        | 初始化配置与工作区       |
| `picoclaw onboard weixin` | 扫码连接微信个人号       |
| `picoclaw agent -m "..."` | 与 Agent 对话           |
| `picoclaw agent`          | 交互式对话模式           |
| `picoclaw gateway`        | 启动网关                |
| `picoclaw status`         | 查看状态                |
| `picoclaw version`        | 查看版本信息             |
| `picoclaw cron list`      | 列出所有定时任务         |
| `picoclaw cron add ...`   | 添加定时任务             |
| `picoclaw cron disable`   | 禁用定时任务             |
| `picoclaw cron remove`    | 删除定时任务             |
| `picoclaw skills list`    | 列出已安装技能           |
| `picoclaw skills install` | 安装技能                |
| `picoclaw migrate`        | 从旧版本迁移数据         |
| `picoclaw auth login`     | 认证提供商               |
| `picoclaw model`          | 查看或切换默认模型 |

### 定时任务 / 提醒

PicoClaw 通过 `cron` 工具支持定时提醒和重复任务：

* **一次性提醒**: "10分钟后提醒我" → 10分钟后触发一次
* **重复任务**: "每2小时提醒我" → 每2小时触发
* **Cron 表达式**: "每天上午9点提醒我" → 使用 cron 表达式

## 🤝 贡献与路线图

欢迎提交 PR！代码库刻意保持小巧和可读。🤗

查看完整的 [社区路线图](https://github.com/sipeed/picoclaw/blob/main/ROADMAP.md)。

开发者群组正在组建中，入群门槛：至少合并过 1 个 PR。

用户群组：

Discord: <https://discord.gg/V4sAZ9XWpN>

<img src="assets/wechat.png" alt="PicoClaw" width="512">
