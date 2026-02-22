package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// ServerInstance manages a connected MCP server session.
type ServerInstance struct {
	session  *sdkmcp.ClientSession
	done     chan struct{} // closed when session ends; created once per session
	tools    []*sdkmcp.Tool
	lastUsed time.Time
	crashes  []time.Time // track crash times for rate limiting
	isHTTP   bool
	mu       sync.Mutex
}

// Manager manages the lifecycle of MCP servers.
type Manager struct {
	configs map[string]config.MCPServerConfig
	servers map[string]*ServerInstance
	mu      sync.RWMutex
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewManager creates a new MCP Manager and starts the idle reaper.
func NewManager(configs map[string]config.MCPServerConfig) *Manager {
	if configs == nil {
		configs = make(map[string]config.MCPServerConfig)
	}

	m := &Manager{
		configs: configs,
		servers: make(map[string]*ServerInstance),
		stopCh:  make(chan struct{}),
	}

	// Start idle reaper goroutine
	m.wg.Add(1)
	go m.idleReaper()

	return m
}

// ListServers returns server names and descriptions without starting processes.
func (m *Manager) ListServers() []ServerSummary {
	m.mu.RLock()
	type entry struct {
		name string
		cfg  config.MCPServerConfig
		inst *ServerInstance
	}
	var entries []entry
	for name, cfg := range m.configs {
		if !cfg.Enabled {
			continue
		}
		entries = append(entries, entry{name: name, cfg: cfg, inst: m.servers[name]})
	}
	m.mu.RUnlock()

	var result []ServerSummary
	for _, e := range entries {
		status := "stopped"
		if e.inst != nil {
			e.inst.mu.Lock()
			if e.inst.session != nil {
				status = "running"
			}
			e.inst.mu.Unlock()
		}
		result = append(result, ServerSummary{
			Name:        e.name,
			Description: e.cfg.Description,
			Status:      status,
		})
	}
	return result
}

// GetTools returns the tool list for a server, starting it if needed.
func (m *Manager) GetTools(ctx context.Context, serverName string) ([]*sdkmcp.Tool, error) {
	inst, err := m.ensureRunning(ctx, serverName)
	if err != nil {
		return nil, err
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	// Return cached tools if available
	if len(inst.tools) > 0 {
		inst.lastUsed = time.Now()
		return inst.tools, nil
	}

	// Fetch tools via SDK (handles pagination automatically)
	result, err := inst.session.ListTools(ctx, nil)
	if err != nil {
		m.handleSessionError(serverName, inst, err)
		return nil, fmt.Errorf("tools/list: %w", err)
	}

	inst.tools = result.Tools
	inst.lastUsed = time.Now()

	logger.InfoCF("mcp", fmt.Sprintf("Server %q: loaded %d tools", serverName, len(result.Tools)),
		map[string]interface{}{
			"server": serverName,
			"tools":  len(result.Tools),
		})

	return result.Tools, nil
}

// CallTool executes a tool on an MCP server.
func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, args map[string]interface{}) (string, error) {
	inst, err := m.ensureRunning(ctx, serverName)
	if err != nil {
		return "", err
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	inst.lastUsed = time.Now()

	result, err := inst.session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		m.handleSessionError(serverName, inst, err)
		return "", fmt.Errorf("tools/call %s: %w", toolName, err)
	}

	text := extractText(result)

	if result.IsError {
		return "", fmt.Errorf("tool error: %s", text)
	}

	return text, nil
}

// ReadResource reads a resource from an MCP server by URI.
func (m *Manager) ReadResource(ctx context.Context, serverName, uri string) (string, error) {
	inst, err := m.ensureRunning(ctx, serverName)
	if err != nil {
		return "", err
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	inst.lastUsed = time.Now()

	result, err := inst.session.ReadResource(ctx, &sdkmcp.ReadResourceParams{
		URI: uri,
	})
	if err != nil {
		m.handleSessionError(serverName, inst, err)
		return "", fmt.Errorf("resources/read %s: %w", uri, err)
	}

	var parts []string
	for _, content := range result.Contents {
		if content.Text != "" {
			parts = append(parts, content.Text)
		} else if len(content.Blob) > 0 {
			parts = append(parts, fmt.Sprintf("[blob: %s, %d bytes]", content.MIMEType, len(content.Blob)))
		}
	}

	if len(parts) == 0 {
		return "(no content)", nil
	}

	return strings.Join(parts, "\n"), nil
}

// BuildSummary generates XML for the system prompt using config only (no process start).
func (m *Manager) BuildSummary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var enabled []config.MCPServerConfig
	var names []string
	for name, cfg := range m.configs {
		if cfg.Enabled {
			enabled = append(enabled, cfg)
			names = append(names, name)
		}
	}

	if len(enabled) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<mcp_servers>\n")
	for i, cfg := range enabled {
		sb.WriteString("  <server>\n")
		sb.WriteString(fmt.Sprintf("    <name>%s</name>\n", names[i]))
		if cfg.Description != "" {
			sb.WriteString(fmt.Sprintf("    <description>%s</description>\n", cfg.Description))
		}
		transport := "stdio"
		if cfg.URL != "" {
			transport = "http"
		}
		sb.WriteString(fmt.Sprintf("    <transport>%s</transport>\n", transport))
		sb.WriteString("  </server>\n")
	}
	sb.WriteString("</mcp_servers>")

	return sb.String()
}

// Stop shuts down all running servers and the idle reaper.
func (m *Manager) Stop() {
	close(m.stopCh)

	m.mu.Lock()
	for name, inst := range m.servers {
		inst.mu.Lock()
		if inst.session != nil {
			logger.InfoCF("mcp", fmt.Sprintf("Stopping server %q", name), nil)
			inst.session.Close()
			inst.session = nil
		}
		inst.mu.Unlock()
	}
	m.servers = make(map[string]*ServerInstance)
	m.mu.Unlock()

	m.wg.Wait()
}

// ensureRunning starts a server if not already running.
func (m *Manager) ensureRunning(ctx context.Context, serverName string) (*ServerInstance, error) {
	m.mu.RLock()
	cfg, ok := m.configs[serverName]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown MCP server: %q", serverName)
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("MCP server %q is disabled", serverName)
	}

	m.mu.Lock()
	inst, exists := m.servers[serverName]
	if !exists {
		inst = &ServerInstance{}
		m.servers[serverName] = inst
	}
	m.mu.Unlock()

	inst.mu.Lock()
	defer inst.mu.Unlock()

	// Already running — check if session is still alive
	if inst.session != nil {
		select {
		case <-inst.done:
			logger.WarnCF("mcp", fmt.Sprintf("Server %q session closed, restarting", serverName), nil)
			inst.session = nil
			inst.tools = nil
		default:
			return inst, nil
		}
	}

	// Check crash rate limit (max 3 in 60 seconds)
	now := time.Now()
	var recentCrashes []time.Time
	for _, t := range inst.crashes {
		if now.Sub(t) < 60*time.Second {
			recentCrashes = append(recentCrashes, t)
		}
	}
	inst.crashes = recentCrashes
	if len(recentCrashes) >= 3 {
		return nil, fmt.Errorf("MCP server %q crashed too frequently (3 times in 60s)", serverName)
	}

	// Create SDK client
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "picoclaw", Version: "1.0.0"},
		nil,
	)

	// Create transport based on config
	var transport sdkmcp.Transport
	if cfg.URL != "" {
		// HTTP (Streamable HTTP) transport
		httpClient := &http.Client{}
		if len(cfg.Headers) > 0 {
			httpClient.Transport = &headerTransport{
				headers: cfg.Headers,
				base:    http.DefaultTransport,
			}
		}
		transport = &sdkmcp.StreamableClientTransport{
			Endpoint:             cfg.URL,
			HTTPClient:           httpClient,
			DisableStandaloneSSE: true,
		}
		inst.isHTTP = true

		logger.InfoCF("mcp", fmt.Sprintf("Connecting to HTTP server %q: %s", serverName, cfg.URL),
			map[string]interface{}{
				"server": serverName,
				"url":    cfg.URL,
			})
	} else {
		// Stdio (Command) transport
		var env []string
		if len(cfg.Env) > 0 {
			env = os.Environ()
			for k, v := range cfg.Env {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
		}

		cmd := exec.Command(cfg.Command, cfg.Args...)
		if len(env) > 0 {
			cmd.Env = env
		}
		transport = &sdkmcp.CommandTransport{Command: cmd}

		logger.InfoCF("mcp", fmt.Sprintf("Starting server %q: %s %s", serverName, cfg.Command, strings.Join(cfg.Args, " ")),
			map[string]interface{}{
				"server":  serverName,
				"command": cfg.Command,
			})
	}

	// Connect performs the full MCP handshake (initialize + notifications/initialized)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		inst.crashes = append(inst.crashes, now)
		return nil, fmt.Errorf("connect MCP server %q: %w", serverName, err)
	}

	inst.session = session
	inst.lastUsed = now
	inst.tools = nil // Clear cached tools for fresh fetch

	// Monitor session lifecycle — single goroutine per session
	inst.done = make(chan struct{})
	go func() {
		session.Wait()
		close(inst.done)
	}()

	initResult := session.InitializeResult()
	logger.InfoCF("mcp", fmt.Sprintf("Server %q initialized (protocol: %s, server: %s %s)",
		serverName, initResult.ProtocolVersion, initResult.ServerInfo.Name, initResult.ServerInfo.Version),
		map[string]interface{}{
			"server":   serverName,
			"protocol": initResult.ProtocolVersion,
		})

	return inst, nil
}

