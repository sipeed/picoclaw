# Tools API Documentation

## Overview

PicoClaw's tools system provides a extensible way for the AI agent to interact with the host system, web, hardware, and external services. Tools are registered in a centralized `ToolRegistry` and can be exposed to LLM providers via JSON Schema definitions.

## Architecture

### Tool Interface

All tools implement the base `Tool` interface (`pkg/tools/registry.go`):

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]any  // JSON Schema
    Execute(ctx context.Context, args map[string]any) *ToolResult
}
```

Optional interfaces for enhanced behavior:

- **`AsyncExecutor`** - Tools that support async execution with callback
- **`mediaStoreAware`** - Tools that need access to media storage
- **`PromptMetadataProvider`** - Tools that provide prompt layer/slot metadata

### Tool Registry

The `ToolRegistry` (`pkg/tools/registry.go`) manages all tools:

- **Core Tools**: Registered with `Register(tool)` - always available, no TTL
- **Hidden Tools**: Registered with `RegisterHidden(tool)` - have TTL (Time To Live), can be promoted
- **Tool Definitions**: `GetDefinitions()` returns JSON Schema for LLM providers
- **Provider Format**: `ToProviderDefs()` converts to provider-specific format (OpenAI, Anthropic, etc.)

### Tool Execution Flow

1. Tool called by agent with arguments
2. Arguments validated against tool's JSON Schema
3. Channel/ChatID context injected into `ctx`
4. `Execute()` or `ExecuteAsync()` called
5. Result normalized and returned as `ToolResult`
6. Panics recovered to prevent agent crashes

## Available Tools

### Filesystem Tools

| Tool Name | Description | Category | Config Key |
|-----------|-------------|----------|------------|
| `read_file` | Read file content from workspace or allowed paths | filesystem | `read_file` |
| `write_file` | Create or overwrite files within workspace | filesystem | `write_file` |
| `list_dir` | Inspect directories and enumerate files | filesystem | `list_dir` |
| `edit_file` | Apply targeted edits to existing files | filesystem | `edit_file` |
| `append_file` | Append content to end of existing file | filesystem | `append_file` |

**Implementation**: `pkg/tools/fs/` package
- Path validation against workspace restrictions
- Symlink resolution to prevent escaping workspace
- Configurable allow/deny path patterns
- Max file size limit: 64KB (configurable via `MaxReadFileSize`)

**Expected Arguments** (example for `read_file`):
```json
{
  "path": "/path/to/file.txt",
  "workspace": "/workspace"  // injected automatically
}
```

### Shell/Exec Tool

| Tool Name | Description | Category | Config Key |
|-----------|-------------|----------|------------|
| `exec` | Run shell commands in workspace sandbox | filesystem | `exec` |

**Implementation**: `pkg/tools/shell.go`
- **Security**: Deny patterns block dangerous commands:
  - `rm -rf`, `dd`, `shutdown`, `reboot`, `chmod`, `chown`, `sudo`
  - Command substitution: `$(...)`, backticks
  - Pipe to shell: `\| sh`, `\| bash`
- **Session Management**: Persistent shell sessions via `SessionManager`
- **Timeout**: Configurable command timeout
- **Working Directory**: Restricted to workspace by default

**Expected Arguments**:
```json
{
  "command": "ls -la",
  "workdir": "/workspace",  // optional
  "timeout": 30  // seconds, optional
}
```

### Automation Tools

| Tool Name | Description | Category | Config Key |
|-----------|-------------|----------|------------|
| `cron` | Schedule one-time or recurring tasks | automation | `cron` |

**Implementation**: `pkg/tools/cron.go`
- Schedule reminders, shell commands, and jobs
- One-time or recurring (cron expression support)

**Expected Arguments**:
```json
{
  "action": "add",  // add, list, remove
  "schedule": "0 9 * * *",  // cron format or "in 5m"
  "command": "echo 'reminder'",  // optional
  "message": "Daily reminder"  // optional
}
```

### Web Tools

| Tool Name | Description | Category | Config Key |
|-----------|-------------|----------|------------|
| `web_search` | Search the web using configured providers | web | `web` |
| `web_fetch` | Fetch and summarize webpage contents | web | `web_fetch` |

**Web Search Providers** (configured in `tools.web`):
- **Sogou** - Chinese search engine
- **DuckDuckGo** - Privacy-focused search
- **Brave Search** - Independent search (requires API key)
- **Tavily** - AI-optimized search (requires API key)
- **Perplexity** - AI search engine (requires API key)
- **SearXNG** - Metasearch engine (self-hosted)
- **GLM Search** - Chinese AI search (requires API key)
- **Baidu Search** - Chinese search engine (requires API key)

**Web Search Expected Arguments**:
```json
{
  "query": "latest AI news",
  "max_results": 5,  // optional, default varies by provider
  "provider": "brave"  // optional, uses default
}
```

**Web Fetch Expected Arguments**:
```json
{
  "url": "https://example.com",
  "max_chars": 10000  // optional
}
```

### Communication Tools

| Tool Name | Description | Category | Config Key |
|-----------|-------------|----------|------------|
| `message` | Send follow-up message to active chat | communication | `message` |
| `send_file` | Send file or media to active chat | communication | `send_file` |

**Implementation**: `pkg/tools/integration_facade.go` → `pkg/tools/integration/`

**Message Tool Expected Arguments**:
```json
{
  "text": "Hello from the agent!",
  "channel": "telegram",  // injected from context
  "chat_id": "123456"  // injected from context
}
```

**Send File Expected Arguments**:
```json
{
  "path": "/workspace/report.pdf",
  "caption": "Here's your file"  // optional
}
```

### Skills Tools

| Tool Name | Description | Category | Config Key |
|-----------|-------------|----------|------------|
| `find_skills` | Search external skill registries | skills | `find_skills` |
| `install_skill` | Install skill from registry | skills | `install_skill` |

**Dependencies**: Requires `skills` config to be enabled

**Find Skills Expected Arguments**:
```json
{
  "query": "pdf",
  "limit": 10  // optional
}
```

**Install Skill Expected Arguments**:
```json
{
  "name": "pdf-tools",
  "source": "registry-url"  // optional
}
```

### Agent/Subagent Tools

| Tool Name | Description | Category | Config Key |
|-----------|-------------|----------|------------|
| `spawn` | Launch background subagent for delegated work | agents | `spawn` |
| `spawn_status` | Query status of spawned subagents | agents | `spawn_status` |

**Dependencies**: Requires `subagent` config to be enabled

**Spawn Tool Expected Arguments**:
```json
{
  "task": "Research latest AI papers",
  "model": "gpt-4",  // optional
  "max_tokens": 2000,  // optional
  "temperature": 0.7,  // optional
  "async": true  // optional, run in background
}
```

**Spawn Status Expected Arguments**:
```json
{
  "task_id": "abc123"  // optional, returns specific task or all
}
```

### Hardware Tools

| Tool Name | Description | Category | Config Key | Platform |
|-----------|-------------|----------|------------|----------|
| `i2c` | Interact with I2C devices | hardware | `i2c` | Linux only |
| `spi` | Interact with SPI devices | hardware | `spi` | Linux only |
| `serial` | Interact with serial ports | hardware | `serial` | Linux/macOS/Windows |

**Implementation**: `pkg/tools/hardware_facade.go` → `pkg/tools/hardware/`

**I2C Expected Arguments**:
```json
{
  "action": "read",  // read, write
  "bus": "/dev/i2c-1",
  "address": 0x48,
  "register": 0x00,  // optional
  "data": [0x01, 0x02]  // for write
}
```

**Serial Expected Arguments**:
```json
{
  "port": "/dev/ttyUSB0",
  "baud": 9600,
  "data": "hello"  // string or bytes
}
```

### Discovery Tools (Hidden, TTL-based)

| Tool Name | Description | Category | Config Key |
|-----------|-------------|----------|------------|
| `tool_search_tool_regex` | Discover hidden MCP tools by regex | discovery | `mcp.discovery.use_regex` |
| `tool_search_tool_bm25` | Discover hidden MCP tools by semantics | discovery | `mcp.discovery.use_bm25` |
| `request_permission` | Request user permission for outside-workspace access | permission | `exec.ask_permission` |

**Dependencies**: Requires `mcp` and `mcp.discovery` to be enabled

### Permission Tools

| Tool Name | Description | Category | Config Key |
|-----------|-------------|----------|------------|
| `request_permission` | Request user permission for outside-workspace access | permission | `exec.ask_permission` |

## Backend API Endpoints

### Base URL
```
http://localhost:<port>/api
```

### Tool Management

#### List All Tools
```
GET /api/tools
```

**Response**:
```json
{
  "tools": [
    {
      "name": "read_file",
      "description": "Read file content from the workspace",
      "category": "filesystem",
      "config_key": "read_file",
      "status": "enabled",  // enabled, disabled, blocked
      "reason_code": ""  // e.g., "requires_skills"
    }
  ]
}
```

#### Enable/Disable Tool
```
PUT /api/tools/{name}/state
```

**Request Body**:
```json
{
  "enabled": true
}
```

**Response**:
```json
{
  "status": "ok"
}
```

### Web Search Configuration

#### Get Web Search Config
```
GET /api/tools/web-search-config
```

**Response**:
```json
{
  "provider": "auto",  // auto, sogou, duckduckgo, brave, tavily, etc.
  "current_service": "brave",
  "prefer_native": false,
  "proxy": "",
  "providers": [
    {
      "id": "brave",
      "label": "Brave Search",
      "configured": true,
      "current": true,
      "requires_auth": true
    }
  ],
  "settings": {
    "brave": {
      "enabled": true,
      "max_results": 10,
      "api_key_set": true
    }
  }
}
```

#### Update Web Search Config
```
PUT /api/tools/web-search-config
```

**Request Body**:
```json
{
  "provider": "brave",
  "prefer_native": false,
  "proxy": "",
  "settings": {
    "brave": {
      "enabled": true,
      "max_results": 10,
      "api_key": "BSA...",  // or "api_keys": ["key1", "key2"]
      "base_url": ""  // optional for self-hosted
    }
  }
}
```

## Tool Result Structure

Tools return `*ToolResult` with the following fields:

```go
type ToolResult struct {
    ForLLM      string        // Text returned to LLM for processing
    ForUser     string        // Text shown to end user in chat
    MediaURLs   []string      // Media attachment URLs (media:// or http://)
    IsError     bool          // Whether execution failed
    Async       bool          // True if running asynchronously
    Err         error         // Underlying Go error (not serialized to JSON)
}
```

## MCP (Model Context Protocol) Integration

PicoClaw exposes tools via MCP server (`pkg/mcp/manager.go`):

- External MCP servers can be integrated
- Tools from MCP servers appear as hidden tools with TTL
- Discovery tools (`tool_search_tool_regex`, `tool_search_tool_bm25`) make hidden tools available
- MCP manager handles tool execution via isolated command transport

**MCP Tool Discovery Flow**:
1. MCP server registered with PicoClaw
2. Tools exposed as hidden (TTL=0, not visible to LLM)
3. Agent uses `tool_search_tool_regex` or `tool_search_tool_bm25`
4. Matching tools promoted (TTL set >0)
5. Promoted tools appear in next LLM context

## Configuration

Tools configured in `config.json` under `tools` section:

```json
{
  "tools": {
    "read_file": {"enabled": true},
    "write_file": {"enabled": true},
    "exec": {"enabled": true},
    "web": {
      "enabled": true,
      "provider": "brave",
      "brave": {
        "enabled": true,
        "max_results": 10,
        "api_keys": ["BSA..."]
      }
    },
    "mcp": {
      "enabled": true,
      "discovery": {
        "enabled": true,
        "use_regex": true,
        "use_bm25": false
      }
    }
  }
}
```

## Security Considerations

1. **Path Restrictions**: Filesystem tools restrict access to workspace by default
2. **Shell Command Filtering**: Dangerous commands blocked via regex patterns
3. **Tool TTL**: Hidden tools auto-expire to prevent context bloat
4. **Media Store**: File paths converted to `media://` URLs for safe transport
5. **Panic Recovery**: Tool panics recovered to prevent agent crashes
6. **Symlink Resolution**: Prevents escaping workspace via symlinks

