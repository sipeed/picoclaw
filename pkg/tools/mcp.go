package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sipeed/picoclaw/pkg/config"
)

const (
	defaultMCPStartupTimeout = 8 * time.Second
	defaultMCPCallTimeout    = 30 * time.Second
	defaultMCPTerminateWait  = 1 * time.Second
	maxToolNameLength        = 64
)

var toolNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// LoadMCPTools discovers tools from configured MCP servers and returns them as local tools.
// Discovery is best-effort across servers: individual server failures are aggregated in the returned error.
func LoadMCPTools(ctx context.Context, cfg config.MCPToolsConfig, workspace string) ([]Tool, error) {
	if !cfg.Enabled || len(cfg.Servers) == 0 {
		return nil, nil
	}

	usedNames := make(map[string]int)
	loaded := make([]Tool, 0)
	errs := make([]error, 0)

	for _, serverCfg := range cfg.Servers {
		serverTools, err := loadMCPServerTools(ctx, serverCfg, workspace, usedNames)
		loaded = append(loaded, serverTools...)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return loaded, errors.Join(errs...)
}

func loadMCPServerTools(ctx context.Context, serverCfg config.MCPServerConfig, workspace string, usedNames map[string]int) ([]Tool, error) {
	if !serverCfg.Enabled {
		return nil, nil
	}

	client := newMCPClient(serverCfg, workspace)
	startupTimeout := durationFromMS(serverCfg.StartupTimeoutMS, defaultMCPStartupTimeout)

	connectCtx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()

	remoteTools, err := client.ListTools(connectCtx)
	if err != nil {
		return nil, fmt.Errorf("mcp server %q discovery failed: %w", serverCfg.Name, err)
	}

	callTimeout := durationFromMS(serverCfg.CallTimeoutMS, defaultMCPCallTimeout)
	loaded := make([]Tool, 0, len(remoteTools))
	for _, rt := range remoteTools {
		if rt == nil || strings.TrimSpace(rt.Name) == "" {
			continue
		}

		loaded = append(loaded, &MCPTool{
			localName:   buildLocalToolName(serverCfg, rt.Name, usedNames),
			remoteName:  rt.Name,
			description: buildMCPToolDescription(serverCfg.Name, rt.Name, rt.Description),
			parameters:  normalizeMCPInputSchema(rt.InputSchema),
			callTimeout: callTimeout,
			client:      client,
		})
	}

	return loaded, nil
}

type MCPTool struct {
	localName   string
	remoteName  string
	description string
	parameters  map[string]interface{}
	callTimeout time.Duration
	client      *mcpClient
}

func (t *MCPTool) Name() string {
	return t.localName
}

func (t *MCPTool) Description() string {
	return t.description
}

func (t *MCPTool) Parameters() map[string]interface{} {
	return t.parameters
}

func (t *MCPTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	callCtx := ctx
	if t.callTimeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, t.callTimeout)
		defer cancel()
	}
	return t.client.CallTool(callCtx, t.remoteName, args)
}

type mcpClient struct {
	cfg       config.MCPServerConfig
	workspace string
	client    *mcp.Client
}

func newMCPClient(cfg config.MCPServerConfig, workspace string) *mcpClient {
	implName := strings.TrimSpace(cfg.Name)
	if implName == "" {
		implName = "picoclaw-mcp"
	}
	return &mcpClient{
		cfg:       cfg,
		workspace: workspace,
		client: mcp.NewClient(&mcp.Implementation{
			Name:    "picoclaw-" + sanitizeToolName(implName),
			Version: "v0.1.0",
		}, nil),
	}
}

func (c *mcpClient) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	session, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	all := make([]*mcp.Tool, 0)
	cursor := ""
	for {
		params := &mcp.ListToolsParams{}
		if cursor != "" {
			params.Cursor = cursor
		}
		res, err := session.ListTools(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("list tools: %w", err)
		}
		all = append(all, res.Tools...)
		if res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}
	return all, nil
}

func (c *mcpClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	session, err := c.connect(ctx)
	if err != nil {
		return "", err
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		return "", fmt.Errorf("call tool %q: %w", toolName, err)
	}

	return formatMCPCallToolResult(result)
}

func (c *mcpClient) connect(ctx context.Context) (*mcp.ClientSession, error) {
	transport, err := c.buildTransport()
	if err != nil {
		return nil, err
	}
	session, err := c.client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect mcp server %q: %w", c.cfg.Name, err)
	}
	return session, nil
}

func (c *mcpClient) buildTransport() (mcp.Transport, error) {
	transport := strings.ToLower(strings.TrimSpace(c.cfg.Transport))
	if transport == "" {
		transport = "command"
	}

	switch transport {
	case "command":
		return c.buildCommandTransport()
	case "streamable_http":
		endpoint, err := c.requiredServerURL("streamable_http")
		if err != nil {
			return nil, err
		}
		return &mcp.StreamableClientTransport{
			Endpoint: endpoint,
		}, nil
	case "sse":
		endpoint, err := c.requiredServerURL("sse")
		if err != nil {
			return nil, err
		}
		return &mcp.SSEClientTransport{
			Endpoint: endpoint,
		}, nil
	default:
		return nil, fmt.Errorf("mcp server %q: unsupported transport %q", c.cfg.Name, c.cfg.Transport)
	}
}

