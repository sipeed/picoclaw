package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiProvider_ChatSeparatesThoughtAndToolCall(t *testing.T) {
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, ":generateContent") {
			t.Fatalf("path = %s, expected generateContent endpoint", r.URL.Path)
		}
		if got := r.Header.Get("X-Goog-Api-Key"); got != "test-key" {
			t.Fatalf("X-Goog-Api-Key = %q, want %q", got, "test-key")
		}
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"role": "model",
						"parts": []any{
							map[string]any{"text": "hidden", "thought": true},
							map[string]any{"text": "visible"},
							map[string]any{
								"functionCall": map[string]any{
									"id":   "call_1",
									"name": "search",
									"args": map[string]any{"q": "hi"},
								},
								"thoughtSignature": "sig-1",
							},
						},
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     2,
				"candidatesTokenCount": 3,
				"totalTokenCount":      5,
			},
		})
	}))
	defer server.Close()

	provider := NewGeminiProvider("test-key", server.URL, "", "picoclaw-test", 0, nil, nil)
	resp, err := provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"gemini-3-flash-preview",
		map[string]any{"thinking_level": "high"},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Content != "visible" {
		t.Fatalf("Content = %q, want %q", resp.Content, "visible")
	}
	if resp.ReasoningContent != "hidden" {
		t.Fatalf("ReasoningContent = %q, want %q", resp.ReasoningContent, "hidden")
	}
	if resp.FinishReason != "tool_calls" {
		t.Fatalf("FinishReason = %q, want %q", resp.FinishReason, "tool_calls")
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 5 {
		t.Fatalf("Usage = %#v, expected total tokens = 5", resp.Usage)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ID != "call_1" {
		t.Fatalf("ToolCall ID = %q, want %q", resp.ToolCalls[0].ID, "call_1")
	}
	if resp.ToolCalls[0].Name != "search" {
		t.Fatalf("ToolCall Name = %q, want %q", resp.ToolCalls[0].Name, "search")
	}
	if resp.ToolCalls[0].ThoughtSignature != "sig-1" {
		t.Fatalf("ToolCall ThoughtSignature = %q, want %q", resp.ToolCalls[0].ThoughtSignature, "sig-1")
	}
	if resp.ToolCalls[0].Function == nil || !strings.Contains(resp.ToolCalls[0].Function.Arguments, `"q":"hi"`) {
		t.Fatalf("ToolCall Function arguments = %#v, want q=hi", resp.ToolCalls[0].Function)
	}

	generationConfig, ok := capturedBody["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("request missing generationConfig: %#v", capturedBody)
	}
	thinkingConfig, ok := generationConfig["thinkingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("request missing thinkingConfig: %#v", generationConfig)
	}
	if includeThoughts, ok := thinkingConfig["includeThoughts"].(bool); !ok || !includeThoughts {
		t.Fatalf("thinkingConfig.includeThoughts = %#v, want true", thinkingConfig["includeThoughts"])
	}
	if got := thinkingConfig["thinkingLevel"]; got != "high" {
		t.Fatalf("thinkingConfig.thinkingLevel = %#v, want %q", got, "high")
	}
}

func TestGeminiProvider_ChatStreamParsesThoughtTextAndToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ":streamGenerateContent") {
			t.Fatalf("path = %s, expected streamGenerateContent endpoint", r.URL.Path)
		}
		if got := r.URL.Query().Get("alt"); got != "sse" {
			t.Fatalf("alt query = %q, want %q", got, "sse")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer is not flushable")
		}

		chunks := []map[string]any{
			{
				"candidates": []any{map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"text": "think ", "thought": true},
							map[string]any{"text": "Hello "},
						},
					},
				}},
			},
			{
				"candidates": []any{map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"text": "World"},
							map[string]any{
								"functionCall": map[string]any{
									"id":   "call_stream",
									"name": "search",
									"args": map[string]any{"q": "stream"},
								},
							},
						},
					},
					"finishReason": "STOP",
				}},
				"usageMetadata": map[string]any{
					"promptTokenCount":     1,
					"candidatesTokenCount": 2,
					"totalTokenCount":      3,
				},
			},
		}

		for _, chunk := range chunks {
			raw, err := json.Marshal(chunk)
			if err != nil {
				t.Fatalf("marshal chunk: %v", err)
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", raw); err != nil {
				t.Fatalf("write chunk: %v", err)
			}
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	provider := NewGeminiProvider("test-key", server.URL, "", "", 0, nil, nil)
	updates := make([]string, 0)
	resp, err := provider.ChatStream(
		t.Context(),
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"gemini-2.5-flash",
		nil,
		func(accumulated string) {
			updates = append(updates, accumulated)
		},
	)
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	if resp.Content != "Hello World" {
		t.Fatalf("Content = %q, want %q", resp.Content, "Hello World")
	}
	if resp.ReasoningContent != "think " {
		t.Fatalf("ReasoningContent = %q, want %q", resp.ReasoningContent, "think ")
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].ID != "call_stream" {
		t.Fatalf("ToolCalls = %#v, want single call_stream", resp.ToolCalls)
	}
	if resp.FinishReason != "tool_calls" {
		t.Fatalf("FinishReason = %q, want %q", resp.FinishReason, "tool_calls")
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 3 {
		t.Fatalf("Usage = %#v, expected total tokens = 3", resp.Usage)
	}
	if len(updates) < 2 || updates[len(updates)-1] != "Hello World" {
		t.Fatalf("stream updates = %#v, expected final accumulated text", updates)
	}
}

func TestGeminiProvider_ChatStreamSkipsEmptyDataFrames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer is not flushable")
		}

		_, _ = fmt.Fprint(w, "data: \n\n")
		flusher.Flush()

		chunk := map[string]any{
			"candidates": []any{map[string]any{
				"content": map[string]any{
					"parts": []any{map[string]any{"text": "ok"}},
				},
				"finishReason": "STOP",
			}},
		}
		raw, err := json.Marshal(chunk)
		if err != nil {
			t.Fatalf("marshal chunk: %v", err)
		}
		_, _ = fmt.Fprintf(w, "data: %s\n\n", raw)
		flusher.Flush()
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	provider := NewGeminiProvider("test-key", server.URL, "", "", 0, nil, nil)
	resp, err := provider.ChatStream(
		t.Context(),
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"gemini-2.5-flash",
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("Content = %q, want %q", resp.Content, "ok")
	}
}

