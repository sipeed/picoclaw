// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// QwenCliProvider implements LLMProvider by wrapping the qwen CLI as a subprocess.
type QwenCliProvider struct {
	command   string
	workspace string
}

// NewQwenCliProvider creates a new Qwen CLI provider.
func NewQwenCliProvider(workspace string) *QwenCliProvider {
	return &QwenCliProvider{
		command:   "qwen",
		workspace: workspace,
	}
}

// Chat implements LLMProvider.Chat by executing the qwen CLI in non-interactive mode.
func (p *QwenCliProvider) Chat(
	ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]any,
) (*LLMResponse, error) {
	if p.command == "" {
		return nil, fmt.Errorf("qwen command not configured")
	}

	prompt := p.buildPrompt(messages, tools)

	args := []string{
		"-p",
		"--output-format=json",
		"--yolo",
	}
	if model != "" && model != "qwen-cli" {
		args = append(args, "-m", model)
	}
	args = append(args, "-") // read prompt from stdin

	cmd := exec.CommandContext(ctx, p.command, args...)
	cmd.Stdin = bytes.NewReader([]byte(prompt))
	if p.workspace != "" {
		cmd.Dir = p.workspace
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Parse JSON output even if exit code is non-zero,
	// because qwen may write diagnostic noise to stderr
	// but still produce valid JSON output.
	if stdoutStr := stdout.String(); stdoutStr != "" {
		resp, parseErr := p.parseJSONEvents(stdoutStr)
		if parseErr == nil && resp != nil && (resp.Content != "" || len(resp.ToolCalls) > 0) {
			return resp, nil
		}
	}

	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, ctx.Err()
		}
		if stderrStr := stderr.String(); stderrStr != "" {
			return nil, fmt.Errorf("qwen cli error: %s", stderrStr)
		}
		return nil, fmt.Errorf("qwen cli error: %w", err)
	}

	return p.parseJSONEvents(stdout.String())
}

// GetDefaultModel returns the default model identifier.
func (p *QwenCliProvider) GetDefaultModel() string {
	return "qwen-cli"
}

// buildPrompt converts messages to a prompt string for the Qwen CLI.
// System messages are prepended as instructions since Qwen CLI has no --system-prompt flag.
func (p *QwenCliProvider) buildPrompt(messages []Message, tools []ToolDefinition) string {
	var systemParts []string
	var conversationParts []string

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemParts = append(systemParts, msg.Content)
		case "user":
			conversationParts = append(conversationParts, msg.Content)
		case "assistant":
			conversationParts = append(conversationParts, "Assistant: "+msg.Content)
		case "tool":
			conversationParts = append(conversationParts,
				fmt.Sprintf("[Tool Result for %s]: %s", msg.ToolCallID, msg.Content))
		}
	}

	var sb strings.Builder

	if len(systemParts) > 0 {
		sb.WriteString("## System Instructions\n\n")
		sb.WriteString(strings.Join(systemParts, "\n\n"))
		sb.WriteString("\n\n## Task\n\n")
	}

	if len(tools) > 0 {
		sb.WriteString(buildCLIToolsPrompt(tools))
		sb.WriteString("\n\n")
	}

	// Simplify single user message (no prefix)
	if len(conversationParts) == 1 && len(systemParts) == 0 && len(tools) == 0 {
		return conversationParts[0]
	}

	sb.WriteString(strings.Join(conversationParts, "\n"))
	return sb.String()
}

// qwenEvent represents a single event from qwen CLI JSON output.
type qwenEvent struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype,omitempty"`
	UUID      string          `json:"uuid,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Message   *qwenMessage    `json:"message,omitempty"`
	Result    string          `json:"result,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	Usage     *qwenUsage      `json:"usage,omitempty"`
	Error     *qwenEventError `json:"error,omitempty"`
}

type qwenMessage struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	Role    string             `json:"role"`
	Model   string             `json:"model,omitempty"`
	Content []qwenContentBlock `json:"content,omitempty"`
	Usage   *qwenUsage         `json:"usage,omitempty"`
}

type qwenContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type qwenUsage struct {
	InputTokens          int `json:"input_tokens"`
	OutputTokens         int `json:"output_tokens"`
	CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
	TotalTokens          int `json:"total_tokens,omitempty"`
}

type qwenEventError struct {
	Message string `json:"message"`
}

// parseJSONEvents processes the JSON output from qwen CLI.
func (p *QwenCliProvider) parseJSONEvents(output string) (*LLMResponse, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, fmt.Errorf("qwen cli returned empty output")
	}

	// Qwen CLI outputs a JSON array of events
	var events []qwenEvent
	if err := json.Unmarshal([]byte(output), &events); err != nil {
		return nil, fmt.Errorf("failed to parse qwen cli response: %w", err)
	}

	var contentParts []string
	var usage *UsageInfo
	var lastError string

	for _, event := range events {
		switch event.Type {
		case "assistant":
			if event.Message != nil && len(event.Message.Content) > 0 {
				for _, block := range event.Message.Content {
					if block.Type == "text" && block.Text != "" {
						contentParts = append(contentParts, block.Text)
					}
				}
			}
		case "result":
			if event.IsError {
				lastError = event.Result
				if event.Error != nil && event.Error.Message != "" {
					lastError = event.Error.Message
				}
			} else if event.Result != "" {
				// Result may contain the full response text
				// Only use it if we haven't collected content from assistant events
				if len(contentParts) == 0 {
					contentParts = append(contentParts, event.Result)
				}
			}
			if event.Usage != nil {
				usage = &UsageInfo{
					PromptTokens:     event.Usage.InputTokens,
					CompletionTokens: event.Usage.OutputTokens,
					TotalTokens:      event.Usage.TotalTokens,
				}
			}
		case "error":
			lastError = event.Error.Message
		}
	}

	if lastError != "" && len(contentParts) == 0 {
		return nil, fmt.Errorf("qwen cli: %s", lastError)
	}

	content := strings.Join(contentParts, "\n")

	// Extract tool calls from response text (same pattern as other CLI providers)
	toolCalls := extractToolCallsFromText(content)

	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
		content = stripToolCallsFromText(content)
	}

	return &LLMResponse{
		Content:      strings.TrimSpace(content),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}
