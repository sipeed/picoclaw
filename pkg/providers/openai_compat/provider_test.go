package openai_compat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

func TestProviderChat_UsesMaxCompletionTokensForGLM(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"glm-4.7",
		map[string]any{"max_tokens": 1234},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if _, ok := requestBody["max_completion_tokens"]; !ok {
		t.Fatalf("expected max_completion_tokens in request body")
	}
	if _, ok := requestBody["max_tokens"]; ok {
		t.Fatalf("did not expect max_tokens key for glm model")
	}
}

func TestProviderChat_ParsesToolCalls(t *testing.T) {
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
									"arguments": "{\"city\":\"SF\"}",
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "gpt-4o", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
	if out.ToolCalls[0].Name != "get_weather" {
		t.Fatalf("ToolCalls[0].Name = %q, want %q", out.ToolCalls[0].Name, "get_weather")
	}
	if out.ToolCalls[0].Arguments["city"] != "SF" {
		t.Fatalf("ToolCalls[0].Arguments[city] = %v, want SF", out.ToolCalls[0].Arguments["city"])
	}
}

func TestProviderChat_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "gpt-4o", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProviderChat_StripsMoonshotPrefixAndNormalizesKimiTemperature(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"moonshot/kimi-k2.5",
		map[string]any{"temperature": 0.3},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if requestBody["model"] != "kimi-k2.5" {
		t.Fatalf("model = %v, want kimi-k2.5", requestBody["model"])
	}
	if requestBody["temperature"] != 1.0 {
		t.Fatalf("temperature = %v, want 1.0", requestBody["temperature"])
	}
}

func TestProviderChat_StripsGroqAndOllamaPrefixes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantModel string
	}{
		{
			name:      "strips groq prefix and keeps nested model",
			input:     "groq/openai/gpt-oss-120b",
			wantModel: "openai/gpt-oss-120b",
		},
		{
			name:      "strips ollama prefix",
			input:     "ollama/qwen2.5:14b",
			wantModel: "qwen2.5:14b",
		},
		{
			name:      "strips deepseek prefix",
			input:     "deepseek/deepseek-chat",
			wantModel: "deepseek-chat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestBody map[string]any

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
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
			}))
			defer server.Close()

			p := NewProvider("key", server.URL, "")
			_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, tt.input, nil)
			if err != nil {
				t.Fatalf("Chat() error = %v", err)
			}

			if requestBody["model"] != tt.wantModel {
				t.Fatalf("model = %v, want %s", requestBody["model"], tt.wantModel)
			}
		})
	}
}

func TestProvider_ProxyConfigured(t *testing.T) {
	proxyURL := "http://127.0.0.1:8080"
	p := NewProvider("key", "https://example.com", proxyURL)

	transport, ok := p.httpClient.Transport.(*http.Transport)
	if !ok || transport == nil {
		t.Fatalf("expected http transport with proxy, got %T", p.httpClient.Transport)
	}

	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "api.example.com"}}
	gotProxy, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("proxy function returned error: %v", err)
	}
	if gotProxy == nil || gotProxy.String() != proxyURL {
		t.Fatalf("proxy = %v, want %s", gotProxy, proxyURL)
	}
}

func TestProviderChat_AcceptsNumericOptionTypes(t *testing.T) {
	var requestBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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
	}))
	defer server.Close()

	p := NewProvider("key", server.URL, "")
	_, err := p.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		"gpt-4o",
		map[string]any{"max_tokens": float64(512), "temperature": 1},
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if requestBody["max_tokens"] != float64(512) {
		t.Fatalf("max_tokens = %v, want 512", requestBody["max_tokens"])
	}
	if requestBody["temperature"] != float64(1) {
		t.Fatalf("temperature = %v, want 1", requestBody["temperature"])
	}
}

func TestNormalizeModel_UsesAPIBase(t *testing.T) {
	if got := normalizeModel("deepseek/deepseek-chat", "https://api.deepseek.com/v1"); got != "deepseek-chat" {
		t.Fatalf("normalizeModel(deepseek) = %q, want %q", got, "deepseek-chat")
	}
	if got := normalizeModel("openrouter/auto", "https://openrouter.ai/api/v1"); got != "openrouter/auto" {
		t.Fatalf("normalizeModel(openrouter) = %q, want %q", got, "openrouter/auto")
	}
}

func TestNormalizeModel_OpenAIPrefix(t *testing.T) {
	if got := normalizeModel("openai/gpt-5.2", "https://api.openai.com/v1"); got != "gpt-5.2" {
		t.Fatalf("normalizeModel(openai/gpt-5.2) = %q, want %q", got, "gpt-5.2")
	}
}

