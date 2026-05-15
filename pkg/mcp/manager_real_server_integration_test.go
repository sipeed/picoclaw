//go:build integration

package mcp

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sipeed/picoclaw/pkg/config"
)

// TestIntegration_RealConfiguredServer is an opt-in smoke test for a real MCP
// server configured via environment variables.
//
// Run with:
//
//	go test -tags=integration ./pkg/mcp -run TestIntegration_RealConfiguredServer -v
//
// Minimum configuration:
//
//	PICOCLAW_MCP_REAL_SERVER_JSON='{"enabled":true,"type":"http","url":"http://127.0.0.1:8080/mcp"}'
//
// Optional tool invocation:
//
//	PICOCLAW_MCP_REAL_TOOL_NAME=echo
//	PICOCLAW_MCP_REAL_TOOL_ARGS_JSON='{"message":"hello"}'
//	PICOCLAW_MCP_REAL_EXPECT_SUBSTRING=hello
//
// Stdio subprocess example:
//
//	PICOCLAW_MCP_REAL_SERVER_JSON='{"enabled":true,"type":"stdio","command":"npx","args":["-y","@modelcontextprotocol/server-filesystem","."]}'
func TestIntegration_RealConfiguredServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	serverJSON := strings.TrimSpace(os.Getenv("PICOCLAW_MCP_REAL_SERVER_JSON"))
	if serverJSON == "" {
		t.Skip("skipping integration test (set PICOCLAW_MCP_REAL_SERVER_JSON to enable)")
	}

	serverCfg, err := loadRealServerConfig(serverJSON)
	if err != nil {
		t.Fatalf("loadRealServerConfig() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mgr := NewManager()
	if err := mgr.ConnectServer(ctx, "real", serverCfg); err != nil {
		t.Fatalf("ConnectServer() error = %v", err)
	}
	defer func() {
		if err := mgr.Close(); err != nil {
			t.Errorf("Manager.Close() error = %v", err)
		}
	}()

	tools := mgr.GetAllTools()["real"]
	if len(tools) == 0 {
		t.Fatal("expected at least one discovered tool from real MCP server")
	}

	t.Logf("connected to real MCP server via %s with %d tool(s)", config.EffectiveMCPTransportType(serverCfg), len(tools))
	for _, tool := range tools {
		if tool != nil {
			t.Logf("discovered tool: %s", tool.Name)
		}
	}

	if expectedCountRaw := strings.TrimSpace(os.Getenv("PICOCLAW_MCP_REAL_EXPECT_TOOL_COUNT")); expectedCountRaw != "" {
		expectedCount, err := strconv.Atoi(expectedCountRaw)
		if err != nil {
			t.Fatalf("invalid PICOCLAW_MCP_REAL_EXPECT_TOOL_COUNT %q: %v", expectedCountRaw, err)
		}
		if len(tools) != expectedCount {
			t.Fatalf("tool count = %d, want %d", len(tools), expectedCount)
		}
	}

	toolName := strings.TrimSpace(os.Getenv("PICOCLAW_MCP_REAL_TOOL_NAME"))
	if toolName == "" {
		return
	}

	toolArgs, err := loadRealToolArgs(os.Getenv("PICOCLAW_MCP_REAL_TOOL_ARGS_JSON"))
	if err != nil {
		t.Fatalf("loadRealToolArgs() error = %v", err)
	}

	result, err := mgr.CallTool(ctx, "real", toolName, toolArgs)
	if err != nil {
		t.Fatalf("CallTool(%q) error = %v", toolName, err)
	}

	textPayload := joinTextContents(result)
	t.Logf("tool %q returned text payload: %q", toolName, textPayload)

	if want := os.Getenv("PICOCLAW_MCP_REAL_EXPECT_SUBSTRING"); want != "" && !strings.Contains(textPayload, want) {
		t.Fatalf("tool result %q does not contain expected substring %q", textPayload, want)
	}
}

func loadRealServerConfig(raw string) (config.MCPServerConfig, error) {
	var cfg config.MCPServerConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return config.MCPServerConfig{}, err
	}
	if !cfg.Enabled {
		cfg.Enabled = true
	}
	return cfg, nil
}

func loadRealToolArgs(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	return args, nil
}

func joinTextContents(result *sdkmcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}

	parts := make([]string, 0, len(result.Content))
	for _, content := range result.Content {
		if text, ok := content.(*sdkmcp.TextContent); ok && text != nil {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}
