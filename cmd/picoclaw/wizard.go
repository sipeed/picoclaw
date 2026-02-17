package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chzyer/readline"
	"github.com/sipeed/picoclaw/pkg/config"
	"golang.org/x/sys/unix"
)

// ─── ANSI color helpers ────────────────────────────────────────────────────────

const (
	cReset  = "\033[0m"
	cBold   = "\033[1m"
	cDim    = "\033[2m"
	cRed    = "\033[31m"
	cGreen  = "\033[32m"
	cYellow = "\033[33m"
	cBlue   = "\033[34m"
	cCyan   = "\033[36m"
	cWhite  = "\033[97m"
	cBgBlue = "\033[44m"
)

// ─── Data types ────────────────────────────────────────────────────────────────

type setupProviderOption struct {
	Label        string
	Provider     string
	DefaultModel string
	Cloud        bool
	Description  string
	KeyURL       string
}

type setupChannelOption struct {
	Label       string
	Channel     string
	Description string
}

type envDetectResult struct {
	OllamaFound    bool
	NetworkOnline  bool
	NPUDetected    bool
	NPUDevice      string
}

// ─── Entry point ───────────────────────────────────────────────────────────────

func maybeRunZeroConfigWizard() bool {
	home, _ := os.UserHomeDir()
	if hasConfigInPaths(defaultConfigSearchPaths(home)) {
		return false
	}

	if !isInteractiveTerminal() {
		fmt.Println("No configuration found and no interactive terminal detected.")
		fmt.Println("Run: picoclaw onboard")
		os.Exit(1)
	}

	if err := runZeroConfigWizard(); err != nil {
		fmt.Printf("\n%s%s Error: %v%s\n", cBold, cRed, err, cReset)
		os.Exit(1)
	}
	return true
}

func defaultConfigSearchPaths(home string) []string {
	paths := []string{"./config.json", "./config.yaml"}
	if home != "" {
		paths = append(paths,
			filepath.Join(home, ".picoclaw", "config.json"),
			filepath.Join(home, ".picoclaw", "config.yaml"),
		)
	}
	paths = append(paths, "/etc/picoclaw/config.json", "/etc/picoclaw/config.yaml")
	return paths
}

func hasConfigInPaths(paths []string) bool {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func isInteractiveTerminal() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (st.Mode() & os.ModeCharDevice) != 0
}

// ─── UI drawing helpers ────────────────────────────────────────────────────────

func printBanner() {
	fmt.Println()
	art := []string{
		`██████╗ ██╗ ██████╗ ██████╗  ██████╗██╗      █████╗ ██╗    ██╗`,
		`██╔══██╗██║██╔════╝██╔═══██╗██╔════╝██║     ██╔══██╗██║    ██║`,
		`██████╔╝██║██║     ██║   ██║██║     ██║     ███████║██║ █╗ ██║`,
		`██╔═══╝ ██║██║     ██║   ██║██║     ██║     ██╔══██║██║███╗██║`,
		`██║     ██║╚██████╗╚██████╔╝╚██████╗███████╗██║  ██║╚███╔███╔╝`,
		`╚═╝     ╚═╝ ╚═════╝ ╚═════╝  ╚═════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝ `,
	}
	for _, line := range art {
		fmt.Printf("  %s%s%s%s\n", cBold, cCyan, line, cReset)
	}
	fmt.Println()
	fmt.Printf("  %s%s🦞 Interactive Setup Wizard%s\n", cBold, cWhite, cReset)
	fmt.Println()
}

func printDivider() {
	fmt.Printf("  %s────────────────────────────────────────────────────────────%s\n", cDim, cReset)
}

func printStepHeader(current, total int, title string) {
	fmt.Println()
	printDivider()

	// Progress bar
	barWidth := 30
	filled := (current * barWidth) / total
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Printf("  %s%sStep %d/%d%s  %s%s%s  %s%s%s\n",
		cBold, cBgBlue, current, total, cReset,
		cCyan, bar, cReset,
		cBold, title, cReset)
	printDivider()
	fmt.Println()
}

func printDetectLine(name string, found bool, detail string) {
	icon := fmt.Sprintf("%s✗%s", cRed, cReset)
	status := fmt.Sprintf("%snot found%s", cDim, cReset)
	if found {
		icon = fmt.Sprintf("%s✓%s", cGreen, cReset)
		status = fmt.Sprintf("%s%s%s", cGreen, detail, cReset)
	}
	fmt.Printf("  %s  %-16s %s\n", icon, name, status)
}

