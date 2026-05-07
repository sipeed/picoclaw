package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

// --- NewHTTPClient tests ---

func TestNewHTTPClient_DefaultTimeout(t *testing.T) {
	client := NewHTTPClient("")
	if client.Timeout != DefaultRequestTimeout {
		t.Errorf("timeout = %v, want %v", client.Timeout, DefaultRequestTimeout)
	}
}

func TestNewHTTPClient_WithProxy(t *testing.T) {
	client := NewHTTPClient("http://127.0.0.1:8080")
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport == nil {
		t.Fatalf("expected http.Transport with proxy, got %T", client.Transport)
	}
	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "api.example.com"}}
	gotProxy, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("proxy function error: %v", err)
	}
	if gotProxy == nil || gotProxy.String() != "http://127.0.0.1:8080" {
		t.Errorf("proxy = %v, want http://127.0.0.1:8080", gotProxy)
	}
}

func TestNewHTTPClient_NoProxy(t *testing.T) {
	client := NewHTTPClient("")
	if client.Transport != nil {
		t.Errorf("expected nil transport without proxy, got %T", client.Transport)
	}
}

func TestNewHTTPClient_InvalidProxy(t *testing.T) {
	// Should not panic, just log and return client without proxy
	client := NewHTTPClient("://bad-url")
	if client == nil {
		t.Fatal("expected non-nil client even with invalid proxy")
	}
}

// --- SerializeMessages tests ---

func TestSerializeMessages_PlainText(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi", ReasoningContent: "thinking..."},
	}
	result := SerializeMessages(messages)

	data, _ := json.Marshal(result)
	var msgs []map[string]any
	json.Unmarshal(data, &msgs)

	if msgs[0]["content"] != "hello" {
		t.Errorf("expected plain string content, got %v", msgs[0]["content"])
	}
	if msgs[1]["reasoning_content"] != "thinking..." {
		t.Errorf("reasoning_content not preserved, got %v", msgs[1]["reasoning_content"])
	}
}

func TestSerializeMessages_WithMedia(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "describe this", Media: []string{"data:image/png;base64,abc123"}},
	}
	result := SerializeMessages(messages)

	data, _ := json.Marshal(result)
	var msgs []map[string]any
	json.Unmarshal(data, &msgs)

	content, ok := msgs[0]["content"].([]any)
	if !ok {
		t.Fatalf("expected array content for media message, got %T", msgs[0]["content"])
	}
	if len(content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(content))
	}
}

func TestSerializeMessages_WithAudioMedia(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "transcribe this", Media: []string{"data:audio/ogg;base64,abc123"}},
	}
	result := SerializeMessages(messages)

	data, _ := json.Marshal(result)
	var msgs []map[string]any
	json.Unmarshal(data, &msgs)

	content, ok := msgs[0]["content"].([]any)
	if !ok {
		t.Fatalf("expected array content for media message, got %T", msgs[0]["content"])
	}
	if len(content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(content))
	}

	audioPart, ok := content[1].(map[string]any)
	if !ok {
		t.Fatalf("expected audio content part to be an object, got %T", content[1])
	}
	if audioPart["type"] != "input_audio" {
		t.Fatalf("audio part type = %v, want input_audio", audioPart["type"])
	}

	inputAudio, ok := audioPart["input_audio"].(map[string]any)
	if !ok {
		t.Fatalf("expected input_audio object, got %T", audioPart["input_audio"])
	}
	if inputAudio["format"] != "ogg" {
		t.Fatalf("audio format = %v, want ogg", inputAudio["format"])
	}
	if inputAudio["data"] != "abc123" {
		t.Fatalf("audio data = %v, want abc123", inputAudio["data"])
	}
}

func TestSerializeMessages_MediaWithToolCallID(t *testing.T) {
	messages := []Message{
		{Role: "tool", Content: "result", Media: []string{"data:image/png;base64,xyz"}, ToolCallID: "call_1"},
	}
	result := SerializeMessages(messages)

	data, _ := json.Marshal(result)
	var msgs []map[string]any
	json.Unmarshal(data, &msgs)

	if msgs[0]["tool_call_id"] != "call_1" {
		t.Errorf("tool_call_id not preserved, got %v", msgs[0]["tool_call_id"])
	}
}

