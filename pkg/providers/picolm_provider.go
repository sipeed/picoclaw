package providers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// PicoLMProvider implements LLMProvider by spawning the picolm binary as a subprocess.
type PicoLMProvider struct {
	binary    string
	model     string
	maxTokens int
	threads   int
	template  string
}

// NewPicoLMProvider creates a new PicoLM provider from config.
func NewPicoLMProvider(cfg config.PicoLMProviderConfig) *PicoLMProvider {
	binary := expandHome(cfg.Binary)
	model := expandHome(cfg.Model)
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 256
	}
	threads := cfg.Threads
	if threads <= 0 {
		threads = 4
	}
	template := cfg.Template
	if template == "" {
		template = "chatml"
	}
	return &PicoLMProvider{
		binary:    binary,
		model:     model,
		maxTokens: maxTokens,
		threads:   threads,
		template:  template,
	}
}

// Chat implements LLMProvider.Chat by executing the picolm binary.
func (p *PicoLMProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	if p.binary == "" {
		return nil, fmt.Errorf("picolm binary path not configured")
	}
	if p.model == "" {
		return nil, fmt.Errorf("picolm model path not configured")
	}

	prompt := p.buildPrompt(messages, tools)

	args := []string{
		p.model,
		"-n", fmt.Sprintf("%d", p.maxTokens),
		"-j", fmt.Sprintf("%d", p.threads),
		"-t", "0.7",
	}

	if len(tools) > 0 {
		args = append(args, "--json")
	}

	cmd := exec.CommandContext(ctx, p.binary, args...)
	cmd.Stdin = bytes.NewReader([]byte(prompt))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Try to parse stdout even on non-zero exit, as picolm may print diagnostics to stderr.
	if output := strings.TrimSpace(stdout.String()); output != "" {
		toolCalls := extractToolCallsFromText(output)
		finishReason := "stop"
		content := output
		if len(toolCalls) > 0 {
			finishReason = "tool_calls"
			content = stripToolCallsFromText(output)
		}
		return &LLMResponse{
			Content:      strings.TrimSpace(content),
			ToolCalls:    toolCalls,
			FinishReason: finishReason,
		}, nil
	}

	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, ctx.Err()
		}
		if stderrStr := stderr.String(); stderrStr != "" {
			return nil, fmt.Errorf("picolm error: %s", stderrStr)
		}
		return nil, fmt.Errorf("picolm error: %w", err)
	}

	return &LLMResponse{
		Content:      "",
		FinishReason: "stop",
	}, nil
}

// GetDefaultModel returns the default model identifier.
func (p *PicoLMProvider) GetDefaultModel() string {
	return "picolm-local"
}

// buildPrompt formats messages into a ChatML template for picolm.
func (p *PicoLMProvider) buildPrompt(messages []Message, tools []ToolDefinition) string {
	var sb strings.Builder

	var systemParts []string
	for _, msg := range messages {
		if msg.Role == "system" {
			systemParts = append(systemParts, msg.Content)
		}
	}

	if len(systemParts) > 0 {
		sb.WriteString("<|system|>\n")
		sb.WriteString(strings.Join(systemParts, "\n\n"))
		sb.WriteString("</s>\n")
	}

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			sb.WriteString("<|user|>\n")
			sb.WriteString(msg.Content)
			sb.WriteString("</s>\n")
		case "assistant":
			sb.WriteString("<|assistant|>\n")
			sb.WriteString(msg.Content)
			sb.WriteString("</s>\n")
		case "tool":
			sb.WriteString("<|user|>\n")
			sb.WriteString(fmt.Sprintf("[Tool Result for %s]: %s", msg.ToolCallID, msg.Content))
			sb.WriteString("</s>\n")
		}
	}

	// Open assistant turn for the model to complete
	sb.WriteString("<|assistant|>\n")

	return sb.String()
}

// expandHome expands ~ to the user's home directory.
func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:]
		}
		return home
	}
	return path
}
