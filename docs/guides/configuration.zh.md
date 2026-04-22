# ⚙️ 配置指南

> 返回 [README](../project/README.zh.md)

## ⚙️ 配置详解

配置文件路径: `~/.picoclaw/config.json`

### 环境变量

你可以使用环境变量覆盖默认路径。这对于便携安装、容器化部署或将 picoclaw 作为系统服务运行非常有用。这些变量是独立的，控制不同的路径。

| 变量              | 描述                                                                                                                             | 默认路径                  |
|-------------------|-----------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `PICOCLAW_CONFIG` | 覆盖配置文件的路径。这直接告诉 picoclaw 加载哪个 `config.json`，忽略所有其他位置。 | `~/.picoclaw/config.json` |
| `PICOCLAW_HOME`   | 覆盖 picoclaw 数据根目录。这会更改 `workspace` 和其他数据目录的默认位置。          | `~/.picoclaw`             |

**示例：**

```bash
# 使用特定的配置文件运行 picoclaw
# 工作区路径将从该配置文件中读取
PICOCLAW_CONFIG=/etc/picoclaw/production.json picoclaw gateway

# 在 /opt/picoclaw 中存储所有数据运行 picoclaw
# 配置将从默认的 ~/.picoclaw/config.json 加载
# 工作区将在 /opt/picoclaw/workspace 创建
PICOCLAW_HOME=/opt/picoclaw picoclaw agent

# 同时使用两者进行完全自定义设置
PICOCLAW_HOME=/srv/picoclaw PICOCLAW_CONFIG=/srv/picoclaw/main.json picoclaw gateway
```

### Gateway 日志等级

`gateway.log_level` 控制 Gateway 的日志详细程度，可在 `config.json` 中配置：

```json
{
  "gateway": {
    "log_level": "warn"
  }
}
```

默认值为 `warn`。支持的值：`debug`、`info`、`warn`、`error`、`fatal`。

也可通过环境变量覆盖：`PICOCLAW_LOG_LEVEL=info`

### 工作区布局 (Workspace Layout)

PicoClaw 将数据存储在您配置的工作区中（默认：`~/.picoclaw/workspace`）：

```
~/.picoclaw/workspace/
├── sessions/          # 对话会话和历史
├── memory/           # 长期记忆 (MEMORY.md)
├── state/            # 持久化状态 (最后一次频道等)
├── cron/             # 定时任务数据库
├── skills/           # 自定义技能
├── AGENT.md          # Agent 行为指南
├── HEARTBEAT.md      # 周期性任务提示词 (每 30 分钟检查一次)
├── IDENTITY.md       # Agent 身份设定
├── SOUL.md           # Agent 灵魂/性格
└── USER.md           # 用户偏好
```

> **提示：** 对 `AGENT.md`、`SOUL.md`、`USER.md` 和 `memory/MEMORY.md` 的修改会通过文件修改时间（mtime）在运行时自动检测。**无需重启 gateway**，Agent 将在下一次请求时自动加载最新内容。

### Web 启动器控制台

用 **picoclaw-launcher** 打开浏览器控制台前需要先使用密码登录。首次启动时打开 `/launcher-setup` 创建 dashboard 登录密码；后续手动登录使用 `/launcher-login`。

- **配置文件**：与 `config.json` 同一目录（若设置了 `PICOCLAW_CONFIG`，则与它所指的文件同目录）。启动器专用文件名为 `launcher-config.json`。
- **密码存储**：支持的平台会把 bcrypt 后的密码哈希存入 `launcher-auth.db`。如果当前平台不支持 SQLite 密码存储，则把 bcrypt 哈希存入 `launcher-config.json`。
- **旧配置迁移**：旧版 `launcher_token` 会一次性迁移为密码登录，并从保存后的 launcher 配置中移除。
- **本地自动登录**：launcher 启动后自动打开本地浏览器时，会使用仅允许 loopback 访问的一次性引导入口自动设置会话 Cookie。
- **不再支持的鉴权方式**：不再支持 URL token 登录（`?token=...`）、`PICOCLAW_LAUNCHER_TOKEN` 和 `Authorization: Bearer` dashboard 鉴权。
- **退出登录**：应使用 **`POST /api/auth/logout`**，且请求头为 **`Content-Type: application/json`**（请求体可为 `{}`），勿使用可被第三方页面触发的 GET 链接登出。
- **暴力尝试**：`POST /api/auth/login` 对同一远程地址有 **每分钟尝试次数上限**（超限返回 HTTP 429）。
- **会话时长**：登录后的 HttpOnly 会话 Cookie 默认约 **31 天**有效，但 launcher 进程重启后已有会话会失效。

