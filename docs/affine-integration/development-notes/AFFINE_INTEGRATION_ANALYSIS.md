# Affine Integration Analysis for PicoClaw

## Executive Summary

This document analyzes the PicoClaw architecture and provides a comprehensive plan for integrating Affine workspace management capabilities while maintaining the project's structure and leveraging existing patterns.

## 1. PicoClaw Architecture Overview

### 1.1 Core Components

**Agent System** (`pkg/agent/`)
- `AgentInstance`: Main agent with workspace, session manager, context builder, and tool registry
- `ContextBuilder`: Manages agent context from workspace files
- Agent loop handles tool execution and LLM interactions

**Tool System** (`pkg/tools/`)
- `ToolRegistry`: Central registry for all tools with thread-safe registration
- `Tool` interface: All tools implement `Name()`, `Description()`, `Parameters()`, `Execute()`
- Optional interfaces:
  - `ContextualTool`: Receives channel/chatID context
  - `AsyncTool`: Supports async execution with callbacks

**Provider System** (`pkg/providers/`)
- Abstraction layer for LLM providers (OpenAI, Anthropic, Gemini, etc.)
- Model-centric configuration via `model_list` in config
- Supports multiple protocols: OpenAI-compatible, Anthropic, custom

**Configuration** (`pkg/config/`)
- JSON-based configuration with environment variable overrides
- Tool configuration under `tools` section
- Extensible structure for new tool categories

### 1.2 Tool Implementation Pattern

Based on analysis of existing tools (`web.go`, `message.go`, `cron.go`):

```go
type MyTool struct {
    // Configuration fields
    apiKey string
    baseURL string
    
    // Optional context
    channel string
    chatID string
}

func NewMyTool(config MyToolConfig) *MyTool {
    return &MyTool{
        apiKey: config.APIKey,
        baseURL: config.BaseURL,
    }
}

func (t *MyTool) Name() string {
    return "my_tool"
}

func (t *MyTool) Description() string {
    return "Tool description for LLM"
}

func (t *MyTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "param1": map[string]any{
                "type": "string",
                "description": "Parameter description",
            },
        },
        "required": []string{"param1"},
    }
}

func (t *MyTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
    // Implementation
    return &ToolResult{
        ForLLM: "Result for LLM",
        ForUser: "Result for user",
    }
}

// Optional: Implement ContextualTool
func (t *MyTool) SetContext(channel, chatID string) {
    t.channel = channel
    t.chatID = chatID
}
```

## 2. Affine Overview

### 2.1 What is Affine?

Affine is an open-source, all-in-one workspace that combines:
- Note-taking and knowledge management
- Whiteboarding and visual collaboration
- Task management
- Real-time collaboration with CRDT sync

### 2.2 Affine API Capabilities

Based on research, Affine provides:

**GraphQL API** for:
- Workspace management (create, list, delete)
- Document/page operations (create, read, update, delete)
- Search operations across workspaces
- Comments and collaboration features
- Version history and content management
- User token management
- Publishing and access control

**Connection Methods**:
- WebSocket (for real-time operations)
- HTTP/HTTPS (for standard GraphQL queries)
- Self-hosted instances supported

### 2.3 Existing MCP Server

There's an existing Affine MCP server (`dawncr0w/affine-mcp-server`) that provides:
- Workspace management tools
- Document CRUD operations
- Search functionality
- Comment management
- Version history access
- Publishing controls

## 3. Integration Strategy

### 3.1 Design Principles

1. **Non-invasive**: Add Affine support without modifying core PicoClaw structure
2. **Modular**: Self-contained tool implementation following existing patterns
3. **Configurable**: Use existing config system for API credentials
4. **Consistent**: Match existing tool interfaces and conventions

### 3.2 Recommended Approach

**Option A: Native Go Implementation** (Recommended)
- Implement Affine GraphQL client in Go
- Create tools following PicoClaw patterns
- Full control over implementation
- Better performance and integration

**Option B: MCP Bridge**
- Wrap existing Affine MCP server
- Faster initial implementation
- Dependency on external MCP server
- Less control over behavior

**Recommendation**: Option A for better integration and maintainability

## 4. Implementation Plan

### 4.1 File Structure

```
pkg/tools/
├── affine.go              # Main Affine tool implementations
├── affine_client.go       # GraphQL client for Affine API
├── affine_test.go         # Unit tests
└── affine_types.go        # Type definitions

pkg/config/
└── config.go              # Add Affine config section
```

