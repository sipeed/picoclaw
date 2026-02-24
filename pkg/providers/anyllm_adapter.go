package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	anyllm "github.com/mozilla-ai/any-llm-go"
	"github.com/mozilla-ai/any-llm-go/providers/anthropic"
	"github.com/mozilla-ai/any-llm-go/providers/deepseek"
	"github.com/mozilla-ai/any-llm-go/providers/gemini"
	"github.com/mozilla-ai/any-llm-go/providers/groq"
	"github.com/mozilla-ai/any-llm-go/providers/llamacpp"
	"github.com/mozilla-ai/any-llm-go/providers/llamafile"
	"github.com/mozilla-ai/any-llm-go/providers/mistral"
	"github.com/mozilla-ai/any-llm-go/providers/ollama"
	"github.com/mozilla-ai/any-llm-go/providers/openai"
)

// AnyLLMAdapter wraps an any-llm-go provider to implement LLMProvider.
type AnyLLMAdapter struct {
	provider     anyllm.Provider // any-llm-go Provider interface
	defaultModel string          // e.g. "openai/gpt-5.2-chat-latest"
	modelName    string          // e.g. "gpt-5.2-chat-latest" (passed per-request)
}

// parseModel splits "provider/model_name" at the first "/".
func parseModel(model string) (providerName, modelName string) {
	idx := strings.Index(model, "/")
	if idx == -1 {
		return "", model
	}
	return model[:idx], model[idx+1:]
}

// providerAliases maps convenience names to canonical provider names.
var providerAliases = map[string]string{
	"claude": "anthropic",
	"google": "gemini",
}

// NewAnyLLMAdapter creates an AnyLLMAdapter from a model string (provider/model_name),
// an API key, and an optional base URL override.
func NewAnyLLMAdapter(model, apiKey, baseURL string) (*AnyLLMAdapter, error) {
	providerName, modelName := parseModel(model)

	// Apply aliases
	if canonical, ok := providerAliases[providerName]; ok {
		providerName = canonical
	}

	if providerName == "" {
		return nil, fmt.Errorf("model must be in provider/model_name format (e.g. openai/gpt-4): %s", model)
	}

	p, err := createAnyLLMProvider(providerName, apiKey, baseURL)
	if err != nil {
		return nil, fmt.Errorf("creating provider %q: %w", providerName, err)
	}

	return &AnyLLMAdapter{
		provider:     p,
		defaultModel: model,
		modelName:    modelName,
	}, nil
}

// createAnyLLMProvider creates the appropriate any-llm-go provider by name.
func createAnyLLMProvider(name, apiKey, baseURL string) (anyllm.Provider, error) {
	var opts []anyllm.Option
	if apiKey != "" {
		opts = append(opts, anyllm.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		opts = append(opts, anyllm.WithBaseURL(baseURL))
	}

	switch name {
	case "anthropic":
		return anthropic.New(opts...)
	case "deepseek":
		return deepseek.New(opts...)
	case "gemini":
		return gemini.New(opts...)
	case "groq":
		return groq.New(opts...)
	case "llamacpp":
		return llamacpp.New(opts...)
	case "llamafile":
		return llamafile.New(opts...)
	case "mistral":
		return mistral.New(opts...)
	case "ollama":
		return ollama.New(opts...)
	case "openai":
		return openai.New(opts...)
	default:
		return nil, fmt.Errorf("unsupported provider %q", name)
	}
}

// Chat implements LLMProvider.
func (a *AnyLLMAdapter) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error) {
	params := anyllm.CompletionParams{
		Model:    a.modelName,
		Messages: convertMessagesToAnyLLM(messages),
		Tools:    convertToolsToAnyLLM(tools),
	}

	switch v := options["max_tokens"].(type) {
	case int:
		params.MaxTokens = &v
	case float64:
		mt := int(v)
		params.MaxTokens = &mt
	}
	if temperature, ok := options["temperature"].(float64); ok {
		params.Temperature = &temperature
	}

	result, err := a.provider.Completion(ctx, params)
	if err != nil {
		return nil, err
	}

	return convertAnyLLMResult(result), nil
}

// GetDefaultModel implements LLMProvider.
func (a *AnyLLMAdapter) GetDefaultModel() string {
	return a.defaultModel
}

// convertMessagesToAnyLLM converts internal messages to any-llm-go messages.
func convertMessagesToAnyLLM(messages []Message) []anyllm.Message {
	result := make([]anyllm.Message, 0, len(messages))
	for _, msg := range messages {
		m := anyllm.Message{
			Role: msg.Role,
		}

		// Tool result messages
		if msg.ToolCallID != "" {
			m.Role = anyllm.RoleTool
			m.Content = msg.Content
			m.ToolCallID = msg.ToolCallID
			result = append(result, m)
			continue
		}

		// Build content: plain text or multimodal
		if len(msg.Media) > 0 {
			// Multimodal: text + images as ContentPart slice
			var parts []anyllm.ContentPart
			if msg.Content != "" {
				parts = append(parts, anyllm.ContentPart{
					Type: "text",
					Text: msg.Content,
				})
			}
			for _, mediaURL := range msg.Media {
				parts = append(parts, anyllm.ContentPart{
					Type:     "image_url",
					ImageURL: &anyllm.ImageURL{URL: mediaURL},
				})
			}
			m.Content = parts
		} else {
			m.Content = msg.Content
		}

		// Assistant messages with tool calls
		if len(msg.ToolCalls) > 0 {
			m.ToolCalls = make([]anyllm.ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				name := ""
				args := ""
				if tc.Function != nil {
					name = tc.Function.Name
					args = tc.Function.Arguments
				}
				if name == "" {
					name = tc.Name
				}
				if args == "" && tc.Arguments != nil {
					argsJSON, _ := json.Marshal(tc.Arguments)
					args = string(argsJSON)
				}
				m.ToolCalls = append(m.ToolCalls, anyllm.ToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: anyllm.FunctionCall{
						Name:      name,
						Arguments: args,
					},
				})
			}
		}

		result = append(result, m)
	}
	return result
}

// convertToolsToAnyLLM converts internal tool definitions to any-llm-go tools.
func convertToolsToAnyLLM(tools []ToolDefinition) []anyllm.Tool {
	result := make([]anyllm.Tool, 0, len(tools))
	for _, t := range tools {
		result = append(result, anyllm.Tool{
			Type: "function",
			Function: anyllm.Function{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		})
	}
	return result
}

// convertAnyLLMResult converts an any-llm-go ChatCompletion to our LLMResponse.
func convertAnyLLMResult(result *anyllm.ChatCompletion) *LLMResponse {
	if len(result.Choices) == 0 {
		return &LLMResponse{}
	}

	choice := result.Choices[0]
	resp := &LLMResponse{
		Content:      choice.Message.ContentString(),
		FinishReason: choice.FinishReason,
	}

	// Convert tool calls
	for _, tc := range choice.Message.ToolCalls {
		var args map[string]interface{}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)

		resp.ToolCalls = append(resp.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
			Function: &FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	// Convert usage
	if result.Usage != nil {
		resp.Usage = &UsageInfo{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		}
	}

	return resp
}