## Tool Registration Example

```go
// Register a core tool (always available)
registry.Register(tools.NewReadFileTool(workspace, true, 64*1024))

// Register a hidden tool (TTL-based)
registry.RegisterHidden(tools.NewRegexSearchTool(registry, 5, 10))

// Promote hidden tools (make them available to LLM)
registry.PromoteTools([]string{"tool_search_tool_regex"}, 10)  // TTL=10 turns
```

## Agent Management API

PicoClaw provides REST API endpoints for managing custom agents in the cockpit. Agents are stored as Markdown files with YAML frontmatter in `~/.picoclaw/workspace/agents/`.

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/agents` | List all agents |
| GET | `/api/agent?slug={slug}` | Get agent by slug |
| POST | `/api/agent/create` | Create new agent |
| PUT | `/api/agent/update?slug={slug}` | Update agent |
| DELETE | `/api/agent/delete?slug={slug}` | Delete agent |
| POST | `/api/agent/import` | Import agent from Markdown content |

### Data Types

```typescript
interface Agent {
  slug: string
  name: string
  description: string
  system_prompt: string
  model: string
  tool_permissions: string[]
  status: "enabled" | "disabled"
  created_at: string
  updated_at: string
}

interface AgentCreateRequest {
  name: string
  description?: string
  system_prompt: string
  model: string
  tool_permissions?: string[]
}
```

### Agent File Format

Agents are stored as `.md` files with YAML frontmatter:

```markdown
---
name: researcher
description: Research assistant agent
model: claude-3-5-sonnet
slug: researcher
---

You are a research assistant specialized in finding and summarizing information...
```
