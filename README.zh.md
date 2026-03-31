<div align="center">
<img src="assets/logo.webp" alt="PicoClaw" width="512">

<h1>PicoClaw: 基于Go语言的超高效 AI 助手</h1>

<h3>$10 硬件 · 10MB 内存 · 毫秒启动 · 皮皮虾，我们走！</h3>
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

**中文** | [日本語](README.ja.md) | [Português](README.pt-br.md) | [Tiếng Việt](README.vi.md) | [Français](README.fr.md) | [Italiano](README.it.md) | [Bahasa Indonesia](README.id.md) | [Malay](README.my.md) | [English](README.md)

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

2026-03-25 🚀 **v0.2.4 发布！** Agent 架构全面重构（SubTurn、Hook、Steering、EventBus）、微信/企业微信深度集成、安全体系升级（.security.yml、敏感数据过滤）、新增 Provider（AWS Bedrock、Azure、小米 MiMo），以及 35 项 Bug 修复。PicoClaw 已达 **26K ⭐**！

2026-03-17 🚀 **v0.2.3 发布！** 系统托盘 UI（Windows & Linux）、子 Agent 状态查询 (`spawn_status`)、实验性 Gateway 热重载、Cron 安全门控，以及 2 项安全修复。PicoClaw 已达 **25K ⭐**！

2026-03-09 🎉 **v0.2.1 — 史上最大更新！** MCP 协议支持、4 个新频道 (Matrix/IRC/WeCom/Discord Proxy)、3 个新 Provider (Kimi/Minimax/Avian)、视觉管线、JSONL 记忆存储、模型路由。

2026-02-28 📦 **v0.2.0** 发布，支持 Docker Compose 和 Web UI 启动器。

<details>
<summary>更早的新闻...</summary>

2026-02-26 🎉 PicoClaw 仅 17 天突破 **20K Stars**！频道自动编排和能力接口上线。

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

<div align="center">

|                                | OpenClaw      | NanoBot                  | **PicoClaw**                           |
| ------------------------------ | ------------- | ------------------------ | -------------------------------------- |
| **语言**                       | TypeScript    | Python                   | **Go**                                 |
| **RAM**                        | >1GB          | >100MB                   | **< 10MB***                            |
| **启动时间**</br>(0.8GHz core) | >500s         | >30s                     | **<1s**                                |
| **成本**                       | Mac Mini $599 | 大多数 Linux 开发板 ~$50 | **任意 Linux 开发板**</br>**低至 $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

</div>

> 📋 **[硬件兼容列表](docs/zh/hardware-compatibility.md)** — 查看所有已测试的板卡，从 $5 RISC-V 到树莓派到安卓手机。你的板卡没在列表中？欢迎提交 PR！

<p align="center">
<img src="assets/hardware-banner.jpg" alt="PicoClaw Hardware Compatibility" width="100%">
</p>

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

# 构建核心二进制文件
make build

# 构建 Web UI Launcher（WebUI 模式必需）
make build-launcher

# 为多平台构建
make build-all

# 为 Raspberry Pi Zero 2 W 构建（32位: make build-linux-arm; 64位: make build-linux-arm64）
make build-pi-zero

# 构建并安装
make install
```

**Raspberry Pi Zero 2 W:** 请使用与系统匹配的二进制文件：32 位 Raspberry Pi OS → `make build-linux-arm`；64 位 → `make build-linux-arm64`。或运行 `make build-pi-zero` 同时构建两者。

## 🚀 快速开始

### 🌐 WebUI Launcher（推荐桌面用户）

WebUI Launcher 提供基于浏览器的配置与聊天界面，是最简单的上手方式——无需命令行知识。

**方式一：双击启动（桌面）**

从 [picoclaw.io](https://picoclaw.io) 下载后，双击 `picoclaw-launcher`（Windows 上为 `picoclaw-launcher.exe`），浏览器将自动打开 `http://localhost:18800`。

**方式二：命令行**

```bash
picoclaw-launcher
# 在浏览器中打开 http://localhost:18800
```

> [!TIP]
> **远程访问 / Docker / 虚拟机：** 添加 `-public` 参数以监听所有网络接口：
> ```bash
> picoclaw-launcher -public
> ```

<p align="center">
<img src="assets/launcher-webui.jpg" alt="WebUI Launcher" width="600">
</p>

**开始使用：**

