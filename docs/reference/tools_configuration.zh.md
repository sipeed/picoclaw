# 🔧 工具配置

> 返回 [README](../project/README.zh.md)

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

## 敏感数据过滤

在将工具结果发送给 LLM 之前，PicoClaw 可以从输出中过滤敏感值（API 密钥、令牌、密码）。这可以防止 LLM 看到自己的凭据。

详细说明请参阅[敏感数据过滤](../security/sensitive_data_filtering.zh.md)。

| 配置项 | 类型 | 默认值 | 描述 |
|--------|------|--------|------|
| `filter_sensitive_data` | bool | `true` | 启用/禁用过滤 |
| `filter_min_length` | int | `8` | 触发过滤的最小内容长度 |

## Web 工具

Web 工具用于网页搜索和抓取。

### Web Fetcher
用于抓取和处理网页内容的通用设置。

| 配置项              | 类型   | 默认值        | 描述                                                                                   |
|---------------------|--------|---------------|----------------------------------------------------------------------------------------|
| `enabled`           | bool   | true          | 启用网页抓取功能。                                                                     |
| `fetch_limit_bytes` | int    | 10485760      | 抓取网页负载的最大大小，单位为字节（默认 10MB）。                                      |
| `format`            | string | "plaintext"   | 抓取内容的输出格式。选项：`plaintext` 或 `markdown`（推荐）。                          |

### 百度搜索

