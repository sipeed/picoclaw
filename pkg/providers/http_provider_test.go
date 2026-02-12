package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPProvider_NvidiaOptions(t *testing.T) {
	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
		}

		err := json.NewDecoder(r.Body).Decode(&capturedBody)
		if err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "Hello from Nvidia!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewHTTPProvider("test-key", server.URL)
	ctx := context.Background()
	messages := []Message{{Role: "user", Content: "Hi"}}
	options := map[string]interface{}{
		"chat_template_kwargs": map[string]interface{}{
			"thinking": true,
		},
		"top_p": 1.0,
	}

	resp, err := provider.Chat(ctx, messages, nil, "nvidia/kimi", options)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Content != "Hello from Nvidia!" {
		t.Errorf("Expected content 'Hello from Nvidia!', got %s", resp.Content)
	}

	// Verify captured body contains the custom options
	if kwargs, ok := capturedBody["chat_template_kwargs"].(map[string]interface{}); !ok || !kwargs["thinking"].(bool) {
		t.Errorf("Missing or incorrect chat_template_kwargs in request body: %v", capturedBody)
	}
	if capturedBody["top_p"].(float64) != 1.0 {
		t.Errorf("Missing or incorrect top_p in request body: %v", capturedBody)
	}
}