func TestProviderChat_StreamingTextResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/text/chatcompletion_v2" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body["stream"] != true {
			t.Error("expected stream=true in request body")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		chunks := []string{
			`data: {"choices":[{"delta":{"content":"Hello"},"finish_reason":""}]}`,
			`data: {"choices":[{"delta":{"content":" world"},"finish_reason":""}]}`,
			`data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			fmt.Fprintln(w, c)
			fmt.Fprintln(w) // blank line between events
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	p := NewProviderWithOptions("key", server.URL, "", Options{
		EndpointPath: "/text/chatcompletion_v2",
		Stream:       true,
	})
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "MiniMax-M1", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if out.Content != "Hello world" {
		t.Fatalf("Content = %q, want %q", out.Content, "Hello world")
	}
	if out.FinishReason != "stop" {
		t.Fatalf("FinishReason = %q, want %q", out.FinishReason, "stop")
	}
	if out.Usage == nil || out.Usage.TotalTokens != 7 {
		t.Fatalf("Usage.TotalTokens = %v, want 7", out.Usage)
	}
}

func TestProviderChat_StreamingToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		chunks := []string{
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_weather","arguments":""}}]},"finish_reason":""}]}`,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":"}}]},"finish_reason":""}]}`,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"SF\"}"}}]},"finish_reason":""}]}`,
			`data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":8,"total_tokens":18}}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			fmt.Fprintln(w, c)
			fmt.Fprintln(w)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	p := NewProviderWithOptions("key", server.URL, "", Options{Stream: true})
	out, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "weather?"}}, nil, "test", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
	tc := out.ToolCalls[0]
	if tc.ID != "call_1" {
		t.Fatalf("ToolCalls[0].ID = %q, want %q", tc.ID, "call_1")
	}
	if tc.Name != "get_weather" {
		t.Fatalf("ToolCalls[0].Name = %q, want %q", tc.Name, "get_weather")
	}
	if tc.Arguments["city"] != "SF" {
		t.Fatalf("ToolCalls[0].Arguments[city] = %v, want SF", tc.Arguments["city"])
	}
}

func TestProviderChat_CustomEndpointPath(t *testing.T) {
	var hitPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitPath = r.URL.Path
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "ok"}, "finish_reason": "stop"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProviderWithOptions("key", server.URL, "", Options{
		EndpointPath: "/text/chatcompletion_v2",
	})
	_, err := p.Chat(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "test", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if hitPath != "/text/chatcompletion_v2" {
		t.Fatalf("endpoint path = %q, want %q", hitPath, "/text/chatcompletion_v2")
	}
}

func TestReadSSEIntoChannel_TextAndToolCalls(t *testing.T) {
	sseData := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"Hello"},"finish_reason":""}]}`,
		``,
		`data: {"choices":[{"delta":{"content":" world"},"finish_reason":""}]}`,
		``,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"greet","arguments":"{\"n"}}]},"finish_reason":""}]}`,
		``,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ame\":\"Bob\"}"}}]},"finish_reason":""}]}`,
		``,
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":4,"total_tokens":7}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	ch := make(chan protocoltypes.StreamEvent, 32)
	go func() {
		defer close(ch)
		readSSEIntoChannel(context.Background(), strings.NewReader(sseData), ch)
	}()

	var events []protocoltypes.StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) < 3 {
		t.Fatalf("got %d events, want at least 3", len(events))
	}

	// Check content deltas
	if events[0].ContentDelta != "Hello" {
		t.Errorf("events[0].ContentDelta = %q, want %q", events[0].ContentDelta, "Hello")
	}
	if events[1].ContentDelta != " world" {
		t.Errorf("events[1].ContentDelta = %q, want %q", events[1].ContentDelta, " world")
	}

	// Check tool call deltas
	if len(events[2].ToolCallDeltas) != 1 || events[2].ToolCallDeltas[0].ID != "call_1" {
		t.Errorf("events[2] should contain tool call with ID=call_1")
	}
	if events[2].ToolCallDeltas[0].Name != "greet" {
		t.Errorf("events[2].ToolCallDeltas[0].Name = %q, want %q", events[2].ToolCallDeltas[0].Name, "greet")
	}

	// Check finish event
	lastEv := events[len(events)-1]
	if lastEv.FinishReason != "stop" {
		t.Errorf("last event FinishReason = %q, want %q", lastEv.FinishReason, "stop")
	}
	if lastEv.Usage == nil || lastEv.Usage.TotalTokens != 7 {
		t.Errorf("last event Usage.TotalTokens = %v, want 7", lastEv.Usage)
	}
}

