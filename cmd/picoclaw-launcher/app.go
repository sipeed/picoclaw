package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// App is the main Wails application struct.
// All exported methods are automatically bound to the frontend.
type App struct {
	ctx        context.Context
	configPath string
	forceQuit  bool

	// Chat state
	chatMu    sync.Mutex
	agentLoop *agent.AgentLoop
	msgBus    *bus.MessageBus
}

// NewApp creates a new App instance.
func NewApp(configPath string) *App {
	return &App{configPath: configPath}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.setupTray()
}

// shutdown is called when the app is closing.
func (a *App) shutdown(ctx context.Context) {
	a.chatMu.Lock()
	defer a.chatMu.Unlock()
	if a.msgBus != nil {
		a.msgBus.Close()
	}
}

// ── Setup ───────────────────────────────────────────

// SetupStatus returns whether initial setup is needed.
type SetupStatusResult struct {
	NeedsSetup bool   `json:"needs_setup"`
	ConfigPath string `json:"config_path"`
}

func (a *App) GetSetupStatus() SetupStatusResult {
	needsSetup := true
	if cfg, err := config.LoadConfig(a.configPath); err == nil && cfg != nil {
		// Config is valid if either providers or model_list has entries
		needsSetup = cfg.Providers.IsEmpty() && len(cfg.ModelList) == 0
	}
	return SetupStatusResult{
		NeedsSetup: needsSetup,
		ConfigPath: a.configPath,
	}
}

// TestLLM tests an LLM connection without persisting anything.
type TestLLMRequest struct {
	APIKey  string `json:"api_key"`
	APIBase string `json:"api_base"`
	Model   string `json:"model"`
}

type TestLLMResult struct {
	Success  bool   `json:"success"`
	Response string `json:"response"`
	Model    string `json:"model"`
	Protocol string `json:"protocol"`
	Error    string `json:"error"`
}

func (a *App) TestLLM(req TestLLMRequest) TestLLMResult {
	if req.APIKey == "" || req.Model == "" {
		return TestLLMResult{Error: "API key and model are required"}
	}
	if req.APIBase == "" {
		req.APIBase = "https://api.openai.com/v1"
	}

	protocol := detectProtocol(req.APIBase)
	modelID := buildModelField(protocol, req.Model)

	modelCfg := &config.ModelConfig{
		ModelName: req.Model,
		Model:     modelID,
		APIBase:   req.APIBase,
		APIKey:    req.APIKey,
	}

	provider, resolvedModel, err := providers.CreateProviderFromConfig(modelCfg)
	if err != nil {
		return TestLLMResult{Error: fmt.Sprintf("Provider creation failed: %v", err)}
	}
	if resolvedModel == "" {
		resolvedModel = req.Model
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	resp, err := provider.Chat(ctx, []providers.Message{
		{Role: "user", Content: "Reply with exactly one word: PONG"},
	}, nil, resolvedModel, nil)

	if err != nil {
		return TestLLMResult{Error: fmt.Sprintf("LLM call failed: %v", err)}
	}

	return TestLLMResult{
		Success:  true,
		Response: strings.TrimSpace(resp.Content),
		Model:    resolvedModel,
		Protocol: protocol,
	}
}

// SaveSetup saves a minimal config from the setup wizard.
type SaveSetupResult struct {
	Success    bool   `json:"success"`
	ConfigPath string `json:"config_path"`
	Workspace  string `json:"workspace"`
	Error      string `json:"error"`
}

func (a *App) SaveSetup(req TestLLMRequest) SaveSetupResult {
	if req.APIKey == "" || req.Model == "" {
		return SaveSetupResult{Error: "API key and model are required"}
	}
	if req.APIBase == "" {
		req.APIBase = "https://api.openai.com/v1"
	}

	protocol := detectProtocol(req.APIBase)
	modelID := buildModelField(protocol, req.Model)
	defaults := config.DefaultConfig()
	workspace := defaults.Agents.Defaults.Workspace

	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:           workspace,
				RestrictToWorkspace: true,
				ModelName:           req.Model,
				MaxTokens:           32768,
				MaxToolIterations:   50,
			},
		},
		ModelList: []config.ModelConfig{
			{ModelName: req.Model, Model: modelID, APIBase: req.APIBase, APIKey: req.APIKey},
		},
		Gateway: defaults.Gateway,
		Tools: config.ToolsConfig{
			Exec: config.ExecConfig{EnableDenyPatterns: true},
			Web: config.WebToolsConfig{
				DuckDuckGo: config.DuckDuckGoConfig{Enabled: true, MaxResults: 5},
			},
		},
	}

	os.MkdirAll(filepath.Dir(a.configPath), 0755)
	os.MkdirAll(workspace, 0755)

	if err := config.SaveConfig(a.configPath, cfg); err != nil {
		return SaveSetupResult{Error: fmt.Sprintf("Save failed: %v", err)}
	}

	return SaveSetupResult{Success: true, ConfigPath: a.configPath, Workspace: workspace}
}

// ── Config ──────────────────────────────────────────

func (a *App) GetConfig() (map[string]any, error) {
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return map[string]any{"config": raw, "path": a.configPath}, nil
}

func (a *App) SaveConfig(cfgData string) error {
	// Parse into generic map for cleanup
	var raw any
	if err := json.Unmarshal([]byte(cfgData), &raw); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Remove empty/zero/null values
	cleaned := cleanJSON(raw)

	data, err := json.MarshalIndent(cleaned, "", "  ")
	if err != nil {
		return fmt.Errorf("format failed: %w", err)
	}

	dir := filepath.Dir(a.configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir failed: %w", err)
	}
	return os.WriteFile(a.configPath, append(data, '\n'), 0o600)
}

