// PicoClaw Doctor - Diagnostic tool for checking configuration and connectivity
package doctor

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// Check represents a single diagnostic check
type Check struct {
	Name    string
	Status  Status
	Message string
	Details []string
}

// Status represents the status of a check
type Status int

const (
	StatusOK Status = iota
	StatusWarning
	StatusError
)

func (s Status) String() string {
	switch s {
	case StatusOK:
		return "✅"
	case StatusWarning:
		return "⚠️"
	case StatusError:
		return "❌"
	default:
		return "❓"
	}
}

// Doctor runs all diagnostic checks
type Doctor struct {
	configPath string
	cfg        *config.Config
	checks     []Check
}

// NewDoctor creates a new Doctor instance
func NewDoctor(configPath string) *Doctor {
	return &Doctor{
		configPath: configPath,
	}
}

// Run executes all diagnostic checks
func (d *Doctor) Run() {
	fmt.Println("🏥 PicoClaw Doctor")
	fmt.Println("==================")
	fmt.Println()

	// Load configuration
	d.checkConfig()

	// If config loaded successfully, run more checks
	if d.cfg != nil {
		d.checkWorkspace()
		d.checkProviders()
		d.checkTools()
		d.checkChannels()
	}

	// Print summary
	d.printSummary()
}

func (d *Doctor) checkConfig() {
	check := Check{
		Name: "Configuration File",
	}

	// Check if config file exists
	if _, err := os.Stat(d.configPath); os.IsNotExist(err) {
		check.Status = StatusError
		check.Message = "Config file not found"
		check.Details = append(check.Details, fmt.Sprintf("Expected at: %s", d.configPath))
		check.Details = append(check.Details, "Run: picoclaw onboard")
		d.checks = append(d.checks, check)
		return
	}

	// Try to load config
	cfg, err := config.LoadConfig(d.configPath)
	if err != nil {
		check.Status = StatusError
		check.Message = "Failed to load configuration"
		check.Details = append(check.Details, fmt.Sprintf("Error: %v", err))
		d.checks = append(d.checks, check)
		return
	}

	d.cfg = cfg
	check.Status = StatusOK
	check.Message = "Configuration loaded successfully"
	check.Details = append(check.Details, fmt.Sprintf("Path: %s", d.configPath))
	d.checks = append(d.checks, check)
}

func (d *Doctor) checkWorkspace() {
	check := Check{
		Name: "Workspace",
	}

	workspace := d.cfg.WorkspacePath()
	info, err := os.Stat(workspace)
	if os.IsNotExist(err) {
		check.Status = StatusWarning
		check.Message = "Workspace directory does not exist"
		check.Details = append(check.Details, fmt.Sprintf("Path: %s", workspace))
		check.Details = append(check.Details, "Run: picoclaw onboard")
		d.checks = append(d.checks, check)
		return
	}

	if err != nil {
		check.Status = StatusError
		check.Message = "Cannot access workspace"
		check.Details = append(check.Details, fmt.Sprintf("Error: %v", err))
		d.checks = append(d.checks, check)
		return
	}

	if !info.IsDir() {
		check.Status = StatusError
		check.Message = "Workspace path is not a directory"
		d.checks = append(d.checks, check)
		return
	}

	// Check write permissions
	testFile := filepath.Join(workspace, ".write_test")
	f, err := os.Create(testFile)
	if err != nil {
		check.Status = StatusError
		check.Message = "Workspace is not writable"
		check.Details = append(check.Details, fmt.Sprintf("Error: %v", err))
		d.checks = append(d.checks, check)
		return
	}
	f.Close()
	os.Remove(testFile)

	check.Status = StatusOK
	check.Message = "Workspace is accessible and writable"
	check.Details = append(check.Details, fmt.Sprintf("Path: %s", workspace))
	d.checks = append(d.checks, check)
}