打开 WebUI，然后：**1)** 配置 Provider（填入 LLM API Key）-> **2)** 配置 Channel（如 Telegram）-> **3)** 启动 Gateway -> **4)** 开始聊天！

详细 WebUI 文档请参阅 [docs.picoclaw.io](https://docs.picoclaw.io)。

<details>
<summary><b>Docker（备选方案）</b></summary>

```bash
# 1. 克隆本仓库
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. 首次运行——自动生成 docker/data/config.json 后退出
#    （仅在 config.json 和 workspace/ 均不存在时触发）
docker compose -f docker/docker-compose.yml --profile launcher up
# 容器打印 "First-run setup complete." 后停止。

# 3. 填写 API Key
vim docker/data/config.json

# 4. 启动
docker compose -f docker/docker-compose.yml --profile launcher up -d
# 打开 http://localhost:18800
```

> **Docker / 虚拟机用户：** Gateway 默认监听 `127.0.0.1`。设置 `PICOCLAW_GATEWAY_HOST=0.0.0.0` 或使用 `-public` 参数以允许从宿主机访问。

```bash
# 查看日志
docker compose -f docker/docker-compose.yml logs -f

# 停止
docker compose -f docker/docker-compose.yml --profile launcher down

# 更新
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

</details>

<details>
<summary><b>macOS — 首次启动安全警告</b></summary>

macOS 可能会在首次启动时拦截 `picoclaw-launcher`，因为它从互联网下载，未经 Mac App Store 公证。

**第一步：** 双击 `picoclaw-launcher`，会出现安全警告：

<p align="center">
<img src="assets/macos-gatekeeper-warning.jpg" alt="macOS Gatekeeper 警告" width="400">
</p>

> *"picoclaw-launcher" 无法打开 — Apple 无法验证 "picoclaw-launcher" 不含可能损害 Mac 或危及隐私的恶意软件。*

**第二步：** 打开**系统设置** → **隐私与安全性** → 向下滚动找到**安全性**部分 → 点击**仍要打开** → 在弹窗中再次点击**打开**。

<p align="center">
<img src="assets/macos-gatekeeper-allow.jpg" alt="macOS 隐私与安全性 — 仍要打开" width="600">
</p>

完成这一次操作后，后续启动 `picoclaw-launcher` 将不再弹出警告。

</details>

### 💻 TUI Launcher（推荐无头环境 / SSH）

TUI（终端 UI）Launcher 提供功能完整的终端配置与管理界面，适合服务器、树莓派等无显示器环境。

```bash
picoclaw-launcher-tui
```

<p align="center">
<img src="assets/launcher-tui.jpg" alt="TUI Launcher" width="600">
</p>

**开始使用：**

通过 TUI 菜单：**1)** 配置 Provider -> **2)** 配置 Channel -> **3)** 启动 Gateway -> **4)** 开始聊天！

详细 TUI 文档请参阅 [docs.picoclaw.io](https://docs.picoclaw.io)。

### 📱 Android

让你十年前的旧手机焕发新生！将它变成你的 AI 助手。

**方式一：Termux（现已可用）**

1. 安装 [Termux](https://github.com/termux/termux-app)（可从 [GitHub Releases](https://github.com/termux/termux-app/releases) 下载，或在 F-Droid / Google Play 中搜索）
2. 执行以下命令：

```bash
# 从 Release 页面下载最新版本
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./picoclaw onboard   # chroot 提供标准 Linux 文件系统布局
```

然后跟随下面的"Terminal Launcher"章节继续配置。

<img src="assets/termux.jpg" alt="PicoClaw on Termux" width="512">

**方式二：APK 安装（即将推出）**

内置 WebUI 的独立 Android APK 正在开发中，敬请期待！

<details>
<summary><b>Terminal Launcher（适用于资源受限环境）</b></summary>

对于只有 `picoclaw` 核心二进制文件的极简环境（无 Launcher UI），可通过命令行和 JSON 配置文件完成所有配置。

**1. 初始化**

```bash
picoclaw onboard
```

此命令会创建 `~/.picoclaw/config.json` 和工作区目录。

**2. 配置** (`~/.picoclaw/config.json`)

```json
{
  "agents": {
    "defaults": {
      "model_name": "gpt-5.4"
    }
  },
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_key": "sk-your-api-key"
    }
  ]
}
```

> 完整配置模板请参阅仓库中的 `config/config.example.json`。

**3. 开始聊天**

```bash
# 单次提问
picoclaw agent -m "What is 2+2?"

