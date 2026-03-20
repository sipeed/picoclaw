# ⚙️ 配置指南

> 返回 [README](../../README.zh.md)

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

### 技能来源 (Skill Sources)

默认情况下，技能会按以下顺序加载：

1. `~/.picoclaw/workspace/skills`（工作区）
2. `~/.picoclaw/skills`（全局）
3. `<current-working-directory>/skills`（内置）

在高级/测试场景下，可通过以下环境变量覆盖内置技能目录：

```bash
export PICOCLAW_BUILTIN_SKILLS=/path/to/skills
```

### 统一命令执行策略

- 通用斜杠命令通过 `pkg/agent/loop.go` 中的 `commands.Executor` 统一执行。
- Channel 适配器不再在本地消费通用命令；它们只负责把入站文本转发到 bus/agent 路径。Telegram 仍会在启动时自动注册其支持的命令菜单。
- 未注册的斜杠命令（例如 `/foo`）会透传给 LLM 按普通输入处理。
- 已注册但当前 channel 不支持的命令（例如 WhatsApp 上的 `/show`）会返回明确的用户可见错误，并停止后续处理。

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