使用[千帆 AI 搜索 API](https://cloud.baidu.com/doc/qianfan-api/s/Wmbq4z7e5)，国内访问稳定，中文搜索效果好。

| 配置项        | 类型   | 默认值                                                          | 描述                  |
|---------------|--------|----------------------------------------------------------------|-----------------------|
| `enabled`     | bool   | false                                                          | 启用百度搜索          |
| `api_key`     | string | -                                                              | 千帆 API 密钥         |
| `base_url`    | string | `https://qianfan.baidubce.com/v2/ai_search/web_search`        | 百度搜索 API URL      |
| `max_results` | int    | 10                                                             | 最大结果数            |

```json
{
  "tools": {
    "web": {
      "baidu_search": {
        "enabled": true,
        "api_key": "YOUR_BAIDU_QIANFAN_API_KEY",
        "max_results": 10
      }
    }
  }
}
```

### Tavily

| 配置项        | 类型   | 默认值 | 描述                              |
|---------------|--------|--------|-----------------------------------|
| `enabled`     | bool   | false  | 启用 Tavily 搜索                  |
| `api_key`     | string | -      | Tavily API 密钥                   |
| `base_url`    | string | -      | 自定义 Tavily API 基础 URL        |
| `max_results` | int    | 0      | 最大结果数（0 = 默认）            |

### GLM Search

| 配置项          | 类型   | 默认值                                               | 描述                  |
|-----------------|--------|------------------------------------------------------|-----------------------|
| `enabled`       | bool   | false                                                | 启用 GLM 搜索        |
| `api_key`       | string | -                                                    | GLM API 密钥          |
| `base_url`      | string | `https://open.bigmodel.cn/api/paas/v4/web_search`   | GLM Search API URL    |
| `search_engine` | string | `search_std`                                         | 搜索引擎类型          |
| `max_results`   | int    | 5                                                    | 最大结果数            |

### DuckDuckGo

> ⚠️ 国内访问困难，建议搭配代理使用。

| 配置项        | 类型 | 默认值 | 描述                  |
|---------------|------|--------|-----------------------|
| `enabled`     | bool | true   | 启用 DuckDuckGo 搜索  |
| `max_results` | int  | 5      | 最大结果数            |

### Perplexity

> ⚠️ 国内访问困难，建议搭配代理使用。

| 配置项        | 类型     | 默认值 | 描述                                           |
|---------------|----------|--------|------------------------------------------------|
| `enabled`     | bool     | false  | 启用 Perplexity 搜索                           |
| `api_key`     | string   | -      | Perplexity API 密钥                            |
| `api_keys`    | string[] | -      | 多个 API 密钥轮换（优先于 `api_key`）          |
| `max_results` | int      | 5      | 最大结果数                                     |

### Brave

> ⚠️ 国内访问困难，建议搭配代理使用。

| 配置项        | 类型     | 默认值 | 描述                                           |
|---------------|----------|--------|------------------------------------------------|
| `enabled`     | bool     | false  | 启用 Brave 搜索                                |
| `api_key`     | string   | -      | Brave Search API 密钥                          |
| `api_keys`    | string[] | -      | 多个 API 密钥轮换（优先于 `api_key`）          |
| `max_results` | int      | 5      | 最大结果数                                     |

### SearXNG

| 配置项        | 类型   | 默认值                   | 描述                  |
|---------------|--------|--------------------------|-----------------------|
| `enabled`     | bool   | false                    | 启用 SearXNG 搜索     |
| `base_url`    | string | `http://localhost:8888`  | SearXNG 实例 URL      |
| `max_results` | int    | 5                        | 最大结果数            |

### 其他 Web 设置

| 配置项                   | 类型     | 默认值 | 描述                                           |
|--------------------------|----------|--------|-------------------------------------------------|
| `prefer_native`          | bool     | true   | 优先使用 provider 原生搜索而非配置的搜索引擎    |
| `private_host_whitelist` | string[] | `[]`   | 允许 Web 抓取的私有/内部主机白名单              |

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
| `allow_command`        | bool | false  | 允许 cron 任务执行 shell 命令       |

<a id="mcp-tool"></a>
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
          "type": "slack",
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

<a id="skills-tool"></a>
## Skills 工具

Skills 工具配置通过 ClawHub 等注册表进行技能发现和安装。

### 注册表

| 配置项                             | 类型   | 默认值               | 描述                                 |
|------------------------------------|--------|----------------------|--------------------------------------|
| `registries.clawhub.enabled`       | bool   | true                 | 启用 ClawHub 注册表                  |
| `registries.clawhub.base_url`      | string | `https://clawhub.ai` | ClawHub 基础 URL                     |
| `registries.clawhub.auth_token`    | string | `""`                 | 可选的 Bearer 令牌，用于更高速率限制 |
| `registries.clawhub.search_path`   | string | `""`                 | 搜索 API 路径                        |
| `registries.clawhub.skills_path`   | string | `""`                 | Skills API 路径                      |
| `registries.clawhub.download_path` | string | `""`                 | 下载 API 路径                        |
| `registries.clawhub.timeout`       | int    | 0                    | 请求超时时间（秒），0 = 默认         |
| `registries.clawhub.max_zip_size`  | int    | 0                    | 技能 zip 最大大小（字节），0 = 默认  |
| `registries.clawhub.max_response_size` | int | 0                   | API 响应最大大小（字节），0 = 默认   |

### GitHub 集成

| 配置项           | 类型   | 默认值 | 描述                          |
|------------------|--------|--------|-------------------------------|
| `github.proxy`   | string | `""`   | GitHub API 请求的 HTTP 代理   |
| `github.token`   | string | `""`   | GitHub 个人访问令牌           |

### 搜索设置

| 配置项                     | 类型 | 默认值 | 描述                     |
|----------------------------|------|--------|--------------------------|
| `max_concurrent_searches`  | int  | 2      | 最大并发技能搜索请求数   |
| `search_cache.max_size`    | int  | 50     | 最大缓存搜索结果数       |
| `search_cache.ttl_seconds` | int  | 300    | 缓存 TTL（秒）           |

### 配置示例

```json
{
  "tools": {
    "skills": {
      "registries": {
        "clawhub": {
          "enabled": true,
          "base_url": "https://clawhub.ai",
          "auth_token": ""
        }
      },
      "github": {
        "proxy": "",
        "token": ""
      },
      "max_concurrent_searches": 2,
      "search_cache": {
        "max_size": 50,
        "ttl_seconds": 300
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

## Skills Tool

Skills 工具用于通过仓库源发现和安装 Skill，支持 ClawHub 与 GitHub。

### Registries

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `registries.clawhub.enabled` | bool | true | 是否启用 ClawHub |
| `registries.clawhub.base_url` | string | `https://clawhub.ai` | ClawHub 基础地址 |
| `registries.clawhub.auth_token` | string | `""` | ClawHub 认证令牌 |
| `registries.github.enabled` | bool | true | 是否启用 GitHub |
| `registries.github.base_url` | string | `https://github.com` | GitHub 或 GitHub Enterprise 基础地址 |
| `registries.github.auth_token` | string | `""` | GitHub 访问令牌 |
| `registries.github.proxy` | string | `""` | GitHub 请求代理 |

### 旧版 GitHub 配置

`github.*` 已废弃，建议迁移到 `registries.github.*`。当前仍保留兼容，后续可移除。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `github.base_url` | string | `https://github.com` | 已废弃 |
| `github.proxy` | string | `""` | 已废弃 |
| `github.token` | string | `""` | 已废弃 |