func TestReadSSEIntoChannel_ContextCancel(t *testing.T) {
	// Simulate a slow SSE stream that gets cancelled.
	ctx, cancel := context.WithCancel(context.Background())

	// Create a reader that blocks after sending one chunk.
	sseData := `data: {"choices":[{"delta":{"content":"first"},"finish_reason":""}]}` + "\n\n"

	ch := make(chan protocoltypes.StreamEvent, 32)
	go func() {
		defer close(ch)
		readSSEIntoChannel(ctx, strings.NewReader(sseData), ch)
	}()

	// Read the first event.
	ev := <-ch
	if ev.ContentDelta != "first" {
		t.Fatalf("ContentDelta = %q, want %q", ev.ContentDelta, "first")
	}

	// Cancel the context; the channel should close.
	cancel()
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed after context cancel")
	}
}

func TestAccumulateStream_FullResponse(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 8)

	go func() {
		ch <- protocoltypes.StreamEvent{ContentDelta: "Hello"}
		ch <- protocoltypes.StreamEvent{ContentDelta: " world"}
		ch <- protocoltypes.StreamEvent{
			ToolCallDeltas: []protocoltypes.StreamToolCallDelta{
				{Index: 0, ID: "call_1", Name: "test_tool", ArgumentsDelta: `{"key"`},
			},
		}
		ch <- protocoltypes.StreamEvent{
			ToolCallDeltas: []protocoltypes.StreamToolCallDelta{
				{Index: 0, ArgumentsDelta: `:"value"}`},
			},
		}
		ch <- protocoltypes.StreamEvent{
			FinishReason: "stop",
			Usage:        &UsageInfo{PromptTokens: 5, CompletionTokens: 3, TotalTokens: 8},
		}
		close(ch)
	}()

	resp, err := AccumulateStream(ch)
	if err != nil {
		t.Fatalf("AccumulateStream() error = %v", err)
	}

	if resp.Content != "Hello world" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello world")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 8 {
		t.Errorf("Usage.TotalTokens = %v, want 8", resp.Usage)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "test_tool" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", resp.ToolCalls[0].Name, "test_tool")
	}
	if resp.ToolCalls[0].Arguments["key"] != "value" {
		t.Errorf("ToolCalls[0].Arguments[key] = %v, want %q", resp.ToolCalls[0].Arguments["key"], "value")
	}
}

func TestAccumulateStream_Error(t *testing.T) {
	ch := make(chan protocoltypes.StreamEvent, 4)

	go func() {
		ch <- protocoltypes.StreamEvent{ContentDelta: "partial"}
		ch <- protocoltypes.StreamEvent{Err: fmt.Errorf("connection reset")}
		close(ch)
	}()

	_, err := AccumulateStream(ch)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "connection reset") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "connection reset")
	}
}

func TestChatStream_EndToEnd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		chunks := []string{
			`data: {"choices":[{"delta":{"content":"stream"},"finish_reason":""}]}`,
			`data: {"choices":[{"delta":{"content":"ed"},"finish_reason":""}]}`,
			`data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1,"total_tokens":3}}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			fmt.Fprintln(w, c)
			fmt.Fprintln(w)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	p := NewProviderWithOptions("key", server.URL, "", Options{Stream: true})

	ch, err := p.ChatStream(t.Context(), []Message{{Role: "user", Content: "hi"}}, nil, "test", nil)
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}

	resp, err := AccumulateStream(ch)
	if err != nil {
		t.Fatalf("AccumulateStream() error = %v", err)
	}

	if resp.Content != "streamed" {
		t.Errorf("Content = %q, want %q", resp.Content, "streamed")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 3 {
		t.Errorf("Usage.TotalTokens = %v, want 3", resp.Usage)
	}
}

func TestChatStream_EarlyCancel(t *testing.T) {
	serverDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(serverDone)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		// Send many chunks; expect the client to cancel early.
		for i := 0; i < 1000; i++ {
			select {
			case <-r.Context().Done():
				return
			default:
			}
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"x\"},\"finish_reason\":\"\"}]}\n\n")
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	p := NewProviderWithOptions("key", server.URL, "", Options{Stream: true})

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := p.ChatStream(ctx, []Message{{Role: "user", Content: "hi"}}, nil, "test", nil)
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}

	// Read a few events, then cancel.
	count := 0
	for ev := range ch {
		if ev.Err != nil {
			break
		}
		count++
		if count >= 5 {
			cancel()
		}
	}

	if count < 5 {
		t.Errorf("expected at least 5 events before cancel, got %d", count)
	}

	// Server should have received the cancellation.
	<-serverDone
}

func TestCanStream(t *testing.T) {
	p1 := NewProvider("key", "https://example.com", "")
	if p1.CanStream() {
		t.Error("CanStream() = true for non-stream provider")
	}

	p2 := NewProviderWithOptions("key", "https://example.com", "", Options{Stream: true})
	if !p2.CanStream() {
		t.Error("CanStream() = false for stream provider")
	}
}