# 交互式对话模式
picoclaw agent

# 启动 Gateway 以接入聊天应用
picoclaw gateway
```

</details>

## 🔌 Providers (LLM)

PicoClaw 通过 `model_list` 配置支持 30+ LLM Provider，使用 `协议/模型` 格式：

| Provider | 协议 | API Key | 备注 |
|----------|------|---------|------|
| [OpenAI](https://platform.openai.com/api-keys) | `openai/` | 必填 | GPT-5.4、GPT-4o、o3 等 |
| [Anthropic](https://console.anthropic.com/settings/keys) | `anthropic/` | 必填 | Claude Opus 4.6、Sonnet 4.6 等 |
| [Google Gemini](https://aistudio.google.com/apikey) | `gemini/` | 必填 | Gemini 3 Flash、2.5 Pro 等 |
| [OpenRouter](https://openrouter.ai/keys) | `openrouter/` | 必填 | 200+ 模型，统一 API |
| [智谱 (GLM)](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) | `zhipu/` | 必填 | GLM-4.7、GLM-5 等 |
| [DeepSeek](https://platform.deepseek.com/api_keys) | `deepseek/` | 必填 | DeepSeek-V3、DeepSeek-R1 |
| [火山引擎](https://console.volcengine.com) | `volcengine/` | 必填 | 豆包、Ark 系列模型 |
| [Qwen](https://dashscope.console.aliyun.com/apiKey) | `qwen/` | 必填 | Qwen3、Qwen-Max 等 |
| [Groq](https://console.groq.com/keys) | `groq/` | 必填 | 快速推理（Llama、Mixtral） |
| [Moonshot (Kimi)](https://platform.moonshot.cn/console/api-keys) | `moonshot/` | 必填 | Kimi 系列模型 |
| [Minimax](https://platform.minimaxi.com/user-center/basic-information/interface-key) | `minimax/` | 必填 | MiniMax 系列模型 |
| [Mistral](https://console.mistral.ai/api-keys) | `mistral/` | 必填 | Mistral Large、Codestral |
| [NVIDIA NIM](https://build.nvidia.com/) | `nvidia/` | 必填 | NVIDIA 托管模型 |
| [Cerebras](https://cloud.cerebras.ai/) | `cerebras/` | 必填 | 快速推理 |
| [Novita AI](https://novita.ai/) | `novita/` | 必填 | 多种开源模型 |
| [小米 MiMo](https://platform.xiaomimimo.com/) | `mimo/` | 必填 | MiMo 系列模型 |
| [Ollama](https://ollama.com/) | `ollama/` | 无需 | 本地模型，自托管 |
| [vLLM](https://docs.vllm.ai/) | `vllm/` | 无需 | 本地部署，兼容 OpenAI |
| [LiteLLM](https://docs.litellm.ai/) | `litellm/` | 视情况 | 100+ Provider 代理 |
| [Azure OpenAI](https://portal.azure.com/) | `azure/` | 必填 | 企业级 Azure 部署 |
| [GitHub Copilot](https://github.com/features/copilot) | `github-copilot/` | OAuth | 设备码登录 |
| [Antigravity](https://console.cloud.google.com/) | `antigravity/` | OAuth | Google Cloud AI |

<details>
<summary><b>本地部署（Ollama、vLLM 等）</b></summary>

**Ollama:**
```json
{
  "model_list": [
    {
      "model_name": "local-llama",
      "model": "ollama/llama3.1:8b",
      "api_base": "http://localhost:11434/v1"
    }
  ]
}
```

**vLLM:**
```json
{
  "model_list": [
    {
      "model_name": "local-vllm",
      "model": "vllm/your-model",
      "api_base": "http://localhost:8000/v1"
    }
  ]
}
```

完整 Provider 配置详情请参阅 [Providers & Models](docs/zh/providers.md)。

</details>

## 💬 Channels（聊天应用）

通过 17+ 消息平台与你的 PicoClaw 对话：

| Channel | 配置难度 | 协议 | 文档 |
|---------|----------|------|------|
| **Telegram** | 简单（bot token） | 长轮询 | [指南](docs/channels/telegram/README.zh.md) |
| **Discord** | 简单（bot token + intents） | WebSocket | [指南](docs/channels/discord/README.zh.md) |
| **WhatsApp** | 简单（扫码或 bridge URL） | 原生 / Bridge | [指南](docs/zh/chat-apps.md#whatsapp) |
| **微信 (Weixin)** | 简单（扫码登录） | iLink API | [指南](docs/zh/chat-apps.md#weixin) |
| **QQ** | 简单（AppID + AppSecret） | WebSocket | [指南](docs/channels/qq/README.zh.md) |
| **Slack** | 简单（bot + app token） | Socket Mode | [指南](docs/channels/slack/README.zh.md) |
| **Matrix** | 中等（homeserver + token） | Sync API | [指南](docs/channels/matrix/README.zh.md) |
| **钉钉** | 中等（client credentials） | Stream | [指南](docs/channels/dingtalk/README.zh.md) |
| **飞书 / Lark** | 中等（App ID + Secret） | WebSocket/SDK | [指南](docs/channels/feishu/README.zh.md) |
| **LINE** | 中等（credentials + webhook） | Webhook | [指南](docs/channels/line/README.zh.md) |
| **企业微信** | 简单（扫码登录或手动配置） | WebSocket | [指南](docs/channels/wecom/README.zh.md) |
| **IRC** | 中等（server + nick） | IRC 协议 | [指南](docs/zh/chat-apps.md#irc) |
| **OneBot** | 中等（WebSocket URL） | OneBot v11 | [指南](docs/channels/onebot/README.zh.md) |
| **MaixCam** | 简单（启用即可） | TCP socket | [指南](docs/channels/maixcam/README.zh.md) |
| **Pico** | 简单（启用即可） | 原生协议 | 内置 |
| **Pico Client** | 简单（WebSocket URL） | WebSocket | 内置 |

> 所有基于 Webhook 的 Channel 共用同一个 Gateway HTTP 服务器（`gateway.host`:`gateway.port`，默认 `127.0.0.1:18790`）。飞书使用 WebSocket/SDK 模式，不使用共享 HTTP 服务器。

> 日志详细程度通过 `gateway.log_level` 控制（默认：`warn`）。支持的值：`debug`、`info`、`warn`、`error`、`fatal`。也可通过 `PICOCLAW_LOG_LEVEL` 环境变量设置。详见[配置指南](docs/zh/configuration.md#gateway-日志等级)。

详细 Channel 配置说明请参阅 [聊天应用配置](docs/zh/chat-apps.md)。

## 🔧 Tools

### 🔍 网络搜索

PicoClaw 可以搜索网络以提供最新信息。在 `tools.web` 中配置：

| 搜索引擎 | API Key | 免费额度 | 链接 |
|---------|---------|---------|------|
| [百度搜索](https://cloud.baidu.com/doc/qianfan-api/s/Wmbq4z7e5) | 必填 | 1000 次/天 | AI 搜索，国内首选 |
| [Tavily](https://tavily.com) | 必填 | 1000 次/月 | 专为 AI Agent 优化 |
| [GLM Search](https://open.bigmodel.cn/) | 必填 | 视情况 | 智谱网络搜索 |
| DuckDuckGo | 无需 | 无限制 | 内置备用（国内访问困难） |
| [Perplexity](https://www.perplexity.ai) | 必填 | 付费 | AI 驱动搜索（国内访问困难） |
| [Brave Search](https://brave.com/search/api) | 必填 | 2000 次/月 | 快速且注重隐私（国内访问困难） |
| [SearXNG](https://github.com/searxng/searxng) | 无需 | 自托管 | 免费元搜索引擎 |

### ⚙️ 其他工具

PicoClaw 内置文件操作、代码执行、定时任务等工具。详情请参阅 [工具配置](docs/zh/tools_configuration.md)。

## 🎯 Skills

Skills 是扩展 Agent 能力的模块化插件，从工作区的 `SKILL.md` 文件加载。

**从 ClawHub 安装 Skills：**

```bash
picoclaw skills search "web scraping"
picoclaw skills install <skill-name>
```

**配置 ClawHub token**（可选，用于提高速率限制）：

在 `config.json` 中添加：
```json
{
  "tools": {
    "skills": {
      "registries": {
        "clawhub": {
          "auth_token": "your-clawhub-token"
        }
      }
    }
  }
}
```

更多详情请参阅 [工具配置 - Skills](docs/zh/tools_configuration.md#skills-tool)。

## 🔗 MCP (Model Context Protocol)

PicoClaw 原生支持 [MCP](https://modelcontextprotocol.io/) — 连接任意 MCP 服务器，通过外部工具和数据源扩展 Agent 能力。

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "filesystem": {
          "enabled": true,
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
        }
      }
    }
  }
}
```