func TestSerializeMessages_StripsSystemParts(t *testing.T) {
	messages := []Message{
		{
			Role:    "system",
			Content: "you are helpful",
			SystemParts: []protocoltypes.ContentBlock{
				{Type: "text", Text: "you are helpful"},
			},
		},
	}
	result := SerializeMessages(messages)

	data, _ := json.Marshal(result)
	if strings.Contains(string(data), "system_parts") {
		t.Error("system_parts should not appear in serialized output")
	}
}

func TestSerializeMessages_StripsInternalToolCallExtraContent(t *testing.T) {
	messages := []Message{
		{
			Role: "assistant",
			ToolCalls: []ToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: &FunctionCall{
					Name:             "read_file",
					Arguments:        `{"path":"README.md"}`,
					ThoughtSignature: "sig-1",
				},
				ExtraContent: &ExtraContent{
					Google: &GoogleExtra{
						ThoughtSignature: "sig-ignored-here",
					},
					ToolFeedbackExplanation: "Read README.md first.",
				},
			}},
		},
	}

	result := SerializeMessages(messages)

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	payload := string(data)
	if strings.Contains(payload, "extra_content") {
		t.Fatalf("serialized payload should not include internal extra_content: %s", payload)
	}
	if !strings.Contains(payload, "thought_signature") {
		t.Fatalf("serialized payload should preserve function thought_signature: %s", payload)
	}
}

func TestSerializeMessages_PreservesTopLevelThoughtSignature(t *testing.T) {
	messages := []Message{
		{
			Role: "assistant",
			ToolCalls: []ToolCall{{
				ID:               "call_1",
				Type:             "function",
				ThoughtSignature: "sig-1",
				Function: &FunctionCall{
					Name:      "read_file",
					Arguments: `{"path":"README.md"}`,
				},
			}},
		},
	}

	result := SerializeMessages(messages)

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	payload := string(data)
	if !strings.Contains(payload, `"thought_signature":"sig-1"`) {
		t.Fatalf("serialized payload should preserve top-level thought signature: %s", payload)
	}
}

func TestSerializeMessages_PreservesGoogleExtraThoughtSignature(t *testing.T) {
	messages := []Message{
		{
			Role: "assistant",
			ToolCalls: []ToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: &FunctionCall{
					Name:      "read_file",
					Arguments: `{"path":"README.md"}`,
				},
				ExtraContent: &ExtraContent{
					Google: &GoogleExtra{ThoughtSignature: "sig-1"},
				},
			}},
		},
	}

	result := SerializeMessages(messages)

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	payload := string(data)
	if strings.Contains(payload, "extra_content") {
		t.Fatalf("serialized payload should not include extra_content: %s", payload)
	}
	if !strings.Contains(payload, `"thought_signature":"sig-1"`) {
		t.Fatalf("serialized payload should preserve google thought signature: %s", payload)
	}
}

// --- ParseResponse tests ---

func TestParseResponse_BasicContent(t *testing.T) {
	body := `{"choices":[{"message":{"content":"hello world"},"finish_reason":"stop"}]}`
	out, err := ParseResponse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if out.Content != "hello world" {
		t.Errorf("Content = %q, want %q", out.Content, "hello world")
	}
	if out.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", out.FinishReason, "stop")
	}
}

func TestParseResponse_EmptyChoices(t *testing.T) {
	body := `{"choices":[]}`
	out, err := ParseResponse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if out.Content != "" {
		t.Errorf("Content = %q, want empty", out.Content)
	}
	if out.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", out.FinishReason, "stop")
	}
}

func TestParseResponse_WithToolCalls(t *testing.T) {
	body := `{"choices":[{"message":{"content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"SF\"}"}}]},"finish_reason":"tool_calls"}]}`
	out, err := ParseResponse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
	if out.ToolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", out.ToolCalls[0].Name, "get_weather")
	}
	if out.ToolCalls[0].Arguments["city"] != "SF" {
		t.Errorf("ToolCalls[0].Arguments[city] = %v, want SF", out.ToolCalls[0].Arguments["city"])
	}
}