### 技能来源 (Skill Sources)

默认情况下，技能会按以下顺序加载：

1. `~/.picoclaw/workspace/skills`（工作区）
2. `~/.picoclaw/skills`（全局）
3. `<构建时嵌入路径>/skills`（内置）

在高级/测试场景下，可通过以下环境变量覆盖内置技能目录：

```bash
export PICOCLAW_BUILTIN_SKILLS=/path/to/skills
```

### 在聊天频道中使用技能

技能安装完成后，可以直接在聊天频道里查看并显式启用它们：

- `/list skills`：显示当前 Agent 可用的已安装技能名称。
- `/use <skill> <message>`：只对当前这一条请求强制使用指定技能。
- `/use <skill>`：为同一会话中的下一条消息预先启用该技能。
- `/use clear`：取消通过 `/use <skill>` 设置的待应用技能。
- `/btw <question>`：发起一个即时的旁支提问，且不改动当前会话历史。`/btw` 会按一次无工具的直接问答处理，不会进入常规的工具执行流程。

示例：

```text
/list skills
/use git explain how to squash the last 3 commits
/btw 帮我回顾一下刚才关于发布方案的结论
/use italiapersonalfinance
dammi le ultime news
```

### 统一命令执行策略

- 通用斜杠命令通过 `pkg/agent/loop.go` 中的 `commands.Executor` 统一执行。
- Channel 适配器不再在本地消费通用命令；它们只负责把入站文本转发到 bus/agent 路径。Telegram 仍会在启动时自动注册其支持的命令菜单，例如 `/start`、`/help`、`/show`、`/list`、`/use` 和 `/btw`。
- 未注册的斜杠命令（例如 `/foo`）会透传给 LLM 按普通输入处理。
- 已注册但当前 channel 不支持的命令（例如 WhatsApp 上的 `/show`）会返回明确的用户可见错误，并停止后续处理。

### Session 隔离

Session scope 决定了聊天、用户、线程和 space 之间共享多少上下文。

- 全局默认值使用 `session.dimensions`
- 如果只想让某条路由例外，使用 dispatch rule 上的 `session_dimensions`

如果你想看完整的隔离方案和配置配方，请看 [Session 使用指南](session-guide.zh.md)。

### Routing

Routing 通过 `agents.dispatch.rules` 配置。

每条规则都针对 channel 归一化后的 inbound context 做匹配。
规则按从上到下顺序检查，第一条命中的规则立即生效。若没有规则命中，PicoClaw 会回退到默认 agent。

支持的匹配字段：

* `channel`
* `account`
* `space`
* `chat`
* `topic`
* `sender`
* `mentioned`

这些值使用和 session system 一致的归一化词汇：

* `space`: `workspace:t001`、`guild:123456`
* `chat`: `direct:user123`、`group:-100123`、`channel:c123`
* `topic`: `topic:42`
* `sender`: 平台归一化后的 sender 标识

规则也可以通过 `session_dimensions` 覆盖全局 `session.dimensions`，这样路由和会话隔离就能保持一致，而不必回到旧的 `bindings` 或 `dm_scope` 配置。

示例：

```json
{
  "agents": {
    "list": [
      { "id": "main", "default": true },
      { "id": "support" },
      { "id": "sales" }
    ],
    "dispatch": {
      "rules": [
        {
          "name": "vip in support group",
          "agent": "sales",
          "when": {
            "channel": "telegram",
            "chat": "group:-1001234567890",
            "sender": "12345"
          },
          "session_dimensions": ["chat", "sender"]
        },
        {
          "name": "telegram support group",
          "agent": "support",
          "when": {
            "channel": "telegram",
            "chat": "group:-1001234567890"
          },
          "session_dimensions": ["chat"]
        }
      ]
    }
  },
  "session": {
    "dimensions": ["chat"]
  }
}
```