### 4.2 Configuration Schema

Add to `config.json`:

```json
{
  "tools": {
    "affine": {
      "enabled": true,
      "api_url": "https://app.affine.pro/graphql",
      "api_key": "YOUR_AFFINE_API_KEY",
      "workspace_id": "default-workspace-id",
      "timeout_seconds": 30
    }
  }
}
```

Add to `pkg/config/config.go`:

```go
type AffineConfig struct {
    Enabled        bool   `json:"enabled"         env:"PICOCLAW_TOOLS_AFFINE_ENABLED"`
    APIURL         string `json:"api_url"         env:"PICOCLAW_TOOLS_AFFINE_API_URL"`
    APIKey         string `json:"api_key"         env:"PICOCLAW_TOOLS_AFFINE_API_KEY"`
    WorkspaceID    string `json:"workspace_id"    env:"PICOCLAW_TOOLS_AFFINE_WORKSPACE_ID"`
    TimeoutSeconds int    `json:"timeout_seconds" env:"PICOCLAW_TOOLS_AFFINE_TIMEOUT_SECONDS"`
}

type ToolsConfig struct {
    Web    WebToolsConfig    `json:"web"`
    Cron   CronToolsConfig   `json:"cron"`
    Exec   ExecConfig        `json:"exec"`
    Skills SkillsToolsConfig `json:"skills"`
    Affine AffineConfig      `json:"affine"`  // Add this
}
```

### 4.3 Tool Implementation

Create multiple focused tools instead of one monolithic tool:

1. **affine_workspace** - Workspace management
   - list_workspaces
   - create_workspace
   - get_workspace_info

2. **affine_document** - Document operations
   - create_document
   - read_document
   - update_document
   - delete_document
   - list_documents

3. **affine_search** - Search operations
   - search_content
   - search_documents

4. **affine_collaborate** - Collaboration features
   - add_comment
   - list_comments
   - share_document
   - manage_permissions

### 4.4 GraphQL Client Implementation

```go
// pkg/tools/affine_client.go
package tools

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type AffineClient struct {
    apiURL     string
    apiKey     string
    httpClient *http.Client
}

type GraphQLRequest struct {
    Query     string                 `json:"query"`
    Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
    Data   json.RawMessage `json:"data"`
    Errors []GraphQLError  `json:"errors,omitempty"`
}

type GraphQLError struct {
    Message string `json:"message"`
    Path    []any  `json:"path,omitempty"`
}

func NewAffineClient(apiURL, apiKey string, timeout time.Duration) *AffineClient {
    return &AffineClient{
        apiURL: apiURL,
        apiKey: apiKey,
        httpClient: &http.Client{
            Timeout: timeout,
        },
    }
}

func (c *AffineClient) Query(ctx context.Context, query string, variables map[string]interface{}) (json.RawMessage, error) {
    reqBody := GraphQLRequest{
        Query:     query,
        Variables: variables,
    }
    
    bodyBytes, err := json.Marshal(reqBody)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(bodyBytes))
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("execute request: %w", err)
    }
    defer resp.Body.Close()
    
    var gqlResp GraphQLResponse
    if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }
    
    if len(gqlResp.Errors) > 0 {
        return nil, fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
    }
    
    return gqlResp.Data, nil
}
```

### 4.5 Example Tool: Workspace Management