// handleSessionError records a crash and cleans up the session on transport errors.
func (m *Manager) handleSessionError(serverName string, inst *ServerInstance, err error) {
	errStr := err.Error()
	isTransportError := strings.Contains(errStr, "write") || strings.Contains(errStr, "read") ||
		strings.Contains(errStr, "pipe") || strings.Contains(errStr, "process") ||
		strings.Contains(errStr, "http") || strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "EOF")

	if isTransportError {
		logger.WarnCF("mcp", fmt.Sprintf("Server %q transport error, marking for restart: %v", serverName, err), nil)
		if inst.session != nil {
			inst.session.Close()
			inst.session = nil
		}
		inst.tools = nil
		inst.crashes = append(inst.crashes, time.Now())
	}
}

// idleReaper periodically checks for idle servers and stops them.
func (m *Manager) idleReaper() {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.reapIdleServers()
		}
	}
}

func (m *Manager) reapIdleServers() {
	m.mu.RLock()
	serverNames := make([]string, 0, len(m.servers))
	for name := range m.servers {
		serverNames = append(serverNames, name)
	}
	m.mu.RUnlock()

	for _, name := range serverNames {
		m.mu.RLock()
		cfg := m.configs[name]
		inst, ok := m.servers[name]
		m.mu.RUnlock()
		if !ok {
			continue
		}

		timeout := cfg.IdleTimeout
		if timeout <= 0 {
			timeout = 300 // default 5 minutes
		}

		inst.mu.Lock()
		if inst.session != nil && time.Since(inst.lastUsed) > time.Duration(timeout)*time.Second {
			if inst.isHTTP {
				logger.InfoCF("mcp", fmt.Sprintf("Closing idle HTTP session for %q (idle %v)", name, time.Since(inst.lastUsed).Round(time.Second)), nil)
			} else {
				logger.InfoCF("mcp", fmt.Sprintf("Stopping idle server %q (idle %v)", name, time.Since(inst.lastUsed).Round(time.Second)), nil)
			}
			inst.session.Close()
			inst.session = nil
			inst.tools = nil
		}
		inst.mu.Unlock()
	}
}

