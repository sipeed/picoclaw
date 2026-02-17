package agent

import (
	"context"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/mcp"
)

const (
	mcpBootstrapMinTimeout   = 10 * time.Second
	mcpBootstrapMaxTimeout   = 5 * time.Minute
	mcpBootstrapGraceTimeout = 5 * time.Second
)

type mcpBootstrapResult struct {
	Manager *mcp.Manager
	Tools   []mcp.RegisteredTool
}

func bootstrapMCP(cfg config.MCPToolsConfig) (*mcpBootstrapResult, error) {
	serverConfigs := buildMCPServerConfigs(cfg)
	if len(serverConfigs) == 0 {
		return nil, nil
	}

	manager := mcp.NewManager(serverConfigs)

	discoveryTimeout := calculateMCPDiscoveryTimeout(serverConfigs)
	discoveryCtx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
	defer cancel()

	discoveredTools, err := manager.DiscoverTools(discoveryCtx)
	if err != nil {
		_ = manager.Close()
		return nil, err
	}

	return &mcpBootstrapResult{
		Manager: manager,
		Tools:   discoveredTools,
	}, nil
}

func calculateMCPDiscoveryTimeout(serverConfigs map[string]mcp.ServerConfig) time.Duration {
	maxInitTimeout := mcpBootstrapMinTimeout

	for _, serverConfig := range serverConfigs {
		initTimeout := serverConfig.InitTimeout()
		if initTimeout > maxInitTimeout {
			maxInitTimeout = initTimeout
		}
	}

	timeout := maxInitTimeout + mcpBootstrapGraceTimeout
	if timeout < mcpBootstrapMinTimeout {
		return mcpBootstrapMinTimeout
	}
	if timeout > mcpBootstrapMaxTimeout {
		return mcpBootstrapMaxTimeout
	}
	return timeout
}

func buildMCPServerConfigs(cfg config.MCPToolsConfig) map[string]mcp.ServerConfig {
	servers := make(map[string]mcp.ServerConfig, len(cfg.Servers))

	for serverName, serverCfg := range cfg.Servers {
		if !serverCfg.Enabled {
			continue
		}

		envCopy := make(map[string]string, len(serverCfg.Env))
		for key, value := range serverCfg.Env {
			envCopy[key] = value
		}

		servers[serverName] = mcp.ServerConfig{
			Name:               serverName,
			Command:            serverCfg.Command,
			Args:               append([]string{}, serverCfg.Args...),
			Env:                envCopy,
			WorkingDir:         serverCfg.WorkingDir,
			Protocol:           inferMCPProtocol(serverCfg.Protocol, serverCfg.Command),
			InitTimeoutSeconds: serverCfg.InitTimeoutSeconds,
			CallTimeoutSeconds: serverCfg.CallTimeoutSeconds,
			MaxResponseBytes:   serverCfg.MaxResponseBytes,
			IncludeTools:       append([]string{}, serverCfg.IncludeTools...),
			ExcludeTools:       append([]string{}, serverCfg.ExcludeTools...),
		}
	}

	return servers
}

func inferMCPProtocol(configuredProtocol, command string) string {
	if protocol := strings.TrimSpace(configuredProtocol); protocol != "" {
		return protocol
	}

	// Context7 currently emits JSON-RPC messages as JSONL on stdio,
	// so defaulting avoids long startup waits when protocol is omitted.
	if strings.Contains(strings.ToLower(command), "context7-mcp") {
		return mcp.ProtocolJSONLines
	}

	return ""
}