```go
// pkg/tools/affine.go
package tools

import (
    "context"
    "encoding/json"
    "fmt"
)

type AffineWorkspaceTool struct {
    client *AffineClient
}

func NewAffineWorkspaceTool(config AffineConfig) *AffineWorkspaceTool {
    timeout := time.Duration(config.TimeoutSeconds) * time.Second
    if timeout == 0 {
        timeout = 30 * time.Second
    }
    
    return &AffineWorkspaceTool{
        client: NewAffineClient(config.APIURL, config.APIKey, timeout),
    }
}

func (t *AffineWorkspaceTool) Name() string {
    return "affine_workspace"
}

func (t *AffineWorkspaceTool) Description() string {
    return "Manage Affine workspaces. List, create, or get information about workspaces."
}

func (t *AffineWorkspaceTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "action": map[string]any{
                "type": "string",
                "enum": []string{"list", "create", "get"},
                "description": "Action to perform",
            },
            "workspace_id": map[string]any{
                "type": "string",
                "description": "Workspace ID (required for 'get' action)",
            },
            "name": map[string]any{
                "type": "string",
                "description": "Workspace name (required for 'create' action)",
            },
        },
        "required": []string{"action"},
    }
}

func (t *AffineWorkspaceTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
    action, ok := args["action"].(string)
    if !ok {
        return ErrorResult("action is required")
    }
    
    switch action {
    case "list":
        return t.listWorkspaces(ctx)
    case "create":
        name, ok := args["name"].(string)
        if !ok {
            return ErrorResult("name is required for create action")
        }
        return t.createWorkspace(ctx, name)
    case "get":
        workspaceID, ok := args["workspace_id"].(string)
        if !ok {
            return ErrorResult("workspace_id is required for get action")
        }
        return t.getWorkspace(ctx, workspaceID)
    default:
        return ErrorResult(fmt.Sprintf("unknown action: %s", action))
    }
}

func (t *AffineWorkspaceTool) listWorkspaces(ctx context.Context) *ToolResult {
    query := `
        query ListWorkspaces {
            workspaces {
                id
                name
                createdAt
                memberCount
            }
        }
    `
    
    data, err := t.client.Query(ctx, query, nil)
    if err != nil {
        return ErrorResult(fmt.Sprintf("failed to list workspaces: %v", err))
    }
    
    var result struct {
        Workspaces []struct {
            ID          string `json:"id"`
            Name        string `json:"name"`
            CreatedAt   string `json:"createdAt"`
            MemberCount int    `json:"memberCount"`
        } `json:"workspaces"`
    }
    
    if err := json.Unmarshal(data, &result); err != nil {
        return ErrorResult(fmt.Sprintf("failed to parse response: %v", err))
    }
    
    if len(result.Workspaces) == 0 {
        return &ToolResult{
            ForLLM: "No workspaces found",
            ForUser: "No workspaces found",
        }
    }
    
    output := "Workspaces:\n"
    for _, ws := range result.Workspaces {
        output += fmt.Sprintf("- %s (ID: %s, Members: %d)\n", ws.Name, ws.ID, ws.MemberCount)
    }
    
    return &ToolResult{
        ForLLM: output,
        ForUser: output,
    }
}

func (t *AffineWorkspaceTool) createWorkspace(ctx context.Context, name string) *ToolResult {
    query := `
        mutation CreateWorkspace($name: String!) {
            createWorkspace(name: $name) {
                id
                name
            }
        }
    `
    
    variables := map[string]interface{}{
        "name": name,
    }
    
    data, err := t.client.Query(ctx, query, variables)
    if err != nil {
        return ErrorResult(fmt.Sprintf("failed to create workspace: %v", err))
    }
    
    var result struct {
        CreateWorkspace struct {
            ID   string `json:"id"`
            Name string `json:"name"`
        } `json:"createWorkspace"`
    }
    
    if err := json.Unmarshal(data, &result); err != nil {
        return ErrorResult(fmt.Sprintf("failed to parse response: %v", err))
    }
    
    output := fmt.Sprintf("Created workspace '%s' (ID: %s)", result.CreateWorkspace.Name, result.CreateWorkspace.ID)
    
    return &ToolResult{
        ForLLM: output,
        ForUser: output,
    }
}

func (t *AffineWorkspaceTool) getWorkspace(ctx context.Context, workspaceID string) *ToolResult {
    query := `
        query GetWorkspace($id: ID!) {
            workspace(id: $id) {
                id
                name
                createdAt
                memberCount
                owner {
                    id
                    name
                }
            }
        }
    `
    
    variables := map[string]interface{}{
        "id": workspaceID,
    }
    
    data, err := t.client.Query(ctx, query, variables)
    if err != nil {
        return ErrorResult(fmt.Sprintf("failed to get workspace: %v", err))
    }
    
    var result struct {
        Workspace struct {
            ID          string `json:"id"`
            Name        string `json:"name"`
            CreatedAt   string `json:"createdAt"`
            MemberCount int    `json:"memberCount"`
            Owner       struct {
                ID   string `json:"id"`
                Name string `json:"name"`
            } `json:"owner"`
        } `json:"workspace"`
    }
    
    if err := json.Unmarshal(data, &result); err != nil {
        return ErrorResult(fmt.Sprintf("failed to parse response: %v", err))
    }
    
    ws := result.Workspace
    output := fmt.Sprintf(
        "Workspace: %s\nID: %s\nOwner: %s\nMembers: %d\nCreated: %s",
        ws.Name, ws.ID, ws.Owner.Name, ws.MemberCount, ws.CreatedAt,
    )
    
    return &ToolResult{
        ForLLM: output,
        ForUser: output,
    }
}
```