func printInfoBox(lines []string) {
	fmt.Printf("  %s┌─────────────────────────────────────────────────────────┐%s\n", cDim, cReset)
	for _, line := range lines {
		padded := line + strings.Repeat(" ", max(0, 55-visibleLen(line)))
		fmt.Printf("  %s│%s %s %s│%s\n", cDim, cReset, padded, cDim, cReset)
	}
	fmt.Printf("  %s└─────────────────────────────────────────────────────────┘%s\n", cDim, cReset)
}

func printSummaryRow(label, value string) {
	fmt.Printf("  %-20s %s%s%s\n", label, cCyan, value, cReset)
}

func visibleLen(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		n++
	}
	return n
}

// ─── Environment detection ─────────────────────────────────────────────────────

func detectEnvironment() envDetectResult {
	var result envDetectResult

	_, err := exec.LookPath("ollama")
	result.OllamaFound = err == nil

	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.Dial("tcp", "1.1.1.1:53")
	if err == nil {
		_ = conn.Close()
		result.NetworkOnline = true
	}

	for _, p := range []string{"/dev/apex_0", "/dev/npu", "/dev/rknn"} {
		if _, err := os.Stat(p); err == nil {
			result.NPUDetected = true
			result.NPUDevice = p
			break
		}
	}

	return result
}

func printEnvironmentScan(env envDetectResult) {
	fmt.Printf("  %s%sScanning your system...%s\n\n", cBold, cYellow, cReset)

	printDetectLine("Ollama", env.OllamaFound, "installed")
	printDetectLine("Internet", env.NetworkOnline, "connected")
	if env.NPUDetected {
		printDetectLine("NPU", true, env.NPUDevice)
	} else {
		printDetectLine("NPU", false, "")
	}
	fmt.Println()
}

// ─── Arrow-key interactive selector ────────────────────────────────────────────

func interactiveSelect(title string, labels []string, descriptions []string, defaultIdx int) (int, error) {
	// Try to enter raw terminal mode for arrow-key input
	fd := int(os.Stdin.Fd())
	oldState, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		// Fallback to number-based prompt if raw mode not available
		return fallbackSelect(labels, defaultIdx)
	}

	// Set raw mode (disable canonical mode and echo)
	raw := *oldState
	raw.Lflag &^= unix.ECHO | unix.ICANON
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &raw); err != nil {
		return fallbackSelect(labels, defaultIdx)
	}
	defer unix.IoctlSetTermios(fd, unix.TCSETS, oldState)

	selected := defaultIdx
	total := len(labels)

	// Hide cursor
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	renderMenu := func() {
		fmt.Printf("  %s%s? %s%s  %s(↑/↓ arrows, Enter to confirm)%s\n", cBold, cCyan, title, cReset, cDim, cReset)
		for i, label := range labels {
			if i == selected {
				fmt.Printf("  %s%s ❯ %s%s", cCyan, cBold, label, cReset)
			} else {
				fmt.Printf("    %s%s%s", cDim, label, cReset)
			}
			if i < len(descriptions) && descriptions[i] != "" {
				fmt.Printf("  %s%s%s", cDim, descriptions[i], cReset)
			}
			fmt.Println()
		}
	}

	clearMenu := func() {
		// Move up total+1 lines and clear them
		for i := 0; i < total+1; i++ {
			fmt.Print("\033[A\033[2K")
		}
	}

	renderMenu()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return selected, nil
		}

		if n == 1 {
			switch buf[0] {
			case '\r', '\n':
				clearMenu()
				fmt.Printf("  %s%s✓ %s%s %s%s%s\n", cGreen, cBold, title, cReset, cCyan, labels[selected], cReset)
				return selected, nil
			case 'k', 'K': // vim up
				if selected > 0 {
					selected--
				} else {
					selected = total - 1
				}
			case 'j', 'J': // vim down
				if selected < total-1 {
					selected++
				} else {
					selected = 0
				}
			case 3: // Ctrl+C
				fmt.Print("\033[?25h")
				unix.IoctlSetTermios(fd, unix.TCSETS, oldState)
				fmt.Println("\n  Aborted.")
				os.Exit(1)
			}
		} else if n == 3 && buf[0] == '\033' && buf[1] == '[' {
			switch buf[2] {
			case 'A': // Up arrow
				if selected > 0 {
					selected--
				} else {
					selected = total - 1
				}
			case 'B': // Down arrow
				if selected < total-1 {
					selected++
				} else {
					selected = 0
				}
			}
		}

		clearMenu()
		renderMenu()
	}
}

