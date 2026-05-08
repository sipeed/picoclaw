package oauthprovider

import (
	"testing"
)

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

func TestBuildRequest_PreservesComplexToolSchemasByDefault(t *testing.T) {
	p := &AntigravityProvider{}
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"parent": map[string]any{
				"anyOf": []any{
					map[string]any{"$ref": "#/$defs/pageParent"},
					map[string]any{"$ref": "#/$defs/databaseParent"},
				},
			},
			"icon": map[string]any{
				"anyOf": []any{
					map[string]any{"type": "null"},
					map[string]any{"$ref": "#/$defs/emoji"},
				},
			},
		},
		"$defs": map[string]any{
			"pageParent": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page_id": map[string]any{"type": "string"},
				},
				"required": []any{"page_id"},
			},
			"databaseParent": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"database_id": map[string]any{"type": "string"},
				},
				"required": []any{"database_id"},
			},
			"emoji": map[string]any{
				"type":    "string",
				"pattern": "^:[a-z_]+:$",
			},
		},
	}

	req := p.buildRequest(
		[]Message{{Role: "user", Content: "hello"}},
		[]ToolDefinition{{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "mcp_notion_create",
				Description: "Create a Notion object",
				Parameters:  schema,
			},
		}},
		"gemini-3-flash",
		nil,
	)

	if len(req.Tools) != 1 || len(req.Tools[0].FunctionDeclarations) != 1 {
		t.Fatalf("request tools = %#v, want one function declaration", req.Tools)
	}

	got, ok := req.Tools[0].FunctionDeclarations[0].Parameters.(map[string]any)
	if !ok {
		t.Fatalf("parameters = %#v, want map", req.Tools[0].FunctionDeclarations[0].Parameters)
	}
	if got["$defs"] == nil {
		t.Fatalf("parameters = %#v, want raw schema with $defs preserved by default", got)
	}
}
