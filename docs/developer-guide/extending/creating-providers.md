# Creating LLM Providers

This guide explains how to create custom LLM providers for PicoClaw.

## Overview

LLM providers handle communication with language model APIs. Each provider implements the `LLMProvider` interface and is responsible for:

1. Formatting requests for the specific API
2. Sending requests to the API
3. Parsing responses and tool calls
4. Handling errors and retries

## LLMProvider Interface

All providers must implement this interface:

```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDefinition,
         model string, options map[string]interface{}) (*LLMResponse, error)
    GetDefaultModel() string
}
```

### Supporting Types

```go
// Message represents a chat message
type Message struct {
    Role       string     `json:"role"`                  // "system", "user", "assistant", "tool"
    Content    string     `json:"content"`               // Message content
    ToolCalls  []ToolCall `json:"tool_calls,omitempty"`  // Tool calls from assistant
    ToolCallID string     `json:"tool_call_id,omitempty"` // ID for tool response
}

// ToolCall represents a tool call request
type ToolCall struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type,omitempty"`
    Function  *FunctionCall          `json:"function,omitempty"`
    Name      string                 `json:"name,omitempty"`
    Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// FunctionCall represents function call details
type FunctionCall struct {
    Name      string `json:"name"`
    Arguments string `json:"arguments"` // JSON string
}

// LLMResponse is the response from the LLM
type LLMResponse struct {
    Content      string     `json:"content"`
    ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
    FinishReason string     `json:"finish_reason"`
    Usage        *UsageInfo `json:"usage,omitempty"`
}

// UsageInfo contains token usage information
type UsageInfo struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}

// ToolDefinition defines a tool for the LLM
type ToolDefinition struct {
    Type     string                 `json:"type"`
    Function ToolFunctionDefinition `json:"function"`
}

// ToolFunctionDefinition describes a tool function
type ToolFunctionDefinition struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Parameters  map[string]interface{} `json:"parameters"`
}
```

## Creating a Basic Provider

### Step 1: Define the Provider Struct

```go
package myprovider

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type Provider struct {
    apiKey     string
    apiBase    string
    httpClient *http.Client
}

func NewProvider(apiKey, apiBase string) *Provider {
    return &Provider{
        apiKey:  apiKey,
        apiBase: apiBase,
        httpClient: &http.Client{
            Timeout: 120 * time.Second,
        },
    }
}
```

### Step 2: Implement GetDefaultModel

```go
func (p *Provider) GetDefaultModel() string {
    return "my-provider/default-model"
}
```

### Step 3: Implement Chat

```go
func (p *Provider) Chat(ctx context.Context, messages []protocoltypes.Message,
    tools []protocoltypes.ToolDefinition, model string,
    options map[string]interface{}) (*protocoltypes.LLMResponse, error) {

    // Build request body
    requestBody := map[string]interface{}{
        "model":    model,
        "messages": messages,
    }

    // Add tools if provided
    if len(tools) > 0 {
        requestBody["tools"] = tools
        requestBody["tool_choice"] = "auto"
    }

    // Add options
    if maxTokens, ok := options["max_tokens"]; ok {
        requestBody["max_tokens"] = maxTokens
    }
    if temperature, ok := options["temperature"]; ok {
        requestBody["temperature"] = temperature
    }

    // Marshal request
    jsonData, err := json.Marshal(requestBody)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    // Create HTTP request
    req, err := http.NewRequestWithContext(ctx, "POST",
        p.apiBase+"/chat/completions", bytes.NewReader(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+p.apiKey)

    // Send request
    resp, err := p.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    // Read response
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }

    // Check status code
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API error: status=%d body=%s", resp.StatusCode, string(body))
    }

    // Parse response
    return p.parseResponse(body)
}
```

### Step 4: Implement Response Parsing

```go
func (p *Provider) parseResponse(body []byte) (*protocoltypes.LLMResponse, error) {
    // Define response structure matching the API
    var apiResponse struct {
        Choices []struct {
            Message struct {
                Content   string `json:"content"`
                ToolCalls []struct {
                    ID       string `json:"id"`
                    Type     string `json:"type"`
                    Function *struct {
                        Name      string `json:"name"`
                        Arguments string `json:"arguments"`
                    } `json:"function"`
                } `json:"tool_calls"`
            } `json:"message"`
            FinishReason string `json:"finish_reason"`
        } `json:"choices"`
        Usage *protocoltypes.UsageInfo `json:"usage"`
    }

    // Parse JSON
    if err := json.Unmarshal(body, &apiResponse); err != nil {
        return nil, fmt.Errorf("failed to unmarshal response: %w", err)
    }

    // Handle empty response
    if len(apiResponse.Choices) == 0 {
        return &protocoltypes.LLMResponse{
            Content:      "",
            FinishReason: "stop",
        }, nil
    }

    // Extract choice
    choice := apiResponse.Choices[0]

    // Convert tool calls
    toolCalls := make([]protocoltypes.ToolCall, 0, len(choice.Message.ToolCalls))
    for _, tc := range choice.Message.ToolCalls {
        arguments := make(map[string]interface{})
        name := ""

        if tc.Function != nil {
            name = tc.Function.Name
            if tc.Function.Arguments != "" {
                if err := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); err != nil {
                    // Handle parse error - store raw arguments
                    arguments["raw"] = tc.Function.Arguments
                }
            }
        }

        toolCalls = append(toolCalls, protocoltypes.ToolCall{
            ID:        tc.ID,
            Name:      name,
            Arguments: arguments,
        })
    }

    return &protocoltypes.LLMResponse{
        Content:      choice.Message.Content,
        ToolCalls:    toolCalls,
        FinishReason: choice.FinishReason,
        Usage:        apiResponse.Usage,
    }, nil
}
```

## Error Classification

For fallback chain support, classify errors:

```go
import "github.com/sipeed/picoclaw/pkg/providers"