func TestGeminiProvider_ChatStreamReturnsErrorOnInvalidDataFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer is not flushable")
		}

		_, _ = fmt.Fprint(w, "data: {invalid-json}\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	provider := NewGeminiProvider("test-key", server.URL, "", "", 0, nil, nil)
	_, err := provider.ChatStream(
		t.Context(),
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"gemini-2.5-flash",
		nil,
		nil,
	)
	if err == nil {
		t.Fatal("ChatStream() expected error for invalid SSE data frame")
	}
	if !strings.Contains(err.Error(), "invalid gemini stream chunk") {
		t.Fatalf("error = %v, want contains %q", err, "invalid gemini stream chunk")
	}
}

func TestGeminiProvider_BuildRequestBody_UsesCamelCaseThoughtSignatureOnly(t *testing.T) {
	provider := NewGeminiProvider("test-key", "https://example.com/v1beta", "", "", 0, nil, nil)

	body := provider.buildRequestBody(
		[]Message{{
			Role: "assistant",
			ToolCalls: []ToolCall{{
				ID:        "call_1",
				Name:      "search",
				Arguments: map[string]any{"q": "hello"},
				Function: &FunctionCall{
					Name:             "search",
					Arguments:        `{"q":"hello"}`,
					ThoughtSignature: "sig-1",
				},
			}},
		}},
		nil,
		"gemini-2.5-flash",
		nil,
	)

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	jsonBody := string(raw)

	if !strings.Contains(jsonBody, `"thoughtSignature":"sig-1"`) {
		t.Fatalf("request body = %s, expected camelCase thoughtSignature", jsonBody)
	}
	if strings.Contains(jsonBody, `"thought_signature"`) {
		t.Fatalf("request body = %s, unexpected snake_case thought_signature", jsonBody)
	}
}