### 4.6 Tool Registration

Modify `pkg/agent/instance.go` to register Affine tools:

```go
func NewAgentInstance(
    agentCfg *config.AgentConfig,
    defaults *config.AgentDefaults,
    cfg *config.Config,
    provider providers.LLMProvider,
) *AgentInstance {
    // ... existing code ...
    
    // Register Affine tools if enabled
    if cfg.Tools.Affine.Enabled {
        toolsRegistry.Register(tools.NewAffineWorkspaceTool(cfg.Tools.Affine))
        toolsRegistry.Register(tools.NewAffineDocumentTool(cfg.Tools.Affine))
        toolsRegistry.Register(tools.NewAffineSearchTool(cfg.Tools.Affine))
        // Add more Affine tools as needed
    }
    
    // ... rest of existing code ...
}
```

## 5. Testing Strategy

### 5.1 Unit Tests

```go
// pkg/tools/affine_test.go
package tools

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestAffineWorkspaceTool_ListWorkspaces(t *testing.T) {
    // Mock client or use test server
    tool := NewAffineWorkspaceTool(AffineConfig{
        APIURL: "http://localhost:3000/graphql",
        APIKey: "test-key",
    })
    
    result := tool.Execute(context.Background(), map[string]any{
        "action": "list",
    })
    
    assert.False(t, result.IsError)
    assert.Contains(t, result.ForLLM, "Workspaces")
}
```

### 5.2 Integration Tests

- Test against local Affine instance
- Verify GraphQL queries work correctly
- Test error handling and edge cases

## 6. Documentation

### 6.1 User Documentation

Add to README.md:

```markdown
### Affine Integration

PicoClaw can integrate with Affine workspaces for note-taking and collaboration.

**Configuration:**

```json
{
  "tools": {
    "affine": {
      "enabled": true,
      "api_url": "https://app.affine.pro/graphql",
      "api_key": "YOUR_API_KEY",
      "workspace_id": "default-workspace"
    }
  }
}
```

**Available Tools:**
- `affine_workspace` - Manage workspaces
- `affine_document` - Create and edit documents
- `affine_search` - Search across workspaces

**Example Usage:**
```
User: Create a new workspace called "Project Alpha"
Agent: [Uses affine_workspace tool to create workspace]
```
```

### 6.2 Developer Documentation

Create `docs/tools/affine.md` with:
- Architecture overview
- GraphQL schema reference
- Tool implementation details
- Extension guidelines

## 7. Future Enhancements

### 7.1 Phase 2 Features

- Real-time collaboration via WebSocket
- Whiteboard/canvas operations
- Task management integration
- File attachment handling
- Advanced search with filters

### 7.2 Phase 3 Features

- Bidirectional sync with local workspace
- Affine as knowledge base for agent context
- Automated note-taking from conversations
- Integration with cron for scheduled updates

## 8. Security Considerations

1. **API Key Management**
   - Store API keys securely in config
   - Support environment variables
   - Never log API keys

2. **Access Control**
   - Respect Affine workspace permissions
   - Validate user authorization
   - Implement rate limiting

3. **Data Privacy**
   - Don't cache sensitive workspace data
   - Clear sensitive data from memory
   - Follow Affine's data policies

## 9. Performance Considerations

1. **Caching**
   - Cache workspace metadata
   - Implement TTL for cached data
   - Invalidate cache on updates

2. **Batch Operations**
   - Support bulk document operations
   - Minimize API calls
   - Use GraphQL efficiently

3. **Async Operations**
   - Use AsyncTool interface for long operations
   - Implement progress reporting
   - Handle timeouts gracefully

## 10. Conclusion

This integration plan provides a comprehensive approach to adding Affine support to PicoClaw while:

✅ Maintaining PicoClaw's architecture and patterns
✅ Following existing tool implementation conventions
✅ Providing modular, testable code
✅ Supporting both self-hosted and cloud Affine instances
✅ Enabling future enhancements

The implementation can be done incrementally:
1. Start with basic workspace and document tools
2. Add search and collaboration features
3. Implement advanced features based on user feedback

This approach ensures minimal disruption to the existing codebase while providing powerful Affine integration capabilities.
