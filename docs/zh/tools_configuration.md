# 🔧 工具配置

> 返回 [README](../../README.zh.md)

PicoClaw 的工具配置位于 `config.json` 的 `tools` 字段中。

## 目录结构

```json
{
  "tools": {
    "web": {
      ...
    },
    "mcp": {
      ...
    },
    "exec": {
      ...
    },
    "cron": {
      ...
    },
    "skills": {
      ...
    }
  }
}
```

## Web 工具

Web 工具用于网页搜索和抓取。

### Web Fetcher
用于抓取和处理网页内容的通用设置。

| 配置项              | 类型   | 默认值        | 描述                                                                                   |
|---------------------|--------|---------------|----------------------------------------------------------------------------------------|
| `enabled`           | bool   | true          | 启用网页抓取功能。                                                                     |
| `fetch_limit_bytes` | int    | 10485760      | 抓取网页负载的最大大小，单位为字节（默认 10MB）。                                      |
| `format`            | string | "plaintext"   | 抓取内容的输出格式。选项：`plaintext` 或 `markdown`（推荐）。                          |

### Brave

| 配置项        | 类型   | 默认值 | 描述               |
|---------------|--------|--------|--------------------|
| `enabled`     | bool   | false  | 启用 Brave 搜索    |
| `api_key`     | string | -      | Brave Search API 密钥 |
| `max_results` | int    | 5      | 最大结果数         |

### DuckDuckGo

| 配置项        | 类型 | 默认值 | 描述                  |
|---------------|------|--------|-----------------------|
| `enabled`     | bool | true   | 启用 DuckDuckGo 搜索  |
| `max_results` | int  | 5      | 最大结果数            |

### Perplexity

| 配置项        | 类型   | 默认值 | 描述                  |
|---------------|--------|--------|-----------------------|
| `enabled`     | bool   | false  | 启用 Perplexity 搜索  |
| `api_key`     | string | -      | Perplexity API 密钥   |
| `max_results` | int    | 5      | 最大结果数            |

## Exec 工具

Exec 工具用于执行 shell 命令。

| 配置项                 | 类型  | 默认值 | 描述                           |
|------------------------|-------|--------|--------------------------------|
| `enabled`              | bool  | true   | 启用 exec 工具                 |
| `enable_deny_patterns` | bool  | true   | 启用默认的危险命令拦截         |
| `custom_deny_patterns` | array | []     | 自定义拒绝模式（正则表达式）   |

### 禁用 Exec 工具

要完全禁用 `exec` 工具，请将 `enabled` 设置为 `false`：

**通过配置文件：**
```json
{
  "tools": {
    "exec": {
      "enabled": false
    }
  }
}
```

**通过环境变量：**
```bash
PICOCLAW_TOOLS_EXEC_ENABLED=false
```

> **注意：** 禁用后，代理将无法执行 shell 命令。这也会影响 Cron 工具运行计划 shell 命令的能力。

### 功能说明

- **`enable_deny_patterns`**：设为 `false` 可完全禁用默认的危险命令拦截模式
- **`custom_deny_patterns`**：添加自定义拒绝正则模式；匹配的命令将被拦截

### 默认拦截的命令模式

默认情况下，PicoClaw 会拦截以下危险命令：

- 删除命令：`rm -rf`、`del /f/q`、`rmdir /s`
- 磁盘操作：`format`、`mkfs`、`diskpart`、`dd if=`、写入 `/dev/sd*`
- 系统操作：`shutdown`、`reboot`、`poweroff`
- 命令替换：`$()`、`${}`、反引号
- 管道到 shell：`| sh`、`| bash`
- 权限提升：`sudo`、`chmod`、`chown`
- 进程控制：`pkill`、`killall`、`kill -9`
- 远程操作：`curl | sh`、`wget | sh`、`ssh`
- 包管理：`apt`、`yum`、`dnf`、`npm install -g`、`pip install --user`
- 容器：`docker run`、`docker exec`
- Git：`git push`、`git force`
- 其他：`eval`、`source *.sh`

