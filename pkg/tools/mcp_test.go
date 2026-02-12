package tools

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sipeed/picoclaw/pkg/config"
)

const mcpHelperEnv = "PICOCLAW_MCP_TEST_HELPER"

func TestMain(m *testing.M) {
	if os.Getenv(mcpHelperEnv) == "1" {
		runMCPHelperServer()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runMCPHelperServer() {
	type GreetInput struct {
		Name string `json:"name" jsonschema:"name to greet"`
	}
	type GreetOutput struct {
		Greeting string `json:"greeting"`
	}
	type SumInput struct {
		A int `json:"a" jsonschema:"first number"`
		B int `json:"b" jsonschema:"second number"`
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "picoclaw-test-server", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "return a greeting"}, func(_ context.Context, _ *mcp.CallToolRequest, in GreetInput) (*mcp.CallToolResult, GreetOutput, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Hello " + in.Name},
			},
		}, GreetOutput{Greeting: "Hello " + in.Name}, nil
	})
	mcp.AddTool(server, &mcp.Tool{Name: "sum", Description: "sum two integers"}, func(_ context.Context, _ *mcp.CallToolRequest, in SumInput) (*mcp.CallToolResult, map[string]int, error) {
		return nil, map[string]int{"sum": in.A + in.B}, nil
	})

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		os.Exit(1)
	}
}

func TestLoadMCPTools_CommandTransport(t *testing.T) {
	cfg := config.MCPToolsConfig{
		Enabled: true,
		Servers: []config.MCPServerConfig{
			{
				Name:             "helper",
				Enabled:          true,
				Transport:        "command",
				Command:          os.Args[0],
				Args:             []string{},
				Env:              map[string]string{mcpHelperEnv: "1"},
				StartupTimeoutMS: 8000,
				CallTimeoutMS:    5000,
				ToolPrefix:       "mcp_helper",
			},
		},
	}

	tools, err := LoadMCPTools(context.Background(), cfg, t.TempDir())
	if err != nil {
		t.Fatalf("LoadMCPTools() error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("LoadMCPTools() got %d tools, want 2", len(tools))
	}

	var greetTool Tool
	var sumTool Tool
	for _, tool := range tools {
		switch tool.Name() {
		case "mcp_helper_greet":
			greetTool = tool
		case "mcp_helper_sum":
			sumTool = tool
		}
	}

	if greetTool == nil {
		t.Fatalf("missing discovered tool mcp_helper_greet; got names=%v", toolNames(tools))
	}
	if sumTool == nil {
		t.Fatalf("missing discovered tool mcp_helper_sum; got names=%v", toolNames(tools))
	}

	gotGreeting, err := greetTool.Execute(context.Background(), map[string]interface{}{"name": "Ada"})
	if err != nil {
		t.Fatalf("greetTool.Execute() error: %v", err)
	}
	if !strings.Contains(gotGreeting, "Hello Ada") {
		t.Fatalf("greetTool.Execute() missing greeting: %s", gotGreeting)
	}

	gotSum, err := sumTool.Execute(context.Background(), map[string]interface{}{"a": 2, "b": 3})
	if err != nil {
		t.Fatalf("sumTool.Execute() error: %v", err)
	}
	if !strings.Contains(gotSum, `"sum": 5`) {
		t.Fatalf("sumTool.Execute() output missing sum result: %s", gotSum)
	}
}

func TestBuildLocalToolName_EnsuresUniqueness(t *testing.T) {
	used := map[string]int{}
	cfg := config.MCPServerConfig{Name: "my server", ToolPrefix: "mcp_my_server"}

	name1 := buildLocalToolName(cfg, "echo", used)
	name2 := buildLocalToolName(cfg, "echo", used)

	if name1 == name2 {
		t.Fatalf("expected unique names, got both %q", name1)
	}
	if len(name1) > maxToolNameLength || len(name2) > maxToolNameLength {
		t.Fatalf("tool name length exceeded %d: %q / %q", maxToolNameLength, name1, name2)
	}
}

func TestNormalizeMCPInputSchema_DefaultObject(t *testing.T) {
	schema := normalizeMCPInputSchema(nil)
	if schema["type"] != "object" {
		t.Fatalf("schema.type = %v, want object", schema["type"])
	}
	if _, ok := schema["properties"]; !ok {
		t.Fatalf("schema.properties missing")
	}
}

func TestResolvePath_RelativeUsesWorkspace(t *testing.T) {
	got := resolvePath("servers/time", "/tmp/workspace")
	if got != "/tmp/workspace/servers/time" {
		t.Fatalf("resolvePath() = %q, want %q", got, "/tmp/workspace/servers/time")
	}
}

func TestLoadMCPTools_InvalidServerAggregatesError(t *testing.T) {
	cfg := config.MCPToolsConfig{
		Enabled: true,
		Servers: []config.MCPServerConfig{
			{
				Name:      "broken",
				Enabled:   true,
				Transport: "command",
				Command:   "",
			},
		},
	}

	tools, err := LoadMCPTools(context.Background(), cfg, t.TempDir())
	if len(tools) != 0 {
		t.Fatalf("expected no tools, got %d", len(tools))
	}
	if err == nil {
		t.Fatalf("expected discovery error, got nil")
	}
	if !strings.Contains(err.Error(), "discovery failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildTransport_CommandTerminateDefaults(t *testing.T) {
	assertCommandTransportTerminateDuration(t, config.MCPServerConfig{
		Name:      "default-terminate",
		Enabled:   true,
		Transport: "command",
		Command:   "test-command",
	}, defaultMCPTerminateWait)
}

func TestBuildTransport_CommandTerminateOverride(t *testing.T) {
	assertCommandTransportTerminateDuration(t, config.MCPServerConfig{
		Name:               "override-terminate",
		Enabled:            true,
		Transport:          "command",
		Command:            "test-command",
		TerminateTimeoutMS: 2500,
	}, 2500*time.Millisecond)
}

func assertCommandTransportTerminateDuration(t *testing.T, cfg config.MCPServerConfig, want time.Duration) {
	t.Helper()

	client := newMCPClient(cfg, "")
	tr, err := client.buildTransport()
	if err != nil {
		t.Fatalf("buildTransport() error: %v", err)
	}

	cmdTr, ok := tr.(*mcp.CommandTransport)
	if !ok {
		t.Fatalf("buildTransport() returned %T, want *mcp.CommandTransport", tr)
	}
	if cmdTr.TerminateDuration != want {
		t.Fatalf("TerminateDuration = %v, want %v", cmdTr.TerminateDuration, want)
	}
}

func toolNames(tools []Tool) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		out = append(out, tool.Name())
	}
	return out
}