func fallbackSelect(labels []string, defaultIdx int) (int, error) {
	reader := bufio.NewReader(os.Stdin)
	for i, label := range labels {
		fmt.Printf("  %d) %s\n", i+1, label)
	}
	for {
		fmt.Printf("\n  Enter choice [1-%d] (default %d): ", len(labels), defaultIdx+1)
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return defaultIdx, nil
		}
		var c int
		if _, err := fmt.Sscanf(line, "%d", &c); err != nil || c < 1 || c > len(labels) {
			fmt.Printf("  %sInvalid choice. Please try again.%s\n", cRed, cReset)
			continue
		}
		return c - 1, nil
	}
}

// ─── Main wizard flow ──────────────────────────────────────────────────────────

func runZeroConfigWizard() error {
	reader := bufio.NewReader(os.Stdin)
	cfg := config.DefaultConfig()
	configPath := getConfigPath()

	totalSteps := 4

	printBanner()
	printInfoBox([]string{
		fmt.Sprintf("%sWelcome!%s It looks like this is your first time running PicoClaw.", cBold, cReset),
		"This wizard will guide you through the initial setup.",
		fmt.Sprintf("It only takes %s~1 minute%s. Let's get started!", cBold, cReset),
	})
	fmt.Println()

	// ── Step 1: Environment & Workspace ─────────────────────────────────
	printStepHeader(1, totalSteps, "Environment & Workspace")

	env := detectEnvironment()
	printEnvironmentScan(env)

	fmt.Printf("  %sPaths:%s\n", cBold, cReset)
	installDir := detectInstallDir()
	fmt.Printf("    Binary:    %s%s%s\n", cDim, installDir, cReset)
	fmt.Printf("    Config:    %s%s%s\n", cDim, configPath, cReset)
	fmt.Printf("    Workspace: %s%s%s\n", cDim, cfg.WorkspacePath(), cReset)
	fmt.Println()

	workspaceInput, err := promptString(reader,
		fmt.Sprintf("  %s?%s Workspace path", cCyan, cReset),
		cfg.WorkspacePath())
	if err != nil {
		return err
	}
	if strings.TrimSpace(workspaceInput) != "" {
		cfg.Agents.Defaults.Workspace = strings.TrimSpace(workspaceInput)
	}

	// ── Step 2: AI Provider ─────────────────────────────────────────────
	printStepHeader(2, totalSteps, "AI Provider")

	// Determine smart default based on environment
	defaultProviderIdx := 0
	if !env.OllamaFound && env.NetworkOnline {
		defaultProviderIdx = 1 // Default to OpenRouter if no Ollama but online
	}

	provider, err := promptProviderChoice(defaultProviderIdx)
	if err != nil {
		return err
	}
	if err := applyProviderChoice(cfg, provider, reader); err != nil {
		return err
	}

	// ── Step 3: Channel ─────────────────────────────────────────────────
	printStepHeader(3, totalSteps, "Chat Channel")

	channel, err := promptChannelChoice()
	if err != nil {
		return err
	}
	if err := applyChannelChoice(cfg, channel, reader); err != nil {
		return err
	}

	// ── Step 4: Summary & Save ──────────────────────────────────────────
	printStepHeader(4, totalSteps, "Review & Save")

	providerDisplay := provider.Provider
	if provider.Cloud {
		providerDisplay += " (cloud)"
	} else {
		providerDisplay += " (local)"
	}

	fmt.Printf("  %s%sConfiguration Summary%s\n\n", cBold, cWhite, cReset)
	printSummaryRow("Provider:", providerDisplay)
	printSummaryRow("Model:", cfg.Agents.Defaults.Model)
	printSummaryRow("Channel:", channel.Label)
	printSummaryRow("Workspace:", cfg.WorkspacePath())
	printSummaryRow("Config file:", configPath)
	fmt.Println()

	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	createWorkspaceTemplates(cfg.WorkspacePath())

	fmt.Printf("  %s%s✓ Configuration saved successfully!%s\n", cBold, cGreen, cReset)
	fmt.Println()

	printDivider()
	fmt.Println()

	startNow, err := promptYesNo(reader, fmt.Sprintf("  %s?%s Start chatting with the agent now?", cCyan, cReset), true)
	if err != nil {
		return err
	}

	if startNow {
		fmt.Println()
		fmt.Printf("  %s%s🚀 Launching agent...%s\n\n", cBold, cGreen, cReset)
		original := os.Args
		os.Args = []string{original[0], "agent"}
		defer func() { os.Args = original }()
		agentCmd()
		return nil
	}

	fmt.Println()
	printInfoBox([]string{
		fmt.Sprintf("%sYou're all set!%s Here are some useful commands:", cBold, cReset),
		"",
		fmt.Sprintf("  %spicoclaw agent%s        Chat in terminal", cCyan, cReset),
		fmt.Sprintf("  %spicoclaw agent -m \"Hi\"%s  Send a single message", cCyan, cReset),
		fmt.Sprintf("  %spicoclaw gateway%s      Start all chat channels", cCyan, cReset),
		fmt.Sprintf("  %spicoclaw status%s       Show current configuration", cCyan, cReset),
	})
	fmt.Println()

	return nil
}

func detectInstallDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "(unknown)"
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}
	return filepath.Dir(exePath)
}

// ─── Provider selection ────────────────────────────────────────────────────────

func promptProviderChoice(defaultIdx int) (setupProviderOption, error) {
	options := []setupProviderOption{
		{Label: "Ollama / vLLM (Local)", Provider: "vllm", DefaultModel: "qwen2.5:7b-instruct", Cloud: false,
			Description: "Run models on your own machine"},
		{Label: "OpenRouter (Cloud)", Provider: "openrouter", DefaultModel: "openai/gpt-4o-mini", Cloud: true,
			Description: "Access 100+ models via one API key", KeyURL: "https://openrouter.ai/keys"},
		{Label: "OpenAI (Cloud)", Provider: "openai", DefaultModel: "gpt-4o-mini", Cloud: true,
			Description: "GPT-4o, GPT-4o-mini", KeyURL: "https://platform.openai.com/api-keys"},
		{Label: "Anthropic (Cloud)", Provider: "anthropic", DefaultModel: "claude-sonnet-4-20250514", Cloud: true,
			Description: "Claude Sonnet, Opus, Haiku", KeyURL: "https://console.anthropic.com/settings/keys"},
		{Label: "Gemini (Cloud)", Provider: "gemini", DefaultModel: "gemini-2.5-flash", Cloud: true,
			Description: "Google's Gemini models", KeyURL: "https://aistudio.google.com/apikey"},
		{Label: "Zhipu (Cloud)", Provider: "zhipu", DefaultModel: "glm-4.7", Cloud: true,
			Description: "GLM-4 Chinese LLM", KeyURL: "https://open.bigmodel.cn/usercenter/apikeys"},
		{Label: "Groq (Cloud)", Provider: "groq", DefaultModel: "llama-3.3-70b-versatile", Cloud: true,
			Description: "Ultra-fast inference", KeyURL: "https://console.groq.com/keys"},
	}

	labels := make([]string, len(options))
	descriptions := make([]string, len(options))
	for i, opt := range options {
		labels[i] = opt.Label
		descriptions[i] = opt.Description
	}

	idx, err := interactiveSelect("Select your AI Provider", labels, descriptions, defaultIdx)
	if err != nil {
		return setupProviderOption{}, err
	}
	return options[idx], nil
}