// extractText converts SDK content blocks and structured content into text.
func extractText(result *sdkmcp.CallToolResult) string {
	var parts []string

	for _, content := range result.Content {
		switch c := content.(type) {
		case *sdkmcp.TextContent:
			parts = append(parts, c.Text)
		case *sdkmcp.ImageContent:
			parts = append(parts, fmt.Sprintf("[image: %s, %d bytes]", c.MIMEType, len(c.Data)))
		case *sdkmcp.AudioContent:
			parts = append(parts, fmt.Sprintf("[audio: %s, %d bytes]", c.MIMEType, len(c.Data)))
		case *sdkmcp.ResourceLink:
			parts = append(parts, fmt.Sprintf("[resource_link: %s]", c.URI))
		case *sdkmcp.EmbeddedResource:
			if c.Resource != nil {
				if c.Resource.Text != "" {
					parts = append(parts, c.Resource.Text)
				} else if len(c.Resource.Blob) > 0 {
					parts = append(parts, fmt.Sprintf("[embedded resource: %s, %s, %d bytes]",
						c.Resource.URI, c.Resource.MIMEType, len(c.Resource.Blob)))
				} else {
					parts = append(parts, fmt.Sprintf("[embedded resource: %s]", c.Resource.URI))
				}
			}
		}
	}

	if result.StructuredContent != nil {
		if data, err := json.MarshalIndent(result.StructuredContent, "", "  "); err == nil {
			parts = append(parts, string(data))
		}
	}

	if len(parts) == 0 {
		return "(no content)"
	}

	return strings.Join(parts, "\n")
}