### 已知架构限制

exec 守卫仅验证发送给 PicoClaw 的顶层命令。它**不会**递归检查该命令启动后由构建工具或脚本生成的子进程。

以下工作流在初始命令被允许后可以绕过直接命令守卫：

- `make run`
- `go run ./cmd/...`
- `cargo run`
- `npm run build`

这意味着守卫对于拦截明显危险的直接命令很有用，但它**不是**未审查构建管道的完整沙箱。如果你的威胁模型包括工作区中的不受信任代码，请使用更强的隔离措施，如容器、虚拟机或围绕构建和运行命令的审批流程。

### 配置示例

```json
{
  "tools": {
    "exec": {
      "enable_deny_patterns": true,
      "custom_deny_patterns": [
        "\\brm\\s+-r\\b",
        "\\bkillall\\s+python"
      ]
    }
  }
}
```

## Cron 工具

Cron 工具用于调度周期性任务。

| 配置项                 | 类型 | 默认值 | 描述                                |
|------------------------|------|--------|-------------------------------------|
| `exec_timeout_minutes` | int  | 5      | 执行超时时间（分钟），0 表示无限制  |

## MCP 工具

MCP 工具支持与外部 Model Context Protocol 服务器集成。

### 工具发现（延迟加载）

当连接多个 MCP 服务器时，同时暴露数百个工具可能会耗尽 LLM 的上下文窗口并增加 API 成本。**Discovery** 功能通过默认*隐藏* MCP 工具来解决此问题。

LLM 不会加载所有工具，而是获得一个轻量级搜索工具（使用 BM25 关键词匹配或正则表达式）。当 LLM 需要特定功能时，它会搜索隐藏的工具库。匹配的工具随后被临时"解锁"并注入上下文中，持续配置的轮数（`ttl`）。

### 全局配置

| 配置项      | 类型   | 默认值 | 描述                                 |
|-------------|--------|--------|--------------------------------------|
| `enabled`   | bool   | false  | 全局启用 MCP 集成                    |
| `discovery` | object | `{}`   | 工具发现配置（见下文）               |
| `servers`   | object | `{}`   | 服务器名称到服务器配置的映射         |

### Discovery 配置（`discovery`）

| 配置项               | 类型 | 默认值 | 描述                                                                                                          |
|----------------------|------|--------|---------------------------------------------------------------------------------------------------------------|
| `enabled`            | bool | false  | 如果为 true，MCP 工具将被隐藏并按需通过搜索加载。如果为 false，所有工具都会被加载                             |
| `ttl`                | int  | 5      | 已发现工具保持解锁状态的对话轮数                                                                              |
| `max_search_results` | int  | 5      | 每次搜索查询返回的最大工具数                                                                                  |
| `use_bm25`           | bool | true   | 启用自然语言/关键词搜索工具（`tool_search_tool_bm25`）。**警告**：比正则搜索消耗更多资源                       |
| `use_regex`          | bool | false  | 启用正则模式搜索工具（`tool_search_tool_regex`）                                                              |

> **注意：** 如果 `discovery.enabled` 为 `true`，你**必须**启用至少一个搜索引擎（`use_bm25` 或 `use_regex`），
> 否则应用程序将无法启动。

### 单服务器配置

| 配置项     | 类型   | 必需     | 描述                               |
|------------|--------|----------|------------------------------------|
| `enabled`  | bool   | 是       | 启用此 MCP 服务器                  |
| `type`     | string | 否       | 传输类型：`stdio`、`sse`、`http`   |
| `command`  | string | stdio    | stdio 传输的可执行命令             |
| `args`     | array  | 否       | stdio 传输的命令参数               |
| `env`      | object | 否       | stdio 进程的环境变量               |
| `env_file` | string | 否       | stdio 进程的环境文件路径           |
| `url`      | string | sse/http | `sse`/`http` 传输的端点 URL        |
| `headers`  | object | 否       | `sse`/`http` 传输的 HTTP 头        |

