package oauthprovider

import "testing"

func TestBuildRequestUsesFunctionFieldsWhenToolCallNameMissing(t *testing.T) {
	p := &AntigravityProvider{}

	messages := []Message{
		{
			Role: "assistant",
			ToolCalls: []ToolCall{{
				ID: "call_read_file_123",
				Function: &FunctionCall{
					Name:      "read_file",
					Arguments: `{"path":"README.md"}`,
				},
			}},
		},
		{
			Role:       "tool",
			ToolCallID: "call_read_file_123",
			Content:    "ok",
		},
	}

	req := p.buildRequest(messages, nil, "", nil)
	if len(req.Contents) != 2 {
		t.Fatalf("expected 2 contents, got %d", len(req.Contents))
	}

	modelPart := req.Contents[0].Parts[0]
	if modelPart.FunctionCall == nil {
		t.Fatal("expected functionCall in assistant message")
	}
	if modelPart.FunctionCall.Name != "read_file" {
		t.Fatalf("expected functionCall name read_file, got %q", modelPart.FunctionCall.Name)
	}
	if got := modelPart.FunctionCall.Args["path"]; got != "README.md" {
		t.Fatalf("expected functionCall args[path] to be README.md, got %v", got)
	}

	toolPart := req.Contents[1].Parts[0]
	if toolPart.FunctionResponse == nil {
		t.Fatal("expected functionResponse in tool message")
	}
	if toolPart.FunctionResponse.Name != "read_file" {
		t.Fatalf("expected functionResponse name read_file, got %q", toolPart.FunctionResponse.Name)
	}
}

func TestParseSSEResponse_SplitsThoughtAndVisibleContent(t *testing.T) {
	p := &AntigravityProvider{}
	body := "data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"hidden reasoning\",\"thought\":true},{\"text\":\"visible answer\"}],\"role\":\"model\"},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":8,\"candidatesTokenCount\":17,\"totalTokenCount\":216}}}\n" +
		"data: [DONE]\n"

	resp, err := p.parseSSEResponse(body)
	if err != nil {
		t.Fatalf("parseSSEResponse() error = %v", err)
	}

	if resp.Content != "visible answer" {
		t.Fatalf("Content = %q, want %q", resp.Content, "visible answer")
	}
	if resp.ReasoningContent != "hidden reasoning" {
		t.Fatalf("ReasoningContent = %q, want %q", resp.ReasoningContent, "hidden reasoning")
	}
	if resp.FinishReason != "stop" {
		t.Fatalf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 216 {
		t.Fatalf("Usage.TotalTokens = %v, want %d", resp.Usage, 216)
	}
}