func TestParseResponse_WithUsage(t *testing.T) {
	body := `{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
	out, err := ParseResponse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if out.Usage == nil {
		t.Fatal("Usage is nil")
	}
	if out.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", out.Usage.PromptTokens)
	}
}

func TestParseResponse_WithReasoningContent(t *testing.T) {
	body := `{"choices":[{"message":{"content":"2","reasoning_content":"Let me think... 1+1=2"},"finish_reason":"stop"}]}`
	out, err := ParseResponse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if out.ReasoningContent != "Let me think... 1+1=2" {
		t.Errorf("ReasoningContent = %q, want %q", out.ReasoningContent, "Let me think... 1+1=2")
	}
}

func TestParseResponse_WithToolFeedbackExplanationExtraContent(t *testing.T) {
	body := `{"choices":[{"message":{"content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"test_tool","arguments":"{}"},"extra_content":{"tool_feedback_explanation":"Check the current config before editing."}}]},"finish_reason":"tool_calls"}]}`
	out, err := ParseResponse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
	if out.ToolCalls[0].ExtraContent == nil {
		t.Fatal("ExtraContent is nil")
	}
	if out.ToolCalls[0].ExtraContent.ToolFeedbackExplanation != "Check the current config before editing." {
		t.Fatalf(
			"ToolFeedbackExplanation = %q, want %q",
			out.ToolCalls[0].ExtraContent.ToolFeedbackExplanation,
			"Check the current config before editing.",
		)
	}
}

func TestParseResponse_InvalidJSON(t *testing.T) {
	_, err := ParseResponse(strings.NewReader("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- DecodeToolCallArguments tests ---

func TestDecodeToolCallArguments_ObjectJSON(t *testing.T) {
	raw := json.RawMessage(`{"city":"Seattle","units":"metric"}`)
	args := DecodeToolCallArguments(raw, "test")
	if args["city"] != "Seattle" {
		t.Errorf("city = %v, want Seattle", args["city"])
	}
	if args["units"] != "metric" {
		t.Errorf("units = %v, want metric", args["units"])
	}
}

func TestDecodeToolCallArguments_ObjectJSON_NewlineEscape(t *testing.T) {
	raw := json.RawMessage(`{"content":"line1\nline2"}`)
	args := DecodeToolCallArguments(raw, "write_file")
	if args["content"] != "line1\nline2" {
		t.Errorf("content = %q, want newline-expanded string", args["content"])
	}
}

func TestDecodeToolCallArguments_ObjectJSON_LiteralBackslashN(t *testing.T) {
	raw := json.RawMessage(`{"content":"line1\\nline2"}`)
	args := DecodeToolCallArguments(raw, "write_file")
	if args["content"] != `line1\nline2` {
		t.Errorf("content = %q, want literal backslash-n", args["content"])
	}
}

func TestDecodeToolCallArguments_StringJSON(t *testing.T) {
	raw := json.RawMessage(`"{\"city\":\"SF\"}"`)
	args := DecodeToolCallArguments(raw, "test")
	if args["city"] != "SF" {
		t.Errorf("city = %v, want SF", args["city"])
	}
}

func TestDecodeToolCallArguments_StringJSON_NewlineEscape(t *testing.T) {
	raw := json.RawMessage(`"{\"content\":\"line1\\nline2\"}"`)
	args := DecodeToolCallArguments(raw, "write_file")
	if args["content"] != "line1\nline2" {
		t.Errorf("content = %q, want newline-expanded string", args["content"])
	}
}

func TestDecodeToolCallArguments_StringJSON_LiteralBackslashN(t *testing.T) {
	raw := json.RawMessage(`"{\"content\":\"line1\\\\nline2\"}"`)
	args := DecodeToolCallArguments(raw, "write_file")
	if args["content"] != `line1\nline2` {
		t.Errorf("content = %q, want literal backslash-n", args["content"])
	}
}

func TestDecodeToolCallArguments_EmptyInput(t *testing.T) {
	args := DecodeToolCallArguments(nil, "test")
	if len(args) != 0 {
		t.Errorf("expected empty map, got %v", args)
	}
}

func TestDecodeToolCallArguments_NullInput(t *testing.T) {
	args := DecodeToolCallArguments(json.RawMessage(`null`), "test")
	if len(args) != 0 {
		t.Errorf("expected empty map, got %v", args)
	}
}

func TestDecodeToolCallArguments_InvalidJSON(t *testing.T) {
	args := DecodeToolCallArguments(json.RawMessage(`not-json`), "test")
	if _, ok := args["raw"]; !ok {
		t.Error("expected 'raw' fallback key for invalid JSON")
	}
}

func TestDecodeToolCallArguments_EmptyStringJSON(t *testing.T) {
	args := DecodeToolCallArguments(json.RawMessage(`"  "`), "test")
	if len(args) != 0 {
		t.Errorf("expected empty map for whitespace string, got %v", args)
	}
}

// --- HandleErrorResponse tests ---

func TestHandleErrorResponse_JSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("http.Get() error = %v", err)
	}
	defer resp.Body.Close()
	err = HandleErrorResponse(resp, server.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error should contain status code, got %v", err)
	}
	if strings.Contains(err.Error(), "HTML") {
		t.Errorf("should not mention HTML for JSON error, got %v", err)
	}
}

func TestHandleErrorResponse_HTMLError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("<!DOCTYPE html><html><body>bad gateway</body></html>"))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("http.Get() error = %v", err)
	}
	defer resp.Body.Close()
	err = HandleErrorResponse(resp, server.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HTML instead of JSON") {
		t.Errorf("expected HTML error message, got %v", err)
	}
}

// --- ReadAndParseResponse tests ---

func TestReadAndParseResponse_ValidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("http.Get() error = %v", err)
	}
	defer resp.Body.Close()
	out, err := ReadAndParseResponse(resp, server.URL)
	if err != nil {
		t.Fatalf("ReadAndParseResponse() error = %v", err)
	}
	if out.Content != "ok" {
		t.Errorf("Content = %q, want %q", out.Content, "ok")
	}
}