func (p *Provider) Chat(ctx context.Context, messages []protocoltypes.Message,
    tools []protocoltypes.ToolDefinition, model string,
    options map[string]interface{}) (*protocoltypes.LLMResponse, error) {

    // ... make request ...

    if resp.StatusCode != http.StatusOK {
        // Classify error
        reason := p.classifyError(resp.StatusCode, body)
        return nil, &providers.FailoverError{
            Reason:   reason,
            Provider: "myprovider",
            Model:    model,
            Status:   resp.StatusCode,
            Wrapped:  fmt.Errorf("API error: %s", string(body)),
        }
    }

    // ...
}

func (p *Provider) classifyError(status int, body []byte) providers.FailoverReason {
    switch status {
    case 401, 403:
        return providers.FailoverAuth
    case 429:
        return providers.FailoverRateLimit
    case 402:
        return providers.FailoverBilling
    case 504, 502:
        return providers.FailoverTimeout
    case 503:
        return providers.FailoverOverloaded
    case 400:
        return providers.FailoverFormat
    default:
        return providers.FailoverUnknown
    }
}
```

## Registering Providers

Add your provider to the factory:

```go
// In pkg/providers/factory.go

func NewProviderFromConfig(cfg *config.Config) (LLMProvider, error) {
    // ... existing checks ...

    // Add your provider
    if cfg.Providers.MyProvider.APIKey != "" {
        return myprovider.NewProvider(
            cfg.Providers.MyProvider.APIKey,
            cfg.Providers.MyProvider.APIBase,
        ), nil
    }

    // ...
}
```

## Complete Example: Custom Provider

```go
package myprovider

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/sipeed/picoclaw/pkg/providers"
    "github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type Provider struct {
    apiKey     string
    apiBase    string
    httpClient *http.Client
    defaultModel string
}

type Config struct {
    APIKey       string
    APIBase      string
    DefaultModel string
    Timeout      time.Duration
    Proxy        string
}

func NewProvider(cfg Config) *Provider {
    client := &http.Client{
        Timeout: cfg.Timeout,
    }

    if cfg.Timeout == 0 {
        client.Timeout = 120 * time.Second
    }

    if cfg.APIBase == "" {
        cfg.APIBase = "https://api.myprovider.com/v1"
    }

    if cfg.DefaultModel == "" {
        cfg.DefaultModel = "myprovider/default"
    }

    return &Provider{
        apiKey:       cfg.APIKey,
        apiBase:      cfg.APIBase,
        httpClient:   client,
        defaultModel: cfg.DefaultModel,
    }
}

func (p *Provider) GetDefaultModel() string {
    return p.defaultModel
}

func (p *Provider) Chat(ctx context.Context, messages []protocoltypes.Message,
    tools []protocoltypes.ToolDefinition, model string,
    options map[string]interface{}) (*protocoltypes.LLMResponse, error) {

    if p.apiKey == "" {
        return nil, fmt.Errorf("API key not configured")
    }

    // Normalize model name (strip provider prefix if present)
    model = normalizeModel(model)

    // Build request
    requestBody := p.buildRequestBody(messages, tools, model, options)

    // Marshal
    jsonData, err := json.Marshal(requestBody)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }

    // Create request
    url := p.apiBase + "/chat/completions"
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+p.apiKey)

    // Send
    resp, err := p.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("send request: %w", err)
    }
    defer resp.Body.Close()

    // Read body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("read response: %w", err)
    }

    // Handle errors
    if resp.StatusCode != http.StatusOK {
        return nil, p.createError(resp.StatusCode, body, model)
    }

    // Parse response
    return p.parseResponse(body)
}

