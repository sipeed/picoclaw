package test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/heartbeat"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type mockProvider struct{}

func (m *mockProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	return &providers.LLMResponse{
		Content:      "pong",
		FinishReason: "stop",
	}, nil
}

func (m *mockProvider) GetDefaultModel() string {
	return "local"
}

func TestLocalStartupConfig(t *testing.T) {
	configPath := testConfigPath(t)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if _, _, err = providers.CreateProvider(cfg); err != nil {
		t.Fatalf("create provider: %v", err)
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, &mockProvider{})

	resp, err := agentLoop.ProcessDirect(context.Background(), "ping", "cli:test")
	if err != nil {
		t.Fatalf("process direct: %v", err)
	}
	if resp != "pong" {
		t.Fatalf("response = %q, want %q", resp, "pong")
	}

	info := agentLoop.GetStartupInfo()
	toolsInfo, ok := info["tools"].(map[string]any)
	if !ok {
		t.Fatalf("tools info missing")
	}
	names, ok := toolsInfo["names"].([]string)
	if !ok {
		t.Fatalf("tools names missing")
	}
	required := []string{"read_file", "write_file", "list_dir", "exec"}
	for _, name := range required {
		if !contains(names, name) {
			t.Fatalf("missing tool: %s", name)
		}
	}
	if contains(names, "web_search") {
		t.Fatalf("unexpected tool: web_search")
	}
}

func TestLocalScenarioCoverage(t *testing.T) {
	workspace := t.TempDir()
	copyStartupFixtures(t, workspace)

	execConfig := &config.Config{}
	execConfig.Tools.Exec.EnableDenyPatterns = true

	registry := tools.NewToolRegistry()
	registry.Register(tools.NewReadFileTool(workspace, true))
	registry.Register(tools.NewWriteFileTool(workspace, true))
	registry.Register(tools.NewListDirTool(workspace, true))
	registry.Register(tools.NewExecToolWithConfig(workspace, true, execConfig))
	registry.Register(tools.NewEditFileTool(workspace, true))
	registry.Register(tools.NewAppendFileTool(workspace, true))

	cb := agent.NewContextBuilder(workspace)
	prompt := cb.BuildSystemPrompt()

	required := []string{
		"## AGENTS.md",
		"AGENT: friendly",
		"## IDENTITY.md",
		"IDENTITY: picoclaw",
		"## SOUL.md",
		"SOUL: calm",
		"## USER.md",
		"USER: prefers Chinese",
		"## Long-term Memory",
		"long-term: prefers concise answers",
		"## Recent Daily Notes",
		"daily note: met user",
	}
	for _, item := range required {
		if !strings.Contains(prompt, item) {
			t.Fatalf("system prompt missing %q", item)
		}
	}

	writeResult := registry.Execute(context.Background(), "write_file", map[string]any{
		"path":    "notes/demo.txt",
		"content": "hello",
	})
	if writeResult.IsError {
		t.Fatalf("write_file error: %s", writeResult.ForLLM)
	}

	readResult := registry.Execute(context.Background(), "read_file", map[string]any{
		"path": "notes/demo.txt",
	})
	if readResult.IsError || !strings.Contains(readResult.ForLLM, "hello") {
		t.Fatalf("read_file result = %q", readResult.ForLLM)
	}

	listResult := registry.Execute(context.Background(), "list_dir", map[string]any{
		"path": "notes",
	})
	if listResult.IsError || !strings.Contains(listResult.ForLLM, "demo.txt") {
		t.Fatalf("list_dir result = %q", listResult.ForLLM)
	}

	execResult := registry.Execute(context.Background(), "exec", map[string]any{
		"command":     "cat demo.txt",
		"working_dir": filepath.Join(workspace, "notes"),
	})
	if execResult.IsError || !strings.Contains(execResult.ForUser, "hello") {
		t.Fatalf("exec result = %q", execResult.ForUser)
	}
}