func TestReadAndParseResponse_HTMLResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<!DOCTYPE html><html><body>login page</body></html>"))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("http.Get() error = %v", err)
	}
	defer resp.Body.Close()
	_, err = ReadAndParseResponse(resp, server.URL)
	if err == nil {
		t.Fatal("expected error for HTML response")
	}
	if !strings.Contains(err.Error(), "HTML instead of JSON") {
		t.Errorf("expected HTML error, got %v", err)
	}
}

// --- LooksLikeHTML tests ---

func TestLooksLikeHTML_ContentTypeHTML(t *testing.T) {
	if !LooksLikeHTML(nil, "text/html; charset=utf-8") {
		t.Error("expected true for text/html content type")
	}
}

func TestLooksLikeHTML_ContentTypeXHTML(t *testing.T) {
	if !LooksLikeHTML(nil, "application/xhtml+xml") {
		t.Error("expected true for xhtml content type")
	}
}

func TestLooksLikeHTML_BodyPrefix(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"doctype", "<!DOCTYPE html><html>"},
		{"html tag", "<html><body>"},
		{"head tag", "<head><title>"},
		{"body tag", "<body>content"},
		{"whitespace before", "  \n\t<!DOCTYPE html>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !LooksLikeHTML([]byte(tt.body), "application/json") {
				t.Errorf("expected true for body %q", tt.body)
			}
		})
	}
}

func TestLooksLikeHTML_NotHTML(t *testing.T) {
	if LooksLikeHTML([]byte(`{"error":"bad"}`), "application/json") {
		t.Error("expected false for JSON body")
	}
}

// --- ResponsePreview tests ---