func (p *Provider) buildRequestBody(messages []protocoltypes.Message,
    tools []protocoltypes.ToolDefinition, model string,
    options map[string]interface{}) map[string]interface{} {

    body := map[string]interface{}{
        "model":    model,
        "messages": messages,
    }

    if len(tools) > 0 {
        body["tools"] = tools
        body["tool_choice"] = "auto"
    }

    if maxTokens, ok := asInt(options["max_tokens"]); ok {
        body["max_tokens"] = maxTokens
    }

    if temperature, ok := asFloat(options["temperature"]); ok {
        body["temperature"] = temperature
    }

    return body
}

func (p *Provider) createError(status int, body []byte, model string) error {
    reason := providers.FailoverUnknown

    switch status {
    case 401, 403:
        reason = providers.FailoverAuth
    case 429:
        reason = providers.FailoverRateLimit
    case 402:
        reason = providers.FailoverBilling
    case 504, 502:
        reason = providers.FailoverTimeout
    case 503:
        reason = providers.FailoverOverloaded
    case 400:
        reason = providers.FailoverFormat
    }

    return &providers.FailoverError{
        Reason:   reason,
        Provider: "myprovider",
        Model:    model,
        Status:   status,
        Wrapped:  fmt.Errorf("API error: %s", string(body)),
    }
}

func normalizeModel(model string) string {
    // Strip "myprovider/" prefix if present
    if len(model) > 11 && model[:11] == "myprovider/" {
        return model[11:]
    }
    return model
}

func asInt(v interface{}) (int, bool) {
    switch val := v.(type) {
    case int:
        return val, true
    case int64:
        return int(val), true
    case float64:
        return int(val), true
    default:
        return 0, false
    }
}

func asFloat(v interface{}) (float64, bool) {
    switch val := v.(type) {
    case float64:
        return val, true
    case float32:
        return float64(val), true
    case int:
        return float64(val), true
    default:
        return 0, false
    }
}
```

## Testing Providers

```go
package myprovider

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

func TestProviderChat(t *testing.T) {
    // Create test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        if r.Header.Get("Authorization") != "Bearer test-key" {
            t.Error("missing authorization header")
        }

        // Send mock response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{
            "choices": [{
                "message": {
                    "content": "Hello!",
                    "tool_calls": []
                },
                "finish_reason": "stop"
            }],
            "usage": {
                "prompt_tokens": 10,
                "completion_tokens": 5,
                "total_tokens": 15
            }
        }`))
    }))
    defer server.Close()

    // Create provider
    provider := NewProvider(Config{
        APIKey:  "test-key",
        APIBase: server.URL,
    })

    // Test chat
    messages := []protocoltypes.Message{
        {Role: "user", Content: "Hi"},
    }

    resp, err := provider.Chat(context.Background(), messages, nil, "test-model", nil)
    if err != nil {
        t.Fatalf("Chat failed: %v", err)
    }

    if resp.Content != "Hello!" {
        t.Errorf("expected 'Hello!', got %q", resp.Content)
    }
}

func TestProviderError(t *testing.T) {
    // Create test server that returns error
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusUnauthorized)
        w.Write([]byte(`{"error": "invalid API key"}`))
    }))
    defer server.Close()

    provider := NewProvider(Config{
        APIKey:  "bad-key",
        APIBase: server.URL,
    })

    messages := []protocoltypes.Message{
        {Role: "user", Content: "Hi"},
    }

    _, err := provider.Chat(context.Background(), messages, nil, "test-model", nil)
    if err == nil {
        t.Error("expected error for 401 response")
    }
}
```

## Best Practices

1. **Context Support**: Always use context for cancellation
2. **Error Classification**: Classify errors for fallback support
3. **Timeout Configuration**: Allow configurable timeouts
4. **Model Normalization**: Handle provider prefixes in model names
5. **Retry Logic**: Consider adding retry for transient errors
6. **Logging**: Log requests and responses for debugging
7. **Rate Limiting**: Respect API rate limits

## See Also

- [Provider Interface Reference](../api/provider-interface.md)
- [Fallback Chain](../api/provider-interface.md#fallback-chain)
- [OpenAI-Compatible Provider](https://github.com/sipeed/picoclaw/tree/main/pkg/providers/openai_compat)
