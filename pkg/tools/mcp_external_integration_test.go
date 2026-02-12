package tools

import (
	"context"
	"fmt"
	"io"
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
	rootCanonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(root) error: %v", err)
	}
	testFile := filepath.Join(rootCanonical, "hello.txt")
	if err := os.WriteFile(testFile, []byte("hello from filesystem mcp"), 0644); err != nil {
		t.Fatalf("WriteFile(testFile) error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tools := loadExternalMCPTools(t, ctx, config.MCPServerConfig{
		Name:             "filesystem",
		Enabled:          true,
		Transport:        "command",
		Command:          "npx",
		Args:             []string{"-y", "@modelcontextprotocol/server-filesystem", rootCanonical},
		ToolPrefix:       "mcp_fs",
		StartupTimeoutMS: 30000,
		CallTimeoutMS:    30000,
	})

	listAllowedDirs := requireToolByName(t, tools, "mcp_fs_list_allowed_directories")
	readFile := requireToolByName(t, tools, "mcp_fs_read_file")

	out, err := listAllowedDirs.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute(list_allowed_directories) error: %v", err)
	}
	if !strings.Contains(out, rootCanonical) {
		t.Fatalf("expected allowed directory %q in output: %s", rootCanonical, out)
	}

	readOut, err := readFile.Execute(ctx, map[string]interface{}{"path": testFile})
	if err != nil {
		t.Fatalf("Execute(read_file) error: %v", err)
	}
	if !strings.Contains(readOut, "hello from filesystem mcp") {
		t.Fatalf("expected file content in output, got: %s", readOut)
	}
}

func TestMCPExternalPopularMemoryCommand(t *testing.T) {
	requireExternalMCPTests(t)

	memoryFile := filepath.Join(t.TempDir(), "memory.jsonl")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tools := loadExternalMCPTools(t, ctx, config.MCPServerConfig{
		Name:             "memory",
		Enabled:          true,
		Transport:        "command",
		Command:          "npx",
		Args:             []string{"-y", "@modelcontextprotocol/server-memory"},
		Env:              map[string]string{"MEMORY_FILE_PATH": memoryFile},
		ToolPrefix:       "mcp_memory",
		StartupTimeoutMS: 30000,
		CallTimeoutMS:    30000,
	})

	readGraph := requireToolByName(t, tools, "mcp_memory_read_graph")

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
	startEverythingServer(t, port, "sse")
	waitForTCPPort(t, fmt.Sprintf("127.0.0.1:%d", port), 15*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tools := loadExternalMCPTools(t, ctx, config.MCPServerConfig{
		Name:             "everything",
		Enabled:          true,
		Transport:        "sse",
		URL:              fmt.Sprintf("http://127.0.0.1:%d/sse", port),
		ToolPrefix:       "mcp_every",
		StartupTimeoutMS: 30000,
		CallTimeoutMS:    30000,
	})

	echoTool := requireToolByName(t, tools, "mcp_every_echo")

	out, err := echoTool.Execute(ctx, map[string]interface{}{"message": "hello from sse"})
	if err != nil {
		t.Fatalf("Execute(echo) error: %v", err)
	}
	if !strings.Contains(out, "hello from sse") {
		t.Fatalf("unexpected echo output: %s", out)
	}
}

func TestMCPExternalPopularEverythingStreamableHTTP(t *testing.T) {
	requireExternalMCPTests(t)

	port := pickFreePort(t)
	startEverythingServer(t, port, "streamableHttp")
	waitForTCPPort(t, fmt.Sprintf("127.0.0.1:%d", port), 15*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	tools := loadExternalMCPTools(t, ctx, config.MCPServerConfig{
		Name:             "everything-http",
		Enabled:          true,
		Transport:        "streamable_http",
		URL:              fmt.Sprintf("http://127.0.0.1:%d/mcp", port),
		ToolPrefix:       "mcp_http",
		StartupTimeoutMS: 30000,
		CallTimeoutMS:    30000,
	})

	echoTool := requireToolByName(t, tools, "mcp_http_echo")

	out, err := echoTool.Execute(ctx, map[string]interface{}{"message": "hello from streamable-http"})
	if err != nil {
		t.Fatalf("Execute(echo) error: %v", err)
	}
	if !strings.Contains(out, "hello from streamable-http") {
		t.Fatalf("unexpected echo output: %s", out)
	}
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

func loadExternalMCPTools(t *testing.T, ctx context.Context, server config.MCPServerConfig) []Tool {
	t.Helper()

	cfg := config.MCPToolsConfig{
		Enabled: true,
		Servers: []config.MCPServerConfig{server},
	}

	tools, err := LoadMCPTools(ctx, cfg, "")
	if err != nil {
		t.Fatalf("LoadMCPTools() error: %v", err)
	}
	return tools
}

func requireToolByName(t *testing.T, tools []Tool, name string) Tool {
	t.Helper()

	tool := findToolByName(tools, name)
	if tool == nil {
		t.Fatalf("missing tool %s; got %v", name, toolNames(tools))
	}
	return tool
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

func startEverythingServer(t *testing.T, port int, mode string) {
	t.Helper()

	cmd := exec.Command("npx", "-y", "@modelcontextprotocol/server-everything", mode)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		t.Fatalf("start everything %s server: %v", mode, err)
	}

	t.Cleanup(func() {
		if cmd.Process == nil {
			return
		}
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
}
