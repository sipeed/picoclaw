package tools

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// These tests exercise real/popular MCP servers and therefore require network/npm.
// Run explicitly with:
//   PICOCLAW_RUN_EXTERNAL_MCP_TESTS=1 go test ./pkg/tools -run MCPExternal -v

func TestMCPExternalPopularFilesystemCommand(t *testing.T) {
	requireExternalMCPTests(t)

	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cfg := config.MCPToolsConfig{
		Enabled: true,
		Servers: []config.MCPServerConfig{
			{
				Name:             "filesystem",
				Enabled:          true,
				Transport:        "command",
				Command:          "npx",
				Args:             []string{"-y", "@modelcontextprotocol/server-filesystem", root},
				ToolPrefix:       "mcp_fs",
				StartupTimeoutMS: 30000,
				CallTimeoutMS:    30000,
			},
		},
	}

	tools, err := LoadMCPTools(ctx, cfg, "")
	if err != nil {
		t.Fatalf("LoadMCPTools() error: %v", err)
	}

	tool := findToolByName(tools, "mcp_fs_list_allowed_directories")
	if tool == nil {
		t.Fatalf("missing tool mcp_fs_list_allowed_directories; got %v", toolNames(tools))
	}

	out, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute(list_allowed_directories) error: %v", err)
	}
	if !strings.Contains(out, root) {
		t.Fatalf("expected allowed directory %q in output: %s", root, out)
	}
}

func TestMCPExternalPopularMemoryCommand(t *testing.T) {
	requireExternalMCPTests(t)

	memoryFile := filepath.Join(t.TempDir(), "memory.jsonl")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cfg := config.MCPToolsConfig{
		Enabled: true,
		Servers: []config.MCPServerConfig{
			{
				Name:             "memory",
				Enabled:          true,
				Transport:        "command",
				Command:          "npx",
				Args:             []string{"-y", "@modelcontextprotocol/server-memory"},
				Env:              map[string]string{"MEMORY_FILE_PATH": memoryFile},
				ToolPrefix:       "mcp_memory",
				StartupTimeoutMS: 30000,
				CallTimeoutMS:    30000,
			},
		},
	}

	tools, err := LoadMCPTools(ctx, cfg, "")
	if err != nil {
		t.Fatalf("LoadMCPTools() error: %v", err)
	}

	readGraph := findToolByName(tools, "mcp_memory_read_graph")
	if readGraph == nil {
		t.Fatalf("missing tool mcp_memory_read_graph; got %v", toolNames(tools))
	}

	out, err := readGraph.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute(read_graph) error: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "entities") {
		t.Fatalf("expected graph output to include entities: %s", out)
	}
}

func TestMCPExternalPopularEverythingSSE(t *testing.T) {
	requireExternalMCPTests(t)

	port := pickFreePort(t)
	cmd := startEverythingSSEServer(t, port)
	waitForTCPPort(t, fmt.Sprintf("127.0.0.1:%d", port), 15*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cfg := config.MCPToolsConfig{
		Enabled: true,
		Servers: []config.MCPServerConfig{
			{
				Name:             "everything",
				Enabled:          true,
				Transport:        "sse",
				URL:              fmt.Sprintf("http://127.0.0.1:%d/sse", port),
				ToolPrefix:       "mcp_every",
				StartupTimeoutMS: 30000,
				CallTimeoutMS:    30000,
			},
		},
	}

	tools, err := LoadMCPTools(ctx, cfg, "")
	if err != nil {
		t.Fatalf("LoadMCPTools() error: %v", err)
	}

	echoTool := findToolByName(tools, "mcp_every_echo")
	if echoTool == nil {
		t.Fatalf("missing tool mcp_every_echo; got %v", toolNames(tools))
	}

	out, err := echoTool.Execute(ctx, map[string]interface{}{"message": "hello from sse"})
	if err != nil {
		t.Fatalf("Execute(echo) error: %v", err)
	}
	if !strings.Contains(out, "Echo: hello from sse") {
		t.Fatalf("unexpected echo output: %s", out)
	}

	_ = cmd
}

func requireExternalMCPTests(t *testing.T) {
	t.Helper()
	if os.Getenv("PICOCLAW_RUN_EXTERNAL_MCP_TESTS") != "1" {
		t.Skip("set PICOCLAW_RUN_EXTERNAL_MCP_TESTS=1 to run external MCP integration tests")
	}
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not found")
	}
}

func findToolByName(tools []Tool, name string) Tool {
	for _, tool := range tools {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}

func pickFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pick free port: %v", err)
	}
	defer ln.Close()

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected listener addr type: %T", ln.Addr())
	}
	return addr.Port
}

func waitForTCPPort(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(120 * time.Millisecond)
	}
	t.Fatalf("port %s did not become ready within %v", addr, timeout)
}

func startEverythingSSEServer(t *testing.T, port int) *exec.Cmd {
	t.Helper()

	cmd := exec.Command("npx", "-y", "@modelcontextprotocol/server-everything", "sse")
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start everything sse server: %v", err)
	}

	t.Cleanup(func() {
		if cmd.Process == nil {
			return
		}
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	return cmd
}
