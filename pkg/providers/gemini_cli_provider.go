package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GeminiCliProvider implements LLMProvider using the gemini CLI as a subprocess.
type GeminiCliProvider struct {
	command   string
	workspace string
	timeout   time.Duration
}

// NewGeminiCliProvider creates a new Gemini CLI provider.
func NewGeminiCliProvider(workspace string) *GeminiCliProvider {
	return &GeminiCliProvider{
		command:   "gemini",
		workspace: workspace,
	}
}

// NewGeminiCliProviderWithTimeout creates a new Gemini CLI provider with a request timeout.
func NewGeminiCliProviderWithTimeout(workspace string, timeout time.Duration) *GeminiCliProvider {
	return &GeminiCliProvider{
		command:   "gemini",
		workspace: workspace,
		timeout:   timeout,
	}
}

// Chat implements LLMProvider.Chat by executing the gemini CLI.
func (p *GeminiCliProvider) Chat(
	ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]any,
) (*LLMResponse, error) {
	if p.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	prompt := p.buildPrompt(messages, tools)

	// --prompt "" triggers non-interactive stdin mode; the empty string is appended to stdin input.
	args := []string{"--yolo", "--output-format", "json", "--prompt", ""}
	if model != "" && model != "gemini-cli" {
		args = append(args, "--model", model)
	}

	cmd := exec.CommandContext(ctx, p.command, args...)
	if p.workspace != "" {
		cmd.Dir = p.workspace
	}
	cmd.Stdin = bytes.NewReader([]byte(prompt))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		stdoutStr := strings.TrimSpace(stdout.String())
		switch {
		case stderrStr != "" && stdoutStr != "":
			return nil, fmt.Errorf("gemini cli error: %w\nstderr: %s\nstdout: %s", err, stderrStr, stdoutStr)
		case stderrStr != "":
			return nil, fmt.Errorf("gemini cli error: %s", stderrStr)
		case stdoutStr != "":
			return nil, fmt.Errorf("gemini cli error: %w\noutput: %s", err, stdoutStr)
		default:
			return nil, fmt.Errorf("gemini cli error: %w", err)
		}
	}

	return p.parseGeminiCliResponse(stdout.String())
}

// GetDefaultModel returns the default model identifier.
func (p *GeminiCliProvider) GetDefaultModel() string {
	return "gemini-cli"
}

// buildPrompt converts messages to a prompt string for the Gemini CLI.
// System messages are prepended as instructions since Gemini CLI has no --system-prompt flag.
func (p *GeminiCliProvider) buildPrompt(messages []Message, tools []ToolDefinition) string {
	var systemParts []string
	var conversationParts []string

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemParts = append(systemParts, msg.Content)
		case "user":
			conversationParts = append(conversationParts, "User: "+msg.Content)
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

	// Simplify single user message (no prefix) when there is no system or tools context
	if len(conversationParts) == 1 && len(systemParts) == 0 && len(tools) == 0 {
		return strings.TrimPrefix(conversationParts[0], "User: ")
	}

	sb.WriteString(strings.Join(conversationParts, "\n"))
	return sb.String()
}

// parseGeminiCliResponse parses the JSON output from the gemini CLI.
func (p *GeminiCliProvider) parseGeminiCliResponse(output string) (*LLMResponse, error) {
	var resp geminiCliJSONResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse gemini cli response: %w", err)
	}

	toolCalls := extractToolCallsFromText(resp.Response)

	finishReason := "stop"
	content := resp.Response
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
		content = stripToolCallsFromText(resp.Response)
	}

	var usage *UsageInfo
	if resp.Stats.Models != nil {
		var totalInput, totalCandidates, totalAll int
		for _, m := range resp.Stats.Models {
			totalInput += m.Tokens.Input
			totalCandidates += m.Tokens.Candidates
			totalAll += m.Tokens.Total
		}
		if totalInput > 0 || totalCandidates > 0 || totalAll > 0 {
			usage = &UsageInfo{
				PromptTokens:     totalInput,
				CompletionTokens: totalCandidates,
				TotalTokens:      totalAll,
			}
		}
	}

	return &LLMResponse{
		Content:      strings.TrimSpace(content),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}

// geminiCliJSONResponse represents the JSON output from the gemini CLI.
type geminiCliJSONResponse struct {
	SessionID string              `json:"session_id"`
	Response  string              `json:"response"`
	Stats     geminiCliStatsBlock `json:"stats"`
}

// geminiCliStatsBlock holds the stats section of the gemini CLI response.
type geminiCliStatsBlock struct {
	Models map[string]geminiCliModelStats `json:"models"`
}

// geminiCliModelStats holds token usage for a single model in the stats block.
type geminiCliModelStats struct {
	Tokens geminiCliTokens `json:"tokens"`
}

// geminiCliTokens holds the token counts for a model.
type geminiCliTokens struct {
	Input      int `json:"input"`
	Candidates int `json:"candidates"`
	Total      int `json:"total"`
}