func (d *Doctor) checkProviders() {
	providers := []struct {
		name   string
		apiKey string
		proxy  string
	}{
		{"Anthropic", d.cfg.Providers.Anthropic.APIKey, d.cfg.Providers.Anthropic.Proxy},
		{"OpenAI", d.cfg.Providers.OpenAI.APIKey, d.cfg.Providers.OpenAI.Proxy},
		{"OpenRouter", d.cfg.Providers.OpenRouter.APIKey, d.cfg.Providers.OpenRouter.Proxy},
		{"Groq", d.cfg.Providers.Groq.APIKey, d.cfg.Providers.Groq.Proxy},
		{"Zhipu", d.cfg.Providers.Zhipu.APIKey, d.cfg.Providers.Zhipu.Proxy},
		{"Gemini", d.cfg.Providers.Gemini.APIKey, d.cfg.Providers.Gemini.Proxy},
		{"Nvidia", d.cfg.Providers.Nvidia.APIKey, d.cfg.Providers.Nvidia.Proxy},
		{"Moonshot", d.cfg.Providers.Moonshot.APIKey, d.cfg.Providers.Moonshot.Proxy},
		{"DeepSeek", d.cfg.Providers.DeepSeek.APIKey, d.cfg.Providers.DeepSeek.Proxy},
		{"Mistral", d.cfg.Providers.Mistral.APIKey, d.cfg.Providers.Mistral.Proxy},
	}

	configuredCount := 0
	for _, p := range providers {
		if p.apiKey != "" {
			configuredCount++
		}
	}

	check := Check{
		Name: "LLM Providers",
	}

	if configuredCount == 0 {
		check.Status = StatusError
		check.Message = "No LLM providers configured"
		check.Details = append(check.Details, "Add an API key to your config")
		check.Details = append(check.Details, "Supported: OpenRouter, Anthropic, OpenAI, Groq, etc.")
		d.checks = append(d.checks, check)
		return
	}

	check.Status = StatusOK
	check.Message = fmt.Sprintf("%d provider(s) configured", configuredCount)

	for _, p := range providers {
		if p.apiKey != "" {
			detail := fmt.Sprintf("✓ %s", p.name)
			if p.proxy != "" {
				detail += fmt.Sprintf(" (proxy: %s)", p.proxy)
			}
			check.Details = append(check.Details, detail)
		}
	}

	d.checks = append(d.checks, check)

	// Test connectivity for configured providers
	d.testProviderConnectivity()
}