func applyProviderChoice(cfg *config.Config, opt setupProviderOption, reader *bufio.Reader) error {
	cfg.Agents.Defaults.Provider = opt.Provider
	cfg.Agents.Defaults.Model = opt.DefaultModel

	fmt.Println()
	if opt.Provider == "vllm" {
		printInfoBox([]string{
			"PicoClaw will connect to a local OpenAI-compatible API.",
			fmt.Sprintf("Default endpoint: %shttp://127.0.0.1:11434/v1%s", cCyan, cReset),
		})
		fmt.Println()

		endpoint, err := promptString(reader,
			fmt.Sprintf("  %s?%s API endpoint", cCyan, cReset),
			"http://127.0.0.1:11434/v1")
		if err != nil {
			return err
		}
		cfg.Providers.VLLM.APIBase = endpoint

		key, err := promptString(reader,
			fmt.Sprintf("  %s?%s API key %s(optional, press Enter to skip)%s", cCyan, cReset, cDim, cReset),
			"")
		if err != nil {
			return err
		}
		cfg.Providers.VLLM.APIKey = key

		// Try to fetch models from the local endpoint
		fmt.Println()
		fmt.Printf("  %s⏳ Fetching models from %s...%s\n", cYellow, endpoint, cReset)
		models, fetchErr := fetchModelList("vllm", key, endpoint)
		if fetchErr == nil && len(models) > 0 {
			fmt.Printf("  %s✓ Found %d model%s%s\n\n", cGreen, len(models),
				func() string {
					if len(models) != 1 {
						return "s"
					}
					return ""
				}(), cReset)
			model, err := interactiveModelSelect("Select a model", models, opt.DefaultModel)
			if err != nil {
				return err
			}
			cfg.Agents.Defaults.Model = model
		} else {
			if fetchErr != nil {
				fmt.Printf("  %s⚠ Could not fetch models: %v%s\n", cDim, fetchErr, cReset)
			}
			fmt.Printf("  %sEnter model name manually:%s\n", cDim, cReset)
			model, err := promptString(reader,
				fmt.Sprintf("  %s?%s Model name", cCyan, cReset),
				opt.DefaultModel)
			if err != nil {
				return err
			}
			cfg.Agents.Defaults.Model = model
		}
		return nil
	}

	// Cloud provider
	if opt.KeyURL != "" {
		printInfoBox([]string{
			fmt.Sprintf("Get your API key at: %s%s%s", cCyan, opt.KeyURL, cReset),
		})
		fmt.Println()
	}

	fmt.Printf("  %s%sYour input is hidden for security.%s\n", cDim, cYellow, cReset)
	apiKey, err := promptSecret(fmt.Sprintf("  %s?%s Enter API key", cCyan, cReset))
	if err != nil {
		return err
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		fmt.Printf("  %s⚠ No API key entered. You can add it later in the config file.%s\n", cYellow, cReset)
	} else {
		fmt.Printf("  %s✓ API key saved%s\n", cGreen, cReset)
	}

	switch opt.Provider {
	case "openrouter":
		cfg.Providers.OpenRouter.APIKey = apiKey
	case "openai":
		cfg.Providers.OpenAI.APIKey = apiKey
	case "anthropic":
		cfg.Providers.Anthropic.APIKey = apiKey
	case "gemini":
		cfg.Providers.Gemini.APIKey = apiKey
	case "zhipu":
		cfg.Providers.Zhipu.APIKey = apiKey
	case "groq":
		cfg.Providers.Groq.APIKey = apiKey
	}

	// Fetch models from provider API
	fmt.Println()
	fmt.Printf("  %s⏳ Fetching available models...%s\n", cYellow, cReset)
	models, fetchErr := fetchModelList(opt.Provider, apiKey, "")
	if fetchErr == nil && len(models) > 0 {
		fmt.Printf("  %s✓ Found %d model%s%s\n\n", cGreen, len(models),
			func() string {
				if len(models) != 1 {
					return "s"
				}
				return ""
			}(), cReset)
		model, err := interactiveModelSelect("Select a model", models, opt.DefaultModel)
		if err != nil {
			return err
		}
		cfg.Agents.Defaults.Model = model
	} else {
		if fetchErr != nil {
			fmt.Printf("  %s⚠ Could not fetch models: %v%s\n", cDim, fetchErr, cReset)
		}
		fmt.Printf("  %sEnter model name manually:%s\n", cDim, cReset)
		model, err := promptString(reader,
			fmt.Sprintf("  %s?%s Model", cCyan, cReset),
			opt.DefaultModel)
		if err != nil {
			return err
		}
		cfg.Agents.Defaults.Model = model
	}

	return nil
}

// ─── Model list fetching from provider APIs ────────────────────────────────────