func TestGeminiProvider_ChatStreamCoalescesToolCallWithoutWireID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer is not flushable")
		}

		chunks := []map[string]any{
			{
				"candidates": []any{map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"functionCall": map[string]any{
									"name": "search",
									"args": map[string]any{"q": "first"},
								},
							},
						},
					},
				}},
			},
			{
				"candidates": []any{map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"functionCall": map[string]any{
									"name": "search",
									"args": map[string]any{"q": "second"},
								},
							},
						},
					},
					"finishReason": "STOP",
				}},
			},
		}

		for _, chunk := range chunks {
			raw, err := json.Marshal(chunk)
			if err != nil {
				t.Fatalf("marshal chunk: %v", err)
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", raw); err != nil {
				t.Fatalf("write chunk: %v", err)
			}
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	provider := NewGeminiProvider("test-key", server.URL, "", "", 0, nil, nil)
	resp, err := provider.ChatStream(
		t.Context(),
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"gemini-2.5-flash",
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "search#1" {
		t.Fatalf("ToolCall ID = %q, want %q", tc.ID, "search#1")
	}
	if tc.Name != "search" {
		t.Fatalf("ToolCall Name = %q, want %q", tc.Name, "search")
	}
	if argQ, ok := tc.Arguments["q"].(string); !ok || argQ != "second" {
		t.Fatalf("ToolCall Arguments = %#v, want q=second", tc.Arguments)
	}
	if resp.FinishReason != "tool_calls" {
		t.Fatalf("FinishReason = %q, want %q", resp.FinishReason, "tool_calls")
	}
}

func TestGeminiProvider_BuildRequestBodyIncludesMediaAndThinkingConfig(t *testing.T) {
	provider := NewGeminiProvider("test-key", "https://example.com/v1beta", "", "", 0, nil, nil)

	body := provider.buildRequestBody(
		[]Message{{
			Role:    "user",
			Content: "analyze attachments",
			Media: []string{
				"data:application/pdf;base64,UEZERGF0YQ==",
				"data:image/png;base64,aW1hZ2VEYXRh",
			},
		}},
		nil,
		"gemini-3-flash-preview",
		map[string]any{
			"thinking_level": "low",
			"max_tokens":     128,
			"temperature":    0.2,
		},
	)

	contents, ok := body["contents"].([]geminiContent)
	if !ok || len(contents) != 1 {
		t.Fatalf("contents = %#v, want one gemini content", body["contents"])
	}
	parts := contents[0].Parts
	mimeSet := map[string]bool{}
	for _, part := range parts {
		if part.InlineData != nil {
			mimeSet[part.InlineData.MIMEType] = true
		}
	}
	if !mimeSet["application/pdf"] {
		t.Fatalf("inline media missing application/pdf: %#v", parts)
	}
	if !mimeSet["image/png"] {
		t.Fatalf("inline media missing image/png: %#v", parts)
	}

	generationConfig, ok := body["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("generationConfig = %#v, want map", body["generationConfig"])
	}
	if got := generationConfig["maxOutputTokens"]; got != 128 {
		t.Fatalf("maxOutputTokens = %#v, want 128", got)
	}
	if got := generationConfig["temperature"]; got != 0.2 {
		t.Fatalf("temperature = %#v, want 0.2", got)
	}
	thinkingConfig, ok := generationConfig["thinkingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("thinkingConfig = %#v, want map", generationConfig["thinkingConfig"])
	}
	if includeThoughts, ok := thinkingConfig["includeThoughts"].(bool); !ok || !includeThoughts {
		t.Fatalf("includeThoughts = %#v, want true", thinkingConfig["includeThoughts"])
	}
	if got := thinkingConfig["thinkingLevel"]; got != "low" {
		t.Fatalf("thinkingLevel = %#v, want %q", got, "low")
	}
}

func TestGeminiProvider_BuildRequestBody_UsesThinkingBudgetForGemini25(t *testing.T) {
	provider := NewGeminiProvider("test-key", "https://example.com/v1beta", "", "", 0, nil, nil)
	body := provider.buildRequestBody(
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"gemini-2.5-flash",
		map[string]any{"thinking_level": "medium"},
	)

	generationConfig, ok := body["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("generationConfig = %#v, want map", body["generationConfig"])
	}
	thinkingConfig, ok := generationConfig["thinkingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("thinkingConfig = %#v, want map", generationConfig["thinkingConfig"])
	}
	if got := thinkingConfig["thinkingBudget"]; got != 4096 {
		t.Fatalf("thinkingBudget = %#v, want 4096", got)
	}
	if _, hasLevel := thinkingConfig["thinkingLevel"]; hasLevel {
		t.Fatalf("thinkingLevel should not be set for Gemini 2.5: %#v", thinkingConfig)
	}
}

func TestGeminiProvider_BuildRequestBody_OmitsThinkingConfigForGemini20(t *testing.T) {
	provider := NewGeminiProvider("test-key", "https://example.com/v1beta", "", "", 0, nil, nil)
	body := provider.buildRequestBody(
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"gemini-2.0-flash-exp",
		map[string]any{"thinking_level": "high"},
	)

	if _, ok := body["generationConfig"]; ok {
		t.Fatalf("generationConfig should be omitted for Gemini 2.0 when only thinking_level is set: %#v", body)
	}
}

func TestGeminiProvider_BuildRequestBody_DefaultsThinkingOffForGemini25(t *testing.T) {
	provider := NewGeminiProvider("test-key", "https://example.com/v1beta", "", "", 0, nil, nil)
	body := provider.buildRequestBody(
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"gemini-2.5-flash",
		nil,
	)

	generationConfig, ok := body["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("generationConfig = %#v, want map", body["generationConfig"])
	}
	thinkingConfig, ok := generationConfig["thinkingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("thinkingConfig = %#v, want map", generationConfig["thinkingConfig"])
	}
	if got := thinkingConfig["thinkingBudget"]; got != 0 {
		t.Fatalf("thinkingBudget = %#v, want 0 for default/off", got)
	}
	if includeThoughts, ok := thinkingConfig["includeThoughts"].(bool); !ok || includeThoughts {
		t.Fatalf("includeThoughts = %#v, want false for default/off", thinkingConfig["includeThoughts"])
	}
}

func TestGeminiProvider_BuildRequestBody_DefaultsThinkingOffForGemini3(t *testing.T) {
	provider := NewGeminiProvider("test-key", "https://example.com/v1beta", "", "", 0, nil, nil)
	body := provider.buildRequestBody(
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"gemini-3-flash-preview",
		nil,
	)

	generationConfig, ok := body["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("generationConfig = %#v, want map", body["generationConfig"])
	}
	thinkingConfig, ok := generationConfig["thinkingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("thinkingConfig = %#v, want map", generationConfig["thinkingConfig"])
	}
	if got := thinkingConfig["thinkingLevel"]; got != "minimal" {
		t.Fatalf("thinkingLevel = %#v, want minimal for default/off", got)
	}
	if includeThoughts, ok := thinkingConfig["includeThoughts"].(bool); !ok || includeThoughts {
		t.Fatalf("includeThoughts = %#v, want false for default/off", thinkingConfig["includeThoughts"])
	}
}

func TestGeminiProvider_BuildRequestBody_DefaultsThinkingOffForGemini25Pro(t *testing.T) {
	provider := NewGeminiProvider("test-key", "https://example.com/v1beta", "", "", 0, nil, nil)
	body := provider.buildRequestBody(
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"gemini-2.5-pro",
		nil,
	)

	generationConfig, ok := body["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("generationConfig = %#v, want map", body["generationConfig"])
	}
	thinkingConfig, ok := generationConfig["thinkingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("thinkingConfig = %#v, want map", generationConfig["thinkingConfig"])
	}
	if includeThoughts, ok := thinkingConfig["includeThoughts"].(bool); !ok || includeThoughts {
		t.Fatalf("includeThoughts = %#v, want false for default/off", thinkingConfig["includeThoughts"])
	}
	if _, hasBudget := thinkingConfig["thinkingBudget"]; hasBudget {
		t.Fatalf("thinkingBudget should be omitted for Gemini 2.5 Pro default/off: %#v", thinkingConfig)
	}
}

func TestGeminiProvider_BuildRequestBody_DefaultsThinkingOffForGemini31Pro(t *testing.T) {
	provider := NewGeminiProvider("test-key", "https://example.com/v1beta", "", "", 0, nil, nil)
	body := provider.buildRequestBody(
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"gemini-3.1-pro",
		nil,
	)

	generationConfig, ok := body["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("generationConfig = %#v, want map", body["generationConfig"])
	}
	thinkingConfig, ok := generationConfig["thinkingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("thinkingConfig = %#v, want map", generationConfig["thinkingConfig"])
	}
	if includeThoughts, ok := thinkingConfig["includeThoughts"].(bool); !ok || includeThoughts {
		t.Fatalf("includeThoughts = %#v, want false for default/off", thinkingConfig["includeThoughts"])
	}
	if _, hasLevel := thinkingConfig["thinkingLevel"]; hasLevel {
		t.Fatalf("thinkingLevel should be omitted for Gemini 3.1 Pro default/off: %#v", thinkingConfig)
	}
}

func TestGeminiProvider_BuildRequestBody_PreservesMultipleSystemMessages(t *testing.T) {
	provider := NewGeminiProvider("test-key", "https://example.com/v1beta", "", "", 0, nil, nil)
	body := provider.buildRequestBody(
		[]Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "system", Content: "Be concise."},
			{Role: "user", Content: "hello"},
		},
		nil,
		"gemini-3-flash-preview",
		nil,
	)

	systemInstruction, ok := body["systemInstruction"].(*geminiContent)
	if !ok || systemInstruction == nil {
		t.Fatalf("systemInstruction = %#v, want *geminiContent", body["systemInstruction"])
	}
	if len(systemInstruction.Parts) != 2 {
		t.Fatalf("systemInstruction.Parts len = %d, want 2", len(systemInstruction.Parts))
	}
	if systemInstruction.Parts[0].Text != "You are helpful." || systemInstruction.Parts[1].Text != "Be concise." {
		t.Fatalf("systemInstruction.Parts = %#v, want ordered system prompts", systemInstruction.Parts)
	}
}