完整 MCP 配置（stdio、SSE、HTTP 传输、Tool Discovery）请参阅 [工具配置 - MCP](docs/zh/tools_configuration.md#mcp-tool)。

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> 加入 Agent 社交网络

通过 CLI 或任何已集成的聊天应用发送一条消息，即可将 PicoClaw 连接到 Agent 社交网络。

**阅读 `https://clawdchat.ai/skill.md` 并按照说明加入 [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ CLI 命令行参考

| 命令                       | 说明                   |
| ------------------------- | ---------------------- |
| `picoclaw onboard`        | 初始化配置与工作区       |
| `picoclaw auth weixin` | 扫码连接微信个人号 |
| `picoclaw agent -m "..."` | 与 Agent 对话           |
| `picoclaw agent`          | 交互式对话模式           |
| `picoclaw gateway`        | 启动网关                |
| `picoclaw status`         | 查看状态                |
| `picoclaw version`        | 查看版本信息             |
| `picoclaw model`          | 查看或切换默认模型       |
| `picoclaw cron list`      | 列出所有定时任务         |
| `picoclaw cron add ...`   | 添加定时任务             |
| `picoclaw cron disable`   | 禁用定时任务             |
| `picoclaw cron remove`    | 删除定时任务             |
| `picoclaw skills list`    | 列出已安装 Skills        |
| `picoclaw skills install` | 安装 Skill              |
| `picoclaw migrate`        | 从旧版本迁移数据         |
| `picoclaw auth login`     | 认证 Provider           |

### ⏰ 定时任务 / 提醒

PicoClaw 通过 `cron` 工具支持定时提醒和重复任务：

* **一次性提醒**: "10分钟后提醒我" → 10分钟后触发一次
* **重复任务**: "每2小时提醒我" → 每2小时触发
* **Cron 表达式**: "每天上午9点提醒我" → 使用 cron 表达式

## 📚 文档

详细指南请参阅以下文档，README 仅涵盖快速入门。

| 主题 | 说明 |
|------|------|
| 🐳 [Docker 与快速开始](docs/zh/docker.md) | Docker Compose 配置、Launcher/Agent 模式、快速开始 |
| 💬 [聊天应用配置](docs/zh/chat-apps.md) | 全部 17+ Channel 配置指南 |
| ⚙️ [配置指南](docs/zh/configuration.md) | 环境变量、工作区布局、安全沙箱 |
| 🔌 [提供商与模型配置](docs/zh/providers.md) | 30+ LLM Provider、模型路由、model_list 配置 |
| 🔄 [异步任务与 Spawn](docs/zh/spawn-tasks.md) | 快速任务、长任务与 Spawn、异步子 Agent 编排 |
| 🪝 [Hook 系统](docs/hooks/README.zh.md) | 事件驱动 Hook：观察者、拦截器、审批 Hook |
| 🎯 [Steering](docs/steering.md) | 在工具调用间向运行中的 Agent 注入消息 |
| 🔀 [SubTurn](docs/subturn.md) | 子 Agent 协调、并发控制、生命周期管理 |
| 🐛 [疑难解答](docs/zh/troubleshooting.md) | 常见问题与解决方案 |
| 🔧 [工具配置](docs/zh/tools_configuration.md) | 工具启用/禁用、执行策略、MCP、Skills |
| 📋 [硬件兼容列表](docs/zh/hardware-compatibility.md) | 已测试板卡、最低要求 |

## 🤝 贡献与路线图

欢迎提交 PR！代码库刻意保持小巧和可读。🤗

查看完整的 [社区路线图](https://github.com/sipeed/picoclaw/issues/988) 和 [CONTRIBUTING.md](CONTRIBUTING.md)。

开发者群组正在组建中，入群门槛：至少合并过 1 个 PR。

用户群组：

Discord: <https://discord.gg/V4sAZ9XWpN>

WeChat:
<img src="assets/wechat.png" alt="WeChat group QR code" width="512">