### 传输行为

- 如果省略 `type`，传输方式将自动检测：
    - 设置了 `url` → `sse`
    - 设置了 `command` → `stdio`
- `http` 和 `sse` 都使用 `url` + 可选的 `headers`。
- `env` 和 `env_file` 仅应用于 `stdio` 服务器。

### 配置示例

#### 1) Stdio MCP 服务器

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "filesystem": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-filesystem",
            "/tmp"
          ]
        }
      }
    }
  }
}
```

#### 2) 远程 SSE/HTTP MCP 服务器

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "remote-mcp": {
          "enabled": true,
          "type": "sse",
          "url": "https://example.com/mcp",
          "headers": {
            "Authorization": "Bearer YOUR_TOKEN"
          }
        }
      }
    }
  }
}
```

#### 3) 启用工具发现的大规模 MCP 设置

*在此示例中，LLM 只会看到 `tool_search_tool_bm25`。它将仅在用户请求时动态搜索并解锁 Github 或 Postgres 工具。*

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "discovery": {
        "enabled": true,
        "ttl": 5,
        "max_search_results": 5,
        "use_bm25": true,
        "use_regex": false
      },
      "servers": {
        "github": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-github"
          ],
          "env": {
            "GITHUB_PERSONAL_ACCESS_TOKEN": "YOUR_GITHUB_TOKEN"
          }
        },
        "postgres": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-postgres",
            "postgresql://user:password@localhost/dbname"
          ]
        },
        "slack": {
          "enabled": true,
          "command": "npx",
          "args": [
            "-y",
            "@modelcontextprotocol/server-slack"
          ],
          "env": {
            "SLACK_BOT_TOKEN": "YOUR_SLACK_BOT_TOKEN",
            "SLACK_TEAM_ID": "YOUR_SLACK_TEAM_ID"
          }
        }
      }
    }
  }
}
```

## Skills 工具

Skills 工具配置通过 ClawHub 等注册表进行技能发现和安装。

### 注册表

| 配置项                             | 类型   | 默认值               | 描述                                 |
|------------------------------------|--------|----------------------|--------------------------------------|
| `registries.clawhub.enabled`       | bool   | true                 | 启用 ClawHub 注册表                  |
| `registries.clawhub.base_url`      | string | `https://clawhub.ai` | ClawHub 基础 URL                     |
| `registries.clawhub.auth_token`    | string | `""`                 | 可选的 Bearer 令牌，用于更高速率限制 |
| `registries.clawhub.search_path`   | string | `/api/v1/search`     | 搜索 API 路径                        |
| `registries.clawhub.skills_path`   | string | `/api/v1/skills`     | Skills API 路径                      |
| `registries.clawhub.download_path` | string | `/api/v1/download`   | 下载 API 路径                        |

### 配置示例

```json
{
  "tools": {
    "skills": {
      "registries": {
        "clawhub": {
          "enabled": true,
          "base_url": "https://clawhub.ai",
          "auth_token": "",
          "search_path": "/api/v1/search",
          "skills_path": "/api/v1/skills",
          "download_path": "/api/v1/download"
        }
      }
    }
  }
}
```

## 环境变量

所有配置选项都可以通过格式为 `PICOCLAW_TOOLS_<SECTION>_<KEY>` 的环境变量覆盖：

例如：

- `PICOCLAW_TOOLS_WEB_BRAVE_ENABLED=true`
- `PICOCLAW_TOOLS_EXEC_ENABLED=false`
- `PICOCLAW_TOOLS_EXEC_ENABLE_DENY_PATTERNS=false`
- `PICOCLAW_TOOLS_CRON_EXEC_TIMEOUT_MINUTES=10`
- `PICOCLAW_TOOLS_MCP_ENABLED=true`

注意：嵌套的映射式配置（例如 `tools.mcp.servers.<name>.*`）在 `config.json` 中配置，而非通过环境变量。
