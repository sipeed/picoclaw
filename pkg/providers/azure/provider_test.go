package azure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// writeValidResponse writes a minimal valid Azure OpenAI chat completion response.
func writeValidResponse(w http.ResponseWriter) {
	resp := map[string]any{
		"choices": []map[string]any{
			{
				"message":       map[string]any{"content": "ok"},
				"finish_reason": "stop",
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func TestProviderChat_AzureURLConstruction(t *testing.T) {
	var capturedPath string
	var capturedAPIVersion string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedAPIVersion = r.URL.Query().Get("api-version")
		writeValidResponse(w)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "my-gpt5-deployment", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	wantPath := "/openai/deployments/my-gpt5-deployment/chat/completions"
	if capturedPath != wantPath {
		t.Errorf("URL path = %q, want %q", capturedPath, wantPath)
	}
	if capturedAPIVersion != azureAPIVersion {
		t.Errorf("api-version = %q, want %q", capturedAPIVersion, azureAPIVersion)
	}
}

func TestProviderChat_AzureAuthHeader(t *testing.T) {
	var capturedAPIKey string
	var capturedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAPIKey = r.Header.Get("api-key")
		capturedAuth = r.Header.Get("Authorization")
		writeValidResponse(w)
	}))
	defer server.Close()

	p := NewProvider("test-azure-key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "deployment", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if capturedAPIKey != "test-azure-key" {
		t.Errorf("api-key header = %q, want %q", capturedAPIKey, "test-azure-key")
	}
	if capturedAuth != "" {
		t.Errorf("Authorization header should be empty, got %q", capturedAuth)
	}
}

func TestProviderChat_AzureOmitsModelFromBody(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&requestBody)
		writeValidResponse(w)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "deployment", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if _, exists := requestBody["model"]; exists {
		t.Error("request body should not contain 'model' field for Azure OpenAI")
	}
}

func TestProviderChat_AzureUsesMaxCompletionTokens(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&requestBody)
		writeValidResponse(w)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"deployment",
		map[string]any{"max_tokens": 2048},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if _, exists := requestBody["max_completion_tokens"]; !exists {
		t.Error("request body should contain 'max_completion_tokens'")
	}
	if _, exists := requestBody["max_tokens"]; exists {
		t.Error("request body should not contain 'max_tokens'")
	}
}

func TestProviderChat_AzureHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	p := NewProvider("bad-key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "deployment", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProviderChat_AzureParseToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "",
						"tool_calls": []map[string]any{
							{
								"id":   "call_1",
								"type": "function",
								"function": map[string]any{
									"name":      "get_weather",
									"arguments": `{"city":"Seattle"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "weather?"}}, nil, "deployment", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
	if out.ToolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", out.ToolCalls[0].Name, "get_weather")
	}
}

func TestProvider_AzureEmptyAPIBase(t *testing.T) {
	p := NewProvider("test-key", "", "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "deployment", nil)
	if err == nil {
		t.Fatal("expected error for empty API base")
	}
}

func TestProvider_AzureRequestTimeoutDefault(t *testing.T) {
	p := NewProvider("test-key", "https://example.com", "")
	if p.httpClient.Timeout != defaultRequestTimeout {
		t.Errorf("timeout = %v, want %v", p.httpClient.Timeout, defaultRequestTimeout)
	}
}

func TestProvider_AzureRequestTimeoutOverride(t *testing.T) {
	p := NewProvider("test-key", "https://example.com", "", WithRequestTimeout(300*time.Second))
	if p.httpClient.Timeout != 300*time.Second {
		t.Errorf("timeout = %v, want %v", p.httpClient.Timeout, 300*time.Second)
	}
}

func TestProvider_AzureNewProviderWithTimeout(t *testing.T) {
	p := NewProviderWithTimeout("test-key", "https://example.com", "", 180)
	if p.httpClient.Timeout != 180*time.Second {
		t.Errorf("timeout = %v, want %v", p.httpClient.Timeout, 180*time.Second)
	}
}

func TestProviderChat_AzureDeploymentNameEscaped(t *testing.T) {
	var capturedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.RawPath // use RawPath to see percent-encoding
		if capturedPath == "" {
			capturedPath = r.URL.Path
		}
		writeValidResponse(w)
	}))
	defer server.Close()

	p := NewProvider("test-key", server.URL, "")

	// Deployment name with characters that could cause path injection
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "my deploy/../../admin", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// The slash and special chars in the deployment name must be escaped, not treated as path separators
	if capturedPath == "/openai/deployments/my deploy/../../admin/chat/completions" {
		t.Fatal("deployment name was interpolated without escaping — path injection possible")
	}
}