func TestGeminiProvider_BuildRequestBody_PreservesToolResponseMedia(t *testing.T) {
	provider := NewGeminiProvider("test-key", "https://example.com/v1beta", "", "", 0, nil, nil)
	body := provider.buildRequestBody(
		[]Message{
			{
				Role: "assistant",
				ToolCalls: []ToolCall{{
					ID:        "call_1",
					Name:      "load_image",
					Arguments: map[string]any{"path": "demo.png"},
				}},
			},
			{
				Role:       "tool",
				ToolCallID: "call_1",
				Content:    "tool result",
				Media: []string{
					"data:image/png;base64,aW1hZ2VEYXRh",
					"data:application/pdf;base64,UEZERGF0YQ==",
				},
			},
		},
		nil,
		"gemini-3-flash-preview",
		nil,
	)

	contents, ok := body["contents"].([]geminiContent)
	if !ok || len(contents) != 2 {
		t.Fatalf("contents = %#v, want two content entries", body["contents"])
	}
	parts := contents[1].Parts
	if len(parts) != 1 || parts[0].FunctionResponse == nil {
		t.Fatalf("tool response part = %#v, want functionResponse", parts)
	}
	response := parts[0].FunctionResponse
	if response.Name != "load_image" {
		t.Fatalf("functionResponse.Name = %q, want %q", response.Name, "load_image")
	}
	if response.Response["result"] != "tool result" {
		t.Fatalf("functionResponse.Response = %#v, want result=tool result", response.Response)
	}
	if len(response.Parts) != 2 {
		t.Fatalf("functionResponse.Parts len = %d, want 2", len(response.Parts))
	}
}