在这个例子里，VIP 规则必须放在更宽泛的群规则前面。
因为 routing 是严格按顺序执行的，所以更具体的规则要放前面，兜底规则放后面。

如果你想看更完整的 agent 路由和模型分层示例，请看 [路由使用指南](routing-guide.zh.md)。

### 🔒 安全沙箱 (Security Sandbox)

PicoClaw 默认在沙箱环境中运行。Agent 只能访问配置的工作区内的文件和执行命令。

#### 默认配置

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true
    }
  }
}
```

| 选项                    | 默认值                  | 描述                          |
| ----------------------- | ----------------------- | ----------------------------- |
| `workspace`             | `~/.picoclaw/workspace` | Agent 的工作目录              |
| `restrict_to_workspace` | `true`                  | 限制文件/命令访问在工作区内   |

#### 受保护的工具

当 `restrict_to_workspace: true` 时，以下工具会被沙箱化：

| 工具          | 功能         | 限制                           |
| ------------- | ------------ | ------------------------------ |
| `read_file`   | 读取文件     | 仅限工作区内的文件             |
| `write_file`  | 写入文件     | 仅限工作区内的文件             |
| `list_dir`    | 列出目录     | 仅限工作区内的目录             |
| `edit_file`   | 编辑文件     | 仅限工作区内的文件             |
| `append_file` | 追加文件     | 仅限工作区内的文件             |
| `exec`        | 执行命令     | 命令路径必须在工作区内         |

#### 额外的 Exec 保护

即使 `restrict_to_workspace: false`，`exec` 工具也会阻止以下危险命令：

* `rm -rf`、`del /f`、`rmdir /s` — 批量删除
* `format`、`mkfs`、`diskpart` — 磁盘格式化
* `dd if=` — 磁盘镜像
* 写入 `/dev/sd[a-z]` — 直接磁盘写入
* `shutdown`、`reboot`、`poweroff` — 系统关机
* Fork bomb `:(){ :|:& };:`

### 文件访问控制

| 配置键 | 类型 | 默认值 | 描述 |
|--------|------|--------|------|
| `tools.allow_read_paths` | string[] | `[]` | 允许在工作区外读取的额外路径 |
| `tools.allow_write_paths` | string[] | `[]` | 允许在工作区外写入的额外路径 |

### Exec 安全配置

| 配置键 | 类型 | 默认值 | 描述 |
|--------|------|--------|------|
| `tools.exec.allow_remote` | bool | `false` | 允许从远程渠道（Telegram/Discord 等）执行 exec 工具 |
| `tools.exec.enable_deny_patterns` | bool | `true` | 启用危险命令拦截 |
| `tools.exec.custom_deny_patterns` | string[] | `[]` | 自定义阻止的正则表达式模式 |
| `tools.exec.custom_allow_patterns` | string[] | `[]` | 自定义允许的正则表达式模式 |

> **安全提示：** Symlink 保护默认启用——所有文件路径在白名单匹配前都会通过 `filepath.EvalSymlinks` 解析，防止符号链接逃逸攻击。

#### 已知限制：构建工具的子进程

exec 安全守卫仅检查 PicoClaw 直接启动的命令行。它不会递归检查由 `make`、`go run`、`cargo`、`npm run` 或自定义构建脚本等开发工具产生的子进程。

这意味着顶层命令通过初始守卫检查后，仍可以编译或启动其他二进制文件。实际上，应将构建脚本、Makefile、包脚本和生成的二进制文件视为与直接 shell 命令同等级别的可执行代码进行审查。

对于高风险环境：

* 执行前审查构建脚本。
* 对编译并运行的工作流优先使用审批/手动审查。
* 如果需要比内置守卫更强的隔离，请在容器或虚拟机中运行 PicoClaw。

#### 错误示例

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (path outside working dir)}
```

```
[ERROR] tool: Tool execution failed
{tool=exec, error=Command blocked by safety guard (dangerous pattern detected)}
```

#### 禁用限制（安全风险）

