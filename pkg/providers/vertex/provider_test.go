package vertex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
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
			name:     "Override with base URL without method",
			apiBase:  "http://localhost:8080/v1/models",
			model:    "gemini-1.0-pro",
			expected: "http://localhost:8080/v1/models/gemini-1.0-pro:generateContent?key=key",
		},
		{
			name:     "Override with full endpoint URL",
			apiBase:  "https://my-custom-proxy.com/my-endpoint:generateContent",
			model:    "gemini-1.5-pro",
			expected: "https://my-custom-proxy.com/my-endpoint:generateContent?key=key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProvider("key", tt.apiBase, "", tt.projectID, tt.region)
			actual := p.buildURL(tt.model, "generateContent")
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestProvider_buildRequestBody(t *testing.T) {
	p := NewProvider("key", "", "", "proj", "us-central1")

	messages := []protocoltypes.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello!", Media: []string{"data:image/png;base64,iVBORw0KGgo"}},
		{
			Role: "assistant",
			ToolCalls: []protocoltypes.ToolCall{
				{Name: "get_weather", Arguments: map[string]any{"location": "Tokyo"}},
			},
		},
		{Role: "tool", ToolCallID: "get_weather", Content: "Sunny"},
		{
			Role: "assistant",
			ToolCalls: []protocoltypes.ToolCall{
				{Name: "get_time", Arguments: map[string]any{"location": "Tokyo"}},
			},
		},
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
		// Because we're using the query 'key=test-key' we no longer have Bearer authentication
		// assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Contains(t, r.URL.String(), "key=test-key")

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

func TestProvider_ChatStream(t *testing.T) {
	// Create a mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.String(), "key=test-key")
		assert.Contains(t, r.URL.String(), "alt=sse")

		var reqBody map[string]any
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "text/event-stream")
		// Write mock chunks
		w.Write([]byte(`data: {"candidates":[{"content":{"parts":[{"text":"Hello"}]}}]}` + "\n\n"))
		w.Write(
			[]byte(
				`data: {"candidates":[{"content":{"parts":[{"text":", world!"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}` + "\n\n",
			),
		)
	}))
	defer ts.Close()

	p := NewProvider("test-key", ts.URL, "", "my-project", "us-central1")
	opts := make(map[string]any)

	var chunks []string
	resp, err := p.ChatStream(
		context.Background(),
		[]protocoltypes.Message{{Role: "user", Content: "Say hello!"}},
		nil,
		"gemini-1.5-pro",
		opts,
		func(accumulated string) {
			chunks = append(chunks, accumulated)
		},
	)

	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "Hello, world!", resp.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.NotNil(t, resp.Usage)
	if resp.Usage != nil {
		assert.Equal(t, 10, resp.Usage.PromptTokens)
		assert.Equal(t, 5, resp.Usage.CompletionTokens)
		assert.Equal(t, 15, resp.Usage.TotalTokens)
	}

	assert.Equal(t, []string{"Hello", "Hello, world!"}, chunks)
}

func TestProvider_Chat_Standard(t *testing.T) {
	// Create a mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		// Standard Vertex without apiBase should use Bearer authentication
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
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	}))
	defer ts.Close()

	// Use an empty apiBase so it builds standard Vertex URLs
	p := NewProvider("test-key", "", "", "my-project", "us-central1")
	// Since buildURL will use aiplatform.googleapis.com, we override the httpClient Transport
	// to redirect requests to our mock server for this test by swapping the base URL out in a custom RoundTripper.
	p.httpClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL, _ = url.Parse(ts.URL)
		return http.DefaultTransport.RoundTrip(req)
	})

	opts := make(map[string]any)

	resp, err := p.Chat(
		context.Background(),
		[]protocoltypes.Message{{Role: "user", Content: "Say hello!"}},
		nil,
		"gemini-1.5-pro",
		opts,
	)

	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "Hello, world!", resp.Content)
	assert.Equal(t, "stop", resp.FinishReason)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
