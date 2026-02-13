package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// AntigravityProvider implements LLMProvider using the Gemini CLI (Antigravity) as a subprocess.
// Antigravity/Gemini CLI (https://github.com/google-gemini/gemini-cli) is a free,
// open-source AI coding agent that runs in the terminal.
type AntigravityProvider struct {
	command   string
	workspace string
}

// NewAntigravityProvider creates a new Antigravity/Gemini CLI provider.
func NewAntigravityProvider(workspace string) *AntigravityProvider {
	return &AntigravityProvider{
		command:   "gemini",
		workspace: workspace,
	}
}

// Chat implements LLMProvider.Chat by executing the gemini CLI.
func (p *AntigravityProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	systemPrompt := p.buildSystemPrompt(messages, tools)
	prompt := p.messagesToPrompt(messages)

	if systemPrompt != "" {
		prompt = systemPrompt + "\n\n" + prompt
	}

	// gemini -p "prompt" --json-log (headless/non-interactive mode)
	args := []string{"-p", prompt}

	// Use sandbox=false to auto-approve permissions in non-interactive mode
	args = append(args, "--sandbox", "false")

	cmd := exec.CommandContext(ctx, p.command, args...)
	if p.workspace != "" {
		cmd.Dir = p.workspace
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderrStr := stderr.String(); stderrStr != "" {
			return nil, fmt.Errorf("antigravity/gemini cli error: %s", stderrStr)
		}
		return nil, fmt.Errorf("antigravity/gemini cli error: %w", err)
	}

	return p.parseAntigravityResponse(stdout.String())
}

// GetDefaultModel returns the default model identifier.
func (p *AntigravityProvider) GetDefaultModel() string {
	return "antigravity"
}

// messagesToPrompt converts messages to a CLI-compatible prompt string.
func (p *AntigravityProvider) messagesToPrompt(messages []Message) string {
	var parts []string

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			// handled separately in buildSystemPrompt
		case "user":
			parts = append(parts, "User: "+msg.Content)
		case "assistant":
			parts = append(parts, "Assistant: "+msg.Content)
		case "tool":
			parts = append(parts, fmt.Sprintf("[Tool Result for %s]: %s", msg.ToolCallID, msg.Content))
		}
	}

	// Simplify single user message
	if len(parts) == 1 && strings.HasPrefix(parts[0], "User: ") {
		return strings.TrimPrefix(parts[0], "User: ")
	}

	return strings.Join(parts, "\n")
}

// buildSystemPrompt combines system messages and tool definitions.
func (p *AntigravityProvider) buildSystemPrompt(messages []Message, tools []ToolDefinition) string {
	var parts []string

	for _, msg := range messages {
		if msg.Role == "system" {
			parts = append(parts, msg.Content)
		}
	}

	if len(tools) > 0 {
		parts = append(parts, p.buildToolsPrompt(tools))
	}

	return strings.Join(parts, "\n\n")
}

// buildToolsPrompt creates the tool definitions section for the system prompt.
func (p *AntigravityProvider) buildToolsPrompt(tools []ToolDefinition) string {
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

// parseAntigravityResponse parses the output from the gemini CLI.
func (p *AntigravityProvider) parseAntigravityResponse(output string) (*LLMResponse, error) {
	// Try to parse as structured JSON first
	var resp antigravityJSONResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		// If JSON parsing fails, treat as plain text response
		trimmed := strings.TrimSpace(output)
		if trimmed == "" {
			return &LLMResponse{
				Content:      "",
				FinishReason: "stop",
			}, nil
		}

		// Try to find JSON object in the output
		start := strings.Index(trimmed, "{")
		if start >= 0 {
			if err2 := json.Unmarshal([]byte(trimmed[start:]), &resp); err2 != nil {
				// Return the raw text as content
				return &LLMResponse{
					Content:      trimmed,
					FinishReason: "stop",
				}, nil
			}
		} else {
			// Return raw text as content
			return &LLMResponse{
				Content:      trimmed,
				FinishReason: "stop",
			}, nil
		}
	}

	content := resp.Result
	if content == "" {
		content = resp.Message
	}
	if content == "" {
		content = resp.Response
	}
	if content == "" {
		content = resp.Text
	}

	toolCalls := p.extractToolCalls(content)

	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
		content = p.stripToolCallsJSON(content)
	}

	var usage *UsageInfo
	if resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0 {
		usage = &UsageInfo{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		}
	}

	return &LLMResponse{
		Content:      strings.TrimSpace(content),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}

// extractToolCalls parses tool call JSON from the response text.
func (p *AntigravityProvider) extractToolCalls(text string) []ToolCall {
	start := strings.Index(text, `{"tool_calls"`)
	if start == -1 {
		return nil
	}

	end := findMatchingBrace(text, start)
	if end == start {
		return nil
	}

	jsonStr := text[start:end]

	var wrapper struct {
		ToolCalls []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		return nil
	}

	var result []ToolCall
	for _, tc := range wrapper.ToolCalls {
		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)

		result = append(result, ToolCall{
			ID:        tc.ID,
			Type:      tc.Type,
			Name:      tc.Function.Name,
			Arguments: args,
			Function: &FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	return result
}

// stripToolCallsJSON removes tool call JSON from response text.
func (p *AntigravityProvider) stripToolCallsJSON(text string) string {
	start := strings.Index(text, `{"tool_calls"`)
	if start == -1 {
		return text
	}

	end := findMatchingBrace(text, start)
	if end == start {
		return text
	}

	return strings.TrimSpace(text[:start] + text[end:])
}

// antigravityJSONResponse represents the JSON output from the gemini/antigravity CLI.
type antigravityJSONResponse struct {
	Result   string               `json:"result"`
	Message  string               `json:"message"`
	Response string               `json:"response"`
	Text     string               `json:"text"`
	IsError  bool                 `json:"is_error"`
	Usage    antigravityUsageInfo `json:"usage"`
}

// antigravityUsageInfo represents token usage from the antigravity CLI response.
type antigravityUsageInfo struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