如果需要 Agent 访问工作区外的路径：

**方法 1: 配置文件**

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**方法 2: 环境变量**

```bash
export PICOCLAW_AGENTS_DEFAULTS_RESTRICT_TO_WORKSPACE=false
```

> ⚠️ **警告**: 禁用此限制将允许 Agent 访问系统上的任何路径。仅在受控环境中谨慎使用。

#### 安全边界一致性

`restrict_to_workspace` 设置在所有执行路径中一致应用：

| 执行路径         | 安全边界                     |
| ---------------- | ---------------------------- |
| 主 Agent         | `restrict_to_workspace` ✅   |
| 子 Agent / Spawn | 继承相同限制 ✅              |
| 心跳任务         | 继承相同限制 ✅              |

所有路径共享相同的工作区限制——无法通过子 Agent 或定时任务绕过安全边界。

### 心跳 / 周期性任务 (Heartbeat)

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

| 特性             | 描述                                     |
| ---------------- | ---------------------------------------- |
| **spawn**        | 创建异步子 Agent，不阻塞主心跳进程       |
| **独立上下文**   | 子 Agent 拥有独立上下文，无会话历史      |
| **message tool** | 子 Agent 通过 message 工具直接与用户通信 |
| **非阻塞**       | spawn 后，心跳继续处理下一个任务         |

**配置：**

```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

| 选项       | 默认值 | 描述                         |
| ---------- | ------ | ---------------------------- |
| `enabled`  | `true` | 启用/禁用心跳                |
| `interval` | `30`   | 检查间隔，单位分钟 (最小: 5) |

**环境变量:**

- `PICOCLAW_HEARTBEAT_ENABLED=false` 禁用
- `PICOCLAW_HEARTBEAT_INTERVAL=60` 更改间隔

#### 子 Agent 通信流程

```
心跳触发
    ↓
Agent 读取 HEARTBEAT.md
    ↓
遇到耗时任务：spawn 子 Agent
    ↓                           ↓
继续处理下一个任务         子 Agent 独立运行
    ↓                           ↓
所有任务完成               子 Agent 使用 "message" 工具
    ↓                           ↓