func (c *mcpClient) buildCommandTransport() (mcp.Transport, error) {
	command := strings.TrimSpace(c.cfg.Command)
	if command == "" {
		return nil, fmt.Errorf("mcp server %q: command is required for command transport", c.cfg.Name)
	}

	cmd := exec.Command(command, c.cfg.Args...)
	if wd := resolvePath(c.cfg.WorkingDir, c.workspace); wd != "" {
		cmd.Dir = wd
	}
	if len(c.cfg.Env) > 0 {
		cmd.Env = mergeEnv(os.Environ(), c.cfg.Env)
	}
	cmd.Stderr = os.Stderr

	tr := &mcp.CommandTransport{
		Command: cmd,
	}
	tr.TerminateDuration = durationFromMS(c.cfg.TerminateTimeoutMS, defaultMCPTerminateWait)
	return tr, nil
}

func (c *mcpClient) requiredServerURL(transport string) (string, error) {
	endpoint := strings.TrimSpace(c.cfg.URL)
	if endpoint == "" {
		return "", fmt.Errorf("mcp server %q: url is required for %s transport", c.cfg.Name, transport)
	}
	return endpoint, nil
}

func formatMCPCallToolResult(result *mcp.CallToolResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("empty MCP response")
	}

	textOnly, ok := singleTextResult(result)
	if ok && result.StructuredContent == nil {
		if result.IsError {
			return "MCP tool error: " + textOnly, nil
		}
		return textOnly, nil
	}

	out := map[string]interface{}{
		"is_error": result.IsError,
	}
	if len(result.Content) > 0 {
		out["content"] = result.Content
	}
	if result.StructuredContent != nil {
		out["structured_content"] = result.StructuredContent
	}

	if len(out) == 1 && !result.IsError {
		return "(empty MCP tool response)", nil
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal MCP tool response: %w", err)
	}
	return string(data), nil
}

func singleTextResult(result *mcp.CallToolResult) (string, bool) {
	if result == nil || len(result.Content) != 1 {
		return "", false
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return "", false
	}
	return tc.Text, true
}

func buildLocalToolName(serverCfg config.MCPServerConfig, remoteToolName string, used map[string]int) string {
	prefix := strings.TrimSpace(serverCfg.ToolPrefix)
	if prefix == "" {
		baseServer := sanitizeToolName(serverCfg.Name)
		if baseServer == "" {
			baseServer = "server"
		}
		prefix = "mcp_" + baseServer
	}

	base := sanitizeToolName(prefix + "_" + remoteToolName)
	if base == "" {
		base = "mcp_tool"
	}

	candidate := truncateToolName(base)
	if used[candidate] == 0 {
		used[candidate] = 1
		return candidate
	}

	for i := 2; ; i++ {
		suffix := fmt.Sprintf("_%d", i)
		candidate = truncateWithSuffix(base, suffix)
		if used[candidate] == 0 {
			used[candidate] = 1
			return candidate
		}
	}
}

func buildMCPToolDescription(serverName, remoteName, rawDescription string) string {
	base := strings.TrimSpace(rawDescription)
	if base == "" {
		base = fmt.Sprintf("Call MCP tool %q.", remoteName)
	}

	serverName = strings.TrimSpace(serverName)
	if serverName == "" {
		return "[MCP] " + base
	}

	return fmt.Sprintf("[MCP %s/%s] %s", serverName, remoteName, base)
}

func normalizeMCPInputSchema(schema any) map[string]interface{} {
	fallback := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}

	if schema == nil {
		return fallback
	}

	var out map[string]interface{}
	switch v := schema.(type) {
	case map[string]interface{}:
		out = v
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fallback
		}
		if err := json.Unmarshal(data, &out); err != nil {
			return fallback
		}
	}

	if out == nil {
		return fallback
	}
	if _, ok := out["type"]; !ok {
		out["type"] = "object"
	}
	if out["type"] == "object" {
		if _, ok := out["properties"]; !ok {
			out["properties"] = map[string]interface{}{}
		}
	}
	return out
}

func sanitizeToolName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, " ", "_")
	name = toolNameSanitizer.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_-")
	return name
}

func truncateToolName(name string) string {
	if len(name) <= maxToolNameLength {
		return name
	}
	return name[:maxToolNameLength]
}

func truncateWithSuffix(base, suffix string) string {
	if len(suffix) >= maxToolNameLength {
		return suffix[len(suffix)-maxToolNameLength:]
	}
	maxBase := maxToolNameLength - len(suffix)
	if len(base) > maxBase {
		base = base[:maxBase]
	}
	return base + suffix
}

func durationFromMS(value int, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return time.Duration(value) * time.Millisecond
}

func resolvePath(path, workspace string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = expandHome(path)
	if filepath.IsAbs(path) {
		return path
	}
	if workspace != "" {
		return filepath.Join(workspace, path)
	}
	return path
}

func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if len(path) == 1 {
		return home
	}
	if path[1] == '/' {
		return home + path[1:]
	}
	return path
}

func mergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	merged := append([]string{}, base...)
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		merged = append(merged, fmt.Sprintf("%s=%s", k, extra[k]))
	}
	return merged
}