type openAIModelList struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type geminiModelList struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type ollamaModelList struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func fetchModelList(provider, apiKey, endpoint string) ([]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	var req *http.Request
	var err error
	var parseFunc func([]byte) ([]string, error)

	switch provider {
	case "openrouter":
		req, err = http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
		if err != nil {
			return nil, err
		}
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
		parseFunc = parseOpenAIModels

	case "openai":
		if apiKey == "" {
			return nil, fmt.Errorf("API key required")
		}
		req, err = http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		parseFunc = parseOpenAIModels

	case "anthropic":
		if apiKey == "" {
			return nil, fmt.Errorf("API key required")
		}
		req, err = http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		parseFunc = parseOpenAIModels

	case "gemini":
		if apiKey == "" {
			return nil, fmt.Errorf("API key required")
		}
		url := "https://generativelanguage.googleapis.com/v1beta/models?key=" + apiKey
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		parseFunc = parseGeminiModels

	case "groq":
		if apiKey == "" {
			return nil, fmt.Errorf("API key required")
		}
		req, err = http.NewRequest("GET", "https://api.groq.com/openai/v1/models", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		parseFunc = parseOpenAIModels

	case "zhipu":
		if apiKey == "" {
			return nil, fmt.Errorf("API key required")
		}
		req, err = http.NewRequest("GET", "https://open.bigmodel.cn/api/paas/v4/models", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		parseFunc = parseOpenAIModels

	case "vllm":
		// Try Ollama endpoint first (/api/tags), then OpenAI-compatible (/models)
		base := strings.TrimSuffix(endpoint, "/v1")
		base = strings.TrimSuffix(base, "/")
		models, ollamaErr := fetchOllamaModels(client, base)
		if ollamaErr == nil && len(models) > 0 {
			return models, nil
		}
		// Fall back to OpenAI-compatible endpoint
		modelsURL := strings.TrimSuffix(endpoint, "/") + "/models"
		req, err = http.NewRequest("GET", modelsURL, nil)
		if err != nil {
			return nil, err
		}
		parseFunc = parseOpenAIModels

	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseFunc(body)
}

func fetchOllamaModels(client *http.Client, base string) ([]string, error) {
	resp, err := client.Get(base + "/api/tags")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var list ollamaModelList
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, err
	}

	models := make([]string, len(list.Models))
	for i, m := range list.Models {
		models[i] = m.Name
	}
	return models, nil
}

func parseOpenAIModels(data []byte) ([]string, error) {
	var list openAIModelList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(list.Data))
	for _, m := range list.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}
	sort.Strings(models)
	return models, nil
}

func parseGeminiModels(data []byte) ([]string, error) {
	var list geminiModelList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(list.Models))
	for _, m := range list.Models {
		name := strings.TrimPrefix(m.Name, "models/")
		if name != "" {
			models = append(models, name)
		}
	}
	sort.Strings(models)
	return models, nil
}

// ─── Searchable model selector ─────────────────────────────────────────────────

func interactiveModelSelect(title string, models []string, defaultModel string) (string, error) {
	if len(models) == 0 {
		return defaultModel, nil
	}

	// Find default index
	defaultIdx := 0
	for i, m := range models {
		if m == defaultModel {
			defaultIdx = i
			break
		}
	}

	// For small lists (≤20), use the regular arrow-key selector
	if len(models) <= 20 {
		descs := make([]string, len(models))
		idx, err := interactiveSelect(title, models, descs, defaultIdx)
		if err != nil {
			return "", err
		}
		return models[idx], nil
	}

	// For large lists (>20), use searchable selector
	return interactiveSearchableSelect(title, models, defaultModel)
}

