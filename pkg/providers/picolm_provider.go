package providers

import (
	"bytes"
	"context"
	"encoding/json"
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
}

// NewPicoLMProvider creates a new PicoLM provider from config.
// Returns an error if the binary path does not point to an executable file.
func NewPicoLMProvider(cfg config.PicoLMProviderConfig) (*PicoLMProvider, error) {
	binary, err := expandHome(cfg.Binary)
	if err != nil {
		return nil, fmt.Errorf("picolm: failed to expand binary path: %w", err)
	}
	model, err := expandHome(cfg.Model)
	if err != nil {
		return nil, fmt.Errorf("picolm: failed to expand model path: %w", err)
	}

	if binary != "" {
		info, err := os.Stat(binary)
		if err != nil {
			return nil, fmt.Errorf("picolm: binary not found at %q: %w", binary, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("picolm: binary path %q is a directory, not an executable", binary)
		}
		if info.Mode()&0111 == 0 {
			return nil, fmt.Errorf("picolm: binary %q is not executable", binary)
		}
	}

	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 256
	}
	threads := cfg.Threads
	if threads <= 0 {
		threads = 4
	}
	return &PicoLMProvider{
		binary:    binary,
		model:     model,
		maxTokens: maxTokens,
		threads:   threads,
	}, nil
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
	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, ctx.Err()
		}
		if stderrStr := stderr.String(); stderrStr != "" {
			return nil, fmt.Errorf("picolm error: %s", stderrStr)
		}
		return nil, fmt.Errorf("picolm error: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return &LLMResponse{
			Content:      "",
			FinishReason: "stop",
		}, nil
	}

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

// GetDefaultModel returns the default model identifier.
func (p *PicoLMProvider) GetDefaultModel() string {
	return "picolm-local"
}

// buildPrompt formats messages into a ChatML template for picolm.
func (p *PicoLMProvider) buildPrompt(messages []Message, tools []ToolDefinition) string {
	var sb strings.Builder

	// Collect system message parts and append tool definitions.
	var systemParts []string
	for _, msg := range messages {
		if msg.Role == "system" {
			systemParts = append(systemParts, msg.Content)
		}
	}
	if len(tools) > 0 {
		systemParts = append(systemParts, buildToolsPrompt(tools))
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

// buildToolsPrompt creates the tool definitions section for injection into the system prompt.
func buildToolsPrompt(tools []ToolDefinition) string {
	var sb strings.Builder

	sb.WriteString("## Available Tools\n\n")
	sb.WriteString("When you need to use a tool, respond with ONLY a JSON object:\n\n")
	sb.WriteString("```json\n")
	sb.WriteString(`{"tool_calls":[{"id":"call_xxx","type":"function","function":{"name":"tool_name","arguments":"{...}"}}]}`)
	sb.WriteString("\n```\n\n")
	sb.WriteString("CRITICAL: The 'arguments' field MUST be a JSON-encoded STRING.\n\n")
	sb.WriteString("### Tool Definitions:\n\n")

	for _, tool := range tools {
		if tool.Type != "function" {
			continue
		}
		sb.WriteString(fmt.Sprintf("#### %s\n", tool.Function.Name))
		if tool.Function.Description != "" {
			sb.WriteString(fmt.Sprintf("Description: %s\n", tool.Function.Description))
		}
		if len(tool.Function.Parameters) > 0 {
			paramsJSON, _ := json.Marshal(tool.Function.Parameters)
			sb.WriteString(fmt.Sprintf("Parameters:\n```json\n%s\n```\n", string(paramsJSON)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// expandHome expands ~ to the user's home directory.
func expandHome(path string) (string, error) {
	if path == "" {
		return path, nil
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory: %w", err)
		}
		if len(path) > 1 && path[1] == '/' {
			return home + path[1:], nil
		}
		return home, nil
	}
	return path, nil
}