func TestGeminiProvider_ChatAllowsCustomAuthHeaderWithoutAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer test-token")
		}
		if got := r.Header.Get("X-Goog-Api-Key"); got != "" {
			t.Fatalf("X-Goog-Api-Key = %q, want empty", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "ok"}},
					},
					"finishReason": "STOP",
				},
			},
		})
	}))
	defer server.Close()

	provider := NewGeminiProvider(
		"",
		server.URL,
		"",
		"",
		0,
		nil,
		map[string]string{"Authorization": "Bearer test-token"},
	)

	resp, err := provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"gemini-2.5-flash",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("Content = %q, want %q", resp.Content, "ok")
	}
}

func TestGeminiProvider_ChatAllowsMissingAPIKeyForCustomAPIBase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Goog-Api-Key"); got != "" {
			t.Fatalf("X-Goog-Api-Key = %q, want empty", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []any{
				map[string]any{
					"content":      map[string]any{"parts": []any{map[string]any{"text": "ok"}}},
					"finishReason": "STOP",
				},
			},
		})
	}))
	defer server.Close()

	provider := NewGeminiProvider("", server.URL, "", "", 0, nil, nil)
	resp, err := provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hello"}},
		nil,
		"gemini-2.5-flash",
		nil,
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("Content = %q, want %q", resp.Content, "ok")
	}
}
