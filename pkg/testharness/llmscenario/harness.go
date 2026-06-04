package llmscenario

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// ProviderCall captures one model request made by the runtime.
type ProviderCall struct {
	Messages []providers.Message
	Tools    []providers.ToolDefinition
	Model    string
	Options  map[string]any
}

// ProviderStep is one scripted model response in a deterministic scenario.
type ProviderStep struct {
	Name     string
	Response *providers.LLMResponse
	Err      error
	Assert   func(ProviderCall) error
}

// ScriptedProvider is a deterministic LLMProvider for end-to-end runtime tests.
// It records every request and returns scripted responses in order.
type ScriptedProvider struct {
	Model string

	mu    sync.Mutex
	steps []ProviderStep
	calls []ProviderCall
}

func NewScriptedProvider(model string, steps ...ProviderStep) *ScriptedProvider {
	if model == "" {
		model = "mock-scenario-model"
	}
	return &ScriptedProvider{
		Model: model,
		steps: append([]ProviderStep(nil), steps...),
	}
}

func (p *ScriptedProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	toolDefs []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	call := ProviderCall{
		Messages: cloneMessages(messages),
		Tools:    append([]providers.ToolDefinition(nil), toolDefs...),
		Model:    model,
		Options:  cloneMap(options),
	}
	p.calls = append(p.calls, call)

	idx := len(p.calls) - 1
	if idx >= len(p.steps) {
		return nil, fmt.Errorf("unexpected LLM call %d; scenario has %d scripted step(s)", idx+1, len(p.steps))
	}

	step := p.steps[idx]
	if step.Assert != nil {
		if err := step.Assert(call); err != nil {
			name := step.Name
			if name == "" {
				name = fmt.Sprintf("step %d", idx+1)
			}
			return nil, fmt.Errorf("scenario assertion failed at %s: %w", name, err)
		}
	}
	if step.Err != nil {
		return nil, step.Err
	}
	if step.Response == nil {
		return nil, fmt.Errorf("scenario step %d returned nil response", idx+1)
	}
	return cloneResponse(step.Response), nil
}

func (p *ScriptedProvider) GetDefaultModel() string {
	return p.Model
}

func (p *ScriptedProvider) Calls() []ProviderCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]ProviderCall(nil), p.calls...)
}

func (p *ScriptedProvider) AssertExhausted() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.calls) != len(p.steps) {
		return fmt.Errorf("scenario consumed %d LLM step(s), want %d", len(p.calls), len(p.steps))
	}
	return nil
}

func TextResponse(content string) *providers.LLMResponse {
	return &providers.LLMResponse{
		Content:      content,
		FinishReason: "stop",
	}
}

func ToolCallResponse(content string, calls ...providers.ToolCall) *providers.LLMResponse {
	return &providers.LLMResponse{
		Content:      content,
		ToolCalls:    calls,
		FinishReason: "tool_calls",
	}
}

func ToolCall(id, name string, args map[string]any) providers.ToolCall {
	return providers.ToolCall{
		ID:        id,
		Type:      "function",
		Name:      name,
		Arguments: cloneMap(args),
	}
}

type ToolCallRecord struct {
	Args    map[string]any
	Channel string
	ChatID  string
}

// StubTool records invocations and returns a deterministic ToolResult.
type StubTool struct {
	NameValue        string
	DescriptionValue string
	ParametersValue  map[string]any
	Result           *tools.ToolResult
	ExecuteFunc      func(context.Context, map[string]any) *tools.ToolResult

	mu    sync.Mutex
	calls []ToolCallRecord
}

func NewStubTool(name string, result *tools.ToolResult) *StubTool {
	return &StubTool{
		NameValue:        name,
		DescriptionValue: "deterministic scenario stub tool",
		ParametersValue: map[string]any{
			"type":                 "object",
			"additionalProperties": true,
		},
		Result: result,
	}
}

func (t *StubTool) Name() string {
	return t.NameValue
}

func (t *StubTool) Description() string {
	return t.DescriptionValue
}

func (t *StubTool) Parameters() map[string]any {
	return cloneMap(t.ParametersValue)
}

func (t *StubTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	t.mu.Lock()
	t.calls = append(t.calls, ToolCallRecord{
		Args:    cloneMap(args),
		Channel: tools.ToolChannel(ctx),
		ChatID:  tools.ToolChatID(ctx),
	})
	t.mu.Unlock()

	if t.ExecuteFunc != nil {
		return t.ExecuteFunc(ctx, args)
	}
	if t.Result != nil {
		return t.Result
	}
	return tools.NewToolResult("")
}

func (t *StubTool) Calls() []ToolCallRecord {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]ToolCallRecord(nil), t.calls...)
}

func RequireToolDefinition(name string) func(ProviderCall) error {
	return func(call ProviderCall) error {
		for _, def := range call.Tools {
			if def.Function.Name == name {
				return nil
			}
		}
		return fmt.Errorf("tool definition %q not found", name)
	}
}

func RequireLastMessage(role, contains string) func(ProviderCall) error {
	return func(call ProviderCall) error {
		if len(call.Messages) == 0 {
			return fmt.Errorf("no messages in provider call")
		}
		last := call.Messages[len(call.Messages)-1]
		if last.Role != role {
			return fmt.Errorf("last message role = %q, want %q", last.Role, role)
		}
		if contains != "" && !strings.Contains(last.Content, contains) {
			return fmt.Errorf("last %s message does not contain %q: %q", role, contains, last.Content)
		}
		return nil
	}
}

func cloneResponse(resp *providers.LLMResponse) *providers.LLMResponse {
	if resp == nil {
		return nil
	}
	out := *resp
	out.ToolCalls = append([]providers.ToolCall(nil), resp.ToolCalls...)
	if resp.Usage != nil {
		usage := *resp.Usage
		out.Usage = &usage
	}
	return &out
}

func cloneMessages(messages []providers.Message) []providers.Message {
	out := append([]providers.Message(nil), messages...)
	for i := range out {
		out[i].Media = append([]string(nil), messages[i].Media...)
		out[i].Attachments = append([]providers.Attachment(nil), messages[i].Attachments...)
		out[i].SystemParts = append([]providers.ContentBlock(nil), messages[i].SystemParts...)
		out[i].ToolCalls = append([]providers.ToolCall(nil), messages[i].ToolCalls...)
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
