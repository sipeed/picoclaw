package mcp

import "time"

const (
	defaultInitTimeoutSeconds = 60
	defaultCallTimeoutSeconds = 30
	defaultMaxResponseBytes   = 64 * 1024
	defaultScannerBufferBytes = 64 * 1024
	maxFrameBytes             = 2 * 1024 * 1024
	maxToolListPages          = 50
)

const (
	ProtocolMCPFrames = "mcp"
	ProtocolJSONLines = "jsonl"
)

// ServerConfig defines one MCP server connection.
type ServerConfig struct {
	Name               string
	Command            string
	Args               []string
	Env                map[string]string
	WorkingDir         string
	Protocol           string
	InitTimeoutSeconds int
	CallTimeoutSeconds int
	MaxResponseBytes   int
	IncludeTools       []string
	ExcludeTools       []string
}

func (c ServerConfig) InitTimeout() time.Duration {
	seconds := c.InitTimeoutSeconds
	if seconds <= 0 {
		seconds = defaultInitTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

func (c ServerConfig) CallTimeout() time.Duration {
	seconds := c.CallTimeoutSeconds
	if seconds <= 0 {
		seconds = defaultCallTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

func (c ServerConfig) ResponseLimit() int {
	if c.MaxResponseBytes <= 0 {
		return defaultMaxResponseBytes
	}
	return c.MaxResponseBytes
}

// RemoteTool is an MCP tool discovered from a server.
type RemoteTool struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// RegisteredTool is a discovered tool with a PicoClaw-facing qualified name.
type RegisteredTool struct {
	QualifiedName string
	ServerName    string
	ToolName      string
	Description   string
	Parameters    map[string]any
}

// CallResult is a normalized MCP tool call result.
type CallResult struct {
	Content string
	IsError bool
}
