package mcp

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	clientName    = "picoclaw"
	clientVersion = "1.0.0"

	logModule = "mcp"
	maxFails  = 3
)

var (
	ErrManagerClosed = errors.New("MCP manager is closed")

	ErrInvalidServerConfig = errors.New("either URL or command must be provided")

	ErrStdioCommandRequired = errors.New("command is required for stdio transport")
)

// headerTransport is an http.RoundTripper that adds custom headers to requests
type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for key, value := range t.headers {
		req.Header.Add(key, value)
	}
	base := cmp.Or(t.base, http.DefaultTransport)

	return base.RoundTrip(req)
}

// Manager manages multiple MCP server connections
type Manager struct {
	servers map[string]*ServerConnection
	mu      sync.RWMutex
	closed  atomic.Bool
	wg      sync.WaitGroup
}

// NewManager creates a new MCP manager
func NewManager() *Manager {
	return &Manager{
		servers: make(map[string]*ServerConnection),
	}
}

// LoadFromConfig loads MCP servers from configuration
func (m *Manager) LoadFromConfig(ctx context.Context, cfg *config.Config) error {
	return m.LoadFromMCPConfig(ctx, cfg.Tools.MCP, cfg.WorkspacePath())
}

// LoadFromMCPConfig loads MCP servers from MCP configuration and workspace path.
// This is the minimal dependency version that doesn't require the full Config object.
func (m *Manager) LoadFromMCPConfig(
	ctx context.Context,
	mcpCfg config.MCPConfig,
	workspacePath string,
) error {
	if !mcpCfg.Enabled {
		logger.InfoCF(logModule, "MCP integration is disabled", nil)
		return nil
	}

	if len(mcpCfg.Servers) == 0 {
		logger.InfoCF(logModule, "No MCP servers configured", nil)
		return nil
	}

	logger.InfoCF(logModule, "Initializing MCP servers", map[string]any{
		"count": len(mcpCfg.Servers),
	})

	var wg sync.WaitGroup
	errs := make(chan error, len(mcpCfg.Servers))
	enabledCount := 0

	for name, serverCfg := range mcpCfg.Servers {
		if !serverCfg.Enabled {
			logger.DebugCF(logModule, "Skipping disabled server", map[string]any{
				"server": name,
			})
			continue
		}

		if err := m.validateConfig(mcpCfg); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}

		enabledCount++
		wg.Add(1)
		go func(name string, serverCfg config.MCPServerConfig, workspace string) {
			defer wg.Done()

			// Resolve relative envFile paths relative to workspace
			serverCfg.EnvFile = filepath.Join(workspace, serverCfg.EnvFile)

			if err := m.ConnectServer(ctx, name, serverCfg); err != nil {
				logger.ErrorCF(logModule, "Failed to connect to MCP server", map[string]any{
					"server": name,
					"error":  err.Error(),
				})
				errs <- fmt.Errorf("failed to connect to server %s: %w", name, err)
			}
		}(name, serverCfg, workspacePath)
	}

	wg.Wait()
	close(errs)

	// Collect errors
	var allErrors []error
	for err := range errs {
		allErrors = append(allErrors, err)
	}

	connectedCount := len(m.GetServers())

	if len(allErrors) > 0 {
		err := errors.Join(allErrors...)
		if connectedCount == 0 && enabledCount > 0 {
			logger.ErrorCF(logModule, "All MCP servers failed to connect", map[string]any{
				"failed": len(allErrors),
				"total":  enabledCount,
			})

			// only all mcp servers  was failed, then return
			return fmt.Errorf("all MCP servers failed to connect: %w", err)
		}

		logger.WarnCF(logModule, "Initialized with partial failures", map[string]any{
			"failed":    len(allErrors),
			"connected": connectedCount,
			"total":     enabledCount,
			"error":     err.Error(),
		})
	}

	logger.InfoCF(logModule, "MCP server initialization complete", map[string]any{
		"connected": connectedCount,
		"total":     enabledCount,
	})

	return nil
}

func (m *Manager) validateConfig(mcpCfg config.MCPConfig) error {
	if !mcpCfg.Enabled {
		return nil
	}
	for name, serverCfg := range mcpCfg.Servers {
		if !serverCfg.Enabled {
			continue
		}
		if serverCfg.URL == "" && serverCfg.Command == "" {
			return fmt.Errorf("server %s: missing URL (for SSE/HTTP) or command (for stdio)", name)
		}

		if serverCfg.EnvFile != "" && !filepath.IsAbs(serverCfg.EnvFile) {
			logger.WarnCF(logModule, "Relative env file path", map[string]any{
				"server_name": name,
				"env_file":    serverCfg.EnvFile,
			})
		}
	}
	return nil
}