回复 HEARTBEAT_OK          用户直接收到结果
```

子 Agent 拥有工具访问权限（message、web_search 等），可以独立与用户通信，无需经过主 Agent。

### Providers（模型提供商）

> [!NOTE]
> Groq 通过 Whisper 提供免费语音转录。配置后，任意渠道的语音消息都会在 Agent 层自动转录为文字。

| 提供商       | 用途                                    | 获取 API Key                                                 |
| ------------ | --------------------------------------- | ------------------------------------------------------------ |
| `gemini`     | LLM（Gemini 直连）                      | [aistudio.google.com](https://aistudio.google.com)           |
| `zhipu`      | LLM（智谱直连）                         | [bigmodel.cn](https://bigmodel.cn)                           |
| `volcengine` | LLM（火山引擎直连）                     | [volcengine.com](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| `openrouter` | LLM（推荐，可访问所有模型）             | [openrouter.ai](https://openrouter.ai)                       |
| `anthropic`  | LLM（Claude 直连）                      | [console.anthropic.com](https://console.anthropic.com)       |
| `openai`     | LLM（GPT 直连）                         | [platform.openai.com](https://platform.openai.com)           |
| `deepseek`   | LLM（DeepSeek 直连）                    | [platform.deepseek.com](https://platform.deepseek.com)       |
| `qwen`       | LLM（通义千问直连）                     | [dashscope.console.aliyun.com](https://dashscope.console.aliyun.com) |
| `groq`       | LLM + **语音转录**（Whisper）           | [console.groq.com](https://console.groq.com)                 |
| `cerebras`   | LLM（Cerebras 直连）                    | [cerebras.ai](https://cerebras.ai)                           |
| `vivgrid`    | LLM（Vivgrid 直连）                     | [vivgrid.com](https://vivgrid.com)                           |

### 模型配置 (model_list)

> **新特性：** PicoClaw 现在优先推荐显式 `provider` + 原生 `model` 的配置方式，例如 `"provider": "zhipu", "model": "glm-4.7"`。如果未设置 `provider`，旧的单字段 `provider/model` 写法仍然兼容。

这一设计同时支持**多 Agent**场景，灵活选择提供商：

- **不同 Agent 使用不同提供商**：每个 Agent 可以使用独立的 LLM 提供商
- **模型降级**：配置主模型和备用模型，提升可用性
- **负载均衡**：将请求分发到多个端点
- **集中管理**：在一处管理所有提供商配置

#### 所有支持的厂商

| 厂商                    | `provider` 值     | 默认 API Base                                       | 协议      | API Key                                                          |
| ----------------------- | ----------------- | --------------------------------------------------- | --------- | ---------------------------------------------------------------- |
| **OpenAI**              | `openai`          | `https://api.openai.com/v1`                         | OpenAI    | [获取](https://platform.openai.com)                              |
| **Anthropic**           | `anthropic`       | `https://api.anthropic.com/v1`                      | Anthropic | [获取](https://console.anthropic.com)                            |
| **智谱 AI (GLM)**       | `zhipu`           | `https://open.bigmodel.cn/api/paas/v4`              | OpenAI    | [获取](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys)    |
| **DeepSeek**            | `deepseek`        | `https://api.deepseek.com/v1`                       | OpenAI    | [获取](https://platform.deepseek.com)                            |
| **Google Gemini**       | `gemini`          | `https://generativelanguage.googleapis.com/v1beta`  | Gemini    | [获取](https://aistudio.google.com/api-keys)                     |
| **Groq**                | `groq`            | `https://api.groq.com/openai/v1`                    | OpenAI    | [获取](https://console.groq.com)                                 |
| **Moonshot**            | `moonshot`        | `https://api.moonshot.cn/v1`                        | OpenAI    | [获取](https://platform.moonshot.cn)                             |
| **通义千问 (Qwen)**     | `qwen`            | `https://dashscope.aliyuncs.com/compatible-mode/v1` | OpenAI    | [获取](https://dashscope.console.aliyun.com)                     |
| **NVIDIA**              | `nvidia`          | `https://integrate.api.nvidia.com/v1`               | OpenAI    | [获取](https://build.nvidia.com)                                 |
| **Ollama**              | `ollama`          | `http://localhost:11434/v1`                         | OpenAI    | 本地（无需 Key）                                                 |
| **LM Studio**           | `lmstudio`        | `http://localhost:1234/v1`                          | OpenAI    | 可选（本地默认无需密钥）                                         |
| **OpenRouter**          | `openrouter`      | `https://openrouter.ai/api/v1`                      | OpenAI    | [获取](https://openrouter.ai/keys)                               |
| **LiteLLM Proxy**       | `litellm`         | `http://localhost:4000/v1`                          | OpenAI    | 你的 LiteLLM 代理 Key                                            |
| **VLLM**                | `vllm`            | `http://localhost:8000/v1`                          | OpenAI    | 本地                                                             |
| **Cerebras**            | `cerebras`        | `https://api.cerebras.ai/v1`                        | OpenAI    | [获取](https://cerebras.ai)                                      |
| **火山引擎 (豆包)**     | `volcengine`      | `https://ark.cn-beijing.volces.com/api/v3`          | OpenAI    | [获取](https://www.volcengine.com/activity/codingplan?utm_campaign=PicoClaw&utm_content=PicoClaw&utm_medium=devrel&utm_source=OWO&utm_term=PicoClaw) |
| **神算云**              | `shengsuanyun`    | `https://router.shengsuanyun.com/api/v1`            | OpenAI    | —                                                                |
| **BytePlus**            | `byteplus`        | `https://ark.ap-southeast.bytepluses.com/api/v3`    | OpenAI    | [获取](https://www.byteplus.com)                                 |
| **Vivgrid**             | `vivgrid`         | `https://api.vivgrid.com/v1`                        | OpenAI    | [获取](https://vivgrid.com)                                      |
| **LongCat**             | `longcat`         | `https://api.longcat.chat/openai`                   | OpenAI    | [获取](https://longcat.chat/platform)                            |
| **ModelScope (魔搭)**   | `modelscope`      | `https://api-inference.modelscope.cn/v1`            | OpenAI    | [获取](https://modelscope.cn/my/tokens)                          |
| **Antigravity**         | `antigravity`     | Google Cloud                                        | Custom    | 仅 OAuth                                                         |
| **GitHub Copilot**      | `github-copilot`  | `localhost:4321`                                    | gRPC      | —                                                                |

#### 基础配置

```json
{
  "model_list": [
    {
      "model_name": "ark-code-latest",
      "provider": "volcengine",
      "model": "ark-code-latest",
      "api_keys": ["sk-your-api-key"]
    },
    {
      "model_name": "gpt-5.4",
      "provider": "openai",
      "model": "gpt-5.4",
      "api_keys": ["sk-your-openai-key"]
    },
    {
      "model_name": "claude-sonnet-4.6",
      "provider": "anthropic",
      "model": "claude-sonnet-4.6",
      "api_keys": ["sk-ant-your-key"]
    },
    {
      "model_name": "glm-4.7",
      "provider": "zhipu",
      "model": "glm-4.7",
      "api_keys": ["your-zhipu-key"]
    }
  ],
  "agents": {
    "defaults": {
      "model": "gpt-5.4"
    }
  }
}
```

解析规则：

- 推荐显式写成 `"provider": "openai", "model": "gpt-5.4"`。
- 如果设置了 `provider`，PicoClaw 会将 `model` 原样发送。
- 如果未设置 `provider`，PicoClaw 会把 `model` 第一个 `/` 之前的字段当作 provider，并把第一个 `/` 之后的全部内容当作最终模型 ID。
- 这意味着 `"model": "openrouter/openai/gpt-5.4"` 这样的兼容写法仍然可用，并会把 `openai/gpt-5.4` 发送给 OpenRouter。

#### 各厂商配置示例

<details>
<summary><b>OpenAI</b></summary>

```json
{
  "model_name": "gpt-5.4",
  "provider": "openai",
  "model": "gpt-5.4",
  "api_keys": ["sk-..."]
}
```

</details>

<details>
<summary><b>火山引擎（豆包）</b></summary>

```json
{
  "model_name": "ark-code-latest",
  "provider": "volcengine",
  "model": "ark-code-latest",
  "api_keys": ["sk-..."]
}
```

</details>

<details>
<summary><b>智谱 AI (GLM)</b></summary>

```json
{
  "model_name": "glm-4.7",
  "provider": "zhipu",
  "model": "glm-4.7",
  "api_keys": ["your-key"]
}
```

</details>

<details>
<summary><b>DeepSeek</b></summary>

```json
{
  "model_name": "deepseek-chat",
  "provider": "deepseek",
  "model": "deepseek-chat",
  "api_keys": ["sk-..."]
}
```

</details>

<details>
<summary><b>Anthropic</b></summary>

```json
{
  "model_name": "claude-sonnet-4.6",
  "provider": "anthropic",
  "model": "claude-sonnet-4.6",
  "api_keys": ["sk-ant-your-key"]
}
```

> 运行 `picoclaw auth login --provider anthropic` 粘贴 API Token。

如需直连 Anthropic 原生接口（不兼容 OpenAI 格式的端点）：

```json
{
  "model_name": "claude-opus-4-6",
  "provider": "anthropic-messages",
  "model": "claude-opus-4-6",
  "api_keys": ["sk-ant-your-key"],
  "api_base": "https://api.anthropic.com"
}
```

> 当端点不支持 OpenAI 兼容格式（`/v1/chat/completions`），需要 Anthropic 原生 `/v1/messages` 时使用 `anthropic-messages`。

</details>

<details>
<summary><b>Ollama（本地）</b></summary>

```json
{
  "model_name": "llama3",
  "provider": "ollama",
  "model": "llama3"
}
```

</details>

<details>
<summary><b>LM Studio（本地）</b></summary>

```json
{
  "model_name": "lmstudio-local",
  "provider": "lmstudio",
  "model": "openai/gpt-oss-20b"
}
```

`api_base` 默认是 `http://localhost:1234/v1`。除非你在 LM Studio 侧启用了认证，否则不需要配置 API Key。
显式设置 `provider` 后，PicoClaw 会把 `openai/gpt-oss-20b` 原样发送给 LM Studio。旧的兼容写法 `"model": "lmstudio/openai/gpt-oss-20b"` 在未设置 `provider` 时也会解析成相同的上游模型 ID。

</details>

<details>
<summary><b>自定义代理 / LiteLLM</b></summary>

```json
{
  "model_name": "my-custom-model",
  "provider": "openai",
  "model": "custom-model",
  "api_base": "https://my-proxy.com/v1",
  "api_keys": ["sk-..."]
}
```

显式设置 `provider` 后，PicoClaw 会将 `model` 原样发送。因此 `"provider": "litellm", "model": "lite-gpt4"` 会发送 `lite-gpt4`，而 `"provider": "litellm", "model": "openai/gpt-4o"` 会发送 `openai/gpt-4o`。旧的兼容写法 `litellm/lite-gpt4` 和 `litellm/openai/gpt-4o` 在未设置 `provider` 时也会得到相同结果。

</details>

#### 负载均衡

为同一模型名称配置多个端点，PicoClaw 会自动轮询：

```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "provider": "openai",
      "model": "gpt-5.4",
      "api_base": "https://api1.example.com/v1",
      "api_keys": ["sk-key1"]
    },
    {
      "model_name": "gpt-5.4",
      "provider": "openai",
      "model": "gpt-5.4",
      "api_base": "https://api2.example.com/v1",
      "api_keys": ["sk-key2"]
    }
  ]
}
```

#### 从旧版 `providers` 配置迁移

旧版 `providers` 配置**已废弃**，V2 中已移除。现有 V0/V1 配置会自动迁移。完整迁移指南见 [docs/migration/model-list-migration.md](../migration/model-list-migration.md)。

### Provider 架构

PicoClaw 按协议族路由提供商：

- **OpenAI 兼容**：OpenRouter、Groq、智谱、vLLM 风格端点及大多数其他提供商。
- **Gemini 原生**：Google Gemini 通过原生 `models/*:generateContent` 和 `models/*:streamGenerateContent` 端点接入。
- **Anthropic**：Claude 原生 API 行为。
- **Codex/OAuth**：OpenAI OAuth/Token 认证路由。

这使运行时保持轻量，同时让接入新的 OpenAI 兼容后端基本只需配置 `api_base` + `api_keys`。

<details>
<summary><b>智谱（旧版 providers 格式）</b></summary>

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
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
    }
  }
}
```

</details>

<details>
<summary><b>完整配置示例</b></summary>

```json
{
  "agents": {
    "defaults": {
      "model_name": "claude-opus-4-5"
    }
  },
  "session": {
    "dm_scope": "per-channel-peer",
    "backlog_limit": 20
  },
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx"
    },
    "groq": {
      "api_key": "gsk_xxx"
    }
  },
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "123456:ABC...",
      "allow_from": ["123456789"]
    }
  },
  "tools": {
    "web": {
      "duckduckgo": {
        "enabled": true,
        "max_results": 5
      }
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 30
  }
}
```

</details>

### 定时任务 / 提醒

PicoClaw 通过 `cron` 工具支持 cron 风格的定时任务。Agent 可以设置、列出和取消在指定时间触发的提醒或周期性任务。

```json
{
  "tools": {
    "cron": {
      "enabled": true,
      "exec_timeout_minutes": 5
    }
  }
}
```

定时任务在重启后持久保存，存储于 `~/.picoclaw/workspace/cron/`。

### 进阶主题

| 主题 | 说明 |
| ---- | ---- |
| [敏感数据过滤](../security/sensitive_data_filtering.zh.md) | 在发送给 LLM 前，从工具结果中过滤 API 密钥和令牌 |
| [Hook 系统](../architecture/hooks/README.zh.md) | 事件驱动 Hook：观察者、拦截器、审批 Hook |
| [Steering](../architecture/steering.md) | 在工具调用间向运行中的 Agent 注入消息 |
| [SubTurn](../architecture/subturn.md) | 子 Agent 协调、并发控制、生命周期管理 |
| [上下文管理](../architecture/agent-refactor/context.md) | 上下文边界检测、主动预算检查、压缩策略 |