func interactiveSearchableSelect(title string, items []string, defaultValue string) (string, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return fallbackSearchSelect(items, defaultValue)
	}

	raw := *oldState
	raw.Lflag &^= unix.ECHO | unix.ICANON
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &raw); err != nil {
		return fallbackSearchSelect(items, defaultValue)
	}
	defer unix.IoctlSetTermios(fd, unix.TCSETS, oldState)

	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	const maxVisible = 15
	searchQuery := ""
	selected := 0
	scrollOffset := 0
	filtered := items

	// Pre-select default value
	for i, item := range filtered {
		if item == defaultValue {
			selected = i
			if selected >= maxVisible {
				scrollOffset = selected - maxVisible/2
			}
			break
		}
	}

	filterItems := func() {
		if searchQuery == "" {
			filtered = items
		} else {
			query := strings.ToLower(searchQuery)
			filtered = make([]string, 0)
			for _, item := range items {
				if strings.Contains(strings.ToLower(item), query) {
					filtered = append(filtered, item)
				}
			}
		}
		if selected >= len(filtered) {
			selected = 0
		}
		scrollOffset = 0
		if selected >= maxVisible {
			scrollOffset = selected - maxVisible/2
		}
	}

	lastRenderedLines := 0

	renderSearchMenu := func() {
		lines := 0

		fmt.Printf("  %s%s? %s%s  %s(%d models, type to search, ↑/↓ navigate, Enter confirm)%s\n",
			cBold, cCyan, title, cReset, cDim, len(items), cReset)
		lines++

		if searchQuery != "" {
			fmt.Printf("  %s🔍 %s%s  %s(%d match%s)%s\n",
				cYellow, searchQuery, cReset, cDim, len(filtered),
				func() string {
					if len(filtered) != 1 {
						return "es"
					}
					return ""
				}(), cReset)
		} else {
			fmt.Printf("  %s🔍 Type to filter...%s\n", cDim, cReset)
		}
		lines++

		if len(filtered) == 0 {
			fmt.Printf("  %sNo matching models found.%s\n", cDim, cReset)
			lines++
		} else {
			if selected < scrollOffset {
				scrollOffset = selected
			}
			if selected >= scrollOffset+maxVisible {
				scrollOffset = selected - maxVisible + 1
			}
			if scrollOffset < 0 {
				scrollOffset = 0
			}

			end := scrollOffset + maxVisible
			if end > len(filtered) {
				end = len(filtered)
			}

			if scrollOffset > 0 {
				fmt.Printf("  %s  ↑ %d more above%s\n", cDim, scrollOffset, cReset)
				lines++
			}

			for i := scrollOffset; i < end; i++ {
				if i == selected {
					fmt.Printf("  %s%s ❯ %s%s\n", cCyan, cBold, filtered[i], cReset)
				} else {
					fmt.Printf("    %s%s%s\n", cDim, filtered[i], cReset)
				}
				lines++
			}

			remaining := len(filtered) - end
			if remaining > 0 {
				fmt.Printf("  %s  ↓ %d more below%s\n", cDim, remaining, cReset)
				lines++
			}
		}

		lastRenderedLines = lines
	}

	clearSearchMenu := func() {
		for i := 0; i < lastRenderedLines; i++ {
			fmt.Print("\033[A\033[2K")
		}
	}

	renderSearchMenu()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			if len(filtered) > 0 {
				return filtered[selected], nil
			}
			return defaultValue, nil
		}

		if n == 1 {
			switch {
			case buf[0] == '\r' || buf[0] == '\n':
				clearSearchMenu()
				result := defaultValue
				if len(filtered) > 0 {
					result = filtered[selected]
				}
				fmt.Printf("  %s%s✓ %s%s %s%s%s\n", cGreen, cBold, title, cReset, cCyan, result, cReset)
				return result, nil

			case buf[0] == 3: // Ctrl+C
				fmt.Print("\033[?25h")
				unix.IoctlSetTermios(fd, unix.TCSETS, oldState)
				fmt.Println("\n  Aborted.")
				os.Exit(1)

			case buf[0] == 127 || buf[0] == 8: // Backspace
				if len(searchQuery) > 0 {
					_, size := utf8.DecodeLastRuneInString(searchQuery)
					searchQuery = searchQuery[:len(searchQuery)-size]
					filterItems()
				}

			case buf[0] >= 32 && buf[0] < 127: // Printable ASCII
				searchQuery += string(buf[0])
				filterItems()
			}
		} else if n == 3 && buf[0] == '\033' && buf[1] == '[' {
			switch buf[2] {
			case 'A': // Up
				if len(filtered) > 0 {
					if selected > 0 {
						selected--
					} else {
						selected = len(filtered) - 1
					}
				}
			case 'B': // Down
				if len(filtered) > 0 {
					if selected < len(filtered)-1 {
						selected++
					} else {
						selected = 0
					}
				}
			}
		}

		clearSearchMenu()
		renderSearchMenu()
	}
}

func fallbackSearchSelect(items []string, defaultValue string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("  Available models (%d total). Enter model name:\n", len(items))
	fmt.Printf("  %s?%s Model %s(%s)%s: ", cCyan, cReset, cDim, defaultValue, cReset)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue, nil
	}
	return line, nil
}

// ─── Channel selection ─────────────────────────────────────────────────────────

func promptChannelChoice() (setupChannelOption, error) {
	options := []setupChannelOption{
		{Label: "Terminal (CLI)", Channel: "cli", Description: "Chat right here in the terminal"},
		{Label: "Telegram Bot", Channel: "telegram", Description: "Requires a bot token from @BotFather"},
		{Label: "Discord Bot", Channel: "discord", Description: "Requires a bot token from Discord Dev Portal"},
		{Label: "Slack Bot", Channel: "slack", Description: "Requires bot + app tokens"},
	}

	labels := make([]string, len(options))
	descriptions := make([]string, len(options))
	for i, opt := range options {
		labels[i] = opt.Label
		descriptions[i] = opt.Description
	}

	idx, err := interactiveSelect("Select your primary Channel", labels, descriptions, 0)
	if err != nil {
		return setupChannelOption{}, err
	}
	return options[idx], nil
}

