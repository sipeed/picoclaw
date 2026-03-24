package vertex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"


	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_buildURL(t *testing.T) {
	tests := []struct {
		name      string
		apiBase   string
		projectID string
		region    string
		model     string
		expected  string
	}{
		{
			name:      "Default construction",
			projectID: "my-project",
			region:    "us-central1",
			model:     "gemini-1.5-pro",
			expected:  "https://us-central1-aiplatform.googleapis.com/v1/projects/my-project/locations/us-central1/publishers/google/models/gemini-1.5-pro:generateContent",
		},
		{
			name:      "Default region",
			projectID: "my-project",
			model:     "gemini-1.5-flash",
			expected:  "https://us-central1-aiplatform.googleapis.com/v1/projects/my-project/locations/us-central1/publishers/google/models/gemini-1.5-flash:generateContent",
		},
		{
			name:      "Override with base URL without method",
			apiBase:   "http://localhost:8080/v1",
			model:     "gemini-1.0-pro",
			expected:  "http://localhost:8080/v1/models/gemini-1.0-pro:generateContent",
		},
		{
			name:      "Override with full endpoint URL",
			apiBase:   "https://my-custom-proxy.com/my-endpoint:generateContent",
			model:     "gemini-1.5-pro",
			expected:  "https://my-custom-proxy.com/my-endpoint:generateContent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProvider("key", tt.apiBase, "", tt.projectID, tt.region)
			actual := p.buildURL(tt.model)
			assert.Equal(t, tt.expected, actual)
		})
	}
}


func TestProvider_buildRequestBody(t *testing.T) {
	p := NewProvider("key", "", "", "proj", "us-central1")

	messages := []protocoltypes.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello!", Media: []string{"data:image/png;base64,iVBORw0KGgo"}},
		{Role: "assistant", ToolCalls: []protocoltypes.ToolCall{{Name: "get_weather", Arguments: map[string]any{"location": "Tokyo"}}}},
		{Role: "tool", ToolCallID: "get_weather", Content: "Sunny"},
		{Role: "assistant", ToolCalls: []protocoltypes.ToolCall{{Name: "get_time", Arguments: map[string]any{"location": "Tokyo"}}}},
		{Role: "tool", ToolCallID: "get_time", Content: "12:00 PM"},
	}

	tools := []protocoltypes.ToolDefinition{
		{
			Type: "function",
			Function: protocoltypes.ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get the current weather",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{"type": "string"},
					},
					"required": []any{"location"},
				},
			},
		},
	}

	options := map[string]any{
		"temperature": 0.5,
		"max_tokens":  1000,
	}

	req, err := p.buildRequestBody(messages, tools, options)
	require.NoError(t, err)

	// Check generation config
	genCfg, ok := req["generationConfig"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 0.5, genCfg["temperature"])
	assert.Equal(t, 1000, genCfg["maxOutputTokens"])

	// Check system instruction
	sysInstr, ok := req["systemInstruction"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "system", sysInstr["role"])
	parts := sysInstr["parts"].([]map[string]any)
	assert.Equal(t, "You are a helpful assistant.", parts[0]["text"])

	// Check contents grouping (should combine the two tool responses into one user message)
	contents, ok := req["contents"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, contents, 5) // user, assistant, user(tool), assistant, user(tool)
	assert.Equal(t, "user", contents[0]["role"])
	userParts := contents[0]["parts"].([]map[string]any)
	assert.Equal(t, "Hello!", userParts[0]["text"])
	assert.Equal(t, "image/png", userParts[1]["inlineData"].(map[string]any)["mimeType"])

	assert.Equal(t, "model", contents[1]["role"])
	assert.Equal(t, "user", contents[2]["role"])

	toolParts := contents[2]["parts"].([]map[string]any)
	assert.Equal(t, "get_weather", toolParts[0]["functionResponse"].(map[string]any)["name"])

	assert.Equal(t, "model", contents[3]["role"])
}


func TestProvider_Chat(t *testing.T) {
	// Create a mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var reqBody map[string]any
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		// Return a mock response
		mockResp := `{
			"candidates": [
				{
					"content": {
						"parts": [
							{"text": "Hello, world!"}
						]
					},
					"finishReason": "STOP"
				}
			],
			"usageMetadata": {
				"promptTokenCount": 10,
				"candidatesTokenCount": 5,
				"totalTokenCount": 15
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	}))
	defer ts.Close()

	p := NewProvider("test-key", ts.URL, "", "", "")

	messages := []protocoltypes.Message{
		{Role: "user", Content: "Hi"},
	}

	resp, err := p.Chat(context.Background(), messages, nil, "gemini-1.5-pro", nil)
	require.NoError(t, err)

	assert.Equal(t, "Hello, world!", resp.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 5, resp.Usage.CompletionTokens)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}