func TestLocalStartupLoadFlow(t *testing.T) {
	workspace := t.TempDir()
	copyStartupFixtures(t, workspace)

	configPath := filepath.Join(workspace, "config.json")
	payload := map[string]any{
		"agents": map[string]any{
			"defaults": map[string]any{
				"workspace":           workspace,
				"model":               "local",
				"max_tokens":          4096,
				"max_tool_iterations": 10,
			},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	writeErr := os.WriteFile(configPath, data, 0o644)
	if writeErr != nil {
		t.Fatalf("write config: %v", writeErr)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, &mockProvider{})

	resp, err := agentLoop.ProcessDirect(context.Background(), "ping", "cli:startup")
	if err != nil {
		t.Fatalf("process direct: %v", err)
	}
	if resp != "pong" {
		t.Fatalf("response = %q, want %q", resp, "pong")
	}
}

func TestLocalHeartbeatSpawnScenario(t *testing.T) {
	workspace := t.TempDir()
	heartbeatPath := filepath.Join(workspace, "HEARTBEAT.md")
	if err := os.WriteFile(heartbeatPath, []byte("- Report status"), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	stateMgr := state.NewManager(workspace)
	if err := stateMgr.SetLastChannel("telegram:123"); err != nil {
		t.Fatalf("set last channel: %v", err)
	}

	msgBus := bus.NewMessageBus()
	subagentManager := tools.NewSubagentManager(&mockProvider{}, "local", workspace, msgBus)
	spawnTool := tools.NewSpawnTool(subagentManager)

	promptCh := make(chan string, 1)
	channelCh := make(chan string, 1)
	chatIDCh := make(chan string, 1)

	hs := heartbeat.NewHeartbeatService(workspace, 5, true)
	hs.SetBus(msgBus)
	hs.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
		promptCh <- prompt
		channelCh <- channel
		chatIDCh <- chatID
		spawnTool.SetContext(channel, chatID)
		return spawnTool.Execute(context.Background(), map[string]any{
			"task":  "Return pong",
			"label": "heartbeat",
		})
	})

	if err := hs.Start(); err != nil {
		t.Fatalf("start heartbeat: %v", err)
	}
	defer hs.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	msg, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatalf("no inbound subagent message")
	}
	if msg.Channel != "system" {
		t.Fatalf("unexpected inbound channel: %s", msg.Channel)
	}
	if msg.ChatID != "telegram:123" {
		t.Fatalf("unexpected inbound chatID: %s", msg.ChatID)
	}
	if !strings.Contains(msg.Content, "Result:\n") {
		t.Fatalf("unexpected inbound content: %s", msg.Content)
	}

	select {
	case prompt := <-promptCh:
		if !strings.Contains(prompt, "Heartbeat Check") {
			t.Fatalf("heartbeat prompt missing header: %s", prompt)
		}
		if !strings.Contains(prompt, "Report status") {
			t.Fatalf("heartbeat prompt missing task: %s", prompt)
		}
	case <-ctx.Done():
		t.Fatalf("heartbeat handler not called")
	}

	select {
	case channel := <-channelCh:
		if channel != "telegram" {
			t.Fatalf("unexpected handler channel: %s", channel)
		}
	case <-ctx.Done():
		t.Fatalf("handler channel not set")
	}

	select {
	case chatID := <-chatIDCh:
		if chatID != "123" {
			t.Fatalf("unexpected handler chatID: %s", chatID)
		}
	case <-ctx.Done():
		t.Fatalf("handler chatID not set")
	}
}

func TestLocalE2EGLM47Flash(t *testing.T) {
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		t.Fatalf("ZAI_API_KEY is required")
	}

	configPath := testConfigPath(t)
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	updated := false
	for i := range cfg.ModelList {
		if cfg.ModelList[i].ModelName == cfg.Agents.Defaults.Model {
			cfg.ModelList[i].APIKey = apiKey
			updated = true
			break
		}
	}
	if !updated {
		t.Fatalf("model config not found for %q", cfg.Agents.Defaults.Model)
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if modelID != "" {
		cfg.Agents.Defaults.Model = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := agentLoop.ProcessDirect(ctx, "Please reply OK", "cli:e2e")
	if err != nil {
		t.Fatalf("process direct: %v", err)
	}
	if resp == "" {
		t.Fatalf("response is empty")
	}
}

func testConfigPath(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := wd
	if filepath.Base(wd) == "test" {
		root = filepath.Dir(wd)
	}
	return filepath.Join(root, "test", "config.json")
}

func testRepoRoot(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if filepath.Base(wd) == "test" {
		return filepath.Dir(wd)
	}
	return wd
}

func copyStartupFixtures(t *testing.T, workspace string) {
	root := testRepoRoot(t)
	fixturesRoot := filepath.Join(root, "test", "testdata", "startup")
	fixtures := map[string]string{
		"AGENTS.md":        "AGENTS.md",
		"IDENTITY.md":      "IDENTITY.md",
		"SOUL.md":          "SOUL.md",
		"USER.md":          "USER.md",
		"memory/MEMORY.md": filepath.Join("memory", "MEMORY.md"),
	}

	for dstRel, srcRel := range fixtures {
		src := filepath.Join(fixturesRoot, srcRel)
		dst := filepath.Join(workspace, dstRel)
		copyFixtureFile(t, src, dst)
	}

	today := time.Now().Format("20060102")
	monthDir := today[:6]
	src := filepath.Join(fixturesRoot, "memory", "daily.md")
	dst := filepath.Join(workspace, "memory", monthDir, today+".md")
	copyFixtureFile(t, src, dst)
}

func copyFixtureFile(t *testing.T, src, dst string) {
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture %s: %v", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dst, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