func applyChannelChoice(cfg *config.Config, opt setupChannelOption, reader *bufio.Reader) error {
	fmt.Println()

	switch opt.Channel {
	case "cli":
		fmt.Printf("  %s✓ Terminal mode selected — no additional setup needed.%s\n", cGreen, cReset)
		return nil
	case "telegram":
		printInfoBox([]string{
			fmt.Sprintf("Create a bot via %s@BotFather%s on Telegram to get your token.", cCyan, cReset),
		})
		fmt.Println()
		token, err := promptSecret(fmt.Sprintf("  %s?%s Telegram bot token", cCyan, cReset))
		if err != nil {
			return err
		}
		cfg.Channels.Telegram.Enabled = true
		cfg.Channels.Telegram.Token = strings.TrimSpace(token)
	case "discord":
		printInfoBox([]string{
			fmt.Sprintf("Get a token at: %shttps://discord.com/developers/applications%s", cCyan, cReset),
		})
		fmt.Println()
		token, err := promptSecret(fmt.Sprintf("  %s?%s Discord bot token", cCyan, cReset))
		if err != nil {
			return err
		}
		cfg.Channels.Discord.Enabled = true
		cfg.Channels.Discord.Token = strings.TrimSpace(token)
	case "slack":
		printInfoBox([]string{
			"You need both a Bot Token and an App Token from Slack.",
			fmt.Sprintf("Create an app at: %shttps://api.slack.com/apps%s", cCyan, cReset),
		})
		fmt.Println()
		botToken, err := promptSecret(fmt.Sprintf("  %s?%s Slack bot token (xoxb-...)", cCyan, cReset))
		if err != nil {
			return err
		}
		appToken, err := promptSecret(fmt.Sprintf("  %s?%s Slack app token (xapp-...)", cCyan, cReset))
		if err != nil {
			return err
		}
		cfg.Channels.Slack.Enabled = true
		cfg.Channels.Slack.BotToken = strings.TrimSpace(botToken)
		cfg.Channels.Slack.AppToken = strings.TrimSpace(appToken)
	}

	// Allowlist
	fmt.Println()
	fmt.Printf("  %sOptional:%s Restrict who can talk to your bot.\n", cDim, cReset)
	allowList, err := promptString(reader,
		fmt.Sprintf("  %s?%s Allowed user IDs %s(comma-separated, Enter to allow all)%s", cCyan, cReset, cDim, cReset),
		"")
	if err != nil {
		return err
	}
	items := parseCSV(allowList)

	switch opt.Channel {
	case "telegram":
		cfg.Channels.Telegram.AllowFrom = config.FlexibleStringSlice(items)
	case "discord":
		cfg.Channels.Discord.AllowFrom = config.FlexibleStringSlice(items)
	case "slack":
		cfg.Channels.Slack.AllowFrom = config.FlexibleStringSlice(items)
	}

	return nil
}

// ─── Prompt helpers ────────────────────────────────────────────────────────────

func promptYesNo(reader *bufio.Reader, text string, defaultYes bool) (bool, error) {
	defaultHint := fmt.Sprintf("%sy%s/N", cBold, cReset)
	if defaultYes {
		defaultHint = fmt.Sprintf("Y/%sn%s", cDim, cReset)
	}
	for {
		fmt.Printf("%s [%s]: ", text, defaultHint)
		line, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "" {
			return defaultYes, nil
		}
		if line == "y" || line == "yes" {
			return true, nil
		}
		if line == "n" || line == "no" {
			return false, nil
		}
		fmt.Printf("  %sPlease answer y or n.%s\n", cYellow, cReset)
	}
}

func promptString(reader *bufio.Reader, label string, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s %s(%s)%s: ", label, cDim, defaultValue, cReset)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue, nil
	}
	return line, nil
}

func promptSecret(label string) (string, error) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:     label + ": ",
		EnableMask: true,
		MaskRune:   '*',
		Stdin:      os.Stdin,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	})
	if err != nil {
		return "", err
	}
	defer rl.Close()
	line, err := rl.Readline()
	if err != nil {
		return "", err
	}
	return line, nil
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
