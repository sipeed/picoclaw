package mcp

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	TransportTypeSSE   = "sse"
	TransportTypeHTTP  = "http"
	TransportTypeStdio = "stdio"
)

type ServerStatus int

const (
	StatusOnline ServerStatus = iota
	StatusOffline
	StatusConnecting
)

// ServerConnection represents a connection to an MCP server
type ServerConnection struct {
	Name    string
	Config  config.MCPServerConfig // save config for potential reconnection
	Client  *mcp.Client
	Session *mcp.ClientSession
	Tools   []*mcp.Tool

	status     atomic.Value       // ServerStatus
	mu         sync.Mutex         // protect session switching
	cancelFunc context.CancelFunc // for canceling ongoing operations during reconnection
}

func newServerConnection(
	ctx context.Context,
	name string,
	cfg config.MCPServerConfig,
) (*ServerConnection, error) {
	conn := &ServerConnection{
		Name:   name,
		Config: cfg,
		Client: mcp.NewClient(&mcp.Implementation{
			Name:    clientName,
			Version: clientVersion,
		}, nil),
		mu: sync.Mutex{},
	}

	transport, err := conn.createTransport(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	session, err := conn.Client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	initResult := session.InitializeResult()
	logger.InfoCF(logModule, "Connected to MCP server", map[string]any{
		"server":        name,
		"serverName":    initResult.ServerInfo.Name,
		"serverVersion": initResult.ServerInfo.Version,
		"protocol":      initResult.ProtocolVersion,
	})

	var tools []*mcp.Tool
	if initResult.Capabilities.Tools != nil {
		for tool, err := range session.Tools(ctx, nil) {
			if err != nil {
				logger.WarnCF(logModule, "Error listing tool", map[string]any{
					"server": name,
					"error":  err.Error(),
				})
				continue
			}
			tools = append(tools, tool)
		}
		logger.InfoCF(logModule, "Listed tools from MCP server", map[string]any{
			"server":    name,
			"toolCount": len(tools),
		})
	}

	conn.Tools = tools
	conn.Session = session
	return conn, nil
}

func (conn *ServerConnection) createTransport(cfg config.MCPServerConfig) (mcp.Transport, error) {
	transportType := conn.detectTransportType(cfg)

	switch transportType {
	case TransportTypeSSE, TransportTypeHTTP:
		return conn.newSSETransport(context.Background(), "temp", cfg)
	case TransportTypeStdio:
		return conn.newStdioTransport(context.Background(), "temp", cfg)
	default:
		return nil, fmt.Errorf("unsupported transport type: %s (supported: stdio, sse, http)", transportType)
	}
}

func (conn *ServerConnection) detectTransportType(cfg config.MCPServerConfig) string {
	if cfg.Type != "" {
		return cfg.Type
	}
	if cfg.URL != "" {
		return TransportTypeSSE
	}
	if cfg.Command != "" {
		return TransportTypeStdio
	}
	return ""
}

// Build StdioTransport
func (conn *ServerConnection) newStdioTransport(
	ctx context.Context,
	name string,
	cfg config.MCPServerConfig,
) (mcp.Transport, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		if cfg.Command == "" {
			return nil, ErrStdioCommandRequired
		}

		logger.DebugCF(logModule, "Using stdio transport", map[string]any{
			"server":  name,
			"command": cfg.Command,
			"args":    cfg.Args,
		})

		cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}

		cmdEnv := cmd.Environ()
		envMap := make(map[string]string, len(cmdEnv)/2)

		for _, e := range cmdEnv {
			if idx := strings.SplitN(e, "=", 2); len(idx) == 2 {
				envMap[idx[0]] = idx[1]
			}
		}

		if cfg.EnvFile != "" {
			envVars, err := godotenv.Read(cfg.EnvFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load env file %s: %w", cfg.EnvFile, err)
			}

			maps.Copy(envMap, envVars)

			logger.DebugCF(logModule, "Loaded environment variables from file", map[string]any{
				"server":  name,
				"envFile": cfg.EnvFile,
				"args":    envVars,
			})
		}

		maps.Copy(envMap, cfg.Env)

		env := make([]string, 0, len(envMap))
		for k, v := range envMap {
			env = append(env, k+"="+v)
		}

		cmd.Env = make([]string, len(env))
		copy(cmd.Env, env)

		transport := &mcp.CommandTransport{Command: cmd}
		return transport, nil
	}
}

// Build SSETransport
func (conn *ServerConnection) newSSETransport(
	ctx context.Context,
	name string,
	cfg config.MCPServerConfig,
) (mcp.Transport, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:

		if cfg.URL == "" {
			return nil, ErrInvalidServerConfig
		}

		logger.DebugCF(logModule, "Using SSE/HTTP transport", map[string]any{
			"server": name,
			"url":    cfg.URL,
		})

		sseTransport := &mcp.StreamableClientTransport{
			Endpoint: cfg.URL,
		}

		if len(cfg.Headers) > 0 {
			sseTransport.HTTPClient = &http.Client{
				Transport: &headerTransport{
					base:    http.DefaultTransport,
					headers: cfg.Headers,
				},
			}
			logger.DebugCF(logModule, "Added custom HTTP headers", map[string]any{
				"server":       name,
				"header_count": len(cfg.Headers),
			})
		}
		return sseTransport, nil
	}
}

func (conn *ServerConnection) GetStatus() ServerStatus {
	return conn.status.Load().(ServerStatus)
}