// cleanJSON recursively removes null, empty string, false, zero, empty object/array values.
func cleanJSON(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any)
		for k, child := range val {
			c := cleanJSON(child)
			if !isZeroValue(c) {
				out[k] = c
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	case []any:
		var out []any
		for _, child := range val {
			c := cleanJSON(child)
			if !isZeroValue(c) {
				out = append(out, c)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		return v
	}
}

func isZeroValue(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case map[string]any:
		return len(val) == 0
	case []any:
		return len(val) == 0
	}
	return false
}


// ── Gateway Process ─────────────────────────────────

var gatewayLogs = NewLogBuffer(500)

type GatewayStatus struct {
	Status string   `json:"status"` // "running", "stopped", "error"
	Model  string   `json:"model"`
	Logs   []string `json:"logs"`
	Total  int      `json:"total"`
}

func (a *App) GetGatewayStatus() GatewayStatus {
	cfg, err := config.LoadConfig(a.configPath)
	host := "127.0.0.1"
	port := 18790
	if err == nil && cfg != nil {
		if cfg.Gateway.Host != "" && cfg.Gateway.Host != "0.0.0.0" {
			host = cfg.Gateway.Host
		}
		if cfg.Gateway.Port != 0 {
			port = cfg.Gateway.Port
		}
	}

	url := fmt.Sprintf("http://%s/health", net.JoinHostPort(host, strconv.Itoa(port)))
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)

	status := "stopped"
	model := "-"
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			status = "running"
		}
	}

	if err == nil && cfg != nil {
		model = cfg.Agents.Defaults.GetModelName()
	}

	lines, total, _ := gatewayLogs.LinesSince(0)
	if lines == nil {
		lines = []string{}
	}

	return GatewayStatus{Status: status, Model: model, Logs: lines, Total: total}
}

// findBinary locates the picoclaw binary: same dir as launcher, PATH, or project build dir.
func findBinary(name string) (string, error) {
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	if exe, err := os.Executable(); err == nil {
		// Same directory as launcher
		if p := filepath.Join(filepath.Dir(exe), name+suffix); fileExists(p) {
			return p, nil
		}
		// Project build dir (dev mode: ../../build/)
		if p := filepath.Join(filepath.Dir(exe), "..", "..", "build", name+suffix); fileExists(p) {
			return p, nil
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("%s not found (checked: same dir, PATH, project build)", name)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (a *App) StartGateway() (string, error) {
	execPath, err := findBinary("picoclaw")
	if err != nil {
		return "", err
	}

	cmd := exec.Command(execPath, "gateway")
	hideProcessWindow(cmd)

	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	gatewayLogs.Reset()

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start failed: %w", err)
	}

	go scanPipe(stdoutPipe)
	go scanPipe(stderrPipe)
	go func() { cmd.Wait() }()

	return fmt.Sprintf("Started (PID: %d)", cmd.Process.Pid), nil
}

func scanPipe(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		gatewayLogs.Append(scanner.Text())
	}
}

func (a *App) StopGateway() (string, error) {
	var err error
	if runtime.GOOS == "windows" {
		psCmd := `Get-WmiObject Win32_Process | Where-Object { $_.CommandLine -match 'picoclaw.*gateway' } | ForEach-Object { Stop-Process $_.ProcessId -Force }`
		err = exec.Command("powershell", "-Command", psCmd).Run()
	} else {
		err = exec.Command("pkill", "-f", "picoclaw gateway").Run()
	}
	if err != nil {
		return "Gateway may not be running", nil
	}
	return "Stopped", nil
}

func (a *App) RestartGateway() (string, error) {
	a.StopGateway()
	time.Sleep(500 * time.Millisecond)

	// Reset chat agent loop so it picks up new config
	a.chatMu.Lock()
	a.agentLoop = nil
	a.chatMu.Unlock()

	return a.StartGateway()
}

func (a *App) GetLogs(offset int) map[string]any {
	lines, total, runID := gatewayLogs.LinesSince(offset)
	if lines == nil {
		lines = []string{}
	}
	return map[string]any{"logs": lines, "total": total, "run_id": runID}
}

// ── Chat ────────────────────────────────────────────

type ChatResult struct {
	Success  bool   `json:"success"`
	Response string `json:"response"`
	Error    string `json:"error"`
}

func (a *App) SendChat(message string) ChatResult {
	if message == "" {
		return ChatResult{Error: "message is required"}
	}

	a.chatMu.Lock()
	defer a.chatMu.Unlock()

	// Lazy-init agent loop
	if a.agentLoop == nil {
		if err := a.initAgentLoop(); err != nil {
			return ChatResult{Error: fmt.Sprintf("Init failed: %v", err)}
		}
	}

	ctx, cancel := context.WithTimeout(a.ctx, 120*time.Second)
	defer cancel()

	resp, err := a.agentLoop.ProcessDirect(ctx, message, "launcher:chat")
	if err != nil {
		return ChatResult{Error: fmt.Sprintf("Chat error: %v", err)}
	}

	return ChatResult{Success: true, Response: resp}
}

func (a *App) initAgentLoop() error {
	cfg, err := config.LoadConfig(a.configPath)
	if err != nil {
		return fmt.Errorf("config load failed: %w", err)
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		return fmt.Errorf("provider creation failed: %w", err)
	}
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	a.msgBus = bus.NewMessageBus()
	a.agentLoop = agent.NewAgentLoop(cfg, a.msgBus, provider)
	return nil
}