// ConnectServer connects to a single MCP server
func (m *Manager) ConnectServer(
	ctx context.Context,
	name string,
	cfg config.MCPServerConfig,
) error {
	logger.InfoCF(logModule, "Connecting to MCP server", map[string]any{
		"server":     name,
		"command":    cfg.Command,
		"args_count": len(cfg.Args),
	})

	conn, err := newServerConnection(ctx, name, cfg)
	if err != nil {
		return fmt.Errorf("failed to create server connection: %w", err)
	}

	m.mu.Lock()
	if oldConn, exists := m.servers[name]; exists {
		logger.WarnCF(logModule, "Overwriting existing server connection, closing old session",
			map[string]any{"server": name},
		)
		_ = oldConn.Session.Close()
	}
	m.servers[name] = conn
	m.mu.Unlock()

	// start health monitoring for this server
	m.startMonitor(name)
	return nil
}

// GetServers returns all connected servers
func (m *Manager) GetServers() map[string]*ServerConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*ServerConnection, len(m.servers))
	maps.Copy(result, m.servers)
	return result
}

// GetServer returns a specific server connection
func (m *Manager) GetServer(name string) (*ServerConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, ok := m.servers[name]
	return conn, ok
}

// CallTool calls a tool on a specific server
func (m *Manager) CallTool(
	ctx context.Context,
	serverName, toolName string,
	arguments map[string]any,
) (*mcp.CallToolResult, error) {
	m.mu.RLock()
	if m.closed.Load() {
		m.mu.RUnlock()
		return nil, fmt.Errorf("manager is closed")
	}

	conn, ok := m.servers[serverName]
	if !ok {
		m.mu.RUnlock()
		return nil, fmt.Errorf("server %s not found", serverName)
	}

	m.wg.Add(1)
	m.mu.RUnlock()
	defer m.wg.Done()

	params := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	}

	return conn.Session.CallTool(ctx, params)
}

// Close closes all server connections
func (m *Manager) Close() error {
	m.mu.Lock()
	if m.closed.Swap(true) {
		m.mu.Unlock()
		return nil // already closed
	}

	m.mu.Unlock()

	m.wg.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	logger.InfoCF(logModule, "Closing all MCP server connections", map[string]any{
		"count": len(m.servers),
	})

	var errs []error
	for name, conn := range m.servers {
		if conn.cancelFunc != nil {
			conn.cancelFunc()
		}

		if conn.Session != nil {
			if err := conn.Session.Close(); err != nil {
				logger.ErrorCF(logModule, "Failed to close server connection", map[string]any{
					"server": name,
					"error":  err.Error(),
				})
				errs = append(errs, fmt.Errorf("server %s: %w", name, err))
			}
		}
	}

	m.servers = make(map[string]*ServerConnection)

	if len(errs) > 0 {
		return fmt.Errorf("failed to close %d server(s): %w", len(errs), errors.Join(errs...))
	}

	return nil
}

// GetAllTools returns all tools from all connected servers
func (m *Manager) GetAllTools() map[string][]*mcp.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]*mcp.Tool)
	for name, conn := range m.servers {
		if len(conn.Tools) > 0 {
			result[name] = conn.Tools
		}
	}
	return result
}

func (m *Manager) startMonitor(name string) {
	ctx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	if conn, ok := m.servers[name]; ok {
		if conn.cancelFunc != nil {
			conn.cancelFunc()
		}
		conn.cancelFunc = cancel
	}
	m.mu.Unlock()

	go func() {
		defer cancel()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		count := 0

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !m.checkHealth(name) {
					count++
					if count >= maxFails {
						m.handleServerOffline(name)
						return
					}
				} else {
					count = 0
				}
			}
		}
	}()
}

// checkHealth performs a health check by calling ListTools. If it fails, it returns false.
func (m *Manager) checkHealth(name string) bool {
	conn, ok := m.GetServer(name)
	if !ok || conn.Session == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := conn.Session.ListTools(ctx, nil)
	return err == nil
}

// handleServerOffline marks the server as offline and starts async reconnection attempts
func (m *Manager) handleServerOffline(name string) {
	m.mu.RLock()
	conn, ok := m.servers[name]
	m.mu.RUnlock()
	if !ok {
		return
	}

	conn.status.Store(StatusOffline)
	logger.WarnCF("mcp", "Server is offline, starting async reconnection", map[string]any{"server": name})

	go func() {
		backoff := []time.Duration{
			1 * time.Second,
			2 * time.Second,
			4 * time.Second,
			8 * time.Second,
			16 * time.Second,
		}
		attempt := 0

		for {
			wait := backoff[len(backoff)-1]
			if attempt < len(backoff) {
				wait = backoff[attempt]
			}

			time.Sleep(wait)
			attempt++

			logger.DebugCF("mcp", "Reconnection attempt", map[string]any{"server": name, "attempt": attempt})

			if m.closed.Load() {
				return
			}
			err := m.ConnectServer(context.Background(), name, conn.Config)
			if err == nil {
				logger.InfoCF("mcp", "Reconnection successful", map[string]any{"server": name})
				conn.status.Store(StatusOnline)
				m.startMonitor(name)
				return
			}
		}
	}()
}