func TestResponsePreview_Short(t *testing.T) {
	got := ResponsePreview([]byte("hello"), 128)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestResponsePreview_Truncated(t *testing.T) {
	body := strings.Repeat("a", 200)
	got := ResponsePreview([]byte(body), 128)
	if len(got) != 131 { // 128 + "..."
		t.Errorf("len = %d, want 131", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("expected ... suffix")
	}
}

func TestResponsePreview_Empty(t *testing.T) {
	got := ResponsePreview([]byte(""), 128)
	if got != "<empty>" {
		t.Errorf("got %q, want %q", got, "<empty>")
	}
}

func TestResponsePreview_Whitespace(t *testing.T) {
	got := ResponsePreview([]byte("  \n\t  "), 128)
	if got != "<empty>" {
		t.Errorf("got %q, want %q for whitespace-only body", got, "<empty>")
	}
}

// --- AsInt tests ---

func TestAsInt(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want int
		ok   bool
	}{
		{"int", 42, 42, true},
		{"int64", int64(99), 99, true},
		{"float64", float64(512), 512, true},
		{"float32", float32(256), 256, true},
		{"string", "nope", 0, false},
		{"nil", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := AsInt(tt.val)
			if ok != tt.ok || got != tt.want {
				t.Errorf("AsInt(%v) = (%d, %v), want (%d, %v)", tt.val, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// --- AsFloat tests ---

func TestAsFloat(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want float64
		ok   bool
	}{
		{"float64", float64(0.7), 0.7, true},
		{"float32", float32(0.5), float64(float32(0.5)), true},
		{"int", 1, 1.0, true},
		{"int64", int64(100), 100.0, true},
		{"string", "nope", 0, false},
		{"nil", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := AsFloat(tt.val)
			if ok != tt.ok || got != tt.want {
				t.Errorf("AsFloat(%v) = (%f, %v), want (%f, %v)", tt.val, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// --- ParseDataAudioURL tests ---

func TestParseDataAudioURL(t *testing.T) {
	tests := []struct {
		name       string
		mediaURL   string
		wantFormat string
		wantData   string
		wantOK     bool
	}{
		{"valid mp3", "data:audio/mp3;base64,SGVsbG8=", "mp3", "SGVsbG8=", true},
		{"valid wav", "data:audio/wav;base64,AAAA", "wav", "AAAA", true},
		{"not audio", "data:image/png;base64,abc", "", "", false},
		{"no comma", "data:audio/mp3;base64", "", "", false},
		{"empty data", "data:audio/mp3;base64,", "", "", false},
		{"empty string", "", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, data, ok := ParseDataAudioURL(tt.mediaURL)
			if ok != tt.wantOK || format != tt.wantFormat || data != tt.wantData {
				t.Errorf(
					"ParseDataAudioURL(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.mediaURL, format, data, ok,
					tt.wantFormat, tt.wantData, tt.wantOK,
				)
			}
		})
	}
}

// --- WrapHTMLResponseError tests ---

func TestWrapHTMLResponseError(t *testing.T) {
	err := WrapHTMLResponseError(502, []byte("<html>bad</html>"), "text/html", "https://api.example.com")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "502") {
		t.Errorf("expected status code in error, got %v", msg)
	}
	if !strings.Contains(msg, "https://api.example.com") {
		t.Errorf("expected api base in error, got %v", msg)
	}
	if !strings.Contains(msg, "HTML instead of JSON") {
		t.Errorf("expected HTML mention in error, got %v", msg)
	}
}

// --- HandleErrorResponse with read failure ---

func TestHandleErrorResponse_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		// empty body
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("http.Get() error = %v", err)
	}
	defer resp.Body.Close()
	err = HandleErrorResponse(resp, server.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status code, got %v", err)
	}
}

// --- ReadAndParseResponse with invalid JSON ---

func TestReadAndParseResponse_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("http.Get() error = %v", err)
	}
	defer resp.Body.Close()
	_, err = ReadAndParseResponse(resp, server.URL)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- ParseResponse with thought_signature (Google/Gemini) ---

func TestParseResponse_WithThoughtSignature(t *testing.T) {
	body := `{"choices":[{"message":{"content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"test_tool","arguments":"{}"},"extra_content":{"google":{"thought_signature":"sig123"}}}]},"finish_reason":"tool_calls"}]}`
	out, err := ParseResponse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
	if out.ToolCalls[0].ThoughtSignature != "sig123" {
		t.Errorf("ThoughtSignature = %q, want %q", out.ToolCalls[0].ThoughtSignature, "sig123")
	}
	if out.ToolCalls[0].ExtraContent == nil || out.ToolCalls[0].ExtraContent.Google == nil {
		t.Fatal("ExtraContent.Google is nil")
	}
	if out.ToolCalls[0].ExtraContent.Google.ThoughtSignature != "sig123" {
		t.Errorf("ExtraContent.Google.ThoughtSignature = %q, want %q",
			out.ToolCalls[0].ExtraContent.Google.ThoughtSignature, "sig123")
	}
}

func TestParseResponse_WithFunctionThoughtSignature(t *testing.T) {
	body := `{"choices":[{"message":{"content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"test_tool","arguments":"{}","thought_signature":"sig456"}}]},"finish_reason":"tool_calls"}]}`
	out, err := ParseResponse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(out.ToolCalls))
	}
	if out.ToolCalls[0].ThoughtSignature != "sig456" {
		t.Fatalf("ThoughtSignature = %q, want %q", out.ToolCalls[0].ThoughtSignature, "sig456")
	}
	if out.ToolCalls[0].ExtraContent == nil || out.ToolCalls[0].ExtraContent.Google == nil {
		t.Fatal("ExtraContent.Google is nil")
	}
	if out.ToolCalls[0].ExtraContent.Google.ThoughtSignature != "sig456" {
		t.Fatalf(
			"ExtraContent.Google.ThoughtSignature = %q, want %q",
			out.ToolCalls[0].ExtraContent.Google.ThoughtSignature,
			"sig456",
		)
	}
}