func (d *Doctor) testProviderConnectivity() {
	check := Check{
		Name: "Provider Connectivity",
	}

	testURLs := map[string]string{
		"OpenRouter": "https://openrouter.ai/api/v1/models",
		"Groq":       "https://api.groq.com/openai/v1/models",
		"Mistral":    "https://api.mistral.ai/v1/models",
		"Moonshot":   "https://api.moonshot.cn/v1/models",
	}

	client := &http.Client{Timeout: 5 * time.Second}
	passed := 0
	failed := 0

	for name, url := range testURLs {
		// Only test if provider is configured
		var configured bool
		switch name {
		case "OpenRouter":
			configured = d.cfg.Providers.OpenRouter.APIKey != ""
		case "Groq":
			configured = d.cfg.Providers.Groq.APIKey != ""
		case "Mistral":
			configured = d.cfg.Providers.Mistral.APIKey != ""
		case "Moonshot":
			configured = d.cfg.Providers.Moonshot.APIKey != ""
		}

		if !configured {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := client.Do(req)
		cancel()

		if err != nil {
			check.Details = append(check.Details, fmt.Sprintf("❌ %s: %v", name, err))
			failed++
		} else {
			resp.Body.Close()
			if resp.StatusCode == 200 || resp.StatusCode == 401 { // 401 is OK - means API is reachable
				check.Details = append(check.Details, fmt.Sprintf("✓ %s: Reachable", name))
				passed++
			} else {
				check.Details = append(check.Details, fmt.Sprintf("⚠️ %s: HTTP %d", name, resp.StatusCode))
				failed++
			}
		}
	}

	if passed > 0 && failed == 0 {
		check.Status = StatusOK
		check.Message = fmt.Sprintf("All %d tested providers reachable", passed)
	} else if passed > 0 && failed > 0 {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("%d reachable, %d failed", passed, failed)
	} else if passed == 0 && failed > 0 {
		check.Status = StatusError
		check.Message = "All connectivity tests failed"
	} else {
		check.Status = StatusOK
		check.Message = "No providers to test"
	}

	d.checks = append(d.checks, check)
}

func (d *Doctor) checkTools() {
	check := Check{
		Name: "Tools",
	}

	tools := []struct {
		name    string
		enabled bool
		config  string
	}{
		{"Brave Search", d.cfg.Tools.Web.Brave.Enabled, d.cfg.Tools.Web.Brave.APIKey},
		{"DuckDuckGo", d.cfg.Tools.Web.DuckDuckGo.Enabled, ""},
		{"Firecrawl", d.cfg.Tools.Firecrawl.Enabled, d.cfg.Tools.Firecrawl.APIKey},
		{"SerpAPI", d.cfg.Tools.SerpAPI.Enabled, d.cfg.Tools.SerpAPI.APIKey},
	}

	enabledCount := 0
	for _, t := range tools {
		if t.enabled {
			enabledCount++
			detail := fmt.Sprintf("✓ %s", t.name)
			if t.config != "" {
				detail += " (configured)"
			}
			check.Details = append(check.Details, detail)
		}
	}

	if enabledCount == 0 {
		check.Status = StatusWarning
		check.Message = "No tools enabled"
		check.Details = append(check.Details, "At least DuckDuckGo search is recommended")
	} else {
		check.Status = StatusOK
		check.Message = fmt.Sprintf("%d tool(s) enabled", enabledCount)
	}

	d.checks = append(d.checks, check)
}

func (d *Doctor) checkChannels() {
	check := Check{
		Name: "Channels",
	}

	channels := []struct {
		name    string
		enabled bool
	}{
		{"Telegram", d.cfg.Channels.Telegram.Enabled},
		{"Discord", d.cfg.Channels.Discord.Enabled},
		{"Slack", d.cfg.Channels.Slack.Enabled},
		{"LINE", d.cfg.Channels.LINE.Enabled},
		{"WhatsApp", d.cfg.Channels.WhatsApp.Enabled},
		{"Feishu", d.cfg.Channels.Feishu.Enabled},
		{"OneBot", d.cfg.Channels.OneBot.Enabled},
	}

	enabledCount := 0
	for _, c := range channels {
		if c.enabled {
			enabledCount++
			check.Details = append(check.Details, fmt.Sprintf("✓ %s", c.name))
		}
	}

	if enabledCount == 0 {
		check.Status = StatusOK
		check.Message = "No channels enabled (CLI mode only)"
	} else {
		check.Status = StatusOK
		check.Message = fmt.Sprintf("%d channel(s) enabled", enabledCount)
	}

	d.checks = append(d.checks, check)
}

func (d *Doctor) printSummary() {
	fmt.Println()
	fmt.Println("📊 Summary")
	fmt.Println("==========")
	fmt.Println()

	okCount := 0
	warningCount := 0
	errorCount := 0

	for _, check := range d.checks {
		fmt.Printf("%s %s\n", check.Status, check.Name)
		if check.Message != "" {
			fmt.Printf("   %s\n", check.Message)
		}
		for _, detail := range check.Details {
			fmt.Printf("   %s\n", detail)
		}
		fmt.Println()

		switch check.Status {
		case StatusOK:
			okCount++
		case StatusWarning:
			warningCount++
		case StatusError:
			errorCount++
		}
	}

	fmt.Println("----------")
	fmt.Printf("✅ %d passed  ⚠️ %d warnings  ❌ %d errors\n", okCount, warningCount, errorCount)
	fmt.Println()

	if errorCount > 0 {
		fmt.Println("❌ Please fix the errors above before using picoclaw")
		os.Exit(1)
	} else if warningCount > 0 {
		fmt.Println("⚠️ PicoClaw should work, but consider addressing the warnings")
	} else {
		fmt.Println("✅ All checks passed! PicoClaw is ready to use")
	}
}

// IsHealthy returns true if all checks passed (no errors)
func (d *Doctor) IsHealthy() bool {
	for _, check := range d.checks {
		if check.Status == StatusError {
			return false
		}
	}
	return true
}
