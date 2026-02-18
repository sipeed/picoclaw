<div align="center">
<img src="assets/logo.jpg" alt="PicoClaw" width="512">

<h1>PicoClaw: 基于Go语言的超高效 AI 助手</h1>

<h3>10$硬件 · 10MB内存 · 1秒启动 · 皮皮虾，我们走！</h3>

  <p>
    <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20RISC--V-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://picoclaw.io"><img src="https://img.shields.io/badge/Website-picoclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://x.com/SipeedIO"><img src="https://img.shields.io/badge/X_(Twitter)-SipeedIO-black?style=flat&logo=x&logoColor=white" alt="Twitter"></a>
  </p>

 **中文** | [日本語](README.ja.md) | [Português](README.pt-br.md) | [English](README.md)
</div>

---

🦐 **PicoClaw** 是一个受 [nanobot](https://github.com/HKUDS/nanobot) 启发的超轻量级个人 AI 助手。它采用 **Go 语言** 从零重构，经历了一个“自举”过程——即由 AI Agent 自身驱动了整个架构迁移和代码优化。

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

注意：人手有限，中文文档可能略有滞后，请优先查看英文文档。

> [!CAUTION]
> **🚨 SECURITY & OFFICIAL CHANNELS / 安全声明**
> * **无加密货币 (NO CRYPTO):** PicoClaw **没有** 发行任何官方代币、Token 或虚拟货币。所有在 `pump.fun` 或其他交易平台上的相关声称均为 **诈骗**。
> * **官方域名:** 唯一的官方网站是 **[picoclaw.io](https://picoclaw.io)**，公司官网是 **[sipeed.com](https://sipeed.com)**。
> * **警惕:** 许多 `.ai/.org/.com/.net/...` 后缀的域名被第三方抢注，请勿轻信。
> * **注意:** picoclaw正在初期的快速功能开发阶段，可能有尚未修复的网络安全问题，在1.0正式版发布前，请不要将其部署到生产环境中
> * **注意:** picoclaw最近合并了大量PRs，近期版本可能内存占用较大(10~20MB)，我们将在功能较为收敛后进行资源占用优化.


## 📢 新闻 (News)
2026-02-16 🎉 PicoClaw 在一周内突破了12K star! 感谢大家的关注！PicoClaw 的成长速度超乎我们预期. 由于PR数量的快速膨胀，我们亟需社区开发者参与维护. 我们需要的志愿者角色和roadmap已经发布到了[这里](docs/picoclaw_community_roadmap_260216.md), 期待你的参与！

2026-02-13 🎉 **PicoClaw 在 4 天内突破 5000 Stars！** 感谢社区的支持！由于正值中国春节假期，PR 和 Issue 涌入较多，我们正在利用这段时间敲定 **项目路线图 (Roadmap)** 并组建 **开发者群组**，以便加速 PicoClaw 的开发。
🚀 **行动号召：** 请在 GitHub Discussions 中提交您的功能请求 (Feature Requests)。我们将在接下来的周会上进行审查和优先级排序。

2026-02-09 🎉 **PicoClaw 正式发布！** 仅用 1 天构建，旨在将 AI Agent 带入 10 美元硬件与 <10MB 内存的世界。🦐 PicoClaw（皮皮虾），我们走！

## ✨ 特性

🪶 **超轻量级**: 核心功能内存占用 <10MB — 比 Clawdbot 小 99%。

💰 **极低成本**: 高效到足以在 10 美元的硬件上运行 — 比 Mac mini 便宜 98%。

⚡️ **闪电启动**: 启动速度快 400 倍，即使在 0.6GHz 单核处理器上也能在 1 秒内启动。

🌍 **真正可移植**: 跨 RISC-V、ARM 和 x86 架构的单二进制文件，一键运行！

🤖 **AI 自举**: 纯 Go 语言原生实现 — 95% 的核心代码由 Agent 生成，并经由“人机回环 (Human-in-the-loop)”微调。

|  | OpenClaw | NanoBot | **PicoClaw** |
| --- | --- | --- | --- |
| **语言** | TypeScript | Python | **Go** |
| **RAM** | >1GB | >100MB | **< 10MB** |
| **启动时间**</br>(0.8GHz core) | >500s | >30s | **<1s** |
| **成本** | Mac Mini $599 | 大多数 Linux 开发板 ~$50 | **任意 Linux 开发板**</br>**低至 $10** |

<img src="assets/compare.jpg" alt="PicoClaw" width="512">

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
picoclaw 可以将你10年前的老旧手机废物利用，变身成为你的AI助理！快速指南:
1. 先去应用商店下载安装Termux
2. 打开后执行指令
```bash
# 注意: 下面的v0.1.1 可以换为你实际看到的最新版本
wget https://github.com/sipeed/picoclaw/releases/download/v0.1.1/picoclaw-linux-arm64
chmod +x picoclaw-linux-arm64
pkg install proot
termux-chroot ./picoclaw-linux-arm64 onboard
```
然后跟随下面的“快速开始”章节继续配置picoclaw即可使用！   
<img src="assets/termux.jpg" alt="PicoClaw" width="512">




### 🐜 创新的低占用部署

PicoClaw 几乎可以部署在任何 Linux 设备上！

* $9.9 [LicheeRV-Nano](https://www.aliexpress.com/item/1005006519668532.html) E(网口) 或 W(WiFi6) 版本，用于极简家庭助手。
* $30~50 [NanoKVM](https://www.aliexpress.com/item/1005007369816019.html)，或 $100 [NanoKVM-Pro](https://www.aliexpress.com/item/1005010048471263.html)，用于自动化服务器运维。
* $50 [MaixCAM](https://www.aliexpress.com/item/1005008053333693.html) 或 $100 [MaixCAM2](https://www.kickstarter.com/projects/zepan/maixcam2-build-your-next-gen-4k-ai-camera)，用于智能监控。

[https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4](https://private-user-images.githubusercontent.com/83055338/547056448-e7b031ff-d6f5-4468-bcca-5726b6fecb5c.mp4)

🌟 更多部署案例敬请期待！

## 📦 安装

### 使用预编译二进制文件安装

从 [Release 页面](https://github.com/sipeed/picoclaw/releases) 下载适用于您平台的固件。

### 从源码安装（获取最新特性，开发推荐）

```bash
git clone https://github.com/sipeed/picoclaw.git

cd picoclaw
make deps

# 构建（无需安装）
make build

# 为多平台构建
make build-all

# 构建并安装
make install

```

## 🐳 Docker Compose

您也可以使用 Docker Compose 运行 PicoClaw，无需在本地安装任何环境。

```bash
# 1. 克隆仓库
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw

# 2. 设置 API Key
cp config/config.example.json config/config.json
vim config/config.json      # 设置 DISCORD_BOT_TOKEN, API keys 等

# 3. 构建并启动
docker compose --profile gateway up -d

# 4. 查看日志
docker compose logs -f picoclaw-gateway

# 5. 停止
docker compose --profile gateway down

```

### Agent 模式 (一次性运行)

```bash
# 提问
docker compose run --rm picoclaw-agent -m "2+2 等于几？"

# 交互模式
docker compose run --rm picoclaw-agent

```

### 重新构建

```bash
docker compose --profile gateway build --no-cache
docker compose --profile gateway up -d

```

### 🚀 快速开始

> [!TIP]
> 在 `~/.picoclaw/config.json` 中设置您的 API Key。
> 获取 API Key: [OpenRouter](https://openrouter.ai/keys) (LLM) · [Zhipu (智谱)](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) (LLM)
> 网络搜索是 **可选的** - 获取免费的 [Brave Search API](https://brave.com/search/api) (每月 2000 次免费查询)

**1. 初始化 (Initialize)**

```bash
picoclaw onboard

```

**2. 配置 (Configure)** (`~/.picoclaw/config.json`)

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "allow_patterns": [],
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "xxx",
      "api_base": "https://openrouter.ai/api/v1"
    }
  },
  "tools": {
    "web": {
      "search": {
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      }
    },
    "cron": {
      "exec_timeout_minutes": 5
    }
  }
}

```

**3. 获取 API Key**

* **LLM 提供商**: [OpenRouter](https://openrouter.ai/keys) · [Zhipu](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) · [Anthropic](https://console.anthropic.com) · [OpenAI](https://platform.openai.com) · [Gemini](https://aistudio.google.com/api-keys)
* **网络搜索** (可选): [Brave Search](https://brave.com/search/api) - 提供免费层级 (2000 请求/月)

> **注意**: 完整的配置模板请参考 `config.example.json`。

**4. 对话 (Chat)**

```bash
picoclaw agent -m "2+2 等于几？"

```

就是这样！您在 2 分钟内就拥有了一个可工作的 AI 助手。

---

## 💬 聊天应用集成 (Chat Apps)

通过 Telegram, Discord 或钉钉与您的 PicoClaw 对话。

| 渠道 | 设置难度 |
| --- | --- |
| **Telegram** | 简单 (仅需 token) |
| **Discord** | 简单 (bot token + intents) |
| **QQ** | 简单 (AppID + AppSecret) |
| **钉钉 (DingTalk)** | 中等 (app credentials) |

<details>
<summary><b>Telegram</b> (推荐)</summary>

**1. 创建机器人**

* 打开 Telegram，搜索 `@BotFather`
* 发送 `/newbot`，按照提示操作
* 复制 token

**2. 配置**

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allowFrom": ["YOUR_USER_ID"]
    }
  }
}

```

> 从 Telegram 上的 `@userinfobot` 获取您的用户 ID。

**3. 运行**

```bash
picoclaw gateway

```

</details>

<details>
<summary><b>Discord</b></summary>

**1. 创建机器人**

* 前往 [https://discord.com/developers/applications](https://discord.com/developers/applications)
* Create an application → Bot → Add Bot
* 复制 bot token

**2. 开启 Intents**

* 在 Bot 设置中，开启 **MESSAGE CONTENT INTENT**
* (可选) 如果计划基于成员数据使用白名单，开启 **SERVER MEMBERS INTENT**

**3. 获取您的 User ID**

* Discord 设置 → Advanced → 开启 **Developer Mode**
* 右键点击您的头像 → **Copy User ID**

**4. 配置**

```json
{
  "channels": {
    "discord": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "allowFrom": ["YOUR_USER_ID"]
    }
  }
}

```

**5. 邀请机器人**

* OAuth2 → URL Generator
* Scopes: `bot`
* Bot Permissions: `Send Messages`, `Read Message History`
* 打开生成的邀请 URL，将机器人添加到您的服务器

**6. 运行**

```bash
picoclaw gateway

```

</details>

<details>
<summary><b>QQ</b></summary>

**1. 创建机器人**

* 前往 [QQ 开放平台](https://q.qq.com/#)
* 创建应用 → 获取 **AppID** 和 **AppSecret**

**2. 配置**

```json
{
  "channels": {
    "qq": {
      "enabled": true,
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}

```

> 将 `allow_from` 设为空以允许所有用户，或指定 QQ 号以限制访问。

**3. 运行**

```bash
picoclaw gateway

```

</details>

<details>
<summary><b>钉钉 (DingTalk)</b></summary>

**1. 创建机器人**

* 前往 [开放平台](https://open.dingtalk.com/)
* 创建内部应用
* 复制 Client ID 和 Client Secret

**2. 配置**

```json
{
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}

```

> 将 `allow_from` 设为空以允许所有用户，或指定 ID 以限制访问。

**3. 运行**

```bash
picoclaw gateway

```

</details>

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> 加入 Agent 社交网络

只需通过 CLI 或任何集成的聊天应用发送一条消息，即可将 PicoClaw 连接到 Agent 社交网络。

**阅读 `https://clawdchat.ai/skill.md` 并按照说明加入 [ClawdChat.ai**](https://clawdchat.ai)

## ⚙️ 配置详解

配置文件路径: `~/.picoclaw/config.json`

### 工作区布局 (Workspace Layout)

PicoClaw 将数据存储在您配置的工作区中（默认：`~/.picoclaw/workspace`）：

```
~/.picoclaw/workspace/
├── sessions/          # 对话会话和历史
├── memory/           # 长期记忆 (MEMORY.md)
├── state/            # 持久化状态 (最后一次频道等)
├── cron/             # 定时任务数据库
├── skills/           # 自定义技能
├── AGENTS.md         # Agent 行为指南
├── HEARTBEAT.md      # 周期性任务提示词 (每 30 分钟检查一次)
├── IDENTITY.md       # Agent 身份设定
├── SOUL.md           # Agent 灵魂/性格
├── TOOLS.md          # 工具描述
└── USER.md           # 用户偏好

```

### 心跳 / 周期性任务 (Heartbeat)

### 🔒 安全沙箱 (Security Sandbox)

默认情况下，PicoClaw 在沙箱中运行，Agent 仅能在工作区内访问文件与执行命令。

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "allow_patterns": []
    }
  }
}
```

| 选项 | 默认值 | 描述 |
| --- | --- | --- |
| `workspace` | `~/.picoclaw/workspace` | Agent 工作目录 |
| `restrict_to_workspace` | `true` | 限制文件/命令访问仅在工作区内 |
| `allow_patterns` | `[]` | 可选：对 `exec` 命令内容做正则白名单匹配；设置后命令必须命中至少一条规则 |

`allow_patterns` 会对完整命令进行匹配（不区分大小写）。为空数组时，不启用白名单过滤。

示例：

```json
{
  "agents": {
    "defaults": {
      "allow_patterns": [
        "^ls(\\s|$)",
        "^pwd$",
        "^cat\\s+README\\.md$"
      ]
    }
  }
}
```

> ⚠️ 说明：`allow_patterns` 只影响 `exec` 工具；仍受 `restrict_to_workspace` 约束。

PicoClaw 可以自动执行周期性任务。在工作区创建 `HEARTBEAT.md` 文件：

```markdown
# Periodic Tasks

- Check my email for important messages
- Review my calendar for upcoming events
- Check the weather forecast

```

Agent 将每隔 30 分钟（可配置）读取此文件，并使用可用工具执行任务。

#### 使用 Spawn 的异步任务

对于耗时较长的任务（网络搜索、API 调用），使用 `spawn` 工具创建一个 **子 Agent (subagent)**：

```markdown
# Periodic Tasks

## Quick Tasks (respond directly)
- Report current time

## Long Tasks (use spawn for async)
- Search the web for AI news and summarize
- Check email and report important messages

```

**关键行为：**

| 特性 | 描述 |
| --- | --- |
| **spawn** | 创建异步子 Agent，不阻塞主心跳进程 |
| **独立上下文** | 子 Agent 拥有独立上下文，无会话历史 |
| **message tool** | 子 Agent 通过 message 工具直接与用户通信 |
| **非阻塞** | spawn 后，心跳继续处理下一个任务 |

#### 子 Agent 通信原理

```
心跳触发 (Heartbeat triggers)
    ↓
Agent 读取 HEARTBEAT.md
    ↓
对于长任务: spawn 子 Agent
    ↓                           ↓
继续下一个任务               子 Agent 独立工作
    ↓                           ↓
所有任务完成                 子 Agent 使用 "message" 工具
    ↓                           ↓
响应 HEARTBEAT_OK            用户直接收到结果

```

子 Agent 可以访问工具（message, web_search 等），并且无需通过主 Agent 即可独立与用户通信。

**配置：**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}

```

| 选项 | 默认值 | 描述 |
| --- | --- | --- |
| `enabled` | `true` | 启用/禁用心跳 |
| `interval` | `30` | 检查间隔，单位分钟 (最小: 5) |

**环境变量:**

* `PICOCLAW_HEARTBEAT_ENABLED=false` 禁用
* `PICOCLAW_HEARTBEAT_INTERVAL=60` 更改间隔

### 提供商 (Providers)

> [!NOTE]
> Groq 通过 Whisper 提供免费的语音转录。如果配置了 Groq，Telegram 语音消息将被自动转录为文字。

| 提供商 | 用途 | 获取 API Key |
| --- | --- | --- |
| `gemini` | LLM (Gemini 直连) | [aistudio.google.com](https://aistudio.google.com) |
| `zhipu` | LLM (智谱直连) | [bigmodel.cn](bigmodel.cn) |
| `openrouter(待测试)` | LLM (推荐，可访问所有模型) | [openrouter.ai](https://openrouter.ai) |
| `anthropic(待测试)` | LLM (Claude 直连) | [console.anthropic.com](https://console.anthropic.com) |
| `openai(待测试)` | LLM (GPT 直连) | [platform.openai.com](https://platform.openai.com) |
| `deepseek(待测试)` | LLM (DeepSeek 直连) | [platform.deepseek.com](https://platform.deepseek.com) |
| `groq` | LLM + **语音转录** (Whisper) | [console.groq.com](https://console.groq.com) |

<details>
<summary><b>智谱 (Zhipu) 配置示例</b></summary>

**1. 获取 API key 和 base URL**

* 获取 [API key](https://bigmodel.cn/usercenter/proj-mgmt/apikeys)

**2. 配置**

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "allow_patterns": [],
      "model": "glm-4.7",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "providers": {
    "zhipu": {
      "api_key": "Your API Key",
      "api_base": "https://open.bigmodel.cn/api/paas/v4"
    },
  },
}

```

**3. 运行**

```bash
picoclaw agent -m "你好"

```

</details>

<details>
<summary><b>完整配置示例</b></summary>

```json
{
  "agents": {
    "defaults": {
      "model": "anthropic/claude-opus-4-5"
    }
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    },
    "groq": {
      "api_key": "gsk_xxx"
    }
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "123456:ABC...",
      "allow_from": ["123456789"]
    },
    "discord": {
      "enabled": true,
      "token": "",
      "allow_from": [""]
    },
    "whatsapp": {
      "enabled": false
    },
    "feishu": {
      "enabled": false,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    },
    "qq": {
      "enabled": false,
      "app_id": "",
      "app_secret": "",
      "allow_from": []
    }
  },
  "tools": {
    "web": {
      "search": {
        "api_key": "BSA..."
      }
    },
    "cron": {
      "exec_timeout_minutes": 5
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}

```

</details>

## CLI 命令行参考

| 命令 | 描述 |
| --- | --- |
| `picoclaw onboard` | 初始化配置和工作区 |
| `picoclaw agent -m "..."` | 与 Agent 对话 |
| `picoclaw agent` | 交互式聊天模式 |
| `picoclaw gateway` | 启动网关 (Gateway) |
| `picoclaw status` | 显示状态 |
| `picoclaw cron list` | 列出所有定时任务 |
| `picoclaw cron add ...` | 添加定时任务 |

### 定时任务 / 提醒 (Scheduled Tasks)

PicoClaw 通过 `cron` 工具支持定时提醒和重复任务：

* **一次性提醒**: "Remind me in 10 minutes" (10分钟后提醒我) → 10分钟后触发一次
* **重复任务**: "Remind me every 2 hours" (每2小时提醒我) → 每2小时触发
* **Cron 表达式**: "Remind me at 9am daily" (每天上午9点提醒我) → 使用 cron 表达式

任务存储在 `~/.picoclaw/workspace/cron/` 中并自动处理。

## 🤝 贡献与路线图 (Roadmap)

欢迎提交 PR！代码库刻意保持小巧和可读。🤗

路线图即将发布...

开发者群组正在组建中，入群门槛：至少合并过 1 个 PR。

用户群组：

Discord:  [https://discord.gg/V4sAZ9XWpN](https://discord.gg/V4sAZ9XWpN)

<img src="assets/wechat.png" alt="PicoClaw" width="512">

## 🐛 疑难解答 (Troubleshooting)

### 网络搜索提示 "API 配置问题"

如果您尚未配置搜索 API Key，这是正常的。PicoClaw 会提供手动搜索的帮助链接。

启用网络搜索：

1. 在 [https://brave.com/search/api](https://brave.com/search/api) 获取免费 API Key (每月 2000 次免费查询)
2. 添加到 `~/.picoclaw/config.json`:
```json
{
  "tools": {
    "web": {
      "search": {
        "api_key": "YOUR_BRAVE_API_KEY",
        "max_results": 5
      }
    }
  }
}

```



### 遇到内容过滤错误 (Content Filtering Errors)

某些提供商（如智谱）有严格的内容过滤。尝试改写您的问题或使用其他模型。

### Telegram bot 提示 "Conflict: terminated by other getUpdates"

这表示有另一个机器人实例正在运行。请确保同一时间只有一个 `picoclaw gateway` 进程在运行。

---

## 📝 API Key 对比

| 服务 | 免费层级 | 适用场景 |
| --- | --- | --- |
| **OpenRouter** | 200K tokens/月 | 多模型聚合 (Claude, GPT-4 等) |
| **智谱 (Zhipu)** | 200K tokens/月 | 最适合中国用户 |
| **Brave Search** | 2000 次查询/月 | 网络搜索功能 |
| **Groq** | 提供免费层级 | 极速推理 (Llama, Mixtral) |